package management

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/Su3h7aM/LLMProxyAPI/internal/usagedb"
	coreusage "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/usage"
	"github.com/gin-gonic/gin"
)

func withUsageDB(t *testing.T, retentionDays int) {
	t.Helper()
	usagedb.Shutdown(5 * time.Second)
	usagedb.Configure(usagedb.Options{
		Enabled:       true,
		Path:          filepath.Join(t.TempDir(), "usage.db"),
		RetentionDays: retentionDays,
		BufferSize:    64,
	})
	t.Cleanup(func() { usagedb.Shutdown(5 * time.Second) })
}

func waitForUsageEvents(t *testing.T, want int64) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if usagedb.Status().Events >= want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("usage events did not reach %d", want)
}

func TestGetUsageDBStatus(t *testing.T) {
	withUsageDB(t, 90)
	record := coreusage.Record{
		RequestID:   "status-1",
		Provider:    "codex",
		Model:       "gpt-5",
		RequestedAt: time.UnixMilli(1720000000000),
		Latency:     50 * time.Millisecond,
	}
	usagedb.Collector{}.HandleUsage(context.Background(), record)
	waitForUsageEvents(t, 1)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/v8/management/observability/usage/status", nil)

	h := &Handler{}
	h.GetUsageDBStatus(ginCtx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var payload struct {
		Enabled       bool   `json:"enabled"`
		Path          string `json:"path"`
		SizeBytes     int64  `json:"size_bytes"`
		SchemaVersion int    `json:"schema_version"`
		Events        int64  `json:"events"`
		OldestTsMs    int64  `json:"oldest_ts_ms"`
		NewestTsMs    int64  `json:"newest_ts_ms"`
		BufferCap     int    `json:"buffer_capacity"`
		RetentionDays int64  `json:"retention_days"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal status: %v", err)
	}
	if !payload.Enabled {
		t.Fatal("enabled = false, want true")
	}
	if payload.Path == "" {
		t.Fatal("path is empty")
	}
	if payload.SchemaVersion != usagedb.SchemaVersion {
		t.Fatalf("schema = %d, want %d", payload.SchemaVersion, usagedb.SchemaVersion)
	}
	if payload.Events != 1 {
		t.Fatalf("events = %d, want 1", payload.Events)
	}
	if payload.OldestTsMs != 1720000000000 || payload.NewestTsMs != 1720000000000 {
		t.Fatalf("oldest/newest = %d/%d, want 1720000000000", payload.OldestTsMs, payload.NewestTsMs)
	}
	if payload.BufferCap != 64 {
		t.Fatalf("buffer cap = %d, want 64", payload.BufferCap)
	}
	if payload.RetentionDays != 90 {
		t.Fatalf("retention = %d, want 90", payload.RetentionDays)
	}
	if payload.SizeBytes <= 0 {
		t.Fatalf("size = %d, want positive", payload.SizeBytes)
	}
}
