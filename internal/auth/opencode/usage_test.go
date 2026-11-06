package opencode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseUsageWindows(t *testing.T) {
	body := []byte(`{"usage":{` +
		`"rolling":{"status":"ok","percent":37,"resetsAt":"2026-08-26T14:12:03.000Z"},` +
		`"weekly":{"status":"ok","percent":0,"resetsAt":"2026-09-01T00:00:00.000Z"},` +
		`"monthly":{"status":"rate-limited","percent":99,"resetsAt":"2026-09-01T01:26:45.000Z"}}}`)
	u, err := parseUsage(body)
	if err != nil {
		t.Fatalf("parseUsage: %v", err)
	}
	if u.Plan != "OpenCode Go" {
		t.Fatalf("plan = %q", u.Plan)
	}
	if len(u.Windows) != 3 {
		t.Fatalf("windows = %d, want 3", len(u.Windows))
	}
	if u.Windows[0].Name != "5 hours" || u.Windows[0].UsedPercent != 37 {
		t.Fatalf("rolling = %+v", u.Windows[0])
	}
	if u.Windows[0].ResetsAt.IsZero() {
		t.Fatal("rolling resetsAt missing")
	}
	if u.Windows[1].Name != "7 days" || !u.Windows[1].ResetsAt.IsZero() {
		t.Fatalf("weekly 0%% must skip resetsAt: %+v", u.Windows[1])
	}
	if u.Windows[2].Name != "Month" || !u.Windows[2].RateLimited || u.Windows[2].UsedPercent != 100 {
		t.Fatalf("monthly rate-limited must read spent: %+v", u.Windows[2])
	}
}

func TestParseUsageNoWindows(t *testing.T) {
	if _, err := parseUsage([]byte(`{"usage":{}}`)); err == nil {
		t.Fatal("want error for empty usage")
	}
	if _, err := parseUsage([]byte(`not json`)); err == nil {
		t.Fatal("want error for bad json")
	}
}

func TestFetchUsageStatuses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("missing bearer key")
		}
		if r.Header.Get("Accept-Language") == "" {
			t.Error("missing Accept-Language")
		}
		session := r.Header.Get("x-opencode-session")
		if session == "" || r.Header.Get("x-opencode-session-id") != session {
			t.Error("missing session attribution headers")
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"subscription required"}`))
	}))
	defer srv.Close()
	if _, err := fetchUsage(context.Background(), "bad", srv.URL); err == nil {
		t.Fatal("want error for 403")
	}
}

func TestNewSessionIDShape(t *testing.T) {
	id := newSessionID(time.Now())
	if !strings.HasPrefix(id, "ses_") || len(id) != 4+12+14 {
		t.Fatalf("bad session id shape: %q", id)
	}
	if id != newSessionID(time.Now()) {
		return
	}
	t.Fatal("session ids must differ")
}
