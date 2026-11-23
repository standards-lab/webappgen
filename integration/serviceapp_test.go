//go:build integration

package integration_test

import (
	"context"
	"os"
	"os/exec"
	"path"
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

var writtenClient = []string{
	".gitignore",
	".prettierignore",
	".prettierrc.json",
	"README.md",
	"client.go",
	"eslint.config.js",
	"mise.toml",
	"package.json",
	"pnpm-lock.yaml",
	"pnpm-workspace.yaml",
	"scripts",
	"scripts/currency.sh",
	"scripts/preflight.sh",
	"scripts/size.ts",
	"src",
	"src/config.ts",
	"src/main.ts",
	"src/page.ts",
	"src/theme.ts",
	"src/views",
	"src/views/home.ts",
	"src/views/not-found.ts",
	"test",
	"test/browser",
	"test/browser/app.test.ts",
	"test/browser/dialogs.test.ts",
	"test/browser/glyphs.test.ts",
	"test/browser/support",
	"test/browser/support/app.ts",
	"test/browser/support/build.ts",
	"test/browser/theme.test.ts",
	"test/unit",
	"test/unit/config.test.ts",
	"test/unit/size.test.ts",
	"tsconfig.json",
	"tsconfig.test.json",
	"vite.config.ts",
}

func TestNewServiceApp_WritesTheClient(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-service")

	res := integration.RunEnv(t, goOffline(t), "new", "service-app", "example.com/demo", dir,
		"--client-dir", "web/My-UI", "--name", "console", "--title", `Ops & "Console"`,
		"--api-base", "https://api.example.com/v1/")

	if res.Code != process.ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", res.Code, process.ExitOK, res.Stderr)
	}
	want := "Next steps:\n  cd " + dir + "\n  git init\n  mise install\n" +
		"  mise -C web/My-UI install\n  mise -C web/My-UI run install\n  mise -C web/My-UI run build\n" +
		"  mise run check\n"
	if res.Stdout != want {
		t.Errorf("stdout = %q, want %q", res.Stdout, want)
	}
	client := filepath.Join(dir, "web", "My-UI")
	if got := integration.Tree(t, client); !slices.Equal(got, writtenClient) {
		t.Errorf("client tree = %q, want %q", got, writtenClient)
	}
	for name, wants := range map[string][]string{
		"package.json": {`"name": "console",`},
		"client.go":    {"\npackage myui\n", `const Entry = "src/main.ts"`, `var RenderBlocking = []string{"src/theme.ts", "src/page.ts"}`, "//go:embed all:dist\n"},
		"src/page.ts":  {`const title = "Ops &amp; &#34;Console&#34;";`},
		"vite.config.ts": {`base: "./",`, "publicDir: false,", "manifest: true,",
			`input: { main: "src/main.ts", theme: "src/theme.ts", page: "src/page.ts" },`},
		"README.md":  {"# console\n", "`https://api.example.com/v1/`", "`mise run serve`"},
		".gitignore": {"\ndist/\n"},
	} {
		content := read(t, client, name)
		for _, want := range wants {
			if !strings.Contains(content, want) {
				t.Errorf("%s = %q, want it to hold %q", name, content, want)
			}
		}
	}
	config := read(t, client, "mise.toml")
	for _, task := range []string{"[tasks.dev]", "[tasks.preview]"} {
		if strings.Contains(config, task) {
			t.Errorf("mise.toml holds %s, want no dev server or preview task", task)
		}
	}
	for _, name := range writtenClient {
		if path.Ext(name) == "" {
			continue
		}
		content := strings.ReplaceAll(read(t, client, name), "{{ version }}", "")
		if strings.Contains(content, "{{") {
			t.Errorf("%s holds an unrendered {{", name)
		}
	}
}

func TestNewServiceApp_WritesThenChecks(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-service")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	env := append(goOffline(t), offline(t)...)

	res := integration.RunEnv(t, env, "new", "service-app", "example.com/demo", dir)

	if res.Code != process.ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", res.Code, process.ExitOK, res.Stderr)
	}
	want := "Next steps:\n  cd " + dir + "\n  git init\n  mise install\n" +
		"  mise -C client install\n  mise -C client run install\n  mise -C client run build\n" +
		"  mise run check\n"
	if res.Stdout != want {
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
	client := filepath.Join(dir, "client")
	for _, args := range [][]string{{"run", "install"}, {"run", "check"}} {
		cmd := exec.CommandContext(ctx, "mise", args...)
		cmd.Dir, cmd.Env = client, append(os.Environ(), env...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("mise %s in the written client: %v\n%s", strings.Join(args, " "), err, out)
		}
		t.Logf("$ mise %s\n%s", strings.Join(args, " "), out)
	}
	if _, err := os.Stat(filepath.Join(client, "dist", ".vite", "manifest.json")); err != nil {
		t.Fatalf("the client's check built no manifest: %v", err)
	}
	for _, args := range [][]string{{"vet", "./..."}, {"test", "-count=1", "./..."}} {
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Dir, cmd.Env = dir, append(os.Environ(), env...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go %s in the written service: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
}
