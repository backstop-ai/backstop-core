---
title: Runtime Hook Integration — Agent-First Enforcement at Code Generation Time
schema_version: bundle/v1

bundle:
  name: runtime-hooks
  version: "0.2.0"
  created: "2026-03-30"
  updated: "2026-03-30"
  category: feature

status:
  maturity: defined

problem:
  summary: >
    Backstop's enforcement model requires intercepting agent tool use at code
    generation time — before violations land on disk and compound. ADR-0014
    defines a runtime-agnostic compile-target model where enforcement logic is
    defined once and emitted to each supported agent runtime. Currently no
    runtime integration exists. The two target runtimes (Claude Code and GitHub
    Copilot CLI via copilot-sdk) expose functionally equivalent hook primitives
    but with different wire formats: Claude Code uses declarative JSON config
    with shell-script hooks that receive JSON on stdin, while Copilot SDK uses
    programmatic TypeScript functions in an extension entry point. A common
    enforcement contract must abstract over both.

  user_story: >
    As a developer using backstop with any supported agent runtime, I want
    enforcement checks to fire automatically when the agent writes or edits
    files — without me configuring hooks manually, reminding the agent to run
    checks, or reviewing output. I run backstop init, declare my runtimes,
    and enforcement is wired in. Violations are caught and fed back to the
    agent as context so it self-corrects before I ever see the code.

  success_criteria:
    - PostToolUse hooks fire automatically on every file write/edit
    - Code file violations are detected via compiled semgrep rules from .backstop/rules/
    - Artifact file violations are detected via the existing pkg/validate validators
    - Violations are injected as agent context for self-correction
    - File scope enforcement blocks writes outside the active plan task's file set
    - Context survives Claude Code compaction via PostCompact re-injection
    - Hook configuration merges cleanly with user-defined hooks
    - Alpha hooks work without the full CLI being built

solution:
  approach: >
    A runtime provider model where backstop defines an enforcement contract
    (what checks to run, when, what to do with results) and compiles it to
    runtime-specific configurations. Each runtime provider emits the hook
    wiring for its target. For alpha, only the Claude Code provider is
    implemented. Alpha hooks use a hybrid execution model: raw tool invocation
    for code files (semgrep against compiled rule manifests) and a thin Go
    binary (cmd/backstop-hook) for artifact validation. Both are explicitly
    disposable scaffolding, replaced wholesale when backstop check ships in
    the CLI. The provider interface is designed to accommodate Copilot SDK
    as a future provider without implementation in v1.

  assumptions:
    - Claude Code hook system is stable and supports all documented events
    - semgrep is installed and available on PATH in developer environments
    - golangci-lint is installed for Go projects
    - The compiled enforcement manifests in .backstop/rules/ are up to date
    - Plan artifacts have passed D-081 validation (disjoint file sets)
---

# Runtime Hook Integration — Agent-First Enforcement at Code Generation Time

## Current Thinking

### The Common Hook Surface

Analysis of both runtimes confirms ADR-0014's prediction. Both expose three
equivalent primitives under different wire formats:

| Primitive | Claude Code | Copilot SDK |
|-----------|-------------|-------------|
| System context injection | CLAUDE.md + SessionStart hook | systemMessage + onSessionStart |
| Pre-tool gate | PreToolUse (exit 2 blocks) | onPreToolUse (allow/deny/ask) |
| Post-tool action | PostToolUse (additionalContext) | onPostToolUse (additionalContext) |

Additional shared surface:
- User prompt interception (UserPromptSubmit / onUserPromptSubmitted)
- Session end (SessionEnd / onSessionEnd)
- Error handling (PostToolUseFailure / onErrorOccurred)
- Permission control (PermissionRequest / onPermissionRequest)
- MCP server support (both)
- Skills/custom commands (both via SKILL.md)

### What Differs

**Wire format:** Claude Code hooks are shell commands that read JSON from stdin
and write JSON to stdout with exit codes (0=ok, 2=block). Copilot SDK hooks are
TypeScript async functions that return typed objects.

**Configuration model:** Claude Code is declarative JSON in .claude/settings.json
with regex matchers. Copilot SDK is programmatic — all config at session creation
in TypeScript.

**Hook granularity:** Claude Code has 24 event types with fine-grained matchers
(FileChanged, PreCompact, PostCompact, ConfigChange, etc.). Copilot SDK has 6
core lifecycle hooks covering the essentials.

**Subagent model:** Claude Code spawns separate processes with isolated context
windows. Copilot SDK switches agent modes within the same session.

**Context re-injection:** Claude Code has PreCompact/PostCompact hooks for
surviving context compression. Copilot SDK has no equivalent — context injection
must happen through other hook events.

### Enforcement Contract

The runtime-agnostic enforcement contract defines five hook points:

1. **SessionStart**: Inject backstop project context. The agent reads the active
   artifacts (specs, plans, directives) directly — they are machine-readable by
   design. The hook tells the agent where to find them and what workflow contract
   to follow.

2. **PostToolUse (file writes)**: After Edit/Write/create/edit tool use, run
   checks against the written file. Route by file type using the compiled
   enforcement manifests in .backstop/rules/ as the source of truth for code
   files, and the backstop-hook artifact validator for artifact files. Inject
   violations as additionalContext so the agent self-corrects.

3. **PostToolUse (test runs)**: After Bash tool use that looks like a test
   command (go test, npm test, pytest, etc.), capture results and check against
   spec claims — are mandated tests passing? Is coverage met?

4. **PreToolUse (file writes)**: Gate writes to files outside the current plan
   task's file scope. Hard block — the plan passed D-081 validation so file
   scopes are disjoint by construction. Any out-of-scope write is a violation.

5. **PostCompact (Claude Code specific)**: Re-inject dynamic session state —
   active directive, current plan task, file scope, progress. Read from
   .backstop/session-state.json which the agent maintains during execution.

### The Provider Interface

Each runtime provider implements:

```
type RuntimeProvider interface {
    // Name returns the provider identifier (e.g., "claude-code", "copilot-cli")
    Name() string

    // Generate produces runtime-specific configuration files
    Generate(config RuntimeConfig) ([]GeneratedFile, error)

    // Validate checks that generated config is still in sync with backstop state
    Validate(projectRoot string) error
}
```

RuntimeConfig contains:
- Project root path
- Active enforcement rules (from compiled manifests)
- Active spec/plan (if any)
- Hook behavior preferences (verbosity)
- File scope constraints (from plan task)

GeneratedFile contains:
- Path (relative to project root)
- Content (bytes)
- Managed flag (backstop owns this file — regenerated on sync)

### Claude Code Provider Output

```
.claude/
  settings.json             ← hook entries merged into existing config
  rules/
    backstop.md             ← stable backstop instructions for the agent
  hooks/
    backstop-post-edit.sh   ← PostToolUse handler for file writes
    backstop-post-test.sh   ← PostToolUse handler for test runs
    backstop-session.sh     ← SessionStart context injection
    backstop-pre-edit.sh    ← PreToolUse file scope gate
    backstop-compact.sh     ← PostCompact state re-injection
```

### Alpha Execution Model

For the alpha, hooks bypass the CLI entirely:

**Code files** — Hook scripts invoke semgrep directly against the compiled
rule files in .backstop/rules/. The manifest's `language` field maps file
extensions to the appropriate .semgrep.yml config. golangci-lint runs for
Go files. No hardcoded language-to-tool mapping — the compiler output is the
source of truth.

**Artifact files** — A thin cmd/backstop-hook/main.go binary (~30 lines)
imports pkg/validate, pkg/schema, and pkg/artifact directly. Takes a file
path, routes by artifact extension, runs the appropriate validator, outputs
JSON. Gets absorbed into backstop check when the CLI ships.

**Session state** — .backstop/session-state.json tracks the active directive,
current plan task, file scope, and progress. The PostCompact hook reads this
file and re-injects it as agent context.

Both the hook scripts and the backstop-hook binary are explicitly disposable
scaffolding. They are replaced wholesale when backstop check ships.

### Resolved Design Questions

**OQ-1: Hook script execution model** → Hybrid. Alpha hooks call raw tools
directly for code files (semgrep against compiled manifests, golangci-lint)
and a thin cmd/backstop-hook/main.go binary for artifact validation. The hook
binary imports pkg/validate directly — no new abstractions, just a main
function that routes by file extension. Explicitly disposable; replaced by
backstop check when the CLI ships.

**OQ-2: Merge strategy for .claude/settings.json** → Merge with path-convention
identification. Backstop-managed hooks point to .claude/hooks/backstop-* scripts.
During sync, entries matching that path prefix are replaced; all other user hooks
are preserved. The file path convention is the marker — no special comment syntax
or metadata needed.

**OQ-3: What checks run per file type** → File-type dispatch using compiled
enforcement manifests as the source of truth. The manifest's language field maps
file extensions to semgrep rule configs. Artifact files route to the backstop-hook
validator. No hardcoded language-to-tool mapping. The compiler already did the
work; hooks consume its output.

**OQ-4: Copilot SDK extension build/distribution** → Out of scope for alpha.
The runtime provider interface is designed to accommodate Copilot SDK as a
future provider, but no extension.mjs generation or Copilot-specific code ships
in v1. Claude Code is the only implemented provider.

**OQ-5: Context re-injection after compaction** → Layered. .claude/rules/backstop.md
carries stable project-level instructions that survive compaction natively.
PostCompact hook re-reads dynamic session state from .backstop/session-state.json
(active directive, current plan task, file scope, progress) and re-injects it.
The agent updates session-state.json as it progresses through plan tasks.

**OQ-6: Should provider generate system prompt content** → The provider generates
.claude/rules/backstop.md that teaches the agent how to work within backstop —
where artifacts live, how to read them, what hooks enforce, the workflow contract.
It does NOT duplicate spec/plan/directive content. The artifacts are the source of
truth — machine-readable by design (zeroth principle). PostCompact hook points
the agent back to the active artifacts to re-read.

**OQ-7: File scope enforcement strictness** → Hard block. If the plan passed
D-081 validation, file scopes are disjoint by construction. Any write outside
the current task's file set is a violation — denied via PreToolUse exit 2. The
plan is the contract; the hook enforces it.

## Design Decisions

- **DD-1:** Alpha hybrid execution — raw tool invocation for code files (semgrep,
  golangci-lint), thin Go binary (cmd/backstop-hook) for artifact validation.
  Both disposable; replaced by backstop check when CLI ships.
- **DD-2:** File-type dispatch using compiled enforcement manifests as source of
  truth. Manifest language field maps extensions to semgrep configs. No hardcoded
  language-to-tool mapping.
- **DD-3:** Post-write context injection for code violations — inject as agent
  context for self-correction. Pre-write hard blocking for file scope violations —
  the plan contract is non-negotiable.
- **DD-4:** Merge-with-path-convention for settings.json — backstop hooks use
  .claude/hooks/backstop-* path prefix for identification during sync. User hooks
  at other paths are preserved.
- **DD-5:** Layered context injection — .claude/rules/backstop.md for stable
  instructions, PostCompact hook for dynamic state recovery from
  .backstop/session-state.json. Artifacts are the source of truth, not generated
  prose duplicating their content.
- **DD-6:** Hard file scope enforcement — plan passed D-081, scopes are disjoint
  by construction, out-of-scope writes are denied via PreToolUse exit 2.
- **DD-7:** Copilot SDK provider deferred — interface designed for it, no
  implementation in alpha. Claude Code is the only shipped provider.
- **DD-8:** Session state persistence — .backstop/session-state.json tracks
  active directive, current plan task, file scope, and progress. Agent updates
  it; PostCompact hook reads it.

## Draft Requirements

requirements:
  - id: REQ-001
    text: >
      backstop init must generate Claude Code hook configurations including
      settings.json entries and hook scripts in .claude/hooks/backstop-*
  - id: REQ-002
    text: >
      backstop hooks sync must regenerate hook configurations without clobbering
      user-defined hooks, using .claude/hooks/backstop-* path prefix to identify
      managed entries
  - id: REQ-003
    text: >
      PostToolUse hooks must fire on all file write/edit operations and route
      checks by file type using compiled enforcement manifests as source of truth
  - id: REQ-004
    text: >
      Code file checks must invoke semgrep against the compiled .semgrep.yml
      config matched by the manifest's language field to the file extension
  - id: REQ-005
    text: >
      Artifact file checks must invoke the backstop-hook binary which imports
      pkg/validate directly and routes by artifact extension
  - id: REQ-006
    text: >
      PostToolUse hooks must inject violations as agent context (additionalContext)
      for self-correction, not block the completed write
  - id: REQ-007
    text: >
      PreToolUse hooks must hard-block writes to files outside the current plan
      task's file scope via exit code 2
  - id: REQ-008
    text: >
      SessionStart hook must inject backstop project context directing the agent
      to read active artifacts (specs, plans, directives) directly
  - id: REQ-009
    text: >
      PostCompact hook must re-inject dynamic session state from
      .backstop/session-state.json including active directive, current plan task,
      file scope, and progress
  - id: REQ-010
    text: >
      The runtime provider interface must support Generate and Validate operations
      and be extensible for future runtime providers (Copilot SDK)
  - id: REQ-011
    text: >
      .claude/rules/backstop.md must teach the agent the backstop workflow contract
      without duplicating artifact content — artifacts are the source of truth
  - id: REQ-012
    text: >
      .backstop/session-state.json must persist active directive, current plan task,
      file scope, and progress for PostCompact recovery and session continuity
  - id: REQ-013
    text: >
      Alpha hook scripts and cmd/backstop-hook binary are explicitly disposable
      scaffolding, replaced wholesale when backstop check ships in the CLI
  - id: REQ-014
    text: >
      Hook execution must complete within 2 seconds for file-write checks (ADR-0014)
  - id: REQ-015
    text: >
      PostToolUse hooks for test runs (go test, npm test, pytest, etc.) must
      capture results and check against spec claim mandated test names

## Spec Seeds

- **SPEC-NNN:** Runtime Provider Interface — the Go interface, RuntimeConfig,
  GeneratedFile types, provider registration. Designed for Claude Code now,
  extensible to Copilot SDK later.
- **SPEC-NNN:** Claude Code Provider (Alpha) — settings.json merge with
  path-convention identification, hook script generation (post-edit, post-test,
  pre-edit scope gate, session start, post-compact), .claude/rules/backstop.md
  generation.
- **SPEC-NNN:** backstop-hook binary — thin cmd/backstop-hook/main.go that
  imports pkg/validate for artifact validation. File-path routing by extension.
  JSON output. Disposable; absorbed into backstop check.
- **SPEC-NNN:** Session state persistence — .backstop/session-state.json schema,
  read/write operations, PostCompact recovery integration.

## Open Questions

None remaining. All seven original OQs resolved 2026-03-30.

## Version History

- 0.1.0 (2026-03-30): Initial bundle. Captured common hook surface across Claude
  Code and Copilot SDK. Seven open design questions. Exploring maturity.
- 0.2.0 (2026-03-30): All 7 questions resolved. Maturity advanced to defined.
  Key decisions: hybrid alpha execution (raw tools + thin Go binary), manifest-
  driven file-type dispatch, hard file scope blocking, layered context injection
  with PostCompact recovery, Copilot SDK deferred to post-alpha, session state
  persistence for compaction survival.

## References

- ADR-0014: Runtime Integration — Hooks, Runtimes, Passive Enforcement
- ADR-0005: Backstop.yml — Namespaced Manifest
- ADR-0008: CLI Design
- ADR-0010: Verification Kill Chain
- D-080: Agent-bounded tasks
- D-081: Disjoint file sets for parallel tasks
- Claude Code hooks documentation (24 event types, 4 handler types)
- Copilot SDK source (6 lifecycle hooks, TypeScript async functions)
