#!/usr/bin/env bash
# Prints `<where>: <item> <pin> -> <latest>` for the go directive's minor and any toolchain line,
# each direct Go requirement, each mise tool, and each GitHub Actions pin behind its latest, and
# exits 1 if any; prints nothing when all are current. Reads the network and writes nothing;
# diagnostics go to stderr.
# Adapted from go-cli-sdk-template (a4d314c): template/scripts/currency.sh, on one module, with a
# toolchain line reported.
# Adapted from web-app-sdk (fbaba50): scripts/currency.sh, its report collected and printed once.
#
# Each scan captures its command's output in a variable before reading it, so under set -e a scan
# command that fails (a go list missing a go.sum entry) fails currency instead of reporting
# nothing. The go directive follows the current Go minor, with no toolchain line, as the mise pin
# selects the toolchain. Go requirements are checked within their major. The mise tools include
# the starter's shared pins, so a tool reported here trails in a written project too; mise runs at
# error level, as it warns on every run that a GitHub release has no assets. Each workflow's
# actions, the release workflow's included, are checked against their repositories' latest
# releases.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOWORK=off

trailing=""
report() {
  trailing+="$1"$'\n'
}

go_latest=$(MISE_LOG_LEVEL=error mise latest go)
go_minor=$(cut -d. -f1,2 <<<"$go_latest")
gomod=$(go mod edit -json)
directive=$(jq -r .Go <<<"$gomod" | cut -d. -f1,2)
[[ $directive == "$go_minor" ]] || report "go.mod: go $directive -> $go_minor"
toolchain=$(jq -r '.Toolchain // empty' <<<"$gomod")
[[ -z $toolchain ]] || report "go.mod: toolchain $toolchain -> none"

updates=$(go list -m -u -f \
  '{{if and (not .Main) (not .Indirect) .Update}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}' all)
while read -r line; do
  [[ -z $line ]] || report "go.mod: $line"
done <<<"$updates"

outdated=$(MISE_LOG_LEVEL=error mise outdated --bump --no-global --json |
  jq -r 'to_entries[] | "\(.key) \(.value.requested) -> \(.value.bump)"')
while read -r line; do
  [[ -z $line ]] || report "mise.toml: $line"
done <<<"$outdated"

uses=$(grep -ho 'uses: *[^ ]*@[^ ]*' .github/workflows/*.yml | sed 's/uses: *//' | sort -u)
while read -r action; do
  [[ -n $action ]] || continue
  repo=$(cut -d/ -f1,2 <<<"${action%@*}")
  latest=$(gh api "repos/$repo/releases/latest" --jq .tag_name)
  [[ ${action#*@} == "$latest" ]] || report ".github/workflows: ${action%@*} ${action#*@} -> $latest"
done <<<"$uses"

if [[ -n $trailing ]]; then
  printf '%s' "$trailing"
  exit 1
fi
