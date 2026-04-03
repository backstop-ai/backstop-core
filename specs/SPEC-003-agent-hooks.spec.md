---
title: "SPEC-003: Agent File-Type Enforcement Hooks"
number: SPEC-003
created: "2026-03-31"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    A PreToolUse hook script (.claude/hooks/backstop-agent-guard.sh) that
    enforces per-agent file-type write restrictions. The hook is invoked by
    Claude Code before Edit and Write tool calls. It reads the tool input
    JSON from stdin, extracts the agent name from the session context and
    the target file path from the tool parameters, matches the path against
    an allowlist keyed by agent type, and exits 2 to block unauthorized
    writes. Hook registration entries in .claude/settings.json wire the
    script to PreToolUse for Edit and Write tools. Reviewer agents are
    additionally protected by disallowedTools but the hook provides
    belt-and-suspenders enforcement. This is the first real backstop hook
    in production — the lightweight wedge for the control surface before
    full runtime hooks ship.
  package: .claude/hooks

verification:
  level: integration
  test_command: bash specs/tests/spec-003-agent-hooks-test.sh
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      A hook script must exist at .claude/hooks/backstop-agent-guard.sh
      that is executable and can be invoked as a PreToolUse hook by
      Claude Code.
    supports: agent-definitions:REQ-008

  - id: REQ-002
    text: >
      The hook script must read JSON from stdin containing the tool name,
      tool input (with file_path), and the agent name. The agent name is
      available in the CLAUDE_AGENT_NAME environment variable set by
      Claude Code when running an agent.

  - id: REQ-003
    text: >
      The hook must enforce the following per-agent file-type allowlists.
      An agent may only write (Edit or Write) to files matching its
      allowed patterns. Any write to a file not matching the agent's
      allowlist must be blocked (exit 2). The allowlists are:
      bundle-author may only write to files matching *.bundle.md;
      spec-author may only write to files matching *.spec.md;
      adr-author may only write to files matching *.adr.md;
      issue-author may only write to files matching *.issue.md;
      planner may only write to files matching *.plan.yml;
      implementer may only write to files matching *.go, go.mod, go.sum,
      Makefile, or files matching *.json or *.yml within .backstop/ or
      artifacts/ directories.
    supports:
      - agent-definitions:REQ-002
      - agent-definitions:REQ-008

  - id: REQ-004
    text: >
      Reviewer agents (spec-reviewer, plan-reviewer, impl-reviewer)
      must be blocked from all file writes by the hook. The hook must
      exit 2 for any file path when the agent is a reviewer. This
      provides belt-and-suspenders enforcement alongside disallowedTools.
    supports: agent-definitions:REQ-008

  - id: REQ-005
    text: >
      When no agent name is detected (CLAUDE_AGENT_NAME is empty or
      unset), the hook must allow the write (exit 0). This permits
      normal non-agent Claude Code usage and the orchestrator to
      function without restriction.

  - id: REQ-006
    text: >
      When an unrecognized agent name is detected (not one of the nine
      known agents), the hook must block the write (exit 2). Unknown
      agents default to deny, not allow.

  - id: REQ-007
    text: >
      The hook must exit 0 to allow a permitted write, exit 2 to block
      a denied write with a JSON reason on stdout, and must never exit 1
      (which Claude Code interprets as a hook error, not a block). The
      block reason JSON must include a "decision" field set to "block"
      and a "reason" field explaining which agent was denied which file.

  - id: REQ-008
    text: >
      Hook registration in .claude/settings.json must configure
      PreToolUse hooks for both the Edit and Write tools, invoking
      .claude/hooks/backstop-agent-guard.sh for each.

  - id: REQ-009
    text: >
      The hook script must handle the case where stdin is empty or
      contains invalid JSON without crashing. In this case the hook
      must exit 0 (allow) to avoid blocking legitimate tool calls
      when hook input is malformed.

  - id: REQ-010
    text: >
      The implementer agent's allowlist must permit writes to schema
      and configuration files (*.json, *.yml) only when the path is
      within the .backstop/ or artifacts/ directory trees. JSON and
      YAML files outside those directories must be blocked.

claims:
  # REQ-001: Hook script exists and is executable
  - id: CLM-001
    requirement: REQ-001
    text: Hook script exists at .claude/hooks/backstop-agent-guard.sh and is executable
    tests:
      - TestHook_ScriptExists
      - TestHook_ScriptIsExecutable

  # REQ-002: Hook reads input correctly
  - id: CLM-002
    requirement: REQ-002
    text: Hook reads agent name from CLAUDE_AGENT_NAME environment variable
    tests:
      - TestHook_ReadsAgentNameFromEnv

  - id: CLM-003
    requirement: REQ-002
    text: Hook reads file_path from tool input JSON on stdin
    tests:
      - TestHook_ReadsFilePathFromStdin

  # REQ-003: Per-agent allowlist enforcement — DEPENDENCY MATRIX
  # bundle-author: *.bundle.md
  - id: CLM-004
    requirement: REQ-003
    text: bundle-author writing a .bundle.md file is allowed
    tests:
      - TestHook_BundleAuthor_AllowsBundleMd

  - id: CLM-005
    requirement: REQ-003
    text: bundle-author writing a .spec.md file is blocked
    tests:
      - TestHook_BundleAuthor_BlocksSpecMd

  - id: CLM-006
    requirement: REQ-003
    text: bundle-author writing a .go file is blocked
    tests:
      - TestHook_BundleAuthor_BlocksGoFile

  # spec-author: *.spec.md
  - id: CLM-007
    requirement: REQ-003
    text: spec-author writing a .spec.md file is allowed
    tests:
      - TestHook_SpecAuthor_AllowsSpecMd

  - id: CLM-008
    requirement: REQ-003
    text: spec-author writing a .bundle.md file is blocked
    tests:
      - TestHook_SpecAuthor_BlocksBundleMd

  - id: CLM-009
    requirement: REQ-003
    text: spec-author writing a .go file is blocked
    tests:
      - TestHook_SpecAuthor_BlocksGoFile

  # adr-author: *.adr.md
  - id: CLM-010
    requirement: REQ-003
    text: adr-author writing a .adr.md file is allowed
    tests:
      - TestHook_AdrAuthor_AllowsAdrMd

  - id: CLM-011
    requirement: REQ-003
    text: adr-author writing a .spec.md file is blocked
    tests:
      - TestHook_AdrAuthor_BlocksSpecMd

  - id: CLM-012
    requirement: REQ-003
    text: adr-author writing a .go file is blocked
    tests:
      - TestHook_AdrAuthor_BlocksGoFile

  # issue-author: *.issue.md
  - id: CLM-013
    requirement: REQ-003
    text: issue-author writing a .issue.md file is allowed
    tests:
      - TestHook_IssueAuthor_AllowsIssueMd

  - id: CLM-014
    requirement: REQ-003
    text: issue-author writing a .spec.md file is blocked
    tests:
      - TestHook_IssueAuthor_BlocksSpecMd

  - id: CLM-015
    requirement: REQ-003
    text: issue-author writing a .go file is blocked
    tests:
      - TestHook_IssueAuthor_BlocksGoFile

  # planner: *.plan.yml
  - id: CLM-016
    requirement: REQ-003
    text: planner writing a .plan.yml file is allowed
    tests:
      - TestHook_Planner_AllowsPlanYml

  - id: CLM-017
    requirement: REQ-003
    text: planner writing a .spec.md file is blocked
    tests:
      - TestHook_Planner_BlocksSpecMd

  - id: CLM-018
    requirement: REQ-003
    text: planner writing a .go file is blocked
    tests:
      - TestHook_Planner_BlocksGoFile

  # implementer: *.go, go.mod, go.sum, Makefile, plus scoped *.json/*.yml
  - id: CLM-019
    requirement: REQ-003
    text: implementer writing a .go file is allowed
    tests:
      - TestHook_Implementer_AllowsGoFile

  - id: CLM-020
    requirement: REQ-003
    text: implementer writing a _test.go file is allowed
    tests:
      - TestHook_Implementer_AllowsTestGoFile

  - id: CLM-021
    requirement: REQ-003
    text: implementer writing go.mod is allowed
    tests:
      - TestHook_Implementer_AllowsGoMod

  - id: CLM-022
    requirement: REQ-003
    text: implementer writing go.sum is allowed
    tests:
      - TestHook_Implementer_AllowsGoSum

  - id: CLM-023
    requirement: REQ-003
    text: implementer writing Makefile is allowed
    tests:
      - TestHook_Implementer_AllowsMakefile

  - id: CLM-024
    requirement: REQ-003
    text: implementer writing a .spec.md file is blocked
    tests:
      - TestHook_Implementer_BlocksSpecMd

  - id: CLM-025
    requirement: REQ-003
    text: implementer writing a .bundle.md file is blocked
    tests:
      - TestHook_Implementer_BlocksBundleMd

  - id: CLM-026
    requirement: REQ-003
    text: implementer writing a .plan.yml file is blocked
    tests:
      - TestHook_Implementer_BlocksPlanYml

  # REQ-004: Reviewer agents blocked from all writes
  - id: CLM-027
    requirement: REQ-004
    text: spec-reviewer writing any file is blocked
    tests:
      - TestHook_SpecReviewer_BlocksAnyFile

  - id: CLM-028
    requirement: REQ-004
    text: plan-reviewer writing any file is blocked
    tests:
      - TestHook_PlanReviewer_BlocksAnyFile

  - id: CLM-029
    requirement: REQ-004
    text: impl-reviewer writing any file is blocked
    tests:
      - TestHook_ImplReviewer_BlocksAnyFile

  # REQ-005: No agent name allows write
  - id: CLM-030
    requirement: REQ-005
    text: Empty CLAUDE_AGENT_NAME allows the write
    tests:
      - TestHook_NoAgentName_Allows

  - id: CLM-031
    requirement: REQ-005
    text: Unset CLAUDE_AGENT_NAME allows the write
    tests:
      - TestHook_UnsetAgentName_Allows

  # REQ-006: Unknown agent blocked
  - id: CLM-032
    requirement: REQ-006
    text: Unrecognized agent name blocks the write
    tests:
      - TestHook_UnknownAgent_Blocks

  # REQ-007: Exit codes and JSON output
  - id: CLM-033
    requirement: REQ-007
    text: Allowed write exits 0 with no output
    tests:
      - TestHook_AllowedWrite_ExitsZero

  - id: CLM-034
    requirement: REQ-007
    text: Blocked write exits 2 with JSON containing decision and reason fields
    tests:
      - TestHook_BlockedWrite_ExitsTwoWithJson

  - id: CLM-035
    requirement: REQ-007
    text: Hook never exits 1 for any valid input scenario
    tests:
      - TestHook_NeverExitsOne

  # REQ-008: Settings.json registration
  - id: CLM-036
    requirement: REQ-008
    text: settings.json contains PreToolUse hook entry for Edit tool
    tests:
      - TestHook_SettingsJson_EditHook

  - id: CLM-037
    requirement: REQ-008
    text: settings.json contains PreToolUse hook entry for Write tool
    tests:
      - TestHook_SettingsJson_WriteHook

  # REQ-009: Malformed input handling
  - id: CLM-038
    requirement: REQ-009
    text: Empty stdin causes hook to exit 0 (allow)
    tests:
      - TestHook_EmptyStdin_Allows

  - id: CLM-039
    requirement: REQ-009
    text: Invalid JSON on stdin causes hook to exit 0 (allow)
    tests:
      - TestHook_InvalidJson_Allows

  # REQ-010: Implementer scoped JSON/YAML
  - id: CLM-040
    requirement: REQ-010
    text: implementer writing .json within artifacts/ is allowed
    tests:
      - TestHook_Implementer_AllowsJsonInArtifacts

  - id: CLM-041
    requirement: REQ-010
    text: implementer writing .yml within artifacts/ is allowed
    tests:
      - TestHook_Implementer_AllowsYmlInArtifacts

  - id: CLM-042
    requirement: REQ-010
    text: implementer writing .json within .backstop/ is allowed
    tests:
      - TestHook_Implementer_AllowsJsonInBackstop

  - id: CLM-043
    requirement: REQ-010
    text: implementer writing .yml within .backstop/ is allowed
    tests:
      - TestHook_Implementer_AllowsYmlInBackstop

  - id: CLM-044
    requirement: REQ-010
    text: implementer writing .json outside artifacts/ and .backstop/ is blocked
    tests:
      - TestHook_Implementer_BlocksJsonOutsideScope

  - id: CLM-045
    requirement: REQ-010
    text: implementer writing .yml outside artifacts/ and .backstop/ is blocked
    tests:
      - TestHook_Implementer_BlocksYmlOutsideScope

contracts:
  - file: .claude/hooks/backstop-agent-guard.sh
    provides:
      - name: backstop-agent-guard
        kind: function
        signature: "backstop-agent-guard.sh (reads JSON from stdin, reads CLAUDE_AGENT_NAME from env, exits 0 or 2)"
        notes: "Shell script, not Go. Invoked by Claude Code PreToolUse hook mechanism."
    consumes: []

  - file: .claude/settings.json
    provides:
      - name: PreToolUse hooks configuration
        kind: variable
        signature: "hooks.PreToolUse[] entries for Edit and Write tools"
        notes: "JSON configuration consumed by Claude Code runtime"
    consumes: []
---

# SPEC-003: Agent File-Type Enforcement Hooks

## Overview

Each backstop agent role has a narrow scope of files it should write to. The
bundle-author writes bundles, the spec-author writes specs, the implementer
writes code. Reviewer agents should not write anything at all.

This spec defines a PreToolUse hook script that mechanically enforces these
restrictions. Claude Code invokes the hook before every Edit and Write tool
call. The hook reads the agent name from the CLAUDE_AGENT_NAME environment
variable, extracts the target file path from the tool input JSON on stdin,
and checks the path against a per-agent allowlist. Permitted writes exit 0.
Denied writes exit 2 with a JSON explanation. The hook never exits 1 (which
Claude Code interprets as a hook error).

This is the first real backstop hook in production. It validates the hook
pattern — stdin JSON, exit code semantics, settings.json registration —
before the full runtime hooks (post-write validation, session state) ship
in a later bundle.

Reviewer agents are additionally protected by disallowedTools (Edit, Write
are removed from their tool set), but the hook provides belt-and-suspenders
enforcement. If a reviewer agent somehow receives a write tool call, the
hook blocks it independently.

## Requirements

Requirements are defined in frontmatter. Claims are defined in frontmatter.

### Per-Agent File-Type Allowlists

| Agent | Allowed write patterns | Prohibited (examples) |
|-------|----------------------|----------------------|
| bundle-author | `*.bundle.md` | `*.spec.md`, `*.go`, all others |
| spec-author | `*.spec.md` | `*.bundle.md`, `*.go`, all others |
| adr-author | `*.adr.md` | `*.spec.md`, `*.go`, all others |
| issue-author | `*.issue.md` | `*.spec.md`, `*.go`, all others |
| planner | `*.plan.yml` | `*.spec.md`, `*.go`, all others |
| implementer | `*.go`, `go.mod`, `go.sum`, `Makefile`, plus `*.json`/`*.yml` within `.backstop/` or `artifacts/` | `*.spec.md`, `*.bundle.md`, `*.plan.yml`, `*.json` outside scoped dirs |
| spec-reviewer | nothing (all writes blocked) | everything |
| plan-reviewer | nothing (all writes blocked) | everything |
| impl-reviewer | nothing (all writes blocked) | everything |

### Exit Code Semantics

| Exit code | Meaning | When used |
|-----------|---------|-----------|
| 0 | Allow the write | File matches agent's allowlist, or no agent name set |
| 1 | Hook error (never used) | Never — this would cause Claude Code to treat it as a hook failure |
| 2 | Block the write | File does not match agent's allowlist, or unknown agent |

### Default-Deny for Unknown Agents

If CLAUDE_AGENT_NAME contains a value not matching any of the nine known
agents (bundle-author, spec-author, adr-author, issue-author, planner,
implementer, spec-reviewer, plan-reviewer, impl-reviewer), the hook blocks
the write. Unknown agents default to deny, not allow.

### Default-Allow for No Agent

If CLAUDE_AGENT_NAME is empty or unset, the hook allows the write. This
permits normal non-agent Claude Code usage (direct user interaction) and
the orchestrator to function without restriction.

## Implementation

### Hook Script (.claude/hooks/backstop-agent-guard.sh)

The script performs these steps in order:

1. **Read agent name** — from CLAUDE_AGENT_NAME environment variable. If
   empty or unset, exit 0 (allow).

2. **Read stdin** — parse JSON to extract the file path from the tool input.
   The JSON structure is `{"tool_name": "...", "tool_input": {"file_path": "..."}}`.
   If stdin is empty or JSON is invalid, exit 0 (allow).

3. **Match agent to allowlist** — look up the agent name in a case block:
   - `bundle-author`: check if file path ends with `.bundle.md`
   - `spec-author`: check if file path ends with `.spec.md`
   - `adr-author`: check if file path ends with `.adr.md`
   - `issue-author`: check if file path ends with `.issue.md`
   - `planner`: check if file path ends with `.plan.yml`
   - `implementer`: check if file path ends with `.go`, or basename is
     `go.mod`, `go.sum`, or `Makefile`, or (file path ends with `.json`
     or `.yml` AND path contains `.backstop/` or `artifacts/`)
   - `spec-reviewer`, `plan-reviewer`, `impl-reviewer`: always block
   - Any other value: block (unknown agent, default-deny)

4. **Output result** — if allowed, exit 0 with no output. If blocked,
   write JSON to stdout with `decision` and `reason` fields, then exit 2.

### Settings Registration (.claude/settings.json)

The settings file must contain hook entries in this structure:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit",
        "hooks": [
          {
            "type": "command",
            "command": ".claude/hooks/backstop-agent-guard.sh"
          }
        ]
      },
      {
        "matcher": "Write",
        "hooks": [
          {
            "type": "command",
            "command": ".claude/hooks/backstop-agent-guard.sh"
          }
        ]
      }
    ]
  }
}
```

### Dependencies

The hook script depends only on standard POSIX shell utilities (sh, cat,
grep) and `jq` for JSON parsing. No Go compilation required — this is a
shell script, not a Go package.

## Verification

Verification is defined in frontmatter. Integration-level verification
because the tests invoke the actual shell script as a subprocess and verify
exit codes and stdout output.

Claims are defined in frontmatter. Each claim maps to a test function in
the integration test script that invokes the hook with controlled
environment variables and stdin, then asserts the exit code and output.

### Test Approach

Tests are bash scripts that:
1. Set CLAUDE_AGENT_NAME to the agent under test
2. Pipe a JSON payload with a file_path to the hook's stdin
3. Capture the exit code
4. Assert exit code is 0 (allowed) or 2 (blocked)
5. For blocked writes, parse stdout JSON and verify decision/reason fields

## Sharp Edges

- **CLAUDE_AGENT_NAME availability.** This spec assumes Claude Code sets the
  CLAUDE_AGENT_NAME environment variable when running agent subprocesses.
  If Claude Code changes the mechanism for communicating agent identity to
  hooks (e.g., moves it into the stdin JSON), the hook must be updated. The
  spec requires reading the env var, not a specific Claude Code version.

- **jq dependency.** The hook requires jq for JSON parsing. If jq is not
  installed, the hook will fail with exit 1 (hook error), which Claude Code
  treats differently from exit 2 (block). Systems without jq will see hook
  errors, not blocked writes. The test suite must verify jq is available
  before running.

- **Path canonicalization.** The hook matches file paths using suffix
  patterns (ends-with, contains). If Claude Code passes absolute paths but
  the allowlist checks use relative patterns, or vice versa, matches could
  fail. The implementer's scoped directory check (`.backstop/` and
  `artifacts/`) uses substring matching, which works for both absolute and
  relative paths but could false-positive on paths like
  `/home/user/my-artifacts/unrelated.json`.

- **Race between disallowedTools and hooks.** Reviewer agents have both
  disallowedTools (Edit, Write removed) and hook enforcement. If Claude Code
  evaluates disallowedTools before invoking hooks, the hook never fires for
  reviewers. The belt-and-suspenders design means correctness doesn't depend
  on evaluation order, but the hook claims for reviewers may not exercise in
  normal operation — they catch the case where disallowedTools is
  misconfigured or bypassed.

- **Settings.json merge conflicts.** If other features add their own hooks
  to settings.json, manual merge is required. The PreToolUse array supports
  multiple entries, but concurrent edits to the same JSON file will conflict.

- **Implementer's scoped directory matching is substring-based.** Checking
  if the path "contains" `.backstop/` or `artifacts/` could match
  unintended directories. A path like `pkg/artifacts-legacy/schema.json`
  would match the `artifacts` substring. The implementation should match
  `.backstop/` and `artifacts/` as path segments, not arbitrary substrings.

## References

- Bundle: agent-definitions (spec seed: agent definitions, DD-5)
- ADR-0012: Review Model — Independent Reviewer
- ADR-0018: Workflow State Machine
- D-102: Review is always a separate agent session
- Claude Code hooks documentation (PreToolUse, exit code semantics)
- SPEC-001: Standards Compiler (reference for spec format)
- SPEC-002: Plan Schema Evolution (sibling spec from same bundle)
