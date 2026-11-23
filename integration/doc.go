// Package integration is webappgen's integration tier: the harness that
// runs the program as the built binary, one shot at a time, and, under the
// integration build tag, the suite that asserts its behavior through the
// interfaces a terminal uses: its arguments, its two output streams, its
// exit code, and the tree it writes. The harness is built on go-core's
// processtest; nothing in the runtime exists for the tests' sake.
//
//   - [Main] is the suite's TestMain: it clears the WEBAPPGEN_ environment
//     and builds cmd/webappgen once per run.
//   - [Run] runs the built program once with its arguments and returns its
//     processtest.Result.
//   - [RunEnv] runs it once with its arguments and environment overrides.
//   - [Tree] lists a directory's paths, for a test asserting what a run
//     wrote there, or that it wrote nothing.
//
// The harness is untagged, so the unit tier type-checks it on every pull
// request; the suite files carry the integration tag.
//
// The full proof writes a project with --install and runs its own check,
// with no network: mise installs only the pins it has, which webappgen's
// own mise install installs, as the starter pins only what the root pins;
// and pnpm installs only from the store webappgen's mise run install fills
// from the starter's lockfile, failing on a package it lacks rather than
// downloading it.
package integration
