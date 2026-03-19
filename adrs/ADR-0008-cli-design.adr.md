---
number: ADR-0008
created: "2026-03-18"
status: Accepted
deciders: "@bmanson"
decisions: "D-025–D-029, D-037, D-076"
schema_version: adr/v2
---

# ADR-0008: CLI Design — init, new, validate, baseline, cd

## Context

The CLI is the universal agent API (D-069). Every agent, regardless of language or runtime, interacts with backstop by shelling out to CLI commands. The CLI must be optimized for agent consumption: structured JSON output, predictable exit codes, and commands that map to agent workflow steps.

Humans use the CLI too — but when there's a tension between agent UX and human UX, the agent wins (ADR-0001).

## Decision

### Core commands

**`backstop init`** — scaffold a new backstop project

```bash
$ backstop init
# Detects language from go.mod/package.json/etc.
# Generates backstop.yml with sensible defaults
# Creates .backstop/ directory structure
# Generates runtime hooks for declared runtimes
```

Options:
- `--runtime copilot-cli,claude-code` — declare target runtimes
- `--cd vercel` — include CD provider setup
- `--ceremony full|standard|minimal` — preset workflow configuration

**`backstop new <type>`** — scaffold a new artifact

```bash
$ backstop new spec
# Auto-assigns next available ID: SPEC-0042
# Creates SPEC-0042-<slug>.spec.md from template
# Opens in editor (if interactive) or writes to stdout (if piped)
```

Supports all artifact types: `spec`, `plan`, `issue`, `adr`, `directive`, `bundle`, `capability`. The `--slug` flag provides the semantic name; interactive mode prompts for it.

**`backstop validate`** — the MVP command

```bash
$ backstop validate --json
{
  "pass": true,
  "violations": [],
  "warnings": [
    {
      "rule": "go:core@1.0.0:GO-0015",
      "file": "internal/config/loader.go",
      "line": 23,
      "message": "Function exceeds 50 lines",
      "severity": "warning"
    }
  ],
  "waivers_applied": [
    {
      "waiver": "WAIVER-0001",
      "rule": "go:security@2.0.0:SEC-0012",
      "scope": "internal/auth/legacy.go",
      "expires": "2026-09-01"
    }
  ],
  "baseline_delta": { "new": 0, "resolved": 2 }
}
```

Options:
- `--json` — structured JSON output (agent default)
- `--scope changed` — validate only changed files (default, D-037)
- `--scope all` — validate entire codebase
- `--service api-gateway` — validate a specific monorepo service

Exit codes:
- `0` — all checks pass
- `1` — violations found
- `2` — backstop configuration error (invalid backstop.yml, missing schemas)

**`backstop baseline`** — record pre-existing violations for adoption

```bash
$ backstop baseline
# Scans entire codebase
# Records violation counts per rule per file
# Writes .backstop/baseline.yml

$ backstop baseline --update
# Re-scans, lowers baseline where violations are fixed
# Refuses to increase baseline — new violations must be fixed or waived
```

**`backstop cd add <provider>`** — wire up continuous deployment

```bash
$ backstop cd add vercel
# Prompts for credentials (or reads from env)
# Stores credentials as GitHub repository secrets
# Generates deployment workflow in .github/workflows/
```

### Output philosophy

The CLI has two modes:
- **JSON mode** (`--json`) — structured output for agent consumption. This is the API contract.
- **Human mode** (default) — formatted terminal output with colors, tables, and summaries.

Agents always use `--json`. Humans get the pretty output by default. The underlying data is identical.

### Changed-files-only as default scope

`backstop validate` operates on changed files by default (D-037). In a git repository, "changed" means files modified since the merge base. This makes validation fast (milliseconds on typical PRs) and relevant (only shows violations in code you're actually changing). `--scope all` is available for full-codebase scans and baseline generation.

### Command discovery

`backstop help` and `backstop <command> --help` provide structured documentation. `backstop commands --json` returns the full command tree for agent discovery — agents can learn what backstop offers without documentation.

## Consequences

### What this enables
- **Agent-native workflow.** `backstop validate --json` gives agents everything they need to decide "fix or ship."
- **Fast iteration.** Changed-files-only default means validation runs in milliseconds, not minutes.
- **Progressive adoption.** `init` → `validate` is the minimum path. `new`, `baseline`, `cd` are added as needed.

### What this requires
- **Stable JSON schema.** The `--json` output format is a versioned API contract. Changes follow D-070 evolution rules.
- **Git integration.** Changed-files detection requires git. Non-git projects fall back to `--scope all`.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Subcommand-heavy CLI (backstop artifact new spec) | Too verbose for agent consumption. `backstop new spec` is shorter and unambiguous. |
| Config file for CLI options | Over-engineering. CLI flags are sufficient. The manifest (backstop.yml) handles project config. |
| Interactive-first design | Violates agent-first principle. Interactive mode is nice for humans but agents need flags and JSON. |
| Validate all files by default | Too slow for large codebases. Changed-files is the right default for PR-level validation. |

## References

- D-025–D-029: CLI as first build target, command design, embedded schemas
- D-037: Changed-files-only as default scope
- D-069: CLI is the universal agent API
- D-076: Baseline scan for adoption amnesty
- ADR-0004: Validation engine (the library the CLI wraps)
- ADR-0005: backstop.yml manifest (what `backstop init` generates)
- ADR-0009: CI/CD pipeline (forthcoming — what `backstop cd add` wires up)
