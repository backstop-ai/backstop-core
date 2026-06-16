---
title: "SPEC-032: Pack Fixture Engine Execution"
number: SPEC-032
created: "2026-06-16"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Generalize the validation-time fixture-execution path (locus B, pkg/packval,
    the `pack test` / phase-3 path) onto the same shared engine model that the
    gate-time path (SPEC-031, locus A) uses. Today phase 3 dispatches fixtures
    through a bespoke FixtureExecutor whose arms are hardcoded one-per-tool:
    RunSemgrep (semgrep, `--config rule fixture`) and RunToolConfig (a one-arm
    `switch tool` on golangci-lint). This spec replaces those two findings arms
    with a single engine-dispatched arm driven by the rule's declared `engine`
    field and its EngineBinding ({command, input_mode, input_flag, convert?}),
    resolved through the same engine table SPEC-031 builds. A fixture run becomes:
    resolve the rule's EngineBinding, gather the rule's inputs per input_mode,
    invoke the command capturing stdout cleanly (not CombinedOutput), pipe through
    the pack-declared `convert` step when present, parse the resulting SARIF to
    determine whether the engine flagged the fixture, and apply the existing
    positive-clean / negative-must-trigger fixture contract unchanged. The
    non-findings arms (layer-3 sandbox validators via RunValidator, scaffold tests
    via RunScaffoldTest) are exit-code edges and do NOT converge onto SARIF; they
    keep their current behavior. The positive/negative fixture semantics, the
    PhaseResult/ValidationError shape, the negative-fixture engine-limitation fix
    hint, and the rule-ID-match precheck are all preserved — only the per-tool
    dispatch is replaced by engine dispatch.
  package: pkg/packval

verification:
  level: unit
  test_command: go test ./pkg/packval/... -run TestPackVal_Engine -race -coverprofile=cover.out
  coverage_threshold: 90

requirements:
  - id: REQ-001
    text: >
      The fixture-execution path (phase 3, pkg/packval) must dispatch findings-rule
      fixtures on the rule's declared `engine` field through the shared EngineBinding
      model, not through the bespoke per-tool RunSemgrep method or the one-arm
      RunToolConfig `switch tool`. A fixture run for a findings rule must resolve the
      rule's EngineBinding from the shared engine table and execute that binding's
      command, exactly as the gate-time path (SPEC-031) does for the same rule.
      Adding a new findings engine (ast-grep, eslint, ruff, clippy) must require no
      new arm in the fixture executor — it is satisfied entirely by the engine
      declaration plus, at most, a registered runner shared with the gate path.
    supports: pluggable-pack-engines:REQ-012
    follows: error-handling-recipe
  - id: REQ-002
    text: >
      Fixture execution must gather a findings rule's inputs by the EngineBinding's
      declared `input_mode` enum, with no per-engine Go branch:
      `config-file` passes a single optional pack-supplied config via `input_flag`
      and the tool runs its own built-in rules;
      `rule-flags` passes each rule file as a repeated `input_flag` occurrence
      (e.g. `--config X`); `rule-dir` collects the engine's rule files into a
      directory passed via `input_flag`; `none` injects no rules/config and runs the
      command as-is. The dispatch must be table-driven over `input_mode`, identical
      in behavior to the gate-time gather step, so a single fixture and a single
      gate run of the same rule assemble the same invocation.
    supports: pluggable-pack-engines:REQ-012
  - id: REQ-003
    text: >
      For findings engines, fixture execution must determine pass/fail from parsed
      SARIF, not from raw exit codes or raw tool JSON. The executor must capture the
      command's stdout cleanly and separately from stderr (replacing CombinedOutput,
      which merges stderr into stdout and corrupts SARIF), pipe stdout through the
      EngineBinding's pack-declared `convert` executable when present (`tool stdout →
      convert stdin → SARIF`, run in-process with no shell), and parse the resulting
      SARIF to decide whether the engine flagged the fixture. A non-empty SARIF
      results set for the rule's ID means the engine flagged the fixture; an empty
      set means it did not.
    supports: pluggable-pack-engines:REQ-012
  - id: REQ-004
    text: >
      The positive/negative fixture contract must be preserved exactly under engine
      dispatch and applied uniformly across every findings engine: a positive fixture
      passes only when the engine does NOT flag it (clean), and fails when it is
      flagged (false positive); a negative fixture passes only when the engine DOES
      flag it, and fails when it is not flagged. A command error (the engine binary
      failing to run, or a `convert`/SARIF-parse failure) is a hard fixture error
      distinct from a clean run, and must be reported as such — it must never be
      silently treated as "not flagged."
    supports: pluggable-pack-engines:REQ-012
  - id: REQ-005
    text: >
      When a negative fixture is not flagged by its engine, the existing
      engine-limitation fix hint must still be emitted on the resulting
      ValidationError, unchanged in intent: it must explain that the fixture may
      represent a pattern the engine cannot detect and should be removed and
      documented rather than shipped as an untestable claim. This hint must apply to
      every findings engine, not only semgrep.
    supports: pluggable-pack-engines:REQ-012
  - id: REQ-006
    text: >
      Non-findings fixture arms must NOT converge onto the engine/SARIF path and must
      retain their current exit-code semantics. Layer-3 sandbox validators (the
      `none`-mode custom-script edge) continue to run via RunValidator / SandboxedRun
      with exit 0 = pass and non-zero = fail, including the multi-file input_scope
      aggregation. Scaffold test execution (RunScaffoldTest) continues to run its
      test_command by exit code. SARIF parsing must not be applied to either; doing so
      is prohibited.
    supports: pluggable-pack-engines:REQ-012
  - id: REQ-007
    text: >
      A findings rule that declares an `engine` for which no EngineBinding is
      resolvable from the shared engine table must produce a loud, blocking phase-3
      error (a hard ValidationError identifying the rule and the unresolved engine),
      not a silent skip and not a default-to-semgrep fallback. Misdeclaration is a
      class-1 broken declaration: loud AND blocking.
    supports: pluggable-pack-engines:REQ-012
    follows: error-handling-recipe
  - id: REQ-008
    text: >
      The existing pre-execution rule-ID-match precheck must be retained and re-keyed
      to the engine. For an engine whose rule files carry rule IDs (semgrep-shaped),
      phase 3 must still verify the pack rule ID is present in the referenced rule
      file before running fixtures. For engines whose input_mode carries no
      per-rule-file ID (`none`, and `config-file` where the tool runs its own rules),
      the precheck must be skipped rather than spuriously failing.
    supports: pluggable-pack-engines:REQ-012
  - id: REQ-009
    text: >
      The shared EngineBinding type and engine table consumed here must be the same
      ones SPEC-031 defines and places to avoid an import cycle across pkg/check,
      pkg/packval, and cmd/backstop. pkg/packval must import that shared type rather
      than redefining an EngineBinding or an engine enum locally. No second,
      packval-local copy of the engine model may exist.
    supports: pluggable-pack-engines:REQ-012

claims:
  # --- REQ-001: engine dispatch replaces per-tool arms ---
  - id: CLM-001
    requirement: REQ-001
    text: Phase 3 dispatches a findings rule's fixtures by resolving its declared engine's EngineBinding, not by calling a per-tool RunSemgrep/RunToolConfig arm
    tests:
      - TestPackVal_EngineDispatch_ResolvesBindingForDeclaredEngine
  - id: CLM-002
    requirement: REQ-001
    text: A semgrep-engine rule's fixtures execute the semgrep EngineBinding command and assemble the same invocation the gate path would
    tests:
      - TestPackVal_EngineDispatch_SemgrepRuleUsesBinding
  - id: CLM-003
    requirement: REQ-001
    text: An ast-grep-engine rule's fixtures execute via engine dispatch with no new fixture-executor arm added for it
    tests:
      - TestPackVal_EngineDispatch_AstGrepRuleUsesBinding
  - id: CLM-004
    requirement: REQ-001
    text: A rule declaring a newly registered findings engine dispatches purely from its declaration with no packval code change
    tests:
      - TestPackVal_EngineDispatch_NewEngineNoExecutorArm

  # --- REQ-002: input_mode gather matrix (config-file / rule-flags / rule-dir / none) ---
  - id: CLM-005
    requirement: REQ-002
    text: input_mode config-file passes a single pack-supplied config via input_flag and injects no rule files
    tests:
      - TestPackVal_EngineGather_ConfigFile
  - id: CLM-006
    requirement: REQ-002
    text: input_mode rule-flags passes each rule file as a repeated input_flag occurrence
    tests:
      - TestPackVal_EngineGather_RuleFlags
  - id: CLM-007
    requirement: REQ-002
    text: input_mode rule-dir collects rule files into a directory passed via input_flag
    tests:
      - TestPackVal_EngineGather_RuleDir
  - id: CLM-008
    requirement: REQ-002
    text: input_mode none injects no rules or config and runs the command as-is
    tests:
      - TestPackVal_EngineGather_None
  - id: CLM-009
    requirement: REQ-002
    text: The fixture gather step produces the identical invocation the gate-time gather step produces for the same rule
    tests:
      - TestPackVal_EngineGather_MatchesGateInvocation

  # --- REQ-003: SARIF determines pass/fail; clean stdout capture; convert pipe ---
  - id: CLM-010
    requirement: REQ-003
    text: Findings pass/fail is derived from parsed SARIF results, not from the raw exit code
    tests:
      - TestPackVal_EngineSarif_PassFailFromSarif
  - id: CLM-011
    requirement: REQ-003
    text: A non-empty SARIF result set for the rule ID counts as the engine flagging the fixture
    tests:
      - TestPackVal_EngineSarif_NonEmptyMeansFlagged
  - id: CLM-012
    requirement: REQ-003
    text: An empty SARIF result set counts as the engine not flagging the fixture
    tests:
      - TestPackVal_EngineSarif_EmptyMeansNotFlagged
  - id: CLM-013
    requirement: REQ-003
    text: The executor captures stdout separately from stderr so stderr noise does not corrupt SARIF parsing
    tests:
      - TestPackVal_EngineSarif_StdoutSeparateFromStderr
  - id: CLM-014
    requirement: REQ-003
    text: A SARIF-native engine binding with no convert step parses stdout directly as SARIF
    tests:
      - TestPackVal_EngineConvert_NativeSarifNoConvert
  - id: CLM-015
    requirement: REQ-003
    text: A non-SARIF engine binding pipes stdout through the declared convert executable before SARIF parsing
    tests:
      - TestPackVal_EngineConvert_NonSarifPipesThroughConvert

  # --- REQ-004: positive/negative contract preserved and command-error distinction ---
  - id: CLM-016
    requirement: REQ-004
    text: A positive fixture passes when the engine does not flag it
    tests:
      - TestPackVal_EngineContract_PositiveCleanPasses
  - id: CLM-017
    requirement: REQ-004
    text: A positive fixture fails when the engine flags it (false positive)
    tests:
      - TestPackVal_EngineContract_PositiveFlaggedFails
  - id: CLM-018
    requirement: REQ-004
    text: A negative fixture passes when the engine flags it
    tests:
      - TestPackVal_EngineContract_NegativeFlaggedPasses
  - id: CLM-019
    requirement: REQ-004
    text: A negative fixture fails when the engine does not flag it
    tests:
      - TestPackVal_EngineContract_NegativeNotFlaggedFails
  - id: CLM-020
    requirement: REQ-004
    text: A command/convert/SARIF-parse failure is reported as a hard fixture error, distinct from a clean run and never treated as not-flagged
    tests:
      - TestPackVal_EngineContract_CommandErrorIsHardError

  # --- REQ-005: engine-limitation fix hint preserved for all engines ---
  - id: CLM-021
    requirement: REQ-005
    text: A negative fixture not flagged by a semgrep-engine rule still emits the engine-limitation fix hint
    tests:
      - TestPackVal_EngineHint_SemgrepNegativeNotFlaggedHint
  - id: CLM-022
    requirement: REQ-005
    text: A negative fixture not flagged by a non-semgrep findings engine emits the same engine-limitation fix hint
    tests:
      - TestPackVal_EngineHint_AstGrepNegativeNotFlaggedHint

  # --- REQ-006: non-findings arms stay exit-code, must not ride SARIF ---
  - id: CLM-023
    requirement: REQ-006
    text: Layer-3 sandbox validator fixtures still run via RunValidator/SandboxedRun on exit-code semantics
    tests:
      - TestPackVal_NonFindings_ValidatorExitCodeUnchanged
  - id: CLM-024
    requirement: REQ-006
    text: Layer-3 multi-file input_scope validator aggregation is preserved
    tests:
      - TestPackVal_NonFindings_ValidatorMultiFileUnchanged
  - id: CLM-025
    requirement: REQ-006
    text: Scaffold test_command fixtures still run by exit code, not SARIF
    tests:
      - TestPackVal_NonFindings_ScaffoldExitCodeUnchanged
  - id: CLM-026
    requirement: REQ-006
    text: SARIF parsing is not applied to validator or scaffold runs
    tests:
      - TestPackVal_NonFindings_NoSarifAppliedToExitCodeArms

  # --- REQ-007: unresolved engine is loud and blocking ---
  - id: CLM-027
    requirement: REQ-007
    text: A findings rule declaring an engine with no resolvable EngineBinding produces a hard phase-3 error identifying the rule and engine
    tests:
      - TestPackVal_EngineUnresolved_HardError
  - id: CLM-028
    requirement: REQ-007
    text: An unresolved engine does not silently skip the rule's fixtures
    tests:
      - TestPackVal_EngineUnresolved_NoSilentSkip
  - id: CLM-029
    requirement: REQ-007
    text: An unresolved engine does not fall back to semgrep
    tests:
      - TestPackVal_EngineUnresolved_NoSemgrepFallback

  # --- REQ-008: rule-ID-match precheck re-keyed to engine ---
  - id: CLM-030
    requirement: REQ-008
    text: For a rule-file-ID-bearing engine, the pack rule ID must be present in the referenced rule file or phase 3 fails before running fixtures
    tests:
      - TestPackVal_EnginePrecheck_RuleIDMatchEnforced
  - id: CLM-031
    requirement: REQ-008
    text: For a rule-file-ID-bearing engine, a matching rule ID passes the precheck
    tests:
      - TestPackVal_EnginePrecheck_RuleIDMatchPasses
  - id: CLM-032
    requirement: REQ-008
    text: For an input_mode none engine, the rule-ID-match precheck is skipped rather than failing
    tests:
      - TestPackVal_EnginePrecheck_NoneModeSkipsPrecheck
  - id: CLM-033
    requirement: REQ-008
    text: For an input_mode config-file engine running its own rules, the rule-ID-match precheck is skipped rather than failing
    tests:
      - TestPackVal_EnginePrecheck_ConfigFileSkipsPrecheck

  # --- REQ-009: shared EngineBinding, no local copy ---
  - id: CLM-034
    requirement: REQ-009
    text: pkg/packval consumes the shared EngineBinding type rather than declaring a packval-local engine binding or engine enum
    tests:
      - TestPackVal_EngineShared_UsesSharedBindingType
  - id: CLM-035
    requirement: REQ-009
    text: The same engine table that gate-time dispatch uses resolves bindings for fixture-time dispatch
    tests:
      - TestPackVal_EngineShared_SameTableAsGate

contracts:
  - file: pkg/packval/executor.go
    provides:
      - name: FixtureExecutor
        kind: interface
        signature: "type FixtureExecutor interface"
        notes: >
          Retains RunValidator and RunScaffoldTest (exit-code arms) unchanged. The
          two findings arms RunSemgrep and RunToolConfig are replaced by a single
          engine-dispatched method RunEngine(packDir string, binding engine.EngineBinding,
          ruleID string, ruleFiles []string, fixturePath string) (ExecutionResult, error).
      - name: RunEngine
        kind: method
        signature: "RunEngine(packDir string, binding engine.EngineBinding, ruleID string, ruleFiles []string, fixturePath string) (ExecutionResult, error)"
        notes: >
          Gathers inputs per binding.InputMode, runs binding.Command capturing stdout
          cleanly, pipes through binding.Convert when present, parses SARIF, and sets
          ExecutionResult.Passed = engine flagged the fixture (Flagged), with a distinct
          error return for command/convert/parse failure.
      - name: DefaultExecutor
        kind: type
        signature: "type DefaultExecutor struct"
        notes: "Implements RunEngine using real OS commands and the shared SARIF parse."
      - name: MockExecutor
        kind: type
        signature: "type MockExecutor struct"
        notes: "Adds RunEngineFn for phase-3 dispatch tests without real engine binaries."
      - name: ExecutionResult
        kind: type
        signature: "type ExecutionResult struct"
        notes: "Passed reflects whether the engine flagged the fixture; ExitCode/Output/Diagnostics retained for exit-code arms."
    consumes:
      - source: pkg/engine
        name: EngineBinding
        kind: type
      - source: pkg/engine
        name: ResolveBinding
        kind: function
  - file: pkg/packval/phase3.go
    provides:
      - name: RunFixtures
        kind: function
        signature: "func RunFixtures(pack *PackManifest, packDir string, executor FixtureExecutor) *PhaseResult"
        notes: >
          Findings rules now dispatch through executor.RunEngine via the rule's
          resolved EngineBinding; the per-tool RunSemgrep/RunToolConfig loops are
          replaced. Validator/scaffold/SDK arms and the negative-fixture engine-limitation
          fix hint are preserved.
    consumes:
      - source: pkg/engine
        name: EngineBinding
        kind: type
      - source: pkg/engine
        name: ResolveBinding
        kind: function
  - file: pkg/packval/manifest.go
    provides:
      - name: Rule
        kind: type
        signature: "type Rule struct"
        notes: >
          Gains the `engine` field (string) consumed by phase-3 dispatch; `layer`
          retires as the execution selector per SPEC-031's schema cutover.
    consumes: []
---

# SPEC-032: Pack Fixture Engine Execution

## Overview

BUNDLE-010 first-classes the execution engine of a pack rule and dispatches it through an explicit `engine → {command, convert?, input_mode, input_flag, requires[], forbids[]}` table instead of the implicit `layer==2 ⇒ semgrep` routing and the one-arm `RunToolConfig` switch. The bundle identifies **two convergence loci** (DD-3) that both hardcode tool dispatch independently and must both converge onto the shared engine model:

- **Locus A — gate-time engine dispatch** (`pkg/check`, `cmd/backstop`). Replaces the semgrep-only gate executor with group-by-engine dispatch. Owned by **SPEC-031** (Seed 2).
- **Locus B — validation-time fixture execution** (`pkg/packval`, the `pack test` / phase-3 path). The genuinely separate path that runs a rule against its positive/negative fixtures to prove the rule works. **This spec (Seed 3) owns locus B.**

This spec applies the same engine generalization to locus B. Today `pkg/packval` phase 3 runs findings fixtures through a `FixtureExecutor` whose findings arms are hardcoded per tool: `RunSemgrep` invokes `semgrep --config rule fixture`, and `RunToolConfig` is a one-arm `switch tool` on `golangci-lint`. This spec replaces those two arms with a single engine-dispatched arm (`RunEngine`) driven by the rule's declared `engine` field and the shared `EngineBinding` resolved from the same engine table SPEC-031 builds. The result: a pack author can fixture-test an ast-grep rule (or any declared engine's rule) against positive/negative fixtures **identically to how the gate-time path runs them**, and adding a new engine never touches the fixture executor.

The convergence is deliberately scoped to **findings engines** (semgrep, ast-grep, lint). The bundle's DD-2/DD-8 carve-out is preserved: layer-3 sandbox validators and scaffold tests are **exit-code edges, not located findings**, so they keep their current `RunValidator`/`RunScaffoldTest` exit-code semantics and must not ride SARIF. The positive/negative fixture contract, the `PhaseResult`/`ValidationError` shape, the negative-fixture engine-limitation fix hint, and the rule-ID-match precheck are all preserved — only the per-tool dispatch is replaced.

This spec **depends on SPEC-031** for the shared `EngineBinding` type, the engine table, the `input_mode` gather semantics, and the `convert`-pipe + clean-stdout-capture mechanics. It is authored from the bundle independently and re-specifies none of locus A's gate-time dispatch.

## Requirements

Requirements are defined in frontmatter. Claims are defined in frontmatter. This spec covers BUNDLE-010 Seed 3 — `pluggable-pack-engines:REQ-012` (validation-time fixture execution receives the same engine generalization as the gate-time path).

### Fixture-execution arm taxonomy

Every phase-3 fixture arm is exactly one of two kinds. The convergence applies only to the findings kind.

| Arm | Rule shape | Dispatch after this spec | Pass/fail signal | SARIF? |
|-----|-----------|--------------------------|------------------|--------|
| Findings | `engine` is a findings engine (semgrep, ast-grep, lint) | `RunEngine` via resolved `EngineBinding` | Engine flagged the fixture (parsed SARIF) | Yes |
| Sandbox validator | layer-3 custom script (`input_mode none`, exit-code edge) | `RunValidator` / `SandboxedRun` (unchanged) | Exit 0 = pass, non-zero = fail | No (prohibited) |
| Scaffold test | scaffold `test_command` | `RunScaffoldTest` (unchanged) | Exit 0 = pass, non-zero = fail | No (prohibited) |

### input_mode gather matrix (findings dispatch)

The fixture gather step is table-driven over the binding's `input_mode`, with no per-engine Go branch, and must produce the identical invocation the gate-time gather step produces for the same rule (REQ-002).

| input_mode | Rule/config injection | input_flag use | Representative engine | Rule-ID precheck (REQ-008) |
|------------|----------------------|----------------|-----------------------|----------------------------|
| `config-file` | single optional pack-supplied config; tool runs its OWN rules | one flag, the config path | golangci-lint / eslint / tsc | Skipped (no per-rule-file ID) |
| `rule-flags` | each rule file → repeated flag | repeated, once per rule file | semgrep | Enforced (rule file carries IDs) |
| `rule-dir` | rule files collected into a dir | one flag, the dir path | ast-grep | Enforced (rule files carry IDs) |
| `none` | no injection; the executable is the logic | unused | sandbox custom script | Skipped (no rule file) |

### Engine pass/fail (findings)

| Engine binding | stdout handling | SARIF source | Flagged decision |
|----------------|-----------------|--------------|------------------|
| SARIF-native (`{command}`) | captured clean, separate from stderr | stdout directly | non-empty SARIF results for rule ID ⇒ flagged |
| non-SARIF (`{command, convert}`) | captured clean, piped `stdout → convert stdin` | convert output | non-empty SARIF results for rule ID ⇒ flagged |
| command/convert/parse failure | n/a | n/a | hard fixture error (NOT "not flagged") |

## Implementation

### Scope boundary against SPEC-031

SPEC-031 (locus A) defines and places the shared `EngineBinding` type, the `engine → binding` resolution table, the `input_mode` gather logic, the `convert`-pipe execution, the clean stdout/stderr capture, and the shared SARIF parse. This spec **consumes** those from their shared home (`pkg/engine`, the import-cycle-safe placement SPEC-031 owns per `pluggable-pack-engines:REQ-013`) and applies them to the `pkg/packval` fixture path. This spec does not re-specify gate-time dispatch.

### Phase-3 findings dispatch (replaces RunSemgrep + RunToolConfig)

The current phase-3 loops over `pack.Content.Ruleset.Rules` (calling `RunSemgrep`) and `pack.ToolConfig` (calling `RunToolConfig`) collapse into a single findings dispatch:

1. **Resolve the binding.** For each findings rule, resolve its `EngineBinding` from the shared engine table via the rule's declared `engine`. If no binding resolves, emit a hard, blocking `ValidationError` identifying the rule and the unresolved engine (REQ-007) — no silent skip, no semgrep fallback.
2. **Rule-ID precheck (engine-keyed).** For rule-file-ID-bearing input modes (`rule-flags`, `rule-dir`), verify the pack rule ID is present in the referenced rule file(s) before running fixtures, retaining today's `semgrepFileContainsRuleID` check generalized by engine. For `none` and `config-file`, skip the precheck (REQ-008).
3. **Gather inputs by input_mode.** Assemble the invocation per the input_mode gather matrix above (REQ-002), identical to the gate-time gather.
4. **Run + normalize to SARIF.** Invoke `binding.Command` capturing stdout cleanly and separately from stderr (replacing `CombinedOutput`), pipe through `binding.Convert` when present (`tool stdout → convert stdin → SARIF`, in-process, no shell), and parse the SARIF. A command, convert, or parse failure is a hard fixture error distinct from a clean run (REQ-003, REQ-004).
5. **Apply the fixture contract.** `Flagged` = non-empty SARIF results for the rule ID. Positive fixtures pass iff NOT flagged; negative fixtures pass iff flagged (REQ-004). On a negative fixture that is not flagged, emit the existing engine-limitation fix hint on the `ValidationError`, for every engine (REQ-005).

### Non-findings arms (unchanged)

The layer-3 sandbox validator arm (`RunValidator` → `SandboxedRun`, including multi-file `input_scope` aggregation) and the scaffold `test_command` arm (`RunScaffoldTest`) keep their exit-code semantics verbatim. SARIF normalization is not applied to them (REQ-006). The SDK `provides` check and the `go mod tidy` pre-check for Go packs are likewise untouched.

### FixtureExecutor surface change

`RunSemgrep` and `RunToolConfig` are removed from the `FixtureExecutor` interface and replaced by a single `RunEngine` method (see contracts). `RunValidator` and `RunScaffoldTest` remain. `MockExecutor` gains `RunEngineFn` so phase-3 dispatch is testable without real engine binaries, mirroring the existing mock pattern. `ExecutionResult.Passed` now reflects "engine flagged the fixture" for the findings arm and retains exit-code meaning for the non-findings arms.

### Manifest field

`Rule` gains the `engine` field consumed by dispatch. `layer` retires as the execution selector under SPEC-031's schema cutover; this spec reads `engine`. (The schema-validation re-key from layer→engine field-contracts is SPEC-031's; this spec only consumes the resolved `engine`.)

## Verification

Verification config is defined in frontmatter. Claims are defined in frontmatter. The test command targets `pkg/packval` with the `TestPackVal_Engine` prefix; unit-level because the convergence is a within-package dispatch change exercised through a `MockExecutor` (real engine binaries are not required to prove dispatch, gather, SARIF-decision, and contract behavior).

### Test strategy

- **Engine dispatch** — assert findings rules route through `RunEngine` with the resolved binding, and that `RunSemgrep`/`RunToolConfig` are gone from the executor surface.
- **input_mode gather matrix** — one test per mode (`config-file`, `rule-flags`, `rule-dir`, `none`) asserting the assembled invocation, plus a test asserting parity with the gate-time gather.
- **SARIF decision** — flagged/not-flagged from parsed SARIF, clean stdout capture, and the convert-vs-native split, using a `MockExecutor` / canned SARIF.
- **Fixture contract** — the four positive/negative cells plus the command-error-is-hard-error cell.
- **Engine-limitation hint** — preserved for semgrep and a non-semgrep findings engine.
- **Non-findings arms** — validator (single + multi-file) and scaffold exit-code behavior unchanged; SARIF not applied.
- **Unresolved engine** — hard error, no silent skip, no semgrep fallback.
- **Precheck re-key** — enforced for rule-file-ID engines, skipped for `none`/`config-file`.
- **Shared type** — packval consumes the shared `EngineBinding` and the same table as the gate.

## Sharp Edges

1. **Hard dependency on SPEC-031's shared `EngineBinding` placement.** This spec consumes `EngineBinding`, the engine table/resolver, the `input_mode` gather, the `convert` pipe, the clean-stdout capture, and the SARIF parse from a shared home (`pkg/engine`). That package and those symbols do not exist on `main` yet — SPEC-031 creates them and owns the import-cycle-safe placement (`pluggable-pack-engines:REQ-013`). If SPEC-031 places the type elsewhere or names the resolver differently, the `consumes` contracts here must follow it. Implementing Seed 3 before Seed 2 is not possible; the plan must sequence SPEC-031 first.

2. **"Flagged" is the inverse of the old `Passed`.** Today `RunSemgrep`/`RunToolConfig` return `Passed=true` when the *command exited 0* (tool found nothing), and phase 3 treats positive-fixture "Passed" as good and negative-fixture "Passed" as a failure. Under engine dispatch the decision is "did the engine flag the fixture," derived from SARIF, not exit code. The semantic of `ExecutionResult.Passed` shifts for the findings arm; mixing the two interpretations (e.g., a findings engine that exits non-zero merely because it found something) is exactly the bug clean-stdout + SARIF parsing exists to prevent. Reviewers must confirm exit code is not consulted for findings pass/fail.

3. **Command error vs not-flagged collapse.** The most dangerous failure mode is silently treating an engine that failed to run (missing binary, bad convert, malformed SARIF) as "produced no findings," which would make a negative fixture spuriously fail and a positive fixture spuriously pass. REQ-004 mandates a distinct hard-error path; an implementation that folds command failure into the empty-results branch passes naive tests but is wrong.

4. **Precheck over-application.** The existing `semgrepFileContainsRuleID` precheck assumes a rule file carrying the pack rule ID. Re-keying it to the engine must skip it for `none` and `config-file` modes, not run it and fail. Applying the precheck uniformly would block every config-driven linter rule and every sandbox rule from ever fixture-testing.

5. **Carve-out leakage.** It is tempting to "unify everything" and route layer-3 validators and scaffold tests through the engine/SARIF path too. That is prohibited (REQ-006): those arms are exit-code edges with no located findings. Converging them would force a fake SARIF shape onto pass/fail signals that are not findings-shaped.

6. **Two mock surfaces during migration.** Removing `RunSemgrep`/`RunToolConfig` from `FixtureExecutor` and adding `RunEngine` is a breaking interface change. Existing phase-3 tests built on `MockExecutor.SemgrepFn`/`ToolConfigFn` must migrate to `RunEngineFn` in lockstep; a half-migrated executor that keeps both old and new methods invites callers to drift back onto the per-tool arms — the exact integration-gap drift the convergence exists to remove.

7. **`go mod tidy` pre-check is tool-config-era coupling.** The current `RunToolConfig` loop runs `goModTidyTempCopy` before tool execution. Under engine dispatch this pre-check is Go-specific environment setup, not engine dispatch; it must be preserved for Go config-file engines but must not be assumed for every engine (an ast-grep or semgrep rule does not need a tidied Go module). Misattaching it to all engines would break non-Go fixture runs.

## Review Questions

1. Does findings pass/fail derive **solely** from parsed SARIF results for the rule ID, with the engine's process exit code never consulted as the pass/fail signal for the findings arm?

2. Is a failure to run the engine binary, a `convert` failure, or a SARIF-parse failure reported as a **distinct hard `ValidationError`**, and is it impossible for any of those to be silently coerced into the "engine produced no findings" branch?

3. For `input_mode none` and `config-file` rules, is the rule-ID-match precheck **skipped** (not run-and-failed), and does a `rule-flags`/`rule-dir` rule whose pack ID is absent from its rule file still fail the precheck before fixtures run?

4. Do the layer-3 sandbox-validator arm (including multi-file `input_scope` aggregation) and the scaffold `test_command` arm execute with **unchanged exit-code semantics**, with no SARIF parsing applied to either?

5. Does `pkg/packval` import the **shared** `EngineBinding`/engine table (the SPEC-031 placement) rather than defining a packval-local binding or engine enum, and does the same resolver the gate path uses back fixture-time resolution?

6. For a given rule, does the fixture-time gather assemble the **same invocation** the gate-time gather assembles (same command, same per-mode flag layout), so a rule that passes fixtures behaves identically at the gate?

7. Is the negative-fixture engine-limitation fix hint emitted for **every** findings engine (not just semgrep), preserving the intent of the original hint?

8. Are the now-dead `RunSemgrep` / `RunToolConfig` methods and their `MockExecutor` fields fully **removed** (not left beside `RunEngine`), so callers cannot drift back onto per-tool dispatch?

## References

- **BUNDLE-010** — Pluggable Pack Engines (source bundle; this spec is Seed 3, covering REQ-012; DD-3 two-loci decomposition, DD-2 SARIF carve-out, DD-8 ladder, DD-9 input_mode).
- **SPEC-031** — Pluggable Engine Dispatch (Seed 2, locus A). Owns the shared `EngineBinding`, the engine table, `input_mode` gather, the `convert` pipe, clean stdout capture, and the SARIF parse this spec consumes. **Hard dependency.**
- **SPEC-014** — Pack Validation Pipeline. Defines the phase-3 fixture execution this spec generalizes (the FixtureExecutor interface, positive/negative contract, engine-limitation fix hint, layer-3 sandbox, scaffold validation).
- **SPEC-030** — Packs-only / native-standards removal (Seed 1). Collapses locus A's input to a single source (packs); sequenced before Seed 2.
- **ISSUE-003** — Data-driven toolchain registry. Precedent for the declared `{command, format}` + named-parser substrate the engine model converges onto.
