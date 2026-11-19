// Package project writes a project from trees of files, each laid over the
// ones before it: the starter's, then a kind's own. A file whose name ends
// in .tmpl is a text/template, rendered with the kind's values and written
// without the suffix; every other file is copied byte for byte. Nothing is
// fetched: the trees are embedded in the binary, so one version names both
// the CLI and what it writes.
package project
