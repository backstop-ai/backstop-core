---
title: "Remove final-phase category-coverage check (delete fileCategory)"
schema_version: issue/v1

issue:
  id: ISSUE-033
  title: "Remove final-phase category-coverage check (delete fileCategory)"
  type: technical-debt
  status: closed
  created: "2026-07-05"
  closed: "2026-07-09"

resolved-by: 3756e1c

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Remove final-phase category-coverage check (delete fileCategory)

## Problem

`pkg/validate/plan.go`'s `fileCategory()` (~line 704) classifies a plan
task's touched files into work categories — `"artifact"`, `"code"`, or `""`
(uncategorized) — by file-extension matching. `validateFinalPhase`
(~line 601) uses that classification to drive a "final-phase category
coverage" requirement (`plan/final-phase-missing-category`, ~line 692): every
category of file touched anywhere in the plan must be covered by a
verification task in the plan's *final* phase, or the plan is rejected.

The `"code"` bucket is decided by a single baked literal:

```go
if strings.HasSuffix(path, ".go") { // nosemgrep: no-language-literal-on-neutral-spine — de-Go tracked by ISSUE-033
    return "code"
}
```

This was originally scoped (2026-07-05) as a de-Go'ing job: replace the
baked `.go` suffix with a pack-declared, language-neutral classifier
(SPEC-043's `gate.SourceClassifier`), mirroring what `mergeSourceClassifier`
already does for the gate's coverage step. That framing is **rejected**.
Sourcing the classifier language-neutrally would still leave the check
itself intact, and the check itself is the wrong idea:

- **It's a redundant filter on top of a check that already does the real
  job.** `plan/gate-cadence-missing` (~line 920) already requires every
  phase containing implementation/refactor tasks to also contain a
  verification task, and `plan/final-phase-no-verification` (~line 660)
  already requires the final phase specifically to contain at least one
  verification task. Together those two rules encode backstop's actual
  invariant: verify at every step. Category-coverage doesn't add a new
  invariant — it second-guesses those verification tasks by inspecting
  *which files* they happen to touch, as if a verification task could only
  be "real" if its file list literally overlaps the categories touched
  earlier in the plan.
- **It's antithetical to "gates run at every verification step."** A
  verification task in backstop's model runs the full gate — every
  installed pack's rules, not a scoped subset keyed to touched-file
  categories. Filtering "did verification cover the `code` category vs the
  `artifact` category" assumes verification is partial/scoped when it
  isn't. The classification exists to answer a question the model doesn't
  actually ask.
- **It bakes a Go-specific literal on backstop's own neutral spine** (the
  proximate finding that triggered the original issue) — `strings.HasSuffix`
  on `.go` silently mis-classifies every non-Go plan's task files as `""`
  (uncategorized), which is the dangerous direction of failure: it doesn't
  wrongly block a non-Go plan, it silently exempts one from a requirement
  that was never sound to begin with. De-Go'ing it would fix invariant (1)
  but leave the actual defect — a redundant, over-engineered filter — in
  place.

The sound resolution is deletion, not language-neutralization.

## Solution

Delete `fileCategory` and the category-coverage half of
`validateFinalPhase` entirely. Keep the no-verification half and both
cadence rules untouched — they are the correct, already-language-neutral
encoding of "verify at every step."

1. In `pkg/validate/plan.go`:
   - Delete `fileCategory()` (~line 704-718).
   - In `validateFinalPhase` (~line 601), delete the category-coverage block
     that follows the `hasVerification` check (~line 664-698): the
     `requiredCategories` collection, the `coveredCategories` collection,
     the per-category loop, and the `plan/final-phase-missing-category`
     violation it emits. Leave the `hasVerification` /
     `plan/final-phase-no-verification` block and the function's early
     returns unchanged.
   - Confirm no other call site references `fileCategory` after deletion
     (the two call sites at ~line 672 and ~line 682 disappear along with
     the block that contains them).
2. In `pkg/validate/plan_final_test.go`: delete
   `TestPlan_FinalPhase_ComprehensiveVerification` (~line 39-57) and
   `TestPlan_FinalPhase_IncompleteVerification` (~line 59-78) — the two
   tests that assert on `plan/final-phase-missing-category`. Leave
   `TestPlan_FinalPhase_HasVerification` and `TestPlan_FinalPhase_NoVerification`
   (the `plan/final-phase-no-verification` tests) untouched.
3. Leave `plan/gate-cadence-missing` and `plan/final-phase-no-verification`
   — and their tests in `pkg/validate/plan_gate_test.go` /
   `pkg/validate/plan_final_test.go` — completely untouched. This issue
   removes the redundant filter, not the underlying cadence invariant.

**Acceptance:** `fileCategory` no longer exists anywhere in
`pkg/validate/plan.go`; `plan/final-phase-missing-category` is no longer
emitted or referenced anywhere in the codebase; the corresponding two test
functions in `plan_final_test.go` are gone; `plan/gate-cadence-missing` and
`plan/final-phase-no-verification` still pass their existing tests
unmodified. The `backstop/self` `no-language-literal-on-neutral-spine`
suppression on `pkg/validate/plan.go` (added by ISSUE-018's interim fix,
referencing this issue by ID) is drained because the line it suppresses no
longer exists — not because it was replaced with a classifier.

## References

- **ISSUE-048 (Reconcile Stranded Terminal Lineage) — SPEC-002 reconciliation
  tracked separately, NOT in this issue's scope.** SPEC-002
  (`specs/SPEC-002-plan-schema-evolution.spec.md`) mandates the
  category-coverage behavior this issue deletes: REQ-004's text
  ("...covering every category of work the plan performs...") plus
  CLM-010/CLM-011/CLM-026/CLM-027 (`requirement: REQ-004`, ~lines
  177-188, 319-327) map directly onto `plan/final-phase-missing-category`
  and the two test functions this issue removes. Deleting the check strands
  those claims and half of REQ-004 (the other half — final phase must
  contain *a* verification task — survives via
  `plan/final-phase-no-verification` and is unaffected). SPEC-002 itself is
  a `closed`/success-terminal spec; editing its requirements/claims to
  match reality is exactly the kind of stranded-lineage reconciliation
  ISSUE-048 exists to track. Do not fold that edit into this issue — this
  issue is the code-and-tests deletion only.
- `pkg/validate/plan.go:601` — `validateFinalPhase`, the function this
  issue trims (keeps the `hasVerification` block, deletes the
  category-coverage block)
- `pkg/validate/plan.go:660` — `plan/final-phase-no-verification` violation
  emission (kept)
- `pkg/validate/plan.go:692` — `plan/final-phase-missing-category` violation
  emission (deleted)
- `pkg/validate/plan.go:704-718` — `fileCategory()` (deleted in full)
- `pkg/validate/plan.go:920` — `plan/gate-cadence-missing` violation
  emission (kept, untouched)
- `pkg/validate/plan_final_test.go:39-57` —
  `TestPlan_FinalPhase_ComprehensiveVerification` (deleted)
- `pkg/validate/plan_final_test.go:59-78` —
  `TestPlan_FinalPhase_IncompleteVerification` (deleted)
- `pkg/validate/plan_final_test.go:11-35` —
  `TestPlan_FinalPhase_HasVerification` /
  `TestPlan_FinalPhase_NoVerification` (kept, untouched)
- `pkg/validate/plan_gate_test.go` — `plan/gate-cadence-missing` tests
  (kept, untouched)
- ISSUE-018 — the deletion issue whose interim `// nosemgrep` suppression on
  the now-deleted `.go` literal referenced this issue by ID; that
  suppression is drained (not converted) by this issue's fix
- `backstop/self` pack rule `no-language-literal-on-neutral-spine` — the
  dogfood finding this issue eradicates by deleting the offending line,
  not by replacing it with a classifier
- CLAUDE.md, "Enforcement philosophy" — "the enemy is silent/vacuous green,
  not passing"; category-coverage inverted this by silently exempting
  non-Go plans rather than failing loud, which is part of why deletion
  (not repair) is the right call

## Resolution

The final-phase category-coverage check was DELETED rather than de-Go'd, per
the founder's decision that inferring verification adequacy from which files
a task touched is a filter that skips checks — antithetical to "every
verification step runs the full gate." Commit 3756e1c removed:
`fileCategory()` (and its baked `.go` literal), the
`checkFinalPhaseCategoryCoverage` category block emitting
`plan/final-phase-missing-category`, the now-unused `allTasks` parameter of
`validateFinalPhase`, and the two tests
`TestPlan_FinalPhase_ComprehensiveVerification` /
`TestPlan_FinalPhase_IncompleteVerification`. The sound, language-neutral
cadence checks remain untouched: `plan/gate-cadence-missing` (every working
phase needs a verification task) and `plan/final-phase-no-verification`
(final phase needs a verification task) — both task-type based. The
ISSUE-018 deletion-guard `TestPlan_FileCategory_NoStandardMd`
(CLM-003/011) was refreshed in place (same name, so ISSUE-018's
mandated-test pointer stays valid) to prove the stronger post-deletion fact.

Note the consequences: this drained the last `backstop/self`
no-language-literal suppression (formerly plan.go:714) — backstop/self now
has zero baked-language carve-outs — and removed the final residual literal
that DIR-014 tracked. It intentionally strands SPEC-002's category-coverage
requirement + claims (CLM-026/CLM-027), whose reconciliation (retirement,
not repoint) is tracked under ISSUE-048.

Verification actually run: `go build ./...` ok, `go test ./...` green,
`./bin/backstop gate` exit 0 (8 passed / 0 failed), `./bin/backstop artifact
validate` all checks passed.
