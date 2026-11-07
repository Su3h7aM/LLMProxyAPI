package cliproxy

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Su3h7aM/LLMProxyAPI/internal/config"
	"github.com/Su3h7aM/LLMProxyAPI/internal/kvstore"
	coreauth "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/auth"
	"github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/executor"
)

func TestAffinityPersistenceFlushesBeforeKVClose(t *testing.T) {
	defer kvstore.ShutdownLocal()
	path := filepath.Join(t.TempDir(), "kv.db")
	newPath := filepath.Join(t.TempDir(), "new-kv.db")
	selector := coreauth.NewSessionAffinitySelector(nil)
	defer selector.Stop()
	manager := coreauth.NewManager(nil, selector, nil)
	credential := &coreauth.Auth{ID: "auth-a", Provider: "anthropic"}
	if _, err := manager.Register(context.Background(), credential); err != nil {
		t.Fatal(err)
	}
	s := &Service{coreManager: manager}
	s.applyKVStoreConfig(&config.Config{KVStore: config.KVStoreConfig{Path: path}})
	s.startAffinityPersistence(context.Background())
	opts := executor.Options{Headers: map[string][]string{"X-Claude-Code-Session-Id": {"session"}}}
	if _, err := selector.Pick(context.Background(), "anthropic", "model", opts, []*coreauth.Auth{credential}); err != nil {
		t.Fatal(err)
	}
	s.applyKVStoreConfig(&config.Config{KVStore: config.KVStoreConfig{Path: newPath}})
	oldStore, err := kvstore.OpenBoltStore(path)
	if err != nil {
		t.Fatal(err)
	}
	_, exists, err := oldStore.KVGet(context.Background(), "session-affinity/bindings/v1")
	if err != nil || !exists {
		t.Fatalf("old snapshot exists = %v, error = %v", exists, err)
	}
	if err := oldStore.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	store, err := kvstore.OpenBoltStore(newPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	p := coreauth.NewAffinityPersister(store)
	if err := p.Load(context.Background(), map[string]struct{}{"auth-a": {}}); err != nil {
		t.Fatal(err)
	}
	if len(p.Bindings()) != 1 {
		t.Fatalf("bindings: %#v", p.Bindings())
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}
