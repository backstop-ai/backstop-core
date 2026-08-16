---
name: sweep-axis-definition-drift
description: When a plan mandates named grep "sweeps" for spec drift, re-run each literal command and check the SAME sweep is defined identically in every task that names it
metadata:
  type: project
---

A plan that prescribes named corpus sweeps (PROSE sweep / SIGNATURE sweep / MECHANISM
sweep) as its reconciliation METHOD will drift on the sweep DEFINITIONS across tasks, and
will then claim a new "axis" was needed when an existing sweep already returned the hits.

**Why:** PLAN-ISSUE-122 round 6 (2026-08-16) announced a THIRD "mechanism" sweep axis
(`grep -n 'FindUngatedArtifacts' specs/SPEC-070-*`) as round 6's method contribution.
Running the plan's OWN signature sweep from its notes block —
`grep -rn 'FindUngatedArtifacts\|DiscoverArtifacts\|realArtifactValidator\|Classification' specs/`
— returned every one of those sites already. The plan carried FOUR incompatible
definitions of "the signature sweep": notes (4 alternations), TASK-014 (3 — dropped
`Classification`), TASK-016 (`Classification` alone), TASK-015 (4 again). The real
earlier miss was reading the sweep's OUTPUT, not the sweep's coverage — a distinction
that matters because "add another sweep" doesn't fix an eyeballing failure.

**How to apply:** for any plan whose sweeps are the deliverable — (1) run each command
LITERALLY as written and diff the hit set against the task's enumeration; (2) grep the
plan for every place the sweep is restated and confirm byte-identical alternation lists;
(3) verify each "genuinely clear" hit individually (see
[[verified_enumeration_do_not_rederive]]); (4) run the sweep yourself corpus-wide and
pre-clear anything the plan omits, or the final verification task generates reports it
told the implementer not to dismiss.

Also check the plan's LANE STATUS / "tree is clean" assertions against live `git status`
— a plan that says "free surface" while a parallel lane holds uncommitted edits in a
cited file ships stale line numbers and a false green light.
