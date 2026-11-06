#!/usr/bin/env bash
#MISE description="Build the web panel bundle"
#MISE depends=["install"]
set -euo pipefail
cd "$(dirname "$0")/.."
npm --workspace apps/web run build
