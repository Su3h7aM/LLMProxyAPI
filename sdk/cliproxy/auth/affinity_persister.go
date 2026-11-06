package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Su3h7aM/LLMProxyAPI/internal/kvstore"
)

const affinitySnapshotKey = "session-affinity/bindings/v1"

// AffinityBinding is an explicit session binding and its expiry.
type AffinityBinding struct {
	AuthID    string    `json:"authID"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type affinitySnapshot struct {
	Version  int                        `json:"version"`
	Bindings map[string]AffinityBinding `json:"bindings"`
}

type affinityKV interface {
	KVGet(context.Context, string) ([]byte, bool, error)
	KVSet(context.Context, string, []byte, kvstore.KVSetOptions) (bool, error)
}

// AffinityPersister holds the live bindings and writes one snapshot at a time.
type AffinityPersister struct {
	store     affinityKV
	mu        sync.Mutex
	bindings  map[string]AffinityBinding
	wake      chan struct{}
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	err       error
	revision  uint64
	flushed   uint64
}

// NewAffinityPersister creates a persister for an active KV store.
func NewAffinityPersister(store affinityKV) *AffinityPersister {
	if store == nil {
		return nil
	}
	p := &AffinityPersister{store: store, bindings: make(map[string]AffinityBinding), wake: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{})}
	go p.run()
	return p
}

// Load reads the snapshot and drops expired or unknown credentials.
// Call Load before attaching the persister to a cache.
func (p *AffinityPersister) Load(ctx context.Context, known map[string]struct{}) error {
	if p == nil {
		return nil
	}
	data, ok, err := p.store.KVGet(ctx, affinitySnapshotKey)
	if err != nil || !ok {
		return err
	}
	var snapshot affinitySnapshot
	if err = json.Unmarshal(data, &snapshot); err != nil {
		return err
	}
	if snapshot.Version != 1 {
		return fmt.Errorf("unsupported session affinity snapshot version %d", snapshot.Version)
	}
	now := time.Now()
	p.mu.Lock()
	for key, b := range snapshot.Bindings {
		if _, exists := known[b.AuthID]; key != "" && exists && now.Before(b.ExpiresAt) {
			p.bindings[key] = b
		}
	}
	p.revision++
	p.mu.Unlock()
	p.notify()
	return nil
}

func (p *AffinityPersister) notify() {
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

// Bind updates the mirror without waiting for storage.
func (p *AffinityPersister) Bind(key, authID string, expiresAt time.Time) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.bindings[key] = AffinityBinding{AuthID: authID, ExpiresAt: expiresAt}
	p.revision++
	p.mu.Unlock()
	p.notify()
}

// Unbind removes a binding from the mirror.
func (p *AffinityPersister) Unbind(key string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	delete(p.bindings, key)
	p.revision++
	p.mu.Unlock()
	p.notify()
}

// Bindings returns a copy of the live mirror.
func (p *AffinityPersister) Bindings() map[string]AffinityBinding {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make(map[string]AffinityBinding, len(p.bindings))
	for key, b := range p.bindings {
		if time.Now().Before(b.ExpiresAt) {
			out[key] = b
		}
	}
	return out
}

func (p *AffinityPersister) flush() error {
	p.mu.Lock()
	revision := p.revision
	if revision == p.flushed {
		p.mu.Unlock()
		return nil
	}
	bindings := make(map[string]AffinityBinding, len(p.bindings))
	for key, binding := range p.bindings {
		if time.Now().Before(binding.ExpiresAt) {
			bindings[key] = binding
		}
	}
	p.mu.Unlock()
	data, err := json.Marshal(affinitySnapshot{Version: 1, Bindings: bindings})
	if err != nil {
		return err
	}
	_, err = p.store.KVSet(context.Background(), affinitySnapshotKey, data, kvstore.KVSetOptions{})
	if err == nil {
		p.mu.Lock()
		p.flushed = revision
		p.mu.Unlock()
	}
	return err
}

func (p *AffinityPersister) run() {
	defer close(p.done)
	for {
		select {
		case <-p.wake:
			timer := time.NewTimer(100 * time.Millisecond)
			select {
			case <-timer.C:
			case <-p.stop:
				timer.Stop()
				p.err = p.flush()
				return
			}
			p.err = p.flush()
		case <-p.stop:
			p.err = p.flush()
			return
		}
	}
}

// Close waits for the last snapshot write.
func (p *AffinityPersister) Close() error {
	if p == nil {
		return nil
	}
	p.closeOnce.Do(func() { close(p.stop) })
	<-p.done
	return p.err
}
