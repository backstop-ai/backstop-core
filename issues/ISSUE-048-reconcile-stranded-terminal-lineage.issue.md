---
title: "Reconcile Stranded Terminal Lineage"
schema_version: issue/v1

issue:
  id: ISSUE-048
  title: "Reconcile Stranded Terminal Lineage"
  type: technical-debt
  status: open
  created: "2026-07-08"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# Reconcile Stranded Terminal Lineage

## Problem

This is the tracked follow-up ISSUE-042 explicitly deferred. ISSUE-042 ("Gate
Flags Artifact Status Reality Drift", shipped `0ce7f3d`, parent DIR-016) added
a new `artifact_status_drift` gate dimension that resolves each artifact's
mandated tests and cross-checks their existence against the artifact's
declared `status`. Running that dimension full-sweep over this repo surfaced
three distinct, related problems that ISSUE-042 named but did not fix.

### Problem 1 — ~39 stranded success-terminal broken promises

The drift dimension found 39 success-terminal artifacts (`status: closed` /
`implemented`) whose mandated tests are ABSENT. All 39 trace to deliberate
past eradications, not regressions — the underlying capability was
intentionally removed by later superseding work, and the removal took the
verifying tests with it. They are correctly grandfathered into the baseline
(`applies-to: new-code`), so they do not currently block the gate, but they
are real stranded lineage that should be honestly reconciled rather than left
as permanent baseline exceptions:

- **ISSUE-002, ISSUE-003, ISSUE-005, ISSUE-006, ISSUE-008** (closed) — roughly
  30 mandated `TestCodeCheck_*` tests, deleted when the `backstop code check`
  command and the `pkg/check` engine were eradicated as part of ISSUE-018 /
  the packs-only cutover.
- **ISSUE-018** (closed) — `TestCutover_GateNeverWiresStepCodeCheck` and
  `TestCutover_GateNeverCallsCheckRun`, its own cutover-assertion tests,
  later removed once the thing they asserted the absence of was fully gone.
- **ISSUE-036** (closed) — the contracts-pack compiler signature rule-id
  "tests" (`type-signature-go`, `const-signature-go`, `var-signature-go`,
  `method-signature-go`, `interface-signature-go`) from the func-only-
  signatures work.
- **SPEC-041** (implemented) —
  `TestExemption_BindingDeclaresExemptFromScopeFilterDecoupledFromScopeKind`
  and `TestExemption_ScopeKindDecoupledFromExemptDecision`, 2 mandated tests
  absent.

Each of these artifacts was delivered legitimately, closed honestly, and only
later stranded when a *different*, later piece of work eradicated the thing
it verified. That is the recurring deletion-strands-lineage pattern (see
`project_deletion_strands_spec_lineage` in agent memory) recurring at scale.
Reconciliation means going through the 39 artifact by artifact, via the
appropriate author agents, and either restoring/re-pointing the mandated
test, repointing the claim to a surviving test that still proves the intent,
or moving the artifact to whatever terminal state Problem 2 below resolves on.
This issue does not prescribe which resolution fits which artifact — that is
planning work, and per-artifact evidence needs to be read case by case.

### Problem 2 — terminal-vocab gap: "delivered, then obsoleted"

Every artifact in Problem 1 was DELIVERED — closed honestly, its tests green
at close time — and only later had that delivery removed wholesale by
superseding work, with no 1:1 successor artifact. The current terminal
vocabulary (ISSUE-031; `project_artifact_terminal_states` in agent memory)
is: `closed` (success), `replaced` (superseded by a NAMED 1:1 successor —
requires `replaced-by`), `canceled` (work abandoned, never delivered), and
`deprecated` (specs only). None of these cleanly fits the Problem 1 shape:

- `replaced` is wrong — there is no single successor artifact; the
  `pkg/check` engine and `backstop code check` command were *removed*, not
  replaced by a named equivalent.
- `canceled` is wrong — the work did ship and was verified at the time.
- `closed` is now actively misleading — the gate's new drift dimension reads
  it as a broken promise, since the tests it names no longer exist.

**Open question for the eventual design** (not decided here): does this
warrant a new terminal state — e.g. `obsoleted` / `superseded-by-removal` —
distinct from `replaced`, or a lighter convention layered on top of `closed`
(e.g. a `mandated-tests-removed-by: <artifact-id>` pointer that the drift
dimension treats as a legitimate, self-documenting exclusion rather than a
baseline grandfather)? Either answer has to reconcile with ISSUE-031's
existing terminal-state design and with how the ISSUE-042 drift dimension
currently classifies success-terminal absent-test artifacts. This is framed
as an open question for the planner, not a decision made by this issue.

### Problem 3 — drift dimension does not check completed-plan task tests

ISSUE-042's resolver checks mandated-test existence for issues and specs (via
`claims[].tests`) but not for plans — plans carry `status` only in the
resolver today, so a completed plan (`status: completed`) whose task-level
`test_names` were deleted is not caught by the drift dimension at all.
ISSUE-042's own issue text named "a plan's tasks carry their own mandated
test names" as in-scope, and it was deliberately deferred at ship time to
keep that issue's landed scope tight — plan task-tests substantially overlap
the backing spec/issue's mandated tests, which the shipped dimension *does*
check, so the deferral was documented, not silent.

Scope of this facet: extend the drift resolver to also resolve
`tasks[].test_names` on plans and apply the same success-terminal (plan
`completed`) absent-test check already applied to issues and specs. Doing
this will surface additional stranded lineage beyond the 39 already found in
Problem 1 — for example, the standards-compiler plan (PLAN-SPEC-001) has on
the order of 54 absent mandated tests — which will need the same
grandfathering/reconciliation treatment as Problem 1, not a separate process.

### Problem 4 — SPEC-002's final-phase category-coverage promise is about to be stranded by ISSUE-033

ISSUE-033 is deleting `pkg/validate/plan.go`'s `fileCategory()` and the
`plan/final-phase-missing-category` check outright — the founder rejected the
underlying approach of gating verification adequacy by categorizing touched
files (`.go` → "code", `.spec.md`/`.plan.yml`/etc. → "artifact") by file
extension, not just its Go-specific literal. The sound task-type-based
cadence checks — `plan/gate-cadence-missing` (every phase with
implementation/refactor work needs a verification task) and
`plan/final-phase-no-verification` (the final phase must contain at least one
verification task) — are unaffected and remain exactly as they are.

That deletion strands SPEC-002 (`specs/SPEC-002-plan-schema-evolution.spec.md`)
the moment it lands, the same pattern as Problem 1 but caught *before* the
deletion merges rather than swept up after: SPEC-002's REQ-004 currently
conflates two behaviors in one requirement — "final phase must contain
verification tasks" (sound, stays — backed by
`plan/final-phase-no-verification`, proven by CLM-010/CLM-011 via
`TestPlan_FinalPhase_HasVerification`/`TestPlan_FinalPhase_NoVerification`)
and "...covering every category of work the plan performs" (the categorization
approach ISSUE-033 removes — backed by `plan/final-phase-missing-category`,
proven by CLM-026/CLM-027 via
`TestPlan_FinalPhase_ComprehensiveVerification`/
`TestPlan_FinalPhase_IncompleteVerification`, in
`pkg/validate/plan_final_test.go`). TASK-011 in
`plans/PLAN-SPEC-002-plan-schema-evolution.plan.yml` mandates all four of
those tests together for the same reason. Once ISSUE-033 deletes the
category-coverage check and its tests, CLM-026/CLM-027 and the
category-coverage clause of REQ-004 name tests that no longer exist —
orphaned the moment ISSUE-033 merges, not later.

**Scope of this facet — an explicit retirement, not a repoint.** Unlike
Problem 1 (where reconciliation per-artifact is left open for the planner to
decide restore/repoint/terminal-state), this facet's resolution is already
decided: the category-coverage behavior is being intentionally *removed*, not
moved to a successor, so there is no surviving test to repoint to. Route this
through the spec-author agent (per CLAUDE.md's artifact-workflow rule — never
hand-edit) to reconcile SPEC-002 by:

- Narrowing REQ-004's text to drop the "covering every category of work the
  plan performs" clause and everything describing category derivation from
  file extensions, leaving only the "final phase must contain at least one
  verification task" behavior (still backed by
  `plan/final-phase-no-verification`).
- Retiring CLM-026 and CLM-027 (their mandated tests,
  `TestPlan_FinalPhase_ComprehensiveVerification` and
  `TestPlan_FinalPhase_IncompleteVerification`, are deleted by ISSUE-033) —
  removed from SPEC-002's claims, not repointed to a substitute test.
- Leaving CLM-010 and CLM-011 untouched — they back the surviving
  "final phase must contain verification" behavior and their tests
  (`TestPlan_FinalPhase_HasVerification`, `TestPlan_FinalPhase_NoVerification`)
  are not touched by ISSUE-033.
- Leaving every other SPEC-002 requirement/claim alone, in particular REQ-003 /
  CLM-008 / CLM-009 / CLM-030 (`plan/gate-cadence-missing`) — the task-type-
  based cadence checks are sound and explicitly out of scope for both
  ISSUE-033 and this reconciliation.
- Updating SPEC-002's `## Requirements` prose section and its "Comprehensive
  verification is gameable" Sharp Edge (both currently describe the
  category-derivation mechanism being removed) to match the narrowed REQ-004,
  and reconciling `plans/PLAN-SPEC-002-plan-schema-evolution.plan.yml`
  TASK-011's claims list / description prose (which currently mandates
  CLM-026/CLM-027 alongside CLM-010/CLM-011) to match.

This facet should land alongside or immediately after ISSUE-033's merge, not
wait for the broader Problem 1 sweep — it is a known, decided retirement with
a named cause, not an artifact-by-artifact judgment call.

### Why these four belong in one issue

All four are the same lifecycle-hardening theme (DIR-016): keeping artifact
lineage (mandated tests, claims, requirements) honest as the code they verify
is deliberately removed. Problem 1 is the concrete backlog of stranded
artifacts surfaced by ISSUE-042 (`0ce7f3d`), Problem 2 is the vocabulary gap
that blocks reconciling them honestly, Problem 3 is the coverage gap that
means the backlog in Problem 1 is known to be incomplete, and Problem 4 is a
new instance of the same stranding pattern — caught proactively, before the
causing deletion (ISSUE-033) lands, rather than swept up after the fact by a
drift-dimension sweep. They could be resolved as one reconciliation effort or
split into separate plans — that sequencing decision is left to the planner,
except Problem 4's own note that it should not wait on the broader Problem 1
sweep.

## References

- ISSUE-042 (`0ce7f3d`) — the `artifact_status_drift` gate dimension whose
  full-sweep run surfaced Problems 1-3 here; its issue text explicitly
  deferred the plan-task-tests coverage (Problem 3) and did not address
  reconciling drifted artifacts found once the dimension shipped (Problems 1
  and 2)
- ISSUE-031 — prior art for the terminal-state vocabulary Problem 2 needs to
  extend or work around
- ISSUE-018 / the packs-only cutover — the eradication that stranded the bulk
  of Problem 1 (the `backstop code check` / `pkg/check` removal)
- ISSUE-002, ISSUE-003, ISSUE-005, ISSUE-006, ISSUE-008, ISSUE-036 — the
  closed issues whose mandated tests are now absent (Problem 1)
- SPEC-041 — the implemented spec whose exemption tests are now absent
  (Problem 1)
- ISSUE-033 (`De-Go plan validation file classification`) — deletes
  `pkg/validate/plan.go`'s `fileCategory()` and the
  `plan/final-phase-missing-category` check, the deletion that creates the
  Problem 4 stranding
- SPEC-002 (`specs/SPEC-002-plan-schema-evolution.spec.md`) — the spec Problem
  4 reconciles: REQ-004, CLM-026, CLM-027 (final-phase category-coverage) are
  retired; REQ-003 (`plan/gate-cadence-missing`) and REQ-004's surviving
  "final phase must contain verification" clause / CLM-010 / CLM-011
  (`plan/final-phase-no-verification`) are unaffected and remain
- `plans/PLAN-SPEC-002-plan-schema-evolution.plan.yml` — TASK-011, whose
  claims list and description prose mandate the now-retired
  `TestPlan_FinalPhase_ComprehensiveVerification` /
  `TestPlan_FinalPhase_IncompleteVerification` alongside the surviving
  `TestPlan_FinalPhase_HasVerification` / `TestPlan_FinalPhase_NoVerification`
  (Problem 4)
- `pkg/validate/plan_final_test.go` — houses both the retired
  category-coverage tests and the surviving final-phase-verification tests
  (Problem 4)
- DIR-016 — parent directive (issue/plan lifecycle hardening)
