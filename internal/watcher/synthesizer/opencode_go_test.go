package synthesizer

import (
	"testing"
	"time"

	"github.com/Su3h7aM/LLMProxyAPI/internal/config"
)

// Config-backed OpenCode Go keys have no file path. They must carry
// runtime_only so the management auth-files listing, and the quota page, include them.
func TestConfigSynthesizer_OpenCodeGoKeysAreListable(t *testing.T) {
	synth := NewConfigSynthesizer()
	ctx := &SynthesisContext{
		Config: &config.Config{
			OpenCodeGoKey: []config.OpenCodeGoKey{{APIKey: "go-secret"}},
		},
		Now:         time.Now(),
		IDGenerator: NewStableIDGenerator(),
	}

	auths, err := synth.Synthesize(ctx)
	if err != nil {
		t.Fatalf("Synthesize() error = %v", err)
	}
	var found bool
	for _, auth := range auths {
		if auth == nil || auth.Provider != "opencode-go" {
			continue
		}
		found = true
		if auth.Attributes["runtime_only"] != "true" {
			t.Fatalf("runtime_only = %q, want true", auth.Attributes["runtime_only"])
		}
		if auth.Attributes["api_key"] != "go-secret" {
			t.Fatalf("api_key not carried over")
		}
	}
	if !found {
		t.Fatal("no opencode-go auth synthesized")
	}
}
