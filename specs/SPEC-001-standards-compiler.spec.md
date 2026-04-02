---
title: "SPEC-001: Standards Compiler"
number: SPEC-001
created: "2026-03-29"
status: active
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Compile backstop standard artifacts (.standard.md) into per-standard
    enforcement manifests that the CLI lazy-merges at runtime. The compiler
    reads the rules block from standard frontmatter, routes each rule by its
    detection strategy, and produces three outputs: (1) an enforcement manifest
    listing every rule with its enforcement method and compliance tier, (2) a
    semgrep YAML config for pattern and regex rules, and (3) a thin backstop-native
    checks file for structural metric rules (e.g., test_file_exists). Most metric
    enforcement is delegated to ecosystem tools. Delegated rules are recorded in
    the manifest so the CLI can verify the external tool ran and the specific
    rule was enabled. Schema resolution uses Go embed for CLI consumers and an
    explicit path override for library consumers.
  package: pkg/compile

verification:
  level: unit
  test_command: go test ./pkg/compile/ -race -coverprofile=cover.out
  coverage_threshold: 90

requirements:
  - id: REQ-001
    text: >
      The compiler must read a standard artifact file, parse its frontmatter,
      and extract the rules array with all nested detection blocks intact.
  - id: REQ-002
    text: >
      For each rule with detection.strategy "pattern" and a semgrep field,
      the compiler must emit a valid semgrep YAML rule with the pattern,
      rule ID, message, severity mapping, and language.
  - id: REQ-003
    text: >
      The semgrep severity mapping must be: error → ERROR, warning → WARNING,
      info → INFO.
  - id: REQ-004
    text: >
      For each rule with detection.strategy "metric", the compiler must emit
      a backstop-native check definition containing the metric name, operator,
      threshold, and any exclude patterns. The native metric layer is thin —
      reserved for structural checks (e.g., test_file_exists) that no ecosystem
      tool covers. Most metric enforcement is delegated to ecosystem tools via
      the "delegated" strategy.
  - id: REQ-005
    text: >
      For each rule with detection.strategy "regex", the compiler must emit
      a semgrep YAML rule using the pattern-regex key.
  - id: REQ-006
    text: >
      The compiler must validate the standard artifact before compiling.
      If validation fails, compilation must abort and return the violations.
  - id: REQ-007
    text: >
      The compiler output directory must be configurable. Default is
      .backstop/rules/ relative to the project root.
  - id: REQ-008
    text: >
      The compiler must produce a single enforcement manifest JSON file that
      lists every rule from the standard, its enforcement method (semgrep,
      native, or delegated), and the metadata the CLI needs to orchestrate
      execution. Output filenames must derive from the standard number.
    note: >
      REQ-009 was removed (includes feature dropped per bundle DD-4).
  - id: REQ-010
    text: >
      The compiler must be idempotent — running it twice on the same input
      must produce identical output.
  - id: REQ-011
    text: >
      Rules with detection.strategy "delegated" must be included in the
      enforcement manifest with their enforced_by tool name and rule ID,
      but must not produce semgrep or native check output. The CLI uses
      this information to verify delegated tools ran successfully.
  - id: REQ-012
    text: >
      Rules with a note field but no enforceable detection fields are advisory
      documentation and must be excluded from both the semgrep config and the
      enforcement manifest. Enforceable fields are: semgrep, metric, pattern
      (for regex), and enforced_by (for delegated). A rule is advisory when
      NONE of these fields are present in its detection block, regardless of
      the stated detection strategy. This applies to any detection strategy.
  - id: REQ-013
    text: >
      Every rule in the enforcement manifest must include its compliance_tier
      value (baseline, standard, or strict). The CLI uses this to filter rules
      at runtime based on the project's configured tier.
  - id: REQ-014
    text: >
      If the standard artifact has status "deprecated", the compiler must still
      produce valid output but emit a visible deprecation warning. The warning
      must include the standard number and any superseded_by reference.
  - id: REQ-015
    text: >
      For universal-scope standards, pattern and regex rules must specify a
      languages field in their detection block. The compiler must use the
      per-rule languages field instead of a standard-level language field
      when routing to semgrep.
  - id: REQ-016
    text: >
      Schema resolution must support two modes: filesystem path (default in v1
      for both library and CLI) and embedded schemas (via Go embed, deferred to
      CLI build phase). CompileOptions must accept an optional SchemaSource
      interface. When nil, the compiler defaults to filesystem resolution using
      the standard artifacts/ directory. Embedded schemas will be implemented
      when the CLI is built — each CLI version becomes a schema cohort.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: Compiler parses standard artifact and extracts rules array
    tests:
      - TestCompile_ParsesRulesFromStandard
  - id: CLM-002
    requirement: REQ-002
    text: Pattern rules emit valid semgrep YAML with correct fields
    tests:
      - TestEmitSemgrepRule_Pattern
      - TestCompile_PatternRuleEmitsSemgrep
  - id: CLM-003
    requirement: REQ-003
    text: Severity levels map correctly to semgrep severity values
    tests:
      - TestSemgrepRule_SeverityUppercase
      - TestEmitSemgrepRule_SeverityMapping
  - id: CLM-004
    requirement: REQ-004
    text: Metric rules emit backstop-native check definitions
    tests:
      - TestEmitNativeCheck_MetricRule
      - TestCompile_MetricRuleEmitsNativeCheck
  - id: CLM-005
    requirement: REQ-005
    text: Regex rules emit semgrep pattern-regex rules
    tests:
      - TestEmitSemgrepRule_Regex
      - TestCompile_RegexRuleEmitsSemgrepRegex
  - id: CLM-006
    requirement: REQ-006
    text: Invalid standard artifacts abort compilation with violations
    tests:
      - TestCompile_InvalidStandardReturnsViolations
  - id: CLM-007
    requirement: REQ-007
    text: Output directory is configurable with correct default
    tests:
      - TestCompile_OutputDirectoryConfigurable
      - TestCompile_OutputDirectoryDefault
  - id: CLM-008
    requirement: REQ-008
    text: Enforcement manifest lists all rules with correct enforcement methods
    tests:
      - TestCompile_ManifestContainsAllRules
      - TestCompile_ManifestEnforcementMethods
      - TestCompile_OutputFilenameFromStandardNumber
  - id: CLM-009
    requirement: REQ-009
    text: "Removed — includes feature dropped per bundle DD-4"
    tests: []
  - id: CLM-010
    requirement: REQ-010
    text: Compilation is idempotent
    tests:
      - TestCompile_Idempotent
  - id: CLM-011
    requirement: REQ-011
    text: Delegated rules appear in manifest but not in semgrep or native output
    tests:
      - TestCompile_DelegatedRulesInManifest
      - TestCompile_DelegatedRulesNotInSemgrep
  - id: CLM-012
    requirement: REQ-012
    text: >
      Advisory rules are excluded from all output. A rule is advisory when its
      detection block contains a note field and NONE of the enforceable fields
      (semgrep, metric, pattern, enforced_by) regardless of detection strategy.
    tests:
      - TestCompile_AdvisoryRulesExcluded
      - TestCompile_AdvisoryRulesExcludedFromManifest
  - id: CLM-013
    requirement: REQ-013
    text: Manifest rules include compliance tier tags
    tests:
      - TestCompile_ManifestIncludesComplianceTier
      - TestCompile_ManifestTierDefaultsToBaseline
  - id: CLM-014
    requirement: REQ-014
    text: Deprecated standards compile with warning on CompileResult.Warnings
    tests:
      - TestCompile_DeprecatedStandardEmitsWarning
      - TestCompile_DeprecatedWarningContainsStandardNumber
      - TestCompile_DeprecatedStandardStillProducesOutput
  - id: CLM-015
    requirement: REQ-015
    text: >
      Universal-scope pattern and regex rules use per-rule languages.
      A universal-scope pattern or regex rule without per-rule languages
      must produce a compilation error, not broken semgrep output.
    tests:
      - TestCompile_UniversalPatternRuleUsesPerRuleLanguages
      - TestCompile_UniversalRegexRuleUsesPerRuleLanguages
      - TestCompile_UniversalMetricRuleNoLanguageRequired
      - TestCompile_UniversalPatternWithoutLanguagesFails
  - id: CLM-016
    requirement: REQ-016
    text: >
      Schema resolution supports filesystem path (default in v1) and
      embedded schemas (via Go embed, deferred to CLI build). Library
      consumers override via CompileOptions.SchemaSource.
    tests:
      - TestCompile_DefaultSchemaResolution
      - TestCompile_FilesystemSchemaOverride

contracts:
  - file: pkg/compile/types.go
    provides:
      - name: Rule
        kind: type
        signature: "type Rule struct"
        notes: "Contains ID, Name, Severity, Detection map, ComplianceTier, Languages. Methods: Strategy(), IsAdvisory()"
      - name: CompileOptions
        kind: type
        signature: "type CompileOptions struct"
      - name: CompileResult
        kind: type
        signature: "type CompileResult struct"
        notes: "Contains Manifest, SemgrepRules, NativeChecks, Warnings []string, OutputPaths"
      - name: EnforcementManifest
        kind: type
        signature: "type EnforcementManifest struct"
      - name: ManifestRule
        kind: type
        signature: "type ManifestRule struct"
        notes: "Methods: EffectiveTier() defaults to baseline"
      - name: DelegatedTarget
        kind: type
        signature: "type DelegatedTarget struct"
      - name: SemgrepRule
        kind: type
        signature: "type SemgrepRule struct"
      - name: NativeCheck
        kind: type
        signature: "type NativeCheck struct"
      - name: SchemaSource
        kind: interface
        signature: "type SchemaSource interface"
      - name: MapSeverity
        kind: function
        signature: "func MapSeverity(severity string) string"
    consumes:
      - source: pkg/schema
        name: Schema
        kind: type

  - file: pkg/compile/compile.go
    provides:
      - name: Compile
        kind: function
        signature: "func Compile(standardPath string, opts CompileOptions) (*CompileResult, error)"
    consumes:
      - source: pkg/artifact
        name: ParseFile
        kind: function
      - source: pkg/validate
        name: Standard
        kind: function
      - source: pkg/schema
        name: LoadArtifactSchema
        kind: function
      - source: pkg/schema
        name: ResolveSchemaPath
        kind: function

  - file: pkg/compile/semgrep.go
    provides:
      - name: EmitSemgrepRule
        kind: function
        signature: "func EmitSemgrepRule(rule Rule, languages []string) SemgrepRule"
      - name: WriteSemgrepFile
        kind: function
        signature: "func WriteSemgrepFile(rules []SemgrepRule, path string) error"
    consumes: []

  - file: pkg/compile/native.go
    provides:
      - name: EmitNativeCheck
        kind: function
        signature: "func EmitNativeCheck(rule Rule) NativeCheck"
      - name: WriteNativeChecksFile
        kind: function
        signature: "func WriteNativeChecksFile(checks []NativeCheck, path string) error"
    consumes: []
---

# SPEC-001: Standards Compiler

## Overview

The standards compiler transforms backstop standard artifacts (`.standard.md`) into a unified enforcement manifest and executable rule configs. Standard artifacts are the human-authored source of truth — they contain rule definitions, rationale, examples, and primary sources. The compiler serializes the machine-readable portions into formats that enforcement engines can execute, and produces a manifest the CLI uses to orchestrate everything.

The CLI is the single orchestrator. Whether a rule is enforced by semgrep, by the CLI's own metric checks, or by an external tool like golangci-lint, the enforcement manifest is the single file that says "here are all the checks for this language and here's how to run each one."

## Requirements

Requirements are defined in frontmatter (REQ-001 through REQ-016).

Key design decisions:

- **Manifest-centric output**: The primary output is an enforcement manifest JSON file. Semgrep YAML and native check JSON are secondary artifacts referenced by the manifest. The CLI reads the manifest and orchestrates all enforcement from it.
- **Four detection strategies**: `pattern` (semgrep AST), `metric` (native CLI checks), `regex` (semgrep regex), `delegated` (external tool). Each strategy maps to a different enforcement path, but all appear in the manifest.
- **Validate before compile**: The compiler calls `validate.Standard()` before processing. A standard with validation errors produces no output.
- **Delegated rules**: Rules enforced by external tools (golangci-lint, goimports, etc.) are recorded in the manifest with the tool name and specific rule ID. The CLI can verify the tool ran and the rule was enabled — closing the trust loop.
- **Advisory rules**: Rules with a `note` but no enforceable detection fields are advisory documentation. They are excluded from all output — not the manifest, not semgrep, nothing. They exist only in the standard file for human readers. This applies to any detection strategy, not just pattern.
- **Compliance tier tagging**: Every rule in the manifest carries its compliance tier (baseline/standard/strict). Rules without an explicit tier default to baseline. The CLI filters at runtime based on the project's configured tier.
- **Deprecated standard handling**: Deprecated standards compile with a visible warning returned on `CompileResult.Warnings`. The warning includes the standard number and `superseded_by` reference if present. Output is still produced.
- **Universal-scope pattern rules**: Universal standards can include pattern/regex rules if they specify `languages` per-rule in their detection block. The compiler uses these per-rule languages instead of a standard-level language field.
- **Schema resolution**: Embedded via Go embed for CLI consumers (each CLI version is a schema cohort). Library consumers override with an explicit filesystem path via `CompileOptions.SchemaSource`.
- **No includes**: Split standard files (includes) are not supported in v1. One file per standard. The CLI lazy-merges per-standard manifests at runtime.

## Implementation

### Package Structure

```
pkg/compile/
├── compile.go      # Orchestrator: parse → validate → route → manifest → write
├── semgrep.go      # Semgrep YAML emitter for pattern and regex rules
├── native.go       # Native check emitter for metric rules
├── compile_test.go # Tests for orchestrator and manifest
├── semgrep_test.go # Tests for semgrep emission
└── native_test.go  # Tests for native check emission
```

### Compile Flow

```
standard.md ─→ ParseFile() ─→ validate.Standard() ─→ route by strategy
                                                        ├─ pattern   → EmitSemgrepRule() + manifest entry
                                                        ├─ regex     → EmitSemgrepRule() + manifest entry
                                                        ├─ metric    → EmitNativeCheck() + manifest entry
                                                        ├─ delegated → manifest entry only
                                                        └─ advisory  → skip entirely
                                                     ─→ WriteManifest()
                                                     ─→ WriteSemgrepFile()
                                                     ─→ WriteNativeChecksFile()
```

### Enforcement Manifest Format

The manifest is the single file the CLI reads to know what needs to run:

```json
{
  "standard": "STD-GO-001",
  "language": "go",
  "compiled_from": "standards/go/STD-GO-001-go-code-standards.standard.md",
  "semgrep_config": "STD-GO-001.semgrep.yml",
  "native_checks": "STD-GO-001.checks.json",
  "rules": [
    {
      "id": "GO-001",
      "name": "max-file-length",
      "severity": "error",
      "compliance_tier": "baseline",
      "enforcement": "native"
    },
    {
      "id": "GO-010",
      "name": "no-ignored-errors",
      "severity": "error",
      "compliance_tier": "baseline",
      "enforcement": "semgrep"
    },
    {
      "id": "GO-020",
      "name": "no-stuttering-exports",
      "severity": "warning",
      "compliance_tier": "standard",
      "enforcement": "delegated",
      "delegated_to": {
        "tool": "golangci-lint",
        "rule": "revive/exported"
      }
    }
  ]
}
```

The CLI reads this and:
1. Collects `enforcement: semgrep` → runs semgrep once with the referenced config
2. Collects `enforcement: native` → runs metric checks itself
3. Collects `enforcement: delegated` → verifies the named tool ran and rule passed

### Semgrep Config Format

Generated from pattern and regex rules:

```yaml
rules:
  - id: GO-010
    message: "Error return values must not be silently discarded"
    severity: ERROR
    languages: [go]
    pattern: |
      $VAL, _ := $FUNC(...)
```

### Native Checks Format

Generated from metric rules:

```json
{
  "checks": [
    {
      "id": "GO-001",
      "message": "Go source files must not exceed 500 lines",
      "severity": "error",
      "language": "go",
      "metric": "file_lines",
      "operator": ">",
      "threshold": 500,
      "exclude": ["_test.go"]
    }
  ]
}
```

## Verification

Verification is defined in frontmatter. Unit-level verification with 90% coverage threshold.

Test command: `go test ./pkg/compile/ -race -coverprofile=cover.out`

Claims CLM-001 through CLM-016 map each requirement to specific test functions.

## Sharp Edges

- **Semgrep pattern fidelity**: Not all semgrep pattern features can be represented in YAML frontmatter. Complex patterns with metavariable constraints or taint tracking will need a `semgrep_raw` field in a future schema version.
- **Metric execution**: The compiler emits metric check definitions but does not execute them. The CLI runtime is responsible for running metric checks — this spec covers compilation only.
- **YAML escaping**: Semgrep patterns containing YAML-sensitive characters (colons, brackets) must be block-scalar quoted in the standard file. The compiler inherits whatever the YAML parser produces.
- **Delegated verification depth**: The manifest records which external tool should enforce a rule, but verifying that the tool actually ran with that rule enabled is the CLI's responsibility, not the compiler's.
- **Universal pattern without languages**: A universal-scope standard with a pattern or regex rule that lacks per-rule `languages` must produce a compilation error. Emitting a semgrep rule with empty languages would cause semgrep to reject it silently.
- **MetricEvaluator interface**: DD-3 calls for an interface-based native metric layer. The `MetricEvaluator` interface is deferred to SPEC-002 (Metric Evaluator). This spec emits static check definitions only.
- **Duplicate rule IDs**: The standard validator enforces per-file uniqueness (DD-5). The compiler should defensively error on duplicate rule IDs rather than silently producing a manifest with duplicates.
