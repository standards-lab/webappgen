// Package starter is the starter app every kind of project begins from:
// one tree, embedded in the binary, that [project.Write] writes first and
// each kind lays its own files over. It holds the app's source, its
// toolchain and check (mise.toml), its packages, the layers among them by
// exact release tag, and the lockfile that pins them. Its templates read
// two values, .Name, the project's package name, and .Version, the
// webappgen version that wrote it; each kind's own templates read the rest
// of its values.
package starter

import (
	"embed"
	"io/fs"
)

//go:embed all:files
var files embed.FS

// Files is the starter tree, rooted at its own top, so its paths are the
// written project's.
var Files fs.FS = mustSub(files, "files")

// mustSub returns fsys's subtree at dir, which the embed above guarantees.
func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
