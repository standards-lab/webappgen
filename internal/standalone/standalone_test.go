package standalone_test

import (
	"testing"

	"github.com/standards-lab/webappgen/internal/standalone"
)

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

// Each refusal names the flag, the value, and the fix: what to give
// instead.
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

// The leaf's flags and their defaults: --name and --title default to
// values derived from the directory, so their flags have none.
func TestCommand_Flags(t *testing.T) {
	fs := standalone.Command().Flags()
	for name, def := range map[string]string{
		"name":     "",
		"title":    "",
		"base":     "/",
		"api-base": "/api/",
		"install":  "false",
	} {
		f := fs.Lookup(name)
		if f == nil {
			t.Errorf("no --%s flag", name)
			continue
		}
		if f.DefValue != def {
			t.Errorf("--%s default = %q, want %q", name, f.DefValue, def)
		}
	}
}
