# webappgen standards

The judgement calls the standards-reviewer applies to webappgen, beyond what `mise run check`
enforces: each one line, with a pointer to the architecture page that applies, for a CLI and the
project it writes. webappgen is a Web Elemental member that ships a Go binary
(`architecture/standards/web-elemental/README.md`).

## Written project

- A written project resolves each layer and pins its packages as a Web Elemental layer does
  (`architecture/standards/web-elemental/principles/packages-and-releases.md`).
- The starter app sits above the layers and wires them; it imports no code a layer keeps
  internal, and a layer never knows the app (`architecture/principles/downward-dependencies.md`).
- The starter holds only what every app of its kind needs, and each package and tool it pins is
  deliberate (`architecture/principles/minimal-footprint.md`).
- Every tool, package, layer tag, and action pin, the CLI's and the written project's, is
  explicit and at its latest, except a hold recorded beside its pin
  (`architecture/principles/rolling-currency.md`).
- The app reads its config and checks it before a view sees it, failing with the element or file
  it names (`architecture/principles/validation-first.md`).
- The app's entry is its composition root: it reads the config and wires the router over the
  page, and no view constructs what it uses (`architecture/principles/composition-root.md`).
- The app binds single keys only, for critical operations, never chords, and ignores them inside
  text fields; a component binds none.

## Go CLI

- The module depends only on go-core and go-cli-sdk, below it, and on no provider
  (`architecture/standards/go-elemental/principles/dependencies.md`).
- `internal/app`'s graph holds configuration as nodes, and each command builds only the nodes it
  declares with `Use`; a kind's leaf declares none
  (`architecture/standards/go-elemental/principles/cli-applications.md`).
- A leaf checks its arguments and flags in `Args` and `Validate`, which build nothing, and help
  and usage errors exit 2 (`architecture/standards/go-elemental/principles/cli-applications.md`).
- The module is rooted at the repository's top, with `cmd/webappgen` and `internal/app`'s layer
  files, and each kind of project in its own package
  (`architecture/standards/go-elemental/principles/topology-and-naming.md`).
- Every new exported function and constructor takes at most four required inputs and one
  trailing options struct, and the integration harness's `Run(t, args...)` takes the program's
  arguments (`architecture/standards/go-elemental/principles/construction-and-execution.md`).
- Package docs cover `cmd/webappgen`, `internal/app`, `apptest`, and each kind's package, and the
  integration suite runs the binary over `processtest`'s one-shot runner
  (`architecture/standards/go-elemental/principles/tests-and-docs.md`).
- The check and currency run from the root `mise.toml`, and a release is a plain `vX.Y.Z` tag on
  main's merge commit with its `CHANGELOG.md` section
  (`architecture/standards/go-elemental/principles/release-and-ci.md`).

## Code

- Every comment, the CLI's and the written project's, is one of the five kinds
  (`architecture/principles/comments.md`).
- Tests are TypeScript run under Node's type stripping, erasable syntax only, type-checked by
  `tsconfig.test.json`; browser-served fixtures stay `.js`.
