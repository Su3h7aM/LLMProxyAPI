package management

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Su3h7aM/LLMProxyAPI/internal/config"
	"github.com/Su3h7aM/LLMProxyAPI/internal/usagedb"
	coreauth "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/auth"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// seedUsageQueryRows stores four known events: two priced upstream (e1, e2),
// one late-priced (e3, failed), one unpriced (e4, another model and key).
// It returns the base timestamp.
func seedUsageQueryRows(t *testing.T) int64 {
	t.Helper()
	store := usagedb.ActiveStore()
	if store == nil {
		t.Fatal("no active usage store")
	}
	base := int64(1719997200000)
	mkRow := func(id string, ts int64) usagedb.UsageRow {
		return usagedb.UsageRow{
			RequestID: id, TraceID: "trace-" + id, TsMs: ts,
			HourMs:       ts / 3600000 * 3600000,
			Provider:     "codex",
			Model:        "uq-model-5",
			Alias:        "uq-model-5",
			Executor:     "codex",
			AuthID:       "codex-1.json",
			AuthType:     "oauth",
			AccountLabel: "ab***@example.com",
			APIKeyID:     "9f2c4a0e11d3b7aa",
			Stream:       false,
			StatusClass:  "2xx",
			LatencyMs:    100,
			TTFTMs:       30,
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
			ServiceTier:  "default",
		}
	}
	upstream := int64(100)
	e1 := mkRow("uq-e1", base+1000)
	e1.UpstreamCostMicroUSD = &upstream
	e2 := mkRow("uq-e2", base+2000)
	e2.Stream = true
	e2.UpstreamCostMicroUSD = &upstream
	e3 := mkRow("uq-e3", base+3000)
	e3.Failed = true
	e3.StatusClass = "5xx"
	e3.FailCode = "rate_limit"
	e3.FailSummary = `=2+2, "quoted"`
	e3.LatencyMs = 500
	e4 := mkRow("uq-e4", base+24*60*60*1000+1000)
	e4.Model = "uq-model-6"
	e4.Alias = "uq-model-6"
	e4.APIKeyID = "b3c4d5e6f7081920"
	if _, err := store.InsertBatch([]usagedb.UsageRow{e1, e2, e3, e4}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	// Late price for uq-model-5 fills e3 only (e1, e2 already have a cost).
	if _, err := store.AddPriceVersion(usagedb.Price{
		ProviderKey: "openai", ModelID: "uq-model-5", ValidFromMs: 0,
		Input: 1000000, Output: 2000000, Source: usagedb.PriceSourceModelsDev,
	}); err != nil {
		t.Fatalf("AddPriceVersion: %v", err)
	}
	if _, err := store.FillUnpricedForKeys([][2]string{{"openai", "uq-model-5"}}); err != nil {
		t.Fatalf("FillUnpricedForKeys: %v", err)
	}
	return base
}

func getUsage(t *testing.T, h *Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec, ginCtx := usagePriceContext(t, http.MethodGet, target, nil)
	switch {
	case strings.HasPrefix(target, "/v8/management/observability/usage/summary"):
		h.GetUsageSummary(ginCtx)
	case strings.HasPrefix(target, "/v8/management/observability/usage/timeseries"):
		h.GetUsageTimeseries(ginCtx)
	case strings.HasPrefix(target, "/v8/management/observability/usage/breakdown"):
		h.GetUsageBreakdown(ginCtx)
	case strings.HasPrefix(target, "/v8/management/observability/usage/events/"):
		ginCtx.Params = gin.Params{{Key: "id", Value: strings.TrimPrefix(target, "/v8/management/observability/usage/events/")}}
		h.GetUsageEvent(ginCtx)
	case strings.HasPrefix(target, "/v8/management/observability/usage/events"):
		h.GetUsageEvents(ginCtx)
	case strings.HasPrefix(target, "/v8/management/observability/usage/failures"):
		h.GetUsageFailures(ginCtx)
	case strings.HasPrefix(target, "/v8/management/observability/usage/heatmap"):
		h.GetUsageHeatmap(ginCtx)
	case strings.HasPrefix(target, "/v8/management/observability/usage/export"):
		h.GetUsageExport(ginCtx)
	case strings.HasPrefix(target, "/v8/management/observability/usage/key-labels"):
		h.GetUsageKeyLabels(ginCtx)
	default:
		t.Fatalf("unknown usage target %s", target)
	}
	return rec
}

func TestUsageQueryDisabled(t *testing.T) {
	withUsageDB(t, 90)
	usagedb.Shutdown(5 * time.Second)
	h := &Handler{}
	targets := []string{
		"/v8/management/observability/usage/summary",
		"/v8/management/observability/usage/timeseries",
		"/v8/management/observability/usage/breakdown",
		"/v8/management/observability/usage/events",
		"/v8/management/observability/usage/events/1",
		"/v8/management/observability/usage/failures",
		"/v8/management/observability/usage/heatmap",
		"/v8/management/observability/usage/export",
		"/v8/management/observability/usage/key-labels",
	}
	for _, target := range targets {
		if rec := getUsage(t, h, target); rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("GET %s status = %d, want 503", target, rec.Code)
		} else if strings.TrimSpace(rec.Body.String()) != `{"error":"usage db disabled"}` {
			t.Fatalf("GET %s body = %s, want contract 503 body", target, rec.Body.String())
		}
	}
	rec, ginCtx := usagePriceContext(t, http.MethodPut,
		"/v8/management/observability/usage/key-labels", []byte(`{"api_key_id":"x","label":"y"}`))
	h.PutUsageKeyLabel(ginCtx)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("PUT key-labels status = %d, want 503", rec.Code)
	}
	rec, ginCtx = usagePriceContext(t, http.MethodPost,
		"/v8/management/observability/usage/maintenance", []byte(`{"action":"vacuum"}`))
	h.PostUsageMaintenance(ginCtx)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST maintenance status = %d, want 503", rec.Code)
	}
	withUsageDB(t, 90)
}

func withRange(target string, base int64) string {
	sep := "?"
	if strings.Contains(target, "?") {
		sep = "&"
	}
	return target + sep + "from_ms=" + strconv.FormatInt(base, 10) +
		"&to_ms=" + strconv.FormatInt(base+48*60*60*1000, 10)
}

func formatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}

func TestUsageSummaryRoute(t *testing.T) {
	withUsageDB(t, 90)
	base := seedUsageQueryRows(t)
	h := &Handler{}
	rec := getUsage(t, h, withRange("/v8/management/observability/usage/summary", base))
	if rec.Code != http.StatusOK {
		t.Fatalf("summary status = %d body=%s", rec.Code, rec.Body.String())
	}
	payload := decodePriceBody(t, rec)
	totals, _ := payload["totals"].(map[string]any)
	if totals["requests"] != float64(4) || totals["failures"] != float64(1) {
		t.Fatalf("totals = %v", totals)
	}
	previous, _ := payload["previous"].(map[string]any)
	if previous["requests"] != float64(0) {
		t.Fatalf("previous = %v", previous)
	}
	latency, _ := payload["latency"].(map[string]any)
	if latency["source"] != "events" || latency["p50_ms"] != float64(100) || latency["p95_ms"] != float64(500) {
		t.Fatalf("latency = %v", latency)
	}
	if payload["first_event_ms"] != float64(base+1000) {
		t.Fatalf("first_event_ms = %v", payload["first_event_ms"])
	}
	rng, _ := payload["range"].(map[string]any)
	if rng["from_ms"] != float64(base) {
		t.Fatalf("range = %v", rng)
	}
	// Bad range and bad enum give 400.
	for _, target := range []string{
		"/v8/management/observability/usage/summary?from_ms=5&to_ms=5",
		"/v8/management/observability/usage/summary?from_ms=0&to_ms=40000000000",
		"/v8/management/observability/usage/summary?status=bogus",
		"/v8/management/observability/usage/summary?q=x",
	} {
		if rec := getUsage(t, h, target); rec.Code != http.StatusBadRequest {
			t.Fatalf("GET %s status = %d, want 400", target, rec.Code)
		}
	}
}

func TestUsageTimeseriesRoute(t *testing.T) {
	withUsageDB(t, 90)
	base := seedUsageQueryRows(t)
	h := &Handler{}
	rec := getUsage(t, h, withRange("/v8/management/observability/usage/timeseries?bucket=day&group_by=model&top=5", base))
	if rec.Code != http.StatusOK {
		t.Fatalf("timeseries status = %d body=%s", rec.Code, rec.Body.String())
	}
	payload := decodePriceBody(t, rec)
	if payload["bucket"] != "day" || payload["tz"] != "UTC" {
		t.Fatalf("bucket/tz = %v/%v", payload["bucket"], payload["tz"])
	}
	points, _ := payload["points"].([]any)
	var total float64
	for _, point := range points {
		if counters, ok := point.(map[string]any)["counters"].(map[string]any); ok {
			total += counters["requests"].(float64)
		}
	}
	if total != 4 {
		t.Fatalf("series total = %v, want 4", total)
	}
	groups, _ := payload["groups"].([]any)
	if len(groups) != 2 {
		t.Fatalf("groups = %d, want 2", len(groups))
	}
	first := groups[0].(map[string]any)
	if first["key"] != "uq-model-5" || first["label"] != "uq-model-5" {
		t.Fatalf("first group = %v", first)
	}
	if _, ok := payload["groups"]; !ok {
		t.Fatal("groups missing")
	}
	rec = getUsage(t, h, withRange("/v8/management/observability/usage/timeseries", base))
	if rec.Code != http.StatusOK {
		t.Fatalf("plain timeseries status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), `"groups"`) {
		t.Fatal("groups present without group_by")
	}
	for _, target := range []string{
		"/v8/management/observability/usage/timeseries?bucket=bogus",
		"/v8/management/observability/usage/timeseries?group_by=bogus",
		"/v8/management/observability/usage/timeseries?top=99",
	} {
		if rec := getUsage(t, h, withRange(target, base)); rec.Code != http.StatusBadRequest {
			t.Fatalf("GET %s status = %d, want 400", target, rec.Code)
		}
	}
}

func TestUsageBreakdownRoute(t *testing.T) {
	withUsageDB(t, 90)
	base := seedUsageQueryRows(t)
	h := &Handler{}
	rec := getUsage(t, h, withRange("/v8/management/observability/usage/breakdown?group_by=credential", base))
	if rec.Code != http.StatusOK {
		t.Fatalf("breakdown status = %d body=%s", rec.Code, rec.Body.String())
	}
	payload := decodePriceBody(t, rec)
	if payload["total"] != float64(1) {
		t.Fatalf("total = %v", payload)
	}
	items, _ := payload["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %d", len(items))
	}
	item := items[0].(map[string]any)
	if item["key"] != "codex-1.json" || item["label"] != "ab***@example.com" {
		t.Fatalf("item key/label = %v/%v", item["key"], item["label"])
	}
	meta, _ := item["meta"].(map[string]any)
	if meta["provider"] != "codex" || meta["auth_type"] != "oauth" {
		t.Fatalf("meta = %v", meta)
	}
	// Key labels apply to api_key groups.
	rec, ginCtx := usagePriceContext(t, http.MethodPut,
		"/v8/management/observability/usage/key-labels",
		[]byte(`{"api_key_id":"9f2c4a0e11d3b7aa","label":"my-harness"}`))
	h.PutUsageKeyLabel(ginCtx)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT label status = %d body=%s", rec.Code, rec.Body.String())
	}
	rec = getUsage(t, h, withRange("/v8/management/observability/usage/breakdown?group_by=api_key", base))
	listed := decodePriceBody(t, rec)
	keyItems, _ := listed["items"].([]any)
	found := false
	for _, entry := range keyItems {
		row := entry.(map[string]any)
		if row["key"] == "9f2c4a0e11d3b7aa" && row["label"] == "my-harness" {
			found = true
		}
		if row["key"] == "b3c4d5e6f7081920" && row["label"] != "key-b3c4d5" {
			t.Fatalf("unlabeled key label = %v", row["label"])
		}
	}
	if !found {
		t.Fatalf("labeled key missing in %v", keyItems)
	}
	for _, target := range []string{
		"/v8/management/observability/usage/breakdown",
		"/v8/management/observability/usage/breakdown?group_by=model&sort=bogus",
		"/v8/management/observability/usage/breakdown?group_by=model&order=sideways",
		"/v8/management/observability/usage/breakdown?group_by=model&limit=zzz",
	} {
		if rec := getUsage(t, h, withRange(target, base)); rec.Code != http.StatusBadRequest {
			t.Fatalf("GET %s status = %d, want 400", target, rec.Code)
		}
	}
}

func TestUsageCredentialLabelFromManager(t *testing.T) {
	withUsageDB(t, 90)
	base := seedUsageQueryRows(t)
	manager := coreauth.NewManager(nil, nil, nil)
	if _, err := manager.Register(context.Background(), &coreauth.Auth{
		ID: "auth-codex-1", FileName: "codex-1.json",
		Label: "Manager Credential", Provider: "codex",
	}); err != nil {
		t.Fatalf("register auth: %v", err)
	}
	h := &Handler{authManager: manager}
	rec := getUsage(t, h, withRange("/v8/management/observability/usage/breakdown?group_by=credential", base))
	if rec.Code != http.StatusOK {
		t.Fatalf("breakdown status = %d body=%s", rec.Code, rec.Body.String())
	}
	payload := decodePriceBody(t, rec)
	items, _ := payload["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	item := items[0].(map[string]any)
	if item["key"] != "codex-1.json" || item["label"] != "Manager Credential" {
		t.Fatalf("item key/label = %v/%v, want codex-1.json/Manager Credential", item["key"], item["label"])
	}
	meta, _ := item["meta"].(map[string]any)
	if meta["provider"] != "codex" {
		t.Fatalf("meta provider = %v, want codex", meta["provider"])
	}
	// The failures route uses the same lookup for auth_label.
	rec = getUsage(t, h, withRange("/v8/management/observability/usage/failures", base))
	if rec.Code != http.StatusOK {
		t.Fatalf("failures status = %d body=%s", rec.Code, rec.Body.String())
	}
	failed := decodePriceBody(t, rec)
	groups, _ := failed["groups"].([]any)
	if len(groups) != 1 {
		t.Fatalf("failure groups = %d, want 1", len(groups))
	}
	group := groups[0].(map[string]any)
	if group["auth_label"] != "Manager Credential" {
		t.Fatalf("failure auth_label = %v, want Manager Credential", group["auth_label"])
	}
	// Without a manager the stored account label applies.
	plain := &Handler{}
	rec = getUsage(t, plain, withRange("/v8/management/observability/usage/breakdown?group_by=credential", base))
	plainPayload := decodePriceBody(t, rec)
	plainItems, _ := plainPayload["items"].([]any)
	if len(plainItems) != 1 || plainItems[0].(map[string]any)["label"] != "ab***@example.com" {
		t.Fatalf("fallback label = %v", plainItems)
	}
}

func TestUsageEventsRoute(t *testing.T) {
	withUsageDB(t, 90)
	base := seedUsageQueryRows(t)
	h := &Handler{}
	rec := getUsage(t, h, withRange("/v8/management/observability/usage/events?limit=2", base))
	if rec.Code != http.StatusOK {
		t.Fatalf("events status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, `"fail_summary"`) {
		t.Fatal("list contains fail_summary")
	}
	if strings.Contains(body, `"api_key":`) {
		t.Fatal("list leaks a raw api_key field")
	}
	payload := decodePriceBody(t, rec)
	items, _ := payload["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	first := items[0].(map[string]any)
	if first["request_id"] != "uq-e4" {
		t.Fatalf("first = %v, want uq-e4", first["request_id"])
	}
	if first["api_key_label"] != "key-b3c4d5" {
		t.Fatalf("key label = %v", first["api_key_label"])
	}
	if payload["total_estimate"] != float64(4) {
		t.Fatalf("total = %v", payload["total_estimate"])
	}
	if _, ok := payload["total_exact"]; ok {
		t.Fatal("total_exact present for an exact total")
	}
	cursor, _ := payload["next_cursor"].(string)
	if cursor == "" {
		t.Fatal("next_cursor empty")
	}
	rec = getUsage(t, h, withRange("/v8/management/observability/usage/events?limit=2", base)+"&cursor="+cursor)
	next := decodePriceBody(t, rec)
	if nextItems, _ := next["items"].([]any); len(nextItems) != 2 {
		t.Fatalf("page 2 items = %d, want 2", len(nextItems))
	}
	rec = getUsage(t, h, withRange("/v8/management/observability/usage/events?order=asc&limit=10", base))
	asc := decodePriceBody(t, rec)
	if ascItems, _ := asc["items"].([]any); len(ascItems) != 4 ||
		ascItems[0].(map[string]any)["request_id"] != "uq-e1" {
		t.Fatalf("asc items = %v", ascItems)
	}
	for _, target := range []string{
		"/v8/management/observability/usage/events?cursor=bogus!!",
		"/v8/management/observability/usage/events?limit=500",
		"/v8/management/observability/usage/events?order=sideways",
	} {
		if rec := getUsage(t, h, withRange(target, base)); rec.Code != http.StatusBadRequest {
			t.Fatalf("GET %s status = %d, want 400", target, rec.Code)
		}
	}
	// Text search hits the failure text.
	rec = getUsage(t, h, withRange("/v8/management/observability/usage/events?q=2%2B2", base))
	found := decodePriceBody(t, rec)
	if foundItems, _ := found["items"].([]any); len(foundItems) != 1 {
		t.Fatalf("search items = %v", foundItems)
	}
}

func eventIDByRequest(t *testing.T, h *Handler, base int64, requestID string) string {
	t.Helper()
	rec := getUsage(t, h, withRange("/v8/management/observability/usage/events?order=asc&limit=100", base))
	payload := decodePriceBody(t, rec)
	items, _ := payload["items"].([]any)
	for _, entry := range items {
		row := entry.(map[string]any)
		if row["request_id"] == requestID {
			if id, ok := row["id"].(float64); ok {
				return formatInt(int64(id))
			}
		}
	}
	t.Fatalf("event %s not found", requestID)
	return ""
}

func TestUsageEventDetail(t *testing.T) {
	withUsageDB(t, 90)
	base := seedUsageQueryRows(t)
	h := &Handler{}
	id := eventIDByRequest(t, h, base, "uq-e3")
	rec := getUsage(t, h, "/v8/management/observability/usage/events/"+id)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail status = %d body=%s", rec.Code, rec.Body.String())
	}
	payload := decodePriceBody(t, rec)
	if payload["fail_summary"] != "=2+2, \"quoted\"" {
		t.Fatalf("fail_summary = %v", payload["fail_summary"])
	}
	price, _ := payload["price"].(map[string]any)
	if price["provider"] != "openai" || price["model"] != "uq-model-5" || price["source"] != "models.dev" {
		t.Fatalf("price = %v", price)
	}
	if payload["cost_source"] != "price-book-late" {
		t.Fatalf("cost_source = %v", payload["cost_source"])
	}
	if _, ok := payload["session_id"]; !ok {
		t.Fatal("session_id missing from detail")
	}
	// Unpriced event has a null price.
	plainID := eventIDByRequest(t, h, base, "uq-e4")
	rec = getUsage(t, h, "/v8/management/observability/usage/events/"+plainID)
	plain := decodePriceBody(t, rec)
	if plain["price"] != nil || plain["cost_micro_usd"] != nil {
		t.Fatalf("unpriced detail price/cost = %v/%v", plain["price"], plain["cost_micro_usd"])
	}
	for _, target := range []string{
		"/v8/management/observability/usage/events/999999999",
		"/v8/management/observability/usage/events/bogus",
	} {
		if rec := getUsage(t, h, target); rec.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want 404", target, rec.Code)
		}
	}
}

func TestUsageFailuresRoute(t *testing.T) {
	withUsageDB(t, 90)
	base := seedUsageQueryRows(t)
	h := &Handler{}
	rec := getUsage(t, h, withRange("/v8/management/observability/usage/failures", base))
	if rec.Code != http.StatusOK {
		t.Fatalf("failures status = %d body=%s", rec.Code, rec.Body.String())
	}
	payload := decodePriceBody(t, rec)
	groups, _ := payload["groups"].([]any)
	if len(groups) != 1 {
		t.Fatalf("groups = %d", len(groups))
	}
	group := groups[0].(map[string]any)
	if group["fail_code"] != "rate_limit" || group["count"] != float64(1) {
		t.Fatalf("group = %v", group)
	}
	if group["auth_label"] != "ab***@example.com" {
		t.Fatalf("auth_label = %v", group["auth_label"])
	}
	recent, _ := payload["recent"].([]any)
	if len(recent) != 1 {
		t.Fatalf("recent = %d", len(recent))
	}
	if recent[0].(map[string]any)["fail_summary"] != "=2+2, \"quoted\"" {
		t.Fatalf("recent item = %v", recent[0])
	}
	if rec := getUsage(t, h, withRange("/v8/management/observability/usage/failures?limit=500", base)); rec.Code != http.StatusBadRequest {
		t.Fatalf("oversize limit status = %d, want 400", rec.Code)
	}
}

func TestUsageHeatmapRoute(t *testing.T) {
	withUsageDB(t, 90)
	base := seedUsageQueryRows(t)
	h := &Handler{}
	rec := getUsage(t, h, withRange("/v8/management/observability/usage/heatmap?metric=cost", base))
	if rec.Code != http.StatusOK {
		t.Fatalf("heatmap status = %d body=%s", rec.Code, rec.Body.String())
	}
	payload := decodePriceBody(t, rec)
	matrix, _ := payload["matrix"].([]any)
	if len(matrix) != 7 {
		t.Fatalf("rows = %d, want 7", len(matrix))
	}
	var total float64
	for weekday, row := range matrix {
		cells, _ := row.([]any)
		if len(cells) != 24 {
			t.Fatalf("row %d cells = %d, want 24", weekday, len(cells))
		}
		for _, cell := range cells {
			total += cell.(float64)
		}
	}
	// Cost: 2x100 upstream plus the late event cost (10 in at 1.0/M and 20
	// out at 2.0/M = 50).
	if total != 250 {
		t.Fatalf("heatmap cost total = %v, want 250", total)
	}
	if payload["tz"] != "UTC" || payload["metric"] != "cost" {
		t.Fatalf("tz/metric = %v/%v", payload["tz"], payload["metric"])
	}
	if rec := getUsage(t, h, withRange("/v8/management/observability/usage/heatmap?metric=bogus", base)); rec.Code != http.StatusBadRequest {
		t.Fatalf("bogus metric status = %d, want 400", rec.Code)
	}
}

func TestUsageExportRoute(t *testing.T) {
	withUsageDB(t, 90)
	base := seedUsageQueryRows(t)
	h := &Handler{}
	rec := getUsage(t, h, withRange("/v8/management/observability/usage/export", base))
	if rec.Code != http.StatusOK {
		t.Fatalf("export status = %d body=%s", rec.Code, rec.Body.String())
	}
	if disposition := rec.Header().Get("Content-Disposition"); !strings.Contains(disposition, ".jsonl") {
		t.Fatalf("disposition = %q", disposition)
	}
	lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("jsonl lines = %d, want 4", len(lines))
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("parse jsonl: %v", err)
	}
	if _, ok := first["fail_summary"]; !ok {
		t.Fatal("jsonl misses fail_summary")
	}
	if first["request_id"] != "uq-e1" {
		t.Fatalf("jsonl order starts at %v", first["request_id"])
	}
	// CSV quotes and guards formula cells.
	rec = getUsage(t, h, withRange("/v8/management/observability/usage/export?format=csv", base))
	if rec.Code != http.StatusOK {
		t.Fatalf("csv status = %d body=%s", rec.Code, rec.Body.String())
	}
	reader := csv.NewReader(strings.NewReader(rec.Body.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(records) != 5 {
		t.Fatalf("csv rows = %d, want header + 4", len(records))
	}
	header := records[0]
	if header[0] != "id" || header[len(header)-1] != "fail_summary" {
		t.Fatalf("csv header = %v", header)
	}
	var guarded bool
	for _, row := range records[1:] {
		if row[1] == "uq-e3" && row[len(row)-1] == "'=2+2, \"quoted\"" {
			guarded = true
		}
	}
	if !guarded {
		t.Fatalf("formula guard missing in %v", records)
	}
	if rec := getUsage(t, h, withRange("/v8/management/observability/usage/export?format=xml", base)); rec.Code != http.StatusBadRequest {
		t.Fatalf("bogus format status = %d, want 400", rec.Code)
	}
}

func TestUsageExportLargeRange(t *testing.T) {
	withUsageDB(t, 90)
	store := usagedb.ActiveStore()
	base := int64(1719997200000)
	var rows []usagedb.UsageRow
	for i := range 2500 {
		rows = append(rows, usagedb.UsageRow{
			RequestID: "big-" + formatInt(int64(i)), TsMs: base + int64(i)*1000,
			HourMs:      (base + int64(i)*1000) / 3600000 * 3600000,
			Provider:    "codex",
			Model:       "uq-model-5",
			Alias:       "uq-model-5",
			StatusClass: "2xx",
		})
	}
	if _, err := store.InsertBatch(rows); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	h := &Handler{}
	rec := getUsage(t, h, withRange("/v8/management/observability/usage/export?format=jsonl", base))
	if rec.Code != http.StatusOK {
		t.Fatalf("export status = %d", rec.Code)
	}
	lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n")
	if len(lines) != 2500 {
		t.Fatalf("jsonl lines = %d, want 2500", len(lines))
	}
}

func TestUsageKeyLabelsRoute(t *testing.T) {
	withUsageDB(t, 90)
	base := seedUsageQueryRows(t)
	h := &Handler{}
	rec := getUsage(t, h, "/v8/management/observability/usage/key-labels")
	if rec.Code != http.StatusOK {
		t.Fatalf("key-labels status = %d body=%s", rec.Code, rec.Body.String())
	}
	payload := decodePriceBody(t, rec)
	labels, _ := payload["labels"].([]any)
	if len(labels) != 2 {
		t.Fatalf("labels = %d, want 2 seen ids", len(labels))
	}
	for _, entry := range labels {
		row := entry.(map[string]any)
		if row["label"] != "" || row["seen"] != true {
			t.Fatalf("unlabeled row = %v", row)
		}
	}
	// Set then clear one label.
	rec, ginCtx := usagePriceContext(t, http.MethodPut,
		"/v8/management/observability/usage/key-labels",
		[]byte(`{"api_key_id":"9f2c4a0e11d3b7aa","label":"my-harness"}`))
	h.PutUsageKeyLabel(ginCtx)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d body=%s", rec.Code, rec.Body.String())
	}
	updated := decodePriceBody(t, rec)
	if updated["label"] != "my-harness" || updated["seen"] != true {
		t.Fatalf("updated = %v", updated)
	}
	rec, ginCtx = usagePriceContext(t, http.MethodPut,
		"/v8/management/observability/usage/key-labels",
		[]byte(`{"api_key_id":"9f2c4a0e11d3b7aa","label":""}`))
	h.PutUsageKeyLabel(ginCtx)
	if cleared := decodePriceBody(t, rec); cleared["label"] != "" {
		t.Fatalf("cleared = %v", cleared)
	}
	for _, body := range []string{
		`{"label":"x"}`,
		`{bad`,
	} {
		rec, ginCtx := usagePriceContext(t, http.MethodPut,
			"/v8/management/observability/usage/key-labels", []byte(body))
		h.PutUsageKeyLabel(ginCtx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("PUT %s status = %d, want 400", body, rec.Code)
		}
	}
	_ = base
}

func TestUsageMaintenanceRoute(t *testing.T) {
	withUsageDB(t, 90)
	seedUsageQueryRows(t)
	h := &Handler{}
	for _, action := range []string{"prune", "vacuum", "checkpoint"} {
		rec, ginCtx := usagePriceContext(t, http.MethodPost,
			"/v8/management/observability/usage/maintenance",
			[]byte(`{"action":"`+action+`"}`))
		h.PostUsageMaintenance(ginCtx)
		if rec.Code != http.StatusOK {
			t.Fatalf("POST %s status = %d body=%s", action, rec.Code, rec.Body.String())
		}
		payload := decodePriceBody(t, rec)
		if payload["action"] != action || payload["duration_ms"] == nil {
			t.Fatalf("%s payload = %v", action, payload)
		}
	}
	rec, ginCtx := usagePriceContext(t, http.MethodPost,
		"/v8/management/observability/usage/maintenance", []byte(`{"action":"bogus"}`))
	h.PostUsageMaintenance(ginCtx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bogus action status = %d, want 400", rec.Code)
	}
	rec, ginCtx = usagePriceContext(t, http.MethodPost,
		"/v8/management/observability/usage/maintenance", []byte(`{bad`))
	h.PostUsageMaintenance(ginCtx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad body status = %d, want 400", rec.Code)
	}
	var events float64
	store := usagedb.ActiveStore()
	_ = store
	rec = getUsage(t, h, withRange("/v8/management/observability/usage/summary", 1719997200000))
	payload := decodePriceBody(t, rec)
	if totals, ok := payload["totals"].(map[string]any); ok {
		events = totals["requests"].(float64)
	}
	if events != 4 {
		t.Fatalf("rows after vacuum = %v, want 4", events)
	}
}

func TestUsageRoutesAuth(t *testing.T) {
	withUsageDB(t, 90)
	hash, err := bcrypt.GenerateFromPassword([]byte("test-key"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	cfg := &config.Config{}
	cfg.RemoteManagement.AllowRemote = true
	cfg.RemoteManagement.SecretKey = string(hash)
	h := NewHandler(cfg, "", nil)
	engine := gin.New()
	group := engine.Group("/v8/management", h.Middleware())
	group.GET("/observability/usage/summary", h.GetUsageSummary)
	group.GET("/observability/usage/timeseries", h.GetUsageTimeseries)
	group.GET("/observability/usage/breakdown", h.GetUsageBreakdown)
	group.GET("/observability/usage/events", h.GetUsageEvents)
	group.GET("/observability/usage/events/:id", h.GetUsageEvent)
	group.GET("/observability/usage/failures", h.GetUsageFailures)
	group.GET("/observability/usage/heatmap", h.GetUsageHeatmap)
	group.GET("/observability/usage/export", h.GetUsageExport)
	group.GET("/observability/usage/key-labels", h.GetUsageKeyLabels)
	group.PUT("/observability/usage/key-labels", h.PutUsageKeyLabel)
	group.POST("/observability/usage/maintenance", h.PostUsageMaintenance)
	targets := [][2]string{
		{"GET", "/v8/management/observability/usage/summary"},
		{"GET", "/v8/management/observability/usage/timeseries"},
		{"GET", "/v8/management/observability/usage/breakdown?group_by=model"},
		{"GET", "/v8/management/observability/usage/events"},
		{"GET", "/v8/management/observability/usage/events/1"},
		{"GET", "/v8/management/observability/usage/failures"},
		{"GET", "/v8/management/observability/usage/heatmap"},
		{"GET", "/v8/management/observability/usage/export"},
		{"GET", "/v8/management/observability/usage/key-labels"},
		{"PUT", "/v8/management/observability/usage/key-labels"},
		{"POST", "/v8/management/observability/usage/maintenance"},
	}
	for i, target := range targets {
		// Fresh client IPs dodge the management key ban counter.
		request := httptest.NewRequest(target[0], target[1], bytes.NewReader(nil))
		request.RemoteAddr = "192.0.2." + strconv.Itoa(10+i) + ":1234"
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, request)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d, want 401", target[0], target[1], rec.Code)
		}
	}
	request := httptest.NewRequest("GET", "/v8/management/observability/usage/summary", nil)
	request.Header.Set("X-Management-Key", "test-key")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, request)
	if rec.Code != http.StatusOK {
		t.Fatalf("authed summary status = %d body=%s", rec.Code, rec.Body.String())
	}
}
