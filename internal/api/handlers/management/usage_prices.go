package management

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Su3h7aM/LLMProxyAPI/internal/usagedb"
	"github.com/gin-gonic/gin"
)

// priceTierJSON is one tier row of a price version response.
type priceTierJSON struct {
	Kind       string `json:"kind"`
	Threshold  int64  `json:"threshold_tokens"`
	Service    string `json:"service"`
	Input      *int64 `json:"input"`
	Output     *int64 `json:"output"`
	CacheRead  *int64 `json:"cache_read"`
	CacheWrite *int64 `json:"cache_write"`
}

// priceJSON is one price version response. Rates are integer micro-USD
// per 1M tokens; divide by 1e6 for USD per 1M.
type priceJSON struct {
	ID          int64           `json:"id"`
	Provider    string          `json:"provider"`
	Model       string          `json:"model"`
	ValidFromMs int64           `json:"valid_from_ms"`
	ValidToMs   *int64          `json:"valid_to_ms"`
	Input       int64           `json:"input"`
	Output      int64           `json:"output"`
	CacheRead   int64           `json:"cache_read"`
	CacheWrite  int64           `json:"cache_write"`
	Reasoning   *int64          `json:"reasoning"`
	Source      string          `json:"source"`
	SyncedAtMs  int64           `json:"synced_at_ms"`
	Note        string          `json:"note"`
	Tiers       []priceTierJSON `json:"tiers"`
}

func toPriceJSON(price usagedb.Price) priceJSON {
	out := priceJSON{
		ID:          price.ID,
		Provider:    price.ProviderKey,
		Model:       price.ModelID,
		ValidFromMs: price.ValidFromMs,
		ValidToMs:   price.ValidToMs,
		Input:       price.Input,
		Output:      price.Output,
		CacheRead:   price.CacheRead,
		CacheWrite:  price.CacheWrite,
		Reasoning:   price.Reasoning,
		Source:      price.Source,
		SyncedAtMs:  price.SyncedAtMs,
		Note:        price.Note,
		Tiers:       []priceTierJSON{},
	}
	for _, tier := range price.Tiers {
		out.Tiers = append(out.Tiers, priceTierJSON{
			Kind:       tier.Kind,
			Threshold:  tier.Threshold,
			Service:    tier.Service,
			Input:      tier.Input,
			Output:     tier.Output,
			CacheRead:  tier.CacheRead,
			CacheWrite: tier.CacheWrite,
		})
	}
	return out
}

// usagePriceStore returns the open usage database or reports 503.
func (h *Handler) usagePriceStore(c *gin.Context) *usagedb.Store {
	store := usagedb.ActiveStore()
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "usage database is disabled"})
		return nil
	}
	return store
}

func queryInt(c *gin.Context, key string, def int) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return def
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return value
}

// GetUsagePrices lists current price versions with filters and paging.
func (h *Handler) GetUsagePrices(c *gin.Context) {
	store := h.usagePriceStore(c)
	if store == nil {
		return
	}
	prices, total, err := store.CurrentPrices(
		c.Query("provider"), c.Query("model"), c.Query("source"), c.Query("q"),
		queryInt(c, "limit", 100), queryInt(c, "offset", 0))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "price list failed"})
		return
	}
	out := make([]priceJSON, 0, len(prices))
	for _, price := range prices {
		out = append(out, toPriceJSON(price))
	}
	c.JSON(http.StatusOK, gin.H{
		"prices": out,
		"total":  total,
		"limit":  queryInt(c, "limit", 100),
		"offset": queryInt(c, "offset", 0),
	})
}

// GetUsagePriceHistory returns every version of one model, oldest first.
func (h *Handler) GetUsagePriceHistory(c *gin.Context) {
	store := h.usagePriceStore(c)
	if store == nil {
		return
	}
	provider := strings.TrimSpace(c.Query("provider"))
	model := strings.TrimSpace(c.Query("model"))
	if provider == "" || model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider and model are required"})
		return
	}
	versions, err := store.PriceHistory(provider, model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "price history failed"})
		return
	}
	out := make([]priceJSON, 0, len(versions))
	for _, price := range versions {
		out = append(out, toPriceJSON(price))
	}
	c.JSON(http.StatusOK, gin.H{"provider": provider, "model": model, "versions": out})
}

// putUsagePrice is the body of PUT /observability/usage/prices. Rates are
// USD per 1M tokens, matching models.dev; they convert to integer micro-USD
// with half-up rounding.
type putUsagePrice struct {
	Provider        string   `json:"provider"`
	Model           string   `json:"model"`
	Input           *float64 `json:"input"`
	Output          *float64 `json:"output"`
	CacheRead       *float64 `json:"cache_read"`
	CacheWrite      *float64 `json:"cache_write"`
	Reasoning       *float64 `json:"reasoning"`
	EffectiveFromMs *int64   `json:"effective_from_ms"`
	Note            string   `json:"note"`
}

// PutUsagePrice closes the current version at effective_from_ms (default
// now) and appends a manual version. History rows are never rewritten.
func (h *Handler) PutUsagePrice(c *gin.Context) {
	store := h.usagePriceStore(c)
	if store == nil {
		return
	}
	var body putUsagePrice
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid price body"})
		return
	}
	provider := strings.TrimSpace(body.Provider)
	model := strings.TrimSpace(body.Model)
	if provider == "" || model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider and model are required"})
		return
	}
	if body.Input == nil || body.Output == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "input and output rates are required"})
		return
	}
	for _, rate := range []*float64{body.Input, body.Output, body.CacheRead, body.CacheWrite, body.Reasoning} {
		if rate != nil && *rate < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "rates must not be negative"})
			return
		}
	}
	effective := time.Now().UnixMilli()
	if body.EffectiveFromMs != nil {
		effective = *body.EffectiveFromMs
	}
	price := usagedb.Price{
		ProviderKey: provider,
		ModelID:     model,
		ValidFromMs: effective,
		Input:       usagedb.MicroPerMillion(*body.Input),
		Output:      usagedb.MicroPerMillion(*body.Output),
		Note:        strings.TrimSpace(body.Note),
	}
	if body.CacheRead != nil {
		price.CacheRead = usagedb.MicroPerMillion(*body.CacheRead)
	}
	if body.CacheWrite != nil {
		price.CacheWrite = usagedb.MicroPerMillion(*body.CacheWrite)
	}
	if body.Reasoning != nil {
		value := usagedb.MicroPerMillion(*body.Reasoning)
		price.Reasoning = &value
	}
	id, err := store.AddManualPrice(price, time.Now().UnixMilli())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	store.FillUnpricedAsync([][2]string{{provider, model}})
	version, err := store.PriceVersion(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "price stored, reload failed"})
		return
	}
	c.JSON(http.StatusOK, toPriceJSON(version))
}

// DeleteUsagePriceManual closes the current manual version so the synced
// price applies again from now. It never deletes rows.
func (h *Handler) DeleteUsagePriceManual(c *gin.Context) {
	store := h.usagePriceStore(c)
	if store == nil {
		return
	}
	provider := strings.TrimSpace(c.Query("provider"))
	model := strings.TrimSpace(c.Query("model"))
	if provider == "" || model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider and model are required"})
		return
	}
	restored, err := store.RemoveManualPrice(provider, model, time.Now().UnixMilli())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	store.FillUnpricedAsync([][2]string{{provider, model}})
	c.JSON(http.StatusOK, gin.H{"provider": provider, "model": model, "restored": restored})
}

// PostUsagePricesSync runs one models.dev sync pass now.
func (h *Handler) PostUsagePricesSync(c *gin.Context) {
	store := h.usagePriceStore(c)
	if store == nil {
		return
	}
	result := store.SyncPricesNow(c.Request.Context(), usagedb.ActivePriceSyncConfig(), time.Now().UnixMilli())
	c.JSON(http.StatusOK, gin.H{
		"added":             result.Added,
		"updated":           result.Updated,
		"unchanged":         result.Unchanged,
		"skipped_overrides": result.SkippedOverrides,
		"error":             result.Error,
		"duration_ms":       result.DurationMs,
	})
}

func parseSettingMs(raw string) int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

// GetUsagePricesStatus reports the last sync, counts by source, and models
// seen in usage with no price.
func (h *Handler) GetUsagePricesStatus(c *gin.Context) {
	store := h.usagePriceStore(c)
	if store == nil {
		return
	}
	counts, err := store.PricesBySource()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "price status failed"})
		return
	}
	unmatched, err := store.UnmatchedModels(100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "price status failed"})
		return
	}
	models := make([]gin.H, 0, len(unmatched))
	for _, item := range unmatched {
		models = append(models, gin.H{
			"provider": item.Provider,
			"model":    item.Model,
			"events":   item.Events,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"last_sync_ms":     parseSettingMs(store.PriceSetting("price_sync_last_ms")),
		"last_attempt_ms":  parseSettingMs(store.PriceSetting("price_sync_attempt_ms")),
		"last_error":       store.PriceSetting("price_sync_error"),
		"counts_by_source": counts,
		"unmatched":        models,
	})
}

// priceUnpricedBody bounds POST /observability/usage/prices/price-unpriced.
type priceUnpricedBody struct {
	FromMs   *int64 `json:"from_ms"`
	ToMs     *int64 `json:"to_ms"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// PostUsagePricesPriceUnpriced fills NULL-cost events with the version valid
// at each event time (or the nearest). It never touches priced events.
func (h *Handler) PostUsagePricesPriceUnpriced(c *gin.Context) {
	store := h.usagePriceStore(c)
	if store == nil {
		return
	}
	var body priceUnpricedBody
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
	}
	start := time.Now()
	result, err := store.PriceUnpriced(usagedb.FillScope{
		FromMs:   body.FromMs,
		ToMs:     body.ToMs,
		Provider: strings.TrimSpace(body.Provider),
		Model:    strings.TrimSpace(body.Model),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "price-unpriced failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"priced":           result.Priced,
		"skipped_no_price": result.SkippedNoPrice,
		"duration_ms":      time.Since(start).Milliseconds(),
	})
}

// correctUsagePriceBody is the audited correction request. Confirm must be
// true and the range must not exceed 366 days.
type correctUsagePriceBody struct {
	FromMs   *int64 `json:"from_ms"`
	ToMs     *int64 `json:"to_ms"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Confirm  bool   `json:"confirm"`
	Reason   string `json:"reason"`
}

// PostUsagePricesCorrect recomputes cost for a range with the time-valid
// versions, fixes rollups in the same transaction, and writes an audit row.
func (h *Handler) PostUsagePricesCorrect(c *gin.Context) {
	store := h.usagePriceStore(c)
	if store == nil {
		return
	}
	var body correctUsagePriceBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if !body.Confirm {
		c.JSON(http.StatusBadRequest, gin.H{"error": "confirm must be true"})
		return
	}
	if strings.TrimSpace(body.Reason) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason is required"})
		return
	}
	id, updated, oldTotal, newTotal, err := store.CorrectCosts(usagedb.FillScope{
		FromMs:   body.FromMs,
		ToMs:     body.ToMs,
		Provider: strings.TrimSpace(body.Provider),
		Model:    strings.TrimSpace(body.Model),
	}, strings.TrimSpace(body.Reason), "management-api")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"correction_id":      id,
		"events_updated":     updated,
		"old_cost_micro_usd": oldTotal,
		"new_cost_micro_usd": newTotal,
	})
}
