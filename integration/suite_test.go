//go:build integration

package integration_test

import (
	"testing"

	"github.com/standards-lab/webappgen/integration"
)

func TestMain(m *testing.M) { integration.Main(m) }
