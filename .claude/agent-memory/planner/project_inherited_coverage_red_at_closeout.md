---
name: inherited-coverage-red-at-closeout
description: How to close a plan whose only remaining gate red is an inherited coverage-floor shortfall — prove it with a control-worktree baseline, name the owning issue, never waive
metadata:
  type: project
---

When a lane's final gate leaves ONE red and it is a coverage-floor shortfall on a file
the lane merely touched, close the plan with the red RECORDED, not waived — but only
after proving inheritance with three numbers:

1. post-change coverage (covered/total, %),
2. clean-HEAD coverage for the same file measured in a **separate control worktree**
   (never by cleaning/stashing/checking out over a shared tree carrying other lanes'
   uncommitted work),
3. the counterfactual: what the file would read if every statement this lane added were
   removed. If that is still under the floor, the lane demonstrably cannot fix it.

Then name the issue that OWNS the underlying cause by ID, and state explicitly:
accepted as inherited, not waived, not hacked green — with a grep confirming zero
`@waiver:` tokens in the lane's files.

**Why:** coverage floors are structural and the honest close-out is the one that shows a
second side. PLAN-ISSUE-124 (2026-08-16) hit exactly this: 73.1% after vs 74.4% at clean
HEAD (floor 80), counterfactual 76.4% — the real fix needed 14 failing `cmd/backstop`
tests to pass again, blocked on ISSUE-148 (a pack-content fixture-polarity defect).
Without the control-worktree number the close-out would have been an assertion.

**How to apply:** at any AS-BUILT banner where the residual red is coverage. TASK-007's
like-for-like rule is the framing — a dimension already red at baseline and red the same
way now is UNCHANGED and is not this lane's finding. See [[plan-closeout-convention]].
