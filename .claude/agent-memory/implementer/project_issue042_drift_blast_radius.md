---
name: issue042-drift-blast-radius
description: ISSUE-042 artifact_status_drift measured real-tree firing count (39 block + 10 warn), correcting the plan's ~84 estimate
metadata:
  type: project
---

The ISSUE-042 `artifact_status_drift` dimension, measured on the real tree via
`backstop gate --all` BEFORE baseline refresh, fires **39 block violations across 8
success-terminal artifacts** + **10 advisory WARN** (delivered-but-open), NOT the
plan's estimated ~84.

**Why the estimate was high:** the plan attributed "54 mandated tests" to *completed
PLAN-SPEC-001*, but PLANS carry no mandated tests in the resolver — mandated tests live
on specs/issues (claims[].tests), never on plan artifacts. So the plan double-counted.

**The measured stranded set (grandfathered by the user's `backstop baseline generate`,
CLM-015):**
- closed ISSUE-002 (9), ISSUE-003 (10), ISSUE-005 (4), ISSUE-006 (4), ISSUE-008 (3) —
  the deleted `TestCodeCheck_*` from the code-check eradication (30 total, matches the
  plan's ~30 prediction).
- closed ISSUE-018 (2), ISSUE-036 (5); implemented SPEC-041 (2).

**How to apply:** when the user files the CLM-016 follow-up issue (TASK-011), name the
ACTUAL 8 artifacts / 39 findings, not "~84 / PLAN-SPEC-001". The retirement-vocab gap
(delivered-then-obsoleted fits none of replaced/canceled/deprecated) still holds. WARN
(advisory) is non-policied and never enters the baseline count. See
[[project_deletion_strands_spec_lineage]].
