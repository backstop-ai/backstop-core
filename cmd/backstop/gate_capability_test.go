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

	for _, dim := range []gate.TraceabilityDimension{
		gate.DimensionSubstantiveness, gate.DimensionCoverage, gate.DimensionContracts,
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
