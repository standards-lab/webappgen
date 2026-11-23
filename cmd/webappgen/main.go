// Adapted from go-cli-sdk-template (a4d314c): template/cmd/app/main.go, renamed webappgen.

// Command webappgen writes web client projects on the standards-lab stack.
// Its main derives the signal context and makes one call into the
// composition root, internal/app. That call sits in run, apart from main,
// so run's deferred stop releases the signal registration before main's
// os.Exit, which skips deferred calls.
package main

import (
	"os"

	"github.com/standards-lab/go-cli-sdk"
	"github.com/standards-lab/go-core/process"

	"github.com/standards-lab/webappgen/internal/app"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := process.SignalContext()
	defer stop()
	return app.New(cli.Streams{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}).Run(ctx, os.Args[1:])
}
