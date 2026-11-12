package usagedb

import (
	"fmt"
	"math"
	"time"
)

// QueryError is a client error (bad parameter). Routes map it to 400;
// other errors map to 500.
type QueryError struct{ Msg string }

func (e *QueryError) Error() string { return "usagedb: " + e.Msg }

// groupColumn maps a group_by value to its hourly column. Column names come
// from this fixed table only, never from user input.
func groupColumn(groupBy string) (string, error) {
	switch groupBy {
	case "model":
		return "model", nil
	case "provider":
		return "provider", nil
	case "alias":
		return "alias", nil
	case "credential":
		return "auth_id", nil
	case "api_key":
		return "api_key_id", nil
	case "status_class":
		return "status_class", nil
	case "stream":
		return "stream", nil
	}
	return "", &QueryError{Msg: "invalid group_by"}
}

// resolveBucket validates the bucket parameter and resolves auto from the
// range span: hour up to 3 days, day up to 120 days, else week.
func resolveBucket(bucket string, fromMs, toMs int64) (string, error) {
	switch bucket {
	case "hour", "day", "week":
		return bucket, nil
	case "", "auto":
		span := toMs - fromMs
		switch {
		case span <= 3*24*60*60*1000:
			return "hour", nil
		case span <= 120*24*60*60*1000:
			return "day", nil
		default:
			return "week", nil
		}
	}
	return "", &QueryError{Msg: "invalid bucket"}
}

// dayStartMs returns the UTC ms of the local midnight holding tsMs.
func dayStartMs(loc *time.Location, tsMs int64) int64 {
	local := time.UnixMilli(tsMs).In(loc)
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return midnight.UTC().UnixMilli()
}

// weekStartMs returns the UTC ms of the Monday 00:00 holding tsMs.
func weekStartMs(loc *time.Location, tsMs int64) int64 {
	local := time.UnixMilli(tsMs).In(loc)
	weekday := (int(local.Weekday()) + 6) % 7 // Monday first
	monday := time.Date(local.Year(), local.Month(), local.Day()-weekday, 0, 0, 0, 0, loc)
	return monday.UTC().UnixMilli()
}

// bucketStarts lists every bucket start (UTC ms) covering [fromMs, toMs).
func bucketStarts(kind string, loc *time.Location, fromMs, toMs int64) []int64 {
	var starts []int64
	switch kind {
	case "hour":
		for start := fromMs / millisPerHour * millisPerHour; start < toMs; start += millisPerHour {
			starts = append(starts, start)
		}
	case "day":
		for start := dayStartMs(loc, fromMs); start < toMs; {
			starts = append(starts, start)
			next := time.UnixMilli(start).In(loc)
			start = time.Date(next.Year(), next.Month(), next.Day()+1, 0, 0, 0, 0, loc).UTC().UnixMilli()
			if len(starts) > 5000 {
				break
			}
		}
	case "week":
		for start := weekStartMs(loc, fromMs); start < toMs; start += 7 * 24 * 60 * 60 * 1000 {
			starts = append(starts, start)
		}
	}
	return starts
}

// bucketOfHour maps one UTC hour bucket to its bucket start. Whole-hour
// offsets are exact; zones with 30 or 45 minute offsets put the hour bucket
// in the day and week of its start.
func bucketOfHour(kind string, loc *time.Location, hourMs int64) int64 {
	switch kind {
	case "day":
		return dayStartMs(loc, hourMs)
	case "week":
		return weekStartMs(loc, hourMs)
	default:
		return hourMs
	}
}

// hourlyAggRow is one grouped hourly row before bucketing.
type hourlyAggRow struct {
	hourMs                                     int64
	group                                      string
	requests, failures                         int64
	input, output, reasoning                   int64
	cacheRead, cacheWrite                      int64
	cost, estimated, unpriced                  int64
	latencySum, latencyMax, ttftSum, ttftCount int64
}

// fetchHourlyAgg reads grouped hourly sums for the filter. groupBy is "" for
// no grouping or a validated group_by value.
func (s *Store) fetchHourlyAgg(f UsageFilter, groupBy string) ([]hourlyAggRow, error) {
	selectCol := ""
	groupSuffix := ""
	if groupBy != "" {
		column, err := groupColumn(groupBy)
		if err != nil {
			return nil, err
		}
		if column == "stream" {
			selectCol = ", stream"
			groupSuffix = ", stream"
		} else {
			selectCol = ", " + column
			groupSuffix = ", " + column
		}
	}
	where, args := f.whereHourly()
	query := `SELECT hour_ms` + selectCol + `,
		SUM(requests), SUM(failures),
		SUM(input_tokens), SUM(output_tokens), SUM(reasoning_tokens),
		SUM(cache_read_tokens), SUM(cache_write_tokens),
		SUM(cost_micro_usd), SUM(late_cost_micro_usd), SUM(unpriced_requests),
		SUM(latency_sum_ms), MAX(latency_max_ms), SUM(ttft_sum_ms), SUM(ttft_count)
		FROM usage_hourly WHERE ` + where + ` GROUP BY hour_ms` + groupSuffix
	rows, err := s.readDB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("usagedb: hourly aggregate: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []hourlyAggRow
	for rows.Next() {
		var row hourlyAggRow
		var group any
		scan := []any{&row.hourMs}
		if groupBy != "" {
			if selectCol == ", stream" {
				var stream int64
				scan = append(scan, &stream)
				group = &stream
			} else {
				var text string
				scan = append(scan, &text)
				group = &text
			}
		}
		scan = append(scan, &row.requests, &row.failures,
			&row.input, &row.output, &row.reasoning, &row.cacheRead, &row.cacheWrite,
			&row.cost, &row.estimated, &row.unpriced,
			&row.latencySum, &row.latencyMax, &row.ttftSum, &row.ttftCount)
		if err := rows.Scan(scan...); err != nil {
			return nil, fmt.Errorf("usagedb: scan hourly aggregate: %w", err)
		}
		if groupBy != "" {
			switch value := group.(type) {
			case *int64:
				if *value == 1 {
					row.group = "true"
				} else {
					row.group = "false"
				}
			case *string:
				row.group = *value
			}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// UsagePoint is one bucket with its counters.
type UsagePoint struct {
	TsMs int64
	UsageCounters
}

// UsageSeries is one group key with bucket-aligned points.
type UsageSeries struct {
	Key    string
	Points []UsagePoint
}

// TimeseriesResult holds the bucketed series. Starts lists every bucket
// start in order; Points and every group run aligned with Starts.
type TimeseriesResult struct {
	Bucket string
	Starts []int64
	Points []UsagePoint
	Groups []UsageSeries
}

// QueryTimeseries buckets the hourly rollup. groupBy is "" for no grouping.
// top caps groups (default 5, max 20); the rest merge into key "other".
func (s *Store) QueryTimeseries(f UsageFilter, bucket, groupBy string, top int) (TimeseriesResult, error) {
	var result TimeseriesResult
	if s == nil {
		return result, fmt.Errorf("usagedb: nil store")
	}
	kind, err := resolveBucket(bucket, f.FromMs, f.ToMs)
	if err != nil {
		return result, err
	}
	if groupBy != "" {
		if _, err := groupColumn(groupBy); err != nil {
			return result, err
		}
	}
	if top <= 0 {
		top = 5
	}
	if top > 20 {
		return result, &QueryError{Msg: "invalid top"}
	}
	rows, err := s.fetchHourlyAgg(f, groupBy)
	if err != nil {
		return result, err
	}
	result.Bucket = kind
	result.Starts = bucketStarts(kind, f.Loc, f.FromMs, f.ToMs)
	index := make(map[int64]int, len(result.Starts))
	for i, start := range result.Starts {
		index[start] = i
	}
	zero := func(ts int64) UsagePoint { return UsagePoint{TsMs: ts} }
	result.Points = make([]UsagePoint, len(result.Starts))
	for i, start := range result.Starts {
		result.Points[i] = zero(start)
	}
	accumulate := func(point *UsagePoint, row hourlyAggRow) {
		point.Requests += row.requests
		point.Failures += row.failures
		point.InputTokens += row.input
		point.OutputTokens += row.output
		point.ReasoningTokens += row.reasoning
		point.CacheReadTokens += row.cacheRead
		point.CacheWriteTokens += row.cacheWrite
		point.CostMicroUSD += row.cost
		point.EstimatedCostMicroUSD += row.estimated
		point.UnpricedRequests += row.unpriced
		point.LatencyMaxMs = max(point.LatencyMaxMs, row.latencyMax)
	}
	// Keep raw sums on the side for averages; UsageCounters on points holds
	// partial sums until finalize below.
	latSums := make([]int64, len(result.Starts))
	ttftSums := make([]int64, len(result.Starts))
	ttftCounts := make([]int64, len(result.Starts))
	totals := make(map[string]int64)
	for _, row := range rows {
		bucket := bucketOfHour(kind, f.Loc, row.hourMs)
		i, ok := index[bucket]
		if !ok {
			continue
		}
		accumulate(&result.Points[i], row)
		latSums[i] += row.latencySum
		ttftSums[i] += row.ttftSum
		ttftCounts[i] += row.ttftCount
		if groupBy != "" {
			totals[row.group] += row.requests
		}
	}
	finalize := func(point *UsagePoint, i int) {
		point.TotalTokens = totalFromParts(point.InputTokens, point.OutputTokens,
			point.ReasoningTokens, point.CacheReadTokens, point.CacheWriteTokens)
		if point.Requests > 0 {
			point.FailureRate = float64(point.Failures) / float64(point.Requests)
			point.LatencyAvgMs = int64(float64(latSums[i])/float64(point.Requests) + 0.5)
		}
		if ttftCounts[i] > 0 {
			point.TTFTAvgMs = int64(float64(ttftSums[i])/float64(ttftCounts[i]) + 0.5)
		}
	}
	for i := range result.Points {
		finalize(&result.Points[i], i)
	}
	if groupBy == "" {
		return result, nil
	}
	// Rank groups by range requests; keep top, merge the rest as "other".
	type ranked struct {
		key      string
		requests int64
	}
	var order []ranked
	for key, requests := range totals {
		order = append(order, ranked{key, requests})
	}
	for i := 1; i < len(order); i++ {
		for j := i; j > 0 && (order[j].requests > order[j-1].requests ||
			(order[j].requests == order[j-1].requests && order[j].key < order[j-1].key)); j-- {
			order[j], order[j-1] = order[j-1], order[j]
		}
	}
	kept := make(map[string]bool)
	for i, item := range order {
		if i < top {
			kept[item.key] = true
		}
	}
	keys := make([]string, 0, top+1)
	for _, item := range order {
		if kept[item.key] {
			keys = append(keys, item.key)
		}
	}
	hasOther := len(order) > len(keys)
	if hasOther {
		keys = append(keys, "other")
	}
	series := make(map[string]*UsageSeries, len(keys))
	seriesLat := make(map[string][]int64, len(keys))
	seriesTTFT := make(map[string][][]int64, len(keys))
	for _, key := range keys {
		points := make([]UsagePoint, len(result.Starts))
		for i, start := range result.Starts {
			points[i] = zero(start)
		}
		series[key] = &UsageSeries{Key: key, Points: points}
		seriesLat[key] = make([]int64, len(result.Starts))
		seriesTTFT[key] = [][]int64{make([]int64, len(result.Starts)), make([]int64, len(result.Starts))}
	}
	for _, row := range rows {
		bucket := bucketOfHour(kind, f.Loc, row.hourMs)
		i, ok := index[bucket]
		if !ok {
			continue
		}
		key := row.group
		if !kept[key] {
			key = "other"
		}
		entry := series[key]
		accumulate(&entry.Points[i], row)
		seriesLat[key][i] += row.latencySum
		seriesTTFT[key][0][i] += row.ttftSum
		seriesTTFT[key][1][i] += row.ttftCount
	}
	for _, key := range keys {
		entry := series[key]
		for i := range entry.Points {
			point := &entry.Points[i]
			point.TotalTokens = totalFromParts(point.InputTokens, point.OutputTokens,
				point.ReasoningTokens, point.CacheReadTokens, point.CacheWriteTokens)
			if point.Requests > 0 {
				point.FailureRate = float64(point.Failures) / float64(point.Requests)
				point.LatencyAvgMs = int64(float64(seriesLat[key][i])/float64(point.Requests) + 0.5)
			}
			if seriesTTFT[key][1][i] > 0 {
				point.TTFTAvgMs = int64(float64(seriesTTFT[key][0][i])/float64(seriesTTFT[key][1][i]) + 0.5)
			}
		}
		result.Groups = append(result.Groups, *entry)
	}
	return result, nil
}

// BreakdownItem is one group with its counters and its share of requests.
type BreakdownItem struct {
	Key   string
	Share float64
	UsageCounters
}

// QueryBreakdown groups the hourly rollup. sort is requests, tokens, cost,
// failures, or latency; order is desc or asc; limit caps items (default 20,
// max 200); offset skips items.
func (s *Store) QueryBreakdown(f UsageFilter, groupBy, sort, order string, limit, offset int) ([]BreakdownItem, int64, error) {
	if s == nil {
		return nil, 0, fmt.Errorf("usagedb: nil store")
	}
	if _, err := groupColumn(groupBy); err != nil {
		return nil, 0, err
	}
	switch sort {
	case "", "requests", "tokens", "cost", "failures", "latency":
		if sort == "" {
			sort = "requests"
		}
	default:
		return nil, 0, &QueryError{Msg: "invalid sort"}
	}
	switch order {
	case "", "desc":
		order = "desc"
	case "asc":
	default:
		return nil, 0, &QueryError{Msg: "invalid order"}
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		return nil, 0, &QueryError{Msg: "invalid limit"}
	}
	if offset < 0 {
		return nil, 0, &QueryError{Msg: "invalid offset"}
	}
	rows, err := s.fetchHourlyAgg(f, groupBy)
	if err != nil {
		return nil, 0, err
	}
	byKey := make(map[string]*BreakdownItem)
	latSums := make(map[string]int64)
	ttftSums := make(map[string]int64)
	ttftCounts := make(map[string]int64)
	for _, row := range rows {
		item, ok := byKey[row.group]
		if !ok {
			item = &BreakdownItem{Key: row.group}
			byKey[row.group] = item
		}
		item.Requests += row.requests
		item.Failures += row.failures
		item.InputTokens += row.input
		item.OutputTokens += row.output
		item.ReasoningTokens += row.reasoning
		item.CacheReadTokens += row.cacheRead
		item.CacheWriteTokens += row.cacheWrite
		item.CostMicroUSD += row.cost
		item.EstimatedCostMicroUSD += row.estimated
		item.UnpricedRequests += row.unpriced
		item.LatencyMaxMs = max(item.LatencyMaxMs, row.latencyMax)
		latSums[row.group] += row.latencySum
		ttftSums[row.group] += row.ttftSum
		ttftCounts[row.group] += row.ttftCount
	}
	items := make([]BreakdownItem, 0, len(byKey))
	var totalRequests int64
	for _, item := range byKey {
		totalRequests += item.Requests
	}
	for _, item := range byKey {
		item.TotalTokens = totalFromParts(item.InputTokens, item.OutputTokens,
			item.ReasoningTokens, item.CacheReadTokens, item.CacheWriteTokens)
		if item.Requests > 0 {
			item.FailureRate = float64(item.Failures) / float64(item.Requests)
			item.LatencyAvgMs = int64(float64(latSums[item.Key])/float64(item.Requests) + 0.5)
		}
		if ttftCounts[item.Key] > 0 {
			item.TTFTAvgMs = int64(float64(ttftSums[item.Key])/float64(ttftCounts[item.Key]) + 0.5)
		}
		if totalRequests > 0 {
			item.Share = float64(item.Requests) / float64(totalRequests)
		}
		items = append(items, *item)
	}
	key := func(item BreakdownItem) float64 {
		switch sort {
		case "tokens":
			return float64(item.TotalTokens)
		case "cost":
			return float64(item.CostMicroUSD)
		case "failures":
			return float64(item.Failures)
		case "latency":
			return float64(item.LatencyAvgMs)
		default:
			return float64(item.Requests)
		}
	}
	for i := 1; i < len(items); i++ {
		for j := i; j > 0; j-- {
			a, b := key(items[j]), key(items[j-1])
			swap := a > b || (a == b && items[j].Key < items[j-1].Key)
			if order == "asc" {
				swap = a < b || (a == b && items[j].Key < items[j-1].Key)
			}
			if !swap {
				break
			}
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
	total := int64(len(items))
	if int64(offset) >= total {
		return []BreakdownItem{}, total, nil
	}
	end := offset + limit
	if int64(end) > total {
		end = int(total)
	}
	return items[offset:end], total, nil
}

// HeatmapResult is a 7x24 matrix, Monday first, with the peak cell.
type HeatmapResult struct {
	Matrix [7][24]int64
	Max    int64
}

// QueryHeatmap aggregates one metric over hour-of-week cells in the filter
// zone. metric is requests, tokens, cost, or failures.
func (s *Store) QueryHeatmap(f UsageFilter, metric string) (HeatmapResult, error) {
	var result HeatmapResult
	if s == nil {
		return result, fmt.Errorf("usagedb: nil store")
	}
	var column string
	switch metric {
	case "", "requests":
		column = "SUM(requests)"
		metric = "requests"
	case "tokens":
		column = "SUM(input_tokens+output_tokens+reasoning_tokens+cache_read_tokens+cache_write_tokens)"
	case "cost":
		column = "SUM(cost_micro_usd)"
	case "failures":
		column = "SUM(failures)"
	default:
		return result, &QueryError{Msg: "invalid metric"}
	}
	where, args := f.whereHourly()
	rows, err := s.readDB.Query(`SELECT hour_ms, `+column+` FROM usage_hourly WHERE `+where+` GROUP BY hour_ms`, args...)
	if err != nil {
		return result, fmt.Errorf("usagedb: heatmap: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var hourMs, value int64
		if err := rows.Scan(&hourMs, &value); err != nil {
			return result, fmt.Errorf("usagedb: scan heatmap: %w", err)
		}
		local := time.UnixMilli(hourMs).In(f.Loc)
		weekday := (int(local.Weekday()) + 6) % 7
		result.Matrix[weekday][local.Hour()] += value
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("usagedb: heatmap rows: %w", err)
	}
	for weekday := range 7 {
		for hour := range 24 {
			result.Max = max(result.Max, result.Matrix[weekday][hour])
		}
	}
	return result, nil
}

// SummaryResult is the summary route payload data.
type SummaryResult struct {
	Totals        UsageCounters
	Previous      UsageCounters
	P50Ms         *int64
	P95Ms         *int64
	LatencySource string
	FirstEventMs  int64
}

// QuerySummary returns range totals, previous-period totals, exact latency
// percentiles (or the hourly fallback), and the oldest stored event.
func (s *Store) QuerySummary(f UsageFilter) (SummaryResult, error) {
	var result SummaryResult
	if s == nil {
		return result, fmt.Errorf("usagedb: nil store")
	}
	totals, err := s.QueryCounters(f)
	if err != nil {
		return result, err
	}
	result.Totals = totals
	span := f.ToMs - f.FromMs
	previous := f
	previous.FromMs = f.FromMs - span
	previous.ToMs = f.FromMs
	previousTotals, err := s.countersFromHourly(previous)
	if err != nil {
		return result, err
	}
	result.Previous = previousTotals
	p50, p95, source, err := s.QueryLatencyPercentiles(f)
	if err != nil {
		return result, err
	}
	result.P50Ms = p50
	result.P95Ms = p95
	result.LatencySource = source
	var first int64
	if err := s.readDB.QueryRow(`SELECT COALESCE(MIN(ts_ms),0) FROM usage_events`).Scan(&first); err != nil {
		return result, fmt.Errorf("usagedb: first event: %w", err)
	}
	result.FirstEventMs = first
	return result, nil
}

// QueryLatencyPercentiles returns exact P50/P95 of latency_ms from raw
// events when the range holds at most maxExactPercentileRows rows. Above the
// limit it returns nulls with source "hourly". The rank method is
// nearest-rank: rank ceil(p*N), offset rank-1.
func (s *Store) QueryLatencyPercentiles(f UsageFilter) (*int64, *int64, string, error) {
	if s == nil {
		return nil, nil, "", fmt.Errorf("usagedb: nil store")
	}
	where, args := f.whereEvents()
	var count int64
	if err := s.readDB.QueryRow(`SELECT COUNT(*) FROM usage_events WHERE `+where, args...).Scan(&count); err != nil {
		return nil, nil, "", fmt.Errorf("usagedb: percentile count: %w", err)
	}
	if count == 0 {
		return nil, nil, "events", nil
	}
	if count > maxExactPercentileRows {
		return nil, nil, "hourly", nil
	}
	percentile := func(p float64) (int64, error) {
		rank := int64(math.Ceil(p * float64(count)))
		offset := rank - 1
		offset = min(max(offset, 0), count-1)
		queryArgs := append(append([]any{}, args...), offset)
		var value int64
		if err := s.readDB.QueryRow(`SELECT latency_ms FROM usage_events WHERE `+where+
			` ORDER BY latency_ms LIMIT 1 OFFSET ?`, queryArgs...).Scan(&value); err != nil {
			return 0, fmt.Errorf("usagedb: percentile: %w", err)
		}
		return value, nil
	}
	p50, err := percentile(0.5)
	if err != nil {
		return nil, nil, "", err
	}
	p95, err := percentile(0.95)
	if err != nil {
		return nil, nil, "", err
	}
	return &p50, &p95, "events", nil
}
