---
title: "code-check pass executors are stubs — lint/build/test/semgrep never execute"
schema_version: issue/v1

issue:
  id: ISSUE-002
  title: "code-check pass executors are stubs — lint/build/test/semgrep never execute"
  type: technical-debt
  status: obsoleted
  created: "2026-06-11"
  closed: "2026-06-11"

obsoleted-by: ISSUE-018

complexity:
  scope: contained
  uncertainty: known
  risk: moderate

verification:
  level: unit
  coverage_threshold: 90
  test_command: "go test ./pkg/check/..."

implementation:
  summary: >
    Implement the four pass executors in pkg/check for real: golangci-lint
    (JSON output), go build (compiler errors), go test (failures), and
    semgrep (ensured binary, compiled project rules plus pack layer-2
    ExtraSemgrepConfigs, JSON output with pack-namespaced violations).
    Preserve engine semantics: timeout/cancellation, skip-with-warning on
    unavailable tools, no short-circuiting between passes. Executors take
    an injectable command runner so unit tests run against fixture output,
    not live tools.
  package: pkg/check

requirements:
  - id: REQ-001
    text: >
      The lint pass must execute golangci-lint against scoped files and
      convert its JSON findings into check violations with file, line,
      message, and severity.
  - id: REQ-002
    text: >
      The build pass must execute go build and convert compiler errors
      into check violations.
  - id: REQ-003
    text: >
      The test pass must execute go test and convert test failures into
      check violations, honoring file-mode scoping.
  - id: REQ-004
    text: >
      The semgrep pass must execute the EnsureSemgrep-resolved binary with
      the compiled project rules and all pack-provided ExtraSemgrepConfigs,
      convert JSON findings into violations, and preserve pack-namespaced
      rule IDs so violations are attributable to their source pack.
  - id: REQ-005
    text: >
      Real executors must preserve existing engine semantics: context
      cancellation surfaces as a timeout violation, an unavailable tool
      skips its pass with a warning, and a failing pass does not
      short-circuit later passes.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: lintExecutor parses golangci-lint JSON output into violations with correct file, line, message, and severity.
    tests:
      - TestCodeCheck_LintExecutor_ParsesGolangciJSON
  - id: CLM-002
    requirement: REQ-001
    text: lintExecutor returns a passing result with zero violations on clean lint output.
    tests:
      - TestCodeCheck_LintExecutor_CleanOutputNoViolations
  - id: CLM-003
    requirement: REQ-002
    text: buildExecutor parses go build compiler errors into violations.
    tests:
      - TestCodeCheck_BuildExecutor_ParsesCompileErrors
  - id: CLM-004
    requirement: REQ-003
    text: testExecutor parses go test failures into violations and scopes to a single file's package in file mode.
    tests:
      - TestCodeCheck_TestExecutor_ParsesTestFailures
      - TestCodeCheck_TestExecutor_FileModeScopesToPackage
  - id: CLM-005
    requirement: REQ-004
    text: semgrepExecutor invokes the ensured binary with project rules plus all ExtraSemgrepConfigs and parses JSON findings into violations.
    tests:
      - TestCodeCheck_SemgrepExecutor_RunsProjectAndPackConfigs
  - id: CLM-006
    requirement: REQ-004
    text: semgrep violations originating from pack rules carry their pack-namespaced rule IDs.
    tests:
      - TestCodeCheck_SemgrepExecutor_PreservesPackNamespacedRuleIDs
  - id: CLM-007
    requirement: REQ-005
    text: a cancelled context surfaces as a timeout violation and an unavailable tool skips its pass with a warning, unchanged from stub-era engine behavior.
    tests:
      - TestCodeCheck_Executors_ContextCancellationSurfacesTimeout
      - TestCodeCheck_Executors_UnavailableToolSkipsWithWarning

contracts:
  - file: pkg/check/check.go
    consumes:
      - source: pkg/check/semgrep.go
        name: EnsureSemgrep
        kind: function
      - source: pkg/check/manifest.go
        name: RouteFile
        kind: method
---

# code-check pass executors are stubs — lint/build/test/semgrep never execute

## Problem

All four default pass executors in `pkg/check/check.go` — `lintExecutor`,
`buildExecutor`, `testExecutor`, and `semgrepExecutor` — have `Execute()`
methods that return an empty `&PassResult{Pass: ct}` without invoking any
tool. The engine dispatches them at `check.go:149` (`executor.Execute`).

```go
// lintExecutor — lines 321-323
func (e *lintExecutor) Execute(ctx context.Context, files []string) (*PassResult, error) {
    return &PassResult{Pass: CheckTypeLint}, nil
}

// buildExecutor — lines 337-339
func (e *buildExecutor) Execute(ctx context.Context, files []string) (*PassResult, error) {
    return &PassResult{Pass: CheckTypeBuild}, nil
}

// testExecutor — lines 350-352
func (e *testExecutor) Execute(ctx context.Context, files []string) (*PassResult, error) {
    return &PassResult{Pass: CheckTypeTest}, nil
}

// semgrepExecutor — lines 365-367
func (e *semgrepExecutor) Execute(ctx context.Context, files []string) (*PassResult, error) {
    return &PassResult{Pass: CheckTypeSemgrep}, nil
}
```

Consequence: the lint, build, test, and semgrep passes in both `backstop
check` and `backstop gate` step 2 (`code_check`) always pass vacuously.
Step 2 is silent non-enforcement.

The supporting machinery is real and currently wasted:

- `EnsureSemgrep` in `pkg/check/semgrep.go` pip-installs a pinned semgrep
  binary into `.backstop/tools/`.
- `mergePackRules` in `cmd/backstop/pack_gate.go:97-123` collects installed
  packs' layer-2 rule files into `check.Options.ExtraSemgrepConfigs`. The
  stub `semgrepExecutor` stores these in its `extraSemgrepConfigs` field but
  never passes them to a process.
- Pack layer-2 semgrep rules are therefore never enforced.

Manifest routing in `pkg/check/manifest.go` is also real and correct: the
default manifest routes `.go` files to lint/build/test/semgrep and all other
files to semgrep-only (`routeFileDefaults`, lines 133-141). `ScopeModeAll`
walks the full project tree, so all language files reach the semgrep pass
once it is real.

## Impact

This is the primary enforcement step of the gate. A green `backstop gate`
currently means all lint, build, test, and semgrep passes were skipped
rather than passed. The TypeScript-pack story is also blocked: a TS pack's
layer-2 semgrep rules can only be enforced once `semgrepExecutor` shells out.

## Solution

Implement the four executors for real. Existing engine contracts must be
preserved: timeout/cancellation handled at `check.go:100-180`, skip-with-
warning when the tool is unavailable (`IsAvailable` already gates dispatch),
no short-circuiting between passes.

**lintExecutor** — shell out to `golangci-lint` with `--out-format json`,
parse each finding into `check.Violation{Pass, File, Line, Message, Severity}`.
`IsAvailable` already calls `findExecutable("golangci-lint")` on PATH.

**buildExecutor** — run `go build ./...` in the project root, parse compiler
error lines (`file:line:col: message`) into Violations. Go is assumed
available (`IsAvailable` returns true unconditionally — no change needed).

**testExecutor** — run `go test` against affected packages (or a single file
in file-mode via the existing `fileMode bool` field), parse test failure
output into Violations. Go is assumed available.

**semgrepExecutor** — invoke the binary resolved by `EnsureSemgrep` with
`--config .backstop/rules` plus all paths in `extraSemgrepConfigs`, `--json`
output, parse findings into Violations. Violations from pack rules should
set `SourcePack` (the gate `Violation` type at `pkg/gate/result.go:41` has
this field; rule IDs are pack-namespaced via `pack.NamespacedRuleID`). Note:
`check.Violation` and `gate.Violation` are distinct types — the gate step
that bridges them will need to map `SourcePack` attribution across.

## References

- `pkg/check/check.go` — executor stubs at lines 318-371; engine dispatch at line 149
- `pkg/check/semgrep.go` — `EnsureSemgrep`; real install logic already present
- `pkg/check/manifest.go` — `routeFileDefaults`; routing is correct and unchanged
- `cmd/backstop/pack_gate.go:97-123` — `mergePackRules`; populates `ExtraSemgrepConfigs` that stubs ignore
- `pkg/gate/result.go:41` — gate `Violation.SourcePack` field
- `pkg/pack/coordinate.go:42` — `pack.NamespacedRuleID`
- ISSUE-003 (follow-up): stack-keyed toolchain registry + TypeScript toolchain — generalizes the executor binding this issue implements; this issue should be resolved before ISSUE-003 is started
