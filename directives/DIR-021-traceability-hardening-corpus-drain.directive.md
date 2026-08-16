---
title: "Traceability Hardening & Corpus Drain"
number: DIR-021
created: "2026-07-15"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "ISSUE-057"
    - "ISSUE-048"
    - "ISSUE-012"
    - "ISSUE-061"
    - "ISSUE-138"
---

## Description

Harden the requirement-traceability substrate BUNDLE-014 shipped
(delivered 2026-07-15) and drain the corpus debt it and its predecessors
left behind. Five threads, one theme: close the residual gaps between what
the gate structurally verifies today and what it still takes on prose/trust.

1. **Cited-bundle maturity floor (ISSUE-057).** BUNDLE-014's
   `ResolveSupports` resolves a spec's `supports` ref fully — bundle
   exists, REQ is declared, the pinned version is logged — but never checks
   the cited bundle's own `status.maturity`. A spec can cite
   `bundle:REQ-NNN@X.Y.Z` against a bundle still at `idea`/`exploring` and
   resolve fully clean today. This is a **`chain_outrun`-family** gap in the
   same DD-11 sense BUNDLE-014 already named (a downstream artifact citing
   an upstream chain that hasn't actually verified) — it just isn't
   detected by the shipped `chain_outrun` classification, which only
   catches non-verifying specs/plans, not non-promoted bundles. Low
   day-to-day exposure (founder-confirmed, 2026-07-14) but a real hole in
   the mechanism that prevents fossil specs (the BUNDLE-003 SPEC-020..029
   incident this issue exists to close for good).
2. **Stranded terminal lineage tail (ISSUE-048).** Two closed issues
   (ISSUE-018, ISSUE-036) still carry 7 live `artifact_status_drift`
   violations — mandated tests that were genuinely never restored/repointed
   when the underlying code moved on. Six of the original eight artifacts in
   this cluster are already reconciled; this is the last tail, needing a
   per-artifact judgment call (repoint to a surviving equivalent, or retire
   honestly via `obsoleted`) rather than a mechanical fix.
3. **BUNDLE-010 mandated-test debt (ISSUE-012).** SPEC-017 and SPEC-031
   (BUNDLE-010, `pack-gate-integration` / `pluggable-engine-dispatch`)
   mandate roughly 58 test functions that were never written — concealed
   for months because `test_verification`'s mandated-test existence check
   was diff-scoped, so an untouched spec's absent tests never surfaced
   until an unrelated edit dragged the spec file back into scope. The
   diff-scope design question this surfaced may itself need a
   plan-time decision (existence-checking probably needs to be scope-wide
   for `implemented` specs, independent of whether pass/fail evaluation
   stays diff-scoped).
4. **163-gap `requirement_traceability_advisory` drain.** The advisory
   surface SPEC-052 shipped alongside the blocking `requirement_traceability`
   step currently warns on 163 in-flight coverage gaps (per
   `HANDOFF-requirement-traceability.md`) — this is, by design, the
   spec-writing backlog for bundles that have REQs without yet-implemented
   supporting specs. There is no issue artifact for this by design: the
   gate itself IS the tracker (`requirement_traceability_advisory` is the
   live worklist, not a snapshot to reconcile once). Draining it is
   spec-authoring work against the in-flight bundles it's already pointing
   at, tracked by watching the warn count fall, not by a checklist.
5. **Dormant SPEC-041 substantiveness claims, untouched-file class
   (ISSUE-138).** Five `SPEC-041` (`pkg/gate`, `implemented`) mandated
   tests across two files — `TestExemption_PerViolationResolutionNoGateTypeAggregation`
   (CLM-018), `TestExemption_TrueConflictExemptingValueWins` (CLM-019),
   `TestSharedRunner_Eradicated` (CLM-004), `TestSharedRunner_WiringRemovedFromGate`
   (CLM-005), `TestSharedRunner_NoRenamedWholeModuleGoTestRunner` (CLM-006) —
   carry LATENT `test_substantiveness` "does not call package gate"
   violations that only surface once their file next re-enters diff scope
   for an unrelated change, exactly the same mechanism as item 3
   (ISSUE-012): a scope-gated check whose absence-of-evidence stays
   concealed until an untouched artifact is dragged back into scope by
   something else entirely. Disposition (founder ruling, 2026-08-16): three
   of the five (CLM-004/005/006, all in `shared_testrun_eradication_test.go`)
   prove an absence by design — an `os.Stat` for a deleted file, a grep of
   source text for a forbidden construct — so annotate them `kind: absence`
   in `SPEC-041`, matching the existing CLM-020..024 precedent in the same
   spec (`specs/SPEC-041-coverage-reimpl-checktype-catalog.spec.md:363-407`).
   The remaining two (CLM-018/CLM-019) are shape-ambiguous — they genuinely
   drive `pkg/gate` through the same-package helper `dispatchOneEngine`,
   one hop past what the extraction rule can see — but route the same way
   rather than teaching the rule to see through helper indirection, since
   `PLAN-ISSUE-113` (concurrent) is actively pinning the join's current
   helper-blind raise as intended non-regression behavior, cutting against
   ever changing it. Spec-author work only; no core/pack code change. ISSUE-061 (`go-standards`
`error-type-suffix` false positive on `ValidateConfig`,
`cmd/backstop/artifact_validate.go:17`) is suppressed by an inline waiver
that **expires 2026-10-12**. The fix belongs in the `backstop/go-standards`
pack repo (not here) — a relational rule scoped to a struct's own `Error()`
method, replacing the current whole-file DOTALL regex that treats "any
struct" and "any later `Error()` method anywhere in the file" as related
evidence. On expiry, `backstop gate` goes red on a false positive with no
fix in flight today unless this lands first. This is why this directive
sits first among the four newly-added ones.

## Notes

This directive bundles four traceability/gate-correctness threads that
share a "close the residual verification gap" theme, rather than becoming
four one-issue directives — per directive-authoring convention, granular
work rolls up under a directive; it doesn't each get its own. ISSUE-057 is
explicitly founder-flagged LOW/non-urgent; ISSUE-048 and ISSUE-012 are
judgment-heavy cleanup with no external deadline; ISSUE-061 is the one item
with a hard 2026-10-12 deadline and should be picked up first regardless of
how the rest of this directive sequences.

Placed first among the four directives added in this pass (ahead of
Contracts Engine Hardening, Pack Distribution Hardening, and Gate/Engine
Quality) specifically because of the ISSUE-061 deadline — position is
priority, and the founder should reprioritize freely if that changes.
