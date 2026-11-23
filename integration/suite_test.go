// Adapted from go-cli-sdk-template (a4d314c): template/integration/suite_test.go, renamed webappgen.

//go:build integration

package integration_test

import (
	"testing"

	"github.com/standards-lab/webappgen/integration"
)

func TestMain(m *testing.M) { integration.Main(m) }
