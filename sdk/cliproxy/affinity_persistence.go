package cliproxy

import (
	"context"

	"github.com/Su3h7aM/LLMProxyAPI/internal/kvstore"
	coreauth "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/auth"
	log "github.com/sirupsen/logrus"
)

func (s *Service) startAffinityPersistence(ctx context.Context) {
	if s == nil {
		return
	}
	s.affinityMu.Lock()
	defer s.affinityMu.Unlock()
	s.affinityReady = true
	s.startAffinityPersistenceLocked(ctx, nil)
}

func (s *Service) startAffinityPersistenceLocked(ctx context.Context, seed map[string]coreauth.AffinityBinding) {
	if s.coreManager == nil {
		return
	}
	store, active, err := kvstore.CurrentKVClient()
	if err != nil {
		log.WithError(err).Warn("session affinity persistence is unavailable")
		return
	}
	if !active || store == nil {
		return
	}
	var empty bool
	if len(seed) > 0 {
		_, exists, readErr := store.KVGet(ctx, "session-affinity/bindings/v1")
		if readErr != nil {
			log.WithError(readErr).Warn("failed to read session affinity bindings")
			return
		}
		empty = !exists
	}
	p := coreauth.NewAffinityPersister(store)
	known := make(map[string]struct{})
	for _, credential := range s.coreManager.List() {
		if credential != nil {
			known[credential.ID] = struct{}{}
		}
	}
	if err := p.Load(ctx, known); err != nil {
		log.WithError(err).Warn("failed to load session affinity bindings")
		_ = p.Close()
		return
	}
	if empty {
		for key, binding := range seed {
			if _, ok := known[binding.AuthID]; ok {
				p.Bind(key, binding.AuthID, binding.ExpiresAt)
			}
		}
	}
	s.affinityPersister = p
	s.coreManager.SetAffinityPersister(p)
}

func (s *Service) stopAffinityPersistence() {
	if s == nil {
		return
	}
	s.affinityMu.Lock()
	defer s.affinityMu.Unlock()
	s.affinityReady = false
	s.stopAffinityPersistenceLocked()
}

func (s *Service) stopAffinityPersistenceLocked() map[string]coreauth.AffinityBinding {
	if s.affinityPersister == nil {
		return nil
	}
	if s.coreManager != nil {
		s.coreManager.SetAffinityPersister(nil)
	}
	bindings := s.affinityPersister.Bindings()
	if err := s.affinityPersister.Close(); err != nil {
		log.WithError(err).Error("failed to save session affinity bindings")
	}
	s.affinityPersister = nil
	return bindings
}
