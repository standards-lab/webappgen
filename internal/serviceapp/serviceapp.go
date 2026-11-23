package serviceapp

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/standards-lab/go-cli-sdk"

	"github.com/standards-lab/webappgen/internal/project"
	"github.com/standards-lab/webappgen/internal/standalone"
)

const (
	// TemplatePin is the go-web-sdk-template module the service is copied
	// from, at the version copied, as module@version.
	TemplatePin = "github.com/standards-lab/go-web-sdk-template/template@v0.16.0"
	// ToolsPin is the golang.org/x/tools module whose cmd/gonew copies the
	// template, at the version run, as module@version.
	ToolsPin = "golang.org/x/tools@v0.53.0"
)

type options struct {
	module    string
	dir       string
	name      string
	title     string
	base      string
	apiBase   string
	clientDir string
}

// Command returns the service-app leaf, `new service-app <module> <dir>`.
// Its Args and Validate refuse, as usage errors that name the fix, a
// missing or extra argument, a module path [CheckModule] refuses, a
// directory that exists and isn't empty, a --base
// [standalone.ParseBase] refuses, and a --client-dir [ParseClientDir]
// refuses; an unknown flag is the dispatcher's usage error. Every refusal
// comes before anything is written. It declares no graph nodes, so it
// builds nothing and reads no configuration.
//
// A valid run copies the template at [TemplatePin] into dir as module,
// running gonew at [ToolsPin] through go run, its progress on stderr. It
// then prints the next steps, a cd into dir, git init, mise install, and
// mise run check, and exits 0. A go missing from PATH, or a gonew that
// fails, exits nonzero naming it.
func Command() *cli.Command {
	var o options
	cmd := &cli.Command{
		Name:     "service-app",
		Summary:  "Write a Go web service, copied from go-web-sdk-template as module, into dir",
		Synopsis: "<module> <dir>",
		Args:     moduleDirArgs,
		Validate: func(inv *cli.Invocation) error {
			return o.validate(inv.Args[0], inv.Args[1])
		},
		Run: func(ctx context.Context, inv *cli.Invocation) error {
			if err := copyTemplate(ctx, o.module, o.dir, inv); err != nil {
				return err
			}
			return project.WriteNextSteps(inv.Stdout, o.dir, "git init", "mise install", "mise run check")
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&o.name, "name", "", "the web client's `name` (default: the directory's name)")
	flags.StringVar(&o.title, "title", "", "the web client's page `title` (default: the name)")
	flags.StringVar(&o.base, "base", "/", "the absolute `path` the web client is served under")
	flags.StringVar(&o.apiBase, "api-base", "/api/", "the `URL` of the API the web client calls")
	flags.StringVar(&o.clientDir, "client-dir", "client", "the `path` in the service the web client is written into")
	return cmd
}

func moduleDirArgs(args []string) error {
	switch {
	case len(args) == 0 || args[0] == "":
		return cli.Usagef("missing the module path and project directory: give both, as in 'webappgen new service-app example.com/my-service my-service'")
	case len(args) == 1 || args[1] == "":
		return cli.Usagef("missing the project directory: give one after the module path, as in 'webappgen new service-app %s my-service'", args[0])
	case len(args) > 2:
		return cli.Usagef("takes a module path and a project directory, got %d arguments: quote a path that holds spaces", len(args))
	}
	return nil
}

func (o *options) validate(module, dir string) error {
	if err := CheckModule(module); err != nil {
		return err
	}
	if err := project.CheckDir(dir); err != nil {
		return err
	}
	base, err := standalone.ParseBase(o.base)
	if err != nil {
		return err
	}
	clientDir, err := ParseClientDir(o.clientDir)
	if err != nil {
		return err
	}
	o.module, o.dir, o.base, o.clientDir = module, dir, base, clientDir
	if o.name == "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("%s: %w", dir, err)
		}
		o.name = filepath.Base(abs)
	}
	if o.title == "" {
		o.title = o.name
	}
	return nil
}

// CheckModule refuses, naming the fix, a module path gonew refuses to
// write a service as: one whose first element has no dot or isn't a
// lowercase domain name, of ASCII lowercase letters, digits, dots, and
// dashes, not beginning with a dash; or one with an element that is empty,
// begins or ends with a dot, or holds anything but ASCII letters, digits,
// and -._~.
func CheckModule(module string) error {
	for elem := range strings.SplitSeq(module, "/") {
		if elem == "" || strings.HasPrefix(elem, ".") || strings.HasSuffix(elem, ".") ||
			strings.TrimLeftFunc(elem, modulePathRune) != "" {
			return fmt.Errorf("module path %q has the element %q: give elements of ASCII letters, digits, and -._~, "+
				"each non-empty and neither beginning nor ending with a dot, such as example.com/my-service", module, elem)
		}
	}
	first, _, _ := strings.Cut(module, "/")
	if !strings.Contains(first, ".") {
		return fmt.Errorf("module path %q has no dot in its first element: begin it with a domain, such as example.com/%s", module, module)
	}
	if strings.HasPrefix(first, "-") || strings.TrimLeftFunc(first, domainRune) != "" {
		return fmt.Errorf("module path %q begins with %q, which isn't a domain of lowercase letters, digits, dots, and dashes: "+
			"begin it with one, such as example.com", module, first)
	}
	return nil
}

func modulePathRune(r rune) bool {
	return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9' || strings.ContainsRune("-._~", r)
}

func domainRune(r rune) bool {
	return 'a' <= r && r <= 'z' || '0' <= r && r <= '9' || r == '-' || r == '.'
}

// ParseClientDir returns dir, the path in the service the web client is
// written into, slash-separated and relative to the service's root. It
// refuses, naming the fix, a dir that is empty, absolute, the root itself,
// reaches above the root, or isn't clean, path.Clean's own result.
func ParseClientDir(dir string) (string, error) {
	clean := path.Clean(dir)
	switch {
	case dir == "" || clean == ".":
		return "", fmt.Errorf("--client-dir %q is the service's root: give a directory inside it, such as client", dir)
	case path.IsAbs(dir):
		return "", fmt.Errorf("--client-dir %q is absolute: give a path relative to the service's root, such as client", dir)
	case clean == ".." || strings.HasPrefix(clean, "../"):
		return "", fmt.Errorf("--client-dir %q reaches above the service's root: give a directory inside it, such as client", dir)
	case dir != clean:
		return "", fmt.Errorf("--client-dir %q isn't clean: give %s", dir, clean)
	}
	return dir, nil
}

func copyTemplate(ctx context.Context, module, dir string, inv *cli.Invocation) error {
	tools, version, _ := strings.Cut(ToolsPin, "@")
	gonew := tools + "/cmd/gonew@" + version
	goBin, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("go isn't on PATH: install Go (https://go.dev/dl), then run go run %s %s %s %s",
			gonew, TemplatePin, module, dir)
	}
	cmd := exec.CommandContext(ctx, goBin, "run", gonew, TemplatePin, module, dir)
	cmd.Stdout, cmd.Stderr = inv.Stderr, inv.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go run %s: %w: fix what it reports, then run webappgen again into a new or empty directory", gonew, err)
	}
	return nil
}
