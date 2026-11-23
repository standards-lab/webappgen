#!/usr/bin/env bash
# Fills .cache/pnpm-store with every package the starter's lockfile pins, for the integration
# tier's full proof, which installs a written project from it with pnpm offline; then copies the
# lockfile beside the packages, which preflight compares with the starter's to find a store that
# is missing or stale. pnpm fetch runs in a project the CLI writes to a scratch directory, so it
# reads the lockfile with the project's package.json and workspace settings beside it; without
# the package.json, pnpm takes the starter's git dependencies for subdependencies and refuses
# them. Reads the network.
set -euo pipefail
cd "$(dirname "$0")/.."

lockfile=internal/starter/files/pnpm-lock.yaml
store=$PWD/.cache/pnpm-store
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

go run ./cmd/webappgen new standalone "$work/project" >/dev/null
(cd "$work/project" && pnpm fetch --store-dir "$store")
cp "$lockfile" "$store/pnpm-lock.yaml"
