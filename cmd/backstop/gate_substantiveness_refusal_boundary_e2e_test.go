package main

import (
	"strings"
	"testing"
)

// gate_substantiveness_refusal_boundary_e2e_test.go pins the ISSUE-113 refusal's
// BOUNDARY end-to-end: hollow evidence BLOCKS the refusal, and the hollow violations
// survive and are reported.
//
// This is the standing regression guard against a false refusal that an earlier design
// would have shipped. Refusing on (eligible >= 1 && extraction == 0) ALONE fires on this
// very run and, via the short-circuit return, throws the REAL hollow violation away —
// turning a correct RED into an exit-2 that blames the pack's configuration while
// SUPPRESSING the finding the operator needed to see. The `hollow == 0` term is what
// makes the refusal safe. Anyone deleting it as redundant should read this file first.
//
// NO NEW FIXTURE IS NEEDED: the already-shipped newE2EWorkspace has exactly the required
// shape — one root-level HOLLOW mandated test against target package `gate`, giving
// eligible 1 / extraction 0 / hollow 1.
//
// READ THIS BEFORE TRUSTING A GREEN HERE: before the production short-circuit lands, the
// step never sets ConfigErr at all, so assertion 1 below passes VACUOUSLY. This file's
// real value is POST-fix. Confirm it is green for the right reason — ConfigErr false
// because hollow evidence blocked the refusal, not because the refusal does not exist.
//
// ast-grep absence is a t.Fatal via requireAstGrepE2E, NEVER a t.Skip.

// TestE2E_HollowEvidenceBlocksZeroMatchRefusal (CLM-004, CLM-007) — the adversarial
// pairing: the Q2 classification matches nothing (the ISSUE-113 trigger) but the Q1
// hollow rule still fires. The refusal must NOT fire, and the true hollow finding must
// survive.
//
// WHAT A FAILURE HERE MEANS: the refusal condition has been widened past hollow == 0.
// That change also breaks two SPEC-037-mandated tests —
// TestE2E_SubstantivenessInstalledLocalPack_RealGate_HollowRed
// (gate_substantiveness_e2e_test.go, mandated by
// specs/SPEC-037-traceability-substantiveness-pack.spec.md) and
// TestE2E_SubstantivenessUninstalled_NoVacuousGreen's cross-check in the same file. Fix
// the condition, never these tests.
func TestE2E_HollowEvidenceBlocksZeroMatchRefusal(t *testing.T) {
	requireAstGrepE2E(t)

	ws, err := newE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding the e2e workspace: %v", err)
	}
	// The PATCHED, Q2-starved pack: extraction is structurally guaranteed empty while the
	// Q1 rule keeps working.
	if err := ws.installZeroMatchSubstantivenessPack(repoRoot(t)); err != nil {
		t.Fatalf("installing the zero-match (Q2-starved) local pack: %v", err)
	}

	res := ws.runProductionSubstantivenessStep()

	// 1. the refusal must NOT fire — hollow evidence proves the pack ran and classified
	// test files, which FALSIFIES the very diagnosis the refusal message would print.
	if res.ConfigErr {
		t.Fatalf("hollow evidence must BLOCK the refusal: the pack demonstrably ran and demonstrably classified "+
			"test files, so refusing here would print a diagnosis its own findings falsify. Got ConfigErr=true with %#v", res.Violations)
	}

	// 2. and the real hollow violation SURVIVED — it was not swallowed by a short-circuit.
	if !hasSubstantivenessHollowViolation(res) {
		t.Fatalf("the true hollow violation must be reported, not discarded; got %s", renderViolations(res.Violations))
	}

	// 3. no refusal message leaked into the result alongside it.
	for _, v := range res.Violations {
		if strings.Contains(v.Message, zeroMatchRefusalSubstring) {
			t.Fatalf("no refusal message may be emitted while hollow evidence exists; got %q", v.Message)
		}
	}

	// 4. the gate is still RED, reporting a TRUE finding. That is the whole point: this
	// run should block, on the merits, rather than exit 2 blaming the pack.
	if res.Status != "fail" {
		t.Fatalf("the hollow finding must still fail the step; got status %q with %s", res.Status, renderViolations(res.Violations))
	}
}

// TestE2E_HollowEvidenceBlocksRefusal_IsNotVacuous (CLM-004) — the control that keeps the
// test above honest.
//
// STATE THE RATIONALE PRECISELY, because it is easy to get wrong: this is ROBUSTNESS, not
// patch-detection. It is NOT true that the test above would pass identically against an
// unpatched pack for some other reason to be ruled out here — newE2EWorkspace's test body
// is `doSubject()`, an UNQUALIFIED call, which produces no Q2 extraction finding even
// under the unpatched pack. For THIS workspace the patch changes nothing observable, so
// this control cannot and does not detect whether the pack was patched. What the patched
// pack buys is that extraction == 0 is a STRUCTURAL guarantee — true regardless of what
// newE2EWorkspace's body happens to contain — rather than an accident of one fixture's
// body. If a future edit gives that fixture a body that WOULD produce a Q2 match, the
// boundary cell this file pins (eligible 1 / extraction 0 / hollow 1) stays put.
//
// WHAT THIS CONTROL ACTUALLY PROVES: that the Q2 starvation is genuinely reproducible in
// THIS run — the join really did come up empty and really did raise a noTarget verdict.
// Pinning the reproduction is what stops the test above from going green because the join
// quietly stopped running at all.
//
// The unpatched-pack control for this workspace is already shipped as
// TestE2E_SubstantivenessInstalledLocalPack_RealGate_HollowRed; it is not duplicated here.
func TestE2E_HollowEvidenceBlocksRefusal_IsNotVacuous(t *testing.T) {
	requireAstGrepE2E(t)

	ws, err := newE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding the e2e workspace: %v", err)
	}
	if err := ws.installZeroMatchSubstantivenessPack(repoRoot(t)); err != nil {
		t.Fatalf("installing the zero-match (Q2-starved) local pack: %v", err)
	}

	res := ws.runProductionSubstantivenessStep()

	// Matched on the per-test verdict's `test function <NAME> does not call package <PKG>`
	// prefix rather than on "does not call package" alone: the refusal message QUOTES that
	// phrase when naming what it declined to report, so the looser substring would let a
	// refusal masquerade as the starvation evidence this control exists to prove.
	starved := false
	for _, v := range res.Violations {
		if strings.Contains(v.Message, "test function ") && strings.Contains(v.Message, "does not call package") {
			starved = true
		}
	}
	if !starved {
		t.Fatalf("the Q2 starvation must be genuinely reproduced in this run — an empty join really did raise a "+
			"noTarget verdict. Without it the boundary test above could go green merely because the join stopped "+
			"running. Got %s", renderViolations(res.Violations))
	}
}
