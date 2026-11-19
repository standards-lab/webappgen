// Adapted from go-cli-sdk-template (a4d314c): template/internal/app/doc.go, for webappgen's layers.

// Package app is webappgen's composition root. [New] describes the
// program's dependencies as one go-core graph and builds the command tree
// on go-cli-sdk over it; [App.Run] runs that tree, as a go-cli-sdk
// cli.Program, on the process arguments. Each architecture layer has one
// file, which defines the layer's nodes in its define function and owns
// its mount, so the layer files are the architecture's layer list:
//
//   - app.go: [App], [New], and [Nodes], which call the layer files'
//     define and mount functions in layer order, lowest first;
//   - infrastructure.go: defineInfrastructure, the configuration, read from
//     WEBAPPGEN_ variables, and the services the program is composed on
//     (the lifecycle configuration alone);
//   - domain.go: defineDomain and mountDomain, the domain services (none)
//     and the new command, whose leaves are the kinds of project, each
//     its own package, such as internal/standalone;
//   - version.go: the version command, which declares no nodes.
//
// [Nodes] holds one handle per node: each define function fills its own
// part of one Nodes value, and its constructors read the lower layers'
// nodes from it with Use. Each field's node is named for the field, so no
// comment names it; graph_test.go checks that rule with go-core's
// graphtest.CheckNames, and checks that every leaf of the tree, as
// Command.Leaves yields it, builds its nodes without dropping an order
// between layers. Defining the nodes constructs nothing. A command
// declares the nodes it needs with Use; the dispatcher builds them, with
// the lifecycle configuration node, only when the command runs, starts
// what was built layer by layer, and shuts it down in reverse when the
// command returns. A run that builds nothing, such as help, a usage error,
// version, or a kind's leaf, reads no configuration.
//
// The App publishes its composition: [App.Graph], [App.Nodes], and
// [App.Root] return the graph, a handle on each of its nodes, and the
// command tree. The program itself never calls them; they are for a caller
// that observes what a run builds, Replaces a node's constructor with a
// substitute, or adds a command over the nodes, before the App runs.
package app
