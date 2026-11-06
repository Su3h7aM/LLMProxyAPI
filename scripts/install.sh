#!/usr/bin/env bash
#MISE description="Install web dependencies"
set -euo pipefail
cd "$(dirname "$0")/../apps/web"
bun install
