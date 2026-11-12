package usagedb

import (
	"database/sql"
	"testing"
)

func hourlyCostLate(t *testing.T, store *Store) (cost, late, unpriced int64) {
	t.Helper()
	if err := store.readDB.QueryRow(`SELECT cost_micro_usd, late_cost_micro_usd, unpriced_requests FROM usage_hourly`).Scan(
		&cost, &late, &unpriced); err != nil {
		t.Fatalf("query hourly: %v", err)
	}
	return cost, late, unpriced
}

func eventCost(t *testing.T, store *Store, requestID string) (int64, string, bool) {
	t.Helper()
	var cost sql.NullInt64
	var source sql.NullString
	if err := store.readDB.QueryRow(`SELECT cost_micro_usd, cost_source FROM usage_events WHERE request_id = ?`,
		requestID).Scan(&cost, &source); err != nil {
		t.Fatalf("query event: %v", err)
	}
	return cost.Int64, source.String, cost.Valid
}

func TestLateCostFillAndCorrect(t *testing.T) {
	store := openTestStore(t)
	ts := int64(1720000000000)
	row := testRow("late-1", ts)
	row.Provider = "codex"
	row.Model = "gpt-5"
	row.Alias = "gpt-5"
	if _, err := store.InsertBatch([]UsageRow{row}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	if _, _, hasCost := eventCost(t, store, "late-1"); hasCost {
		t.Fatal("event has cost before any price exists")
	}

	if _, err := store.AddPriceVersion(Price{
		ProviderKey: "openai", ModelID: "gpt-5", ValidFromMs: 0,
		Input: 1000000, Output: 2000000, Source: PriceSourceModelsDev,
	}); err != nil {
		t.Fatalf("AddPriceVersion: %v", err)
	}
	result, err := store.FillUnpricedForKeys([][2]string{{"openai", "gpt-5"}})
	if err != nil {
		t.Fatalf("FillUnpricedForKeys: %v", err)
	}
	if result.Priced != 1 {
		t.Fatalf("priced = %d, want 1", result.Priced)
	}
	// Input 10 at 1.0/M plus output 20 at 2.0/M = 50 micro-USD.
	if cost, source, _ := eventCost(t, store, "late-1"); cost != 50 || source != CostSourcePriceBookLate {
		t.Fatalf("event cost = %d %q, want 50 price-book-late", cost, source)
	}
	if cost, late, unpriced := hourlyCostLate(t, store); cost != 50 || late != 50 || unpriced != 0 {
		t.Fatalf("hourly cost/late/unpriced = %d/%d/%d, want 50/50/0", cost, late, unpriced)
	}

	from, to := ts-1, ts+1
	if _, _, _, _, err := store.CorrectCosts(FillScope{FromMs: &from, ToMs: &to}, "test reason", "test"); err != nil {
		t.Fatalf("CorrectCosts: %v", err)
	}
	if cost, source, _ := eventCost(t, store, "late-1"); cost != 50 || source != CostSourcePriceBook {
		t.Fatalf("after correct: cost = %d %q, want 50 price-book", cost, source)
	}
	if cost, late, _ := hourlyCostLate(t, store); cost != 50 || late != 0 {
		t.Fatalf("after correct: hourly cost/late = %d/%d, want 50/0", cost, late)
	}
}
