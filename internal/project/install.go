package project

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// installSteps are what [Install] runs in a written project, in order:
// mise installs the pins the project's mise.toml names, then the project's
// own install task installs the rest, its node_modules.
var installSteps = [][]string{
	{"install"},
	{"run", "install"},
}

// Install runs, in the project written into dir, mise install and then mise
// run install, the first two of the next steps a run prints without it. It
// writes a line naming each step to w before running it, and gives the
// step w for both its output streams, as the steps' output is progress,
// not the command's result. Each step reads the environment Install was
// given, so a caller's MISE_ and pnpm settings, such as offline modes,
// reach the steps.
//
// A mise missing from PATH, or a step that fails, is an error that names
// the step and the steps left to run once it is fixed; nothing a step
// installed or the project holds is removed.
func Install(ctx context.Context, dir string, w io.Writer) error {
	steps := make([]string, len(installSteps))
	for i, args := range installSteps {
		steps[i] = "mise " + strings.Join(args, " ")
	}
	mise, err := exec.LookPath("mise")
	if err != nil {
		return fmt.Errorf("%s: mise isn't on PATH: install mise (https://mise.jdx.dev), then run %s in %s",
			steps[0], strings.Join(steps, " and "), dir)
	}
	for i, args := range installSteps {
		if _, err := fmt.Fprintf(w, "Running %s in %s\n", steps[i], dir); err != nil {
			return err
		}
		cmd := exec.CommandContext(ctx, mise, args...)
		cmd.Dir, cmd.Stdout, cmd.Stderr = dir, w, w
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w: fix what it reports, then run %s in %s",
				steps[i], err, strings.Join(steps[i:], " and "), dir)
		}
	}
	return nil
}
