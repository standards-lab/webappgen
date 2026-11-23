# webappgen

The web client CLI of the standards-lab web client stack. `webappgen new <kind>` writes a client
project for one kind of app, from files embedded in its binary, so one version names both the CLI
and every file it writes. The written project depends on the stack's layers,
[web-foundation](https://github.com/standards-lab/web-foundation),
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

The app's page takes web-foundation's full entry, so its Settings menu, opened by the header's
button or the S key, offers every built-in color, space, and type profile, and the reader's
stored choice applies before first paint.

## Deploy a standalone build

`mise run build` writes the app to `dist/`, for any static host:

- Serve `dist/` under the `--base` path the project was written with; the build's
  `<base href>` and its asset URLs name that path.
- Replace the built `config.json` with the deployment's own: it names the API the app calls, as
  `{"apiBase": "<URL>"}`, and the app reads it at runtime, so one build serves every deployment.
- The host must answer `index.html` for any unknown path under the base, so a deep link or a
  reload reaches the app's router.

## Capabilities

- **CLI**: `version`, and `new` with one leaf per kind of app; usage errors name the fix.
- **Standalone kind**: `new standalone <dir>` with `--name`, `--title`, `--base`, `--api-base`,
  and `--install`.

## Contributing

The repository is the Go module `github.com/standards-lab/webappgen` on
[go-cli-sdk](https://github.com/standards-lab/go-cli-sdk). Every tool is pinned in `mise.toml`:

```sh
mise run install   # the pins, the module's requirements, and the warmed pnpm store; reads the network
mise run check     # vet, lint, and the unit and integration tiers; reads no network
mise run currency  # every Go requirement, tool, and action pin behind its latest; reads the network
```

`mise run install` is the one task that reads the network; no other task installs a missing tool,
and each names `mise run install` when one is missing. It warms `.cache/pnpm-store` with the
starter's packages, so the integration tier's full proof writes a project with `--install` and
runs the project's own check offline. The module is built alone, with `GOWORK=off`, never in a
workspace a parent directory holds. Tests are black-box, in each package's `_test` package:
golangci-lint's testpackage allows only an `export_test.go`, and only to export a clock or probe
hook the API can't reach deterministically. nolintlint requires every `//nolint` to name its
linter and give its reason.

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
