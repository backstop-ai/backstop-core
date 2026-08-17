---
name: site-split-fence-holds
description: Two lanes editing one shared test file survived a mid-lane collision because ownership was split BY SITE, not by file — and the sibling merged the shared docstring rather than overwriting it
metadata:
  type: project
---

When two in-flight plans must both edit one test file, an ownership split **by SITE**
(named assertion block) rather than by file works, and it works even when the sibling
lands in the middle of your lane.

Measured 2026-08-17, PLAN-ISSUE-106 vs PLAN-ISSUE-097 on
`pkg/gate/step_verdict_severity_test.go` (`TestClass3Sites_ViolationsAreErrorSeverityByConstruction`):
106 owned SITE 2 (the substantiveness join), 097 owned SITE 1 (`waiverDiagToViolation`),
SITE 3 was owned by neither. 097's implementer rewrote SITE 1 *between* my TASK-006 edit
and my TASK-007 verification — it replaced the converter-level assertion with a call-site
one through `computeWaiverResult` and deleted the `pkg/waiver` import. Both lanes' edits
coexisted with zero conflict and the test stayed green.

**The non-obvious part: the sibling MERGED the shared header docstring instead of
overwriting it.** Both plans told their implementer to correct the same stale header. 097
rewrote its first sentence and *kept my added paragraph verbatim*, including its
"DO NOT reintroduce a count" instruction. Prescribing "merge rather than overwrite; keep
whatever accurate statement is there" in BOTH plans is what produced that.

**Why:** a file-level fence would have forced one lane to block on the other. A site-level
fence plus a symmetric "whoever lands second re-verifies against what ACTUALLY landed, not
against its own plan text" rule let them run concurrently.

**How to apply:** when your plan hands you a shared-file fence, (1) `git status` the file
and read the other lane's site FRESH immediately before editing — not from your plan's
description of it; (2) re-read it AGAIN after your verification run, because the sibling
may have landed in between; (3) a changed foreign site that matches the other plan's
description is that lane doing its job — leave it exactly as found; (4) only a state
neither plan describes is a stop-and-report. See [[feedback_never_stash_shared_tree]] and
[[project_signature_change_strands_crosslane_caller]].
