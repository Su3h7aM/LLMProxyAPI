package usagedb

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	coreusage "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/usage"
)

func testRecord() coreusage.Record {
	return coreusage.Record{
		RequestID:     "req-1",
		TraceID:       "trace-abcdef",
		Provider:      "codex",
		ExecutorType:  "codex",
		Model:         "gpt-5",
		Alias:         "work",
		APIKey:        "secret-client-key",
		AuthID:        "auth1",
		AuthIndex:     "0",
		AuthType:      "oauth",
		Source:        "alice@example.com",
		AccessProvider: "config-inline",
		SessionID:     "session-1",
		Stream:        true,
		RequestedAt:   time.UnixMilli(1720000000000),
		Latency:       120 * time.Millisecond,
		TTFT:          30 * time.Millisecond,
		ServiceTier:   "default",
		Detail: coreusage.Detail{
			InputTokens:         10,
			OutputTokens:        20,
			TotalTokens:         30,
			CacheReadTokens:     4,
			CacheCreationTokens: 6,
		},
	}
}

func TestAPIKeyIDHashed(t *testing.T) {
	row := RecordToRow(testRecord())
	sum := sha256.Sum256([]byte("secret-client-key"))
	want := hex.EncodeToString(sum[:])[:16]
	if row.APIKeyID != want {
		t.Fatalf("api_key_id = %q, want %q", row.APIKeyID, want)
	}
	if strings.Contains(row.FailSummary+row.AccountLabel, "secret-client-key") {
		t.Fatal("row contains the raw key")
	}
	empty := testRecord()
	empty.APIKey = ""
	if got := RecordToRow(empty).APIKeyID; got != "" {
		t.Fatalf("empty key api_key_id = %q, want empty", got)
	}
}

func TestSourceRawKeyNeverStored(t *testing.T) {
	record := testRecord()
	record.APIKey = "sk-live-raw-key-0123456789abcdef"
	record.Source = "sk-live-raw-key-0123456789abcdef"
	row := RecordToRow(record)
	dump := row.AccountLabel + "|" + row.APIKeyID + "|" + row.FailSummary + "|" +
		row.Provider + "|" + row.Model + "|" + row.Alias
	if strings.Contains(dump, "sk-live-raw-key-0123456789abcdef") {
		t.Fatalf("row contains the raw key: %q", dump)
	}
	if row.AccountLabel != "" {
		t.Fatalf("key source label = %q, want empty", row.AccountLabel)
	}
}

func TestMaskAccountLabelTable(t *testing.T) {
	cases := []struct {
		name   string
		source string
		key    string
		want   string
	}{
		{"empty", "", "", ""},
		{"email", "abigail@example.com", "", "ab***@example.com"},
		{"short email", "a@x.com", "", "a***@x.com"},
		{"plain label", "vertex-project-1", "", "vertex-project-1"},
		{"token-like", "sk-ant-abcdefghijklmnopqrstuvwxyz0123456789", "", "m:sk-a...6789"},
		{"raw key", "the-key", "the-key", ""},
		{"key inside text", "login with the-key ok", "the-key", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := maskAccountLabel(tc.source, tc.key); got != tc.want {
				t.Fatalf("mask = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFailSummaryCutAndRedacted(t *testing.T) {
	record := testRecord()
	record.Failed = true
	record.Fail = coreusage.Failure{StatusCode: 500, Body: "boom " + strings.Repeat("x", 5000)}
	row := RecordToRow(record)
	if len(row.FailSummary) > maxFailSummaryBytes {
		t.Fatalf("fail summary length = %d, want at most %d", len(row.FailSummary), maxFailSummaryBytes)
	}
	if !strings.HasPrefix(row.FailSummary, "boom ") {
		t.Fatal("fail summary lost its head")
	}

	withKey := testRecord()
	withKey.APIKey = "sk-secret-123"
	withKey.Failed = true
	withKey.Fail = coreusage.Failure{StatusCode: 429, Body: "denied for sk-secret-123 today"}
	keyed := RecordToRow(withKey)
	if strings.Contains(keyed.FailSummary, "sk-secret-123") {
		t.Fatalf("fail summary keeps the key: %q", keyed.FailSummary)
	}
	if !strings.Contains(keyed.FailSummary, "[redacted]") {
		t.Fatalf("fail summary missing redaction marker: %q", keyed.FailSummary)
	}
	if keyed.StatusClass != "4xx" {
		t.Fatalf("status class = %q, want 4xx", keyed.StatusClass)
	}
}

func TestStatusClassTable(t *testing.T) {
	cases := []struct {
		name string
		failed bool
		code int
		want string
	}{
		{"success", false, 0, "2xx"},
		{"failed 200", true, 200, "2xx"},
		{"failed 404", true, 404, "4xx"},
		{"failed 503", true, 503, "5xx"},
		{"failed zero", true, 0, "other"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := testRecord()
			record.Failed = tc.failed
			record.Fail = coreusage.Failure{StatusCode: tc.code}
			if got := RecordToRow(record).StatusClass; got != tc.want {
				t.Fatalf("status class = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRecordDefaults(t *testing.T) {
	record := testRecord()
	record.Alias = ""
	record.Provider = ""
	record.Model = ""
	record.ExecutorType = ""
	record.AuthType = ""
	record.ServiceTier = ""
	row := RecordToRow(record)
	if row.Alias != row.Model || row.Model != "unknown" {
		t.Fatalf("alias/model = %q/%q, want unknown/unknown", row.Alias, row.Model)
	}
	if row.Provider != "unknown" || row.Executor != "unknown" || row.AuthType != "unknown" {
		t.Fatalf("defaults = %q/%q/%q, want unknown", row.Provider, row.Executor, row.AuthType)
	}
	if row.ServiceTier != "default" {
		t.Fatalf("service tier = %q, want default", row.ServiceTier)
	}
	wantTs := int64(1720000000000)
	if row.TsMs != wantTs {
		t.Fatalf("ts = %d, want %d", row.TsMs, wantTs)
	}
	if wantHour := wantTs / millisPerHour * millisPerHour; row.HourMs != wantHour {
		t.Fatalf("hour = %d, want %d", row.HourMs, wantHour)
	}
}

func TestEffectiveServiceTierWins(t *testing.T) {
	record := testRecord()
	record.EffectiveServiceTier = "flex"
	if got := RecordToRow(record).ServiceTier; got != "flex" {
		t.Fatalf("service tier = %q, want flex", got)
	}
}
