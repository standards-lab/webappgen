package standalone

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/standards-lab/go-cli-sdk"

	"github.com/standards-lab/webappgen/internal/project"
	"github.com/standards-lab/webappgen/internal/starter"
)

//go:embed all:files
var files embed.FS

// Files is the standalone kind's own tree, rooted at its own top, which
// [Command] lays over [starter.Files]: the page, with its <base href>; the
// Vite config, at the same base; and public/config.json, which the app
// reads its API's address from. Its templates read [Values].
var Files fs.FS = mustSub(files, "files")

// mustSub returns fsys's subtree at dir, which the embed above guarantees.
func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

// Values are what the standalone project's templates, the starter's and
// the kind's own, are rendered with.
type Values struct {
	// Name is the project's package name.
	Name string
	// Title is the page's title.
	Title string
	// Base is the absolute path the app is served under, as [ParseBase]
	// returns it.
	Base string
	// APIBase is the URL of the API the app calls, written to its
	// config.json.
	APIBase string
	// Version is the webappgen version that wrote the project, named in its
	// mise.toml.
	Version string
}

// options are one run's settings: its flags as given, and what Validate
// derives from them and the directory.
type options struct {
	dir     string
	name    string
	title   string
	base    string
	apiBase string
	install bool
}

// Command returns the standalone leaf, `new standalone <dir>`. Its Args
// and Validate refuse, as usage errors that name the fix, a missing or
// extra directory argument, a directory that exists and isn't empty, and a
// --base [ParseBase] refuses; an unknown flag is the dispatcher's usage
// error. Every refusal comes before anything is written. It declares no
// graph nodes, so it builds nothing and reads no configuration.
//
// A valid run writes the project into dir, never prompting: the starter,
// then [Files] over it, rendered with the flags' [Values] and version, the
// webappgen version writing it. It then prints the next steps and exits 0.
func Command(version string) *cli.Command {
	var o options
	cmd := &cli.Command{
		Name:     "standalone",
		Summary:  "Write a standalone web client project, served on its own, into dir",
		Synopsis: "<dir>",
		Args:     dirArg,
		Validate: func(inv *cli.Invocation) error {
			return o.validate(inv.Args[0])
		},
		Run: func(_ context.Context, inv *cli.Invocation) error {
			values := Values{Name: o.name, Title: o.title, Base: o.base, APIBase: o.apiBase, Version: version}
			if err := project.Write(o.dir, values, starter.Files, Files); err != nil {
				return err
			}
			return writeNextSteps(inv.Stdout, o.dir)
		},
	}
	fs := cmd.Flags()
	fs.StringVar(&o.name, "name", "", "the project's `name` (default: the directory's name)")
	fs.StringVar(&o.title, "title", "", "the page's `title` (default: the name)")
	fs.StringVar(&o.base, "base", "/", "the absolute `path` the app is served under, written as its <base href>")
	fs.StringVar(&o.apiBase, "api-base", "/api/", "the `URL` of the API the app calls, written to its config.json")
	fs.BoolVar(&o.install, "install", false, "then run mise install and mise run install in the project")
	return cmd
}

// dirArg is the leaf's Args: exactly one non-empty directory.
func dirArg(args []string) error {
	switch {
	case len(args) == 0 || len(args) == 1 && args[0] == "":
		return cli.Usagef("missing the project directory: give one, as in 'webappgen new standalone my-app'")
	case len(args) > 1:
		return cli.Usagef("takes one project directory, got %d arguments: quote a path that holds spaces", len(args))
	}
	return nil
}

// validate checks dir and the flags o holds, and fills in what they leave
// to defaults: the name from dir's own name, the title from the name, and
// the base as [ParseBase] returns it.
func (o *options) validate(dir string) error {
	if err := checkDir(dir); err != nil {
		return err
	}
	base, err := ParseBase(o.base)
	if err != nil {
		return err
	}
	o.dir, o.base = dir, base
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

// checkDir refuses a dir that exists and is not an empty directory. A dir
// that doesn't exist is the project's to create.
func checkDir(dir string) error {
	info, err := os.Stat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("%w: give a new or empty directory", err)
	case !info.IsDir():
		return fmt.Errorf("%s exists and isn't a directory: give a new or empty directory", dir)
	}
	entries, err := os.ReadDir(dir)
	switch {
	case err != nil:
		return fmt.Errorf("%w: give a new or empty directory", err)
	case len(entries) > 0:
		return fmt.Errorf("%s exists and isn't empty: give a new or empty directory", dir)
	}
	return nil
}

// ParseBase returns base, the path the app is served under, ending in one
// slash, as a <base href> must for the app's relative URLs to resolve under
// it: "/app" and "/app/" both return "/app/", and "/" returns "/". It
// refuses, naming the fix, a base that doesn't begin with "/", that carries
// a query or fragment, or that isn't clean, path.Clean's own result with at
// most one slash added at its end, such as "/app//", "/a/../app/", or "//".
func ParseBase(base string) (string, error) {
	if !strings.HasPrefix(base, "/") {
		return "", fmt.Errorf("--base %q doesn't begin with \"/\": give an absolute path, such as /%s", base, strings.TrimPrefix(base, "./"))
	}
	if i := strings.IndexAny(base, "?#"); i >= 0 {
		return "", fmt.Errorf("--base %q carries a query or fragment: give the path alone, such as %s", base, base[:i])
	}
	clean := path.Clean(base)
	slashed := clean
	if clean != "/" {
		slashed += "/"
	}
	if base != clean && base != slashed {
		return "", fmt.Errorf("--base %q isn't clean: give %s", base, slashed)
	}
	return slashed, nil
}

// writeNextSteps writes to w what to run once the project is in dir.
func writeNextSteps(w io.Writer, dir string) error {
	_, err := fmt.Fprintf(w, "Next steps:\n  cd %s\n  mise install\n  mise run install\n  mise run check\n", dir)
	return err
}
