//go:build integration

package integration_test

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/process"

	"github.com/standards-lab/webappgen/integration"
)

// `new standalone <dir>`, the standalone kind's leaf, as a running binary:
// its help, the input it refuses, and what a valid run prints.

// The leaf's help lists every flag with its default.
func TestNewStandalone_Help(t *testing.T) {
	res := integration.Run(t, "new", "standalone", "--help")

	if res.Code != process.ExitUsage {
		t.Errorf("code = %d, want %d; stderr = %q", res.Code, process.ExitUsage, res.Stderr)
	}
	for _, want := range []string{
		"webappgen new standalone [flags] <dir>",
		"--name name",
		"--title title",
		"--base path", "(default /)",
		"--api-base URL", "(default /api/)",
		"--install",
	} {
		if !strings.Contains(res.Stdout, want) {
			t.Errorf("stdout = %q, want it to hold %q", res.Stdout, want)
		}
	}
}

// Each refusal is a usage error naming the fix, on stderr with the leaf's
// short usage, and leaves the target as it was: absent when it was absent,
// and holding what it held. The dir argument, when a case has one, is
// "{dir}", replaced with the case's absolute target.
func TestNewStandalone_UsageErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		holding []string // files to put in the target first; nil leaves it absent
		empty   bool     // create the target as an empty directory
		want    string   // the first line of stderr, after "webappgen new standalone: "
	}{
		{
			name: "missing dir",
			args: nil,
			want: "missing the project directory: give one, as in 'webappgen new standalone my-app'",
		},
		{
			name: "empty dir argument",
			args: []string{""},
			want: "missing the project directory: give one, as in 'webappgen new standalone my-app'",
		},
		{
			name: "two dirs",
			args: []string{"{dir}", "other"},
			want: "takes one project directory, got 2 arguments: quote a path that holds spaces",
		},
		{
			name:    "dir not empty",
			args:    []string{"{dir}"},
			holding: []string{"README.md"},
			want:    "{dir} exists and isn't empty: give a new or empty directory",
		},
		{
			name: "unknown flag",
			args: []string{"{dir}", "--bogus"},
			want: "flag provided but not defined: -bogus",
		},
		{
			name: "base not absolute",
			args: []string{"{dir}", "--base", "app/"},
			want: `--base "app/" doesn't begin with "/": give an absolute path, such as /app/`,
		},
		{
			name:  "base not clean",
			args:  []string{"{dir}", "--base=/app//"},
			empty: true,
			want:  `--base "/app//" isn't clean: give /app/`,
		},
		{
			name: "base with a query",
			args: []string{"{dir}", "--base", "/app/?v=1"},
			want: `--base "/app/?v=1" carries a query or fragment: give the path alone, such as /app/`,
		},
		{
			name: "base with a fragment",
			args: []string{"--base", "/app#top", "{dir}"},
			want: `--base "/app#top" carries a query or fragment: give the path alone, such as /app`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "my-app")
			if tc.holding != nil || tc.empty {
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
			args := []string{"new", "standalone"}
			for _, arg := range tc.args {
				args = append(args, strings.ReplaceAll(arg, "{dir}", dir))
			}

			res := integration.Run(t, args...)

			if res.Code != process.ExitUsage {
				t.Errorf("code = %d, want %d; stderr = %q", res.Code, process.ExitUsage, res.Stderr)
			}
			want := "webappgen new standalone: " + strings.ReplaceAll(tc.want, "{dir}", dir) + "\n" +
				"Usage: webappgen new standalone [flags] <dir>\n" +
				"Run 'webappgen new standalone --help' for details.\n"
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

// written is every path a standalone run writes, directories included, in
// Tree's order: the starter's files with the kind's laid over them.
var written = []string{
	".gitignore",
	".prettierignore",
	".prettierrc.json",
	"eslint.config.js",
	"index.html",
	"mise.toml",
	"package.json",
	"pnpm-lock.yaml",
	"pnpm-workspace.yaml",
	"public",
	"public/config.json",
	"scripts",
	"scripts/preflight.sh",
	"scripts/size.ts",
	"src",
	"src/config.ts",
	"src/main.ts",
	"src/theme.ts",
	"src/views",
	"src/views/home.ts",
	"src/views/not-found.ts",
	"test",
	"test/browser",
	"test/browser/app.test.ts",
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

// read returns the file at name under dir.
func read(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// version returns what the built binary's version command prints.
func version(t *testing.T) string {
	t.Helper()
	res := integration.Run(t, "version")
	if res.Code != process.ExitOK {
		t.Fatalf("version: code = %d; stderr = %q", res.Code, res.Stderr)
	}
	return strings.TrimSuffix(res.Stdout, "\n")
}

// A valid run, into a directory that doesn't exist or is empty, with every
// flag given or none, writes the project, prints the next steps, and exits
// 0. The flags' values, or their defaults, are filled in where the project
// reads them.
func TestNewStandalone_Writes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		flags   []string
		empty   bool
		pkg     string // package.json's name, JSON-quoted
		title   string // the page's <title>, HTML-escaped
		base    string
		apiBase string // config.json's apiBase, JSON-quoted
	}{
		{name: "defaults", pkg: `"my-app"`, title: "my-app", base: "/", apiBase: `"/api/"`},
		{name: "empty dir", empty: true, pkg: `"my-app"`, title: "my-app", base: "/", apiBase: `"/api/"`},
		{
			name:  "every flag",
			flags: []string{"--name", "console", "--title", `Ops & "Console"`, "--base", "/app", "--api-base", "https://api.example.com/v1/"},
			pkg:   `"console"`, title: "Ops &amp; &#34;Console&#34;", base: "/app/", apiBase: `"https://api.example.com/v1/"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "my-app")
			if tc.empty {
				if err := os.Mkdir(dir, 0o755); err != nil {
					t.Fatal(err)
				}
			}

			res := integration.Run(t, append([]string{"new", "standalone", dir}, tc.flags...)...)

			if res.Code != process.ExitOK {
				t.Fatalf("code = %d, want %d; stderr = %q", res.Code, process.ExitOK, res.Stderr)
			}
			want := "Next steps:\n  cd " + dir + "\n  mise install\n  mise run install\n  mise run check\n"
			if res.Stdout != want {
				t.Errorf("stdout = %q, want %q", res.Stdout, want)
			}
			if res.Stderr != "" {
				t.Errorf("stderr = %q, want empty", res.Stderr)
			}
			if got := integration.Tree(t, dir); !slices.Equal(got, written) {
				t.Errorf("tree = %q, want %q", got, written)
			}
			for name, wants := range map[string][]string{
				"package.json":       {`"name": ` + tc.pkg + ",\n", `"private": true,`},
				"index.html":         {`<base href="` + tc.base + `" />`, "<title>" + tc.title + "</title>"},
				"vite.config.ts":     {`base: "` + tc.base + `",`},
				"public/config.json": {"{\n  \"apiBase\": " + tc.apiBase + "\n}\n"},
			} {
				content := read(t, dir, name)
				for _, want := range wants {
					if !strings.Contains(content, want) {
						t.Errorf("%s = %q, want it to hold %q", name, content, want)
					}
				}
			}
			for _, name := range written {
				if path.Ext(name) == "" {
					continue
				}
				// mise.toml's download URLs hold mise's own {{ version }}, written as such.
				content := strings.ReplaceAll(read(t, dir, name), "{{ version }}", "")
				if strings.Contains(content, "{{") {
					t.Errorf("%s holds an unrendered {{", name)
				}
			}
			info, err := os.Stat(filepath.Join(dir, "scripts", "preflight.sh"))
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm()&0o111 == 0 {
				t.Errorf("scripts/preflight.sh mode = %v, want it executable", info.Mode())
			}
		})
	}
}

// The project takes each layer by its exact release tag, as a private
// package, and pins them with a lockfile; its mise config names the
// webappgen that wrote it in one comment, and pins no webappgen.
func TestNewStandalone_Pins(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-app")

	res := integration.Run(t, "new", "standalone", dir)

	if res.Code != process.ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", res.Code, process.ExitOK, res.Stderr)
	}
	var manifest struct {
		Private         bool              `json:"private"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal([]byte(read(t, dir, "package.json")), &manifest); err != nil {
		t.Fatalf("package.json: %v", err)
	}
	if !manifest.Private {
		t.Error(`package.json: "private" isn't true`)
	}
	lock := read(t, dir, "pnpm-lock.yaml")
	for layer, tag := range map[string]string{
		"web-foundation": "v0.3.0",
		"web-components": "v0.1.0",
		"web-app-sdk":    "v0.1.0",
		"web-tooling":    "v0.1.0",
	} {
		name, spec := "@standards-lab/"+layer, "github:standards-lab/"+layer+"#"+tag
		got, ok := manifest.Dependencies[name]
		if !ok {
			got = manifest.DevDependencies[name]
		}
		if got != spec {
			t.Errorf("package.json: %s = %q, want %q", name, got, spec)
		}
		if !strings.Contains(lock, "specifier: "+spec+"\n") {
			t.Errorf("pnpm-lock.yaml doesn't pin %s at %s", name, spec)
		}
	}

	config := read(t, dir, "mise.toml")
	want := "# Written by webappgen " + version(t) + ".\n"
	if !strings.HasPrefix(config, want) {
		t.Errorf("mise.toml starts %q, want %q", config[:min(len(config), 80)], want)
	}
	// The browser tier's browser, pinned for each platform the CLI ships for, each download's URL
	// naming mise's version.
	for _, platform := range []string{
		`linux-x64 = { url = "https://storage.googleapis.com/chrome-for-testing-public/{{ version }}/linux64/chrome-headless-shell-linux64.zip", checksum = "sha256:`,
		`linux-arm64 = { url = "https://storage.googleapis.com/chrome-for-testing-public/{{ version }}/linux-arm64/chrome-headless-shell-linux-arm64.zip", checksum = "sha256:`,
		`macos-x64 = { url = "https://storage.googleapis.com/chrome-for-testing-public/{{ version }}/mac-x64/chrome-headless-shell-mac-x64.zip", checksum = "sha256:`,
		`macos-arm64 = { url = "https://storage.googleapis.com/chrome-for-testing-public/{{ version }}/mac-arm64/chrome-headless-shell-mac-arm64.zip", checksum = "sha256:`,
	} {
		if !strings.Contains(config, "\n"+platform) {
			t.Errorf("mise.toml doesn't pin chrome-headless-shell as %q", platform)
		}
	}
	var naming []string
	for line := range strings.Lines(config) {
		if strings.Contains(line, "webappgen") {
			naming = append(naming, line)
		}
	}
	if len(naming) != 1 {
		t.Errorf("mise.toml's lines naming webappgen = %q, want the one comment", naming)
	}
}

// Every file the project copies or adapts from spike-html-apps names its
// source in a provenance line, which the written project keeps.
func TestNewStandalone_Provenance(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-app")

	res := integration.Run(t, "new", "standalone", dir)

	if res.Code != process.ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", res.Code, process.ExitOK, res.Stderr)
	}
	for _, name := range []string{
		".gitignore",
		".prettierignore",
		"eslint.config.js",
		"index.html",
		"mise.toml",
		"pnpm-workspace.yaml",
		"scripts/preflight.sh",
		"src/config.ts",
		"src/main.ts",
		"src/theme.ts",
		"src/views/not-found.ts",
		"test/browser/app.test.ts",
		"test/browser/support/app.ts",
		"test/browser/theme.test.ts",
		"tsconfig.json",
		"tsconfig.test.json",
		"vite.config.ts",
	} {
		if !strings.Contains(read(t, dir, name), "spike-html-apps (2883d78): ") {
			t.Errorf("%s has no provenance line naming spike-html-apps (2883d78)", name)
		}
	}
}
