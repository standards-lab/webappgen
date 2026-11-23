#!/usr/bin/env bash
# Fills .cache/pnpm-store with every package the starter's lockfile pins, for the integration
# tier's full proof, which installs a written project from it with pnpm offline; then copies the
# lockfile beside the packages, which preflight compares with the starter's to find a store that
# is missing or stale. pnpm fetch reads the lockfile alone, with the workspace's settings beside
# it, in a scratch directory. Reads the network.
set -euo pipefail
cd "$(dirname "$0")/.."

lockfile=internal/starter/files/pnpm-lock.yaml
store=$PWD/.cache/pnpm-store
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

cp "$lockfile" internal/starter/files/pnpm-workspace.yaml "$work/"
(cd "$work" && pnpm fetch --store-dir "$store")
cp "$lockfile" "$store/pnpm-lock.yaml"
