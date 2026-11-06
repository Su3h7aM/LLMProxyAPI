#!/usr/bin/env bash
#MISE description="Stage the web bundle as management.html"
#MISE depends=["build-web"]
set -euo pipefail
cd "$(dirname "$0")/.."
cp apps/web/dist/index.html apps/web/dist/management.html
