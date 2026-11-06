package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// XDGAppName is the application directory name under the XDG base directories.
const XDGAppName = "llm-proxy-api"

// xdgDir resolves $XDG_VAR, falling back to $HOME/fallback when unset.
func xdgDir(envVar, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(envVar)); value != "" {
		return filepath.Clean(value)
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, filepath.FromSlash(fallback))
}

// DefaultDataDir returns the data directory for auth files and the KV store:
// $XDG_DATA_HOME/llm-proxy-api, or ~/.local/share/llm-proxy-api.
func DefaultDataDir() string {
	if base := xdgDir("XDG_DATA_HOME", ".local/share"); base != "" {
		return filepath.Join(base, XDGAppName)
	}
	return ""
}

// LegacyDataDir returns the pre-XDG data directory (~/.llm-proxy-api),
// "" when the home directory is unknown.
func LegacyDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, ".llm-proxy-api")
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// MigrateLegacyDataDir moves the pre-XDG data directory (~/.llm-proxy-api)
// to the XDG data directory. When the destination already exists, only
// entries missing there are moved; existing destination files are never
// overwritten. It returns whether anything moved.
func MigrateLegacyDataDir() (bool, error) {
	legacy := LegacyDataDir()
	if legacy == "" || !dirExists(legacy) {
		return false, nil
	}
	dest := DefaultDataDir()
	if dest == "" {
		return false, errors.New("xdg data directory is unknown")
	}
	if legacy == dest {
		return false, nil
	}
	if !dirExists(dest) {
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return false, err
		}
		if err := os.Rename(legacy, dest); err != nil {
			return false, err
		}
		log.Infof("migrated data directory %s to %s", legacy, dest)
		return true, nil
	}
	entries, err := os.ReadDir(legacy)
	if err != nil {
		return false, err
	}
	moved := 0
	for _, entry := range entries {
		target := filepath.Join(dest, entry.Name())
		if _, err := os.Lstat(target); err == nil {
			continue
		}
		if err := os.Rename(filepath.Join(legacy, entry.Name()), target); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return moved > 0, err
		}
		moved++
	}
	if moved > 0 {
		log.Infof("migrated %d data entries %s to %s", moved, legacy, dest)
	}
	return moved > 0, nil
}
