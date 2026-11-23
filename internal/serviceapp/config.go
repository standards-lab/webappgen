package serviceapp

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/standards-lab/webappgen/internal/project"
)

// The configuration files [EditConfig] edits or writes, slash-separated and
// relative to the service's root.
const (
	MiseFile = "mise.toml"
	CIFile   = ".github/workflows/ci.yml"
	AirFile  = ".air.toml"
)

// AirTool is the pin [EditConfig] adds to the service's mise.toml, air,
// which its dev task runs, as the line it writes into [tools].
const AirTool = `"aqua:air-verse/air" = "1.67.4"`

// Insert is one insertion of a [ConfigEdit].
type Insert struct {
	// Anchor is the text the insertion is placed against, as the pinned
	// template has it, named by its first line or its job; empty for an
	// insertion at the end of the file, or of the whole file.
	Anchor string
	// Found is whether the file holds the anchor exactly once.
	Found bool
	// Placement is where the code goes, relative to the anchor.
	Placement string
	// Code is the text inserted.
	Code string
}

// ConfigEdit is what [EditConfig] did to one configuration file.
type ConfigEdit struct {
	// Path is the file, slash-separated and relative to the service's root.
	Path string
	// Applied is whether the file was edited, every insertion made.
	Applied bool
	// Defined are what the edit adds that the file already defines: a
	// task's table, or the file itself when the edit writes it whole.
	Defined []string
	// Inserts are the edit's insertions, made all together or left all
	// together.
	Inserts []Insert
}

// Reason returns why the edit was left, as a predicate of its file, such
// as "lacks [tasks.check] as go-web-sdk-template has it", or "" when it
// was applied.
func (e ConfigEdit) Reason() string {
	if e.Applied {
		return ""
	}
	var missing, reasons []string
	for _, in := range e.Inserts {
		if in.Anchor != "" && !in.Found {
			missing = append(missing, in.Anchor)
		}
	}
	if len(missing) > 0 {
		reasons = append(reasons, "lacks "+strings.Join(missing, ", ")+" as go-web-sdk-template has it")
	}
	for _, d := range e.Defined {
		if d == e.Path {
			reasons = append(reasons, "exists")
		} else {
			reasons = append(reasons, "already defines "+d)
		}
	}
	return strings.Join(reasons, ", and ")
}

type insertion struct {
	Insert
	head, tail string
	text       string
	table      bool
}

// EditConfig edits the service in dir, a go-web-sdk-template service with
// the web client in clientDir, so its tasks build the client and its CI
// builds it, and writes [AirFile] for its dev loop, returning one
// [ConfigEdit] for each file in [MiseFile], [CIFile], [AirFile] order.
//
// [MiseFile] gains [AirTool], at the start of [tools]; a client:build
// task, the client's mise run build run in its directory with mise -C,
// which check, test, integration, and serve depend on; a client:check
// task, the client's own check, which check depends on too; and a dev
// task, which runs the client's vite build --watch beside air. [CIFile]'s
// check and integration jobs gain steps that install the client, its tools
// and its packages, and build it, before they check or test. [AirFile],
// written whole, rebuilds the server into tmp/main on each Go, JS, and CSS
// change, outside the client's sources, tests, scripts, and node_modules.
//
// Each edit's anchors are the text the pinned template holds where the
// edit goes: the tables it adds a pin or a dependency to and the jobs it
// adds steps to. A file is edited only when it holds every anchor exactly
// once, each at the start of a line, and defines none of the tasks the
// edit adds; [AirFile] only when it doesn't exist. Otherwise the file is
// left as it is and its edit returned unapplied. It returns an error when
// a file can't be read or written.
func EditConfig(dir, clientDir string) ([]ConfigEdit, error) {
	var edits []ConfigEdit
	for _, plan := range []func(string, string) (ConfigEdit, string, error){planMise, planCI, planAir} {
		edit, content, err := plan(dir, clientDir)
		if err != nil {
			return nil, err
		}
		if edit.Applied {
			name := filepath.Join(dir, filepath.FromSlash(edit.Path))
			if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
				return nil, err
			}
		}
		edits = append(edits, edit)
	}
	return edits, nil
}

func readConfig(dir, name string) (string, bool, error) {
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	return string(data), err == nil, err
}

func planMise(dir, clientDir string) (ConfigEdit, string, error) {
	content, _, err := readConfig(dir, MiseFile)
	if err != nil {
		return ConfigEdit{}, "", err
	}
	quoted := project.ShellQuote(clientDir)
	tables := []string{
		"[tasks.\"client:build\"]\n" +
			"description = " + tomlString("Build the web client in "+clientDir+" with its own mise run build") + "\n" +
			"run = " + tomlString("mise -C "+quoted+" run build"),
		"[tasks.\"client:check\"]\n" +
			"description = " + tomlString("Run the web client's own check in "+clientDir+", after its build") + "\n" +
			"depends = [\"client:build\"]\n" +
			"run = " + tomlString("mise -C "+quoted+" run check"),
		"[tasks.dev]\n" +
			"description = " + tomlString("Run the dev loop: vite build --watch in "+clientDir+
			" beside air, which rebuilds and restarts the server on each Go, JS, and CSS change (.air.toml)") + "\n" +
			"depends = [\"client:build\"]\n" +
			"run = " + tomlMultiline("set -e\ntrap 'kill 0' EXIT INT TERM\n"+
			"(mise -C "+quoted+" exec -- pnpm exec vite build --watch) &\nair &\nwait\n"),
	}
	build := `depends = ["client:build"]`
	inserts := []insertion{
		tableInsert(toolsTable, AirTool, "add this pin at its start"),
		tableInsert(checkTable, `depends = ["client:build", "client:check"]`, "add this line after its header"),
		tableInsert(testTable, build, "add this line after its header"),
		tableInsert(integrationTable, build, "add this line after its header"),
		tableInsert(serveTable, build, "add this line after its header"),
		{Placement: "append these tables to it", Code: strings.Join(tables, "\n\n")},
	}
	var defined []string
	for _, header := range []string{`[tasks."client:build"]`, `[tasks."client:check"]`, "[tasks.dev]"} {
		if hasLine(content, header) {
			defined = append(defined, header)
		}
	}
	edit, written := planFile(MiseFile, content, defined, inserts)
	return edit, written, nil
}

func tableInsert(table, code, placement string) insertion {
	header, rest, _ := strings.Cut(table, "\n")
	return insertion{
		Anchor: header, Placement: placement, Code: code,
		head: header + "\n", tail: rest, text: code + "\n", table: true,
	}
}

func planCI(dir, clientDir string) (ConfigEdit, string, error) {
	content, _, err := readConfig(dir, CIFile)
	if err != nil {
		return ConfigEdit{}, "", err
	}
	mise := "mise -C " + project.ShellQuote(clientDir)
	steps := "      - name: Install the web client's tools and packages\n" +
		"        run: |\n" +
		"          " + mise + " install\n" +
		"          " + mise + " run install\n" +
		"\n" +
		"      - name: Build the web client\n" +
		"        run: |\n" +
		"          " + mise + " run build"
	inserts := []insertion{
		jobSteps("the check job", checkJob, "      - name: mise run check\n", steps),
		jobSteps("the integration job", integrationJob, "      - name: go test -tags integration\n", steps),
	}
	edit, written := planFile(CIFile, content, nil, inserts)
	return edit, written, nil
}

func jobSteps(name, job, before, steps string) insertion {
	head, _, _ := strings.Cut(job, before)
	return insertion{
		Anchor: name, Placement: "insert these steps before its " + strings.TrimPrefix(strings.TrimSpace(before), "- name: ") + " step", Code: steps,
		head: head, tail: job[len(head):], text: steps + "\n\n",
	}
}

func planAir(dir, clientDir string) (ConfigEdit, string, error) {
	_, exists, err := readConfig(dir, AirFile)
	if err != nil {
		return ConfigEdit{}, "", err
	}
	var defined []string
	if exists {
		defined = []string{AirFile}
	}
	excludes := []string{"tmp", "bin"}
	for _, sub := range []string{"node_modules", "src", "test", "scripts"} {
		excludes = append(excludes, path.Join(clientDir, sub))
	}
	for i, e := range excludes {
		excludes[i] = tomlString(e)
	}
	air := "# The service's live reload under air, which mise run dev runs.\n" +
		"# Adapted from herald (c697566): .air.toml, for the service's server and its web client.\n" +
		"root = \".\"\n" +
		"tmp_dir = \"tmp\"\n" +
		"\n" +
		"[build]\n" +
		"bin = \"./tmp/main\"\n" +
		"cmd = \"go build -o ./tmp/main ./cmd/server\"\n" +
		"delay = 1000\n" +
		"exclude_dir = [" + strings.Join(excludes, ", ") + "]\n" +
		"include_ext = [\"go\", \"js\", \"css\"]\n" +
		"kill_delay = 500\n" +
		"send_interrupt = true\n" +
		"\n" +
		"[log]\n" +
		"time = false\n" +
		"\n" +
		"[misc]\n" +
		"clean_on_exit = true"
	inserts := []insertion{{Placement: "write it with this text", Code: air}}
	edit, written := planFile(AirFile, "", defined, inserts)
	return edit, written, nil
}

func planFile(name, content string, defined []string, inserts []insertion) (ConfigEdit, string) {
	edit := ConfigEdit{Path: name, Defined: defined}
	applied := len(defined) == 0
	for i := range inserts {
		in := &inserts[i]
		in.Found = in.Anchor == "" || countAnchors(content, in.head+in.tail, in.table) == 1
		applied = applied && in.Found
		edit.Inserts = append(edit.Inserts, in.Insert)
	}
	if !applied {
		return edit, ""
	}
	for _, in := range inserts {
		if in.Anchor != "" {
			content = strings.Replace(content, in.head+in.tail, in.head+in.text+in.tail, 1)
			continue
		}
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if content != "" {
			content += "\n"
		}
		content += in.Code + "\n"
	}
	edit.Applied = true
	return edit, content
}

func countAnchors(content, text string, table bool) int {
	n := 0
	for i := 0; ; {
		j := strings.Index(content[i:], text)
		if j < 0 {
			return n
		}
		at := i + j
		if (at == 0 || content[at-1] == '\n') && (!table || endsTable(content[at+len(text):])) {
			n++
		}
		i = at + 1
	}
}

func endsTable(rest string) bool {
	for line := range strings.Lines(rest) {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			return strings.HasPrefix(line, "[")
		}
	}
	return true
}

func hasLine(content, line string) bool {
	for l := range strings.Lines(content) {
		if strings.TrimSpace(l) == line {
			return true
		}
	}
	return false
}

func tomlString(s string) string {
	return `"` + tomlEscape(s, false) + `"`
}

func tomlMultiline(s string) string {
	return `"""` + "\n" + tomlEscape(s, true) + `"""`
}

func tomlEscape(s string, multiline bool) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\n' && multiline:
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, `\u%04X`, r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func writeConfigLeft(w io.Writer, edits []ConfigEdit, after bool) error {
	var b strings.Builder
	for _, e := range edits {
		if e.Applied {
			continue
		}
		if after || b.Len() > 0 {
			b.WriteString("\n")
		}
		these := "these edits"
		if len(e.Inserts) == 1 {
			these = "this edit"
		}
		fmt.Fprintf(&b, "Edited no %s: it %s. Make %s by hand:\n", e.Path, e.Reason(), these)
		for _, in := range e.Inserts {
			if in.Anchor == "" {
				fmt.Fprintf(&b, "\n%s: %s\n", e.Path, in.Placement)
			} else {
				found := "found"
				if !in.Found {
					found = "missing"
				}
				fmt.Fprintf(&b, "\n%s, at %s (%s): %s\n", e.Path, in.Anchor, found, in.Placement)
			}
			for line := range strings.SplitSeq(in.Code, "\n") {
				if line == "" {
					b.WriteString("\n")
					continue
				}
				fmt.Fprintf(&b, "\t%s\n", line)
			}
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

const toolsTable = `[tools]
"github:standards-lab/claude-plugins" = { version = "0.22.0", version_prefix = "marathon/v" }
go = "1.27.4"
golangci-lint = "2.15.0"
jq = "1.8.2"
`

const checkTable = `[tasks.check]
description = "Build, vet, format, fix, tidy, test the unit tier, and lint every module, vetting, fixing, and linting the integration suite too, then run lint-context and lint-references at the project root and, when the project is nested, at the git top level; writes nothing"
run = '''
set -eu
export GOWORK=off
for mod in $GO_MODULES; do
  echo "check: $mod"
  cd "$MISE_PROJECT_ROOT/$mod"
  go build ./...
  go vet -tags integration ./...
  unformatted=$(gofmt -l .)
  if [ -n "$unformatted" ]; then echo "gofmt: unformatted files:"; echo "$unformatted"; exit 1; fi
  go fix -tags integration -diff ./...
  go mod tidy -diff
  go test -race ./...
  golangci-lint run --build-tags integration ./...
done
cd "$MISE_PROJECT_ROOT"
root=$(pwd -P)
top=$(git rev-parse --show-toplevel 2>/dev/null || echo "$root")
fail=0
for lint in lint-context lint-references; do
  if ! command -v "$lint" >/dev/null; then
    echo "FAIL: $lint is not on PATH; run mise install" >&2
    fail=1
    continue
  fi
  "$lint" || fail=1
  [ "$top" = "$root" ] || (cd "$top" && "$lint") || fail=1
done
exit "$fail"
'''
`

const serveTable = `[tasks.serve]
description = "Run the service locally"
run = "go run ./cmd/server"
`

const testTable = `[tasks.test]
description = "Run the unit tier"
run = "go test -race ./..."
`

const integrationTable = `[tasks.integration]
description = "Run the integration tier against the built service, as a subprocess"
run = "go test -race -count=1 -tags integration ./integration/"
`

const checkJob = `  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7.0.1

      - uses: jdx/mise-action@v5.2.10

      - name: mise run check
        run: mise run check
`

const integrationJob = `  integration:
    if: github.event_name != 'pull_request'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7.0.1

      - uses: jdx/mise-action@v5.2.10

      - name: go test -tags integration
        run: go test -race -count=1 -tags integration ./integration/
`
