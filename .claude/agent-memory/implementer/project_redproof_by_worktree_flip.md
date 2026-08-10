---
name: redproof-by-worktree-flip
description: Sequence-red (write test, run, then fix) is weak evidence a reviewer will re-run themselves — prove it by control-vs-treatment in a detached worktree at HEAD: copy in ONLY tests+fixtures, observe red, then flip ONE production file and watch it go green
metadata:
  type: project
---

PLAN-ISSUE-104 (2026-07-29). TDD gives you a red-then-green SEQUENCE, but the
sequence is not a controlled comparison: nothing in the record rules out the
test having been red for an unrelated reason, or the fix having dragged other
edits along. The lead was about to re-run exactly this check independently,
which is the tell that sequence-red under-delivers as evidence.

The cheap airtight form, ~3 minutes:

    git worktree add --detach $SCRATCH/wt HEAD
    # TREATMENT = tests + fixtures ONLY. Control = HEAD's production files.
    cp <new test files> <new fixture dir> into the worktree
    git -C $SCRATCH/wt status --porcelain   # MUST show only tests/fixtures
    go test ...                             # expect the exact red, cite line+message
    cp <the ONE fixed production file> in
    go test ...                             # green
    git -C $SCRATCH/wt diff --stat <that file>   # the sole delta, in numbers
    git worktree remove --force $SCRATCH/wt

**Why:** it converts "I saw it fail earlier" into "the ONLY difference between
red and green is this file, and here is the line count." It also proves the
fixture actually exercises the defect path — a falsifier that was never red
against pristine code is vacuous, and this is the check that catches that.

**How to apply:** run it for any lane whose central claim is "this used to
resolve wrong", before reporting the boundary. Never `git stash` to get the
control — siblings have live uncommitted work
([[feedback_never_stash_shared_tree]]). The `git status` assertion inside the
worktree is the load-bearing step: without it you have not shown the treatment
was tests-only. Report the red as file:line plus the assertion message, not as
"tests failed". Related: [[feedback_choose_compile_red_or_behavioral_red]],
[[project_scratch_module_probes_production]].
