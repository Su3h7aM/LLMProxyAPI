package usagedb

import (
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Price sources stored in model_prices.source.
const (
	PriceSourceModelsDev = "models.dev"
	PriceSourceManual    = "manual"
	PriceSourceEmbedded  = "embedded"
)

// Cost sources stored in usage_events.cost_source.
const (
	CostSourceUpstream     = "upstream"
	CostSourcePriceBook    = "price-book"
	CostSourcePriceBookLate = "price-book-late"
	CostSourceNone         = "none"
)

// Price tier kinds stored in model_price_tiers.kind.
const (
	PriceTierContext = "context"
	PriceTierService = "service"
)

// microPerMillion converts USD per 1M tokens to integer micro-USD per 1M
// tokens. Floats are rounded half up; NaN and negative values become 0.
// The integer form keeps cost math exact until the final per-event rounding.
func microPerMillion(usdPerMillion float64) int64 {
	if math.IsNaN(usdPerMillion) || math.IsInf(usdPerMillion, 0) || usdPerMillion <= 0 {
		return 0
	}
	return int64(math.Round(usdPerMillion * 1e6))
}

// PriceTier is one context-threshold or service-tier override of a price
// version. A nil rate falls back to the version base rate.
type PriceTier struct {
	Kind       string
	Threshold  int64
	Service    string
	Input      *int64
	Output     *int64
	CacheRead  *int64
	CacheWrite *int64
}

// Price is one version of the price book for (provider_key, model_id).
// ValidToMs is nil for the current version. Reasoning is nil when reasoning
// tokens are priced at the output rate.
type Price struct {
	ID           int64
	ProviderKey  string
	ModelID      string
	ValidFromMs  int64
	ValidToMs    *int64
	Input        int64
	Output       int64
	CacheRead    int64
	CacheWrite   int64
	Reasoning    *int64
	Source       string
	SyncedAtMs   int64
	Note         string
	Tiers        []PriceTier
}

// ratesFor resolves the rate set for one event: a service-tier override wins
// over a context-threshold override, which wins over the base rates. The
// context tier is the highest threshold strictly below the normalized input
// (uncached + cache read + cache write), matching models.dev semantics where
// an exact-threshold event stays in the lower band.
func (p Price) ratesFor(serviceTier string, uncached, cacheRead, cacheWrite int64) (input, output, cacheReadRate, cacheWriteRate, reasoning int64) {
	input, output, cacheReadRate, cacheWriteRate = p.Input, p.Output, p.CacheRead, p.CacheWrite
	if p.Reasoning != nil {
		reasoning = *p.Reasoning
	} else {
		reasoning = p.Output
	}
	normalized := uncached + cacheRead + cacheWrite
	var selected *PriceTier
	for i := range p.Tiers {
		tier := &p.Tiers[i]
		if tier.Kind != PriceTierContext {
			continue
		}
		if normalized > tier.Threshold && (selected == nil || tier.Threshold > selected.Threshold) {
			selected = tier
		}
	}
	if trimmed := strings.TrimSpace(serviceTier); trimmed != "" {
		for i := range p.Tiers {
			tier := &p.Tiers[i]
			if tier.Kind == PriceTierService && strings.EqualFold(strings.TrimSpace(tier.Service), trimmed) {
				selected = tier
				break
			}
		}
	}
	if selected == nil {
		return input, output, cacheReadRate, cacheWriteRate, reasoning
	}
	if selected.Input != nil {
		input = *selected.Input
	}
	if selected.Output != nil {
		output = *selected.Output
	}
	if selected.CacheRead != nil {
		cacheReadRate = *selected.CacheRead
	}
	if selected.CacheWrite != nil {
		cacheWriteRate = *selected.CacheWrite
	}
	return input, output, cacheReadRate, cacheWriteRate, reasoning
}

// ComputeCost returns the event cost in micro-USD for the token buckets,
// rounded half up once. Buckets must already be non-overlapping (see
// Detail.PricedBuckets): uncached input, cache read, cache write,
// non-reasoning output, and reasoning.
func (p Price) ComputeCost(serviceTier string, uncached, cacheRead, cacheWrite, output, reasoning int64) int64 {
	inputRate, outputRate, readRate, writeRate, reasoningRate := p.ratesFor(serviceTier, uncached, cacheRead, cacheWrite)
	total := float64(uncached)*float64(inputRate) +
		float64(cacheRead)*float64(readRate) +
		float64(cacheWrite)*float64(writeRate) +
		float64(output)*float64(outputRate) +
		float64(reasoning)*float64(reasoningRate)
	return int64(math.Floor(total/1e6 + 0.5))
}

// forkProviderModelsDev maps the fork usage provider to the models.dev
// provider id. A missing entry means models.dev has no provider for the fork
// name; cross-provider exact-id matching still applies.
var forkProviderModelsDev = map[string]string{
	"claude":               "anthropic",
	"codex":                "openai",
	"openai":               "openai",
	"openai-compatibility": "openai",
	"gemini":               "google",
	"gemini-cli":           "google",
	"gemini-interactions":  "google",
	"vertex":               "google",
	"aistudio":             "google",
	"antigravity":          "google",
	"xai":                  "xai",
	"kimi":                 "moonshotai",
	"kimi-ai":              "moonshotai",
	"kimi.ai":              "moonshotai",
	"kimi.com":             "moonshotai",
	"meta":                 "meta",
	"opencode-go":          "opencode-go",
}

// modelsDevPreference orders the cross-provider fallback: an exact model id
// found outside its mapped provider is taken from the earliest provider here.
var modelsDevPreference = []string{
	"openai",
	"anthropic",
	"google",
	"xai",
	"meta",
	"moonshotai",
	"opencode-go",
}

// normalizeForkProvider lowercases the fork usage provider and folds the
// openai-compatible family onto "openai".
func normalizeForkProvider(provider string) string {
	trimmed := strings.ToLower(strings.TrimSpace(provider))
	if strings.HasPrefix(trimmed, "openai-compatible-") {
		return "openai-compatibility"
	}
	return trimmed
}

var dateSuffixPattern = regexp.MustCompile(`-(\d{8}|\d{4}-\d{2}-\d{2})$`)

// stripDateSuffix removes a trailing model date suffix (-YYYYMMDD or
// -YYYY-MM-DD). It returns the input unchanged when there is no suffix.
func stripDateSuffix(model string) string {
	return dateSuffixPattern.ReplaceAllString(strings.TrimSpace(model), "")
}

// priceIndex is the in-memory price book. It is rebuilt from the database
// whenever a version is added.
type priceIndex struct {
	// byProvider maps models.dev provider id to versions sorted by
	// (model_id, valid_from_ms).
	byProvider map[string][]*Price
}

func (s *Store) loadPriceIndex() (*priceIndex, error) {
	rows, err := s.readDB.Query(`SELECT id, provider_key, model_id, valid_from_ms, valid_to_ms,
		input, output, cache_read, cache_write, reasoning, source, synced_at_ms, note
		FROM model_prices ORDER BY provider_key, model_id, valid_from_ms`)
	if err != nil {
		return nil, fmt.Errorf("usagedb: load prices: %w", err)
	}
	defer func() { _ = rows.Close() }()
	index := &priceIndex{byProvider: make(map[string][]*Price)}
	for rows.Next() {
		var price Price
		var validTo sql.NullInt64
		var reasoning sql.NullInt64
		if err := rows.Scan(&price.ID, &price.ProviderKey, &price.ModelID,
			&price.ValidFromMs, &validTo, &price.Input, &price.Output,
			&price.CacheRead, &price.CacheWrite, &reasoning,
			&price.Source, &price.SyncedAtMs, &price.Note); err != nil {
			return nil, fmt.Errorf("usagedb: scan price: %w", err)
		}
		if validTo.Valid {
			value := validTo.Int64
			price.ValidToMs = &value
		}
		if reasoning.Valid {
			value := reasoning.Int64
			price.Reasoning = &value
		}
		index.byProvider[price.ProviderKey] = append(index.byProvider[price.ProviderKey], &price)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("usagedb: load prices: %w", err)
	}
	tierRows, err := s.readDB.Query(`SELECT price_id, kind, threshold_tokens, service,
		input, output, cache_read, cache_write FROM model_price_tiers ORDER BY price_id, threshold_tokens`)
	if err != nil {
		return nil, fmt.Errorf("usagedb: load price tiers: %w", err)
	}
	defer func() { _ = tierRows.Close() }()
	byID := make(map[int64]*Price)
	for _, versions := range index.byProvider {
		for _, price := range versions {
			byID[price.ID] = price
		}
	}
	for tierRows.Next() {
		var priceID int64
		var tier PriceTier
		var input, output, cacheRead, cacheWrite sql.NullInt64
		if err := tierRows.Scan(&priceID, &tier.Kind, &tier.Threshold, &tier.Service,
			&input, &output, &cacheRead, &cacheWrite); err != nil {
			return nil, fmt.Errorf("usagedb: scan price tier: %w", err)
		}
		if input.Valid {
			value := input.Int64
			tier.Input = &value
		}
		if output.Valid {
			value := output.Int64
			tier.Output = &value
		}
		if cacheRead.Valid {
			value := cacheRead.Int64
			tier.CacheRead = &value
		}
		if cacheWrite.Valid {
			value := cacheWrite.Int64
			tier.CacheWrite = &value
		}
		if price, ok := byID[priceID]; ok {
			price.Tiers = append(price.Tiers, tier)
		}
	}
	if err := tierRows.Err(); err != nil {
		return nil, fmt.Errorf("usagedb: load price tiers: %w", err)
	}
	return index, nil
}

// lookupProvider returns the versions of one models.dev provider, or nil.
func (index *priceIndex) lookupProvider(providerKey string) []*Price {
	if index == nil {
		return nil
	}
	return index.byProvider[providerKey]
}

// matchAt returns the version of (providerKey, modelID) valid at tsMs. When
// several versions overlap, a manual version wins, then the latest start.
// It returns nil when no version covers tsMs.
func matchAt(versions []*Price, modelID string, tsMs int64) *Price {
	stripped := stripDateSuffix(modelID)
	var manual, other *Price
	for _, price := range versions {
		if price.ModelID != modelID && stripDateSuffix(price.ModelID) != stripped {
			continue
		}
		if price.ValidFromMs > tsMs {
			continue
		}
		if price.ValidToMs != nil && tsMs >= *price.ValidToMs {
			continue
		}
		if price.Source == PriceSourceManual {
			if manual == nil || price.ValidFromMs > manual.ValidFromMs {
				manual = price
			}
			continue
		}
		if other == nil || price.ValidFromMs > other.ValidFromMs {
			other = price
		}
	}
	if manual != nil {
		return manual
	}
	return other
}

// Lookup resolves the price version for one event: the routed model first,
// then the alias. Rules in order: (a) the mapped models.dev provider of the
// fork provider, (b) exact model id, (c) date-suffix-insensitive match both
// ways, (d) exact model id in any other provider by preference list. It
// returns nil when nothing matches.
func (index *priceIndex) Lookup(provider, model, alias string, tsMs int64) *Price {
	if index == nil {
		return nil
	}
	mapped, ok := forkProviderModelsDev[normalizeForkProvider(provider)]
	models := []string{strings.TrimSpace(model)}
	if trimmed := strings.TrimSpace(alias); trimmed != "" && trimmed != models[0] {
		models = append(models, trimmed)
	}
	if ok {
		for _, id := range models {
			if id == "" {
				continue
			}
			if price := matchAt(index.lookupProvider(mapped), id, tsMs); price != nil {
				return price
			}
		}
	}
	seen := map[string]bool{}
	if ok {
		seen[mapped] = true
	}
	providers := make([]string, 0, len(index.byProvider))
	for _, preferred := range modelsDevPreference {
		if seen[preferred] {
			continue
		}
		if _, exists := index.byProvider[preferred]; exists {
			providers = append(providers, preferred)
			seen[preferred] = true
		}
	}
	rest := make([]string, 0, len(index.byProvider))
	for key := range index.byProvider {
		if !seen[key] {
			rest = append(rest, key)
		}
	}
	sort.Strings(rest)
	providers = append(providers, rest...)
	for _, id := range models {
		if id == "" {
			continue
		}
		for _, providerKey := range providers {
			if price := matchAt(index.lookupProvider(providerKey), id, tsMs); price != nil {
				return price
			}
		}
	}
	return nil
}

// earliestVersion returns the first version of (providerKey, modelID),
// used by late pricing for events older than the first version.
func (index *priceIndex) earliestVersion(providerKey, modelID string) *Price {
	stripped := stripDateSuffix(modelID)
	var first *Price
	for _, price := range index.lookupProvider(providerKey) {
		if price.ModelID != modelID && stripDateSuffix(price.ModelID) != stripped {
			continue
		}
		if first == nil || price.ValidFromMs < first.ValidFromMs {
			first = price
		}
	}
	return first
}

// priceIndexHolder keeps the reloadable in-memory price book of one store.
type priceIndexHolder struct {
	mu    sync.RWMutex
	index *priceIndex
}

func (s *Store) reloadPrices() error {
	index, err := s.loadPriceIndex()
	if err != nil {
		return err
	}
	s.prices.mu.Lock()
	s.prices.index = index
	s.prices.mu.Unlock()
	return nil
}

// lookupPrice resolves the price version for one event from the in-memory
// index. It returns nil when nothing matches.
func (s *Store) lookupPrice(provider, model, alias string, tsMs int64) *Price {
	if s == nil {
		return nil
	}
	s.prices.mu.RLock()
	index := s.prices.index
	s.prices.mu.RUnlock()
	if index == nil {
		return nil
	}
	return index.Lookup(provider, model, alias, tsMs)
}

// priceVersionCount returns the number of price versions in the database.
func (s *Store) priceVersionCount() (int64, error) {
	var count int64
	if err := s.readDB.QueryRow(`SELECT COUNT(*) FROM model_prices`).Scan(&count); err != nil {
		return 0, fmt.Errorf("usagedb: count prices: %w", err)
	}
	return count, nil
}

// pricesBySource counts current versions by source.
func (s *Store) pricesBySource() (map[string]int64, error) {
	counts := map[string]int64{}
	rows, err := s.readDB.Query(`SELECT source, COUNT(*) FROM model_prices
		WHERE valid_to_ms IS NULL GROUP BY source`)
	if err != nil {
		return nil, fmt.Errorf("usagedb: count prices by source: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var source string
		var count int64
		if err := rows.Scan(&source, &count); err != nil {
			return nil, fmt.Errorf("usagedb: scan price source count: %w", err)
		}
		counts[source] = count
	}
	return counts, rows.Err()
}

// addPriceVersionTx closes the current version of (providerKey, modelID) at
// validFromMs and inserts the new version in the same transaction. A first
// version starts at validFromMs as given by the caller (0 for the first sync
// of a model, sync time for later ones, explicit date for manual prices). It
// never UPDATEs an old row except to set valid_to_ms once, and never DELETEs.
// Closing an already-closed version, or a version that is not current, fails.
func addPriceVersionTx(tx *sql.Tx, price Price) (int64, error) {
	if strings.TrimSpace(price.ProviderKey) == "" || strings.TrimSpace(price.ModelID) == "" {
		return 0, fmt.Errorf("usagedb: price needs provider and model")
	}
	var currentID int64
	var currentTo sql.NullInt64
	err := tx.QueryRow(`SELECT id, valid_to_ms FROM model_prices
		WHERE provider_key = ? AND model_id = ?
		ORDER BY valid_from_ms DESC LIMIT 1`,
		price.ProviderKey, price.ModelID).Scan(&currentID, &currentTo)
	switch {
	case err == sql.ErrNoRows:
		// First version: nothing to close.
	case err != nil:
		return 0, fmt.Errorf("usagedb: read current price: %w", err)
	default:
		if currentTo.Valid {
			return 0, fmt.Errorf("usagedb: latest price version %d for %s/%s is already closed",
				currentID, price.ProviderKey, price.ModelID)
		}
		var currentFrom int64
		if err := tx.QueryRow(`SELECT valid_from_ms FROM model_prices WHERE id = ?`,
			currentID).Scan(&currentFrom); err != nil {
			return 0, fmt.Errorf("usagedb: read current price start: %w", err)
		}
		if price.ValidFromMs < currentFrom {
			return 0, fmt.Errorf("usagedb: new price version starts before the current one")
		}
		if price.ValidFromMs == currentFrom {
			return 0, fmt.Errorf("usagedb: price version for %s/%s at %d already exists",
				price.ProviderKey, price.ModelID, price.ValidFromMs)
		}
		if _, err := tx.Exec(`UPDATE model_prices SET valid_to_ms = ? WHERE id = ? AND valid_to_ms IS NULL`,
			price.ValidFromMs, currentID); err != nil {
			return 0, fmt.Errorf("usagedb: close current price: %w", err)
		}
	}
	var reasoning sql.NullInt64
	if price.Reasoning != nil {
		reasoning = sql.NullInt64{Int64: *price.Reasoning, Valid: true}
	}
	result, err := tx.Exec(`INSERT INTO model_prices (provider_key, model_id, valid_from_ms, valid_to_ms,
		input, output, cache_read, cache_write, reasoning, source, synced_at_ms, note)
		VALUES (?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?)`,
		price.ProviderKey, price.ModelID, price.ValidFromMs,
		price.Input, price.Output, price.CacheRead, price.CacheWrite, reasoning,
		price.Source, price.SyncedAtMs, price.Note)
	if err != nil {
		return 0, fmt.Errorf("usagedb: insert price: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("usagedb: price id: %w", err)
	}
	for _, tier := range price.Tiers {
		var input, output, cacheRead, cacheWrite sql.NullInt64
		if tier.Input != nil {
			input = sql.NullInt64{Int64: *tier.Input, Valid: true}
		}
		if tier.Output != nil {
			output = sql.NullInt64{Int64: *tier.Output, Valid: true}
		}
		if tier.CacheRead != nil {
			cacheRead = sql.NullInt64{Int64: *tier.CacheRead, Valid: true}
		}
		if tier.CacheWrite != nil {
			cacheWrite = sql.NullInt64{Int64: *tier.CacheWrite, Valid: true}
		}
		if _, err := tx.Exec(`INSERT INTO model_price_tiers (price_id, kind, threshold_tokens, service,
			input, output, cache_read, cache_write) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			id, tier.Kind, tier.Threshold, tier.Service,
			input, output, cacheRead, cacheWrite); err != nil {
			return 0, fmt.Errorf("usagedb: insert price tier: %w", err)
		}
	}
	return id, nil
}

// AddPriceVersion appends one price version for (providerKey, modelID) and
// reloads the in-memory index. Old rows are never changed except for the
// single valid_to_ms close of the previous current version.
func (s *Store) AddPriceVersion(price Price) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("usagedb: nil store")
	}
	tx, err := s.writeDB.Begin()
	if err != nil {
		return 0, fmt.Errorf("usagedb: begin price version: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	id, err := addPriceVersionTx(tx, price)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("usagedb: commit price version: %w", err)
	}
	if err := s.reloadPrices(); err != nil {
		return 0, err
	}
	return id, nil
}

// priceVersionRow reads one price version with its tiers by id.
func priceVersionRow(tx *sql.Tx, readDB *sql.DB, id int64) (Price, error) {
	var price Price
	var validTo, reasoning sql.NullInt64
	query := `SELECT id, provider_key, model_id, valid_from_ms, valid_to_ms,
		input, output, cache_read, cache_write, reasoning, source, synced_at_ms, note
		FROM model_prices WHERE id = ?`
	var err error
	if tx != nil {
		err = tx.QueryRow(query, id).Scan(&price.ID, &price.ProviderKey, &price.ModelID,
			&price.ValidFromMs, &validTo, &price.Input, &price.Output,
			&price.CacheRead, &price.CacheWrite, &reasoning,
			&price.Source, &price.SyncedAtMs, &price.Note)
	} else {
		err = readDB.QueryRow(query, id).Scan(&price.ID, &price.ProviderKey, &price.ModelID,
			&price.ValidFromMs, &validTo, &price.Input, &price.Output,
			&price.CacheRead, &price.CacheWrite, &reasoning,
			&price.Source, &price.SyncedAtMs, &price.Note)
	}
	if err != nil {
		return price, fmt.Errorf("usagedb: read price %d: %w", id, err)
	}
	if validTo.Valid {
		value := validTo.Int64
		price.ValidToMs = &value
	}
	if reasoning.Valid {
		value := reasoning.Int64
		price.Reasoning = &value
	}
	var rows *sql.Rows
	tierQuery := `SELECT kind, threshold_tokens, service, input, output, cache_read, cache_write
		FROM model_price_tiers WHERE price_id = ? ORDER BY threshold_tokens`
	if tx != nil {
		rows, err = tx.Query(tierQuery, id)
	} else {
		rows, err = readDB.Query(tierQuery, id)
	}
	if err != nil {
		return price, fmt.Errorf("usagedb: read price %d tiers: %w", id, err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var tier PriceTier
		var input, output, cacheRead, cacheWrite sql.NullInt64
		if err := rows.Scan(&tier.Kind, &tier.Threshold, &tier.Service,
			&input, &output, &cacheRead, &cacheWrite); err != nil {
			return price, fmt.Errorf("usagedb: scan price %d tier: %w", id, err)
		}
		if input.Valid {
			value := input.Int64
			tier.Input = &value
		}
		if output.Valid {
			value := output.Int64
			tier.Output = &value
		}
		if cacheRead.Valid {
			value := cacheRead.Int64
			tier.CacheRead = &value
		}
		if cacheWrite.Valid {
			value := cacheWrite.Int64
			tier.CacheWrite = &value
		}
		price.Tiers = append(price.Tiers, tier)
	}
	return price, rows.Err()
}

// currentPriceVersionTx returns the current (open) version of a model, if any.
func currentPriceVersionTx(tx *sql.Tx, providerKey, modelID string) (Price, bool, error) {
	var id int64
	err := tx.QueryRow(`SELECT id FROM model_prices
		WHERE provider_key = ? AND model_id = ? AND valid_to_ms IS NULL`,
		providerKey, modelID).Scan(&id)
	if err == sql.ErrNoRows {
		return Price{}, false, nil
	}
	if err != nil {
		return Price{}, false, fmt.Errorf("usagedb: read current price: %w", err)
	}
	price, err := priceVersionRow(tx, nil, id)
	if err != nil {
		return Price{}, false, err
	}
	return price, true, nil
}
