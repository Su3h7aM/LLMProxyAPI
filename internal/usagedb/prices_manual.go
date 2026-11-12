package usagedb

import (
	"database/sql"
	"fmt"
	"strings"
)

// AddManualPrice closes the current version of the key at
// price.ValidFromMs and appends a manual version. The source is forced to
// manual. A past effective date is allowed; it does not change events that
// already have a cost (use PriceUnpriced or CorrectCosts for those).
func (s *Store) AddManualPrice(price Price, nowMs int64) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("usagedb: nil store")
	}
	price.ProviderKey = strings.TrimSpace(price.ProviderKey)
	price.ModelID = strings.TrimSpace(price.ModelID)
	if price.ProviderKey == "" || price.ModelID == "" {
		return 0, fmt.Errorf("usagedb: manual price needs provider and model")
	}
	if price.ValidFromMs <= 0 {
		price.ValidFromMs = nowMs
	}
	price.Source = PriceSourceManual
	price.SyncedAtMs = nowMs
	tx, err := s.writeDB.Begin()
	if err != nil {
		return 0, fmt.Errorf("usagedb: begin manual price: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	id, err := addPriceVersionTx(tx, price)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("usagedb: commit manual price: %w", err)
	}
	if err := s.reloadPrices(); err != nil {
		return 0, err
	}
	return id, nil
}

// RemoveManualPrice closes the current manual version of the key at nowMs.
// It never deletes rows. When a closed synced version exists for the key,
// its rates are restored as a new current version so the synced price
// applies again from now; otherwise the key is left without a current
// version. It reports whether a version was restored.
func (s *Store) RemoveManualPrice(providerKey, modelID string, nowMs int64) (restored bool, err error) {
	if s == nil {
		return false, fmt.Errorf("usagedb: nil store")
	}
	providerKey = strings.TrimSpace(providerKey)
	modelID = strings.TrimSpace(modelID)
	if providerKey == "" || modelID == "" {
		return false, fmt.Errorf("usagedb: manual price removal needs provider and model")
	}
	tx, err := s.writeDB.Begin()
	if err != nil {
		return false, fmt.Errorf("usagedb: begin manual price removal: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	current, found, err := currentPriceVersionTx(tx, providerKey, modelID)
	if err != nil {
		return false, err
	}
	if !found {
		return false, fmt.Errorf("usagedb: no current price for %s/%s", providerKey, modelID)
	}
	if current.Source != PriceSourceManual {
		return false, fmt.Errorf("usagedb: current price for %s/%s is not manual", providerKey, modelID)
	}
	if _, err := tx.Exec(`UPDATE model_prices SET valid_to_ms = ? WHERE id = ? AND valid_to_ms IS NULL`,
		nowMs, current.ID); err != nil {
		return false, fmt.Errorf("usagedb: close manual price: %w", err)
	}
	var fallbackID int64
	err = tx.QueryRow(`SELECT id FROM model_prices
		WHERE provider_key = ? AND model_id = ? AND valid_to_ms IS NOT NULL
			AND source != ? ORDER BY valid_from_ms DESC LIMIT 1`,
		providerKey, modelID, PriceSourceManual).Scan(&fallbackID)
	switch {
	case err == sql.ErrNoRows:
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("usagedb: commit manual price removal: %w", err)
		}
		if err := s.reloadPrices(); err != nil {
			return false, err
		}
		return false, nil
	case err != nil:
		return false, fmt.Errorf("usagedb: read fallback price: %w", err)
	}
	fallback, err := priceVersionRow(tx, nil, fallbackID)
	if err != nil {
		return false, err
	}
	fallback.ValidFromMs = nowMs
	fallback.SyncedAtMs = nowMs
	fallback.Note = "restored after manual removal"
	result, err := tx.Exec(`INSERT INTO model_prices (provider_key, model_id, valid_from_ms, valid_to_ms,
		input, output, cache_read, cache_write, reasoning, source, synced_at_ms, note)
		VALUES (?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?)`,
		fallback.ProviderKey, fallback.ModelID, fallback.ValidFromMs,
		fallback.Input, fallback.Output, fallback.CacheRead, fallback.CacheWrite,
		nullableInt64(fallback.Reasoning), fallback.Source, fallback.SyncedAtMs, fallback.Note)
	if err != nil {
		return false, fmt.Errorf("usagedb: restore synced price: %w", err)
	}
	restoredID, err := result.LastInsertId()
	if err != nil {
		return false, fmt.Errorf("usagedb: restored price id: %w", err)
	}
	for _, tier := range fallback.Tiers {
		if _, err := tx.Exec(`INSERT INTO model_price_tiers (price_id, kind, threshold_tokens, service,
			input, output, cache_read, cache_write) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			restoredID, tier.Kind, tier.Threshold, tier.Service,
			nullableInt64(tier.Input), nullableInt64(tier.Output),
			nullableInt64(tier.CacheRead), nullableInt64(tier.CacheWrite)); err != nil {
			return false, fmt.Errorf("usagedb: restore price tier: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("usagedb: commit manual price removal: %w", err)
	}
	if err := s.reloadPrices(); err != nil {
		return false, err
	}
	return true, nil
}

func nullableInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

// CurrentPrices returns the current (open) versions, newest first, with
// optional exact filters and a substring query over provider and model.
func (s *Store) CurrentPrices(provider, model, source, query string, limit, offset int) ([]Price, int64, error) {
	conditions := []string{"valid_to_ms IS NULL"}
	args := []any{}
	if trimmed := strings.TrimSpace(provider); trimmed != "" {
		conditions = append(conditions, "provider_key = ?")
		args = append(args, trimmed)
	}
	if trimmed := strings.TrimSpace(model); trimmed != "" {
		conditions = append(conditions, "model_id = ?")
		args = append(args, trimmed)
	}
	if trimmed := strings.TrimSpace(source); trimmed != "" {
		conditions = append(conditions, "source = ?")
		args = append(args, trimmed)
	}
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		conditions = append(conditions, "(provider_key LIKE ? OR model_id LIKE ?)")
		like := "%" + trimmed + "%"
		args = append(args, like, like)
	}
	where := "WHERE " + strings.Join(conditions, " AND ")
	var total int64
	if err := s.readDB.QueryRow(`SELECT COUNT(*) FROM model_prices `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("usagedb: count current prices: %w", err)
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.readDB.Query(`SELECT id FROM model_prices `+where+
		` ORDER BY provider_key, model_id LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("usagedb: list current prices: %w", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, 0, fmt.Errorf("usagedb: scan price id: %w", err)
		}
		ids = append(ids, id)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("usagedb: list current prices: %w", err)
	}
	prices := make([]Price, 0, len(ids))
	for _, id := range ids {
		price, err := priceVersionRow(nil, s.readDB, id)
		if err != nil {
			return nil, 0, err
		}
		prices = append(prices, price)
	}
	return prices, total, nil
}

// PriceHistory returns every version of one key, oldest first.
func (s *Store) PriceHistory(providerKey, modelID string) ([]Price, error) {
	rows, err := s.readDB.Query(`SELECT id FROM model_prices
		WHERE provider_key = ? AND model_id = ? ORDER BY valid_from_ms`,
		strings.TrimSpace(providerKey), strings.TrimSpace(modelID))
	if err != nil {
		return nil, fmt.Errorf("usagedb: price history: %w", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("usagedb: scan history id: %w", err)
		}
		ids = append(ids, id)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("usagedb: price history: %w", err)
	}
	prices := make([]Price, 0, len(ids))
	for _, id := range ids {
		price, err := priceVersionRow(nil, s.readDB, id)
		if err != nil {
			return nil, err
		}
		prices = append(prices, price)
	}
	return prices, nil
}

// UnmatchedModel counts events without cost grouped by provider and model.
type UnmatchedModel struct {
	Provider string
	Model    string
	Events   int64
}

// UnmatchedModels lists models with NULL-cost events, most events first.
func (s *Store) UnmatchedModels(limit int) ([]UnmatchedModel, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.readDB.Query(`SELECT provider, model, COUNT(*) FROM usage_events
		WHERE cost_micro_usd IS NULL GROUP BY provider, model
		ORDER BY COUNT(*) DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("usagedb: unmatched models: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []UnmatchedModel
	for rows.Next() {
		var item UnmatchedModel
		if err := rows.Scan(&item.Provider, &item.Model, &item.Events); err != nil {
			return nil, fmt.Errorf("usagedb: scan unmatched model: %w", err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// PriceSetting reads one key from the settings table.
func (s *Store) PriceSetting(key string) string {
	var value string
	if err := s.readDB.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value); err != nil {
		return ""
	}
	return value
}

// SetPriceSetting writes one key to the settings table.
func (s *Store) SetPriceSetting(key, value string) error {
	if _, err := s.writeDB.Exec(`INSERT INTO settings(key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value); err != nil {
		return fmt.Errorf("usagedb: write setting %s: %w", key, err)
	}
	return nil
}
