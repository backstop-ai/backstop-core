---
title: "SPEC-019: Baseline — CI-Generated Immutable Violation Reference"
number: SPEC-019
created: "2026-05-25"
status: ready-for-implementation
schema_version: spec/v1
spec_version: 1.0.1

source:
  directive: DIR-012
  bundle: BUNDLE-007

implementation:
  summary: >
    Replace the deferred gate step 7 with a real baseline subsystem. CI runs
    backstop gate --all after every merge to main, writes a portable versioned
    JSON baseline containing the full raw violation set, and publishes that
    JSON as a GitHub Actions artifact. Local gate runs cache the latest
    baseline at .backstop/baseline.json with configurable TTL, compare only
    the current scoped violations against the full baseline, and fail only on
    new scoped violations. Missing or unreachable baselines skip comparison
    with an actionable warning rather than treating all existing violations as
    regressions.
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop ./pkg/gate/... -run 'TestBaseline|TestGate' -race -coverprofile=cover.out -v
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      The baseline artifact must be a versioned portable JSON document stored
      locally at .backstop/baseline.json and suitable for publishing through
      any artifact transport. It must include schema_version, git SHA,
      timestamp, backstop version, per-step violation counts with rule IDs, and
      the full raw violation set produced by gate evaluating steps before
      baseline comparison, waiver resolution, or ledger processing.
    traces:
      - BUNDLE-007:REQ-007

  - id: REQ-002
    text: >
      Gate violations must add stable baseline identity fields without breaking
      the existing gate/v1 JSON contract. Existing fields remain valid, while
      additive fields provide enough data to compute identity from rule, file,
      and a content hash or source-region hash of the violating region. Steps
      that cannot provide exact source regions may produce best-effort
      identities, but baseline comparison must not regress to rule+file counts.
    traces:
      - BUNDLE-007:REQ-011

  - id: REQ-003
    text: >
      Gate step 7, named baseline_comparison, must consume accumulated results
      from gate steps 1-6 without rerunning them. The gate orchestration may
      use an accumulated-results-aware step, a run context, or a post-processing
      phase, but the public step name remains baseline_comparison.
    traces:
      - BUNDLE-007:REQ-010

  - id: REQ-004
    text: >
      Baseline comparison must diff the current scoped violation identities
      against the cached full-project baseline. A current scoped violation
      whose identity is absent from the baseline is a new regression and fails
      step 7. A baseline violation absent from the current run is reported as
      fixed/progress only when the violation's file is inside the current gate
      scope; diff and --file runs must not imply fixes for files they did not
      evaluate.
    traces:
      - BUNDLE-007:REQ-003
      - BUNDLE-007:REQ-012

  - id: REQ-005
    text: >
      Gate output must report the baseline differential rather than absolute
      pre-existing violation counts. Human output must distinguish "0 new
      violations beyond baseline" from one or more new violations. JSON output
      must expose the baseline comparison step status and its new violations
      using the existing gate/v1 step structure plus additive diagnostic fields
      if needed.
    traces:
      - BUNDLE-007:REQ-003

  - id: REQ-006
    text: >
      When no baseline is available because CI has not published one, because
      artifact lookup fails and no cache exists, or because the project is on
      its first run, step 7 must report status skipped with an actionable
      warning. The gate must not auto-generate a local baseline and must not
      treat all current violations as new.
    traces:
      - BUNDLE-007:REQ-006
      - BUNDLE-007:REQ-014

  - id: REQ-007
    text: >
      Local gate runs must use .backstop/baseline.json with TTL-based
      freshness. If the cache is fresh, gate uses it without network access. If
      expired, gate checks GitHub Actions for a newer main-branch baseline
      artifact. If remote access fails, gate uses a stale cache with a warning.
      Only when neither fresh nor stale cache exists may gate skip comparison.
    traces:
      - BUNDLE-007:REQ-002
      - BUNDLE-007:REQ-014

  - id: REQ-008
    text: >
      backstop baseline pull must fetch the latest main-branch baseline
      artifact from GitHub Actions, write it to .backstop/baseline.json, and
      bypass TTL. Retrieval must define artifact naming and workflow/run
      selection semantics (including how the latest successful main-branch run
      is selected). Missing origin remote, missing repository resolution,
      missing authentication, missing artifacts, branch filtering mismatches,
      workflow/run selection misses, and offline network failures must produce
      actionable errors or warnings without corrupting an existing cache.
    traces:
      - BUNDLE-007:REQ-005
      - BUNDLE-007:REQ-014

  - id: REQ-009
    text: >
      Baseline TTL must be configurable in backstop.yml under
      enforcement.baseline_ttl and default to 15 minutes when omitted. The TTL
      controls automatic gate refresh only; backstop baseline pull always
      bypasses TTL.
    traces:
      - BUNDLE-007:REQ-002

  - id: REQ-010
    text: >
      CI baseline generation must run the full gate scope, equivalent to
      backstop gate --all, after every merge to main. The generated baseline
      must represent the full project, while local default diff mode and --file
      mode compare only current scoped violations against that full baseline.
    traces:
      - BUNDLE-007:REQ-001
      - BUNDLE-007:REQ-012

  - id: REQ-011
    text: >
      CI must publish the generated baseline JSON through GitHub Actions
      artifacts for v1. Generation of the JSON file and upload/publish must be
      separable so a future storage backend can replace GitHub Actions without
      changing baseline schema or comparison semantics.
    traces:
      - BUNDLE-007:REQ-001
      - BUNDLE-007:REQ-007

  - id: REQ-012
    text: >
      The baseline ratchet applies to developer changes: PR gates fail when
      changed/scoped code introduces new baseline-diff violations, and
      post-merge baselines reflect fixed violations by reducing the stored
      violation set. The v1 baseline subsystem does not allow developers or
      agents to generate, hand-edit, or commit baseline references as a way to
      hide regressions; baseline references are CI-generated and locally cached
      for comparison only.
    traces:
      - BUNDLE-007:REQ-004

  - id: REQ-013
    text: >
      Pack upgrades or other explicit rule-set changes are the only v1
      exception to the ratchet. A rule-set change may seed existing-code
      violations into the next full baseline, while new violations in changed
      code remain enforced immediately by the PR gate. This exception is
      narrowly scoped to explicit rule-set-change context and must not function
      as a general bypass for changed/scoped regressions.
    traces:
      - BUNDLE-007:REQ-008
      - BUNDLE-007:REQ-015

  - id: REQ-014
    text: >
      DIR-012 v1 baseline comparison must ignore waivers only because the
      waiver subsystem is not implemented. This must not encode baseline
      comparison as permanently pre-waiver; future waiver work must define how
      waived violations participate in baseline semantics rather than treating
      them as permanent regressions.
    traces:
      - BUNDLE-007:REQ-013

  - id: REQ-015
    text: >
      This spec supersedes SPEC-010 REQ-005. The baseline file is
      .backstop/baseline.json, not .backstop/baseline.yml, and matching uses
      stable violation identity rather than rule+file/count comparisons.
      SPEC-010's baseline text is historical only and not normative for
      shipped baseline behavior.
    traces:
      - BUNDLE-007:REQ-009

  - id: REQ-016
    text: >
      The implementation must ensure .backstop/baseline.json is not committed
      by adding or preserving an ignore rule for that path. The baseline is a
      local cache and CI artifact, not a source-controlled project artifact;
      ownership remains CI-generated immutable reference + local cache, not
      developer-authored repository state.
    traces:
      - BUNDLE-007:REQ-001
      - BUNDLE-007:REQ-002

supersedes:
  - spec: SPEC-010
    requirement: REQ-005
    reason: >
      BUNDLE-007 defines the real baseline subsystem. It replaces the deferred
      .backstop/baseline.yml rule+file/count placeholder with a CI-generated
      immutable .backstop/baseline.json reference model and stable violation
      identity matching.

claims:
  - id: CLM-001
    requirement: REQ-001
    subject: pkg/gate
    text: Baseline JSON is versioned, portable, and contains metadata, counts, and raw violations.
    tests:
      - TestBaseline_ArtifactJSONRoundTrip_Contract
  - id: CLM-002
    requirement: REQ-002
    subject: pkg/gate
    text: Gate violations expose additive baseline identity data while preserving gate/v1 fields.
    tests:
      - TestBaseline_ViolationJSON_AdditiveIdentityFieldsContract
      - TestBaseline_IdentityHash_DeterministicContract
  - id: CLM-003
    requirement: REQ-003
    text: Baseline comparison consumes accumulated step 1-6 results without rerunning those steps.
    tests:
      - TestGate_BaselineComparison_WiredAfterAccumulatedChecks
  - id: CLM-004
    requirement: REQ-004
    text: Baseline comparison fails the gate on current scoped identities absent from the full baseline (CLI orchestration).
    tests:
      - TestGate_BaselineRatchet_FailsNewAndAllowsReductions_Contract
  - id: CLM-005
    requirement: REQ-006
    text: Missing baseline skips comparison with a warning and does not generate a local baseline.
    tests:
      - TestGate_BaselineComparison_MissingBaselineSkipsWithReason
  - id: CLM-006
    requirement: REQ-007
    text: Gate uses fresh cache, refreshes expired cache, and falls back to stale cache when offline.
    tests:
      - TestGate_BaselineCacheLifecycle_FreshCacheNoNetwork_Contract
      - TestGate_BaselineCacheLifecycle_ExpiredRefreshAndOfflineFallback_Contract
  - id: CLM-007
    requirement: REQ-008
    text: backstop baseline pull bypasses TTL, uses defined artifact naming and workflow/run selection, and handles retrieval failures safely.
    tests:
      - TestBaselinePull_BypassesTTL_Contract
      - TestBaselinePull_ActionableFailureModes_Contract
      - TestBaselineCIContract_ArtifactNamingAndLatestMainSelectionSemantics
  - id: CLM-008
    requirement: REQ-005
    text: Gate output reports baseline differential and clearly distinguishes zero-vs-nonzero new violations in human and JSON output.
    tests:
      - TestGateOutput_BaselineDifferential_HumanMessaging
      - TestGateOutput_BaselineDifferential_JSONDiagnostics
  - id: CLM-009
    requirement: REQ-009
    text: enforcement.baseline_ttl defaults to 15 minutes when omitted and only governs automatic gate refresh.
    tests:
      - TestBaselineCLI_TTL_Default15Minutes_Contract
      - TestBaselineCLI_TTL_Override_Contract
      - TestBaselinePull_BypassesTTL_Contract
  - id: CLM-010
    requirement: REQ-011
    text: Baseline JSON generation is decoupled from upload so publication backends can be swapped without schema changes.
    tests:
      - TestBaselineCIContract_GenerationUsesAllScope
      - TestBaselineCIContract_GenerationAndPublicationAreSeparable
      - TestBaselineCIContract_ArtifactNamingAndLatestMainSelectionSemantics
  - id: CLM-011
    requirement: REQ-013
    text: Rule-set-change seeding is the only ratchet exception and does not permit changed-code regressions (CI/CLI contract).
    tests:
      - TestBaselineCIContract_RuleSetChangeExceptionOnlySeedsOnFullBaseline
      - TestBaselineCIContract_PullRequestGateKeepsChangedCodeEnforcement
  - id: CLM-012
    requirement: REQ-014
    text: V1 ignores waivers only temporarily and preserves a clear integration point for future waiver-aware baseline semantics.
    tests:
      - TestGate_Waiver_SkippedWhenNotImplemented
  - id: CLM-013
    requirement: REQ-016
    kind: absence
    text: .backstop/baseline.json remains untracked via gitignore policy and is treated as cache/artifact-only state.
    tests:
      - TestBaselinePathPresentInGitignore
  - id: CLM-014
    requirement: REQ-010
    text: CI generation uses full gate scope equivalent to backstop gate --all.
    tests:
      - TestBaselineCIContract_GenerationUsesAllScope
  - id: CLM-015
    requirement: REQ-012
    text: New changed-code violations fail the gate while fixed violations reduce later baselines.
    tests:
      - TestGate_BaselineRatchet_FailsNewAndAllowsReductions_Contract
  - id: CLM-016
    requirement: REQ-015
    kind: absence
    text: SPEC-010 baseline placeholder is superseded by JSON path and stable identity matching.
    tests:
      - TestSpec010BaselinePlaceholderSuperseded
  - id: CLM-017
    requirement: REQ-004
    subject: pkg/gate
    text: CompareBaseline logic fails a scoped run whose current identities are absent from the full baseline and does not allow a scoped run to bypass via rule-set-change context.
    tests:
      - TestBaseline_CompareBaseline_ScopedRunDisallowsRuleSetChangeBypass
  - id: CLM-018
    requirement: REQ-013
    subject: pkg/gate
    text: CompareBaseline logic seeds existing-code violations only on a full-scope rule-set change and disallows scoped-run bypass, enforcing rule-set-change seeding as the sole ratchet exception.
    tests:
      - TestBaseline_CompareBaseline_AllScopeAllowsRuleSetChangeSeeding
      - TestBaseline_CompareBaseline_ScopedRunDisallowsRuleSetChangeBypass

contracts:
  - file: pkg/gate/baseline.go
    provides:
      - name: BaselineArtifact
        kind: type
        signature: "type BaselineArtifact struct"
      - name: LoadBaseline
        kind: function
        signature: "func LoadBaseline(path string) (*BaselineArtifact, error)"
      - name: WriteBaseline
        kind: function
        signature: "func WriteBaseline(path string, artifact *BaselineArtifact) error"
      - name: CompareBaseline
        kind: function
        signature: "func CompareBaseline(current []Violation, baseline *BaselineArtifact, options BaselineCompareOptions) BaselineComparison"
    consumes:
      - source: pkg/gate/result.go
        name: Violation
        kind: type
      - source: pkg/gate/scope.go
        name: GateScope
        kind: type
  - file: pkg/gate/result.go
    provides:
      - name: Violation
        kind: type
        signature: "type Violation struct"
        notes: "Adds optional baseline identity fields without removing existing gate/v1 fields."
    consumes: []
  - file: cmd/backstop/gate.go
    provides:
      - name: buildGateSteps
        kind: function
        signature: "func buildGateSteps(projectRoot string, scope ...*gate.GateScope) []gate.StepFunc"
        notes: "Preserves step-7 public name baseline_comparison; accumulated baseline diff is computed in gate orchestration without rerunning prior checks."
    consumes:
      - source: pkg/gate/baseline.go
        name: CompareBaseline
        kind: function
  - file: cmd/backstop/baseline.go
    provides:
      - name: newBaselineCommand
        kind: function
        signature: "func newBaselineCommand() *cobra.Command"
      - name: runBaselinePull
        kind: function
        signature: "func runBaselinePull(cmd *cobra.Command, args []string) error"
    consumes:
      - source: pkg/gate/baseline.go
        name: WriteBaseline
        kind: function
---

# SPEC-019: Baseline — CI-Generated Immutable Violation Reference

## Overview

Gate step 7 is currently deferred, so a gate run reports all violations it
finds. That makes the command noisy: agents cannot tell whether they
introduced a regression or merely encountered violations that already exist on
main. The baseline subsystem turns the gate into a differential signal: compare
current scoped violations against the immutable reference from main, fail on
new violations, and treat reductions as progress.

The baseline is intentionally not a developer-owned file. CI generates it after
merge, publishes it as an artifact, and local commands consume a cached copy.
This avoids local baseline gaming, cross-machine inconsistencies, merge
conflicts, and stale committed baseline files.

## Requirements

See frontmatter requirements REQ-001 through REQ-016. They define the baseline
JSON schema, additive violation identity fields, accumulated-result comparison,
scoped differential semantics, local cache and TTL behavior, `backstop baseline
pull`, CI artifact generation and publication, ratchet semantics, rule-set
change seeding, waiver deferral, SPEC-010 supersession, and gitignore handling.

## Implementation

### 1. Baseline artifact model

Add a baseline artifact model in `pkg/gate` with versioned JSON encoding. The
artifact contains metadata, per-step counts, and raw violations from the gate's
evaluating steps before baseline comparison. Generation and publication must be
separate operations: one function writes portable JSON, while CI or a CLI
adapter uploads or downloads that JSON.

### 2. Stable violation identity

Extend `gate.Violation` with optional identity fields such as line/range,
fingerprint, and/or region hash. Existing JSON fields remain valid for gate/v1
consumers. Baseline matching uses rule + file + content/region hash where
available; best-effort identities are acceptable only for steps that cannot
produce exact source regions.

### 3. Gate orchestration and step 7

Replace the deferred baseline step with a comparison step that can see the
results from steps 1-6. The implementation may update gate orchestration to run
step 7 after collecting prior results or may introduce a run context shared by
steps, but it must not rerun earlier steps to compute the current violation set.

### 4. Cache, TTL, and explicit pull

Store the local cache at `.backstop/baseline.json`. Automatic gate refresh uses
`enforcement.baseline_ttl` from `backstop.yml`, defaulting to 15 minutes.
`backstop baseline pull` bypasses TTL and fetches the latest main-branch GitHub
Actions artifact. Remote lookup failures use a stale cache when present and
otherwise skip comparison with an actionable warning.

### 5. CI generation and publication

Provide CI-facing generation support that runs the full gate scope equivalent to
`backstop gate --all`, writes the portable baseline JSON, and publishes it via
GitHub Actions artifacts for v1. Artifact lookup must define repository
resolution from git remotes, branch filtering to main, artifact naming,
workflow/run selection for the latest successful main run, and failure behavior
for missing auth, missing artifacts, selection misses, and offline access.

### 6. Ratchet and rule-set changes

Developer changes ratchet downward: changed-code violations that are not in the
baseline fail the gate, while fixed violations reduce the next post-merge
baseline. Explicit pack or rule-set changes may seed existing-code violations
into the next full baseline, but that exception does not permit new
changed-code regressions.

## Verification

1. A baseline JSON file round-trips with schema version, SHA, timestamp,
   backstop version, per-step counts, and violations.
2. Existing gate/v1 JSON consumers still see rule, file, message, severity, and
   source_pack fields after violation identity fields are added.
3. Step 7 compares against accumulated step 1-6 violations without rerunning
   those steps.
4. Current scoped violations absent from the baseline fail the gate as new
   regressions; pre-existing baseline matches do not fail.
5. Diff and `--file` runs never claim fixed violations for files outside the
   current scope.
6. Missing baseline skips step 7 with a warning and does not create
   `.backstop/baseline.json`.
7. Gate output presents baseline differential semantics in both UX surfaces:
   human output distinguishes 0-vs-N new violations and gate/v1 JSON includes
   baseline comparison step status plus additive new-violation diagnostics.
8. Fresh cache avoids network access; expired cache refreshes; offline expired
   cache is used with a warning.
9. `backstop baseline pull` bypasses TTL; uses defined artifact naming,
   workflow/run selection, and main-branch filtering; and safely handles
   missing remote, missing auth, missing artifacts, selection misses, and
   network failure without corrupting an existing cache.
10. `enforcement.baseline_ttl` defaults to 15 minutes when omitted, can be
    overridden, and applies only to automatic gate refresh.
11. CI generation uses full scope equivalent to `backstop gate --all` and
    baseline publication keeps JSON generation separable from artifact upload.
12. Rule-set-change seeding is enforced as the only ratchet exception for
    existing-code violations, while changed-code regressions still fail.
13. Waivers are ignored only in v1 and baseline comparison preserves a future
    waiver-aware integration point rather than baking in permanent pre-waiver
    semantics.
14. SPEC-010 REQ-005 supersession is enforced: `.backstop/baseline.json` plus
    stable identity matching replaces `.backstop/baseline.yml` rule+file counts.
15. `.backstop/baseline.json` is covered by gitignore and remains untracked.

## Out of Scope

- Local generation or hand-editing of baselines by developers or agents.
- Committing baseline artifacts to the repository.
- Branch-specific baselines; v1 compares all branches against main.
- Permanent storage backends other than GitHub Actions artifacts.
- Ledger coupling; ledger state must not affect baseline pass/fail semantics.
- Waiver-aware baseline semantics beyond preserving a future integration point.

## Sharp Edges / Open Questions

- Some existing gate steps may not yet expose enough source-region data for a
  robust content hash. Those steps may need incremental best-effort identities,
  but the implementation must avoid reintroducing rule+file/count matching as
  the baseline model.

## Version History

- **1.0.1** (2026-07-07) — Marked CLM-013 and CLM-016 `kind: absence`. Both mandated
  tests are structural assertions over non-code files —
  `TestBaselinePathPresentInGitignore` reads the repo `.gitignore` and asserts the
  baseline path is present; `TestSpec010BaselinePlaceholderSuperseded` reads the
  SPEC-010 and SPEC-019 markdown files and asserts supersession — that by design
  reference no target package. This is the honest model, previously hidden only by
  the vacuous `cmd/`-path skip in the substantiveness `noTarget` guard that ISSUE-047
  removed. `kind: absence` exempts them from the `noTarget` substantiveness join per
  the claims schema.
