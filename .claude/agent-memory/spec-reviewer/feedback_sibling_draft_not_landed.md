---
name: sibling-draft-not-landed
description: a spec citing a co-developed sibling spec's type as "verified on main" when that spec is still draft and the type doesn't exist
metadata:
  type: feedback
---

When a spec grounds itself on a sibling spec's artifact (a type, a step, a parser) and labels it "verified on `main`", check whether that sibling has actually LANDED in code, not just been authored as a draft.

**Why:** SPEC-042 (coverage PRODUCER) repeatedly stated, as `main`-verified fact, that SPEC-041 (coverage CONSUMER) "re-implemented the coverage gate step to consume `[]CoverageRecord{Path, Pct, Measured, Excluded}`." Verified on main: no `CoverageRecord` type, no `ParsePackCoverage`, no records-based step exist anywhere; SPEC-041 is still `status: draft`; the live coverage step is the OLD baked Go `StepCoverageThresholdFunc` (pkg/gate/step_coverage.go) the bundle is eradicating. The spec conflated "the sibling draft declares this shape on paper" with "main ships this shape." This weakened the align-predating-artifacts framing too: there's no landed artifact to predate when both specs are co-developed drafts.

**How to apply:** For any producer↔consumer or Seed-N↔Seed-M cross-spec contract, grep main for the actual symbol (`type X struct`, `func ParseX`) before accepting the spec's "verified on main" reference. If the sibling is draft and the symbol is absent, the correct framing is "agree one canonical type across two co-developed drafts before either lands," NOT "reconcile against the predating consumer shape." Also check whether the sibling draft's own notes already hedge the divergence (SPEC-041's notes said "Pct (or Covered/Total)") — the reconciliation may be smaller than the producer spec portrays. Related: [[feedback_coverage_producer_gap]], [[feedback_rekey_faithfulness]].
