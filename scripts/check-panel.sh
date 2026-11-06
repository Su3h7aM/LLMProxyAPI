#!/usr/bin/env bash
#MISE description="Boot the server and check the panel serves"
#MISE depends=["build-server", "stage-panel"]
set -euo pipefail
cd "$(dirname "$0")/.."

PORT="${PORT:-8327}"
WORK="$(mktemp -d)"
trap 'kill "$PID" 2>/dev/null; rm -rf "$WORK"' EXIT

cat > "$WORK/config.yaml" <<EOF
config-version: 8
server:
  host: "127.0.0.1"
  port: $PORT
management:
  allow-remote: false
  secret-key: "test-key"
remote-management:
  disable-auto-update-panel: true
access:
  api-keys: ["test-key"]
EOF

MANAGEMENT_STATIC_PATH="$PWD/apps/web/dist" ./llmproxy -config "$WORK/config.yaml" > "$WORK/server.log" 2>&1 &
PID=$!

for _ in $(seq 1 30); do
  if curl -sf -o /dev/null "http://127.0.0.1:$PORT/management.html"; then
    break
  fi
  sleep 1
done

TITLE="$(curl -s "http://127.0.0.1:$PORT/management.html" | grep -o "<title>[^<]*</title>" | head -n 1)"
echo "panel: $TITLE"
test "$TITLE" = "<title>CLI Proxy API Management Center</title>"
