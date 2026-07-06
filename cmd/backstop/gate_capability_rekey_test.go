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
		Project: "rt",
		Packs:   config.Packs{"backstop/substantiveness": "local"},
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
	installed := deriveCapabilityState(goCfgWithSubstPack(), dim, "")
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
	absentCfg := &config.Config{Project: "rt"}
	absent := deriveCapabilityState(absentCfg, dim, "")
	if absent.Present {
		t.Errorf("pack absent: want Present=false, got %+v", absent)
	}
	if got := gate.ClassifyDimension(absentCfg, dim, absent); got != gate.ClassCapabilityAbsent {
		t.Errorf("absent + undeclared: class = %v, want ClassCapabilityAbsent", got)
	}

	// Pack absent + DECLARED -> declared-intent-unmet (class 3, block).
	declaredCfg := &config.Config{Project: "rt"}
	declaredCfg.Enforcement.Toolchain = map[string]config.ToolchainPass{
		"substantiveness": {GateType: string(dim)},
	}
	if got := gate.ClassifyDimension(declaredCfg, dim, deriveCapabilityState(declaredCfg, dim, "")); got != gate.ClassDeclaredIntentUnmet {
		t.Errorf("absent + declared: class = %v, want ClassDeclaredIntentUnmet (block)", got)
	}
}

// TestCapability_RekeyIsSubstantivenessOnly_CoverageContractsUnchanged (CLM-036) —
// UPDATED FOR SPEC-041 (align-predating-artifacts): SPEC-041 ERADICATES the baked Go
// coverage analyzer and re-keys COVERAGE onto the installed coverage toolchain pack,
// so ALL THREE traceability dimensions are now pack-keyed (no asymmetry fence). This
// test now verifies that the COVERAGE arm is capability-ABSENT when no coverage
// toolchain pack is installed (the old "coverage stays baked-Go-present" invariant is
// overturned — that was coverage's deferred re-impl, now landed).
func TestCapability_RekeyIsSubstantivenessOnly_CoverageContractsUnchanged(t *testing.T) {
	// No packs installed.
	goCfg := &config.Config{Project: "rt"}

	// COVERAGE arm RE-KEYED — pack-resolvable, NOT the deleted baked analyzer: absent
	// without a coverage toolchain pack.
	covCap := deriveCapabilityState(goCfg, gate.DimensionCoverage, "")
	if covCap.Present || covCap.Working {
		t.Errorf("coverage on go with no toolchain pack must be capability-absent (baked analyzer eradicated); got %+v", covCap)
	}
	if covCap.PackOrCommand == "the baked Go coverage analyzer" {
		t.Errorf("coverage must NOT key on the deleted baked analyzer; got %q", covCap.PackOrCommand)
	}

	// With a go-toolchain pack installed, coverage flips to Present.
	covCfg := &config.Config{Project: "rt", Packs: config.Packs{"backstop/go-toolchain": "local"}}
	covInstalled := deriveCapabilityState(covCfg, gate.DimensionCoverage, "")
	if !covInstalled.Present || !covInstalled.Working {
		t.Errorf("coverage with a go-toolchain pack installed must be Present+Working; got %+v", covInstalled)
	}

	// SUBSTANTIVENESS arm — keyed on the installed pack: absent without it.
	subCap := deriveCapabilityState(goCfg, gate.DimensionSubstantiveness, "")
	if subCap.Present {
		t.Errorf("substantiveness on go with no pack must be capability-absent; got %+v", subCap)
	}
	subInstalled := deriveCapabilityState(goCfgWithSubstPack(), gate.DimensionSubstantiveness, "")
	if !subInstalled.Present {
		t.Errorf("substantiveness with the pack installed must be Present; got %+v", subInstalled)
	}
	// Coverage keying is invariant to the SUBSTANTIVENESS pack (it reads only the
	// toolchain/coverage declaration, not the substantiveness pack).
	covWithSubstPack := deriveCapabilityState(goCfgWithSubstPack(), gate.DimensionCoverage, "")
	if covWithSubstPack != covCap {
		t.Errorf("coverage keying must be invariant to the substantiveness pack; with=%+v without=%+v", covWithSubstPack, covCap)
	}
}

// TestCapability_ShippedSpec036Test_MigratedForSubstantivenessRekey (CLM-037) —
// UPDATED FOR SPEC-041: ALL THREE traceability dimensions are now INSTALLED-pack
// keyed (substantiveness, contracts, and now coverage — the baked Go coverage
// analyzer is eradicated). This guards that the shipped test was migrated, not
// silently broken, and ./cmd/backstop/ stays green.
func TestCapability_ShippedSpec036Test_MigratedForSubstantivenessRekey(t *testing.T) {
	goCfg := &config.Config{Project: "rt"}
	goCfgWithToolchain := &config.Config{Project: "rt", Packs: config.Packs{"backstop/go-toolchain": "local"}}

	// Coverage arm: INSTALLED-pack keying (absent without a coverage toolchain pack,
	// present with). No longer the deleted baked-Go analyzer.
	if deriveCapabilityState(goCfg, gate.DimensionCoverage, "").Present {
		t.Errorf("migrated test: coverage on go with NO toolchain pack must be Absent (re-keyed, analyzer eradicated)")
	}
	if !deriveCapabilityState(goCfgWithToolchain, gate.DimensionCoverage, "").Present {
		t.Errorf("migrated test: coverage with a go-toolchain pack installed must be Present (re-keyed)")
	}

	// Substantiveness arm: INSTALLED-pack keying (absent without the pack, present with).
	if deriveCapabilityState(goCfg, gate.DimensionSubstantiveness, "").Present {
		t.Errorf("migrated test: substantiveness on go with NO pack must be Absent (re-keyed)")
	}
	if !deriveCapabilityState(goCfgWithSubstPack(), gate.DimensionSubstantiveness, "").Present {
		t.Errorf("migrated test: substantiveness with the pack installed must be Present (re-keyed)")
	}
}
