---
name: verdict-decided-after-the-step
description: A gate step's Status is RECOMPUTED by pkg/gate/policy.go after it returns — policy counted violations without reading Severity, so a warning-only step FAILED; when a step's verdict contradicts its own logic, grep the policy layer before re-reading the step
metadata:
  type: project
---

Measured 2026-07-28 (ISSUE-020, CI run 30395875188). `coverage_threshold` reported
FAIL with ONE violation, and that violation was `severity: "warning"` — the
coverage-exclusion notice. `pkg/gate/step_coverage.go` had correctly returned "pass"
(it flips to fail only on `Severity == "error"`). `ApplyPolicy` then overwrote it.

Both policy paths decided by COUNTING: the dimension-default branch
(`len(counted) > 0 -> fail`) and `applyScopedPolicy`'s `blocking` append, which keys
on the POLICY level for the dimension, not the violation's severity. The decisive
check was `grep -c Severity pkg/gate/policy.go` -> **0**. The layer never read the
field.

**Why:** the step is not where the verdict is finally decided. `Gate.Run` applies
policy after the steps run, so a step's own Status is a PROPOSAL. I read
step_coverage, found it correct, and reported the discrepancy as a cosmetic tally
issue — twice — before a falsifier proved it failed the gate.

**How to apply:** when a step's reported status contradicts the logic you just read
IN that step, grep the policy layer before re-reading the step. And when adding any
warning-severity violation, verify end-to-end that it does not block — "loud ≠
blocking" is founder law, and the code did not implement it. Fixed by `blocksVerdict`
(explicit "warning" exempt; UNSET severity still blocks, so an omitting producer
fails closed). Related: [[project_local_baseline_makes_gate_permissive]],
[[feedback_loud_not_blocking]].
