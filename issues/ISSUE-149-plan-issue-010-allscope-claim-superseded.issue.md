---
title: "Plan Issue 010 Allscope Claim Superseded"
schema_version: issue/v1

issue:
  id: ISSUE-149
  title: "Plan Issue 010 Allscope Claim Superseded"
  type: technical-debt
  status: open
  created: "2026-08-16"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# Plan Issue 010 Allscope Claim Superseded

## Problem

`plans/PLAN-ISSUE-010-pack-engines-diff-scope.plan.yml` (status: `completed`) carries claim prose
that PLAN-ISSUE-091's fix (`ISSUE-091` — "gate --all under-reports test findings", just landed in
the working tree) has now superseded. This issue is the record of that supersession — filed per
PLAN-ISSUE-091 TASK-006 item 1 / CLM-009 ("consequences are filed, not absorbed"), rather than by
editing PLAN-ISSUE-010. **Completed plans are never rewritten**; a correction to a completed plan's
claim prose belongs in a new issue that cites both artifacts, not in an edit to the plan itself.

### The stale text, quoted verbatim

PLAN-ISSUE-010's claim ledger (`CLM-004` definition):

> `gate --all` (GateScopeModeAll) / nil scope restores the whole-repo scan (projectRoot) — the
> explicit escape hatch.

And its TASK-001 mandated-test description for the test that pins that claim:

> `TestPackEngines_AllScope_RestoresWholeRepoScan` (CLM-004): with scope == nil AND separately
> with scope.Mode == GateScopeModeAll, assert the engine's scan target is projectRoot exactly as
> today (the explicit --all whole-repo escape hatch is preserved).

Both sentences assert that a `GateScopeModeAll` scope causes the engine to be handed the bare
`projectRoot` directory target — "exactly as today," "the explicit --all whole-repo escape hatch."
That is no longer true.

## What survives and what is retracted

- **NIL-SCOPE HALF — SURVIVES UNCHANGED.** A `nil` scope still hands the engine the `projectRoot`
  directory target. This is deliberate (PLAN-ISSUE-091 CLM-005, "the nil-scope branch is preserved,
  and is not a grandfather clause") and is the honest behavior for a caller that supplied no scope
  at all.
- **ALL-SCOPE HALF — RETRACTED.** Since PLAN-ISSUE-091, a `GateScopeModeAll` scope hands the engine
  its own explicit, testdata-pruned file list instead of the `projectRoot` directory target
  (PLAN-ISSUE-091 CLM-001, CLM-003, CLM-004). The "whole-repo escape hatch … preserved" framing no
  longer describes what `--all` does.
- **THE TEST NAME — SURVIVES, ACCURATELY.** PLAN-ISSUE-091 TASK-003 deliberately preserved
  `TestPackEngines_AllScope_RestoresWholeRepoScan` by name and rewrote only its `all scope` subtest
  body plus its doc comment. The name stays accurate because the test's surviving `nil scope`
  subtest genuinely does restore the whole-repo scan — the name was never exclusively about the
  all-scope arm, and that arm is still covered by the same test file under its new (correct)
  assertions.

## Enforcement scope (stated explicitly so this is not re-derived)

**This is a claim-prose retraction only. No gate-enforced mandated test is at stake and no promise
breaks.**

- `plans/PLAN-ISSUE-010-pack-engines-diff-scope.plan.yml` declares **no** `test_names:` field
  anywhere. It names its tests only in task-description prose. Verified:
  `grep -c test_names plans/PLAN-ISSUE-010-pack-engines-diff-scope.plan.yml` returns `0`.
- `pkg/gate/artifact_status.go` builds a plan's `MandatedTests` **solely** from
  `phases[].tasks[].test_names` (see `planTaskMandatedTests`), and its own doc comment on
  `planFrontmatter` states: "yaml ignores tasks that carry no test_names, so plans authored before
  this field carry no MandatedTests (unchanged)."
- Therefore PLAN-ISSUE-010 contributes **zero** gate-enforced mandated tests, and this retraction
  would **not** red `status_drift` or any other gate dimension. There is nothing to fix in the gate
  path — only the historical claim prose is out of date, and it stays out of date because the plan
  it lives in is a completed, immutable record.

As context (not this issue's own work): PLAN-ISSUE-091 TASK-003 declares
`TestPackEngines_AllScope_RestoresWholeRepoScan` in its **own** `test_names:` block, so the name
becomes gate-enforced for the first time by PLAN-ISSUE-091's doing, not PLAN-ISSUE-010's.

## References

- ISSUE-091 — "gate --all under-reports test findings" (`issues/ISSUE-091-*.issue.md`), the source
  issue whose fix superseded this prose.
- PLAN-ISSUE-091 — `plans/PLAN-ISSUE-091-gate-all-underreports-test-findings.plan.yml`, TASK-006
  item 1 (files this issue) and CLM-009 (the "consequences are filed, not absorbed" claim this
  issue satisfies).
- PLAN-ISSUE-010 — `plans/PLAN-ISSUE-010-pack-engines-diff-scope.plan.yml` (status: `completed`),
  the completed plan carrying the superseded CLM-004 prose quoted above.
- DIR-032 — Gate Verdict Honesty, this issue's home directive; slot alongside the rest of the
  ISSUE-091 follow-on cluster.
