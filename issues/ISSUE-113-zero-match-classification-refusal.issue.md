---
title: "Classification matching zero test files should be a loud config-error refusal, not silent mass join violations"
schema_version: issue/v1

issue:
  id: ISSUE-113
  title: "Classification matching zero test files should be a loud config-error refusal, not silent mass join violations"
  type: enhancement
  status: closed
  created: "2026-07-29"
  closed: "2026-08-16"

delivered_by: PLAN-ISSUE-113
---

# Zero-match classification: refuse loudly

## Resolution

Delivered by PLAN-ISSUE-113 (status: completed), commit `4dbf64b`. `SubstantivenessEvidenceRefusal`
now fires a single config-error refusal (`buildTestSubstantivenessStep`, `pkg/gate/
substantiveness_join.go`) exactly when the join-eligible count is `>= 1` AND the extraction count is
`0` AND the hollow count is `0` (CLM-002) — replacing hundreds of misleading "does not call package
X" violations with one message naming the observed fact and all three candidate causes (CLM-003),
short-circuiting per-test noTarget violations (CLM-004), and structurally unsuppressible because
`ConfigErr` halts the run before waiver/baseline machinery ever sees the step (CLM-008).

**Corrected mid-review, before any code shipped.** An earlier revision of the plan proposed the
narrower `eligible > 0 && extraction == 0` (ignoring the hollow partition). Plan review caught that
this would delete real, correctly-attributed hollow violations whenever a pack's classification
genuinely matched test files but only the Q2 (extraction) rule found nothing — a run already RED
with true findings would have had those true findings destroyed by the refusal. The condition was
tightened to require `hollow == 0` before implementation began, so the narrower version was never a
shipped defect, only a caught draft.

**Two residual gaps accepted and disclosed, not fixed** (documented in the plan's condition
section, "KNOWN, DELIBERATE RESIDUALS"):
- **Residual 1 — under-refusal at `hollow > 0`.** A pack that bakes its globs onto the Q2
  (extraction) rule only, leaving Q1 (assertion-vocabulary) healthy, lands in the hollow > 0 branch
  and does NOT trigger the refusal — its noTarget wall is not collapsed. Both incidents that
  motivated this issue (the missing-ast-grep case, and the published typescript-substantiveness
  1.1.0 baked-globs case) land in the `hollow == 0` branch and ARE covered; a Q2-only starvation
  case, if ever observed in the wild, is scope for a new issue, not a loosening of this condition.
- **Residual 2 — over-refusal at `hollow == 0`.** A test whose only assertion is an unqualified
  helper call (e.g. `assertEqual(t, got, want)`) matches the Q1 vocabulary regex (not hollow) but
  fails the Q2 rule's `selector_expression` requirement (no extraction finding) — empirically
  confirmed against real ast-grep. Such a workspace reaches `eligible >= 1 / extraction 0 / hollow
  0` and refuses, suppressing a noTarget verdict that was actually TRUE. Not fixable by narrowing
  the condition: core has no observable that distinguishes this case from a genuinely-starved
  classification — packs are opaque by design, and inventing information the pack doesn't provide
  was rejected. Mitigated by honesty instead: the refusal message names all three candidate causes
  rather than asserting the pack is broken, so an operator on a bare-helper-assertion codebase is
  pointed at their own test style as one possibility.

Both residuals are pinned by `TestSubstantivenessEvidenceRefusal_ExhaustiveBoundary` (CLM-007) over
the full `(eligible, extraction, hollow)` space, so neither can be silently widened or narrowed by a
later "simplification" without a test catching it.

## Problem

When a pack's classification globs match ZERO test files, the substantiveness join silently emits a
"does not call package X" violation for EVERY mandated test — hundreds of misleading findings whose
real cause (empty classification) is named nowhere. Hit twice in one week by bclabs-portal: (1) the
published typescript-substantiveness 1.1.0 shipping harness-baked globs (397 false violations), and
(2) the missing-ast-grep case (same signature, different root). Both cost hours; both would have been
one line: "classification matched 0 test files".

## Direction

Extend the ISSUE-020 config-error refusal philosophy: when mandated tests exist but the classifier
matches zero test files (or the substantiveness evidence set is empty while mandated tests exist),
the step REFUSES with a config-error naming its cause instead of emitting per-test violations.
Founder-ack'd (Brandon, 2026-07-28) for slotting per PM flow (DIR-024 recommended).
