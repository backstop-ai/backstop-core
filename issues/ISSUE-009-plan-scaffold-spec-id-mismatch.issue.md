---
title: "plan scaffold emits issue_id but the schema requires spec_id"
schema_version: issue/v1

issue:
  id: ISSUE-009
  title: "plan scaffold emits issue_id but the schema requires spec_id"
  type: bug
  status: open
  created: "2026-06-12"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# plan scaffold emits issue_id but the schema requires spec_id

## Problem

`backstop artifact new plan` sourced from an issue scaffolds the plan
frontmatter with `issue_id: ISSUE-NNN`, but the plan schema's
`plan/spec-id-required` rule demands `spec_id:` (whose pattern already
accepts the `ISSUE-NNN` form). Every issue-sourced plan fails validation
as scaffolded until the field is hand-renamed.

This bit all four planner agents that authored PLAN-ISSUE-002 through
PLAN-ISSUE-008 — each independently rediscovered the workaround.

## Adjacent papercut (same fix or sibling)

`backstop artifact validate --plan <ID>` scoped validation works, but
agents naturally reach for whole-repo validation and must grep their
artifact's filename out of unrelated pre-existing failures. Lower
priority; the ID-scoped path exists (fixed for issues in commit 6f39ea5)
and just needs habit/documentation.

## Fix

Make the plan scaffold emit `spec_id:` for issue-sourced plans (or teach
the schema/validator to accept `issue_id` as an alias — pick one;
emitting `spec_id` is the smaller change and matches the four existing
issue-sourced plans on disk).

## References

- cmd/backstop artifact scaffolding (artifact new plan)
- plans/PLAN-ISSUE-00{2,4,5,6,8}-*.plan.yml — all hand-corrected to spec_id
- Reported independently by four planner agents, 2026-06-11/12
