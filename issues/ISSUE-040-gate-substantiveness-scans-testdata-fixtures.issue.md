---
title: "Gate Substantiveness Scans Testdata Fixtures"
schema_version: issue/v1

issue:
  id: ISSUE-040
  title: "Gate Substantiveness Scans Testdata Fixtures"
  type: bug
  status: open
  created: "2026-07-06"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# ISSUE-040: Gate Substantiveness Scans Testdata Fixtures

## Problem

The dogfood gate's rule-fed findings scan (the `test_substantiveness` dimension,
and by extension any semgrep/ast-grep-backed findings engine) walks `testdata/`
directories as if they were ordinary source, so it flags INTENTIONAL test
fixtures — files that deliberately contain violations in order to test the rule
itself — as real findings.

### Repro (observed 2026-07-06)

`pkg/gate/testdata/substantiveness-pack/fixtures/go/testmain_fixture_test.go`
contains `TestGenuinelyHollowStub`, a deliberately-hollow negative fixture added
by ISSUE-035 specifically to PROVE the substantiveness rule still catches
hollow tests (a regression guard for the `TestMain`-exemption fix). Running the
dogfood `test_substantiveness` gate step against a change that touches this
file reports it as a live violation:

```
[test_substantiveness] test function TestGenuinelyHollowStub has no assertions (hollow)
```

This is not a real finding — the test is data proving the rule works, not code
debt to remediate. The pre-existing `TestHollowExample` fixtures
(`pkg/gate/testdata/hollow-test.go`,
`pkg/gate/testdata/substantiveness-pack/fixtures/go/hollow_test.go`) are the
same pattern: intentionally-hollow fixtures that exist to be scanned BY TESTS
of the substantiveness rule, not by the gate's own findings engine.

### Why it matters

Every future change that happens to touch a file under any `testdata/`
directory in the same diff will pick up these fixture-derived false positives
alongside any real findings, degrading trust in the `test_substantiveness`
dimension exactly the way ISSUE-034 (deleted files) and ISSUE-035 (`TestMain` /
absence tests) degrade trust in coverage and substantiveness respectively — the
gate going loud on something that isn't a defect (CLAUDE.md, "Loud ≠
blocking"). It is currently a benign, non-blocking residual (surfaced but not
gating anything at close of the ISSUE-035 session), but it will start blocking
real work the moment a `testdata/` fixture change lands in the same diff as
other changes under gate.

## Root cause

The findings engine diff-scopes to the gate's changed files (ISSUE-010) but
applies no `testdata/` exclusion when building the engine's target list. In
`cmd/backstop/pack_gate.go`, `runFindingsEngine`'s rule-fed branch builds the
scan target directly from the raw diff scope:

```go
// cmd/backstop/pack_gate.go:604-608
if scope == nil || scope.Mode == gate.GateScopeModeAll {
    cmdArgs = append(cmdArgs, projectRoot)
} else {
    cmdArgs = append(cmdArgs, scope.Files...)
}
```

`scope.Files` is the raw diff-scope file list (`pkg/gate/scope.go`) with no
path filtering beyond diff membership — any changed path, including one under
a `testdata/` directory, is appended verbatim to the semgrep/ast-grep target
list. Standard Go tooling convention treats any directory named `testdata`
(and everything under it) as inert data, never compiled or vetted as source
(`go help packages`); the gate's source-scanning dimensions do not yet honor
that convention.

## Solution (fix direction to evaluate — not committed)

Exclude paths with a `testdata` path segment from the findings-engine scan
target set before it is appended to `cmdArgs`, and confirm whether the same
exclusion is needed on the coverage dimension (`coveragePathsInScope`,
`pkg/gate/step_coverage.go`) or any other step that walks `scope.Files`
directly looking for scannable/measurable source.

**Guardrail — must not regress ISSUE-010's diff-scope contract.** The rule-fed
branch in `runFindingsEngine` is guarded by CLM-001/CLM-002/CLM-003/CLM-007 (the
ISSUE-010 diff-scope contract, see the comment immediately above the excerpt
above): never scan out-of-scope files, never silently fall back to
project-wide/whole-repo, and an empty resulting target list must scan nothing
(not fall back to `projectRoot`). A `testdata` filter that empties the target
list (e.g. a diff touching ONLY `testdata/...` paths) must still yield an empty
`cmdArgs` append and scan nothing — it must not trip the `scope == nil ||
scope.Mode == gate.GateScopeModeAll` whole-repo branch.

**Tests to add:**
- A changed file under a `testdata/` directory (e.g. a fixture matching the
  `TestGenuinelyHollowStub` shape) yields no findings from the rule-fed engine.
- A changed real source file (non-`testdata`) in the same diff still yields its
  expected findings (regression guard — the filter narrows, it does not widen
  or disable scanning).
- A diff containing ONLY `testdata/...` changes produces an empty target list
  that scans nothing, not a project-wide fallback scan (CLM-003 regression
  guard).

**Also confirm at plan time:** whether the `backstop/self` and `go-standards`
dogfood packs already avoid `testdata/` via their own path filters (if so, this
fix is scoped to the findings-engine target-building step in
`cmd/backstop/pack_gate.go` only) or whether they share the same gap.

## References

- `cmd/backstop/pack_gate.go:595-608` — `runFindingsEngine`'s rule-fed
  diff-scope branch; builds `cmdArgs` from `scope.Files` with no `testdata`
  exclusion, guarded by CLM-001/CLM-002/CLM-003/CLM-007 (ISSUE-010)
- `pkg/gate/scope.go` — `GateScope.Files`, the raw diff-scope file list with no
  path-category filtering
- `pkg/gate/testdata/substantiveness-pack/fixtures/go/testmain_fixture_test.go`
  — `TestGenuinelyHollowStub`, the concrete fixture that surfaced this defect
- `pkg/gate/testdata/hollow-test.go`,
  `pkg/gate/testdata/substantiveness-pack/fixtures/go/hollow_test.go` — the
  pre-existing `TestHollowExample` fixtures, same pattern
- ISSUE-035 (gate-substantiveness-flags-testmain-absence-tests) — added the
  `TestGenuinelyHollowStub` fixture that surfaced this issue; sibling false-
  positive in the same `test_substantiveness` dimension
- ISSUE-034 (gate-coverage-flags-deleted-files) — sibling in the same family:
  the gate scanning/measuring the wrong file set (deleted files there,
  `testdata/` fixtures here); this issue's fix should mirror ISSUE-034's
  scope-narrowing guard pattern
- ISSUE-010 (pack-engines-not-diff-scoped) — source of the CLM-001/002/003/007
  diff-scope contract this fix must preserve
- CLAUDE.md — "Loud ≠ blocking" enforcement philosophy; block real defects,
  don't manufacture false positives against intentional test fixtures

