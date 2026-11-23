package serviceapp_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/standards-lab/webappgen/internal/serviceapp"
)

var configFiles = []string{serviceapp.MiseFile, serviceapp.CIFile, serviceapp.AirFile}

func configFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(filepath.Join("testdata", "config", "template"))); err != nil {
		t.Fatal(err)
	}
	return dir
}

func readConfig(t *testing.T, dir, name string) string {
	t.Helper()
	return read(t, filepath.Join(dir, filepath.FromSlash(name)))
}

func writeConfig(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestEditConfig_EditsTheTemplate(t *testing.T) {
	dir := configFixture(t)

	edits, err := serviceapp.EditConfig(dir, "client")
	if err != nil {
		t.Fatalf("EditConfig: %v", err)
	}

	var paths []string
	for _, e := range edits {
		paths = append(paths, e.Path)
		if !e.Applied || e.Defined != nil || e.Reason() != "" {
			t.Errorf("%s: Applied = %t, Defined = %q, Reason = %q; want it applied", e.Path, e.Applied, e.Defined, e.Reason())
		}
		for _, in := range e.Inserts {
			if !in.Found {
				t.Errorf("%s: applied insert at %q isn't marked found", e.Path, in.Anchor)
			}
		}
	}
	if !slices.Equal(paths, configFiles) {
		t.Errorf("paths = %q, want %q", paths, configFiles)
	}
	golden := filepath.Join("testdata", "config", "template.golden")
	for _, name := range configFiles {
		if got, want := readConfig(t, dir, name), readConfig(t, golden, name); got != want {
			t.Errorf("%s =\n%s\nwant %s:\n%s", name, got, filepath.Join(golden, name), want)
		}
	}
}

func TestEditConfig_QuotesTheClientDir(t *testing.T) {
	dir := configFixture(t)

	if _, err := serviceapp.EditConfig(dir, `web/My "UI"`); err != nil {
		t.Fatalf("EditConfig: %v", err)
	}

	for name, wants := range map[string][]string{
		serviceapp.MiseFile: {
			`run = "mise -C 'web/My \"UI\"' run build"`,
			`(mise -C 'web/My \"UI\"' exec -- pnpm exec vite build --watch) &`,
		},
		serviceapp.CIFile: {"          mise -C 'web/My \"UI\"' install\n"},
		serviceapp.AirFile: {
			`exclude_dir = ["tmp", "bin", "web/My \"UI\"/node_modules", "web/My \"UI\"/src", "web/My \"UI\"/test", "web/My \"UI\"/scripts"]`,
		},
	} {
		content := readConfig(t, dir, name)
		for _, want := range wants {
			if !strings.Contains(content, want) {
				t.Errorf("%s = %q, want it to hold %q", name, content, want)
			}
		}
	}
}

func TestEditConfig_LeavesADifferingFile(t *testing.T) {
	for _, tc := range []struct {
		name    string
		file    string
		change  func(string) string
		missing []string
		defined []string
		reason  string
	}{
		{
			name: "a task's run differs",
			file: serviceapp.MiseFile,
			change: func(s string) string {
				return strings.Replace(s, `run = "go test -race ./..."`, `run = "go test ./..."`, 1)
			},
			missing: []string{"[tasks.test]"},
			reason:  "lacks [tasks.test] as go-web-sdk-template has it",
		},
		{
			name: "a task already depends",
			file: serviceapp.MiseFile,
			change: func(s string) string {
				return strings.Replace(s, `run = "go run ./cmd/server"`+"\n", `run = "go run ./cmd/server"`+"\n"+`depends = ["fmt"]`+"\n", 1)
			},
			missing: []string{"[tasks.serve]"},
			reason:  "lacks [tasks.serve] as go-web-sdk-template has it",
		},
		{
			name:    "a tool is bumped",
			file:    serviceapp.MiseFile,
			change:  func(s string) string { return strings.Replace(s, `jq = "1.8.2"`, `jq = "1.8.3"`, 1) },
			missing: []string{"[tools]"},
			reason:  "lacks [tools] as go-web-sdk-template has it",
		},
		{
			name:    "a dev task exists",
			file:    serviceapp.MiseFile,
			change:  func(s string) string { return s + "\n[tasks.dev]\nrun = \"air\"\n" },
			defined: []string{"[tasks.dev]"},
			reason:  "already defines [tasks.dev]",
		},
		{
			name: "a check job differs",
			file: serviceapp.CIFile,
			change: func(s string) string {
				return strings.Replace(s, "run: mise run check", "run: mise run check --jobs 2", 1)
			},
			missing: []string{"the check job"},
			reason:  "lacks the check job as go-web-sdk-template has it",
		},
		{
			name:    "the workflow is missing",
			file:    serviceapp.CIFile,
			missing: []string{"the check job", "the integration job"},
			reason:  "lacks the check job, the integration job as go-web-sdk-template has it",
		},
		{
			name:    "an .air.toml exists",
			file:    serviceapp.AirFile,
			change:  func(string) string { return "root = \"app\"\n" },
			defined: []string{serviceapp.AirFile},
			reason:  "exists",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := configFixture(t)
			if tc.change == nil {
				if err := os.Remove(filepath.Join(dir, filepath.FromSlash(tc.file))); err != nil {
					t.Fatal(err)
				}
			} else {
				content := ""
				if tc.file != serviceapp.AirFile {
					content = readConfig(t, dir, tc.file)
				}
				changed := tc.change(content)
				if changed == content {
					t.Fatalf("the change left %s as it was", tc.file)
				}
				writeConfig(t, dir, tc.file, changed)
			}
			before := ""
			if tc.change != nil {
				before = readConfig(t, dir, tc.file)
			}

			edits, err := serviceapp.EditConfig(dir, "client")
			if err != nil {
				t.Fatalf("EditConfig: %v", err)
			}

			golden := filepath.Join("testdata", "config", "template.golden")
			for _, e := range edits {
				if e.Path != tc.file {
					if !e.Applied {
						t.Errorf("%s left: %s; want it applied", e.Path, e.Reason())
					} else if got, want := readConfig(t, dir, e.Path), readConfig(t, golden, e.Path); got != want {
						t.Errorf("%s =\n%s\nwant it as %s has it", e.Path, got, golden)
					}
					continue
				}
				if e.Applied {
					t.Fatalf("%s applied, want it left", e.Path)
				}
				var missing []string
				for _, in := range e.Inserts {
					if !in.Found {
						missing = append(missing, in.Anchor)
					}
				}
				if !slices.Equal(missing, tc.missing) {
					t.Errorf("missing anchors = %q, want %q", missing, tc.missing)
				}
				if !slices.Equal(e.Defined, tc.defined) {
					t.Errorf("Defined = %q, want %q", e.Defined, tc.defined)
				}
				if got := e.Reason(); got != tc.reason {
					t.Errorf("Reason = %q, want %q", got, tc.reason)
				}
			}
			if tc.change == nil {
				if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(tc.file))); err == nil {
					t.Errorf("%s exists, want it still missing", tc.file)
				}
			} else if after := readConfig(t, dir, tc.file); after != before {
				t.Errorf("%s changed:\n%s", tc.file, after)
			}
		})
	}
}

func TestEditConfig_InsertsCarryTheirCode(t *testing.T) {
	dir := configFixture(t)
	writeConfig(t, dir, serviceapp.MiseFile, "")

	edits, err := serviceapp.EditConfig(dir, "client")
	if err != nil {
		t.Fatalf("EditConfig: %v", err)
	}

	var got []string
	for _, in := range edits[0].Inserts {
		got = append(got, in.Anchor+": "+strings.SplitN(in.Code, "\n", 2)[0])
	}
	want := []string{
		"[tools]: " + serviceapp.AirTool,
		`[tasks.check]: depends = ["client:build", "client:check"]`,
		`[tasks.test]: depends = ["client:build"]`,
		`[tasks.integration]: depends = ["client:build"]`,
		`[tasks.serve]: depends = ["client:build"]`,
		`: [tasks."client:build"]`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("inserts =\n%q\nwant\n%q", got, want)
	}
}
