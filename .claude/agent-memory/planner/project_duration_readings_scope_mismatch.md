---
name: duration-readings-scope-mismatch
description: Before/after gate step durations are usually taken at DIFFERENT gate scopes, and a step's `reason` field can mean it never ran at all — label each reading's scope and read its reason, or the comparison is fabricated
metadata:
  type: project
---

A performance close-out compares two `DurationMS` readings that were almost never taken
under the same gate scope: the pre-fix number gets measured mid-implementation on a
`backstop gate` (diff-scoped) run, the post-fix number on the mandated `gate --all` sweep.
Absolute per-step deltas do not survive that difference; the RATIO between two steps does.

**Why:** PLAN-ISSUE-172 measured `coverage_threshold` 294530ms -> 2211ms and
`pack_engines` 318359ms -> 405797ms. The first pair is real (the coverage producer ran a
whole-module suite regardless of gate scope — that WAS the defect, and the diff-scoped run
even logged "no in-scope files to measure" while burning 294530ms). The second pair is
mostly the scope change plus new coverage instrumentation, and quoting it as a regression
or a cost would have been wrong. The honest statement was "coverage was 92% of
pack_engines, it is now 0.5%".

**★ THE VACUOUS-FAST CASE HAS A CONCRETE SHAPE, AND IT ALREADY BIT ONCE (PLAN-ISSUE-179,
2026-08-19).** `step_coverage.go` returns `pass` with `Reason: "no in-scope files to measure
for coverage…"` when `coveragePathsInScope()` is empty — BEFORE dispatching the coverage
engine at all. A diff whose only `.go` files are `_test.go` (plus config/artifacts/docs)
hits that branch. The post-fix CI run for ISSUE-179 read `coverage_threshold` 302ms against
a 602470ms pre-fix counterfactual: a ~2000x "collapse" that was really the cost of deciding
there was nothing to score. The mechanism under test was never evaluated. The `reason` field
is the discriminator and it is right there in `gate-report.json` — a duration with a
`reason` is not a measurement of the work the step normally does. `step_coverage.go`'s own
comment tags this confusion class ISSUE-118 ("that misreading is what cost a day"); it was
written to stop the step being read as a VERDICT, and it recurred as the step being read as
a PERFORMANCE number. A valid post-fix reading needs a diff containing at least one
PRODUCTION (non-`_test.go`) source file.

**How to apply:** when recording durations, write the scope beside each number
(diff-scoped / `--all`) and the pack version in effect at that reading. Lead with the
ratio. And check the fast number is not VACUOUSLY fast — a reuse path that reads an empty
profile is faster than a correct one; cite real per-file measurements from the step's own
output. Local numbers are never CI numbers: say "CI confirmation outstanding" plainly, as
PLAN-ISSUE-099 did. See [[project_ci_evidence_run_is_branch_not_main]].
