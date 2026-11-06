#!/usr/bin/env bash
#MISE description="Build the proxy server binary"
#MISE depends=["stage-panel"]
set -euo pipefail
cd "$(dirname "$0")/.."
go build -o llmproxy ./cmd/server
