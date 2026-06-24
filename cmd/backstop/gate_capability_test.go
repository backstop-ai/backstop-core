package main

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/bmanson/backstop-core/pkg/gate"
)

// TestCapabilityState_NonGoProject_DerivesAbsentClass2 (CLM-029): on the
// existing binary, an UNDECLARED traceability dimension on a non-Go project
// (cfg.Language == "typescript") derives an ABSENT CapabilityState from
// cfg.Language + baked-analyzer presence and classifies as class 2
// (capability-absent → warn, exit 0) — NOT a silent pass and NOT a mis-applied
// Go analyzer. The same undeclared dimension on a Go project with the baked
// analyzer present is capability-present.
func TestCapabilityState_NonGoProject_DerivesAbsentClass2(t *testing.T) {
	tsCfg := &config.Config{Project: "rt", Language: "typescript"}
	goCfg := &config.Config{Project: "rt", Language: "go"}

	// MIGRATED FOR SPEC-038 (CLM-052): DimensionContracts is SPLIT OUT of this
	// baked-Go loop (the contracts analyzer is deleted; it now keys on the INSTALLED
	// contracts pack — asserted separately below), leaving ONLY DimensionCoverage on
	// the baked-Go assertion (coverage descoped, no pack — STAYS baked). The
	// SUBSTANTIVENESS dimension was already split out by Seed 3 (below).
	for _, dim := range []gate.TraceabilityDimension{
		gate.DimensionCoverage,
	} {
		// TypeScript: capability ABSENT, undeclared -> class 2.
		tsCap := deriveCapabilityState(tsCfg, dim)
		if tsCap.Present {
			t.Errorf("dim %s on typescript: CapabilityState.Present = true, want false (no Go analyzer applies)", dim)
		}
		if got := gate.ClassifyDimension(tsCfg, dim, tsCap); got != gate.ClassCapabilityAbsent {
			t.Errorf("dim %s on typescript undeclared: class = %v, want ClassCapabilityAbsent", dim, got)
		}

		// Go: baked analyzer present -> capability present (undeclared+present = none/proceed).
		goCap := deriveCapabilityState(goCfg, dim)
		if !goCap.Present {
			t.Errorf("dim %s on go: CapabilityState.Present = false, want true (baked Go analyzer exists)", dim)
		}
		if got := gate.ClassifyDimension(goCfg, dim, goCap); got != gate.ClassNone {
			t.Errorf("dim %s on go undeclared+present: class = %v, want ClassNone (proceed)", dim, got)
		}
	}

	// SUBSTANTIVENESS arm — re-keyed onto the INSTALLED pack (SPEC-037). Without the
	// pack installed, the dimension is capability-ABSENT regardless of language: a Go
	// project with NO substantiveness pack installed is undeclared+absent -> class 2
	// (NOT capability-present-via-a-baked-analyzer, which no longer exists).
	goNoPack := deriveCapabilityState(goCfg, gate.DimensionSubstantiveness)
	if goNoPack.Present {
		t.Errorf("substantiveness on go with NO pack installed: Present = true, want false (analyzer deleted, pack not installed)")
	}
	if got := gate.ClassifyDimension(goCfg, gate.DimensionSubstantiveness, goNoPack); got != gate.ClassCapabilityAbsent {
		t.Errorf("substantiveness on go undeclared + pack-absent: class = %v, want ClassCapabilityAbsent", got)
	}

	// CONTRACTS arm — re-keyed onto the INSTALLED contracts pack (SPEC-038/CLM-052).
	// Mirrors the substantiveness split: without the contracts pack installed, the
	// dimension is capability-ABSENT regardless of language (the baked go/parser
	// analyzer is deleted). A Go project with NO contracts pack installed is
	// undeclared+absent -> class 2.
	goNoContractsPack := deriveCapabilityState(goCfg, gate.DimensionContracts)
	if goNoContractsPack.Present {
		t.Errorf("contracts on go with NO pack installed: Present = true, want false (analyzer deleted, pack not installed)")
	}
	if got := gate.ClassifyDimension(goCfg, gate.DimensionContracts, goNoContractsPack); got != gate.ClassCapabilityAbsent {
		t.Errorf("contracts on go undeclared + pack-absent: class = %v, want ClassCapabilityAbsent", got)
	}
}

// TestCapabilityState_NonGoUndeclared_NeverAutoPromotes (CLM-012 wiring view): a
// capability-absent dimension never auto-promotes to blocking across repeated
// derivation+classification runs — it stays class 2.
func TestCapabilityState_NonGoUndeclared_NeverAutoPromotes(t *testing.T) {
	tsCfg := &config.Config{Project: "rt", Language: "typescript"}
	for i := 0; i < 5; i++ {
		cap := deriveCapabilityState(tsCfg, gate.DimensionCoverage)
		got := gate.ClassifyDimension(tsCfg, gate.DimensionCoverage, cap)
		if got != gate.ClassCapabilityAbsent {
			t.Fatalf("run %d: class = %v, want ClassCapabilityAbsent (no auto-promotion)", i, got)
		}
	}
}
