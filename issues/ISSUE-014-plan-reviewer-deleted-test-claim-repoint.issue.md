---
title: "plan-reviewer misses when a plan deletes a test whose claim still mandates it — leaving a dangling claim→test mapping that fails test_verification later"
schema_version: issue/v1

issue:
  id: ISSUE-014
  title: "plan-reviewer misses when a plan deletes a test whose claim still mandates it — leaving a dangling claim→test mapping that fails test_verification later"
  type: bug
  status: closed
  created: "2026-06-20"
  closed: "2026-07-08"

resolved-by: 4ea9a3d

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# plan-reviewer misses when a plan deletes a test whose claim still mandates it — leaving a dangling claim→test mapping that fails test_verification later

## Problem

### Observed gap

The plan-reviewer's congruence checks do not cover the following case: a plan task DELETES (or renames) a test function that a surviving spec claim still mandates via the `tests:` list. No check fires; the plan passes review; the spec is left with a dangling claim→test mapping. The mapping is invisible until the spec file re-enters diff scope, at which point `test_verification` reports the mandated test as "not found."

### Concrete evidence — PLAN-SPEC-034 / SPEC-034 (SPEC-034 cutover, 2026-06-20)

PLAN-SPEC-034 was a strangler plan that retired the bespoke Go code-check parsers and replaced them with the new native toolchain path. It was structured in two phases of particular relevance:

- **Phase 4** introduced *transitional equivalence tests* — `TestEquivalence_*` and `TestGoToolchainConverter_*` functions that ran both the old and new execution paths and asserted parity. These tests were scaffolded as temporary verification scaffolding to build confidence during the cutover.
- **Phase 5** deleted those same test functions. They depended on the bespoke parsers removed in an earlier step and could not compile after removal.

SPEC-034's `claims` block contained six claims that mandated the now-deleted test names:

| Claim | Mandated test (deleted by Phase 5) |
|---|---|
| CLM-008 | `TestEquivalence_GoVetOutput` |
| CLM-009 | `TestEquivalence_GolangciLintOutput` |
| CLM-017 | `TestEquivalence_FullProject` |
| CLM-030 | `TestGoToolchainConverter_VetToSarif` |
| CLM-031 | `TestGoToolchainConverter_LintToSarif` |
| CLM-032 | `TestGoToolchainConverter_Roundtrip` |

PLAN-SPEC-034 contained no task to repoint or retire these claims. The plan-reviewer (and the spec/plan author loop) did not flag the incongruence. The plan passed review and was executed.

The gap stayed invisible because `test_verification` is diff-scoped (see ISSUE-012): it only re-checks a spec's mandated tests when that spec's file appears in the current diff. SPEC-034 was not touched again after the plan executed — until an unrelated contract-sentinel cleanup edit pulled it into diff scope, at which point the gate reported six mandated tests as "not found."

Resolution required a spec-author pass that repointed all six claims to their surviving standalone successor tests.

### Why strangler/refactor plans are especially prone to this

Strangler and refactor plans routinely follow the create-then-delete pattern for transitional scaffolding: temporary equivalence tests, compatibility shims, or bridge implementations are written early to prove parity, then removed once the new path is stable. Every such plan is a potential source of dangling claim→test mappings unless the plan author explicitly audits claims when writing the deletion tasks.

The plan-reviewer is the right enforcement point because it sees the whole plan as a unit — it can compute the *net effect* of all tasks on the claim→mandated-test mapping and flag incongruences before any task executes.

### Relationship to ISSUE-012

ISSUE-012 documents the structural vacuous-green hole: `test_verification` is diff-scoped, so mandated-test existence is only checked when a spec file is in the diff. This issue is a related but distinct gap: even if `test_verification` ran repo-wide on every gate invocation, it would only catch the problem *after* the plan had already been approved and executed. The plan-reviewer needs a *pre-execution* congruence check — a forward-looking analysis of what the plan's task set will leave behind.

## Proposed direction

Add a plan-reviewer congruence check: for every test function a plan task deletes or renames, verify that no surviving spec claim in the plan's `spec_id` still mandates that test name. If a claim maps to a test that will not exist at plan end, the reviewer must require one of:

1. A task that repoints the claim to a successor test that *will* exist at plan end, or
2. A task that retires the claim (with justification) and updates the spec.

More generally: the plan-reviewer should verify that the plan's net effect on the claim→mandated-test mapping leaves every claim mapped to a test that exists (or will exist) when the plan is complete. This check is the mirror of the existing check that every new claim introduced by a plan must have a corresponding test task.

This is a tooling/process check on the plan-reviewer agent's prompting and/or the automated review gate — it does not require schema changes.

## Impact

- Plans that delete transitional tests without repointing their claims produce specs with dangling claim→test mappings at plan end.
- The gap is invisible in normal gate runs because `test_verification` is diff-scoped (ISSUE-012). It only surfaces when an unrelated edit happens to pull the spec back into diff scope.
- The latent failure can persist across multiple gate runs and bundle milestones before manifesting, at which point attribution to the original plan is non-obvious.
- The SPEC-034 instance required an unplanned spec-author intervention to resolve, interrupting the cutover sequence.

## References

- SPEC-034 (`specs/SPEC-034-native-toolchain-cutover.spec.md`) — claims CLM-008, CLM-009, CLM-017, CLM-030, CLM-031, CLM-032 were the six dangling mappings
- PLAN-SPEC-034 — the strangler plan whose Phase 4/Phase 5 structure created the gap; no task existed to repoint/retire the six claims when Phase 5 deleted the tests
- ISSUE-012 — the diff-scoped `test_verification` blind spot that allowed the gap to stay invisible; this issue is a *pre-execution* counterpart; both must be addressed for full coverage
- `pkg/gate/step_testverify.go` lines 158–164 — the diff-scope guard in `StepTestVerificationScopedFunc`

## Resolution

Added a plan-reviewer prompt check for deleted-test claim drift (dangling claim→test mappings). Prompt-only fix, no code test.
