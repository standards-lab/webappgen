package serviceapp

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/standards-lab/go-cli-sdk"
)

// InjectCommand returns the service-app leaf under inject, `inject
// service-app <service-dir>`, which installs the web client into an
// existing go-web-sdk-template service, with [Command]'s flags. Its Args
// and Validate refuse, as usage errors that name the fix, a missing or
// extra argument, a service-dir that isn't a directory, holds no go.mod,
// or whose go.mod declares no module, a --base [standalone.ParseBase]
// refuses, a --client-dir [ParseClientDir] refuses or that exists in the
// service, and a service [CheckWire] refuses; an unknown flag is the
// dispatcher's usage error. Every refusal comes before anything is
// written. It declares no graph nodes, so it builds nothing and reads no
// configuration.
//
// A valid run writes the client into the --client-dir, as [Command] does,
// and wires it with [Wire], under the module go.mod declares. When every
// anchor is found it prints the next steps, [Command]'s without git init,
// and exits 0. When any is missing, the client and its client file are
// written and no existing Go file is edited: it prints each edit left, its
// file, its anchor, whether the anchor was found, and the code to insert,
// and exits 1.
func InjectCommand(version string) *cli.Command {
	var o options
	cmd := &cli.Command{
		Name:     "service-app",
		Summary:  "Install a web client into the Go web service in service-dir, copied from go-web-sdk-template",
		Synopsis: "<service-dir>",
		Args:     serviceDirArgs,
		Validate: func(inv *cli.Invocation) error {
			return o.validateService(inv.Args[0])
		},
		Run: func(_ context.Context, inv *cli.Invocation) error {
			wiring, err := o.install(version)
			if err != nil {
				return err
			}
			if len(wiring.Left) > 0 {
				if err := writeLeft(inv.Stdout, wiring); err != nil {
					return err
				}
				return fmt.Errorf("the service lacks %s, where the client is wired in: apply the edits printed above by hand",
					strings.Join(missingAnchors(wiring), ", "))
			}
			return o.writeNextSteps(inv.Stdout)
		},
	}
	o.define(cmd)
	return cmd
}

func serviceDirArgs(args []string) error {
	switch {
	case len(args) == 0 || args[0] == "":
		return cli.Usagef("missing the service directory: give one, as in 'webappgen inject service-app my-service'")
	case len(args) > 1:
		return cli.Usagef("takes a service directory, got %d arguments: quote a path that holds spaces", len(args))
	}
	return nil
}

func (o *options) validateService(dir string) error {
	info, err := os.Stat(dir)
	switch {
	case err != nil:
		return fmt.Errorf("%w: give the directory of a Go web service", err)
	case !info.IsDir():
		return fmt.Errorf("%s isn't a directory: give the directory of a Go web service", dir)
	}
	gomod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s has no go.mod: give the root of a Go module, such as a service copied from go-web-sdk-template", dir)
	}
	if err != nil {
		return err
	}
	if o.module = modulePath(gomod); o.module == "" {
		return fmt.Errorf("%s declares no module: give the root of a Go module, its go.mod beginning with a module directive",
			filepath.Join(dir, "go.mod"))
	}
	if err := o.parse(dir); err != nil {
		return err
	}
	client := filepath.Join(dir, filepath.FromSlash(o.clientDir))
	if _, err := os.Lstat(client); err == nil {
		return fmt.Errorf("--client-dir %q: %s exists: give another --client-dir, or remove it", o.clientDir, client)
	}
	return CheckWire(dir)
}

func modulePath(gomod []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(gomod))
	for scanner.Scan() {
		line, _, _ := strings.Cut(scanner.Text(), "//")
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module")
		if !ok || rest == "" || rest[0] != ' ' && rest[0] != '\t' && rest[0] != '"' {
			continue
		}
		rest = strings.TrimSpace(rest)
		if unquoted, err := strconv.Unquote(rest); err == nil {
			return unquoted
		}
		return rest
	}
	return ""
}

var placements = map[string]string{
	AnchorImports: "add these imports to it",
	AnchorNodes:   "add these fields at the end of it",
	AnchorNew:     "insert this call before it",
	AnchorRouter:  "insert this call before it",
}

func writeLeft(w io.Writer, wiring Wiring) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Wrote %s, but edited no other file: a wiring anchor is missing. Make these edits by hand:\n", wiring.Client)
	for _, e := range wiring.Left {
		found := "found"
		if !e.Found {
			found = "missing"
		}
		fmt.Fprintf(&b, "\n%s, at %s (%s): %s\n", e.Path, e.Anchor, found, placements[e.Anchor])
		for line := range strings.SplitSeq(e.Code, "\n") {
			fmt.Fprintf(&b, "\t%s\n", line)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}
