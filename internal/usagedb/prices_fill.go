package usagedb

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// fillBatchSize bounds one fill transaction. fillBatchPause yields the writer
// between batches.
const (
	fillBatchSize  = 5000
	fillBatchPause = 50 * time.Millisecond
)

// FillScope bounds a late-pricing or correction pass over usage_events.
type FillScope struct {
	FromMs   *int64
	ToMs     *int64
	Provider string
	Model    string
}

// FillResult counts one fill pass.
type FillResult struct {
	Priced         int64
	SkippedNoPrice int64
}

// matchEventToKey reports whether one event belongs to the price key
// (providerKey, modelID) under the step-2 lookup rules: the fork provider
// maps to the key provider, and the event model or alias matches the key
// model date-suffix-insensitively.
func matchEventToKey(provider, model, alias, providerKey, modelID string) bool {
	if forkProviderModelsDev[normalizeForkProvider(provider)] != providerKey {
		return false
	}
	stripped := stripDateSuffix(modelID)
	if stripped == "" {
		return false
	}
	return stripDateSuffix(model) == stripped || stripDateSuffix(alias) == stripped
}

// matchEventScope reports whether one event is inside an on-demand scope.
// Provider compares against the stored fork provider; model compares against
// the event model or alias, date-suffix-insensitively.
func matchEventScope(provider, model, alias string, scope FillScope) bool {
	if trimmed := strings.TrimSpace(scope.Provider); trimmed != "" &&
		!strings.EqualFold(strings.TrimSpace(provider), trimmed) {
		return false
	}
	if trimmed := strings.TrimSpace(scope.Model); trimmed != "" {
		stripped := stripDateSuffix(trimmed)
		if stripDateSuffix(model) != stripped && stripDateSuffix(alias) != stripped {
			return false
		}
	}
	return true
}

// nearestPrice resolves the price for one event: the version valid at the
// event time, or the first version of the mapped key when the event is older
// than it (nearest). It returns nil when the model has no version at all.
func (s *Store) nearestPrice(provider, model, alias string, tsMs int64) *Price {
	if price := s.lookupPrice(provider, model, alias, tsMs); price != nil {
		return price
	}
	s.prices.mu.RLock()
	index := s.prices.index
	s.prices.mu.RUnlock()
	if index == nil {
		return nil
	}
	mapped, ok := forkProviderModelsDev[normalizeForkProvider(provider)]
	if !ok {
		return nil
	}
	if price := index.earliestVersion(mapped, strings.TrimSpace(model)); price != nil {
		return price
	}
	if trimmed := strings.TrimSpace(alias); trimmed != "" {
		return index.earliestVersion(mapped, trimmed)
	}
	return nil
}

// fillBatch reads one page of NULL-cost events and prices them in a single
// transaction, fixing the hourly rollups in the same transaction. Events
// that already have a cost never match the page query. It returns the priced
// and skipped counts plus the last seen id for paging.
func (s *Store) fillBatch(afterID int64, scope FillScope, keyOnly *[2]string, source string) (FillResult, int64, error) {
	var result FillResult
	lastID := afterID
	query := `SELECT id, ts_ms, provider, model, alias, executor, service_tier,
		input_tokens, output_tokens, reasoning_tokens,
		cache_read_tokens, cache_write_tokens, total_tokens,
		auth_id, api_key_id, status_class, stream
		FROM usage_events
		WHERE id > ? AND cost_micro_usd IS NULL
		ORDER BY id LIMIT ?`
	rows, err := s.readDB.Query(query, afterID, fillBatchSize)
	if err != nil {
		return result, lastID, fmt.Errorf("usagedb: page unpriced events: %w", err)
	}
	type candidate struct {
		id                                                     int64
		tsMs                                                   int64
		provider, model, alias, executor, serviceTier          string
		input, output, reasoning, cacheRead, cacheWrite, total int64
		hourMs                                                 int64
		authID, apiKeyID, statusClass                          string
		stream                                                 int64
		price                                                  *Price
		cost                                                   int64
	}
	var priced []candidate
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var event candidate
		var stream sql.NullInt64
		if err := rows.Scan(&event.id, &event.tsMs, &event.provider, &event.model,
			&event.alias, &event.executor, &event.serviceTier, &event.input, &event.output,
			&event.reasoning, &event.cacheRead, &event.cacheWrite, &event.total,
			&event.authID, &event.apiKeyID, &event.statusClass, &stream); err != nil {
			return result, lastID, fmt.Errorf("usagedb: scan unpriced event: %w", err)
		}
		event.hourMs = event.tsMs / millisPerHour * millisPerHour
		event.stream = stream.Int64
		lastID = event.id
		if keyOnly != nil && !matchEventToKey(event.provider, event.model, event.alias, keyOnly[0], keyOnly[1]) {
			continue
		}
		if keyOnly == nil && !matchEventScope(event.provider, event.model, event.alias, scope) {
			continue
		}
		if scope.FromMs != nil && event.tsMs < *scope.FromMs {
			continue
		}
		if scope.ToMs != nil && event.tsMs > *scope.ToMs {
			continue
		}
		price := s.nearestPrice(event.provider, event.model, event.alias, event.tsMs)
		if price == nil {
			result.SkippedNoPrice++
			continue
		}
		detail := rowCostDetail(UsageRow{
			Provider:         event.provider,
			Executor:         event.executor,
			InputTokens:      event.input,
			OutputTokens:     event.output,
			ReasoningTokens:  event.reasoning,
			CacheReadTokens:  event.cacheRead,
			CacheWriteTokens: event.cacheWrite,
			TotalTokens:      event.total,
		})
		uncached, read, write, output, reasoning := detail.PricedBuckets(event.provider, event.executor)
		event.price = price
		event.cost = price.ComputeCost(event.serviceTier, uncached, read, write, output, reasoning)
		priced = append(priced, event)
	}
	if err := rows.Err(); err != nil {
		return result, lastID, fmt.Errorf("usagedb: page unpriced events: %w", err)
	}
	if len(priced) == 0 {
		return result, lastID, nil
	}
	tx, err := s.writeDB.Begin()
	if err != nil {
		return result, lastID, fmt.Errorf("usagedb: begin fill batch: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, event := range priced {
		updated, err := tx.Exec(`UPDATE usage_events
			SET cost_micro_usd = ?, cost_source = ?, price_id = ?
			WHERE id = ? AND cost_micro_usd IS NULL`,
			event.cost, source, event.price.ID, event.id)
		if err != nil {
			return result, lastID, fmt.Errorf("usagedb: fill event cost: %w", err)
		}
		affected, err := updated.RowsAffected()
		if err != nil {
			return result, lastID, fmt.Errorf("usagedb: fill rows affected: %w", err)
		}
		if affected == 0 {
			continue
		}
		if _, err := tx.Exec(`UPDATE usage_hourly
			SET cost_micro_usd = cost_micro_usd + ?,
				late_cost_micro_usd = late_cost_micro_usd + ?,
				unpriced_requests = CASE WHEN unpriced_requests > 0 THEN unpriced_requests - 1 ELSE 0 END
			WHERE hour_ms = ? AND provider = ? AND model = ? AND alias = ?
				AND auth_id = ? AND api_key_id = ? AND status_class = ? AND stream = ?`,
			event.cost, event.cost, event.hourMs, event.provider, event.model, event.alias,
			event.authID, event.apiKeyID, event.statusClass, event.stream); err != nil {
			return result, lastID, fmt.Errorf("usagedb: fill hourly: %w", err)
		}
		if _, err := tx.Exec(`UPDATE usage_daily
			SET cost_micro_usd = cost_micro_usd + ?,
				late_cost_micro_usd = late_cost_micro_usd + ?,
				unpriced_requests = CASE WHEN unpriced_requests > 0 THEN unpriced_requests - 1 ELSE 0 END
			WHERE day_ms = ? AND provider = ? AND model = ? AND alias = ?
				AND auth_id = ? AND api_key_id = ? AND status_class = ? AND stream = ?`,
			event.cost, event.cost, event.hourMs/millisPerDay*millisPerDay,
			event.provider, event.model, event.alias,
			event.authID, event.apiKeyID, event.statusClass, event.stream); err != nil {
			return result, lastID, fmt.Errorf("usagedb: fill daily: %w", err)
		}
		result.Priced++
	}
	if err := tx.Commit(); err != nil {
		return result, lastID, fmt.Errorf("usagedb: commit fill batch: %w", err)
	}
	return result, lastID, nil
}

// fillUnpriced prices NULL-cost events in batches. When keyOnly is set, only
// events of that price key are filled (late pricing after a version is
// added); otherwise the scope filters the pass (on-demand price-unpriced).
// It never touches an event that already has a cost.
func (s *Store) fillUnpriced(scope FillScope, keyOnly *[2]string, source string) (FillResult, error) {
	var total FillResult
	if s == nil {
		return total, fmt.Errorf("usagedb: nil store")
	}
	var afterID int64
	for {
		result, lastID, err := s.fillBatch(afterID, scope, keyOnly, source)
		if err != nil {
			return total, err
		}
		total.Priced += result.Priced
		total.SkippedNoPrice += result.SkippedNoPrice
		if lastID == afterID {
			return total, nil
		}
		afterID = lastID
		time.Sleep(fillBatchPause)
	}
}

// FillUnpricedForKeys fills NULL-cost events of the given price keys with
// the version valid at each event time (or the nearest), marked as late
// estimates. It is idempotent: re-running changes nothing.
func (s *Store) FillUnpricedForKeys(keys [][2]string) (FillResult, error) {
	var total FillResult
	for _, key := range keys {
		scoped := key
		result, err := s.fillUnpriced(FillScope{}, &scoped, CostSourcePriceBookLate)
		if err != nil {
			return total, err
		}
		total.Priced += result.Priced
		total.SkippedNoPrice += result.SkippedNoPrice
	}
	return total, nil
}

// FillUnpricedAsync fills the keys in the background, after the request that
// added the version has returned.
func (s *Store) FillUnpricedAsync(keys [][2]string) {
	if s == nil || len(keys) == 0 {
		return
	}
	go func() {
		result, err := s.FillUnpricedForKeys(keys)
		if err != nil {
			log.WithError(err).Error("usagedb: late pricing failed")
			return
		}
		log.WithFields(log.Fields{
			"keys":   len(keys),
			"priced": result.Priced,
		}).Info("usagedb: late pricing filled unpriced events")
	}()
}

// PriceUnpriced fills NULL-cost events inside the scope on demand. It never
// touches an event that already has a cost.
func (s *Store) PriceUnpriced(scope FillScope) (FillResult, error) {
	return s.fillUnpriced(scope, nil, CostSourcePriceBookLate)
}

// CorrectCosts recomputes cost for every event in the scope (priced or not)
// with the version valid at each event time, fixes the hourly rollups in the
// same transaction, and writes a cost_corrections audit row. Events without
// any resolvable price are skipped. The range must not exceed 366 days.
func (s *Store) CorrectCosts(scope FillScope, reason, actor string) (correctionID, updated, oldTotal, newTotal int64, err error) {
	if s == nil {
		return 0, 0, 0, 0, fmt.Errorf("usagedb: nil store")
	}
	if strings.TrimSpace(reason) == "" {
		return 0, 0, 0, 0, fmt.Errorf("usagedb: correction needs a reason")
	}
	if scope.FromMs == nil || scope.ToMs == nil {
		return 0, 0, 0, 0, fmt.Errorf("usagedb: correction needs from_ms and to_ms")
	}
	if *scope.ToMs < *scope.FromMs {
		return 0, 0, 0, 0, fmt.Errorf("usagedb: correction range is empty")
	}
	if *scope.ToMs-*scope.FromMs > int64(366*24*time.Hour/time.Millisecond) {
		return 0, 0, 0, 0, fmt.Errorf("usagedb: correction range exceeds 366 days")
	}
	tx, err := s.writeDB.Begin()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("usagedb: begin correction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.Query(`SELECT id, ts_ms, provider, model, alias, executor, service_tier,
		input_tokens, output_tokens, reasoning_tokens,
		cache_read_tokens, cache_write_tokens, total_tokens, cost_micro_usd, cost_source,
		auth_id, api_key_id, status_class, stream
		FROM usage_events WHERE ts_ms >= ? AND ts_ms <= ? ORDER BY id`,
		*scope.FromMs, *scope.ToMs)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("usagedb: read correction events: %w", err)
	}
	defer func() { _ = rows.Close() }()
	type correction struct {
		id                                                     int64
		tsMs                                                   int64
		provider, model, alias, executor, serviceTier          string
		input, output, reasoning, cacheRead, cacheWrite, total int64
		oldCost                                                sql.NullInt64
		oldSource                                              sql.NullString
		hourMs                                                 int64
		authID, apiKeyID, statusClass                          string
		stream                                                 int64
		newCost                                                int64
		priceID                                                int64
	}
	var events []correction
	for rows.Next() {
		var event correction
		var stream sql.NullInt64
		if err := rows.Scan(&event.id, &event.tsMs, &event.provider, &event.model,
			&event.alias, &event.executor, &event.serviceTier, &event.input, &event.output,
			&event.reasoning, &event.cacheRead, &event.cacheWrite, &event.total,
			&event.oldCost, &event.oldSource, &event.authID, &event.apiKeyID,
			&event.statusClass, &stream); err != nil {
			return 0, 0, 0, 0, fmt.Errorf("usagedb: scan correction event: %w", err)
		}
		event.hourMs = event.tsMs / millisPerHour * millisPerHour
		event.stream = stream.Int64
		if !matchEventScope(event.provider, event.model, event.alias, scope) {
			continue
		}
		price := s.nearestPrice(event.provider, event.model, event.alias, event.tsMs)
		if price == nil {
			continue
		}
		detail := rowCostDetail(UsageRow{
			Provider:         event.provider,
			Executor:         event.executor,
			InputTokens:      event.input,
			OutputTokens:     event.output,
			ReasoningTokens:  event.reasoning,
			CacheReadTokens:  event.cacheRead,
			CacheWriteTokens: event.cacheWrite,
			TotalTokens:      event.total,
		})
		uncached, read, write, output, reasoning := detail.PricedBuckets(event.provider, event.executor)
		event.newCost = price.ComputeCost(event.serviceTier, uncached, read, write, output, reasoning)
		event.priceID = price.ID
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("usagedb: read correction events: %w", err)
	}
	for _, event := range events {
		var old int64
		if event.oldCost.Valid {
			old = event.oldCost.Int64
		}
		oldTotal += old
		newTotal += event.newCost
		if _, err := tx.Exec(`UPDATE usage_events
			SET cost_micro_usd = ?, cost_source = ?, price_id = ?
			WHERE id = ?`,
			event.newCost, CostSourcePriceBook, event.priceID, event.id); err != nil {
			return 0, 0, 0, 0, fmt.Errorf("usagedb: correct event cost: %w", err)
		}
		delta := event.newCost - old
		unpricedDelta := 0
		if !event.oldCost.Valid {
			unpricedDelta = -1
		}
		// The correction stores cost_source=price-book, so a late estimate
		// stops being late: remove its old cost from the late part.
		lateDelta := int64(0)
		if event.oldSource.Valid && event.oldSource.String == CostSourcePriceBookLate {
			lateDelta = -old
		}
		if _, err := tx.Exec(`UPDATE usage_hourly
			SET cost_micro_usd = cost_micro_usd + ?,
				late_cost_micro_usd = CASE WHEN late_cost_micro_usd + ? < 0 THEN 0
					ELSE late_cost_micro_usd + ? END,
				unpriced_requests = CASE WHEN unpriced_requests + ? < 0 THEN 0
					ELSE unpriced_requests + ? END
			WHERE hour_ms = ? AND provider = ? AND model = ? AND alias = ?
				AND auth_id = ? AND api_key_id = ? AND status_class = ? AND stream = ?`,
			delta, lateDelta, lateDelta, unpricedDelta, unpricedDelta,
			event.hourMs, event.provider, event.model, event.alias,
			event.authID, event.apiKeyID, event.statusClass, event.stream); err != nil {
			return 0, 0, 0, 0, fmt.Errorf("usagedb: correct hourly: %w", err)
		}
		if _, err := tx.Exec(`UPDATE usage_daily
			SET cost_micro_usd = cost_micro_usd + ?,
				late_cost_micro_usd = CASE WHEN late_cost_micro_usd + ? < 0 THEN 0
					ELSE late_cost_micro_usd + ? END,
				unpriced_requests = CASE WHEN unpriced_requests + ? < 0 THEN 0
					ELSE unpriced_requests + ? END
			WHERE day_ms = ? AND provider = ? AND model = ? AND alias = ?
				AND auth_id = ? AND api_key_id = ? AND status_class = ? AND stream = ?`,
			delta, lateDelta, lateDelta, unpricedDelta, unpricedDelta,
			event.hourMs/millisPerDay*millisPerDay,
			event.provider, event.model, event.alias,
			event.authID, event.apiKeyID, event.statusClass, event.stream); err != nil {
			return 0, 0, 0, 0, fmt.Errorf("usagedb: correct daily: %w", err)
		}
		updated++
	}
	result, err := tx.Exec(`INSERT INTO cost_corrections (created_at_ms, from_ms, to_ms,
		provider_key, model_id, old_cost_micro_usd, new_cost_micro_usd, reason, actor)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		unixMilliNow(), *scope.FromMs, *scope.ToMs,
		strings.TrimSpace(scope.Provider), strings.TrimSpace(scope.Model),
		oldTotal, newTotal, reason, actor)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("usagedb: write correction audit: %w", err)
	}
	correctionID, err = result.LastInsertId()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("usagedb: correction id: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("usagedb: commit correction: %w", err)
	}
	return correctionID, updated, oldTotal, newTotal, nil
}
