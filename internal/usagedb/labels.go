package usagedb

import (
	"fmt"
	"strings"
)

// KeyLabel is one API key id label. Seen reports whether the id appears in
// usage_events. Stored labels keep Seen false until an event carries the id.
type KeyLabel struct {
	APIKeyID string `json:"api_key_id"`
	Label    string `json:"label"`
	Seen     bool   `json:"seen"`
}

// KeyLabelOf returns the stored label for one key id, or empty when none.
func (s *Store) KeyLabelOf(id string) string {
	if s == nil {
		return ""
	}
	var label string
	if err := s.readDB.QueryRow(`SELECT label FROM api_key_labels WHERE api_key_id = ?`,
		strings.TrimSpace(id)).Scan(&label); err != nil {
		return ""
	}
	return label
}

// SetKeyLabel stores the label for one key id. An empty label clears it.
// It returns true when a label row remains.
func (s *Store) SetKeyLabel(id, label string, nowMs int64) (bool, error) {
	if s == nil {
		return false, fmt.Errorf("usagedb: nil store")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return false, fmt.Errorf("usagedb: key label needs an api_key_id")
	}
	label = strings.TrimSpace(label)
	if label == "" {
		if _, err := s.writeDB.Exec(`DELETE FROM api_key_labels WHERE api_key_id = ?`, id); err != nil {
			return false, fmt.Errorf("usagedb: clear key label: %w", err)
		}
		return false, nil
	}
	if _, err := s.writeDB.Exec(`INSERT INTO api_key_labels(api_key_id, label, updated_at_ms)
		VALUES (?, ?, ?)
		ON CONFLICT(api_key_id) DO UPDATE SET label = excluded.label, updated_at_ms = excluded.updated_at_ms`,
		id, label, nowMs); err != nil {
		return false, fmt.Errorf("usagedb: set key label: %w", err)
	}
	return true, nil
}

// KeyLabels returns all stored labels plus ids seen in events without a
// label (label empty, seen true). Stored ids carry seen true only when an
// event carries the id.
func (s *Store) KeyLabels() ([]KeyLabel, error) {
	if s == nil {
		return nil, fmt.Errorf("usagedb: nil store")
	}
	rows, err := s.readDB.Query(`SELECT api_key_id, label FROM api_key_labels ORDER BY api_key_id`)
	if err != nil {
		return nil, fmt.Errorf("usagedb: list key labels: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []KeyLabel
	for rows.Next() {
		var item KeyLabel
		if err := rows.Scan(&item.APIKeyID, &item.Label); err != nil {
			return nil, fmt.Errorf("usagedb: scan key label: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("usagedb: list key labels: %w", err)
	}
	known := make(map[string]bool, len(out))
	for i := range out {
		known[out[i].APIKeyID] = true
	}
	seen, err := s.seenAPIKeyIDs()
	if err != nil {
		return nil, err
	}
	for _, id := range seen {
		if known[id] {
			continue
		}
		out = append(out, KeyLabel{APIKeyID: id, Seen: true})
	}
	for i := range out {
		if seenSet(seen, out[i].APIKeyID) {
			out[i].Seen = true
		}
	}
	return out, nil
}

func seenSet(seen []string, id string) bool {
	for _, other := range seen {
		if other == id {
			return true
		}
	}
	return false
}

// seenAPIKeyIDs returns distinct non-empty key ids in usage_events.
func (s *Store) seenAPIKeyIDs() ([]string, error) {
	rows, err := s.readDB.Query(`SELECT DISTINCT api_key_id FROM usage_events WHERE api_key_id != '' ORDER BY api_key_id`)
	if err != nil {
		return nil, fmt.Errorf("usagedb: list seen key ids: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("usagedb: scan seen key id: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// Vacuum rebuilds the database file. It blocks writers while it runs.
func (s *Store) Vacuum() error {
	if s == nil {
		return fmt.Errorf("usagedb: nil store")
	}
	if _, err := s.writeDB.Exec(`VACUUM`); err != nil {
		return fmt.Errorf("usagedb: vacuum: %w", err)
	}
	return nil
}

// CheckpointResult is the outcome of one WAL checkpoint.
type CheckpointResult struct {
	Busy         int64
	LogFrames    int64
	Checkpointed int64
}

// Checkpoint runs wal_checkpoint(TRUNCATE) on the writer connection.
func (s *Store) Checkpoint() (CheckpointResult, error) {
	var result CheckpointResult
	if s == nil {
		return result, fmt.Errorf("usagedb: nil store")
	}
	if err := s.writeDB.QueryRow(`PRAGMA wal_checkpoint(TRUNCATE)`).Scan(
		&result.Busy, &result.LogFrames, &result.Checkpointed); err != nil {
		return result, fmt.Errorf("usagedb: checkpoint: %w", err)
	}
	return result, nil
}

// PruneOlderThan deletes raw events older than cutoffMs in batches. Rollups
// stay. It returns the total deleted row count.
func (s *Store) PruneOlderThan(cutoffMs int64) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("usagedb: nil store")
	}
	var total int64
	for {
		deleted, err := s.PruneBefore(cutoffMs, pruneBatchSize)
		if err != nil {
			return total, err
		}
		total += deleted
		if deleted == 0 {
			return total, nil
		}
		if deleted < pruneBatchSize {
			return total, nil
		}
	}
}

// PruneNow runs retention on the active store now. Retention 0 or below
// keeps all rows and deletes nothing. It returns the deleted row count.
func PruneNow(nowMs int64) (int64, error) {
	mu.Lock()
	db := collectorDB
	mu.Unlock()
	if db == nil {
		return 0, fmt.Errorf("usagedb: usage database is disabled")
	}
	days := collectorDays.Load()
	if days <= 0 {
		return 0, nil
	}
	const dayMs = int64(24 * 60 * 60 * 1000)
	deleted, err := db.PruneOlderThan(nowMs - days*dayMs)
	if err != nil {
		return deleted, err
	}
	lastPruneMs.Store(nowMs)
	return deleted, nil
}
