package starter_test

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/standards-lab/webappgen/internal/project"
	"github.com/standards-lab/webappgen/internal/project/projecttest"
	"github.com/standards-lab/webappgen/internal/starter"
)

// The starter embeds only what git tracks in files/: never node_modules or
// a build a demo run left there.
func TestFiles_Tracked(t *testing.T) {
	projecttest.CheckTracked(t, starter.Files, "files")
}

// toolPins returns each entry of a mise.toml's tool tables, [tools] and
// every [tools."<tool>"...] table, keyed by its table and key, with its
// value as written, so one tool's whole pin compares: its version, its
// options, and, for an http: tool, every platform's url, checksum, and
// size. It reads the one-line entries mise.toml holds, no more of TOML.
func toolPins(config string) map[string]string {
	pins := map[string]string{}
	var table string
	for line := range strings.Lines(config) {
		line = strings.TrimSpace(line)
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "["):
			table = line
		case table == "[tools]" || strings.HasPrefix(table, "[tools."):
			key, value, _ := strings.Cut(line, " = ")
			pins[table+" "+key] = value
		}
	}
	return pins
}

// Every tool the starter's mise.toml pins, the CLI's own root mise.toml
// pins the same: each entry the starter writes for it, rendered, equals the
// root's, so the root's mise install installs every tool a written project
// installs, and the integration tier's full proof installs one offline.
// The root may pin tools the starter doesn't, such as go.
func TestFiles_PinsTheCLIsTools(t *testing.T) {
	dir := t.TempDir()
	values := struct{ Name, Version string }{Name: "my-app", Version: "(devel)"}
	if err := project.Write(dir, values, starter.Files); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(filepath.Join(dir, "mise.toml"))
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.ReadFile(filepath.Join("..", "..", "mise.toml"))
	if err != nil {
		t.Fatal(err)
	}

	starterPins, rootPins := toolPins(string(written)), toolPins(string(root))
	if len(starterPins) == 0 {
		t.Fatal("the starter's mise.toml pins no tools, want its toolchain")
	}
	for _, entry := range slices.Sorted(maps.Keys(starterPins)) {
		got, ok := rootPins[entry]
		switch {
		case !ok:
			t.Errorf("mise.toml: %s isn't pinned; want %s, as the starter pins it", entry, starterPins[entry])
		case got != starterPins[entry]:
			t.Errorf("mise.toml: %s = %s, want %s, as the starter pins it", entry, got, starterPins[entry])
		}
	}
}
