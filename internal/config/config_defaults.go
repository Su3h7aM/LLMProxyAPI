package config

const (
	DefaultPanelGitHubRepository = "https://github.com/router-for-me/Cli-Proxy-API-Management-Center"
	DefaultPprofAddr             = "127.0.0.1:8316"
	// DefaultAuthDir is the legacy pre-XDG data directory. New installs use
	// DefaultDataDir; MigrateLegacyDataDir moves this directory there.
	DefaultAuthDir              = "~/.llm-proxy-api"
	DefaultDiscoveryServiceType = "_ai-gateway._tcp"
	DefaultKVStorePath          = "kv.db"
)
