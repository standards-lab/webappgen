// Adapted from go-cli-sdk-template (a4d314c): template/integration/suite_test.go, renamed webappgen.

//go:build integration

package integration_test

import (
	"testing"

	"github.com/standards-lab/webappgen/integration"
)

// The suite's TestMain builds the program once for the run. The unit tier
// never enters here: without the tag the harness type-checks and needs no
// binary.
func TestMain(m *testing.M) { integration.Main(m) }
