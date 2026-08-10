---
name: local-baseline-makes-gate-permissive
description: A gitignored .backstop/baseline.json on the dev machine makes local `backstop gate` MORE PERMISSIVE than CI — same commit, same findings, PASS locally and FAIL on a clean checkout; check for it before reporting "local gate is green"
metadata:
  type: project
---

Measured 2026-07-28 (ISSUE-020, CI run 30384984403 vs the local run at the same commit):

    LOCAL   artifact_status_drift: status=PASS, violations=7, new=0, seeded=0
    RUNNER  artifact_status_drift: FAIL (7 violations)

Cause: `.backstop/baseline.json` exists on the dev machine (196KB, dated 2026-07-14) and
is GITIGNORED, so a CI checkout has none. Locally the ratchet classifies known findings
as not-new and the step passes; on a clean checkout every finding is new and blocks.

**Why:** "local gate is green" and "CI gate is green" are DIFFERENT PREDICATES whenever a
local baseline exists. This is a concrete mechanism behind the repeated
local-green-then-CI-red pattern, and it silently understates every local report on any
baseline-comparable dimension.

THE POSITIVE TECHNIQUE (ISSUE-098): when a local baseline grandfathers the very
findings your change is supposed to clear, do not try to make the GATE prove it —
it would have been green before your fix too. Write a test that drives the
resolution DIRECTLY (resolve -> build the index -> compute the presence union ->
classify) and asserts on the RAW violation list, never shelling out to the gate
binary. That proof is baseline-independent by construction, runs in CI as an
ordinary test, and stays honest on a dev box whose baseline hides the defect.
Expect such a test to pass ON ARRIVAL if the resolution pieces already exist —
it proves composition, not wiring, and the wiring needs its own red.

**How to apply:** before reporting a local gate result as evidence, check whether
`.backstop/baseline.json` exists and how old it is. If it does, say so and treat the
local verdict as a lower bound on what CI will find. The honest comparison is a run in a
detached worktree (no baseline) or the CI run itself. Ties to ISSUE-086 — the stale
packless baseline artifact is the thing shaping local verdicts today. Related:
[[project_green_gate_by_scope_exit]], [[feedback_netnegative_gate_baseline]],
[[project_gate_all_underreports_vs_diff]].
