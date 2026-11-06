package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Su3h7aM/LLMProxyAPI/internal/registry"
	cliproxyexecutor "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/executor"
)

// newQuotaRebindManagerForTest registers a high-priority and a low-priority credential
// for one provider and model, behind a quota-only session affinity selector.
func newQuotaRebindManagerForTest(t *testing.T, provider, model string) (*Manager, string, string) {
	t.Helper()
	ctx := context.Background()
	highID := provider + "-high"
	lowID := provider + "-low"

	manager := NewManager(nil, nil, nil)
	affinity := NewSessionAffinitySelectorWithConfig(SessionAffinityConfig{
		Fallback: &RoundRobinSelector{},
		TTL:      time.Hour,
		RebindOn: SessionAffinityRebindOnQuota,
	})
	t.Cleanup(affinity.Stop)
	manager.SetSelector(affinity)
	manager.RegisterExecutor(schedulerTestExecutor{provider: provider})

	for _, auth := range []*Auth{
		{ID: highID, Provider: provider, Status: StatusActive, Attributes: map[string]string{"priority": "1"}},
		{ID: lowID, Provider: provider, Status: StatusActive, Attributes: map[string]string{"priority": "0"}},
	} {
		if _, errRegister := manager.Register(WithSkipPersist(ctx), auth); errRegister != nil {
			t.Fatalf("Register(%s): %v", auth.ID, errRegister)
		}
		registry.GetGlobalRegistry().RegisterClient(auth.ID, provider, []*registry.ModelInfo{{ID: model}})
		t.Cleanup(func() { registry.GetGlobalRegistry().UnregisterClient(auth.ID) })
	}
	return manager, highID, lowID
}

func TestManagerSessionAffinityRebindOnQuotaUnbindsRepeatedlyFailingCredential(t *testing.T) {
	const provider = "affinity-dead-credential"
	const model = "affinity-dead-credential-model"
	ctx := context.Background()
	manager, highID, lowID := newQuotaRebindManagerForTest(t, provider, model)
	opts := cliproxyexecutor.Options{Metadata: map[string]any{
		cliproxyexecutor.DerivedSessionIDMetadataKey: "dead-session",
	}}
	pick := func() string {
		t.Helper()
		auth, _, errPick := manager.pickNext(ctx, provider, model, opts, nil)
		if errPick != nil || auth == nil {
			t.Fatalf("pick: auth=%v err=%v", auth, errPick)
		}
		return auth.ID
	}
	report := func(authID string, success bool, err *Error) {
		t.Helper()
		manager.MarkResult(ctx, Result{AuthID: authID, Provider: provider, Model: model, Success: success, Error: err, Options: opts})
	}
	upstream502 := &Error{HTTPStatus: http.StatusBadGateway, Message: "502"}

	if got := pick(); got != highID {
		t.Fatalf("cold binding = %q, want %q", got, highID)
	}
	report(highID, true, nil)

	// First transient failure: the request fails over and the binding stays on highID.
	report(highID, false, upstream502)
	if got := pick(); got != lowID {
		t.Fatalf("first transient failover = %q, want %q", got, lowID)
	}
	report(lowID, true, nil)

	// highID recovers from its cooldown, gets one probe, and fails again.
	expireSessionAffinityPriorityModelCooldown(t, manager, highID, model)
	if got := pick(); got != highID {
		t.Fatalf("probe pick = %q, want bound %q", got, highID)
	}
	report(highID, false, upstream502)

	// Second consecutive failure: highID is dead, so the session unbinds and moves to lowID.
	if got := pick(); got != lowID {
		t.Fatalf("dead credential failover = %q, want %q", got, lowID)
	}
	report(lowID, true, nil)

	expireSessionAffinityPriorityModelCooldown(t, manager, highID, model)
	if got := pick(); got != lowID {
		t.Fatalf("binding after dead credential = %q, want sticky %q", got, lowID)
	}
}

func TestManagerSessionAffinityRebindOnQuotaSuccessResetsTransientStrikes(t *testing.T) {
	const provider = "affinity-strike-reset"
	const model = "affinity-strike-reset-model"
	ctx := context.Background()
	manager, highID, lowID := newQuotaRebindManagerForTest(t, provider, model)
	opts := cliproxyexecutor.Options{Metadata: map[string]any{
		cliproxyexecutor.DerivedSessionIDMetadataKey: "strike-reset-session",
	}}
	pick := func() string {
		t.Helper()
		auth, _, errPick := manager.pickNext(ctx, provider, model, opts, nil)
		if errPick != nil || auth == nil {
			t.Fatalf("pick: auth=%v err=%v", auth, errPick)
		}
		return auth.ID
	}
	report := func(authID string, success bool, err *Error) {
		t.Helper()
		manager.MarkResult(ctx, Result{AuthID: authID, Provider: provider, Model: model, Success: success, Error: err, Options: opts})
	}
	upstream503 := &Error{HTTPStatus: http.StatusServiceUnavailable, Message: "503"}

	if got := pick(); got != highID {
		t.Fatalf("cold binding = %q, want %q", got, highID)
	}
	report(highID, true, nil)

	report(highID, false, upstream503)
	if got := pick(); got != lowID {
		t.Fatalf("first transient failover = %q, want %q", got, lowID)
	}
	report(lowID, true, nil)

	// A successful probe ends the failure run.
	expireSessionAffinityPriorityModelCooldown(t, manager, highID, model)
	if got := pick(); got != highID {
		t.Fatalf("probe pick = %q, want bound %q", got, highID)
	}
	report(highID, true, nil)

	// A new single failure after the success must not count as a dead credential.
	report(highID, false, upstream503)
	if got := pick(); got != lowID {
		t.Fatalf("failover after reset = %q, want %q", got, lowID)
	}
	report(lowID, true, nil)

	expireSessionAffinityPriorityModelCooldown(t, manager, highID, model)
	if got := pick(); got != highID {
		t.Fatalf("binding after reset = %q, want %q", got, highID)
	}
}

func TestManagerSessionAffinityRebindOnQuotaRebindsOnUnauthorized(t *testing.T) {
	const provider = "affinity-unauthorized"
	const model = "affinity-unauthorized-model"
	ctx := context.Background()
	manager, highID, lowID := newQuotaRebindManagerForTest(t, provider, model)
	opts := cliproxyexecutor.Options{Metadata: map[string]any{
		cliproxyexecutor.DerivedSessionIDMetadataKey: "unauthorized-session",
	}}
	pick := func() string {
		t.Helper()
		auth, _, errPick := manager.pickNext(ctx, provider, model, opts, nil)
		if errPick != nil || auth == nil {
			t.Fatalf("pick: auth=%v err=%v", auth, errPick)
		}
		return auth.ID
	}

	if got := pick(); got != highID {
		t.Fatalf("cold binding = %q, want %q", got, highID)
	}
	manager.MarkResult(ctx, Result{AuthID: highID, Provider: provider, Model: model, Success: true, Options: opts})

	manager.MarkResult(ctx, Result{
		AuthID:   highID,
		Provider: provider,
		Model:    model,
		Success:  false,
		Error:    &Error{HTTPStatus: http.StatusUnauthorized, Message: "unauthorized"},
		Options:  opts,
	})
	if got := pick(); got != lowID {
		t.Fatalf("401 failover = %q, want %q", got, lowID)
	}
	manager.MarkResult(ctx, Result{AuthID: lowID, Provider: provider, Model: model, Success: true, Options: opts})

	expireSessionAffinityPriorityModelCooldown(t, manager, highID, model)
	if got := pick(); got != lowID {
		t.Fatalf("binding after 401 = %q, want sticky %q", got, lowID)
	}
}

// TestSessionAffinityQuotaOnlyBoundCredentialStates checks, without a manager, whether the
// binding moves when the bound credential is degraded and then recovers.
func TestSessionAffinityQuotaOnlyBoundCredentialStates(t *testing.T) {
	cooldown := func(auth *Auth, status int) {
		auth.Unavailable = true
		auth.NextRetryAfter = time.Now().Add(time.Minute)
		auth.LastError = &Error{HTTPStatus: status, Message: "upstream"}
	}
	for _, testCase := range []struct {
		name        string
		degrade     func(*Auth)
		wantBinding string
	}{
		{name: "transient 502 cooldown keeps binding", degrade: func(a *Auth) { cooldown(a, http.StatusBadGateway) }, wantBinding: "a"},
		{name: "quota 429 cooldown rebinds", degrade: func(a *Auth) { cooldown(a, http.StatusTooManyRequests) }, wantBinding: "b"},
		{name: "unauthorized 401 rebinds", degrade: func(a *Auth) { cooldown(a, http.StatusUnauthorized) }, wantBinding: "b"},
		{name: "disabled credential rebinds", degrade: func(a *Auth) {
			a.Disabled = true
			a.Status = StatusDisabled
		}, wantBinding: "b"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			selector := NewSessionAffinitySelectorWithConfig(SessionAffinityConfig{
				Fallback: &FillFirstSelector{},
				RebindOn: SessionAffinityRebindOnQuota,
			})
			t.Cleanup(selector.Stop)
			a := &Auth{ID: "a", Provider: "codex", Status: StatusActive}
			b := &Auth{ID: "b", Provider: "codex", Status: StatusActive}
			opts := cliproxyexecutor.Options{Headers: map[string][]string{"Session_id": {"direct-state-session"}}}
			pick := func() string {
				t.Helper()
				selected, err := selector.Pick(context.Background(), "codex", "model", opts, []*Auth{a, b})
				if err != nil || selected == nil {
					t.Fatalf("pick: %v %v", selected, err)
				}
				return selected.ID
			}

			if got := pick(); got != "a" {
				t.Fatalf("cold binding = %s, want a", got)
			}
			testCase.degrade(a)
			if got := pick(); got != "b" {
				t.Fatalf("degraded pick = %s, want failover b", got)
			}

			a.Unavailable = false
			a.NextRetryAfter = time.Time{}
			a.LastError = nil
			a.Disabled = false
			a.Status = StatusActive
			if got := pick(); got != testCase.wantBinding {
				t.Fatalf("binding after recovery = %s, want %s", got, testCase.wantBinding)
			}
		})
	}
}
