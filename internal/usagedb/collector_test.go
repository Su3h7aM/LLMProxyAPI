package usagedb

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func configureTestDB(t *testing.T, retentionDays int) {
	t.Helper()
	Shutdown(5 * time.Second)
	path := filepath.Join(t.TempDir(), "usage.db")
	Configure(Options{Enabled: true, Path: path, RetentionDays: retentionDays, BufferSize: 64})
	t.Cleanup(func() { Shutdown(5 * time.Second) })
}

func waitForEvents(t *testing.T, want int64) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		status := Status()
		if status.Events >= want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("events did not reach %d: %+v", want, Status())
}

func TestCollectorRecordInRowOut(t *testing.T) {
	configureTestDB(t, 90)
	record := testRecord()
	record.RequestID = "collector-1"
	Collector{}.HandleUsage(context.Background(), record)
	waitForEvents(t, 1)
	status := Status()
	if !status.Enabled {
		t.Fatal("collector is disabled")
	}
	if status.Events != 1 {
		t.Fatalf("events = %d, want 1", status.Events)
	}
	if status.SchemaVersion != SchemaVersion {
		t.Fatalf("schema = %d, want %d", status.SchemaVersion, SchemaVersion)
	}
	if status.BufferCap != 64 {
		t.Fatalf("buffer cap = %d, want 64", status.BufferCap)
	}
}

func TestCollectorDisabledDrops(t *testing.T) {
	configureTestDB(t, 90)
	Shutdown(5 * time.Second)
	Configure(Options{Enabled: false, Path: "", RetentionDays: 90, BufferSize: 64})
	Collector{}.HandleUsage(context.Background(), testRecord())
	time.Sleep(300 * time.Millisecond)
	if status := Status(); status.Enabled {
		t.Fatal("collector is enabled after Configure with Enabled false")
	}
}

func TestCollectorIgnoresUsageStatisticsFlag(t *testing.T) {
	configureTestDB(t, 90)
	// usage-statistics-enabled gates only the redis queue plugin; the
	// collector follows observability.usage-db.enabled instead.
	record := testRecord()
	record.RequestID = "collector-2"
	Collector{}.HandleUsage(context.Background(), record)
	waitForEvents(t, 1)
}

func TestCollectorStampsMissingTime(t *testing.T) {
	configureTestDB(t, 90)
	record := testRecord()
	record.RequestID = "collector-3"
	record.RequestedAt = time.Time{}
	before := unixMilliNow()
	Collector{}.HandleUsage(context.Background(), record)
	waitForEvents(t, 1)
	mu.Lock()
	db := collectorDB
	mu.Unlock()
	var ts int64
	if err := db.readDB.QueryRow(`SELECT ts_ms FROM usage_events WHERE request_id = 'collector-3'`).Scan(&ts); err != nil {
		t.Fatalf("query ts: %v", err)
	}
	if ts < before {
		t.Fatalf("ts = %d, want at least %d", ts, before)
	}
}

func TestPruneOnce(t *testing.T) {
	configureTestDB(t, 1)
	now := unixMilliNow()
	mu.Lock()
	db := collectorDB
	mu.Unlock()
	old := testRow("prune-old", now-int64(48*time.Hour/time.Millisecond))
	fresh := testRow("prune-new", now)
	if _, err := db.InsertBatch([]UsageRow{old, fresh}); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	pruneOnce(db)
	stats, err := db.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Events != 1 {
		t.Fatalf("events = %d, want 1", stats.Events)
	}
	if got := lastPruneMs.Load(); got == 0 {
		t.Fatal("last prune timestamp is 0")
	}
}
