package usagedb

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// UsageEvent is one raw event for lists, detail, failures, and export.
// CostMicroUSD is nil for unpriced events. PriceID is nil without a price.
type UsageEvent struct {
	ID               int64
	RequestID        string
	TraceID          string
	TsMs             int64
	Provider         string
	Model            string
	Alias            string
	Executor         string
	AuthID           string
	AuthType         string
	AccountLabel     string
	APIKeyID         string
	Stream           bool
	StatusClass      string
	Failed           bool
	FailCode         string
	FailSummary      string
	LatencyMs        int64
	TTFTMs           int64
	InputTokens      int64
	OutputTokens     int64
	ReasoningTokens  int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	TotalTokens      int64
	ServiceTier      string
	ReasoningEffort  string
	CostMicroUSD     *int64
	CostSource       string
	SessionID        string
	AuthIndex        string
	AccessProvider   string
	PriceID          *int64
}

// ErrUsageEventNotFound answers 404 on the event detail route.
var ErrUsageEventNotFound = errors.New("usagedb: usage event not found")

const eventColumns = `id, request_id, trace_id, ts_ms, provider, model, alias, executor,
	auth_id, auth_type, account_label, api_key_id, stream, status_class, failed,
	fail_code, fail_summary, latency_ms, ttft_ms,
	input_tokens, output_tokens, reasoning_tokens,
	cache_read_tokens, cache_write_tokens, total_tokens,
	service_tier, reasoning_effort, cost_micro_usd, cost_source,
	session_id, auth_index, access_provider, price_id`

// scanEvent scans one eventColumns row. Nullable cost and price stay nil.
func scanEvent(rows *sql.Rows) (UsageEvent, error) {
	var event UsageEvent
	var stream, failed int64
	var cost sql.NullInt64
	var costSource sql.NullString
	var priceID sql.NullInt64
	if err := rows.Scan(&event.ID, &event.RequestID, &event.TraceID, &event.TsMs,
		&event.Provider, &event.Model, &event.Alias, &event.Executor,
		&event.AuthID, &event.AuthType, &event.AccountLabel, &event.APIKeyID,
		&stream, &event.StatusClass, &failed,
		&event.FailCode, &event.FailSummary, &event.LatencyMs, &event.TTFTMs,
		&event.InputTokens, &event.OutputTokens, &event.ReasoningTokens,
		&event.CacheReadTokens, &event.CacheWriteTokens, &event.TotalTokens,
		&event.ServiceTier, &event.ReasoningEffort, &cost, &costSource,
		&event.SessionID, &event.AuthIndex, &event.AccessProvider, &priceID); err != nil {
		return event, fmt.Errorf("usagedb: scan event: %w", err)
	}
	event.Stream = stream != 0
	event.Failed = failed != 0
	if cost.Valid {
		value := cost.Int64
		event.CostMicroUSD = &value
	}
	if costSource.Valid {
		event.CostSource = costSource.String
	}
	if priceID.Valid {
		value := priceID.Int64
		event.PriceID = &value
	}
	return event, nil
}

// EncodeUsageCursor builds the opaque page cursor from (ts_ms, id).
func EncodeUsageCursor(tsMs, id int64) string {
	raw := strconv.FormatInt(tsMs, 10) + ":" + strconv.FormatInt(id, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeUsageCursor parses a page cursor. A bad cursor is a client error.
func DecodeUsageCursor(raw string) (int64, int64, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return 0, 0, &QueryError{Msg: "invalid cursor"}
	}
	tsText, idText, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return 0, 0, &QueryError{Msg: "invalid cursor"}
	}
	tsMs, err := strconv.ParseInt(tsText, 10, 64)
	if err != nil || tsMs < 0 {
		return 0, 0, &QueryError{Msg: "invalid cursor"}
	}
	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id <= 0 {
		return 0, 0, &QueryError{Msg: "invalid cursor"}
	}
	return tsMs, id, nil
}

// EventsResult is one events page with its total estimate.
type EventsResult struct {
	Items         []UsageEvent
	NextCursor    string
	TotalEstimate int64
	// TotalExact is false when the total is an estimate; the route then
	// adds total_exact:false.
	TotalExact bool
}

// QueryEvents returns one page of raw events. limit defaults to 50 and caps
// at 200; order is desc (default) or asc. The cursor is opaque; a bad cursor
// is a client error. Paging on (ts_ms, id) stays stable with equal ts_ms.
func (s *Store) QueryEvents(f UsageFilter, limit int, cursor, order string) (EventsResult, error) {
	var result EventsResult
	if s == nil {
		return result, fmt.Errorf("usagedb: nil store")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		return result, &QueryError{Msg: "invalid limit"}
	}
	var desc bool
	switch order {
	case "", "desc":
		desc = true
	case "asc":
	default:
		return result, &QueryError{Msg: "invalid order"}
	}
	where, args := f.whereEvents()
	direction := "DESC"
	compare := "<"
	if !desc {
		direction = "ASC"
		compare = ">"
	}
	if cursor != "" {
		tsMs, id, err := DecodeUsageCursor(cursor)
		if err != nil {
			return result, err
		}
		where += " AND (ts_ms " + compare + " ? OR (ts_ms = ? AND id " + compare + " ?))"
		args = append(args, tsMs, tsMs, id)
	}
	query := `SELECT ` + eventColumns + ` FROM usage_events WHERE ` + where +
		` ORDER BY ts_ms ` + direction + `, id ` + direction + ` LIMIT ?`
	args = append(args, limit+1)
	rows, err := s.readDB.Query(query, args...)
	if err != nil {
		return result, fmt.Errorf("usagedb: query events: %w", err)
	}
	defer func() { _ = rows.Close() }()
	result.Items = []UsageEvent{}
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, event)
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("usagedb: events rows: %w", err)
	}
	if len(result.Items) > limit {
		last := result.Items[limit-1]
		result.NextCursor = EncodeUsageCursor(last.TsMs, last.ID)
		result.Items = result.Items[:limit]
	}
	total, exact, err := s.estimateTotal(f)
	if err != nil {
		return result, err
	}
	result.TotalEstimate = total
	result.TotalExact = exact
	return result, nil
}

// estimateTotal counts matching events exactly up to maxExactTotalRows.
// Above it, without text search the hourly rollup estimates the total
// (marked inexact: retention may have pruned raw rows); with text search the
// full count runs and stays exact.
func (s *Store) estimateTotal(f UsageFilter) (int64, bool, error) {
	where, args := f.whereEvents()
	cappedArgs := append(append([]any{}, args...), maxExactTotalRows+1)
	var capped int64
	cappedQuery := `SELECT COUNT(*) FROM (SELECT 1 FROM usage_events WHERE ` + where + ` LIMIT ?)`
	if err := s.readDB.QueryRow(cappedQuery, cappedArgs...).Scan(&capped); err != nil {
		return 0, false, fmt.Errorf("usagedb: count events: %w", err)
	}
	if capped <= maxExactTotalRows {
		return capped, true, nil
	}
	if f.HasQ() {
		var total int64
		if err := s.readDB.QueryRow(`SELECT COUNT(*) FROM usage_events WHERE `+where, args...).Scan(&total); err != nil {
			return 0, false, fmt.Errorf("usagedb: count events: %w", err)
		}
		return total, true, nil
	}
	hourlyWhere, hourlyArgs := f.whereHourly()
	var total int64
	if err := s.readDB.QueryRow(`SELECT COALESCE(SUM(requests),0) FROM usage_hourly WHERE `+
		hourlyWhere, hourlyArgs...).Scan(&total); err != nil {
		return 0, false, fmt.Errorf("usagedb: estimate total: %w", err)
	}
	return total, false, nil
}

// QueryEvent returns one raw event with its detail fields. Price data comes
// from the caller through PriceVersion; nil PriceID means no price object.
func (s *Store) QueryEvent(id int64) (UsageEvent, error) {
	if s == nil {
		return UsageEvent{}, fmt.Errorf("usagedb: nil store")
	}
	if id <= 0 {
		return UsageEvent{}, ErrUsageEventNotFound
	}
	rows, err := s.readDB.Query(`SELECT `+eventColumns+` FROM usage_events WHERE id = ?`, id)
	if err != nil {
		return UsageEvent{}, fmt.Errorf("usagedb: query event: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return UsageEvent{}, ErrUsageEventNotFound
	}
	event, err := scanEvent(rows)
	if err != nil {
		return UsageEvent{}, err
	}
	return event, rows.Err()
}

// FailureGroup counts one failure signature.
type FailureGroup struct {
	FailCode    string
	StatusClass string
	Provider    string
	AuthID      string
	Model       string
	Count       int64
	FirstMs     int64
	LastMs      int64
}

// FailuresResult holds failure groups plus recent failed events.
type FailuresResult struct {
	Groups []FailureGroup
	Recent []UsageEvent
}

// QueryFailures groups failed events by code, class, provider, credential,
// and model, and returns recent failed events with fail_summary. limit
// bounds groups (default 20, max 100); recent bounds the recent list
// (default 20, max 100).
func (s *Store) QueryFailures(f UsageFilter, limit, recent int) (FailuresResult, error) {
	var result FailuresResult
	if s == nil {
		return result, fmt.Errorf("usagedb: nil store")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		return result, &QueryError{Msg: "invalid limit"}
	}
	if recent <= 0 {
		recent = 20
	}
	if recent > 100 {
		return result, &QueryError{Msg: "invalid recent"}
	}
	where, args := f.whereEvents()
	groupQuery := `SELECT fail_code, status_class, provider, auth_id, model,
		COUNT(*), MIN(ts_ms), MAX(ts_ms) FROM usage_events WHERE ` + where + ` AND failed = 1
		GROUP BY fail_code, status_class, provider, auth_id, model
		ORDER BY COUNT(*) DESC, MAX(ts_ms) DESC LIMIT ?`
	groupRows, err := s.readDB.Query(groupQuery, append(append([]any{}, args...), limit)...)
	if err != nil {
		return result, fmt.Errorf("usagedb: query failures: %w", err)
	}
	defer func() { _ = groupRows.Close() }()
	result.Groups = []FailureGroup{}
	for groupRows.Next() {
		var group FailureGroup
		if err := groupRows.Scan(&group.FailCode, &group.StatusClass, &group.Provider,
			&group.AuthID, &group.Model, &group.Count, &group.FirstMs, &group.LastMs); err != nil {
			return result, fmt.Errorf("usagedb: scan failure group: %w", err)
		}
		result.Groups = append(result.Groups, group)
	}
	if err := groupRows.Err(); err != nil {
		return result, fmt.Errorf("usagedb: failure group rows: %w", err)
	}
	recentQuery := `SELECT ` + eventColumns + ` FROM usage_events WHERE ` + where + ` AND failed = 1
		ORDER BY ts_ms DESC, id DESC LIMIT ?`
	recentRows, err := s.readDB.Query(recentQuery, append(append([]any{}, args...), recent)...)
	if err != nil {
		return result, fmt.Errorf("usagedb: query recent failures: %w", err)
	}
	defer func() { _ = recentRows.Close() }()
	result.Recent = []UsageEvent{}
	for recentRows.Next() {
		event, err := scanEvent(recentRows)
		if err != nil {
			return result, err
		}
		result.Recent = append(result.Recent, event)
	}
	return result, recentRows.Err()
}

// ExportEvents calls emit for every matching event ordered by (ts_ms, id)
// ascending, in pages of exportPageSize on a read-only connection. It stops
// when the context ends and never buffers the whole result.
func (s *Store) ExportEvents(ctx context.Context, f UsageFilter, emit func(UsageEvent) error) error {
	if s == nil {
		return fmt.Errorf("usagedb: nil store")
	}
	where, args := f.whereEvents()
	var afterTs, afterID int64
	afterID = -1
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		pageWhere := where + ` AND (ts_ms > ? OR (ts_ms = ? AND id > ?))`
		pageArgs := append(append([]any{}, args...), afterTs, afterTs, afterID)
		rows, err := s.readDB.Query(`SELECT `+eventColumns+` FROM usage_events WHERE `+pageWhere+
			` ORDER BY ts_ms, id LIMIT ?`, append(pageArgs, exportPageSize)...)
		if err != nil {
			return fmt.Errorf("usagedb: export page: %w", err)
		}
		var count int
		var pageErr error
		for rows.Next() {
			event, err := scanEvent(rows)
			if err != nil {
				pageErr = err
				break
			}
			if err := emit(event); err != nil {
				pageErr = err
				break
			}
			afterTs, afterID = event.TsMs, event.ID
			count++
		}
		if rowsErr := rows.Err(); rowsErr != nil && pageErr == nil {
			pageErr = rowsErr
		}
		_ = rows.Close()
		if pageErr != nil {
			return pageErr
		}
		if count < exportPageSize {
			return nil
		}
	}
}

// AuthRef is the credential info used for labels.
type AuthRef struct {
	Label    string
	Provider string
}

// AuthLookup finds a credential by auth_id. Nil means no manager is open.
type AuthLookup func(authID string) *AuthRef

// CredentialLabel resolves the display label for one credential: the auth
// manager label when the credential exists, else the latest stored
// account_label, else the id.
func (s *Store) CredentialLabel(authID string, lookup AuthLookup) string {
	if lookup != nil {
		if ref := lookup(authID); ref != nil && strings.TrimSpace(ref.Label) != "" {
			return strings.TrimSpace(ref.Label)
		}
	}
	if s == nil {
		return authID
	}
	var label sql.NullString
	err := s.readDB.QueryRow(`SELECT account_label FROM usage_events
		WHERE auth_id = ? AND account_label != '' ORDER BY ts_ms DESC, id DESC LIMIT 1`,
		authID).Scan(&label)
	if err == nil && label.Valid && label.String != "" {
		return label.String
	}
	return authID
}

// CredentialLabels resolves labels for many ids with few queries.
func (s *Store) CredentialLabels(ids []string, lookup AuthLookup) map[string]string {
	out := make(map[string]string, len(ids))
	for _, id := range ids {
		if _, ok := out[id]; !ok {
			out[id] = s.CredentialLabel(id, lookup)
		}
	}
	return out
}

// StoredAPIKeyLabels returns the stored labels for the given ids.
func (s *Store) StoredAPIKeyLabels(ids []string) map[string]string {
	out := make(map[string]string)
	if s == nil || len(ids) == 0 {
		return out
	}
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	rows, err := s.readDB.Query(`SELECT api_key_id, label FROM api_key_labels WHERE api_key_id IN (`+
		strings.Join(placeholders, ",")+")", args...)
	if err != nil {
		return out
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id, label string
		if err := rows.Scan(&id, &label); err != nil {
			return out
		}
		out[id] = label
	}
	return out
}

// APIKeyDisplay returns the stored label or key-<first 6 of id>.
func APIKeyDisplay(id, storedLabel string) string {
	if strings.TrimSpace(storedLabel) != "" {
		return strings.TrimSpace(storedLabel)
	}
	short := strings.TrimSpace(id)
	if len(short) > 6 {
		short = short[:6]
	}
	return "key-" + short
}

// LatestAuthFields returns the latest stored provider and auth type per
// auth id, for breakdown meta. Unknown ids are absent.
func (s *Store) LatestAuthFields(ids []string) map[string][2]string {
	out := make(map[string][2]string)
	if s == nil || len(ids) == 0 {
		return out
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		var provider, authType string
		if err := s.readDB.QueryRow(`SELECT provider, auth_type FROM usage_events
			WHERE auth_id = ? ORDER BY ts_ms DESC, id DESC LIMIT 1`, id).Scan(&provider, &authType); err != nil {
			continue
		}
		out[id] = [2]string{provider, authType}
	}
	return out
}

// TopModelProviders returns the top provider per model by requests in the
// filter, for breakdown meta.
func (s *Store) TopModelProviders(f UsageFilter, models []string) map[string]string {
	out := make(map[string]string)
	if s == nil || len(models) == 0 {
		return out
	}
	where, args := f.whereHourly()
	placeholders := make([]string, 0, len(models))
	for _, model := range models {
		placeholders = append(placeholders, "?")
		args = append(args, model)
	}
	rows, err := s.readDB.Query(`SELECT model, provider, SUM(requests) FROM usage_hourly WHERE `+
		where+` AND model IN (`+strings.Join(placeholders, ",")+`) GROUP BY model, provider`, args...)
	if err != nil {
		return out
	}
	defer func() { _ = rows.Close() }()
	best := make(map[string]int64)
	for rows.Next() {
		var model, provider string
		var requests int64
		if err := rows.Scan(&model, &provider, &requests); err != nil {
			return out
		}
		if requests > best[model] {
			best[model] = requests
			out[model] = provider
		}
	}
	return out
}
