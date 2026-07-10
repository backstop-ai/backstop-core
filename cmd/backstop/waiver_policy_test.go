package main

import (
	"path/filepath"
	"testing"
)

// waiverE2EFixtureRoot resolves the committed installed-pack waiver e2e fixture.
func waiverE2EFixtureRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "pkg", "gate", "testdata", "waiver-e2e")
}

// TestWaiverPolicy_ExtractedFromInstalledPackManifests proves buildWaiverPolicy
// builds the production waiver.Policy by EXTRACTING the declared non-waivable set
// from the INSTALLED pack manifests (not a hardcoded list): the fixture pack
// declares protected-defect non_waivable, so the resulting Policy reports exactly
// that rule non-waivable while an undeclared rule stays waivable (CLM-069).
func TestWaiverPolicy_ExtractedFromInstalledPackManifests(t *testing.T) {
	policy, err := buildWaiverPolicy(waiverE2EFixtureRoot(t))
	if err != nil {
		t.Fatalf("buildWaiverPolicy: %v", err)
	}

	// The pack-DECLARED non_waivable rule is non-waivable (extracted from manifest).
	if policy.Waivable("backstop/waiver-e2e/protected-defect", "error") {
		t.Error("the pack-declared non_waivable rule must be reported non-waivable (extracted from the manifest)")
	}
	// The normal (undeclared) rule remains waivable — no hardcoded protected list.
	if !policy.Waivable("backstop/waiver-e2e/waivable-defect", "error") {
		t.Error("a rule NOT declared non_waivable must remain waivable")
	}
	// An entirely unknown rule from a manifest declaring nothing is waivable.
	if !policy.Waivable("some/other/rule", "warning") {
		t.Error("an undeclared rule must remain waivable (no core-hardcoded non-waivable list)")
	}
}
