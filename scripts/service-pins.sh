#!/usr/bin/env bash
# Prints `<constant> <module>@<version>` for each pin the service-app kind declares in
# internal/serviceapp/serviceapp.go, a string constant whose name ends in Pin: the template it
# copies and the x/tools gonew runs from. currency, fetch-service, and preflight read the pins
# here, and the kind's unit test holds that this pattern finds every one. Reads only the file.
set -euo pipefail
cd "$(dirname "$0")/.."

sed -nE 's/^[[:space:]]*([A-Za-z]+Pin)[[:space:]]*=[[:space:]]*"([^"@]+@[^"]+)"$/\1 \2/p' \
  internal/serviceapp/serviceapp.go
