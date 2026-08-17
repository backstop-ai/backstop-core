---
name: undefined-claim-roster
description: Issue-sourced plans reference CLM-NNN in every task but the validator never checks the claims are DEFINED anywhere — check the notes for a roster
metadata:
  type: project
---

For an `issue → plan` artifact, the source issue has no `claims:` array, so a plan's
CLM-NNN ids are self-declared. The convention in backstop-core is an explicit
`CLM-NNN  <assertion text>` roster inside the plan's `notes:` block —
PLAN-ISSUE-100, PLAN-ISSUE-106 and PLAN-ISSUE-114 all carry one.

**Why:** `validate.Plan` enforces that tasks HAVE claims, not that claims MEAN
anything. PLAN-ISSUE-097 reached review round 2 with CLM-001..CLM-012 referenced by
every task and defined nowhere; CLM-001/002/003 (attached to the re-key tasks) had
zero descriptive text in the plan or the issue, making it impossible to check task
completion criteria against claim assertions.

**How to apply:** On any issue-sourced plan, `grep -n "CLM-0"` and confirm a
definition block exists, not just task-array references and inline "(CLM-008)"
mentions. Claims attached to `refactor` tasks are the ones most likely to be
undefined — the task prose describes the edit, never the assertion.

Related: [[project_repurposed_test_claim_text_drift]], [[project_retirement_claim_scope]].
