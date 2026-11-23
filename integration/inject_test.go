//go:build integration

package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/process"

	"github.com/standards-lab/webappgen/integration"
	"github.com/standards-lab/webappgen/internal/serviceapp"
)

func TestInjectServiceApp_Help(t *testing.T) {
	res := integration.Run(t, "inject", "service-app", "--help")

	if res.Code != process.ExitUsage {
		t.Errorf("code = %d, want %d; stderr = %q", res.Code, process.ExitUsage, res.Stderr)
	}
	for _, want := range []string{
		"webappgen inject service-app [flags] <service-dir>",
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
	if strings.Contains(res.Stdout, "--install") {
		t.Errorf("stdout = %q, want no --install", res.Stdout)
	}
}

func fixtureService(t *testing.T, dir string) {
	t.Helper()
	if err := os.CopyFS(dir, os.DirFS(filepath.Join("..", "internal", "serviceapp", "testdata", "wire", "template"))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInjectServiceApp_UsageErrors(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		setup func(t *testing.T, dir string)
		want  string
	}{
		{
			name: "missing dir",
			args: nil,
			want: "missing the service directory: give one, as in 'webappgen inject service-app my-service'",
		},
		{
			name: "two arguments",
			args: []string{"{dir}", "other"},
			want: "takes a service directory, got 2 arguments: quote a path that holds spaces",
		},
		{
			name:  "dir missing",
			args:  []string{"{dir}"},
			setup: func(t *testing.T, dir string) { must(t, os.RemoveAll(dir)) },
			want:  "stat {dir}: no such file or directory: give the directory of a Go web service",
		},
		{
			name: "dir is a file",
			args: []string{"{dir}"},
			setup: func(t *testing.T, dir string) {
				must(t, os.RemoveAll(dir))
				must(t, os.WriteFile(dir, []byte("kept\n"), 0o644))
			},
			want: "{dir} isn't a directory: give the directory of a Go web service",
		},
		{
			name:  "no go.mod",
			args:  []string{"{dir}"},
			setup: func(t *testing.T, dir string) { must(t, os.Remove(filepath.Join(dir, "go.mod"))) },
			want:  "{dir} has no go.mod: give the root of a Go module, such as a service copied from go-web-sdk-template",
		},
		{
			name: "go.mod declares no module",
			args: []string{"{dir}"},
			setup: func(t *testing.T, dir string) {
				must(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("go 1.27\n"), 0o644))
			},
			want: "{dir}/go.mod declares no module: give the root of a Go module, its go.mod beginning with a module directive",
		},
		{
			name:  "client dir exists",
			args:  []string{"{dir}"},
			setup: func(t *testing.T, dir string) { must(t, os.Mkdir(filepath.Join(dir, "client"), 0o755)) },
			want:  `--client-dir "client": {dir}/client exists: give another --client-dir, or remove it`,
		},
		{
			name:  "nested client dir exists",
			args:  []string{"{dir}", "--client-dir", "web/ui"},
			setup: func(t *testing.T, dir string) { must(t, os.MkdirAll(filepath.Join(dir, "web", "ui"), 0o755)) },
			want:  `--client-dir "web/ui": {dir}/web/ui exists: give another --client-dir, or remove it`,
		},
		{
			name: "client file exists",
			args: []string{"{dir}"},
			setup: func(t *testing.T, dir string) {
				must(t, os.WriteFile(filepath.Join(dir, "internal", "app", "client.go"), []byte("package app\n"), 0o644))
			},
			want: "internal/app/client.go exists: remove it, or wire the client by hand",
		},
		{
			name: "declares a client identifier",
			args: []string{"{dir}"},
			setup: func(t *testing.T, dir string) {
				must(t, os.WriteFile(filepath.Join(dir, "internal", "app", "dist.go"), []byte("package app\n\nfunc newDist() {}\n"), 0o644))
			},
			want: "internal/app already declares newDist, which client.go declares: rename it",
		},
		{
			name:  "no composition root",
			args:  []string{"{dir}"},
			setup: func(t *testing.T, dir string) { must(t, os.RemoveAll(filepath.Join(dir, "internal"))) },
			want: "internal/app: open {dir}/internal/app: no such file or directory: " +
				"wire the client into a service copied from go-web-sdk-template",
		},
		{
			name: "unknown flag",
			args: []string{"{dir}", "--install"},
			want: "flag provided but not defined: -install",
		},
		{
			name: "base not absolute",
			args: []string{"{dir}", "--base", "app/"},
			want: `--base "app/" doesn't begin with "/": give an absolute path, such as /app/`,
		},
		{
			name: "client dir above the root",
			args: []string{"--client-dir", "../client", "{dir}"},
			want: `--client-dir "../client" reaches above the service's root: give a directory inside it, such as client`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "my-service")
			fixtureService(t, dir)
			if tc.setup != nil {
				tc.setup(t, dir)
			}
			before := snapshot(t, filepath.Dir(dir))
			args := []string{"inject", "service-app"}
			for _, arg := range tc.args {
				args = append(args, strings.ReplaceAll(arg, "{dir}", dir))
			}

			res := integration.Run(t, args...)

			if res.Code != process.ExitUsage {
				t.Errorf("code = %d, want %d; stderr = %q", res.Code, process.ExitUsage, res.Stderr)
			}
			want := "webappgen inject service-app: " + strings.ReplaceAll(tc.want, "{dir}", dir) + "\n" +
				"Usage: webappgen inject service-app [flags] <service-dir>\n" +
				"Run 'webappgen inject service-app --help' for details.\n"
			if res.Stderr != want {
				t.Errorf("stderr = %q, want %q", res.Stderr, want)
			}
			if res.Stdout != "" {
				t.Errorf("stdout = %q, want empty", res.Stdout)
			}
			if after := snapshot(t, filepath.Dir(dir)); !slices.Equal(after, before) {
				t.Errorf("tree after = %q, want it as before, %q", after, before)
			}
		})
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func snapshot(t *testing.T, root string) []string {
	t.Helper()
	var entries []string
	for _, name := range integration.Tree(t, root) {
		p := filepath.Join(root, filepath.FromSlash(name))
		info, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if info.IsDir() {
			entries = append(entries, name+"/")
			continue
		}
		entries = append(entries, name+"="+read(t, root, name))
	}
	return entries
}

func copyTemplate(t *testing.T, env []string, dir string) {
	t.Helper()
	tools, version, _ := strings.Cut(serviceapp.ToolsPin, "@")
	runIn(t.Context(), t, filepath.Dir(dir), env, "go", "run", tools+"/cmd/gonew@"+version, serviceapp.TemplatePin, "example.com/demo", dir)
}

func TestInjectServiceApp_WiresATemplateService(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-service")
	env := append(goOffline(t), offline(t)...)
	copyTemplate(t, env, dir)

	res := integration.RunEnv(t, env, "inject", "service-app", dir, "--title", "Demo")

	if res.Code != process.ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", res.Code, process.ExitOK, res.Stderr)
	}
	want := "Next steps:\n  cd " + dir + "\n  mise install\n" +
		"  mise -C client install\n  mise -C client run install\n" +
		"  mise run check\n"
	if res.Stdout != want {
		t.Errorf("stdout = %q, want %q", res.Stdout, want)
	}
	if res.Stderr != "" {
		t.Errorf("stderr = %q, want empty", res.Stderr)
	}
	client := filepath.Join(dir, "client")
	if got := integration.Tree(t, client); !slices.Equal(got, writtenClient) {
		t.Errorf("client tree = %q, want %q", got, writtenClient)
	}
	if got, want := read(t, client, "package.json"), `"name": "my-service",`; !strings.Contains(got, want) {
		t.Errorf("package.json = %q, want it to hold %q", got, want)
	}
	if got, want := read(t, dir, "internal/app/client.go"), `"example.com/demo/client"`; !strings.Contains(got, want) {
		t.Errorf("internal/app/client.go = %q, want it to import %s", got, want)
	}

	ctx, cancel := context.WithTimeout(t.Context(), checkTimeout)
	defer cancel()
	runIn(ctx, t, client, env, "mise", "run", "install")
	testService(ctx, t, dir, client, env)
	runIn(ctx, t, dir, env, "go", "vet", "./...")

	origin := serve(ctx, t, dir, env)
	shell := wantShell(t, origin, "/", "/api/")
	wantAsset(t, origin, "/", shell)
	if !strings.Contains(shell, "<title>Demo</title>") {
		t.Errorf("GET / = %q, want the title Demo", shell)
	}
}

func TestInjectServiceApp_LeavesTheEditsOnADifferentRoot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-service")
	copyTemplate(t, goOffline(t), dir)
	server := filepath.Join(dir, "internal", "app", "server.go")
	src := read(t, dir, "internal/app/server.go")
	if !strings.Contains(src, "return router, nil") {
		t.Fatalf("server.go = %q, want it to hold return router, nil", src)
	}
	must(t, os.WriteFile(server, []byte(strings.Replace(src, "return router, nil", "return router, check(router)", 1)), 0o644))
	before := map[string]string{
		"internal/app/app.go":    read(t, dir, "internal/app/app.go"),
		"internal/app/server.go": read(t, dir, "internal/app/server.go"),
	}

	res := integration.Run(t, "inject", "service-app", dir, "--client-dir", "web/ui")

	if res.Code != process.ExitFailure {
		t.Fatalf("code = %d, want %d; stderr = %q", res.Code, process.ExitFailure, res.Stderr)
	}
	wantErr := "webappgen inject service-app: the service lacks " + serviceapp.AnchorRouter +
		", where the client is wired in: apply the edits printed above by hand, then take the next steps\n"
	if res.Stderr != wantErr {
		t.Errorf("stderr = %q, want %q", res.Stderr, wantErr)
	}
	for name, src := range before {
		if after := read(t, dir, name); after != src {
			t.Errorf("%s changed:\n%s", name, after)
		}
	}
	if got := integration.Tree(t, filepath.Join(dir, "web", "ui")); !slices.Equal(got, writtenClient) {
		t.Errorf("client tree = %q, want %q", got, writtenClient)
	}
	if got, want := read(t, dir, "internal/app/client.go"), `"example.com/demo/web/ui"`; !strings.Contains(got, want) {
		t.Errorf("internal/app/client.go = %q, want it to import %s", got, want)
	}
	want := "Wrote internal/app/client.go, but edited no other file: a wiring anchor is missing. Make these edits by hand:\n" +
		"\ninternal/app/app.go, at " + serviceapp.AnchorImports + " (found): add these imports to it\n" +
		"\t\"io/fs\"\n\t\"net/http\"\n" +
		"\ninternal/app/app.go, at " + serviceapp.AnchorNodes + " (found): add these fields at the end of it\n" +
		"\t// The web client (client.go): the build it serves and its handler.\n" +
		"\tDist *graph.Node[fs.FS]\n\tClient *graph.Node[http.Handler]\n" +
		"\ninternal/app/app.go, at " + serviceapp.AnchorNew + " (found): insert this call before it\n" +
		"\tdefineClient(a.graph, &a.nodes)\n" +
		"\ninternal/app/server.go, at " + serviceapp.AnchorRouter + " (missing): insert this call before it\n" +
		"\tmountClient(router, s, n)\n" +
		"\nNext steps:\n  cd " + dir + "\n  mise install\n" +
		"  mise -C web/ui install\n  mise -C web/ui run install\n" +
		"  mise run check\n"
	if res.Stdout != want {
		t.Errorf("stdout =\n%s\nwant\n%s", res.Stdout, want)
	}
}

func TestInjectServiceApp_LeavesTheConfigEditsOnDifferingFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-service")
	copyTemplate(t, goOffline(t), dir)
	for name, change := range map[string][2]string{
		"mise.toml":                {`jq = "1.8.2"`, `jq = "1.8.3"`},
		".github/workflows/ci.yml": {"jdx/mise-action@v5.2.10", "jdx/mise-action@v5.3.0"},
	} {
		src := read(t, dir, name)
		if !strings.Contains(src, change[0]) {
			t.Fatalf("%s = %q, want it to hold %s", name, src, change[0])
		}
		must(t, os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(strings.ReplaceAll(src, change[0], change[1])), 0o644))
	}
	must(t, os.WriteFile(filepath.Join(dir, ".air.toml"), []byte("root = \".\"\n"), 0o644))
	before := map[string]string{}
	for _, name := range []string{"mise.toml", ".github/workflows/ci.yml", ".air.toml"} {
		before[name] = read(t, dir, name)
	}

	res := integration.Run(t, "inject", "service-app", dir)

	if res.Code != process.ExitFailure {
		t.Fatalf("code = %d, want %d; stderr = %q", res.Code, process.ExitFailure, res.Stderr)
	}
	wantErr := "webappgen inject service-app: " +
		"mise.toml lacks [tools] as go-web-sdk-template has it; " +
		".github/workflows/ci.yml lacks the check job, the integration job as go-web-sdk-template has it; " +
		".air.toml exists: apply the edits printed above by hand, then take the next steps\n"
	if res.Stderr != wantErr {
		t.Errorf("stderr = %q, want %q", res.Stderr, wantErr)
	}
	for name, src := range before {
		if after := read(t, dir, name); after != src {
			t.Errorf("%s changed:\n%s", name, after)
		}
	}
	if got := read(t, dir, "internal/app/app.go"); !strings.Contains(got, "defineClient(a.graph, &a.nodes)") {
		t.Errorf("internal/app/app.go = %q, want the client wired in", got)
	}
	for _, want := range []string{
		"Edited no mise.toml: it lacks [tools] as go-web-sdk-template has it. Make these edits by hand:\n" +
			"\nmise.toml, at [tools] (missing): add this pin at its start\n\t" + serviceapp.AirTool + "\n" +
			"\nmise.toml, at [tasks.check] (found): add this line after its header\n" +
			"\tdepends = [\"client:build\", \"client:check\"]\n",
		"\nmise.toml: append these tables to it\n\t[tasks.\"client:build\"]\n",
		"\tdepends = [\"client:build\"]\n\trun = \"mise -C client run check\"\n\n\t[tasks.dev]\n",
		"\nEdited no .github/workflows/ci.yml: it lacks the check job, the integration job as go-web-sdk-template has it. " +
			"Make these edits by hand:\n" +
			"\n.github/workflows/ci.yml, at the check job (missing): insert these steps before its mise run check step\n" +
			"\t      - name: Install the web client's tools and packages\n",
		"\n.github/workflows/ci.yml, at the integration job (missing): insert these steps before its go test -tags integration step\n",
		"\nEdited no .air.toml: it exists. Make this edit by hand:\n\n.air.toml: write it with this text\n" +
			"\t# The service's live reload under air, which mise run dev runs.\n",
		"\tsend_interrupt = true\n",
		"\tclean_on_exit = true\n\nNext steps:\n  cd " + dir + "\n  mise install\n" +
			"  mise -C client install\n  mise -C client run install\n  mise -C client run build\n" +
			"  mise run check\n",
	} {
		if !strings.Contains(res.Stdout, want) {
			t.Errorf("stdout =\n%s\nwant it to hold\n%s", res.Stdout, want)
		}
	}
	if !strings.HasPrefix(res.Stdout, "Edited no mise.toml") {
		t.Errorf("stdout = %q, want it to begin with the mise.toml edit", res.Stdout)
	}
}
