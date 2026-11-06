package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDataDirHonorsXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data-test")
	if got := DefaultDataDir(); got != filepath.Join("/tmp/xdg-data-test", XDGAppName) {
		t.Fatalf("DefaultDataDir = %q", got)
	}
}

func TestMigrateLegacyDataDirRenames(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	legacy := filepath.Join(home, ".llm-proxy-api")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "auth.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	migrated, err := MigrateLegacyDataDir()
	if err != nil || !migrated {
		t.Fatalf("migrated = %v, err = %v", migrated, err)
	}
	dest := filepath.Join(home, ".local", "share", XDGAppName, "auth.json")
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("moved file missing: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatal("legacy directory should be gone after rename")
	}
}

func TestMigrateLegacyDataDirMerges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	legacy := filepath.Join(home, ".llm-proxy-api")
	dest := filepath.Join(home, ".local", "share", XDGAppName)
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "keep.json"), []byte(`{"v":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "keep.json"), []byte(`{"v":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "new.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	migrated, err := MigrateLegacyDataDir()
	if err != nil || !migrated {
		t.Fatalf("migrated = %v, err = %v", migrated, err)
	}
	// Destination files are never overwritten.
	kept, err := os.ReadFile(filepath.Join(dest, "keep.json"))
	if err != nil || string(kept) != `{"v":1}` {
		t.Fatalf("destination overwritten: %q %v", kept, err)
	}
	if _, err := os.Stat(filepath.Join(dest, "new.json")); err != nil {
		t.Fatalf("missing entry not merged: %v", err)
	}
}

func TestMigrateLegacyDataDirNoop(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	migrated, err := MigrateLegacyDataDir()
	if err != nil || migrated {
		t.Fatalf("migrated = %v, err = %v", migrated, err)
	}
}
