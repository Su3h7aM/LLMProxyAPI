// Package usagedb persists usage records in an embedded SQLite database.
//
// The database lives in the auth directory (`usage.db` by default) and uses
// a pure-Go driver, so the container binary stays static (CGO_ENABLED=0).
// Writes go through one writer goroutine in batches; reads use a small pool.
package usagedb

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const (
	// SchemaVersion is the current migration version. Open refuses a database
	// file with a newer version.
	SchemaVersion = 4

	// readPoolSize bounds the read connection pool. Writes use one connection.
	readPoolSize = 4

	// pruneBatchSize deletes raw events in small batches with a pause between
	// them, so retention never holds the write lock for long.
	pruneBatchSize = 5000
)

// Store is an opened usage database: one writer connection plus a small
// read pool on the same file.
type Store struct {
	path    string
	writeDB *sql.DB
	readDB  *sql.DB
	prices  priceIndexHolder
}

// Open creates parent directories when needed, applies the pragmas from the
// design (WAL, NORMAL, 5000 ms busy timeout, foreign keys), runs pending
// migrations, and refuses a file with a newer schema version.
func Open(path string) (*Store, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, errors.New("usagedb: empty database path")
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return nil, fmt.Errorf("usagedb: resolve path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, fmt.Errorf("usagedb: create parent directory: %w", err)
	}
	writeDB, err := sql.Open("sqlite", dataSourceName(abs, true))
	if err != nil {
		return nil, fmt.Errorf("usagedb: open writer: %w", err)
	}
	writeDB.SetMaxOpenConns(1)
	writeDB.SetMaxIdleConns(1)
	readDB, err := sql.Open("sqlite", dataSourceName(abs, false))
	if err != nil {
		_ = writeDB.Close()
		return nil, fmt.Errorf("usagedb: open reader: %w", err)
	}
	readDB.SetMaxOpenConns(readPoolSize)
	readDB.SetMaxIdleConns(readPoolSize)
	store := &Store{path: abs, writeDB: writeDB, readDB: readDB}
	var mode string
	if err := writeDB.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("usagedb: read journal mode: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(mode), "wal") {
		_ = store.Close()
		return nil, fmt.Errorf("usagedb: journal mode is %q, want WAL", mode)
	}
	if err := migrate(writeDB); err != nil {
		_ = store.Close()
		return nil, err
	}
	if err := store.reloadPrices(); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

// Close releases both pools.
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	writeErr := s.writeDB.Close()
	readErr := s.readDB.Close()
	if writeErr != nil {
		return writeErr
	}
	return readErr
}

// Path returns the absolute database file path.
func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func dataSourceName(path string, writer bool) string {
	uriPath := filepath.ToSlash(path)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	dsn := &url.URL{Scheme: "file", Path: uriPath}
	query := dsn.Query()
	if writer {
		query.Add("_txlock", "immediate")
	}
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "synchronous(NORMAL)")
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "foreign_keys(ON)")
	if !writer {
		// Read tuning (writes keep the durable defaults above):
		// - cache_size -32768: 32 MiB page cache per read connection
		//   (4 conns, 128 MiB worst case) for the hourly full scans.
		// - mmap_size 268435456: 256 MiB mmap cap; the OS pages it, so it
		//   never commits RAM, and cold scans skip read() copies.
		// - temp_store MEMORY: grouped aggregates sort temp b-trees in
		//   RAM instead of temp files.
		query.Add("_pragma", "cache_size(-32768)")
		query.Add("_pragma", "mmap_size(268435456)")
		query.Add("_pragma", "temp_store(MEMORY)")
	}
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

// migration is one ordered schema step.
type migration struct {
	version int
	stmts   []string
}

var migrations = []migration{
	{
		version: 1,
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS schema_migrations (
				version INTEGER PRIMARY KEY,
				applied_at_ms INTEGER NOT NULL
			)`,
			`CREATE TABLE IF NOT EXISTS usage_events (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				request_id TEXT NOT NULL UNIQUE,
				trace_id TEXT NOT NULL DEFAULT '',
				ts_ms INTEGER NOT NULL,
				provider TEXT NOT NULL DEFAULT '',
				model TEXT NOT NULL DEFAULT '',
				alias TEXT NOT NULL DEFAULT '',
				executor TEXT NOT NULL DEFAULT '',
				auth_id TEXT NOT NULL DEFAULT '',
				auth_index TEXT NOT NULL DEFAULT '',
				auth_type TEXT NOT NULL DEFAULT '',
				account_label TEXT NOT NULL DEFAULT '',
				api_key_id TEXT NOT NULL DEFAULT '',
				access_provider TEXT NOT NULL DEFAULT '',
				session_id TEXT NOT NULL DEFAULT '',
				stream INTEGER NOT NULL DEFAULT 0,
				status_class TEXT NOT NULL DEFAULT '',
				failed INTEGER NOT NULL DEFAULT 0,
				fail_code TEXT NOT NULL DEFAULT '',
				fail_summary TEXT NOT NULL DEFAULT '',
				latency_ms INTEGER NOT NULL DEFAULT 0,
				ttft_ms INTEGER NOT NULL DEFAULT 0,
				input_tokens INTEGER NOT NULL DEFAULT 0,
				output_tokens INTEGER NOT NULL DEFAULT 0,
				reasoning_tokens INTEGER NOT NULL DEFAULT 0,
				cache_read_tokens INTEGER NOT NULL DEFAULT 0,
				cache_write_tokens INTEGER NOT NULL DEFAULT 0,
				total_tokens INTEGER NOT NULL DEFAULT 0,
				service_tier TEXT NOT NULL DEFAULT '',
				reasoning_effort TEXT NOT NULL DEFAULT '',
				cost_micro_usd INTEGER NULL,
				cost_source TEXT NULL
			)`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_events_request_id ON usage_events(request_id)`,
			`CREATE INDEX IF NOT EXISTS idx_usage_events_ts ON usage_events(ts_ms)`,
			`CREATE INDEX IF NOT EXISTS idx_usage_events_auth_ts ON usage_events(auth_id, ts_ms)`,
			`CREATE INDEX IF NOT EXISTS idx_usage_events_model_ts ON usage_events(model, ts_ms)`,
			`CREATE INDEX IF NOT EXISTS idx_usage_events_apikey_ts ON usage_events(api_key_id, ts_ms)`,
			`CREATE TABLE IF NOT EXISTS usage_hourly (
				hour_ms INTEGER NOT NULL,
				provider TEXT NOT NULL DEFAULT '',
				model TEXT NOT NULL DEFAULT '',
				alias TEXT NOT NULL DEFAULT '',
				auth_id TEXT NOT NULL DEFAULT '',
				api_key_id TEXT NOT NULL DEFAULT '',
				status_class TEXT NOT NULL DEFAULT '',
				stream INTEGER NOT NULL DEFAULT 0,
				requests INTEGER NOT NULL DEFAULT 0,
				failures INTEGER NOT NULL DEFAULT 0,
				input_tokens INTEGER NOT NULL DEFAULT 0,
				output_tokens INTEGER NOT NULL DEFAULT 0,
				reasoning_tokens INTEGER NOT NULL DEFAULT 0,
				cache_read_tokens INTEGER NOT NULL DEFAULT 0,
				cache_write_tokens INTEGER NOT NULL DEFAULT 0,
				cost_micro_usd INTEGER NOT NULL DEFAULT 0,
				unpriced_requests INTEGER NOT NULL DEFAULT 0,
				latency_sum_ms INTEGER NOT NULL DEFAULT 0,
				latency_max_ms INTEGER NOT NULL DEFAULT 0,
				ttft_sum_ms INTEGER NOT NULL DEFAULT 0,
				ttft_count INTEGER NOT NULL DEFAULT 0,
				PRIMARY KEY (hour_ms, provider, model, alias, auth_id, api_key_id, status_class, stream)
			)`,
			`CREATE TABLE IF NOT EXISTS settings (
				key TEXT PRIMARY KEY,
				value TEXT NOT NULL DEFAULT ''
			)`,
		},
	},
	{
		version: 2,
		stmts: []string{
			// Price rows are append-only: code never UPDATEs a price row
			// except to set valid_to_ms once when a newer version starts,
			// and never DELETEs one. The UNIQUE index also serves lookups
			// by (provider_key, model_id, valid_from_ms).
			`CREATE TABLE IF NOT EXISTS model_prices (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				provider_key TEXT NOT NULL,
				model_id TEXT NOT NULL,
				valid_from_ms INTEGER NOT NULL,
				valid_to_ms INTEGER NULL,
				input INTEGER NOT NULL DEFAULT 0,
				output INTEGER NOT NULL DEFAULT 0,
				cache_read INTEGER NOT NULL DEFAULT 0,
				cache_write INTEGER NOT NULL DEFAULT 0,
				reasoning INTEGER NULL,
				source TEXT NOT NULL DEFAULT '',
				synced_at_ms INTEGER NOT NULL DEFAULT 0,
				note TEXT NOT NULL DEFAULT ''
			)`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_model_prices_version
				ON model_prices(provider_key, model_id, valid_from_ms)`,
			`CREATE TABLE IF NOT EXISTS model_price_tiers (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				price_id INTEGER NOT NULL REFERENCES model_prices(id),
				kind TEXT NOT NULL DEFAULT '',
				threshold_tokens INTEGER NOT NULL DEFAULT 0,
				service TEXT NOT NULL DEFAULT '',
				input INTEGER NULL,
				output INTEGER NULL,
				cache_read INTEGER NULL,
				cache_write INTEGER NULL
			)`,
			`CREATE INDEX IF NOT EXISTS idx_model_price_tiers_price
				ON model_price_tiers(price_id)`,
			`CREATE TABLE IF NOT EXISTS cost_corrections (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				created_at_ms INTEGER NOT NULL,
				from_ms INTEGER NOT NULL,
				to_ms INTEGER NOT NULL,
				provider_key TEXT NOT NULL DEFAULT '',
				model_id TEXT NOT NULL DEFAULT '',
				old_cost_micro_usd INTEGER NOT NULL DEFAULT 0,
				new_cost_micro_usd INTEGER NOT NULL DEFAULT 0,
				reason TEXT NOT NULL DEFAULT '',
				actor TEXT NOT NULL DEFAULT ''
			)`,
			`ALTER TABLE usage_events ADD COLUMN price_id INTEGER NULL`,
		},
	},
	{
		version: 3,
		stmts: []string{
			// late_cost_micro_usd is the part of cost_micro_usd that comes
			// from late pricing (cost_source='price-book-late'). It is
			// included in cost_micro_usd.
			`ALTER TABLE usage_hourly ADD COLUMN late_cost_micro_usd INTEGER NOT NULL DEFAULT 0`,
			// Backfill from events priced late before this migration.
			// The hour key derives from ts_ms with integer division, the
			// same truncation the writer uses.
			`UPDATE usage_hourly SET late_cost_micro_usd = COALESCE((
				SELECT SUM(e.cost_micro_usd) FROM usage_events e
				WHERE e.cost_source = 'price-book-late'
				AND e.ts_ms / 3600000 * 3600000 = usage_hourly.hour_ms
				AND e.provider = usage_hourly.provider
				AND e.model = usage_hourly.model
				AND e.alias = usage_hourly.alias
				AND e.auth_id = usage_hourly.auth_id
				AND e.api_key_id = usage_hourly.api_key_id
				AND e.status_class = usage_hourly.status_class
				AND e.stream = usage_hourly.stream
			), 0)`,
			`CREATE TABLE IF NOT EXISTS api_key_labels (
				api_key_id TEXT PRIMARY KEY,
				label TEXT NOT NULL,
				updated_at_ms INTEGER NOT NULL
			)`,
			// Each index below serves a query of the usage query routes:
			// range plus provider filter (breakdown, timeseries, events),
			// range plus failed flag (events, failures), range plus status
			// class (hourly status filter maps here; events detail also
			// filters it). Verified with EXPLAIN QUERY PLAN (see Log).
			`CREATE INDEX IF NOT EXISTS idx_usage_events_provider_ts ON usage_events(provider, ts_ms)`,
			`CREATE INDEX IF NOT EXISTS idx_usage_events_failed_ts ON usage_events(failed, ts_ms)`,
			`CREATE INDEX IF NOT EXISTS idx_usage_events_status_ts ON usage_events(status_class, ts_ms)`,
		},
	},
	{
		version: 4,
		stmts: []string{
			// A failed record is never 2xx (see statusClass): a failed
			// 2xx or missing code is error. Fix rows stored before the
			// rule. The error class is new, so every failed error row
			// below is a moved one.
			`UPDATE usage_events SET status_class = 'error' WHERE failed = 1 AND status_class = '2xx'`,
			// Drop the hourly rows of affected groups: 2xx rows that
			// mixed moved failures in, plus any error rows.
			`DELETE FROM usage_hourly AS h WHERE h.status_class IN ('2xx', 'error') AND EXISTS (
					SELECT 1 FROM usage_events e
					WHERE e.failed = 1 AND e.status_class = 'error'
					AND e.ts_ms / 3600000 * 3600000 = h.hour_ms
					AND e.provider = h.provider
					AND e.model = h.model
					AND e.alias = h.alias
					AND e.auth_id = h.auth_id
					AND e.api_key_id = h.api_key_id
					AND e.stream = h.stream
				)`,
			// Rebuild the dropped rows from the fixed events. Cost,
			// late-cost, unpriced, latency, and TTFT math matches the
			// writer (see InsertBatch).
			`INSERT INTO usage_hourly (
					hour_ms, provider, model, alias, auth_id, api_key_id, status_class, stream,
					requests, failures, input_tokens, output_tokens, reasoning_tokens,
					cache_read_tokens, cache_write_tokens, cost_micro_usd, unpriced_requests,
					latency_sum_ms, latency_max_ms, ttft_sum_ms, ttft_count, late_cost_micro_usd
				) SELECT e.ts_ms / 3600000 * 3600000, e.provider, e.model, e.alias,
					e.auth_id, e.api_key_id, e.status_class, e.stream,
					COUNT(*), COALESCE(SUM(e.failed), 0),
					COALESCE(SUM(e.input_tokens), 0), COALESCE(SUM(e.output_tokens), 0),
					COALESCE(SUM(e.reasoning_tokens), 0),
					COALESCE(SUM(e.cache_read_tokens), 0), COALESCE(SUM(e.cache_write_tokens), 0),
					COALESCE(SUM(e.cost_micro_usd), 0),
					COALESCE(SUM(CASE WHEN e.cost_micro_usd IS NULL THEN 1 ELSE 0 END), 0),
					COALESCE(SUM(e.latency_ms), 0), COALESCE(MAX(e.latency_ms), 0),
					COALESCE(SUM(CASE WHEN e.ttft_ms > 0 THEN e.ttft_ms ELSE 0 END), 0),
					COALESCE(SUM(CASE WHEN e.ttft_ms > 0 THEN 1 ELSE 0 END), 0),
					COALESCE(SUM(CASE WHEN e.cost_source = 'price-book-late' THEN e.cost_micro_usd ELSE 0 END), 0)
				FROM usage_events e WHERE EXISTS (
					SELECT 1 FROM usage_events m
					WHERE m.failed = 1 AND m.status_class = 'error'
					AND m.ts_ms / 3600000 * 3600000 = e.ts_ms / 3600000 * 3600000
					AND m.provider = e.provider
					AND m.model = e.model
					AND m.alias = e.alias
					AND m.auth_id = e.auth_id
					AND m.api_key_id = e.api_key_id
					AND m.stream = e.stream
				) GROUP BY e.ts_ms / 3600000 * 3600000, e.provider, e.model, e.alias,
					e.auth_id, e.api_key_id, e.status_class, e.stream`,
		},
	},
}

func migrate(db *sql.DB) error { return migrateTo(db, SchemaVersion) }

// migrateTo applies pending migrations up to target. Tests use a target
// below SchemaVersion to build an older database file first.
func migrateTo(db *sql.DB, target int) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at_ms INTEGER NOT NULL
	)`); err != nil {
		return fmt.Errorf("usagedb: create migrations table: %w", err)
	}
	var current int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("usagedb: read schema version: %w", err)
	}
	if current > SchemaVersion {
		return fmt.Errorf("usagedb: database schema version %d is newer than supported %d", current, SchemaVersion)
	}
	if target > SchemaVersion {
		return fmt.Errorf("usagedb: migration target %d is newer than supported %d", target, SchemaVersion)
	}
	for _, step := range migrations {
		if step.version <= current {
			continue
		}
		if step.version > SchemaVersion {
			return fmt.Errorf("usagedb: migration %d is newer than supported %d", step.version, SchemaVersion)
		}
		if step.version > target {
			continue
		}
		if err := applyMigration(db, step); err != nil {
			return err
		}
	}
	return nil
}

func applyMigration(db *sql.DB, step migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("usagedb: begin migration %d: %w", step.version, err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, stmt := range step.stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("usagedb: apply migration %d: %w", step.version, err)
		}
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations(version, applied_at_ms) VALUES (?, ?)`,
		step.version, unixMilliNow()); err != nil {
		return fmt.Errorf("usagedb: record migration %d: %w", step.version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("usagedb: commit migration %d: %w", step.version, err)
	}
	return nil
}
