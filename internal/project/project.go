package project

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

const templateSuffix = ".tmpl"

// Write writes the files of layers into dir, creating it and the
// directories its files need. The layers are laid in order, so a file a
// later layer holds at the same written path replaces an earlier one's,
// whether either is a template or not. A file named x.tmpl is a
// text/template, executed with values, and written as x; its templates may
// call json, which writes a value as JSON, a string quoted and escaped, with
// <, >, and & as they are, and html, text/template's own escaping for HTML
// text and attributes. A field values lacks is an error, never an empty
// string. Every other file is written byte for byte. Files are written 0644, and a .sh
// script 0755, as the embedded trees carry no modes.
//
// Every template is rendered before anything is written, so a template
// that fails writes nothing; a failed write can leave the files before it.
func Write(dir string, values any, layers ...fs.FS) error {
	files, err := render(values, layers)
	if err != nil {
		return err
	}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if path.Ext(name) == ".sh" {
			mode = 0o755
		}
		if err := os.WriteFile(target, files[name], mode); err != nil {
			return err
		}
	}
	return nil
}

func render(values any, layers []fs.FS) (map[string][]byte, error) {
	files := map[string][]byte{}
	for _, layer := range layers {
		err := fs.WalkDir(layer, ".", func(name string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			content, err := fs.ReadFile(layer, name)
			if err != nil {
				return err
			}
			if written, ok := strings.CutSuffix(name, templateSuffix); ok {
				if content, err = execute(name, content, values); err != nil {
					return err
				}
				name = written
			}
			files[name] = content
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

var funcs = template.FuncMap{"json": toJSON}

func execute(name string, text []byte, values any) ([]byte, error) {
	t, err := template.New(name).Funcs(funcs).Option("missingkey=error").Parse(string(text))
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, values); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func toJSON(v any) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", fmt.Errorf("json: %w", err)
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}
