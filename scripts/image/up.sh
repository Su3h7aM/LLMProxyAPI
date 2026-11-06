#!/usr/bin/env bash
#MISE description="Start the services from the compose file"
set -euo pipefail
cd "$(dirname "$0")/../.."
# shellcheck disable=SC1091
source scripts/image/lib-engine.sh
detect_engine
detect_compose

# shellcheck disable=SC2086
$COMPOSE -f "${COMPOSE_FILE:-compose.yaml}" up -d --remove-orphans
