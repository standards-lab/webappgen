# webappgen

The web client CLI of the standards-lab web client stack. `webappgen new <kind>` writes a client
project for one kind of app, and `webappgen inject <kind>` installs a client into an existing
project, from files embedded in its binary, so one version names both the CLI and every file it
writes. The written project depends on the stack's layers,
[web-foundation](https://github.com/standards-lab/web-foundation),
[web-icons](https://github.com/standards-lab/web-icons),
[web-components](https://github.com/standards-lab/web-components),
[web-app-sdk](https://github.com/standards-lab/web-app-sdk), and
[web-tooling](https://github.com/standards-lab/web-tooling), each by its exact release tag, and
pins its toolchain with [mise](https://mise.jdx.dev).

## Install

With mise, from the GitHub release's binaries for linux and macOS on amd64 and arm64:

```sh
mise use github:standards-lab/webappgen
```

Or with Go:

```sh
go install github.com/standards-lab/webappgen/cmd/webappgen@latest
```

`webappgen version` prints the release the binary was built from.

On Windows, run webappgen and the project it writes in WSL: there is no Windows build yet, and the
written project's toolchain and browser tier are pinned for linux and macOS.

## Write a standalone app

```sh
webappgen new standalone my-app
cd my-app
mise install
mise run install
mise run check
```

`new standalone <dir>` writes a static web client, served on its own, into a new or empty
directory, and prints the steps left to run. Its flags:

| Flag | Default | What it sets |
|------|---------|--------------|
| `--name name` | the directory's name | the project's package name |
| `--title title` | the name | the page's title |
| `--base path` | `/` | the absolute path the app is served under, written as its `<base href>` |
| `--api-base URL` | `/api/` | the URL of the API the app calls, written to its `public/config.json` |
| `--install` | off | run `mise install` and `mise run install` in the project, leaving `mise run check` |

`--install` reads the network: it installs the project's pinned tools and its packages from
`pnpm-lock.yaml`, naming each step and its output on stderr. A missing mise or a step that fails
exits nonzero naming the step and the steps left, and the written project stays.

The written project's `mise run check` lints, type-checks, runs the unit tier, builds into
`dist/`, reports the build's size, and runs the browser tier on the build in the pinned
chrome-headless-shell. `mise run dev` serves the app from its source with Vite's dev server, and
`mise run preview` builds it and serves `dist/` with `vite preview`, each at Vite's default host
and port under the `--base` path; the check runs neither. `mise run currency` reports any tool,
package, or layer pin behind its latest; it holds typescript to typescript-eslint's range and
rolldown to vite's.

The written page is web-foundation's application page, laid out by its `layout.css`: a header
with the app's title and its actions, and a main column where the router mounts each view. The
header holds two buttons, drawn with web-icons' glyphs:

- **Help**, on the ? key, opens a dialog listing the page's keyboard shortcuts
  (web-components' `sl-shortcuts`).
- **Settings**, on the S key, opens a dialog to choose the color profile (zen, github,
  catppuccin, or everforest), the space profile (regular or compact), and the type profile (mono
  or sans) (web-components' `sl-settings`). A choice applies and is stored at once, and the stored
  choice applies before first paint on every later visit.

Each dialog closes on its Close button or Escape. The page ignores ? and S inside a text field.
The written project's own `README.md` documents its tasks, its config, deploying it, and how it is
built.

## Deploy a standalone build

`mise run build` writes the app to `dist/`, for any static host:

- Serve `dist/` under the `--base` path the project was written with; the build's
  `<base href>` and its asset URLs name that path.
- Replace the built `config.json` with the deployment's own: it names the API the app calls, as
  `{"apiBase": "<URL>"}`, and the app reads it at runtime, so one build serves every deployment.
- The host must answer `index.html` for any unknown path under the base, so a deep link or a
  reload reaches the app's router.

## Write a service app

```sh
webappgen new service-app example.com/my-service my-service
cd my-service
git init
mise install
go mod download
mise -C client install
mise -C client run install
mise run check
```

`new service-app <module> <dir>` copies
[go-web-sdk-template](https://github.com/standards-lab/go-web-sdk-template), at its pinned
version, into a new or empty directory as `<module>`, with `gonew`, then writes a web client into
it. `inject service-app <service-dir>` writes the client into an existing copy of the template.
Each prints the steps left, `inject`'s from `mise -C client install`. Both take these flags:

| Flag | Default | What it sets |
|------|---------|--------------|
| `--name name` | the directory's name | the client's package name |
| `--title title` | the name | the page's title |
| `--base path` | `/` | the absolute path the service serves the client under |
| `--api-base URL` | `/api/` | the URL of the API the client calls, written into the page's `#app-config` |
| `--client-dir path` | `client` | the client's directory in the service |
| `--install` | off | run the install steps, leaving `mise run check` |

`--install` runs the steps before `mise run check`, as the standalone kind's does.

Each writes the client, its Go package, `internal/app/client.go`, and `.air.toml`, and edits the
service's wiring, `mise.toml`, and CI to build the client. A service that
differs from the template where an edit goes keeps that file: `inject` prints the edits left,
then the next steps, and exits 1, running no install step.
[docs/service-app.md](docs/service-app.md) details each file and the printed edits.

In the service, `mise run check` builds and checks the client, then checks the service;
`mise run serve` builds the client, then runs the service; and `mise run dev` builds the client,
then rebuilds it on each change with `vite build --watch` while air rebuilds and restarts the
service.

## Deploy a service app

Build the client, then the service: the client's `mise run build` writes `dist/`, which the
service's build embeds, so a deployment ships the binary alone.
The service serves the client's shell, with its config, for any path under `--base` that isn't a
built file.

## Capabilities

- **CLI**: `version`, `new` with one leaf per kind of app, and `inject` with one leaf per kind
  that installs into an existing project; usage errors name the fix.
- **Standalone kind**: `new standalone <dir>` with `--name`, `--title`, `--base`, `--api-base`,
  and `--install`.
- **Service-app kind**: `new service-app <module> <dir>` and `inject service-app <service-dir>`,
  with `--name`, `--title`, `--base`, `--api-base`, `--client-dir`, and `--install`.
- **Written project**: the application page with Help and Settings, a home and a not-found
  view, its config read from `config.json` or `#app-config`, its own `README.md`, and a check
  that runs the unit and browser tiers on the build.

## Contributing

The repository is the Go module `github.com/standards-lab/webappgen` on
[go-cli-sdk](https://github.com/standards-lab/go-cli-sdk). Every tool is pinned in `mise.toml`:

```sh
mise run install   # the pins, the module's requirements, and the warmed pnpm store; reads the network
mise run check     # vet, lint, and the unit and integration tiers; reads no network
mise run currency  # every Go requirement, tool, and action pin behind its latest; reads the network
```

Only `mise run install` and `mise run currency` read the network. No other task installs a missing
tool; each names `mise run install` when one is missing. `mise run install` warms
`.cache/pnpm-store` with the starter's packages, so the integration tier's full proof writes a
project with `--install` and runs the project's own check offline. The module is built alone, with
`GOWORK=off`, never in a workspace a parent directory holds. Tests are black-box, in each package's
`_test` package: golangci-lint's testpackage allows only an `export_test.go`, and only to export a
clock or probe hook the API can't reach deterministically. nolintlint requires every `//nolint` to
name its linter and give its reason.

The root `mise.toml` pins every tool the starter's `internal/starter/files/mise.toml.tmpl` pins,
each entry as the starter writes it, and a unit test holds the two equal. Bump a shared tool in
both files at once; bumping chrome-headless-shell also takes each platform's sha256 and size,
taken by hand from its download, in both. A changed starter package regenerates
`internal/starter/files/pnpm-lock.yaml` with the pinned pnpm.

A release is a `vX.Y.Z` tag pushed on main's merge commit, with its `CHANGELOG.md` section; the
release workflow tags nothing. It refuses a tag whose commit isn't on main, checks, and builds
and packs the four binaries with `scripts/pack.sh`, which a clone at the tag also runs
(`scripts/pack.sh vX.Y.Z <outdir>`) and which refuses any binary whose version isn't the tag. It
creates the release as a draft from the tag's `CHANGELOG.md` section, attaches every archive and
`SHA256SUMS`, and only then publishes it, so a release appears only whole: a failed upload leaves
a draft. Runs for one tag are serialized and never cancelled, and each action is pinned by its
exact release tag, which `mise run currency` compares with the action's latest release.

## License

[Apache License 2.0](LICENSE).
