---
name: uncommitted-lane-rows-move-pinned-counts
description: A plan's HEAD-pinned finding counts drift from UNCOMMITTED sibling-lane edits, not just commits — reconcile each extra row to a `git diff -U0` hunk before calling the plan's number wrong
metadata:
  type: project
---

When re-measuring a plan's HEAD-pinned finding counts (Probe-B style), a mismatch is
usually NOT a plan error. Map every extra/missing row to a hunk in `git diff -U0 <file>`
for the files `git status` lists as modified. Only rows you cannot attribute to a
sibling lane's uncommitted work are real discrepancies.

**Why:** verifying PLAN-ISSUE-091 round 9 (2026-08-16) I measured 10 active / 11 active
against the plan's 8 / 8. All three extra rows were uncommitted sibling-lane edits:
`gate_substantiveness_e2e.go:72,185` (unwrapped error return inserted by the
artifact-root lane, visible under BOTH dispatch shapes, net-neutral) and
`pack_gate_gotoolchain_test.go:69` (`producerCommandAlias`, inserted by
PLAN-ISSUE-067's harness work). Subtracting them reproduced the plan's numbers
EXACTLY — RAW/ACTIVE/SUPPRESSED on both shapes. Reporting "the plan's count is wrong"
would have been a false blocker and a tenth review round.

**How to apply:** before flagging a count mismatch, run `git status --short <subtree>`
and `git diff -U0` on each modified file, and check whether each unexplained row's line
sits inside a `+` hunk. Note the ASYMMETRY: an extra row in a NON-test file appears
under both dispatch shapes and cancels out of a before/after net; an extra row in a
`*_test.go` file appears only under explicit-file dispatch and therefore ADDS to the
"gains" set, breaking a NET-ZERO prediction. Also re-check whether the plan's
"complete, verified, do NOT re-derive" enumerations (see
[[verified-enumeration-do-not-rederive]]) have grown since it was written — sibling
closeouts add cross-references within hours.
