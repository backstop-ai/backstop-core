package gate

import "testing"

// substantiveness_join_test.go drives the language-agnostic gate-side half of the
// SPEC-037 substantiveness pack split: the relocated TargetPackageName, the
// NoTargetViolation set-join decision table, routing-by-namespaced-rule-ID, and
// per-test keying of extraction findings parsed from the PINNED SARIF message.
// All assertions are over flat pack SARIF (synthetic []Violation here) — no
// go/parser, no ast-grep invocation in this file.

const (
	testHollowRuleID     = "backstop/substantiveness/hollow-test-go"
	testExtractionRuleID = "backstop/substantiveness/referenced-symbol-go"
)

// TestTargetPackageName_MigratedBehaviorPreserved (SPEC-037 CLM-028, repurposed by
// ISSUE-047) — TargetPackageName holds an OPAQUE subject token and reduces a PATH
// subject to its LAST `/`-segment via the same language-neutral last-segment op used
// by testFileColocatedWithTarget. NO "cmd/"→"" special-case and NO "pkg/"-required
// guard remain: a bare token passes through unchanged, a `cmd/...` subject now yields
// a REAL leaf token (not ""), and only the EMPTY subject yields "" (the empty-target
// input the set-join SKIPS). The mandated test NAME is preserved for SPEC-037 lineage;
// its body asserts the de-baked opaque-token behavior.
func TestTargetPackageName_MigratedBehaviorPreserved(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"pkg/gate", "gate"},
		{"pkg/pack/distribution", "distribution"},
		// De-baked: cmd/... is no longer special-cased to "" — it reduces to its leaf
		// like any other path subject. This is the exact behavior change ISSUE-047 makes.
		{"cmd/backstop", "backstop"},
		{"cmd/foo/bar", "bar"},
		{"internal/foo", "foo"},
		// A bare token (no separator) passes through unchanged.
		{"gate", "gate"},
		// Only the empty subject yields the empty-target skip sentinel.
		{"", ""},
	}
	for _, c := range cases {
		if got := TargetPackageName(c.in); got != c.want {
			t.Errorf("TargetPackageName(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	// Explicit anti-regression: no "cmd/"→"" special-case survives. A cmd/ subject
	// must yield a real leaf token, never the empty-target sentinel.
	if got := TargetPackageName("cmd/backstop"); got == "" {
		t.Errorf(`cmd/ subject must NOT reduce to "" (no baked layout special-case); got empty`)
	}
	// Explicit anti-regression: no "pkg/"-required guard survives. A non-pkg/ path
	// subject must still reduce to its leaf, not be zeroed.
	if got := TargetPackageName("internal/foo"); got != "foo" {
		t.Errorf(`non-pkg/ path subject must reduce to its leaf (no "pkg/"-required guard); got %q`, got)
	}
}

// TestQ2_SetJoin_ReferencesTarget_NoViolation (CLM-007) — target in the extracted
// set → satisfied, no violation.
func TestQ2_SetJoin_ReferencesTarget_NoViolation(t *testing.T) {
	referenced := ReferencedSymbolSet{"gate": true, "other": true}
	v, raised := NoTargetViolation("TestX", "gate", referenced, false)
	if raised {
		t.Fatalf("expected no violation when target is referenced, got %+v", v)
	}
}

// TestQ2_SetJoin_DoesNotReferenceTarget_RaisesNoTarget (CLM-008) — target not in
// set → noTarget violation naming the test and package.
func TestQ2_SetJoin_DoesNotReferenceTarget_RaisesNoTarget(t *testing.T) {
	referenced := ReferencedSymbolSet{"other": true}
	v, raised := NoTargetViolation("TestX", "gate", referenced, false)
	if !raised {
		t.Fatalf("expected a noTarget violation when target is not referenced")
	}
	if v.Rule != StepTestSubstantiveness {
		t.Errorf("violation Rule = %q, want %q", v.Rule, StepTestSubstantiveness)
	}
	if want := "test function TestX does not call package gate"; v.Message != want {
		t.Errorf("violation Message = %q, want %q", v.Message, want)
	}
}

// TestQ2_SetJoin_EmptyTargetPackage_Skipped (CLM-009) — empty targetPkg → skipped
// regardless of the extracted set.
func TestQ2_SetJoin_EmptyTargetPackage_Skipped(t *testing.T) {
	// Even an empty set must NOT raise when the target package is empty.
	if _, raised := NoTargetViolation("TestX", "", ReferencedSymbolSet{}, false); raised {
		t.Fatalf("empty target package must skip the noTarget check (no violation)")
	}
	// And a populated set that lacks any target still must not raise.
	if _, raised := NoTargetViolation("TestX", "", ReferencedSymbolSet{"other": true}, false); raised {
		t.Fatalf("empty target package must skip even with a populated set")
	}
}

// TestQ2_SetJoin_SamePackage_Satisfied (CLM-010) — samePackage true → satisfied
// without requiring a package-qualified reference.
func TestQ2_SetJoin_SamePackage_Satisfied(t *testing.T) {
	// Same-package short-circuit: no reference to the target package, but the test
	// resides in it, so the join is satisfied.
	if _, raised := NoTargetViolation("TestX", "gate", ReferencedSymbolSet{}, true); raised {
		t.Fatalf("same-package test must satisfy the join without a qualified reference")
	}
}

// TestQ2_NoTargetIsGateSetTest_NotBakedAnalyzer (CLM-011) — the verdict is a pure
// set/string membership test: the SAME inputs always yield the SAME verdict, and the
// only thing that flips the disposition is set membership / same-package / empty
// target — never any parse of a file.
func TestQ2_NoTargetIsGateSetTest_NotBakedAnalyzer(t *testing.T) {
	// Membership is the sole discriminator: add the target to the set and the
	// previously-raised violation disappears, with no file ever read.
	if _, raised := NoTargetViolation("TestX", "gate", ReferencedSymbolSet{"other": true}, false); !raised {
		t.Fatalf("precondition: target absent should raise")
	}
	if _, raised := NoTargetViolation("TestX", "gate", ReferencedSymbolSet{"other": true, "gate": true}, false); raised {
		t.Fatalf("adding the target to the set must satisfy the join — pure set membership")
	}
	// Determinism: repeated calls with identical inputs are identical.
	_, a := NoTargetViolation("TestX", "gate", ReferencedSymbolSet{"other": true}, false)
	_, b := NoTargetViolation("TestX", "gate", ReferencedSymbolSet{"other": true}, false)
	if a != b {
		t.Fatalf("set-join must be deterministic over identical inputs")
	}
}

// TestRoute_PartitionsSubstantivenessByRuleID_FromFlatStream (CLM-024) — a FLAT
// stream of substantiveness hollow + extraction findings INTERLEAVED with unrelated
// pack-rule findings is partitioned to ONLY the hollow + extraction findings, matched
// by namespaced rule ID; no gate_type field is consulted (none exists).
func TestRoute_PartitionsSubstantivenessByRuleID_FromFlatStream(t *testing.T) {
	flat := []Violation{
		{Rule: "some-other-pack/lint-rule", File: "a.go", Message: "unrelated"},
		{Rule: testHollowRuleID, File: "h_test.go", Message: "test function TestH has no assertions (hollow) func=TestH"},
		{Rule: "another/rule", File: "b.go", Message: "noise"},
		{Rule: testExtractionRuleID, File: "h_test.go", Message: "referenced-symbol func=TestH symbol=gate"},
		{Rule: testHollowRuleID, File: "h2_test.go", Message: "test function TestH2 has no assertions (hollow) func=TestH2"},
	}
	hollow, extraction := RouteSubstantivenessFindings(flat, testHollowRuleID, testExtractionRuleID)
	if len(hollow) != 2 {
		t.Errorf("hollow partition len = %d, want 2; got %+v", len(hollow), hollow)
	}
	if len(extraction) != 1 {
		t.Errorf("extraction partition len = %d, want 1; got %+v", len(extraction), extraction)
	}
	for _, v := range hollow {
		if v.Rule != testHollowRuleID {
			t.Errorf("hollow partition contains non-hollow rule %q", v.Rule)
		}
	}
	for _, v := range extraction {
		if v.Rule != testExtractionRuleID {
			t.Errorf("extraction partition contains non-extraction rule %q", v.Rule)
		}
	}
}

// TestRoute_KeysExtractionFindingsToMandatedTest (CLM-025) — PINS the extraction
// message format (func=<FN> symbol=<pkg>); FuncName is parsed from Message (NOT
// Line/region, which is dropped), and findings join to a MandatedTest by
// (FilePath, FuncName) so a finding for test X contributes only to X's set.
func TestRoute_KeysExtractionFindingsToMandatedTest(t *testing.T) {
	extraction := []Violation{
		{Rule: testExtractionRuleID, File: "x_test.go", Message: "referenced-symbol func=TestX symbol=gate"},
		{Rule: testExtractionRuleID, File: "x_test.go", Message: "referenced-symbol func=TestX symbol=other"},
		// A finding for a DIFFERENT test in the same file must not leak into TestX.
		{Rule: testExtractionRuleID, File: "x_test.go", Message: "referenced-symbol func=TestY symbol=leak"},
	}
	testX := MandatedTest{FuncName: "TestX", FilePath: "x_test.go"}
	set := ReferencedSetForTest(extraction, testX)
	if !set["gate"] {
		t.Errorf("expected 'gate' in TestX's referenced set; got %+v", set)
	}
	if !set["other"] {
		t.Errorf("expected 'other' in TestX's referenced set; got %+v", set)
	}
	if set["leak"] {
		t.Errorf("TestY's symbol leaked into TestX's set: %+v", set)
	}
}

// TestRoute_NoExtractionFinding_EmptySetFlowsToSetJoin (CLM-026) — a test with NO
// extraction finding yields an EMPTY set that flows to the decision table unchanged
// (here: empty set + non-empty target + not same-package → noTarget).
func TestRoute_NoExtractionFinding_EmptySetFlowsToSetJoin(t *testing.T) {
	extraction := []Violation{
		{Rule: testExtractionRuleID, File: "other_test.go", Message: "referenced-symbol func=TestOther symbol=gate"},
	}
	testX := MandatedTest{FuncName: "TestX", FilePath: "x_test.go"}
	set := ReferencedSetForTest(extraction, testX)
	if len(set) != 0 {
		t.Fatalf("expected empty set for a test with no matching extraction finding; got %+v", set)
	}
	// The empty set flows to the decision table unchanged: non-empty target, not
	// same-package → noTarget.
	if _, raised := NoTargetViolation(testX.FuncName, "gate", set, false); !raised {
		t.Fatalf("empty set + non-empty target + not same-package must raise noTarget unchanged")
	}
}
