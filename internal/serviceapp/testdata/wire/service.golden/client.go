// Adapted from spike-html-apps (2883d78): internal/app/client.go, in the template's node style.

package app

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/standards-lab/go-core/graph"
	"github.com/standards-lab/go-web-sdk"

	"example.com/demo/web/My-UI"
)

const (
	clientBase    = "/app/"
	clientTitle   = "console"
	clientAPIBase = "https://api.example.com/v1/"
)

func defineClient(g *graph.Graph, n *Nodes) {
	n.Dist = g.Define("dist", newDist)
	n.Client = g.Define("client", n.newClient)
}

func newDist(*graph.Scope) (fs.FS, error) {
	return myui.Dist(), nil
}

func (n *Nodes) newClient(s *graph.Scope) (http.Handler, error) {
	return web.NewClient(s.Use(n.Dist), myui.Entry, web.ClientOptions{
		Base:           clientBase,
		Title:          clientTitle,
		RenderBlocking: myui.RenderBlocking,
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
