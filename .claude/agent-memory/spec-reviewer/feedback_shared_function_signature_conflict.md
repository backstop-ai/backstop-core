---
name: shared-function-signature-conflict
description: Sibling specs co-rewriting one function declare contradictory signatures; check the exact signature line, not just "disjoint functions"
metadata:
  type: feedback
---

When a cutover bundle slices work into parallel specs that all edit ONE function, each spec's contract may declare a DIFFERENT signature for that same symbol — and the validator won't catch it (each spec validates alone).

**Why:** BUNDLE-012 SPEC-043 added `classifier SourceClassifier` to `StepCoverageThresholdScopedFunc` (4-arg) while SPEC-044's contract for the SAME function said "public signature UNCHANGED" (3-arg). Both rewrite the body; only one signature can exist. A planner reading either spec alone produces a different arity.

**How to apply:** On any multi-spec review where two specs name the same function in their `contracts[]`, diff the literal `signature:` strings against each other AND against live code. A spec that says "signature UNCHANGED" while a sibling adds a param is a BLOCKING conflict. State the reconciled signature explicitly and name which spec must update its contract entry. Body-division being logically coherent (e.g. guards at disjoint granularities) does NOT excuse a signature contradiction.
