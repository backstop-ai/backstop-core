package gate

import (
	"path/filepath"
	"testing"
)

// absence_annotation_test.go covers the opt-in per-claim `kind: absence` capability
// (ISSUE-035 Category 2). A spec claim may mark its mandated test(s) as an
// absence/structural claim; the gate then SKIPS the noTarget set-join for those tests
// (they prove a directory/symbol is ABSENT, so by design they do NOT call the target
// package). The capability is DEFAULT-OFF: an unannotated claim keeps FULL noTarget
// enforcement — the false-negative guard below proves the annotation does not
// blanket-blind the check.

func absenceFixtureDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("testdata", "absence-annotation"))
	if err != nil {
		t.Fatalf("resolving absence fixture dir: %v", err)
	}
	return abs
}

// mandatedByName finds the extracted MandatedTest with the given func name.
func mandatedByName(tests []MandatedTest, name string) (MandatedTest, bool) {
	for _, mt := range tests {
		if mt.FuncName == name {
			return mt, true
		}
	}
	return MandatedTest{}, false
}

// TestExtractMandatedTests_SetsIsAbsenceFromClaimKind (CLM-005/CLM-006) — a claim
// declaring `kind: absence` propagates IsAbsence=true onto its mandated test(s), while
// an ordinary claim's test keeps IsAbsence=false. Both come from the SAME spec, so this
// proves the flag is per-claim, not per-spec.
func TestExtractMandatedTests_SetsIsAbsenceFromClaimKind(t *testing.T) {
	tests, err := ExtractMandatedTests(absenceFixtureDir(t))
	if err != nil {
		t.Fatalf("ExtractMandatedTests: %v", err)
	}

	absent, ok := mandatedByName(tests, "TestWidgetDirectoryAbsent")
	if !ok {
		t.Fatalf("absence-claim test TestWidgetDirectoryAbsent not extracted; got %+v", tests)
	}
	if !absent.IsAbsence {
		t.Errorf("TestWidgetDirectoryAbsent is on a `kind: absence` claim; IsAbsence must be true, got false")
	}

	ordinary, ok := mandatedByName(tests, "TestWidgetIsBuilt")
	if !ok {
		t.Fatalf("ordinary-claim test TestWidgetIsBuilt not extracted; got %+v", tests)
	}
	if ordinary.IsAbsence {
		t.Errorf("TestWidgetIsBuilt is on an ordinary claim; IsAbsence must be false, got true")
	}
}

// TestNoTargetViolationForTest_SkipsAbsenceTest (CLM-006) — an IsAbsence=true test is
// SKIPPED by the wrapper even under the exact inputs that would otherwise raise a
// noTarget violation: a non-empty target package, not same-package, and absent from the
// referenced set. The pre-join skip mirrors the terminal-spec pre-filter; it delegates
// to the UNCHANGED NoTargetViolation only when the test is not an absence test.
func TestNoTargetViolationForTest_SkipsAbsenceTest(t *testing.T) {
	mt := MandatedTest{
		FuncName:  "TestWidgetDirectoryAbsent",
		TargetPkg: "widget",
		IsAbsence: true,
	}
	referenced := ReferencedSymbolSet{} // does NOT contain "widget"
	if v, raised := NoTargetViolationForTest(mt, referenced, false); raised {
		t.Errorf("an absence-tagged test must SKIP the noTarget join (no violation); got raised violation %+v", v)
	}
}

// TestNoTargetViolationForTest_UnannotatedStillRaises (CLM-007) — THE FALSE-NEGATIVE
// GUARD. An IsAbsence=false test with the SAME not-in-set inputs as the absence case
// STILL raises the noTarget violation. This proves the capability does not
// blanket-blind the check: only an explicitly-annotated claim is excused.
func TestNoTargetViolationForTest_UnannotatedStillRaises(t *testing.T) {
	mt := MandatedTest{
		FuncName:  "TestWidgetIsBuilt",
		TargetPkg: "widget",
		IsAbsence: false,
	}
	referenced := ReferencedSymbolSet{} // does NOT contain "widget"
	v, raised := NoTargetViolationForTest(mt, referenced, false)
	if !raised {
		t.Fatalf("an UNannotated test that does not call its target MUST still raise noTarget (default-off enforcement); got no violation")
	}
	if v.Rule != StepTestSubstantiveness {
		t.Errorf("raised violation must be a %s violation; got %q", StepTestSubstantiveness, v.Rule)
	}
	if v.Message != "test function TestWidgetIsBuilt does not call package widget" {
		t.Errorf("unexpected violation message: %q", v.Message)
	}
}
