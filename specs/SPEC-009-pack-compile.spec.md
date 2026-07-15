---
title: "SPEC-009: Pack Compile — CLI Adapter for Standards Compilation"
number: SPEC-009
created: "2026-04-04"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Implement the `backstop pack compile` CLI command as a thin adapter
    wrapping pkg/compile. The command discovers all .standard.md files in the
    project's standards directories (configured in backstop.yml or defaulting
    to standards/), iterates over each standard file, calls compile.Compile()
    for each, writes enforcement output (manifest JSON, semgrep YAML, native
    checks JSON) to .backstop/rules/, aggregates results and warnings across
    all standards, and reports results in JSON or human output mode. The
    command handles deprecated standard warnings, compilation errors, and
    config errors with consistent exit codes. Output is idempotent — same
    input standards produce identical output files.
  package: cmd/backstop

verification:
  level: unit
  test_command: go test ./cmd/backstop/... -run TestPackCompile -race -coverprofile=cover.out
  coverage_threshold: 90

requirements:
  - id: REQ-001
    text: >
      The backstop pack compile command must discover all .standard.md files
      by recursively searching the project's standards directories. The
      directories to search are configured in backstop.yml (e.g., a
      standards_dirs field) or default to standards/ relative to the project
      root. Files not matching the *.standard.md glob pattern must be ignored.
    supports: cli:REQ-004@1.0.0

  - id: REQ-002
    text: >
      For each discovered .standard.md file, the command must call
      compile.Compile() from pkg/compile with appropriate CompileOptions.
      The output directory must be .backstop/rules/ relative to the project
      root. The command must not reimplement any compilation logic — it is
      a thin adapter over pkg/compile.
    supports: cli:REQ-004@1.0.0

  - id: REQ-003
    text: >
      The command must produce three output artifact types per standard
      (as determined by pkg/compile): an enforcement manifest JSON file,
      a semgrep YAML config file (if the standard contains pattern or regex
      rules), and a native checks JSON file (if the standard contains metric
      rules). All output files are written to .backstop/rules/.
    supports: cli:REQ-004@1.0.0

  - id: REQ-004
    text: >
      The command must support --json and human output modes. In JSON mode,
      the command writes a structured JSON object to stdout containing: the
      list of compiled standards (with their output paths), any warnings
      (including deprecation warnings), any errors, and a summary count of
      standards compiled. In human mode, the command prints a formatted
      summary showing each standard compiled, output files produced, warnings,
      and errors. Both modes must produce identical underlying data. The
      PackCompileResult struct does not carry its own schema_version field —
      schema_version is added to all JSON responses by the SPEC-005 CLI output
      formatter, which wraps command results before writing to stdout.
    supports: cli:REQ-007@1.0.0

  - id: REQ-005
    text: >
      Exit codes must follow the CLI convention: 0 when all standards compile
      successfully, 1 when one or more standards have compilation errors
      (e.g., validation failures, unsupported strategies), 2 when a
      configuration error prevents compilation from starting (e.g., missing
      backstop.yml, invalid config). When backstop.yml specifies multiple
      standards_dirs and some exist but others do not, the command must
      compile from valid directories and emit a warning for each missing
      directory — this is not an exit 2 condition. Exit 2 for missing
      directories applies only when NO configured directories exist.
    supports: cli:REQ-004@1.0.0

  - id: REQ-006
    text: >
      When a standard has status "deprecated", the command must surface the
      deprecation warning from CompileResult.Warnings in its output. Deprecated
      standards must still be compiled — deprecation is a warning, not an error.
      The output must include the standard number and any superseded_by reference.

  - id: REQ-007
    text: >
      The command must be idempotent. Running backstop pack compile twice
      on the same set of .standard.md inputs with no changes must produce
      byte-identical output files in .backstop/rules/. This property is
      inherited from pkg/compile's idempotency but the CLI adapter must not
      introduce any non-determinism (e.g., random ordering of standards,
      timestamps in output).

  - id: REQ-008
    text: >
      The command must create the .backstop/rules/ output directory if it
      does not exist. If the directory already exists, existing files must
      be overwritten by the new compilation output. The command must not
      delete files from previous compilations that correspond to standards
      no longer present — stale file cleanup is out of scope for v1.

  - id: REQ-009
    text: >
      If no .standard.md files are found in the configured standards
      directories, the command must exit with code 0 and report that zero
      standards were compiled. This is not an error — a project may not
      have standards yet.

  - id: REQ-010
    text: >
      When multiple standards are compiled and some succeed while others
      fail, the command must compile all standards that can be compiled
      (not stop at the first error), report all failures, and exit with
      code 1. Partial compilation is preferable to total failure.

  - id: REQ-011
    text: >
      The command must process standards in a deterministic order (e.g.,
      sorted by file path) so that output and log ordering is reproducible
      across runs. Non-deterministic ordering would violate the idempotency
      requirement for human output mode.

  - id: REQ-012
    text: >
      The command must load backstop.yml before execution via the CLI
      foundation's config loader (SPEC-005). If backstop.yml is missing
      or invalid, the command must exit with code 2 before attempting
      any standard discovery or compilation.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: Command discovers .standard.md files recursively in configured standards directories
    tests:
      - TestPackCompile_DiscoversStandardFiles

  - id: CLM-002
    requirement: REQ-001
    text: Command ignores files not matching *.standard.md pattern
    tests:
      - TestPackCompile_IgnoresNonStandardFiles

  - id: CLM-003
    requirement: REQ-001
    text: Command uses default standards/ directory when backstop.yml does not specify standards_dirs
    tests:
      - TestPackCompile_DefaultStandardsDir

  - id: CLM-004
    requirement: REQ-002
    text: Command calls compile.Compile for each discovered standard file
    tests:
      - TestPackCompile_CallsCompileForEachStandard

  - id: CLM-005
    requirement: REQ-002
    text: Command passes .backstop/rules/ as the output directory to CompileOptions
    tests:
      - TestPackCompile_OutputDirIsBackstopRules

  - id: CLM-006
    requirement: REQ-002
    text: Command does not reimplement compilation logic — delegates entirely to pkg/compile
    tests:
      - TestPackCompile_DelegatesCompilationToPkgCompile

  - id: CLM-007
    requirement: REQ-003
    text: Command produces manifest JSON, semgrep YAML, and native checks JSON per standard
    tests:
      - TestPackCompile_ProducesAllOutputTypes

  - id: CLM-008
    requirement: REQ-004
    text: JSON output mode produces structured JSON with compiled standards, warnings, errors, and summary count
    tests:
      - TestPackCompile_JSONOutputStructure

  - id: CLM-009
    requirement: REQ-004
    text: Human output mode prints formatted summary with compiled standards and warnings
    tests:
      - TestPackCompile_HumanOutputFormat

  - id: CLM-010
    requirement: REQ-004
    text: JSON and human modes produce identical underlying data
    tests:
      - TestPackCompile_OutputModesIdenticalData

  - id: CLM-011
    requirement: REQ-005
    text: Exit code 0 when all standards compile successfully
    tests:
      - TestPackCompile_ExitCode0OnSuccess

  - id: CLM-012
    requirement: REQ-005
    text: Exit code 1 when one or more standards have compilation errors
    tests:
      - TestPackCompile_ExitCode1OnCompilationError

  - id: CLM-013
    requirement: REQ-005
    text: Exit code 2 when configuration error prevents compilation
    tests:
      - TestPackCompile_ExitCode2OnConfigError

  - id: CLM-014
    requirement: REQ-005
    text: Exit code 2 when no configured standards directories exist
    tests:
      - TestPackCompile_ExitCode2OnAllStandardsDirsMissing

  - id: CLM-030
    requirement: REQ-005
    text: When some configured standards_dirs exist and others do not, command compiles from valid directories and emits warning for missing ones (not exit 2)
    tests:
      - TestPackCompile_PartialDirsMissingCompilesAndWarns

  - id: CLM-015
    requirement: REQ-006
    text: Deprecated standard warnings are surfaced in command output
    tests:
      - TestPackCompile_DeprecatedWarningInOutput

  - id: CLM-016
    requirement: REQ-006
    text: Deprecated standards are still compiled successfully
    tests:
      - TestPackCompile_DeprecatedStandardStillCompiles

  - id: CLM-017
    requirement: REQ-006
    text: Deprecation warning includes standard number and superseded_by reference
    tests:
      - TestPackCompile_DeprecatedWarningContainsDetails

  - id: CLM-018
    requirement: REQ-007
    text: Running compile twice with same input produces byte-identical output
    tests:
      - TestPackCompile_IdempotentOutput

  - id: CLM-019
    requirement: REQ-008
    text: Command creates .backstop/rules/ directory if it does not exist
    tests:
      - TestPackCompile_CreatesOutputDir

  - id: CLM-020
    requirement: REQ-008
    text: Command overwrites existing files from previous compilations
    tests:
      - TestPackCompile_OverwritesExistingFiles

  - id: CLM-021
    requirement: REQ-009
    text: Command exits with code 0 when no .standard.md files are found
    tests:
      - TestPackCompile_NoStandardsExitCode0

  - id: CLM-022
    requirement: REQ-009
    text: Command reports zero standards compiled when none are found
    tests:
      - TestPackCompile_NoStandardsReportsZero

  - id: CLM-023
    requirement: REQ-010
    text: Command compiles all standards that succeed even when some fail
    tests:
      - TestPackCompile_PartialCompilationOnMixedResults

  - id: CLM-024
    requirement: REQ-010
    text: Command reports all failures when multiple standards fail
    tests:
      - TestPackCompile_ReportsAllFailures

  - id: CLM-025
    requirement: REQ-010
    text: Command exits with code 1 on partial failure
    tests:
      - TestPackCompile_ExitCode1OnPartialFailure

  - id: CLM-026
    requirement: REQ-011
    text: Standards are processed in deterministic sorted order by file path
    tests:
      - TestPackCompile_DeterministicOrder

  - id: CLM-027
    requirement: REQ-012
    text: Command exits with code 2 when backstop.yml is missing
    tests:
      - TestPackCompile_ExitCode2OnMissingConfig

  - id: CLM-028
    requirement: REQ-012
    text: Command exits with code 2 when backstop.yml is invalid
    tests:
      - TestPackCompile_ExitCode2OnInvalidConfig

  - id: CLM-029
    requirement: REQ-012
    text: Config loading happens before standard discovery or compilation
    tests:
      - TestPackCompile_ConfigLoadedBeforeDiscovery

contracts:
  - file: cmd/backstop/pack_compile.go
    provides:
      - name: newPackCompileCommand
        kind: function
        signature: "func newPackCompileCommand(jsonFlag *bool) *cobra.Command"
        notes: "Factory for Cobra pack compile command. Takes shared --json flag pointer."
      - name: runPackCompileWithOpts
        kind: function
        signature: "func runPackCompileWithOpts(opts packCompileOpts) (*PackCompileResult, error)"
        notes: "Testable core — receives resolved options, returns typed result."
      - name: packCompileOpts
        kind: type
        signature: "type packCompileOpts struct"
        notes: "Resolved options for pack compile run: projectRoot, standardsDirs, outputDir, jsonOutput, schemaSource."
      - name: PackCompileResult
        kind: type
        signature: "type PackCompileResult struct"
        notes: "Aggregated compilation results. Does not carry schema_version (added by JSON envelope)."
    consumes:
      - source: pkg/compile
        name: Compile
        kind: function
      - source: pkg/compile
        name: CompileOptions
        kind: type
      - source: pkg/compile
        name: CompileResult
        kind: type
      - source: pkg/compile
        name: SchemaSource
        kind: interface
      - source: pkg/config
        name: LoadConfig
        kind: function
      - source: pkg/config
        name: Config
        kind: type
      - source: pkg/schema
        name: LoadArtifactSchema
        kind: function
      - source: pkg/schema
        name: Schema
        kind: type

  - file: cmd/backstop/discover.go
    provides:
      - name: discoverStandards
        kind: function
        signature: "func discoverStandards(dirs []string) ([]string, error)"
        notes: "Recursively walks directories, returns sorted list of .standard.md file paths."
    consumes: []
---

# SPEC-009: Pack Compile — CLI Adapter for Standards Compilation

## Overview

The `backstop pack compile` command is a thin CLI adapter wrapping pkg/compile
(SPEC-001). The standards compiler already exists as a fully tested Go library
that transforms `.standard.md` files into enforcement manifests. This spec covers
the CLI surface that orchestrates it: discovering all standard files in the project,
invoking the compiler for each, managing the output directory, aggregating results
and warnings, and presenting them in JSON or human output format with correct exit
codes.

The command is the bridge between "standards exist as files" and "enforcement
rules exist in .backstop/rules/". After `backstop pack compile` runs, the
enforcement manifests, semgrep configs, and native check definitions are ready
for `backstop code check` and `backstop gate` to consume.

### Why a CLI adapter spec?

pkg/compile (SPEC-001) compiles a single standard file. The CLI command adds:

1. **Discovery** — finding all .standard.md files across configured directories
2. **Iteration** — compiling each discovered standard
3. **Aggregation** — collecting results, warnings, and errors across all standards
4. **Output formatting** — JSON and human output modes
5. **Exit code handling** — mapping compilation outcomes to exit codes
6. **Config dependency** — loading backstop.yml before compilation
7. **Directory management** — ensuring .backstop/rules/ exists

None of these are pkg/compile's responsibility. They are CLI concerns.

## Requirements

Requirements are defined in frontmatter (REQ-001 through REQ-012).

### Standard Discovery

The command recursively searches configured standards directories for files
matching `*.standard.md`. The search directories default to `standards/`
relative to the project root. Files not matching the glob are ignored. Standards
are processed in deterministic sorted order by file path.

### Compilation Delegation

For each discovered standard, the command calls `compile.Compile()` with
`CompileOptions` specifying `.backstop/rules/` as the output directory. The
command does not reimplement any compilation logic — it is strictly a thin
adapter. Each standard produces up to three output files: enforcement manifest
JSON, semgrep YAML config, and native checks JSON.

### Output Modes

| Mode | Flag | Behavior |
|------|------|----------|
| Human | (default) | Formatted summary: each standard compiled, output files, warnings, errors |
| JSON | `--json` | Structured JSON: compiled standards list, warnings, errors, summary count |

Both modes produce identical underlying data. The PackCompileResult struct does
not carry its own schema_version — the SPEC-005 CLI output formatter wraps all
command results with a schema_version field before writing JSON to stdout.

### Exit Codes

| Code | Meaning | Examples |
|------|---------|---------|
| 0 | All standards compiled successfully, or no standards found | Clean compile, empty project |
| 1 | One or more compilation errors | Validation failure, unsupported strategy |
| 2 | Configuration error prevents compilation | Missing backstop.yml, invalid config, all configured standards directories missing |

Exit code 2 takes precedence — config errors prevent any compilation attempt. When
multiple standards_dirs are configured and some exist but others do not, the command
compiles from valid directories and emits a warning for each missing directory. Exit 2
applies only when no configured directories exist at all.

### Error Handling

When multiple standards are compiled and some fail, the command continues
compiling all remaining standards rather than stopping at the first error.
All failures are reported in the output. This partial-compilation behavior
ensures that valid standards still produce usable output even when one
standard has issues.

### Idempotency

The command inherits idempotency from pkg/compile. The CLI adapter must not
introduce non-determinism (random ordering, timestamps). Running the command
twice on unchanged input produces byte-identical output files.

## Implementation

### Command Registration

The `backstop pack compile` command is a Cobra subcommand registered under the
`pack` namespace (established by SPEC-005 CLI Foundation). It supports the
global `--json` flag for output mode selection.

### Processing Pipeline

The command executes in six sequential steps:

1. **Config loading** — load and validate backstop.yml via the foundation config
   loader. Exit 2 on failure.

2. **Directory resolution** — read standards directories from config, defaulting
   to `standards/`. Verify each directory exists. If some directories exist but
   others do not, emit a warning for each missing directory and proceed with the
   valid ones. Exit 2 only if no configured directories exist at all.

3. **Standard discovery** — recursively walk each standards directory, collecting
   all files matching `*.standard.md`. Sort results by file path for
   deterministic ordering.

4. **Compilation** — for each discovered standard file, call `compile.Compile()`
   with `CompileOptions{OutputDir: ".backstop/rules/"}`. Collect results and
   errors. Do not stop on first error.

5. **Aggregation** — merge all CompileResults: collect output paths, warnings
   (including deprecation warnings), and errors. Count successes and failures.

6. **Output** — format results as JSON or human output based on --json flag.
   Set exit code: 0 if all succeeded, 1 if any failed, 2 if config error
   (already handled in step 1-2).

### File Organization

```
cmd/backstop/
├── pack_compile.go   # Command definition, handler, result aggregation
├── discover.go       # Standard file discovery (recursive glob, sort)
└── pack_compile_test.go  # Tests for the compile command
```

### Discovery Function

`discoverStandards(dirs []string) ([]string, error)` walks each directory
recursively, matches `*.standard.md`, and returns a sorted slice of absolute
paths. The sort ensures deterministic processing order.

### Result Aggregation

The command builds a `PackCompileResult` struct containing:

```go
type PackCompileResult struct {
    Standards []CompileStandardResult  // per-standard results
    Warnings  []string                 // aggregated warnings
    Errors    []string                 // aggregated errors
    Summary   CompileSummary           // counts
}

type CompileStandardResult struct {
    Standard    string   // standard number (e.g., STD-GO-001)
    SourceFile  string   // input .standard.md path
    OutputPaths []string // output file paths
    Warnings    []string // per-standard warnings
    Error       string   // compilation error, empty if success
}

type CompileSummary struct {
    Total    int
    Compiled int
    Failed   int
}
```

## Verification

Verification is defined in frontmatter. Unit-level verification with 90%
coverage threshold targeting the cmd/backstop package.

Claims are defined in frontmatter. Each claim maps a requirement to specific
test functions that verify the behavior.

Tests use a test fixture approach: temporary directories with sample .standard.md
files, a minimal backstop.yml, and assertions on output files, stdout content,
and exit codes. The compile.Compile function is called through the real package
(not mocked) to verify end-to-end behavior of the CLI adapter.

## Sharp Edges

- **Stale output files are not cleaned up.** If a standard is removed from the
  project, its previously compiled output files remain in .backstop/rules/. The
  command does not track which files it previously produced. A user must manually
  delete stale files or wipe .backstop/rules/ before recompiling. A future
  `--clean` flag could address this, but it risks deleting files from other
  sources (e.g., installed packs).

- **Standards directory misconfiguration is silent in some cases.** If
  backstop.yml specifies a standards_dirs entry that exists but contains no
  .standard.md files, the command reports zero standards compiled (exit 0).
  This could mask a misconfigured path where the user expected standards to be
  found. The command cannot distinguish "intentionally empty" from "wrong path"
  without additional heuristics.

- **File ordering depends on OS filepath behavior.** The deterministic ordering
  requirement uses sorted file paths, but path separators and collation may vary
  across operating systems. In practice, Go's filepath.Walk and sort.Strings
  produce consistent results within a given OS, but cross-platform byte-identical
  human output is not guaranteed if paths contain non-ASCII characters.

- **Partial compilation may confuse downstream commands.** When some standards
  compile and others fail (exit 1), .backstop/rules/ contains a mix of fresh
  output from successful standards and potentially stale output from previously
  compiled versions of the failed standards. Downstream commands like
  `backstop code check` may use stale rules from a previously compiled version
  of a now-failing standard. This is preferable to total failure but could
  produce misleading enforcement results.

- **Config loading dependency on SPEC-005.** This command depends on the CLI
  foundation's config loader (SPEC-005) existing and working correctly. If
  SPEC-005 is not yet implemented, this command's config loading behavior must
  be stubbed or implemented inline, creating a potential divergence from the
  foundation's eventual behavior.

- **Concurrent compilation is not specified.** The command processes standards
  sequentially. For projects with many standards, parallel compilation could
  improve performance, but pkg/compile writes to a shared output directory
  and may not be safe for concurrent use. Sequential processing is the safe
  default; parallelism is a future optimization.

## Review Questions

1. If a standard file is valid YAML but fails schema validation (e.g., missing
   required fields), does the command report this as a compilation error (exit 1)
   or a config error (exit 2)? The spec says compilation errors are exit 1, and
   pkg/compile validates standards before compiling — is this distinction clear
   to the implementer?

2. **Resolved.** When backstop.yml specifies multiple standards_dirs entries and
   some exist but others do not, the command compiles from valid directories and
   emits a warning for each missing directory. Exit 2 applies only when no
   configured directories exist. Partial success beats total failure.

3. The discovery function returns absolute paths. Does the human output mode
   display absolute paths (verbose but unambiguous) or relative paths (cleaner
   but context-dependent)? The spec does not prescribe this.

4. If .backstop/rules/ is a symlink to another directory, should the command
   follow it or error? Standard os.MkdirAll follows symlinks, but this could
   write to unexpected locations.

5. **Resolved.** The PackCompileResult struct does not carry its own
   schema_version field. The SPEC-005 CLI output formatter wraps all command
   results with a schema_version before writing JSON to stdout. No per-command
   schema version is needed.

## References

- Bundle: cli (REQ-004, spec seed 4)
- SPEC-001: Standards Compiler (pkg/compile — the library this command wraps)
- SPEC-005: CLI Foundation (config loading, output layer, exit codes, Cobra skeleton)
- ADR-0004: JSON output as API contract
- ADR-0005: backstop.yml project manifest
- D-070: Schema evolution rules
