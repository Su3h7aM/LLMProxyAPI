package usagedb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Su3h7aM/LLMProxyAPI/sdk/proxyutil"
	log "github.com/sirupsen/logrus"
)

// DefaultModelsDevURL is the public price source synced by the price book.
const DefaultModelsDevURL = "https://models.dev/api.json"

// DefaultPriceSyncIntervalHours re-syncs the price book once a day.
const DefaultPriceSyncIntervalHours = 24

const (
	// priceSyncMaxBytes caps one models.dev download at 32 MB.
	priceSyncMaxBytes = 32 << 20
	// priceSyncTimeout bounds one models.dev fetch.
	priceSyncTimeout = 30 * time.Second
	// priceSyncStartupDelay waits this long after start before the first sync.
	priceSyncStartupDelay = 30 * time.Second
	// priceSyncUserAgent is the only identifying header sent with the fetch.
	// No keys and no client data leave the server.
	priceSyncUserAgent = "LLMProxyAPI-price-sync"
)

// Settings keys for the price sync state.
const (
	settingPriceSyncLastMs    = "price_sync_last_ms"
	settingPriceSyncAttemptMs = "price_sync_attempt_ms"
	settingPriceSyncError     = "price_sync_error"
	settingPriceSyncETag      = "price_sync_etag"
)

// PriceSyncConfig configures the models.dev sync. URL and IntervalHours
// carry defaults when empty or non-positive.
type PriceSyncConfig struct {
	Enabled       bool
	URL           string
	IntervalHours int
	ProxyURL      string
}

// EffectiveURL returns the configured URL or the models.dev default.
func (c PriceSyncConfig) EffectiveURL() string {
	if trimmed := strings.TrimSpace(c.URL); trimmed != "" {
		return trimmed
	}
	return DefaultModelsDevURL
}

// EffectiveInterval returns the configured cadence or the daily default.
func (c PriceSyncConfig) EffectiveInterval() time.Duration {
	if c.IntervalHours <= 0 {
		return time.Duration(DefaultPriceSyncIntervalHours) * time.Hour
	}
	return time.Duration(c.IntervalHours) * time.Hour
}

// SyncResult counts one sync pass by outcome.
type SyncResult struct {
	Added            int
	Updated          int
	Unchanged        int
	SkippedOverrides int
	Error            string
	DurationMs       int64
}

// modelsDevTier is one entry of a models.dev cost.tiers array.
type modelsDevTier struct {
	Input      *float64 `json:"input"`
	Output     *float64 `json:"output"`
	CacheRead  *float64 `json:"cache_read"`
	CacheWrite *float64 `json:"cache_write"`
	Reasoning  *float64 `json:"reasoning"`
	Tier       struct {
		Type string `json:"type"`
		Size *int64 `json:"size"`
	} `json:"tier"`
}

// modelsDevCost is the cost object of one models.dev model entry. Prices
// are USD per 1M tokens. Audio rates are parsed but not stored: usage rows
// carry no audio token buckets.
type modelsDevCost struct {
	Input           *float64        `json:"input"`
	Output          *float64        `json:"output"`
	CacheRead       *float64        `json:"cache_read"`
	CacheWrite      *float64        `json:"cache_write"`
	Reasoning       *float64        `json:"reasoning"`
	Tiers           []modelsDevTier `json:"tiers"`
	ContextOver200k *modelsDevTier  `json:"context_over_200k"`
}

// modelsDevModel is one entry of a models.dev provider models map.
type modelsDevModel struct {
	ID   string         `json:"id"`
	Cost *modelsDevCost `json:"cost"`
}

// modelsDevProvider is one entry of the top-level models.dev object.
type modelsDevProvider struct {
	Models map[string]modelsDevModel `json:"models"`
}

// syncedModel is one parsed price ready to apply.
type syncedModel struct {
	Provider string
	Model    string
	Price    Price
}

// ParseModelsDev parses one models.dev api.json document into prices.
// Models without a cost object are skipped. Only the fields the price book
// stores are kept.
func ParseModelsDev(data []byte) ([]syncedModel, error) {
	var doc map[string]modelsDevProvider
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("usagedb: parse models.dev: %w", err)
	}
	var out []syncedModel
	for provider, entry := range doc {
		provider = strings.TrimSpace(provider)
		if provider == "" {
			continue
		}
		for id, model := range entry.Models {
			id = strings.TrimSpace(id)
			if model.ID = strings.TrimSpace(model.ID); model.ID != "" {
				id = model.ID
			}
			if id == "" || model.Cost == nil {
				continue
			}
			cost := model.Cost
			if cost.Input == nil || cost.Output == nil {
				continue
			}
			price := Price{
				ProviderKey: provider,
				ModelID:     id,
				Input:       MicroPerMillion(*cost.Input),
				Output:      MicroPerMillion(*cost.Output),
				CacheRead:   MicroPerMillion(orZero(cost.CacheRead)),
				CacheWrite:  MicroPerMillion(orZero(cost.CacheWrite)),
			}
			if cost.Reasoning != nil {
				price.Reasoning = int64PtrCopy(MicroPerMillion(*cost.Reasoning))
			}
			for _, tier := range cost.Tiers {
				converted, ok := convertModelsDevTier(tier)
				if ok {
					price.Tiers = append(price.Tiers, converted)
				}
			}
			if cost.ContextOver200k != nil {
				legacy := *cost.ContextOver200k
				legacy.Tier.Type = PriceTierContext
				legacy.Tier.Size = int64PtrCopy(200000)
				if converted, ok := convertModelsDevTier(legacy); ok {
					price.Tiers = append(price.Tiers, converted)
				}
			}
			sort.Slice(price.Tiers, func(i, j int) bool {
				if price.Tiers[i].Threshold != price.Tiers[j].Threshold {
					return price.Tiers[i].Threshold < price.Tiers[j].Threshold
				}
				return price.Tiers[i].Kind < price.Tiers[j].Kind
			})
			out = append(out, syncedModel{Provider: provider, Model: id, Price: price})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Provider != out[j].Provider {
			return out[i].Provider < out[j].Provider
		}
		return out[i].Model < out[j].Model
	})
	return out, nil
}

func orZero(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func int64PtrCopy(value int64) *int64 {
	out := value
	return &out
}

// convertModelsDevTier keeps only context-threshold tiers with a positive
// size. Unknown tier types are skipped so a future models.dev shape cannot
// silently misprice events.
func convertModelsDevTier(tier modelsDevTier) (PriceTier, bool) {
	kind := strings.TrimSpace(tier.Tier.Type)
	if kind != PriceTierContext {
		return PriceTier{}, false
	}
	if tier.Tier.Size == nil || *tier.Tier.Size <= 0 {
		return PriceTier{}, false
	}
	converted := PriceTier{Kind: PriceTierContext, Threshold: *tier.Tier.Size}
	if tier.Input != nil {
		converted.Input = int64PtrCopy(MicroPerMillion(*tier.Input))
	}
	if tier.Output != nil {
		converted.Output = int64PtrCopy(MicroPerMillion(*tier.Output))
	}
	if tier.CacheRead != nil {
		converted.CacheRead = int64PtrCopy(MicroPerMillion(*tier.CacheRead))
	}
	if tier.CacheWrite != nil {
		converted.CacheWrite = int64PtrCopy(MicroPerMillion(*tier.CacheWrite))
	}
	if converted.Input == nil && converted.Output == nil &&
		converted.CacheRead == nil && converted.CacheWrite == nil {
		return PriceTier{}, false
	}
	return converted, true
}

// pricesEqual reports whether two versions carry the same rates.
func pricesEqual(a, b Price) bool {
	if a.Input != b.Input || a.Output != b.Output ||
		a.CacheRead != b.CacheRead || a.CacheWrite != b.CacheWrite {
		return false
	}
	if (a.Reasoning == nil) != (b.Reasoning == nil) {
		return false
	}
	if a.Reasoning != nil && *a.Reasoning != *b.Reasoning {
		return false
	}
	if len(a.Tiers) != len(b.Tiers) {
		return false
	}
	for i := range a.Tiers {
		if !tiersEqual(a.Tiers[i], b.Tiers[i]) {
			return false
		}
	}
	return true
}

func tiersEqual(a, b PriceTier) bool {
	return a.Kind == b.Kind && a.Threshold == b.Threshold &&
		strings.EqualFold(a.Service, b.Service) &&
		optionalEqual(a.Input, b.Input) && optionalEqual(a.Output, b.Output) &&
		optionalEqual(a.CacheRead, b.CacheRead) && optionalEqual(a.CacheWrite, b.CacheWrite)
}

func optionalEqual(a, b *int64) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	return a == nil || *a == *b
}

// applySyncedModels writes one provider batch in a single transaction: new
// keys start at valid_from 0, changed versions close the current one at
// nowMs and start the new one at nowMs, equal versions are untouched, and a
// current manual version is never replaced. Source is models.dev or
// embedded; both replace each other the same way.
func (s *Store) applySyncedModels(provider string, models []Price, source string, nowMs int64) (SyncResult, [][2]string, error) {
	var result SyncResult
	var touched [][2]string
	tx, err := s.writeDB.Begin()
	if err != nil {
		return result, nil, fmt.Errorf("usagedb: begin synced batch: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, model := range models {
		current, found, err := currentPriceVersionTx(tx, provider, model.ModelID)
		if err != nil {
			return result, nil, err
		}
		next := model
		next.ProviderKey = provider
		next.Source = source
		next.SyncedAtMs = nowMs
		next.Note = ""
		switch {
		case !found:
			next.ValidFromMs = 0
			if _, err := addPriceVersionTx(tx, next); err != nil {
				return result, nil, err
			}
			result.Added++
			touched = append(touched, [2]string{provider, model.ModelID})
		case pricesEqual(current, next):
			result.Unchanged++
		case current.Source == PriceSourceManual:
			result.SkippedOverrides++
		default:
			next.ValidFromMs = nowMs
			if _, err := addPriceVersionTx(tx, next); err != nil {
				return result, nil, err
			}
			result.Updated++
			touched = append(touched, [2]string{provider, model.ModelID})
		}
	}
	if err := tx.Commit(); err != nil {
		return result, nil, fmt.Errorf("usagedb: commit synced batch: %w", err)
	}
	return result, touched, nil
}

// ApplySyncedModels parses nothing: it stores already-parsed prices from
// the models.dev or embedded source, reloads the index, and fills late
// pricing for touched keys in the background.
func (s *Store) ApplySyncedModels(models []syncedModel, source string, nowMs int64) (SyncResult, error) {
	var total SyncResult
	if s == nil {
		return total, fmt.Errorf("usagedb: nil store")
	}
	byProvider := map[string][]Price{}
	for _, model := range models {
		byProvider[model.Provider] = append(byProvider[model.Provider], model.Price)
	}
	providers := make([]string, 0, len(byProvider))
	for provider := range byProvider {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	var touched [][2]string
	for _, provider := range providers {
		result, keys, err := s.applySyncedModels(provider, byProvider[provider], source, nowMs)
		if err != nil {
			return total, err
		}
		total.Added += result.Added
		total.Updated += result.Updated
		total.Unchanged += result.Unchanged
		total.SkippedOverrides += result.SkippedOverrides
		touched = append(touched, keys...)
	}
	if err := s.reloadPrices(); err != nil {
		return total, err
	}
	s.FillUnpricedAsync(touched)
	return total, nil
}

// fetchModelsDev downloads the price document with ETag caching. It returns
// notModified when the server answers 304.
func fetchModelsDev(ctx context.Context, url, etag, proxyURL string) (data []byte, newETag string, notModified bool, err error) {
	transport, _, errBuild := proxyutil.BuildHTTPTransport(strings.TrimSpace(proxyURL))
	if errBuild != nil {
		return nil, "", false, fmt.Errorf("usagedb: price proxy: %w", errBuild)
	}
	client := &http.Client{Timeout: priceSyncTimeout}
	if transport != nil {
		client.Transport = transport
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", false, fmt.Errorf("usagedb: price request: %w", err)
	}
	request.Header.Set("User-Agent", priceSyncUserAgent)
	request.Header.Set("Accept", "application/json")
	if etag != "" {
		request.Header.Set("If-None-Match", etag)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, "", false, fmt.Errorf("usagedb: price fetch: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode == http.StatusNotModified {
		return nil, response.Header.Get("ETag"), true, nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, "", false, fmt.Errorf("usagedb: price fetch status %d: %s",
			response.StatusCode, strings.TrimSpace(string(body)))
	}
	data, err = io.ReadAll(io.LimitReader(response.Body, priceSyncMaxBytes+1))
	if err != nil {
		return nil, "", false, fmt.Errorf("usagedb: price read: %w", err)
	}
	if int64(len(data)) > priceSyncMaxBytes {
		return nil, "", false, fmt.Errorf("usagedb: price response exceeds %d bytes", priceSyncMaxBytes)
	}
	return data, response.Header.Get("ETag"), false, nil
}

// SyncPricesNow runs one sync pass: fetch, parse, apply. On failure it keeps
// the old prices and records the error in settings. A 304 keeps versions
// and only refreshes the check time.
func (s *Store) SyncPricesNow(ctx context.Context, cfg PriceSyncConfig, nowMs int64) SyncResult {
	start := time.Now()
	result := SyncResult{}
	fail := func(err error) SyncResult {
		result.Error = err.Error()
		result.DurationMs = time.Since(start).Milliseconds()
		_ = s.SetPriceSetting(settingPriceSyncAttemptMs, formatMs(nowMs))
		_ = s.SetPriceSetting(settingPriceSyncError, result.Error)
		log.WithError(err).Error("usagedb: price sync failed, keeping old prices")
		return result
	}
	data, etag, notModified, err := fetchModelsDev(ctx, cfg.EffectiveURL(), s.PriceSetting(settingPriceSyncETag), cfg.ProxyURL)
	if err != nil {
		return fail(err)
	}
	_ = s.SetPriceSetting(settingPriceSyncAttemptMs, formatMs(nowMs))
	if notModified {
		if etag != "" {
			_ = s.SetPriceSetting(settingPriceSyncETag, etag)
		}
		_ = s.SetPriceSetting(settingPriceSyncLastMs, formatMs(nowMs))
		_ = s.SetPriceSetting(settingPriceSyncError, "")
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}
	models, err := ParseModelsDev(data)
	if err != nil {
		return fail(err)
	}
	applied, err := s.ApplySyncedModels(models, PriceSourceModelsDev, nowMs)
	if err != nil {
		return fail(err)
	}
	if etag != "" {
		_ = s.SetPriceSetting(settingPriceSyncETag, etag)
	}
	_ = s.SetPriceSetting(settingPriceSyncLastMs, formatMs(nowMs))
	_ = s.SetPriceSetting(settingPriceSyncError, "")
	result = applied
	result.DurationMs = time.Since(start).Milliseconds()
	return result
}

func formatMs(value int64) string {
	return strconv.FormatInt(value, 10)
}

// ActiveStore returns the open usage database, or nil when persistence is off.
func ActiveStore() *Store {
	mu.Lock()
	defer mu.Unlock()
	return collectorDB
}

// ActivePriceSyncConfig returns the currently applied price sync settings.
func ActivePriceSyncConfig() PriceSyncConfig {
	mu.Lock()
	defer mu.Unlock()
	return priceSyncConfig
}

// startPriceSyncLocked launches the price sync loop for the open database.
// Callers hold mu. The first sync runs after the startup delay, then on the
// configured cadence. A failed sync keeps the old prices and records the
// error in settings.
func startPriceSyncLocked() {
	if priceSyncStop != nil {
		close(priceSyncStop)
		priceSyncStop = nil
	}
	cfg := priceSyncConfig
	db := collectorDB
	if !cfg.Enabled || db == nil {
		return
	}
	stop := make(chan struct{})
	priceSyncStop = stop
	go runPriceSyncLoop(db, cfg, stop)
}

func runPriceSyncLoop(db *Store, cfg PriceSyncConfig, stop <-chan struct{}) {
	delay := time.NewTimer(priceSyncStartupDelay)
	defer delay.Stop()
	select {
	case <-stop:
		return
	case <-delay.C:
	}
	db.SyncPricesNow(context.Background(), cfg, unixMilliNow())
	ticker := time.NewTicker(cfg.EffectiveInterval())
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			db.SyncPricesNow(context.Background(), cfg, unixMilliNow())
		}
	}
}
