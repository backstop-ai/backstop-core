---
title: "SPEC-018: Gate Diff Scoping — Changed-Files-Only Default"
number: SPEC-018
created: "2026-05-20"
status: implemented
schema_version: spec/v1
spec_version: 1.2.0

source:
  directive: DIR-011
  bundle: BUNDLE-008

implementation:
  summary: >
    Change the backstop gate command to default to diff mode (changed files
    only) instead of full-project sweep. Gate computes the changed-file set
    once at startup using the existing ScopeModeDiff 4-step merge-base
    cascade from pkg/check/scope.go and threads that set through all gate
    steps. Add --all flag to restore full-sweep behavior and --file flag
    for explicit file lists. Each file-scoped gate step scopes its checks
    to the changed-file set (except pack lock/validators which always run).
    Gate outputs a scope summary line in diff mode.
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop ./pkg/gate/... -run 'TestGate|TestBackstopGate' -race -coverprofile=cover.out -v
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      backstop gate with no scope flags must default to diff mode. The
      changed-file set is computed using the existing ScopeModeDiff 4-step
      merge-base cascade from pkg/check/scope.go: (1) origin/main
      merge-base, (2) origin/master merge-base, (3) local staged +
      unstaged changes, (4) fallback to full codebase scan with warning.
      The changed-file set must include both tracked modifications (vs
      merge-base) and untracked files so new files are never silently
      skipped.

  - id: REQ-002
    text: >
      backstop gate --all must run in full-project sweep mode, matching
      current behavior (ScopeModeAll). This is the flag for CI post-merge
      or when a comprehensive audit is needed.

  - id: REQ-003
    text: >
      backstop gate --file a.go b.go must accept an explicit file list
      and scope file-scoped/per-file gate steps to exactly those files,
      while preserving always-run structural checks described in REQ-006.
      This is a manual override for when the git diff does not reflect
      reality (e.g., generated files, external tool changes). --file and
      --all are mutually exclusive; specifying both must produce a config
      error (exit 2).

  - id: REQ-004
    text: >
      The changed-file set must be computed once at gate start and shared
      across all gate steps. No step may independently re-compute the
      file set. This ensures consistency (every step sees the same files)
      and avoids redundant git operations.

  - id: REQ-005
    text: >
      In diff mode, each gate step must scope its checks to the
      changed-file set: (a) artifact validation checks only changed
      artifact files (specs, bundles, plans in the diff), (b) code check
      runs lint/build/test/semgrep against changed files (ScopeModeDiff
      already supports this), (c) test verification checks mandated
      tests only in changed test files, (d) test substantiveness checks
      only changed test files, (e) coverage runs for changed packages,
      (f) contract signature checks only changed source files.

  - id: REQ-006
    text: >
      Pack lock verification (VerifyLock) and pack validators must
      always run in all scope modes (diff, file, all). These are
      structural checks on project shape, not per-file checks.

  - id: REQ-007
    text: >
      In diff mode, gate must output a scope summary line before step
      results, e.g., "Gate running against 12 changed files (use --all
      for full sweep)." In --all mode, no scope summary is needed. In
      --file mode, any explicit-file count summary is diagnostic only and
      is not an acceptance requirement for BUNDLE-008/DIR-011.

  - id: REQ-008
    text: >
      This spec supersedes SPEC-010 REQ-012 ("gate accepts no scope
      flags"). Gate now accepts --all and --file. The original intent
      of REQ-012 (gate is comprehensive) is preserved via --all.

  - id: REQ-009
    text: >
      When the changed-file set is empty (no files changed), gate must
      skip file-scoped checks and report zero file-scoped violations
      with a message indicating no changed files were found. Pack lock
      and validator steps still run per REQ-006. The gate succeeds only
      if those always-run structural checks pass; if they fail, the gate
      fails and reports those structural violations even though scoped
      changed-file violations are zero.

supersedes:
  - spec: SPEC-010
    requirement: REQ-012
    reason: >
      Gate now accepts --all and --file scope flags. The original
      intent (gate is comprehensive) is preserved via the --all flag.

claims:
  - id: CLM-001
    requirement: REQ-001
    subject: cmd/backstop
    text: Gate invoked with no scope flags defaults to diff mode.
    tests:
      - TestGate_DefaultsToDiffMode
  - id: CLM-010
    requirement: REQ-001
    subject: pkg/gate
    text: The diff-mode changed-file set is computed via the merge-base cascade and contains both tracked modifications and untracked files, so new files are never silently skipped.
    tests:
      - TestGateScope_IncludesUntrackedFiles
  - id: CLM-002
    requirement: REQ-002
    subject: cmd/backstop
    text: --all restores full-project sweep behavior.
    tests:
      - TestGate_AllFlagUsesFullSweep
  - id: CLM-003
    requirement: REQ-003
    subject: cmd/backstop
    text: --file scopes file-scoped checks to explicit files, preserves always-run structural checks, and conflicts with --all.
    tests:
      - TestGate_FileFlagScopesExplicitFiles
      - TestGate_AllAndFileMutuallyExclusive
  - id: CLM-004
    requirement: REQ-004
    subject: pkg/gate
    text: Gate computes the changed-file set once and shares it across steps.
    tests:
      - TestGateScope_ComputedOnce
  - id: CLM-005
    requirement: REQ-005
    subject: pkg/gate
    text: Gate steps filter checks to the changed-file set in diff mode.
    tests:
      - TestGateSteps_FilterToChangedFiles
  - id: CLM-006
    requirement: REQ-006
    subject: pkg/gate
    text: Pack lock verification and pack validators always run in every scope mode.
    tests:
      - TestGateSteps_PackLockAlwaysRuns
  - id: CLM-007
    requirement: REQ-007
    subject: pkg/gate
    text: Human output includes the required diff-mode scope summary line; any file-mode summary is diagnostic only.
    tests:
      - TestGateOutput_ScopeSummary
  - id: CLM-008
    requirement: REQ-008
    kind: absence
    text: SPEC-010 REQ-012 is superseded by the new scope flags.
    tests:
      - TestSpec010Req012Superseded
  - id: CLM-009
    requirement: REQ-009
    subject: pkg/gate
    text: Empty diffs report zero file-scoped violations while structural checks still run and can fail the gate.
    tests:
      - TestGate_EmptyDiff

contracts:
  - file: pkg/gate/scope.go
    provides:
      - name: GateScope
        kind: type
        signature: "type GateScope struct"
      - name: ComputeGateScope
        kind: function
        signature: "func ComputeGateScope(projectRoot string, mode GateScopeMode, files []string) (*GateScope, error)"
    consumes:
      - source: pkg/check/scope.go
        name: ComputeChangedFiles
        kind: function
  - file: cmd/backstop/gate.go
    provides:
      - name: newGateCommand
        kind: function
        signature: "func newGateCommand(jsonFlag *bool) *cobra.Command"
        notes: "The constructor IS the package-level surface for this file: it builds the Cobra gate command, registers --all and --file, defaults execution to diff mode, and validates --all/--file mutual exclusion before running gate steps. The command VALUE is a function-local (`gateCmd := newGateCommand(&jsonFlag)` in cmd/backstop/root.go, consumed by rootCmd.AddCommand there) and is therefore not a declarable contract symbol."
    consumes: []
  - file: pkg/gate/output.go
    provides:
      - name: FormatHuman
        kind: function
        signature: "func FormatHuman(result GateResult, noColor bool) string"
    consumes: []
---

# SPEC-018: Gate Diff Scoping — Changed-Files-Only Default

## Overview

The backstop gate command currently runs all verification steps against
the entire project (ScopeModeAll hardcoded), producing 500+ violations
on every run. Most violations are pre-existing noise from specs covering
dead or unimplemented features. Agents cannot distinguish violations
they caused from violations that already existed, making the gate
unusable as a feedback signal during implementation.

The fix is straightforward: gate should default to diff mode using the
existing ScopeModeDiff infrastructure. Changed files are computed once,
each step checks only those files, and the agent sees 0-5 actionable
violations instead of 500+.

## Requirements

See frontmatter requirements REQ-001 through REQ-009. The requirements define the scope-mode defaults, explicit override flags, single changed-file computation, per-step filtering for file-scoped checks, always-run structural checks, output summaries, SPEC-010 supersession, and empty-diff behavior. No JSON `scope` field is required by this spec; any such field is diagnostic only and is not part of acceptance unless a later spec requires it.

## Implementation

### 1. Gate CLI flag registration

Add `--all` (bool) and `--file` (string slice) flags to the gate
command in `cmd/backstop/gate.go`. Validate mutual exclusivity.
Default scope mode is `ScopeModeDiff` when neither flag is set.

### 2. Changed-file computation at gate start

At the top of gate execution (before any steps), compute the
changed-file set:
- If `--all`: use `ScopeModeAll` (no file filtering)
- If `--file`: use provided file list directly
- Default: call `scope.ComputeChangedFiles()` using the 4-step
  merge-base cascade from `pkg/check/scope.go`

Store the result in a `GateScope` struct that is threaded through
all steps.

### 3. Per-step scope filtering

Each file-scoped/per-file gate step receives the `GateScope` and filters
its inputs accordingly. Steps that currently operate on the full project
but evaluate per-file inputs must be modified to accept a file filter.
Pack lock/validator steps ignore the filter and always run in diff,
file, and all modes.

### 4. Scope summary output

Emit a human-readable summary line before step results in diff mode.
File mode may emit a similar explicit-file count for operator clarity,
but that line is diagnostic only and not required by this spec.
If JSON output includes scope metadata, that metadata is a diagnostic
implementation detail only: it is not a committed requirement for
BUNDLE-008/DIR-011 and must not be treated as an acceptance criterion
for this spec.

### 5. SPEC-010 REQ-012 supersession

Update or annotate SPEC-010 to reflect that REQ-012 is superseded
by this spec.

## Verification

1. `backstop gate` (no flags) in a repo with changes produces
   violations only for changed files, not the entire project.
2. `backstop gate --all` produces the same output as current behavior.
3. `backstop gate --file a.go b.go` scopes file-scoped checks to exactly
   those files while pack lock/validators still run.
4. `backstop gate --all --file a.go` exits with code 2 (config error).
5. Empty diff produces zero file-scoped violations and an informational
   message. This is not unconditional gate success: pack lock/validators
   still run, and structural violations from those always-run checks
   fail the gate and are reported.
6. Scope summary line appears in human output for diff mode. File-mode
   summary lines and JSON scope metadata, if present, are diagnostic only
   and not required for acceptance by this spec.
7. Untracked (new) files appear in the changed-file set.
8. CLI integration tests in `cmd/backstop` cover Cobra flag registration
   and defaults in `cmd/backstop/gate.go`, including no-flag default diff
   mode selection and `--all`/`--file` mutual exclusion before gate
   execution.

## Out of Scope

- Baseline comparison / violation diffing (BUNDLE-007)
- Spec-level implementation verification ("is this plan complete?")
- Waiver/suppression mechanisms
- CI environment auto-detection for mode selection

## Version History

- **1.2.0** (2026-08-02) — **Multi-target claim modeling fixed; the 1.1.0 closure blocker is
  cleared.** CLM-001 is SPLIT and every non-absence claim now carries an explicit per-claim
  `subject:`. No requirement text, no test name, and no assertion changed — the two tests
  CLM-001 mandated are redistributed, one per claim, and its three assertions are preserved
  across the pair.

  **The split.** CLM-001 asserted three things at once ("Gate defaults to diff mode, includes
  tracked and untracked changes, and uses the merge-base cascade") over tests in two different
  packages, so no single `subject:` could describe it. It is now:
  - **CLM-001** (REQ-001, `subject: cmd/backstop`) — "Gate invoked with no scope flags defaults
    to diff mode", verified by `TestGate_DefaultsToDiffMode`
    (`cmd/backstop/gate_test.go`, `package main`).
  - **CLM-010** (REQ-001, `subject: pkg/gate`) — the changed-file set is computed via the
    merge-base cascade and contains BOTH tracked modifications AND untracked files, verified by
    `TestGateScope_IncludesUntrackedFiles` (`pkg/gate/scope_test.go`, `package gate`), which
    commits a tracked file, modifies it, adds an untracked one, and asserts
    `ComputeGateScope(root, GateScopeModeDiff, nil)` contains both.
  `CLM-010` is the next free id: the claim-id grammar is strictly `^CLM-\d{3}$`
  (`claimIDRe`, `pkg/validate/spec.go`), so suffixed ids like `CLM-001a` are NOT
  representable and appending is the only conforming split. Both halves still trace to REQ-001,
  which retains both the "defaults to diff mode" and the tracked+untracked sentences — the
  requirement was always the union of the two claims.

  **Per-claim subjects.** Each claim's `subject:` was set from the VERIFIED on-disk location of
  its own tests, not from intent: `cmd/backstop` for CLM-001/002/003 (all five of those tests
  are in `cmd/backstop/gate_test.go`) and `pkg/gate` for CLM-004/005/006/007/009/010
  (`pkg/gate/scope_test.go`, `step_delegate_test.go`, `output_test.go`). The `cmd/backstop`
  subjects restate what those claims already inherited from `implementation.package`; they are
  written out so the split is legible and so a future edit to the spec-level default cannot
  silently retarget them. CLM-008 keeps `kind: absence` and deliberately takes NO subject — it
  is pre-join skipped (`NoTargetViolationForTest`), which is why its
  `pkg/validate/spec_test.go` home is not a mismatch. Every claim is now single-package, so
  `TargetPackageName` reduces each subject to a token (`backstop`, `gate`) that the mandated
  tests satisfy by colocation.

  **PLAN-SPEC-018 is deliberately untouched.** It names "CLM-001 through CLM-009" in prose
  acceptance lines. That plan is a completed historical record, its prose is not gate-consumed
  (`requirement_traceability` joins bundle requirements to spec requirements and reads no
  claims), and hand-editing artifacts is barred. The stale range is a cosmetic in a finished
  artifact, not a live gap.

- **1.1.0** (2026-08-02) — **Delivery verified; closure BLOCKED on multi-target claim
  subjects.** Status stays `ready-for-implementation`. A flip to `implemented` was attempted
  in this edit and REVERTED — the claims mandate tests in TWO packages and the single
  spec-level subject cannot express that; splitting claims is required and was not
  authorized. See "Closure blocked" below. The one contract correction from that attempt is
  KEPT, because it is correct independent of the status question.

  **Delivery verified.** **Delivered by** commit `c976408`
  ("feat: BUNDLE-008 — scope gate checks to changed files"), which added `pkg/gate/scope.go`,
  the `cmd/backstop/gate.go` threading, and this spec's plan in one change. **Verified present
  in the tree:** REQ-001 diff default with untracked files (`cmd/backstop/gate.go`
  — `scopeMode := gate.GateScopeModeDiff` with no flags; `pkg/gate/scope.go` merge-base
  cascade plus `git ls-files --others --exclude-standard`); REQ-002 `--all` →
  `gate.GateScopeModeAll`; REQ-003 `--file` plus the mutual-exclusion refusal
  (`"config: --all and --file are mutually exclusive"` returned as `ExitConfigError`, exit 2);
  REQ-004 single computation (one `gate.ComputeGateScopeWithBase` call at gate start, threaded
  to the steps — no step recomputes); REQ-005 (a)–(f) all threaded through `*GateScope`;
  REQ-007 the summary line verbatim in `pkg/gate/output.go`
  (`"Gate running against %d changed files (use --all for full sweep).\n"`); REQ-009 the
  empty-diff message (`"Gate found no changed files; scoped checks have no files to inspect."`)
  with the always-run structural steps intact. All 11 mandated tests exist as real functions.
  **Contract correction (load-bearing, not cosmetic).** The `cmd/backstop/gate.go` contract
  declared a package-level `gateCmd` (`kind: variable`, `signature: "var gateCmd
  *cobra.Command"`). No such package-level variable has ever existed: `gateCmd` is a
  FUNCTION-LOCAL in `cmd/backstop/root.go` (`gateCmd := newGateCommand(&jsonFlag)`, consumed
  two dozen lines later by `rootCmd.AddCommand(...)`). A local is not a contract surface, so
  that entry was unsatisfiable by construction. It is removed; the real declarable symbol on
  this surface — the constructor `func newGateCommand(jsonFlag *bool) *cobra.Command`, read
  verbatim from `cmd/backstop/gate.go` — was already declared alongside it, so the fix is a
  de-duplication onto the true symbol, with the removed entry's note folded into the
  constructor's. The ordering matters: contract enforcement activates only at terminal status
  (`contractsAreDue`, `pkg/gate/step_testverify.go`), and only `provides` entries become
  gate-dispatched `ContractEntry` records (`ExtractContractEntries`) — so the stale entry was
  inert while `ready-for-implementation` and would have gone live, red, on the attempted flip.
  With the flip reverted it is inert again, but the fix is RETAINED so the eventual closure is
  clean on this dimension. The
  three surviving `provides` signatures were re-read against the tree and match:
  `type GateScope struct` and `func ComputeGateScope(projectRoot string, mode GateScopeMode,
  files []string) (*GateScope, error)` (`pkg/gate/scope.go`), and
  `func FormatHuman(result GateResult, noColor bool) string` (`pkg/gate/output.go`).
  DIR-011 is this spec's own parent directive and is already `done` — the same work, not rival
  work.

  **Closure BLOCKED — root cause.** The attempted flip to `implemented` produced SIX new
  `test_substantiveness` violations on a full gate run, all of the form `test function X does
  not call package backstop`. This spec declares `implementation.package: cmd/backstop` and no
  `subject:`; `implementationSubject` (`pkg/gate/step_testverify.go`) coalesces onto the
  deprecated `package` alias, and `TargetPackageName` (`pkg/gate/substantiveness_join.go`)
  reduces `cmd/backstop` to the opaque token `backstop`. But SPEC-018 is genuinely
  MULTI-TARGET — it threads scope through BOTH `cmd/backstop` (flag registration, mutual
  exclusion) and `pkg/gate` (`GateScope`, per-step filtering, output). Six of its eleven
  mandated tests live in `package gate`: `TestGateScope_IncludesUntrackedFiles`,
  `TestGateScope_ComputedOnce` and `TestGate_EmptyDiff` (`pkg/gate/scope_test.go`),
  `TestGateSteps_FilterToChangedFiles` and `TestGateSteps_PackLockAlwaysRuns`
  (`pkg/gate/step_delegate_test.go`), and `TestGateOutput_ScopeSummary`
  (`pkg/gate/output_test.go`). None is colocated with `backstop` and none references it, so
  the noTarget set-join raises for all six. The join is behaving exactly as designed and the
  tests are substantive — they DO call the package they verify. The spec's single-target
  metadata is what is wrong. The findings were dormant only because the noTarget join is
  implemented-only (`ContractsAreDue`, applied in `buildTestSubstantivenessStep`,
  `cmd/backstop/gate.go`, ISSUE-054); terminal-status enforcement is what woke them.

  **Why the supported fix does not reach.** Per-claim `subject:` overrides exist for exactly
  this case (ISSUE-047, honored in `ExtractMandatedTests`), and they resolve eight of the nine
  claims: CLM-002 and CLM-003 are wholly `cmd/backstop` and would inherit the spec default;
  CLM-004, CLM-005, CLM-006, CLM-007 and CLM-009 are wholly `pkg/gate` and would take
  `subject: pkg/gate`; CLM-008 is already exempt via `kind: absence` (its test lives in
  `pkg/validate/spec_test.go`). **CLM-001 is the blocker: it mandates BOTH
  `TestGate_DefaultsToDiffMode` (`cmd/backstop/gate_test.go`, `package main`, referencing no
  `gate.` symbol) and `TestGateScope_IncludesUntrackedFiles` (`pkg/gate/scope_test.go`,
  `package gate`).** A subject is one value per claim: `pkg/gate` clears the second test and
  raises a NEW noTarget on the first, while `cmd/backstop` leaves the second raised. Satisfying
  both requires SPLITTING CLM-001 into a cmd-scoped claim and a gate-scoped claim — a change to
  claim structure that was not authorized in this edit. Status is therefore reverted rather
  than closed with a knowingly-red gate or with a subject that mis-describes the claim.
  Closure unblocks once CLM-001 is split: every claim is then single-package and per-claim
  subjects finish the job. No requirement, claim, or test-name text changed.

- **1.0.1** (2026-07-07) — Marked CLM-008 `kind: absence`.
  `TestSpec010Req012Superseded` is a structural supersession assertion over
  the SPEC-010 and SPEC-018 artifact files that calls no code package by
  design; `kind: absence` exempts it from the noTarget substantiveness join
  per the claims schema. Surfaced and corrected by ISSUE-047's de-baking of
  the substantiveness noTarget guard (removal of the vacuous `cmd/`-path
  skip that had previously hidden this structural claim). No change to the
  claim text, requirement mapping, or test name.
