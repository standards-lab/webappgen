# Changelog

All notable changes to `webappgen`, the module `github.com/standards-lab/webappgen` and the CLI
released from it, are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the module adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html). A written project starts its own
changelog; this one records the CLI's.

## [Unreleased]

## [v0.1.0] - 2026-11-19

The first release: the web client CLI, with its first kind of project, `standalone`. Its CLI shape
is copied from go-cli-sdk-template (a4d314c) and its starter app from spike-html-apps (2883d78),
with a provenance line on every copied file; the spike is archived and never a dependency. The
module depends on `github.com/standards-lab/go-core` and `github.com/standards-lab/go-cli-sdk`.

### Added

- `webappgen version`, which prints the release the binary was built from.
- `webappgen new standalone <dir>`, which writes a static web client served on its own, with
  `--name`, `--title`, `--base`, `--api-base`, and `--install`. The written project depends on
  the web client stack's layers by exact release tag (web-foundation v0.3.0, web-components
  v0.1.0, web-app-sdk v0.1.0, and web-tooling v0.1.0) and pins its toolchain, Node, pnpm,
  typescript-go, and chrome-headless-shell for linux and macOS on x64 and arm64, in `mise.toml`.
  Its `mise run check` lints, type-checks, runs the unit tier, builds, reports the build's size,
  and runs the browser tier on the build; its `mise run currency` reports any pin behind its
  latest.
- `--install`, which runs `mise install` and `mise run install` in the written project.
- The release: a pushed `vX.Y.Z` tag on main's merge commit builds the CLI for linux and darwin
  on amd64 and arm64, and publishes `webappgen-vX.Y.Z-<os>-<arch>.tar.gz` archives and their
  `SHA256SUMS` on the GitHub release, so `mise use github:standards-lab/webappgen` installs it.

[Unreleased]: https://github.com/standards-lab/webappgen/compare/v0.1.0...HEAD
[v0.1.0]: https://github.com/standards-lab/webappgen/releases/tag/v0.1.0
