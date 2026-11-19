// Adapted from go-cli-sdk-template (a4d314c): template/integration/cli_test.go, renamed webappgen,
// with new's listing.

//go:build integration

package integration_test

import (
	"strings"
	"testing"

	"github.com/standards-lab/go-core/process"

	"github.com/standards-lab/webappgen/integration"
)

// The command tree as a running binary: main hands the program its
// arguments and its process streams, and exits with the code the run
// returns. The kinds' leaves take their own cases, in new_test.go.

// version prints the main module's version, one line, to stdout.
func TestCLI_Version(t *testing.T) {
	res := integration.Run(t, "version")

	if res.Code != process.ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", res.Code, process.ExitOK, res.Stderr)
	}
	line, rest, found := strings.Cut(res.Stdout, "\n")
	if line == "" || !found || rest != "" {
		t.Errorf("stdout = %q, want one non-empty line", res.Stdout)
	}
	if res.Stderr != "" {
		t.Errorf("stderr = %q, want empty", res.Stderr)
	}
}

// The root's help, asked for or given by no command at all, exits with the
// usage code on stdout and lists every command.
func TestCLI_RootHelp(t *testing.T) {
	for _, args := range [][]string{nil, {"--help"}, {"-h"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			res := integration.Run(t, args...)

			if res.Code != process.ExitUsage {
				t.Errorf("code = %d, want %d; stderr = %q", res.Code, process.ExitUsage, res.Stderr)
			}
			for _, command := range []string{"new", "version"} {
				if !strings.Contains(res.Stdout, "\n  "+command+" ") {
					t.Errorf("stdout = %q, want the root help listing %s", res.Stdout, command)
				}
			}
			if res.Stderr != "" {
				t.Errorf("stderr = %q, want empty", res.Stderr)
			}
		})
	}
}

// new, run alone or asked for its help, lists its kind leaves on stdout
// with the usage code.
func TestCLI_NewListsKinds(t *testing.T) {
	for _, args := range [][]string{{"new"}, {"new", "--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			res := integration.Run(t, args...)

			if res.Code != process.ExitUsage {
				t.Errorf("code = %d, want %d; stderr = %q", res.Code, process.ExitUsage, res.Stderr)
			}
			if !strings.Contains(res.Stdout, "Commands:\n  standalone ") {
				t.Errorf("stdout = %q, want new's help listing standalone", res.Stdout)
			}
			if res.Stderr != "" {
				t.Errorf("stderr = %q, want empty", res.Stderr)
			}
		})
	}
}

// An unknown command is a usage error: the usage code, and the error and
// usage on stderr alone.
func TestCLI_UnknownCommand(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"bogus"}, `webappgen: unknown command "bogus"`},
		{[]string{"new", "bogus"}, `webappgen new: unknown command "bogus"`},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			res := integration.Run(t, tc.args...)

			if res.Code != process.ExitUsage {
				t.Errorf("code = %d, want %d", res.Code, process.ExitUsage)
			}
			if !strings.HasPrefix(res.Stderr, tc.want) {
				t.Errorf("stderr = %q, want it to start with %q", res.Stderr, tc.want)
			}
			if res.Stdout != "" {
				t.Errorf("stdout = %q, want empty", res.Stdout)
			}
		})
	}
}
