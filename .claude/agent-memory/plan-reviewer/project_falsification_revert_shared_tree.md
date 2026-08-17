---
name: falsification-revert-shared-tree
description: Falsification tasks that mutate a tracked file and revert with `git checkout -- <file>` destroy a concurrent lane's uncommitted work; demand a confirmed-clean target or an untracked probe
metadata:
  type: project
---

A verification task that proves a new guard has teeth by MUTATING a real tracked file and
then reverting with `git checkout -- <that file>` is a shared-tree destruction hazard, even
when the plan correctly bans `git stash` and repo-wide restores.

**Why:** this repo runs many concurrent implementer lanes in ONE working tree. If the chosen
file already carries another lane's uncommitted edits, `git checkout --` discards them. A
"run `git status` before and after and report both readings" instruction only reveals the
loss after it happened, and the usual claim ("the revert is shown to have restored exactly
the prior state") is unachievable by checkout on a file that was dirty.

**How to apply:** when a plan's falsification step appends to / edits a tracked file under a
shared directory, require EITHER (a) an explicit precondition that `git status --porcelain --
<file>` reports the target CLEAN before mutating, with a refusal path if not, OR (b) —
usually better — an UNTRACKED probe file, since content-scanning guards read the filesystem,
not git. E.g. a `zz_<issue>_probe.go` with just a `package X` clause plus the offending
comment: create, confirm fatal, delete. No `git checkout` in the lane at all, and it makes
the positive and negative falsification passes symmetric.

Related: [[scan-boundary-count-mismatch]] (same review, PLAN-ISSUE-139, 2026-08-16).
