---
title: "SPEC-018: Gate Diff Scoping — Changed-Files-Only Default"
number: SPEC-018
created: "2026-05-20"
status: ready-for-implementation
schema_version: spec/v1
spec_version: 1.0.1

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
    text: Gate defaults to diff mode, includes tracked and untracked changes, and uses the merge-base cascade.
    tests:
      - TestGate_DefaultsToDiffMode
      - TestGateScope_IncludesUntrackedFiles
  - id: CLM-002
    requirement: REQ-002
    text: --all restores full-project sweep behavior.
    tests:
      - TestGate_AllFlagUsesFullSweep
  - id: CLM-003
    requirement: REQ-003
    text: --file scopes file-scoped checks to explicit files, preserves always-run structural checks, and conflicts with --all.
    tests:
      - TestGate_FileFlagScopesExplicitFiles
      - TestGate_AllAndFileMutuallyExclusive
  - id: CLM-004
    requirement: REQ-004
    text: Gate computes the changed-file set once and shares it across steps.
    tests:
      - TestGateScope_ComputedOnce
  - id: CLM-005
    requirement: REQ-005
    text: Gate steps filter checks to the changed-file set in diff mode.
    tests:
      - TestGateSteps_FilterToChangedFiles
  - id: CLM-006
    requirement: REQ-006
    text: Pack lock verification and pack validators always run in every scope mode.
    tests:
      - TestGateSteps_PackLockAlwaysRuns
  - id: CLM-007
    requirement: REQ-007
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
      - name: gateCmd
        kind: variable
        signature: "var gateCmd *cobra.Command"
        notes: "Registers --all and --file scope flags on the gate command"
      - name: newGateCommand
        kind: function
        signature: "func newGateCommand(jsonFlag *bool) *cobra.Command"
        notes: "Cobra command registers --all and --file, defaults execution to diff mode, and validates --all/--file mutual exclusion before running gate steps"
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

- **1.0.1** (2026-07-07) — Marked CLM-008 `kind: absence`.
  `TestSpec010Req012Superseded` is a structural supersession assertion over
  the SPEC-010 and SPEC-018 artifact files that calls no code package by
  design; `kind: absence` exempts it from the noTarget substantiveness join
  per the claims schema. Surfaced and corrected by ISSUE-047's de-baking of
  the substantiveness noTarget guard (removal of the vacuous `cmd/`-path
  skip that had previously hidden this structural claim). No change to the
  claim text, requirement mapping, or test name.
