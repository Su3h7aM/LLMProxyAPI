package auth

import (
	"context"
	"strconv"
	"testing"
	"time"

	cliproxyexecutor "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/executor"
)

// TestManagerPickNextUsesSoonestResetSelector runs the real Manager dispatch. The selector
// is not a built-in, so pickNext takes the legacy path and calls Pick with every candidate.
func TestManagerPickNextUsesSoonestResetSelector(t *testing.T) {
	now := time.Now()
	resetAt := func(d time.Duration) map[string]string {
		return map[string]string{
			"X-Codex-Primary-Reset-At": strconv.FormatInt(now.Add(d).Unix(), 10),
		}
	}

	manager := NewManager(nil, nil, nil)
	manager.SetSelector(&SoonestResetSelector{})
	manager.RegisterExecutor(&refreshMockExecutor{id: "codex"})
	for _, auth := range []*Auth{
		{ID: "auth-late", Provider: "codex", Status: StatusActive, Quota: QuotaState{ObservedAt: now, Signals: resetAt(4 * time.Hour)}},
		{ID: "auth-soon", Provider: "codex", Status: StatusActive, Quota: QuotaState{ObservedAt: now, Signals: resetAt(time.Hour)}},
		{ID: "auth-unknown", Provider: "codex", Status: StatusActive},
	} {
		if _, errRegister := manager.Register(context.Background(), auth); errRegister != nil {
			t.Fatalf("Register(%s) error = %v", auth.ID, errRegister)
		}
	}

	selected, exec, errPick := manager.pickNext(context.Background(), "codex", "", cliproxyexecutor.Options{}, nil)
	if errPick != nil {
		t.Fatalf("pickNext error = %v", errPick)
	}
	if exec == nil {
		t.Fatal("pickNext returned no executor")
	}
	if selected == nil || selected.ID != "auth-soon" {
		got := ""
		if selected != nil {
			got = selected.ID
		}
		t.Fatalf("pickNext selected %q, want auth-soon", got)
	}
}
