---
title: "SPEC-001: Standards Compiler"
number: SPEC-001
created: "2026-03-29"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Compile backstop standard artifacts (.standard.md) into a unified enforcement
    manifest that the CLI uses to orchestrate all rule enforcement. The compiler
    reads the rules block from standard frontmatter, routes each rule by its
    detection strategy, and produces three outputs: (1) an enforcement manifest
    listing every rule and how to run it, (2) a semgrep YAML config for pattern
    and regex rules, and (3) a backstop-native checks file for metric rules.
    Delegated rules are recorded in the manifest so the CLI can verify the
    external tool ran and the specific rule was enabled.
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
      threshold, and any exclude patterns.
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
      Pattern rules that have a note field but no semgrep field are
      advisory documentation and must be excluded from both the semgrep
      config and the enforcement manifest.

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
      - TestCompile_PatternRuleEmitsSemgrep
  - id: CLM-003
    requirement: REQ-003
    text: Severity levels map correctly to semgrep severity values
    tests:
      - TestCompile_SeverityMapping
  - id: CLM-004
    requirement: REQ-004
    text: Metric rules emit backstop-native check definitions
    tests:
      - TestCompile_MetricRuleEmitsNativeCheck
  - id: CLM-005
    requirement: REQ-005
    text: Regex rules emit semgrep pattern-regex rules
    tests:
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
    text: Included split files are resolved and compiled
    tests:
      - TestCompile_IncludesResolved
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
    text: Advisory pattern rules are excluded from all output
    tests:
      - TestCompile_AdvisoryRulesExcluded

contracts:
  - file: pkg/compile/compile.go
    provides:
      - name: Compile
        kind: function
        signature: "func Compile(standardPath string, opts CompileOptions) (*CompileResult, error)"
      - name: CompileOptions
        kind: type
        signature: "type CompileOptions struct"
      - name: CompileResult
        kind: type
        signature: "type CompileResult struct"
      - name: EnforcementManifest
        kind: type
        signature: "type EnforcementManifest struct"
      - name: ManifestRule
        kind: type
        signature: "type ManifestRule struct"
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
        signature: "func EmitSemgrepRule(rule Rule, lang string) SemgrepRule"
      - name: SemgrepRule
        kind: type
        signature: "type SemgrepRule struct"
      - name: WriteSemgrepFile
        kind: function
        signature: "func WriteSemgrepFile(rules []SemgrepRule, path string) error"
    consumes: []

  - file: pkg/compile/native.go
    provides:
      - name: EmitNativeCheck
        kind: function
        signature: "func EmitNativeCheck(rule Rule) NativeCheck"
      - name: NativeCheck
        kind: type
        signature: "type NativeCheck struct"
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

Requirements are defined in frontmatter (REQ-001 through REQ-012).

Key design decisions:

- **Manifest-centric output**: The primary output is an enforcement manifest JSON file. Semgrep YAML and native check JSON are secondary artifacts referenced by the manifest. The CLI reads the manifest and orchestrates all enforcement from it.
- **Four detection strategies**: `pattern` (semgrep AST), `metric` (native CLI checks), `regex` (semgrep regex), `delegated` (external tool). Each strategy maps to a different enforcement path, but all appear in the manifest.
- **Validate before compile**: The compiler calls `validate.Standard()` before processing. A standard with validation errors produces no output.
- **Delegated rules**: Rules enforced by external tools (golangci-lint, goimports, etc.) are recorded in the manifest with the tool name and specific rule ID. The CLI can verify the tool ran and the rule was enabled — closing the trust loop.
- **Advisory rules**: Pattern rules with a `note` but no `semgrep` field are advisory documentation. They are excluded from all output — not the manifest, not semgrep, nothing. They exist only in the standard file for human readers.
- **Includes resolution**: Split standard files are resolved relative to the parent standard's directory.

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
      "enforcement": "native"
    },
    {
      "id": "GO-010",
      "name": "no-ignored-errors",
      "severity": "error",
      "enforcement": "semgrep"
    },
    {
      "id": "GO-020",
      "name": "no-stuttering-exports",
      "severity": "warning",
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

Claims CLM-001 through CLM-012 map each requirement to specific test functions.

## Sharp Edges

- **Semgrep pattern fidelity**: Not all semgrep pattern features can be represented in YAML frontmatter. Complex patterns with metavariable constraints or taint tracking will need a `semgrep_raw` field in a future schema version.
- **Metric execution**: The compiler emits metric check definitions but does not execute them. The CLI runtime is responsible for running metric checks — this spec covers compilation only.
- **YAML escaping**: Semgrep patterns containing YAML-sensitive characters (colons, brackets) must be block-scalar quoted in the standard file. The compiler inherits whatever the YAML parser produces.
- **Delegated verification depth**: The manifest records which external tool should enforce a rule, but verifying that the tool actually ran with that rule enabled is the CLI's responsibility, not the compiler's.
