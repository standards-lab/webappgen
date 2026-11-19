#!/usr/bin/env bash
# Fails naming mise run install when a tool a task needs is missing: the node_modules tooling, and
# the layers the build reads, each at its release tag.
# Adapted from spike-html-apps (2883d78): scripts/preflight.sh, with web-components' check of the
# packages the build reads.
set -euo pipefail

fail() {
  echo "preflight: $1 isn't installed; run mise run install" >&2
  exit 1
}

for bin in eslint prettier vite; do
  [[ -x node_modules/.bin/$bin ]] || fail "$bin (devDependencies)"
done
for layer in web-foundation web-components web-app-sdk; do
  [[ -f node_modules/@standards-lab/$layer/package.json ]] || fail "@standards-lab/$layer (dependencies)"
done
