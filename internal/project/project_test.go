package project_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/standards-lab/webappgen/internal/project"
)

type values struct {
	Name, Title string
}

// read returns the file at name under dir, failing the test when there is
// none.
func read(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// tree returns the files under dir, slash-separated and sorted.
func tree(t *testing.T, dir string) []string {
	t.Helper()
	var names []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		names = append(names, filepath.ToSlash(rel))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(names)
	return names
}

func TestWrite_LaysLayersInOrder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app")
	starter := fstest.MapFS{
		"README.md":            {Data: []byte("starter\n")},
		"src/main.ts":          {Data: []byte("starter main\n")},
		"index.html.tmpl":      {Data: []byte("starter index\n")},
		"public/keep.txt":      {Data: []byte("kept\n")},
		"config.json.tmpl":     {Data: []byte("starter config\n")},
		"scripts/preflight.sh": {Data: []byte("#!/bin/sh\n")},
	}
	kind := fstest.MapFS{
		"src/main.ts":  {Data: []byte("kind main\n")},
		"index.html":   {Data: []byte("kind index\n")},
		"config.json":  {Data: []byte("kind config\n")},
		"kind-only.ts": {Data: []byte("kind\n")},
	}

	if err := project.Write(dir, values{}, starter, kind); err != nil {
		t.Fatal(err)
	}

	want := []string{"README.md", "config.json", "index.html", "kind-only.ts", "public/keep.txt", "scripts/preflight.sh", "src/main.ts"}
	if got := tree(t, dir); !slices.Equal(got, want) {
		t.Errorf("tree = %q, want %q", got, want)
	}
	for name, want := range map[string]string{
		"README.md":   "starter\n",
		"src/main.ts": "kind main\n",
		"index.html":  "kind index\n",
		"config.json": "kind config\n",
	} {
		if got := read(t, dir, name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestWrite_RendersTemplatesAndCopiesTheRest(t *testing.T) {
	dir := t.TempDir()
	layer := fstest.MapFS{
		"package.json.tmpl": {Data: []byte(`{"name": {{json .Name}}}` + "\n")},
		"index.html.tmpl":   {Data: []byte("<title>{{html .Title}}</title>\n")},
		"src/main.ts":       {Data: []byte("const t = `{{.Name}}`;\n")},
	}

	err := project.Write(dir, values{Name: `a"b<c>`, Title: `Tom & "Jerry" <3`}, layer)
	if err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]string{
		"package.json": `{"name": "a\"b<c>"}` + "\n",
		"index.html":   "<title>Tom &amp; &#34;Jerry&#34; &lt;3</title>\n",
		"src/main.ts":  "const t = `{{.Name}}`;\n",
	} {
		if got := read(t, dir, name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if want := []string{"index.html", "package.json", "src/main.ts"}; !slices.Equal(tree(t, dir), want) {
		t.Errorf("tree = %q, want %q", tree(t, dir), want)
	}
}

func TestWrite_Modes(t *testing.T) {
	dir := t.TempDir()
	layer := fstest.MapFS{
		"scripts/preflight.sh": {Data: []byte("#!/bin/sh\n"), Mode: 0o644},
		"scripts/size.ts":      {Data: []byte("\n"), Mode: 0o755},
	}

	if err := project.Write(dir, values{}, layer); err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]os.FileMode{"scripts/preflight.sh": 0o755, "scripts/size.ts": 0o644} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s mode = %v, want %v", name, got, want)
		}
	}
}

// A template that fails writes nothing, its siblings that render included,
// and leaves a directory that didn't exist absent.
func TestWrite_FailedTemplateWritesNothing(t *testing.T) {
	for name, text := range map[string]string{
		"unknown field": "{{.Missing}}",
		"parse error":   "{{.Name",
		"unknown func":  "{{yaml .Name}}",
	} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "app")
			layer := fstest.MapFS{
				"a.txt":        {Data: []byte("a\n")},
				"b.txt.tmpl":   {Data: []byte(text)},
				"c/d.txt.tmpl": {Data: []byte("{{.Name}}\n")},
			}

			err := project.Write(dir, values{Name: "app"}, layer)

			if err == nil {
				t.Fatal("Write succeeded, want the template's error")
			}
			if !strings.Contains(err.Error(), "b.txt.tmpl") {
				t.Errorf("error = %q, want it to name b.txt.tmpl", err)
			}
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Errorf("stat %s: %v, want it absent", dir, err)
			}
		})
	}
}
