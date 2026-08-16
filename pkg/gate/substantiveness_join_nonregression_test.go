package gate

import (
	"fmt"
	"testing"
)

// substantiveness_join_nonregression_test.go is CLM-007's dedicated home: the pins that
// say the ISSUE-113 refusal changed the join's BOUNDARY and nothing else.
//
// Two things live here. First, an exhaustive sweep of the refusal predicate over the
// (eligible, extraction, hollow) space, written as the boundary EXPRESSION rather than as
// an enumeration — that is the pin that mechanically prevents the `hollow == 0` term from
// being dropped as "redundant" by a later simplification, which is exactly the false
// refusal this design rejected. Second, the pre-existing decision-table behaviors under
// NON-empty evidence, which the refusal sits next to and must leave untouched.
//
// ISSUE-116 WARNING, still live: HollowFindingsToViolations must keep FORWARDING Line.
// It was once silently zeroed, which made inline test_substantiveness waivers a no-op.
// A refusal added nearby must not re-zero it.

// TestSubstantivenessEvidenceRefusal_ExhaustiveBoundary (CLM-007, CLM-002) — sweeps each
// of the three counts over 0..3 and asserts the decision equals the boundary expression
// in every one of the 64 cells. Stating the expected side as the expression itself makes
// this test a statement OF the boundary rather than a list of remembered answers.
func TestSubstantivenessEvidenceRefusal_ExhaustiveBoundary(t *testing.T) {
	for eligible := 0; eligible <= 3; eligible++ {
		for extraction := 0; extraction <= 3; extraction++ {
			for hollow := 0; hollow <= 3; hollow++ {
				name := fmt.Sprintf("eligible=%d/extraction=%d/hollow=%d", eligible, extraction, hollow)
				t.Run(name, func(t *testing.T) {
					want := eligible >= 1 && extraction == 0 && hollow == 0
					_, refused := SubstantivenessEvidenceRefusal(eligible, extraction, hollow)
					if refused != want {
						t.Fatalf("refusal boundary broken at (eligible=%d, extraction=%d, hollow=%d): refused=%v, want %v",
							eligible, extraction, hollow, refused, want)
					}
				})
			}
		}
	}
}

// TestNoTargetJoinUnchangedUnderEvidence (CLM-007) — the noTarget decision table with a
// NON-empty referenced set still behaves exactly as documented on NoTargetViolation.
// TASK-004 adds functions to that file and must change none of this.
func TestNoTargetJoinUnchangedUnderEvidence(t *testing.T) {
	populated := ReferencedSymbolSet{"gate": true, "testing": true}

	t.Run("target_in_a_populated_set_is_satisfied", func(t *testing.T) {
		v, raised := NoTargetViolation("TestX", "gate", populated, false)
		if raised {
			t.Fatalf("a test referencing its target package must NOT raise; got %#v", v)
		}
	})

	t.Run("target_absent_from_a_populated_set_raises", func(t *testing.T) {
		v, raised := NoTargetViolation("TestX", "check", populated, false)
		if !raised {
			t.Fatal("a test whose target is absent from a NON-empty referenced set must still raise — that is the false-negative guard")
		}
		if v.Rule != StepTestSubstantiveness || v.Severity != "error" {
			t.Fatalf("the noTarget violation must keep its rule/severity; got %#v", v)
		}
		if want := "test function TestX does not call package check"; v.Message != want {
			t.Fatalf("the noTarget message must be unchanged; got %q want %q", v.Message, want)
		}
	})

	t.Run("same_package_is_satisfied_even_with_the_target_absent", func(t *testing.T) {
		if _, raised := NoTargetViolation("TestX", "check", populated, true); raised {
			t.Fatal("a same-package test must never raise regardless of the referenced set")
		}
	})

	t.Run("empty_target_is_skipped", func(t *testing.T) {
		if _, raised := NoTargetViolation("TestX", "", populated, false); raised {
			t.Fatal("an empty target subject must be SKIPPED, not raised")
		}
	})
}

// TestAbsenceSkipAndHollowConversionUnchanged (CLM-007) — the two neighbours of the new
// refusal that a nearby edit could plausibly disturb: the ISSUE-035 Category 2 absence
// skip, and the ISSUE-116 Line forwarding in the hollow conversion.
func TestAbsenceSkipAndHollowConversionUnchanged(t *testing.T) {
	t.Run("absence_annotated_test_skips_even_with_evidence_present", func(t *testing.T) {
		// Non-empty referenced set, target genuinely absent from it — the shape that
		// WOULD raise for an unannotated test.
		mt := MandatedTest{FuncName: "TestAbsence", FilePath: "root_test.go", TargetPkg: "check", IsAbsence: true}
		if v, raised := NoTargetViolationForTest(mt, ReferencedSymbolSet{"gate": true}, false); raised {
			t.Fatalf("a `kind: absence` test must skip the noTarget join even under evidence; got %#v", v)
		}
		// The false-negative guard: the SAME shape unannotated still raises, so the skip
		// above is a real skip and not a broken table.
		mt.IsAbsence = false
		if _, raised := NoTargetViolationForTest(mt, ReferencedSymbolSet{"gate": true}, false); !raised {
			t.Fatal("the unannotated twin must still raise — otherwise the absence skip above proves nothing")
		}
	})

	t.Run("hollow_conversion_still_forwards_file_and_line", func(t *testing.T) {
		hollow := []Violation{{
			Rule:     "pack/hollow-test-go",
			File:     "./pkg/gate/subject_test.go",
			Line:     42,
			Message:  "test TestHollow has no assertions (hollow) func=TestHollow",
			Severity: "error",
		}}
		out := HollowFindingsToViolations(hollow)
		if len(out) != 1 {
			t.Fatalf("one routed hollow finding must yield one violation; got %d: %#v", len(out), out)
		}
		got := out[0]
		if got.Line != 42 {
			t.Fatalf("ISSUE-116 regression: the hollow conversion must FORWARD the finding's Line (inline waivers byte-scan it); got Line=%d in %#v", got.Line, got)
		}
		if want := "pkg/gate/subject_test.go"; got.File != want {
			t.Fatalf("the hollow conversion must carry the canonicalized File; got %q want %q", got.File, want)
		}
		if want := "test TestHollow has no assertions (hollow)"; got.Message != want {
			t.Fatalf("the hollow conversion must keep the report-surface message with the routing token stripped; got %q want %q", got.Message, want)
		}
		if got.Rule != StepTestSubstantiveness {
			t.Fatalf("the converted violation must carry the step rule; got %q", got.Rule)
		}
	})
}
