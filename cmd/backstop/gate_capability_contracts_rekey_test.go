package main

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/bmanson/backstop-core/pkg/gate"
)

// gate_capability_contracts_rekey_test.go (SPEC-038 TASK-035, REQ-015): the
// CONTRACTS-only capability re-key at the live locus deriveCapabilityState. After
// this spec the 3-arm end state is ASYMMETRIC: SUBSTANTIVENESS keys on its installed
// pack (Seed 3, untouched), CONTRACTS keys on the installed contracts pack (this
// spec), COVERAGE stays baked-Go (descoped). (Seed 3's substantiveness re-key tests
// live in gate_capability_rekey_test.go; this file owns the contracts mandated names.)

// goCfgWithContractsPack builds a Go config declaring the contracts pack installed
// (value "local" models the dogfood-installed local pack).
func goCfgWithContractsPack() *config.Config {
	return &config.Config{Project: "rt", Packs: config.Packs{"backstop/contracts": "local"}}
}

// TestCapability_ContractsKeyedOnInstalledPack_NotBakedAnalyzer (CLM-050): for the
// CONTRACTS dimension, deriveCapabilityState's source is "the contracts pack is
// INSTALLED / resolvable" (read from cfg.Packs), NOT the deleted go/parser analyzer
// and NOT a built-in tier. Installed -> Present/Working; absent -> capability-absent.
func TestCapability_ContractsKeyedOnInstalledPack_NotBakedAnalyzer(t *testing.T) {
	cap := deriveCapabilityState(goCfgWithContractsPack(), gate.DimensionContracts, "")
	if !cap.Present || !cap.Working {
		t.Fatalf("contracts with the pack installed must be Present/Working, got %+v (CLM-050)", cap)
	}
	if cap.PackOrCommand == "the baked Go contracts analyzer" {
		t.Errorf("contracts must NOT key on the deleted baked analyzer; got %q (CLM-050)", cap.PackOrCommand)
	}

	absent := &config.Config{Project: "rt"}
	capAbsent := deriveCapabilityState(absent, gate.DimensionContracts, "")
	if capAbsent.Present {
		t.Errorf("contracts with NO pack installed must be capability-absent, got Present=true (CLM-050)")
	}
	if got := gate.ClassifyDimension(absent, gate.DimensionContracts, capAbsent); got != gate.ClassCapabilityAbsent {
		t.Errorf("contracts undeclared + pack-absent: class = %v, want ClassCapabilityAbsent (class-2) (CLM-050)", got)
	}
}

// TestCapability_RekeyIsContractsOnly_CoverageStaysBaked_SubstantivenessUntouched
// (CLM-051): UPDATED FOR SPEC-041 — coverage NO LONGER stays baked. SPEC-041
// eradicates the baked Go coverage analyzer and re-keys coverage onto the installed
// coverage toolchain pack, so for a Go project with NEITHER pack installed, ALL THREE
// dimensions (coverage, contracts, substantiveness) return absent. (The "stays baked"
// in the name is the predating SPEC-038-era invariant this test now overturns.)
func TestCapability_RekeyIsContractsOnly_CoverageStaysBaked_SubstantivenessUntouched(t *testing.T) {
	goNoPacks := &config.Config{Project: "rt"}

	if deriveCapabilityState(goNoPacks, gate.DimensionCoverage, "").Present {
		t.Error("COVERAGE must be absent on a Go project with no toolchain pack (re-keyed, analyzer eradicated) (CLM-051)")
	}
	if deriveCapabilityState(goNoPacks, gate.DimensionContracts, "").Present {
		t.Error("CONTRACTS must be absent on a Go project with no contracts pack (re-keyed) (CLM-051)")
	}
	if deriveCapabilityState(goNoPacks, gate.DimensionSubstantiveness, "").Present {
		t.Error("SUBSTANTIVENESS must be absent on a Go project with no substantiveness pack (Seed 3, untouched) (CLM-051)")
	}
}

// TestCapability_ShippedCapabilityTest_MigratedForContractsRekey (CLM-052):
// UPDATED FOR SPEC-041 — DimensionCoverage is ALSO re-keyed off the baked analyzer
// now (the analyzer is eradicated), so a Go project without a coverage toolchain
// pack is capability-absent for BOTH contracts and coverage. We assert the
// migration's load-bearing property: neither contracts NOR coverage is baked-Go-keyed
// any longer.
func TestCapability_ShippedCapabilityTest_MigratedForContractsRekey(t *testing.T) {
	goNoPacks := &config.Config{Project: "rt"}

	if deriveCapabilityState(goNoPacks, gate.DimensionContracts, "").Present {
		t.Error("post-migration, DimensionContracts must NOT be baked-Go-present on a Go project (CLM-052)")
	}
	if deriveCapabilityState(goNoPacks, gate.DimensionCoverage, "").Present {
		t.Error("post-SPEC-041, DimensionCoverage must NOT be baked-Go-present (analyzer eradicated, now pack-keyed) (CLM-052)")
	}
	if !deriveCapabilityState(goCfgWithContractsPack(), gate.DimensionContracts, "").Present {
		t.Error("with the contracts pack installed, DimensionContracts must be Present (CLM-052)")
	}
}
