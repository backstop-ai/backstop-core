---
title: "SPEC-006: Artifact Validate Command"
number: SPEC-006
created: "2026-04-04"
status: draft
schema_version: spec/v1
spec_version: 1.1.0

implementation:
  summary: >
    Implement the backstop artifact validate command that wraps pkg/validate
    for all six artifact types (spec, plan, adr, bundle, issue, standard).
    The command loads backstop.yml as a prerequisite (project root for discovery),
    resolves artifact type from the schema_version metadata field, routes to the
    correct type-specific validator (validate.Spec, validate.Plan, validate.ADR,
    validate.Bundle, validate.Issue, validate.Standard), loads schemas from the
    go:embed cohort via the refactored fs.FS-accepting schema.LoadArtifactSchema
    (SPEC-005), and formats results as structured JSON (--json) or human-readable
    text. Supports scoping via type-specific flags (--spec, --plan, --adr,
    --bundle, --issue, --standard) with optional artifact ID arguments, plus
    --all to validate every artifact in the project. Default behavior (no flags)
    validates all artifacts. Zero artifacts discovered returns exit 0 with a
    warning. Exit codes follow the CLI contract: 0 (pass), 1 (violations),
    2 (config error).
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/... -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      The backstop artifact validate command must route artifacts to the correct
      type-specific validator based on the schema_version metadata field. The
      mapping is: schema_version starting with "spec/" routes to validate.Spec,
      "plan/" routes to validate.Plan, "adr/" routes to validate.ADR, "bundle/"
      routes to validate.Bundle, "issue/" routes to validate.Issue, "standard/"
      routes to validate.Standard. An artifact with an unrecognized schema_version
      prefix must produce a config error (exit code 2), not a validation error.
    supports: cli:REQ-001

  - id: REQ-002
    text: >
      The command must support type-scoping flags: --spec, --plan, --adr,
      --bundle, --issue, --standard. Each flag accepts an optional artifact ID
      argument (e.g., --spec SPEC-002) to validate a single artifact of that type.
      When a flag is provided without an ID argument, all artifacts of that type
      are validated. Multiple type flags may be combined in a single invocation
      (e.g., --spec --plan validates all specs and plans). When no type flags
      are provided, the default behavior is to validate all artifacts (equivalent
      to --all).
    supports: cli:REQ-001

  - id: REQ-003
    text: >
      The --all flag must explicitly validate every artifact in the project
      across all six types. When --all is provided alongside type-scoping flags,
      --all takes precedence and all artifacts are validated regardless of the
      type flags. The command must discover artifacts by scanning the project
      directory for files matching artifact filename patterns.
    supports: cli:REQ-001

  - id: REQ-004
    text: >
      The command must produce structured JSON output when the --json flag
      is set. The JSON output must include: a schema_version field, a pass
      boolean, a total violations count, and an array of violation objects.
      Each violation object must include rule, file, message, and severity
      fields matching the validate.Violation struct. When --json is not set,
      the command must produce human-readable formatted text to stdout.
    supports: cli:REQ-001

  - id: REQ-005
    text: >
      Exit codes must follow the CLI contract: 0 when all validated artifacts
      pass with no violations, 1 when any artifact has violations, 2 when a
      configuration error occurs (unrecognized schema_version, schema loading
      failure, artifact parse failure, backstop.yml missing or invalid). Exit
      code 2 must take precedence over exit code 1 — if a config error is
      detected, the command must not report partial validation results.
    supports: cli:REQ-001

  - id: REQ-006
    text: >
      Schema loading must use the go:embed filesystem from SPEC-005. The
      command must resolve schema paths via schema.ResolveSchemaPath and load
      schemas via schema.LoadArtifactSchema, passing the embedded fs.FS
      interface. The command must not access the real filesystem for schema
      files. If a schema cannot be loaded from the embedded filesystem, the
      command must produce a config error (exit code 2).
    supports: cli:REQ-008

  - id: REQ-007
    text: >
      Artifact discovery must scan the project directory for files matching
      known artifact filename patterns. The six artifact types and their
      filename patterns are: spec files matching *.spec.md, plan files matching
      *.plan.yml, ADR files matching ADR-*.adr.md, bundle files matching
      *.bundle.md, issue files matching *.issue.md, standard files matching
      *.standard.md. Files that do not match any pattern are ignored. Files
      that match a pattern but fail to parse produce a config error for that
      file (not a validation error).
    supports: cli:REQ-001

  - id: REQ-008
    text: >
      When a type-scoping flag includes an artifact ID (e.g., --spec SPEC-002),
      the command must locate and validate only the artifact matching that ID.
      If no artifact with the given ID is found, the command must produce a
      config error (exit code 2) with a message identifying the missing artifact.
      The ID match must use a per-type metadata field mapping: plan artifacts
      match on the plan_id field, all other artifact types match on the number
      field. The match must not use filename substring.
    supports: cli:REQ-001

  - id: REQ-009
    text: >
      The command must aggregate validation results across all validated
      artifacts into a single output. In JSON mode, the output must be a
      single JSON object containing all violations from all artifacts. In
      human mode, violations must be grouped by file. The pass/fail
      determination is across all artifacts — if any artifact has violations,
      the overall result is fail (exit code 1).
    supports: cli:REQ-007

  - id: REQ-010
    text: >
      The command must be a thin adapter with no validation logic in cmd/.
      It must only: parse flags, discover artifacts, load schemas from the
      embedded filesystem, call the appropriate pkg/validate function for
      each artifact, aggregate results, format output, and set the exit code.
      All validation logic must remain in pkg/validate.

  - id: REQ-011
    text: >
      The command must load and validate backstop.yml as a prerequisite before
      artifact validation begins. If backstop.yml is missing or invalid, the
      command must produce a config error (exit code 2). The backstop.yml
      provides the project root directory used for artifact discovery. This
      is consistent with the CLI foundation prerequisite (SPEC-005 REQ-003)
      and bundle requirement cli:REQ-009.
    supports: cli:REQ-009

  - id: REQ-012
    text: >
      When artifact discovery finds zero artifacts (no files matching any
      artifact filename pattern, or no artifacts matching the requested type
      scope), the command must return exit code 0 with an empty violations
      list and emit a warning message to stderr indicating that no artifacts
      were found to validate.

claims:
  # REQ-001: Type routing (dependency matrix — all 6 types + unknown)
  - id: CLM-001
    requirement: REQ-001
    text: Artifact with schema_version "spec/v1" routes to validate.Spec
    tests:
      - TestArtifactValidate_Route_Spec

  - id: CLM-002
    requirement: REQ-001
    text: Artifact with schema_version "plan/v1" routes to validate.Plan
    tests:
      - TestArtifactValidate_Route_Plan

  - id: CLM-003
    requirement: REQ-001
    text: Artifact with schema_version "adr/v1" routes to validate.ADR
    tests:
      - TestArtifactValidate_Route_ADR

  - id: CLM-004
    requirement: REQ-001
    text: Artifact with schema_version "bundle/v1" routes to validate.Bundle
    tests:
      - TestArtifactValidate_Route_Bundle

  - id: CLM-005
    requirement: REQ-001
    text: Artifact with schema_version "issue/v1" routes to validate.Issue
    tests:
      - TestArtifactValidate_Route_Issue

  - id: CLM-006
    requirement: REQ-001
    text: Artifact with schema_version "standard/v1" routes to validate.Standard
    tests:
      - TestArtifactValidate_Route_Standard

  - id: CLM-007
    requirement: REQ-001
    text: Artifact with unrecognized schema_version prefix produces exit code 2
    tests:
      - TestArtifactValidate_Route_UnknownType_Exit2

  - id: CLM-008
    requirement: REQ-001
    text: Artifact with missing schema_version produces exit code 2
    tests:
      - TestArtifactValidate_Route_MissingSchemaVersion_Exit2

  # REQ-002: Type-scoping flags
  - id: CLM-009
    requirement: REQ-002
    text: "--spec flag without ID validates all spec artifacts"
    tests:
      - TestArtifactValidate_Scope_SpecAll

  - id: CLM-010
    requirement: REQ-002
    text: "--spec SPEC-002 validates only the spec with that ID"
    tests:
      - TestArtifactValidate_Scope_SpecByID

  - id: CLM-011
    requirement: REQ-002
    text: "--plan flag without ID validates all plan artifacts"
    tests:
      - TestArtifactValidate_Scope_PlanAll

  - id: CLM-012
    requirement: REQ-002
    text: "--adr flag without ID validates all ADR artifacts"
    tests:
      - TestArtifactValidate_Scope_ADRAll

  - id: CLM-013
    requirement: REQ-002
    text: "--bundle flag without ID validates all bundle artifacts"
    tests:
      - TestArtifactValidate_Scope_BundleAll

  - id: CLM-014
    requirement: REQ-002
    text: "--issue flag without ID validates all issue artifacts"
    tests:
      - TestArtifactValidate_Scope_IssueAll

  - id: CLM-015
    requirement: REQ-002
    text: "--standard flag without ID validates all standard artifacts"
    tests:
      - TestArtifactValidate_Scope_StandardAll

  - id: CLM-016
    requirement: REQ-002
    text: "Multiple type flags combined (--spec --plan) validates all artifacts of those types"
    tests:
      - TestArtifactValidate_Scope_MultipleFlags

  - id: CLM-017
    requirement: REQ-002
    text: "No type flags provided defaults to validating all artifacts"
    tests:
      - TestArtifactValidate_Scope_DefaultAll

  # REQ-003: --all flag
  - id: CLM-018
    requirement: REQ-003
    text: "--all flag validates every artifact across all six types"
    tests:
      - TestArtifactValidate_AllFlag_ValidatesEverything

  - id: CLM-019
    requirement: REQ-003
    text: "--all takes precedence over type-scoping flags when both are provided"
    tests:
      - TestArtifactValidate_AllFlag_PrecedesTypeFlags

  # REQ-004: JSON and human output
  - id: CLM-020
    requirement: REQ-004
    text: "--json flag produces valid JSON output with schema_version, pass, violations_count, and violations fields"
    tests:
      - TestArtifactValidate_JSON_OutputStructure

  - id: CLM-021
    requirement: REQ-004
    text: "JSON violation objects include rule, file, message, and severity fields"
    tests:
      - TestArtifactValidate_JSON_ViolationFields

  - id: CLM-022
    requirement: REQ-004
    text: "Default output (no --json) produces human-readable formatted text"
    tests:
      - TestArtifactValidate_Human_OutputFormat

  - id: CLM-023
    requirement: REQ-004
    text: "JSON and human output contain identical underlying violation data"
    tests:
      - TestArtifactValidate_OutputParity

  # REQ-005: Exit codes
  - id: CLM-024
    requirement: REQ-005
    text: Exit code 0 when all artifacts pass validation
    tests:
      - TestArtifactValidate_Exit0_AllPass

  - id: CLM-025
    requirement: REQ-005
    text: Exit code 1 when any artifact has violations
    tests:
      - TestArtifactValidate_Exit1_Violations

  - id: CLM-026
    requirement: REQ-005
    text: Exit code 2 on unrecognized schema_version
    tests:
      - TestArtifactValidate_Exit2_UnknownSchemaVersion

  - id: CLM-027
    requirement: REQ-005
    text: Exit code 2 on schema loading failure
    tests:
      - TestArtifactValidate_Exit2_SchemaLoadFailure

  - id: CLM-028
    requirement: REQ-005
    text: Exit code 2 on artifact parse failure
    tests:
      - TestArtifactValidate_Exit2_ParseFailure

  - id: CLM-029
    requirement: REQ-005
    text: Exit code 2 takes precedence over exit code 1 when both config error and violations exist
    tests:
      - TestArtifactValidate_Exit2_PrecedesExit1

  # REQ-006: Embedded schema loading
  - id: CLM-030
    requirement: REQ-006
    text: Schemas are loaded from the go:embed filesystem, not the real filesystem
    tests:
      - TestArtifactValidate_Schema_LoadedFromEmbed

  - id: CLM-031
    requirement: REQ-006
    text: Missing schema in embedded filesystem produces exit code 2
    tests:
      - TestArtifactValidate_Schema_MissingFromEmbed_Exit2

  # REQ-007: Artifact discovery (matrix — all 6 types discovered + non-artifact ignored)
  - id: CLM-032
    requirement: REQ-007
    text: Discovery finds *.spec.md files as spec artifacts
    tests:
      - TestArtifactValidate_Discover_Spec

  - id: CLM-033
    requirement: REQ-007
    text: Discovery finds *.plan.yml files as plan artifacts
    tests:
      - TestArtifactValidate_Discover_Plan

  - id: CLM-034
    requirement: REQ-007
    text: Discovery finds ADR-*.adr.md files as ADR artifacts
    tests:
      - TestArtifactValidate_Discover_ADR

  - id: CLM-035
    requirement: REQ-007
    text: Discovery finds *.bundle.md files as bundle artifacts
    tests:
      - TestArtifactValidate_Discover_Bundle

  - id: CLM-036
    requirement: REQ-007
    text: Discovery finds *.issue.md files as issue artifacts
    tests:
      - TestArtifactValidate_Discover_Issue

  - id: CLM-037
    requirement: REQ-007
    text: Discovery finds *.standard.md files as standard artifacts
    tests:
      - TestArtifactValidate_Discover_Standard

  - id: CLM-038
    requirement: REQ-007
    text: Files not matching any artifact pattern are ignored during discovery
    tests:
      - TestArtifactValidate_Discover_IgnoresNonArtifacts

  - id: CLM-039
    requirement: REQ-007
    text: Files matching a pattern but failing to parse produce a config error for that file
    tests:
      - TestArtifactValidate_Discover_ParseFailure_ConfigError

  # REQ-008: Scoping by artifact ID
  - id: CLM-040
    requirement: REQ-008
    text: "--spec SPEC-002 matches by the number metadata field (non-plan type), not filename"
    tests:
      - TestArtifactValidate_IDScope_SpecMatchesByNumber

  - id: CLM-041
    requirement: REQ-008
    text: "Scoping by ID for a nonexistent artifact produces exit code 2"
    tests:
      - TestArtifactValidate_IDScope_NotFound_Exit2

  # REQ-009: Result aggregation
  - id: CLM-042
    requirement: REQ-009
    text: Violations from multiple artifacts are aggregated into a single JSON output
    tests:
      - TestArtifactValidate_Aggregate_JSON

  - id: CLM-043
    requirement: REQ-009
    text: Human output groups violations by file
    tests:
      - TestArtifactValidate_Aggregate_HumanGroupedByFile

  - id: CLM-044
    requirement: REQ-009
    text: Overall pass/fail is determined across all artifacts (one failure means fail)
    tests:
      - TestArtifactValidate_Aggregate_AnyFailMeansOverallFail

  # REQ-010: Thin adapter (enforcement is a code review concern — see Sharp Edges)
  - id: CLM-045
    requirement: REQ-010
    text: Command calls pkg/validate functions and does not reimplement validation logic inline
    tests:
      - TestArtifactValidate_ThinAdapter_CallsPkgValidate

  # REQ-011: backstop.yml prerequisite
  - id: CLM-046
    requirement: REQ-011
    text: Command loads backstop.yml before artifact validation begins
    tests:
      - TestArtifactValidate_BackstopYml_LoadedAsPrerequisite

  - id: CLM-047
    requirement: REQ-011
    text: Missing backstop.yml produces exit code 2
    tests:
      - TestArtifactValidate_BackstopYml_Missing_Exit2

  - id: CLM-048
    requirement: REQ-011
    text: Invalid backstop.yml produces exit code 2
    tests:
      - TestArtifactValidate_BackstopYml_Invalid_Exit2

  # REQ-012: Zero-artifact scenario
  - id: CLM-049
    requirement: REQ-012
    text: Zero artifacts discovered returns exit code 0 with empty violations list
    tests:
      - TestArtifactValidate_ZeroArtifacts_Exit0_EmptyViolations

  - id: CLM-050
    requirement: REQ-012
    text: Zero artifacts discovered emits a warning message to stderr
    tests:
      - TestArtifactValidate_ZeroArtifacts_WarningMessage

  # REQ-008: Plan ID uses plan_id field
  - id: CLM-051
    requirement: REQ-008
    text: "--plan PLAN-SPEC-002 matches by the plan_id metadata field, not number"
    tests:
      - TestArtifactValidate_IDScope_PlanMatchesByPlanID

contracts:
  - file: cmd/backstop/artifact_validate.go
    provides:
      - name: NewArtifactValidateCommand
        kind: function
        signature: "func NewArtifactValidateCommand() *cobra.Command"
        notes: "Builds the Cobra command for backstop artifact validate with all flags"
      - name: ValidateArtifacts
        kind: function
        signature: "func ValidateArtifacts(cfg ValidateConfig) (ValidateResult, error)"
        notes: "Orchestrates discovery, routing, validation, and aggregation — the command's core logic"
      - name: ValidateConfig
        kind: type
        signature: "type ValidateConfig struct"
        notes: "Holds parsed flags: type filters, artifact IDs, json mode, embedded FS"
      - name: ValidateResult
        kind: type
        signature: "type ValidateResult struct"
        notes: "Aggregated result with pass bool, violations count, and violations list"
    consumes:
      - source: pkg/validate
        name: Spec
        kind: function
      - source: pkg/validate
        name: Plan
        kind: function
      - source: pkg/validate
        name: ADR
        kind: function
      - source: pkg/validate
        name: Bundle
        kind: function
      - source: pkg/validate
        name: Issue
        kind: function
      - source: pkg/validate
        name: Standard
        kind: function
      - source: pkg/validate
        name: ValidationResult
        kind: type
      - source: pkg/validate
        name: Violation
        kind: type
      - source: pkg/artifact
        name: ParseFile
        kind: function
      - source: pkg/artifact
        name: ParsedArtifact
        kind: type
      - source: pkg/schema
        name: ResolveSchemaPath
        kind: function
      - source: pkg/schema
        name: LoadArtifactSchema
        kind: function
        notes: "Refactored to accept fs.FS interface (SPEC-005 scope prerequisite)"
      - source: pkg/schema
        name: Schema
        kind: type
      - source: io/fs
        name: FS
        kind: interface
      - source: cmd/backstop
        name: SchemaFS
        kind: variable
      - source: cmd/backstop
        name: Formatter
        kind: interface
      - source: cmd/backstop
        name: ExitWithResult
        kind: function
      - source: github.com/spf13/cobra
        name: Command
        kind: type
---

# SPEC-006: Artifact Validate Command

## Overview

The `backstop artifact validate` command is the primary artifact enforcement
entry point. It wraps the existing pkg/validate library (which already has
298+ passing tests at 96.7% coverage) behind a CLI interface that agents and
humans use to validate backstop artifacts against embedded schemas.

The command handles three concerns that the library does not:

1. **Artifact discovery** — scanning the project directory for files matching
   artifact filename patterns across all six types
2. **Type routing** — reading the schema_version metadata from each artifact
   and dispatching to the correct type-specific validator
3. **Result aggregation** — collecting violations from all validated artifacts
   into a single output with the correct exit code

The command is a thin adapter (DD-2 from the CLI bundle): it parses flags,
discovers artifacts, loads embedded schemas, calls pkg/validate functions,
aggregates results, formats output, and sets the exit code. No validation
logic lives in cmd/.

This spec depends on SPEC-005 (CLI Foundation) for the Cobra command skeleton,
go:embed schema cohort, output formatter, and exit code infrastructure.

## Requirements

Requirements are defined in frontmatter.

### Type Routing

The command routes artifacts to validators based on the schema_version metadata
prefix. The routing table:

| schema_version prefix | Validator function | Artifact type |
|----------------------|-------------------|---------------|
| `spec/` | `validate.Spec` | Spec |
| `plan/` | `validate.Plan` | Plan |
| `adr/` | `validate.ADR` | ADR |
| `bundle/` | `validate.Bundle` | Bundle |
| `issue/` | `validate.Issue` | Issue |
| `standard/` | `validate.Standard` | Standard |
| anything else | config error (exit 2) | N/A |

An artifact with a missing schema_version field is also a config error — the
command cannot determine which validator to use.

### Scoping Flags

| Flag | Argument | Behavior |
|------|----------|----------|
| `--spec` | optional ID (e.g., `SPEC-002`) | Validate one or all specs |
| `--plan` | optional ID (e.g., `PLAN-SPEC-002`) | Validate one or all plans |
| `--adr` | optional ID (e.g., `ADR-0018`) | Validate one or all ADRs |
| `--bundle` | optional ID | Validate one or all bundles |
| `--issue` | optional ID | Validate one or all issues |
| `--standard` | optional ID | Validate one or all standards |
| `--all` | none | Validate everything (precedence over type flags) |
| (none) | N/A | Default: validate all artifacts (same as --all) |

Multiple type flags may be combined. When no flags are provided, the default
behavior is equivalent to --all.

### Artifact Discovery

The command discovers artifacts by scanning the project directory for files
matching known filename patterns:

| Artifact type | Filename pattern |
|---------------|-----------------|
| Spec | `*.spec.md` |
| Plan | `*.plan.yml` |
| ADR | `ADR-*.adr.md` |
| Bundle | `*.bundle.md` |
| Issue | `*.issue.md` |
| Standard | `*.standard.md` |

Files not matching any pattern are silently ignored. Files matching a pattern
but failing to parse are config errors (the file is expected to be a valid
artifact but is malformed).

### Exit Codes

| Code | Condition |
|------|-----------|
| 0 | All validated artifacts pass with no violations (includes zero-artifact case) |
| 1 | One or more artifacts have violations |
| 2 | Config error: backstop.yml missing/invalid, unrecognized schema_version, schema load failure, parse failure, artifact not found by ID |

Exit code 2 takes precedence over exit code 1 — the command must not report
partial validation results when a config error is detected.

## Implementation

The implementation creates a single file `cmd/backstop/artifact_validate.go`
that registers under the artifact namespace command from SPEC-005.

**Prerequisites:**

- **pkg/schema refactor (SPEC-005 scope):** schema.LoadArtifactSchema must be
  refactored to accept an fs.FS interface parameter instead of a base directory
  string. This refactor is part of SPEC-005 and must be completed before this
  command can consume the embedded schema filesystem.

- **backstop.yml loading (SPEC-005 REQ-003):** The command loads backstop.yml
  to determine the project root directory before artifact discovery begins.

**Note on plan YAML handling:** artifact.ParseFile handles both
Markdown-with-YAML-frontmatter files and pure YAML files (plans). The same
YAML parser is used for frontmatter extraction — pure YAML files are parsed
as all-frontmatter with no body. No separate parse path is required for plan
artifacts.

### Pass 1: Command Registration

Create `NewArtifactValidateCommand()` returning a `*cobra.Command` with:
- Six type-scoping flags (--spec, --plan, --adr, --bundle, --issue, --standard)
  each accepting an optional string argument
- An --all boolean flag
- Short and Long descriptions
- RunE function that orchestrates the validation pipeline

Register the command under the artifact namespace command.

### Pass 2: Artifact Discovery

Implement artifact discovery that scans the project directory tree for files
matching the six artifact filename patterns. The discovery function takes the
project root (from backstop.yml) and optional type/ID filters and returns a
list of file paths grouped by artifact type. Discovery must handle:
- Recursive directory scanning
- Pattern matching per artifact type
- Filtering by type when scoping flags are set
- Filtering by ID when an ID argument is provided (requires parsing each
  candidate to check the per-type ID field: plan_id for plans, number for
  all other types)
- Zero artifacts found: return an empty list (caller emits warning, exit 0)

### Pass 3: Type Router

Implement the routing function that takes a parsed artifact and returns the
appropriate validator function. The router extracts the schema_version prefix
(the part before the `/`) and maps it to one of the six validator functions.
Unknown prefixes and missing schema_version are config errors.

### Pass 4: Validation Pipeline

Implement the orchestration function `ValidateArtifacts` that:
1. Loads and validates backstop.yml (exit 2 on failure)
2. Discovers artifacts based on flags using the project root from backstop.yml
3. If zero artifacts discovered, returns exit 0 with empty violations and warning
4. Parses each artifact via artifact.ParseFile (handles both markdown and YAML)
5. Resolves the schema path via schema.ResolveSchemaPath
6. Loads the schema from the embedded fs.FS via schema.LoadArtifactSchema
7. Routes to the correct validator
8. Collects all violations
9. Returns the aggregated result

Config errors at any step (parse failure, schema resolution, schema load,
unknown type) short-circuit with exit code 2.

### Pass 5: Output Formatting and Exit

Use the Formatter from SPEC-005 to serialize the ValidateResult as JSON or
human-readable text. Use ExitWithResult from SPEC-005 to determine the exit
code. In human mode, violations are grouped by file for readability.

## Verification

Claims are defined in frontmatter. Integration-level verification with 80%
coverage threshold on the cmd/backstop/ package, because this command
integrates artifact parsing, schema loading, type routing, and validation
library calls across multiple packages.

## Sharp Edges

- **Plan artifacts are YAML, not Markdown.** All other artifact types are
  Markdown with YAML frontmatter parsed by artifact.ParseFile. Plan files
  (*.plan.yml) are pure YAML. artifact.ParseFile handles both formats because
  it uses the same YAML parser for frontmatter extraction — pure YAML files
  are parsed as all-frontmatter with no body. No separate parse path is needed.

- **ID matching semantics vary by type.** Plan artifacts use the `plan_id`
  metadata field for ID matching; all other types use `number`. The per-type
  field mapping (REQ-008) must be maintained in the ID-scoping logic. Getting
  the field name wrong means the artifact is "not found" even when it exists.

- **cmd/ files must be thin adapters.** REQ-010 states no validation logic
  in cmd/. This is a code review concern, not a test assertion — verifying
  the absence of logic via automated tests is brittle and couples tests to
  implementation structure. Enforcement relies on review discipline.

- **Default scope is all, not changed-files.** The bundle mentions
  changed-files-only as the default for `backstop validate`, but the spec seed
  says default is all artifacts. Changed-files scoping is a concern for
  `backstop code check` (spec seed 5), not `backstop artifact validate`. If
  the scope default changes later, the discovery layer must be updated.

- **Concurrent artifact validation.** For large projects with many artifacts,
  sequential validation may be slow. The current design validates artifacts
  sequentially. Parallelization is a future optimization but introduces
  complexity around error aggregation and config error short-circuiting.

- **go:embed FS bridging requires pkg/schema refactor.** schema.LoadArtifactSchema
  currently takes a base directory string ("artifacts") for filesystem access.
  It is being refactored to accept an fs.FS interface parameter as part of
  SPEC-005 scope. This command depends on that refactor being complete. Tests
  must inject a test fs.FS to avoid depending on the real embedded filesystem.

- **Type flag with ID but wrong type prefix.** If a user passes
  `--spec PLAN-SPEC-002`, the command searches for a spec with number
  PLAN-SPEC-002, which will not match any spec's number field. The error
  message must be clear that no spec with that ID was found, not that the
  artifact does not exist (it may exist as a plan).

## Review Questions

1. **RESOLVED:** artifact.ParseFile handles both Markdown-with-frontmatter
   (specs, ADRs, bundles, issues, standards) and pure YAML (plans) files.
   The YAML parser extracts frontmatter from both formats — pure YAML files
   are parsed as all-frontmatter with no body. No separate parse path needed.

2. **RESOLVED:** schema.LoadArtifactSchema is being refactored to accept an
   fs.FS interface parameter (SPEC-005 scope). This command consumes the
   refactored signature.

3. When --spec SPEC-002 is provided but SPEC-002 has validation violations,
   should the command still report exit code 1 (violations found), or could
   the single-artifact scoping mode reasonably return a different error
   structure?

4. **RESOLVED:** Zero artifacts discovered returns exit code 0 with an empty
   violations list and a warning message (REQ-012). No artifacts to validate
   means no violations.

5. **RESOLVED:** Plan artifacts use the plan_id metadata field for ID matching.
   All other artifact types use the number field. This per-type mapping is
   specified in REQ-008.

6. **RESOLVED:** backstop.yml must be loaded as a prerequisite (REQ-011),
   consistent with SPEC-005 REQ-003 and bundle cli:REQ-009. The project root
   from backstop.yml is used for artifact discovery.

## References

- Bundle: cli (artifact validate spec seed 2)
- Bundle requirement: REQ-001
- Bundle design decisions: DD-2 (thin adapters), DD-3 (go:embed), DD-4 (config before enforcement), DD-5 (versioned JSON)
- Resolved OQs: OQ-1 (Cobra), OQ-2 (schema_version field), OQ-7 (gate vs validate boundary)
- SPEC-005: CLI Foundation (command skeleton, embed cohort, output layer, exit codes)
- pkg/validate: Existing validators — Spec, Plan, ADR, Bundle, Issue, Standard
- pkg/artifact: ParseFile, ParsedArtifact
- pkg/schema: ResolveSchemaPath, LoadArtifactSchema, Schema
