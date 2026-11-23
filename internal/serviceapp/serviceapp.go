package serviceapp

import (
	"context"
	"embed"
	"fmt"
	"go/token"
	"io"
	"io/fs"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/standards-lab/go-cli-sdk"

	"github.com/standards-lab/webappgen/internal/project"
	"github.com/standards-lab/webappgen/internal/standalone"
	"github.com/standards-lab/webappgen/internal/starter"
)

const (
	// TemplatePin is the go-web-sdk-template module the service is copied
	// from, at the version copied, as module@version.
	TemplatePin = "github.com/standards-lab/go-web-sdk-template/template@v0.16.0"
	// ToolsPin is the golang.org/x/tools module whose cmd/gonew copies the
	// template, at the version run, as module@version.
	ToolsPin = "golang.org/x/tools@v0.53.0"
)

//go:embed all:files
var files embed.FS

// Files is the service-app kind's client tree, rooted at its own top, which
// [Command] lays over [starter.Files] into the service's client directory:
// the Vite config, which builds the entry, theme, and page modules with a
// manifest at base ./ and no public directory; the page module, which
// draws the application page into the shell's empty <body>; the Go
// package that embeds the build; and the browser tier's support and cases
// that serve the build behind the shell go-web-sdk's NewClient renders.
// Its templates read [Values].
var Files fs.FS = mustSub(files, "files")

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

// Values are what the client's templates, the starter's and the kind's
// own, are rendered with.
type Values struct {
	// Name is the client's package name.
	Name string
	// Title is the page's title, which the page module draws in its header.
	Title string
	// Base is the absolute path the client is served under, as
	// [standalone.ParseBase] returns it.
	Base string
	// APIBase is the URL of the API the client calls, which the service
	// writes into the page's #app-config.
	APIBase string
	// Package is the name of the client's Go package, [PackageName] of the
	// client directory.
	Package string
	// Version is the webappgen version that wrote the client, named in its
	// mise.toml.
	Version string
}

// Kind is the kind's name, service-app, which the starter's templates
// read to write what this kind alone needs.
func (Values) Kind() string { return "service-app" }

type options struct {
	module    string
	dir       string
	name      string
	title     string
	base      string
	apiBase   string
	clientDir string
	install   bool
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
// then writes the client into the --client-dir in dir, never prompting:
// the starter, then [Files] over it, rendered with the flags' [Values] and
// version, the webappgen version writing it, wires it into the
// service's composition root with [Wire], and edits the service's tasks
// and CI to build it with [EditConfig]. It then prints the next steps, a
// cd into dir, git init, mise install, go mod download, the client's mise
// install and mise run install, run in its directory with mise -C, and the
// service's mise run check, and exits 0. With --install, it runs the
// install steps, the four after git init, itself, with [project.Install],
// their progress on stderr, and prints the steps left; a program missing
// from PATH or a step that fails exits nonzero naming the step and the
// steps left, and the written service stays. A go missing from PATH, a
// gonew that fails, a client directory the copied template already holds
// files in, a template [CheckWire] refuses, before the client is written,
// or a template [Wire] finds an anchor missing from, or whose
// configuration [EditConfig] leaves a file of, exits nonzero naming it.
func Command(version string) *cli.Command {
	var o options
	cmd := &cli.Command{
		Name:     "service-app",
		Summary:  "Write a Go web service, copied from go-web-sdk-template as module, with a web client in it, into dir",
		Synopsis: "<module> <dir>",
		Args:     moduleDirArgs,
		Validate: func(inv *cli.Invocation) error {
			return o.validate(inv.Args[0], inv.Args[1])
		},
		Run: func(ctx context.Context, inv *cli.Invocation) error {
			if err := copyTemplate(ctx, o.module, o.dir, inv); err != nil {
				return err
			}
			client := filepath.Join(o.dir, filepath.FromSlash(o.clientDir))
			if err := project.CheckDir(client); err != nil {
				return fmt.Errorf("--client-dir %q: the copied template holds it: %w", o.clientDir, err)
			}
			wiring, config, err := o.writeClient(version)
			if err != nil {
				return err
			}
			if missing := missingAnchors(wiring); len(missing) > 0 {
				return fmt.Errorf("the copied template lacks %s, where the client is wired in: report it against %s",
					strings.Join(missing, ", "), TemplatePin)
			}
			for _, e := range config {
				if !e.Applied {
					return fmt.Errorf("the copied template's %s %s: report it against %s", e.Path, e.Reason(), TemplatePin)
				}
			}
			return o.installThenWriteNextSteps(ctx, inv, config, true, "git init")
		},
	}
	o.define(cmd, "then run mise install and go mod download in the service, and mise install and mise run install in the client")
	return cmd
}

func (o *options) define(cmd *cli.Command, install string) {
	flags := cmd.Flags()
	flags.StringVar(&o.name, "name", "", "the web client's `name` (default: the directory's name)")
	flags.StringVar(&o.title, "title", "", "the web client's page `title` (default: the name)")
	flags.StringVar(&o.base, "base", "/", "the absolute `path` the web client is served under")
	flags.StringVar(&o.apiBase, "api-base", "/api/", "the `URL` of the API the web client calls")
	flags.StringVar(&o.clientDir, "client-dir", "client", "the `path` in the service the web client is written into")
	flags.BoolVar(&o.install, "install", false, install)
}

func (o *options) writeClient(version string) (Wiring, []ConfigEdit, error) {
	if err := CheckWire(o.dir); err != nil {
		return Wiring{}, nil, err
	}
	values := Values{
		Name: o.name, Title: o.title, Base: o.base, APIBase: o.apiBase,
		Package: PackageName(o.clientDir), Version: version,
	}
	if err := project.Write(filepath.Join(o.dir, filepath.FromSlash(o.clientDir)), values, starter.Files, Files); err != nil {
		return Wiring{}, nil, err
	}
	wiring, err := Wire(o.dir, o.module, WireOptions{
		Base: o.base, Title: o.title, APIBase: o.apiBase, ClientDir: o.clientDir,
	})
	if err != nil {
		return Wiring{}, nil, err
	}
	config, err := EditConfig(o.dir, o.clientDir)
	if err != nil {
		return Wiring{}, nil, err
	}
	return wiring, config, nil
}

func missingAnchors(wiring Wiring) []string {
	var missing []string
	for _, e := range wiring.Left {
		if !e.Found {
			missing = append(missing, e.Anchor)
		}
	}
	return missing
}

func (o *options) installSteps(service bool) []project.Step {
	var steps []project.Step
	if service {
		steps = []project.Step{{"mise", "install"}, {"go", "mod", "download"}}
	}
	return append(steps, project.Step{"mise", "-C", o.clientDir, "install"}, project.Step{"mise", "-C", o.clientDir, "run", "install"})
}

func (o *options) installThenWriteNextSteps(ctx context.Context, inv *cli.Invocation, config []ConfigEdit, service bool, first ...string) error {
	steps := o.installSteps(service)
	if o.install {
		if err := project.Install(ctx, o.dir, inv.Stderr, steps...); err != nil {
			return err
		}
		steps = nil
	}
	return o.writeNextSteps(inv.Stdout, config, steps, first...)
}

func (o *options) writeNextSteps(w io.Writer, config []ConfigEdit, steps []project.Step, first ...string) error {
	names := slices.Concat(first, project.StepNames(steps))
	if !slices.ContainsFunc(config, func(e ConfigEdit) bool { return e.Path == MiseFile && e.Applied }) {
		names = append(names, project.Step{"mise", "-C", o.clientDir, "run", "build"}.String())
	}
	return project.WriteNextSteps(w, o.dir, append(names, "mise run check")...)
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
	o.module = module
	return o.parse(dir)
}

func (o *options) parse(dir string) error {
	base, err := standalone.ParseBase(o.base)
	if err != nil {
		return err
	}
	clientDir, err := ParseClientDir(o.clientDir)
	if err != nil {
		return err
	}
	o.dir, o.base, o.clientDir = dir, base, clientDir
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

// PackageName returns the name of the client's Go package in clientDir, a
// directory [ParseClientDir] accepts: its last element, lowercased, with
// every rune but ASCII letters, digits, and underscores dropped, or
// "client" when that leaves no name, the blank identifier, a Go keyword,
// or a name that begins with a digit.
func PackageName(clientDir string) string {
	name := strings.Map(func(r rune) rune {
		switch {
		case 'a' <= r && r <= 'z' || '0' <= r && r <= '9' || r == '_':
			return r
		case 'A' <= r && r <= 'Z':
			return r + 'a' - 'A'
		}
		return -1
	}, path.Base(clientDir))
	if name == "" || name == "_" || token.IsKeyword(name) || '0' <= name[0] && name[0] <= '9' {
		return "client"
	}
	return name
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
