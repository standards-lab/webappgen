# webappgen context

webappgen is the web client CLI, a Go command on go-cli-sdk: `webappgen new <kind>` writes a
client project for one kind of app on the stack, from files embedded in its binary. It writes one
kind, `standalone`, a static app. Two embedded kinds, a client inside a new or an existing Go web
service served by go-web-sdk's `NewClient`, are planned for the goal's `embedded` task (v0.2.0).

It is the root Go module `github.com/standards-lab/webappgen`, released by a plain `vX.Y.Z` tag
as binaries for Linux and macOS; its first release is v0.1.0. Its goals live in the coordinator's
roadmap (`standards-lab/roadmap/index.toml`, v1.client.stack), and its strategy is
`standards-lab/context/web-client.md`. Its CLI shape is copied from go-cli-sdk-template and its
starter app from spike-html-apps' app, with a provenance line on every copied file; the spike is
archived and never a dependency. [README.md](../README.md) documents the flags, the written
project's tasks, deployment, and the contributor workflow.

## Capability map

- **CLI** (built): [`internal/app`](../internal/app/): `version`, and `new` with one leaf per
  kind of app. Usage errors name the fix.
- **Standalone kind** (built): [`internal/standalone`](../internal/standalone/): `new standalone
  <dir>` writes the starter, then the standalone kind's own files over it (`index.html`,
  `vite.config.ts`, `public/config.json`). `--install` then runs `mise install` and
  `mise run install` in the project.
- **Embedded kinds** (planned): `new new-service` and `new existing-service` write a client
  inside a new or an existing Go web service, served by go-web-sdk's `NewClient` (v0.2.0).
- **The written project** (built): [`internal/starter/files`](../internal/starter/files/), the
  starter every kind writes before its own files, on the layers by exact tag with a committed
  lockfile. With the standalone kind's files it is the foundation's application page with Help
  and Settings, a home and a not-found view, a config reader, and a check with unit and browser
  tiers. Its [`README.md`](../internal/starter/files/README.md.tmpl) documents its tasks and how
  it is built.
- **The full proof** (built): [`integration`](../integration/): the root `mise run install`
  warms `.cache/pnpm-store`, and the integration tier writes a project with `--install` offline
  and runs the written project's check.
- **Release** (built): [`release.yml`](../.github/workflows/release.yml): a tag on main
  triggers the workflow, which runs `scripts/pack.sh` to build four archives and `SHA256SUMS`,
  attaches them to a draft release, then publishes it.

## Provenance

The JSON and generated files, which take no comment, have their provenance here; every other
copied file carries its own line.

- `internal/starter/files/.prettierrc.json`: from spike-html-apps (2883d78) `.prettierrc.json`,
  byte for byte.
- `internal/starter/files/package.json.tmpl`: adapted from spike-html-apps (2883d78)
  `package.json` and `test/consumers/app/package.json`, moved onto the standards-lab layers.
- `internal/standalone/files/public/config.json.tmpl`: new.
- `internal/starter/files/pnpm-lock.yaml`: produced by `pnpm install` (pnpm 12.21.0).
