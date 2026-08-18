---
name: gha-steps-context-null-coercion
description: A GitHub Actions `if: steps.<id>.conclusion != 'skipped'` guard naming a WRONG/absent id evaluates TRUE (step always runs), never a skip — plans state the inverse
metadata:
  type: project
---

A workflow guard of the form `if: always() && steps.<id>.conclusion != 'skipped'`
fails OPEN, not closed, when `<id>` names no declared step.

**Why:** GitHub expression loose equality coerces to number — `null` → `0`,
a non-numeric string like `'skipped'` → `NaN`, and any comparison with `NaN`
is false. So `null == 'skipped'` is false, `!=` is TRUE, and the guarded step
runs on EVERY run. Plans (PLAN-ISSUE-176 round 3) assert the opposite —
"a guard naming an id no step declares silently evaluates to a permanent skip,
so it would disable the falsifier" — which inverts the real failure mode: the
guard goes INERT and whatever misattribution it was added to prevent returns.

The guard's *intended* case does work: a step with no `if:` whose earlier
sibling failed is marked `TaskResult.Skipped` by the runner (it evaluates every
step's condition, then `Complete()`s it, writing `conclusion: skipped` into the
steps context) — which is why `always()`/`failure()` work at all.

**How to apply:** whenever a plan adds an `if:` guard referencing another
step's `id`/`outcome`/`conclusion`, work out BOTH failure directions by hand
(missing id → null → coerced 0 vs NaN) and check the plan's stated rationale
and prescribed failure message match. The id-consistency assertion is still
right; the *why* attached to it usually is not. Related:
[[project_shortcircuit_dependent_guard]].
