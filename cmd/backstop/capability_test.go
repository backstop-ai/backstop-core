package main

import (
	"os"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// packDeclaringGateType builds a minimal installed-pack manifest whose engines block
// declares ONE engine with the given gate_type. It carries an arbitrary NormalizedName
// so the tests can prove capability is keyed on the DECLARATION, never the name/org
// (ISSUE-063 REQ-001/REQ-003). The single engine is enough — packDeclaresGateType keys
// on any declared engine matching the dimension's gate_type.
func packDeclaringGateType(name string, gt engine.GateType) *pack.Manifest {
	return &pack.Manifest{
		NormalizedName: name,
		Engines: map[string]pack.EngineSpec{
			"eng": {Binding: engine.EngineBinding{GateType: gt}},
		},
	}
}

// gateTypeForDimension maps a traceability dimension to its engine gate_type for the
// table tests (the two spellings are identical strings, but the tests resolve through
// the parser so an accidental drift fails loudly rather than silently).
func gateTypeForDimension(t *testing.T, dim gate.TraceabilityDimension) engine.GateType {
	t.Helper()
	gt, err := engine.ParseGateType(string(dim))
	if err != nil {
		t.Fatalf("ParseGateType(%q): %v", dim, err)
	}
	return gt
}

// TestCapability_PresentWhenPackDeclaresGateType (CLM-001): a dimension's capability is
// present when SOME installed pack declares an engine whose gate_type equals the
// dimension — keyed on manifest.Engines[].GateType, never on the pack name. The provider
// pack carries a non-backstop, non-conventional name to prove the name is irrelevant.
func TestCapability_PresentWhenPackDeclaresGateType(t *testing.T) {
	for _, dim := range []gate.TraceabilityDimension{
		gate.DimensionContracts,
		gate.DimensionSubstantiveness,
		gate.DimensionCoverage,
	} {
		gt := gateTypeForDimension(t, dim)
		packs := []*pack.Manifest{
			packDeclaringGateType("acme/some-random-pack", gt),
		}
		if !packDeclaresGateType(packs, dim) {
			t.Errorf("dimension %q: want present (a pack declares gate_type %q), got absent", dim, dim)
		}
	}
}

// TestCapability_AbsentWhenNoPackDeclaresGateType (CLM-002): a dimension's capability is
// absent when NO installed pack declares an engine with the matching gate_type — an empty
// pack set, and a pack set whose only engine declares a DIFFERENT gate_type, both report
// absent (no name-based fallback).
func TestCapability_AbsentWhenNoPackDeclaresGateType(t *testing.T) {
	for _, dim := range []gate.TraceabilityDimension{
		gate.DimensionContracts,
		gate.DimensionSubstantiveness,
		gate.DimensionCoverage,
	} {
		if packDeclaresGateType(nil, dim) {
			t.Errorf("dimension %q: an empty pack set must be capability-absent", dim)
		}
		// A pack declaring only a lint engine provides none of the three traceability
		// dimensions — capability keys on the declared gate_type, not on the pack existing.
		lintOnly := []*pack.Manifest{packDeclaringGateType("acme/lint-pack", engine.GateTypeLint)}
		if packDeclaresGateType(lintOnly, dim) {
			t.Errorf("dimension %q: a pack declaring only a lint engine must be capability-absent", dim)
		}
	}
}

// TestHardcodedCapabilityPackNamesRemoved (CLM-003, kind: absence): the hardcoded
// backstop/contracts and backstop/substantiveness capability-key literals — and the
// contractsPackName/substantivenessPackName accessors that returned them — are gone from
// the capability path. A pack NAME may survive only as a human display label, never as
// the capability key, so the org-coordinate literals must not appear in gate.go at all.
func TestHardcodedCapabilityPackNamesRemoved(t *testing.T) {
	src, err := os.ReadFile("gate.go")
	if err != nil {
		t.Fatalf("reading gate.go: %v", err)
	}
	text := string(src)
	for _, banned := range []string{
		"func contractsPackName(",
		"func substantivenessPackName(",
		`"backstop/contracts"`,
		`"backstop/substantiveness"`,
	} {
		if strings.Contains(text, banned) {
			t.Errorf("gate.go must not contain %q — capability keys on declared gate_type, not a baked pack coordinate (CLM-003)", banned)
		}
	}
}

// TestCoverageToolchainSuffixHeuristicRemoved (CLM-004, kind: absence): the `-toolchain`
// name-suffix heuristic is gone from COVERAGE capability detection. A pack whose name ends
// in `-toolchain` but declares NO coverage engine must NOT grant the coverage capability
// (the suffix is no longer the key), and a pack with any name declaring a coverage engine
// DOES grant it. Behavioral, so it survives an unrelated refactor of the detector body.
func TestCoverageToolchainSuffixHeuristicRemoved(t *testing.T) {
	// A `-toolchain`-named pack declaring only a lint engine: the old suffix heuristic
	// would have granted coverage; by-declaration must not.
	suffixNamedNoCoverage := []*pack.Manifest{packDeclaringGateType("acme/foo-toolchain", engine.GateTypeLint)}
	if coverageToolchainPackInstalled(suffixNamedNoCoverage) {
		t.Errorf("a -toolchain-named pack with no coverage engine must NOT grant coverage (suffix heuristic removed, CLM-004)")
	}
	// A pack with NO -toolchain suffix that declares a coverage engine DOES grant coverage.
	nonSuffixWithCoverage := []*pack.Manifest{packDeclaringGateType("acme/coverage-provider", engine.GateTypeCoverage)}
	if !coverageToolchainPackInstalled(nonSuffixWithCoverage) {
		t.Errorf("a pack declaring a coverage engine must grant coverage regardless of name (CLM-004)")
	}
}

// TestCapability_OrgAgnosticProvider (CLM-005): a pack under a NON-backstop org that
// declares a contracts gate_type engine provides the contracts capability, with zero
// dependence on its name or org. This is the multi-publisher case the org-coordinate
// key blocked — the exact bclabs-portal TypeScript-contracts scenario.
func TestCapability_OrgAgnosticProvider(t *testing.T) {
	packs := []*pack.Manifest{packDeclaringGateType("acme/ts-contracts", engine.GateTypeContracts)}
	if !contractsPackInstalled(packs) {
		t.Errorf("a non-backstop-org pack declaring a contracts engine must provide the contracts capability (CLM-005)")
	}
	cap := capabilityStateForDimension(packs, gate.DimensionContracts)
	if !cap.Present || !cap.Working {
		t.Errorf("org-agnostic contracts provider must yield Present+Working, got %+v (CLM-005)", cap)
	}
}
