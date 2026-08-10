---
name: ast-locks-untestable-wiring
description: A pure-function test cannot catch missing WIRING; when the call site lives in a build-tagged file the dev platform never compiles, lock it with go/parser (which reads a file regardless of build tags) and falsify it in a detached worktree at the pre-fix commit
metadata:
  type: project
---

Measured 2026-07-28 (ISSUE-020 TASK-025/026). A correct sandbox diagnostic was
discarded because `platformSandboxedRunStdout` set `Stdin`/`Stdout` and left
`Stderr` nil — os/exec routes a nil Stderr to /dev/null. The plan's five mandated
tests all exercised the extracted pure fold, and NONE of them could catch it: a
perfect fold is worthless if nothing hands it any stderr.

**Why:** the call site is in `//go:build linux` code, so no darwin test can
EXECUTE it, and a lock that needed a Linux runner would rebuild the blind spot
that hid the bug through an entire CI run.

**How to apply:** `parser.ParseFile` reads a source file as text REGARDLESS of its
build constraints, so structural assertions work on code the host cannot compile.
Walk the target FuncDecl for the assignments you care about and assert both
presence and distinctness (aliasing two streams to one buffer passes every
behavioural test while corrupting output). Always include a presence assertion on
the function itself so the test cannot pass vacuously after a rename.

Then FALSIFY it: `git worktree add --detach` at the pre-fix commit, copy the new
test in, run it, and confirm it fails with the right message. Control-vs-treatment
is what makes "this test would have caught it" a measurement instead of a claim.
Related: [[project_absence_tests_via_goast]],
[[feedback_never_stash_shared_tree]], [[project_new_file_coverage_floor]].
