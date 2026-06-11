---
title: "Stack-keyed toolchain registry + TypeScript native toolchain"
schema_version: issue/v1

issue:
  id: ISSUE-003
  title: "Stack-keyed toolchain registry + TypeScript native toolchain"
  type: enhancement
  status: open
  created: "2026-06-11"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate
---

# Stack-keyed toolchain registry + TypeScript native toolchain

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

### 1. Toolchain registry

Replace the flat `map[CheckType]PassExecutor` returned by `buildDefaultExecutors` with a
registry keyed by `(stack, pass)` → executor. The pass vocabulary
(lint / build / test / semgrep) remains as the semantic layer; the registry maps each
combination to the concrete command and output parser for that stack.

Stack is selected by `language:` in `backstop.yml` at `Check` invocation time.
`buildDefaultExecutors` (or its replacement) reads `cfg.Language` and returns only the
executor bindings appropriate for that stack, falling back to the Go stack when `language:`
is absent (preserving current behavior).

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
