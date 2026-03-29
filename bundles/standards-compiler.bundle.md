---
title: Standards Compiler and Enforcement Orchestration
schema_version: bundle/v1

bundle:
  name: standards-compiler
  version: "0.1.0"
  created: "2026-03-29"
  category: feature

status:
  maturity: exploring

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

### Open Questions

#### OQ-1: Multi-Standard Compilation

A real project has multiple standards: `STD-GO-001` (Go rules), `STD-CORE-001`
(universal rules), maybe `STD-REACT-001` (framework rules). Each produces its
own manifest. Who merges them?

**Options:**
- **(a) No merge — CLI reads all manifests.** Each standard is compiled independently.
  The CLI discovers all manifests in `.backstop/rules/` and processes them all.
  Simple, no merge logic, but the CLI does multiple semgrep invocations.
- **(b) Merge into one master manifest.** A second compilation pass merges all
  per-standard manifests into one. Single semgrep invocation, cleaner reporting,
  but more complex compiler.
- **(c) Lazy merge — single semgrep config, per-standard manifests.** Merge only
  the semgrep YAML files (since semgrep can take one config), keep manifests
  separate for reporting. Best of both worlds?

**Leaning:** Option (a) for now. The CLI is already the orchestrator — it can
glob for manifests and iterate. Merging is optimization we don't need yet.

#### OQ-2: Semgrep Languages for Universal Rules

Semgrep's `languages` field is required in every rule. A `scope: universal`
standard with no language field produces a problem: what goes in `languages: [?]`?

**Options:**
- **(a) `generic`** — Semgrep supports a `generic` language mode for
  language-agnostic pattern matching. Limited but works for simple patterns.
- **(b) Expand at compile time.** The compiler reads the project's configured
  languages and emits one rule per language. `GO-001` becomes `GO-001-go`,
  `GO-001-typescript`, etc.
- **(c) Universal rules can't be semgrep patterns.** Restrict universal scope
  to metric and regex strategies only, since those don't need language context.
- **(d) Universal standards still specify language per-rule.** The standard is
  `scope: universal` (conceptually applies everywhere) but individual rules
  still have a `languages` field in their detection block for semgrep routing.

**Leaning:** Option (d) feels most honest. "Max file length" applies universally
as a concept, but the detection is always language-contextualized. If a rule
truly doesn't need language context, it's a metric or regex, not a pattern.

#### OQ-3: Metric Vocabulary and Execution

The native checks format references metrics like `file_lines`, `function_lines`,
`test_file_exists`, `import_ordering`. But nowhere is the metric vocabulary
defined — what metrics exist, how they're computed, what file types they apply to.

**Questions:**
- Is the metric vocabulary fixed (baked into the CLI) or extensible?
- How does `function_lines` work for Go vs Python vs TypeScript? Is it
  language-specific under the hood?
- Who defines the metric computation? Is it Go code in the CLI, or can
  packs bring their own metric implementations?

**Initial thinking:** Start with a fixed vocabulary baked into the CLI. The
metrics that matter right now are: `file_lines`, `function_lines`,
`function_complexity`, `test_file_exists`. These can all be computed with
Go's AST packages for Go files, and we can add language-specific metric
evaluators later. Extensible metrics is a v2 concern.

#### OQ-4: Include Cycle Detection

If `A.standard.md` includes `B.standard.md` which includes `A.standard.md`,
the compiler recurses forever. Need cycle detection.

**Approach:** Track visited paths during include resolution. If a path is
visited twice, emit a violation and stop. Maximum include depth of 3 as a
safety bound.

#### OQ-5: Rule ID Uniqueness Across Includes

If the main file has rule `GO-001` and an included file also has `GO-001`,
that's a conflict. The validator catches duplicates within a single file,
but not across includes.

**Approach:** The compiler maintains a global rule ID set across the main file
and all includes. Duplicate IDs across files produce a compilation error, not
a silent merge.

#### OQ-6: Artifacts Root Resolution

The compiler needs to validate the standard before compiling, which requires
loading the schema, which requires knowing the artifacts root path. But
`Compile(standardPath, opts)` doesn't know where backstop-core is installed.

**Options:**
- **(a) Explicit in CompileOptions.** `opts.ArtifactsRoot` is required.
- **(b) Convention-based.** Walk up from the standard file looking for an
  `artifacts/` directory.
- **(c) Embedded schemas.** The CLI embeds schemas via Go embed (per D-099),
  so the compiler uses embedded schemas, not filesystem paths.

**Leaning:** Option (c) for the CLI, option (a) for library usage. The
compiler function accepts an optional schema override; the CLI provides
embedded schemas by default.

## Draft Requirements

- The compiler must produce an enforcement manifest per standard
- The CLI must discover and process all manifests in the rules directory
- Universal rules must handle the semgrep language field explicitly
- Metric checks must have a defined, fixed vocabulary for v1
- Include resolution must detect cycles and cap depth
- Rule IDs must be globally unique across includes
- Schema resolution must work in both embedded (CLI) and filesystem (library) modes
- The compiler must be idempotent
- Delegated rules must appear in the manifest for CLI verification
- Advisory-only rules must be excluded from all output

## Draft Design Decisions

- **DD-1:** One manifest per standard, no merge step. CLI iterates all manifests.
- **DD-2:** Universal rules use per-rule language hints in detection blocks, not
  a global language inference.
- **DD-3:** Fixed metric vocabulary for v1: `file_lines`, `function_lines`,
  `function_complexity`, `test_file_exists`.
- **DD-4:** Include cycle detection via visited-path tracking, max depth 3.
- **DD-5:** Global rule ID uniqueness across includes, duplicates are compile errors.
- **DD-6:** Schema resolution via Go embed for CLI, optional filesystem override for library.

## Spec Seeds

- **SPEC-001 (update):** Standards Compiler — update to reflect resolved open questions
- **SPEC-002:** Metric Evaluator — fixed vocabulary, Go AST-based computation, CLI integration
- **SPEC-003:** CLI Enforcement Orchestrator — manifest discovery, semgrep invocation,
  native check execution, delegated verification, unified violation reporting

## Open Questions

- Should the compiler validate that delegated tools are actually configured in the
  project (e.g., check `.golangci.yml` for the specific rule)? Or is that the CLI's job?
- How does the compliance tier (baseline/standard/strict) interact with the manifest?
  Does the CLI filter rules by the project's configured tier, or does the compiler
  emit tier-filtered manifests?
- What happens when a standard is `status: deprecated`? Should the compiler still
  compile it with a warning, or refuse?

## Version History

- 0.1.0 (2026-03-29): Initial bundle. Captured six open questions from SPEC-001
  draft review. Exploring maturity.
