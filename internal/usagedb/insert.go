package usagedb

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	coreusage "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/usage"
)

func unixMilliNow() int64 { return time.Now().UnixMilli() }

const insertEventSQL = `INSERT INTO usage_events (
	request_id, trace_id, ts_ms, provider, model, alias, executor,
	auth_id, auth_index, auth_type, account_label, api_key_id,
	access_provider, session_id, stream, status_class, failed,
	fail_code, fail_summary, latency_ms, ttft_ms,
	input_tokens, output_tokens, reasoning_tokens,
	cache_read_tokens, cache_write_tokens, total_tokens,
	service_tier, reasoning_effort, cost_micro_usd, cost_source, price_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(request_id) DO NOTHING`

const upsertHourlySQL = `INSERT INTO usage_hourly (
	hour_ms, provider, model, alias, auth_id, api_key_id, status_class, stream,
	requests, failures, input_tokens, output_tokens, reasoning_tokens,
	cache_read_tokens, cache_write_tokens, cost_micro_usd, unpriced_requests,
	latency_sum_ms, latency_max_ms, ttft_sum_ms, ttft_count, late_cost_micro_usd
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(hour_ms, provider, model, alias, auth_id, api_key_id, status_class, stream)
DO UPDATE SET
	requests = usage_hourly.requests + 1,
	failures = usage_hourly.failures + excluded.failures,
	input_tokens = usage_hourly.input_tokens + excluded.input_tokens,
	output_tokens = usage_hourly.output_tokens + excluded.output_tokens,
	reasoning_tokens = usage_hourly.reasoning_tokens + excluded.reasoning_tokens,
	cache_read_tokens = usage_hourly.cache_read_tokens + excluded.cache_read_tokens,
	cache_write_tokens = usage_hourly.cache_write_tokens + excluded.cache_write_tokens,
	cost_micro_usd = usage_hourly.cost_micro_usd + excluded.cost_micro_usd,
	unpriced_requests = usage_hourly.unpriced_requests + excluded.unpriced_requests,
	latency_sum_ms = usage_hourly.latency_sum_ms + excluded.latency_sum_ms,
	latency_max_ms = max(usage_hourly.latency_max_ms, excluded.latency_max_ms),
	ttft_sum_ms = usage_hourly.ttft_sum_ms + excluded.ttft_sum_ms,
	ttft_count = usage_hourly.ttft_count + excluded.ttft_count`

const upsertDailySQL = `INSERT INTO usage_daily (
	day_ms, provider, model, alias, auth_id, api_key_id, status_class, stream,
	requests, failures, input_tokens, output_tokens, reasoning_tokens,
	cache_read_tokens, cache_write_tokens, cost_micro_usd, unpriced_requests,
	latency_sum_ms, latency_max_ms, ttft_sum_ms, ttft_count, late_cost_micro_usd
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(day_ms, provider, model, alias, auth_id, api_key_id, status_class, stream)
DO UPDATE SET
	requests = usage_daily.requests + 1,
	failures = usage_daily.failures + excluded.failures,
	input_tokens = usage_daily.input_tokens + excluded.input_tokens,
	output_tokens = usage_daily.output_tokens + excluded.output_tokens,
	reasoning_tokens = usage_daily.reasoning_tokens + excluded.reasoning_tokens,
	cache_read_tokens = usage_daily.cache_read_tokens + excluded.cache_read_tokens,
	cache_write_tokens = usage_daily.cache_write_tokens + excluded.cache_write_tokens,
	cost_micro_usd = usage_daily.cost_micro_usd + excluded.cost_micro_usd,
	unpriced_requests = usage_daily.unpriced_requests + excluded.unpriced_requests,
	latency_sum_ms = usage_daily.latency_sum_ms + excluded.latency_sum_ms,
	latency_max_ms = max(usage_daily.latency_max_ms, excluded.latency_max_ms),
	ttft_sum_ms = usage_daily.ttft_sum_ms + excluded.ttft_sum_ms,
	ttft_count = usage_daily.ttft_count + excluded.ttft_count`

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

// resolveRowCost fills the cost fields of one row inside the writer
// transaction, before insert. An upstream-reported cost wins and is stored
// with cost_source=upstream. Else the price version valid at the event time
// prices the non-overlapping token buckets with cost_source=price-book.
// Without a matching version the cost stays NULL with cost_source=none.
func (s *Store) resolveRowCost(row *UsageRow) {
	if row.UpstreamCostMicroUSD != nil {
		cost := *row.UpstreamCostMicroUSD
		row.CostMicroUSD = &cost
		source := CostSourceUpstream
		row.CostSource = &source
		row.PriceID = nil
		return
	}
	detail := rowCostDetail(*row)
	uncached, cacheRead, cacheWrite, output, reasoning := detail.PricedBuckets(row.Provider, row.Executor)
	price := s.lookupPrice(row.Provider, row.Model, row.Alias, row.TsMs)
	if price == nil {
		row.CostMicroUSD = nil
		source := CostSourceNone
		row.CostSource = &source
		row.PriceID = nil
		return
	}
	cost := price.ComputeCost(row.ServiceTier, uncached, cacheRead, cacheWrite, output, reasoning)
	row.CostMicroUSD = &cost
	source := CostSourcePriceBook
	row.CostSource = &source
	id := price.ID
	row.PriceID = &id
}

// rowCostDetail rebuilds the usage detail behind one row for cost math.
func rowCostDetail(row UsageRow) coreusage.Detail {
	return coreusage.Detail{
		InputTokens:         row.InputTokens,
		OutputTokens:        row.OutputTokens,
		ReasoningTokens:     row.ReasoningTokens,
		CacheReadTokens:     row.CacheReadTokens,
		CacheCreationTokens: row.CacheWriteTokens,
		TotalTokens:         row.TotalTokens,
	}
}

// InsertBatch inserts rows in one transaction. A duplicate request_id is
// skipped, and only inserted rows count in the hourly rollup, so a duplicate
// never counts twice. Cost is resolved in the same transaction, before
// insert, so the event and its hourly rollup always agree. It returns the
// inserted row count.
func (s *Store) InsertBatch(rows []UsageRow) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("usagedb: nil store")
	}
	if len(rows) == 0 {
		return 0, nil
	}
	tx, err := s.writeDB.Begin()
	if err != nil {
		return 0, fmt.Errorf("usagedb: begin batch: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var inserted int64
	for i := range rows {
		row := rows[i]
		s.resolveRowCost(&row)
		result, err := tx.Exec(insertEventSQL,
			row.RequestID, row.TraceID, row.TsMs, row.Provider, row.Model, row.Alias, row.Executor,
			row.AuthID, row.AuthIndex, row.AuthType, row.AccountLabel, row.APIKeyID,
			row.AccessProvider, row.SessionID, boolInt(row.Stream), row.StatusClass, boolInt(row.Failed),
			row.FailCode, row.FailSummary, row.LatencyMs, row.TTFTMs,
			row.InputTokens, row.OutputTokens, row.ReasoningTokens,
			row.CacheReadTokens, row.CacheWriteTokens, row.TotalTokens,
			row.ServiceTier, row.ReasoningEffort, row.CostMicroUSD, row.CostSource, row.PriceID,
		)
		if err != nil {
			return inserted, fmt.Errorf("usagedb: insert event: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return inserted, fmt.Errorf("usagedb: rows affected: %w", err)
		}
		if affected == 0 {
			continue
		}
		failures := int64(0)
		if row.Failed {
			failures = 1
		}
		unpriced := int64(0)
		var cost int64
		if row.CostMicroUSD == nil {
			unpriced = 1
		} else {
			cost = *row.CostMicroUSD
		}
		ttftSum := int64(0)
		ttftCount := int64(0)
		if row.TTFTMs > 0 {
			ttftSum = row.TTFTMs
			ttftCount = 1
		}
		if _, err := tx.Exec(upsertHourlySQL,
			row.HourMs, row.Provider, row.Model, row.Alias, row.AuthID, row.APIKeyID, row.StatusClass, boolInt(row.Stream),
			failures, row.InputTokens, row.OutputTokens, row.ReasoningTokens,
			row.CacheReadTokens, row.CacheWriteTokens, cost, unpriced,
			row.LatencyMs, row.LatencyMs, ttftSum, ttftCount, 0,
		); err != nil {
			return inserted, fmt.Errorf("usagedb: upsert hourly: %w", err)
		}
		// The day rollup carries the same dimensions at UTC day grain, in
		// the same transaction, so wide ranges can read it instead.
		if _, err := tx.Exec(upsertDailySQL,
			row.HourMs/millisPerDay*millisPerDay, row.Provider, row.Model, row.Alias,
			row.AuthID, row.APIKeyID, row.StatusClass, boolInt(row.Stream),
			failures, row.InputTokens, row.OutputTokens, row.ReasoningTokens,
			row.CacheReadTokens, row.CacheWriteTokens, cost, unpriced,
			row.LatencyMs, row.LatencyMs, ttftSum, ttftCount, 0,
		); err != nil {
			return inserted, fmt.Errorf("usagedb: upsert daily: %w", err)
		}
		inserted++
	}
	if err := tx.Commit(); err != nil {
		return inserted, fmt.Errorf("usagedb: commit batch: %w", err)
	}
	return inserted, nil
}

// PruneBefore deletes raw events older than cutoffMs in one batch of at most
// limit rows. The caller loops with a pause between batches. Rollups stay.
// It returns the deleted row count.
func (s *Store) PruneBefore(cutoffMs int64, limit int) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("usagedb: nil store")
	}
	if limit <= 0 {
		limit = pruneBatchSize
	}
	result, err := s.writeDB.Exec(`DELETE FROM usage_events WHERE rowid IN (
		SELECT rowid FROM usage_events WHERE ts_ms < ? ORDER BY ts_ms LIMIT ?
	)`, cutoffMs, limit)
	if err != nil {
		return 0, fmt.Errorf("usagedb: prune: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("usagedb: prune rows affected: %w", err)
	}
	return deleted, nil
}

// EventStats holds raw table counts for the status route and retention.
type EventStats struct {
	Events     int64
	OldestTsMs int64
	NewestTsMs int64
}

// Stats returns the raw event count and oldest/newest timestamps.
// Timestamps stay 0 when the table is empty.
func (s *Store) Stats() (EventStats, error) {
	var stats EventStats
	if s == nil {
		return stats, fmt.Errorf("usagedb: nil store")
	}
	var oldest, newest sql.NullInt64
	if err := s.readDB.QueryRow(`SELECT COUNT(*), MIN(ts_ms), MAX(ts_ms) FROM usage_events`).Scan(
		&stats.Events, &oldest, &newest); err != nil {
		return stats, fmt.Errorf("usagedb: stats: %w", err)
	}
	stats.OldestTsMs = oldest.Int64
	stats.NewestTsMs = newest.Int64
	return stats, nil
}

// SizeBytes sums the database file and its WAL/SHM sidecars. Missing files
// count as 0, so a fresh database reports 0 before the first checkpoint.
func (s *Store) SizeBytes() int64 {
	if s == nil {
		return 0
	}
	var total int64
	for _, name := range []string{s.path, s.path + "-wal", s.path + "-shm"} {
		info, err := os.Stat(name)
		if err != nil {
			continue
		}
		total += info.Size()
	}
	return total
}
