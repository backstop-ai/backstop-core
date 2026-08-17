---
name: gate-blind-to-nonmandated-test-failures
description: test_verification only sees MANDATED tests, so a failing test no spec mandates is invisible to `gate --all`; a green gate is not "the suite passes"
metadata:
  type: project
---

`backstop gate --all` can go GREEN over a genuinely failing `go test`. The
`test_verification` dimension joins against the **mandated** test names declared by
`implemented` specs — a test that no spec mandates is simply not looked at.

**Why:** measured on PLAN-ISSUE-092 TASK-015 (2026-08-16). `gate --all` reported exactly
5 `mandated_test_failed` violations, all of the substantiveness family (ISSUE-148). Two
OTHER tests — `TestPackNew_ScaffoldPassesCheckAndTest` and `TestPackAuthoringLoop_EndToEnd`
(ISSUE-146) — were failing under `go test` at the same moment and appeared **nowhere** in
the gate output, in any dimension. Had ISSUE-146 been the only breakage in the tree, the
full kill chain would have passed over it.

**How to apply:** never report "gate --all passed" as equivalent to "the test suite
passes" — they are different claims and the gate makes the weaker one. Run `go test ./...`
separately before calling a lane green, and when a lane's blast radius lands in tests, check
whether those tests are mandated by any spec before assuming the gate will catch a
regression in them. Corollary for authoring: a behavior worth protecting needs a MANDATED
test name in its spec, or the gate is not watching it. Pairs with
[[project_gate_all_underreports_vs_diff]] (a different under-reporting axis: `--all` is not
a superset of diff scope) and [[project_verdict_decided_after_the_step]].
