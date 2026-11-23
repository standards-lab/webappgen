package project

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// A Step is one of a project's install steps: a program and its arguments,
// run in the project's directory.
type Step []string

// String returns the step as it pastes into a POSIX shell, each word quoted
// as [ShellQuote] quotes it, the form both [Install] and the next steps
// name it in.
func (s Step) String() string {
	words := make([]string, len(s))
	for i, word := range s {
		words[i] = ShellQuote(word)
	}
	return strings.Join(words, " ")
}

// StepNames returns the names of steps, as [Step.String] writes them.
func StepNames(steps []Step) []string {
	names := make([]string, len(steps))
	for i, s := range steps {
		names[i] = s.String()
	}
	return names
}

var installHints = map[string]string{
	"mise": "mise (https://mise.jdx.dev)",
	"go":   "Go (https://go.dev/dl)",
}

// Install runs steps, in order, in the project written into dir, the first
// of the next steps a run prints without it. It writes a line naming each
// step to w before running it, and gives the step w for both its output
// streams, as the steps' output is progress, not the command's result.
// Each step reads the environment Install was given, so a caller's MISE_,
// Go, and pnpm settings, such as offline modes, reach the steps.
//
// A step's program missing from PATH, checked before any step runs, or a
// step that fails, is an error that names the step and the steps left to
// run once it is fixed; nothing a step installed or the project holds is
// removed.
func Install(ctx context.Context, dir string, w io.Writer, steps ...Step) error {
	names := StepNames(steps)
	programs := make([]string, len(steps))
	for i, step := range steps {
		program, err := exec.LookPath(step[0])
		if err != nil {
			hint, ok := installHints[step[0]]
			if !ok {
				hint = step[0]
			}
			return fmt.Errorf("%s: %s isn't on PATH: install %s, then run %s in %s",
				names[i], step[0], hint, strings.Join(names, " and "), dir)
		}
		programs[i] = program
	}
	for i, step := range steps {
		if _, err := fmt.Fprintf(w, "Running %s in %s\n", names[i], dir); err != nil {
			return err
		}
		cmd := exec.CommandContext(ctx, programs[i], step[1:]...)
		cmd.Dir, cmd.Stdout, cmd.Stderr = dir, w, w
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w: fix what it reports, then run %s in %s",
				names[i], err, strings.Join(names[i:], " and "), dir)
		}
	}
	return nil
}
