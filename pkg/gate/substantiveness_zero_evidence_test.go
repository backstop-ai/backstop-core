package gate

import (
	"reflect"
	"strings"
	"testing"
)

// substantiveness_zero_evidence_test.go pins the ISSUE-113 guard: the eligibility
// predicate that decides WHICH mandated tests a starved Q2 join would have raised
// against, and the refusal decision that replaces those N unfounded verdicts with ONE
// honest config-error.
//
// The refusal is a HEURISTIC over an inherently ambiguous observable — core cannot see
// inside a pack (thin executor), so "the extraction partition is empty" is the same
// signal whether the pack's classification matched nothing or the scanned tests really
// make no package-qualified calls. These tests therefore pin BOTH directions: that it
// fires on the starved shape, and — the part a later "simplification" is most likely to
// break — that it does NOT fire whenever any evidence at all exists.

// TestJoinEligibleForNoTarget_MatchesTheNoTargetDecisionTable (CLM-001) — eligibility is
// DEFINED as the noTarget decision table evaluated against an EMPTY evidence set, so the
// guard and the loop it guards can never disagree about which tests are at stake. Each
// row cross-checks the predicate against NoTargetViolationForTest directly; that
// cross-check is the point of the test and stays meaningful if a fifth disposition is
// ever added to the table.
func TestJoinEligibleForNoTarget_MatchesTheNoTargetDecisionTable(t *testing.T) {
	cases := []struct {
		name         string
		mt           MandatedTest
		samePackage  bool
		wantEligible bool
	}{
		{
			name:         "ordinary_test_with_target_is_eligible",
			mt:           MandatedTest{FuncName: "TestOrdinary", FilePath: "root_test.go", TargetPkg: "gate"},
			samePackage:  false,
			wantEligible: true,
		},
		{
			name:         "empty_target_subject_is_not_eligible",
			mt:           MandatedTest{FuncName: "TestNoSubject", FilePath: "root_test.go", TargetPkg: ""},
			samePackage:  false,
			wantEligible: false,
		},
		{
			name:         "same_package_test_is_not_eligible",
			mt:           MandatedTest{FuncName: "TestColocated", FilePath: "gate/x_test.go", TargetPkg: "gate"},
			samePackage:  true,
			wantEligible: false,
		},
		{
			name:         "absence_annotated_test_is_not_eligible",
			mt:           MandatedTest{FuncName: "TestAbsence", FilePath: "root_test.go", TargetPkg: "gate", IsAbsence: true},
			samePackage:  false,
			wantEligible: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := JoinEligibleForNoTarget(tc.mt, tc.samePackage)
			if got != tc.wantEligible {
				t.Fatalf("JoinEligibleForNoTarget(%+v, samePackage=%v) = %v, want %v",
					tc.mt, tc.samePackage, got, tc.wantEligible)
			}
			// The cross-check: "join-eligible" must mean EXACTLY "an empty evidence set
			// would make this test raise". If these two ever diverge, the guard has
			// started deciding on its own predicates instead of the table's.
			_, raised := NoTargetViolationForTest(tc.mt, ReferencedSymbolSet{}, tc.samePackage)
			if got != raised {
				t.Fatalf("eligibility drifted from the decision table for %+v: JoinEligibleForNoTarget=%v but NoTargetViolationForTest(empty set) raised=%v",
					tc.mt, got, raised)
			}
		})
	}
}

// TestSubstantivenessEvidenceRefusal_FiresOnlyOnStarvedJoin (CLM-002) — the refusal fires
// if and only if at least one mandated test is join-eligible AND both routed partitions
// are empty. Every row where any evidence exists must NOT refuse.
func TestSubstantivenessEvidenceRefusal_FiresOnlyOnStarvedJoin(t *testing.T) {
	cases := []struct {
		name       string
		eligible   int
		extraction int
		hollow     int
		wantRefuse bool
	}{
		{name: "starved_join_with_three_eligible", eligible: 3, extraction: 0, hollow: 0, wantRefuse: true},
		{name: "one_eligible_test_is_enough", eligible: 1, extraction: 0, hollow: 0, wantRefuse: true},
		{name: "nothing_eligible_nothing_would_have_been_raised", eligible: 0, extraction: 0, hollow: 0, wantRefuse: false},
		{name: "nothing_eligible_with_hollow_evidence", eligible: 0, extraction: 0, hollow: 5, wantRefuse: false},
		{name: "extraction_evidence_means_the_join_is_honest", eligible: 3, extraction: 1, hollow: 0, wantRefuse: false},
		// THE B1 ROW. Hollow evidence is core's only proof that the pack's engine ran and
		// classified test files at all, so this is NOT a zero-match classification and
		// refusing would DELETE two real hollow violations while printing a diagnosis
		// those very findings falsify. The measured case this guards is the shipped
		// newE2EWorkspace fixture (eligible 1 / extraction 0 / hollow 1): refusing there
		// breaks TestE2E_SubstantivenessInstalledLocalPack_RealGate_HollowRed.
		{name: "hollow_evidence_blocks_refusal", eligible: 3, extraction: 0, hollow: 2, wantRefuse: false},
		{name: "both_channels_populated", eligible: 3, extraction: 7, hollow: 2, wantRefuse: false},
		{name: "the_newE2EWorkspace_shape_exactly", eligible: 1, extraction: 0, hollow: 1, wantRefuse: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, refused := SubstantivenessEvidenceRefusal(tc.eligible, tc.extraction, tc.hollow)
			if refused != tc.wantRefuse {
				t.Fatalf("SubstantivenessEvidenceRefusal(eligible=%d, extraction=%d, hollow=%d) refused=%v, want %v (violation %#v)",
					tc.eligible, tc.extraction, tc.hollow, refused, tc.wantRefuse, v)
			}
			if !tc.wantRefuse {
				if !reflect.DeepEqual(v, Violation{}) {
					t.Fatalf("a non-refusing decision must return the zero Violation; got %#v", v)
				}
				return
			}
			if v.Rule != StepTestSubstantiveness {
				t.Fatalf("the refusal violation must carry Rule %q; got %q", StepTestSubstantiveness, v.Rule)
			}
			if v.Severity != "error" {
				t.Fatalf("the refusal violation must carry Severity %q; got %q", "error", v.Severity)
			}
		})
	}
}

// TestSubstantivenessEvidenceRefusal_MessageNamesTheCause (CLM-003) — there is exactly
// ONE message (condition (c) removed the hollow-count fork's whole reachable state), and
// it must state the observed FACT and name ALL THREE candidate causes without asserting
// any one of them as the diagnosis.
//
// Assertions are on stable substrings, not the whole sentence: the wording is meant to be
// improvable without breaking the pin.
func TestSubstantivenessEvidenceRefusal_MessageNamesTheCause(t *testing.T) {
	v, refused := SubstantivenessEvidenceRefusal(3, 0, 0)
	if !refused {
		t.Fatalf("a starved join with 3 eligible tests must refuse; got refused=%v", refused)
	}
	msg := v.Message

	required := []struct {
		what      string
		substring string
	}{
		{what: "the join-eligible count, so the operator sees the scale of what would otherwise have been reported", substring: "3"},
		{what: "the OBSERVED FACT — the only thing core actually knows", substring: "produced no findings of any kind"},
		{what: "candidate cause (1): the pack's engine did not run", substring: "engine did not run"},
		{what: "candidate cause (2): the pack's classification matched no test files", substring: "classification matched 0 test files"},
		// Candidate cause (3) is the BARE-HELPER-ASSERTION shape, empirically verified
		// against real ast-grep:
		//     func TestUsesBareHelperAssertion(t *testing.T) {
		//         got := Build()
		//         assertEqual(t, got, "x")
		//     }
		// yields ZERO findings from BOTH rules — the Q1 assertion-vocabulary regex matches
		// "assert" inside `assertEqual` (so not a hollow finding), and the Q2 rule requires
		// the call's `function` field to be a selector_expression, so a BARE call produces
		// no extraction finding either. That workspace reaches this refusal with TRUE
		// noTarget verdicts at stake. Naming it is the ONLY mitigation available for that
		// residual, so this pin is not optional garnish — an editor who deletes the cause
		// is hiding the case where this refusal is wrong.
		{what: "candidate cause (3): the tests genuinely make no package-qualified calls", substring: "no package-qualified calls"},
		{what: "candidate cause (3), continued: while still satisfying the pack's assertion vocabulary", substring: "assertion vocabulary"},
		{what: "WHAT is being refused, in words the operator who saw the violation wall will recognize", substring: "refusing instead of reporting"},
		{what: "the per-test verdicts being declined, named verbatim", substring: "does not call package"},
	}
	for _, req := range required {
		if !strings.Contains(msg, req.substring) {
			t.Fatalf("the refusal message must state %s (substring %q); got %q", req.what, req.substring, msg)
		}
	}

	// A message discussing hollow findings would describe a state condition (c) makes
	// UNREACHABLE — the refusal cannot fire while hollow evidence exists. Its reappearance
	// is the regression this catches. Cause (3) is fully expressible without the word (it
	// is worded that way above), so the two assertions are compatible BY DESIGN; if they
	// ever conflict it is the WORDING that gives, never this pin.
	if strings.Contains(msg, "hollow") {
		t.Fatalf("the refusal message must never mention hollow findings — the refusal cannot fire while they exist; got %q", msg)
	}
}
