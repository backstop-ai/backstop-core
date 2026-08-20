---
title: "Contracts Pack Phase3 Fixtures Fail On Linux Ci"
schema_version: issue/v1
delivered_by: PLAN-ISSUE-166

issue:
  id: ISSUE-166
  title: "Contracts Pack Phase3 Fixtures Fail On Linux Ci"
  type: bug
  status: closed
  created: "2026-08-18"
  closed: "2026-08-19"

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

## Verification ceiling — fix landed and CI-confirmed, closure held on one external block (2026-08-18)

**The fix has landed on `main` (commit `f8b3846`, via `PLAN-ISSUE-166`, 2 independent
plan-reviewer rounds to signoff). This issue stays `open` — do not read this note as a close.**
Formal closure via `delivered_by: PLAN-ISSUE-166` requires that plan's own `status` field to read
`completed`; it currently reads `draft`, and — more than a stale field — one of its own tasks
(`TASK-008` step 5 / `TASK-009`, Phase 3) has not actually finished executing, for a reason
external to this fix. See "What is genuinely still open" below before assuming this is ready to
close.

### The fix, restated precisely

Two halves, both required (neither alone closes the gap):

- **Half A — force the input shape.** Every repo-owned convert-bearing `command: grep`
  declaration (six manifests, one of them `pkg/pack/engine/testdata/contracts-grep-engine.yml`,
  not named `pack.yml` and found only by a structural sweep, not a filename-keyed one) and every
  in-repo test helper that shells real grep (four call sites) now pass both `-H` (forces the
  filename header GNU grep omits for a single explicit-file target) and `-I` (suppresses grep's
  non-match `Binary file … matches` stdout line, which a naive loud refusal would otherwise choke
  on).
- **Half B — make the silent drop loud.** Every repo-owned grep→SARIF convert script (five of
  them, discovered structurally rather than listed) now REFUSES — nonzero exit, a stderr
  diagnostic naming the offending line and the `-H -I` remedy — on a stdin line it cannot parse as
  `<file>:<line>:<text>`, instead of silently dropping it. A heuristic 2-field parse was measured
  and rejected: `6:42: text` is genuinely ambiguous and such a parser fabricates a finding at a
  nonexistent file named `6` (a silent false positive traded for the silent false negative being
  fixed).

A 2-field parse was the alternative this issue's own "fix shape" section floated; it was measured
and rejected in favor of forcing the input shape at the source, for the reason above.

### Two facts this issue got slightly wrong, now corrected

- **Three in-repo copies of the identical convert script, not two.** This issue named
  `packs/contracts/grep/to-sarif.sh` and `pkg/gate/testdata/traceability-pack/grep/to-sarif.sh` as
  byte-identical. A third, `pkg/gate/testdata/ts-proof-pack/grep/to-sarif.sh`, was also
  byte-identical and carried the identical defect, but was not named here — found only by
  `PLAN-ISSUE-166`'s own discovery sweep. A fourth copy, the installed external mirror
  (`.backstop/packs/backstop-ai/go-contracts/grep/to-sarif.sh`), also carried it. See `ISSUE-174`
  below for the general gap this asymmetry surfaced.
- **BSD grep DOES prefix the filename for a single-file `-rn` target.** This issue's "fix shape"
  section left open whether local (darwin) testing could even observe the platform divergence.
  It's now measured directly: BSD grep 2.6.0-FreeBSD at `/usr/bin/grep` DOES print the filename
  under `-rn` for a single explicit file, which is exactly why this defect was invisible on every
  local run and only GNU grep (Linux CI) omits it. The divergence is real and now confirmed on
  both sides, not merely inferred from GNU's behavior alone.

### Local evidence

Re-run 2026-08-18 against the committed tree at `f8b3846`, the plan's own mandated test command:

```
go test ./pkg/pack/engine/ ./pkg/gate/ ./pkg/packval/ -run "TestGrepConvert|TestGrepEngineDeclarations|TestGrepTestHelpers|TestThinExecutor_NoGrepInvocation|TestRealGrep|TestInstalledGoContractsPack|TestContract_Absence|TestEngine_Grep|TestTSPack_ContractAbsenceGrep|TestEquivalence_GoAbsence|TestPackVerdict_|TestPackContractResult_|TestContractsPack_PatternArg" -race -count=1
```

Exactly one failure: `TestInstalledGoContractsPack_CarriesFilenameHeaderFix` — a DELIBERATE
true-RED (see "What is genuinely still open" below). Everything else green, no skips.
`backstop gate --all` run locally the same day: 2 violations, neither attributable to this lane
(a pre-existing empty-`phases` defect on an unrelated plan, and this machine's own
`go-arch-lint`-not-on-PATH capability gap) — zero violations on any file this lane touched.

### Real Linux CI evidence — TASK-012's obligation, satisfied by direct comparison

Per this plan's own rule, none of the above is evidence the ~30-test Linux CI cluster is actually
fixed — only a real Linux run is. Two real CI runs' `gate-report.json` were read directly and
compared: `32172705491` (commit `9aa278e`, the plan commit BEFORE the fix's implementation tasks)
against `32179966270` (commit `f8b3846`, the fix itself).

**BEFORE: 35 blocking `pack_engines` errors, including this issue's entire original symptom
cluster** — every test named in "The broad symptom" and "A separate, more specific symptom
cluster" above (`gate_contract_e2e_test.go` x5, `gate_contract_novacuous_test.go` x3,
`gate_contract_wiring_test.go`, `init_acceptance_test.go` x7, `init_seams_test.go` x2,
`contract_equivalence_test.go`, `contract_pack_paths_test.go` x3, `contracts_go_rules_test.go`
x3, `contracts_grep_engine_test.go`, `contracts_ts_rules_test.go`, `contracts_pack_dispatch_test.go`
including the `TestContractsPack_PatternArgFixturesDispatchAndDiscriminate` semgrep-labeled red
herring), plus 3x `bun_ratchet_flip_test.go` and 1x `contracts_local_install_test.go`.

**AFTER: 5 blocking errors.** Every test in that cluster is GONE, including the specific eight
named in the plan's own `TASK-012`: `TestContract_AbsencePresentSymbolGrepMatchViolation`,
`TestContract_AbsenceScopeFileOrPathParameterized`, `TestContract_AbsenceUsesGrepTextPresenceNotAstGrep`,
`TestEngine_GrepConvertScriptEmitsValidSarif`, `TestEquivalence_GoAbsencePresentAndAbsentMatchLegacy`,
`TestPackVerdict_PresentAndAbsencePolarities`, `TestPackContractResult_AllPolaritiesOverRealEngines`,
`TestPackContractResult_ScopeFallbackAndMissingFile` — all confirmed green. The
`pack add … packs/contracts … phase3-fixtures` refusal this issue opened with is gone. The
semgrep-labeled red herring is ALSO resolved, reported here as an observation per the plan's own
instruction not to claim it as part of this lane's targeted fix.

**The 5 remaining errors were each checked byte-identical between the BEFORE and AFTER runs —
genuinely pre-existing, not caused or left behind by this fix — and are now each attributed to
their own filed issue rather than absorbed:**

1. **3x `bun_ratchet_flip_test.go`** — CI's `gate` job has no `.backstop/baseline.json` to read at
   all (gitignored, nothing in `ci.yml` restores or generates one before "Run the gate"). Filed as
   `ISSUE-176`; explicitly not the same gap as `ISSUE-086` (which covers the separate `baseline`
   job's packless generation, not the `gate` job having nothing to read).
2. **1x `contracts_local_install_test.go: TestInstallContractsLocalPack_InstallsWithSuppliedCommand`**
   — was named in this issue's own original affected-test list, but did NOT clear despite going
   through the same `pack add`/`pack test` `phase3-fixtures` path as roughly a dozen structural
   siblings that all cleared. A real anomaly, not expected residue — filed as `ISSUE-177` for its
   own investigation (the itemized 14 validation errors have not yet been read; CI's gate output
   truncates to the summary line and there is no separate verbose test step).
3. **1x `grep_installed_pack_test.go: TestInstalledGoContractsPack_CarriesFilenameHeaderFix`** —
   the deliberate true-RED, exactly as designed. See below.

**Not yet measured, recorded as an outstanding gap rather than claimed:** the plan's `TASK-012`
also asks what stream and wording GNU grep's own binary-file notice uses without `-I` (sharp edge
15's decision — `-I` was chosen specifically to make this question moot rather than match
unmeasurable wording). That has not been spent on a dedicated CI round as of this writing; it is
non-load-bearing for the fix (the SARIF output is unaffected either way, per the plan's own
measurement) but is left open here rather than silently dropped.

### What is genuinely still open — why this is not yet a close

`TestInstalledGoContractsPack_CarriesFilenameHeaderFix` is a DELIBERATE true-RED, not a bug: it
pins that `backstop.lock` still records `backstop-ai/go-contracts` at `1.2.0`, and that the
INSTALLED pack — the one core's OWN contracts gate actually consumes — still carries neither the
`-H -I` flags nor the loud-refusal convert. The external mirror repo was independently fixed,
version-bumped to `1.3.0`, tagged and pushed, and verified from a fresh clone. But
`./bin/backstop pack update backstop-ai/go-contracts` in THIS repo is blocked: `pack update`
re-runs the full `packval` validation pipeline against the new tag, and `1.3.0` still carries
`ISSUE-157`'s SEPARATE, PRE-EXISTING, already-filed, founder-gated inverted-fixture-polarity
defect (a different rule family — signature/ast-grep, not this issue's grep-absence family) —
confirmed byte-identical on the pristine pre-fix `v1.2.0` tag via a real `pack test`
control-vs-treatment comparison, so this is genuinely unrelated to this fix, not caused by it.

This is not a paperwork gap. It is the exact production risk this issue's own "Root cause
confirmed" section names: until the INSTALLED pack is updated, core's own contracts gate remains
silently vacuous on Linux for any file-scoped absence contract, regardless of what the file
contains — the same silent false negative this issue exists to close, just not yet closed in the
one place backstop-core's own gate actually reads from. `PLAN-ISSUE-166`'s own `TASK-008` text
anticipated this exact outcome as a legitimate stopping point ("If `pack update` refuses, report
the typed refusal verbatim and STOP... not an obstacle to route around") — so this is the plan
working as designed, not a defect in this lane's execution. It genuinely blocks that plan's own
`CLM-006` ("core consumes it... via the real `pack update`") from being true yet, which is why the
plan cannot honestly be marked `completed`, and why `delivered_by` cannot yet be used to close this
issue.

**Path to close:** `ISSUE-157` has been separately resolved (`PLAN-ISSUE-157`, `status: completed`,
closed 2026-08-18) — its fix landed as `1.4.0`, not the `1.3.0` this paragraph originally expected,
because `1.3.0` still carried `ISSUE-157`'s inverted-fixture-polarity defect and was never actually
adoptable. `pack update backstop-ai/go-contracts` has already been re-run as part of that plan and
core is now relocked to `1.4.0`; `TestInstalledGoContractsPack_CarriesFilenameHeaderFix` is
confirmed green. What remains is to flip `PLAN-ISSUE-166` to `completed` and close this issue via
`delivered_by: PLAN-ISSUE-166`. Until then this issue accurately reflects reality by staying `open`
with this note, rather than a `closed` status its own close-out has not yet performed.

## Resolution

Fixed at commit `f8b3846`, delivered by `PLAN-ISSUE-166` (`status: completed`). Root cause: GNU
grep (Linux CI's `ubuntu-latest`) silently omits the filename prefix when scanning exactly one
explicit file, unlike BSD grep (darwin), which always includes it — breaking the
`file:line:text` assumption in every repo-owned grep→SARIF convert script and producing a silent
false negative on the `contract-absence` rule. Fixed with both halves: force the input shape
(`-H -I` at every grep dispatch site) and make the drop loud (every convert script now refuses,
rather than silently drops, a stdin line it cannot parse).

**Lineage correction, stated plainly.** This issue's own plan originally expected the relock
that satisfies its `TASK-008`/`TASK-009` to land core at `backstop-ai/go-contracts` v1.3.0. That
version turned out to still carry a separate, unrelated, pre-existing inverted-fixture-polarity
defect (`ISSUE-157`) that blocked `pack update` from ever landing it. The actual relock that
satisfied this plan's tasks came from `PLAN-ISSUE-157` (commit `0943ec4`), landing core at
`backstop-ai/go-contracts` v1.4.0, not v1.3.0.

Confirmed genuinely proven, not just locally verified: real Linux CI runs `32314302525` and
`32315586649`, both `conclusion: success` on `main`.

4 follow-on issues were filed from this investigation and remain their own separate,
already-existing closures, cited here rather than absorbed: `ISSUE-174` (pack-source/external-
mirror sync gap), `ISSUE-175` (orphaned `convert:` reference), `ISSUE-176` (CI `gate` job's
missing `.backstop/baseline.json`), `ISSUE-177` (`TestInstallContractsLocalPack_
InstallsWithSuppliedCommand` anomaly). `SPEC-069`'s stale prose was also corrected as part of
this same lineage, separately.

## References

- CI run `32108003542`, `gate-report.json` (`git_sha: 970512b`), `pack_engines` step — the
  source of every test name and message quoted above, downloaded and inspected directly.
- CI runs `32172705491` (commit `9aa278e`) and `32179966270` (commit `f8b3846`) — the real
  before/after comparison establishing the fix's effect and the 5 residual, pre-existing errors.
- `packs/contracts/ast-grep/to-sarif.sh`, `packs/contracts/grep/to-sarif.sh` — the pack's own
  convert scripts, inspected and confirmed to contain no `/dev/null` redirects.
- `PLAN-ISSUE-166` — the implementing plan (2 plan-reviewer rounds to signoff), currently
  `status: draft` pending `TASK-008`/`TASK-009`.
- `ISSUE-157` — the pre-existing, unrelated, founder-gated `backstop-ai/go-contracts` defect
  (inverted signature/ast-grep fixture polarity) that blocks `pack update` and therefore this
  issue's own formal closure.
- `ISSUE-174` — the general pack-source/external-mirror sync gap this fix's own discovery
  surfaced (four copies of one script, three carrying the same defect).
- `ISSUE-175` — the orphaned `convert:` reference found during the same discovery sweep.
- `ISSUE-176` — the CI `gate` job's missing `.backstop/baseline.json`, one of the 5 residual
  errors, confirmed pre-existing.
- `ISSUE-177` — the `TestInstallContractsLocalPack_InstallsWithSuppliedCommand` anomaly, the
  other residual error, confirmed pre-existing but unexplained.
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
