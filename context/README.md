# webappgen context

webappgen is the web client CLI, a Go command on go-cli-sdk: `webappgen new <kind>` writes a
client project for one kind of app on the stack, and `webappgen inject <kind>` installs a client
into an existing project, from files embedded in its binary. It writes two kinds: `standalone`, a
static app, and `service-app`, a client inside a new or an existing go-web-sdk-template service,
served by go-web-sdk's `NewClient` (v0.2.0).

It is the root Go module `github.com/standards-lab/webappgen`, released by a plain `vX.Y.Z` tag
as binaries for Linux and macOS; its first release is v0.1.0. Its goals live in the coordinator's
roadmap (`standards-lab/roadmap/index.toml`, v1.client.stack), and its strategy is
`standards-lab/context/web-client.md`. Its CLI shape is copied from go-cli-sdk-template and its
starter app from spike-html-apps' app, with a provenance line on every copied file; the spike is
archived and never a dependency. [README.md](../README.md) documents the flags, the written
project's tasks, deployment, and the contributor workflow.

## Capability map

- **CLI** (built): [`internal/app`](../internal/app/): `version`, `new` with one leaf per kind
  of app, and `inject` with one leaf per kind it installs. Usage errors name the fix.
- **Standalone kind** (built): [`internal/standalone`](../internal/standalone/): `new standalone
  <dir>` writes the starter, then the standalone kind's own files over it (`index.html`,
  `vite.config.ts`, `public/config.json`). `--install` then runs `mise install` and
  `mise run install` in the project.
- **Service-app kind** (built): [`internal/serviceapp`](../internal/serviceapp/): `new
  service-app <module> <dir>` copies go-web-sdk-template with gonew, and `inject service-app
  <service-dir>` takes an existing copy; each writes the client, its Go package, and
  `internal/app/client.go`, and edits the service's wiring, `mise.toml`, CI, and `.air.toml`,
  printing the edits a differing service leaves. `--install` runs the install steps.
  [docs/service-app.md](../docs/service-app.md) details what is written.
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
- `internal/standalone/testdata/README.md.golden`: generated, the starter's `README.md.tmpl` as
  the standalone kind writes it for `my-app`, byte for byte.
