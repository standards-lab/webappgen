# webappgen context

webappgen is the web client CLI, a Go command on go-cli-sdk: `webappgen new <kind>` writes a
client project for one kind of app on the stack, from files embedded in its binary. It writes one
kind, `standalone`, a static app. Two embedded kinds, a client inside a new or an existing Go web
service served by go-web-sdk's `NewClient`, are planned for the goal's `embedded` task (v0.2.0).

It is the root Go module `github.com/standards-lab/webappgen`, released by a plain `vX.Y.Z` tag
as binaries for Linux and macOS; its first release is v0.1.0. Its goals live in the coordinator's
roadmap (`standards-lab/context/roadmap.toml`, v1.client.stack), and its strategy is
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
- **The written project** (built): [`internal/starter/files`](../internal/starter/files/), what
  `new standalone` writes.
  - It takes each layer by an exact `github:` release tag, with a committed lockfile.
  - It pins chrome-headless-shell for linux-x64, linux-arm64, mac-x64, and mac-arm64.
  - Its check runs lint, types, the unit tier, the build, and a size report with no budget, then
    the browser tier on web-tooling.
  - Its `currency` task holds typescript to typescript-eslint's range and rolldown to vite's.
  - It takes web-foundation's full entry, `page-full.css` and `register-full.js`, so its theme
    menu offers every built-in profile, and applies the stored theme before first paint, from a
    render-blocking Vite input.
  - Its header's Settings button, bound to S through the foundation's `keys.js`, opens the theme
    menu; `src/app.css` lays out the header and the main column.
  - Its `dev` and `preview` tasks serve the source and the build at Vite's defaults; no check
    runs them.
  - Its router mounts a home view and a not-found view.
  - Its config reader prefers the page's `#app-config` element and otherwise fetches
    `config.json`.
- **The full proof** (built): [`integration`](../integration/): the root `mise run install`
  warms `.cache/pnpm-store`, and the integration tier writes a project with `--install` offline
  and runs the written project's check.
- **Release** (built): [`release.yml`](../.github/workflows/release.yml): a tag on main
  triggers the workflow, which runs `scripts/pack.sh` to build four archives and `SHA256SUMS`,
  attaches them to a draft release, then publishes it.

## Facts the code relies on

- `src/theme.ts` is its own Vite input because Vite folds every module script `index.html`
  names into one injected entry and drops `blocking="render"`; a plugin in `vite.config.ts`
  writes the theme chunk's script tag ahead of the entry.
- `strictExecutionOrder` is on because the registry and the preferences module land in a chunk
  the theme menu shares; without it, that chunk applies the stored theme before the theme chunk
  registers the profiles the theme names.
- `src/theme.ts` imports `register-full.js` ahead of `preferences.js`: `preferences.js` drops a
  stored name no module has registered, so a stored non-default profile needs the full
  registration first, and the browser tier's first-paint case stores one.
- The foundation's tool sheets hold no layout class, and it leaves page layout to the page, so
  `src/app.css` holds the page's layout in a layer of its own, after the foundation's.
- The root `mise.toml`'s tools and the starter's `mise.toml.tmpl` move together: a unit test in
  `internal/starter` holds every starter pin equal to the root's, so a chrome-headless-shell bump
  changes both files' version, checksums, and sizes.
- The embedded trees under `internal/*/files` must hold only git-tracked files. `go:embed all:`
  takes whatever the working copy holds, such as `node_modules` or `dist` from a demo run, and
  each kind's `TestFiles_Tracked` fails on it.
- `scripts/pack.sh` builds and version-checks all four binaries before it writes any archive.
  An archive written first into an output directory inside the checkout is an untracked file,
  and `go build` stamps the next binary `+dirty`.

## Provenance

The JSON and generated files, which take no comment, have their provenance here; every other
copied file carries its own line.

- `internal/starter/files/.prettierrc.json`: from spike-html-apps (2883d78) `.prettierrc.json`,
  byte for byte.
- `internal/starter/files/package.json.tmpl`: adapted from spike-html-apps (2883d78)
  `package.json` and `test/consumers/app/package.json`, moved onto the standards-lab layers.
- `internal/standalone/files/public/config.json.tmpl`: new.
- `internal/starter/files/pnpm-lock.yaml`: produced by `pnpm install` (pnpm 12.21.0).
