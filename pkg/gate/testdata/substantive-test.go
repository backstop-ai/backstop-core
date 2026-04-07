package testdata

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
)

// TestSubstantiveExample is a test function with assertions and a target package call.
func TestSubstantiveExample(t *testing.T) {
	result := gate.NewGateResult(nil)
	if !result.Pass {
		t.Fatal("expected pass to be true for empty gate result")
	}
}
