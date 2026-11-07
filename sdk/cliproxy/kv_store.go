package cliproxy

import (
	"context"

	"github.com/Su3h7aM/LLMProxyAPI/internal/config"
	"github.com/Su3h7aM/LLMProxyAPI/internal/home"
	"github.com/Su3h7aM/LLMProxyAPI/internal/kvstore"
	coreauth "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/auth"
	log "github.com/sirupsen/logrus"
)

func (s *Service) applyKVStoreConfig(cfg *config.Config) {
	if s == nil {
		if errShutdown := kvstore.ShutdownLocal(); errShutdown != nil {
			log.Errorf("failed to shutdown local kv store: %v", errShutdown)
		}
		return
	}
	s.affinityMu.Lock()
	defer s.affinityMu.Unlock()
	active := cfg != nil && !cfg.Home.Enabled && home.Current() == nil && !cfg.KVStore.Disabled
	path := ""
	if active {
		path = cfg.EffectiveKVStorePath()
	}
	if s.affinityReady && s.affinityPersister != nil && active && path == s.affinityKVPath {
		return
	}
	s.affinityKVPath = path
	var seed map[string]coreauth.AffinityBinding
	if s.affinityReady {
		seed = s.stopAffinityPersistenceLocked()
		defer func() { s.startAffinityPersistenceLocked(context.Background(), seed) }()
	}
	s.kvStoreManaged.Store(true)
	if cfg == nil || cfg.Home.Enabled || home.Current() != nil || cfg.KVStore.Disabled {
		if errShutdown := kvstore.ShutdownLocal(); errShutdown != nil {
			log.Errorf("failed to shutdown local kv store: %v", errShutdown)
		}
		return
	}
	_ = kvstore.ConfigureLocal(true, cfg.EffectiveKVStorePath())
}

func (s *Service) shutdownKVStore() error {
	if s == nil {
		return nil
	}
	s.kvStoreManaged.Store(false)
	return kvstore.ShutdownLocal()
}
