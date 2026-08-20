---
title: "Contracts Local Install Phase3 Anomaly"
schema_version: issue/v1

issue:
  id: ISSUE-177
  title: "Contracts Local Install Phase3 Anomaly"
  type: bug
  status: closed
  created: "2026-08-18"
  closed: "2026-08-19"

complexity:
  scope: isolated
  uncertainty: exploratory
  risk: moderate

resolved-by: ISSUE-180
---

# Contracts Local Install Phase3 Anomaly

## Resolution

The anomaly's real mechanism was found and fixed by `ISSUE-180` / `PLAN-ISSUE-180`, not by any
`ISSUE-166` residue this issue's own investigation had guessed at (that guess is falsified — see
below). Root cause: `pkg/pack/distribution`'s test binary declared no `TestMain`, so on Linux the
packval sandbox trampoline re-exec'd it, Go's default generated test main reran the whole 43-file
suite in the pack's scratch-copy directory instead of installing the sandbox, and the recursive
run's stdout-only failure was swallowed by `foldHelperStderrIntoError` reading stderr only —
producing the observed "14 validation error(s)" signature, byte-identical whether or not
`ISSUE-166`'s `-H -I` fix was present, because it was never that fix's mechanism. `PLAN-ISSUE-180`
added the missing `TestMain` (mirroring `pkg/packval/main_test.go`'s pattern) and confirmed the
fix on real Linux CI: run `32314302525`, commit `9f4763b`, `pass: true`, `total_violations: 0`,
including this test passing. The immediately-prior main commit (`22c7574`, run `32307615655`)
carries the pre-fix failure verbatim, so the fix is a measured one-commit delta, not an inference.

**This issue's own citation was wrong and is corrected below**: the failing test does not live at
`pkg/pack/engine/contracts_local_install_test.go` — that file does not exist. The only file
declaring `TestInstallContractsLocalPack_InstallsWithSuppliedCommand` is
`pkg/pack/distribution/contracts_local_install_test.go`. A bare `grep -rln
"TestInstallContractsLocalPack_InstallsWithSuppliedCommand" --include="*.go" pkg cmd` now returns
THREE hits (two of `PLAN-ISSUE-180`'s own new files mention the test name in comments) — use the
DECLARING-form grep, `grep -rln "func TestInstallContractsLocalPack_InstallsWithSuppliedCommand"
--include="*.go" pkg cmd`, to confirm there is exactly one real declaration, in
`pkg/pack/distribution`.

Closed via `resolved-by: ISSUE-180` — the underlying issue that ran this investigation to ground
and delivered the fix — not `delivered_by: PLAN-ISSUE-180` (schema-illegal: a plan's `delivered_by`
back-match requires the closing issue to be that plan's own `spec_id`, which is `ISSUE-180`, not
`ISSUE-177`) and not `resolved-by: PLAN-ISSUE-180` (malformed: the accept-grammar
`^(BUNDLE|SPEC|ISSUE|PLAN|DIR)-[0-9]{3}$` does not match `PLAN-ISSUE-180`).

## Problem

`pkg/pack/distribution/contracts_local_install_test.go: TestInstallContractsLocalPack_InstallsWithSuppliedCommand`
(corrected 2026-08-19 — this issue originally cited `pkg/pack/engine/contracts_local_install_test.go`,
which does not exist; see "## Resolution" above)
still fails on real Linux CI after `PLAN-ISSUE-166`'s `-H -I` fix landed (commit `f8b3846`), with
an error byte-identical before and after that fix:

```
pack test for .../packs/contracts failed: pack validation (test) of the validation copy failed
in phase3-fixtures: 14 validation error(s)
```

Confirmed by a direct before/after comparison of two real CI runs' `gate-report.json`: run
`32172705491` (commit `9aa278e`, the plan commit immediately BEFORE the fix's implementation
tasks) and run `32179966270` (commit `f8b3846`, the fix itself). Same file, same message, same
error count (14) in both.

## Why this is a real anomaly, not expected residue

This test IS named in `ISSUE-166`'s own original affected-test list
(`TestInstallContractsLocalPack_InstallsWithSuppliedCommand`, quoted in that issue's "confirmed at
14 validation errors specifically"), and it goes through the exact same `pack add`/`pack test`
`phase3-fixtures` validation path as roughly a dozen structurally similar sibling tests that ALL
went green as a direct result of the `-H -I` fix (`TestE2E_ContractsInstalledLocalPack_RealGate_*`,
`TestE2E_ContractsUninstalled_NoVacuousGreen`,
`TestE2E_ContractsRealAstGrepAndGrep_AndSandboxedConvert`, `TestNoVacuousGreen_*`,
`TestDogfood_BackstopOwnContractSignatureTurnsGreen`, `TestDispatchContractEntry_UnscannedAndCompileError`,
and `TestContractsPack_PatternArgFixturesDispatchAndDiscriminate` among others — all confirmed
green on CI run `32179966270`, per the same before/after comparison). This one test, alone among
its structural siblings, did NOT clear. That asymmetry — same validation path, same underlying
pack, same fix applied, different outcome for one specific test — is the anomaly this issue exists
to investigate; it is not simply "one more thing the fix didn't happen to cover."

## What is NOT yet known

The actual content of the 14 validation errors has not been read. Backstop's own gate output
truncates `phase3-fixtures` failures to the summary line (`14 validation error(s)`) with no
itemized detail, and `.github/workflows/ci.yml` has no separate `go test -v` step for this
package — only the two `backstop gate` invocations (`gate` job and `baseline` job) — so nothing in
the current CI output surfaces what the 14 errors actually are. Reproducing and reading them
likely needs the same technique `ISSUE-166`'s own root cause used: a throwaway diagnostic pushed
to a `debug/*` branch with a live test that prints the itemized `phase3-fixtures` errors (or an
equivalent `-v`/verbose invocation) rather than relying on `backstop gate`'s summarized output.
That has not been attempted for this test as of this filing.

## Impact

Low urgency in isolation (this single test's own scope is unclear until investigated), but it is
exactly the kind of "the fix worked everywhere except this one place, and nobody knows why yet"
gap this repo's own conventions say must be filed rather than left as an unexplained residual line
in a CI report.

## Solution

Not prescribed here — the investigation itself is the first deliverable. Candidate first step: a
throwaway `debug/*` branch with a diagnostic that captures and prints the 14 `phase3-fixtures`
validation errors verbatim (mirroring the technique that established `ISSUE-166`'s own root
cause), to determine whether this test's failure shares `ISSUE-166`'s mechanism, is caused by
something specific to `TestInstallContractsLocalPack_InstallsWithSuppliedCommand`'s own setup
(e.g. a different pack-copy/install path than its siblings use), or is an unrelated pre-existing
defect that happened to also live in `packs/contracts`' `phase3-fixtures` validation.

## References

- `pkg/pack/distribution/contracts_local_install_test.go` —
  `TestInstallContractsLocalPack_InstallsWithSuppliedCommand`, the failing test. **CORRECTED
  2026-08-19**: originally cited as `pkg/pack/engine/contracts_local_install_test.go`, which does
  not exist — see "## Resolution" above. Use the declaring-form grep (`func
  TestInstallContractsLocalPack_InstallsWithSuppliedCommand`) to confirm; a bare name grep now
  returns three hits since two of `PLAN-ISSUE-180`'s new files mention the name in comments.
- `ISSUE-180` (`distribution-testmain-missing-sandbox-guard`) and `PLAN-ISSUE-180` — the real root
  cause and fix for this anomaly: `pkg/pack/distribution`'s missing `TestMain` let the Linux
  packval sandbox trampoline re-exec the test binary and rerun the whole suite recursively,
  producing this exact "14 validation error(s)" signature independent of `ISSUE-166`. Confirmed
  fixed on real Linux CI (run `32314302525`, commit `9f4763b`).
- CI runs `32172705491` (commit `9aa278e`) and `32179966270` (commit `f8b3846`) — both read
  directly via `gate-report.json`; the before/after comparison that established this test's
  failure is byte-identical (hence pre-existing) while its structural siblings all cleared.
- `ISSUE-166` (`contracts-pack-phase3-fixtures-fail-on-linux-ci`) — the original filing that named
  this test among the affected set, and whose fix (`PLAN-ISSUE-166`) resolved every other test in
  that set but not this one.
- The `debug/issue166-contracts-grep-repro` branch (PR #3, closed without merging) — the prior
  precedent for the throwaway-diagnostic-branch technique this investigation likely needs.

### Existence-in-world check

Performed 2026-08-18 before filing: searched `issues/` and `bundles/` for
"TestInstallContractsLocalPack", "phase3-fixtures", and "14 validation error". No open issue or
bundle charter already owns this specific test's residual failure; `ISSUE-166` names the test as
part of its original symptom list but its own fix (via `PLAN-ISSUE-166`) does not resolve it, so
this is filed as a distinct residual rather than folded into that issue's closure.
