package usagedb

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// maxUsageRangeMs caps every usage query range at 366 days.
	maxUsageRangeMs = int64(366 * 24 * 60 * 60 * 1000)
	// millisPerDay is one UTC day in milliseconds.
	millisPerDay = int64(24 * 60 * 60 * 1000)
	// exportPageSize pages export streams on (ts_ms, id).
	exportPageSize = 2000
)

// maxExactTotalRows caps the cheap exact total: up to this many matching
// events the count is exact, above it the route estimates.
// maxExactPercentileRows caps exact P50/P95 math on raw events.
// Both are vars (not consts) so unit tests can shrink them and cover the
// over-limit branches without million-row fixtures. Production defaults stay
// as documented in the contract.
var (
	maxExactTotalRows      int64 = 100000
	maxExactPercentileRows int64 = 1000000
)

// UsageFilter holds the parsed common query parameters of the usage routes.
// All values come from the URL query; SQL builders use bound parameters
// only, never string-built user input.
type UsageFilter struct {
	FromMs int64
	ToMs   int64
	TZ     string
	Loc    *time.Location
	// Each list is an OR set; empty means no filter on that dimension.
	Providers []string
	Models    []string
	Aliases   []string
	AuthIDs   []string
	APIKeyIDs []string
	// Status is "", "ok", or "failed".
	Status string
	// Stream is nil for both, else the wanted value.
	Stream *bool
	// Q is text search for events, failures, and export only.
	Q string
}

// HasQ reports whether text search is active.
func (f UsageFilter) HasQ() bool { return strings.TrimSpace(f.Q) != "" }

// isPlainRange reports a filter with only a time range (plus tz): no
// dimension, status, stream, or text constraints. The events page loads with
// this shape, so it gets the cheapest total.
func (f UsageFilter) isPlainRange() bool {
	return len(f.Providers) == 0 && len(f.Models) == 0 && len(f.Aliases) == 0 &&
		len(f.AuthIDs) == 0 && len(f.APIKeyIDs) == 0 &&
		f.Status == "" && f.Stream == nil && !f.HasQ()
}

// ParseUsageFilter parses and validates the common usage query parameters.
// nowMs supplies the default range end (last 24 hours). An error means the
// route must answer 400.
func ParseUsageFilter(values url.Values, nowMs int64) (UsageFilter, error) {
	var filter UsageFilter
	if nowMs < 0 {
		nowMs = 0
	}
	filter.ToMs = nowMs
	filter.FromMs = nowMs - 24*60*60*1000
	if raw := strings.TrimSpace(values.Get("from_ms")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 0 {
			return filter, &QueryError{Msg: "invalid from_ms"}
		}
		filter.FromMs = value
	}
	if raw := strings.TrimSpace(values.Get("to_ms")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 0 {
			return filter, &QueryError{Msg: "invalid to_ms"}
		}
		filter.ToMs = value
	}
	if filter.ToMs <= filter.FromMs {
		return filter, &QueryError{Msg: "to_ms must be after from_ms"}
	}
	if filter.ToMs-filter.FromMs > maxUsageRangeMs {
		return filter, &QueryError{Msg: "range exceeds 366 days"}
	}
	filter.TZ = strings.TrimSpace(values.Get("tz"))
	if filter.TZ == "" {
		filter.TZ = "UTC"
	}
	loc, err := time.LoadLocation(filter.TZ)
	if err != nil {
		return filter, &QueryError{Msg: "invalid tz"}
	}
	filter.Loc = loc
	collect := func(key string) []string {
		raws, ok := values[key]
		if !ok {
			return nil
		}
		var out []string
		for _, raw := range raws {
			for _, part := range strings.Split(raw, ",") {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					out = append(out, trimmed)
				}
			}
		}
		return out
	}
	filter.Providers = collect("provider")
	filter.Models = collect("model")
	filter.Aliases = collect("alias")
	filter.AuthIDs = collect("auth_id")
	filter.APIKeyIDs = collect("api_key_id")
	filter.Status = strings.TrimSpace(values.Get("status"))
	switch filter.Status {
	case "", "ok", "failed":
	default:
		return filter, &QueryError{Msg: "invalid status"}
	}
	if raw := strings.TrimSpace(values.Get("stream")); raw != "" {
		switch raw {
		case "true":
			value := true
			filter.Stream = &value
		case "false":
			value := false
			filter.Stream = &value
		default:
			return filter, &QueryError{Msg: "invalid stream"}
		}
	}
	filter.Q = strings.TrimSpace(values.Get("q"))
	return filter, nil
}

// appendIn adds a bound `column IN (...)` clause for a non-empty value set.
func appendIn(sb *strings.Builder, args *[]any, column string, values []string) {
	if len(values) == 0 {
		return
	}
	sb.WriteString(" AND ")
	sb.WriteString(column)
	sb.WriteString(" IN (")
	for i, value := range values {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("?")
		*args = append(*args, value)
	}
	sb.WriteString(")")
}

// escapeLike escapes the LIKE wildcards in user text.
func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "%", `\%`)
	value = strings.ReplaceAll(value, "_", `\_`)
	return "%" + value + "%"
}

// whereEvents builds the WHERE clause on usage_events. The status filter
// maps to the failed flag here. Callers must use the returned args in order.
func (f UsageFilter) whereEvents() (string, []any) {
	var sb strings.Builder
	var args []any
	sb.WriteString("ts_ms >= ? AND ts_ms < ?")
	args = append(args, f.FromMs, f.ToMs)
	appendIn(&sb, &args, "provider", f.Providers)
	appendIn(&sb, &args, "model", f.Models)
	appendIn(&sb, &args, "alias", f.Aliases)
	appendIn(&sb, &args, "auth_id", f.AuthIDs)
	appendIn(&sb, &args, "api_key_id", f.APIKeyIDs)
	switch f.Status {
	case "ok":
		sb.WriteString(" AND failed = 0")
	case "failed":
		sb.WriteString(" AND failed = 1")
	}
	if f.Stream != nil {
		if *f.Stream {
			sb.WriteString(" AND stream = 1")
		} else {
			sb.WriteString(" AND stream = 0")
		}
	}
	if f.HasQ() {
		pattern := escapeLike(f.Q)
		sb.WriteString(` AND (request_id LIKE ? ESCAPE '\' OR trace_id LIKE ? ESCAPE '\'` +
			` OR model LIKE ? ESCAPE '\' OR alias LIKE ? ESCAPE '\'` +
			` OR account_label LIKE ? ESCAPE '\' OR fail_summary LIKE ? ESCAPE '\')`)
		for range 6 {
			args = append(args, pattern)
		}
	}
	return sb.String(), args
}

// whereHourly builds the WHERE clause on usage_hourly. The status filter
// maps to status_class here: failed means every class except 2xx, because a
// failed record is never stored as 2xx. Callers must use the returned args
// in order.
func (f UsageFilter) whereHourly() (string, []any) {
	var sb strings.Builder
	var args []any
	sb.WriteString("hour_ms >= ? AND hour_ms < ?")
	args = append(args, f.FromMs/millisPerHour*millisPerHour, f.ToMs)
	appendIn(&sb, &args, "provider", f.Providers)
	appendIn(&sb, &args, "model", f.Models)
	appendIn(&sb, &args, "alias", f.Aliases)
	appendIn(&sb, &args, "auth_id", f.AuthIDs)
	appendIn(&sb, &args, "api_key_id", f.APIKeyIDs)
	switch f.Status {
	case "ok":
		sb.WriteString(" AND status_class = '2xx'")
	case "failed":
		sb.WriteString(" AND status_class IN ('4xx','5xx','error','other')")
	}
	if f.Stream != nil {
		if *f.Stream {
			sb.WriteString(" AND stream = 1")
		} else {
			sb.WriteString(" AND stream = 0")
		}
	}
	return sb.String(), args
}

// useDaily reports whether the range reads the internal day rollup: a span
// over 14 days with whole-day UTC bounds. Day rows cannot serve partial
// days, so unaligned ranges stay on the hourly rollup.
func (f UsageFilter) useDaily() bool {
	if f.ToMs-f.FromMs <= 14*millisPerDay {
		return false
	}
	return f.FromMs%millisPerDay == 0 && f.ToMs%millisPerDay == 0
}

// whereDaily builds the WHERE clause on usage_daily: same dimensions as
// whereHourly, day grain. Callers must use the returned args in order.
func (f UsageFilter) whereDaily() (string, []any) {
	var sb strings.Builder
	var args []any
	sb.WriteString("day_ms >= ? AND day_ms < ?")
	args = append(args, f.FromMs/millisPerDay*millisPerDay, f.ToMs)
	appendIn(&sb, &args, "provider", f.Providers)
	appendIn(&sb, &args, "model", f.Models)
	appendIn(&sb, &args, "alias", f.Aliases)
	appendIn(&sb, &args, "auth_id", f.AuthIDs)
	appendIn(&sb, &args, "api_key_id", f.APIKeyIDs)
	switch f.Status {
	case "ok":
		sb.WriteString(" AND status_class = '2xx'")
	case "failed":
		sb.WriteString(" AND status_class IN ('4xx','5xx','error','other')")
	}
	if f.Stream != nil {
		if *f.Stream {
			sb.WriteString(" AND stream = 1")
		} else {
			sb.WriteString(" AND stream = 0")
		}
	}
	return sb.String(), args
}

// rollupSel is one rollup read: table plus its time column, WHERE, and args.
type rollupSel struct {
	table   string
	timeCol string
	where   string
	args    []any
}

// rollup picks the rollup table for the filter: usage_daily for wide
// day-aligned ranges, else usage_hourly. Hour-bucketed series always need the
// hourly grain, and the heatmap needs hour-of-day, so those callers pass
// hourOK=false and stay on usage_hourly.
func (f UsageFilter) rollup(hourOK bool) rollupSel {
	if hourOK && f.useDaily() {
		where, args := f.whereDaily()
		return rollupSel{table: "usage_daily", timeCol: "day_ms", where: where, args: args}
	}
	where, args := f.whereHourly()
	return rollupSel{table: "usage_hourly", timeCol: "hour_ms", where: where, args: args}
}

// UsageCounters is the counters object shared by summary, timeseries,
// breakdown, and the panel. EstimatedCostMicroUSD is the late-priced part of
// CostMicroUSD.
type UsageCounters struct {
	Requests              int64
	Failures              int64
	FailureRate           float64
	InputTokens           int64
	OutputTokens          int64
	ReasoningTokens       int64
	CacheReadTokens       int64
	CacheWriteTokens      int64
	TotalTokens           int64
	CostMicroUSD          int64
	EstimatedCostMicroUSD int64
	UnpricedRequests      int64
	LatencyAvgMs          int64
	LatencyMaxMs          int64
	TTFTAvgMs             int64
}

// finalize fills the derived fields of a counters row.
func finalizeCounters(requests, failures, input, output, reasoning, cacheRead, cacheWrite, total,
	cost, estimated, unpriced, latencySum, latencyMax, ttftSum, ttftCount int64) UsageCounters {
	counters := UsageCounters{
		Requests:              requests,
		Failures:              failures,
		InputTokens:           input,
		OutputTokens:          output,
		ReasoningTokens:       reasoning,
		CacheReadTokens:       cacheRead,
		CacheWriteTokens:      cacheWrite,
		TotalTokens:           total,
		CostMicroUSD:          cost,
		EstimatedCostMicroUSD: estimated,
		UnpricedRequests:      unpriced,
		LatencyMaxMs:          latencyMax,
	}
	if requests > 0 {
		counters.FailureRate = float64(failures) / float64(requests)
		counters.LatencyAvgMs = int64(float64(latencySum)/float64(requests) + 0.5)
	}
	if ttftCount > 0 {
		counters.TTFTAvgMs = int64(float64(ttftSum)/float64(ttftCount) + 0.5)
	}
	return counters
}

// totalFromParts sums the token buckets when no stored total exists
// (usage_hourly has no total column).
func totalFromParts(input, output, reasoning, cacheRead, cacheWrite int64) int64 {
	return input + output + reasoning + cacheRead + cacheWrite
}

// QueryCounters returns the counters for the filter. Without text search it
// reads the hourly rollup; with q it reads raw events, because q spans
// fields the rollup does not keep.
func (s *Store) QueryCounters(f UsageFilter) (UsageCounters, error) {
	if s == nil {
		return UsageCounters{}, fmt.Errorf("usagedb: nil store")
	}
	if f.HasQ() {
		return s.countersFromEvents(f)
	}
	return s.countersFromHourly(f)
}

const countersRollupSQL = `SELECT COALESCE(SUM(requests),0), COALESCE(SUM(failures),0),
	COALESCE(SUM(input_tokens),0), COALESCE(SUM(output_tokens),0), COALESCE(SUM(reasoning_tokens),0),
	COALESCE(SUM(cache_read_tokens),0), COALESCE(SUM(cache_write_tokens),0),
	COALESCE(SUM(cost_micro_usd),0), COALESCE(SUM(late_cost_micro_usd),0), COALESCE(SUM(unpriced_requests),0),
	COALESCE(SUM(latency_sum_ms),0), COALESCE(MAX(latency_max_ms),0),
	COALESCE(SUM(ttft_sum_ms),0), COALESCE(SUM(ttft_count),0)
	FROM %s WHERE `

// countersFromHourly reads the routed rollup: the day table for wide
// day-aligned ranges, else the hourly table. The name stays: callers and
// history call it hourly.
func (s *Store) countersFromHourly(f UsageFilter) (UsageCounters, error) {
	sel := f.rollup(true)
	var requests, failures, input, output, reasoning, cacheRead, cacheWrite int64
	var cost, estimated, unpriced, latencySum, latencyMax, ttftSum, ttftCount int64
	if err := s.readDB.QueryRow(fmt.Sprintf(countersRollupSQL, sel.table)+sel.where, sel.args...).Scan(
		&requests, &failures, &input, &output, &reasoning, &cacheRead, &cacheWrite,
		&cost, &estimated, &unpriced, &latencySum, &latencyMax, &ttftSum, &ttftCount); err != nil {
		return UsageCounters{}, fmt.Errorf("usagedb: hourly counters: %w", err)
	}
	return finalizeCounters(requests, failures, input, output, reasoning, cacheRead, cacheWrite,
		totalFromParts(input, output, reasoning, cacheRead, cacheWrite),
		cost, estimated, unpriced, latencySum, latencyMax, ttftSum, ttftCount), nil
}

const countersEventsSQL = `SELECT COUNT(*), COALESCE(SUM(failed),0),
	COALESCE(SUM(input_tokens),0), COALESCE(SUM(output_tokens),0), COALESCE(SUM(reasoning_tokens),0),
	COALESCE(SUM(cache_read_tokens),0), COALESCE(SUM(cache_write_tokens),0), COALESCE(SUM(total_tokens),0),
	COALESCE(SUM(cost_micro_usd),0),
	COALESCE(SUM(CASE WHEN cost_source = 'price-book-late' THEN cost_micro_usd ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN cost_micro_usd IS NULL THEN 1 ELSE 0 END),0),
	COALESCE(SUM(latency_ms),0), COALESCE(MAX(latency_ms),0),
	COALESCE(SUM(CASE WHEN ttft_ms > 0 THEN ttft_ms ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN ttft_ms > 0 THEN 1 ELSE 0 END),0)
	FROM usage_events WHERE `

func (s *Store) countersFromEvents(f UsageFilter) (UsageCounters, error) {
	where, args := f.whereEvents()
	var requests, failures, input, output, reasoning, cacheRead, cacheWrite, total int64
	var cost, estimated, unpriced, latencySum, latencyMax, ttftSum, ttftCount int64
	if err := s.readDB.QueryRow(countersEventsSQL+where, args...).Scan(
		&requests, &failures, &input, &output, &reasoning, &cacheRead, &cacheWrite, &total,
		&cost, &estimated, &unpriced, &latencySum, &latencyMax, &ttftSum, &ttftCount); err != nil {
		return UsageCounters{}, fmt.Errorf("usagedb: events counters: %w", err)
	}
	return finalizeCounters(requests, failures, input, output, reasoning, cacheRead, cacheWrite,
		total, cost, estimated, unpriced, latencySum, latencyMax, ttftSum, ttftCount), nil
}
