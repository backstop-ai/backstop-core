---
name: falsify-via-scratch-module-replace
description: Measure a defect in an EXPORTED symbol by building a scratch module outside the repo with a `replace` directive — real verdicts without writing a test file into a shared tree
metadata:
  type: project
---

To falsify a defect at authoring time when the buggy behavior lives behind an
EXPORTED symbol, build a throwaway module in the scratchpad instead of dropping a
`_test.go` into the package:

    module falsify
    go 1.23
    require github.com/backstop-ai/backstop-core v0.0.0
    replace github.com/backstop-ai/backstop-core => /Users/bmanson/src/projects/backstop-core

Then `go run .` a `main.go` that calls the symbol directly. Multiple defect probes in
one program means one compile and a verdict table you can paste into the plan verbatim.

**Why:** a planner is supposed to measure, not assert, but the two obvious ways to
measure both cost something. Writing a `_test.go` into `pkg/<x>` pollutes a tree other
lanes are live in (and `git stash` there is a known hazard); reasoning from the source
produces a claim a reviewer will (correctly) discount. The scratch module gives real
runtime verdicts with a zero-byte footprint in the repo — `git status` stays clean, so
the cross-lane collision section stays honest.

Used 2026-08-17 authoring PLAN-ISSUE-160: four probes against
`packval.DefaultExecutor.RunEngine` returned the IDENTICAL
`Passed=false ExitCode=0 err=<nil>` for CrashGuard, StrictSarif, Producer and
missing-Producer. That uniformity was the plan's strongest framing — it is the success
condition for a negative phase-3 fixture — and it is not something source-reading would
have produced with the same authority.

**How to apply:** reach for this whenever the defect is reachable through an exported
API (executors, validators, parsers, command constructors). It does NOT work for
unexported behavior — for that, prefer windowing the source and stating the reasoning
as reasoning. Put the module in the session scratchpad, never `/tmp` and never inside
the repo (a stray `go.mod` under the repo root breaks the build for everyone).
Related: [[project_closeout_real_gate_in_worktree]] (the heavier worktree version, for
when you need a real gate reading rather than one symbol's verdict),
[[feedback_verify_issue_premises]].
