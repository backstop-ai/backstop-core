---
name: matrix-test-subject-provenance
description: A matrix/parameterized test that "evaluates the CONDITION" must say WHERE the condition text comes from; both obvious hardcodings contradict the plan's own red-then-green trajectory
metadata:
  type: project
---

When a plan adds a matrix test that runs "the condition / the expression / the
rule" under N environments (shells, tools, platforms), check whether the plan
states the PROVENANCE of the thing under test. It usually does not.

Run the trajectory yourself, both ways:
- Hardcode the CORRECTED expression in the test → the matrix is GREEN in the
  test task, before the impl task flips anything. It is a permanently-green
  guard that cannot distinguish the fixed artifact from the broken one — the
  same weak-pin defect class these plans usually exist to fix.
- Hardcode the CURRENT/broken expression → red in the test task, but the impl
  task's `files:` scope covers only the artifact, so the matrix stays RED after
  the fix.

Only a test that READS the expression out of the artifact under test (the
fixture script, the pack rule, the generated config) is red-before and
green-after. Demand that the plan name the file, the extraction, and a
hard-fail when the extraction finds nothing.

**Why:** PLAN-ISSUE-179 round 2 (2026-08-19) — `..._ReuseIsShellPrecision
Independent` was specified as "runs the reuse CONDITION (not the whole producer
script) under every shell it can resolve"; sharp edge 10 and the design section
both repeated it and neither said where the condition came from, while TASK-002
demanded it go RED on zsh/ksh and TASK-003 (fixture-script-only scope) demanded
it go GREEN.

**How to apply:** any task adding a matrix/table test over an expression, argv,
regex or command extracted from a shipped artifact. Reconstruct the red→green
trajectory under each plausible implementation before passing.

Related: [[registry-derived-premise-per-test]], [[single-authority-refactor-unfalsifiable]].
