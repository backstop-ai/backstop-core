---
title: "Testgate Succeeds Without Standards Lost Assertion"
schema_version: issue/v1

issue:
  id: ISSUE-039
  title: "Testgate Succeeds Without Standards Lost Assertion"
  type: bug
  status: open
  created: "2026-07-06"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Testgate Succeeds Without Standards Lost Assertion

## Problem

`TestGate_SucceedsWithoutStandards` in `cmd/backstop/standards_removal_test.go` is
the mandated test for SPEC-030 claim CLM-015. CLM-015 is a BEHAVIORAL claim:

> A gate / code-check run succeeds (no config error, no missing-standard error) on
> a project with no STD-GO-001 artifact and no compiled standards directory.

The test no longer runs a gate at all. Its body today:

```go
func TestGate_SucceedsWithoutStandards(t *testing.T) {
	dir := t.TempDir()
	// writes backstop.yml + main.go
	// os.MkdirAll(filepath.Join(dir, ".backstop"))
	// os.Stat(filepath.Join(dir, ".backstop")) — asserts only that the dir exists
}
```

It writes a `backstop.yml` and a `main.go`, does `MkdirAll(.backstop)`, then
`os.Stat(.backstop)` and fails only if that stat errors. It asserts NOTHING about a
gate run actually succeeding — no config error is absent because no config was ever
loaded; no missing-standard error is absent because no check ever ran. This is a
genuinely hollow test: it was correctly flagged by the `test_substantiveness` gate
step, and deliberately NOT annotated `kind: absence` during ISSUE-035's work,
because it is behavioral-gone-hollow (a real regression), not a legitimate
structural/absence assertion in disguise.

### How the assertions were lost

Per the test's own header comment, two separate cutovers each removed one half of
the original assertion, in sequence, without either replacing what it removed:

1. The `*realCodeChecker.runCheck` assertion (the actual "does a check run succeed"
   half) was removed by SPEC-040's toolchain-pack cutover.
2. The `check.LoadManifest` routing-tail assertion (the "no missing-standard
   routing error" half) was removed by ISSUE-018's code-check deletion (the
   in-process routing manifest and `realCodeChecker`/`code_check.go` path no longer
   exist — confirmed deleted from the current tree).

Each removal was locally correct (the thing it asserted against was deleted), but
neither cutover restored an equivalent assertion against the new gate path, so the
test degraded to a scaffold with a single directory-existence check.

## Why it matters

CLM-015's guarantee — that the gate succeeds cleanly on a project with no native
STD-GO-001 standard and no compiled standards directory — is currently UNVERIFIED.
The test passes vacuously regardless of whether that guarantee holds. This is
exactly the vacuous-green pattern the substantiveness gate dimension exists to
catch, and it is a real coverage hole in the packs-only migration's own acceptance
story: nothing today proves the gate actually stays green when a project has no
standards.

## Current status: grandfathered, not fixed

`test_substantiveness` currently carries `baseline: true` in this repo's
`backstop.yml` (a baseline refresh done while closing out the session that
surfaced this issue), so the hollow-test finding on
`TestGate_SucceedsWithoutStandards` is non-blocking today. This issue is the
tracked ratchet-down: the baseline entry is a temporary allowance, not a
disposition, and this test should not remain hollow permanently.

## Fix directions (open for the planner)

1. **Restore a real assertion (preferred).** Make the test actually invoke the
   gate against the no-standards fixture project and assert success (no config
   error, no missing-standard error). Post-cutover, the current gate entrypoint is
   `runGate` (`cmd/backstop/gate.go`), invoked via the Cobra command constructed by
   `newGateCommand` — other current tests in this package (e.g.
   `TestRunGate_UnexpectedArgsReturnConfigExit`,
   `TestRunGate_InvalidBaselineTTLReturnsConfigExit` in `cmd/backstop/gate_test.go`)
   already exercise `runGate`/`newGateCommand` in-process against a constructed
   command and flags, and `gate_discovery_e2e_test.go` shows the pattern for
   driving the assembled gate steps (`buildGateSteps`) over a real temp/testdata
   project directory. Either pattern (full `runGate` via the Cobra command, or the
   assembled `buildGateSteps` result) is a viable restoration path — confirm which
   gives the cleanest "no config error, no missing-standard error" assertion
   surface at plan time.
2. **Or re-scope CLM-015**, only if its behavioral guarantee is genuinely verified
   elsewhere in the current suite. This must be verified before being chosen — do
   NOT re-scope the claim just to match the hollow test's current (accidental)
   shape; that would be papering over a lost guarantee rather than restoring or
   retiring it honestly.

## References

- `cmd/backstop/standards_removal_test.go` — `TestGate_SucceedsWithoutStandards`,
  the hollow test itself; also home to the sibling absence tests
  (`TestNoProductionImportOfCompile`, `TestCompiledStandardsArtifactsAbsent`,
  `TestPkgCompileDirectoryAbsent`, `TestStdGo001SourceAbsent`) that ARE correctly
  `kind: absence` and are not in scope here
- `specs/SPEC-030-packs-only-native-standards-removal.spec.md` — REQ-006 / CLM-015,
  the claim this test is mandated against
- `specs/SPEC-040-toolchain-pack-cutover.spec.md` — the cutover that removed the
  `*realCodeChecker.runCheck` half of the original assertion
- `issues/ISSUE-018-remove-vestigial-baked-in-code.issue.md` — the cutover that
  removed the `check.LoadManifest` routing-tail half of the original assertion
- `issues/ISSUE-035-gate-substantiveness-flags-testmain-absence-tests.issue.md` —
  the substantiveness work that surfaced this test in scope and confirmed it is
  correctly flagged (not a false positive) rather than annotating it away
- `cmd/backstop/gate.go` — `runGate`, `newGateCommand`, the current gate entrypoint
- `cmd/backstop/gate_test.go` — existing in-process `runGate`/`newGateCommand`
  test patterns
- `cmd/backstop/gate_discovery_e2e_test.go` — existing pattern for driving the
  assembled gate steps (`buildGateSteps`) over a real project directory
- `backstop.yml` — `test_substantiveness: baseline: true`, the temporary
  grandfathering that makes this finding non-blocking today
