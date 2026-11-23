// Adapted from go-cli-sdk-template (a4d314c): template/internal/app/version.go, renamed webappgen.

package app

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/standards-lab/go-cli-sdk"
)

func versionCommand() *cli.Command {
	return &cli.Command{
		Name:    "version",
		Summary: "Print the webappgen version",
		Args:    cli.NoArgs,
		Run: func(_ context.Context, inv *cli.Invocation) error {
			_, err := fmt.Fprintln(inv.Stdout, moduleVersion())
			return err
		},
	}
}

func moduleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "(devel)"
	}
	return info.Main.Version
}
