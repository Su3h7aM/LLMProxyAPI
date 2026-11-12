//go:build usageperf

package usagedb

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// perfProfile selects the seed shape: uniform (worst case: traffic spread
// over every combination, about 1M hourly rows), realistic (traffic
// concentrated on a few combinations with hour-of-day bursts), or both.
var perfProfile = flag.String("profile", "both", "usageperf seed profile: uniform, realistic, or both")

// TestPerfUsageQueries fills temp databases with 1M synthetic events each
// through the real writer path and times each query route against the
// contract targets. It never runs in the normal gate; run it by hand:
//
//	/usr/bin/go test -tags usageperf ./internal/usagedb/ -run TestPerfUsageQueries -v -timeout 30m
//	/usr/bin/go test -tags usageperf ./internal/usagedb/ -run TestPerfUsageQueries -v -timeout 30m -args -profile=realistic
//
// The databases live in t.TempDir and are not committed.
func TestPerfUsageQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("perf only")
	}
	profiles := []string{"uniform", "realistic"}
	switch *perfProfile {
	case "uniform":
		profiles = []string{"uniform"}
	case "realistic":
		profiles = []string{"realistic"}
	case "both":
	default:
		t.Fatalf("unknown -profile %q", *perfProfile)
	}
	for _, profile := range profiles {
		t.Run(profile, func(t *testing.T) {
			runPerfProfile(t, profile)
		})
	}
}

func runPerfProfile(t *testing.T, profile string) {
	t.Helper()
	store := openTestStore(t)
	now := time.Now().UnixMilli()
	dayMs := int64(24 * 60 * 60 * 1000)
	spanMs := int64(120) * dayMs
	const total = 1000000
	const batch = 5000
	upstream := int64(100)
	seedStart := time.Now()
	for start := 0; start < total; start += batch {
		end := min(start+batch, total)
		rows := makeSeedBatch(profile, start, end, now, spanMs, upstream)
		if _, err := store.InsertBatch(rows); err != nil {
			t.Fatalf("seed batch: %v", err)
		}
	}
	t.Logf("seed %s 1M rows in %s", profile, time.Since(seedStart).Round(time.Millisecond))
	var hourlyRows, eventRows int64
	if err := store.readDB.QueryRow(`SELECT COUNT(*) FROM usage_hourly`).Scan(&hourlyRows); err != nil {
		t.Fatalf("count hourly: %v", err)
	}
	if err := store.readDB.QueryRow(`SELECT COUNT(*) FROM usage_events`).Scan(&eventRows); err != nil {
		t.Fatalf("count events: %v", err)
	}
	t.Logf("rows: events=%d hourly=%d", eventRows, hourlyRows)
	filter := UsageFilter{FromMs: now - spanMs, ToMs: now, TZ: "UTC", Loc: time.UTC}
	// Cold: fresh connections (SQLite caches empty) after close and reopen.
	// The OS page cache stays warm from seeding, so this is SQLite-cold.
	if err := store.Close(); err != nil {
		t.Fatalf("close for cold: %v", err)
	}
	path := store.Path()
	cold, err := Open(path)
	if err != nil {
		t.Fatalf("reopen for cold: %v", err)
	}
	defer func() { _ = cold.Close() }()
	measure := func(name string, run func(*Store) error) {
		t.Helper()
		first := time.Now()
		if err := run(cold); err != nil {
			t.Fatalf("%s cold: %v", name, err)
		}
		coldDur := time.Since(first)
		second := time.Now()
		if err := run(cold); err != nil {
			t.Fatalf("%s warm: %v", name, err)
		}
		warmDur := time.Since(second)
		t.Logf("%s: cold %s warm %s", name, coldDur.Round(time.Millisecond), warmDur.Round(time.Millisecond))
	}
	measure("summary 120d (hourly)", func(s *Store) error {
		_, err := s.QuerySummary(filter)
		return err
	})
	week := filter
	week.FromMs = now - 7*dayMs
	measure("summary 7d with exact percentiles", func(s *Store) error {
		_, err := s.QuerySummary(week)
		return err
	})
	measure("timeseries day grouped", func(s *Store) error {
		_, err := s.QueryTimeseries(filter, "day", "model", 5)
		return err
	})
	measure("breakdown model", func(s *Store) error {
		_, _, err := s.QueryBreakdown(filter, "model", "requests", "desc", 20, 0)
		return err
	})
	measure("events page", func(s *Store) error {
		_, err := s.QueryEvents(filter, 50, "", "desc")
		return err
	})
	month := filter
	month.FromMs = now - 30*dayMs
	measure("failures 30d", func(s *Store) error {
		_, err := s.QueryFailures(month, 20, 20)
		return err
	})
	measure("heatmap 30d", func(s *Store) error {
		_, err := s.QueryHeatmap(month, "requests")
		return err
	})
	measure("export full scan", func(s *Store) error {
		var count int64
		return s.ExportEvents(context.Background(), filter, func(UsageEvent) error {
			count++
			return nil
		})
	})
	// Component split for the summary route, to find the hot spot.
	measure("part: events COUNT 120d", func(s *Store) error {
		var n int64
		return s.readDB.QueryRow(`SELECT COUNT(*) FROM usage_events WHERE ts_ms >= ? AND ts_ms < ?`,
			filter.FromMs, filter.ToMs).Scan(&n)
	})
	measure("part: hourly SUM requests 120d", func(s *Store) error {
		var n int64
		return s.readDB.QueryRow(`SELECT COALESCE(SUM(requests),0) FROM usage_hourly WHERE hour_ms >= ? AND hour_ms < ?`,
			filter.FromMs/millisPerHour*millisPerHour, filter.ToMs).Scan(&n)
	})
}

// makeSeedBatch builds one writer-path batch for the profile. Row n is
// deterministic in n, so reruns compare the same data.
func makeSeedBatch(profile string, start, end int, now, spanMs int64, upstream int64) []UsageRow {
	if profile == "realistic" {
		return makeRealisticBatch(start, end, now, spanMs, upstream)
	}
	return makeUniformBatch(start, end, now, spanMs, upstream)
}

func makeUniformBatch(start, end int, now, spanMs int64, upstream int64) []UsageRow {
	providers := []string{"codex", "claude", "gemini"}
	models := []string{"model-a", "model-b", "model-c", "model-d", "model-e"}
	rows := make([]UsageRow, 0, end-start)
	for n := start; n < end; n++ {
		failed := n%50 == 0
		status := "2xx"
		if failed {
			status = "5xx"
		}
		ts := now - spanMs + int64(n)*spanMs/int64(1000000)
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
	return rows
}

// realisticCred ties one credential to its models and key, as production
// traffic does: each credential serves one or two models, so traffic
// concentrates on a few combinations.
type realisticCred struct {
	auth   string
	models []string
	key    string
}

// modelProvider ties each model to its provider.
var modelProvider = map[string]string{
	"gpt-5": "codex", "gpt-6": "codex", "gpt-5-mini": "codex", "o4-mini": "codex", "gpt-4o": "openai",
	"claude-sonnet-5": "claude", "claude-opus-4": "claude", "claude-haiku-4": "claude", "claude-sonnet-4": "claude",
	"gemini-3-pro": "gemini", "gemini-3-flash": "gemini", "gemini-3-lite": "gemini",
}

var (
	realisticCreds = []realisticCred{
		{"alice.json", []string{"gpt-5", "gpt-6"}, "key-0001"},
		{"bob.json", []string{"claude-sonnet-4", "claude-sonnet-5"}, "key-0001"},
		{"carol.json", []string{"gemini-3-pro"}, "key-0002"},
		{"dave.json", []string{"gpt-5-mini"}, "key-0002"},
		{"erin.json", []string{"claude-haiku-4", "gemini-3-flash"}, "key-0003"},
		{"frank.json", []string{"gemini-3-lite"}, "key-0003"},
		{"grace.json", []string{"o4-mini", "gpt-4o"}, "key-0004"},
		{"heidi.json", []string{"claude-opus-4", "gemini-3-pro"}, "key-0004"},
	}
	realisticCredW = []float64{0.35, 0.25, 0.12, 0.08, 0.07, 0.05, 0.05, 0.03}
	// realisticHourW bursts by hour of day (UTC): work hours heavy, night light.
	realisticHourW = [24]float64{
		0.15, 0.1, 0.08, 0.08, 0.1, 0.15, 0.3, 0.5,
		0.7, 0.9, 1.0, 1.0, 0.9, 1.0, 1.0, 0.9,
		0.8, 0.6, 0.45, 0.35, 0.3, 0.25, 0.2, 0.18,
	}
)

func pickWeighted(rng *rand.Rand, weights []float64) int {
	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	draw := rng.Float64() * sum
	for i, w := range weights {
		draw -= w
		if draw <= 0 {
			return i
		}
	}
	return len(weights) - 1
}

// makeRealisticBatch builds a concentrated-traffic batch: 8 credentials,
// each with 1-2 models and one key (12 models, 4 keys total), top
// credentials dominant, hour-of-day bursts, 2% failures. A batch-local RNG
// seeded by the batch start keeps rows deterministic in n.
func makeRealisticBatch(start, end int, now, spanMs int64, upstream int64) []UsageRow {
	rng := rand.New(rand.NewSource(int64(start)))
	hourW := realisticHourW[:]
	rows := make([]UsageRow, 0, end-start)
	days := int(spanMs / (24 * 60 * 60 * 1000))
	for n := start; n < end; n++ {
		cred := realisticCreds[pickWeighted(rng, realisticCredW)]
		model := cred.models[rng.Intn(len(cred.models))]
		provider := modelProvider[model]
		day := rng.Intn(days)
		hour := pickWeighted(rng, hourW)
		ts := now - spanMs + int64(day)*24*60*60*1000 + int64(hour)*60*60*1000 + rng.Int63n(60*60*1000)
		failed := rng.Float64() < 0.02
		status := "2xx"
		failCode := ""
		if failed {
			if rng.Float64() < 0.7 {
				status = "5xx"
				failCode = "upstream_error"
			} else {
				status = "4xx"
				failCode = "rate_limit"
			}
		}
		rows = append(rows, UsageRow{
			RequestID:   fmt.Sprintf("real-%d", n),
			TraceID:     fmt.Sprintf("trace-%d", n%1000),
			TsMs:        ts,
			HourMs:      ts / millisPerHour * millisPerHour,
			Provider:    provider,
			Model:       model,
			Alias:       model,
			Executor:    provider,
			AuthID:      cred.auth,
			APIKeyID:    cred.key,
			Stream:      rng.Float64() < 0.5,
			StatusClass: status,
			Failed:      failed,
			FailCode:    failCode,
			LatencyMs:   80 + rng.Int63n(2500),
			TTFTMs:      rng.Int63n(600),
			InputTokens: 100 + rng.Int63n(8000), OutputTokens: 50 + rng.Int63n(3000),
			TotalTokens: 200 + rng.Int63n(10000),
			ServiceTier: "default",
			// Zero cost would price everything at 0; upstream keeps
			// cost math in the loop like production.
			UpstreamCostMicroUSD: &upstream,
		})
	}
	return rows
}
