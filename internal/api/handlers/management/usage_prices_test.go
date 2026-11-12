package management

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Su3h7aM/LLMProxyAPI/internal/usagedb"
	"github.com/gin-gonic/gin"
)

func usagePriceContext(t *testing.T, method, target string, body []byte) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	request := httptest.NewRequest(method, target, bytes.NewReader(body))
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	ginCtx.Request = request
	return rec, ginCtx
}

func decodePriceBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal %s: %v", rec.Body.String(), err)
	}
	return payload
}

func putTestPrice(t *testing.T, h *Handler, body string) map[string]any {
	t.Helper()
	rec, ginCtx := usagePriceContext(t, http.MethodPut, "/v8/management/observability/usage/prices", []byte(body))
	h.PutUsagePrice(ginCtx)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}
	return decodePriceBody(t, rec)
}

func TestPutAndListAndHistoryPrices(t *testing.T) {
	withUsageDB(t, 90)
	h := &Handler{}
	created := putTestPrice(t, h, `{"provider":"openai","model":"gpt-5t","input":1.25,"output":10}`)
	if created["source"] != usagedb.PriceSourceManual {
		t.Fatalf("source = %v, want manual", created["source"])
	}
	if created["input"] != float64(1250000) {
		t.Fatalf("input = %v, want 1250000", created["input"])
	}

	rec, ginCtx := usagePriceContext(t, http.MethodGet,
		"/v8/management/observability/usage/prices?provider=openai&model=gpt-5t", nil)
	h.GetUsagePrices(ginCtx)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d body=%s", rec.Code, rec.Body.String())
	}
	listed := decodePriceBody(t, rec)
	if listed["total"] != float64(1) {
		t.Fatalf("total = %v, want 1 body=%s", listed["total"], rec.Body.String())
	}
	if prices, ok := listed["prices"].([]any); !ok || len(prices) != 1 {
		t.Fatalf("prices = %v, want 1 row", listed["prices"])
	}

	rec, ginCtx = usagePriceContext(t, http.MethodGet,
		"/v8/management/observability/usage/prices/history?provider=openai&model=gpt-5t", nil)
	h.GetUsagePriceHistory(ginCtx)
	if rec.Code != http.StatusOK {
		t.Fatalf("history status = %d body=%s", rec.Code, rec.Body.String())
	}
	history := decodePriceBody(t, rec)
	if versions, ok := history["versions"].([]any); !ok || len(versions) != 1 {
		t.Fatalf("versions = %v, want 1", history["versions"])
	}
}

func TestPutPriceValidation(t *testing.T) {
	withUsageDB(t, 90)
	h := &Handler{}
	for _, body := range []string{
		`{"provider":"","model":"m","input":1,"output":1}`,
		`{"provider":"openai","model":"m","output":1}`,
		`{"provider":"openai","model":"m","input":-1,"output":1}`,
		`{bad`,
	} {
		rec, ginCtx := usagePriceContext(t, http.MethodPut, "/v8/management/observability/usage/prices", []byte(body))
		h.PutUsagePrice(ginCtx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("PUT %s status = %d, want 400", body, rec.Code)
		}
	}
}

func TestDeleteManualRestoresSynced(t *testing.T) {
	withUsageDB(t, 90)
	h := &Handler{}
	store := usagedb.ActiveStore()
	if _, err := store.AddPriceVersion(usagedb.Price{
		ProviderKey: "openai", ModelID: "gpt-5r", ValidFromMs: 0,
		Input: 1000000, Output: 2000000, Source: usagedb.PriceSourceModelsDev,
	}); err != nil {
		t.Fatalf("AddPriceVersion: %v", err)
	}
	putTestPrice(t, h, `{"provider":"openai","model":"gpt-5r","input":9,"output":9,"effective_from_ms":1000}`)

	rec, ginCtx := usagePriceContext(t, http.MethodDelete,
		"/v8/management/observability/usage/prices/manual?provider=openai&model=gpt-5r", nil)
	h.DeleteUsagePriceManual(ginCtx)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d body=%s", rec.Code, rec.Body.String())
	}
	if payload := decodePriceBody(t, rec); payload["restored"] != true {
		t.Fatalf("restored = %v, want true body=%s", payload["restored"], rec.Body.String())
	}
	history, err := store.PriceHistory("openai", "gpt-5r")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 3 || history[2].Source != usagedb.PriceSourceModelsDev || history[2].Input != 1000000 {
		t.Fatalf("history = %+v, want restored synced rates", history)
	}

	rec, ginCtx = usagePriceContext(t, http.MethodDelete,
		"/v8/management/observability/usage/prices/manual?provider=openai&model=gpt-5r", nil)
	h.DeleteUsagePriceManual(ginCtx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("second DELETE status = %d, want 400 (no manual current)", rec.Code)
	}
}

func TestPriceUnpricedAndStatusRoutes(t *testing.T) {
	withUsageDB(t, 90)
	h := &Handler{}
	store := usagedb.ActiveStore()
	row := usagedb.UsageRow{
		RequestID: "route-unpriced", TsMs: 1000, HourMs: 0,
		Provider: "codex", Model: "gpt-5u", Alias: "gpt-5u",
		Executor: "codex",
	}
	if _, err := store.InsertBatch([]usagedb.UsageRow{row}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	if _, err := store.AddPriceVersion(usagedb.Price{
		ProviderKey: "openai", ModelID: "gpt-5u", ValidFromMs: 0,
		Input: 1000000, Output: 2000000, Source: usagedb.PriceSourceModelsDev,
	}); err != nil {
		t.Fatalf("AddPriceVersion: %v", err)
	}

	rec, ginCtx := usagePriceContext(t, http.MethodGet,
		"/v8/management/observability/usage/prices/status", nil)
	h.GetUsagePricesStatus(ginCtx)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	status := decodePriceBody(t, rec)
	if unmatched, ok := status["unmatched"].([]any); !ok || len(unmatched) != 1 {
		t.Fatalf("unmatched = %v, want 1", status["unmatched"])
	}

	rec, ginCtx = usagePriceContext(t, http.MethodPost,
		"/v8/management/observability/usage/prices/price-unpriced", []byte(`{}`))
	h.PostUsagePricesPriceUnpriced(ginCtx)
	if rec.Code != http.StatusOK {
		t.Fatalf("price-unpriced = %d body=%s", rec.Code, rec.Body.String())
	}
	if payload := decodePriceBody(t, rec); payload["priced"] != float64(1) {
		t.Fatalf("priced = %v, want 1 body=%s", payload["priced"], rec.Body.String())
	}
}

func TestCorrectRouteValidationAndSuccess(t *testing.T) {
	withUsageDB(t, 90)
	h := &Handler{}
	store := usagedb.ActiveStore()
	row := usagedb.UsageRow{
		RequestID: "route-correct", TsMs: 1000, HourMs: 0,
		Provider: "codex", Model: "gpt-5c", Alias: "gpt-5c", Executor: "codex",
		InputTokens: 1000000, OutputTokens: 1000000, TotalTokens: 2000000,
	}
	if _, err := store.InsertBatch([]usagedb.UsageRow{row}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	if _, err := store.AddPriceVersion(usagedb.Price{
		ProviderKey: "openai", ModelID: "gpt-5c", ValidFromMs: 0,
		Input: 1000000, Output: 1000000, Source: usagedb.PriceSourceModelsDev,
	}); err != nil {
		t.Fatalf("AddPriceVersion: %v", err)
	}

	for body, want := range map[string]int{
		`{"from_ms":0,"to_ms":2000,"reason":"x"}`:                         http.StatusBadRequest,
		`{"from_ms":0,"to_ms":2000,"confirm":true}`:                       http.StatusBadRequest,
		`{"from_ms":0,"to_ms":2000,"confirm":true,"reason":"x"}`:          http.StatusOK,
		`{"from_ms":0,"to_ms":2000000000000,"confirm":true,"reason":"x"}`: http.StatusBadRequest,
	} {
		rec, ginCtx := usagePriceContext(t, http.MethodPost,
			"/v8/management/observability/usage/prices/correct", []byte(body))
		h.PostUsagePricesCorrect(ginCtx)
		if rec.Code != want {
			t.Fatalf("POST %s status = %d, want %d body=%s", body, rec.Code, want, rec.Body.String())
		}
		if want == http.StatusOK {
			payload := decodePriceBody(t, rec)
			if payload["events_updated"] != float64(1) || payload["correction_id"] == float64(0) {
				t.Fatalf("correct = %v, want 1 event and audit id", payload)
			}
		}
	}
}

func TestSyncRouteRunsNow(t *testing.T) {
	withUsageDB(t, 90)
	h := &Handler{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"openai":{"models":{"gpt-5s":{"cost":{"input":1,"output":2}}}}}`))
	}))
	defer server.Close()
	usagedb.Configure(usagedb.Options{
		Enabled:          true,
		Path:             usagedb.ActiveStore().Path() + "-sync",
		RetentionDays:    90,
		BufferSize:       64,
		PriceSyncEnabled: true,
		PriceSyncURL:     server.URL,
		PriceSyncHours:   24,
	})
	t.Cleanup(func() { usagedb.Shutdown(5 * time.Second) })

	rec, ginCtx := usagePriceContext(t, http.MethodPost,
		"/v8/management/observability/usage/prices/sync", nil)
	h.PostUsagePricesSync(ginCtx)
	if rec.Code != http.StatusOK {
		t.Fatalf("sync = %d body=%s", rec.Code, rec.Body.String())
	}
	if payload := decodePriceBody(t, rec); payload["added"] != float64(1) || payload["error"] != "" {
		t.Fatalf("sync = %v, want added 1, no error", payload)
	}
}
