#!/bin/sh
# Generates internal/usagedb/snapshot.json.gz from https://models.dev/api.json.
# Keeps only the fields the price book stores (ids, names, cost with tiers)
# and only the providers the fork serves plus a small set of common ones.
# Target: under 400 KB gzip. The build and the tests never call the network;
# the committed snapshot lets the first start work offline.
set -eu

OUT="${1:-internal/usagedb/snapshot.json.gz}"
URL="${MODELS_DEV_URL:-https://models.dev/api.json}"

tmp_json="$(mktemp)"
trap 'rm -f "$tmp_json"' EXIT

curl -sS -m 120 -A "LLMProxyAPI-price-tool" "$URL" -o "$tmp_json"

python3 - "$tmp_json" > "${OUT%.gz}.tmp" <<'EOF'
import json, sys

providers = [
    # Providers the fork serves.
    "anthropic", "openai", "google", "xai", "moonshotai", "meta",
    "opencode-go",
    # Small set of common ones.
    "deepseek", "cohere", "mistral", "groq", "cerebras", "togetherai",
    "fireworks-ai", "alibaba", "minimax",
]

keep_cost_keys = ("input", "output", "cache_read", "cache_write",
                  "reasoning", "tiers", "context_over_200k")

with open(sys.argv[1], encoding="utf-8") as handle:
    doc = json.load(handle)

snapshot = {"providers": {}}
for provider in providers:
    entry = doc.get(provider)
    if not entry:
        continue
    models = {}
    for model_id, model in (entry.get("models") or {}).items():
        cost = (model or {}).get("cost") or {}
        if not cost.get("input") or not cost.get("output"):
            continue
        trimmed = {key: cost[key] for key in keep_cost_keys if key in cost}
        models[model_id] = {"name": (model or {}).get("name", ""), "cost": trimmed}
    if models:
        snapshot["providers"][provider] = {"models": models}

print(json.dumps(snapshot, separators=(",", ":")))
EOF

mv "${OUT%.gz}.tmp" "${OUT%.gz}"
gzip -9 -f "${OUT%.gz}"
ls -la "$OUT"
