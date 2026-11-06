#!/usr/bin/env sh
# Print the fork release version: YY.MM.DD, plus a letter when the date
# version already exists (26.11.06, then 26.11.06a, 26.11.06b, ...).
# Usage: VERSION="$(sh scripts/version.sh)" docker build --build-arg VERSION="$VERSION" .
set -eu
cd "$(dirname "$0")/.."
base="$(date -u +%y.%m.%d)"
suffix=""
if git tag -l "v${base}*" 2>/dev/null | grep -q .; then
  letter="a"
  for _ in $(seq 1 26); do
    if git rev-parse -q --verify "refs/tags/v${base}${letter}" >/dev/null 2>&1; then
      letter=$(printf '%s' "$letter" | tr 'a-y' 'b-z')
    else
      break
    fi
  done
  suffix="$letter"
fi
printf '%s%s\n' "$base" "$suffix"
