package usagedb

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
)

//go:embed snapshot.json.gz
var priceSnapshotGzip []byte

// snapshotDoc is the trimmed models.dev extract committed with the binary.
// Only the fields the price book stores are kept: ids, names, and cost with
// tiers. Cost shapes match models.dev, so the shared parser applies.
type snapshotDoc struct {
	Providers map[string]modelsDevProvider `json:"providers"`
}

// ParseEmbeddedSnapshot parses the committed snapshot without touching the
// network. It is used at first start and covered by tests.
func ParseEmbeddedSnapshot() ([]syncedModel, error) {
	reader, err := gzip.NewReader(bytes.NewReader(priceSnapshotGzip))
	if err != nil {
		return nil, fmt.Errorf("usagedb: open price snapshot: %w", err)
	}
	raw, err := io.ReadAll(io.LimitReader(reader, priceSyncMaxBytes))
	_ = reader.Close()
	if err != nil {
		return nil, fmt.Errorf("usagedb: read price snapshot: %w", err)
	}
	var doc snapshotDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("usagedb: parse price snapshot: %w", err)
	}
	reshaped, err := json.Marshal(doc.Providers)
	if err != nil {
		return nil, fmt.Errorf("usagedb: reshape price snapshot: %w", err)
	}
	return ParseModelsDev(reshaped)
}

// LoadEmbeddedSnapshot stores the snapshot as source=embedded versions when
// the price table is empty. With existing versions it is a no-op.
func (s *Store) LoadEmbeddedSnapshot(nowMs int64) (SyncResult, error) {
	var result SyncResult
	if s == nil {
		return result, fmt.Errorf("usagedb: nil store")
	}
	count, err := s.priceVersionCount()
	if err != nil {
		return result, err
	}
	if count > 0 {
		return result, nil
	}
	models, err := ParseEmbeddedSnapshot()
	if err != nil {
		return result, err
	}
	return s.ApplySyncedModels(models, PriceSourceEmbedded, nowMs)
}
