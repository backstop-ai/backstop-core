---
title: "Gate Allows Vacuous Done Issue"
schema_version: issue/v1

issue:
  id: ISSUE-071
  title: "Gate Allows Vacuous Done Issue"
  type: enhancement
  status: open
  created: "2026-07-18"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# Gate Allows Vacuous Done Issue

## Problem

`artifact_status_drift` reds a success-terminal artifact only when a DECLARED mandated test is
ABSENT. An issue that declares ZERO mandated tests and is marked `closed` therefore passes the
dimension VACUOUSLY — the gate has nothing to check, so it cannot mechanically verify the issue
was actually done, and a later revert of the fix would not red anything.

Plan-less reactive fixes are the common path here: a small direct bug fix closed via
`resolved-by` (or even a bare closed issue with empty `claims`/`requirements`) carries no
`test_names` at all. In that shape, "closed" is purely an LLM/human assertion — the gate never
backs it with a mechanical check.

### Why it matters

This is a vacuous-green hole in the ARTIFACT LIFECYCLE itself — the same "no silent/vacuous
green" principle backstop enforces on code (see `feedback_zero_baked_checks`,
`feedback_loud_not_blocking`) is not yet dogfooded onto issue closure. Mechanical verifiability
of "done" is the whole point of the gate; an issue that can reach `closed` with zero mechanical
proof undermines the same guarantee the status-drift dimension exists to provide for issues that
DO declare tests.

### Contrast (localizes the gap)

`artifact_status_drift` correctly reds a closed issue when it declares a mandated test that is
then absent from the codebase (broken promise). The gap is the complementary case: a closed issue
that declares NO mandated test at all sails through the same dimension with no signal, because
there is nothing declared to check for absence.

## Direction (to be specified in a plan when prioritized)

A success-terminal ISSUE should carry ≥1 mandated test — on the issue itself OR its backing plan
(the issue → plan linkage already exists in `pkg/validate/artifact_status.go`) — that EXISTS
(`artifact_status_drift`) and PASSES (`pack_engines`). A closed issue with no mechanical proof
anywhere in that chain should warn or block, per the loud-not-blocking philosophy.

Must not force ceremony on genuinely test-free changes (docs/config, pure deletions, etc.) —
consider an explicit "no-test-justification" escape, mirroring the waiver pattern, rather than a
hard universal block.

## Impact

Low urgency, foundational scope. Doesn't red anything today; the risk is silent — a reverted fix
on a test-less closed issue produces no gate signal, and the backlog has no way to distinguish
"closed and mechanically proven" from "closed and merely asserted." Worth fixing before the
issue → plan track scales further, but not blocking any in-flight work.

## Acceptance

- A closed issue (or its backing plan) with zero mandated tests anywhere in the traceability
  chain surfaces a WARN (or configurable block) from `artifact_status_drift`, distinct from the
  existing broken-promise RED.
- A closed issue with an explicit, declared justification (docs/config/no-test-needed) does not
  warn — ceremony isn't forced onto genuinely test-free changes.
- Existing broken-promise behavior (declared mandated test now absent) is unchanged.
- A regression test proves: a closed issue with an empty `claims`/no mandated tests and no
  justification triggers the new signal; one with a valid `resolved-by`/`delivered_by` chain
  that DOES carry a mandated test does not.

## Notes / references

- Surfaced 2026-07-18 while reasoning about `artifact_status_drift`'s broken-promise check
  (`pkg/validate/artifact_status.go`) and the `resolved-by` / `delivered_by` close-relaxation
  paths (see `artifacts/issue/v1/schema.json` `closed-requires-traceability` enforcement note).
- Backlog capture only — deliberately deferred behind the in-flight recipe/init build chain
  (`project_recipe_capability`, `project_init_release_specd_not_built`). Do not schedule ahead of
  that work.
- This issue will receive a PLAN when prioritized; mandated test names are deferred to that plan,
  not declared here.
