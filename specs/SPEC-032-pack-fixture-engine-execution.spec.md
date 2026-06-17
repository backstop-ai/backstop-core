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
    the `pack test` / phase-3 path) onto the same shared engine data model that the
    gate-time path (SPEC-031, locus A) uses. Today phase 3 dispatches fixtures
    through a bespoke FixtureExecutor whose arms are hardcoded one-per-tool:
    RunSemgrep (semgrep, `--config rule fixture`) and RunToolConfig (a one-arm
    `switch tool` on golangci-lint). This spec replaces those two findings arms —
    and folds the ToolConfigEntry (golangci-lint config-file) loop — into a single
    engine-dispatched arm driven by the rule's declared `engine`
    field and its EngineBinding ({command, input_mode, input_flag, convert?}),
    resolved by Registry map lookup against the same pkg/pack/engine data model
    SPEC-031 builds. Because SPEC-031 places only that data model in an importable
    leaf package (the gate's gather/convert/parseSarif execution lives in cmd/backstop
    and unexported pkg/check), this spec re-implements the gather/convert/SARIF-parse
    execution in pkg/packval and asserts invocation parity with the gate by test.
    A fixture run becomes:
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
      command as-is. The dispatch must be table-driven over `input_mode`. The same
      gather must serve config-file linter entries: a `tool_config` ToolConfigEntry
      (golangci-lint) resolves its `config-file` EngineBinding and routes through the
      same `config-file` gather and RunEngine dispatch as any findings rule, replacing
      the retired RunToolConfig arm; the bespoke `switch tool` arm is prohibited.
      Because the gate's execution path (cmd/backstop) is not importable from
      pkg/packval, the gather logic is re-implemented here; it must assemble, for any
      given rule and binding, the identical invocation the gate-time gather would —
      enforced by an explicit parity test asserting the assembled invocation equals the
      gate's for the same binding inputs (REQ-002 parity is asserted, not guaranteed by
      shared execution code).
    supports: pluggable-pack-engines:REQ-012
  - id: REQ-003
    text: >
      For findings engines, fixture execution must determine pass/fail from parsed
      SARIF, not from raw exit codes or raw tool JSON. The executor must capture the
      command's stdout cleanly and separately from stderr (replacing CombinedOutput,
      which merges stderr into stdout and corrupts SARIF), pipe stdout through the
      EngineBinding's pack-declared `convert` executable when present (`tool stdout →
      convert stdin → SARIF`, run in-process with no shell), and parse the resulting
      SARIF to decide whether the engine flagged the fixture. SARIF parsing is a
      packval-local parse (the gate's `parseSarif` lives unexported in pkg/check and is
      not importable here); it must apply the same SARIF results-for-rule-ID contract.
      A non-empty SARIF results set for the rule's ID means the engine flagged the
      fixture; an empty set means it did not.
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
      to the engine. The precheck is ENFORCED only for the `rule-flags` (semgrep) input
      mode, whose rule files are semgrep-shaped YAML carrying an `id` field the existing
      `semgrepFileContainsRuleID` parser already extracts: phase 3 must still verify the
      pack rule ID is present in the referenced rule file before running fixtures. The
      precheck is SKIPPED for every other input mode — `none` and `config-file` carry no
      per-rule-file ID, and `rule-dir` (ast-grep) rule files are not semgrep-shaped and
      have no defined ID-extraction here. ast-grep rule-ID prechecking is explicitly OUT
      OF SCOPE for this spec (rationale: ast-grep rule files use a different schema, and
      defining a per-engine ID extractor is engine-model surface owned by SPEC-031's
      EngineBinding, not this fixture-path spec); a `rule-dir` rule skips the precheck
      rather than spuriously failing. The precheck must never run-and-fail for a mode it
      does not cover.
    supports: pluggable-pack-engines:REQ-012
  - id: REQ-009
    text: >
      The shared EngineBinding type and engine table consumed here must be the same
      ones SPEC-031 defines and places to avoid an import cycle across pkg/check,
      pkg/packval, and cmd/backstop. pkg/packval must import that shared type rather
      than redefining an EngineBinding or an engine enum locally. No second,
      packval-local copy of the engine model may exist.
    supports: pluggable-pack-engines:REQ-012
  - id: REQ-010
    text: >
      The `go mod tidy` pre-check (`goModTidyTempCopy`), today run unconditionally
      before every ToolConfigEntry, must be conditioned on the Go config-file engine
      and run only for it. Under engine dispatch this pre-check is Go-specific
      environment setup, not engine dispatch: it must run only when the resolved engine
      is the Go `config-file` engine (golangci-lint) and must NOT run for any other
      engine (semgrep, ast-grep, or a non-Go config-file linter). Attaching the
      go-mod-tidy pre-check to a non-Go engine run is prohibited.
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
    text: A parity test asserts the packval gather assembles the identical invocation the gate-time gather assembles for the same binding inputs (asserted parity, not shared execution code)
    tests:
      - TestPackVal_EngineGather_ParityWithGateInvocation
  - id: CLM-036
    requirement: REQ-002
    text: A tool_config ToolConfigEntry (golangci-lint) routes through the config-file gather and RunEngine dispatch, not the retired RunToolConfig arm
    tests:
      - TestPackVal_EngineGather_ToolConfigEntryRoutesThroughRunEngine

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
    text: For the rule-flags (semgrep) input mode, the pack rule ID must be present in the referenced rule file or phase 3 fails before running fixtures
    tests:
      - TestPackVal_EnginePrecheck_RuleFlagsRuleIDEnforced
  - id: CLM-031
    requirement: REQ-008
    text: For the rule-flags (semgrep) input mode, a matching rule ID passes the precheck
    tests:
      - TestPackVal_EnginePrecheck_RuleFlagsRuleIDPasses
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
  - id: CLM-037
    requirement: REQ-008
    text: For the rule-dir (ast-grep) input mode, the rule-ID-match precheck is skipped (ast-grep precheck out of scope) rather than run against semgrep-shaped extraction and failing
    tests:
      - TestPackVal_EnginePrecheck_RuleDirSkipsPrecheck

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

  # --- REQ-010: go-mod-tidy pre-check conditioned on the Go config-file engine ---
  - id: CLM-038
    requirement: REQ-010
    text: The go-mod-tidy pre-check runs for the Go config-file engine (golangci-lint) run before its fixtures
    tests:
      - TestPackVal_EngineGoModTidy_RunsForGoConfigFileEngine
  - id: CLM-039
    requirement: REQ-010
    text: The go-mod-tidy pre-check does NOT run for a semgrep (rule-flags) engine run
    tests:
      - TestPackVal_EngineGoModTidy_SkippedForSemgrep
  - id: CLM-040
    requirement: REQ-010
    text: The go-mod-tidy pre-check does NOT run for an ast-grep (rule-dir) engine run
    tests:
      - TestPackVal_EngineGoModTidy_SkippedForAstGrep

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
          The ToolConfigEntry (config-file linter) loop also folds into RunEngine: a
          ToolConfigEntry resolves the `config-file` EngineBinding for its `tool` and
          dispatches through RunEngine like any other findings rule, replacing the bespoke
          RunToolConfig arm.
      - name: RunEngine
        kind: method
        signature: "RunEngine(packDir string, binding engine.EngineBinding, ruleID string, ruleFiles []string, fixturePath string) (ExecutionResult, error)"
        notes: >
          Gathers inputs per binding.InputMode (the same data-driven gather the gate's
          dispatchPackEngines performs, re-implemented in pkg/packval since the gate's
          execution lives in cmd/backstop and is not importable here), runs binding.Command
          capturing stdout cleanly, pipes through binding.Convert when present, parses the
          normalized SARIF via a packval-local parse, and sets ExecutionResult.Passed =
          engine flagged the fixture (Flagged), with a distinct error return for
          command/convert/parse failure.
      - name: DefaultExecutor
        kind: type
        signature: "type DefaultExecutor struct"
        notes: "Implements RunEngine using real OS commands and a packval-local SARIF parse (the gate's parseSarif is unexported in pkg/check)."
      - name: MockExecutor
        kind: type
        signature: "type MockExecutor struct"
        notes: "Adds RunEngineFn for phase-3 dispatch tests without real engine binaries."
      - name: ExecutionResult
        kind: type
        signature: "type ExecutionResult struct"
        notes: "Passed reflects whether the engine flagged the fixture; ExitCode/Output/Diagnostics retained for exit-code arms."
    consumes:
      - source: pkg/pack/engine
        name: EngineBinding
        kind: type
      - source: pkg/pack/engine
        name: InputMode
        kind: type
      - source: pkg/pack/engine
        name: ParseInputMode
        kind: function
  - file: pkg/packval/phase3.go
    provides:
      - name: RunFixtures
        kind: function
        signature: "func RunFixtures(pack *PackManifest, packDir string, executor FixtureExecutor) *PhaseResult"
        notes: >
          Findings rules now dispatch through executor.RunEngine via the rule's
          resolved EngineBinding; the per-tool RunSemgrep/RunToolConfig loops are
          replaced; the ToolConfigEntry/RunToolConfig loop folds into the same
          RunEngine dispatch via each entry's resolved `config-file` EngineBinding.
          Validator/scaffold/SDK arms and the negative-fixture engine-limitation fix
          hint are preserved.
    consumes:
      - source: pkg/pack/engine
        name: EngineBinding
        kind: type
      - source: pkg/pack/engine
        name: Registry
        kind: type
  - file: pkg/packval/manifest.go
    provides:
      - name: Rule
        kind: type
        signature: "type Rule struct"
        notes: >
          Gains the `engine` field (string) consumed by phase-3 dispatch; `layer`
          retires as the execution selector per SPEC-031's schema cutover.
      - name: ToolConfigEntry
        kind: type
        signature: "type ToolConfigEntry struct"
        notes: >
          Gains an `engine` field (string) so config-file linter entries (golangci-lint)
          resolve a `config-file` EngineBinding and route through RunEngine instead of the
          retired RunToolConfig arm. When `engine` is empty it defaults to the `config-file`
          binding keyed by the entry's existing `tool` field; ToolConfigEntry remains a
          separate top-level container from Content.Ruleset.Rules (the containers are not
          merged, mirroring SPEC-031's REQ-013 separation).
    consumes:
      - source: pkg/pack/engine
        name: EngineBinding
        kind: type
---

# SPEC-032: Pack Fixture Engine Execution

## Overview

BUNDLE-010 first-classes the execution engine of a pack rule and dispatches it through an explicit `engine → {command, convert?, input_mode, input_flag, requires[], forbids[]}` table instead of the implicit `layer==2 ⇒ semgrep` routing and the one-arm `RunToolConfig` switch. The bundle identifies **two convergence loci** (DD-3) that both hardcode tool dispatch independently and must both converge onto the shared engine model:

- **Locus A — gate-time engine dispatch** (`pkg/check`, `cmd/backstop`). Replaces the semgrep-only gate executor with group-by-engine dispatch. Owned by **SPEC-031** (Seed 2).
- **Locus B — validation-time fixture execution** (`pkg/packval`, the `pack test` / phase-3 path). The genuinely separate path that runs a rule against its positive/negative fixtures to prove the rule works. **This spec (Seed 3) owns locus B.**

This spec applies the same engine generalization to locus B. Today `pkg/packval` phase 3 runs findings fixtures through a `FixtureExecutor` whose findings arms are hardcoded per tool: `RunSemgrep` invokes `semgrep --config rule fixture`, and `RunToolConfig` is a one-arm `switch tool` on `golangci-lint`. This spec replaces those two arms with a single engine-dispatched arm (`RunEngine`) driven by the rule's declared `engine` field and the shared `EngineBinding` resolved by `Registry` map lookup from the same `pkg/pack/engine` data model SPEC-031 builds. The `ToolConfigEntry` (golangci-lint) loop folds into the same `RunEngine` dispatch via its `config-file` binding. The result: a pack author can fixture-test an ast-grep rule (or any declared engine's rule) against positive/negative fixtures with the **same invocation the gate-time path assembles** (asserted by a parity test), and adding a new engine never touches the fixture executor.

The convergence is deliberately scoped to **findings engines** (semgrep, ast-grep, lint). The bundle's DD-2/DD-8 carve-out is preserved: layer-3 sandbox validators and scaffold tests are **exit-code edges, not located findings**, so they keep their current `RunValidator`/`RunScaffoldTest` exit-code semantics and must not ride SARIF. The positive/negative fixture contract, the `PhaseResult`/`ValidationError` shape, the negative-fixture engine-limitation fix hint, and the rule-ID-match precheck are all preserved — only the per-tool dispatch is replaced.

This spec **depends on SPEC-031** for the shared `EngineBinding`/`InputMode`/`Registry` data model in the `pkg/pack/engine` leaf package. SPEC-031 places only that data model in an importable package; the gather/convert/SARIF-parse **execution** is not importable from `pkg/packval` (it lives in `cmd/backstop` and unexported `pkg/check`), so this spec re-implements that execution in `pkg/packval` and asserts parity with the gate. It is authored from the bundle independently and re-specifies none of locus A's gate-time dispatch.

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

The fixture gather step is table-driven over the binding's `input_mode`, with no per-engine Go branch. It is re-implemented in `pkg/packval` and asserted equal to the gate-time gather for the same binding inputs by an explicit parity test (REQ-002, CLM-009). The same gather serves `ToolConfigEntry` (config-file linter) entries.

| input_mode | Rule/config injection | input_flag use | Representative engine | Rule-ID precheck (REQ-008) |
|------------|----------------------|----------------|-----------------------|----------------------------|
| `config-file` | single optional pack-supplied config; tool runs its OWN rules | one flag, the config path | golangci-lint / eslint / tsc | Skipped (no per-rule-file ID) |
| `rule-flags` | each rule file → repeated flag | repeated, once per rule file | semgrep | Enforced (rule file carries IDs) |
| `rule-dir` | rule files collected into a dir | one flag, the dir path | ast-grep | Skipped (ast-grep precheck out of scope; rule files not semgrep-shaped) |
| `none` | no injection; the executable is the logic | unused | sandbox custom script | Skipped (no rule file) |

### Engine pass/fail (findings)

| Engine binding | stdout handling | SARIF source | Flagged decision |
|----------------|-----------------|--------------|------------------|
| SARIF-native (`{command}`) | captured clean, separate from stderr | stdout directly | non-empty SARIF results for rule ID ⇒ flagged |
| non-SARIF (`{command, convert}`) | captured clean, piped `stdout → convert stdin` | convert output | non-empty SARIF results for rule ID ⇒ flagged |
| command/convert/parse failure | n/a | n/a | hard fixture error (NOT "not flagged") |

## Implementation

### Scope boundary against SPEC-031

SPEC-031 (locus A) defines and places the shared **data model** — the `EngineBinding` type, the `InputMode` enum + `ParseInputMode`, the `Provision` descriptor, and the `Registry` (`engine name → EngineBinding`) — in a leaf package, `pkg/pack/engine`, importable by `pkg/check`, `pkg/packval`, and `cmd/backstop` without an import cycle (`pluggable-pack-engines:REQ-013`). This spec **consumes that data model** from `pkg/pack/engine` and resolves a rule's binding by `Registry` map lookup keyed on the rule's declared `engine` (there is no `ResolveBinding` function — lookup is the map plus a fail-loud miss).

What SPEC-031 does **not** place in an importable package is the **gather / convert / SARIF-parse execution**: the gate's gather-convert-parse lives in `cmd/backstop/dispatchPackEngines` (not importable from `pkg/packval`), and the gate's SARIF parser `parseSarif` is unexported in `pkg/check`. So no shared importable execution exists. This spec therefore **re-implements** the input-mode gather, the clean-stdout `convert` pipe, and a packval-local SARIF parse inside `pkg/packval`, and **asserts parity** with the gate's gather via an explicit parity test (CLM-009) rather than claiming a shared-code guarantee. This spec does not re-specify gate-time dispatch.

### Phase-3 findings dispatch (replaces RunSemgrep + RunToolConfig)

The current phase-3 loops over `pack.Content.Ruleset.Rules` (calling `RunSemgrep`) and `pack.ToolConfig` (calling `RunToolConfig`) collapse into a single findings dispatch. Both `Content.Ruleset.Rules` (findings rules) and `ToolConfig` entries (config-file linters, e.g. golangci-lint) feed the same dispatch; the two remain separate top-level containers (not merged), but both resolve an `EngineBinding` and run through `RunEngine`. A `ToolConfigEntry` resolves the `config-file` binding for its `tool`/`engine` and gathers via the `config-file` input mode.

1. **Resolve the binding.** For each findings rule (and each `ToolConfigEntry`), resolve its `EngineBinding` from the shared `Registry` (`pkg/pack/engine`) by `Registry` map lookup keyed on the declared `engine`. If the lookup misses, emit a hard, blocking `ValidationError` identifying the rule and the unresolved engine (REQ-007) — no silent skip, no semgrep fallback.
2. **Rule-ID precheck (engine-keyed).** ENFORCED only for the `rule-flags` (semgrep) input mode: verify the pack rule ID is present in the referenced rule file before running fixtures, retaining today's `semgrepFileContainsRuleID` check. SKIPPED for `none`, `config-file`, and `rule-dir` (ast-grep) — ast-grep ID prechecking is out of scope (REQ-008).
3. **Go-mod-tidy pre-check (conditioned).** Run the `goModTidyTempCopy` pre-check only when the resolved engine is the Go `config-file` engine (golangci-lint); do not run it for semgrep, ast-grep, or any non-Go engine (REQ-010).
4. **Gather inputs by input_mode.** Assemble the invocation per the input_mode gather matrix above (REQ-002). The gather is re-implemented in `pkg/packval` (the gate's gather in `cmd/backstop` is not importable) and is asserted equal to the gate-time gather by an explicit parity test (CLM-009).
5. **Run + normalize to SARIF.** Invoke `binding.Command` capturing stdout cleanly and separately from stderr (replacing `CombinedOutput`), pipe through `binding.Convert` when present (`tool stdout → convert stdin → SARIF`, in-process, no shell), and parse the SARIF via a packval-local parse (the gate's `parseSarif` is unexported). A command, convert, or parse failure is a hard fixture error distinct from a clean run (REQ-003, REQ-004).
6. **Apply the fixture contract.** `Flagged` = non-empty SARIF results for the rule ID. Positive fixtures pass iff NOT flagged; negative fixtures pass iff flagged (REQ-004). On a negative fixture that is not flagged, emit the existing engine-limitation fix hint on the `ValidationError`, for every engine (REQ-005).

### Non-findings arms (unchanged)

The layer-3 sandbox validator arm (`RunValidator` → `SandboxedRun`, including multi-file `input_scope` aggregation) and the scaffold `test_command` arm (`RunScaffoldTest`) keep their exit-code semantics verbatim. SARIF normalization is not applied to them (REQ-006). The SDK `provides` check is likewise untouched. The `go mod tidy` pre-check (`goModTidyTempCopy`) is preserved but is now **conditioned on the Go config-file engine** — it runs only for that engine, not unconditionally per `ToolConfigEntry`, and never for semgrep or ast-grep (REQ-010).

### FixtureExecutor surface change

`RunSemgrep` and `RunToolConfig` are removed from the `FixtureExecutor` interface and replaced by a single `RunEngine` method (see contracts). `RunValidator` and `RunScaffoldTest` remain. `MockExecutor` gains `RunEngineFn` so phase-3 dispatch is testable without real engine binaries, mirroring the existing mock pattern. `ExecutionResult.Passed` now reflects "engine flagged the fixture" for the findings arm and retains exit-code meaning for the non-findings arms.

### Manifest field

`Rule` gains the `engine` field consumed by dispatch. `ToolConfigEntry` gains an `engine` field too, so config-file linter entries (golangci-lint) resolve a `config-file` binding and route through `RunEngine`; when empty it defaults to the `config-file` binding keyed by the entry's existing `tool`. `ToolConfigEntry` stays a separate top-level container from `Content.Ruleset.Rules` (not merged). `layer` retires as the execution selector under SPEC-031's schema cutover; this spec reads `engine`. (The schema-validation re-key from layer→engine field-contracts is SPEC-031's; this spec only consumes the resolved `engine`.)

## Verification

Verification config is defined in frontmatter. Claims are defined in frontmatter. The test command targets `pkg/packval` with the `TestPackVal_Engine` prefix; unit-level because the convergence is a within-package dispatch change exercised through a `MockExecutor` (real engine binaries are not required to prove dispatch, gather, SARIF-decision, and contract behavior).

### Test strategy

- **Engine dispatch** — assert findings rules route through `RunEngine` with the resolved binding, and that `RunSemgrep`/`RunToolConfig` are gone from the executor surface.
- **input_mode gather matrix** — one test per mode (`config-file`, `rule-flags`, `rule-dir`, `none`) asserting the assembled invocation, a test asserting a `ToolConfigEntry` routes through the `config-file` gather + `RunEngine`, plus a parity test asserting the packval gather equals the gate-time gather for the same binding inputs.
- **SARIF decision** — flagged/not-flagged from parsed SARIF, clean stdout capture, and the convert-vs-native split, using a `MockExecutor` / canned SARIF.
- **Fixture contract** — the four positive/negative cells plus the command-error-is-hard-error cell.
- **Engine-limitation hint** — preserved for semgrep and a non-semgrep findings engine.
- **Non-findings arms** — validator (single + multi-file) and scaffold exit-code behavior unchanged; SARIF not applied.
- **Unresolved engine** — hard error, no silent skip, no semgrep fallback.
- **Precheck re-key** — enforced for the `rule-flags` (semgrep) mode, skipped for `none`, `config-file`, and `rule-dir` (ast-grep precheck out of scope).
- **Go-mod-tidy conditioning** — runs for the Go config-file engine, skipped for semgrep and ast-grep.
- **Shared type** — packval consumes the shared `EngineBinding`/`Registry` from `pkg/pack/engine` and the same registry the gate uses.

## Sharp Edges

1. **Hard dependency on SPEC-031's shared data model — but NOT its execution.** This spec consumes the `EngineBinding` type, the `InputMode` enum + `ParseInputMode`, and the `Registry` (map lookup, no `ResolveBinding` function) from `pkg/pack/engine`. That leaf package and those symbols do not exist on `main` yet — SPEC-031 creates them and owns the import-cycle-safe placement (`pluggable-pack-engines:REQ-013`). The trap: SPEC-031 places only the **data model** in that importable package; the gate's gather/convert/parse **execution** lives in `cmd/backstop/dispatchPackEngines` and unexported `pkg/check.parseSarif`, neither importable from `pkg/packval`. This spec must therefore re-implement that execution locally and prove parity by test — it must NOT assume a shared `RunEngine`-equivalent exists to import, and must NOT expand SPEC-031 to export one (that is sibling-spec scope). If SPEC-031 re-keys the package path or the `Registry`/`ParseInputMode` symbols, the `consumes` contracts here must follow. Implementing Seed 3 before Seed 2 is not possible; the plan must sequence SPEC-031 first.

2. **"Flagged" is the inverse of the old `Passed`.** Today `RunSemgrep`/`RunToolConfig` return `Passed=true` when the *command exited 0* (tool found nothing), and phase 3 treats positive-fixture "Passed" as good and negative-fixture "Passed" as a failure. Under engine dispatch the decision is "did the engine flag the fixture," derived from SARIF, not exit code. The semantic of `ExecutionResult.Passed` shifts for the findings arm; mixing the two interpretations (e.g., a findings engine that exits non-zero merely because it found something) is exactly the bug clean-stdout + SARIF parsing exists to prevent. Reviewers must confirm exit code is not consulted for findings pass/fail.

3. **Command error vs not-flagged collapse.** The most dangerous failure mode is silently treating an engine that failed to run (missing binary, bad convert, malformed SARIF) as "produced no findings," which would make a negative fixture spuriously fail and a positive fixture spuriously pass. REQ-004 mandates a distinct hard-error path; an implementation that folds command failure into the empty-results branch passes naive tests but is wrong.

4. **Precheck over-application — including ast-grep.** The existing `semgrepFileContainsRuleID` precheck assumes a *semgrep-shaped* rule file carrying the pack rule ID in a YAML `id` field. Re-keying it to the engine must ENFORCE it only for `rule-flags` (semgrep) and SKIP it for `none`, `config-file`, **and `rule-dir` (ast-grep)** — ast-grep rule files are not semgrep-shaped, so running `semgrepFileContainsRuleID` against them would always fail to find the ID and spuriously block every ast-grep rule. ast-grep ID prechecking is out of scope (REQ-008); a per-engine ID extractor is engine-model surface owned by SPEC-031, not this spec. Applying the precheck uniformly would block config-driven linter rules, sandbox rules, and ast-grep rules from ever fixture-testing.

5. **Carve-out leakage.** It is tempting to "unify everything" and route layer-3 validators and scaffold tests through the engine/SARIF path too. That is prohibited (REQ-006): those arms are exit-code edges with no located findings. Converging them would force a fake SARIF shape onto pass/fail signals that are not findings-shaped.

6. **Two mock surfaces during migration.** Removing `RunSemgrep`/`RunToolConfig` from `FixtureExecutor` and adding `RunEngine` is a breaking interface change. Existing phase-3 tests built on `MockExecutor.SemgrepFn`/`ToolConfigFn` must migrate to `RunEngineFn` in lockstep; a half-migrated executor that keeps both old and new methods invites callers to drift back onto the per-tool arms — the exact integration-gap drift the convergence exists to remove.

7. **`go mod tidy` pre-check is tool-config-era coupling.** The current `ToolConfig` loop runs `goModTidyTempCopy` **unconditionally before every `ToolConfigEntry`**. Under engine dispatch this pre-check is Go-specific environment setup, not engine dispatch; REQ-010 mandates conditioning it on the Go config-file engine — preserved for that engine, never run for semgrep or ast-grep (which do not need a tidied Go module), and never run for a non-Go config-file linter. The conditioning is testable: CLM-038/039/040 assert it runs for the Go config-file engine and not for semgrep or ast-grep. Misattaching it to all engines would break non-Go fixture runs.

## Review Questions

1. Does findings pass/fail derive **solely** from parsed SARIF results for the rule ID, with the engine's process exit code never consulted as the pass/fail signal for the findings arm?

2. Is a failure to run the engine binary, a `convert` failure, or a SARIF-parse failure reported as a **distinct hard `ValidationError`**, and is it impossible for any of those to be silently coerced into the "engine produced no findings" branch?

3. For `input_mode none` and `config-file` rules, is the rule-ID-match precheck **skipped** (not run-and-failed), and does a `rule-flags`/`rule-dir` rule whose pack ID is absent from its rule file still fail the precheck before fixtures run?

4. Do the layer-3 sandbox-validator arm (including multi-file `input_scope` aggregation) and the scaffold `test_command` arm execute with **unchanged exit-code semantics**, with no SARIF parsing applied to either?

5. Does `pkg/packval` import the **shared** `EngineBinding`/`Registry` from `pkg/pack/engine` (the SPEC-031 placement) and resolve via `Registry` map lookup rather than defining a packval-local binding or engine enum, and is it the same `Registry` the gate path uses?

6. For a given rule, does the fixture-time gather assemble the **same invocation** the gate-time gather assembles (same command, same per-mode flag layout), so a rule that passes fixtures behaves identically at the gate?

7. Is the negative-fixture engine-limitation fix hint emitted for **every** findings engine (not just semgrep), preserving the intent of the original hint?

8. Are the now-dead `RunSemgrep` / `RunToolConfig` methods and their `MockExecutor` fields fully **removed** (not left beside `RunEngine`), so callers cannot drift back onto per-tool dispatch?

9. Does a `tool_config` `ToolConfigEntry` (golangci-lint) resolve a `config-file` `EngineBinding` and dispatch through `RunEngine` via the `config-file` gather, with **no surviving `switch tool` arm**, while `ToolConfigEntry` stays a separate container from `Content.Ruleset.Rules`?

10. Is the `goModTidyTempCopy` pre-check **conditioned on the Go config-file engine** — running for it but provably not for semgrep or ast-grep runs — rather than unconditionally per `ToolConfigEntry`?

## References

- **BUNDLE-010** — Pluggable Pack Engines (source bundle; this spec is Seed 3, covering REQ-012; DD-3 two-loci decomposition, DD-2 SARIF carve-out, DD-8 ladder, DD-9 input_mode).
- **SPEC-031** — Pluggable Engine Dispatch (Seed 2, locus A). Owns the shared **data model** this spec consumes from `pkg/pack/engine`: the `EngineBinding` type, the `InputMode` enum + `ParseInputMode`, the `Provision` descriptor, and the `Registry` map. SPEC-031's gather/`convert`/SARIF-parse **execution** lives in `cmd/backstop` and unexported `pkg/check` and is NOT importable here; this spec re-implements that execution in `pkg/packval` and asserts parity. **Hard dependency (on the data model).**
- **SPEC-014** — Pack Validation Pipeline. Defines the phase-3 fixture execution this spec generalizes (the FixtureExecutor interface, positive/negative contract, engine-limitation fix hint, layer-3 sandbox, scaffold validation).
- **SPEC-030** — Packs-only / native-standards removal (Seed 1). Collapses locus A's input to a single source (packs); sequenced before Seed 2.
- **ISSUE-003** — Data-driven toolchain registry. Precedent for the declared `{command, format}` + named-parser substrate the engine model converges onto.
