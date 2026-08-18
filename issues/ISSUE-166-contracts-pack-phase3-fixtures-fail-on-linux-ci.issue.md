---
title: "Contracts Pack Phase3 Fixtures Fail On Linux Ci"
schema_version: issue/v1

issue:
  id: ISSUE-166
  title: "Contracts Pack Phase3 Fixtures Fail On Linux Ci"
  type: bug
  status: open
  created: "2026-08-18"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# Contracts Pack Phase3 Fixtures Fail On Linux Ci

## Problem

On the CI run at commit `970512b` (`ISSUE-163`'s `TestMain` fix — the fix that let more of the
suite run to completion on GitHub's Linux runner for the first time), `packs/contracts` fails
its own `pack add`/`pack test` phase3-fixtures validation, dogfooding-on-itself, on Linux CI.
This was confirmed directly by downloading and inspecting `gate-report.json` from CI run
`32108003542` (`pack_engines` step). **What follows is the observed shape of the failure, not a
traced root cause** — real investigation is still needed; this issue should not be read as
stating more certainty than that.

### The broad symptom: packs/contracts refuses its own validation copy

Roughly a dozen distinct tests fail identically, all bottoming out in the same `pack add`
refusal (14 validation errors for most of them):

```
pack add /home/runner/work/backstop-core/backstop-core/packs/contracts: pack test for
/home/runner/work/backstop-core/backstop-core/packs/contracts failed: pack validation (test) of
the validation copy failed in phase3-fixtures: 14 validation error(s)
```

Affected test names, confirmed present in the CI report:
`TestE2E_ContractsInstalledLocalPack_RealGate_MissingSignatureRed`,
`TestE2E_ContractsInstalledLocalPack_RealGate_PresentForbiddenSymbolRed`,
`TestE2E_ContractsUninstalled_NoVacuousGreen`,
`TestE2E_ContractsRealAstGrepAndGrep_AndSandboxedConvert`,
`TestE2E_ContractsGrepGatedByAllowlist`, `TestNoVacuousGreen_MissingSignatureBlocks`,
`TestNoVacuousGreen_PresentForbiddenSymbolBlocks`,
`TestDogfood_BackstopOwnContractSignatureTurnsGreen`,
`TestDispatchContractEntry_UnscannedAndCompileError`, and
`TestInstallContractsLocalPack_InstallsWithSuppliedCommand` (confirmed at 14 validation errors
specifically).

### A separate, more specific symptom cluster: grep-based absence probes reporting zero matches

Several other contracts-related tests fail with different, more precise messages, all shaped
like a grep-based "absence" probe (a forbidden-symbol-presence check) not finding a symbol that
should be present:

- `TestContract_AbsencePresentSymbolGrepMatchViolation`: "a present forbidden symbol must
  produce a grep match (absence VIOLATION), got 0"
- `TestContract_AbsenceScopeFileOrPathParameterized`: "file-scoped absence probe must run over a
  single file (CLM-010)"
- `TestContract_AbsenceUsesGrepTextPresenceNotAstGrep`: "grep text-presence must flag the token
  wherever it appears (comment + decl), got 0 matches"
- `TestEngine_GrepConvertScriptEmitsValidSarif`: "SARIF must carry at least one result from the
  real grep match, got: {"
- `TestContractsPack_PatternArgFixturesDispatchAndDiscriminate/the_fixtures_discriminate_through_the_real_engine`:
  "phase3 error: check=semgrep-negative rule=contract-absence claim=contract-absence-go message=
  negative fixture not triggered"
- `TestEquivalence_GoAbsencePresentAndAbsentMatchLegacy`: "pack verdict (false) != analyzer
  verdict (true) — equivalence broken"
- `TestPackVerdict_PresentAndAbsencePolarities`, `TestPackContractResult_AllPolaritiesOverRealEngines`,
  `TestPackContractResult_ScopeFallbackAndMissingFile`: various "absent match" / grep-not-finding-
  expected-symbol shapes.

### The pattern, stated as observation only

Across both clusters, the common shape is: something about how `packs/contracts`'s grep-based
absence probe behaves on Linux CI differs from how it behaves on the darwin machine this pack
was authored and tested on — grep matches that should fire (finding a present forbidden symbol)
report zero matches on Linux CI. **The actual mechanism was not traced tonight.** Candidate
explanations that have NOT been investigated or ruled in: a flag/behavior difference between BSD
grep (darwin) and GNU grep (Linux), a path-resolution difference between platforms, something in
how the sandboxed convert script is invoked on Linux, or something else entirely. None of these
should be treated as the answer — this needs real investigation, not a guess written into this
issue.

### What was checked and ruled out

This was checked NOT to obviously be a `/dev/null`-related failure: `packs/contracts`'s own
convert scripts (`packs/contracts/ast-grep/to-sarif.sh`, `packs/contracts/grep/to-sarif.sh`)
were inspected directly and contain no `/dev/null` redirects. The two `/dev/null`-specific
sandbox test failures observed on the same CI run are a distinct defect, filed separately as
`ISSUE-168`, and should not be conflated with this one.

### Adjacency to ISSUE-158 — similar shape, not assumed same mechanism

`ISSUE-158` ("Zero Match Harness Patch Makes Pack Unvalidatable," closed) is a prior defect with
a similarly-shaped symptom: a different pack (`packs/substantiveness`) failing its own
`phase3-fixtures` self-validation. That issue's root cause was a test harness patching a rule's
`files:` scope in a way that collided with packval's unconditional validate-on-`pack-add`
pipeline against the pack's own fixture tree — an `ast-grep`-glob-based mechanism. This issue's
symptom cluster is grep-based, not `ast-grep`-glob-based, and involves a platform difference
(Linux CI vs. darwin) that `ISSUE-158` did not. Reference `ISSUE-158` as a precedent for the
general symptom shape ("a pack fails its own phase3-fixtures self-validation"), but this issue
should NOT be assumed to share `ISSUE-158`'s specific root cause until investigation says so.

## Impact

`packs/contracts` — one of backstop-core's own dogfooded packs — cannot validate on Linux CI as
of this filing, which blocks the contracts gate step and everything downstream of it on any
Linux CI run that reaches this pack. The scope of affected tests (roughly a dozen, spanning
E2E, no-vacuous-green, dispatch-wiring, and unit-level contract-engine tests) suggests this is
not an isolated fixture bug but something systemic to how the pack's grep-based checks behave
under Linux CI — but that suggestion is exactly what needs to be confirmed by investigation, not
assumed by this filing.

## Root cause confirmed (2026-08-18)

The root cause is now definitively confirmed via a real, fast, targeted GitHub Actions run on
Linux — not guessed, not inferred from logs. A live diagnostic test was pushed to a throwaway
branch and run directly on `ubuntu-latest`.

### The bug

`pkg/gate/testdata/traceability-pack/grep/to-sarif.sh` and `packs/contracts/grep/to-sarif.sh` are
BYTE-IDENTICAL files (confirmed via `diff`, exit 0 — no difference at all), both containing an
awk script that converts raw `grep -rn -e <pattern> <target>` output into SARIF. The awk script
assumes every grep match line has the 3-field shape `file:line:text` (`NF -F: ... NF >= 3 {
file=$1; line=$2; text=$0; sub(...) ... }`) and silently DROPS any line with fewer than 3
colon-separated fields via that `NF >= 3` guard — no error, no warning, the line simply never
becomes a SARIF result.

### Why this only breaks on Linux

GNU grep (what CI's `ubuntu-latest` runners use) OMITS the filename prefix from its output when
given exactly ONE explicit file as a target (as opposed to a directory or multiple files) —
producing bare `line:text` (2 fields) instead of `file:line:text` (3 fields). BSD grep (macOS,
what every bit of local testing in this repo has used, tonight and presumably always) evidently
DOES include the filename even for a single-file target, which is why this defect has been
invisible on every local run and only surfaced once real Linux CI actually exercised this code
path for the first time (tracing back through tonight's cascade: the `ISSUE-163` fix let CI run
far enough to reach this test for the first time ever).

### Direct evidence, captured from a real Linux CI run

A throwaway diagnostic was pushed to branch `debug/issue166-contracts-grep-repro` (PR #3, now
closed without merging, branch deleted). `grep -rn -e "legacyProbeSymbol" <single-file-path>` on
Linux produced this exact raw stdout:

```
6:// "legacyProbeSymbol" appears here (even in a comment-adjacent identifier), the
8:func legacyProbeSymbol() string { return "should have been deleted" }
```

No filename prefix, just `<line>:<text>` — confirming grep itself works CORRECTLY and finds the
real matches (2 of them, exactly as expected), but the awk conversion script's `NF >= 3` guard
treats each of these 2-field lines as unparseable and drops them, so the SARIF output has ZERO
results despite grep having found genuine matches. This is a **silent false negative** in a
security/compliance-relevant absence-probe rule — the whole point of `contract-absence` is to
BLOCK when a forbidden symbol is present, and on Linux, for any single-file-scoped scan, it
silently finds nothing regardless of what's actually there.

### Blast radius, now precisely explainable

Every one of the roughly 30 failing tests documented in this issue's original symptom list
traces to this ONE mechanism — this supersedes the earlier "pattern observed, mechanism not
traced" framing (which was correct and appropriately cautious at the time, but is now resolved).

The `TestContractsPack_PatternArgFixturesDispatchAndDiscriminate` semgrep-labeled failure in the
original symptom list is a red herring for THIS issue specifically: `phase3.go`'s
"semgrep-negative"/"semgrep-positive" check-name strings are hardcoded literals used for EVERY
findings-engine dispatch regardless of actual engine (confirmed earlier tonight by reading
`pkg/packval/phase3.go` directly), so that failure's label doesn't actually indicate semgrep
involvement — it may or may not share this same grep root cause. Leave that specific one as still
worth double-checking rather than asserting it's covered, since it wasn't part of tonight's direct
reproduction.

### The fix (shape only — not this issue's job to implement)

The awk script needs to handle BOTH grep output shapes correctly, via one or both of:

- Always forcing grep to include the filename (e.g. passing `-H` to the grep invocation wherever
  it's dispatched) — arguably the more robust fix since it fixes the INPUT shape rather than
  working around it; and/or
- Making the awk parsing logic robust to both `file:line:text` and `line:text` shapes rather than
  assuming one universally.

Both `pkg/gate/testdata/traceability-pack/grep/to-sarif.sh` (the testdata fixture script) and
`packs/contracts/grep/to-sarif.sh` (the real production pack script) need the same fix since
they're currently byte-identical. Open design question for the plan: should these two files be
kept manually byte-identical as a convention (and if so, is there a way to make that structural
rather than a hope), or should the fix instead go through wherever grep gets INVOKED (i.e. add
`-H` to the dispatch call sites) so the conversion script itself never needs the awk robustness
fix at all — both are legitimate approaches and the plan should decide, not this issue.

### Why static analysis alone didn't find this

This issue's original filing ruled out the grep pattern, ruled out fixture content, ruled out
sandbox involvement, and ruled out a semgrep-crash-class explanation (via the hardcoded-label
finding) — all correctly, but none of that reached the actual mechanism. It took a real Linux
reproduction, run directly on `ubuntu-latest` with a live diagnostic test, to observe GNU grep's
single-file filename-omission behavior directly; this was not visible from reading the scripts or
CI logs alone.

## References

- CI run `32108003542`, `gate-report.json` (`git_sha: 970512b`), `pack_engines` step — the
  source of every test name and message quoted above, downloaded and inspected directly.
- `packs/contracts/ast-grep/to-sarif.sh`, `packs/contracts/grep/to-sarif.sh` — the pack's own
  convert scripts, inspected and confirmed to contain no `/dev/null` redirects.
- `ISSUE-158` — "Zero Match Harness Patch Makes Pack Unvalidatable" (closed) — similarly-shaped
  prior defect (a different pack failing its own phase3-fixtures self-validation), explicitly
  flagged here as likely a DIFFERENT mechanism, not assumed to share this issue's root cause.
- `ISSUE-163` / commit `970512b` — the `cmd/backstop` `TestMain` fix that let Linux CI reach far
  enough into the suite for this failure cluster to surface for the first time; not itself the
  cause.
- `ISSUE-168` — the separate, fully-traced `/dev/null`-write-denied sandbox defect observed on
  the same CI run; explicitly not the same defect as this one.

### Existence-in-world check

Performed 2026-08-18 before authoring: `grep -ril` over `issues/` and `bundles/` for
"contracts phase3", "phase3 contracts", "absence probe", and "grep absence" matched
`ISSUE-142` (pattern-arg fixtures never dispatch — a different, already-diagnosed dispatch-wiring
defect), `ISSUE-065` (contracts engine capability wiring — unrelated to this symptom), and
`BUNDLE-009`/`BUNDLE-005` (traceability and pack-validation bundle charters, neither of which
owns this specific Linux-CI grep-absence-probe symptom). None duplicate this issue's surface.
