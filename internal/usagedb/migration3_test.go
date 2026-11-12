package usagedb

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

// openRawAtVersion opens a database file and migrates it to target only, so
// tests can build an older database with rows in it.
func openRawAtVersion(t *testing.T, path string, target int) *sql.DB {
	t.Helper()
	raw, err := sql.Open("sqlite", dataSourceName(path, true))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := migrateTo(raw, target); err != nil {
		_ = raw.Close()
		t.Fatalf("migrateTo %d: %v", target, err)
	}
	return raw
}

// simulateLateFillV2 marks events late and fixes the hourly rollup the way
// the version-2 fill code did (cost up, unpriced down, no late column).
func simulateLateFillV2(t *testing.T, raw *sql.DB, requestIDs []string, cost int64) {
	t.Helper()
	for _, id := range requestIDs {
		if _, err := raw.Exec(`UPDATE usage_events SET cost_micro_usd = ?, cost_source = 'price-book-late' WHERE request_id = ?`,
			cost, id); err != nil {
			t.Fatalf("mark late: %v", err)
		}
		if _, err := raw.Exec(`UPDATE usage_hourly SET cost_micro_usd = cost_micro_usd + ?,
			unpriced_requests = CASE WHEN unpriced_requests > 0 THEN unpriced_requests - 1 ELSE 0 END
			WHERE (hour_ms, provider, model, alias, auth_id, api_key_id, status_class, stream) = (
				SELECT ts_ms / 3600000 * 3600000, provider, model, alias, auth_id, api_key_id, status_class, stream
				FROM usage_events WHERE request_id = ?)`, cost, id); err != nil {
			t.Fatalf("fix hourly: %v", err)
		}
	}
}

// insertV2Row inserts one event plus its hourly rollup with the version-2
// column set, as a database built by migration 2 holds them.
func insertV2Row(t *testing.T, raw *sql.DB, id, model string, ts, hour int64) {
	t.Helper()
	if _, err := raw.Exec(`INSERT INTO usage_events (
		request_id, trace_id, ts_ms, provider, model, alias, executor,
		auth_id, auth_index, auth_type, account_label, api_key_id,
		access_provider, session_id, stream, status_class, failed,
		fail_code, fail_summary, latency_ms, ttft_ms,
		input_tokens, output_tokens, reasoning_tokens,
		cache_read_tokens, cache_write_tokens, total_tokens,
		service_tier, reasoning_effort, cost_micro_usd, cost_source, price_id
	) VALUES (?, '', ?, 'codex', ?, ?, 'codex',
		'codex-1.json', '', '', '', 'key1',
		'', '', 0, '2xx', 0,
		'', '', 120, 30,
		10, 20, 0,
		0, 0, 30,
		'default', '', NULL, NULL, NULL)`, id, ts, model, model); err != nil {
		t.Fatalf("insert v2 event: %v", err)
	}
	if _, err := raw.Exec(`INSERT INTO usage_hourly (
		hour_ms, provider, model, alias, auth_id, api_key_id, status_class, stream,
		requests, failures, input_tokens, output_tokens, reasoning_tokens,
		cache_read_tokens, cache_write_tokens, cost_micro_usd, unpriced_requests,
		latency_sum_ms, latency_max_ms, ttft_sum_ms, ttft_count
	) VALUES (?, 'codex', ?, ?, 'codex-1.json', 'key1', '2xx', 0,
		1, 0, 10, 20, 0,
		0, 0, 0, 1,
		120, 120, 30, 1)
	ON CONFLICT(hour_ms, provider, model, alias, auth_id, api_key_id, status_class, stream)
	DO UPDATE SET
		requests = usage_hourly.requests + 1,
		input_tokens = usage_hourly.input_tokens + 10,
		output_tokens = usage_hourly.output_tokens + 20,
		unpriced_requests = usage_hourly.unpriced_requests + 1,
		latency_sum_ms = usage_hourly.latency_sum_ms + 120,
		latency_max_ms = max(usage_hourly.latency_max_ms, 120),
		ttft_sum_ms = usage_hourly.ttft_sum_ms + 30,
		ttft_count = usage_hourly.ttft_count + 1`, hour, model, model); err != nil {
		t.Fatalf("insert v2 hourly: %v", err)
	}
}

func explainPlan(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	rows, err := db.Query("EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var parts []string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatalf("scan plan: %v", err)
		}
		parts = append(parts, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("plan rows: %v", err)
	}
	return strings.Join(parts, " | ")
}

func TestMigration3BackfillsLateCost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.db")
	raw := openRawAtVersion(t, path, 2)
	hour := int64(1720000000000 / millisPerHour * millisPerHour)
	insertV2Row(t, raw, "m3-e1", "gpt-5", hour+1000, hour)
	insertV2Row(t, raw, "m3-e2", "gpt-5", hour+2000, hour)
	insertV2Row(t, raw, "m3-e3", "gpt-6", hour+3000, hour)
	// Version-2 late fill priced e2 and e3 late; e1 stays unpriced.
	simulateLateFillV2(t, raw, []string{"m3-e2"}, 500)
	simulateLateFillV2(t, raw, []string{"m3-e3"}, 700)
	if err := raw.Close(); err != nil {
		t.Fatalf("close raw: %v", err)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open after v2: %v", err)
	}
	defer func() { _ = store.Close() }()
	if SchemaVersion != 4 {
		t.Fatalf("SchemaVersion = %d, want 4", SchemaVersion)
	}

	type hourly struct {
		model    string
		cost     int64
		late     int64
		unpriced int64
	}
	got := map[string]hourly{}
	hrows, err := store.readDB.Query(`SELECT model, cost_micro_usd, late_cost_micro_usd, unpriced_requests FROM usage_hourly`)
	if err != nil {
		t.Fatalf("query hourly: %v", err)
	}
	defer func() { _ = hrows.Close() }()
	for hrows.Next() {
		var h hourly
		if err := hrows.Scan(&h.model, &h.cost, &h.late, &h.unpriced); err != nil {
			t.Fatalf("scan hourly: %v", err)
		}
		got[h.model] = h
	}
	if err := hrows.Err(); err != nil {
		t.Fatalf("hourly rows: %v", err)
	}
	if got["gpt-5"].late != 500 || got["gpt-5"].cost != 500 || got["gpt-5"].unpriced != 1 {
		t.Fatalf("gpt-5 hourly = %+v, want late 500 cost 500 unpriced 1", got["gpt-5"])
	}
	if got["gpt-6"].late != 700 || got["gpt-6"].cost != 700 || got["gpt-6"].unpriced != 0 {
		t.Fatalf("gpt-6 hourly = %+v, want late 700 cost 700 unpriced 0", got["gpt-6"])
	}

	// Key labels table exists and round-trips.
	kept, err := store.SetKeyLabel("key1", "harness", 1720000000000)
	if err != nil || !kept {
		t.Fatalf("SetKeyLabel = %v, %v", kept, err)
	}
	if label := store.KeyLabelOf("key1"); label != "harness" {
		t.Fatalf("KeyLabelOf = %q, want harness", label)
	}

	// Each new index is used by its route query.
	from, to := hour, hour+millisPerHour
	plan := explainPlan(t, store.readDB,
		`SELECT id FROM usage_events WHERE ts_ms >= ? AND ts_ms < ? AND provider = ? ORDER BY ts_ms, id LIMIT 50`,
		from, to, "codex")
	if !strings.Contains(plan, "idx_usage_events_provider_ts") {
		t.Fatalf("provider query plan does not use provider index: %s", plan)
	}
	plan = explainPlan(t, store.readDB,
		`SELECT id FROM usage_events WHERE ts_ms >= ? AND ts_ms < ? AND failed = ? ORDER BY ts_ms, id LIMIT 50`,
		from, to, 1)
	if !strings.Contains(plan, "idx_usage_events_failed_ts") {
		t.Fatalf("failed query plan does not use failed index: %s", plan)
	}
	plan = explainPlan(t, store.readDB,
		`SELECT id FROM usage_events WHERE ts_ms >= ? AND ts_ms < ? AND status_class = ? ORDER BY ts_ms, id LIMIT 50`,
		from, to, "4xx")
	if !strings.Contains(plan, "idx_usage_events_status_ts") {
		t.Fatalf("status query plan does not use status index: %s", plan)
	}
}
