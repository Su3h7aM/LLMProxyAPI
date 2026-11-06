package auth

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	cliproxyexecutor "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/executor"
)

func TestSoonestResetSelectorPick(t *testing.T) {
	now := time.Now()
	at := func(d time.Duration) string {
		return strconv.FormatInt(now.Add(d).Unix(), 10)
	}
	codex := func(id string, signals map[string]string) *Auth {
		return &Auth{ID: id, Provider: "codex", Quota: QuotaState{ObservedAt: now, Signals: signals}}
	}

	tests := []struct {
		name  string
		auths []*Auth
		want  string
	}{
		{
			name: "soonest codex reset wins",
			auths: []*Auth{
				codex("auth-a", map[string]string{"X-Codex-Primary-Reset-At": at(4 * time.Hour), "X-Codex-Primary-Window-Minutes": "10080", "X-Codex-Primary-Used-Percent": "2"}),
				codex("auth-b", map[string]string{"X-Codex-Primary-Reset-At": at(time.Hour), "X-Codex-Primary-Window-Minutes": "10080", "X-Codex-Primary-Used-Percent": "40"}),
			},
			want: "auth-b",
		},
		{
			name: "relative codex reset counts from observation",
			auths: []*Auth{
				codex("auth-a", map[string]string{"X-Codex-Primary-Reset-After-Seconds": "3600"}),
				codex("auth-b", map[string]string{"X-Codex-Primary-Reset-At": at(2 * time.Hour)}),
			},
			want: "auth-a",
		},
		{
			name: "longest codex window decides the reset",
			auths: []*Auth{
				codex("auth-a", map[string]string{
					"X-Codex-Primary-Reset-At":              at(time.Hour),
					"X-Codex-Primary-Window-Minutes":        "300",
					"X-Codex-Secondary-Reset-At":            at(48 * time.Hour),
					"X-Codex-Secondary-Window-Minutes":      "10080",
					"X-Codex-Secondary-Used-Percent":        "10",
					"X-Codex-Primary-Used-Percent":          "5",
					"X-Codex-Secondary-Reset-After-Seconds": "172800",
				}),
				codex("auth-b", map[string]string{"X-Codex-Secondary-Reset-At": at(24 * time.Hour), "X-Codex-Secondary-Window-Minutes": "10080"}),
			},
			want: "auth-b",
		},
		{
			name: "unknown reset sorts after known reset",
			auths: []*Auth{
				{ID: "auth-a", Provider: "codex"},
				codex("auth-b", map[string]string{"X-Codex-Primary-Reset-At": at(4 * time.Hour)}),
			},
			want: "auth-b",
		},
		{
			name: "unknown resets keep ID order",
			auths: []*Auth{
				{ID: "auth-c", Provider: "codex"},
				{ID: "auth-a", Provider: "codex"},
				{ID: "auth-b", Provider: "codex"},
			},
			want: "auth-a",
		},
		{
			name: "equal resets keep ID order",
			auths: []*Auth{
				codex("auth-b", map[string]string{"X-Codex-Primary-Reset-At": at(2 * time.Hour)}),
				codex("auth-a", map[string]string{"X-Codex-Primary-Reset-At": at(2 * time.Hour)}),
			},
			want: "auth-a",
		},
		{
			name: "exhausted codex window ranks last",
			auths: []*Auth{
				codex("auth-a", map[string]string{"X-Codex-Primary-Reset-At": at(time.Hour), "X-Codex-Primary-Used-Percent": "100"}),
				codex("auth-b", map[string]string{"X-Codex-Primary-Reset-At": at(4 * time.Hour), "X-Codex-Primary-Used-Percent": "30"}),
			},
			want: "auth-b",
		},
		{
			name: "rolled over exhausted window is not exhausted",
			auths: []*Auth{
				codex("auth-a", map[string]string{
					"X-Codex-Primary-Reset-At":         at(-time.Minute),
					"X-Codex-Primary-Used-Percent":     "100",
					"X-Codex-Secondary-Reset-At":       at(2 * time.Hour),
					"X-Codex-Secondary-Window-Minutes": "10080",
				}),
				codex("auth-b", map[string]string{"X-Codex-Primary-Reset-At": at(4 * time.Hour)}),
			},
			want: "auth-a",
		},
		{
			name: "past reset ranks as unknown",
			auths: []*Auth{
				codex("auth-a", map[string]string{"X-Codex-Primary-Reset-At": at(-time.Hour), "X-Codex-Primary-Used-Percent": "10"}),
				codex("auth-b", map[string]string{"X-Codex-Primary-Reset-At": at(4 * time.Hour)}),
			},
			want: "auth-b",
		},
		{
			name: "claude longest window decides the reset",
			auths: []*Auth{
				{ID: "auth-a", Provider: "claude", Quota: QuotaState{ObservedAt: now, Signals: map[string]string{
					"Anthropic-Ratelimit-Unified-5h-Reset":  at(time.Hour),
					"Anthropic-Ratelimit-Unified-7d-Reset":  at(48 * time.Hour),
					"Anthropic-Ratelimit-Unified-5h-Status": "allowed",
					"Anthropic-Ratelimit-Unified-7d-Status": "allowed",
				}}},
				{ID: "auth-b", Provider: "claude", Quota: QuotaState{ObservedAt: now, Signals: map[string]string{
					"Anthropic-Ratelimit-Unified-7d-Reset": at(24 * time.Hour),
				}}},
			},
			want: "auth-b",
		},
		{
			name: "rejected claude five hour window is exhausted",
			auths: []*Auth{
				{ID: "auth-a", Provider: "claude", Quota: QuotaState{ObservedAt: now, Signals: map[string]string{
					"Anthropic-Ratelimit-Unified-5h-Status": "rejected",
					"Anthropic-Ratelimit-Unified-5h-Reset":  at(time.Hour),
				}}},
				{ID: "auth-b", Provider: "claude", Quota: QuotaState{ObservedAt: now, Signals: map[string]string{
					"Anthropic-Ratelimit-Unified-7d-Reset": at(72 * time.Hour),
				}}},
			},
			want: "auth-b",
		},
		{
			name: "claude overage rejection does not exhaust shared windows",
			auths: []*Auth{
				{ID: "auth-a", Provider: "claude", Quota: QuotaState{ObservedAt: now, Signals: map[string]string{
					"Anthropic-Ratelimit-Unified-Status":         "rejected",
					"Anthropic-Ratelimit-Unified-Overage-Status": "rejected",
					"Anthropic-Ratelimit-Unified-7d-Status":      "allowed",
					"Anthropic-Ratelimit-Unified-7d-Reset":       at(24 * time.Hour),
				}}},
				{ID: "auth-b", Provider: "claude", Quota: QuotaState{ObservedAt: now, Signals: map[string]string{
					"Anthropic-Ratelimit-Unified-7d-Reset": at(48 * time.Hour),
				}}},
			},
			want: "auth-a",
		},
		{
			name: "higher priority tier wins over an earlier reset",
			auths: []*Auth{
				{ID: "auth-a", Provider: "codex", Attributes: map[string]string{"priority": "0"}, Quota: QuotaState{ObservedAt: now, Signals: map[string]string{"X-Codex-Primary-Reset-At": at(time.Hour)}}},
				{ID: "auth-b", Provider: "codex", Attributes: map[string]string{"priority": "1"}, Quota: QuotaState{ObservedAt: now, Signals: map[string]string{"X-Codex-Primary-Reset-At": at(4 * time.Hour)}}},
			},
			want: "auth-b",
		},
		{
			name: "credential in cooldown is skipped",
			auths: []*Auth{
				{ID: "auth-a", Provider: "codex", Quota: QuotaState{
					Exceeded:      true,
					Reason:        "credential_quota",
					NextRecoverAt: now.Add(2 * time.Hour),
					ObservedAt:    now,
					Signals:       map[string]string{"X-Codex-Primary-Reset-At": at(30 * time.Minute)},
				}},
				codex("auth-b", map[string]string{"X-Codex-Primary-Reset-At": at(4 * time.Hour)}),
			},
			want: "auth-b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := &SoonestResetSelector{}
			got, err := selector.Pick(context.Background(), "mixed", "gpt-5", cliproxyexecutor.Options{}, tt.auths)
			if err != nil {
				t.Fatalf("Pick() error = %v", err)
			}
			if got.ID != tt.want {
				t.Fatalf("Pick() = %q, want %q", got.ID, tt.want)
			}
		})
	}
}

func TestSoonestResetSelectorKeepsSessionBinding(t *testing.T) {
	now := time.Now()
	reset := func(d time.Duration) map[string]string {
		return map[string]string{"X-Codex-Primary-Reset-At": strconv.FormatInt(now.Add(d).Unix(), 10)}
	}
	authA := &Auth{ID: "auth-a", Provider: "codex", Quota: QuotaState{ObservedAt: now, Signals: reset(4 * time.Hour)}}
	authB := &Auth{ID: "auth-b", Provider: "codex", Quota: QuotaState{ObservedAt: now, Signals: reset(time.Hour)}}
	auths := []*Auth{authA, authB}

	selector := NewSessionAffinitySelectorWithConfig(SessionAffinityConfig{
		Fallback: &SoonestResetSelector{},
		TTL:      time.Minute,
	})
	t.Cleanup(selector.Stop)

	options := func(sessionID string) cliproxyexecutor.Options {
		return cliproxyexecutor.Options{
			Headers:  http.Header{"Session_id": []string{sessionID}},
			Metadata: map[string]any{},
		}
	}
	pick := func(sessionID string) string {
		t.Helper()
		got, err := selector.Pick(context.Background(), "codex", "gpt-5", options(sessionID), auths)
		if err != nil {
			t.Fatalf("Pick(%s) error = %v", sessionID, err)
		}
		return got.ID
	}

	if got := pick("session-1"); got != "auth-b" {
		t.Fatalf("new session picked %q, want auth-b", got)
	}

	// The reset times swap. The existing binding must stay on auth-b, while a new
	// session follows the new ranking.
	authA.Quota.Signals = reset(time.Hour)
	authB.Quota.Signals = reset(6 * time.Hour)
	if got := pick("session-1"); got != "auth-b" {
		t.Fatalf("bound session moved to %q, want auth-b", got)
	}
	if got := pick("session-2"); got != "auth-a" {
		t.Fatalf("new session picked %q, want auth-a", got)
	}
}
