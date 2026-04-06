---
title: "SPEC-008: Code Check — Implementation Validation with Lint, Build, Test, Semgrep"
number: SPEC-008
created: "2026-04-04"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Implement the backstop code check command that runs implementation validation
    (golangci-lint, go build, go test, semgrep rules) against changed files by
    default, the full codebase with --all, or a single file with --file. The
    --file mode is designed for runtime hook dispatch with a 2-second execution
    budget and concurrent invocation safety. File-type routing uses compiled
    enforcement manifests from .backstop/rules/ to determine which checks apply
    to which files. Semgrep is auto-installed to .backstop/tools/ if not found
    on PATH. Output is JSON (--json) or human-readable, with exit codes 0/1/2.
    This command replaces the alpha backstop-hook binary.
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/... ./pkg/check/... -run "TestCodeCheck" -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      backstop code check must run four implementation validation passes in
      order: lint (golangci-lint), build (go build), test (go test), and
      semgrep rules. Each pass produces zero or more violations. All four
      passes run regardless of earlier pass failures — the command collects
      all violations before reporting. A pass that is not applicable to the
      current scope (e.g., semgrep when no rules match) is skipped silently.
    supports: cli:REQ-003

  - id: REQ-002
    text: >
      The default scope for backstop code check (no flags) is changed files
      only. Changed files are determined by git merge-base with the default
      branch (origin/main or origin/master) for PR scope, or git diff
      (staged + unstaged) for local scope. If git is not available or the
      working directory is not a git repository, the command must fall back
      to --all scope and emit a warning in the output.
    supports: cli:REQ-010

  - id: REQ-003
    text: >
      The --all flag must run all four validation passes against the entire
      codebase, ignoring changed-files detection. When --all is set, the
      command does not invoke git at all for scope determination.
    supports: cli:REQ-003

  - id: REQ-004
    text: >
      The --file flag accepts a single file path and runs only the validation
      passes applicable to that file type, as determined by the compiled
      enforcement manifests in .backstop/rules/. The --file flag must not be
      combined with --all — specifying both is a configuration error (exit 2).
    supports: cli:REQ-003

  - id: REQ-005
    text: >
      When --file is specified, the entire command must complete within a
      2-second execution budget. If any validation pass exceeds the budget,
      the command must cancel it and return a timeout warning in the output
      rather than blocking. The timeout applies to the total command execution,
      not to individual passes. The exit code for a timeout is 1 (violations),
      with the timeout reported as a violation.
    supports: cli:REQ-003

  - id: REQ-006
    text: >
      File-type routing must use compiled enforcement manifests from
      .backstop/rules/ to determine which validation passes apply to a given
      file. The manifest maps file extensions and path patterns to applicable
      check types (lint, build, test, semgrep rule IDs). Files that do not
      match any manifest entry receive no checks and produce zero violations.
      If no manifest files exist in .backstop/rules/, the command must use
      built-in defaults: .go files get lint, build, test, and semgrep; all
      other files get semgrep only if matching rules exist.
    supports: cli:REQ-003

  - id: REQ-007
    text: >
      If semgrep is not found on PATH, the command must auto-install it to
      .backstop/tools/semgrep on first invocation. The installed version must
      match the version pinned in backstop.yml (if specified). Subsequent
      invocations must verify the installed version matches the pin. Two
      distinct failure modes exist: (a) semgrep is installed but at the wrong
      version (version mismatch) — this is a configuration error and must
      produce exit code 2 (hard stop, no checks run); (b) semgrep is not
      installed and auto-install fails (no network, download error) — the
      command must skip semgrep checks and emit a warning in degraded mode,
      it must not fail the entire command. These are different scenarios:
      a version mismatch means deliberate misconfiguration; an install
      failure means infrastructure unavailability.
    supports: cli:REQ-003

  - id: REQ-008
    text: >
      The command must support --json for structured JSON output and default
      to human-readable output. Both modes must produce identical underlying
      violation data. JSON output must include a schema_version field. Human
      output must respect the NO_COLOR environment variable.
    supports: cli:REQ-007

  - id: REQ-009
    text: >
      Exit codes must be: 0 when all checks pass with no violations, 1 when
      one or more violations are found, 2 when a configuration error occurs
      (invalid backstop.yml, --file combined with --all, missing .backstop/
      directory). Exit code 2 takes precedence — if config is invalid, no
      checks run.
    supports: cli:REQ-003

  - id: REQ-010
    text: >
      The --file mode must be safe for concurrent invocations. Multiple
      instances of backstop code check --file running simultaneously (e.g.,
      multiple hook dispatches) must not corrupt shared state, produce
      interleaved output, or interfere with each other. Each invocation must
      operate independently. Shared resources (semgrep install, manifest
      reads) must use file-level locking or be idempotent.
    supports: cli:REQ-003

  - id: REQ-011
    text: >
      The command must load backstop.yml before executing any checks. If
      backstop.yml is not found or fails validation, the command must exit
      with code 2. The config loading must use the same walk-up discovery
      and BACKSTOP_CONFIG env var override as defined in SPEC-005.
    supports: cli:REQ-009

  - id: REQ-013
    text: >
      Changed-files detection in default (diff) mode must follow a 4-step
      cascade: (1) git merge-base HEAD origin/main to find the fork point,
      then git diff --name-only <merge-base>..HEAD to list changed files;
      (2) if origin/main does not exist, try origin/master with the same
      merge-base approach; (3) if neither remote branch exists, fall back to
      git diff --name-only (staged + unstaged local changes) and emit a
      warning; (4) if git is not available or the directory is not a git
      repository, fall back to --all scope and emit a warning. Steps 3 and
      4 correspond to the fallback behaviors in REQ-002.
    supports: cli:REQ-010

  - id: REQ-014
    text: >
      If golangci-lint is not found on PATH, the lint pass must be skipped
      with a warning (same pattern as semgrep install failure). The go and
      go test tools are assumed available because backstop operates within
      a Go project — their absence is not handled gracefully.
    supports: cli:REQ-003

claims:
  # REQ-001: Four validation passes
  - id: CLM-001
    requirement: REQ-001
    text: All four validation passes (lint, build, test, semgrep) execute and produce violations
    tests:
      - TestCodeCheck_AllPassesRun

  - id: CLM-002
    requirement: REQ-001
    text: All passes run even when an earlier pass produces violations
    tests:
      - TestCodeCheck_PassesContinueAfterViolation

  - id: CLM-003
    requirement: REQ-001
    text: A non-applicable pass is skipped silently with no violations
    tests:
      - TestCodeCheck_NonApplicablePassSkipped

  - id: CLM-004
    requirement: REQ-001
    text: Passes execute in order lint then build then test then semgrep
    tests:
      - TestCodeCheck_PassOrderLintBuildTestSemgrep

  # REQ-002: Default scope is changed files
  - id: CLM-005
    requirement: REQ-002
    text: Default scope detects changed files via git merge-base
    tests:
      - TestCodeCheck_DefaultScope_UsesGitMergeBase

  - id: CLM-006
    requirement: REQ-002
    text: Non-git directory falls back to all scope with a warning
    tests:
      - TestCodeCheck_DefaultScope_NonGitFallsBackToAll

  # REQ-003: --all flag
  - id: CLM-007
    requirement: REQ-003
    text: With --all flag, all passes run against the entire codebase
    tests:
      - TestCodeCheck_AllFlag_FullCodebase

  - id: CLM-008
    requirement: REQ-003
    text: With --all flag, git is not invoked for scope determination
    tests:
      - TestCodeCheck_AllFlag_NoGitCall

  # REQ-004: --file flag
  - id: CLM-009
    requirement: REQ-004
    text: --file runs only passes applicable to the file type
    tests:
      - TestCodeCheck_FileFlag_RoutesByType

  - id: CLM-010
    requirement: REQ-004
    text: --file combined with --all produces exit code 2
    tests:
      - TestCodeCheck_FileAndAllConflict_ExitCode2

  # REQ-005: 2-second budget for --file
  - id: CLM-011
    requirement: REQ-005
    text: --file mode completes within 2-second budget under normal conditions
    tests:
      - TestCodeCheck_FileMode_CompletesWithinBudget

  - id: CLM-012
    requirement: REQ-005
    text: --file mode cancels and returns timeout violation when budget exceeded
    tests:
      - TestCodeCheck_FileMode_TimeoutReturnsViolation

  - id: CLM-013
    requirement: REQ-005
    text: Timeout violation has exit code 1
    tests:
      - TestCodeCheck_FileMode_TimeoutExitCode1

  # REQ-006: File-type routing via manifests
  - id: CLM-014
    requirement: REQ-006
    text: Go files route to lint, build, test, and semgrep per manifest
    tests:
      - TestCodeCheck_Routing_GoFileAllPasses

  - id: CLM-015
    requirement: REQ-006
    text: Files not matching any manifest entry receive no checks
    tests:
      - TestCodeCheck_Routing_UnmatchedFileNoChecks

  - id: CLM-016
    requirement: REQ-006
    text: Built-in defaults apply when no manifest files exist
    tests:
      - TestCodeCheck_Routing_DefaultsWhenNoManifest

  - id: CLM-017
    requirement: REQ-006
    text: Manifest path patterns route files to correct check types
    tests:
      - TestCodeCheck_Routing_PathPatternMatching

  # REQ-007: Semgrep auto-install
  - id: CLM-018
    requirement: REQ-007
    text: Semgrep auto-installs to .backstop/tools/ when not on PATH
    tests:
      - TestCodeCheck_Semgrep_AutoInstallWhenMissing

  - id: CLM-019
    requirement: REQ-007
    text: Installed semgrep at wrong version produces exit code 2 (config error, hard stop — not degraded mode)
    tests:
      - TestCodeCheck_Semgrep_WrongVersionExitCode2

  - id: CLM-020
    requirement: REQ-007
    text: Semgrep not installed and auto-install fails skips with warning in degraded mode (not exit code 2)
    tests:
      - TestCodeCheck_Semgrep_InstallFailureDegradedMode

  - id: CLM-021
    requirement: REQ-007
    text: Semgrep on PATH is used directly without auto-install
    tests:
      - TestCodeCheck_Semgrep_UsesPathWhenAvailable

  # REQ-008: Output modes
  - id: CLM-022
    requirement: REQ-008
    text: --json produces structured JSON with schema_version field
    tests:
      - TestCodeCheck_Output_JSONWithSchemaVersion

  - id: CLM-023
    requirement: REQ-008
    text: Human output and JSON output contain identical violation data
    tests:
      - TestCodeCheck_Output_IdenticalViolationData

  - id: CLM-024
    requirement: REQ-008
    text: Human output omits ANSI color codes when NO_COLOR is set
    tests:
      - TestCodeCheck_Output_NoColorRespected

  # REQ-009: Exit codes
  - id: CLM-025
    requirement: REQ-009
    text: Exit code 0 when all checks pass with no violations
    tests:
      - TestCodeCheck_ExitCode_0OnPass

  - id: CLM-026
    requirement: REQ-009
    text: Exit code 1 when one or more violations are found
    tests:
      - TestCodeCheck_ExitCode_1OnViolations

  - id: CLM-027
    requirement: REQ-009
    text: Exit code 2 on configuration error (invalid backstop.yml)
    tests:
      - TestCodeCheck_ExitCode_2OnConfigError

  - id: CLM-028
    requirement: REQ-009
    text: Exit code 2 takes precedence over exit code 1
    tests:
      - TestCodeCheck_ExitCode_2PrecedesOver1

  - id: CLM-029
    requirement: REQ-009
    text: Exit code 2 when --file combined with --all
    tests:
      - TestCodeCheck_ExitCode_2OnFlagConflict

  - id: CLM-030
    requirement: REQ-009
    text: Exit code 2 when .backstop/ directory is missing
    tests:
      - TestCodeCheck_ExitCode_2OnMissingBackstopDir

  # REQ-010: Concurrent invocation safety
  - id: CLM-031
    requirement: REQ-010
    text: Concurrent --file invocations do not corrupt shared state
    tests:
      - TestCodeCheck_Concurrent_NoSharedStateCorruption

  - id: CLM-032
    requirement: REQ-010
    text: Concurrent invocations produce independent output without interleaving
    tests:
      - TestCodeCheck_Concurrent_IndependentOutput

  - id: CLM-033
    requirement: REQ-010
    text: Concurrent semgrep install attempts are idempotent
    tests:
      - TestCodeCheck_Concurrent_SemgrepInstallIdempotent

  # REQ-011: Config loading
  - id: CLM-034
    requirement: REQ-011
    text: backstop.yml is loaded before any checks execute
    tests:
      - TestCodeCheck_Config_LoadedBeforeChecks

  - id: CLM-035
    requirement: REQ-011
    text: Missing backstop.yml produces exit code 2
    tests:
      - TestCodeCheck_Config_MissingYmlExitCode2

  - id: CLM-036
    requirement: REQ-011
    text: BACKSTOP_CONFIG env var overrides walk-up discovery
    tests:
      - TestCodeCheck_Config_EnvVarOverride

  # REQ-013: Changed-files detection (4-step cascade)
  - id: CLM-037
    requirement: REQ-013
    text: Uses git merge-base HEAD origin/main for fork point detection (step 1)
    tests:
      - TestCodeCheck_ChangedFiles_MergeBaseOriginMain

  - id: CLM-038
    requirement: REQ-013
    text: Falls back to origin/master when origin/main does not exist (step 2)
    tests:
      - TestCodeCheck_ChangedFiles_FallbackOriginMaster

  - id: CLM-039
    requirement: REQ-013
    text: Falls back to git diff staged+unstaged with warning when neither remote branch exists (step 3)
    tests:
      - TestCodeCheck_ChangedFiles_FallbackLocalStagedUnstaged

  - id: CLM-040
    requirement: REQ-013
    text: Falls back to --all scope with warning in non-git directory (step 4)
    tests:
      - TestCodeCheck_ChangedFiles_NonGitFallbackToAll

  # REQ-002: Local scope path (staged+unstaged)
  - id: CLM-041
    requirement: REQ-002
    text: Local scope detects staged and unstaged changes when no remote divergence exists
    tests:
      - TestCodeCheck_DefaultScope_LocalStagedAndUnstaged

  # REQ-014: golangci-lint availability
  - id: CLM-042
    requirement: REQ-014
    text: Lint pass is skipped with warning when golangci-lint is not on PATH
    tests:
      - TestCodeCheck_Lint_SkippedWhenGolangciLintMissing

  - id: CLM-043
    requirement: REQ-014
    text: Build and test passes still run when golangci-lint is not on PATH
    tests:
      - TestCodeCheck_Lint_OtherPassesContinueWithoutLint

contracts:
  - file: cmd/backstop/code_check.go
    provides:
      - name: codeCheckCmd
        kind: variable
        signature: "var codeCheckCmd *cobra.Command"
        notes: "Cobra command for backstop code check, registered under the code namespace"
    consumes:
      - source: cmd/backstop
        name: codeCmd
        kind: variable
      - source: pkg/check
        name: Run
        kind: function
      - source: cmd/backstop
        name: loadConfig
        kind: function
      - source: cmd/backstop
        name: formatOutput
        kind: function

  - file: pkg/check/check.go
    provides:
      - name: Run
        kind: function
        signature: "func Run(ctx context.Context, opts Options) (*Result, error)"
        notes: "Executes validation passes against the given scope and returns aggregated results"
      - name: Options
        kind: type
        signature: "type Options struct"
        notes: "Configuration for a check run: scope, file path, manifest path, timeout"
      - name: Result
        kind: type
        signature: "type Result struct"
        notes: "Aggregated results from all validation passes: violations, warnings, pass/fail"
    consumes:
      - source: pkg/check
        name: Manifest
        kind: type

  - file: pkg/check/manifest.go
    provides:
      - name: LoadManifest
        kind: function
        signature: "func LoadManifest(dir string) (*Manifest, error)"
        notes: "Loads compiled enforcement manifests from .backstop/rules/"
      - name: Manifest
        kind: type
        signature: "type Manifest struct"
        notes: "Compiled enforcement manifest mapping file patterns to check types"
      - name: RouteFile
        kind: method
        signature: "func (m *Manifest) RouteFile(path string) []CheckType"
        notes: "Returns applicable check types for a given file path"

  - file: pkg/check/scope.go
    provides:
      - name: ResolveScope
        kind: function
        signature: "func ResolveScope(mode ScopeMode, filePath string) ([]string, []string, error)"
        notes: "Resolves the list of files to check based on mode (diff, all, file). Second return is warnings (e.g., remote fallback notices)."
      - name: ScopeMode
        kind: type
        signature: "type ScopeMode int"
        notes: "Enum: ScopeModeDiff, ScopeModeAll, ScopeModeFile"

  - file: pkg/check/semgrep.go
    provides:
      - name: EnsureSemgrep
        kind: function
        signature: "func EnsureSemgrep(backstopDir string, pinnedVersion string) (string, error)"
        notes: "Returns path to semgrep binary, auto-installing to .backstop/tools/ if needed"
    consumes: []
---

# SPEC-008: Code Check — Implementation Validation with Lint, Build, Test, Semgrep

## Overview

The `backstop code check` command is the implementation validation surface of the
CLI. It runs four validation passes — lint (golangci-lint), build (go build), test
(go test), and semgrep rules — against a determined file scope and reports violations
as structured JSON or human-readable output.

Three scope modes exist:

- **Default (diff):** Changed files since the git merge-base with the default branch.
  This is the fast path for development — only check what you changed.
- **--all:** Full codebase scan. Used in CI or for comprehensive audits.
- **--file \<path\>:** Single-file dispatch for runtime hook integration. Must complete
  within a 2-second budget. Must handle concurrent invocations safely.

File-type routing uses compiled enforcement manifests from `.backstop/rules/` to
determine which passes apply to which files. This means the check surface is
configurable via standards compilation — new language support or custom rules are
added by compiling new standards, not by modifying the CLI.

This command replaces the alpha `backstop-hook` binary and the raw semgrep
invocations in hook scripts.

## Requirements

Requirements are defined in frontmatter.

### Validation Passes

Four passes execute in fixed order (REQ-001): lint, build, test, semgrep. All
passes run regardless of earlier failures. Non-applicable passes are skipped
silently.

| Pass | Tool | Scope applicability |
|------|------|-------------------|
| lint | golangci-lint | .go files |
| build | go build | .go files |
| test | go test | .go files (runs tests in affected packages) |
| semgrep | semgrep | Any file with matching rules in the manifest |

### Scope Modes

| Mode | Flag | Behavior |
|------|------|----------|
| diff (default) | none | Changed files via git merge-base (REQ-002, REQ-013) |
| all | --all | Full codebase, no git (REQ-003) |
| file | --file \<path\> | Single file, 2-second budget (REQ-004, REQ-005) |

The --file and --all flags are mutually exclusive. Specifying both is exit code 2
(REQ-004).

### Changed-Files Detection (REQ-002, REQ-013)

The 4-step detection cascade:

1. `git merge-base HEAD origin/main` then `git diff --name-only <merge-base>..HEAD` — PR scope against main
2. `git merge-base HEAD origin/master` then `git diff --name-only <merge-base>..HEAD` — fallback for repos using master
3. `git diff --name-only` (staged + unstaged local changes) — when neither remote branch exists, with warning
4. Fall back to --all scope — non-git environments (with warning)

Step 3 captures both staged and unstaged changes (equivalent to `git diff HEAD`
for tracked files plus untracked detection). This is the local scope path,
distinct from the PR scope in steps 1-2.

### File-Type Routing (REQ-006)

Compiled enforcement manifests in `.backstop/rules/` map file extensions and path
patterns to check types. When no manifests exist, built-in defaults apply:

| File type | Default checks |
|-----------|---------------|
| .go | lint, build, test, semgrep |
| other | semgrep (if matching rules exist) |

### Semgrep Auto-Install (REQ-007)

If semgrep is not on PATH:

1. Download to `.backstop/tools/semgrep`
2. Pin version per backstop.yml
3. Verify version on subsequent runs

Two distinct failure modes:

- **Version mismatch** (installed but wrong version): exit code 2 (configuration error, hard stop). This indicates deliberate misconfiguration.
- **Install failure** (not installed, auto-install fails due to network/download error): skip semgrep checks with a warning (degraded mode). Do not fail the entire command.

### Tool Availability (REQ-014)

If golangci-lint is not found on PATH, the lint pass is skipped with a warning
(same degradation pattern as semgrep install failure). The `go` and `go test`
tools are assumed available — backstop operates within a Go project, so Go
being installed is a precondition, not a graceful-degradation scenario.

### Exit Codes (REQ-009)

| Code | Meaning |
|------|---------|
| 0 | All checks pass, no violations |
| 1 | One or more violations found |
| 2 | Configuration error (invalid config, flag conflict, missing .backstop/) |

Exit code 2 takes precedence over exit code 1.

## Implementation

### Command Registration (cmd/backstop/code_check.go)

The `codeCheckCmd` Cobra command is registered under the `codeCmd` namespace group.
It accepts `--all`, `--file`, and `--json` flags. The command handler:

1. Validates flag combinations (--file and --all are mutually exclusive)
2. Loads backstop.yml via the shared config loader from SPEC-005
3. Constructs a `check.Options` struct from flags and config
4. Calls `check.Run(ctx, opts)` with a context that carries the 2-second timeout
   when --file is specified
5. Formats the result via the shared output formatter
6. Sets the exit code based on the result

### Validation Engine (pkg/check/check.go)

The `Run` function orchestrates all validation passes:

1. **Scope resolution** — calls `ResolveScope` to determine the file list
2. **Manifest loading** — calls `LoadManifest` to load routing rules
3. **Semgrep availability** — calls `EnsureSemgrep` to locate or install semgrep
4. **Pass execution** — for each pass (lint, build, test, semgrep) in order:
   - Check if the pass applies to any files in scope (via manifest routing)
   - If applicable, execute the pass and collect violations
   - If not applicable, skip silently
   - If context is cancelled (timeout), stop and record a timeout violation
5. **Result aggregation** — merge violations from all passes into a single Result

### Scope Resolution (pkg/check/scope.go)

`ResolveScope` handles three modes:

- `ScopeModeDiff`: Execute the git merge-base cascade (REQ-013)
- `ScopeModeAll`: Walk the project directory for all relevant files
- `ScopeModeFile`: Return the single file path (after verifying it exists)

### Manifest Loading (pkg/check/manifest.go)

`LoadManifest` reads compiled enforcement manifests from `.backstop/rules/`:

- Parses `.manifest.json` files to build the routing table
- `RouteFile` maps a file path to applicable check types using extension matching
  and path pattern matching
- Returns built-in defaults when no manifest files exist

### Semgrep Management (pkg/check/semgrep.go)

`EnsureSemgrep` handles the semgrep lifecycle:

- Check PATH for an existing semgrep binary
- If not found, download to `.backstop/tools/semgrep`
- Verify the installed version matches the backstop.yml pin
- Use file-level locking for concurrent install safety

### Concurrent Safety (REQ-010)

The --file mode design ensures concurrent safety:

- Each invocation constructs its own `Options` and `Result` — no shared mutable state
- Manifest loading is read-only after initialization
- Semgrep auto-install uses a lockfile (`.backstop/tools/.semgrep.lock`) to serialize
  concurrent install attempts
- Output goes to stdout per-invocation; no shared log file

## Verification

Claims are defined in frontmatter. Integration-level verification with 80% coverage
threshold. Tests use the `TestCodeCheck` prefix and cover all requirements through
positive and negative claims.

## Sharp Edges

- **Semgrep version drift across team members.** If backstop.yml does not pin a
  semgrep version, different developers may have different semgrep versions on PATH,
  producing inconsistent results. The version pin in backstop.yml mitigates this,
  but only if all developers use auto-install rather than their own semgrep. A CI
  environment should always use auto-install for reproducibility.

- **2-second budget is aggressive for large files.** A .go file that triggers lint,
  build, test, and semgrep in sequence may exceed 2 seconds on slow machines.
  The timeout returns a violation rather than blocking, which means hooks may
  report false negatives (timeout instead of real violations). The mitigation is
  that --file routes by manifest and may skip passes, and that the full check
  (without --file) catches everything.

- **Merge-base detection assumes origin remote.** The cascade uses origin/main
  and origin/master. Repos using a different remote name (e.g., upstream) or
  different default branch name (e.g., develop) will fall back to local diff.
  A future enhancement could read the default branch from git config or
  backstop.yml.

- **Concurrent auto-install race condition.** Even with file locking, the first
  invocation to acquire the lock downloads semgrep while others wait. If the
  lock holder is killed mid-download, a partial binary may exist. The install
  function should download to a temp file and atomically rename on completion.

- **Manifest-less default routing is opinionated.** The built-in defaults (Go
  files get all four passes, others get semgrep only) assume a Go project.
  Multi-language projects without manifests will get incomplete coverage.
  The correct fix is to compile standards and produce manifests — the defaults
  are a bootstrapping convenience, not a permanent solution.

- **cmd/ files must be thin adapters.** The design requires that no enforcement
  logic reside in cmd/ — the command should parse flags, load config, delegate
  to pkg/check, format results, and set exit codes. Enforcement of this
  constraint is a code review concern, not a test assertion — there is no
  reliable automated way to distinguish "enforcement logic" from "glue code"
  in a thin adapter. Reviewers must verify this during implementation review.

- **go test scope explosion.** When checking changed .go files, the test pass
  must determine which packages to test. A change to a widely-imported package
  could trigger tests across the entire dependency tree, blowing the 2-second
  budget in --file mode. The implementation should limit test scope to the
  changed file's own package in --file mode.

## Review Questions

1. Does the implementation use `context.WithTimeout` (not a manual timer) for the 2-second budget, and does it propagate the context to all subprocess invocations (golangci-lint, go build, go test, semgrep)?

2. When `EnsureSemgrep` downloads to a temp file and renames atomically, does it handle the case where `.backstop/tools/` does not yet exist (first-ever invocation)?

3. Does the concurrent lockfile implementation use `os.OpenFile` with `O_CREATE|O_EXCL` or `flock(2)` — and does it handle stale locks from killed processes?

4. In --file mode, does the test pass scope itself to the single file's package only, or does it run all tests in the repository?

5. Does the merge-base cascade correctly handle detached HEAD states (common in CI environments)?

6. When semgrep auto-install fails and is skipped with a warning, does the warning appear in both JSON and human output modes, and does it not cause exit code 2?

7. Does `RouteFile` correctly handle files with multiple extensions (e.g., `foo.test.go`) or no extension?

## References

- Bundle: cli (spec seed 5 — backstop code check)
- Bundle requirements: cli:REQ-003, cli:REQ-010
- SPEC-005: CLI Foundation (config loading, output formatting, exit codes)
- ADR-0014: Runtime Hooks
- D-037: Changed-files detection
- D-069: CLI as universal agent API
- D-099: Embedded baseline rule pack
- OQ-9 (resolved): Semgrep auto-install to .backstop/tools/
