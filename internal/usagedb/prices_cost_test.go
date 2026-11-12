package usagedb

import (
	"database/sql"
	"testing"

	coreusage "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/usage"
)

func pricedRow(requestID string, tsMs int64) UsageRow {
	row := testRow(requestID, tsMs)
	row.Provider = "codex"
	row.Executor = "codex"
	row.Model = "gpt-5"
	row.Alias = "gpt-5"
	row.InputTokens = 1000000
	row.OutputTokens = 1000000
	row.TotalTokens = 2000000
	return row
}

func queryEventCost(t *testing.T, store *Store, requestID string) (cost sql.NullInt64, source sql.NullString, priceID sql.NullInt64) {
	t.Helper()
	if err := store.readDB.QueryRow(`SELECT cost_micro_usd, cost_source, price_id
		FROM usage_events WHERE request_id = ?`, requestID).Scan(&cost, &source, &priceID); err != nil {
		t.Fatalf("query event cost: %v", err)
	}
	return cost, source, priceID
}

func hourlyCost(t *testing.T, store *Store) (cost, unpriced int64) {
	t.Helper()
	if err := store.readDB.QueryRow(`SELECT COALESCE(SUM(cost_micro_usd), 0), COALESCE(SUM(unpriced_requests), 0)
		FROM usage_hourly`).Scan(&cost, &unpriced); err != nil {
		t.Fatalf("hourly cost: %v", err)
	}
	return cost, unpriced
}

func TestInsertCostBeforeAndAfterPriceChange(t *testing.T) {
	store := openTestStore(t)
	addTestPrice(t, store, "openai", "gpt-5", 0, 1000000, 2000000, PriceSourceModelsDev)
	addTestPrice(t, store, "openai", "gpt-5", 2000, 2000000, 4000000, PriceSourceModelsDev)
	before := pricedRow("cost-before", 1000)
	after := pricedRow("cost-after", 3000)
	if _, err := store.InsertBatch([]UsageRow{before, after}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	cost, source, priceID := queryEventCost(t, store, "cost-before")
	if !cost.Valid || cost.Int64 != 3000000 || source.String != CostSourcePriceBook || !priceID.Valid {
		t.Fatalf("before: cost=%+v source=%q price=%+v, want 3000000/price-book/set", cost, source.String, priceID)
	}
	cost, _, _ = queryEventCost(t, store, "cost-after")
	if !cost.Valid || cost.Int64 != 6000000 {
		t.Fatalf("after: cost=%+v, want 6000000", cost)
	}
	if total, unpriced := hourlyCost(t, store); total != 9000000 || unpriced != 0 {
		t.Fatalf("hourly: cost=%d unpriced=%d, want 9000000/0", total, unpriced)
	}
}

func TestInsertCostDuplicateNoDoubleCount(t *testing.T) {
	store := openTestStore(t)
	addTestPrice(t, store, "openai", "gpt-5", 0, 1000000, 2000000, PriceSourceModelsDev)
	row := pricedRow("cost-dup", 1000)
	if _, err := store.InsertBatch([]UsageRow{row}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	if _, err := store.InsertBatch([]UsageRow{row}); err != nil {
		t.Fatalf("InsertBatch duplicate: %v", err)
	}
	if total, _ := hourlyCost(t, store); total != 3000000 {
		t.Fatalf("hourly cost = %d, want 3000000 (duplicate must not count)", total)
	}
}

func TestInsertCostUpstreamWins(t *testing.T) {
	store := openTestStore(t)
	addTestPrice(t, store, "openai", "gpt-5", 0, 1000000, 2000000, PriceSourceModelsDev)
	row := pricedRow("cost-upstream", 1000)
	upstream := int64(42)
	row.UpstreamCostMicroUSD = &upstream
	if _, err := store.InsertBatch([]UsageRow{row}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	cost, source, priceID := queryEventCost(t, store, "cost-upstream")
	if !cost.Valid || cost.Int64 != 42 || source.String != CostSourceUpstream || priceID.Valid {
		t.Fatalf("upstream: cost=%+v source=%q price=%+v, want 42/upstream/NULL", cost, source.String, priceID)
	}
}

func TestInsertCostNoneWhenNoPrice(t *testing.T) {
	store := openTestStore(t)
	row := pricedRow("cost-none", 1000)
	if _, err := store.InsertBatch([]UsageRow{row}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	cost, source, priceID := queryEventCost(t, store, "cost-none")
	if cost.Valid || source.String != CostSourceNone || priceID.Valid {
		t.Fatalf("none: cost=%+v source=%q price=%+v, want NULL/none/NULL", cost, source.String, priceID)
	}
	if _, unpriced := hourlyCost(t, store); unpriced != 1 {
		t.Fatalf("hourly unpriced = %d, want 1", unpriced)
	}
}

func TestUpstreamCostConversion(t *testing.T) {
	record := testRecord()
	record.Detail.CostDetails = &coreusage.CostDetails{Cost: coreusage.Float64Ptr(0.0045)}
	row := RecordToRow(record)
	if row.UpstreamCostMicroUSD == nil || *row.UpstreamCostMicroUSD != 4500 {
		t.Fatalf("upstream micro = %+v, want 4500", row.UpstreamCostMicroUSD)
	}
	record.Detail.CostDetails = &coreusage.CostDetails{
		UpstreamInferencePromptCost:      coreusage.Float64Ptr(0.001),
		UpstreamInferenceCompletionsCost: coreusage.Float64Ptr(0.002),
	}
	row = RecordToRow(record)
	if row.UpstreamCostMicroUSD == nil || *row.UpstreamCostMicroUSD != 3000 {
		t.Fatalf("split upstream micro = %+v, want 3000", row.UpstreamCostMicroUSD)
	}
	record.Detail.CostDetails = &coreusage.CostDetails{CostInUSDTicks: coreusage.Int64Ptr(99)}
	if row = RecordToRow(record); row.UpstreamCostMicroUSD != nil {
		t.Fatalf("ticks upstream micro = %+v, want nil (unit not defined)", row.UpstreamCostMicroUSD)
	}
}

func TestLatePricingFillsAfterVersionAppears(t *testing.T) {
	store := openTestStore(t)
	before := pricedRow("late-1", 1000)
	after := pricedRow("late-2", 3000)
	if _, err := store.InsertBatch([]UsageRow{before, after}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	if _, unpriced := hourlyCost(t, store); unpriced != 2 {
		t.Fatalf("hourly unpriced = %d, want 2", unpriced)
	}
	oldID := addTestPrice(t, store, "openai", "gpt-5", 0, 1000000, 2000000, PriceSourceModelsDev)
	newID := addTestPrice(t, store, "openai", "gpt-5", 2000, 2000000, 4000000, PriceSourceModelsDev)
	result, err := store.FillUnpricedForKeys([][2]string{{"openai", "gpt-5"}})
	if err != nil {
		t.Fatalf("FillUnpricedForKeys: %v", err)
	}
	if result.Priced != 2 {
		t.Fatalf("priced = %d, want 2", result.Priced)
	}
	// Each event takes the version valid at its own time.
	cost, source, priceID := queryEventCost(t, store, "late-1")
	if !cost.Valid || cost.Int64 != 3000000 || source.String != CostSourcePriceBookLate ||
		!priceID.Valid || priceID.Int64 != oldID {
		t.Fatalf("late-1: cost=%+v source=%q price=%+v, want 3000000/price-book-late/%d",
			cost, source.String, priceID, oldID)
	}
	cost, _, priceID = queryEventCost(t, store, "late-2")
	if !cost.Valid || cost.Int64 != 6000000 || !priceID.Valid || priceID.Int64 != newID {
		t.Fatalf("late-2: cost=%+v price=%+v, want 6000000/%d", cost, priceID, newID)
	}
	if total, unpriced := hourlyCost(t, store); unpriced != 0 || total != 9000000 {
		t.Fatalf("hourly: cost=%d unpriced=%d, want 9000000/0", total, unpriced)
	}
	// Idempotent: a second run changes nothing.
	again, err := store.FillUnpricedForKeys([][2]string{{"openai", "gpt-5"}})
	if err != nil {
		t.Fatalf("refill: %v", err)
	}
	if again.Priced != 0 {
		t.Fatalf("refill priced = %d, want 0", again.Priced)
	}
}

func TestLatePricingKeepsPricedEvents(t *testing.T) {
	store := openTestStore(t)
	addTestPrice(t, store, "openai", "gpt-5", 0, 1000000, 2000000, PriceSourceModelsDev)
	priced := pricedRow("keep-priced", 1000)
	if _, err := store.InsertBatch([]UsageRow{priced}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	before, _, _ := queryEventCost(t, store, "keep-priced")
	addTestPrice(t, store, "openai", "gpt-5", 2000, 9000000, 9000000, PriceSourceModelsDev)
	if _, err := store.FillUnpricedForKeys([][2]string{{"openai", "gpt-5"}}); err != nil {
		t.Fatalf("fill: %v", err)
	}
	after, source, _ := queryEventCost(t, store, "keep-priced")
	if !after.Valid || after.Int64 != before.Int64 || source.String != CostSourcePriceBook {
		t.Fatalf("priced event changed: %+v/%q, want %+v/price-book", after, source.String, before)
	}
}

func TestLatePricingModelsDoNotMix(t *testing.T) {
	store := openTestStore(t)
	row := pricedRow("mix-1", 1000)
	row.Model = "other-model"
	row.Alias = "other-model"
	if _, err := store.InsertBatch([]UsageRow{row}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	addTestPrice(t, store, "openai", "gpt-5", 0, 1000000, 2000000, PriceSourceModelsDev)
	result, err := store.FillUnpricedForKeys([][2]string{{"openai", "gpt-5"}})
	if err != nil {
		t.Fatalf("fill: %v", err)
	}
	if result.Priced != 0 {
		t.Fatalf("priced = %d, want 0 (other model must not mix)", result.Priced)
	}
	cost, _, _ := queryEventCost(t, store, "mix-1")
	if cost.Valid {
		t.Fatalf("other model cost = %+v, want NULL", cost)
	}
}

func TestLatePricingOlderThanFirstTakesFirst(t *testing.T) {
	store := openTestStore(t)
	row := pricedRow("late-old", 1000)
	if _, err := store.InsertBatch([]UsageRow{row}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	// The first version starts after the event: the event takes the
	// nearest version, which is the first one.
	firstID := addTestPrice(t, store, "openai", "gpt-5", 2000, 1000000, 2000000, PriceSourceModelsDev)
	result, err := store.FillUnpricedForKeys([][2]string{{"openai", "gpt-5"}})
	if err != nil {
		t.Fatalf("fill: %v", err)
	}
	if result.Priced != 1 {
		t.Fatalf("priced = %d, want 1", result.Priced)
	}
	cost, source, priceID := queryEventCost(t, store, "late-old")
	if !cost.Valid || cost.Int64 != 3000000 || source.String != CostSourcePriceBookLate ||
		!priceID.Valid || priceID.Int64 != firstID {
		t.Fatalf("late-old: cost=%+v source=%q price=%+v, want 3000000/price-book-late/%d",
			cost, source.String, priceID, firstID)
	}
}

func TestCorrectCostsRecomputesAndAudits(t *testing.T) {
	store := openTestStore(t)
	addTestPrice(t, store, "openai", "gpt-5", 0, 1000000, 2000000, PriceSourceModelsDev)
	row := pricedRow("correct-1", 1000)
	if _, err := store.InsertBatch([]UsageRow{row}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	// A wrong manual price must not change the stored event by itself.
	if _, err := store.AddManualPrice(Price{
		ProviderKey: "openai", ModelID: "gpt-5", ValidFromMs: 500,
		Input: 9000000, Output: 9000000,
	}, 500); err != nil {
		t.Fatalf("AddManualPrice: %v", err)
	}
	cost, _, _ := queryEventCost(t, store, "correct-1")
	if !cost.Valid || cost.Int64 != 3000000 {
		t.Fatalf("stored cost changed to %+v, want 3000000", cost)
	}
	from, to := int64(0), int64(2000)
	correctionID, updated, oldTotal, newTotal, err := store.CorrectCosts(FillScope{
		FromMs: &from, ToMs: &to,
	}, "wrong manual price", "test")
	if err != nil {
		t.Fatalf("CorrectCosts: %v", err)
	}
	if correctionID == 0 || updated != 1 || oldTotal != 3000000 || newTotal != 18000000 {
		t.Fatalf("correction = (%d, %d, %d, %d), want (id, 1, 3000000, 18000000)",
			correctionID, updated, oldTotal, newTotal)
	}
	cost, source, _ := queryEventCost(t, store, "correct-1")
	if !cost.Valid || cost.Int64 != 18000000 || source.String != CostSourcePriceBook {
		t.Fatalf("corrected: cost=%+v source=%q, want 18000000/price-book", cost, source.String)
	}
	if total, _ := hourlyCost(t, store); total != 18000000 {
		t.Fatalf("hourly cost = %d, want 18000000", total)
	}
	var auditCount int64
	if err := store.readDB.QueryRow(`SELECT COUNT(*) FROM cost_corrections WHERE id = ?`,
		correctionID).Scan(&auditCount); err != nil {
		t.Fatalf("audit: %v", err)
	}
	if auditCount != 1 {
		t.Fatal("correction audit row missing")
	}
}

func TestCorrectCostsRejects(t *testing.T) {
	store := openTestStore(t)
	from, to := int64(0), int64(2000)
	if _, _, _, _, err := store.CorrectCosts(FillScope{FromMs: &from, ToMs: &to}, "", "test"); err == nil {
		t.Fatal("empty reason accepted, want error")
	}
	wideTo := int64(367 * 24 * 60 * 60 * 1000)
	if _, _, _, _, err := store.CorrectCosts(FillScope{FromMs: &from, ToMs: &wideTo}, "r", "test"); err == nil {
		t.Fatal("range over 366 days accepted, want error")
	}
}
