---
title: Standards Compiler and Enforcement Orchestration
schema_version: bundle/v1

bundle:
  name: standards-compiler
  version: "0.2.0"
  created: "2026-03-29"
  category: feature

status:
  maturity: defined

problem:
  summary: >
    Backstop standard artifacts (.standard.md) define enforcement rules in a
    human-readable format with machine-readable frontmatter. These rules must
    be compiled into executable formats and orchestrated by the CLI. The current
    gap is between "rules are defined" and "rules are enforced." The compiler
    bridges that gap, but several design questions remain unresolved around
    multi-standard merging, language-agnostic rules, metric execution, and
    the relationship between the compiler, the CLI, and external tools.

  user_story: >
    As a developer using backstop, I want my code standards to be automatically
    enforced without me configuring semgrep, linters, or metric checks manually.
    I adopt a standards pack, run backstop, and violations are reported — regardless
    of whether the rule is a semgrep pattern, a file size metric, or a delegation
    to golangci-lint.

solution:
  approach: >
    A standards compiler that reads .standard.md files and produces an enforcement
    manifest per standard. The CLI reads all manifests for the project's configured
    packs and orchestrates enforcement across three execution paths: semgrep for
    pattern/regex rules, native CLI checks for metric rules, and delegation
    verification for rules enforced by external tools.
---

# Standards Compiler and Enforcement Orchestration

## Current Thinking

### What We Know

The standard artifact format is designed and validated. Each rule has a detection
block with one of four strategies: pattern (semgrep AST matching), regex (semgrep
regex matching), metric (structural measurements like file/function length), and
delegated (external tool like golangci-lint). The compiler's job is to read these
rules and produce executable output.

We have SPEC-001 in draft, but working through it surfaced six open questions that
need resolution before implementation.

### The Enforcement Manifest Model

The key insight from our design session: the CLI is the single orchestrator. The
compiler's primary output is an **enforcement manifest** — a JSON file listing
every rule and how to enforce it. The CLI reads the manifest and handles
everything: invoking semgrep, running native checks, verifying delegated tools.

The semgrep YAML and native checks JSON are secondary artifacts referenced by
the manifest. They're implementation details of the manifest, not standalone
outputs.

### Resolved Questions

#### OQ-1: Multi-Standard Compilation → Lazy merge at CLI runtime

Per-standard manifests stay as the source of truth (traceability). At runtime,
the CLI merges all semgrep rules into a single combined config for one invocation.
Similarly batches native metric checks and delegated verifications. The merge is
a transient execution artifact, not a persisted compiler output. Violations map
back to source standards via rule ID prefixes.

#### OQ-2: Semgrep Languages for Universal Rules → Support both strategies

Universal-scope standards default to metric/regex strategies (no language context
needed). However, per-rule `languages` override is allowed in detection blocks
for pattern strategy. This is ~20-30 lines of extra code across schema, validator,
and compiler. Avoids painting ourselves into a corner.

#### OQ-3: Metric Vocabulary and Execution → Delegate to ecosystem tools

Most "metrics" (function length, complexity, nesting depth, parameter count) are
already measured by mature ecosystem tools: golangci-lint (Go), ruff/pylint
(Python), eslint (TypeScript), clippy (Rust), PMD (Java). Backstop's native
`metric` strategy is reserved for thin structural checks that no ecosystem tool
covers (e.g., `test_file_exists`). Interface-based internally (`MetricEvaluator`)
so native metrics can be added later. This embodies the backstop philosophy:
we're not reinventing linters, we're making the right thing the easy thing.
The tools exist; the discipline doesn't. That's the gap.

#### OQ-4: Include Cycle Detection → Eliminated (no includes in v1)

The `includes` feature is dropped for v1. One file per standard. The CLI's lazy
merge at runtime eliminates the need for cross-file composition within a single
standard. Users write intuitively scoped standard files and the CLI merges them.
Includes can be added later if a real need emerges.

#### OQ-5: Rule ID Uniqueness Across Includes → Eliminated (no includes in v1)

With includes dropped, rule ID uniqueness is enforced per-file only, which the
validator already handles. No cross-file ID tracking needed.

#### OQ-6: Artifacts Root Resolution → Go embed for all schemas

The CLI ships with ALL artifact schemas and validators embedded via Go embed.
Each CLI version is a schema cohort — a locked set of schemas with backward
compatibility. Library consumers can override via explicit `opts.ArtifactsRoot`.
This extends beyond just the standard schema to every artifact type.

#### BQ-1: Delegated Tool Config Verification → Proactive remediation at adoption

At pack adoption (`backstop adopt`), the CLI scans delegated rules, checks project
configs, and offers interactive fixes for any gaps: "errcheck needs to be enabled
in .golangci.yml. Add it? [Y/n]". At runtime, unfixed gaps produce hard failures.
This makes the right thing the easy thing — backstop doesn't just tell you what's
wrong, it offers to fix it.

#### BQ-2: Compliance Tier Filtering → CLI filters at runtime

The manifest contains all rules tagged with their compliance tier. The CLI filters
at runtime based on the project's configured tier (baseline/standard/strict). No
tier-filtered manifests from the compiler.

#### BQ-3: Deprecated Standard Handling → Compile with warning

Deprecated standards compile successfully but emit a visible warning on every
invocation. The warning creates persistent pressure to migrate without breaking
anything — it shows up in CI logs, terminal output, and adoption reports.

## Design Decisions

- **DD-1:** Per-standard manifests for traceability. CLI lazy-merges semgrep rules,
  metric checks, and delegated verifications at runtime for execution efficiency.
- **DD-2:** Universal rules default to metric/regex. Per-rule `languages` override
  allowed in detection blocks for pattern strategy edge cases.
- **DD-3:** Native `metric` strategy is thin — structural checks only (e.g.,
  `test_file_exists`). Ecosystem tools handle complexity, function length, nesting
  depth, etc. via `delegated` strategy. Interface-based internally for future extension.
- **DD-4:** No `includes` in v1. One file per standard. CLI merge handles composition.
- **DD-5:** Rule ID uniqueness enforced per-file by the validator. No cross-file tracking.
- **DD-6:** All schemas and validators embedded in CLI via Go embed. Each CLI version
  is a schema cohort with backward compatibility. Library override via explicit path.
- **DD-7:** Delegated tool config: proactive remediation at pack adoption (interactive
  fix prompts), hard-fail at runtime for unfixed gaps.
- **DD-8:** Manifest contains all rules tagged with compliance tier. CLI filters at
  runtime based on project config.
- **DD-9:** Deprecated standards compile with visible warning. Not blocked, but noisy.

## Requirements

- The compiler must produce an enforcement manifest per standard
- The CLI must lazy-merge all manifests at runtime for execution efficiency
- The CLI must map violations back to source standards via rule ID prefixes
- Universal-scope standards must support metric/regex strategies without language
- Universal-scope pattern rules must specify `languages` per-rule in detection block
- Native metric evaluator must implement `MetricEvaluator` interface
- Native metric vocabulary for v1: `test_file_exists` (structural checks only)
- Ecosystem metrics (complexity, function length, etc.) use `delegated` strategy
- Schema resolution must work via Go embed (CLI) and filesystem override (library)
- The compiler must be idempotent
- Delegated rules must appear in the manifest for CLI verification
- Advisory-only rules must be excluded from enforcement output
- Pack adoption must scan delegated rules and offer interactive config fixes
- Runtime enforcement must hard-fail on unfixed delegated tool gaps
- Manifest rules must include compliance tier tags
- Deprecated standards must compile with visible deprecation warning

## Spec Seeds

- **SPEC-001 (update):** Standards Compiler — update to reflect all resolved questions,
  remove includes, add lazy merge, update metric model
- **SPEC-002:** Metric Evaluator — `MetricEvaluator` interface, `test_file_exists`
  native implementation, delegation model for ecosystem metrics
- **SPEC-003:** CLI Enforcement Orchestrator — manifest discovery, lazy merge, semgrep
  invocation, native check execution, delegated verification, unified violation reporting,
  compliance tier filtering
- **SPEC-004:** Pack Adoption — delegated tool config scanning, interactive remediation,
  deprecation warnings

## Open Questions

None remaining. All original OQs and bonus questions resolved 2026-03-29.

## Version History

- 0.1.0 (2026-03-29): Initial bundle. Captured six open questions from SPEC-001
  draft review. Exploring maturity.
- 0.2.0 (2026-03-29): All 9 questions resolved. Maturity advanced to defined.
  Key decisions: lazy merge at CLI runtime, ecosystem delegation for metrics,
  includes dropped for v1, proactive remediation at pack adoption.
