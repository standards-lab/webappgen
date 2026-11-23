package projecttest

import (
	"io/fs"
	"os/exec"
	"path"
	"slices"
	"strings"
	"testing"
)

// CheckTracked fails t unless fsys, a tree embedded from dir, a directory
// relative to the test's package, holds exactly the files git tracks
// there: no file a working copy leaves beside them, such as node_modules
// or a build in dist, and none the commit lacks. It also fails a path with
// a node_modules or dist element even when git tracks it. It runs git, so
// it needs the repository's checkout.
func CheckTracked(t *testing.T, fsys fs.FS, dir string) {
	t.Helper()
	var embedded []string
	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		embedded = append(embedded, name)
		for elem := range strings.SplitSeq(name, "/") {
			if elem == "node_modules" || elem == "dist" {
				t.Errorf("embedded %s, under %s: an install's or a build's output, never the tree's", name, elem)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("git", "ls-files", "-z", "--", dir).Output()
	if err != nil {
		t.Fatalf("git ls-files %s: %v", dir, err)
	}
	var tracked []string
	for name := range strings.SplitSeq(strings.TrimSuffix(string(out), "\x00"), "\x00") {
		if rel, ok := strings.CutPrefix(name, path.Clean(dir)+"/"); ok {
			tracked = append(tracked, rel)
		}
	}
	slices.Sort(embedded)
	slices.Sort(tracked)
	for _, name := range embedded {
		if _, found := slices.BinarySearch(tracked, name); !found {
			t.Errorf("embedded %s/%s, which git doesn't track", dir, name)
		}
	}
	for _, name := range tracked {
		if _, found := slices.BinarySearch(embedded, name); !found {
			t.Errorf("git tracks %s/%s, which the tree doesn't embed", dir, name)
		}
	}
}
