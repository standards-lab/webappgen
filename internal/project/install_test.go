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

var steps = []project.Step{{"mise", "install"}, {"mise", "run", "install"}}

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

func TestInstall_RunsTheStepsInOrder(t *testing.T) {
	dir := t.TempDir()
	log := fakeMise(t, "", 0)
	var w bytes.Buffer

	if err := project.Install(context.Background(), dir, &w, steps...); err != nil {
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

func TestInstall_MiseMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	var w bytes.Buffer

	err := project.Install(context.Background(), dir, &w, steps...)

	want := "mise install: mise isn't on PATH: install mise (https://mise.jdx.dev), then run mise install and mise run install in " + dir
	if err == nil || err.Error() != want {
		t.Errorf("Install() error = %v, want %q", err, want)
	}
	if w.Len() != 0 {
		t.Errorf("output = %q, want none", w.String())
	}
}

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

			err := project.Install(context.Background(), dir, new(bytes.Buffer), steps...)

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

func TestInstall_ProgramMissing(t *testing.T) {
	dir := t.TempDir()
	log := fakeMise(t, "", 0)
	var w bytes.Buffer

	err := project.Install(context.Background(), dir, &w,
		project.Step{"mise", "install"}, project.Step{"go", "mod", "download"}, project.Step{"mise", "-C", "web ui", "install"})

	want := "go mod download: go isn't on PATH: install Go (https://go.dev/dl), " +
		"then run mise install and go mod download and mise -C 'web ui' install in " + dir
	if err == nil || err.Error() != want {
		t.Errorf("Install() error = %v, want %q", err, want)
	}
	if got := runs(t, log); got != nil {
		t.Errorf("mise runs = %q, want none", got)
	}
	if w.Len() != 0 {
		t.Errorf("output = %q, want none", w.String())
	}
}

func TestStep_String(t *testing.T) {
	if got, want := (project.Step{"mise", "-C", "it's", "run", "install"}).String(), `mise -C 'it'\''s' run install`; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
