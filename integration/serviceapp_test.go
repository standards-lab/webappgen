//go:build integration

package integration_test

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/standards-lab/go-core/process"

	"github.com/standards-lab/webappgen/integration"
	"github.com/standards-lab/webappgen/internal/serviceapp"
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
		"--install", "then run mise install and go mod download in the service, and mise install and mise run install in the client",
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
			args: []string{"example.com/demo", "{dir}", "--watch"},
			want: "flag provided but not defined: -watch",
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
	want := "Next steps:\n  cd " + dir + "\n  git init\n  mise install\n  go mod download\n" +
		"  mise -C web/My-UI install\n  mise -C web/My-UI run install\n" +
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
	for name, wants := range map[string][]string{
		"mise.toml": {serviceapp.AirTool, `run = "mise -C web/My-UI run build"`, `run = "mise -C web/My-UI run check"`,
			"(mise -C web/My-UI exec -- pnpm exec vite build --watch) &\n"},
		".github/workflows/ci.yml": {"          mise -C web/My-UI install\n", "          mise -C web/My-UI run build\n"},
		".air.toml":                {`"web/My-UI/node_modules", "web/My-UI/src", "web/My-UI/test", "web/My-UI/scripts"`},
	} {
		content := read(t, dir, name)
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

	res := integration.RunEnv(t, env, "new", "service-app", "example.com/demo", dir, "--install")

	if res.Code != process.ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", res.Code, process.ExitOK, res.Stderr)
	}
	want := "Next steps:\n  cd " + dir + "\n  git init\n  mise run check\n"
	if res.Stdout != want {
		t.Errorf("stdout = %q, want %q", res.Stdout, want)
	}
	wantSteps(t, res.Stderr, dir, "gonew: initialized example.com/demo in "+dir,
		"mise install", "go mod download", "mise -C client install", "mise -C client run install")
	if got := read(t, dir, "go.mod"); !strings.HasPrefix(got, "module example.com/demo\n") {
		t.Errorf("go.mod = %q, want it to declare module example.com/demo", got)
	}

	ctx, cancel := context.WithTimeout(t.Context(), checkTimeout)
	defer cancel()
	client := filepath.Join(dir, "client")
	runIn(ctx, t, client, env, "mise", "run", "check")
	if _, err := os.Stat(filepath.Join(client, "dist", ".vite", "manifest.json")); err != nil {
		t.Fatalf("the client's check built no manifest: %v", err)
	}
	must(t, os.RemoveAll(filepath.Join(client, "dist")))
	testService(ctx, t, dir, client, env)
	runIn(ctx, t, dir, env, "go", "vet", "./...")

	origin := serve(ctx, t, dir, env)
	shell := wantShell(t, origin, "/", "/api/")
	wantAsset(t, origin, "/", shell)
	if body := wantStatus(t, origin+"/settings/x", http.StatusOK); body != shell {
		t.Errorf("GET /settings/x = %q, want the shell", body)
	}
	wantStatus(t, origin+"/healthz", http.StatusOK)
	if body := wantStatus(t, origin+"/api/x", http.StatusNotFound); body == shell {
		t.Errorf("GET /api/x = the shell, want the API's answer")
	}
}

// wantSteps checks that stderr names each of steps as run in dir, in order,
// after the line first.
func wantSteps(t *testing.T, stderr, dir, first string, steps ...string) {
	t.Helper()
	rest := stderr
	for _, line := range append([]string{first}, steps...) {
		if line != first {
			line = "Running " + line + " in " + dir
		}
		_, after, ok := strings.Cut(rest, line+"\n")
		if !ok {
			t.Errorf("stderr = %q, want it to hold %q after the lines before it", stderr, line)
			return
		}
		rest = after
	}
}

// fakeMise returns a PATH of the go running the tests and a mise that
// succeeds but for the step whose arguments are failArgs, which exits 3.
func fakeMise(t *testing.T, failArgs string) string {
	t.Helper()
	bin := t.TempDir()
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	must(t, os.Symlink(goBin, filepath.Join(bin, "go")))
	script := "#!/bin/sh\nif [ \"$*\" = '" + failArgs + "' ]; then echo \"mise $*: failed\"; exit 3; fi\n"
	must(t, os.WriteFile(filepath.Join(bin, "mise"), []byte(script), 0o755))
	return "PATH=" + bin
}

func TestNewServiceApp_InstallStepFails(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-service")
	env := append(goOffline(t), fakeMise(t, "-C web/ui install"))

	res := integration.RunEnv(t, env, "new", "service-app", "example.com/demo", dir, "--client-dir", "web/ui", "--install")

	if res.Code != process.ExitFailure {
		t.Fatalf("code = %d, want %d; stderr = %q", res.Code, process.ExitFailure, res.Stderr)
	}
	wantErr := "webappgen new service-app: mise -C web/ui install: exit status 3: fix what it reports, " +
		"then run mise -C web/ui install and mise -C web/ui run install in " + dir + "\n"
	if !strings.HasSuffix(res.Stderr, wantErr) {
		t.Errorf("stderr = %q, want it to end with %q", res.Stderr, wantErr)
	}
	wantSteps(t, res.Stderr, dir, "gonew: initialized example.com/demo in "+dir,
		"mise install", "go mod download", "mise -C web/ui install")
	if strings.Contains(res.Stderr, "Running mise -C web/ui run install") {
		t.Errorf("stderr = %q, want no step run after the one that failed", res.Stderr)
	}
	if res.Stdout != "" {
		t.Errorf("stdout = %q, want empty", res.Stdout)
	}
	if got := integration.Tree(t, filepath.Join(dir, "web", "ui")); !slices.Equal(got, writtenClient) {
		t.Errorf("client tree = %q, want the written client, %q", got, writtenClient)
	}
}

func TestNewServiceApp_ServesUnderABase(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-service")
	env := append(goOffline(t), offline(t)...)

	res := integration.RunEnv(t, env, "new", "service-app", "example.com/demo", dir,
		"--base", "/app/", "--client-dir", "web/ui", "--api-base", "https://api.example.com/v1/")

	if res.Code != process.ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", res.Code, process.ExitOK, res.Stderr)
	}
	ctx, cancel := context.WithTimeout(t.Context(), checkTimeout)
	defer cancel()
	client := filepath.Join(dir, "web", "ui")
	runIn(ctx, t, client, env, "mise", "run", "install")
	testService(ctx, t, dir, client, env)
	runIn(ctx, t, dir, env, "go", "vet", "./...")

	origin := serve(ctx, t, dir, env)
	shell := wantShell(t, origin, "/app/", "https://api.example.com/v1/")
	wantAsset(t, origin, "/app/", shell)
	if body := wantStatus(t, origin+"/app/settings/x", http.StatusOK); body != shell {
		t.Errorf("GET /app/settings/x = %q, want the shell", body)
	}
	wantStatus(t, origin+"/healthz", http.StatusOK)
	wantStatus(t, origin+"/readyz", http.StatusOK)
	for _, p := range []string{"/api/x", "/settings/x", "/"} {
		if body := wantStatus(t, origin+p, http.StatusNotFound); body == shell {
			t.Errorf("GET %s = the shell, want the service's answer outside the base", p)
		}
	}
}

func testService(ctx context.Context, t *testing.T, dir, client string, env []string) {
	t.Helper()
	runIn(ctx, t, dir, env, "mise", "run", "test")
	if _, err := os.Stat(filepath.Join(client, "dist", ".vite", "manifest.json")); err != nil {
		t.Fatalf("the service's mise run test built no client manifest: %v", err)
	}
}

func runIn(ctx context.Context, t *testing.T, dir string, env []string, name string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir, cmd.Env = dir, append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s in %s: %v\n%s", name, strings.Join(args, " "), dir, err, out)
	}
}

var readyAddr = regexp.MustCompile(`msg="server ready" addr=(\S+)`)

func serve(ctx context.Context, t *testing.T, dir string, env []string) string {
	t.Helper()
	server := filepath.Join(t.TempDir(), "server")
	runIn(ctx, t, dir, env, "go", "build", "-o", server, "./cmd/server")
	cmd := exec.CommandContext(ctx, server)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "APP_ENV=local", "APP_SERVER_PORT=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start the service: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Signal(os.Interrupt)
		_ = cmd.Wait()
		if t.Failed() {
			t.Logf("the service's stderr:\n%s", stderr.String())
		}
	})
	addr := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if m := readyAddr.FindStringSubmatch(scanner.Text()); m != nil {
				addr <- m[1]
			}
		}
		close(addr)
	}()
	select {
	case a, ok := <-addr:
		if !ok {
			t.Fatal("the service exited before it logged server ready")
		}
		return "http://" + a
	case <-time.After(time.Minute):
		t.Fatal("the service logged no server ready within a minute")
	}
	return ""
}

func wantStatus(t *testing.T, url string, status int) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	if resp.StatusCode != status {
		t.Errorf("GET %s = %d, want %d; body = %q", url, resp.StatusCode, status, body)
	}
	return string(body)
}

func wantShell(t *testing.T, origin, base, apiBase string) string {
	t.Helper()
	shell := wantStatus(t, origin+base, http.StatusOK)
	for _, want := range []string{
		`<base href="` + base + `">`,
		`<script type="application/json" id="app-config">{"apiBase":"` + apiBase + `"}</script>`,
	} {
		if !strings.Contains(shell, want) {
			t.Errorf("GET %s = %q, want it to hold %q", base, shell, want)
		}
	}
	return shell
}

var entryScript = regexp.MustCompile(`<script type="module" src="([^"]+)"></script>`)

func wantAsset(t *testing.T, origin, base, shell string) {
	t.Helper()
	m := entryScript.FindStringSubmatch(shell)
	if m == nil {
		t.Fatalf("the shell %q names no entry script", shell)
	}
	asset := wantStatus(t, origin+path.Join(base, m[1]), http.StatusOK)
	if asset == shell || asset == "" {
		t.Errorf("GET %s = %q, want the entry module", path.Join(base, m[1]), asset)
	}
}
