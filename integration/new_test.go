//go:build integration

package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/process"

	"github.com/standards-lab/webappgen/integration"
)

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

func TestNewStandalone_UsageErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		holding []string
		empty   bool
		want    string
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

var written = []string{
	".gitignore",
	".prettierignore",
	".prettierrc.json",
	"README.md",
	"eslint.config.js",
	"index.html",
	"mise.toml",
	"package.json",
	"pnpm-lock.yaml",
	"pnpm-workspace.yaml",
	"public",
	"public/config.json",
	"scripts",
	"scripts/currency.sh",
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

func read(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func version(t *testing.T) string {
	t.Helper()
	res := integration.Run(t, "version")
	if res.Code != process.ExitOK {
		t.Fatalf("version: code = %d; stderr = %q", res.Code, res.Stderr)
	}
	return strings.TrimSuffix(res.Stdout, "\n")
}

func TestNewStandalone_Writes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		dir     string
		cd      string
		flags   []string
		empty   bool
		pkg     string
		title   string
		base    string
		apiBase string
	}{
		{name: "defaults", pkg: `"my-app"`, title: "my-app", base: "/", apiBase: `"/api/"`},
		{name: "empty dir", empty: true, pkg: `"my-app"`, title: "my-app", base: "/", apiBase: `"/api/"`},
		{
			name:  "every flag",
			flags: []string{"--name", "console", "--title", `Ops & "Console"`, "--base", "/app", "--api-base", "https://api.example.com/v1/"},
			pkg:   `"console"`, title: "Ops &amp; &#34;Console&#34;", base: "/app/", apiBase: `"https://api.example.com/v1/"`,
		},
		{name: "dir with a space", dir: "My App", cd: "'{parent}/My App'", pkg: `"My App"`, title: "My App", base: "/", apiBase: `"/api/"`},
		{name: "dir with a quote", dir: "it's", cd: `'{parent}/it'\''s'`, pkg: `"it's"`, title: "it&#39;s", base: "/", apiBase: `"/api/"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := t.TempDir()
			name := tc.dir
			if name == "" {
				name = "my-app"
			}
			dir := filepath.Join(parent, name)
			cd := dir
			if tc.cd != "" {
				cd = strings.ReplaceAll(tc.cd, "{parent}", parent)
			}
			if tc.empty {
				if err := os.Mkdir(dir, 0o755); err != nil {
					t.Fatal(err)
				}
			}

			res := integration.Run(t, append([]string{"new", "standalone", dir}, tc.flags...)...)

			if res.Code != process.ExitOK {
				t.Fatalf("code = %d, want %d; stderr = %q", res.Code, process.ExitOK, res.Stderr)
			}
			want := "Next steps:\n  cd " + cd + "\n  mise install\n  mise run install\n  mise run check\n"
			if res.Stdout != want {
				t.Errorf("stdout = %q, want %q", res.Stdout, want)
			}
			cdLine, _, _ := strings.Cut(strings.TrimPrefix(res.Stdout, "Next steps:\n  "), "\n")
			if out, err := exec.Command("sh", "-c", cdLine+" && pwd -P").Output(); err != nil {
				t.Errorf("sh -c %q: %v", cdLine+" && pwd -P", err)
			} else if real, err := filepath.EvalSymlinks(dir); err != nil {
				t.Fatal(err)
			} else if got := strings.TrimSuffix(string(out), "\n"); got != real {
				t.Errorf("sh -c %q printed %q, want %q", cdLine+" && pwd -P", got, real)
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
				"README.md":          {"# " + strings.Trim(tc.pkg, `"`) + "\n", "mise run check", "public/config.json"},
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

var releaseTag = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

var browserPin = regexp.MustCompile(`^[a-z0-9-]+ = \{ url = "https://storage\.googleapis\.com/chrome-for-testing-public/\{\{ version \}\}/[^"]+/chrome-headless-shell-[a-z0-9-]+\.zip", checksum = "sha256:[0-9a-f]{64}"`)

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
	for _, layer := range []string{"web-foundation", "web-icons", "web-components", "web-app-sdk", "web-tooling"} {
		name := "@standards-lab/" + layer
		spec, ok := manifest.Dependencies[name]
		if !ok {
			spec = manifest.DevDependencies[name]
		}
		tag, found := strings.CutPrefix(spec, "github:standards-lab/"+layer+"#")
		if !found || !releaseTag.MatchString(tag) {
			t.Errorf("package.json: %s = %q, want github:standards-lab/%s#vX.Y.Z", name, spec, layer)
			continue
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
	pinned := map[string]bool{}
	var naming []string
	for line := range strings.Lines(config) {
		if platform, _, ok := strings.Cut(line, " = "); ok && browserPin.MatchString(line) {
			pinned[platform] = true
		}
		if strings.Contains(line, "webappgen") {
			naming = append(naming, line)
		}
	}
	for _, platform := range []string{"linux-x64", "linux-arm64", "macos-x64", "macos-arm64"} {
		if !pinned[platform] {
			t.Errorf("mise.toml doesn't pin chrome-headless-shell for %s by a URL naming {{ version }}, with its sha256", platform)
		}
	}
	if len(naming) != 1 {
		t.Errorf("mise.toml's lines naming webappgen = %q, want the one comment", naming)
	}
}
