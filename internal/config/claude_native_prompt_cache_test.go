package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfigBytesClaudeNativeOAuthPromptCache1h(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want bool
	}{
		{name: "defaults to false", yaml: "port: 8317\n", want: false},
		{name: "legacy layout", yaml: "claude:\n  native-oauth-prompt-cache-1h: true\n", want: true},
		{name: "v8 layout", yaml: "config-version: 8\nupstream:\n  claude:\n    native-oauth-prompt-cache-1h: true\n", want: true},
		{name: "v8 explicit false", yaml: "config-version: 8\nupstream:\n  claude:\n    native-oauth-prompt-cache-1h: false\n", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, errParse := ParseConfigBytes([]byte(tt.yaml))
			if errParse != nil {
				t.Fatalf("ParseConfigBytes() error = %v", errParse)
			}
			if got := cfg.Claude.NativeOAuthPromptCache1h; got != tt.want {
				t.Fatalf("cfg.Claude.NativeOAuthPromptCache1h = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestSaveConfigPreserveComments_NativeOAuthPromptCache1hRoundTrip(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if errWrite := os.WriteFile(configPath, []byte("port: 8317\n"), 0o600); errWrite != nil {
		t.Fatalf("os.WriteFile() error = %v", errWrite)
	}

	cfg := &Config{Claude: ClaudeConfig{NativeOAuthPromptCache1h: true}}
	if errSave := SaveConfigPreserveComments(configPath, cfg); errSave != nil {
		t.Fatalf("SaveConfigPreserveComments() error = %v", errSave)
	}

	savedBytes, errRead := os.ReadFile(configPath)
	if errRead != nil {
		t.Fatalf("os.ReadFile() error = %v", errRead)
	}
	if !strings.Contains(string(savedBytes), "native-oauth-prompt-cache-1h: true") {
		t.Fatalf("saved YAML missing native-oauth-prompt-cache-1h: true:\n%s", savedBytes)
	}
	loaded, errLoad := LoadConfigOptional(configPath, false)
	if errLoad != nil {
		t.Fatalf("LoadConfigOptional() error = %v", errLoad)
	}
	if !loaded.Claude.NativeOAuthPromptCache1h {
		t.Fatal("loaded NativeOAuthPromptCache1h = false, want true")
	}
}
