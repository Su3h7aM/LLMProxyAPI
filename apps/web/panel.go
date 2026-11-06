// Package web exposes the control panel bundle built from this directory.
// Build it first (mise run build-web); dist stays gitignored. The server
// binary embeds dist/management.html, so one binary serves everything.
package web

import _ "embed"

//go:embed dist/management.html
var managementHTML []byte

// ManagementHTML returns the bundled control panel.
func ManagementHTML() []byte {
	return managementHTML
}
