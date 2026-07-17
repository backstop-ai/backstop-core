package main

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// TestCountToolchainPacks_ByDeclarationNotName proves countToolchainPacks (the
// SPEC-040 enforcement-configured signal) keys on a DECLARED mechanism engine, not the
// "-toolchain" name convention — so a toolchain pack from any org, under any name,
// counts, and a "-toolchain"-named pack that declares no mechanism engine does not.
func TestCountToolchainPacks_ByDeclarationNotName(t *testing.T) {
	// Non-conventionally-named pack that DECLARES a mechanism engine -> counts.
	nonConventional := &pack.Manifest{
		Name:           "acme/rust-tooling",
		NormalizedName: "acme/rust-tooling",
		Engines: map[string]pack.EngineSpec{
			"test": {Binding: engine.EngineBinding{GateType: engine.GateTypeTest}},
		},
	}
	if got := countToolchainPacks([]*pack.Manifest{nonConventional}); got != 1 {
		t.Fatalf("a non-'-toolchain'-named pack declaring a mechanism engine must count as a toolchain pack, got %d", got)
	}

	// "-toolchain"-NAMED pack that declares NO mechanism engine -> does NOT count
	// (the name alone is no longer sufficient).
	nameOnly := &pack.Manifest{Name: "backstop/fake-toolchain", NormalizedName: "backstop/fake-toolchain"}
	if got := countToolchainPacks([]*pack.Manifest{nameOnly}); got != 0 {
		t.Fatalf("a '-toolchain'-named pack declaring no mechanism engine must NOT count, got %d", got)
	}

	// A rules-only pack (findings, e.g. secrets) is NOT a toolchain pack.
	rulesOnly := &pack.Manifest{
		Name:           "backstop/secrets",
		NormalizedName: "backstop/secrets",
		Engines: map[string]pack.EngineSpec{
			"gitleaks": {Binding: engine.EngineBinding{GateType: engine.GateTypeFindings}},
		},
	}
	if got := countToolchainPacks([]*pack.Manifest{rulesOnly}); got != 0 {
		t.Fatalf("a findings-only rules pack must NOT count as a toolchain pack, got %d", got)
	}
}
