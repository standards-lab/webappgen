package project_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/standards-lab/webappgen/internal/project"
)

// fakeMise puts, alone on PATH, a mise that appends its working directory
// and arguments to a log, one line a run, prints a line of output, and
// exits with code when its arguments are failArgs. It returns the log's
// path.
func fakeMise(t *testing.T, failArgs string, code int) string {
	t.Helper()
	bin := t.TempDir()
	log := filepath.Join(t.TempDir(), "mise.log")
	script := "#!/bin/sh\n" +
		`echo "$PWD $*" >>'` + log + "'\n" +
		`echo "mise $* output"` + "\n" +
		`if [ "$*" = '` + failArgs + `' ]; then exit ` + strconv.Itoa(code) + "; fi\n"
	if err := os.WriteFile(filepath.Join(bin, "mise"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	return log
}

// runs returns the lines the fake mise logged, or none when it never ran.
func runs(t *testing.T, log string) []string {
	t.Helper()
	b, err := os.ReadFile(log)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

// Install runs mise install, then mise run install, in dir, naming each
// step on w before the step's own output.
func TestInstall_RunsTheStepsInOrder(t *testing.T) {
	dir := t.TempDir()
	log := fakeMise(t, "", 0)
	var w bytes.Buffer

	if err := project.Install(context.Background(), dir, &w); err != nil {
		t.Fatalf("Install() error = %v; output = %q", err, w.String())
	}
	if got, want := runs(t, log), []string{dir + " install", dir + " run install"}; !slices.Equal(got, want) {
		t.Errorf("mise runs = %q, want %q", got, want)
	}
	want := "Running mise install in " + dir + "\nmise install output\n" +
		"Running mise run install in " + dir + "\nmise run install output\n"
	if w.String() != want {
		t.Errorf("output = %q, want %q", w.String(), want)
	}
}

// A mise missing from PATH fails the first step, naming both steps to run
// once mise is installed.
func TestInstall_MiseMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	var w bytes.Buffer

	err := project.Install(context.Background(), dir, &w)

	want := "mise install: mise isn't on PATH: install mise (https://mise.jdx.dev), then run mise install and mise run install in " + dir
	if err == nil || err.Error() != want {
		t.Errorf("Install() error = %v, want %q", err, want)
	}
	if w.Len() != 0 {
		t.Errorf("output = %q, want none", w.String())
	}
}

// A failing step stops the install, its error naming the step, its exit,
// and the steps left to run; the steps after it never run.
func TestInstall_StepFails(t *testing.T) {
	for _, tc := range []struct {
		failArgs string
		ran      []string
		want     string
	}{
		{
			failArgs: "install",
			ran:      []string{"install"},
			want:     "mise install: exit status 3: fix what it reports, then run mise install and mise run install in {dir}",
		},
		{
			failArgs: "run install",
			ran:      []string{"install", "run install"},
			want:     "mise run install: exit status 3: fix what it reports, then run mise run install in {dir}",
		},
	} {
		t.Run(tc.failArgs, func(t *testing.T) {
			dir := t.TempDir()
			log := fakeMise(t, tc.failArgs, 3)

			err := project.Install(context.Background(), dir, new(bytes.Buffer))

			if want := strings.ReplaceAll(tc.want, "{dir}", dir); err == nil || err.Error() != want {
				t.Errorf("Install() error = %v, want %q", err, want)
			}
			var want []string
			for _, args := range tc.ran {
				want = append(want, dir+" "+args)
			}
			if got := runs(t, log); !slices.Equal(got, want) {
				t.Errorf("mise runs = %q, want %q", got, want)
			}
		})
	}
}
