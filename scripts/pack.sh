#!/usr/bin/env bash
# Usage: scripts/pack.sh <vX.Y.Z> <outdir>
# Builds the CLI for linux and darwin on amd64 and arm64 with CGO_ENABLED=0 from a clean checkout
# of the tag, refuses any binary whose version isn't the tag, and writes
# webappgen-<tag>-<os>-<arch>.tar.gz, the binary and LICENSE, for each into outdir, with their
# SHA256SUMS. The release workflow runs it, and a developer can, on a clone at the tag.
#
# go build stamps the main module's version from git: the tag when HEAD is the tagged commit and
# the tree is clean, a pseudo-version or +dirty otherwise, so the version check also refuses a
# build from any other commit or a dirty tree. Each binary's stamped version is read with go
# version -m, which reads any target's binary; the one the host can run is also run, so its
# version command's output is checked too.
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: scripts/pack.sh <vX.Y.Z> <outdir>" >&2
  exit 2
fi
tag=$1
out=$2
if [[ ! $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "pack: $tag isn't a vX.Y.Z tag" >&2
  exit 1
fi

root=$(cd "$(dirname "$0")/.." && pwd)
export GOWORK=off CGO_ENABLED=0
host=$(go env GOHOSTOS)/$(go env GOHOSTARCH)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
targets=(linux/amd64 linux/arm64 darwin/amd64 darwin/arm64)

# Every binary is built and checked before any archive is written, so no build sees an archive
# under an outdir inside the checkout as an untracked file, which go build stamps as +dirty.
for target in "${targets[@]}"; do
  dir=$work/${target/\//-}
  mkdir "$dir"
  (cd "$root" && GOOS=${target%/*} GOARCH=${target#*/} go build -trimpath -o "$dir/webappgen" ./cmd/webappgen)

  # The mod line reads `mod <path> <version> [<sum>]`.
  stamped=$(go version -m "$dir/webappgen" | awk '$1 == "mod" { print $3 }')
  if [[ $stamped != "$tag" ]]; then
    echo "pack: the $target binary's version is ${stamped:-none}, not $tag" >&2
    exit 1
  fi
  if [[ $target == "$host" ]]; then
    printed=$("$dir/webappgen" version)
    if [[ $printed != "$tag" ]]; then
      echo "pack: the $target binary's version command prints $printed, not $tag" >&2
      exit 1
    fi
  fi
done

mkdir -p "$out"
cd "$out"
archives=()
for target in "${targets[@]}"; do
  dir=$work/${target/\//-}
  cp "$root/LICENSE" "$dir/LICENSE"
  archive=webappgen-$tag-${target/\//-}.tar.gz
  tar -czf "$archive" -C "$dir" webappgen LICENSE
  archives+=("$archive")
  echo "pack: $archive"
done

if command -v sha256sum >/dev/null; then
  sha256sum -- "${archives[@]}" >SHA256SUMS
else
  shasum -a 256 -- "${archives[@]}" >SHA256SUMS
fi
echo "pack: SHA256SUMS"
