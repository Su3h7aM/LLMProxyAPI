package config

const (
	DefaultPanelGitHubRepository = "https://github.com/router-for-me/Cli-Proxy-API-Management-Center"
	DefaultPprofAddr             = "127.0.0.1:8316"
	// DefaultAuthDir is the legacy pre-XDG data directory. New installs use
	// DefaultDataDir; MigrateLegacyDataDir moves this directory there.
	DefaultAuthDir              = "~/.llm-proxy-api"
	DefaultDiscoveryServiceType = "_ai-gateway._tcp"
	DefaultKVStorePath          = "kv.db"
	// DefaultUsageDBPath is the file name of the embedded usage SQLite database.
	// When no explicit path is set it lives at "<auth-dir>/usage.db".
	DefaultUsageDBPath = "usage.db"
	// DefaultUsageDBRetentionDays keeps raw usage events for 90 days; 0 keeps all.
	DefaultUsageDBRetentionDays = 90
	// DefaultUsageDBBufferSize bounds the in-process usage writer channel.
	DefaultUsageDBBufferSize = 4096
	// DefaultPriceSyncURL is the public price source for the usage price book.
	DefaultPriceSyncURL = "https://models.dev/api.json"
	// DefaultPriceSyncIntervalHours re-syncs the price book once a day.
	DefaultPriceSyncIntervalHours = 24
)
