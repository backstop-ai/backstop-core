package main

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// TestCapability_CurrentInstallUnchanged (CLM-009): behavior-preserving regression. With
// the present backstop-first install (backstop/contracts, backstop/substantiveness,
// backstop/go-toolchain all declared in this repo's backstop.yml), every traceability
// dimension resolves capability-PRESENT exactly as before the ISSUE-063 detection swap.
// This loads the REAL installed pack manifests (not synthetic ones), so it fails if the
// by-declaration detector diverges from the shipped install for any dimension.
func TestCapability_CurrentInstallUnchanged(t *testing.T) {
	root := repoRoot(t)
	packs, err := loadInstalledPacks(root)
	if err != nil {
		t.Fatalf("loadInstalledPacks(%s): %v", root, err)
	}
	if len(packs) == 0 {
		t.Fatal("test precondition: the backstop-first packs must resolve from this repo's backstop.yml")
	}

	for _, dim := range []gate.TraceabilityDimension{
		gate.DimensionContracts,
		gate.DimensionSubstantiveness,
		gate.DimensionCoverage,
	} {
		if !packDeclaresGateType(packs, dim) {
			t.Errorf("dimension %q must be capability-present on the current install (a pack declares its gate_type)", dim)
		}
		cap := deriveCapabilityState(packs, dim, "")
		if !cap.Present || !cap.Working {
			t.Errorf("dimension %q must derive Present+Working on the current install, got %+v", dim, cap)
		}
	}

	// The contracts and substantiveness dispatch packs must resolve to EXACTLY ONE pack
	// (deterministic, no ambiguity) on the current single-provider install.
	for _, dim := range []gate.TraceabilityDimension{gate.DimensionContracts, gate.DimensionSubstantiveness} {
		m, err := resolveCapabilityPack(packs, dim)
		if err != nil {
			t.Errorf("resolveCapabilityPack(%q) on the current install must not error, got %v", dim, err)
		}
		if m == nil {
			t.Errorf("resolveCapabilityPack(%q) must resolve a dispatch pack on the current install", dim)
		}
	}
}
