package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Su3h7aM/LLMProxyAPI/internal/registry"
	cliproxyexecutor "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/executor"
)

func TestManagerSessionAffinityQuotaPinsFailoverWhileBoundCools(t *testing.T) {
	const provider = "affinity-failover-pin"
	const model = "affinity-failover-pin-model"
	ctx := context.Background()

	manager := NewManager(nil, nil, nil)
	affinity := NewSessionAffinitySelectorWithConfig(SessionAffinityConfig{
		Fallback: &RoundRobinSelector{},
		TTL:      time.Hour,
		RebindOn: SessionAffinityRebindOnQuota,
	})
	t.Cleanup(affinity.Stop)
	manager.SetSelector(affinity)
	manager.RegisterExecutor(schedulerTestExecutor{provider: provider})

	highID := provider + "-high"
	low1ID := provider + "-low1"
	low2ID := provider + "-low2"
	for _, auth := range []*Auth{
		{ID: highID, Provider: provider, Status: StatusActive, Attributes: map[string]string{"priority": "1"}},
		{ID: low1ID, Provider: provider, Status: StatusActive, Attributes: map[string]string{"priority": "0"}},
		{ID: low2ID, Provider: provider, Status: StatusActive, Attributes: map[string]string{"priority": "0"}},
	} {
		if _, errRegister := manager.Register(WithSkipPersist(ctx), auth); errRegister != nil {
			t.Fatalf("Register(%s): %v", auth.ID, errRegister)
		}
		registry.GetGlobalRegistry().RegisterClient(auth.ID, provider, []*registry.ModelInfo{{ID: model}})
		t.Cleanup(func() { registry.GetGlobalRegistry().UnregisterClient(auth.ID) })
	}

	opts := cliproxyexecutor.Options{Metadata: map[string]any{
		cliproxyexecutor.DerivedSessionIDMetadataKey: "failover-pin-session",
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

	// The bound credential fails once and cools down. The first request fails over.
	report(highID, false, upstream502)
	failover := pick()
	if failover != low1ID && failover != low2ID {
		t.Fatalf("first failover = %q, want a low-priority credential", failover)
	}
	report(failover, true, nil)

	// While highID cools down, every request of the session stays on the pinned failover.
	for i := 0; i < 4; i++ {
		if got := pick(); got != failover {
			t.Fatalf("in-window pick %d = %q, want pinned %q", i, got, failover)
		}
		report(failover, true, nil)
	}

	// The pinned credential fails too. Its cooldown removes it from the candidates,
	// so the session moves to the other low-priority credential and re-pins there.
	report(failover, false, upstream502)
	moved := pick()
	if moved == failover || moved == highID {
		t.Fatalf("pinned credential failure pick = %q, want the other low-priority credential", moved)
	}
	report(moved, true, nil)
	for i := 0; i < 3; i++ {
		if got := pick(); got != moved {
			t.Fatalf("re-pinned pick %d = %q, want %q", i, got, moved)
		}
		report(moved, true, nil)
	}

	// highID recovers. The session returns to its original credential, and the binding stays there.
	expireSessionAffinityPriorityModelCooldown(t, manager, highID, model)
	if got := pick(); got != highID {
		t.Fatalf("pick after recovery = %q, want original %q", got, highID)
	}
	report(highID, true, nil)
	if got := pick(); got != highID {
		t.Fatalf("binding after return = %q, want %q", got, highID)
	}
}
