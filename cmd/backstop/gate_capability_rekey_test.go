package main

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/bmanson/backstop-core/pkg/gate"
)

// gate_capability_rekey_test.go pins the SUBSTANTIVENESS-ONLY capability re-key at the
// live locus deriveCapabilityState (SPEC-037 REQ-009 / CLM-035 / CLM-036). The
// substantiveness arm keys on "the substantiveness pack is INSTALLED" (not the deleted
// baked Go analyzer); the coverage and contracts arms are UNCHANGED.

// goCfgWithSubstPack returns a Go config that has the substantiveness pack installed
// (declared in the packs map, value "local" — the dogfood local install).
func goCfgWithSubstPack() *config.Config {
	return &config.Config{
		Project:  "rt",
		Language: "go",
		Packs:    config.Packs{"backstop/substantiveness": "local"},
	}
}

// TestCapability_SubstantivenessKeyedOnInstalledPack_NotBakedAnalyzer (CLM-035) — the
// substantiveness dimension's CapabilityState source is "the substantiveness pack is
// installed/resolvable": installed -> Present/Working (gate RUNS it); absent+undeclared
// -> class-2 (capability-absent, warn, exit 0); absent+declared -> class-3 (declared-
// intent-unmet, block).
func TestCapability_SubstantivenessKeyedOnInstalledPack_NotBakedAnalyzer(t *testing.T) {
	const dim = gate.DimensionSubstantiveness

	// Pack installed -> Present/Working, and the PackOrCommand names the PACK, not a
	// baked Go analyzer.
	installed := deriveCapabilityState(goCfgWithSubstPack(), dim)
	if !installed.Present || !installed.Working {
		t.Errorf("pack installed: want Present+Working, got %+v", installed)
	}
	if installed.PackOrCommand == "the baked Go substantiveness analyzer" {
		t.Errorf("substantiveness must NOT key on the deleted baked analyzer; got %q", installed.PackOrCommand)
	}
	// Installed + undeclared -> none/proceed (the gate runs the pack step).
	if got := gate.ClassifyDimension(goCfgWithSubstPack(), dim, installed); got != gate.ClassNone {
		t.Errorf("installed + undeclared: class = %v, want ClassNone (proceed)", got)
	}

	// Pack absent + undeclared -> capability-absent (class 2).
	absentCfg := &config.Config{Project: "rt", Language: "go"}
	absent := deriveCapabilityState(absentCfg, dim)
	if absent.Present {
		t.Errorf("pack absent: want Present=false, got %+v", absent)
	}
	if got := gate.ClassifyDimension(absentCfg, dim, absent); got != gate.ClassCapabilityAbsent {
		t.Errorf("absent + undeclared: class = %v, want ClassCapabilityAbsent", got)
	}

	// Pack absent + DECLARED -> declared-intent-unmet (class 3, block).
	declaredCfg := &config.Config{Project: "rt", Language: "go"}
	declaredCfg.Enforcement.Toolchain = map[string]config.ToolchainPass{
		"substantiveness": {GateType: string(dim)},
	}
	if got := gate.ClassifyDimension(declaredCfg, dim, deriveCapabilityState(declaredCfg, dim)); got != gate.ClassDeclaredIntentUnmet {
		t.Errorf("absent + declared: class = %v, want ClassDeclaredIntentUnmet (block)", got)
	}
}

// TestCapability_RekeyIsSubstantivenessOnly_CoverageContractsUnchanged (CLM-036) — the
// re-key is DIMENSION-ASYMMETRIC. UPDATED FOR SPEC-038 (align-predating-artifacts):
// Seed 3 left CONTRACTS on the baked-Go keying, but SPEC-038 deletes the contracts
// analyzer and re-keys CONTRACTS onto the installed pack too. So now only COVERAGE
// stays baked-Go here; CONTRACTS' installed-pack keying is asserted in
// gate_capability_contracts_rekey_test.go. This test keeps verifying the SUBSTANTIVENESS
// re-key vs the STILL-baked COVERAGE arm.
func TestCapability_RekeyIsSubstantivenessOnly_CoverageContractsUnchanged(t *testing.T) {
	// No substantiveness pack installed.
	goCfg := &config.Config{Project: "rt", Language: "go"}

	// COVERAGE arm UNCHANGED — baked Go analyzer present (CONTRACTS split out per SPEC-038).
	for _, dim := range []gate.TraceabilityDimension{gate.DimensionCoverage} {
		cap := deriveCapabilityState(goCfg, dim)
		if !cap.Present || !cap.Working {
			t.Errorf("dim %s on go must stay on the baked-Go keying (Present+Working); got %+v", dim, cap)
		}
		if cap.PackOrCommand != "the baked Go "+string(dim)+" analyzer" {
			t.Errorf("dim %s must keep the baked-Go PackOrCommand; got %q", dim, cap.PackOrCommand)
		}
	}

	// SUBSTANTIVENESS arm — keyed on the installed pack: absent without it.
	subCap := deriveCapabilityState(goCfg, gate.DimensionSubstantiveness)
	if subCap.Present {
		t.Errorf("substantiveness on go with no pack must be capability-absent; got %+v", subCap)
	}
	// With the pack installed, substantiveness flips to Present while coverage/contracts
	// are identical either way (they don't read the packs map).
	subInstalled := deriveCapabilityState(goCfgWithSubstPack(), gate.DimensionSubstantiveness)
	if !subInstalled.Present {
		t.Errorf("substantiveness with the pack installed must be Present; got %+v", subInstalled)
	}
	covWithPack := deriveCapabilityState(goCfgWithSubstPack(), gate.DimensionCoverage)
	covNoPack := deriveCapabilityState(goCfg, gate.DimensionCoverage)
	if covWithPack != covNoPack {
		t.Errorf("coverage keying must be invariant to the substantiveness pack; with=%+v without=%+v", covWithPack, covNoPack)
	}
}

// TestCapability_ShippedSpec036Test_MigratedForSubstantivenessRekey (CLM-037) — the
// migrated form of the shipped SPEC-036 test: the SUBSTANTIVENESS arm asserts the
// INSTALLED-pack keying while the COVERAGE and CONTRACTS arms are left UNCHANGED (still
// the baked-Go assertion). This guards that the shipped test was migrated, not silently
// broken, and ./cmd/backstop/ stays green.
func TestCapability_ShippedSpec036Test_MigratedForSubstantivenessRekey(t *testing.T) {
	goCfg := &config.Config{Project: "rt", Language: "go"}
	tsCfg := &config.Config{Project: "rt", Language: "typescript"}

	// Coverage arm: UNCHANGED baked-Go keying (go Present, ts Absent). CONTRACTS split
	// out per SPEC-038 (re-keyed onto the installed pack — asserted separately).
	for _, dim := range []gate.TraceabilityDimension{gate.DimensionCoverage} {
		if !deriveCapabilityState(goCfg, dim).Present {
			t.Errorf("migrated test: %s on go must remain Present (baked-Go, unchanged)", dim)
		}
		if deriveCapabilityState(tsCfg, dim).Present {
			t.Errorf("migrated test: %s on typescript must remain Absent (baked-Go, unchanged)", dim)
		}
	}

	// Substantiveness arm: INSTALLED-pack keying (absent without the pack, present with).
	if deriveCapabilityState(goCfg, gate.DimensionSubstantiveness).Present {
		t.Errorf("migrated test: substantiveness on go with NO pack must be Absent (re-keyed)")
	}
	if !deriveCapabilityState(goCfgWithSubstPack(), gate.DimensionSubstantiveness).Present {
		t.Errorf("migrated test: substantiveness with the pack installed must be Present (re-keyed)")
	}
}
