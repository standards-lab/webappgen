#!/usr/bin/env bash
# Fails naming mise run install when a tool a task needs is missing: each tool mise.toml pins; the
# module's requirements in the module cache, which the build reads with the network off; the
# service-app kind's pins in the module cache, which the integration tier's service-app run reads
# with the network off; and the pnpm store the integration tier's full proof installs a written
# project from, filled from the starter's lockfile as it stands.
set -euo pipefail
cd "$(dirname "$0")/.."

fail() {
  echo "preflight: $1 isn't installed; run mise run install" >&2
  exit 1
}

for tool in actionlint air go golangci-lint jq node pnpm shellcheck tsc chrome-headless-shell; do
  mise which "$tool" >/dev/null 2>&1 || fail "$tool (mise.toml)"
done
GOPROXY=off go mod download >/dev/null 2>&1 || fail "go.mod's module cache"
mapfile -t pins < <(scripts/service-pins.sh | awk '{ print $2 }')
GOPROXY=off go mod download "${pins[@]}" >/dev/null 2>&1 || fail "the service-app kind's pins (internal/serviceapp)"
cmp -s internal/starter/files/pnpm-lock.yaml .cache/pnpm-store/pnpm-lock.yaml ||
  fail "the starter's packages (.cache/pnpm-store)"
