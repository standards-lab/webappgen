package standalone_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/standards-lab/webappgen/internal/project"
	"github.com/standards-lab/webappgen/internal/project/projecttest"
	"github.com/standards-lab/webappgen/internal/standalone"
	"github.com/standards-lab/webappgen/internal/starter"
)

func TestFiles_Tracked(t *testing.T) {
	projecttest.CheckTracked(t, standalone.Files, "files")
}

func TestParseBase_Accepts(t *testing.T) {
	for base, want := range map[string]string{
		"/":             "/",
		"/app":          "/app/",
		"/app/":         "/app/",
		"/apps/console": "/apps/console/",
		"/a.b/c-d_e/":   "/a.b/c-d_e/",
	} {
		t.Run(base, func(t *testing.T) {
			got, err := standalone.ParseBase(base)
			if err != nil {
				t.Fatalf("ParseBase(%q) error = %v, want %q", base, err, want)
			}
			if got != want {
				t.Errorf("ParseBase(%q) = %q, want %q", base, got, want)
			}
		})
	}
}

func TestParseBase_Refuses(t *testing.T) {
	for base, want := range map[string]string{
		"":           `--base "" doesn't begin with "/": give an absolute path, such as /`,
		"app/":       `--base "app/" doesn't begin with "/": give an absolute path, such as /app/`,
		"./app":      `--base "./app" doesn't begin with "/": give an absolute path, such as /app`,
		"//":         `--base "//" isn't clean: give /`,
		"/app//":     `--base "/app//" isn't clean: give /app/`,
		"//app":      `--base "//app" isn't clean: give /app/`,
		"/a/../app/": `--base "/a/../app/" isn't clean: give /app/`,
		"/./app":     `--base "/./app" isn't clean: give /app/`,
		"/app?x=1":   `--base "/app?x=1" carries a query or fragment: give the path alone, such as /app`,
		"/app/#top":  `--base "/app/#top" carries a query or fragment: give the path alone, such as /app/`,
		"/?":         `--base "/?" carries a query or fragment: give the path alone, such as /`,
	} {
		t.Run(base, func(t *testing.T) {
			got, err := standalone.ParseBase(base)
			if err == nil {
				t.Fatalf("ParseBase(%q) = %q, want an error", base, got)
			}
			if err.Error() != want {
				t.Errorf("ParseBase(%q) error = %q, want %q", base, err, want)
			}
			if got != "" {
				t.Errorf("ParseBase(%q) = %q with its error, want empty", base, got)
			}
		})
	}
}

func TestFiles_WriteTheREADMEAndMiseTomlAsGolden(t *testing.T) {
	dir := t.TempDir()
	values := standalone.Values{Name: "my-app", Title: "my-app", Base: "/", APIBase: "/api/", Version: "(devel)"}
	if err := project.Write(dir, values, starter.Files, standalone.Files); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"README.md", "mise.toml"} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join("testdata", name+".golden"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s isn't byte for byte testdata/%s.golden:\n%s", name, name, got)
		}
	}
}
