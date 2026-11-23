#!/usr/bin/env bash
# Fails naming mise run install when a tool a task needs is missing: the node_modules tooling, and
# the layers the build reads, each at its release tag; and with `browser`, the pinned
# chrome-headless-shell.
#
# The browser's version is checked against its pin: mise caches downloads by checksum, so a version
# bumped without its sha256 installs the old build.
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

[[ "${1:-}" == browser ]] || exit 0
shell=$(mise which chrome-headless-shell 2>/dev/null) || fail "chrome-headless-shell"
pin=$(mise current http:chrome-headless-shell)
if [[ "$("$shell" --version)" != *" $pin" ]]; then
  echo "preflight: $shell isn't chrome-headless-shell $pin; give the pin its own sha256" >&2
  exit 1
fi
