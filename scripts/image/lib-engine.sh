#!/usr/bin/env bash
# Shared container engine detection. Source it, do not run it.
# Honors CONTAINER_ENGINE and CONTAINER_COMPOSE when set.

detect_engine() {
  if [ -n "${CONTAINER_ENGINE:-}" ]; then
    ENGINE="$CONTAINER_ENGINE"
  elif command -v podman >/dev/null 2>&1; then
    ENGINE="podman"
  elif command -v docker >/dev/null 2>&1; then
    ENGINE="docker"
  else
    echo "error: no container engine found (set CONTAINER_ENGINE to podman or docker)" >&2
    return 1
  fi
}

detect_compose() {
  if [ -n "${CONTAINER_COMPOSE:-}" ]; then
    COMPOSE="$CONTAINER_COMPOSE"
  elif "$ENGINE" compose version >/dev/null 2>&1; then
    COMPOSE="$ENGINE compose"
  elif command -v podman-compose >/dev/null 2>&1; then
    COMPOSE="podman-compose"
  elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE="docker-compose"
  else
    echo "error: no compose command found" >&2
    return 1
  fi
}
