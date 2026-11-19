# webappgen context

webappgen is the web client CLI, a Go command on go-cli-sdk: `webappgen new <kind>` writes a
client project for one kind of app on the stack, from files embedded in its binary. Its first kind,
`standalone`, writes a static app; the embedded kinds, a client inside a new or an existing Go web
service served by go-web-sdk's `NewClient`, follow in the goal's `embedded` task. It was named
web-app-sdk-template until the `template` task's plan made it a CLI.

It is the root Go module `github.com/standards-lab/webappgen`, starting at v0.1.0 and released by
a plain `vX.Y.Z` tag as binaries for Linux and macOS. Its goals live in the coordinator's roadmap
(`standards-lab/context/roadmap.toml`, v1.client.stack), and its strategy is
`standards-lab/context/web-client.md`. Its CLI shape is copied from go-cli-sdk-template and its
starter app from spike-html-apps' app, with a provenance line on every copied file; the spike is
archived and never a dependency.

## Capability map

- **CLI**: `version`, and `new` with one leaf per kind of app; usage errors name the fix.
- **Standalone kind**: `new standalone <dir>` with `--name`, `--title`, `--base`, `--api-base`,
  and `--install`.

## Provenance

The JSON and generated files, which take no comment, have their provenance here; every other
copied file carries its own line.

- `internal/starter/files/.prettierrc.json`: from spike-html-apps (2883d78) `.prettierrc.json`,
  byte for byte.
- `internal/starter/files/package.json.tmpl`: adapted from spike-html-apps (2883d78)
  `package.json` and `test/consumers/app/package.json`, moved onto the standards-lab layers.
- `internal/standalone/files/public/config.json.tmpl`: new.
- `internal/starter/files/pnpm-lock.yaml`: produced by `pnpm install` (pnpm 12.21.0).
