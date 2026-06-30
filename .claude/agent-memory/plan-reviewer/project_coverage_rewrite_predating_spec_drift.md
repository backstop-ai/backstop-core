---
name: coverage-rewrite-predating-spec-drift
description: rewriting/deleting baked machinery (step_coverage.go, shared_testrun.go) strands EARLIER specs' mandated tests that assert that exact machinery; plan must re-point/retire those claims
metadata:
  type: project
---

When a plan DELETES or REWRITES baked machinery, grep EVERY spec (not just the
parent + named siblings) for the existing test names living in the touched/deleted
test files. An EARLIER spec's claims often mandate tests that assert the very
machinery being eradicated.

**Why:** SPEC-041 (coverage re-impl) rewrote `pkg/gate/step_coverage.go` (deleting
the `go test` CommandRunner path + `parseCoverageLine`/`coverage:` regex) and deleted
`cmd/backstop/shared_testrun.go`. SPEC-010 (the gate spec, status draft, NOT
superseded) mandates 6 coverage tests that assert exactly that deleted path
(CLM-048/049/050/057/058/059 — mockCommandRunner + "coverage: N% of statements" +
`TestCommand: "go test ./..."`). SPEC-010 REQ-016 literally commits to "run the test
suite with coverage profiling" — which the packs-only direction inverts. The plan had
NO task to re-point or retire those claims, and `test_verification` is diff-scoped, so
the dangling claim->deleted-test mapping stays invisible until SPEC-010 re-enters scope.

**How to apply:** This is the [[align_predating_artifacts]] rule operationalized. For
any eradication/rewrite plan: (1) list the existing test funcs in every file the plan
deletes or reuses-by-filename; (2) `grep -rwl <testname> specs/` for ALL specs; (3)
any surviving-spec claim mapping to a destroyed test is a BLOCKER unless the plan has a
task to update that spec (retire/re-point the claim via the spec author agent). The
fix is a documentation/spec-update task routed to /spec, not a silent break.
