package auth

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	cliproxyexecutor "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/executor"
)

// SoonestResetSelector prefers the credential whose quota window resets first,
// so quota that would expire unused at the reset is spent before quota that
// lasts longer.
//
// Reset times come from the passive quota snapshot (QuotaState.Signals) that
// upstream response headers already record. The selector makes no upstream
// calls. It ranks only the candidates it receives; cooldown and priority
// filtering run before Pick, as for the other selectors. Ranking order:
//
//  1. Credentials with a known future reset and no exhausted window.
//  2. Credentials without usable reset information, in ID order.
//  3. Credentials with an exhausted window that has not yet reset, in ID order.
type SoonestResetSelector struct{}

// Rank classes. A lower class sorts first.
const (
	soonestResetKnown = iota
	soonestResetUnknown
	soonestResetExhausted
)

// Window lengths of the Claude windows, which the headers do not report.
const (
	claudeFiveHourWindowMinutes = 300
	claudeSevenDayWindowMinutes = 10080
)

// soonestResetWindow is one quota window decoded from a quota snapshot.
type soonestResetWindow struct {
	minutes   int64
	resetAt   time.Time
	exhausted bool
}

// soonestResetRank orders one candidate. Ties in class and reset fall back to ID.
type soonestResetRank struct {
	class   int
	resetAt time.Time
	id      string
}

func (r soonestResetRank) before(other soonestResetRank) bool {
	if r.class != other.class {
		return r.class < other.class
	}
	if !r.resetAt.Equal(other.resetAt) {
		return r.resetAt.Before(other.resetAt)
	}
	return r.id < other.id
}

// Pick selects the available credential whose quota window resets soonest.
func (s *SoonestResetSelector) Pick(ctx context.Context, provider, model string, opts cliproxyexecutor.Options, auths []*Auth) (*Auth, error) {
	_ = opts
	now := time.Now()
	available, err := getSelectorAvailableAuths(ctx, auths, provider, model, now)
	if err != nil {
		return nil, err
	}
	available = preferCodexWebsocketAuths(ctx, provider, available)

	var picked *Auth
	var pickedRank soonestResetRank
	for _, candidate := range available {
		rank := soonestResetRankOf(candidate, now)
		if picked == nil || rank.before(pickedRank) {
			picked = candidate
			pickedRank = rank
		}
	}
	if picked == nil {
		return nil, &Error{Code: "auth_not_found", Message: "selector returned no auth"}
	}
	return picked, nil
}

// soonestResetRankOf classifies one credential from its quota snapshot.
//
// A window whose reset has passed is ignored: its snapshot describes quota that
// has already renewed. An exhausted window makes the credential rank last. Among
// the remaining windows, the longest one decides the reset time.
func soonestResetRankOf(auth *Auth, now time.Time) soonestResetRank {
	rank := soonestResetRank{class: soonestResetUnknown, id: auth.ID}

	var best *soonestResetWindow
	exhausted := false
	for _, window := range soonestResetWindows(auth.Quota) {
		if !window.resetAt.IsZero() && !window.resetAt.After(now) {
			continue
		}
		if window.exhausted {
			exhausted = true
			continue
		}
		if window.resetAt.IsZero() {
			continue
		}
		if best == nil || window.minutes > best.minutes {
			best = &window
		}
	}

	switch {
	case exhausted:
		rank.class = soonestResetExhausted
	case best != nil:
		rank.class = soonestResetKnown
		rank.resetAt = best.resetAt
	}
	return rank
}

// soonestResetWindows decodes the windows that a Claude or Codex snapshot reports.
// Windows that carry no reset and no exhaustion signal are returned with zero values.
func soonestResetWindows(quota QuotaState) []soonestResetWindow {
	signals := quota.Signals
	if len(signals) == 0 {
		return nil
	}
	return []soonestResetWindow{
		{
			minutes:   claudeFiveHourWindowMinutes,
			resetAt:   soonestResetUnixSignal(signals, "Anthropic-Ratelimit-Unified-5h-Reset"),
			exhausted: soonestResetRejected(signals, "Anthropic-Ratelimit-Unified-5h-Status"),
		},
		{
			minutes:   claudeSevenDayWindowMinutes,
			resetAt:   soonestResetUnixSignal(signals, "Anthropic-Ratelimit-Unified-7d-Reset"),
			exhausted: soonestResetRejected(signals, "Anthropic-Ratelimit-Unified-7d-Status"),
		},
		codexSoonestResetWindow(quota, "Primary"),
		codexSoonestResetWindow(quota, "Secondary"),
	}
}

// codexSoonestResetWindow decodes one Codex window. The absolute reset header is
// preferred. The relative header counts from the time of the observation.
func codexSoonestResetWindow(quota QuotaState, name string) soonestResetWindow {
	prefix := "X-Codex-" + name + "-"
	signals := quota.Signals

	resetAt := soonestResetUnixSignal(signals, prefix+"Reset-At")
	if resetAt.IsZero() && !quota.ObservedAt.IsZero() {
		if after, ok := soonestResetNumberSignal(signals, prefix+"Reset-After-Seconds"); ok {
			resetAt = quota.ObservedAt.Add(time.Duration(after * float64(time.Second)))
		}
	}
	minutes, _ := soonestResetNumberSignal(signals, prefix+"Window-Minutes")
	usedPercent, _ := soonestResetNumberSignal(signals, prefix+"Used-Percent")
	return soonestResetWindow{
		minutes:   int64(minutes),
		resetAt:   resetAt,
		exhausted: usedPercent >= 100,
	}
}

// soonestResetUnixSignal parses a Unix timestamp in seconds. It returns the zero time when
// the signal is absent or invalid.
func soonestResetUnixSignal(signals map[string]string, name string) time.Time {
	seconds, ok := soonestResetNumberSignal(signals, name)
	if !ok {
		return time.Time{}
	}
	return time.Unix(int64(seconds), 0)
}

// soonestResetNumberSignal parses a finite, non-negative number from a signal value.
func soonestResetNumberSignal(signals map[string]string, name string) (float64, bool) {
	raw := strings.TrimSpace(signals[name])
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0, false
	}
	return value, true
}

func soonestResetRejected(signals map[string]string, name string) bool {
	return strings.EqualFold(strings.TrimSpace(signals[name]), "rejected")
}
