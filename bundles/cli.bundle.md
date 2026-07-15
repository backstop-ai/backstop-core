---
title: Backstop CLI — Universal Agent API
schema_version: bundle/v1

bundle:
  name: cli
  version: "0.4.0"
  created: "2026-04-04"
  updated: "2026-04-04"
  category: tool

status:
  maturity: defined

problem:
  summary: >
    The backstop CLI is the universal agent API — every agent, runtime, and
    workflow interacts with backstop by shelling out to CLI commands (D-069).
    The validation engine (pkg/validate), schema infrastructure (pkg/schema),
    artifact parsers (pkg/artifact), and standards compiler (pkg/compile) exist
    as Go libraries, but there is no CLI binary to expose them. The cmd/backstop/
    directory contains only a .gitkeep. Without the CLI, agents cannot validate
    artifacts, scaffolding cannot happen, packs cannot be installed, baselines
    cannot be recorded, and the workflow state machine cannot be driven. The CLI
    is the bridge between "libraries that verify" and "agents that build."

  user_story: >
    As a developer (or agent) using backstop, I want a single binary that
    exposes all backstop capabilities as composable commands with structured
    JSON output, predictable exit codes, and fast execution — so that every
    agent runtime can shell out to backstop, every CI pipeline can run
    enforcement, and every human can interact with the framework from the
    terminal without understanding the internal library structure.

  success_criteria:
    - backstop artifact validate checks all six artifact types against embedded schemas and reports violations as structured JSON
    - backstop artifact new scaffolds any artifact type with auto-assigned ID via git tag reservation
    - backstop code check runs semgrep and lint against changed files by default, completing single-file dispatch within 2 seconds
    - backstop pack compile produces enforcement manifests from .standard.md files via pkg/compile
    - backstop pack new scaffolds rule pack and code pack directory structures
    - backstop gate runs the full verification kill chain and produces a pass/fail result suitable for CI gating
    - All six bootstrap commands produce identical violation data in --json and human output modes
    - Exit codes are consistent across all commands — 0 (pass), 1 (violations), 2 (config error)
    - The CLI binary is self-contained via go:embed — schemas and baseline rules require no filesystem access at runtime

solution:
  approach: >
    A Go CLI binary (cmd/backstop/) built with a command framework (cobra or
    similar) that wraps the existing pkg/ libraries. Each command is a thin
    adapter: parse flags, call the library, format the result as JSON or human
    output, set the exit code. Schemas are embedded via go:embed so each CLI
    version is a self-contained schema cohort. The CLI ships with an embedded
    baseline rule pack (D-099) so backstop init works offline. Commands are
    organized into three tiers: core commands (init, new, validate, baseline,
    check, compile, hooks sync), pack/registry commands (pack install, pack
    list, pack sync, pack publish), and workflow commands (status, gate).

  assumptions:
    - pkg/validate, pkg/schema, pkg/artifact, and pkg/compile exist and are tested — the CLI wraps them, not reimplements them
    - Cobra is the command framework (OQ-1 resolved)
    - go:embed is used for all schema files and the baseline rule pack — each CLI version is a self-contained schema cohort
    - semgrep is available on PATH or auto-installed to .backstop/tools/ on first backstop code check invocation (OQ-9 resolved)
    - git is available for changed-files detection (merge-base for PR scope, diff for local scope) and for tag-based ID reservation (OQ-8 resolved)
    - backstop.yml exists at a discoverable project root (walk-up from cwd, like go.mod) with BACKSTOP_CONFIG env var as escape hatch (OQ-12 resolved)
    - The verifier (claim-to-test tracing) does not exist yet — backstop gate initially runs everything except verification steps that depend on it

requirements:
  - id: REQ-001
    version: "1.0.0"
    text: >
      backstop artifact validate must validate artifacts against embedded schemas,
      support --spec/--plan/--adr style scoping, and produce structured JSON (--json)
      or formatted human output with exit codes 0/1/2
  - id: REQ-002
    version: "1.0.0"
    text: >
      backstop artifact new must scaffold artifacts for all seven types (spec, plan,
      issue, adr, directive, bundle, capability) with auto-assigned next-available ID
      via git annotated tag reservation
  - id: REQ-003
    version: "1.0.0"
    text: >
      backstop code check must run implementation validation (lint, build, test,
      semgrep) with changed-files as default scope (--diff implicit), --all for full
      codebase, and --file for single-file hook dispatch within a 2-second budget
  - id: REQ-004
    version: "1.0.0"
    text: >
      backstop pack compile must wrap pkg/compile to produce enforcement manifests
      (semgrep YAML, native checks, manifest JSON) from .standard.md files
  - id: REQ-005
    version: "1.0.0"
    text: >
      backstop pack new must scaffold rule pack (--type rule) and code pack
      (--type code) directory structures with language-specific templates
  - id: REQ-006
    version: "1.0.0"
    text: >
      backstop gate must run the full verification kill chain: artifact validation,
      pack rule enforcement, test verification, coverage thresholds, contract
      verification, baseline comparison, and ledger integrity (ADR-0010)
  - id: REQ-007
    version: "1.0.0"
    text: >
      All commands must produce identical violation data in both JSON and human
      output modes, with JSON output including a schema_version field for
      independent contract evolution (D-070)
  - id: REQ-008
    version: "1.0.0"
    text: >
      All artifact schemas and the baseline rule pack must be embedded via go:embed
      so the CLI binary has no runtime filesystem dependency for core enforcement
  - id: REQ-009
    version: "1.0.0"
    text: >
      backstop.yml must be loaded and validated before any enforcement command
      executes, with invalid manifest producing exit code 2 (config error)
  - id: REQ-010
    version: "1.0.0"
    text: >
      Changed-files detection must use git merge-base for PR scope, git diff for
      local scope, and fall back to --scope all in non-git environments
  - id: REQ-011
    version: "1.0.0"
    text: >
      The CLI must expose backstop commands --json for full command tree discovery
      by agent runtimes and backstop version for schema cohort identification
---

# Backstop CLI — Universal Agent API

## Current Thinking

### What Exists Today

The validation engine is implemented and tested: 298+ passing tests at 96.7%
coverage across parsers, schema validators, and artifact validators for all
six primitive types plus contracts and capabilities. The standards compiler
(pkg/compile) produces enforcement manifests. The schema infrastructure
(pkg/schema) handles schema loading and version resolution. The artifact
package (pkg/artifact) handles parsing.

What does NOT exist: the cmd/backstop/ binary, any CLI command implementations,
the go:embed schema cohort, the backstop.yml loader, the pack management
subsystem, the baseline recording system, the changed-files detection, the
JSON/human output formatting layer, or the command discovery endpoint.

### The CLI as API Contract

The CLI's --json output is a versioned API contract (ADR-0004, ADR-0008).
Agents always use --json. Humans get formatted terminal output. The underlying
data is identical. Breaking changes to the JSON output follow D-070 evolution
rules — additive fields with sensible defaults, deprecated fields emit
warnings, breaking removals only on major version bumps.

Exit codes are part of the contract:
- 0: all checks pass
- 1: violations found
- 2: backstop configuration error (invalid backstop.yml, missing schemas)

### Command Taxonomy

Commands are namespaced by what they operate on: artifacts, code, or the
full reconciliation gate. This separation reflects three distinct concerns:
artifact validation (schema conformance), implementation validation (lint,
build, test, semgrep), and full reconciliation (artifact content matches
implementation).

**Bootstrap priority — what we need now:**

`backstop artifact` namespace — artifact lifecycle:
- `backstop artifact validate` — validate artifacts against schemas.
  Supports `--spec SPEC-002`, `--plan PLAN-SPEC-002`, etc. for scoping.
- `backstop artifact new <type>` — scaffold artifacts (spec, plan, issue,
  adr, directive, bundle, capability). Auto-assigns next available ID.

`backstop code` namespace — implementation validation:
- `backstop code check` — run implementation validation: lint, build, test,
  semgrep rules. Default scope is changed files (D-037, --diff implicit).
- `backstop code check --all` — full codebase scan.
- `backstop code check --file <path>` — single-file dispatch for hooks.
  2-second budget.

`backstop pack` namespace — enforcement content (rule packs + code packs):
- `backstop pack compile` — compile .standard.md files to enforcement
  manifests (semgrep YAML, native checks, manifest JSON). Wraps pkg/compile.
- `backstop pack new --type rule --language go` — scaffold a new rule pack.
- `backstop pack new --type code --language go` — scaffold a new code pack.

`backstop gate` — full reconciliation:
- Artifact validation + implementation validation + claim→test tracing +
  contract verification + coverage thresholds + ledger integrity. The
  mechanical "if it's green, it ships." Stands alone because it spans
  artifact, code, and pack domains.

**Future commands (not in bootstrap scope):**

`backstop init` — project scaffolding, backstop.yml generation, runtime hooks.
`backstop pack install/list/sync/publish/remove/update/search/vendor` — registry.
`backstop baseline` — pre-existing violation recording with ratchet.
`backstop status` — directive/phase/progress display.
`backstop hooks sync` — runtime hook regeneration.

**Infrastructure commands:**
- `backstop commands --json` — full command tree for agent discovery.
- `backstop help` / `backstop <command> --help` — structured documentation.
- `backstop version` — version, embedded schema cohort, embedded pack version.

### Changed-Files Detection (D-037)

`backstop validate` operates on changed files by default. In a git repo,
"changed" means files modified since the merge base with the default branch.
This makes validation fast (milliseconds on typical PRs) and relevant (only
shows violations in code being changed). `--scope all` is available for
full-codebase scans. Non-git projects fall back to --scope all.

The merge-base detection needs to handle several scenarios:
- PR branch vs main: git merge-base HEAD origin/main
- Local changes: git diff --name-only (unstaged + staged)
- CI environment: typically have the merge base available as env vars
- Monorepo: --service flag scopes to a specific service path

### Go Embed Schema Cohort

Each CLI version embeds all artifact schemas via go:embed (D-028). The binary
is self-contained — no filesystem dependency for schemas at runtime. This
means each CLI version is a "schema cohort" — a locked set of schemas that
the binary knows how to validate against. Schema evolution follows D-070:
new fields are additive, old configs work without them.

The embedded baseline rule pack (D-099) means backstop init works fully
offline with CWE Top 25 + OWASP Top 10 rules. No registry dependency for
the basic "don't ship something stupid" workflow.

### backstop.yml Loading

The CLI's first action on any enforcement command (validate, check, compile,
gate) is loading and validating backstop.yml against its own schema. This
is the project manifest (ADR-0005) that declares:
- Project identity and target runtimes
- Pack configuration (rules and code packs with versions)
- Workflow ceremony level
- Monorepo service definitions with per-service overrides
- Registry configuration (scope-based resolution)

The loader must handle:
- Schema validation of backstop.yml itself
- Pack resolution from the lockfile (.backstop/pack-lock.yml)
- Service-scoped configuration for monorepo commands
- Default values for the enforcement-only minimum path
- Version migration warnings for deprecated fields

### Pack Lockfile

.backstop/pack-lock.yml pins exact versions with integrity hashes (ADR-0017).
No floating versions. Deterministic across environments. The lockfile records:
- Rule pack versions and integrity hashes
- Code pack versions, integrity hashes, and native package mappings
- Dependency graph between rule packs
- Registry source for each pack

### backstop init Experience (D-104)

backstop init is a silent scaffolding command with no interactive mode in v1
(OQ-3 resolved). It detects the project language from cwd, installs default
standards for that language, compiles enforcement manifests, runs a baseline
scan, generates runtime hooks, and prints "start an agent session" guidance.
The full init sequence:
1. Detect language from cwd (file extensions, go.mod, package.json, etc.)
2. Generate backstop.yml with sensible defaults for the detected language
3. Create .backstop/ directory structure
4. Install and compile default standards for the detected language
5. Run baseline scan to record pre-existing violations
6. Generate runtime hooks for declared runtimes
7. Print guidance: "start an agent session to begin your first task"

No --interactive flag. No guided conversation. The interactive agent session
from ADR-0018 is a post-bootstrap concern for the workflow engine.

### backstop check — Hook Dispatch

backstop check --file <path> is the single-file enforcement command designed
for runtime hooks (ADR-0014, runtime-hooks bundle). It must:
- Complete within 2 seconds (hard budget)
- Route by file type using compiled enforcement manifests
- Run semgrep for code files, pkg/validate for artifact files
- Output violations as JSON for hook consumption
- Handle concurrent invocations (multiple files being edited)

This command replaces the alpha backstop-hook binary and the raw semgrep
invocations in the hook scripts (runtime-hooks bundle DD-1, REQ-013).

### backstop gate — Kill Chain

backstop gate runs the full verification kill chain (ADR-0010):
1. Artifact validation (all specs, plans, directives)
2. Pack rule enforcement (semgrep against full scope)
3. Test verification (mandated test names exist and pass)
4. Test substantiveness (AST analysis of test bodies)
5. Coverage threshold verification
6. Contract signature verification
7. Baseline comparison (no new violations above baseline)
8. Waiver resolution (active waivers suppress matching rules)
9. Ledger integrity verification (hash chain, completeness)

The gate is the mechanical embodiment of "if it's green, it ships" (D-059).
Its output is the input to the GitHub Actions gate action (ADR-0009).

### Output Formatting

Two output modes sharing identical underlying data:
- **JSON mode (--json):** Structured JSON to stdout. This is the API contract.
  Versioned, stable, machine-parseable. Agents always use this.
- **Human mode (default):** Formatted terminal output with colors, tables,
  summaries. Respects NO_COLOR and TERM environment variables.

The output layer is a formatter that takes the library's result types and
serializes them. Zero logic in the formatter — just presentation.

## Draft Design Decisions

- **DD-1:** Cobra (or similar) as the command framework. Standard Go CLI
  pattern, supports subcommands, flag parsing, help generation, shell
  completion. The command tree is the API surface.

- **DD-2:** Each command is a thin adapter — no business logic in cmd/.
  Commands parse flags, call pkg/ functions, format output. Testable by
  testing the library; CLI tests are integration tests only.

- **DD-3:** go:embed for all schemas and the baseline rule pack. Each CLI
  version is a self-contained schema cohort. No runtime filesystem dependency
  for core enforcement. Library consumers can override via explicit paths.

- **DD-4:** backstop.yml is loaded and validated before any enforcement
  command executes. Invalid manifest is exit code 2, not exit code 1.
  Config errors and enforcement violations are distinct failure modes.

- **DD-5:** JSON output format is a versioned API contract. Evolution
  follows D-070: additive fields only, deprecated fields warn, breaking
  removals require major CLI version bumps.

- **DD-6:** Changed-files-only as the default scope for validate (D-037).
  Merge-base detection via git. Non-git projects fall back to --scope all.
  CI environments may provide the merge base via environment variables.

- **DD-7:** backstop check has a hard 2-second execution budget. If
  enforcement takes longer, it returns a timeout warning rather than
  blocking the agent. Performance is a correctness requirement for hooks.

- **DD-8:** Pack commands follow a lockfile-first model. backstop pack
  install writes to backstop.yml AND .backstop/pack-lock.yml atomically.
  No state where the manifest and lockfile disagree.

- **DD-9:** backstop init generates runtime hooks for all declared runtimes
  (backstop.yml runtimes field). The hooks call backstop check, replacing
  the alpha raw-tool invocations from the runtime-hooks bundle.

- **DD-10:** backstop gate is the superset — it runs everything validate
  does plus test verification, coverage, contracts, and ledger integrity.
  validate is the fast path (artifact + pack rules). gate is the full kill
  chain.

## Draft Requirements

- The CLI must produce identical violation data in both JSON and human output modes
- backstop validate must default to changed-files scope and support --scope all
- backstop validate --json output must conform to a versioned JSON schema
- backstop validate exit codes: 0 (pass), 1 (violations), 2 (config error)
- backstop check --file must complete within 2 seconds
- backstop check must route by file type using compiled enforcement manifests
- backstop init must work fully offline using embedded schemas and baseline rules
- backstop init must generate backstop.yml, .backstop/ directory, and runtime hooks
- backstop new must auto-assign the next available artifact ID
- backstop new must support all artifact types: spec, plan, issue, adr, directive, bundle, capability
- backstop baseline must scan the full codebase and record violations per rule per file
- backstop baseline --update must lower counts but refuse to increase them (ratchet)
- backstop compile must produce enforcement manifests from .standard.md files
- backstop hooks sync must regenerate hooks without clobbering user-defined hooks
- backstop pack install must write to backstop.yml and pack-lock.yml atomically
- backstop pack list must show installed packs with versions and integrity status
- backstop pack sync must resolve the full dependency graph and update the lockfile
- backstop gate must run the complete verification kill chain (ADR-0010)
- backstop status must show current directive, phase, and progress
- backstop commands --json must return the full command tree for agent discovery
- All schemas must be embedded via go:embed — no runtime filesystem dependency
- The embedded baseline rule pack must include CWE Top 25 and OWASP Top 10 rules
- backstop.yml must be loaded and validated before any enforcement command
- Pack lockfile must use exact version pins with integrity hashes — no ranges

## Spec Seeds

### Bootstrap priority (implement first)

1. **CLI Foundation (cmd/backstop, output layer, config loading):**
   Cobra command skeleton, go:embed schema cohort, backstop.yml loader and
   validator, JSON/human output formatting, exit code handling, version
   command, help/commands discovery. The `backstop artifact` and `backstop code`
   namespaces with shared infrastructure. This is the frame everything hangs on.

2. **backstop artifact validate:**
   Wraps pkg/validate for all artifact types. Supports `--spec SPEC-002` style
   scoping. JSON and human output. Exit codes: 0/1/2.

3. **backstop artifact new:**
   Artifact scaffolding for all seven types. Auto-ID assignment (scan existing
   artifacts, find next available). Template rendering. Slug handling.

4. **backstop artifact compile:**
   Wraps pkg/compile. CLI adapter for standards compilation. Already exists as
   a library; the spec covers the thin CLI wrapper.

5. **backstop code check:**
   Implementation validation: lint, build, test, semgrep. Default scope is
   changed files (--diff implicit). --all for full codebase. --file for single-
   file hook dispatch with 2-second budget. Replaces alpha backstop-hook binary.

6. **backstop gate:**
   Full reconciliation. Artifact validation + code check + claim→test tracing +
   contract verification + coverage thresholds + ledger integrity. The kill
   chain. Depends on the verifier (claim→test tracing) which doesn't exist yet
   — initial implementation runs everything we have and surfaces gaps.

### Future specs (not blocking bootstrap)

7. **backstop init** — project scaffolding, language detection, backstop.yml
   generation, runtime hook generation, interactive session.
8. **backstop baseline** — violation recording, ratchet logic.
9. **backstop pack** — install, list, sync, remove, update, search, vendor.
   Registry client, lockfile management, dependency resolution.
10. **backstop pack publish** — registry-as-publisher pipeline.
11. **backstop hooks sync** — runtime hook regeneration via provider interface.
12. **backstop status** — directive/phase/progress display.

## Open Questions

### OQ-1: Command framework choice

Which Go CLI framework should the CLI use?

- **(a) Cobra** — Industry standard. Used by kubectl, docker, gh. Rich
  subcommand support, flag parsing, shell completion, help generation.
  Heavy dependency but well-maintained.
- **(b) urfave/cli** — Lighter alternative. Simpler API. Less ecosystem
  tooling but adequate for the command surface.
- **(c) Custom with flag package** — Minimal dependencies. Full control.
  More code to write and maintain. No shell completion for free.

**Lean:** (a) Cobra. The command surface is large enough (20+ commands)
that a real framework pays for itself. Shell completion and help generation
matter for the human UX. Agent UX doesn't care about the framework — it
cares about the JSON output.

### OQ-2: JSON output schema versioning strategy

How should the CLI's JSON output schema be versioned?

- **(a) Embedded in CLI version** — JSON schema is part of the CLI's semver.
  Breaking JSON changes require a CLI major version bump. Simple but couples
  JSON schema evolution to CLI release cadence.
- **(b) Independent version field in JSON output** — Each JSON response
  includes a `schema_version` field. CLI can evolve JSON independently of
  binary version. More flexible but adds a version to track.
- **(c) Content-type header style** — `backstop validate --json --api-version 2`.
  Explicit version selection. Most flexible, most complex.

**Lean:** (b). Include a `schema_version` field in every JSON response.
The CLI version implies a default schema version, but the JSON contract
evolves independently. This matches how backstop handles artifact schema
evolution (D-070).

### OQ-3: backstop init interactive session — scope and implementation

ADR-0018 (D-104) envisions backstop init dropping into an interactive agent
session that guides the user through creating their first bundle. How much
of this is in scope for the CLI bundle vs the workflow engine?

- **(a) Full interactive session in CLI** — The CLI itself drives the
  conversation, using the agent runtime to interact with the user. The
  CLI is the workflow engine for init.
- **(b) CLI scaffolds, then hands off** — backstop init creates the
  project structure (backstop.yml, .backstop/, hooks) and prints guidance.
  The interactive session is a runtime hook or separate command.
- **(c) CLI scaffolds with --interactive flag** — Default is silent
  scaffolding. --interactive (or detection of a TTY) triggers the guided
  session. The session logic lives in a separate package.

**Lean:** (c). The CLI's core job is scaffolding — backstop.yml, .backstop/,
hooks. The interactive agent session is a higher-order feature that depends
on a runtime being available. Separating them means init works in CI
(non-interactive) and in a developer's terminal (interactive). The
interactive session is a spec seed of its own or part of the workflow engine
bundle.

### OQ-4: Monorepo service scoping for validate and check

How should the CLI handle monorepo service boundaries?

- **(a) --service flag on validate** — Explicit service selection.
  Scopes changed-files detection and pack selection to the service's
  path and configuration. Requires backstop.yml services: block.
- **(b) Automatic detection from cwd** — If you're in services/api-gateway/,
  backstop figures out which service you're in. Convenient but fragile.
- **(c) Both** — Auto-detect from cwd, allow --service override. Auto-detect
  is a convenience; the flag is the contract.

**Lean:** (c). Auto-detect is convenient for humans; --service is required
for agents and CI. The flag is the API contract.

### OQ-5: Pack management — registry client architecture

The pack commands need a registry client. How should this be structured?

- **(a) Built into the CLI** — Registry client lives in pkg/registry/ and
  the CLI calls it directly. Simplest path. Registry protocol is an internal
  detail.
- **(b) Separate registry SDK** — pkg/registry/ is a standalone package
  that could be used by other tools (GitHub Actions, IDE plugins). More
  abstraction, more surface area.
- **(c) CLI shells out to a registry CLI** — The registry has its own CLI
  tool. backstop pack commands wrap it. Adds a dependency but separates
  concerns.

**Lean:** (a). The registry client is a Go package (pkg/registry/) that the
CLI imports. The Actions can also import it. No separate binary, no extra
dependency. The registry HTTP API is the contract; the client is an
implementation detail.

### OQ-6: Baseline storage format

backstop baseline records pre-existing violations. What's the storage format?

- **(a) Per-rule-per-file counts** — .backstop/baseline.yml maps rule IDs
  to file paths to violation counts. Compact. Ratchet compares counts.
  Doesn't track specific line numbers (violations move as code changes).
- **(b) Full violation snapshots** — Store the complete violation list.
  More data but enables richer diff reporting ("these specific violations
  were fixed"). Larger file, more churn in git.
- **(c) Per-rule aggregate counts** — Only track total violations per rule,
  not per file. Simplest. Loses file-level granularity.

**Lean:** (a). Per-rule-per-file counts balance granularity with stability.
Line numbers are meaningless across commits (code moves). File-level
granularity is enough for "this file got better" reporting. Total counts
are too coarse — a new violation in one file masked by a fix in another
should not pass the ratchet.

### OQ-7: backstop gate vs backstop validate — command boundary

Gate and validate overlap significantly. Where exactly is the boundary?

- **(a) Gate is validate + verifier** — validate does artifact validation
  and pack rules. gate adds test verification, coverage, contracts, and
  ledger. Clear boundary at "does semgrep/schema handle this?"
- **(b) Gate is a mode of validate** — `backstop validate --gate` runs
  the full kill chain. Fewer top-level commands. But muddies the exit
  code semantics (violation vs verification failure).
- **(c) Gate is an entirely separate pipeline** — Gate reads the output
  of validate plus other check results and makes an aggregate decision.
  More like the GitHub Actions gate (ADR-0009).

**Lean:** (a). validate is the fast path that agents run frequently during
implementation. gate is the full kill chain that runs at the transition
boundary. Different commands for different moments in the workflow. The
GitHub Actions gate action can call backstop gate directly.

### OQ-8: How should backstop new handle ID assignment in concurrent environments?

Auto-assigning the next available ID (SPEC-0042) requires scanning existing
artifacts. In a team environment, two developers running backstop new spec
simultaneously could get the same ID.

- **(a) Local scan only, accept collisions** — Scan the local filesystem
  for existing IDs. If two people get the same ID, git merge will conflict.
  Simple, git is the lock.
- **(b) Registry-backed ID reservation** — The hosted layer assigns IDs.
  Requires network. Overkill for a local CLI operation.
- **(c) Include randomness** — SPEC-0042-a7f3. Unique enough to avoid
  collisions. Ugly IDs that humans won't want to reference in conversation.
- **(d) Branch-aware scan** — Scan local + fetch remote to reduce collision
  probability. Network-optional enhancement over (a).

**Lean:** (a). Git merge conflict is the natural resolution for concurrent
ID assignment. The probability is low (how often do two people create the
same artifact type at the same instant?), and the resolution is standard
git workflow. Over-engineering this is a trap.

### OQ-9: Semgrep as a runtime dependency — installation and version pinning

backstop check and backstop validate need semgrep. How does the CLI ensure
it's available?

- **(a) CLI checks for semgrep on PATH** — If not found, error with
  installation instructions. User manages semgrep installation. Simplest.
- **(b) CLI embeds semgrep** — Ship semgrep as part of the backstop binary.
  No external dependency. Massive binary size increase. Licensing concerns
  (semgrep is LGPL-2.1).
- **(c) CLI auto-installs semgrep** — On first run, download and cache
  semgrep in .backstop/tools/. Version pinned in backstop.yml or the
  lockfile. Convenient but network-dependent on first run.
- **(d) backstop.yml declares semgrep version** — The manifest pins the
  semgrep version. The CLI verifies the installed version matches. Mismatch
  is a config error (exit 2).

**Lean:** (a) with (d) as an enhancement. Check PATH, report missing. Pin
the version in backstop.yml for reproducibility. Auto-install is a future
convenience, not MVP. The embedded baseline rules (D-099) work without
semgrep for pure artifact validation, so backstop validate on artifacts
works even without semgrep installed.

### OQ-10: backstop hooks sync — scope relative to runtime-hooks bundle

The runtime-hooks bundle (defined maturity) specifies the hook generation
system in detail. How does the CLI command relate to that bundle's specs?

- **(a) backstop hooks sync calls the runtime provider interface** — The
  CLI command is a thin wrapper around the RuntimeProvider.Generate() from
  the runtime-hooks bundle. The hook generation logic is in pkg/hooks/.
  The CLI just calls it.
- **(b) backstop hooks sync is independently specified** — The CLI spec
  covers the command surface, the runtime-hooks specs cover the generation
  logic. They meet at the package boundary.
- **(c) backstop hooks sync IS one of the runtime-hooks specs** — The
  command is specified as part of the runtime-hooks bundle's spec
  decomposition, not the CLI bundle.

**Lean:** (b). The CLI bundle specifies the command surface (flags, output,
behavior). The runtime-hooks bundle specifies the generation logic (what
files to produce, how to merge settings.json). They compose at the
pkg/hooks/ package boundary. This avoids either bundle needing to own the
other's scope.

### OQ-11: Pack dependency resolution — algorithm choice

ADR-0017 says "Go's minimum version selection." How strictly should this
be followed?

- **(a) Strict MVS** — Exactly Go's algorithm. Select the minimum version
  that satisfies all constraints. Deterministic. Well-understood. May
  surprise users who expect "latest compatible."
- **(b) Latest compatible (npm-style)** — Select the highest version that
  satisfies all constraints. More familiar to the npm/cargo audience. Less
  deterministic without a lockfile (but we always have a lockfile).
- **(c) MVS with --latest flag** — Default is MVS for determinism. --latest
  flag during update resolves to highest compatible. Best of both worlds.

**Lean:** (a). MVS is deterministic and well-understood. The lockfile
provides the actual pinning. MVS only matters during `pack sync` and
`pack update` — the lockfile governs runtime. Users who want the latest
run `backstop pack update`.

### OQ-12: Should the CLI support a --config flag to override backstop.yml location?

- **(a) No** — backstop.yml is always at the project root. Convention over
  configuration. The project root is detected by walking up from cwd to
  find backstop.yml (similar to how go.mod works).
- **(b) Yes, --config flag** — Allows non-standard locations. Useful for
  testing, monorepo edge cases, and CI with unusual directory structures.
- **(c) BACKSTOP_CONFIG env var** — Environment variable override. Less
  visible than a flag but useful for CI.

**Lean:** (a) with (c) as an escape hatch. The project root discovery
(walk up to find backstop.yml) is the primary mechanism. An env var
provides an override for edge cases without cluttering every command's
flag set.

### OQ-13: How should backstop validate report waiver expirations approaching?

Waivers have expiry dates (D-075). Should validate warn about upcoming
expirations?

- **(a) Warn at 30 days** — Waivers expiring within 30 days appear as
  warnings in validate output. Gives teams time to fix the underlying
  violation.
- **(b) Configurable threshold** — `enforcement.waiver_warning_days: 30`
  in backstop.yml. Different teams have different planning horizons.
- **(c) No warning — fail on expiry only** — The waiver either suppresses
  or it doesn't. Approaching expiry is a project management concern, not
  a validation concern.

**Lean:** (b). Configurable with a sensible default (30 days). The warning
shows up in validate output and CI, creating natural pressure to address
the underlying violation before the waiver expires.

## Resolved Questions (Bootstrap)

**OQ-1: Command framework** → Cobra. Industry standard, handles the nested
namespace structure (artifact, code, pack) natively. Shell completion free.

**OQ-2: JSON output versioning** → Independent `schema_version` field in every
JSON response. CLI ships with all prior schema versions embedded via go:embed,
matching the artifact schema cohort model.

**OQ-7: Gate vs validate boundary** → Three separate concerns: `backstop artifact
validate` (schema conformance), `backstop code check` (lint/build/test/semgrep),
`backstop gate` (full reconciliation spanning both domains + verifier). Resolved
during bundle scoping discussion.

**OQ-8: Concurrent ID assignment** → Git annotated tags for atomic ID reservation.
Format: `backstop/<type>/<number>`. `git push --tags` is the atomic claim. Fetch
before assign, retry on conflict. Gaps from unused reservations are acceptable.
Offline fallback to local filesystem scan.

**OQ-9: Semgrep dependency** → Auto-install on first run. Download to
`.backstop/tools/`, pin version in backstop.yml, verify on subsequent runs.
Artifact validation works without semgrep. `backstop code check` triggers the
install if semgrep is missing.

**OQ-12: Config file discovery** → Walk up from cwd to find backstop.yml (like
go.mod discovery). `BACKSTOP_CONFIG` env var as escape hatch for CI edge cases.
No --config flag.

**OQ-3: backstop init interactive session** → No interactive mode in v1. Init
scaffolds silently: detects language from cwd, installs default standards,
compiles enforcement manifests, runs baseline scan, generates runtime hooks,
prints "start an agent session" guidance. The interactive agent session from
ADR-0018 is deferred to the workflow engine.

**OQ-4: Monorepo service scoping** → (c) Auto-detect from cwd + --service flag
override. Auto-detect is a convenience for humans; --service is the contract for
agents and CI. Deferred to post-bootstrap scope.

**OQ-5: Pack registry client architecture** → (a) Registry client built into CLI
as pkg/registry/. The CLI imports it directly; GitHub Actions can also import it.
No separate binary. The registry HTTP API is the contract; the client is an
implementation detail.

**OQ-6: Baseline storage format** → (a) Per-rule-per-file counts in
.backstop/baseline.yml. Maps rule IDs to file paths to violation counts. Ratchet
compares at file level. Line numbers are meaningless across commits. File-level
granularity prevents a fix in one file from masking a new violation in another.

**OQ-10: hooks sync scope** → (b) Split ownership at the package boundary. The
CLI bundle specifies the command surface (flags, output, behavior) for
`backstop hooks sync`. The runtime-hooks bundle specifies the generation logic
(what files to produce, how to merge settings.json) in pkg/hooks/. They compose
at the package boundary without either bundle owning the other's scope.

**OQ-11: Pack dependency resolution** → (a) Strict MVS (Go's minimum version
selection). Deterministic, well-understood. The lockfile provides actual pinning
at runtime. MVS only matters during `pack sync` and `pack update`. Users who
want latest run `backstop pack update`.

**OQ-13: Waiver expiration warnings** → (b) Configurable waiver warning threshold.
`enforcement.waiver_warning_days: 30` in backstop.yml (30-day default). Waivers
approaching expiry appear as warnings in validate output and CI, creating natural
pressure to address the underlying violation before expiry.

## Version History

- 0.4.0 (2026-04-04): All 13 OQs resolved. Moved 7 deferred OQs (3, 4, 5, 6,
  10, 11, 13) into Resolved Questions with full rationale. Key resolutions:
  init is silent with language detection and baseline scan (no interactive mode
  in v1), registry client built into CLI as pkg/registry/, baseline uses
  per-rule-per-file counts, hooks sync splits ownership at package boundary,
  strict MVS for dependency resolution, configurable waiver warning threshold.
  Updated backstop init section in Current Thinking to reflect the new init
  experience. Confirmed maturity at defined.
- 0.3.0 (2026-04-04): Advanced to defined maturity. Added success criteria
  (9 criteria focused on bootstrap commands), assumptions (7 dependencies),
  and formal requirements (REQ-001 through REQ-011). Bootstrap scope locked:
  artifact validate, artifact new, code check, pack compile, pack new, gate.
  7 OQs remain deferred to future scope.
- 0.2.0 (2026-04-04): Resolved 6 bootstrap OQs. Renamespaced commands to
  artifact/code/pack/gate. Scoped bootstrap priority (6 specs) vs future
  (6 specs). Key decisions: Cobra, JSON schema_version field, git tags for
  ID reservation, semgrep auto-install, cwd walk for config discovery.
- 0.1.0 (2026-04-04): Initial bundle at exploring maturity. Captured full
  CLI scope across three command tiers (core, pack/registry, workflow).
  Identified 13 open questions spanning command framework, output versioning,
  init experience, monorepo scoping, registry architecture, baseline format,
  gate boundaries, ID assignment, semgrep dependency, hooks integration,
  dependency resolution, config discovery, and waiver warnings. Suggested
  11-spec decomposition in implementation order.

## References

- ADR-0004: Validation Engine Architecture (pkg/validate, three consumers)
- ADR-0005: Backstop.yml Manifest (config format, ceremony dial)
- ADR-0006: Standards Packs (semgrep engine, waivers, baseline)
- ADR-0008: CLI Design (commands, output philosophy, exit codes, D-037)
- ADR-0009: CI/CD Pipeline (GitHub Actions, gate action)
- ADR-0010: Verification Kill Chain (gate command scope)
- ADR-0014: Runtime Integration (hooks, backstop check)
- ADR-0017: Pack Registry and Plugin Architecture (D-089–D-099)
- ADR-0018: Workflow State Machine (init experience, D-100–D-104)
- Bundle: standards-compiler (enforcement manifest model, CLI orchestration)
- Bundle: runtime-hooks (hook generation, backstop check replaces alpha hooks)
- D-028: Schemas embedded via go:embed
- D-037: Changed-files-only default scope
- D-059: "If it's green, it ships"
- D-069: CLI is the universal agent API
- D-070: Backward-compatible schema evolution
- D-076: Baseline scan for adoption amnesty
- D-099: Offline bootstrapping with embedded baseline rules
