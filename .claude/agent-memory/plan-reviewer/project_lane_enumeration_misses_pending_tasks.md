---
name: lane-enumeration-misses-pending-tasks
description: A plan's cross-lane contention section enumerates lanes from `git status`/`git diff` (LIVE edits only) and misses in-progress plans whose file scope includes yours but whose touching task is still pending
metadata:
  type: project
---

A plan's "CROSS-LANE COLLISION CHECK" section is almost always derived from
`git status` / `git diff` — i.e. from edits that have ALREADY landed in the shared
tree. That systematically misses a live lane whose *touching task has not started
yet*. Re-derive lanes from `plans/*.yml` `files:` lists filtered to non-terminal
status, not from the working tree.

**Why:** PLAN-ISSUE-093 (round 3, 2026-08-17) enumerated exactly two concurrent
lanes on `cmd/backstop/gate.go` (ISSUE-106, ISSUE-100) and asserted
`cmd/backstop/pack_gate.go` "shows NO live edits". Both statements were TRUE as
working-tree snapshots. But PLAN-ISSUE-097 was in-progress (its phase 1 running,
its phase 3/4 pending) and its `files:` arrays scope BOTH `cmd/backstop/gate.go`
(helpers beside `buildWaiverPolicy`) AND `cmd/backstop/pack_gate.go` (a `@waiver:`
token re-key in `splitCommand`). Nothing in the tree revealed it; only reading the
sibling plan's task file lists did.

**How to apply:** for every production file a plan edits, run
`grep -ln "<file>" plans/*.yml`, then filter to plans whose `status:` is
`draft`/`ready`/`in_progress` AND that have a live implementer. Read the OWNING
TASK's description to get the enclosing FUNCTION, then judge disjointness at
function granularity. A lane enumeration that only names files with live diffs is
incomplete by construction — and it matters most in plans that (correctly) tell the
implementer "PROCEED, DO NOT STALL", because an unnamed lane's later diff is exactly
what triggers the stall the plan was trying to prevent.

**A DIRECTED CORRECTION FIXES ONLY THE SIBLING IT NAMES — RE-RUN THE GREP YOURSELF
EVERY ROUND.** PLAN-ISSUE-144 (round 3, 2026-08-17): round 2 said "add PLAN-ISSUE-142
to the enumeration (eleven plans, not ten)". The planner added exactly 142 and nothing
else, and the review brief asked me to verify *that* addition. Re-running
`grep -rln "packval/executor.go" plans/` returned 13 files — TWELVE other plans — with
PLAN-ISSUE-146 AND PLAN-ISSUE-151 (both same-day untracked `draft`s) never enumerated in
any round, while the notes still asserted "Enumerated EVERY plan referencing X". 151 was
the material one: it writes `pkg/packval/pathscope.go`/`phase2.go` with red-first test
tasks under the SAME `test_command`, yet the plan's own sentence read "THE ONE LIVE
SIBLING LANE IN THIS PACKAGE: PLAN-ISSUE-142". Two tells: (a) the plan's count was one
short of the real grep because a non-matching plan (PLAN-SPEC-048) had been counted as a
set member, so the number LOOKED audited; (b) the omitted sibling already named THIS plan
as disjoint in its own text — fences are usually reciprocal, so grep the siblings for
your plan's own ID and check the reverse edge exists.

Related: [[sibling_lane_exclusivity_fence]], [[uncommitted_lane_rows_move_pinned_counts]],
[[prior_lane_planted_fence]], [[falsification_revert_shared_tree]],
[[completeness_claimed_comment_set]], [[stale_sibling_scaffold_evidence]].
