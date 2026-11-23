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

func TestFiles_Tracked(t *testing.T) {
	projecttest.CheckTracked(t, starter.Files, "files")
}

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
