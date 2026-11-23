# Changelog

All notable changes to `webappgen`, the module `github.com/standards-lab/webappgen` and the CLI
released from it, are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the module adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html). A written project starts its own
changelog; this one records the CLI's.

## [Unreleased]

## [v0.2.1] - 2026-11-23

A patch release: the projects webappgen writes, and its own sources, drop the provenance comments
that named the workspace sources they were copied from; that lineage now lives in the workspace's
references catalog. The written projects' pins are unchanged from v0.2.0: the layers
web-foundation v0.4.0, web-icons v0.1.0, web-components v0.2.0, web-app-sdk v0.1.0, and
web-tooling v0.1.0; pnpm 12.23.0; and, for service-app, go-web-sdk-template v0.16.0 and gonew from
golang.org/x/tools v0.53.0.

### Changed

- Projects written by `new standalone`, `new service-app`, and `inject service-app` carry no
  provenance comments naming workspace sources: not in the starter files, the service-app client
  and its wiring, or the `.air.toml` a new or injected service gets.
- webappgen's own sources carry none either.

## [v0.2.0] - 2026-11-23

The service-app kind: a web client inside a Go web service copied from go-web-sdk-template, which
embeds the client's build and serves it with go-web-sdk's `NewClient`, written into a new service
or installed into an existing one. Its `.air.toml` is adapted from herald (c697566), with a
provenance line.

### Added

- The `inject` group, whose leaves install a kind's web client into an existing project.
- `webappgen new service-app <module> <dir>`, which copies go-web-sdk-template v0.16.0 into a new
  or empty directory as `<module>`, with gonew from golang.org/x/tools v0.53.0 run through
  `go run`, then writes the client into it.
- `webappgen inject service-app <service-dir>`, which writes the client into an existing
  go-web-sdk-template service, under the module its `go.mod` declares.
- Their flags: `--name`, `--title`, `--base`, `--api-base`, `--client-dir`, and `--install`.
- What they write into the service: the client in `--client-dir`, built with a manifest for
  `NewClient`, its page drawn into the shell's empty `<body>`, and its Go package, which embeds the
  build; `internal/app/client.go`, the client's graph nodes and handler, defined and mounted from
  `internal/app/app.go` and `internal/app/server.go`; the service's `mise.toml` tasks,
  `client:build`, which `check`, `test`, `integration`, and `serve` depend on, `client:check`, and
  `dev`, and its air pin; `.air.toml`, air's live reload for `dev`; and `ci.yml` steps that install
  and build the client before the check and integration jobs run.
- Printed edits: a service that differs from go-web-sdk-template where an edit goes keeps that
  file as it is, and `inject` prints each edit left, then the next steps, and exits 1; under `new`,
  a copied template that differs is reported as a bug in the template.
- `--install` for service-app: under `new`, `mise install` and `go mod download` in the service,
  then `mise install` and `mise run install` in the client; under `inject`, the client's two.
- `mise run currency` reports the go-web-sdk-template and gonew pins behind their latest.

### Changed

- The CLI's marathon pin, 0.24.0.
- The pnpm pin, the CLI's and the written project's, 12.23.0.

## [v0.1.0] - 2026-11-23

The first release: the web client CLI, with its first kind of project, `standalone`. Its CLI shape
is copied from go-cli-sdk-template (a4d314c), its starter app from spike-html-apps (2883d78), and
its page's Settings and Help markup from web-components (6052866), with a provenance line on every
copied file; the spike is archived and never a dependency. The module depends on
`github.com/standards-lab/go-core` and `github.com/standards-lab/go-cli-sdk`.

### Added

- `webappgen version`, which prints the release the binary was built from.
- `webappgen new standalone <dir>`, which writes a static web client served on its own, with
  `--name`, `--title`, `--base`, `--api-base`, and `--install`. The written project depends on
  the web client stack's layers by exact release tag (web-foundation v0.4.0, web-icons v0.1.0,
  web-components v0.2.0, web-app-sdk v0.1.0, and web-tooling v0.1.0) and pins its toolchain,
  Node, pnpm, shellcheck, typescript-go, and chrome-headless-shell for linux and macOS on x64 and
  arm64, in `mise.toml`. Its `mise run check` lints, type-checks, runs the unit tier, builds,
  reports the build's size, and runs the browser tier on the build; its `mise run currency`
  reports any pin behind its latest.
- The written page: web-foundation's application page, with Help on the ? key, a dialog listing
  the page's shortcuts, and Settings on the S key, a dialog choosing the color, space, and type
  profiles. A choice applies and is stored at once, and applies before first paint on later
  visits.
- The written project's `README.md`: its tasks, its config, deploying it, and how it is built.
- `--install`, which runs `mise install` and `mise run install` in the written project.
- The release: a pushed `vX.Y.Z` tag on main's merge commit builds the CLI for linux and darwin
  on amd64 and arm64, and publishes `webappgen-vX.Y.Z-<os>-<arch>.tar.gz` archives and their
  `SHA256SUMS` on the GitHub release, so `mise use github:standards-lab/webappgen` installs it.

[Unreleased]: https://github.com/standards-lab/webappgen/compare/v0.2.1...HEAD
[v0.2.1]: https://github.com/standards-lab/webappgen/compare/v0.2.0...v0.2.1
[v0.2.0]: https://github.com/standards-lab/webappgen/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/standards-lab/webappgen/releases/tag/v0.1.0
