#!/usr/bin/env bash
# Fails naming mise run install when a tool a task needs is missing: each tool mise.toml pins, and
# the module's requirements in the module cache, which the build reads with the network off.
# Adapted from web-app-sdk (fbaba50): scripts/preflight.sh, for the CLI's tools and go.mod.
set -euo pipefail
cd "$(dirname "$0")/.."

fail() {
  echo "preflight: $1 isn't installed; run mise run install" >&2
  exit 1
}

for tool in actionlint go golangci-lint jq shellcheck; do
  mise which "$tool" >/dev/null 2>&1 || fail "$tool (mise.toml)"
done
GOPROXY=off go mod download >/dev/null 2>&1 || fail "go.mod's module cache"
