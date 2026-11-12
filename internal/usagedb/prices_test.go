package usagedb

import (
	"database/sql"
	"testing"
)

func int64Ptr(value int64) *int64 { return &value }

func TestMicroPerMillionRoundsHalfUp(t *testing.T) {
	cases := []struct {
		usd  float64
		want int64
	}{
		{1, 1000000},
		{0.1, 100000},
		{0.0000005, 1}, // 0.5 micro rounds up
		{0.0000004, 0},
		{2.5, 2500000},
		{0, 0},
		{-1, 0},
	}
	for _, test := range cases {
		if got := MicroPerMillion(test.usd); got != test.want {
			t.Errorf("MicroPerMillion(%v) = %d, want %d", test.usd, got, test.want)
		}
	}
}

func TestComputeCostBuckets(t *testing.T) {
	price := Price{
		Input:      1000000, // 1 USD per 1M
		Output:     2000000,
		CacheRead:  100000,
		CacheWrite: 200000,
	}
	// 1M uncached input + 1M output = 3 USD = 3000000 micro.
	if got := price.ComputeCost("default", 1000000, 0, 0, 1000000, 0); got != 3000000 {
		t.Fatalf("cost = %d, want 3000000", got)
	}
	// Cache buckets price separately; reasoning falls back to output.
	if got := price.ComputeCost("default", 0, 1000000, 1000000, 0, 1000000); got != 2300000 {
		t.Fatalf("cost = %d, want 2300000", got)
	}
	// Explicit reasoning rate wins over output.
	reasoning := int64(5000000)
	price.Reasoning = &reasoning
	if got := price.ComputeCost("default", 0, 0, 0, 0, 1000000); got != 5000000 {
		t.Fatalf("cost = %d, want 5000000", got)
	}
	// One token at half a micro-USD rounds up once.
	half := Price{Input: 500000}
	if got := half.ComputeCost("default", 1, 0, 0, 0, 0); got != 1 {
		t.Fatalf("cost = %d, want 1", got)
	}
}

func TestComputeCostContextTierStrictlyGreater(t *testing.T) {
	high := int64(10000000)
	price := Price{
		Input:  1000000,
		Output: 1000000,
		Tiers: []PriceTier{
			{Kind: PriceTierContext, Threshold: 100, Input: &high, Output: &high},
		},
	}
	// Exactly at the threshold stays in the base band.
	if got := price.ComputeCost("default", 100, 0, 0, 0, 0); got != 100 {
		t.Fatalf("at threshold cost = %d, want 100", got)
	}
	// Above the threshold takes the tier.
	if got := price.ComputeCost("default", 101, 0, 0, 0, 0); got != 1010 {
		t.Fatalf("above threshold cost = %d, want 1010", got)
	}
}

func TestComputeCostServiceTierOverride(t *testing.T) {
	flex := int64(500000)
	price := Price{
		Input:  1000000,
		Output: 1000000,
		Tiers: []PriceTier{
			{Kind: PriceTierService, Service: "flex", Input: &flex, Output: &flex},
		},
	}
	if got := price.ComputeCost("flex", 1000000, 0, 0, 0, 0); got != 500000 {
		t.Fatalf("flex cost = %d, want 500000", got)
	}
	if got := price.ComputeCost("default", 1000000, 0, 0, 0, 0); got != 1000000 {
		t.Fatalf("default cost = %d, want 1000000", got)
	}
}

func addTestPrice(t *testing.T, store *Store, provider, model string, from int64, input, output int64, source string) int64 {
	t.Helper()
	id, err := store.AddPriceVersion(Price{
		ProviderKey: provider,
		ModelID:     model,
		ValidFromMs: from,
		Input:       input,
		Output:      output,
		Source:      source,
	})
	if err != nil {
		t.Fatalf("AddPriceVersion(%s/%s): %v", provider, model, err)
	}
	return id
}

func TestPriceLookupMappedProvider(t *testing.T) {
	store := openTestStore(t)
	id := addTestPrice(t, store, "openai", "gpt-5", 0, 1000000, 2000000, PriceSourceModelsDev)
	price := store.lookupPrice("codex", "gpt-5", "work", 1000)
	if price == nil || price.ID != id {
		t.Fatalf("lookup = %+v, want id %d", price, id)
	}
}

func TestPriceLookupAliasFallback(t *testing.T) {
	store := openTestStore(t)
	id := addTestPrice(t, store, "openai", "gpt-5-real", 0, 1000000, 2000000, PriceSourceModelsDev)
	price := store.lookupPrice("codex", "unknown-model", "gpt-5-real", 1000)
	if price == nil || price.ID != id {
		t.Fatalf("alias lookup = %+v, want id %d", price, id)
	}
}

func TestPriceLookupDateSuffixBothWays(t *testing.T) {
	store := openTestStore(t)
	id := addTestPrice(t, store, "anthropic", "claude-opus-4-5-20251101", 0, 1000000, 2000000, PriceSourceModelsDev)
	// Stripped event model matches the suffixed version.
	if price := store.lookupPrice("claude", "claude-opus-4-5", "", 1000); price == nil || price.ID != id {
		t.Fatalf("stripped lookup = %+v, want id %d", price, id)
	}
	// Suffixed event model matches a stripped version.
	id2 := addTestPrice(t, store, "anthropic", "claude-haiku-4-5", 0, 1000000, 2000000, PriceSourceModelsDev)
	if price := store.lookupPrice("claude", "claude-haiku-4-5-20251001", "", 1000); price == nil || price.ID != id2 {
		t.Fatalf("suffixed lookup = %+v, want id %d", price, id2)
	}
	// Dash date suffix also strips.
	id3 := addTestPrice(t, store, "openai", "gpt-5-2025-01-01", 0, 1000000, 2000000, PriceSourceModelsDev)
	if price := store.lookupPrice("codex", "gpt-5", "", 1000); price == nil || price.ID != id3 {
		t.Fatalf("dash suffix lookup = %+v, want id %d", price, id3)
	}
}

func TestPriceLookupCrossProviderFallback(t *testing.T) {
	store := openTestStore(t)
	id := addTestPrice(t, store, "openai", "shared-model", 0, 1000000, 2000000, PriceSourceModelsDev)
	// The fork provider "devin" has no models.dev mapping; exact model id
	// still resolves through the preference list.
	price := store.lookupPrice("devin", "shared-model", "", 1000)
	if price == nil || price.ID != id {
		t.Fatalf("cross-provider lookup = %+v, want id %d", price, id)
	}
	if price := store.lookupPrice("devin", "no-such-model", "", 1000); price != nil {
		t.Fatalf("unknown lookup = %+v, want nil", price)
	}
}

func TestPriceLookupTimeValidity(t *testing.T) {
	store := openTestStore(t)
	oldID := addTestPrice(t, store, "openai", "gpt-5", 0, 1000000, 2000000, PriceSourceModelsDev)
	newID := addTestPrice(t, store, "openai", "gpt-5", 2000, 2000000, 4000000, PriceSourceModelsDev)
	if price := store.lookupPrice("codex", "gpt-5", "", 1000); price == nil || price.ID != oldID {
		t.Fatalf("before change lookup = %+v, want id %d", price, oldID)
	}
	// At the boundary the new version already applies.
	if price := store.lookupPrice("codex", "gpt-5", "", 2000); price == nil || price.ID != newID {
		t.Fatalf("at change lookup = %+v, want id %d", price, newID)
	}
	if price := store.lookupPrice("codex", "gpt-5", "", 3000); price == nil || price.ID != newID {
		t.Fatalf("after change lookup = %+v, want id %d", price, newID)
	}
}

func TestPriceLookupManualWins(t *testing.T) {
	store := openTestStore(t)
	syncedID := addTestPrice(t, store, "openai", "gpt-5", 0, 1000000, 2000000, PriceSourceModelsDev)
	manualID, err := store.AddManualPrice(Price{
		ProviderKey: "openai",
		ModelID:     "gpt-5",
		ValidFromMs: 1000,
		Input:       500000,
		Output:      500000,
	}, 1000)
	if err != nil {
		t.Fatalf("AddManualPrice: %v", err)
	}
	if price := store.lookupPrice("codex", "gpt-5", "", 500); price == nil || price.ID != syncedID {
		t.Fatalf("before manual lookup = %+v, want id %d", price, syncedID)
	}
	if price := store.lookupPrice("codex", "gpt-5", "", 1500); price == nil || price.ID != manualID {
		t.Fatalf("manual lookup = %+v, want id %d", price, manualID)
	}
}

func TestPriceAppendOnlyGuard(t *testing.T) {
	store := openTestStore(t)
	addTestPrice(t, store, "openai", "gpt-5", 0, 1000000, 2000000, PriceSourceModelsDev)
	// A second version at the same start must fail: history is frozen and
	// the public API offers no way to rewrite an old row.
	if _, err := store.AddPriceVersion(Price{
		ProviderKey: "openai", ModelID: "gpt-5", ValidFromMs: 0,
		Input: 1, Output: 1, Source: PriceSourceModelsDev,
	}); err == nil {
		t.Fatal("duplicate version start accepted, want error")
	}
	// A later version closes the old one but never changes its rates.
	addTestPrice(t, store, "openai", "gpt-5", 2000, 3000000, 4000000, PriceSourceModelsDev)
	var firstInput int64
	var validTo sql.NullInt64
	if err := store.readDB.QueryRow(`SELECT input, valid_to_ms FROM model_prices
		WHERE provider_key = 'openai' AND model_id = 'gpt-5' ORDER BY valid_from_ms LIMIT 1`,
	).Scan(&firstInput, &validTo); err != nil {
		t.Fatalf("read first version: %v", err)
	}
	if firstInput != 1000000 {
		t.Fatalf("first version input = %d, want 1000000 (old rows never change)", firstInput)
	}
	if !validTo.Valid || validTo.Int64 != 2000 {
		t.Fatalf("first version valid_to = %+v, want 2000 (closed exactly once)", validTo)
	}
}
