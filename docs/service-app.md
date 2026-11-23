# The service-app kind

What `webappgen new service-app` and `webappgen inject service-app` write into a Go web service
copied from [go-web-sdk-template](https://github.com/standards-lab/go-web-sdk-template), and what
they print when the service differs from it. [README.md](../README.md) has the commands and their
flags.

## What is written

- **The client**, in `--client-dir`: the starter, as the standalone kind writes it, with a Vite
  build that writes a manifest for go-web-sdk's `NewClient`, a page module that draws the
  application page into the shell's empty `<body>`, and a Go package beside it that embeds the
  build, `dist/`, and exports it with the entry and render-blocking modules `NewClient` names. Its
  browser tier serves the build behind the shell `NewClient` renders, so its check needs no Go. Its
  own `README.md` documents its tasks and deployment.
- **`internal/app/client.go`**: the client's graph nodes, `NewClient` over the client's build
  with the base, title, and API base as constants, and the mount of its handler under the base.
  `internal/app/app.go` and `internal/app/server.go` gain the calls that define and mount it.
- **`mise.toml`**: the air pin, and three tasks. `client:build` runs the client's
  `mise run build`, and `check`, `test`, `integration`, and `serve` depend on it. `client:check`
  runs the client's own check, and `check` depends on it too. `dev` runs the client's
  `vite build --watch` beside air.
- **`.air.toml`**: air's live reload of the service for `dev`, rebuilding and restarting it on
  each Go, JS, and CSS change.
- **`.github/workflows/ci.yml`**: steps in the check and integration jobs that install the
  client's tools and packages and build it before they check or test.

## Install steps

`--install` runs, in order, from the service's root, each named on stderr:

| Step | `new` | `inject` |
|------|-------|----------|
| `mise install` | yes | no |
| `go mod download` | yes | no |
| `mise -C <client-dir> install` | yes | yes |
| `mise -C <client-dir> run install` | yes | yes |

A program missing from `PATH` or a step that fails exits nonzero naming the step and the steps
left, and what was written stays. Without `--install`, the next steps list these steps; with it,
only `mise run check` is left, after `git init` under `new`.

## Printed edits

An edit to an existing file is made only when the file holds what go-web-sdk-template has at every
place the edit goes: for the Go wiring, the import declaration, the `Nodes` struct, the
`defineServer` call in `New`, and `newRouter`'s return; for `mise.toml` and the CI workflow, each
table or job the edit changes. `mise.toml` must also define none of `client:build`,
`client:check`, and `dev`, and `.air.toml` is written only when none exists. Other differences
don't block an edit. The Go edits are made all together or left all together, and each
configuration file is edited whole or left whole.

Under `new`, a copied template that fails these conditions is a bug in it, and the run exits
nonzero naming the template's version. Under `inject`, a service that fails them keeps that file as
it is, and the run prints each edit left, with its file, its anchor, whether the anchor was found,
and the text to insert, then the next steps, and exits 1; `--install` then runs nothing. When
`mise.toml` is left, the next steps build the client with `mise -C <client-dir> run build` before
`mise run check`.

## Deploying

Build the client, then the service: the client's `mise run build` writes `dist/`, and the
service's Go build embeds it through the client's Go package, so the binary carries the client.
The Go package compiles only once `dist/` is built; `dist/` is never committed. The service serves
the client under `--base`, answering the shell for any path under it that isn't a built file, so a
deep link or a reload reaches the client's router, and writes the client's config into the shell,
so there is no `config.json` to replace.
