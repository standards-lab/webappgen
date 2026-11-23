package serviceapp_test

import (
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/standards-lab/webappgen/internal/serviceapp"
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
