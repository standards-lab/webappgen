package serviceapp_test

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/standards-lab/webappgen/internal/project"
	"github.com/standards-lab/webappgen/internal/project/projecttest"
	"github.com/standards-lab/webappgen/internal/serviceapp"
	"github.com/standards-lab/webappgen/internal/standalone"
	"github.com/standards-lab/webappgen/internal/starter"
)

func TestServicePins_FindsEveryPin(t *testing.T) {
	out, err := exec.Command("bash", "../../scripts/service-pins.sh").Output()
	if err != nil {
		t.Fatalf("scripts/service-pins.sh: %v", err)
	}
	got := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	want := []string{"TemplatePin " + serviceapp.TemplatePin, "ToolsPin " + serviceapp.ToolsPin}
	if !slices.Equal(got, want) {
		t.Errorf("scripts/service-pins.sh = %q, want %q", got, want)
	}
}

func TestCheckModule_Accepts(t *testing.T) {
	for _, module := range []string{
		"example.com/demo",
		"example.com",
		"github.com/Org/My_Service~2",
		"go.example-corp.io/a/b.c/d-e",
	} {
		t.Run(module, func(t *testing.T) {
			if err := serviceapp.CheckModule(module); err != nil {
				t.Errorf("CheckModule(%q) error = %v, want nil", module, err)
			}
		})
	}
}

func TestCheckModule_Refuses(t *testing.T) {
	const elements = "give elements of ASCII letters, digits, and -._~, " +
		"each non-empty and neither beginning nor ending with a dot, such as example.com/my-service"
	const domain = "which isn't a domain of lowercase letters, digits, dots, and dashes: begin it with one, such as example.com"
	for module, want := range map[string]string{
		"demo":               `module path "demo" has no dot in its first element: begin it with a domain, such as example.com/demo`,
		"demo/x":             `module path "demo/x" has no dot in its first element: begin it with a domain, such as example.com/demo/x`,
		"Example.com/demo":   `module path "Example.com/demo" begins with "Example.com", ` + domain,
		"-example.com/demo":  `module path "-example.com/demo" begins with "-example.com", ` + domain,
		"ex_ample.com/demo":  `module path "ex_ample.com/demo" begins with "ex_ample.com", ` + domain,
		"/example.com/demo":  `module path "/example.com/demo" has the element "": ` + elements,
		"example.com/demo/":  `module path "example.com/demo/" has the element "": ` + elements,
		"example.com//demo":  `module path "example.com//demo" has the element "": ` + elements,
		"example.com/.demo":  `module path "example.com/.demo" has the element ".demo": ` + elements,
		"example.com/demo.":  `module path "example.com/demo." has the element "demo.": ` + elements,
		"example.com/../x":   `module path "example.com/../x" has the element "..": ` + elements,
		"example.com/my app": `module path "example.com/my app" has the element "my app": ` + elements,
		"example.com/démo":   `module path "example.com/démo" has the element "démo": ` + elements,
	} {
		t.Run(module, func(t *testing.T) {
			err := serviceapp.CheckModule(module)
			if err == nil {
				t.Fatalf("CheckModule(%q) = nil, want an error", module)
			}
			if err.Error() != want {
				t.Errorf("CheckModule(%q) error = %q, want %q", module, err, want)
			}
		})
	}
}

func TestParseClientDir_Accepts(t *testing.T) {
	for _, dir := range []string{"client", "web/client", "a.b/c-d_e"} {
		t.Run(dir, func(t *testing.T) {
			got, err := serviceapp.ParseClientDir(dir)
			if err != nil {
				t.Fatalf("ParseClientDir(%q) error = %v, want %q", dir, err, dir)
			}
			if got != dir {
				t.Errorf("ParseClientDir(%q) = %q, want %q", dir, got, dir)
			}
		})
	}
}

func TestParseClientDir_Refuses(t *testing.T) {
	for dir, want := range map[string]string{
		"":            `--client-dir "" is the service's root: give a directory inside it, such as client`,
		".":           `--client-dir "." is the service's root: give a directory inside it, such as client`,
		"a/..":        `--client-dir "a/.." is the service's root: give a directory inside it, such as client`,
		"/client":     `--client-dir "/client" is absolute: give a path relative to the service's root, such as client`,
		"..":          `--client-dir ".." reaches above the service's root: give a directory inside it, such as client`,
		"../client":   `--client-dir "../client" reaches above the service's root: give a directory inside it, such as client`,
		"client/":     `--client-dir "client/" isn't clean: give client`,
		"./client":    `--client-dir "./client" isn't clean: give client`,
		"web//client": `--client-dir "web//client" isn't clean: give web/client`,
	} {
		t.Run(dir, func(t *testing.T) {
			got, err := serviceapp.ParseClientDir(dir)
			if err == nil {
				t.Fatalf("ParseClientDir(%q) = %q, want an error", dir, got)
			}
			if err.Error() != want {
				t.Errorf("ParseClientDir(%q) error = %q, want %q", dir, err, want)
			}
			if got != "" {
				t.Errorf("ParseClientDir(%q) = %q with its error, want empty", dir, got)
			}
		})
	}
}

func TestFiles_Tracked(t *testing.T) {
	projecttest.CheckTracked(t, serviceapp.Files, "files")
}

func TestPackageName(t *testing.T) {
	for dir, want := range map[string]string{
		"client":        "client",
		"web/ui":        "ui",
		"web/My-Client": "myclient",
		"a.b/c_d":       "c_d",
		"web/2d":        "client",
		"web/go":        "client",
		"web/_":         "client",
		"web/---":       "client",
	} {
		t.Run(dir, func(t *testing.T) {
			if got := serviceapp.PackageName(dir); got != want {
				t.Errorf("PackageName(%q) = %q, want %q", dir, got, want)
			}
		})
	}
}

func write(t *testing.T, values any, layers ...fs.FS) string {
	t.Helper()
	dir := t.TempDir()
	if err := project.Write(dir, values, layers...); err != nil {
		t.Fatal(err)
	}
	return dir
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

var (
	standaloneBody = regexp.MustCompile(`(?s)<body class="([^"]*)">\n(.*)\n  </body>`)
	pageTitle      = regexp.MustCompile(`(?m)^const title = (".*");$`)
	pageClass      = regexp.MustCompile(`(?m)^document\.body\.className = "([^"]*)";$`)
	pageMarkup     = regexp.MustCompile("(?s)document\\.body\\.innerHTML = `\n(.*)`;\n")
	iconImport     = regexp.MustCompile(`(?m)^import (\w+) from "@standards-lab/web-icons/(\w+/[\w-]+)";$`)
	interpolation  = regexp.MustCompile(`\$\{(\w+)\}`)
)

func markupLines(markup string) []string {
	var lines []string
	for line := range strings.Lines(markup) {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func TestFiles_PageModuleDrawsStandalonesPageBody(t *testing.T) {
	const title = "Ops & \"Console\" <b> it's ${x} `q` \\"
	page := readFile(t, write(t, serviceapp.Values{
		Name: "my-app", Title: title, Base: "/", APIBase: "/api/", Package: "client", Version: "(devel)",
	}, starter.Files, serviceapp.Files), "src/page.ts")
	html := readFile(t, write(t, standalone.Values{
		Name: "my-app", Title: title, Base: "/", APIBase: "/api/", Version: "(devel)",
	}, starter.Files, standalone.Files), "index.html")

	body := standaloneBody.FindStringSubmatch(html)
	if body == nil {
		t.Fatalf("index.html = %q, want a <body class> holding the page", html)
	}
	if got := pageClass.FindStringSubmatch(page); got == nil || got[1] != body[1] {
		t.Errorf("src/page.ts sets the body's class as %q, want document.body.className = %q", got, body[1])
	}
	quoted := pageTitle.FindStringSubmatch(page)
	markup := pageMarkup.FindStringSubmatch(page)
	if quoted == nil || markup == nil {
		t.Fatalf("src/page.ts = %q, want its title constant and its body's markup", page)
	}
	values := map[string]string{}
	var text string
	if err := json.Unmarshal([]byte(quoted[1]), &text); err != nil {
		t.Fatalf("src/page.ts's title %s: %v", quoted[1], err)
	}
	values["title"] = text
	for _, icon := range iconImport.FindAllStringSubmatch(page, -1) {
		values[icon[1]] = `<svg data-icon="` + icon[2] + `"></svg>`
	}
	drawn := interpolation.ReplaceAllStringFunc(markup[1], func(s string) string {
		name := interpolation.FindStringSubmatch(s)[1]
		value, ok := values[name]
		if !ok {
			t.Errorf("src/page.ts's markup interpolates %s, neither its title nor an icon", s)
		}
		return value
	})

	if got, want := markupLines(drawn), markupLines(body[2]); !slices.Equal(got, want) {
		t.Errorf("src/page.ts draws, glyphs filled:\n%s\nwant index.html's body:\n%s",
			strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
