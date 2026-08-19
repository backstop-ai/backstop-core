---
name: duration-readings-scope-mismatch
description: Before/after gate step durations are usually taken at DIFFERENT gate scopes — label each reading's scope or the comparison is dishonest
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

**How to apply:** when recording durations, write the scope beside each number
(diff-scoped / `--all`) and the pack version in effect at that reading. Lead with the
ratio. And check the fast number is not VACUOUSLY fast — a reuse path that reads an empty
profile is faster than a correct one; cite real per-file measurements from the step's own
output. Local numbers are never CI numbers: say "CI confirmation outstanding" plainly, as
PLAN-ISSUE-099 did. See [[project_ci_evidence_run_is_branch_not_main]].
