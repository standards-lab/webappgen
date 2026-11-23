#!/usr/bin/env bash
# Fills the module cache with what a service-app run reads, for the integration tier, which runs
# one offline: the pinned x/tools and its requirements, which go run builds gonew from; the pinned
# template; and the template's requirements, which the written service's tests build from. It
# writes a scratch service with the pinned gonew, then downloads that service's requirements and
# installs its mise tools, which the written service's tasks run with. Reads the network.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOWORK=off

pins=$(scripts/service-pins.sh)
template=$(awk '$1 == "TemplatePin" { print $2 }' <<<"$pins")
tools=$(awk '$1 == "ToolsPin" { print $2 }' <<<"$pins")
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

go run "${tools%@*}/cmd/gonew@${tools#*@}" "$template" example.com/service "$work/service"
(cd "$work/service" && go mod download)
MISE_TRUSTED_CONFIG_PATHS="$work/service" mise -C "$work/service" install
