package usagedb

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	coreusage "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/usage"
	log "github.com/sirupsen/logrus"
)

func init() {
	coreusage.RegisterNamedPlugin("usage-db", &Collector{})
}

// Collector is the built-in usage.Plugin that persists every record in the
// embedded database. HandleUsage never blocks: it converts the record and
// hands it to the writer channel. It runs when enabled in
// observability.usage-db and ignores usage-statistics-enabled.
type Collector struct{}

// HandleUsage implements coreusage.Plugin.
func (Collector) HandleUsage(_ context.Context, record coreusage.Record) {
	if !Enabled() {
		return
	}
	if record.RequestedAt.IsZero() {
		record.RequestedAt = time.Now()
	}
	Submit(RecordToRow(record))
}

// Options configures the package-level collector state.
type Options struct {
	Enabled          bool
	Path             string
	RetentionDays    int
	BufferSize       int
	HomeEnabled      bool
	PriceSyncEnabled bool
	PriceSyncURL     string
	PriceSyncHours   int
	ProxyURL         string
}

var (
	mu              sync.Mutex
	collectorOn     atomic.Bool
	collectorPath   atomic.Value // string
	collectorDays   atomic.Int64
	collectorDB     *Store
	collectorOut    *Writer
	retentionStop   chan struct{}
	priceSyncStop   chan struct{}
	priceSyncConfig PriceSyncConfig
	lastPruneMs     atomic.Int64
)

func init() {
	collectorOn.Store(true)
	collectorDays.Store(90)
	collectorPath.Store("")
}

// Enabled reports whether the collector persists records.
func Enabled() bool { return collectorOn.Load() }

// Submit queues one row on the active writer, if any.
func Submit(row UsageRow) {
	mu.Lock()
	writer := collectorOut
	mu.Unlock()
	if writer == nil {
		return
	}
	writer.Submit(row)
}

// Configure opens the database at path (resolved by the caller from the
// config and auth dir), starts the writer and the retention job, and applies
// the enabled flag and retention days. A path or buffer-size change replaces
// the writer: queued rows flush with a timeout first. It is safe to call
// Configure repeatedly, including with Enabled false to stop persistence.
func Configure(opts Options) {
	mu.Lock()
	defer mu.Unlock()
	collectorDays.Store(int64(opts.RetentionDays))
	if opts.BufferSize <= 0 {
		opts.BufferSize = 4096
	}
	if !opts.Enabled {
		collectorOn.Store(false)
		stopLocked(10 * time.Second)
		return
	}
	if opts.HomeEnabled {
		collectorOn.Store(false)
		stopLocked(10 * time.Second)
		log.Info("usagedb: home mode enabled, usage persistence disabled")
		return
	}
	syncConfig := PriceSyncConfig{
		Enabled:       opts.PriceSyncEnabled,
		URL:           opts.PriceSyncURL,
		IntervalHours: opts.PriceSyncHours,
		ProxyURL:      opts.ProxyURL,
	}
	if collectorDB != nil && collectorOut != nil && collectorPath.Load().(string) == opts.Path &&
		collectorOut.Capacity() == opts.BufferSize && priceSyncConfig == syncConfig {
		collectorOn.Store(true)
		return
	}
	stopLocked(10 * time.Second)
	db, err := Open(opts.Path)
	if err != nil {
		collectorOn.Store(false)
		log.WithError(err).WithField("path", opts.Path).Error("usagedb: open failed, usage persistence disabled")
		return
	}
	collectorDB = db
	collectorPath.Store(db.Path())
	collectorOut = NewWriter(db, opts.BufferSize)
	collectorOut.Start()
	collectorOn.Store(true)
	startRetentionLocked()
	priceSyncConfig = syncConfig
	startPriceSyncLocked()
	if _, err := db.LoadEmbeddedSnapshot(unixMilliNow()); err != nil {
		log.WithError(err).Error("usagedb: embedded price load failed, sync fills prices later")
	}
	log.WithField("path", db.Path()).Info("usagedb: usage database open")
}

// Shutdown stops the writer (flushing with timeout) and closes the database.
// The retention job stops too. Status keeps the last known path and counters.
func Shutdown(timeout time.Duration) {
	mu.Lock()
	defer mu.Unlock()
	collectorOn.Store(false)
	stopLocked(timeout)
}

// stopLocked flushes the writer and closes the database. Callers hold mu.
func stopLocked(timeout time.Duration) {
	if priceSyncStop != nil {
		close(priceSyncStop)
		priceSyncStop = nil
	}
	if retentionStop != nil {
		close(retentionStop)
		retentionStop = nil
	}
	if collectorOut != nil {
		collectorOut.Stop(timeout)
		collectorOut = nil
	}
	if collectorDB != nil {
		if err := collectorDB.Close(); err != nil {
			log.WithError(err).Error("usagedb: close failed")
		}
		collectorDB = nil
	}
}

// Snapshot describes the database state for the status route.
type Snapshot struct {
	Enabled       bool
	Path          string
	SizeBytes     int64
	SchemaVersion int
	Events        int64
	OldestTsMs    int64
	NewestTsMs    int64
	BufferDepth   int
	BufferCap     int
	Dropped       int64
	WriteErrors   int64
	LastError     string
	RetentionDays int64
	LastPruneMs   int64
}

// Status reads the current collector, writer, and database state.
func Status() Snapshot {
	mu.Lock()
	writer := collectorOut
	db := collectorDB
	mu.Unlock()
	status := Snapshot{
		Enabled:       collectorOn.Load(),
		SchemaVersion: SchemaVersion,
		RetentionDays: collectorDays.Load(),
		LastPruneMs:   lastPruneMs.Load(),
	}
	if path, ok := collectorPath.Load().(string); ok {
		status.Path = path
	}
	if writer != nil {
		status.BufferDepth = writer.Depth()
		status.BufferCap = writer.Capacity()
		status.Dropped = writer.Dropped()
		status.WriteErrors = writer.WriteErrors()
		status.LastError = writer.LastError()
	}
	if db != nil {
		status.SizeBytes = db.SizeBytes()
		if stats, err := db.Stats(); err == nil {
			status.Events = stats.Events
			status.OldestTsMs = stats.OldestTsMs
			status.NewestTsMs = stats.NewestTsMs
		} else {
			log.WithError(err).Warn("usagedb: stats failed")
		}
	}
	return status
}
