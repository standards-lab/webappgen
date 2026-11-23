#!/usr/bin/env bash
# Prints `<where>: <item> <pin> -> <latest>` for each mise tool, package.json pin, and layer's tag
# behind its latest, and exits 1 if any; prints nothing when all are current. Reads the network and
# writes nothing.
#
# mise runs at error level, as it warns on every run that a GitHub release has no assets. A layer
# is pinned to its repository's release tag, which the registry doesn't hold, so its tag is checked
# against that repository's latest release.
set -euo pipefail
cd "$(dirname "$0")/.."

outdated=$(MISE_LOG_LEVEL=error mise outdated --bump --local --json)
trailing=$(node -p '
  Object.entries(JSON.parse(process.argv[1]))
    .map(([tool, { requested, bump }]) => "mise.toml: " + [tool, requested, "->", bump].join(" "))
    .join("\n")
' "$outdated")

newest() {
  local accepted
  accepted=$(pnpm view "$1@$2" version --json)
  node -p '[].concat(JSON.parse(process.argv[1])).join("\n")' "$accepted" | sort -V | tail -n 1
}

pins=$(node -p '
  const { dependencies, devDependencies } = require("./package.json");
  Object.entries({ ...dependencies, ...devDependencies })
    .map(([name, pin]) => name + " " + pin)
    .join("\n")
')
pin_of() {
  node -p 'require("./package.json").devDependencies[process.argv[1]]' "$1"
}
while read -r package pin; do
  if [[ $pin == github:* ]]; then
    spec=${pin#github:}
    latest=$(gh api "repos/${spec%%#*}/releases/latest" --jq .tag_name)
    if [[ ${spec#*#} != "$latest" ]]; then
      trailing+=$'\n'"package.json: $package ${spec#*#} -> $latest"
    fi
    continue
  fi
  case $package in
    # typescript is held to the newest version typescript-eslint's peer range accepts, and
    # rolldown to the newest vite's dependency range accepts, so the build runs the one vite tests.
    typescript)
      latest=$(newest typescript "$(pnpm view "typescript-eslint@$(pin_of typescript-eslint)" peerDependencies.typescript)")
      ;;
    rolldown)
      latest=$(newest rolldown "$(pnpm view "vite@$(pin_of vite)" dependencies.rolldown)")
      ;;
    *)
      latest=$(pnpm view "$package" version)
      ;;
  esac
  if [[ $pin != "$latest" ]]; then
    trailing+=$'\n'"package.json: $package $pin -> $latest"
  fi
done <<<"$pins"

if [[ -n ${trailing//$'\n'/} ]]; then
  sed '/^$/d' <<<"$trailing"
  exit 1
fi
