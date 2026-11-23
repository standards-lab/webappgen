package serviceapp_test

import (
	"bytes"
	"go/format"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/standards-lab/webappgen/internal/serviceapp"
)

func copyFixture(t *testing.T, shape string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(filepath.Join("testdata", "wire", shape))); err != nil {
		t.Fatal(err)
	}
	return dir
}

func appFile(t *testing.T, dir, name string) string {
	t.Helper()
	return read(t, filepath.Join(dir, "internal", "app", name))
}

func read(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func writeAppFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "internal", "app", name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func comments(t *testing.T, src string) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, src)
	}
	var texts []string
	for _, group := range file.Comments {
		for _, c := range group.List {
			texts = append(texts, c.Text)
		}
	}
	return texts
}

const nodesComment = "// The web client (client.go): the build it serves and its handler."

func checkGoFile(t *testing.T, name, src string) {
	t.Helper()
	formatted, err := format.Source([]byte(src))
	if err != nil {
		t.Fatalf("%s doesn't parse: %v\n%s", name, err, src)
	}
	if !bytes.Equal(formatted, []byte(src)) {
		t.Errorf("%s isn't gofmt-clean:\n%s", name, src)
	}
}

func TestWire_EditsEachShape(t *testing.T) {
	for _, tc := range []struct {
		shape string
		opts  serviceapp.WireOptions
	}{
		{"template", serviceapp.WireOptions{Base: "/", Title: `Ops & "Console"`, APIBase: "/api/", ClientDir: "client"}},
		{"service", serviceapp.WireOptions{Base: "/app/", Title: "console", APIBase: "https://api.example.com/v1/", ClientDir: "web/My-UI"}},
	} {
		t.Run(tc.shape, func(t *testing.T) {
			dir := copyFixture(t, tc.shape)
			before := map[string]string{"app.go": appFile(t, dir, "app.go"), "server.go": appFile(t, dir, "server.go")}

			got, err := serviceapp.Wire(dir, "example.com/demo", tc.opts)
			if err != nil {
				t.Fatalf("Wire: %v", err)
			}

			if got.Client != "internal/app/client.go" {
				t.Errorf("Client = %q, want internal/app/client.go", got.Client)
			}
			if len(got.Left) != 0 {
				t.Errorf("Left = %+v, want none", got.Left)
			}
			var anchors []string
			for _, e := range got.Applied {
				anchors = append(anchors, e.Anchor)
				if !e.Found {
					t.Errorf("applied edit %+v isn't marked found", e)
				}
			}
			want := []string{serviceapp.AnchorImports, serviceapp.AnchorNodes, serviceapp.AnchorNew, serviceapp.AnchorRouter}
			if !slices.Equal(anchors, want) {
				t.Errorf("Applied anchors = %q, want %q", anchors, want)
			}
			golden := filepath.Join("testdata", "wire", tc.shape+".golden")
			for _, name := range []string{"app.go", "server.go", "client.go"} {
				content := appFile(t, dir, name)
				checkGoFile(t, name, content)
				if want := read(t, filepath.Join(golden, name)); content != want {
					t.Errorf("%s =\n%s\nwant %s/%s:\n%s", name, content, golden, name, want)
				}
			}
			for name, src := range before {
				got := slices.DeleteFunc(comments(t, appFile(t, dir, name)), func(c string) bool { return c == nodesComment })
				if want := comments(t, src); !slices.Equal(got, want) {
					t.Errorf("%s's comments, the Nodes edit's aside = %q, want them as before, %q", name, got, want)
				}
			}
		})
	}
}

func TestWire_MissingAnchorLeavesEveryEdit(t *testing.T) {
	for _, tc := range []struct {
		anchor   string
		file     string
		old, new string
	}{
		{serviceapp.AnchorNodes, "app.go", "type Nodes struct {", "type Parts struct {"},
		{serviceapp.AnchorNew, "app.go", "\tdefineServer(a.graph, &a.nodes)\n", ""},
		{serviceapp.AnchorRouter, "server.go", "return router, nil", "return router, check(router)"},
	} {
		t.Run(tc.anchor, func(t *testing.T) {
			dir := copyFixture(t, "template")
			content := appFile(t, dir, tc.file)
			if !strings.Contains(content, tc.old) {
				t.Fatalf("%s lacks %q", tc.file, tc.old)
			}
			writeAppFile(t, dir, tc.file, strings.Replace(content, tc.old, tc.new, 1))
			before := map[string]string{"app.go": appFile(t, dir, "app.go"), "server.go": appFile(t, dir, "server.go")}

			got, err := serviceapp.Wire(dir, "example.com/demo", serviceapp.WireOptions{
				Base: "/", Title: "demo", APIBase: "/api/", ClientDir: "client",
			})
			if err != nil {
				t.Fatalf("Wire: %v", err)
			}

			if len(got.Applied) != 0 {
				t.Errorf("Applied = %+v, want none", got.Applied)
			}
			want := []serviceapp.Edit{
				{Path: "internal/app/app.go", Anchor: serviceapp.AnchorImports, Found: true, Code: "\"io/fs\"\n\"net/http\""},
				{Path: "internal/app/app.go", Anchor: serviceapp.AnchorNodes, Found: true,
					Code: nodesComment + "\n" +
						"Dist *graph.Node[fs.FS]\nClient *graph.Node[http.Handler]"},
				{Path: "internal/app/app.go", Anchor: serviceapp.AnchorNew, Found: true, Code: "defineClient(a.graph, &a.nodes)"},
				{Path: "internal/app/server.go", Anchor: serviceapp.AnchorRouter, Found: true, Code: "mountClient(router, s, n)"},
			}
			for i := range want {
				if want[i].Anchor == tc.anchor {
					want[i].Found = false
				}
			}
			if tc.anchor == serviceapp.AnchorNodes {
				want[0].Found = false
				want[1].Code = "Dist *graph.Node[fs.FS]\nClient *graph.Node[http.Handler]"
			}
			if !slices.Equal(got.Left, want) {
				t.Errorf("Left =\n%+v\nwant\n%+v", got.Left, want)
			}
			for name, src := range before {
				if after := appFile(t, dir, name); after != src {
					t.Errorf("%s changed:\n%s", name, after)
				}
			}
			checkGoFile(t, "client.go", appFile(t, dir, "client.go"))
		})
	}
}

func TestWire_AliasesAClientPackageNameTaken(t *testing.T) {
	dir := copyFixture(t, "template")
	writeAppFile(t, dir, "routes.go", "package app\n\nfunc routes() {}\n")

	if _, err := serviceapp.Wire(dir, "example.com/demo", serviceapp.WireOptions{
		Base: "/", Title: "demo", APIBase: "/api/", ClientDir: "web/routes",
	}); err != nil {
		t.Fatalf("Wire: %v", err)
	}

	client := appFile(t, dir, "client.go")
	for _, want := range []string{
		"\twebclient \"example.com/demo/web/routes\"\n",
		"return webclient.Dist(), nil",
		"webclient.Entry",
		"RenderBlocking: webclient.RenderBlocking,",
	} {
		if !strings.Contains(client, want) {
			t.Errorf("client.go =\n%s\nwant it to hold %q", client, want)
		}
	}
}

func TestWire_KeepsImportsPresent(t *testing.T) {
	dir := copyFixture(t, "template")
	app := appFile(t, dir, "app.go")
	writeAppFile(t, dir, "app.go", strings.Replace(app, "\t\"io\"\n", "\t\"io\"\n\tiofs \"io/fs\"\n\t\"net/http\"\n", 1)+
		"\nvar _ iofs.FS\nvar _ http.Handler\n")

	got, err := serviceapp.Wire(dir, "example.com/demo", serviceapp.WireOptions{
		Base: "/", Title: "demo", APIBase: "/api/", ClientDir: "client",
	})
	if err != nil {
		t.Fatalf("Wire: %v", err)
	}

	if len(got.Applied) != 3 || got.Applied[0].Anchor != serviceapp.AnchorNodes {
		t.Fatalf("Applied = %+v, want the Nodes, New, and newRouter edits", got.Applied)
	}
	if want := "Dist *graph.Node[iofs.FS]"; !strings.Contains(got.Applied[0].Code, want) {
		t.Errorf("Nodes edit = %q, want it to hold %q", got.Applied[0].Code, want)
	}
	app = appFile(t, dir, "app.go")
	checkGoFile(t, "app.go", app)
	file, err := parser.ParseFile(token.NewFileSet(), "", app, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, spec := range file.Imports {
		paths = append(paths, spec.Path.Value)
	}
	if n := len(slices.DeleteFunc(paths, func(p string) bool { return p != `"io/fs"` && p != `"net/http"` })); n != 2 {
		t.Errorf("app.go imports io/fs and net/http %d times in all, want once each", n)
	}
}

func TestWire_Refuses(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, dir string)
		want  string
	}{
		{
			name:  "client file exists",
			setup: func(t *testing.T, dir string) { writeAppFile(t, dir, "client.go", "package app\n") },
			want:  "internal/app/client.go exists: remove it, or wire the client by hand",
		},
		{
			name:  "declares a client identifier",
			setup: func(t *testing.T, dir string) { writeAppFile(t, dir, "dist.go", "package app\n\nfunc newDist() {}\n") },
			want:  "internal/app already declares newDist, which client.go declares: rename it",
		},
		{
			name: "Nodes has Client",
			setup: func(t *testing.T, dir string) {
				app := appFile(t, dir, "app.go")
				writeAppFile(t, dir, "app.go", strings.Replace(app, "type Nodes struct {\n", "type Nodes struct {\n\tClient int\n", 1))
			},
			want: "internal/app/app.go: Nodes already has a Client field: wire the client by hand",
		},
		{
			name: "no composition root",
			setup: func(t *testing.T, dir string) {
				if err := os.RemoveAll(filepath.Join(dir, "internal")); err != nil {
					t.Fatal(err)
				}
			},
			want: "internal/app: open ",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := copyFixture(t, "template")
			tc.setup(t, dir)
			before := snapshot(t, dir)

			if err := serviceapp.CheckWire(dir); err == nil || !strings.HasPrefix(err.Error(), tc.want) {
				t.Errorf("CheckWire error = %v, want it to begin %q", err, tc.want)
			}
			_, err := serviceapp.Wire(dir, "example.com/demo", serviceapp.WireOptions{
				Base: "/", Title: "demo", APIBase: "/api/", ClientDir: "client",
			})

			if err == nil || !strings.HasPrefix(err.Error(), tc.want) {
				t.Errorf("Wire error = %v, want it to begin %q", err, tc.want)
			}
			if after := snapshot(t, dir); !maps.Equal(after, before) {
				t.Errorf("Wire changed the service: %q, want %q", slices.Sorted(maps.Keys(after)), slices.Sorted(maps.Keys(before)))
			}
		})
	}
}

func TestCheckWire_Accepts(t *testing.T) {
	for _, tc := range []struct {
		name     string
		shape    string
		old, new string
	}{
		{name: "template", shape: "template"},
		{name: "go-web-service", shape: "service"},
		{name: "anchor missing", shape: "template", old: "return router, nil", new: "return router, check(router)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := copyFixture(t, tc.shape)
			if tc.old != "" {
				writeAppFile(t, dir, "server.go", strings.Replace(appFile(t, dir, "server.go"), tc.old, tc.new, 1))
			}
			before := snapshot(t, dir)

			if err := serviceapp.CheckWire(dir); err != nil {
				t.Errorf("CheckWire error = %v, want nil", err)
			}
			if after := snapshot(t, dir); !maps.Equal(after, before) {
				t.Errorf("CheckWire changed the service: %q, want %q", slices.Sorted(maps.Keys(after)), slices.Sorted(maps.Keys(before)))
			}
		})
	}
}

func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		files[p] = read(t, p)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
