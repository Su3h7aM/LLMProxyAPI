package auth

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Su3h7aM/LLMProxyAPI/internal/kvstore"
	cliproxyexecutor "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/executor"
)

func TestAffinityCacheMirror(t *testing.T) {
	tests := []struct {
		name   string
		change func(*SessionCache)
		want   bool
	}{
		{"invalidate", func(c *SessionCache) { c.Invalidate("session") }, false},
		{"invalidate auth", func(c *SessionCache) { c.InvalidateAuth("auth") }, false},
		{"compare and delete", func(c *SessionCache) { c.CompareAndDelete("session", "auth") }, false},
		{"touch", func(c *SessionCache) { c.Touch("session", "auth") }, true},
		{"refresh", func(c *SessionCache) { c.GetAndRefresh("session") }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			before := p.Bindings()["session"].ExpiresAt
			tt.change(c)
			got, ok := p.Bindings()["session"]
			if ok != tt.want {
				t.Fatalf("binding present = %v, want %v", ok, tt.want)
			}
			if tt.want && !got.ExpiresAt.After(before) {
				t.Fatalf("expiry did not advance")
			}
			c.SetPersister(nil)
			if err := p.Close(); err != nil {
				t.Fatal(err)
			}
			p = NewAffinityPersister(store)
			if err := p.Load(context.Background(), map[string]struct{}{"auth": {}}); err != nil {
				t.Fatal(err)
			}
			persisted, exists := p.Bindings()["session"]
			if exists != tt.want || (exists && !persisted.ExpiresAt.Equal(got.ExpiresAt)) {
				t.Fatalf("snapshot binding = %#v, present = %v", persisted, exists)
			}
			if err := p.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAffinitySelectorSwapFencesOldCache(t *testing.T) {
	store, err := kvstore.OpenBoltStore(filepath.Join(t.TempDir(), "kv.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	p := NewAffinityPersister(store)
	old := NewSessionAffinitySelectorWithConfig(SessionAffinityConfig{})
	m := NewManager(nil, old, nil)
	m.SetAffinityPersister(p)
	old.cache.Set("session", "auth")
	next := NewSessionAffinitySelectorWithConfig(SessionAffinityConfig{})
	defer next.Stop()
	m.SetSelector(next)
	old.cache.Set("session", "wrong")
	if got, ok := next.cache.Get("session"); !ok || got != "auth" {
		t.Fatalf("new binding = %q, %v", got, ok)
	}
	if got := p.Bindings()["session"].AuthID; got != "auth" {
		t.Fatalf("mirror auth = %q", got)
	}
	m.SetAffinityPersister(nil)
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	p = NewAffinityPersister(store)
	if err := p.Load(context.Background(), map[string]struct{}{"auth": {}, "wrong": {}}); err != nil {
		t.Fatal(err)
	}
	if got := p.Bindings()["session"].AuthID; got != "auth" {
		t.Fatalf("snapshot auth = %q", got)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAffinityRestartRestoresPrimaryAndRebindsAlias(t *testing.T) {
	store, err := kvstore.OpenBoltStore(filepath.Join(t.TempDir(), "kv.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	p := NewAffinityPersister(store)
	first := NewSessionAffinitySelector(nil)
	first.cache.SetPersister(p)
	auth := &Auth{ID: "auth-a", Provider: "openai"}
	opts := cliproxyexecutor.Options{OriginalRequest: []byte(`{"prompt_cache_key":"primary","conversation":"alias"}`), Metadata: map[string]any{}}
	if _, err := first.Pick(context.Background(), "openai", "model", opts, []*Auth{auth}); err != nil {
		t.Fatal(err)
	}
	first.cache.SetPersister(nil)
	first.Stop()
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	p = NewAffinityPersister(store)
	if err := p.Load(context.Background(), map[string]struct{}{"auth-a": {}}); err != nil {
		t.Fatal(err)
	}
	next := NewSessionAffinitySelector(nil)
	defer next.Stop()
	next.cache.SetPersister(p)
	picked, err := next.Pick(context.Background(), "openai", "model", opts, []*Auth{{ID: "auth-b", Provider: "openai"}, auth})
	if err != nil || picked == nil || picked.ID != auth.ID {
		t.Fatalf("pick = %v, %v", picked, err)
	}
	if got, ok := next.cache.Get("openai::conv:alias::model"); !ok || got != auth.ID {
		t.Fatalf("alias = %q, %v", got, ok)
	}
	next.cache.SetPersister(nil)
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}
