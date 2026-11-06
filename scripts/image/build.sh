#!/usr/bin/env bash
#MISE description="Build the container image from source"
set -euo pipefail
cd "$(dirname "$0")/../.."
# shellcheck disable=SC1091
source scripts/image/lib-engine.sh
detect_engine

IMAGE="${LLM_PROXY_IMAGE:-llm-proxy-api:local}"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo none)}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

$ENGINE build \
  -f Containerfile \
  --build-arg "VERSION=$VERSION" \
  --build-arg "COMMIT=$COMMIT" \
  --build-arg "BUILD_DATE=$BUILD_DATE" \
  -t "$IMAGE" .
echo "built $IMAGE"
