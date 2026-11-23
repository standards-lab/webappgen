// Adapted from go-cli-sdk-template (a4d314c): template/internal/app/domain.go, mounting new.

package app

import (
	"github.com/standards-lab/go-cli-sdk"
	"github.com/standards-lab/go-core/graph"

	"github.com/standards-lab/webappgen/internal/standalone"
)

func defineDomain(g *graph.Graph, n *Nodes) {}

func mountDomain(root *cli.Command, n *Nodes) {
	root.Add(newCommand())
}

func newCommand() *cli.Command {
	return (&cli.Command{
		Name:    "new",
		Summary: "Write a new web client project of one kind",
	}).Add(standalone.Command(moduleVersion()))
}
