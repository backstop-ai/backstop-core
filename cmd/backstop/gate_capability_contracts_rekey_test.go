package main

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// gate_capability_contracts_rekey_test.go pins the CONTRACTS capability signal at the live
// locus deriveCapabilityState. MIGRATED FOR ISSUE-063: the contracts capability is no
// longer keyed on the installed pack's NAME (backstop/contracts) — it keys on whether some
// installed pack DECLARES a `gate_type: contracts` engine (packDeclaresGateType), so a
// contracts pack under ANY name/org fills the slot.

// contractsPacks returns an installed-pack set declaring a contracts engine.
func contractsPacks() []*pack.Manifest {
	return []*pack.Manifest{packDeclaringGateType("backstop/contracts", engine.GateTypeContracts)}
}

// TestCapability_ContractsKeyedOnInstalledPack_NotBakedAnalyzer: for the CONTRACTS
// dimension, deriveCapabilityState's source is "some installed pack declares a contracts
// engine", NOT the deleted go/parser analyzer and NOT a built-in tier. Present ->
// Present/Working; no provider -> capability-absent.
func TestCapability_ContractsKeyedOnInstalledPack_NotBakedAnalyzer(t *testing.T) {
	cap := deriveCapabilityState(contractsPacks(), gate.DimensionContracts, "")
	if !cap.Present || !cap.Working {
		t.Fatalf("contracts with a contracts-declaring pack installed must be Present/Working, got %+v", cap)
	}
	if cap.PackOrCommand == "the baked Go contracts analyzer" {
		t.Errorf("contracts must NOT key on the deleted baked analyzer; got %q", cap.PackOrCommand)
	}

	undeclared := &config.Config{Project: "rt"}
	capAbsent := deriveCapabilityState(nil, gate.DimensionContracts, "")
	if capAbsent.Present {
		t.Errorf("contracts with NO provider must be capability-absent, got Present=true")
	}
	if got := gate.ClassifyDimension(undeclared, gate.DimensionContracts, capAbsent); got != gate.ClassCapabilityAbsent {
		t.Errorf("contracts undeclared + absent: class = %v, want ClassCapabilityAbsent (class-2)", got)
	}
}

// TestCapability_RekeyIsContractsOnly_CoverageStaysBaked_SubstantivenessUntouched: with
// NO packs installed, ALL THREE dimensions (coverage, contracts, substantiveness) return
// absent — each is declaration-keyed and none is baked-Go-present.
func TestCapability_RekeyIsContractsOnly_CoverageStaysBaked_SubstantivenessUntouched(t *testing.T) {
	if deriveCapabilityState(nil, gate.DimensionCoverage, "").Present {
		t.Error("COVERAGE must be absent with no coverage-declaring pack (declaration-keyed)")
	}
	if deriveCapabilityState(nil, gate.DimensionContracts, "").Present {
		t.Error("CONTRACTS must be absent with no contracts-declaring pack (declaration-keyed)")
	}
	if deriveCapabilityState(nil, gate.DimensionSubstantiveness, "").Present {
		t.Error("SUBSTANTIVENESS must be absent with no substantiveness-declaring pack (declaration-keyed)")
	}
}

// TestCapability_ShippedCapabilityTest_MigratedForContractsRekey: neither contracts nor
// coverage is baked-Go-keyed; both are absent with no declaring pack and present with the
// matching gate_type declaration.
func TestCapability_ShippedCapabilityTest_MigratedForContractsRekey(t *testing.T) {
	if deriveCapabilityState(nil, gate.DimensionContracts, "").Present {
		t.Error("post-migration, DimensionContracts must NOT be baked-Go-present")
	}
	if deriveCapabilityState(nil, gate.DimensionCoverage, "").Present {
		t.Error("post-migration, DimensionCoverage must NOT be baked-Go-present")
	}
	if !deriveCapabilityState(contractsPacks(), gate.DimensionContracts, "").Present {
		t.Error("with a contracts-declaring pack installed, DimensionContracts must be Present")
	}
}
