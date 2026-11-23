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

	"github.com/standards-lab/go-core/process"

	"github.com/standards-lab/webappgen/integration"
)

func TestNewServiceApp_Help(t *testing.T) {
	res := integration.Run(t, "new", "service-app", "--help")

	if res.Code != process.ExitUsage {
		t.Errorf("code = %d, want %d; stderr = %q", res.Code, process.ExitUsage, res.Stderr)
	}
	for _, want := range []string{
		"webappgen new service-app [flags] <module> <dir>",
		"--name name",
		"--title title",
		"--base path", "(default /)",
		"--api-base URL", "(default /api/)",
		"--client-dir path", "(default client)",
	} {
		if !strings.Contains(res.Stdout, want) {
			t.Errorf("stdout = %q, want it to hold %q", res.Stdout, want)
		}
	}
}

func TestNewServiceApp_UsageErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		holding []string
		want    string
	}{
		{
			name: "missing both",
			args: nil,
			want: "missing the module path and project directory: give both, as in 'webappgen new service-app example.com/my-service my-service'",
		},
		{
			name: "missing dir",
			args: []string{"example.com/demo"},
			want: "missing the project directory: give one after the module path, as in 'webappgen new service-app example.com/demo my-service'",
		},
		{
			name: "three arguments",
			args: []string{"example.com/demo", "{dir}", "other"},
			want: "takes a module path and a project directory, got 3 arguments: quote a path that holds spaces",
		},
		{
			name: "module without a dot",
			args: []string{"demo", "{dir}"},
			want: `module path "demo" has no dot in its first element: begin it with a domain, such as example.com/demo`,
		},
		{
			name:    "dir not empty",
			args:    []string{"example.com/demo", "{dir}"},
			holding: []string{"README.md"},
			want:    "{dir} exists and isn't empty: give a new or empty directory",
		},
		{
			name: "unknown flag",
			args: []string{"example.com/demo", "{dir}", "--install"},
			want: "flag provided but not defined: -install",
		},
		{
			name: "base not absolute",
			args: []string{"example.com/demo", "{dir}", "--base", "app/"},
			want: `--base "app/" doesn't begin with "/": give an absolute path, such as /app/`,
		},
		{
			name: "client dir above the root",
			args: []string{"--client-dir", "../client", "example.com/demo", "{dir}"},
			want: `--client-dir "../client" reaches above the service's root: give a directory inside it, such as client`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "my-service")
			if tc.holding != nil {
				if err := os.Mkdir(dir, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			for _, name := range tc.holding {
				if err := os.WriteFile(filepath.Join(dir, name), []byte("kept\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			before := integration.Tree(t, filepath.Dir(dir))
			args := []string{"new", "service-app"}
			for _, arg := range tc.args {
				args = append(args, strings.ReplaceAll(arg, "{dir}", dir))
			}

			res := integration.Run(t, args...)

			if res.Code != process.ExitUsage {
				t.Errorf("code = %d, want %d; stderr = %q", res.Code, process.ExitUsage, res.Stderr)
			}
			want := "webappgen new service-app: " + strings.ReplaceAll(tc.want, "{dir}", dir) + "\n" +
				"Usage: webappgen new service-app [flags] <module> <dir>\n" +
				"Run 'webappgen new service-app --help' for details.\n"
			if res.Stderr != want {
				t.Errorf("stderr = %q, want %q", res.Stderr, want)
			}
			if res.Stdout != "" {
				t.Errorf("stdout = %q, want empty", res.Stdout)
			}
			if after := integration.Tree(t, filepath.Dir(dir)); !slices.Equal(after, before) {
				t.Errorf("tree after = %q, want it as before, %q", after, before)
			}
		})
	}
}

func TestNewServiceApp_WithoutGo(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-service")

	res := integration.RunEnv(t, []string{"PATH=" + t.TempDir()}, "new", "service-app", "example.com/demo", dir)

	if res.Code != process.ExitFailure {
		t.Errorf("code = %d, want %d; stderr = %q", res.Code, process.ExitFailure, res.Stderr)
	}
	want := "webappgen new service-app: go isn't on PATH: install Go (https://go.dev/dl), then run " +
		"go run golang.org/x/tools/cmd/gonew@v0.53.0 github.com/standards-lab/go-web-sdk-template/template@v0.16.0 " +
		"example.com/demo " + dir + "\n"
	if res.Stderr != want {
		t.Errorf("stderr = %q, want %q", res.Stderr, want)
	}
	if res.Stdout != "" {
		t.Errorf("stdout = %q, want empty", res.Stdout)
	}
	if got := integration.Tree(t, dir); got != nil {
		t.Errorf("tree = %q, want no directory", got)
	}
}

func goOffline(t *testing.T) []string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Fatalf("go env GOMODCACHE: %v", err)
	}
	cache := filepath.Join(strings.TrimSpace(string(out)), "cache", "download")
	return []string{"GOPROXY=file://" + filepath.ToSlash(cache)}
}

func TestNewServiceApp_WritesThenTests(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-service")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	env := goOffline(t)

	res := integration.RunEnv(t, env, "new", "service-app", "example.com/demo", dir)

	if res.Code != process.ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", res.Code, process.ExitOK, res.Stderr)
	}
	if want := "Next steps:\n  cd " + dir + "\n  git init\n  mise install\n  mise run check\n"; res.Stdout != want {
		t.Errorf("stdout = %q, want %q", res.Stdout, want)
	}
	if want := "gonew: initialized example.com/demo in " + dir + "\n"; !strings.HasSuffix(res.Stderr, want) {
		t.Errorf("stderr = %q, want it to end with %q", res.Stderr, want)
	}
	if got := read(t, dir, "go.mod"); !strings.HasPrefix(got, "module example.com/demo\n") {
		t.Errorf("go.mod = %q, want it to declare module example.com/demo", got)
	}

	ctx, cancel := context.WithTimeout(t.Context(), checkTimeout)
	defer cancel()
	for _, args := range [][]string{{"vet", "./..."}, {"test", "-count=1", "./..."}} {
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Dir, cmd.Env = dir, append(os.Environ(), env...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go %s in the written service: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
}
