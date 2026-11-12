package usagedb

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

// insertV4Hourly writes one hourly rollup row with the version-4 column set,
// as a database at migration 4 holds it.
func insertV4Hourly(t *testing.T, raw *sql.DB, hour int64, requests, failures int64) {
	t.Helper()
	if _, err := raw.Exec(`INSERT INTO usage_hourly (
		hour_ms, provider, model, alias, auth_id, api_key_id, status_class, stream,
		requests, failures, input_tokens, output_tokens, reasoning_tokens,
		cache_read_tokens, cache_write_tokens, cost_micro_usd, unpriced_requests,
		latency_sum_ms, latency_max_ms, ttft_sum_ms, ttft_count, late_cost_micro_usd
	) VALUES (?, 'codex', 'gpt-5', 'gpt-5', 'codex-1.json', 'key1', '2xx', 0,
		?, ?, 10 * ?, 20 * ?, 0,
		0, 0, 0, ?,
		120, 120, 30, 1, 0)`, hour, requests, failures, requests, requests, requests); err != nil {
		t.Fatalf("insert v4 hourly: %v", err)
	}
}

// TestMigration5BackfillsDaily builds a version-4 database with hourly rows
// across two UTC days, migrates, and checks the day rollup math.
func TestMigration5BackfillsDaily(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.db")
	raw := openRawAtVersion(t, path, 4)
	day := int64(1720000000000 / millisPerDay * millisPerDay)
	insertV4Hourly(t, raw, day+1000*3600, 3, 1)
	insertV4Hourly(t, raw, day+2000*3600, 2, 0)
	insertV4Hourly(t, raw, day+millisPerDay+1000*3600, 4, 2)
	if err := raw.Close(); err != nil {
		t.Fatalf("close raw: %v", err)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open after v4: %v", err)
	}
	defer func() { _ = store.Close() }()
	if SchemaVersion != 5 {
		t.Fatalf("SchemaVersion = %d, want 5", SchemaVersion)
	}

	type bucket struct {
		requests int64
		failures int64
		input    int64
		unpriced int64
		latMax   int64
	}
	got := map[int64]bucket{}
	rows, err := store.readDB.Query(
		`SELECT day_ms, requests, failures, input_tokens, unpriced_requests, latency_max_ms FROM usage_daily`)
	if err != nil {
		t.Fatalf("query daily: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var dayMs int64
		var b bucket
		if err := rows.Scan(&dayMs, &b.requests, &b.failures, &b.input, &b.unpriced, &b.latMax); err != nil {
			t.Fatalf("scan daily: %v", err)
		}
		got[dayMs] = b
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("daily rows: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("daily buckets = %d, want 2", len(got))
	}
	// Day one merges two hourly rows: latency max stays the max.
	if got[day] != (bucket{requests: 5, failures: 1, input: 50, unpriced: 5, latMax: 120}) {
		t.Fatalf("day one = %+v", got[day])
	}
	if got[day+millisPerDay] != (bucket{requests: 4, failures: 2, input: 40, unpriced: 4, latMax: 120}) {
		t.Fatalf("day two = %+v", got[day+millisPerDay])
	}
}

// TestDailyRollupWriterAndRouting writes events across 20 UTC days through
// the real writer path and checks the day rollup plus the routing rule:
// wide day-aligned ranges read the same totals as unaligned ones.
func TestDailyRollupWriterAndRouting(t *testing.T) {
	store := openTestStore(t)
	day := int64(1720000000000 / millisPerDay * millisPerDay)
	var rows []UsageRow
	for d := range 20 {
		for i := range 3 {
			ts := day + int64(d)*millisPerDay + int64(i+1)*3600*1000
			row := testRow("daily-"+string(rune('a'+d))+string(rune('0'+i)), ts)
			row.Failed = i == 2
			row.StatusClass = "2xx"
			if i == 2 {
				row.StatusClass = "5xx"
			}
			rows = append(rows, row)
		}
	}
	if _, err := store.InsertBatch(rows); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	var dailyRows, hourlyRows int64
	if err := store.readDB.QueryRow(`SELECT COUNT(*) FROM usage_daily`).Scan(&dailyRows); err != nil {
		t.Fatalf("count daily: %v", err)
	}
	if err := store.readDB.QueryRow(`SELECT COUNT(*) FROM usage_hourly`).Scan(&hourlyRows); err != nil {
		t.Fatalf("count hourly: %v", err)
	}
	if dailyRows != 40 || hourlyRows != 60 {
		t.Fatalf("daily/hourly rows = %d/%d, want 40/60 (daily splits 2xx/5xx per day)", dailyRows, hourlyRows)
	}
	var dailyReq, hourlyReq int64
	if err := store.readDB.QueryRow(`SELECT SUM(requests) FROM usage_daily`).Scan(&dailyReq); err != nil {
		t.Fatalf("sum daily: %v", err)
	}
	if err := store.readDB.QueryRow(`SELECT SUM(requests) FROM usage_hourly`).Scan(&hourlyReq); err != nil {
		t.Fatalf("sum hourly: %v", err)
	}
	if dailyReq != 60 || hourlyReq != 60 {
		t.Fatalf("daily/hourly requests = %d/%d, want 60/60", dailyReq, hourlyReq)
	}

	aligned := UsageFilter{FromMs: day, ToMs: day + 20*millisPerDay, TZ: "UTC", Loc: time.UTC}
	if !aligned.useDaily() {
		t.Fatal("aligned 20d range must use the day rollup")
	}
	shifted := aligned
	shifted.FromMs++
	if shifted.useDaily() {
		t.Fatal("unaligned range must stay on the hourly rollup")
	}
	short := UsageFilter{FromMs: day, ToMs: day + 7*millisPerDay, TZ: "UTC", Loc: time.UTC}
	if short.useDaily() {
		t.Fatal("7d range must stay on the hourly rollup")
	}
	if sel := aligned.rollup(true); sel.table != "usage_daily" || sel.timeCol != "day_ms" {
		t.Fatalf("aligned rollup = %v", sel)
	}
	if sel := shifted.rollup(true); sel.table != "usage_hourly" || sel.timeCol != "hour_ms" {
		t.Fatalf("shifted rollup = %v", sel)
	}

	wide, err := store.QueryCounters(aligned)
	if err != nil {
		t.Fatalf("aligned counters: %v", err)
	}
	narrow, err := store.QueryCounters(shifted)
	if err != nil {
		t.Fatalf("shifted counters: %v", err)
	}
	if wide.Requests != 60 || wide.Failures != 20 || narrow.Requests != 60 || narrow.Failures != 20 {
		t.Fatalf("wide/narrow = %d/%d and %d/%d, want 60/20 both",
			wide.Requests, wide.Failures, narrow.Requests, narrow.Failures)
	}

	// Breakdown over the aligned range reads the day table with the same
	// status split: 40 ok in 2xx, 20 failed in 5xx.
	items, total, err := store.QueryBreakdown(aligned, "status_class", "requests", "desc", 20, 0)
	if err != nil {
		t.Fatalf("aligned breakdown: %v", err)
	}
	if total != 2 {
		t.Fatalf("status classes = %d, want 2", total)
	}
	split := map[string]int64{}
	for _, item := range items {
		split[item.Key] = item.Requests
	}
	if split["2xx"] != 40 || split["5xx"] != 20 {
		t.Fatalf("status split = %v, want 2xx:40 5xx:20", split)
	}

	// Timeseries day buckets over the aligned range cover all 20 days.
	series, err := store.QueryTimeseries(aligned, "day", "", 0)
	if err != nil {
		t.Fatalf("aligned timeseries: %v", err)
	}
	if len(series.Points) != 20 {
		t.Fatalf("day points = %d, want 20", len(series.Points))
	}
	var summed int64
	for _, point := range series.Points {
		summed += point.Requests
	}
	if summed != 60 {
		t.Fatalf("series total = %d, want 60", summed)
	}
}

// TestDailyRollupLateFillAndCorrect checks the late fill and the audited
// correction keep the day rollup with the hourly one.
func TestDailyRollupLateFillAndCorrect(t *testing.T) {
	store := openTestStore(t)
	ts := int64(1720000000000)
	row := testRow("daily-late-1", ts)
	row.Provider = "codex"
	row.Model = "gpt-5"
	row.Alias = "gpt-5"
	if _, err := store.InsertBatch([]UsageRow{row}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	if _, err := store.AddPriceVersion(Price{
		ProviderKey: "openai", ModelID: "gpt-5", ValidFromMs: 0,
		Input: 1000000, Output: 2000000, Source: PriceSourceModelsDev,
	}); err != nil {
		t.Fatalf("AddPriceVersion: %v", err)
	}
	if _, err := store.FillUnpricedForKeys([][2]string{{"openai", "gpt-5"}}); err != nil {
		t.Fatalf("FillUnpricedForKeys: %v", err)
	}
	var hourly, daily [3]int64
	if err := store.readDB.QueryRow(
		`SELECT cost_micro_usd, late_cost_micro_usd, unpriced_requests FROM usage_hourly`).Scan(
		&hourly[0], &hourly[1], &hourly[2]); err != nil {
		t.Fatalf("hourly cost: %v", err)
	}
	if err := store.readDB.QueryRow(
		`SELECT cost_micro_usd, late_cost_micro_usd, unpriced_requests FROM usage_daily`).Scan(
		&daily[0], &daily[1], &daily[2]); err != nil {
		t.Fatalf("daily cost: %v", err)
	}
	if hourly != daily {
		t.Fatalf("hourly/daily after fill = %v/%v", hourly, daily)
	}
	if hourly != [3]int64{50, 50, 0} {
		t.Fatalf("cost/late/unpriced = %v, want [50 50 0]", hourly)
	}

	// The audited correction keeps the day rollup with the hourly one.
	from, to := ts-1, ts+1
	if _, _, _, _, err := store.CorrectCosts(FillScope{FromMs: &from, ToMs: &to}, "test reason", "test"); err != nil {
		t.Fatalf("CorrectCosts: %v", err)
	}
	if err := store.readDB.QueryRow(
		`SELECT cost_micro_usd, late_cost_micro_usd, unpriced_requests FROM usage_hourly`).Scan(
		&hourly[0], &hourly[1], &hourly[2]); err != nil {
		t.Fatalf("hourly cost after correct: %v", err)
	}
	if err := store.readDB.QueryRow(
		`SELECT cost_micro_usd, late_cost_micro_usd, unpriced_requests FROM usage_daily`).Scan(
		&daily[0], &daily[1], &daily[2]); err != nil {
		t.Fatalf("daily cost after correct: %v", err)
	}
	if hourly != daily {
		t.Fatalf("hourly/daily after correct = %v/%v", hourly, daily)
	}
	if hourly != [3]int64{50, 0, 0} {
		t.Fatalf("cost/late/unpriced after correct = %v, want [50 0 0]", hourly)
	}
}
