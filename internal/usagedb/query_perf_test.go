//go:build usageperf

package usagedb

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestPerfUsageQueries fills a temp database with 1M synthetic events
// through the real writer path and times each query route against the
// contract targets. It never runs in the normal gate; run it by hand:
//
//	/usr/bin/go test -tags usageperf ./internal/usagedb/ -run TestPerfUsageQueries -v -timeout 30m
//
// The database lives in t.TempDir and is not committed.
func TestPerfUsageQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("perf only")
	}
	store := openTestStore(t)
	providers := []string{"codex", "claude", "gemini"}
	models := []string{"model-a", "model-b", "model-c", "model-d", "model-e"}
	now := time.Now().UnixMilli()
	dayMs := int64(24 * 60 * 60 * 1000)
	spanMs := int64(120) * dayMs
	const total = 1000000
	const batch = 5000
	upstream := int64(100)
	seedStart := time.Now()
	for start := 0; start < total; start += batch {
		rows := make([]UsageRow, 0, batch)
		for i := 0; i < batch; i++ {
			n := start + i
			failed := n%50 == 0
			status := "2xx"
			if failed {
				status = "5xx"
			}
			ts := now - spanMs + int64(n)*spanMs/int64(total)
			rows = append(rows, UsageRow{
				RequestID:   fmt.Sprintf("perf-%d", n),
				TraceID:     fmt.Sprintf("trace-%d", n%1000),
				TsMs:        ts,
				HourMs:      ts / millisPerHour * millisPerHour,
				Provider:    providers[n%len(providers)],
				Model:       models[n%len(models)],
				Alias:       models[n%len(models)],
				Executor:    providers[n%len(providers)],
				AuthID:      fmt.Sprintf("auth-%d.json", n%8),
				APIKeyID:    fmt.Sprintf("key-%04d", n%8),
				Stream:      n%2 == 0,
				StatusClass: status,
				Failed:      failed,
				FailCode:    map[bool]string{true: "rate_limit"}[failed],
				LatencyMs:   int64(n % 3000),
				TTFTMs:      int64(n % 500),
				InputTokens: int64(n % 4000), OutputTokens: int64(n % 2000),
				TotalTokens: int64(n%6000 + 10),
				ServiceTier: "default",
				// Zero cost would price everything at 0; upstream keeps
				// cost math in the loop like production.
				UpstreamCostMicroUSD: &upstream,
			})
		}
		if _, err := store.InsertBatch(rows); err != nil {
			t.Fatalf("seed batch: %v", err)
		}
	}
	t.Logf("seed 1M rows in %s", time.Since(seedStart).Round(time.Millisecond))
	filter := UsageFilter{FromMs: now - spanMs, ToMs: now, TZ: "UTC", Loc: time.UTC}
	measure := func(name string, run func() error) {
		t.Helper()
		start := time.Now()
		if err := run(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		t.Logf("%s: %s", name, time.Since(start).Round(time.Millisecond))
	}
	measure("summary 120d (hourly)", func() error {
		_, err := store.QuerySummary(filter)
		return err
	})
	week := filter
	week.FromMs = now - 7*dayMs
	measure("summary 7d with exact percentiles", func() error {
		_, err := store.QuerySummary(week)
		return err
	})
	measure("timeseries day grouped", func() error {
		_, err := store.QueryTimeseries(filter, "day", "model", 5)
		return err
	})
	measure("breakdown model", func() error {
		_, _, err := store.QueryBreakdown(filter, "model", "requests", "desc", 20, 0)
		return err
	})
	measure("events page", func() error {
		_, err := store.QueryEvents(filter, 50, "", "desc")
		return err
	})
	measure("failures 30d", func() error {
		month := filter
		month.FromMs = now - 30*dayMs
		_, err := store.QueryFailures(month, 20, 20)
		return err
	})
	measure("heatmap 30d", func() error {
		month := filter
		month.FromMs = now - 30*dayMs
		_, err := store.QueryHeatmap(month, "requests")
		return err
	})
	measure("export full scan", func() error {
		var count int64
		return store.ExportEvents(context.Background(), filter, func(UsageEvent) error {
			count++
			return nil
		})
	})
}
