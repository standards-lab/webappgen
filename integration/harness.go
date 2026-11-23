package integration

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/process/processtest"
)

const envPrefix = "WEBAPPGEN_"

// Main is the suite's TestMain: it unsets every WEBAPPGEN_ variable the
// suite inherited, so each run reads only the program's defaults and the
// overrides its test passes, then builds cmd/webappgen once, with the race
// detector so the program runs under it too, runs the tests, and removes
// the build.
func Main(m *testing.M) {
	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); strings.HasPrefix(name, envPrefix) {
			_ = os.Unsetenv(name)
		}
	}
	processtest.Main(m, "./cmd/webappgen")
}

// Run runs the built program once with args, the arguments without the
// program name, on empty standard input, and returns its output and exit
// code. A nonzero code is a result, not a failure. The program runs in the
// module root, so a test gives it absolute paths, under its own
// t.TempDir.
func Run(t testing.TB, args ...string) processtest.Result {
	t.Helper()
	return processtest.Run(t, processtest.Cmd{Args: args})
}

// RunEnv is [Run] with env, KEY=VALUE overrides added to the environment
// the program inherits, such as a PATH without mise, or the offline
// settings --install's steps read.
func RunEnv(t testing.TB, env []string, args ...string) processtest.Result {
	t.Helper()
	return processtest.Run(t, processtest.Cmd{Args: args, Env: env})
}

// Tree returns every path under root, relative to it and slash-separated,
// in lexical order, directories included, or nil when root doesn't exist,
// so a test compares what a run left in a directory with what was there
// before it.
func Tree(t testing.TB, root string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(root, func(p string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p != root {
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			paths = append(paths, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("walk %s: %v", root, err)
	}
	return paths
}
