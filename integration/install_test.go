//go:build integration

package integration_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/standards-lab/go-core/process"

	"github.com/standards-lab/webappgen/integration"
)

// `new standalone <dir> --install` as a running binary: the install it runs
// in the written project, the refusal when it can't, and the full proof,
// a written project installed offline whose own check passes.

// checkTimeout bounds the written project's own check: its lint, types,
// unit tier, build, and browser tier.
const checkTimeout = 5 * time.Minute

// With no mise on PATH, --install exits with the failure code, naming the
// step and the steps left to run, after the project is written; the
// written project stays, and no next steps are printed.
func TestNewStandalone_InstallWithoutMise(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-app")

	res := integration.RunEnv(t, []string{"PATH=" + t.TempDir()}, "new", "standalone", dir, "--install")

	if res.Code != process.ExitFailure {
		t.Errorf("code = %d, want %d; stderr = %q", res.Code, process.ExitFailure, res.Stderr)
	}
	want := "webappgen new standalone: mise install: mise isn't on PATH: install mise (https://mise.jdx.dev), " +
		"then run mise install and mise run install in " + dir + "\n"
	if res.Stderr != want {
		t.Errorf("stderr = %q, want %q", res.Stderr, want)
	}
	if res.Stdout != "" {
		t.Errorf("stdout = %q, want empty", res.Stdout)
	}
	if got := integration.Tree(t, dir); !slices.Equal(got, written) {
		t.Errorf("tree = %q, want the written project, %q", got, written)
	}
}

// offline is the environment that holds the full proof to no network: mise
// installs only the pins it has, which the CLI's own mise install installs
// (the starter pins only what the root pins), and pnpm installs only from
// the store the CLI's mise run install fills from the starter's lockfile,
// failing on a package it lacks rather than downloading it.
func offline(t *testing.T) []string {
	t.Helper()
	store, err := filepath.Abs(filepath.Join("..", ".cache", "pnpm-store"))
	if err != nil {
		t.Fatal(err)
	}
	return []string{"MISE_OFFLINE=1", "pnpm_config_offline=true", "pnpm_config_store_dir=" + store}
}

// The full proof: --install installs a written project, offline, and the
// project's own check, mise run check, passes in it, its browser tier
// loading the app under the base given. The steps' progress goes to
// stderr; stdout is the one step left.
func TestNewStandalone_InstallThenCheck(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "x")
	env := offline(t)

	res := integration.RunEnv(t, env, "new", "standalone", dir, "--install", "--base", "/app/")

	if res.Code != process.ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", res.Code, process.ExitOK, res.Stderr)
	}
	if want := "Next steps:\n  cd " + dir + "\n  mise run check\n"; res.Stdout != want {
		t.Errorf("stdout = %q, want %q", res.Stdout, want)
	}
	for _, step := range []string{"mise install", "mise run install"} {
		if want := "Running " + step + " in " + dir + "\n"; !strings.Contains(res.Stderr, want) {
			t.Errorf("stderr = %q, want it to name %q", res.Stderr, want)
		}
	}
	for _, layer := range []string{"web-foundation", "web-icons", "web-components", "web-app-sdk", "web-tooling"} {
		if _, err := os.Stat(filepath.Join(dir, "node_modules", "@standards-lab", layer, "package.json")); err != nil {
			t.Errorf("--install didn't install @standards-lab/%s: %v", layer, err)
		}
	}

	ctx, cancel := context.WithTimeout(t.Context(), checkTimeout)
	defer cancel()
	check := exec.CommandContext(ctx, "mise", "run", "check")
	check.Dir, check.Env = dir, append(os.Environ(), env...)
	out, err := check.CombinedOutput()
	if err != nil {
		t.Fatalf("mise run check in the written project: %v\n%s", err, out)
	}
	t.Logf("$ mise run check\n%s", out)
}
