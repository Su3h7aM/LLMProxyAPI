package management

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Su3h7aM/LLMProxyAPI/internal/config"
	"github.com/gin-gonic/gin"
)

func TestNormalizeRoutingStrategySoonestReset(t *testing.T) {
	for _, input := range []string{"soonest-reset", "soonestreset", " Soonest-Reset "} {
		got, ok := normalizeRoutingStrategy(input)
		if !ok || got != "soonest-reset" {
			t.Fatalf("normalizeRoutingStrategy(%q) = %q, %v; want soonest-reset, true", input, got, ok)
		}
	}
}

func TestPutRoutingStrategySoonestReset(t *testing.T) {
	cfg := &config.Config{}
	h := &Handler{cfg: cfg, configFilePath: writeTestConfigFile(t)}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/v0/management/routing/strategy",
		strings.NewReader(`{"value":"soonestreset"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	h.PutRoutingStrategy(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if got := cfg.Routing.Strategy; got != "soonest-reset" {
		t.Fatalf("Routing.Strategy = %q, want soonest-reset", got)
	}

	rec = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/routing/strategy", nil)
	h.GetRoutingStrategy(ctx)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"strategy":"soonest-reset"`) {
		t.Fatalf("GET status = %d body=%s, want soonest-reset", rec.Code, rec.Body.String())
	}
}
