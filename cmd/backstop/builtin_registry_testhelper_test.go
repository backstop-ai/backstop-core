package main

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/baseengines"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// builtinTestRegistry returns the full built-in engine set the tests use as the
// `engineRegistry` seam after ISSUE-027 deleted engine.DefaultRegistry(): the four
// generic engines from the embedded base-engines pack MERGED with the go-toolchain
// pack's declared engines (go-build/go-test/golangci/go-coverage), loaded from the
// committed go-toolchain testdata fixture. This is the test-only reconstruction of
// the old baked registry — production resolveEngineRegistry seeds from
// baseengines.Registry() and merges each pack's own declared engines, so a test
// seam covering both the generic built-ins and the Go toolchain mirrors that union.
func builtinTestRegistry(t *testing.T) engine.Registry {
	t.Helper()
	reg := baseengines.Registry()
	for name, spec := range goToolchainManifest(t).Engines {
		reg[name] = spec.Binding
	}
	return reg
}
