#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ -f .env ]]; then
  while IFS= read -r line; do
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ -z "${line//[[:space:]]/}" ]] && continue
    case "$line" in
      MIGRATE_DATABASE_URL=*|DATABASE_URL=*)
        key="${line%%=*}"
        value="${line#*=}"
        value="${value%\"}"
        value="${value#\"}"
        export "$key=$value"
        ;;
    esac
  done < .env
fi

DB_URL="${MIGRATE_DATABASE_URL:-${DATABASE_URL:-}}"
if [[ -z "$DB_URL" ]]; then
  echo "error: set MIGRATE_DATABASE_URL or DATABASE_URL in backend/.env" >&2
  exit 1
fi

if ! command -v migrate >/dev/null 2>&1; then
  echo "error: golang-migrate not found" >&2
  echo "install: brew install golang-migrate" >&2
  echo "    or: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" >&2
  exit 1
fi

cmd="${1:-up}"
shift || true

case "$cmd" in
  up)
  migrate -path migrations -database "$DB_URL" up "$@"
  ;;
  down)
  migrate -path migrations -database "$DB_URL" down "$@"
  ;;
  version)
  migrate -path migrations -database "$DB_URL" version
  ;;
  force)
  if [[ $# -lt 1 ]]; then
    echo "usage: $0 force <version>" >&2
    exit 1
  fi
  migrate -path migrations -database "$DB_URL" force "$1"
  ;;
  goto)
  if [[ $# -lt 1 ]]; then
    echo "usage: $0 goto <version>" >&2
    exit 1
  fi
  migrate -path migrations -database "$DB_URL" goto "$1"
  ;;
  *)
  echo "usage: $0 [up|down|version|force|goto] [args...]" >&2
  exit 1
  ;;
esac
