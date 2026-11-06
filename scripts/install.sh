#!/usr/bin/env bash
#MISE description="Install Node dependencies"
set -euo pipefail
cd "$(dirname "$0")/.."
npm install --no-audit --no-fund
