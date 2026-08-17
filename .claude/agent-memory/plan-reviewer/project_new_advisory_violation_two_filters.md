---
name: new-advisory-violation-two-filters
description: A plan that adds a NEW warning-severity gate.Violation must specify File/ProjectWide — the scope filter silently drops it, or the baseline ratchet REDs on it; "warning is non-blocking" only covers step verdict
metadata:
  type: project
---

Any plan introducing a NEW `gate.Violation{Severity: "warning"}` from a pack-engine
dispatch has TWO downstream filters to clear, and the plan must state which one it
intends. `blocksVerdict` non-blockingness answers NEITHER.

1. **Scope filter (drops it).** `packValidatorStep` runs
   `activeScope.FilterViolations(violations)` (`cmd/backstop/gate.go`) BEFORE building
   the StepResult. `filterViolations` (`pkg/gate/scope.go`) keeps a violation only if
   `ProjectWide == true` OR (`File != ""` AND `scope.Contains(File)`). A violation with
   an empty `File` and `ProjectWide:false` NEVER reaches the step result → step reports
   "pass", `StepsWarned` does not increment. A mandated test that drives
   `dispatchPackEngines`/`runFindingsEngine` directly sits PRE-filter and stays green
   while production is silent — vacuous.

2. **Baseline ratchet (REDs on it).** If it DOES survive the filter,
   `accumulatedViolations` (`pkg/gate/gate.go`) has NO severity filter and neither does
   `CompareBaseline` (`pkg/gate/baseline.go`). A new advisory is not in
   `.backstop/baseline.json`, so `NewViolations` is non-empty and the baseline step
   return HARDCODES `Status: "fail"` regardless of severity → gate exit 1.
   Confirmed empirically 2026-08-17: `.backstop/baseline.json` held 174 warning-severity
   entries of 308 — warnings genuinely ride the baseline and are grandfathered by
   identity.

**Why:** PLAN-ISSUE-093 wrote a CLM ("the skip is reported, never silent; exit code
unchanged") plus a verification task mandating exit 0. Under shape (1) the CLM is false
in production; under shape (2) the exit-0 acceptance is unreachable. Both were invisible
because the plan reasoned only about `blocksVerdict`.

**How to apply:** on any plan adding a new advisory violation, demand (a) the explicit
`File`/`ProjectWide` values, (b) at least one assertion at the STEP layer
(post-`FilterViolations`), not just at dispatch, and (c) a stated position on the
ratchet — a baseline-refresh task, or a reason the entry is out of scope. Related:
[[project_sarif_suppressions_measurement_layer]], [[project_unvacuum_baseline_ratchet]].
