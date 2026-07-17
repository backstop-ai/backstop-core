package main

import (
	"reflect"
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
)

// TestByDeclaration_CurrentInstallUnchanged (CLM-008): behavior-preserving regression for
// ISSUE-064. Over the current install, (1) substantiveness findings route by the declared
// substantiveness_role EXACTLY as the deleted rule-id match did (same hollow/subject-join
// verdicts), and (2) the toolchain stack label renders the same string ("go") the old
// "-toolchain" suffix-strip produced. This is a mechanism swap, not a policy change: no
// finding flips hollow<->substantive and no label changes for the already-correct install.
func TestByDeclaration_CurrentInstallUnchanged(t *testing.T) {
	// (1) Substantiveness routing preservation. The installed substantiveness pack now
	// stamps BOTH the namespaced rule id (as before) AND the declared substantiveness_role
	// (new, ISSUE-064). A finding stream shaped exactly as the current install emits must
	// partition IDENTICALLY under the NEW by-role routing and the OLD by-rule-id routing.
	const (
		hollowRuleID     = "backstop/substantiveness/hollow-test-go"
		extractionRuleID = "backstop/substantiveness/referenced-symbol-go"
	)
	flat := []gate.Violation{
		{Rule: "backstop/go-standards/some-lint", File: "a.go"}, // unrelated pack finding
		{Rule: hollowRuleID, File: "h_test.go",
			Properties: map[string]string{"substantiveness_role": "hollow", "func": "TestH"}},
		{Rule: extractionRuleID, File: "h_test.go",
			Properties: map[string]string{"substantiveness_role": "referenced-symbol", "func": "TestH", "symbol": "gate"}},
		{Rule: hollowRuleID, File: "h2_test.go",
			Properties: map[string]string{"substantiveness_role": "hollow", "func": "TestH2"}},
	}

	// NEW: route by the declared role.
	gotHollow, gotExtraction := gate.RouteSubstantivenessFindings(flat)

	// OLD: the pre-swap routing — match the namespaced rule ids. This local re-derivation
	// stands in for the deleted rule-name routing to prove the swap preserves behavior for
	// the current install (where the pack stamps both channels).
	var wantHollow, wantExtraction []gate.Violation
	for _, v := range flat {
		switch v.Rule {
		case hollowRuleID:
			wantHollow = append(wantHollow, v)
		case extractionRuleID:
			wantExtraction = append(wantExtraction, v)
		}
	}

	if !reflect.DeepEqual(gotHollow, wantHollow) {
		t.Errorf("hollow partition diverged from the pre-swap rule-id routing:\n by-role: %+v\n by-id:   %+v", gotHollow, wantHollow)
	}
	if !reflect.DeepEqual(gotExtraction, wantExtraction) {
		t.Errorf("extraction partition diverged from the pre-swap rule-id routing:\n by-role: %+v\n by-id:   %+v", gotExtraction, wantExtraction)
	}

	// The subject-join verdicts are unchanged: the hollow tests are hollow, and the
	// extraction symbol joins to its test.
	if !gate.IsTestHollow(gotHollow, gate.MandatedTest{FuncName: "TestH", FilePath: "h_test.go"}) {
		t.Error("TestH must still be hollow via the by-role partition (verdict preserved)")
	}
	set := gate.ReferencedSetForTest(gotExtraction, gate.MandatedTest{FuncName: "TestH", FilePath: "h_test.go"})
	if !set["gate"] {
		t.Errorf("TestH's referenced symbol must still join (verdict preserved); got %+v", set)
	}

	// (2) Stack label preservation over the REAL installed packs: go-toolchain declares
	// language: go, so the declared-language label renders "go" — identical to the old
	// "-toolchain" suffix strip. This loads the shipped manifests, so it fails if the
	// rehome diverges from the current install.
	root := repoRoot(t)
	packs, err := loadInstalledPacks(root)
	if err != nil {
		t.Fatalf("loadInstalledPacks(%s): %v", root, err)
	}
	if got := declaredToolchainStackLabel(packs); got != "go" {
		t.Errorf("the stack label over the current install must render %q (unchanged by the rehome); got %q", "go", got)
	}
}
