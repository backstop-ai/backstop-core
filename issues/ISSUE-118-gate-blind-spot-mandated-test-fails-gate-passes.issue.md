---
title: "Gate reports full PASS while a mandated test genuinely fails: no gate dimension runs the Go suite to a verdict when only test files change"
schema_version: issue/v1

issue:
  id: ISSUE-118
  title: "Gate reports full PASS while a mandated test genuinely fails: no gate dimension runs the Go suite to a verdict when only test files change"
  type: bug
  status: open
  created: "2026-08-11"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: critical
---

# Gate reports full PASS while a mandated test genuinely fails

## Problem

`backstop gate`'s core promise is that a green run means the mandated tests actually pass. That
promise is false in a reproducible, general case: when a change touches only `_test.go` files (no
production Go changed), **no gate dimension actually executes the Go test suite to completion and
reads its real pass/fail verdict** — a genuinely failing test can sit in the tree indefinitely with
`backstop gate` reporting green, as long as nobody happens to run `go test` by hand.

## Reproduction

Measured on the same tree, with a freshly-rebuilt `bin/backstop` binary, during implementation of
`plans/PLAN-SPEC-067-ci-recipe-pack.plan.yml` (2026-08-11):

```
$ go test ./cmd/backstop/... -race -run TestCIRecipes
--- FAIL: TestCIRecipes_FleetDeclaresPackAtOneVersionInBothFiles
FAIL
exit status 1
```

```
$ ./bin/backstop gate
10 passed, 0 failed
exit 0  (full PASS)
```

Same working tree, same commit, same binary. One command says a mandated test is red; the other
says everything is green.

## Root cause — three gate dimensions, none of which run the suite here

- **`test_verification`** (`pkg/gate/step_testverify.go`) only checks that a mandated test's NAME is
  present in source (`ExtractMandatedTests` + `ResolveMandatedTestPaths`). It never invokes `go
  test` and never reads a pass/fail verdict — a test that exists and is named correctly satisfies
  this step whether it passes, fails, or panics.
- **`test_substantiveness`** only checks that a mandated test's body contains real assertions (not
  a stub/vacuous test). It also never executes the test — a test with a genuine assertion that
  currently evaluates false still satisfies this step.
- **`coverage_threshold`** (`pkg/gate/step_coverage.go`) is the ONE gate dimension that actually
  invokes `go test` — but in this specific, general case it never got the chance to: when the
  scoped change touches only `_test.go` files and no in-scope production file, the step exits early
  with `Status: "pass", Reason: "no in-scope files to measure for coverage"`
  (`pkg/gate/step_coverage.go:98`) — it silently SKIPS running the suite rather than running it to a
  verdict. A change that is 100% test-file diff (exactly the shape of the SPEC-067 fix that failed
  to compile/pass) hits this skip path every time.

Net effect: for any diff whose changed files are entirely `_test.go`, the three gate dimensions
that touch "did the test pass" either don't execute the test at all (`test_verification`,
`test_substantiveness`) or explicitly decline to because there's "nothing to measure"
(`coverage_threshold`). Nothing in the kill-chain runs `go test ./...` to completion and asks
"did it pass" for this shape of change.

## Why this matters

This is not cosmetic — it undermines the gate's central claim across this whole project (see
CLAUDE.md: "Verify, don't assert... running the real command... and reading the result"). A test-
only commit (e.g. a plan's final "make the failing test pass" step, or any test-hardening PR) is
exactly the class of change most likely to introduce a genuinely red test, and it is exactly the
class of change this blind spot cannot see. It is also closely related to, but distinct from,
ISSUE-092 (Pack Test Phase3 Fixtures Cannot Fail) — that issue is about `backstop pack test`'s
fixture-execution phase never running; this one is about `backstop gate`'s own Go-suite-verdict
dimensions never running, for a different (but structurally similar) reason in each of the three
steps named above.

## Direction (not scoped here)

At minimum, the eventual plan should weigh:

1. Whether `coverage_threshold`'s early-skip-on-no-production-files behavior should instead still
   run the affected package's test suite to a verdict (without scoring coverage) when the diff is
   entirely test files.
2. Whether a NEW gate dimension (or a widened `test_verification`) should exist whose job is
   specifically "run every mandated test and read its real exit code," independent of coverage
   scope.
3. A regression fixture proving the fix: a tree with a genuinely failing mandated test and an
   entirely-test-file diff must turn `backstop gate` red. Absent that proof, any fix risks becoming
   another vacuous-green claim, per the fixtures-from-real-output/must-falsify convention.

## Notes / references

- Reported by the implementer during `plans/PLAN-SPEC-067-ci-recipe-pack.plan.yml`
  (2026-08-11), surfaced correctly rather than hand-waved past.
- Sibling to the gate-verdict-honesty cluster (ISSUE-066, ISSUE-067, ISSUE-091, ISSUE-092): a
  validation signal that reads as complete/authoritative and silently isn't. Filed separately per
  the existence-in-world check — this is a `backstop gate` (runtime kill-chain) defect, not a
  `backstop pack test` (authoring-time fixture validator) defect, and no open issue currently names
  it.
