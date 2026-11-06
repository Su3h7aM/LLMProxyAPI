package auth

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Su3h7aM/LLMProxyAPI/internal/kvstore"
)

func TestAffinityPersisterReload(t *testing.T) {
	store, err := kvstore.OpenBoltStore(filepath.Join(t.TempDir(), "kv.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	p := NewAffinityPersister(store)
	now := time.Now()
	p.Bind("live", "known", now.Add(time.Hour))
	p.Bind("expired", "known", now.Add(-time.Hour))
	p.Bind("removed", "unknown", now.Add(time.Hour))
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	p = NewAffinityPersister(store)
	if err := p.Load(context.Background(), map[string]struct{}{"known": {}}); err != nil {
		t.Fatal(err)
	}
	bindings := p.Bindings()
	if len(bindings) != 1 || bindings["live"].AuthID != "known" {
		t.Fatalf("unexpected bindings: %#v", bindings)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}
