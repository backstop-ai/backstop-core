package main

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// gate_capability_rekey_test.go pins the SUBSTANTIVENESS capability signal at the live
// locus deriveCapabilityState. MIGRATED FOR ISSUE-063: the capability is no longer keyed
// on the installed pack's NAME (backstop/substantiveness) — it keys on whether some
// installed pack DECLARES a `gate_type: substantiveness` engine (packDeclaresGateType).
// The provider's name/org is irrelevant; only the declaration matters.

// substPacks returns an installed-pack set declaring a substantiveness engine. The
// pack name is a backstop coordinate here only for readability — detection keys on the
// declared gate_type, proven org-agnostic by TestCapability_OrgAgnosticProvider.
func substPacks() []*pack.Manifest {
	return []*pack.Manifest{packDeclaringGateType("backstop/substantiveness", engine.GateTypeSubstantiveness)}
}

// goToolchainCoveragePacks returns an installed-pack set declaring a coverage engine
// (the go-toolchain's role) — the coverage capability provider.
func goToolchainCoveragePacks() []*pack.Manifest {
	return []*pack.Manifest{packDeclaringGateType("backstop/go-toolchain", engine.GateTypeCoverage)}
}

// TestCapability_SubstantivenessKeyedOnInstalledPack_NotBakedAnalyzer — the
// substantiveness dimension's CapabilityState source is "some installed pack declares a
// substantiveness engine": declared -> Present/Working (gate RUNS it); absent+undeclared
// -> class-2 (capability-absent, warn, exit 0); absent+declared -> class-3 (declared-
// intent-unmet, block).
func TestCapability_SubstantivenessKeyedOnInstalledPack_NotBakedAnalyzer(t *testing.T) {
	const dim = gate.DimensionSubstantiveness
	undeclaredCfg := &config.Config{Project: "rt"}

	// A pack declaring the substantiveness engine -> Present/Working, and PackOrCommand
	// names the declared capability, not a baked Go analyzer.
	installed := deriveCapabilityState(substPacks(), dim, "")
	if !installed.Present || !installed.Working {
		t.Errorf("pack declaring substantiveness engine: want Present+Working, got %+v", installed)
	}
	if installed.PackOrCommand == "the baked Go substantiveness analyzer" {
		t.Errorf("substantiveness must NOT key on the deleted baked analyzer; got %q", installed.PackOrCommand)
	}
	// Present + undeclared -> none/proceed (the gate runs the pack step).
	if got := gate.ClassifyDimension(undeclaredCfg, dim, installed); got != gate.ClassNone {
		t.Errorf("present + undeclared: class = %v, want ClassNone (proceed)", got)
	}

	// No provider + undeclared -> capability-absent (class 2).
	absent := deriveCapabilityState(nil, dim, "")
	if absent.Present {
		t.Errorf("no provider: want Present=false, got %+v", absent)
	}
	if got := gate.ClassifyDimension(undeclaredCfg, dim, absent); got != gate.ClassCapabilityAbsent {
		t.Errorf("absent + undeclared: class = %v, want ClassCapabilityAbsent", got)
	}

	// No provider + DECLARED -> declared-intent-unmet (class 3, block).
	declaredCfg := &config.Config{Project: "rt"}
	declaredCfg.Enforcement.Toolchain = map[string]config.ToolchainPass{
		"substantiveness": {GateType: string(dim)},
	}
	if got := gate.ClassifyDimension(declaredCfg, dim, deriveCapabilityState(nil, dim, "")); got != gate.ClassDeclaredIntentUnmet {
		t.Errorf("absent + declared: class = %v, want ClassDeclaredIntentUnmet (block)", got)
	}
}

// TestCapability_RekeyIsSubstantivenessOnly_CoverageContractsUnchanged — each dimension's
// capability is driven by its OWN declared gate_type: a coverage provider grants coverage
// only, a substantiveness provider grants substantiveness only. No dimension leaks into
// another, and none is baked-Go-present.
func TestCapability_RekeyIsSubstantivenessOnly_CoverageContractsUnchanged(t *testing.T) {
	// COVERAGE — absent without a coverage-declaring pack, present with one.
	covCap := deriveCapabilityState(nil, gate.DimensionCoverage, "")
	if covCap.Present || covCap.Working {
		t.Errorf("coverage with no coverage-declaring pack must be capability-absent; got %+v", covCap)
	}
	if covCap.PackOrCommand == "the baked Go coverage analyzer" {
		t.Errorf("coverage must NOT key on the deleted baked analyzer; got %q", covCap.PackOrCommand)
	}
	covInstalled := deriveCapabilityState(goToolchainCoveragePacks(), gate.DimensionCoverage, "")
	if !covInstalled.Present || !covInstalled.Working {
		t.Errorf("coverage with a coverage-declaring pack installed must be Present+Working; got %+v", covInstalled)
	}

	// SUBSTANTIVENESS — absent without a substantiveness-declaring pack, present with.
	subCap := deriveCapabilityState(nil, gate.DimensionSubstantiveness, "")
	if subCap.Present {
		t.Errorf("substantiveness with no provider must be capability-absent; got %+v", subCap)
	}
	subInstalled := deriveCapabilityState(substPacks(), gate.DimensionSubstantiveness, "")
	if !subInstalled.Present {
		t.Errorf("substantiveness with a provider installed must be Present; got %+v", subInstalled)
	}

	// Coverage keying is invariant to the SUBSTANTIVENESS provider — a substantiveness
	// pack declares no coverage engine, so it grants no coverage.
	covWithSubstPack := deriveCapabilityState(substPacks(), gate.DimensionCoverage, "")
	if covWithSubstPack.Present {
		t.Errorf("a substantiveness-only pack must not grant coverage; got %+v", covWithSubstPack)
	}
}

// TestCapability_ShippedSpec036Test_MigratedForSubstantivenessRekey — guards that ALL
// THREE dimensions are declaration-keyed: absent without a declaring pack, present with
// the matching gate_type declaration. No dimension is baked-Go-present.
func TestCapability_ShippedSpec036Test_MigratedForSubstantivenessRekey(t *testing.T) {
	// Coverage arm: declaration keying.
	if deriveCapabilityState(nil, gate.DimensionCoverage, "").Present {
		t.Errorf("migrated test: coverage with NO coverage-declaring pack must be Absent")
	}
	if !deriveCapabilityState(goToolchainCoveragePacks(), gate.DimensionCoverage, "").Present {
		t.Errorf("migrated test: coverage with a coverage-declaring pack installed must be Present")
	}

	// Substantiveness arm: declaration keying.
	if deriveCapabilityState(nil, gate.DimensionSubstantiveness, "").Present {
		t.Errorf("migrated test: substantiveness with NO provider must be Absent")
	}
	if !deriveCapabilityState(substPacks(), gate.DimensionSubstantiveness, "").Present {
		t.Errorf("migrated test: substantiveness with a provider installed must be Present")
	}
}
