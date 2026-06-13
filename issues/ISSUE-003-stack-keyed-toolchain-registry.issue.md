---
title: "Data-driven toolchain registry — native enforcement for all stacks"
schema_version: issue/v1

issue:
  id: ISSUE-003
  title: "Data-driven toolchain registry — native enforcement for all stacks"
  type: enhancement
  status: closed
  created: "2026-06-11"
  closed: "2026-06-11"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate

verification:
  level: unit
  coverage_threshold: 90
  test_command: "go test ./pkg/check/..."

implementation:
  summary: >
    Replace the hardwired pass→tool bindings with a registry where a stack
    is data, not code: per pass, a command plus a named output format from
    a small parser library. Go and TypeScript ship as built-in registry
    entries; any other stack is declarable in backstop.yml with the same
    shape. Stack selected by language; missing toolchain for the declared
    language is a config error. Scope semantics are per-pass (lint via
    file args, build/typecheck project-wide and never scope-filtered,
    tests dependency-mapped with full-suite fallback).
  package: pkg/check

requirements:
  - id: REQ-001
    text: >
      Executor bindings must be a registry keyed by (stack, pass),
      selected by the backstop.yml language field, with Go as the default
      when language is absent. Built-in stacks are predefined registry
      entries with the same shape as declared ones.
  - id: REQ-002
    text: >
      A stack must be declarable as data in backstop.yml: per pass, a
      command and a named output format. A declared stack gets full
      lint/build/test enforcement without any backstop code changes.
  - id: REQ-003
    text: >
      A TypeScript built-in toolchain must ship: eslint (JSON output),
      tsc --noEmit, and a test command explicitly declared in
      backstop.yml — no package.json detection.
  - id: REQ-004
    text: >
      A parser library must translate tool output to violations via named
      formats: the built-in tool formats (golangci-json, go-build,
      go-test, eslint-json, tsc) plus two generic formats — sarif and
      regex-lines — so arbitrary tools are consumable without new code.
  - id: REQ-005
    text: >
      Manifest routing must route a stack's declared extensions to
      lint/build/test/semgrep the way .go routes today; built-in TS
      covers .ts/.tsx, declared stacks declare their extensions.
  - id: REQ-006
    text: >
      A missing toolchain for the project's declared language must be a
      config error (exit 2), not skip-with-warning. Scope semantics are
      per-pass: lint passes scoped files as arguments; build/typecheck
      runs project-wide with violations never scope-filtered (baseline
      comparison is the suppression mechanism); tests use dependency
      mapping with full-suite fallback.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: the registry selects the toolchain by backstop.yml language, defaulting to Go.
    tests:
      - TestCodeCheck_Registry_SelectsToolchainByLanguage
  - id: CLM-002
    requirement: REQ-002
    text: a custom stack declared in backstop.yml (command + format per pass) produces working executors with no code changes.
    tests:
      - TestCodeCheck_Registry_CustomToolchainFromConfig
  - id: CLM-003
    requirement: REQ-003
    text: TS executors parse eslint JSON and tsc output into violations with correct file/line/message/severity.
    tests:
      - TestCodeCheck_TSExecutors_ParseESLintJSON
      - TestCodeCheck_TSExecutors_ParseTscOutput
  - id: CLM-004
    requirement: REQ-003
    text: the TS test pass requires an explicitly declared test command and config-errors without one.
    tests:
      - TestCodeCheck_TSTestCommand_ExplicitDeclarationRequired
  - id: CLM-005
    requirement: REQ-004
    text: the sarif and regex-lines generic formats parse arbitrary tool output into violations.
    tests:
      - TestCodeCheck_Parsers_SarifFormat
      - TestCodeCheck_Parsers_RegexLinesFormat
  - id: CLM-006
    requirement: REQ-005
    text: .ts/.tsx files route to lint/build/test/semgrep; a declared stack's extensions route equivalently.
    tests:
      - TestCodeCheck_Routing_TSFilesRouteAllPasses
      - TestCodeCheck_Routing_DeclaredStackExtensionsRoute
  - id: CLM-007
    requirement: REQ-006
    text: a missing toolchain for the declared language is an exit-2 config error.
    tests:
      - TestCodeCheck_MissingToolchain_DeclaredLanguageIsConfigError
  - id: CLM-008
    requirement: REQ-006
    text: lint passes receive scoped file args while build/typecheck always runs project-wide with unfiltered reporting.
    tests:
      - TestCodeCheck_ScopeSemantics_LintFileArgsBuildProjectWide

contracts:
  - file: pkg/check/check.go
    consumes:
      - source: pkg/config/config.go
        name: Config
        kind: type
---

# Data-driven toolchain registry — native enforcement for all stacks

## Problem

The code-check pass vocabulary is a closed enum hardwired to Go tools: lint=golangci-lint,
build=go build, test=go test, plus semgrep (`pkg/check/manifest.go:14-23`).
`buildDefaultExecutors` (`pkg/check/check.go:309-316`) returns
`map[CheckType]PassExecutor` with only those bindings. A TypeScript project gets only
the semgrep pass — no native lint, typecheck, or test enforcement.

`backstop.yml` already declares `language` and `runtimes` (`pkg/config/config.go:22-23`)
but nothing in enforcement consumes them. The `language:` field exists and is loaded but
has no effect on which executors are selected.

`routeFileDefaults` (`pkg/check/manifest.go:134-141`) special-cases `.go` to receive all
four passes. Every other extension — including `.ts` and `.tsx` — falls through to
semgrep-only, regardless of what `language:` declares.

## Impact

A TypeScript project using backstop gets no native linting, no type-checking, and no test
enforcement through the gate. The semgrep pass fires but that only covers layer-2 rule
matching. Lint and build violations in TS code are silently skipped — the gate exits clean
on code that would fail `tsc` or `eslint`. This is silent non-enforcement of the primary
language.

## Solution

### 1. Toolchain registry — a stack is data, not code

Replace the flat `map[CheckType]PassExecutor` returned by `buildDefaultExecutors` with a
registry keyed by `(stack, pass)`. The pass vocabulary
(lint / build / test / semgrep) remains as the semantic layer; a registry entry is
**data**: a command plus a named output format. Built-in stacks (go, typescript) are
predefined entries of exactly the same shape; any other stack is declarable in
`backstop.yml`:

```yaml
language: rust
enforcement:
  toolchain:
    lint:  {command: "cargo clippy --message-format json", format: regex-lines}
    build: {command: "cargo build", format: regex-lines}
    test:  {command: "cargo test", format: regex-lines}
```

The goal is all stacks, not a per-language whitelist: TypeScript is the second
built-in because it forces the abstraction to be real, not because it is the target.
A declared stack gets full native enforcement with zero backstop code changes.

Stack is selected by `language:` in `backstop.yml` at `Check` invocation time, falling
back to the Go stack when `language:` is absent (preserving current behavior).

### 1b. Output parser library

The only stack-specific code anywhere is output parsing, so parsers are a small named
library selectable per pass declaration: built-in tool formats (`golangci-json`,
`go-build`, `go-test`, `eslint-json`, `tsc`) plus two generic formats that unlock
arbitrary tools — `sarif` (the static-analysis interchange format most modern tools
can emit) and `regex-lines` (configurable `file:line:col message` line matching).

### 2. TypeScript toolchain

Three executor bindings for the `typescript` stack:

- **lint**: `eslint` with JSON formatter output (`--format json`)
- **build**: `tsc --noEmit` (with `--incremental` / `--build` mode for warm-cache speed;
  always project-wide — see constraint 3 below)
- **test**: command explicitly declared in `backstop.yml` under a new
  `enforcement.test_command:` field — no package.json detection magic (constraint 1)

### 3. Manifest routing

`routeFileDefaults` extended to route `.ts` / `.tsx` to
`[lint, build, test, semgrep]` — matching the treatment `.go` receives today.
The routing is stack-agnostic at the manifest layer; the executor registry handles
the stack-specific binding at execution time.

### 4. Go toolchain as registry entries

The Go executor implementations created by ISSUE-002 become the `go` stack entries in the
registry. No behavioral change — they move from hardwired defaults to named registry slots.

## Ratified Design Constraints

These decisions are settled. Implementors must not re-open them.

1. **TS test command is explicitly declared.** `enforcement.test_command:` in backstop.yml.
   No package.json detection — detection fails silently, and load-bearing rules must be
   enforced, not inferred.

2. **Scope semantics are per-pass, an invocation concern.** Lint passes scope by passing
   the scoped file list as tool arguments. Build/typecheck passes always run project-wide.
   Test passes use dependency mapping (Go: changed-file → package mapping as the coverage
   step does today; TS: `vitest --related` / `jest --findRelatedTests` or equivalent) as a
   per-toolchain knob, with full-suite fallback.

3. **Build/typecheck violations are never scope-filtered.** A change to `a.ts` that breaks
   a type in unchanged `b.ts` must fail a diff-scoped gate. Pre-existing project-wide
   violations are suppressed by the existing baseline-comparison step (gate step 7), not by
   file filtering. Build-pass violations are exempt from scoped-files assertion logic because
   they may legitimately reference out-of-scope files.

4. **Missing toolchain for the declared language is a config error (exit code 2).** A
   silently skipped lint pass for the primary language is silent non-enforcement.

5. **Pack ToolConfig execution is out of scope.** Packs shipping eslint or golangci configs
   via `tool_config` entries are not applied at gate time. That belongs to a future
   engine-generalization bundle. Pack policy enforcement in the meantime is layer-2 semgrep
   rules + layer-3 validators.

6. **Traceability steps stay Go-only.** Coverage thresholds (`pkg/gate/step_coverage.go`),
   test substantiveness, and contract signatures remain Go-bound and are explicitly out of
   scope here.

## Dependencies

Depends on **ISSUE-002** (implement the stubbed Go executors) landing first. This issue
generalizes the bindings ISSUE-002 creates into a stack-keyed registry; ISSUE-002's
concrete executor implementations become the `go` stack entries.

## References

- `pkg/check/manifest.go` — CheckType enum (lines 14-23), routeFileDefaults (lines 134-141)
- `pkg/check/check.go` — buildDefaultExecutors (lines 309-316)
- `pkg/config/config.go` — Config.Language / Config.Runtimes fields (lines 22-23)
- ISSUE-002 (code-check executors are stubs — dependency)
