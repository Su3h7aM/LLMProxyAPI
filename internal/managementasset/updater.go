// Package managementasset serves the control panel bundled with the binary.
//
// The panel is part of this project: apps/web builds it, the server build
// embeds dist/management.html, and one binary serves everything. There is
// no runtime download; MANAGEMENT_STATIC_PATH only overrides the on-disk
// copy used for panel development.
package managementasset

import (
	"os"
	"path/filepath"
	"strings"

	webpanel "github.com/Su3h7aM/LLMProxyAPI/apps/web"
	"github.com/Su3h7aM/LLMProxyAPI/internal/util"
)

// ManagementFileName exposes the control panel asset filename.
const ManagementFileName = "management.html"

// EmbeddedManagementHTML returns the control panel bundled into the binary
// at build time. Run the web build before the server build; see README.
func EmbeddedManagementHTML() []byte {
	return webpanel.ManagementHTML()
}

// StaticDir resolves the directory that stores the management control panel asset.
func StaticDir(configFilePath string) string {
	if override := strings.TrimSpace(os.Getenv("MANAGEMENT_STATIC_PATH")); override != "" {
		cleaned := filepath.Clean(override)
		if strings.EqualFold(filepath.Base(cleaned), ManagementFileName) {
			return filepath.Dir(cleaned)
		}
		return cleaned
	}

	if writable := util.WritablePath(); writable != "" {
		return filepath.Join(writable, "static")
	}

	configFilePath = strings.TrimSpace(configFilePath)
	if configFilePath == "" {
		return ""
	}

	base := filepath.Dir(configFilePath)
	fileInfo, err := os.Stat(configFilePath)
	if err == nil {
		if fileInfo.IsDir() {
			base = configFilePath
		}
	}

	return filepath.Join(base, "static")
}

// FilePath resolves the absolute path to the management control panel asset.
func FilePath(configFilePath string) string {
	if override := strings.TrimSpace(os.Getenv("MANAGEMENT_STATIC_PATH")); override != "" {
		cleaned := filepath.Clean(override)
		if strings.EqualFold(filepath.Base(cleaned), ManagementFileName) {
			return cleaned
		}
		return filepath.Join(cleaned, ManagementFileName)
	}

	dir := StaticDir(configFilePath)
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, ManagementFileName)
}
