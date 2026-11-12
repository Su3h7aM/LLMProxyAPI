package management

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Su3h7aM/LLMProxyAPI/internal/usagedb"
	"github.com/gin-gonic/gin"
)

// usageQueryStore returns the open usage database. A disabled database gives
// 503 with the contract body.
func (h *Handler) usageQueryStore(c *gin.Context) *usagedb.Store {
	store := usagedb.ActiveStore()
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "usage db disabled"})
		return nil
	}
	return store
}

// usageQueryError writes a filter error: 400 for client errors, 500 after.
func usageQueryError(c *gin.Context, err error) {
	var queryErr *usagedb.QueryError
	if errors.As(err, &queryErr) {
		c.JSON(http.StatusBadRequest, gin.H{"error": queryErr.Msg})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "usage query failed"})
}

// usageFilter parses the common query parameters. nowMs is the range default.
func usageFilter(c *gin.Context) (usagedb.UsageFilter, bool) {
	filter, err := usagedb.ParseUsageFilter(c.Request.URL.Query(), time.Now().UnixMilli())
	if err != nil {
		usageQueryError(c, err)
		return usagedb.UsageFilter{}, false
	}
	return filter, true
}

// rejectSearch refuses q on routes that read the hourly rollup only.
func rejectSearch(c *gin.Context, filter usagedb.UsageFilter) bool {
	if filter.HasQ() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q is supported on events, failures, and export only"})
		return false
	}
	return true
}

// queryIntParam reads an optional integer parameter. A bad value is a 400;
// empty means the default.
func queryIntParam(c *gin.Context, key string, def int) (int, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return def, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + key})
		return 0, false
	}
	return value, true
}

// usageCountersJSON is the counters object shared by several routes.
type usageCountersJSON struct {
	Requests              int64   `json:"requests"`
	Failures              int64   `json:"failures"`
	FailureRate           float64 `json:"failure_rate"`
	InputTokens           int64   `json:"input_tokens"`
	OutputTokens          int64   `json:"output_tokens"`
	ReasoningTokens       int64   `json:"reasoning_tokens"`
	CacheReadTokens       int64   `json:"cache_read_tokens"`
	CacheWriteTokens      int64   `json:"cache_write_tokens"`
	TotalTokens           int64   `json:"total_tokens"`
	CostMicroUSD          int64   `json:"cost_micro_usd"`
	EstimatedCostMicroUSD int64   `json:"estimated_cost_micro_usd"`
	UnpricedRequests      int64   `json:"unpriced_requests"`
	LatencyAvgMs          int64   `json:"latency_avg_ms"`
	LatencyMaxMs          int64   `json:"latency_max_ms"`
	TTFTAvgMs             int64   `json:"ttft_avg_ms"`
}

func toCountersJSON(counters usagedb.UsageCounters) usageCountersJSON {
	return usageCountersJSON{
		Requests:              counters.Requests,
		Failures:              counters.Failures,
		FailureRate:           counters.FailureRate,
		InputTokens:           counters.InputTokens,
		OutputTokens:          counters.OutputTokens,
		ReasoningTokens:       counters.ReasoningTokens,
		CacheReadTokens:       counters.CacheReadTokens,
		CacheWriteTokens:      counters.CacheWriteTokens,
		TotalTokens:           counters.TotalTokens,
		CostMicroUSD:          counters.CostMicroUSD,
		EstimatedCostMicroUSD: counters.EstimatedCostMicroUSD,
		UnpricedRequests:      counters.UnpricedRequests,
		LatencyAvgMs:          counters.LatencyAvgMs,
		LatencyMaxMs:          counters.LatencyMaxMs,
		TTFTAvgMs:             counters.TTFTAvgMs,
	}
}

// usageAuthLookup matches an event auth_id to a manager credential by file
// name, id, or index.
func (h *Handler) usageAuthLookup() usagedb.AuthLookup {
	if h == nil || h.authManager == nil {
		return nil
	}
	auths := h.authManager.List()
	byID := make(map[string]*usagedb.AuthRef, len(auths))
	for _, auth := range auths {
		if auth == nil {
			continue
		}
		ref := &usagedb.AuthRef{Label: auth.Label, Provider: auth.Provider}
		for _, key := range []string{auth.FileName, auth.ID, auth.Index} {
			if trimmed := strings.TrimSpace(key); trimmed != "" {
				byID[trimmed] = ref
			}
		}
	}
	if len(byID) == 0 {
		return nil
	}
	return func(authID string) *usagedb.AuthRef { return byID[authID] }
}

// GetUsageSummary serves totals, previous-period totals, latency
// percentiles, and the oldest stored event.
func (h *Handler) GetUsageSummary(c *gin.Context) {
	store := h.usageQueryStore(c)
	if store == nil {
		return
	}
	filter, ok := usageFilter(c)
	if !ok {
		return
	}
	if !rejectSearch(c, filter) {
		return
	}
	summary, err := store.QuerySummary(filter)
	if err != nil {
		usageQueryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"range":        gin.H{"from_ms": filter.FromMs, "to_ms": filter.ToMs},
		"totals":       toCountersJSON(summary.Totals),
		"previous":     toCountersJSON(summary.Previous),
		"latency":      gin.H{"p50_ms": summary.P50Ms, "p95_ms": summary.P95Ms, "source": summary.LatencySource},
		"first_event_ms": summary.FirstEventMs,
	})
}

// usagePointJSON is one bucket with counters.
type usagePointJSON struct {
	TsMs     int64              `json:"ts_ms"`
	Counters usageCountersJSON `json:"counters"`
}

func toPointJSON(point usagedb.UsagePoint) usagePointJSON {
	return usagePointJSON{TsMs: point.TsMs, Counters: toCountersJSON(point.UsageCounters)}
}

// GetUsageTimeseries serves bucketed counters, optionally grouped.
func (h *Handler) GetUsageTimeseries(c *gin.Context) {
	store := h.usageQueryStore(c)
	if store == nil {
		return
	}
	filter, ok := usageFilter(c)
	if !ok {
		return
	}
	if !rejectSearch(c, filter) {
		return
	}
	groupBy := strings.TrimSpace(c.Query("group_by"))
	top, ok := queryIntParam(c, "top", 5)
	if !ok {
		return
	}
	series, err := store.QueryTimeseries(filter, strings.TrimSpace(c.Query("bucket")), groupBy, top)
	if err != nil {
		usageQueryError(c, err)
		return
	}
	points := make([]usagePointJSON, 0, len(series.Points))
	for _, point := range series.Points {
		points = append(points, toPointJSON(point))
	}
	body := gin.H{
		"bucket": series.Bucket,
		"tz":     filter.TZ,
		"points": points,
	}
	if groupBy != "" {
		keys := make([]string, 0, len(series.Groups))
		for _, group := range series.Groups {
			keys = append(keys, group.Key)
		}
		labels := h.usageGroupLabels(store, groupBy, keys)
		groups := make([]gin.H, 0, len(series.Groups))
		for _, group := range series.Groups {
			groupPoints := make([]usagePointJSON, 0, len(group.Points))
			for _, point := range group.Points {
				groupPoints = append(groupPoints, toPointJSON(point))
			}
			groups = append(groups, gin.H{
				"key":    group.Key,
				"label":  labels[group.Key],
				"points": groupPoints,
			})
		}
		body["groups"] = groups
	}
	c.JSON(http.StatusOK, body)
}

// usageGroupLabels resolves display labels for group keys.
func (h *Handler) usageGroupLabels(store *usagedb.Store, groupBy string, keys []string) map[string]string {
	labels := make(map[string]string, len(keys))
	for _, key := range keys {
		labels[key] = key
	}
	switch groupBy {
	case "credential":
		for id, label := range store.CredentialLabels(keys, h.usageAuthLookup()) {
			labels[id] = label
		}
	case "api_key":
		stored := store.StoredAPIKeyLabels(keys)
		for _, key := range keys {
			labels[key] = usagedb.APIKeyDisplay(key, stored[key])
		}
	}
	return labels
}

// GetUsageBreakdown serves grouped counters with sort and paging.
func (h *Handler) GetUsageBreakdown(c *gin.Context) {
	store := h.usageQueryStore(c)
	if store == nil {
		return
	}
	filter, ok := usageFilter(c)
	if !ok {
		return
	}
	if !rejectSearch(c, filter) {
		return
	}
	groupBy := strings.TrimSpace(c.Query("group_by"))
	limit, ok := queryIntParam(c, "limit", 20)
	if !ok {
		return
	}
	offset, ok := queryIntParam(c, "offset", 0)
	if !ok {
		return
	}
	items, total, err := store.QueryBreakdown(filter, groupBy,
		strings.TrimSpace(c.Query("sort")), strings.TrimSpace(c.Query("order")), limit, offset)
	if err != nil {
		usageQueryError(c, err)
		return
	}
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	labels := h.usageGroupLabels(store, groupBy, keys)
	meta := h.usageGroupMeta(store, filter, groupBy, keys)
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, gin.H{
			"key":      item.Key,
			"label":    labels[item.Key],
			"share":    item.Share,
			"counters": toCountersJSON(item.UsageCounters),
			"meta":     meta[item.Key],
		})
	}
	c.JSON(http.StatusOK, gin.H{"group_by": groupBy, "total": total, "items": out})
}

// usageGroupMeta adds per-group fields: provider for model; auth_type and
// provider for credential; empty object otherwise.
func (h *Handler) usageGroupMeta(store *usagedb.Store, filter usagedb.UsageFilter, groupBy string, keys []string) map[string]gin.H {
	meta := make(map[string]gin.H, len(keys))
	for _, key := range keys {
		meta[key] = gin.H{}
	}
	switch groupBy {
	case "model":
		for model, provider := range store.TopModelProviders(filter, keys) {
			meta[model] = gin.H{"provider": provider}
		}
	case "credential":
		fields := store.LatestAuthFields(keys)
		lookup := h.usageAuthLookup()
		for _, key := range keys {
			provider, authType := fields[key][0], fields[key][1]
			if lookup != nil {
				if ref := lookup(key); ref != nil && strings.TrimSpace(ref.Provider) != "" {
					provider = strings.TrimSpace(ref.Provider)
				}
			}
			entry := gin.H{}
			if provider != "" {
				entry["provider"] = provider
			}
			if authType != "" {
				entry["auth_type"] = authType
			}
			meta[key] = entry
		}
	}
	return meta
}

// usageEventJSON is one event for lists, detail, failures, and export.
type usageEventJSON struct {
	ID               int64   `json:"id"`
	RequestID        string  `json:"request_id"`
	TraceID          string  `json:"trace_id"`
	TsMs             int64   `json:"ts_ms"`
	Provider         string  `json:"provider"`
	Model            string  `json:"model"`
	Alias            string  `json:"alias"`
	Executor         string  `json:"executor"`
	AuthID           string  `json:"auth_id"`
	AuthType         string  `json:"auth_type"`
	AccountLabel     string  `json:"account_label"`
	APIKeyID         string  `json:"api_key_id"`
	APIKeyLabel      string  `json:"api_key_label"`
	Stream           bool    `json:"stream"`
	StatusClass      string  `json:"status_class"`
	Failed           bool    `json:"failed"`
	FailCode         string  `json:"fail_code"`
	FailSummary      string  `json:"fail_summary,omitempty"`
	LatencyMs        int64   `json:"latency_ms"`
	TTFTMs           int64   `json:"ttft_ms"`
	InputTokens      int64   `json:"input_tokens"`
	OutputTokens     int64   `json:"output_tokens"`
	ReasoningTokens  int64   `json:"reasoning_tokens"`
	CacheReadTokens  int64   `json:"cache_read_tokens"`
	CacheWriteTokens int64   `json:"cache_write_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	ServiceTier      string  `json:"service_tier"`
	ReasoningEffort  string  `json:"reasoning_effort"`
	CostMicroUSD     *int64  `json:"cost_micro_usd"`
	CostSource       string  `json:"cost_source"`
	SessionID        string  `json:"session_id,omitempty"`
	AuthIndex        string  `json:"auth_index,omitempty"`
	AccessProvider   string  `json:"access_provider,omitempty"`
}

func toEventJSON(event usagedb.UsageEvent, keyLabel string) usageEventJSON {
	return usageEventJSON{
		ID: event.ID, RequestID: event.RequestID, TraceID: event.TraceID, TsMs: event.TsMs,
		Provider: event.Provider, Model: event.Model, Alias: event.Alias, Executor: event.Executor,
		AuthID: event.AuthID, AuthType: event.AuthType, AccountLabel: event.AccountLabel,
		APIKeyID: event.APIKeyID, APIKeyLabel: keyLabel,
		Stream: event.Stream, StatusClass: event.StatusClass, Failed: event.Failed,
		FailCode: event.FailCode, FailSummary: event.FailSummary,
		LatencyMs: event.LatencyMs, TTFTMs: event.TTFTMs,
		InputTokens: event.InputTokens, OutputTokens: event.OutputTokens,
		ReasoningTokens: event.ReasoningTokens,
		CacheReadTokens: event.CacheReadTokens, CacheWriteTokens: event.CacheWriteTokens,
		TotalTokens: event.TotalTokens,
		ServiceTier: event.ServiceTier, ReasoningEffort: event.ReasoningEffort,
		CostMicroUSD: event.CostMicroUSD, CostSource: event.CostSource,
		SessionID: event.SessionID, AuthIndex: event.AuthIndex, AccessProvider: event.AccessProvider,
	}
}

// usageKeyLabels resolves api_key labels for events.
func usageKeyLabels(store *usagedb.Store, events []usagedb.UsageEvent) map[string]string {
	ids := make([]string, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.APIKeyID)
	}
	stored := store.StoredAPIKeyLabels(ids)
	out := make(map[string]string, len(ids))
	for _, id := range ids {
		out[id] = usagedb.APIKeyDisplay(id, stored[id])
	}
	return out
}

// GetUsageEvents serves one page of raw events without fail_summary.
func (h *Handler) GetUsageEvents(c *gin.Context) {
	store := h.usageQueryStore(c)
	if store == nil {
		return
	}
	filter, ok := usageFilter(c)
	if !ok {
		return
	}
	limit, ok := queryIntParam(c, "limit", 50)
	if !ok {
		return
	}
	result, err := store.QueryEvents(filter, limit,
		strings.TrimSpace(c.Query("cursor")), strings.TrimSpace(c.Query("order")))
	if err != nil {
		usageQueryError(c, err)
		return
	}
	keyLabels := usageKeyLabels(store, result.Items)
	items := make([]usageEventJSON, 0, len(result.Items))
	for _, event := range result.Items {
		item := toEventJSON(event, keyLabels[event.APIKeyID])
		item.FailSummary = ""
		items = append(items, item)
	}
	body := gin.H{"items": items, "next_cursor": result.NextCursor, "total_estimate": result.TotalEstimate}
	if !result.TotalExact {
		body["total_exact"] = false
	}
	c.JSON(http.StatusOK, body)
}

// GetUsageEvent serves one event with fail_summary and price version.
func (h *Handler) GetUsageEvent(c *gin.Context) {
	store := h.usageQueryStore(c)
	if store == nil {
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "usage event not found"})
		return
	}
	event, err := store.QueryEvent(id)
	if err != nil {
		if errors.Is(err, usagedb.ErrUsageEventNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "usage event not found"})
			return
		}
		usageQueryError(c, err)
		return
	}
	keyLabels := usageKeyLabels(store, []usagedb.UsageEvent{event})
	item := toEventJSON(event, keyLabels[event.APIKeyID])
	body := gin.H{
		"id": item.ID, "request_id": item.RequestID, "trace_id": item.TraceID, "ts_ms": item.TsMs,
		"provider": item.Provider, "model": item.Model, "alias": item.Alias, "executor": item.Executor,
		"auth_id": item.AuthID, "auth_type": item.AuthType, "account_label": item.AccountLabel,
		"api_key_id": item.APIKeyID, "api_key_label": item.APIKeyLabel,
		"stream": item.Stream, "status_class": item.StatusClass, "failed": item.Failed,
		"fail_code": item.FailCode, "fail_summary": item.FailSummary,
		"latency_ms": item.LatencyMs, "ttft_ms": item.TTFTMs,
		"input_tokens": item.InputTokens, "output_tokens": item.OutputTokens,
		"reasoning_tokens": item.ReasoningTokens,
		"cache_read_tokens": item.CacheReadTokens, "cache_write_tokens": item.CacheWriteTokens,
		"total_tokens": item.TotalTokens,
		"service_tier": item.ServiceTier, "reasoning_effort": item.ReasoningEffort,
		"cost_micro_usd": item.CostMicroUSD, "cost_source": item.CostSource,
		"session_id": item.SessionID, "auth_index": item.AuthIndex, "access_provider": item.AccessProvider,
		"price": nil,
	}
	if event.PriceID != nil {
		if price, err := store.PriceVersion(*event.PriceID); err == nil {
			body["price"] = gin.H{
				"provider": price.ProviderKey, "model": price.ModelID,
				"valid_from_ms": price.ValidFromMs,
				"input":         price.Input, "output": price.Output,
				"cache_read": price.CacheRead, "cache_write": price.CacheWrite,
				"source": price.Source,
			}
		}
	}
	c.JSON(http.StatusOK, body)
}

// GetUsageFailures serves failure groups plus recent failed events.
func (h *Handler) GetUsageFailures(c *gin.Context) {
	store := h.usageQueryStore(c)
	if store == nil {
		return
	}
	filter, ok := usageFilter(c)
	if !ok {
		return
	}
	limit, ok := queryIntParam(c, "limit", 20)
	if !ok {
		return
	}
	recent, ok := queryIntParam(c, "recent", 20)
	if !ok {
		return
	}
	result, err := store.QueryFailures(filter, limit, recent)
	if err != nil {
		usageQueryError(c, err)
		return
	}
	lookup := h.usageAuthLookup()
	ids := make([]string, 0, len(result.Groups))
	for _, group := range result.Groups {
		ids = append(ids, group.AuthID)
	}
	authLabels := store.CredentialLabels(ids, lookup)
	groups := make([]gin.H, 0, len(result.Groups))
	for _, group := range result.Groups {
		groups = append(groups, gin.H{
			"fail_code": group.FailCode, "status_class": group.StatusClass,
			"provider": group.Provider, "auth_id": group.AuthID,
			"auth_label": authLabels[group.AuthID], "model": group.Model,
			"count": group.Count, "first_ms": group.FirstMs, "last_ms": group.LastMs,
		})
	}
	keyLabels := usageKeyLabels(store, result.Recent)
	recentItems := make([]usageEventJSON, 0, len(result.Recent))
	for _, event := range result.Recent {
		recentItems = append(recentItems, toEventJSON(event, keyLabels[event.APIKeyID]))
	}
	c.JSON(http.StatusOK, gin.H{"groups": groups, "recent": recentItems})
}

// GetUsageHeatmap serves the hour-of-week matrix.
func (h *Handler) GetUsageHeatmap(c *gin.Context) {
	store := h.usageQueryStore(c)
	if store == nil {
		return
	}
	filter, ok := usageFilter(c)
	if !ok {
		return
	}
	if !rejectSearch(c, filter) {
		return
	}
	metric := strings.TrimSpace(c.Query("metric"))
	if metric == "" {
		metric = "requests"
	}
	heat, err := store.QueryHeatmap(filter, metric)
	if err != nil {
		usageQueryError(c, err)
		return
	}
	matrix := make([][]int64, 0, 7)
	for weekday := range 7 {
		row := make([]int64, 0, 24)
		row = append(row, heat.Matrix[weekday][:]...)
		matrix = append(matrix, row)
	}
	c.JSON(http.StatusOK, gin.H{"tz": filter.TZ, "metric": metric, "matrix": matrix, "max": heat.Max})
}

// usageExportJSON is one export line: list fields plus fail_summary,
// always present even when empty.
type usageExportJSON struct {
	usageEventJSON
	FailSummary string `json:"fail_summary"`
}

// csvFormulaGuard prefixes a single quote when a text cell starts with a
// spreadsheet formula trigger, so exports never run code on open.
func csvFormulaGuard(cell string) string {
	if cell == "" {
		return cell
	}
	switch cell[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + cell
	}
	return cell
}

// usageExportHeader lists CSV columns: list fields plus fail_summary.
func usageExportHeader() []string {
	return []string{
		"id", "request_id", "trace_id", "ts_ms", "provider", "model", "alias",
		"executor", "auth_id", "auth_type", "account_label", "api_key_id",
		"api_key_label", "stream", "status_class", "failed", "fail_code",
		"latency_ms", "ttft_ms", "input_tokens", "output_tokens",
		"reasoning_tokens", "cache_read_tokens", "cache_write_tokens",
		"total_tokens", "service_tier", "reasoning_effort",
		"cost_micro_usd", "cost_source", "fail_summary",
	}
}

// usageExportRecord renders one event as CSV cells.
func usageExportRecord(item usageEventJSON) []string {
	boolText := func(value bool) string {
		if value {
			return "true"
		}
		return "false"
	}
	cost := ""
	if item.CostMicroUSD != nil {
		cost = strconv.FormatInt(*item.CostMicroUSD, 10)
	}
	return []string{
		strconv.FormatInt(item.ID, 10),
		csvFormulaGuard(item.RequestID),
		csvFormulaGuard(item.TraceID),
		strconv.FormatInt(item.TsMs, 10),
		csvFormulaGuard(item.Provider),
		csvFormulaGuard(item.Model),
		csvFormulaGuard(item.Alias),
		csvFormulaGuard(item.Executor),
		csvFormulaGuard(item.AuthID),
		csvFormulaGuard(item.AuthType),
		csvFormulaGuard(item.AccountLabel),
		csvFormulaGuard(item.APIKeyID),
		csvFormulaGuard(item.APIKeyLabel),
		boolText(item.Stream),
		csvFormulaGuard(item.StatusClass),
		boolText(item.Failed),
		csvFormulaGuard(item.FailCode),
		strconv.FormatInt(item.LatencyMs, 10),
		strconv.FormatInt(item.TTFTMs, 10),
		strconv.FormatInt(item.InputTokens, 10),
		strconv.FormatInt(item.OutputTokens, 10),
		strconv.FormatInt(item.ReasoningTokens, 10),
		strconv.FormatInt(item.CacheReadTokens, 10),
		strconv.FormatInt(item.CacheWriteTokens, 10),
		strconv.FormatInt(item.TotalTokens, 10),
		csvFormulaGuard(item.ServiceTier),
		csvFormulaGuard(item.ReasoningEffort),
		cost,
		csvFormulaGuard(item.CostSource),
		csvFormulaGuard(item.FailSummary),
	}
}

// GetUsageExport streams matching events as JSONL or CSV, ordered by ts_ms.
func (h *Handler) GetUsageExport(c *gin.Context) {
	store := h.usageQueryStore(c)
	if store == nil {
		return
	}
	filter, ok := usageFilter(c)
	if !ok {
		return
	}
	format := strings.TrimSpace(c.Query("format"))
	if format == "" {
		format = "jsonl"
	}
	if format != "jsonl" && format != "csv" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format"})
		return
	}
	ext := "jsonl"
	contentType := "application/x-ndjson"
	if format == "csv" {
		ext = "csv"
		contentType = "text/csv; charset=utf-8"
	}
	c.Header("Content-Disposition", "attachment; filename=usage-"+
		strconv.FormatInt(filter.FromMs, 10)+"-"+strconv.FormatInt(filter.ToMs, 10)+"."+ext)
	c.Header("Content-Type", contentType)
	c.Status(http.StatusOK)
	flusher, _ := c.Writer.(interface{ Flush() })
	flush := func() {
		if flusher != nil {
			flusher.Flush()
		}
	}
	if format == "csv" {
		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()
		if err := writer.Write(usageExportHeader()); err != nil {
			return
		}
		emitErr := store.ExportEvents(c.Request.Context(), filter, func(event usagedb.UsageEvent) error {
			keyLabels := usageKeyLabels(store, []usagedb.UsageEvent{event})
			item := toEventJSON(event, keyLabels[event.APIKeyID])
			if err := writer.Write(usageExportRecord(item)); err != nil {
				return err
			}
			writer.Flush()
			flush()
			return nil
		})
		_ = emitErr
		return
	}
	encoder := json.NewEncoder(c.Writer)
	emitErr := store.ExportEvents(c.Request.Context(), filter, func(event usagedb.UsageEvent) error {
		keyLabels := usageKeyLabels(store, []usagedb.UsageEvent{event})
		item := toEventJSON(event, keyLabels[event.APIKeyID])
		// Export always carries fail_summary, even when empty.
		if err := encoder.Encode(usageExportJSON{
			usageEventJSON: item,
			FailSummary:    item.FailSummary,
		}); err != nil {
			return err
		}
		flush()
		return nil
	})
	_ = emitErr
}

// GetUsageKeyLabels lists stored key labels plus seen ids without a label.
func (h *Handler) GetUsageKeyLabels(c *gin.Context) {
	store := h.usageQueryStore(c)
	if store == nil {
		return
	}
	labels, err := store.KeyLabels()
	if err != nil {
		usageQueryError(c, err)
		return
	}
	out := make([]gin.H, 0, len(labels))
	for _, item := range labels {
		out = append(out, gin.H{
			"api_key_id": item.APIKeyID, "label": item.Label, "seen": item.Seen,
		})
	}
	c.JSON(http.StatusOK, gin.H{"labels": out})
}

// usageKeyLabelBody is the PUT key-labels body.
type usageKeyLabelBody struct {
	APIKeyID string `json:"api_key_id"`
	Label    string `json:"label"`
}

// PutUsageKeyLabel sets or clears one key label and returns the updated item.
func (h *Handler) PutUsageKeyLabel(c *gin.Context) {
	store := h.usageQueryStore(c)
	if store == nil {
		return
	}
	var body usageKeyLabelBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if strings.TrimSpace(body.APIKeyID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_key_id is required"})
		return
	}
	if _, err := store.SetKeyLabel(body.APIKeyID, body.Label, time.Now().UnixMilli()); err != nil {
		usageQueryError(c, err)
		return
	}
	labels, err := store.KeyLabels()
	if err != nil {
		usageQueryError(c, err)
		return
	}
	for _, item := range labels {
		if item.APIKeyID == strings.TrimSpace(body.APIKeyID) {
			c.JSON(http.StatusOK, gin.H{
				"api_key_id": item.APIKeyID, "label": item.Label, "seen": item.Seen,
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"api_key_id": strings.TrimSpace(body.APIKeyID), "label": "", "seen": false,
	})
}

// PostUsageMaintenance runs one maintenance action: prune, vacuum, or
// checkpoint.
func (h *Handler) PostUsageMaintenance(c *gin.Context) {
	store := h.usageQueryStore(c)
	if store == nil {
		return
	}
	var body struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	start := time.Now()
	switch strings.TrimSpace(body.Action) {
	case "prune":
		deleted, err := store.PruneRetention(time.Now().UnixMilli())
		if err != nil {
			usageQueryError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"action": "prune", "deleted": deleted,
			"duration_ms": time.Since(start).Milliseconds(),
		})
	case "vacuum":
		if err := store.Vacuum(); err != nil {
			usageQueryError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"action": "vacuum", "duration_ms": time.Since(start).Milliseconds(),
		})
	case "checkpoint":
		result, err := store.Checkpoint()
		if err != nil {
			usageQueryError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"action": "checkpoint", "busy": result.Busy,
			"log_frames": result.LogFrames, "checkpointed": result.Checkpointed,
			"duration_ms": time.Since(start).Milliseconds(),
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
	}
}
