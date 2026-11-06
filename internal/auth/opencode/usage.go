// Package opencode carries the OpenCode Go subscription integration.
//
// Sketch 1 (core fetcher): FetchUsage reads the three allowance windows
// (5 hours, 7 days, month) at the Go usage endpoint with the workspace
// member's Go key. The management quota/fetch route converts Usage to the
// same normalized buckets Codex/Anthropic credentials return.
package opencode

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

// ProviderName is the auth provider key for OpenCode Go credentials,
// beside "codex" and "claude".
const ProviderName = "opencode-go"

// UsageEndpoint is the fixed route that tells a Go key its windows.
// Reference: magpie planquota.go readOpenCodeGo; GET /zen/go/v1/usage.
const UsageEndpoint = "https://opencode.ai/zen/go/v1/usage"

// Window is one allowance window: rolling (5 hours), weekly, monthly.
type Window struct {
	Name        string
	UsedPercent float64
	ResetsAt    time.Time
	RateLimited bool
}

// Usage is the plan name with its windows.
type Usage struct {
	Plan    string
	Windows []Window
}

// FetchUsage tells key its windows. Stub: body lands after sketch choice.
// FetchUsage asks UsageEndpoint for key's rolling (5 hours), weekly and
// monthly windows. A 401 means the key is bad; a 403 means the key's
// workspace holds no Go subscription.
func FetchUsage(ctx context.Context, key string) (Usage, error) {
	return fetchUsage(ctx, key, UsageEndpoint)
}

func fetchUsage(ctx context.Context, key, endpoint string) (Usage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Usage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en")
	// Go's usage is asked at its gateway, which routes nothing that names
	// no session. Reference: magpie planBody/OpenCodeSession.
	session := newSessionID(time.Now())
	req.Header.Set("x-opencode-session", session)
	req.Header.Set("x-opencode-session-id", session)
	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return Usage{}, err
	}
	defer func() { _ = res.Body.Close }()
	body, err := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	if err != nil {
		return Usage{}, err
	}
	switch res.StatusCode {
	case http.StatusUnauthorized:
		return Usage{}, fmt.Errorf("opencode did not take this key (401)")
	case http.StatusForbidden:
		return Usage{}, fmt.Errorf("no OpenCode Go subscription on this key's workspace (403)")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Usage{}, fmt.Errorf("opencode usage returned status %d: %s", res.StatusCode, truncate(body, 200))
	}
	return parseUsage(body)
}

// parseUsage reads {"usage":{"rolling":{"status","percent","resetsAt"},
// "weekly":{...},"monthly":{...}}}. Percent is used allowance; a
// rate-limited window reads as spent.
func parseUsage(body []byte) (Usage, error) {
	type window struct {
		Status   string   `json:"status"`
		Percent  *float64 `json:"percent"`
		ResetsAt string   `json:"resetsAt"`
	}
	var reply struct {
		Usage map[string]window `json:"usage"`
	}
	if err := json.Unmarshal(body, &reply); err != nil {
		return Usage{}, err
	}
	out := Usage{Plan: "OpenCode Go"}
	for _, def := range []struct {
		key  string
		name string
	}{{"rolling", "5 hours"}, {"weekly", "7 days"}, {"monthly", "Month"}} {
		w, ok := reply.Usage[def.key]
		if !ok || w.Percent == nil {
			continue
		}
		q := Window{Name: def.name, UsedPercent: *w.Percent}
		if w.Status == "rate-limited" {
			q.RateLimited = true
			q.UsedPercent = 100
		}
		// A 0% window carries a reset time nothing happens at; leave it out.
		if t, err := time.Parse(time.RFC3339, w.ResetsAt); err == nil && q.UsedPercent > 0 {
			q.ResetsAt = t
		}
		out.Windows = append(out.Windows, q)
	}
	if len(out.Windows) == 0 {
		return Usage{}, fmt.Errorf("opencode usage reply holds no rolling, weekly or monthly window")
	}
	return out, nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n])
}

var sessionCounter atomic.Uint64
var sessionLast atomic.Int64

const sessionBase62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// newSessionID lays a session id out as OpenCode does: ses_, 12 hex digits
// of timestamp plus a per-millisecond count, then 14 random base62 chars.
// Reference: magpie openCodeNewID("ses", true, now).
func newSessionID(now time.Time) string {
	ms := now.UnixMilli()
	last := sessionLast.Load()
	if ms != last {
		if sessionLast.CompareAndSwap(last, ms) {
			sessionCounter.Store(0)
		}
	}
	current := ^(uint64(ms)*0x1000 + sessionCounter.Add(1))
	var raw [14]byte
	if _, err := rand.Read(raw[:]); err != nil {
		// Deterministic fallback without secret material: session ids only
		// need uniqueness per request, not entropy.
		seed := uint64(now.UnixNano()) + sessionCounter.Add(0x9E3779B9)
		for i := range raw {
			seed = seed*6364136223846793005 + 1442695040888963407
			raw[i] = byte(seed >> 33)
		}
	}
	id := make([]byte, 0, 4+12+14)
	id = append(id, "ses_"...)
	const hexdigits = "0123456789abcdef"
	for i := 5; i >= 0; i-- {
		id = append(id, hexdigits[byte(current>>(i*8))>>4], hexdigits[byte(current>>(i*8))&0x0f])
	}
	for _, c := range raw {
		id = append(id, sessionBase62[int(c)%len(sessionBase62)])
	}
	return string(id)
}
