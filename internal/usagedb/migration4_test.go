package usagedb

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

// insertV3Row inserts one event plus its hourly rollup with the version-3
// column set, as a database built by migration 3 holds it. statusClass and
// failed are given, so the test can store the old buggy shape: failed rows
// with status_class 2xx.
func insertV3Row(t *testing.T, raw *sql.DB, id, status string, failed int64, ts, hour int64) {
	t.Helper()
	if _, err := raw.Exec(`INSERT INTO usage_events (
		request_id, trace_id, ts_ms, provider, model, alias, executor,
		auth_id, auth_index, auth_type, account_label, api_key_id,
		access_provider, session_id, stream, status_class, failed,
		fail_code, fail_summary, latency_ms, ttft_ms,
		input_tokens, output_tokens, reasoning_tokens,
		cache_read_tokens, cache_write_tokens, total_tokens,
		service_tier, reasoning_effort, cost_micro_usd, cost_source, price_id
	) VALUES (?, '', ?, 'codex', 'gpt-5', 'gpt-5', 'codex',
		'codex-1.json', '', 'oauth', '', 'key1',
		'', '', 0, ?, ?,
		'', '', 120, 30,
		10, 20, 0,
		0, 0, 30,
		'default', '', NULL, NULL, NULL)`, id, ts, status, failed); err != nil {
		t.Fatalf("insert v3 event: %v", err)
	}
	if _, err := raw.Exec(`INSERT INTO usage_hourly (
		hour_ms, provider, model, alias, auth_id, api_key_id, status_class, stream,
		requests, failures, input_tokens, output_tokens, reasoning_tokens,
		cache_read_tokens, cache_write_tokens, cost_micro_usd, unpriced_requests,
		latency_sum_ms, latency_max_ms, ttft_sum_ms, ttft_count, late_cost_micro_usd
	) VALUES (?, 'codex', 'gpt-5', 'gpt-5', 'codex-1.json', 'key1', ?, 0,
		1, ?, 10, 20, 0,
		0, 0, 0, 1,
		120, 120, 30, 1, 0)
	ON CONFLICT(hour_ms, provider, model, alias, auth_id, api_key_id, status_class, stream)
	DO UPDATE SET
		requests = usage_hourly.requests + 1,
		failures = usage_hourly.failures + excluded.failures,
		input_tokens = usage_hourly.input_tokens + 10,
		output_tokens = usage_hourly.output_tokens + 20,
		unpriced_requests = usage_hourly.unpriced_requests + 1,
		latency_sum_ms = usage_hourly.latency_sum_ms + 120,
		latency_max_ms = max(usage_hourly.latency_max_ms, 120),
		ttft_sum_ms = usage_hourly.ttft_sum_ms + 30,
		ttft_count = usage_hourly.ttft_count + 1`, hour, status, failed); err != nil {
		t.Fatalf("insert v3 hourly: %v", err)
	}
}

// TestMigration4FixesFailed2xx builds a version-3 database with the old buggy
// shape (two failed rows stored as 2xx), migrates, and checks the events fix
// plus the hourly rebuild. A failed 5xx row in another hour stays untouched.
func TestMigration4FixesFailed2xx(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.db")
	raw := openRawAtVersion(t, path, 3)
	hour := int64(1720000000000 / millisPerHour * millisPerHour)
	other := hour + millisPerHour
	insertV3Row(t, raw, "m4-ok", "2xx", 0, hour+1000, hour)
	insertV3Row(t, raw, "m4-f1", "2xx", 1, hour+2000, hour)
	insertV3Row(t, raw, "m4-f2", "2xx", 1, hour+3000, hour)
	insertV3Row(t, raw, "m4-f3", "5xx", 1, other+1000, other)
	if err := raw.Close(); err != nil {
		t.Fatalf("close raw: %v", err)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open after v3: %v", err)
	}
	defer func() { _ = store.Close() }()
	if SchemaVersion != 5 {
		t.Fatalf("SchemaVersion = %d, want 5", SchemaVersion)
	}

	var leaked int
	if err := store.readDB.QueryRow(
		`SELECT COUNT(*) FROM usage_events WHERE failed = 1 AND status_class = '2xx'`).Scan(&leaked); err != nil {
		t.Fatalf("count leaked: %v", err)
	}
	if leaked != 0 {
		t.Fatalf("failed 2xx events = %d, want 0", leaked)
	}
	var moved int
	if err := store.readDB.QueryRow(
		`SELECT COUNT(*) FROM usage_events WHERE failed = 1 AND status_class = 'error'`).Scan(&moved); err != nil {
		t.Fatalf("count moved: %v", err)
	}
	if moved != 2 {
		t.Fatalf("failed error events = %d, want 2", moved)
	}

	type bucket struct {
		requests int64
		failures int64
		input    int64
	}
	got := map[string]bucket{}
	rows, err := store.readDB.Query(
		`SELECT status_class, requests, failures, input_tokens FROM usage_hourly`)
	if err != nil {
		t.Fatalf("query hourly: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var total int64
	for rows.Next() {
		var class string
		var b bucket
		if err := rows.Scan(&class, &b.requests, &b.failures, &b.input); err != nil {
			t.Fatalf("scan hourly: %v", err)
		}
		if _, dup := got[class]; dup {
			t.Fatalf("duplicate hourly class %q", class)
		}
		got[class] = b
		total += b.requests
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("hourly rows: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("hourly classes = %v, want 2xx, error, 5xx", got)
	}
	if got["2xx"] != (bucket{requests: 1, failures: 0, input: 10}) {
		t.Fatalf("2xx hourly = %+v, want {1 0 10}", got["2xx"])
	}
	if got["error"] != (bucket{requests: 2, failures: 2, input: 20}) {
		t.Fatalf("error hourly = %+v, want {2 2 20}", got["error"])
	}
	if got["5xx"] != (bucket{requests: 1, failures: 1, input: 10}) {
		t.Fatalf("5xx hourly = %+v, want {1 1 10}", got["5xx"])
	}
	if total != 4 {
		t.Fatalf("hourly total = %d, want 4", total)
	}
}

// TestStatusClassHourlyConsistency writes failed records of every code shape
// through the real writer path and checks hourly failures equal the sum over
// non-2xx status classes.
func TestStatusClassHourlyConsistency(t *testing.T) {
	store := openTestStore(t)
	base := int64(1719997200000)
	mkFailed := func(id string, code int, ts int64) UsageRow {
		record := testRecord()
		record.RequestID = id
		record.RequestedAt = time.UnixMilli(ts)
		record.Failed = true
		record.Fail.StatusCode = code
		return RecordToRow(record)
	}
	ok := testRecord()
	ok.RequestID = "cons-ok"
	ok.RequestedAt = time.UnixMilli(base)
	rows := []UsageRow{
		RecordToRow(ok),
		mkFailed("cons-200", 200, base+1000),
		mkFailed("cons-404", 404, base+2000),
		mkFailed("cons-503", 503, base+3000),
		mkFailed("cons-0", 0, base+4000),
		mkFailed("cons-302", 302, base+5000),
	}
	if _, err := store.InsertBatch(rows); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	filter := mustFilter(t, "", base+48*60*60*1000)
	filter.FromMs = base - millisPerHour
	filter.ToMs = base + 48*60*60*1000
	counters, err := store.QueryCounters(filter)
	if err != nil {
		t.Fatalf("QueryCounters: %v", err)
	}
	if counters.Requests != 6 || counters.Failures != 5 {
		t.Fatalf("requests/failures = %d/%d, want 6/5", counters.Requests, counters.Failures)
	}
	items, total, err := store.QueryBreakdown(filter, "status_class", "requests", "desc", 20, 0)
	if err != nil {
		t.Fatalf("breakdown: %v", err)
	}
	if total != 5 {
		t.Fatalf("status classes = %d, want 5 (2xx, 4xx, 5xx, error, other)", total)
	}
	classes := map[string]int64{}
	for _, item := range items {
		classes[item.Key] = item.Requests
	}
	if classes["2xx"] != 1 || classes["4xx"] != 1 || classes["5xx"] != 1 ||
		classes["error"] != 2 || classes["other"] != 1 {
		t.Fatalf("status split = %v, want 2xx:1 4xx:1 5xx:1 error:2 other:1", classes)
	}
	var non2xx int64
	for key, requests := range classes {
		if key != "2xx" {
			non2xx += requests
		}
	}
	if non2xx != counters.Failures {
		t.Fatalf("non-2xx sum = %d, failures = %d", non2xx, counters.Failures)
	}
	failedFilter := filter
	failedFilter.Status = "failed"
	failed, err := store.QueryCounters(failedFilter)
	if err != nil {
		t.Fatalf("failed counters: %v", err)
	}
	if failed.Requests != counters.Failures || failed.Failures != counters.Failures {
		t.Fatalf("failed filter = %d/%d, want 5/5", failed.Requests, failed.Failures)
	}
}
