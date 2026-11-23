package project

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// CheckDir refuses, naming the fix, a dir a project can't be written into:
// one that exists and isn't a directory, one that exists and isn't empty,
// or one that can't be read. A dir that doesn't exist, or is an empty
// directory, passes.
func CheckDir(dir string) error {
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

// WriteNextSteps writes to w the steps left to run after a project is
// written into dir: a cd into dir, quoted as [ShellQuote] quotes it, then
// each of steps, one to a line.
func WriteNextSteps(w io.Writer, dir string, steps ...string) error {
	_, err := fmt.Fprintf(w, "Next steps:\n  cd %s\n  %s\n", ShellQuote(dir), strings.Join(steps, "\n  "))
	return err
}

// ShellQuote returns s quoted so it pastes into a POSIX shell as one word,
// as written: s itself when it is non-empty and holds only letters, digits,
// and /._-+:@,=, and otherwise s in single quotes, each of its own single
// quotes written as a backslash-escaped quote between two quoted runs.
func ShellQuote(s string) string {
	plain := func(r rune) bool {
		return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9' || strings.ContainsRune("/._-+:@,=", r)
	}
	if s != "" && strings.TrimLeftFunc(s, plain) == "" {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
