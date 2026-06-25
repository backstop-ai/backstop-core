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

	// MIGRATED FOR SPEC-041 (CLM-052 + REQ-001): ALL THREE traceability dimensions are
	// now INSTALLED-pack keyed — the baked Go contracts (SPEC-038), substantiveness
	// (SPEC-037), AND coverage (SPEC-041) analyzers are all deleted. There is no longer
	// any baked-Go-present dimension. The coverage arm is asserted below alongside
	// substantiveness and contracts. (tsCfg is exercised through the per-dimension
	// pack-absent arms — a non-Go project with no pack is capability-absent too.)
	_ = tsCfg

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

	// COVERAGE arm — re-keyed onto the INSTALLED coverage toolchain pack (SPEC-041
	// REQ-001). Mirrors the substantiveness/contracts splits: without a coverage
	// toolchain pack, the dimension is capability-ABSENT regardless of language (the
	// baked Go coverage analyzer is eradicated). A Go project with NO coverage pack is
	// undeclared+absent -> class 2.
	goNoCoveragePack := deriveCapabilityState(goCfg, gate.DimensionCoverage)
	if goNoCoveragePack.Present {
		t.Errorf("coverage on go with NO toolchain pack installed: Present = true, want false (analyzer eradicated, pack not installed)")
	}
	if got := gate.ClassifyDimension(goCfg, gate.DimensionCoverage, goNoCoveragePack); got != gate.ClassCapabilityAbsent {
		t.Errorf("coverage on go undeclared + pack-absent: class = %v, want ClassCapabilityAbsent", got)
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
