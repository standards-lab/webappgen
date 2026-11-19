//go:build integration

package integration_test

import (
	"os"
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

// A valid run, into a directory that doesn't exist or is empty, with every
// flag given or none, exits 0 and prints the next steps.
func TestNewStandalone_NextSteps(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flags []string
		empty bool
	}{
		{name: "defaults"},
		{name: "empty dir", empty: true},
		{name: "every flag", flags: []string{"--name", "console", "--title", "Console", "--base", "/app", "--api-base", "https://api.example.com/", "--install"}},
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
		})
	}
}
