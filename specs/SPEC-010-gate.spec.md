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
    (delegating to backstop code check --all), test verification, test
    substantiveness, coverage threshold verification, contract signature
    verification, baseline comparison, and ledger integrity verification.
    Gate produces a unified result in JSON or human output mode. The JSON
    output is the contract consumed by the GitHub Actions gate action
    (ADR-0009). Exit codes follow the CLI contract: 0 (all green), 1
    (failures found), 2 (config error). Steps that depend on the verifier
    (which does not yet exist) are reported as gaps in the output rather
    than silently skipped.
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/... -run TestGate -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      The backstop gate command must run eight verification steps in the
      following fixed order: (1) artifact validation, (2) code check,
      (3) test verification, (4) test substantiveness, (5) coverage threshold
      verification, (6) contract signature verification, (7) baseline
      comparison, (8) ledger integrity verification. All eight steps must
      execute regardless of earlier step failures — gate collects all results
      before producing its final output. No step is short-circuited by a
      preceding failure.
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
      Steps 3 through 6 (test verification, test substantiveness, coverage
      threshold, contract signature) depend on the verifier subsystem which
      does not yet exist. When a step's implementation is not available, gate
      must report that step as status "skipped" with reason "verifier not
      implemented" in the output. A skipped step must not cause exit code 1
      — it is an informational gap, not a failure. Gate must not silently
      omit these steps from the output; every step must appear in the result
      regardless of availability.
    supports: cli:REQ-006

  - id: REQ-005
    text: >
      Step 7 (baseline comparison) must compare current violations against
      the recorded baseline in .backstop/baseline.yml. If the baseline file
      does not exist, the step must report status "skipped" with reason
      "no baseline recorded." If the baseline file exists, any new violation
      (a rule+file combination not present in the baseline, or a count
      exceeding the baseline count for that rule+file) is a failure. Reduced
      violations (counts lower than baseline) are not failures. When the
      baseline subsystem is not yet implemented, this step must report status
      "skipped" with reason "baseline not implemented."
    supports: cli:REQ-006

  - id: REQ-006
    text: >
      Step 8 (ledger integrity) must verify the provenance ledger hash chain.
      If the ledger does not exist, the step must report status "skipped"
      with reason "no ledger found." If the ledger exists, the step must
      verify that every entry's hash is correct and the chain is unbroken.
      A broken hash chain or tampered entry is a failure. When the ledger
      subsystem is not yet implemented, this step must report status "skipped"
      with reason "ledger not implemented."
    supports: cli:REQ-006

  - id: REQ-007
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

  - id: REQ-008
    text: >
      Exit codes must follow the CLI contract: 0 when all executed steps
      pass (skipped steps do not prevent exit code 0), 1 when any step
      has status "fail", 2 when a configuration error occurs (backstop.yml
      missing or invalid, schema loading failure, embedded schema cohort
      error). Exit code 2 must take precedence over exit code 1 — if a
      config error is detected, the command must not report partial gate
      results. Skipped steps alone must not produce exit code 1.
    supports: cli:REQ-006

  - id: REQ-009
    text: >
      The gate JSON output must include a schema_version field for
      independent contract evolution. The initial schema version is
      "gate/v1". Changes to the JSON output structure follow D-070
      evolution rules: additive fields with sensible defaults, deprecated
      fields emit warnings, breaking removals only on major version bumps.
    supports: cli:REQ-007

  - id: REQ-010
    text: >
      The eight gate steps must be identified by the following canonical
      step names in the output: "artifact_validation", "code_check",
      "test_verification", "test_substantiveness", "coverage_threshold",
      "contract_signature", "baseline_comparison", "ledger_integrity".
      These names are part of the JSON output contract and must not change
      without a schema version bump.
    supports: cli:REQ-006

  - id: REQ-011
    text: >
      The gate command must accept no scope flags — it always runs against
      the full project. Unlike backstop artifact validate (which supports
      --spec, --plan, etc.) and backstop code check (which supports --diff,
      --all, --file), gate has no scoping mechanism. Gate always runs all
      steps against all artifacts and the full codebase.
    supports: cli:REQ-006

  - id: REQ-012
    text: >
      The human-readable output mode must display a summary table showing
      each step name, its status (pass/fail/skipped), and the count of
      violations for that step. Steps with status "skipped" must display
      their reason. The summary must end with an overall pass/fail verdict.
      The human output must respect the NO_COLOR environment variable.
    supports: cli:REQ-007

  - id: REQ-013
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
    text: Gate runs all eight steps in the specified order
    tests:
      - TestGate_AllStepsExecuteInOrder

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

  # REQ-004: Verifier-dependent steps report skipped
  - id: CLM-007
    requirement: REQ-004
    text: Test verification step reports status skipped with reason when verifier is not implemented
    tests:
      - TestGate_TestVerification_SkippedWhenNotImplemented

  - id: CLM-008
    requirement: REQ-004
    text: Test substantiveness step reports status skipped with reason when verifier is not implemented
    tests:
      - TestGate_TestSubstantiveness_SkippedWhenNotImplemented

  - id: CLM-009
    requirement: REQ-004
    text: Coverage threshold step reports status skipped with reason when verifier is not implemented
    tests:
      - TestGate_CoverageThreshold_SkippedWhenNotImplemented

  - id: CLM-010
    requirement: REQ-004
    text: Contract signature step reports status skipped with reason when verifier is not implemented
    tests:
      - TestGate_ContractSignature_SkippedWhenNotImplemented

  - id: CLM-011
    requirement: REQ-004
    text: Skipped steps do not appear in the output with status omitted — every step is always present
    tests:
      - TestGate_AllEightStepsAppearInOutput

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

  # REQ-006: Ledger integrity
  - id: CLM-014
    requirement: REQ-006
    text: Ledger integrity reports skipped when no ledger exists
    tests:
      - TestGate_Ledger_SkippedWhenNoLedger

  - id: CLM-015
    requirement: REQ-006
    text: Ledger integrity reports skipped when ledger subsystem is not implemented
    tests:
      - TestGate_Ledger_SkippedWhenNotImplemented

  # REQ-007: JSON output structure
  - id: CLM-016
    requirement: REQ-007
    text: JSON output includes schema_version, pass boolean, and steps array
    tests:
      - TestGate_JSONOutput_StructureComplete

  - id: CLM-017
    requirement: REQ-007
    text: JSON output pass is true only when all executed steps pass
    tests:
      - TestGate_JSONOutput_PassTrueWhenAllGreen

  - id: CLM-018
    requirement: REQ-007
    text: JSON output pass is false when any step has status fail
    tests:
      - TestGate_JSONOutput_PassFalseWhenAnyFail

  - id: CLM-019
    requirement: REQ-007
    text: JSON output pass is true when some steps pass and remaining steps are skipped
    tests:
      - TestGate_JSONOutput_PassTrueWithSkippedSteps

  - id: CLM-020
    requirement: REQ-007
    text: JSON output includes summary counts (total_violations, steps_passed, steps_failed, steps_skipped)
    tests:
      - TestGate_JSONOutput_SummaryCounts

  - id: CLM-021
    requirement: REQ-007
    text: Each step in the JSON steps array includes step_name, status, and violations fields
    tests:
      - TestGate_JSONOutput_StepFieldsPresent

  - id: CLM-022
    requirement: REQ-007
    text: Skipped steps include reason field in JSON output
    tests:
      - TestGate_JSONOutput_SkippedStepHasReason

  - id: CLM-023
    requirement: REQ-007
    text: Human-readable output is produced when --json flag is not set
    tests:
      - TestGate_HumanOutput_ProducedByDefault

  # REQ-008: Exit codes
  - id: CLM-024
    requirement: REQ-008
    text: Exit code 0 when all executed steps pass
    tests:
      - TestGate_ExitCode0_AllPass

  - id: CLM-025
    requirement: REQ-008
    text: Exit code 0 when all steps either pass or are skipped
    tests:
      - TestGate_ExitCode0_PassAndSkipped

  - id: CLM-026
    requirement: REQ-008
    text: Exit code 1 when any step fails
    tests:
      - TestGate_ExitCode1_StepFailed

  - id: CLM-027
    requirement: REQ-008
    text: Exit code 2 when config error occurs
    tests:
      - TestGate_ExitCode2_ConfigError

  - id: CLM-028
    requirement: REQ-008
    text: Exit code 2 takes precedence over exit code 1 when both config error and step failure occur
    tests:
      - TestGate_ExitCode2_PrecedenceOverExitCode1

  # REQ-009: Schema version in output
  - id: CLM-029
    requirement: REQ-009
    text: JSON output schema_version is "gate/v1"
    tests:
      - TestGate_JSONOutput_SchemaVersionGateV1

  # REQ-010: Canonical step names
  - id: CLM-030
    requirement: REQ-010
    text: All eight canonical step names appear in the output steps array
    tests:
      - TestGate_CanonicalStepNames_AllPresent

  - id: CLM-031
    requirement: REQ-010
    text: Step names in output exactly match the canonical names
    tests:
      - TestGate_CanonicalStepNames_ExactMatch

  # REQ-011: No scope flags
  - id: CLM-032
    requirement: REQ-011
    text: Gate command accepts no scope flags and runs against full project
    tests:
      - TestGate_NoScopeFlags_FullProject

  - id: CLM-033
    requirement: REQ-011
    text: Gate does not accept --diff, --file, --spec, --plan, or other scoping flags
    tests:
      - TestGate_RejectsScopeFlags

  # REQ-012: Human-readable output
  - id: CLM-034
    requirement: REQ-012
    text: Human output displays summary table with step name, status, and violation count
    tests:
      - TestGate_HumanOutput_SummaryTable

  - id: CLM-035
    requirement: REQ-012
    text: Human output shows reason for skipped steps
    tests:
      - TestGate_HumanOutput_SkippedStepReason

  - id: CLM-036
    requirement: REQ-012
    text: Human output ends with overall pass/fail verdict
    tests:
      - TestGate_HumanOutput_OverallVerdict

  - id: CLM-037
    requirement: REQ-012
    text: Human output respects NO_COLOR environment variable
    tests:
      - TestGate_HumanOutput_NoColorEnvVar

  # REQ-013: Config loading
  - id: CLM-038
    requirement: REQ-013
    text: Gate exits with code 2 when backstop.yml is not found
    tests:
      - TestGate_ConfigMissing_ExitCode2

  - id: CLM-039
    requirement: REQ-013
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
        signature: "type GateResult struct"
        notes: "Holds the unified gate output including all step results"
      - name: StepResult
        kind: type
        signature: "type StepResult struct"
        notes: "Per-step result with name, status, violations, and optional reason"
      - name: runGate
        kind: function
        signature: "func runGate(cmd *cobra.Command, args []string) error"
        notes: "Cobra RunE handler that orchestrates all eight steps"
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
signatures, baseline comparison, and ledger integrity.

The verifier subsystem (steps 3-6) does not yet exist. Gate initially runs
everything available (artifact validation + code check) and reports verifier-
dependent steps as "skipped" with an explicit reason. This ensures the gate
output contract is stable from day one — consumers always see all eight steps,
and the transition from "skipped" to "implemented" is additive, not structural.

## Requirements

Requirements are defined in frontmatter. The gate command has 13 requirements
covering step orchestration, delegation, deferred steps, output format, exit
codes, and configuration loading.

### Kill Chain Steps

Gate runs eight steps in fixed order. Every step appears in the output regardless
of whether it executed, failed, or was skipped.

| # | Step Name | Source | Initial Status |
|---|-----------|--------|----------------|
| 1 | `artifact_validation` | Delegates to artifact validate --all logic | Implemented |
| 2 | `code_check` | Delegates to code check --all logic | Implemented |
| 3 | `test_verification` | Verifier: mandated test names exist and pass | Skipped (verifier not implemented) |
| 4 | `test_substantiveness` | Verifier: AST analysis of test bodies | Skipped (verifier not implemented) |
| 5 | `coverage_threshold` | Verifier: coverage meets spec thresholds | Skipped (verifier not implemented) |
| 6 | `contract_signature` | Verifier: spec contracts match code | Skipped (verifier not implemented) |
| 7 | `baseline_comparison` | Compares violations against recorded baseline | Skipped (baseline not implemented) |
| 8 | `ledger_integrity` | Verifies provenance ledger hash chain | Skipped (ledger not implemented) |

### Exit Code Semantics

| Code | Meaning | When |
|------|---------|------|
| 0 | All green | All executed steps pass; skipped steps do not prevent 0 |
| 1 | Failures found | Any step has status "fail" |
| 2 | Config error | backstop.yml missing/invalid, schema load failure |

Exit code 2 takes precedence over 1. Skipped steps alone never produce exit code 1.

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

The `runGate` function orchestrates all eight steps sequentially:

1. **Config loading** — Load and validate backstop.yml. Exit code 2 on failure.
2. **Step execution loop** — For each of the eight steps:
   - If the step's implementation is available, execute it and collect results.
   - If the step's implementation is not available, produce a skipped result with reason.
   - Append the step result to the gate result.
3. **Result aggregation** — Compute summary counts (passed, failed, skipped, total violations).
4. **Output formatting** — Format as JSON or human text based on --json flag.
5. **Exit code determination** — 2 if config error, 1 if any failure, 0 otherwise.

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
  code 0) when steps 3-8 are skipped because the verifier, baseline, and
  ledger don't exist yet. A consumer of the gate output (e.g., a GitHub Actions
  gate action) must check the steps_skipped count and decide whether skipped
  steps are acceptable for merge. The gate itself does not make this policy
  decision — it reports the facts. A naive gate action that only checks the
  pass boolean will auto-merge code that has never been verified.

- **Artifact validate and code check errors vs gate errors.** When artifact
  validation or code check fails due to its own config error (e.g., schema
  loading failure), gate must decide whether this is a gate-level config error
  (exit 2) or a step failure (exit 1). The spec requires exit code 2 for
  config errors, but a config error detected inside a delegated step is
  ambiguous. The implementation must propagate config errors from delegated
  steps as gate-level exit code 2, not as step failures.

- **Step ordering is load-bearing for future steps.** Steps 3-6 depend on the
  code check (step 2) having already run — test verification needs test results,
  coverage needs test execution data. The fixed ordering is not arbitrary; it
  reflects data dependencies. If a future optimization attempts to parallelize
  steps, it must respect these dependencies.

- **Baseline comparison is a ratchet, not a snapshot.** The baseline records
  violation counts, not violation identity. If a developer fixes one violation
  but introduces a different one in the same file for the same rule, the count
  stays the same and baseline comparison passes. The ratchet catches regression
  in quantity, not quality.

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
   steps (artifact validate, code check) as gate-level exit code 2, or does
   it swallow them as step failures?

2. When a new verifier step is activated (e.g., test verification becomes
   implemented), does the gate output structure remain backward compatible
   with consumers that were parsing the "skipped" status?

3. Does the human output correctly handle the case where all eight steps are
   skipped (e.g., no verifier, no baseline, no ledger, and both artifact
   validate and code check have config errors)?

4. Is the GateResult.Pass field computed correctly in all combinations of
   pass, fail, and skipped statuses? Specifically: pass+skipped=true,
   fail+skipped=false, all-skipped=true?

5. Does the implementation prevent scope flags (--diff, --file, etc.) from
   being accepted, or does it silently ignore them?

## References

- Bundle: cli (spec seed 6 — backstop gate)
- ADR-0010: Verification Kill Chain — the eight-step chain gate implements
- ADR-0012: Review Model — "if it's green, it ships" (D-059)
- ADR-0009: CI/CD Pipeline — the GitHub Actions gate action that consumes gate JSON
- D-070: Schema evolution rules for the JSON output contract
- SPEC-005: CLI Foundation — config loading, output formatting, exit codes
- SPEC-006: Artifact Validate — step 1 delegation target
- SPEC-008: Code Check — step 2 delegation target
