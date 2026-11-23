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

func TestCLI_RootHelp(t *testing.T) {
	for _, args := range [][]string{nil, {"--help"}, {"-h"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			res := integration.Run(t, args...)

			if res.Code != process.ExitUsage {
				t.Errorf("code = %d, want %d; stderr = %q", res.Code, process.ExitUsage, res.Stderr)
			}
			for _, command := range []string{"inject", "new", "version"} {
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

func TestCLI_InjectListsKinds(t *testing.T) {
	for _, args := range [][]string{{"inject"}, {"inject", "--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			res := integration.Run(t, args...)

			if res.Code != process.ExitUsage {
				t.Errorf("code = %d, want %d; stderr = %q", res.Code, process.ExitUsage, res.Stderr)
			}
			if !strings.Contains(res.Stdout, "Commands:\n  service-app ") {
				t.Errorf("stdout = %q, want inject's help listing service-app", res.Stdout)
			}
			if res.Stderr != "" {
				t.Errorf("stderr = %q, want empty", res.Stderr)
			}
		})
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"bogus"}, `webappgen: unknown command "bogus"`},
		{[]string{"new", "bogus"}, `webappgen new: unknown command "bogus"`},
		{[]string{"inject", "bogus"}, `webappgen inject: unknown command "bogus"`},
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
