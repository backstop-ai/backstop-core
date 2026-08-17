package gate

import "testing"

// substantiveness_severity_test.go — ISSUE-106, hop 3 of the pack-severity chain
// (ISSUE-104 parser → ISSUE-105 step verdicts → this join). A pack's declared SARIF
// level IS its blockingness declaration (the ratified contract on blocksVerdict,
// pkg/gate/policy.go); the substantiveness join used to overwrite it with a hardcoded
// "error", so a pack that declared a `warning` hollow rule still blocked the gate.
//
// THE TWO FUNCTIONS COVERED HERE ARE DELIBERATELY ASYMMETRIC, and that asymmetry is the
// point of putting them in one file:
//   - HollowFindingsToViolations FORWARDS a severity it was HANDED by a pack finding;
//   - NoTargetViolation SYNTHESIZES one it was never handed, from a presence-only
//     set-membership test, and its fixed "error" is a ratified decision (CLM-003).
//
// A reader who "finishes the job" by making the second symmetric with the first must
// first invent a declaration channel for it — a new capability, and a separate issue.

// TestQ1_HollowFindingsToViolations_ForwardsPackDeclaredSeverity (CLM-001, CLM-006) —
// the conversion carries each source hollow finding's OWN Severity onto the violation it
// constructs, VERBATIM, with no re-defaulting at the join.
//
// The batch carries THREE DISTINCT severities in ONE call on purpose: a hardcoded
// literal fails the warning case, and a blanket "always warning" fails the error case.
// No constant satisfies this.
func TestQ1_HollowFindingsToViolations_ForwardsPackDeclaredSeverity(t *testing.T) {
	const emptyIdx = 2
	hollow := []Violation{
		{
			Rule:     testHollowRuleID,
			File:     "./pkg/thing/warn_test.go",
			Line:     31,
			Message:  "test function TestWarn has no assertions (hollow) func=TestWarn",
			Severity: "warning",
		},
		{
			Rule:     testHollowRuleID,
			File:     "pkg/thing/err_test.go",
			Line:     44,
			Message:  "test function TestErr has no assertions (hollow) func=TestErr",
			Severity: "error",
		},
		{
			Rule:    testHollowRuleID,
			File:    "pkg/thing/bare_test.go",
			Line:    9,
			Message: "test function TestBare has no assertions (hollow) func=TestBare",
			// Severity deliberately omitted — the verbatim-forwarding case.
		},
	}

	got := HollowFindingsToViolations(hollow)

	if len(got) != len(hollow) {
		t.Fatalf("expected one violation per hollow finding (%d); got %d: %+v", len(hollow), len(got), got)
	}

	// Each output's Severity is ITS OWN source finding's Severity, positionally.
	for i, v := range got {
		if v.Severity != hollow[i].Severity {
			t.Errorf("violation[%d].Severity = %q, want %q (the source finding's own declared severity, "+
				"forwarded verbatim); overwriting it converts a pack's advisory into a blocker — the "+
				"ISSUE-104/ISSUE-105 defect recurring one hop later, inside the converter both bypass",
				i, v.Severity, hollow[i].Severity)
		}
	}

	// THE EMPTY CASE NEEDS ITS OWN VERDICT ASSERTION, not just the pass-through.
	// Forwarding an empty value is safe precisely because blocksVerdict treats anything
	// that is not "warning" as blocking: the join fails CLOSED by construction, so no
	// second defaulting guard is needed here. The value was ALREADY defaulted upstream by
	// the production bridge (cmd/backstop/pack_gate.go, nonEmpty(v.Severity, "error")),
	// and a second default at this join would be a second spelling of one rule.
	if verdict := StepVerdict([]Violation{got[emptyIdx]}); verdict != "fail" {
		t.Errorf("a forwarded EMPTY severity must still BLOCK: StepVerdict = %q, want %q; if this "+
			"ever reports \"warning\" the join has started failing OPEN and a pack could disable "+
			"enforcement by declaring nothing", verdict, "fail")
	}

	// CLM-006 non-regression, asserted HERE so a severity fix that disturbs the ISSUE-116
	// Line carry or the ISSUE-046 File canonicalization fails locally rather than three
	// packages away. Checked on the warning finding, the one whose behavior changes.
	warn := got[0]
	if warn.Rule != StepTestSubstantiveness {
		t.Errorf("violation[0].Rule = %q, want %q", warn.Rule, StepTestSubstantiveness)
	}
	if want := "pkg/thing/warn_test.go"; warn.File != want {
		t.Errorf("violation[0].File = %q, want the canonicalized repo-relative %q", warn.File, want)
	}
	if warn.Line != 31 {
		t.Errorf("violation[0].Line = %d, want 31 (the source finding's own line, ISSUE-116)", warn.Line)
	}
	if want := "test function TestWarn has no assertions (hollow)"; warn.Message != want {
		t.Errorf("violation[0].Message = %q, want %q (the pinned func= routing token stripped)",
			warn.Message, want)
	}
}

// TestNoTarget_SynthesizedSeverityIsFixedByDesign (CLM-003) — the noTarget half of the
// join keeps a FIXED "error" severity, and that is a ratified decision rather than an
// unexamined hardcode.
//
// THIS TEST EXISTS BECAUSE THIS LANE CREATES THE RISK IT GUARDS. Once the sibling half
// (HollowFindingsToViolations) starts forwarding a pack's severity, a future reader will
// reach for symmetry and invent a severity channel here too. The rationale for the
// asymmetry:
//   - the violation is SYNTHESIZED by the gate's own decision table from a set-membership
//     test, not converted from any one pack finding — there is no contributing finding
//     whose severity could be forwarded;
//   - ReferencedSymbolSet is map[string]bool, presence only: the extraction findings that
//     populate it carry no severity into the set at all;
//   - a noTarget verdict is a GATE-COMPUTED defect about the consumer's tests, not an
//     advisory the pack authored, so a fixed severity is the honest reading of what it is.
//
// Changing this therefore requires building a NEW declaration channel first. That is a
// capability expansion and a separate issue, not an edit here.
func TestNoTarget_SynthesizedSeverityIsFixedByDesign(t *testing.T) {
	// Every RAISING input of the decision table: target present, samePackage false, and
	// the target token absent from the set — with the set both POPULATED and EMPTY, so
	// the fixed severity is pinned across both paths into the raise.
	cases := []struct {
		name       string
		referenced ReferencedSymbolSet
	}{
		{"populated set without the target", ReferencedSymbolSet{"other": true, "unrelated": true}},
		{"empty set", ReferencedSymbolSet{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, raised := NoTargetViolation("TestSomething", "target", tc.referenced, false)
			if !raised {
				t.Fatalf("NoTargetViolation must raise for a test whose referenced set omits its target (%s)", tc.name)
			}
			if v.Severity != "error" {
				t.Errorf("NoTargetViolation returned severity %q, want %q — this value is FIXED BY "+
					"DESIGN (CLM-003), not an oversight the hollow-conversion fix forgot to sweep up",
					v.Severity, "error")
			}
		})
	}

	// SubstantivenessEvidenceRefusal keeps a fixed "error" for the same reason and needs
	// no argument beyond it: it is a ConfigErr refusal about the PACK's configuration, it
	// short-circuits the step, and ApplyPolicy skips ConfigErr steps entirely.
	refusal, refuse := SubstantivenessEvidenceRefusal(1, 0, 0)
	if !refuse {
		t.Fatal("SubstantivenessEvidenceRefusal(1, 0, 0) must refuse: one eligible test, no evidence of any kind")
	}
	if refusal.Severity != "error" {
		t.Errorf("SubstantivenessEvidenceRefusal returned severity %q, want %q", refusal.Severity, "error")
	}
}
