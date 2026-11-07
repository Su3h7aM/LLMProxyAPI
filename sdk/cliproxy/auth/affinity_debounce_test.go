package auth

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/Su3h7aM/LLMProxyAPI/internal/kvstore"
)

func TestAffinityTouchWritesAfterDebounce(t *testing.T) {
	store, err := kvstore.OpenBoltStore(filepath.Join(t.TempDir(), "kv.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	p := NewAffinityPersister(store)
	c := NewSessionCache(time.Hour)
	defer c.Stop()
	c.SetPersister(p)
	c.Set("session", "auth")
	c.Touch("session", "auth")
	want := p.Bindings()["session"].ExpiresAt
	deadline := time.Now().Add(3 * time.Second)
	for {
		data, ok, err := store.KVGet(context.Background(), affinitySnapshotKey)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			var snapshot affinitySnapshot
			if err := json.Unmarshal(data, &snapshot); err != nil {
				t.Fatal(err)
			}
			if snapshot.Bindings["session"].ExpiresAt.Equal(want) {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("snapshot did not contain refreshed expiry")
		}
		time.Sleep(10 * time.Millisecond)
	}
	c.SetPersister(nil)
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}
