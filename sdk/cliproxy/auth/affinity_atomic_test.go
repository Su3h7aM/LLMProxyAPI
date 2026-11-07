package auth

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/Su3h7aM/LLMProxyAPI/internal/kvstore"
)

type refreshSnapshotStore struct {
	mu      sync.Mutex
	writes  int
	missing bool
}

func (*refreshSnapshotStore) KVGet(context.Context, string) ([]byte, bool, error) {
	return nil, false, nil
}
func (s *refreshSnapshotStore) KVSet(_ context.Context, _ string, data []byte, _ kvstore.KVSetOptions) (bool, error) {
	var snapshot affinitySnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writes++
	if snapshot.Bindings["session"].AuthID != "auth" {
		s.missing = true
	}
	return true, nil
}
func TestAffinityRefreshSnapshotIsAtomic(t *testing.T) {
	store := &refreshSnapshotStore{}
	p := NewAffinityPersister(store)
	c := NewSessionCache(time.Hour)
	defer c.Stop()
	c.SetPersister(p)
	c.Set("session", "auth")
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
				c.Touch("session", "auth")
			}
		}
	}()
	time.Sleep(150 * time.Millisecond)
	close(stop)
	<-done
	c.SetPersister(nil)
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.writes == 0 || store.missing {
		t.Fatalf("writes = %d, missing binding = %v", store.writes, store.missing)
	}
}
