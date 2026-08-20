---
name: second-issue-closes-via-resolved-by
description: A plan can only deliver against its OWN source issue; a second issue fixed by the same plan closes via `resolved-by: ISSUE-NNN` — and `resolved-by: PLAN-ISSUE-NNN` is malformed
metadata:
  type: project
---

When one plan's fix also resolves a SECOND, non-source issue, that second issue **cannot**
carry `delivered_by: PLAN-ISSUE-NNN`. `validateDeliveredBy` (`pkg/validate/delivered_by.go`,
final step) back-matches the plan's `spec_id` against the closing issue's own id and emits
`issue/delivered-by-spec-mismatch` at severity `error`. One plan delivers against exactly one
issue — its source. The repo's one-plan-one-issue habit is enforced, not just convention.

The legal lighter path is `resolved-by` (ISSUE-048, `pkg/validate/resolved_by.go`): no backing
plan and no mandated test required, but it DOES require a `## Resolution` section
(`pkg/validate/issue.go`) and an `issue.closed` date. `delivered_by` and `resolved-by` are
mutually exclusive — at most one close pointer.

★ THE TRAP: the accept grammar is `^(BUNDLE|SPEC|ISSUE|PLAN|DIR)-[0-9]{3}$`. **`PLAN-ISSUE-180`
does NOT match** (after `PLAN-` it demands three digits and finds `ISSUE-180`), and it is not a
hex SHA or PR URL either, so it falls to `issue/resolved-by-malformed`. The intuitive spelling
is the illegal one. Spell it `resolved-by: ISSUE-180` — the typed ref then existence-checks
against `issues/ISSUE-180-*.issue.md`. A fix-commit SHA (7-40 hex) is also legal.

**Why:** hit while planning PLAN-ISSUE-180, where ISSUE-177 (a different issue with a falsified
mechanism guess) was resolved by ISSUE-180's fix. Verified by reading both validators and
running the regex, not inferred from convention.

**How to apply:** when a plan resolves an adjacent issue, do NOT give that issue a
`delivered_by` pointer and do NOT route it into this plan's scope. Record the ordering as a
recommendation for a routed `/issue` author: source issue closes `delivered_by`, second issue
closes `resolved-by: ISSUE-<source>` — both contingent on the same verification evidence.
Related: [[project_closeout_convention]] is not a substitute — this is a schema fact.
