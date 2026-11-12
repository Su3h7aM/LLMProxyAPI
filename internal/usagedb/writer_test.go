package usagedb

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestWriterBatching(t *testing.T) {
	store := openTestStore(t)
	writer := NewWriter(store, 4096)
	writer.Start()
	now := unixMilliNow()
	for i := 0; i < 10; i++ {
		writer.Submit(testRow(fmt.Sprintf("req-%d", i), now))
	}
	writer.Stop(10 * time.Second)
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Events != 10 {
		t.Fatalf("events = %d, want 10", stats.Events)
	}
	if got := writer.Dropped(); got != 0 {
		t.Fatalf("dropped = %d, want 0", got)
	}
}

func TestWriterDropOldestOnFull(t *testing.T) {
	store := openTestStore(t)
	writer := NewWriter(store, 2)
	// No Start: nothing drains, so the last submits must drop the oldest.
	now := unixMilliNow()
	for i := 0; i < 5; i++ {
		writer.Submit(testRow(fmt.Sprintf("req-%d", i), now))
	}
	if got := writer.Dropped(); got != 3 {
		t.Fatalf("dropped = %d, want 3", got)
	}
	if got := writer.Depth(); got != 2 {
		t.Fatalf("depth = %d, want 2", got)
	}
	writer.Start()
	writer.Stop(10 * time.Second)
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Events != 2 {
		t.Fatalf("events = %d, want 2", stats.Events)
	}
	var ids []string
	rows, err := store.readDB.Query(`SELECT request_id FROM usage_events ORDER BY request_id`)
	if err != nil {
		t.Fatalf("query ids: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan id: %v", err)
		}
		ids = append(ids, id)
	}
	if len(ids) != 2 || ids[0] != "req-3" || ids[1] != "req-4" {
		t.Fatalf("stored ids = %v, want [req-3 req-4]", ids)
	}
}

func TestWriterShutdownFlush(t *testing.T) {
	store := openTestStore(t)
	writer := NewWriter(store, 4096)
	writer.Start()
	now := unixMilliNow()
	for i := 0; i < 3; i++ {
		writer.Submit(testRow(fmt.Sprintf("flush-%d", i), now))
	}
	// Stop flushes even though the 250 ms batch window has not passed.
	writer.Stop(10 * time.Second)
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Events != 3 {
		t.Fatalf("events = %d, want 3", stats.Events)
	}
}

func TestWriterConcurrentRace(t *testing.T) {
	store := openTestStore(t)
	writer := NewWriter(store, 4096)
	writer.Start()
	now := unixMilliNow()
	var group sync.WaitGroup
	for w := 0; w < 8; w++ {
		group.Add(1)
		go func(slot int) {
			defer group.Done()
			for i := 0; i < 25; i++ {
				writer.Submit(testRow(fmt.Sprintf("req-%d-%d", slot, i), now))
			}
		}(w)
	}
	group.Wait()
	writer.Stop(30 * time.Second)
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Events != 200 {
		t.Fatalf("events = %d, want 200", stats.Events)
	}
}

func TestWriterRetryThenDrop(t *testing.T) {
	store := openTestStore(t)
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	writer := NewWriter(store, 4)
	writer.maxTries = 3
	writer.baseBackoff = time.Millisecond
	writer.maxBackoff = time.Millisecond
	writer.writeWithRetry([]UsageRow{testRow("req-1", unixMilliNow())})
	if got := writer.WriteErrors(); got != 3 {
		t.Fatalf("write errors = %d, want 3", got)
	}
	if got := writer.Dropped(); got != 1 {
		t.Fatalf("dropped = %d, want 1", got)
	}
	if got := writer.LastError(); got == "" {
		t.Fatal("last error is empty, want the insert error")
	}
}
