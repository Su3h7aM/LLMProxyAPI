#!/usr/bin/env bash
#MISE description="Build the web panel bundle"
set -euo pipefail
cd "$(dirname "$0")/../apps/web"
bun install
bun run build
