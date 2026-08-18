---
title: "Contracts Local Install Phase3 Anomaly"
schema_version: issue/v1

issue:
  id: ISSUE-177
  title: "Contracts Local Install Phase3 Anomaly"
  type: bug
  status: open
  created: "2026-08-18"

complexity:
  scope: isolated
  uncertainty: exploratory
  risk: moderate
---

# Contracts Local Install Phase3 Anomaly

## Problem

`pkg/pack/engine/contracts_local_install_test.go: TestInstallContractsLocalPack_InstallsWithSuppliedCommand`
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

- `pkg/pack/engine/contracts_local_install_test.go` —
  `TestInstallContractsLocalPack_InstallsWithSuppliedCommand`, the failing test.
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
