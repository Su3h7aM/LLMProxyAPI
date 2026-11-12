package config

import (
	"path/filepath"
	"testing"
)

func TestParseConfigBytes_UsageDBV8(t *testing.T) {
	v8YAML := []byte("config-version: 8\nobservability:\n  usage-db:\n    enabled: false\n    path: /var/data/usage.db\n    retention-days: 30\n    buffer-size: 512\n")
	if errValidate := ValidateV8Config(v8YAML); errValidate != nil {
		t.Fatalf("ValidateV8Config() error = %v", errValidate)
	}
	cfg, errParse := ParseConfigBytes(v8YAML)
	if errParse != nil {
		t.Fatalf("ParseConfigBytes(v8) error = %v", errParse)
	}
	if cfg.UsageDB.Enabled || cfg.UsageDB.Path != "/var/data/usage.db" ||
		cfg.UsageDB.RetentionDays != 30 || cfg.UsageDB.BufferSize != 512 {
		t.Fatalf("cfg.UsageDB = %+v, want {Enabled:false Path:/var/data/usage.db RetentionDays:30 BufferSize:512}", cfg.UsageDB)
	}
	if got := cfg.EffectiveUsageDBPath(); got != "/var/data/usage.db" {
		t.Fatalf("EffectiveUsageDBPath() = %q, want /var/data/usage.db", got)
	}
}

func TestParseConfigBytes_UsageDBDefaults(t *testing.T) {
	// Omitted usage-db keeps persistence on with 90-day retention and a 4096 buffer.
	cfg, errParse := ParseConfigBytes([]byte("config-version: 8\nserver:\n  port: 8317\n"))
	if errParse != nil {
		t.Fatalf("ParseConfigBytes(omitted) error = %v", errParse)
	}
	if !cfg.UsageDB.Enabled || cfg.UsageDB.Path != "" ||
		cfg.UsageDB.RetentionDays != DefaultUsageDBRetentionDays ||
		cfg.UsageDB.BufferSize != DefaultUsageDBBufferSize {
		t.Fatalf("cfg.UsageDB = %+v, want enabled with defaults", cfg.UsageDB)
	}
	wantDefault := filepath.Join(DefaultDataDir(), DefaultUsageDBPath)
	if got := cfg.EffectiveUsageDBPath(); got != wantDefault {
		t.Fatalf("EffectiveUsageDBPath() = %q, want %q", got, wantDefault)
	}

	// Negative retention and buffer sizes fall back to defaults; 0 keeps all rows.
	zeroCfg, errZero := ParseConfigBytes([]byte("config-version: 8\nobservability:\n  usage-db:\n    retention-days: 0\n    buffer-size: -5\n"))
	if errZero != nil {
		t.Fatalf("ParseConfigBytes(zero) error = %v", errZero)
	}
	if zeroCfg.UsageDB.RetentionDays != 0 {
		t.Fatalf("retention-days = %d, want 0 (keep all)", zeroCfg.UsageDB.RetentionDays)
	}
	if zeroCfg.UsageDB.BufferSize != DefaultUsageDBBufferSize {
		t.Fatalf("buffer-size = %d, want %d", zeroCfg.UsageDB.BufferSize, DefaultUsageDBBufferSize)
	}

	// Explicit path wins over auth-dir.
	authCfg, errAuth := ParseConfigBytes([]byte("config-version: 8\noauth:\n  auth-dir: /tmp/custom-auth\n"))
	if errAuth != nil {
		t.Fatalf("ParseConfigBytes(auth-dir) error = %v", errAuth)
	}
	if got, want := authCfg.EffectiveUsageDBPath(), filepath.Join("/tmp/custom-auth", DefaultUsageDBPath); got != want {
		t.Fatalf("EffectiveUsageDBPath() = %q, want %q", got, want)
	}

	// Nil config falls back to the data dir.
	var nilCfg *Config
	if got := nilCfg.EffectiveUsageDBPath(); got != wantDefault {
		t.Fatalf("nil EffectiveUsageDBPath() = %q, want %q", got, wantDefault)
	}
}

func TestParseConfigBytes_UsageDBPriceSync(t *testing.T) {
	v8YAML := []byte("config-version: 8\nobservability:\n  usage-db:\n    price-sync:\n      enabled: false\n      url: http://localhost:9999/prices\n      interval-hours: 6\n")
	if errValidate := ValidateV8Config(v8YAML); errValidate != nil {
		t.Fatalf("ValidateV8Config() error = %v", errValidate)
	}
	cfg, errParse := ParseConfigBytes(v8YAML)
	if errParse != nil {
		t.Fatalf("ParseConfigBytes(v8) error = %v", errParse)
	}
	sync := cfg.UsageDB.PriceSync
	if sync.Enabled || sync.URL != "http://localhost:9999/prices" || sync.IntervalHours != 6 {
		t.Fatalf("PriceSync = %+v, want {false http://localhost:9999/prices 6}", sync)
	}
}

func TestParseConfigBytes_UsageDBPriceSyncDefaults(t *testing.T) {
	cfg, errParse := ParseConfigBytes([]byte("config-version: 8\nserver:\n  port: 8317\n"))
	if errParse != nil {
		t.Fatalf("ParseConfigBytes(omitted) error = %v", errParse)
	}
	sync := cfg.UsageDB.PriceSync
	if !sync.Enabled || sync.URL != DefaultPriceSyncURL || sync.IntervalHours != DefaultPriceSyncIntervalHours {
		t.Fatalf("PriceSync = %+v, want enabled with models.dev default and 24h", sync)
	}
}
