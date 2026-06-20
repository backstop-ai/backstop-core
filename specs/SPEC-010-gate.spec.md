---
title: "SPEC-010: Gate — Full Reconciliation Command"
number: SPEC-010
created: "2026-04-04"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Implement the backstop gate command that runs the full verification kill
    chain (ADR-0010). Gate is the superset command — it orchestrates artifact
    validation (delegating to backstop artifact validate), code check
    (delegating to backstop code check --all), test verification (mandated
    test names exist as functions), test substantiveness (tests are not hollow
    — contain assertions and call target package), coverage threshold
    verification (coverage meets spec-declared threshold), contract signature
    verification (declared symbols exist with matching signatures), baseline
    comparison, waiver resolution, and ledger integrity verification. Steps
    3-6 use grep and Go AST parsing for mechanical verification — no full
    semantic analysis. Baseline, waivers, and ledger are deferred (reported
    as skipped). Gate produces a unified result in JSON or human output mode.
    The JSON output is the contract consumed by the GitHub Actions gate action
    (ADR-0009). Exit codes follow the CLI contract: 0 (all green), 1
    (failures found), 2 (config error).
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/... ./pkg/gate/... -run "TestGate" -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      The backstop gate command must run nine verification steps in the
      following fixed order: (1) artifact validation, (2) code check,
      (3) test verification, (4) test substantiveness, (5) coverage threshold
      verification, (6) contract signature verification, (7) baseline
      comparison, (8) waiver resolution, (9) ledger integrity verification.
      All nine steps execute regardless of earlier step failures — gate
      collects all results before producing its final output. No step is
      short-circuited by a preceding failure. The one exception: if a
      delegated step (artifact validate or code check) returns a config
      error (exit 2), gate halts remaining steps per REQ-009.
    supports: cli:REQ-006

  - id: REQ-002
    text: >
      Step 1 (artifact validation) must delegate to the same logic as
      backstop artifact validate --all. It validates every artifact in the
      project against embedded schemas. Violations from this step must appear
      in the gate output under a distinct "artifact_validation" section.
    supports: cli:REQ-006

  - id: REQ-003
    text: >
      Step 2 (code check) must delegate to the same logic as backstop code
      check --all. It runs lint, build, test, and semgrep against the full
      codebase. Violations from this step must appear in the gate output
      under a distinct "code_check" section.
    supports: cli:REQ-006

  - id: REQ-004
    text: >
      Step 3 (test verification) must verify that every mandated test name
      from every spec claim exists as an actual test function in the codebase.
      For each spec in the project, extract all claims and their mandated test
      names. For each test name, grep the test files for a function with that
      exact name. A missing test function is a failure. This is a mechanical
      check — function name exists or it doesn't.
    supports: cli:REQ-006

  - id: REQ-015
    text: >
      Step 4 (test substantiveness) must verify that each mandated test
      function is not hollow. For each test function found in step 3, perform
      basic checks: (a) the function body calls at least one function from the
      package under test (not just test helpers), (b) the function body contains
      at least one assertion (t.Fatal, t.Error, t.Errorf, or a comparison that
      would fail). A test function that exists but contains no assertions or
      doesn't call the target package is a failure. This uses Go AST parsing
      or grep-level heuristics — not full semantic analysis.
    supports: cli:REQ-006

  - id: REQ-016
    text: >
      Step 5 (coverage threshold) must run the test suite with coverage
      profiling and compare the result against the threshold declared in the
      spec's verification.coverage_threshold field. Coverage below threshold
      is a failure. The test command from the spec's verification.test_command
      field is used with -coverprofile appended if not already present.
      Coverage is extracted from the go test stdout summary line (format:
      "coverage: NN.N% of statements"). If the summary line is not present
      in the test output, the step reports status "fail" with reason "coverage
      summary line not found in test output". If the test command fails to
      execute (command not found, non-zero exit unrelated to coverage,
      timeout), the step reports status "fail" with the error details. This
      is a step failure (exit 1), not a config error (exit 2).
    supports: cli:REQ-006

  - id: REQ-017
    text: >
      Step 6 (contract signature verification) must verify that every function,
      type, and interface declared in spec contracts actually exists in the
      codebase with the declared signature. For each contract entry, grep or
      AST-parse the declared file for the declared symbol with the declared
      signature. A missing symbol or mismatched signature is a failure. This
      is a mechanical check — the declaration exists or it doesn't.
    supports: cli:REQ-006

  - id: REQ-005
    superseded_by: SPEC-019 REQ-015
    text: >
      Superseded by SPEC-019 REQ-015. This requirement is retained only as a
      historical placeholder for traceability. Baseline behavior is now defined
      by SPEC-019: the reference path is .backstop/baseline.json (not
      .backstop/baseline.yml), comparisons use stable violation identity (not
      rule+file counts), and baseline ownership is CI-generated cache/artifact
      state rather than developer-authored repository content.
    supports: cli:REQ-006

  - id: REQ-006
    text: >
      Step 8 (waiver resolution) must check active waivers and suppress
      matching violations from preceding steps. An active waiver is one
      whose scope (rule+file or rule+package) matches a reported violation
      and whose expiry date has not passed. Suppressed violations must be
      removed from the step's violation list and recorded as waived in the
      output. When the waiver subsystem is not yet implemented, this step
      must report status "skipped" with reason "waivers not implemented."
    supports: cli:REQ-006

  - id: REQ-007
    text: >
      Step 9 (ledger integrity) must verify the provenance ledger hash chain.
      If the ledger does not exist, the step must report status "skipped"
      with reason "no ledger found." If the ledger exists, the step must
      verify that every entry's hash is correct and the chain is unbroken.
      A broken hash chain or tampered entry is a failure. When the ledger
      subsystem is not yet implemented, this step must report status "skipped"
      with reason "ledger not implemented."
    supports: cli:REQ-006

  - id: REQ-008
    text: >
      The gate command must produce structured JSON output when the --json
      flag is set. The JSON output must include: a schema_version field
      (string), a pass boolean (true only when all executed steps pass and
      no step failed), a steps array where each element contains step_name
      (string), status (one of "pass", "fail", "skipped"), violations
      (array of violation objects, empty for pass/skipped), and reason
      (string, present only when status is "skipped"). The JSON output must
      also include summary fields: total_violations (integer count of all
      violations across all steps), steps_passed (integer), steps_failed
      (integer), steps_skipped (integer). When --json is not set, the
      command must produce human-readable formatted text to stdout.
    supports: cli:REQ-007

  - id: REQ-009
    text: >
      Exit codes must follow the CLI contract: 0 when all executed steps
      pass (skipped steps do not prevent exit code 0), 1 when any step
      has status "fail", 2 when a configuration error occurs (backstop.yml
      missing or invalid, schema loading failure, embedded schema cohort
      error). Exit code 2 must take precedence over exit code 1 — if a
      config error is detected, the command must not report partial gate
      results. Skipped steps alone must not produce exit code 1. Config
      errors from delegated steps (artifact validate or code check returning
      exit 2) must propagate as gate exit code 2. Gate must halt remaining
      steps when a delegated step returns a config error.
    supports: cli:REQ-006

  - id: REQ-010
    text: >
      The gate JSON output must include a schema_version field for
      independent contract evolution. The initial schema version is
      "gate/v1". Changes to the JSON output structure follow D-070
      evolution rules: additive fields with sensible defaults, deprecated
      fields emit warnings, breaking removals only on major version bumps.
    supports: cli:REQ-007

  - id: REQ-011
    text: >
      The nine gate steps must be identified by the following canonical
      step names in the output: "artifact_validation", "code_check",
      "test_verification", "test_substantiveness", "coverage_threshold",
      "contract_signature", "baseline_comparison", "waiver_resolution",
      "ledger_integrity". These names are part of the JSON output contract
      and must not change without a schema version bump.
    supports: cli:REQ-006

  - id: REQ-012
    superseded_by: SPEC-018 REQ-008
    text: >
      Superseded by SPEC-018 REQ-008. The gate command originally accepted no
      scope flags and always ran against the full project. Gate now accepts
      --all and --file; the original comprehensive behavior is preserved via
      --all.
    supports: cli:REQ-006

  - id: REQ-013
    text: >
      The human-readable output mode must display a summary table showing
      each step name, its status (pass/fail/skipped), and the count of
      violations for that step. Steps with status "skipped" must display
      their reason. The summary must end with an overall pass/fail verdict.
      The human output must respect the NO_COLOR environment variable.
    supports: cli:REQ-007

  - id: REQ-014
    text: >
      Gate must load backstop.yml before executing any steps. If
      backstop.yml is not found or fails validation, gate must exit with
      code 2 immediately. Gate reuses the config loading infrastructure
      from the CLI foundation (SPEC-005 REQ-003).
    supports: cli:REQ-009

claims:
  # REQ-001: Fixed order, all steps execute
  - id: CLM-001
    requirement: REQ-001
    text: Gate runs all nine steps in the specified order
    tests:
      - TestGate_AllNineStepsExecuteInOrder

  - id: CLM-002
    requirement: REQ-001
    text: Gate continues executing remaining steps after a step fails
    tests:
      - TestGate_ContinuesAfterStepFailure

  # REQ-002: Artifact validation delegation
  - id: CLM-003
    requirement: REQ-002
    text: Gate step 1 runs artifact validation and reports violations under artifact_validation section
    tests:
      - TestGate_ArtifactValidation_ReportsViolations

  - id: CLM-004
    requirement: REQ-002
    text: Gate step 1 reports pass when all artifacts are valid
    tests:
      - TestGate_ArtifactValidation_PassWhenValid

  # REQ-003: Code check delegation
  - id: CLM-005
    requirement: REQ-003
    text: Gate step 2 runs code check against full codebase and reports violations under code_check section
    tests:
      - TestGate_CodeCheck_ReportsViolations

  - id: CLM-006
    requirement: REQ-003
    text: Gate step 2 reports pass when code check finds no violations
    tests:
      - TestGate_CodeCheck_PassWhenClean

  # REQ-004: Test verification — mandated test names exist
  - id: CLM-007
    requirement: REQ-004
    text: Gate finds a mandated test function that exists and reports pass
    tests:
      - TestGate_TestVerification_MandatedTestExists

  - id: CLM-008
    requirement: REQ-004
    text: Gate detects a missing mandated test function and reports failure
    tests:
      - TestGate_TestVerification_MandatedTestMissing

  - id: CLM-009
    requirement: REQ-004
    text: Gate checks all specs in the project and collects all mandated test names
    tests:
      - TestGate_TestVerification_CollectsAllSpecClaims

  # REQ-015: Test substantiveness — not hollow
  - id: CLM-010
    requirement: REQ-015
    text: Gate detects a test function that contains assertions and calls the target package
    tests:
      - TestGate_TestSubstantiveness_SubstantiveTestPasses

  - id: CLM-011
    requirement: REQ-015
    text: Gate detects a hollow test function with no assertions
    tests:
      - TestGate_TestSubstantiveness_HollowTestFails

  - id: CLM-047
    requirement: REQ-015
    text: Gate detects a test function that never calls the package under test
    tests:
      - TestGate_TestSubstantiveness_NoTargetCallFails

  # REQ-016: Coverage threshold
  - id: CLM-048
    requirement: REQ-016
    text: Gate passes when coverage meets the spec threshold
    tests:
      - TestGate_CoverageThreshold_MeetsThreshold

  - id: CLM-049
    requirement: REQ-016
    text: Gate fails when coverage is below the spec threshold
    tests:
      - TestGate_CoverageThreshold_BelowThreshold

  - id: CLM-050
    requirement: REQ-016
    text: Gate owns coverage scheduling rather than executing spec test_command as a plan
    tests:
      - TestGate_CoverageThreshold_IgnoresSpecTestCommandForScheduling

  - id: CLM-057
    requirement: REQ-016
    text: Gate fails when test command fails to execute (command not found) with error details
    tests:
      - TestGate_CoverageThreshold_TestCommandNotFound

  - id: CLM-058
    requirement: REQ-016
    text: Gate fails when coverage summary line is not present in test output
    tests:
      - TestGate_CoverageThreshold_NoCoverageSummaryLine

  - id: CLM-059
    requirement: REQ-016
    text: Gate extracts coverage percentage from go test stdout summary line format
    tests:
      - TestGate_CoverageThreshold_ParsesCoverageSummaryLine

  # REQ-017: Contract signature verification
  - id: CLM-051
    requirement: REQ-017
    text: Gate passes when a declared contract function exists with matching signature
    tests:
      - TestGate_ContractSignature_MatchingSignaturePasses

  - id: CLM-052
    requirement: REQ-017
    text: Gate fails when a declared contract function is missing from the file
    tests:
      - TestGate_ContractSignature_MissingFunctionFails

  - id: CLM-053
    requirement: REQ-017
    text: Gate fails when a declared contract function exists but signature differs
    tests:
      - TestGate_ContractSignature_WrongSignatureFails

  - id: CLM-054
    requirement: REQ-017
    text: Gate verifies contract types and interfaces, not just functions
    tests:
      - TestGate_ContractSignature_TypeAndInterfaceVerified

  # All steps present in output
  - id: CLM-055
    requirement: REQ-001
    text: All nine steps appear in gate output regardless of pass/fail/skip status
    tests:
      - TestGate_AllNineStepsAppearInOutput

  # REQ-005: Baseline comparison
  - id: CLM-012
    requirement: REQ-005
    text: Baseline comparison reports skipped when no baseline file exists
    tests:
      - TestGate_Baseline_SkippedWhenNoFile

  - id: CLM-013
    requirement: REQ-005
    text: Baseline comparison reports skipped when baseline subsystem is not implemented
    tests:
      - TestGate_Baseline_SkippedWhenNotImplemented

  - id: CLM-014
    requirement: REQ-005
    text: Baseline comparison reports fail when a new violation exceeds the baseline count
    tests:
      - TestGate_Baseline_FailOnNewViolation

  - id: CLM-015
    requirement: REQ-005
    text: Baseline comparison reports pass when all violations are at or below baseline counts
    tests:
      - TestGate_Baseline_PassWhenClean

  # REQ-006: Waiver resolution
  - id: CLM-016
    requirement: REQ-006
    text: Waiver resolution reports skipped when waiver subsystem is not implemented
    tests:
      - TestGate_Waiver_SkippedWhenNotImplemented

  # REQ-007: Ledger integrity
  - id: CLM-017
    requirement: REQ-007
    text: Ledger integrity reports skipped when no ledger exists
    tests:
      - TestGate_Ledger_SkippedWhenNoLedger

  - id: CLM-018
    requirement: REQ-007
    text: Ledger integrity reports skipped when ledger subsystem is not implemented
    tests:
      - TestGate_Ledger_SkippedWhenNotImplemented

  - id: CLM-019
    requirement: REQ-007
    text: Ledger integrity reports fail when hash chain is broken
    tests:
      - TestGate_Ledger_FailOnBrokenChain

  - id: CLM-020
    requirement: REQ-007
    text: Ledger integrity reports pass when hash chain is intact
    tests:
      - TestGate_Ledger_PassWhenIntact

  # REQ-008: JSON output structure
  - id: CLM-021
    requirement: REQ-008
    text: JSON output includes schema_version, pass boolean, and steps array
    tests:
      - TestGate_JSONOutput_StructureComplete

  - id: CLM-022
    requirement: REQ-008
    text: JSON output pass is true only when all executed steps pass
    tests:
      - TestGate_JSONOutput_PassTrueWhenAllGreen

  - id: CLM-023
    requirement: REQ-008
    text: JSON output pass is false when any step has status fail
    tests:
      - TestGate_JSONOutput_PassFalseWhenAnyFail

  - id: CLM-024
    requirement: REQ-008
    text: JSON output pass is true when some steps pass and remaining steps are skipped
    tests:
      - TestGate_JSONOutput_PassTrueWithSkippedSteps

  - id: CLM-025
    requirement: REQ-008
    text: JSON output includes summary counts (total_violations, steps_passed, steps_failed, steps_skipped)
    tests:
      - TestGate_JSONOutput_SummaryCounts

  - id: CLM-056
    requirement: REQ-008
    text: JSON output pass is true when all nine steps are skipped (no executed step failed)
    tests:
      - TestGate_JSONOutput_PassTrueWhenAllSkipped

  - id: CLM-026
    requirement: REQ-008
    text: Each step in the JSON steps array includes step_name, status, and violations fields
    tests:
      - TestGate_JSONOutput_StepFieldsPresent

  - id: CLM-027
    requirement: REQ-008
    text: Skipped steps include reason field in JSON output
    tests:
      - TestGate_JSONOutput_SkippedStepHasReason

  - id: CLM-028
    requirement: REQ-008
    text: Human-readable output is produced when --json flag is not set
    tests:
      - TestGate_HumanOutput_ProducedByDefault

  # REQ-009: Exit codes
  - id: CLM-029
    requirement: REQ-009
    text: Exit code 0 when all executed steps pass
    tests:
      - TestGate_ExitCode0_AllPass

  - id: CLM-030
    requirement: REQ-009
    text: Exit code 0 when all steps either pass or are skipped
    tests:
      - TestGate_ExitCode0_PassAndSkipped

  - id: CLM-031
    requirement: REQ-009
    text: Exit code 1 when any step fails
    tests:
      - TestGate_ExitCode1_StepFailed

  - id: CLM-032
    requirement: REQ-009
    text: Exit code 2 when config error occurs
    tests:
      - TestGate_ExitCode2_ConfigError

  - id: CLM-033
    requirement: REQ-009
    text: Exit code 2 takes precedence over exit code 1 when both config error and step failure occur
    tests:
      - TestGate_ExitCode2_PrecedenceOverExitCode1

  - id: CLM-034
    requirement: REQ-009
    text: Config error from delegated artifact validate step (exit 2) propagates as gate exit code 2 and halts remaining steps
    tests:
      - TestGate_ExitCode2_DelegatedArtifactValidateConfigError

  - id: CLM-035
    requirement: REQ-009
    text: Config error from delegated code check step (exit 2) propagates as gate exit code 2 and halts remaining steps
    tests:
      - TestGate_ExitCode2_DelegatedCodeCheckConfigError

  # REQ-010: Schema version in output
  - id: CLM-036
    requirement: REQ-010
    text: JSON output schema_version is "gate/v1"
    tests:
      - TestGate_JSONOutput_SchemaVersionGateV1

  # REQ-011: Canonical step names
  - id: CLM-037
    requirement: REQ-011
    text: All nine canonical step names appear in the output steps array
    tests:
      - TestGate_CanonicalStepNames_AllPresent

  - id: CLM-038
    requirement: REQ-011
    text: Step names in output exactly match the canonical names
    tests:
      - TestGate_CanonicalStepNames_ExactMatch

  # REQ-012: No scope flags
  - id: CLM-039
    requirement: REQ-012
    text: Gate defaults to diff scope when no explicit scope flag is provided
    tests:
      - TestGate_DefaultsToDiffMode

  - id: CLM-040
    requirement: REQ-012
    text: Gate accepts explicit full and file scopes while rejecting conflicting scope flags
    tests:
      - TestGate_AllFlagUsesFullSweep
      - TestGate_FileFlagScopesExplicitFiles
      - TestGate_AllAndFileMutuallyExclusive

  # REQ-013: Human-readable output
  - id: CLM-041
    requirement: REQ-013
    text: Human output displays summary table with step name, status, and violation count
    tests:
      - TestGate_HumanOutput_SummaryTable

  - id: CLM-042
    requirement: REQ-013
    text: Human output shows reason for skipped steps
    tests:
      - TestGate_HumanOutput_SkippedStepReason

  - id: CLM-043
    requirement: REQ-013
    text: Human output ends with overall pass/fail verdict
    tests:
      - TestGate_HumanOutput_OverallVerdict

  - id: CLM-044
    requirement: REQ-013
    text: Human output respects NO_COLOR environment variable
    tests:
      - TestGate_HumanOutput_NoColorEnvVar

  # REQ-014: Config loading
  - id: CLM-045
    requirement: REQ-014
    text: Gate exits with code 2 when backstop.yml is not found
    tests:
      - TestGate_ConfigMissing_ExitCode2

  - id: CLM-046
    requirement: REQ-014
    text: Gate exits with code 2 when backstop.yml fails validation
    tests:
      - TestGate_ConfigInvalid_ExitCode2

contracts:
  - file: cmd/backstop/gate.go
    provides:
      - name: gateCmd
        kind: variable
        signature: "var gateCmd *cobra.Command"
        notes: "Top-level gate command registered on root"
      - name: GateResult
        kind: type
        signature: "type GateResult gate.GateResult"
        notes: "Re-export of pkg/gate.GateResult for the contract; holds the unified gate output including all step results"
      - name: StepResult
        kind: type
        signature: "type StepResult gate.StepResult"
        notes: "Re-export of pkg/gate.StepResult for the contract; per-step result with name, status, violations, and optional reason"
      - name: runGate
        kind: function
        signature: "func runGate(cmd *cobra.Command, args []string) error"
        notes: "Cobra RunE handler that orchestrates all nine steps"
    consumes:
      - source: cmd/backstop
        name: artifactValidateAll
        kind: function
      - source: cmd/backstop
        name: codeCheckAll
        kind: function
      - source: cmd/backstop
        name: formatOutput
        kind: function
      - source: pkg/validate
        name: ValidationResult
        kind: type
      - source: pkg/validate
        name: Violation
        kind: type
---

# SPEC-010: Gate — Full Reconciliation Command

## Overview

The `backstop gate` command is the full reconciliation gate — the mechanical
embodiment of "if it's green, it ships" (D-059, ADR-0012). Gate runs the
complete verification kill chain (ADR-0010) and produces a unified pass/fail
result that the GitHub Actions gate action consumes.

Gate is the superset of `backstop artifact validate` and `backstop code check`.
Where those commands handle artifact schema conformance and implementation
validation respectively, gate orchestrates both plus the verifier steps:
test verification, test substantiveness, coverage thresholds, contract
signatures, baseline comparison, waiver resolution, and ledger integrity.

**Note on pack rule enforcement:** The bundle's kill chain includes "pack rule
enforcement (semgrep against full scope)" as a distinct concern. In this spec,
pack rule enforcement is deliberately subsumed by the code_check step (step 2),
which runs lint, build, test, AND semgrep against the full codebase. This is a
deliberate merge — semgrep is one of the tools code_check orchestrates — not an
omission.

Steps 1-6 are implemented: artifact validation and code check delegate to
existing commands, while steps 3-6 use grep and Go AST parsing for mechanical
verification (function name existence, assertion presence, coverage parsing,
signature matching). Steps 7-9 (baseline comparison, waiver resolution, ledger
integrity) are deferred and report as "skipped" with an explicit reason. This
ensures the gate output contract is stable from day one — consumers always see
all nine steps, and the transition from "skipped" to "implemented" is additive,
not structural.

## Requirements

Requirements are defined in frontmatter. The gate command has 17 requirements
covering step orchestration, delegation, mechanical verification (steps 3-6),
deferred steps, output format, exit codes, and configuration loading.

### Kill Chain Steps

Gate runs nine steps in fixed order. Every step appears in the output regardless
of whether it executed, failed, or was skipped.

| # | Step Name | Source | Initial Status |
|---|-----------|--------|----------------|
| 1 | `artifact_validation` | Delegates to artifact validate --all logic | Implemented |
| 2 | `code_check` | Delegates to code check --all logic (includes semgrep) | Implemented |
| 3 | `test_verification` | Mandated test names from spec claims exist as functions (grep for exact function names) | Implemented |
| 4 | `test_substantiveness` | Test functions are not hollow — contain assertions and call target package (Go AST parsing or grep heuristics) | Implemented |
| 5 | `coverage_threshold` | Test coverage meets spec-declared threshold (run test_command with -coverprofile, parse stdout summary line) | Implemented |
| 6 | `contract_signature` | Spec contract declarations match actual code (grep/AST for declared symbols in declared files) | Implemented |
| 7 | `baseline_comparison` | Compares violations against recorded baseline | Skipped (baseline not implemented) |
| 8 | `waiver_resolution` | Suppresses violations matched by active waivers | Skipped (waivers not implemented) |
| 9 | `ledger_integrity` | Verifies provenance ledger hash chain | Skipped (ledger not implemented) |

### Exit Code Semantics

| Code | Meaning | When |
|------|---------|------|
| 0 | All green | All executed steps pass; skipped steps do not prevent 0 |
| 1 | Failures found | Any step has status "fail" |
| 2 | Config error | backstop.yml missing/invalid, schema load failure |

Exit code 2 takes precedence over 1. Config errors from delegated steps
(artifact validate or code check returning exit 2) propagate as gate exit
code 2 and halt remaining steps. Skipped steps alone never produce exit code 1.

### JSON Output Contract

The JSON output (--json) is the API surface consumed by the GitHub Actions gate
action. Its schema version is "gate/v1" and evolves independently of the CLI
binary version per D-070.

### No Scoping

Gate always runs against the full project. It accepts no scope flags (no --diff,
--file, --spec, --plan, --all). This is intentional — gate is the comprehensive
reconciliation point, not a targeted check.

## Implementation

### Gate Command Registration

The gate command is registered as a top-level Cobra command (not under a
namespace). It reuses the CLI foundation's config loading (SPEC-005) and
output formatting infrastructure.

### Step Orchestration

The `runGate` function orchestrates all nine steps sequentially:

1. **Config loading** — Load and validate backstop.yml. Exit code 2 on failure.
2. **Step execution loop** — For each of the nine steps:
   - If the step's implementation is available, execute it and collect results.
   - If the step's implementation is not available, produce a skipped result with reason.
   - If a delegated step returns a config error (exit 2), halt remaining steps immediately.
   - Append the step result to the gate result.
3. **Result aggregation** — Compute summary counts (passed, failed, skipped, total violations).
4. **Output formatting** — Format as JSON or human text based on --json flag.
5. **Exit code determination** — 2 if config error (including delegated config errors), 1 if any failure, 0 otherwise.

### Step Implementation Pattern

Each step is implemented as a function with a common signature that returns a
`StepResult`. Steps that delegate to existing commands (artifact validate, code
check) call the same library functions those commands use — not the CLI commands
themselves. This avoids subprocess overhead and keeps the gate as a single
process.

Steps that are not yet implemented return a StepResult with status "skipped"
and a reason string. The step function signature allows future implementations
to be plugged in without changing the orchestration loop.

### GateResult Structure

```go
type GateResult struct {
    SchemaVersion   string       `json:"schema_version"`
    Pass            bool         `json:"pass"`
    TotalViolations int          `json:"total_violations"`
    StepsPassed     int          `json:"steps_passed"`
    StepsFailed     int          `json:"steps_failed"`
    StepsSkipped    int          `json:"steps_skipped"`
    Steps           []StepResult `json:"steps"`
}

type StepResult struct {
    StepName   string      `json:"step_name"`
    Status     string      `json:"status"` // "pass", "fail", "skipped"
    Violations []Violation `json:"violations"`
    Reason     string      `json:"reason,omitempty"`
}
```

### Deferred Step Activation

When the verifier, baseline, or ledger subsystems are implemented, the
corresponding step functions are replaced with real implementations. The
orchestration loop, output structure, and JSON contract do not change. This
is the key design constraint — the gate output contract is stable before
all steps are implemented.

## Verification

Verification is defined in frontmatter. Integration-level verification at
80% coverage threshold. Claims are defined in frontmatter.

## Sharp Edges

- **Skipped steps mask missing verification.** Gate reports "all green" (exit
  code 0) when steps 3-9 are skipped because the verifier, baseline, waiver,
  and ledger subsystems don't exist yet. A consumer of the gate output (e.g., a GitHub Actions
  gate action) must check the steps_skipped count and decide whether skipped
  steps are acceptable for merge. The gate itself does not make this policy
  decision — it reports the facts. A naive gate action that only checks the
  pass boolean will auto-merge code that has never been verified.

- **Delegated-step config errors are gate config errors.** When artifact
  validation or code check returns exit 2 (config error — e.g., schema
  loading failure), gate must propagate this as gate-level exit code 2 and
  halt remaining steps. This is not a step failure (exit 1). The rule is
  explicit: delegated exit 2 becomes gate exit 2. Gate does not attempt to
  continue executing subsequent steps after a config error from a delegated
  step.

- **Step ordering is load-bearing for future steps.** Steps 3-6 depend on the
  code check (step 2) having already run — test verification needs test results,
  coverage needs test execution data. The fixed ordering is not arbitrary; it
  reflects data dependencies. If a future optimization attempts to parallelize
  steps, it must respect these dependencies.

- **Baseline semantics moved to SPEC-019.** Historical count-based placeholder
  behavior in this spec is superseded. The shipped model uses immutable
  CI-generated baseline references, stable identity matching, scoped ratchet
  enforcement, and explicit prohibition on developer-generated or committed
  baseline files.

- **Gate is the only command that should be the CI gate.** Teams might be
  tempted to use `backstop artifact validate` or `backstop code check` as
  their CI gate. These are incomplete — they check one dimension each. Only
  `backstop gate` runs the full kill chain. The gate action should always
  invoke `backstop gate`, not individual commands.

- **JSON output contract evolution.** The gate JSON output is consumed by
  external systems (GitHub Actions, CI pipelines). Adding fields is safe
  under D-070, but removing or renaming fields (like step names) is a
  breaking change. The canonical step names in REQ-010 are part of the
  contract and must be treated as immutable identifiers.

## Review Questions

1. Does the implementation correctly propagate config errors from delegated
   steps (artifact validate, code check) as gate-level exit code 2, and does
   it halt remaining steps immediately rather than continuing execution?

2. When a new verifier step is activated (e.g., test verification becomes
   implemented), does the gate output structure remain backward compatible
   with consumers that were parsing the "skipped" status?

3. Does the human output correctly handle the case where all nine steps are
   skipped (e.g., no verifier, no baseline, no waivers, no ledger, and both
   artifact validate and code check have config errors)?

4. Is the GateResult.Pass field computed correctly in all combinations of
   pass, fail, and skipped statuses? Specifically: pass+skipped=true,
   fail+skipped=false, all-skipped=true?

5. Does the implementation prevent scope flags (--diff, --file, etc.) from
   being accepted, or does it silently ignore them?

6. When the waiver subsystem is activated, does the implementation correctly
   handle expired waivers (not suppress) vs active waivers (suppress)?

## References

- Bundle: cli (spec seed 6 — backstop gate)
- ADR-0010: Verification Kill Chain — the nine-step chain gate implements
- ADR-0012: Review Model — "if it's green, it ships" (D-059)
- ADR-0009: CI/CD Pipeline — the GitHub Actions gate action that consumes gate JSON
- D-070: Schema evolution rules for the JSON output contract
- SPEC-005: CLI Foundation — config loading, output formatting, exit codes
- SPEC-006: Artifact Validate — step 1 delegation target
- SPEC-008: Code Check — step 2 delegation target
- SPEC-019: Baseline — CI-generated immutable violation reference (supersedes REQ-005)
