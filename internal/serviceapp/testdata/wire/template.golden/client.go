// Adapted from spike-html-apps (2883d78): internal/app/client.go, in the template's node style.

package app

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/standards-lab/go-core/graph"
	"github.com/standards-lab/go-web-sdk"

	"example.com/demo/client"
)

const (
	clientBase    = "/"
	clientTitle   = "Ops & \"Console\""
	clientAPIBase = "/api/"
)

func defineClient(g *graph.Graph, n *Nodes) {
	n.Dist = g.Define("dist", newDist)
	n.Client = g.Define("client", n.newClient)
}

func newDist(*graph.Scope) (fs.FS, error) {
	return client.Dist(), nil
}

func (n *Nodes) newClient(s *graph.Scope) (http.Handler, error) {
	return web.NewClient(s.Use(n.Dist), client.Entry, web.ClientOptions{
		Base:           clientBase,
		Title:          clientTitle,
		RenderBlocking: client.RenderBlocking,
		Config:         map[string]string{"apiBase": clientAPIBase},
	})
}

func mountClient(router *web.Router, s *graph.Scope, n *Nodes) {
	handler := s.Use(n.Client)
	if clientBase == "/" {
		router.SetNotFound(handler)
		return
	}
	router.Mount(web.NewHandlerModule(strings.TrimSuffix(clientBase, "/"), handler))
}
