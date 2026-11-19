// Adapted from go-cli-sdk-template (a4d314c): template/internal/app/domain.go, mounting new.

package app

import (
	"github.com/standards-lab/go-cli-sdk"
	"github.com/standards-lab/go-core/graph"

	"github.com/standards-lab/webappgen/internal/standalone"
)

// defineDomain defines the domain services on g into n, one node per
// domain layer. webappgen defines none: each kind's command checks its
// input and writes its project itself, and builds nothing.
func defineDomain(g *graph.Graph, n *Nodes) {}

// mountDomain mounts the domain's commands at root: new, with one leaf per
// kind of project, each declaring with Use the nodes it reads, so a run
// builds only what its command needs; no kind declares any yet.
func mountDomain(root *cli.Command, n *Nodes) {
	root.Add(newCommand())
}

// newCommand returns the new command, the parent of one leaf per kind of
// project. Run alone, it prints its help, which lists the kinds.
func newCommand() *cli.Command {
	return (&cli.Command{
		Name:    "new",
		Summary: "Write a new web client project of one kind",
	}).Add(standalone.Command())
}
