package usagedb

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

const syncFixture = `{
	"anthropic": {
		"id": "anthropic",
		"name": "Anthropic",
		"models": {
			"claude-test-1": {
				"id": "claude-test-1",
				"name": "Claude Test",
				"limit": {"context": 200000, "output": 64000},
				"cost": {
					"input": 1, "output": 5,
					"cache_read": 0.1, "cache_write": 1.25,
					"tiers": [
						{"input": 2, "output": 10, "tier": {"type": "context", "size": 200000}}
					],
					"context_over_200k": {"input": 2, "output": 10}
				}
			},
			"claude-nocost": {
				"id": "claude-nocost",
				"name": "No Cost",
				"cost": {}
			},
			"claude-future-tier": {
				"id": "claude-future-tier",
				"cost": {
					"input": 1, "output": 1,
					"tiers": [{"input": 2, "tier": {"type": "hyperdrive", "size": 10}}]
				}
			}
		}
	},
	"openai": {
		"id": "openai",
		"models": {
			"gpt-test": {"id": "gpt-test", "cost": {"input": 2, "output": 8}}
		}
	}
}`

func TestParseModelsDevFixture(t *testing.T) {
	models, err := ParseModelsDev([]byte(syncFixture))
	if err != nil {
		t.Fatalf("ParseModelsDev: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("models = %d, want 3 (nocost skipped)", len(models))
	}
	var claude *syncedModel
	for i := range models {
		if models[i].Model == "claude-test-1" {
			claude = &models[i]
		}
	}
	if claude == nil {
		t.Fatal("claude-test-1 missing")
	}
	price := claude.Price
	if price.Input != 1000000 || price.Output != 5000000 ||
		price.CacheRead != 100000 || price.CacheWrite != 1250000 {
		t.Fatalf("rates = %+v, want 1M/5M/100K/1250K", price)
	}
	if len(price.Tiers) != 2 {
		t.Fatalf("tiers = %d, want 2 (tiers + context_over_200k)", len(price.Tiers))
	}
	// Unknown future tier types are skipped, not mispriced.
	for i := range models {
		if models[i].Model == "claude-future-tier" && len(models[i].Price.Tiers) != 0 {
			t.Fatalf("future tiers kept: %+v", models[i].Price.Tiers)
		}
	}
	if _, err := ParseModelsDev([]byte("{bad")); err == nil {
		t.Fatal("malformed JSON accepted, want error")
	}
}

func syncTestServer(t *testing.T, hits *atomic.Int64, etag string, payload string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if agent := r.Header.Get("User-Agent"); !strings.HasPrefix(agent, "LLMProxyAPI") {
			http.Error(w, "bad agent", http.StatusBadRequest)
			return
		}
		if etag != "" && r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if etag != "" {
			w.Header().Set("ETag", etag)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, payload)
	}))
}

func syncTestConfig(url string) PriceSyncConfig {
	return PriceSyncConfig{Enabled: true, URL: url, IntervalHours: 24}
}

func TestSyncFirstUnchangedChanged(t *testing.T) {
	store := openTestStore(t)
	var hits atomic.Int64
	server := syncTestServer(t, &hits, "", syncFixture, http.StatusOK)
	defer server.Close()
	cfg := syncTestConfig(server.URL)
	now := int64(5000)

	first := store.SyncPricesNow(t.Context(), cfg, now)
	if first.Error != "" {
		t.Fatalf("sync error: %s", first.Error)
	}
	if first.Added == 0 || first.Updated != 0 || first.Unchanged != 0 || first.SkippedOverrides != 0 {
		t.Fatalf("first sync = %+v, want only Added", first)
	}
	price := store.lookupPrice("claude", "claude-test-1", "", now)
	if price == nil || price.ValidFromMs != 0 || price.Source != PriceSourceModelsDev {
		t.Fatalf("first version = %+v, want valid_from 0, models.dev", price)
	}

	second := store.SyncPricesNow(t.Context(), cfg, now+1000)
	if second.Added != 0 || second.Updated != 0 || second.Error != "" {
		t.Fatalf("second sync = %+v, want only Unchanged", second)
	}
	if second.Unchanged != first.Added {
		t.Fatalf("unchanged = %d, want %d", second.Unchanged, first.Added)
	}
}

func TestSyncChangedClosesAndReopens(t *testing.T) {
	store := openTestStore(t)
	var payload atomic.Value
	payload.Store(syncFixture)
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = fmt.Fprint(w, payload.Load().(string))
	}))
	defer server.Close()
	cfg := syncTestConfig(server.URL)

	first := store.SyncPricesNow(t.Context(), cfg, 1000)
	if first.Added == 0 {
		t.Fatalf("first sync = %+v, want Added", first)
	}
	changed := strings.Replace(syncFixture, `"input": 1, "output": 5,`, `"input": 3, "output": 5,`, 1)
	payload.Store(changed)
	second := store.SyncPricesNow(t.Context(), cfg, 2000)
	if second.Error != "" {
		t.Fatalf("sync error: %s", second.Error)
	}
	if second.Updated == 0 {
		t.Fatalf("second sync = %+v, want Updated", second)
	}
	history, err := store.PriceHistory("anthropic", "claude-test-1")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("versions = %d, want 2", len(history))
	}
	if history[0].ValidToMs == nil || *history[0].ValidToMs != 2000 || history[0].Input != 1000000 {
		t.Fatalf("old version = %+v, want input 1000000 closed at 2000", history[0])
	}
	if history[1].ValidFromMs != 2000 || history[1].Input != 3000000 {
		t.Fatalf("new version = %+v, want input 3000000 from 2000", history[1])
	}
	// History still prices each event at its own time.
	price := store.lookupPrice("claude", "claude-test-1", "", 1500)
	if price == nil || price.Input != 1000000 {
		t.Fatalf("lookup at 1500 = %+v, want old rate", price)
	}
}

func TestSyncSkipsManualOverride(t *testing.T) {
	store := openTestStore(t)
	server := syncTestServer(t, &atomic.Int64{}, "", syncFixture, http.StatusOK)
	defer server.Close()
	cfg := syncTestConfig(server.URL)
	if _, err := store.AddManualPrice(Price{
		ProviderKey: "anthropic", ModelID: "claude-test-1",
		ValidFromMs: 500, Input: 7000000, Output: 7000000,
	}, 500); err != nil {
		t.Fatalf("AddManualPrice: %v", err)
	}
	result := store.SyncPricesNow(t.Context(), cfg, 2000)
	if result.Error != "" {
		t.Fatalf("sync error: %s", result.Error)
	}
	if result.SkippedOverrides == 0 {
		t.Fatalf("sync = %+v, want SkippedOverrides", result)
	}
	price := store.lookupPrice("claude", "claude-test-1", "", 3000)
	if price == nil || price.Input != 7000000 || price.Source != PriceSourceManual {
		t.Fatalf("manual overridden: %+v", price)
	}
}

func TestSyncFailureKeepsOldData(t *testing.T) {
	store := openTestStore(t)
	addTestPrice(t, store, "openai", "gpt-5", 0, 1000000, 2000000, PriceSourceModelsDev)
	cfg := syncTestConfig("http://127.0.0.1:1/unreachable")
	result := store.SyncPricesNow(t.Context(), cfg, 2000)
	if result.Error == "" {
		t.Fatal("failed fetch accepted, want error")
	}
	price := store.lookupPrice("codex", "gpt-5", "", 1000)
	if price == nil || price.Input != 1000000 {
		t.Fatalf("old price lost: %+v", price)
	}
	if got := store.PriceSetting(settingPriceSyncError); got == "" {
		t.Fatal("last_error setting empty after failure")
	}
	if got := store.PriceSetting(settingPriceSyncAttemptMs); got == "" {
		t.Fatal("last_attempt setting empty after failure")
	}
}

func TestSyncETagNotModified(t *testing.T) {
	store := openTestStore(t)
	var hits atomic.Int64
	server := syncTestServer(t, &hits, `"v1"`, syncFixture, http.StatusOK)
	defer server.Close()
	cfg := syncTestConfig(server.URL)
	first := store.SyncPricesNow(t.Context(), cfg, 1000)
	if first.Error != "" || first.Added == 0 {
		t.Fatalf("first sync = %+v", first)
	}
	second := store.SyncPricesNow(t.Context(), cfg, 2000)
	if second.Error != "" {
		t.Fatalf("second sync error: %s", second.Error)
	}
	if second.Added != 0 || second.Updated != 0 {
		t.Fatalf("304 sync = %+v, want no changes", second)
	}
	if hits.Load() != 2 {
		t.Fatalf("hits = %d, want 2", hits.Load())
	}
}

func TestEmbeddedLoadOnce(t *testing.T) {
	store := openTestStore(t)
	result, err := store.LoadEmbeddedSnapshot(9000)
	if err != nil {
		t.Fatalf("LoadEmbeddedSnapshot: %v", err)
	}
	if result.Added == 0 {
		t.Fatal("embedded load added nothing")
	}
	version := store.lookupPrice("codex", "gpt-5", "", 100)
	if version == nil || version.Source != PriceSourceEmbedded || version.ValidFromMs != 0 {
		t.Fatalf("embedded version = %+v, want source embedded from 0", version)
	}
	again, err := store.LoadEmbeddedSnapshot(9000)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if again.Added != 0 || again.Updated != 0 {
		t.Fatalf("second load = %+v, want no-op", again)
	}
}
