package serviceapp

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/template"
)

// AppDir is the service's composition root, the go-web-sdk-template
// package [Wire] writes the client file into and edits, slash-separated and
// relative to the service's root.
const AppDir = "internal/app"

// ClientFile is the file [Wire] writes into [AppDir].
const ClientFile = "client.go"

//go:embed wiring/client.go.tmpl
var clientFile string

var clientTemplate = template.Must(template.New(ClientFile).Parse(clientFile))

var (
	clientDecls   = []string{"clientBase", "clientTitle", "clientAPIBase", "defineClient", "newDist", "newClient", "mountClient"}
	clientImports = []string{"fs", "http", "strings", "graph", "web"}
)

// The anchors [Wire] places its edits against, as [Edit.Anchor] names them.
const (
	AnchorImports = "the import declaration"
	AnchorNodes   = "type Nodes struct"
	AnchorNew     = "defineServer(...) in New"
	AnchorRouter  = "return router, nil in newRouter"
)

// WireOptions are the client [Wire] wires into a service.
type WireOptions struct {
	// Base is the absolute path the client is served under, as
	// [standalone.ParseBase] returns it.
	Base string
	// Title is the shell's <title>.
	Title string
	// APIBase is the URL of the API the client calls, which the shell
	// writes into the page's #app-config as apiBase.
	APIBase string
	// ClientDir is the client's directory in the service, as
	// [ParseClientDir] returns it.
	ClientDir string
}

// Edit is one insertion [Wire] makes in an existing Go file of the
// service's composition root.
type Edit struct {
	// Path is the file, slash-separated and relative to the service's root.
	Path string
	// Anchor names the code the edit is placed against, one of the Anchor
	// constants.
	Anchor string
	// Found is whether the file holds the anchor.
	Found bool
	// Code is the Go source inserted: the Nodes fields at the end of the
	// struct, the import specs into the import declaration, and each call
	// before the anchor's statement.
	Code string
}

// Wiring is what [Wire] did.
type Wiring struct {
	// Client is the client file written, slash-separated and relative to
	// the service's root.
	Client string
	// Applied are the edits made, every one when each anchor was found.
	Applied []Edit
	// Left are the edits not made, every one when any anchor was missing.
	Left []Edit
}

// Wire wires the web client in opts.ClientDir into the go-web-sdk-template
// service in dir, whose module is module. It writes [ClientFile] into
// [AppDir]: the dist and client graph nodes, the build of the client's Go
// package and go-web-sdk's web.NewClient over it, with the options' base,
// title, and API base as constants, and mountClient, which mounts the
// handler as the router's not-found handler at base / and as a handler
// module at the base without its trailing slash otherwise.
//
// It then edits the existing files at four anchors, by byte offset into
// their go/parser parse, and formats each with go/format, so every comment
// stays in place: Nodes gains Dist and Client, the import declaration
// io/fs and net/http when it lacks them, New calls defineClient before
// defineServer, and newRouter calls mountClient before it returns the
// router. When any anchor is missing it edits no existing file and returns
// every edit in [Wiring.Left], each marked with whether its anchor was
// found.
//
// It returns an error, writing nothing, when [AppDir] holds no Go file,
// already holds [ClientFile], or declares an identifier the client file
// declares, or Nodes already has a Dist or Client field.
func Wire(dir, module string, opts WireOptions) (Wiring, error) {
	appDir := filepath.Join(dir, filepath.FromSlash(AppDir))
	clientPath := filepath.Join(appDir, ClientFile)
	if _, err := os.Stat(clientPath); err == nil {
		return Wiring{}, fmt.Errorf("%s/%s exists: remove it, or wire the client by hand", AppDir, ClientFile)
	}
	pkg, err := parseApp(appDir)
	if err != nil {
		return Wiring{}, err
	}
	for _, name := range clientDecls {
		if pkg.decls[name] {
			return Wiring{}, fmt.Errorf("%s already declares %s, which %s declares: rename it", AppDir, name, ClientFile)
		}
	}

	source, err := renderClient(pkg, module, opts)
	if err != nil {
		return Wiring{}, err
	}
	edits, err := locate(pkg)
	if err != nil {
		return Wiring{}, err
	}

	result := Wiring{Client: path.Join(AppDir, ClientFile)}
	found := !slices.ContainsFunc(edits, func(e located) bool { return !e.Found })
	edited := map[string][]byte{}
	if found {
		if edited, err = apply(pkg, edits); err != nil {
			return Wiring{}, err
		}
	}
	if err := os.WriteFile(clientPath, source, 0o644); err != nil {
		return Wiring{}, err
	}
	for _, name := range slices.Sorted(maps.Keys(edited)) {
		if err := os.WriteFile(filepath.Join(appDir, name), edited[name], 0o644); err != nil {
			return Wiring{}, err
		}
	}
	for _, e := range edits {
		if found {
			result.Applied = append(result.Applied, e.Edit)
		} else {
			result.Left = append(result.Left, e.Edit)
		}
	}
	return result, nil
}

type appFile struct {
	name string
	src  []byte
	file *ast.File
}

type appPackage struct {
	fset  *token.FileSet
	name  string
	files []appFile
	decls map[string]bool
}

func parseApp(appDir string) (*appPackage, error) {
	entries, err := os.ReadDir(appDir)
	if err != nil {
		return nil, fmt.Errorf("%s: %w: wire the client into a service copied from go-web-sdk-template", AppDir, err)
	}
	pkg := &appPackage{fset: token.NewFileSet(), decls: map[string]bool{}}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(appDir, name))
		if err != nil {
			return nil, err
		}
		file, err := parser.ParseFile(pkg.fset, name, src, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", AppDir, err)
		}
		pkg.name = file.Name.Name
		pkg.files = append(pkg.files, appFile{name: name, src: src, file: file})
		for _, decl := range file.Decls {
			for _, name := range declNames(decl) {
				pkg.decls[name] = true
			}
		}
	}
	if len(pkg.files) == 0 {
		return nil, fmt.Errorf("%s holds no Go file: wire the client into a service copied from go-web-sdk-template", AppDir)
	}
	return pkg, nil
}

func declNames(decl ast.Decl) []string {
	var names []string
	switch d := decl.(type) {
	case *ast.FuncDecl:
		names = append(names, d.Name.Name)
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				names = append(names, s.Name.Name)
			case *ast.ValueSpec:
				for _, n := range s.Names {
					names = append(names, n.Name)
				}
			}
		}
	}
	return names
}

func renderClient(pkg *appPackage, module string, opts WireOptions) ([]byte, error) {
	name := PackageName(opts.ClientDir)
	importName := ""
	taken := func(n string) bool {
		return pkg.decls[n] || slices.Contains(clientDecls, n) || slices.Contains(clientImports, n)
	}
	if taken(name) {
		importName = "webclient"
		for i := 2; taken(importName); i++ {
			importName = "webclient" + strconv.Itoa(i)
		}
		name = importName
	}
	var buf bytes.Buffer
	err := clientTemplate.Execute(&buf, map[string]string{
		"Package":    pkg.name,
		"Import":     module + "/" + opts.ClientDir,
		"ImportName": importName,
		"Name":       name,
		"Base":       opts.Base,
		"Title":      opts.Title,
		"APIBase":    opts.APIBase,
	})
	if err != nil {
		return nil, err
	}
	return format.Source(buf.Bytes())
}

type located struct {
	Edit
	file   string
	offset int
	insert string
}

func locate(pkg *appPackage) ([]located, error) {
	nodes := located{Path: path.Join(AppDir, "app.go"), Anchor: AnchorNodes}
	imports := located{Path: nodes.Path, Anchor: AnchorImports}
	call := located{Path: nodes.Path, Anchor: AnchorNew, Code: "defineClient(a.graph, &a.nodes)"}
	mount := located{Path: path.Join(AppDir, "server.go"), Anchor: AnchorRouter, Code: "mountClient(router, s, n)"}

	for _, f := range pkg.files {
		for _, decl := range f.file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				st := nodesStruct(d)
				if st == nil {
					continue
				}
				if err := locateNodes(pkg, f, st, &nodes, &imports); err != nil {
					return nil, err
				}
			case *ast.FuncDecl:
				switch {
				case d.Recv == nil && d.Name.Name == "New":
					locateNew(pkg, f, d, &call)
				case d.Name.Name == "newRouter":
					locateRouter(pkg, f, d, &mount)
				}
			}
		}
	}
	if !nodes.Found {
		nodes.Code = nodesCode(false, "fs", "http")
		imports.Code = `"io/fs"` + "\n" + `"net/http"`
	}
	edits := []located{imports, nodes, call, mount}
	if imports.Found && imports.Code == "" {
		edits = edits[1:]
	}
	return edits, nil
}

func nodesStruct(d *ast.GenDecl) *ast.StructType {
	if d.Tok != token.TYPE {
		return nil
	}
	for _, spec := range d.Specs {
		if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == "Nodes" {
			if st, ok := ts.Type.(*ast.StructType); ok {
				return st
			}
		}
	}
	return nil
}

func nodesCode(grouped bool, fs, http string) string {
	code := ""
	if grouped {
		code = "// The web client (client.go): the build it serves and its handler.\n"
	}
	return code + "Dist *graph.Node[" + fs + ".FS]\nClient *graph.Node[" + http + ".Handler]"
}

func locateNodes(pkg *appPackage, f appFile, st *ast.StructType, nodes, imports *located) error {
	grouped := false
	for _, field := range st.Fields.List {
		for _, n := range field.Names {
			if n.Name == "Dist" || n.Name == "Client" {
				return fmt.Errorf("%s: Nodes already has a %s field: wire the client by hand", path.Join(AppDir, f.name), n.Name)
			}
		}
		grouped = grouped || field.Doc != nil
	}

	fsName, fsFound := importName(f.file, "io/fs")
	httpName, httpFound := importName(f.file, "net/http")
	nodes.Path, nodes.Found = path.Join(AppDir, f.name), true
	nodes.Code = nodesCode(grouped, fsName, httpName)
	nodes.file = f.name
	nodes.offset = pkg.fset.Position(st.Fields.Closing).Offset
	nodes.insert = "\n" + nodes.Code + "\n"

	var specs []string
	if !fsFound {
		specs = append(specs, `"io/fs"`)
	}
	if !httpFound {
		specs = append(specs, `"net/http"`)
	}
	imports.Path, imports.Found, imports.file = nodes.Path, true, f.name
	imports.Code = strings.Join(specs, "\n")
	if len(specs) == 0 {
		return nil
	}
	imports.offset, imports.insert = importOffset(pkg, f.file, imports.Code)
	return nil
}

func importName(file *ast.File, importPath string) (string, bool) {
	for _, spec := range file.Imports {
		if p, _ := strconv.Unquote(spec.Path.Value); p == importPath {
			if spec.Name != nil {
				return spec.Name.Name, true
			}
			return path.Base(importPath), true
		}
	}
	return path.Base(importPath), false
}

func importOffset(pkg *appPackage, file *ast.File, code string) (int, string) {
	for _, decl := range file.Decls {
		d, ok := decl.(*ast.GenDecl)
		if !ok || d.Tok != token.IMPORT || !d.Lparen.IsValid() {
			continue
		}
		for _, spec := range d.Specs {
			p, _ := strconv.Unquote(spec.(*ast.ImportSpec).Path.Value)
			if first, _, _ := strings.Cut(p, "/"); !strings.Contains(first, ".") {
				return pkg.fset.Position(spec.Pos()).Offset, code + "\n"
			}
		}
		return pkg.fset.Position(d.Lparen).Offset + 1, "\n" + code + "\n"
	}
	return pkg.fset.Position(file.Name.End()).Offset, "\n\nimport (\n" + code + "\n)"
}

func locateNew(pkg *appPackage, f appFile, fn *ast.FuncDecl, call *located) {
	if fn.Body == nil {
		return
	}
	for _, stmt := range fn.Body.List {
		expr, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}
		c, ok := expr.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		if id, ok := c.Fun.(*ast.Ident); !ok || id.Name != "defineServer" {
			continue
		}
		args := f.src[pkg.fset.Position(c.Lparen).Offset+1 : pkg.fset.Position(c.Rparen).Offset]
		call.Path, call.Found, call.file = path.Join(AppDir, f.name), true, f.name
		call.Code = "defineClient(" + string(args) + ")"
		call.offset = pkg.fset.Position(stmt.Pos()).Offset
		call.insert = call.Code + "\n"
		return
	}
}

func locateRouter(pkg *appPackage, f appFile, fn *ast.FuncDecl, mount *located) {
	if fn.Body == nil {
		return
	}
	scope := paramOf(fn.Type.Params, "Scope")
	nodes := paramOf(fn.Recv, "Nodes")
	if nodes == "" {
		nodes = paramOf(fn.Type.Params, "Nodes")
	}
	if scope == "" || nodes == "" {
		return
	}
	for _, stmt := range fn.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 2 {
			continue
		}
		router, ok := ret.Results[0].(*ast.Ident)
		if !ok {
			continue
		}
		if id, ok := ret.Results[1].(*ast.Ident); !ok || id.Name != "nil" {
			continue
		}
		mount.Path, mount.Found, mount.file = path.Join(AppDir, f.name), true, f.name
		mount.Code = "mountClient(" + router.Name + ", " + scope + ", " + nodes + ")"
		mount.offset = pkg.fset.Position(stmt.Pos()).Offset
		mount.insert = mount.Code + "\n"
		return
	}
}

func paramOf(fields *ast.FieldList, typeName string) string {
	if fields == nil {
		return ""
	}
	for _, field := range fields.List {
		star, ok := field.Type.(*ast.StarExpr)
		if !ok || len(field.Names) != 1 {
			continue
		}
		var name string
		switch t := star.X.(type) {
		case *ast.Ident:
			name = t.Name
		case *ast.SelectorExpr:
			name = t.Sel.Name
		}
		if name == typeName {
			return field.Names[0].Name
		}
	}
	return ""
}

func apply(pkg *appPackage, edits []located) (map[string][]byte, error) {
	byFile := map[string][]located{}
	for _, e := range edits {
		byFile[e.file] = append(byFile[e.file], e)
	}
	out := map[string][]byte{}
	for _, f := range pkg.files {
		fileEdits := byFile[f.name]
		if len(fileEdits) == 0 {
			continue
		}
		slices.SortFunc(fileEdits, func(a, b located) int { return b.offset - a.offset })
		src := slices.Clone(f.src)
		for _, e := range fileEdits {
			src = slices.Insert(src, e.offset, []byte(e.insert)...)
		}
		formatted, err := format.Source(src)
		if err != nil {
			return nil, errors.Join(fmt.Errorf("%s: the edited file doesn't parse", path.Join(AppDir, f.name)), err)
		}
		out[f.name] = formatted
	}
	return out, nil
}
