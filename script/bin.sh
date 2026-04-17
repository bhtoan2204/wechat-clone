#!/bin/bash

set -euo pipefail

SCRIPT_PATH="$(
  cd "$(dirname "$0")"
  pwd -P
)"
ROOT_DIR="$(dirname "$SCRIPT_PATH")"

resolve_path() {
  local value="$1"
  if [[ -z "$value" ]]; then
    return 0
  fi
  if [[ "$value" = /* ]]; then
    printf '%s\n' "$value"
    return 0
  fi
  printf '%s/%s\n' "$ROOT_DIR" "$value"
}

load_env_file() {
  local env_file="$1"
  if [[ -z "$env_file" || ! -f "$env_file" ]]; then
    return 0
  fi

  set -a
  # shellcheck disable=SC1090
  . "$env_file"
  set +a
}

apply_env() {
  load_env_file "$ROOT_DIR/secret/.env"

  local overlay_file
  overlay_file="$(resolve_path "${APP_ENV_OVERLAY_FILE:-}")"
  load_env_file "$overlay_file"
}

run_server() {
  apply_env
  echo "Running API server..."
  go run "$ROOT_DIR/cmd/main.go"
}

run_bootstrap() {
  apply_env
  echo "Bootstrapping local infrastructure..."
  go run "$ROOT_DIR/cmd/bootstrap"
}

run_migrate() {
  apply_env
  echo "Running database migrations..."
  go run "$ROOT_DIR/cmd/migrate"
}

case "${1:-}" in
  run)
    run_server
    ;;
  bootstrap)
    run_bootstrap
    ;;
  migrate)
    run_migrate
    ;;
  *)
    echo "Usage: $0 {run|bootstrap|migrate}"
    exit 1
    ;;
esac
