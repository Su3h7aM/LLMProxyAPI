package usagedb

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
	"time"

	coreusage "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/usage"
)

func testRecord() coreusage.Record {
	return coreusage.Record{
		RequestID:      "req-1",
		TraceID:        "trace-abcdef",
		Provider:       "codex",
		ExecutorType:   "codex",
		Model:          "gpt-5",
		Alias:          "work",
		APIKey:         "secret-client-key",
		AuthID:         "auth1",
		AuthIndex:      "0",
		AuthType:       "oauth",
		Source:         "alice@example.com",
		AccessProvider: "config-inline",
		SessionID:      "session-1",
		Stream:         true,
		RequestedAt:    time.UnixMilli(1720000000000),
		Latency:        120 * time.Millisecond,
		TTFT:           30 * time.Millisecond,
		ServiceTier:    "default",
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
		{"plain label", "vertex-project-1", "", ""},
		{"project with colon", "config:codex[0]", "", ""},
		{"token-like", "sk-ant-abcdefghijklmnopqrstuvwxyz0123456789", "", ""},
		{"short key", "stub-upstream-key", "", ""},
		{"raw key", "the-key", "the-key", ""},
		{"key inside text", "login with the-key ok", "the-key", ""},
		{"email with spaces", "a b@example.com", "", ""},
		{"bearer", "Bearer abc123", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := maskAccountLabel(tc.source, tc.key); got != tc.want {
				t.Fatalf("mask = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSafeFieldTable(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{"empty", "", ""},
		{"plain", "session-1", "session-1"},
		{"dots colons", "a.b_c-d:e", "a.b_c-d:e"},
		{"spaces", "Bearer abc123", ""},
		{"slash", "/tmp/x", ""},
		{"query", "https://x/?token=abc", ""},
		{"semicolon", "a;b", ""},
		{"jwt dots long", strings.Repeat("a", 30) + "." + strings.Repeat("b", 30) + "." + strings.Repeat("c", 30), ""},
		{"too long", strings.Repeat("a", 65), ""},
		{"max length", strings.Repeat("a", 64), strings.Repeat("a", 64)},
		{"oauth", "oauth", "oauth"},
		{"email", "a@x.com", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := safeField(tc.value); got != tc.want {
				t.Fatalf("safeField = %q, want %q", got, tc.want)
			}
		})
	}
	if got := safeOrUnknown(""); got != "unknown" {
		t.Fatalf("safeOrUnknown empty = %q, want unknown", got)
	}
	if got := safeOrUnknown("Bearer x"); got != "" {
		t.Fatalf("safeOrUnknown hostile = %q, want empty", got)
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
		name   string
		failed bool
		code   int
		want   string
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

// hostileSecrets is the issue step 1 table: every input here must never
// appear in any stored column (except as a hash in api_key_id).
func hostileSecrets() []string {
	jwt := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9." +
		strings.Repeat("YWIxMjM0NTY3ODkw", 12) +
		".c2lnbmF0dXJlLWFiYzEyMzQ1Njc4OTA"
	return []string{
		"sk-live-raw-key-0123456789abcdef",
		"abcd1234",
		"abcdefghij0123456789",
		"abcdefghijklmnopqrstuvwxyz0123456789ABCD",
		"sk-ant-api03-xyzABCDEF1234567890abcdefgh",
		"AIzaSyD-EXAMPLEKEY0123456789abcdefg",
		"Bearer eyJhbGciOiJIUzI1NiJ9.cGF5bG9hZA.c2ln",
		jwt,
		"https://api.example.com/v1?api_key=SECRET1234567890&x=1",
		"/home/user/.config/keys/openai.json",
	}
}

// rowDump joins every stored string column except api_key_id (a hash) for
// leak scans.
func rowDump(row UsageRow) string {
	return strings.Join([]string{
		row.RequestID, row.TraceID, row.Provider, row.Model, row.Alias,
		row.Executor, row.AuthID, row.AuthIndex, row.AuthType,
		row.AccountLabel, row.AccessProvider, row.SessionID,
		row.StatusClass, row.FailCode, row.FailSummary,
		row.ServiceTier, row.ReasoningEffort,
	}, "|")
}

func TestHostileSecretsNeverStored(t *testing.T) {
	const clientKey = "raw-client-key-AAAABBBBCCCCDDDD"
	for _, secret := range hostileSecrets() {
		record := testRecord()
		record.APIKey = clientKey
		record.Source = secret
		record.SessionID = "session-1"
		record.AccessProvider = "config-inline"
		record.ReasoningEffort = ""
		record.Fail.Code = ""
		record.ExecutorType = "codex"
		record.AuthType = "oauth"
		record.ServiceTier = "default"
		row := RecordToRow(record)
		if row.AccountLabel != "" {
			t.Fatalf("secret %q: account_label = %q, want empty", secret, row.AccountLabel)
		}
		if dump := rowDump(row); strings.Contains(dump, secret) || strings.Contains(dump, clientKey) {
			t.Fatalf("secret %q: row holds a secret: %q", secret, dump)
		}
		if row.APIKeyID == "" || row.APIKeyID == clientKey {
			t.Fatalf("secret %q: api_key_id = %q, want hash", secret, row.APIKeyID)
		}
		// The raw client key as source is the worst case: still empty.
		keyed := testRecord()
		keyed.APIKey = secret
		keyed.Source = secret
		keyedRow := RecordToRow(keyed)
		if keyedRow.AccountLabel != "" {
			t.Fatalf("secret %q as key: account_label = %q, want empty", secret, keyedRow.AccountLabel)
		}
		if dump := rowDump(keyedRow); strings.Contains(dump, secret) {
			t.Fatalf("secret %q as key: row holds it: %q", secret, dump)
		}
		// Shape-hostile values in every other free-text column stay empty.
		// Safe-shaped secrets may stay (the rule allows them); anything with
		// an unsafe character or over 64 characters must not.
		shaped := testRecord()
		shaped.APIKey = clientKey
		shaped.SessionID = secret
		shaped.AccessProvider = secret
		shaped.ReasoningEffort = secret
		shaped.Fail.Code = secret
		shaped.ExecutorType = secret
		shaped.AuthType = secret
		srow := RecordToRow(shaped)
		kept := []string{
			srow.SessionID, srow.AccessProvider, srow.ReasoningEffort,
			srow.FailCode, srow.Executor, srow.AuthType,
		}
		for _, value := range kept {
			if value == "" || value == "unknown" {
				continue
			}
			if !isSafeShape(value) {
				t.Fatalf("secret %q: unsafe value kept: %q", secret, value)
			}
		}
		if !isSafeShape(secret) {
			for _, value := range kept {
				if value != "" && value != "unknown" {
					t.Fatalf("unsafe secret %q: kept %q", secret, value)
				}
			}
		}
	}
}

// isSafeShape mirrors safeField for the test: only to assert the negative
// (no unsafe value is ever kept).
func isSafeShape(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '_', r == '-', r == ':':
		default:
			return false
		}
	}
	return true
}

// TestHostileSecretsDBScan inserts one row per hostile secret and scans every
// text column of usage_events: no secret may survive.
func TestHostileSecretsDBScan(t *testing.T) {
	store := openTestStore(t)
	const clientKey = "raw-client-key-AAAABBBBCCCCDDDD"
	var rows []UsageRow
	for i, secret := range hostileSecrets() {
		record := testRecord()
		record.RequestID = "hostile-req-" + strconv.Itoa(i)
		record.RequestedAt = time.UnixMilli(1720000000000 + int64(i)*1000)
		record.APIKey = clientKey
		record.Source = secret
		record.Fail = coreusage.Failure{StatusCode: 500, Body: "upstream error"}
		rows = append(rows, RecordToRow(record))
	}
	if _, err := store.InsertBatch(rows); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	textCols := []string{"request_id", "trace_id", "provider", "model", "alias",
		"executor", "auth_id", "auth_index", "auth_type", "account_label",
		"api_key_id", "access_provider", "session_id", "status_class",
		"fail_code", "fail_summary", "service_tier", "reasoning_effort"}
	query := `SELECT ` + strings.Join(textCols, ", ") + ` FROM usage_events`
	dbRows, err := store.readDB.Query(query)
	if err != nil {
		t.Fatalf("scan query: %v", err)
	}
	defer func() { _ = dbRows.Close() }()
	cells := make([]any, len(textCols))
	ptrs := make([]any, len(textCols))
	for i := range cells {
		ptrs[i] = &cells[i]
	}
	var found int
	for dbRows.Next() {
		found++
		if err := dbRows.Scan(ptrs...); err != nil {
			t.Fatalf("scan: %v", err)
		}
		for _, cell := range cells {
			var text string
			switch value := cell.(type) {
			case string:
				text = value
			case []byte:
				text = string(value)
			default:
				continue
			}
			for _, secret := range hostileSecrets() {
				if strings.Contains(text, secret) {
					t.Fatalf("DB holds secret %q in column value %q", secret, text)
				}
			}
			if strings.Contains(text, clientKey) {
				t.Fatalf("DB holds the raw client key in %q", text)
			}
		}
	}
	if err := dbRows.Err(); err != nil {
		t.Fatalf("scan rows: %v", err)
	}
	if found != len(hostileSecrets()) {
		t.Fatalf("scanned rows = %d, want %d", found, len(hostileSecrets()))
	}
}
