package starter_test

import (
	"testing"

	"github.com/standards-lab/webappgen/internal/project/projecttest"
	"github.com/standards-lab/webappgen/internal/starter"
)

// The starter embeds only what git tracks in files/: never node_modules or
// a build a demo run left there.
func TestFiles_Tracked(t *testing.T) {
	projecttest.CheckTracked(t, starter.Files, "files")
}
