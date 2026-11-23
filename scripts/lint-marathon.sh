#!/usr/bin/env bash
# Runs marathon's lint-context and lint-references, both, and exits 1 if either fails or is not on
# PATH, naming mise run install for a missing one.
set -euo pipefail
cd "$(dirname "$0")/.."

failed=0

run_lint() {
  local lint=$1
  if ! command -v "$lint" >/dev/null; then
    echo "FAIL: $lint is not on PATH; run mise run install" >&2
    failed=1
  elif ! "$lint"; then
    failed=1
  fi
}

run_lint lint-context
run_lint lint-references
exit "$failed"
