---
title: "Gate Test Verification Vacuous Green Missing Mandated Test"
schema_version: issue/v1

issue:
  id: ISSUE-075
  title: "Gate Test Verification Vacuous Green Missing Mandated Test"
  type: bug
  status: open
  created: "2026-07-25"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# Gate Test Verification Vacuous Green Missing Mandated Test

## Problem

`TestSmoke_GateFailsMissingMandatedTest` (`tests/smoke/smoke_test.go:486`) asserts that
`backstop gate --json` exits 1 when a spec's mandated test (`TestCompute_ReturnsNil`) is absent
from the codebase — the scenario's whole point is to prove `test_verification` catches a broken
test promise. It currently does not: the gate returns `pass=true` / exit 0, and
`test_verification` reports zero violations. The scenario is a false green in the test suite
itself — it exercises the failure path but observes success, and nothing currently fails CI to
flag it (see "Why it hid for 12 days" below).

### Root cause

The smoke fixture's `createSpec` helper (`tests/smoke/smoke_test.go:136-140`) hardcodes
`status: draft` into the generated `SPEC-999` frontmatter, with no way for a caller to override
it — `specOpts` (used at `smoke_test.go:518-523`) has no status field.

ISSUE-054 (closed 2026-07-13, commit `2164994`) scoped `test_verification` to
implemented-only specs by design: `StepTestVerificationScopedFunc`
(`pkg/gate/step_testverify.go:336`) calls `filterDueMandatedTests` at line 353, which drops
mandated tests belonging to specs that are not `status: implemented`. Because `SPEC-999` is
forced to `draft`, its only mandated test (`TestCompute_ReturnsNil`) is filtered out before the
capability guard or discovery walk ever run, and the step hits the `len(mandated) == 0` early
return (`step_testverify.go:355-361`), which is an unconditional clean pass. The gate is behaving
exactly as ISSUE-054 designed it to — a draft spec's mandated tests are not yet a live promise.
The fixture is what's out of date: it was written before ISSUE-054 existed, describes a promise
that the current design deliberately doesn't enforce at `draft`, and nobody updated it to force
the spec to `implemented` so it would still exercise the enforcement path it claims to test.

### Why it hid for 12 days

ISSUE-054's blast-radius sweep updated `pkg/gate/*_test.go` and `cmd/backstop/*_test.go` but
never touched `tests/smoke/smoke_test.go` — `git log` on that file shows commit `2164994` never
appears in its history. Separately, `go test` result caching masked the mismatch even when the
smoke package was in the sweep's nominal scope: the scenario only fails when run with `-count=1`
(or after another change invalidates the cache), so a normal `go test ./...` after ISSUE-054
landed could report a stale pass without ever re-executing the test binary.

### Impact

`test_verification`'s implemented-only scoping (ISSUE-054) has no smoke-level proof that it still
blocks on a genuinely broken promise — the one smoke scenario meant to prove that is currently
unable to fail even when the gate regresses to a full vacuous pass. This is a gap in exactly the
kind of enforcement backstop exists to guarantee.

### Related but out of scope

Scenario 6 (`TestSmoke_CodeCheckCatchesViolation`, `tests/smoke/smoke_test.go:632`) is
`t.Skip`'d and its skip message and commented-out body reference a `backstop code check`
subcommand that does not exist post-cutover (stale pre-packs-only naming — the real path is
`backstop gate`). Noted as an adjacent defect; not fixed by this issue.

## Solution

Add a status override to `specOpts`/`createSpec` (default preserving today's `draft`, so other
scenarios that rely on draft's warn-level "capability absent" behavior are unaffected), set it to
`implemented` for `TestSmoke_GateFailsMissingMandatedTest`'s `SPEC-999`, and re-run the full smoke
suite with `-count=1` to surface any sibling scenarios that were similarly cache-masked by
ISSUE-054 or later changes.

## Verification

verification:
  level: integration
  test_command: go test ./tests/smoke/... -count=1 -race
  coverage_threshold: 80

## References

- `tests/smoke/smoke_test.go:486` — `TestSmoke_GateFailsMissingMandatedTest`
- `tests/smoke/smoke_test.go:136-140` — `createSpec` hardcoding `status: draft`, no override
- `tests/smoke/smoke_test.go:518-523` — the scenario's `createSpec` call, `specOpts` has no
  status field
- `pkg/gate/step_testverify.go:336` — `StepTestVerificationScopedFunc`
- `pkg/gate/step_testverify.go:347-361` — `filterDueMandatedTests` call and the
  `len(mandated) == 0` early clean-pass return
- ISSUE-054 (closed 2026-07-13, commit `2164994`) — introduced implemented-only mandated-test
  scoping; blast-radius sweep did not include `tests/smoke/smoke_test.go`
- `tests/smoke/smoke_test.go:632-650` — Scenario 6, `t.Skip`'d, references stale
  `backstop code check` naming (noted, out of scope)
