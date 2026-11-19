# webappgen standards

The judgement calls the standards-reviewer applies to webappgen, beyond what `mise run check`
enforces: each one line, with a pointer to the architecture page that applies. They come from
spike-html-apps (2883d78)'s `STANDARDS.md`, those that apply to a CLI and the project it writes;
the `standard` task points them at the Web Elemental standard once it is founded
(`standards-lab/context/web-client.md`, "Toward Web Elemental").

## Written project

- A written project depends on a layer only through its release tag, as any consumer does: never
  a link, path alias, source condition, or project reference
  (`architecture/principles/independent-releases.md`).
- The starter app sits above the layers and wires them; it imports no code a layer keeps
  internal, and a layer never knows the app (`architecture/principles/downward-dependencies.md`).
- The starter holds only what every app of its kind needs, and each package and tool it pins is
  deliberate (`architecture/principles/minimal-footprint.md`).
- Every tool, package, layer tag, and action pin, the CLI's and the written project's, is
  explicit and at its latest, but for a hold recorded beside its pin
  (`architecture/principles/rolling-currency.md`).
- The app reads its config and checks it before a view sees it, failing with the element or file
  it names (`architecture/principles/validation-first.md`).
- The app's entry is its composition root: it reads the config and wires the router over the
  page, and no view constructs what it uses (`architecture/principles/composition-root.md`).
- The app binds single keys only, for critical operations, never chords, and ignores them inside
  text fields; a component binds none (`architecture/principles/downward-dependencies.md`).

## Code

- A copied file names its source repository and commit in a provenance line, which reads
  `From|Adapted from <repo> (<commit>): <path>`, naming the immediate source alone; a JSON file,
  which takes no comment, has its line in `context/README.md`'s Provenance
  (`architecture/principles/comments.md`).
- A starter file keeps its provenance line in the project it is written into, so a written file
  names its source as the starter's does (`architecture/principles/comments.md`).
- Tests are TypeScript run under Node's type stripping, erasable syntax only, type-checked by
  `tsconfig.test.json`; browser-served fixtures stay `.js`
  (`architecture/principles/minimal-footprint.md`).
