package usagedb

import (
	"path/filepath"
	"testing"
	"time"
)

func testRow(requestID string, tsMs int64) UsageRow {
	return UsageRow{
		RequestID:    requestID,
		TraceID:      "trace1",
		TsMs:         tsMs,
		HourMs:       tsMs / millisPerHour * millisPerHour,
		Provider:     "codex",
		Model:        "gpt-5",
		Alias:        "gpt-5",
		Executor:     "codex",
		AuthID:       "auth1",
		StatusClass:  "2xx",
		LatencyMs:    120,
		TTFTMs:       30,
		InputTokens:  10,
		OutputTokens: 20,
		TotalTokens:  30,
		ServiceTier:  "default",
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "usage.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func queryHourly(t *testing.T, store *Store) (requests, failures, input, unpriced, latencySum, latencyMax, ttftSum, ttftCount int64) {
	t.Helper()
	err := store.readDB.QueryRow(`SELECT requests, failures, input_tokens, unpriced_requests,
		latency_sum_ms, latency_max_ms, ttft_sum_ms, ttft_count FROM usage_hourly`).Scan(
		&requests, &failures, &input, &unpriced, &latencySum, &latencyMax, &ttftSum, &ttftCount)
	if err != nil {
		t.Fatalf("query hourly: %v", err)
	}
	return requests, failures, input, unpriced, latencySum, latencyMax, ttftSum, ttftCount
}

func TestOpenMigrateReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = reopened.Close() }()
	var version int
	if err := reopened.writeDB.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatalf("schema version: %v", err)
	}
	if version != SchemaVersion {
		t.Fatalf("schema version = %d, want %d", version, SchemaVersion)
	}
	var journal string
	if err := reopened.writeDB.QueryRow(`PRAGMA journal_mode`).Scan(&journal); err != nil {
		t.Fatalf("journal mode: %v", err)
	}
	if journal != "wal" {
		t.Fatalf("journal mode = %q, want wal", journal)
	}
}

func TestOpenRefusesNewerSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := store.writeDB.Exec(`INSERT INTO schema_migrations(version, applied_at_ms) VALUES (?, ?)`,
		SchemaVersion+1, unixMilliNow()); err != nil {
		t.Fatalf("fake newer version: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := Open(path); err == nil {
		t.Fatal("Open with newer schema succeeded, want refusal")
	}
}

func TestInsertDedupSingleCount(t *testing.T) {
	store := openTestStore(t)
	now := unixMilliNow()
	row := testRow("req-1", now)
	inserted, err := store.InsertBatch([]UsageRow{row, row})
	if err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	if inserted != 1 {
		t.Fatalf("inserted = %d, want 1", inserted)
	}
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Events != 1 {
		t.Fatalf("events = %d, want 1", stats.Events)
	}
	requests, _, _, _, _, _, _, _ := queryHourly(t, store)
	if requests != 1 {
		t.Fatalf("hourly requests = %d, want 1 (duplicate must not count twice)", requests)
	}
}

func TestHourlySums(t *testing.T) {
	store := openTestStore(t)
	now := unixMilliNow()
	first := testRow("req-1", now)
	second := testRow("req-2", now)
	second.Failed = true
	second.StatusClass = "5xx"
	second.InputTokens = 5
	second.LatencyMs = 200
	second.TTFTMs = 0
	// Different status class lands in another hourly bucket.
	if _, err := store.InsertBatch([]UsageRow{first}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	if _, err := store.InsertBatch([]UsageRow{second}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	var requests, failures, input int64
	err := store.readDB.QueryRow(`SELECT SUM(requests), SUM(failures), SUM(input_tokens) FROM usage_hourly`).Scan(
		&requests, &failures, &input)
	if err != nil {
		t.Fatalf("sum hourly: %v", err)
	}
	if requests != 2 || failures != 1 || input != 15 {
		t.Fatalf("hourly sums = (%d, %d, %d), want (2, 1, 15)", requests, failures, input)
	}
	var latencyMax, ttftCount int64
	if err := store.readDB.QueryRow(`SELECT MAX(latency_max_ms), SUM(ttft_count) FROM usage_hourly`).Scan(
		&latencyMax, &ttftCount); err != nil {
		t.Fatalf("hourly max: %v", err)
	}
	if latencyMax != 200 {
		t.Fatalf("latency max = %d, want 200", latencyMax)
	}
	if ttftCount != 1 {
		t.Fatalf("ttft count = %d, want 1 (zero TTFT is unmeasured, not zero)", ttftCount)
	}
	// NULL cost counts as unpriced, not zero cost.
	var unpriced int64
	if err := store.readDB.QueryRow(`SELECT SUM(unpriced_requests) FROM usage_hourly`).Scan(&unpriced); err != nil {
		t.Fatalf("unpriced: %v", err)
	}
	if unpriced != 2 {
		t.Fatalf("unpriced = %d, want 2", unpriced)
	}
}

func TestPruneKeepsRollups(t *testing.T) {
	store := openTestStore(t)
	now := unixMilliNow()
	old := testRow("req-old", now-int64(100*24*time.Hour/time.Millisecond))
	fresh := testRow("req-new", now)
	if _, err := store.InsertBatch([]UsageRow{old, fresh}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	cutoff := now - int64(90*24*time.Hour/time.Millisecond)
	deleted, err := store.PruneBefore(cutoff, 5000)
	if err != nil {
		t.Fatalf("PruneBefore: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Events != 1 {
		t.Fatalf("events = %d, want 1", stats.Events)
	}
	var buckets int64
	if err := store.readDB.QueryRow(`SELECT COUNT(*) FROM usage_hourly`).Scan(&buckets); err != nil {
		t.Fatalf("hourly count: %v", err)
	}
	if buckets != 2 {
		t.Fatalf("hourly buckets = %d, want 2 (rollups stay after prune)", buckets)
	}
}

func TestStatsEmpty(t *testing.T) {
	store := openTestStore(t)
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Events != 0 || stats.OldestTsMs != 0 || stats.NewestTsMs != 0 {
		t.Fatalf("empty stats = %+v, want zeros", stats)
	}
}
