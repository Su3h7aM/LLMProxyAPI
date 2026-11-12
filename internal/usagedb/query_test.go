package usagedb

import (
	"context"
	"net/url"
	"testing"
	"time"
)

func mustFilter(t *testing.T, raw string, nowMs int64) UsageFilter {
	t.Helper()
	values, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	filter, err := ParseUsageFilter(values, nowMs)
	if err != nil {
		t.Fatalf("ParseUsageFilter %q: %v", raw, err)
	}
	return filter
}

func mustFilterErr(t *testing.T, raw string, nowMs int64) {
	t.Helper()
	values, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if _, err := ParseUsageFilter(values, nowMs); err == nil {
		t.Fatalf("ParseUsageFilter %q succeeded, want error", raw)
	}
}

func intPtr(value int64) *int64 { return &value }
func strPtr(value string) *string { return &value }

func seedRow(id string, ts int64) UsageRow {
	return UsageRow{
		RequestID: id, TraceID: "trace-" + id, TsMs: ts, HourMs: ts / millisPerHour * millisPerHour,
		Provider: "codex", Model: "gpt-5", Alias: "gpt-5", Executor: "codex",
		AuthID: "codex-1.json", AuthType: "oauth", AccountLabel: "ab***@example.com",
		APIKeyID: "key1", Stream: false, StatusClass: "2xx",
		LatencyMs: 100, TTFTMs: 30,
		InputTokens: 10, OutputTokens: 20, ReasoningTokens: 0,
		CacheReadTokens: 0, CacheWriteTokens: 0, TotalTokens: 30,
		ServiceTier: "default",
		UpstreamCostMicroUSD: intPtr(100),
	}
}

// seedQueryStore builds a store with known rows across two UTC days:
// day A (base): 3 ok events (one streamed) plus 1 failed 5xx;
// day B (base + 24h): 1 ok event of another model.
func seedQueryStore(t *testing.T, base int64) *Store {
	t.Helper()
	store := openTestStore(t)
	rows := []UsageRow{
		func() UsageRow { row := seedRow("q-e1", base+1000); return row }(),
		func() UsageRow { row := seedRow("q-e2", base+2000); row.Stream = true; return row }(),
		func() UsageRow { row := seedRow("q-e3", base+3000); row.LatencyMs = 300; return row }(),
		func() UsageRow {
			row := seedRow("q-f1", base+4000)
			row.Failed = true
			row.StatusClass = "5xx"
			row.FailCode = "rate_limit"
			row.FailSummary = "slow down"
			row.LatencyMs = 500
			return row
		}(),
		func() UsageRow {
			row := seedRow("q-e4", base+24*60*60*1000+1000)
			row.Model = "gpt-6"
			row.Alias = "gpt-6"
			return row
		}(),
	}
	if _, err := store.InsertBatch(rows); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	// Mark q-f1 late: cost 700 with late source, as the fill path does.
	if _, err := store.writeDB.Exec(`UPDATE usage_events SET cost_micro_usd = 700, cost_source = 'price-book-late' WHERE request_id = 'q-f1'`); err != nil {
		t.Fatalf("mark late: %v", err)
	}
	hour := (base + 4000) / millisPerHour * millisPerHour
	if _, err := store.writeDB.Exec(`UPDATE usage_hourly SET cost_micro_usd = cost_micro_usd + 600,
		late_cost_micro_usd = 700 WHERE hour_ms = ? AND model = 'gpt-5' AND status_class = '5xx'`, hour); err != nil {
		t.Fatalf("fix hourly: %v", err)
	}
	return store
}

func TestParseUsageFilter(t *testing.T) {
	now := int64(1720000000000)
	filter := mustFilter(t, "", now)
	if filter.ToMs != now || filter.FromMs != now-24*60*60*1000 {
		t.Fatalf("default range = %d/%d", filter.FromMs, filter.ToMs)
	}
	if filter.TZ != "UTC" || filter.Loc == nil {
		t.Fatalf("default tz = %q", filter.TZ)
	}
	filter = mustFilter(t, "provider=a&provider=b&status=failed&stream=true&tz=Europe%2FBerlin", now)
	if len(filter.Providers) != 2 || filter.Status != "failed" {
		t.Fatalf("multi filter = %+v", filter)
	}
	if filter.Stream == nil || !*filter.Stream {
		t.Fatalf("stream = %+v", filter.Stream)
	}
	for _, raw := range []string{
		"from_ms=5&to_ms=5", "from_ms=10&to_ms=5", "from_ms=abc",
		"from_ms=0&to_ms=40000000000", "status=bogus", "stream=yes",
		"tz=Not%2FAZone",
	} {
		mustFilterErr(t, raw, now)
	}
	if _, _, err := DecodeUsageCursor("bogus!!"); err == nil {
		t.Fatal("bad cursor accepted")
	}
	if _, _, err := DecodeUsageCursor("e30="); err == nil {
		t.Fatal("empty cursor payload accepted")
	}
}

func TestQueryCountersMath(t *testing.T) {
	base := int64(1719997200000) // an exact UTC hour
	store := seedQueryStore(t, base)
	filter := mustFilter(t, "", base+48*60*60*1000)
	filter.FromMs = base
	filter.ToMs = base + 48*60*60*1000
	counters, err := store.QueryCounters(filter)
	if err != nil {
		t.Fatalf("QueryCounters: %v", err)
	}
	// 5 requests, 1 failure; cost 4x100 upstream plus late 700 (late 700
	// replaces the 100 already counted at insert: hourly cost 400+600).
	if counters.Requests != 5 || counters.Failures != 1 {
		t.Fatalf("requests/failures = %d/%d, want 5/1", counters.Requests, counters.Failures)
	}
	if counters.FailureRate != 0.2 {
		t.Fatalf("failure rate = %v, want 0.2", counters.FailureRate)
	}
	if counters.InputTokens != 50 || counters.OutputTokens != 100 || counters.TotalTokens != 150 {
		t.Fatalf("tokens = %d/%d/%d", counters.InputTokens, counters.OutputTokens, counters.TotalTokens)
	}
	if counters.CostMicroUSD != 1100 || counters.EstimatedCostMicroUSD != 700 {
		t.Fatalf("cost/estimated = %d/%d, want 1100/700", counters.CostMicroUSD, counters.EstimatedCostMicroUSD)
	}
	if counters.UnpricedRequests != 0 {
		t.Fatalf("unpriced = %d, want 0", counters.UnpricedRequests)
	}
	// Latencies 100,100,300,500,100: avg 220, max 500. TTFT 30 avg.
	if counters.LatencyAvgMs != 220 || counters.LatencyMaxMs != 500 {
		t.Fatalf("latency = %d/%d, want 220/500", counters.LatencyAvgMs, counters.LatencyMaxMs)
	}
	if counters.TTFTAvgMs != 30 {
		t.Fatalf("ttft = %d, want 30", counters.TTFTAvgMs)
	}
	// Status filter on the hourly path maps to status_class.
	failed := mustFilter(t, "status=failed", base+48*60*60*1000)
	failed.FromMs = base
	failed.ToMs = base + 48*60*60*1000
	fc, err := store.QueryCounters(failed)
	if err != nil {
		t.Fatalf("failed counters: %v", err)
	}
	if fc.Requests != 1 || fc.Failures != 1 {
		t.Fatalf("failed requests/failures = %d/%d, want 1/1", fc.Requests, fc.Failures)
	}
	// Text search switches to the events path.
	searched := failed
	searched.Q = "slow"
	searched.Status = ""
	sc, err := store.QueryCounters(searched)
	if err != nil {
		t.Fatalf("search counters: %v", err)
	}
	if sc.Requests != 1 || sc.TotalTokens != 30 {
		t.Fatalf("search requests/total = %d/%d, want 1/30", sc.Requests, sc.TotalTokens)
	}
}

func TestQuerySummaryPrevious(t *testing.T) {
	base := int64(1719997200000)
	store := seedQueryStore(t, base)
	// Range holds day B only; previous holds day A.
	filter := mustFilter(t, "", base+48*60*60*1000)
	filter.FromMs = base + 24*60*60*1000
	filter.ToMs = base + 48*60*60*1000
	summary, err := store.QuerySummary(filter)
	if err != nil {
		t.Fatalf("QuerySummary: %v", err)
	}
	if summary.Totals.Requests != 1 || summary.Previous.Requests != 4 {
		t.Fatalf("totals/previous = %d/%d, want 1/4", summary.Totals.Requests, summary.Previous.Requests)
	}
	if summary.LatencySource != "events" || summary.P50Ms == nil || summary.P95Ms == nil {
		t.Fatalf("latency = %v %v source %s", summary.P50Ms, summary.P95Ms, summary.LatencySource)
	}
	// Day B holds one event with latency 100.
	if *summary.P50Ms != 100 || *summary.P95Ms != 100 {
		t.Fatalf("p50/p95 = %d/%d, want 100/100", *summary.P50Ms, *summary.P95Ms)
	}
	if summary.FirstEventMs != base+1000 {
		t.Fatalf("first = %d, want %d", summary.FirstEventMs, base+1000)
	}
}

func TestQueryTimeseriesBuckets(t *testing.T) {
	base := int64(1719997200000)
	store := seedQueryStore(t, base)
	filter := mustFilter(t, "", base+48*60*60*1000)
	filter.FromMs = base
	filter.ToMs = base + 48*60*60*1000
	series, err := store.QueryTimeseries(filter, "hour", "", 0)
	if err != nil {
		t.Fatalf("timeseries: %v", err)
	}
	if series.Bucket != "hour" || len(series.Points) != 48 {
		t.Fatalf("hour points = %d, want 48", len(series.Points))
	}
	var total int64
	for _, point := range series.Points {
		total += point.Requests
	}
	if total != 5 {
		t.Fatalf("series total = %d, want 5", total)
	}
	// Day buckets in a +14 zone keep all day-A rows on one day.
	tzFilter := filter
	tzFilter.TZ = "Pacific/Kiritimati"
	loc, err := time.LoadLocation(tzFilter.TZ)
	if err != nil {
		t.Fatalf("load tz: %v", err)
	}
	tzFilter.Loc = loc
	days, err := store.QueryTimeseries(tzFilter, "day", "", 0)
	if err != nil {
		t.Fatalf("day series: %v", err)
	}
	if len(days.Points) != 3 {
		t.Fatalf("day points = %d, want 3", len(days.Points))
	}
	if days.Points[0].Requests != 4 || days.Points[1].Requests != 1 || days.Points[2].Requests != 0 {
		t.Fatalf("day split = %d/%d/%d, want 4/1/0", days.Points[0].Requests, days.Points[1].Requests, days.Points[2].Requests)
	}
	// Auto resolves by span.
	auto, err := store.QueryTimeseries(filter, "", "", 0)
	if err != nil || auto.Bucket != "hour" {
		t.Fatalf("auto 2d = %q, %v", auto.Bucket, err)
	}
	wide := filter
	wide.ToMs = base + 200*24*60*60*1000
	wideSeries, err := store.QueryTimeseries(wide, "", "", 0)
	if err != nil || wideSeries.Bucket != "week" {
		t.Fatalf("auto 200d = %q, %v", wideSeries.Bucket, err)
	}
	// Group by model with top 1 merges the rest into other.
	grouped, err := store.QueryTimeseries(filter, "hour", "model", 1)
	if err != nil {
		t.Fatalf("grouped: %v", err)
	}
	if len(grouped.Groups) != 2 {
		t.Fatalf("groups = %d, want 2", len(grouped.Groups))
	}
	if grouped.Groups[0].Key != "gpt-5" || grouped.Groups[1].Key != "other" {
		t.Fatalf("group keys = %q/%q", grouped.Groups[0].Key, grouped.Groups[1].Key)
	}
}

func TestQueryBreakdown(t *testing.T) {
	base := int64(1719997200000)
	store := seedQueryStore(t, base)
	filter := mustFilter(t, "", base+48*60*60*1000)
	filter.FromMs = base
	filter.ToMs = base + 48*60*60*1000
	items, total, err := store.QueryBreakdown(filter, "model", "requests", "desc", 20, 0)
	if err != nil {
		t.Fatalf("breakdown: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("total/items = %d/%d, want 2/2", total, len(items))
	}
	if items[0].Key != "gpt-5" || items[0].Requests != 4 {
		t.Fatalf("first = %q/%d", items[0].Key, items[0].Requests)
	}
	if items[0].Share != 0.8 {
		t.Fatalf("share = %v, want 0.8", items[0].Share)
	}
	// Sort by cost ascending, limit 1, offset 1.
	paged, total2, err := store.QueryBreakdown(filter, "model", "cost", "asc", 1, 1)
	if err != nil || total2 != 2 || len(paged) != 1 {
		t.Fatalf("paged = %d/%d, %v", total2, len(paged), err)
	}
	if _, _, err := store.QueryBreakdown(filter, "bogus", "", "", 0, 0); err == nil {
		t.Fatal("bogus group accepted")
	}
	if _, _, err := store.QueryBreakdown(filter, "model", "bogus", "", 0, 0); err == nil {
		t.Fatal("bogus sort accepted")
	}
}

func TestQueryEventsPagingAndSearch(t *testing.T) {
	base := int64(1719997200000)
	store := seedQueryStore(t, base)
	filter := mustFilter(t, "", base+48*60*60*1000)
	filter.FromMs = base
	filter.ToMs = base + 48*60*60*1000
	page, err := store.QueryEvents(filter, 2, "", "desc")
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if len(page.Items) != 2 || page.NextCursor == "" {
		t.Fatalf("page = %d cursor %q", len(page.Items), page.NextCursor)
	}
	if page.TotalEstimate != 5 || !page.TotalExact {
		t.Fatalf("total = %d exact %v", page.TotalEstimate, page.TotalExact)
	}
	if page.Items[0].RequestID != "q-e4" {
		t.Fatalf("first = %q, want q-e4", page.Items[0].RequestID)
	}
	page2, err := store.QueryEvents(filter, 2, page.NextCursor, "desc")
	if err != nil {
		t.Fatalf("events page 2: %v", err)
	}
	if len(page2.Items) != 2 || page2.Items[0].RequestID != "q-e3" {
		t.Fatalf("page 2 first = %q", page2.Items[0].RequestID)
	}
	// Equal ts_ms stays stable: q-e1..q-f1 share one hour, cursor splits them.
	if _, _, err := DecodeUsageCursor(page.NextCursor); err != nil {
		t.Fatalf("cursor decode: %v", err)
	}
	if _, err := store.QueryEvents(filter, 2, "bogus!!", "desc"); err == nil {
		t.Fatal("bad cursor accepted")
	}
	if _, err := store.QueryEvents(filter, 500, "", "desc"); err == nil {
		t.Fatal("oversize limit accepted")
	}
	// q search hits fail_summary.
	searched := filter
	searched.Q = "slow down"
	found, err := store.QueryEvents(searched, 50, "", "desc")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(found.Items) != 1 || found.Items[0].RequestID != "q-f1" {
		t.Fatalf("search found %d items", len(found.Items))
	}
	// The query layer returns fail_summary; the list route omits it.
	// Detail carries fail_summary.
	detail, err := store.QueryEvent(found.Items[0].ID)
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail.FailSummary != "slow down" {
		t.Fatalf("detail fail_summary = %q", detail.FailSummary)
	}
	if _, err := store.QueryEvent(999999); err != ErrUsageEventNotFound {
		t.Fatalf("missing detail err = %v", err)
	}
}

func TestQueryFailuresAndHeatmap(t *testing.T) {
	base := int64(1719997200000)
	store := seedQueryStore(t, base)
	filter := mustFilter(t, "", base+48*60*60*1000)
	filter.FromMs = base
	filter.ToMs = base + 48*60*60*1000
	failures, err := store.QueryFailures(filter, 20, 20)
	if err != nil {
		t.Fatalf("failures: %v", err)
	}
	if len(failures.Groups) != 1 || failures.Groups[0].Count != 1 {
		t.Fatalf("groups = %+v", failures.Groups)
	}
	group := failures.Groups[0]
	if group.FailCode != "rate_limit" || group.StatusClass != "5xx" || group.Model != "gpt-5" {
		t.Fatalf("group = %+v", group)
	}
	if len(failures.Recent) != 1 || failures.Recent[0].FailSummary != "slow down" {
		t.Fatalf("recent = %+v", failures.Recent)
	}
	heat, err := store.QueryHeatmap(filter, "requests")
	if err != nil {
		t.Fatalf("heatmap: %v", err)
	}
	var total int64
	for weekday := range 7 {
		for hour := range 24 {
			total += heat.Matrix[weekday][hour]
		}
	}
	if total != 5 {
		t.Fatalf("heatmap total = %d, want 5", total)
	}
	if heat.Max < 2 {
		t.Fatalf("heatmap max = %d, want at least 2", heat.Max)
	}
	if _, err := store.QueryHeatmap(filter, "bogus"); err == nil {
		t.Fatal("bogus metric accepted")
	}
}

func TestQueryPercentiles(t *testing.T) {
	base := int64(1719997200000)
	store := seedQueryStore(t, base)
	filter := mustFilter(t, "", base+48*60*60*1000)
	filter.FromMs = base
	filter.ToMs = base + 48*60*60*1000
	// Latencies in range: 100,100,100,300,500. N=5: p50 rank 3 offset 2
	// = 100; p95 rank 5 offset 4 = 500.
	p50, p95, source, err := store.QueryLatencyPercentiles(filter)
	if err != nil {
		t.Fatalf("percentiles: %v", err)
	}
	if source != "events" || p50 == nil || *p50 != 100 || p95 == nil || *p95 != 500 {
		t.Fatalf("p50/p95/source = %v/%v/%s", p50, p95, source)
	}
	empty := filter
	empty.FromMs = base - 1000
	empty.ToMs = base
	p50, p95, source, err = store.QueryLatencyPercentiles(empty)
	if err != nil || p50 != nil || p95 != nil || source != "events" {
		t.Fatalf("empty percentiles = %v/%v/%s %v", p50, p95, source, err)
	}
}

func TestQueryLabels(t *testing.T) {
	base := int64(1719997200000)
	store := seedQueryStore(t, base)
	lookup := func(authID string) *AuthRef {
		if authID == "codex-1.json" {
			return &AuthRef{Label: "manager label"}
		}
		return nil
	}
	if label := store.CredentialLabel("codex-1.json", lookup); label != "manager label" {
		t.Fatalf("manager label = %q", label)
	}
	if label := store.CredentialLabel("codex-1.json", nil); label != "ab***@example.com" {
		t.Fatalf("stored label = %q", label)
	}
	if label := store.CredentialLabel("missing.json", nil); label != "missing.json" {
		t.Fatalf("fallback label = %q", label)
	}
	if display := APIKeyDisplay("9f2c4a0e11d3b7aa", "my-harness"); display != "my-harness" {
		t.Fatalf("key label = %q", display)
	}
	if display := APIKeyDisplay("9f2c4a0e11d3b7aa", ""); display != "key-9f2c4a" {
		t.Fatalf("key fallback = %q", display)
	}
	kept, err := store.SetKeyLabel("key1", "harness", base)
	if err != nil || !kept {
		t.Fatalf("set label = %v %v", kept, err)
	}
	labels, err := store.KeyLabels()
	if err != nil {
		t.Fatalf("KeyLabels: %v", err)
	}
	if len(labels) != 1 || labels[0].Label != "harness" || !labels[0].Seen {
		t.Fatalf("labels = %+v", labels)
	}
	if _, err := store.SetKeyLabel("key1", "", base); err != nil {
		t.Fatalf("clear label: %v", err)
	}
	cleared, err := store.KeyLabels()
	if err != nil {
		t.Fatalf("KeyLabels after clear: %v", err)
	}
	if len(cleared) != 1 || cleared[0].Label != "" || !cleared[0].Seen {
		t.Fatalf("cleared = %+v", cleared)
	}
}

func TestExportEventsCancel(t *testing.T) {
	base := int64(1719997200000)
	store := seedQueryStore(t, base)
	filter := mustFilter(t, "", base+48*60*60*1000)
	filter.FromMs = base
	filter.ToMs = base + 48*60*60*1000
	var count int
	var first int64
	err := store.ExportEvents(context.Background(), filter, func(event UsageEvent) error {
		if count == 0 {
			first = event.TsMs
		}
		count++
		return nil
	})
	if err != nil || count != 5 {
		t.Fatalf("export count = %d, %v", count, err)
	}
	if first != base+1000 {
		t.Fatalf("export order starts at %d", first)
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	emitErr := store.ExportEvents(cancelled, filter, func(UsageEvent) error { return nil })
	if emitErr == nil {
		t.Fatal("cancelled export succeeded")
	}
}

func TestQuerySQLSafety(t *testing.T) {
	base := int64(1719997200000)
	store := seedQueryStore(t, base)
	filter := mustFilter(t, "", base+48*60*60*1000)
	filter.FromMs = base
	filter.ToMs = base + 48*60*60*1000
	filter.Providers = []string{"codex' OR '1'='1'; DROP TABLE usage_events; --"}
	filter.Models = []string{`"quoted"`, "x'; DELETE FROM usage_hourly; --"}
	if _, err := store.QueryCounters(filter); err != nil {
		t.Fatalf("counters with hostile filter: %v", err)
	}
	filter.Q = "' OR 1=1; --"
	if _, err := store.QueryCounters(filter); err != nil {
		t.Fatalf("search with hostile q: %v", err)
	}
	var events int64
	if err := store.readDB.QueryRow(`SELECT COUNT(*) FROM usage_events`).Scan(&events); err != nil {
		t.Fatalf("count: %v", err)
	}
	if events != 5 {
		t.Fatalf("events after hostile input = %d, want 5", events)
	}
}
