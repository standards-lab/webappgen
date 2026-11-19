#!/usr/bin/env bash
# Fails naming mise run install when a tool a task needs is missing: each tool mise.toml pins; the
# module's requirements in the module cache, which the build reads with the network off; and the
# pnpm store the integration tier's full proof installs a written project from, filled from the
# starter's lockfile as it stands.
# Adapted from web-app-sdk (fbaba50): scripts/preflight.sh, for the CLI's tools and go.mod.
set -euo pipefail
cd "$(dirname "$0")/.."

fail() {
  echo "preflight: $1 isn't installed; run mise run install" >&2
  exit 1
}

for tool in actionlint go golangci-lint jq node pnpm shellcheck tsc chrome-headless-shell; do
  mise which "$tool" >/dev/null 2>&1 || fail "$tool (mise.toml)"
done
GOPROXY=off go mod download >/dev/null 2>&1 || fail "go.mod's module cache"
cmp -s internal/starter/files/pnpm-lock.yaml .cache/pnpm-store/pnpm-lock.yaml ||
  fail "the starter's packages (.cache/pnpm-store)"
