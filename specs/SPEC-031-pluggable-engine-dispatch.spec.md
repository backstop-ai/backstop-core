---
title: "SPEC-031: Pluggable Engine Dispatch"
number: SPEC-031
created: "2026-06-16"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    First-class the execution engine as a declared `engine` field on a pack rule
    and dispatch gate-time enforcement through an explicit engine table
    (`EngineBinding {command, input_mode, input_flag, scope_kind, convert?,
    provision?}`) rather than the implicit `layer==2 ⇒ semgrep` routing. Retire
    the `layer` field; re-key its per-tier field requirements from layer→engine as
    each engine's field-contract. Replace the semgrep-only gate executor with
    group-by-engine dispatch that gathers inputs by declared `input_mode`, runs the
    engine command, pipes non-SARIF output through a sandboxed pack-declared
    `convert` executable, and parses the single output contract via `parseSarif`.
    Capture stdout cleanly (drop CombinedOutput). Wire ast-grep as the first new
    engine with a trivial proof rule end-to-end. Split engine provisioning:
    Layer-0 native toolchain is assumed-present (fail loud); backstop-introduced
    engines (semgrep, ast-grep) are auto-provisioned via a declared+pinned install
    on the `backstop.lock`/`VerifyLock` path, retiring `EnsureSemgrep`. Flag-day
    migrate `backstop-go-pack`'s 14 rules to `engine: semgrep`. This spec covers
    the gate-time locus (locus A) only; the fixture-time locus (`pkg/packval`) is
    SPEC-032, pillar-2 packs-only removal is SPEC-030, and the BUNDLE-009 contract
    seam is SPEC-033.
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/ ./pkg/check/ ./pkg/pack/ ./pkg/pack/engine/ -race -coverprofile=cover.out -v
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      A pack rule must declare its execution engine as a first-class `engine`
      string field on the Rule struct. The gate must dispatch enforcement on the
      declared engine through an explicit `EngineBinding` table, never the implicit
      `layer==2 ⇒ semgrep` routing. Dispatch groups a pack's rules by their
      declared engine and runs each engine once. Adding a new engine must be a
      declaration (an EngineBinding record plus, at most, a registered convert
      script), never a surgical edit to the gate's executor.
    supports: pluggable-pack-engines:REQ-001
    follows: STD-GO-001:GO-010

  - id: REQ-002
    text: >
      The `layer` field (1/2/3) must retire as the execution selector. `engine`
      becomes the single first-class key. A rule that carries `layer` without a
      non-empty `engine` after cutover is a broken declaration that is both loud
      and blocking: manifest parse/validation must return a ConfigError, never a
      silent default. A rule must not be required to carry both `layer` and
      `engine`; `layer` is removed from the Rule struct and its YAML key is no
      longer read.
    supports: pluggable-pack-engines:REQ-002
    follows: STD-GO-001:GO-010

  - id: REQ-003
    text: >
      Each engine carries a field-contract — `requires[]` and `forbids[]` lists of
      Rule field names — re-keyed faithfully from the retired per-layer requirements
      in the live `validateLayerFields` (no forbid is dropped). semgrep (ex-layer-2)
      requires `rule_path` and `standard` and forbids `category`, `input_scope`, and
      `validator`. The `standard` rule field is an authoring field independent of the
      standards compiler that SPEC-030 (Seed 1) removes; the migrated
      `backstop-go-pack` semgrep rules must each carry a non-empty `standard` so they
      satisfy this re-keyed semgrep field-contract after SPEC-030 lands (SPEC-030
      lands first per the ordering in References). ast-grep requires `rule_path` and forbids `category`,
      `input_scope`, and `validator` (it is rule-fed like semgrep, so it inherits
      semgrep's forbids minus the `standard` requirement, which ast-grep does not
      need). The config-file engine (ex-layer-1 native linter) forbids `rule_path`,
      `category`, `input_scope`, and `validator` and requires nothing. sandbox
      (ex-layer-3) requires `validator`, `input_scope`, and `category` and forbids
      `rule_path`; its `category` must be one of `presence`, `structural`, or
      `other` (with `other` requiring a non-empty `justification`), and its
      `input_scope` must be `single-file` or `multi-file`. Validation must verify a
      rule's populated fields satisfy its declared engine's field-contract,
      returning a ValidationError naming the offending field and engine when they do
      not. The layer-keyed `validateLayer` and `validateLayerFields` are replaced by
      engine-keyed equivalents that preserve every existing per-layer check.
    supports: pluggable-pack-engines:REQ-003
    follows: STD-GO-001:GO-010

  - id: REQ-004
    text: >
      Engine-fit validation must only verify field-consistency with the author's
      declared engine. It must not inspect rule content, recommend an engine, or
      reclassify a rule onto a different engine. The author asserts the engine;
      backstop enforces field-fit against that assertion and never questions or
      overrides it.
    supports: pluggable-pack-engines:REQ-004

  - id: REQ-005
    text: >
      Engines must be open and declared — never enumerated as a fixed Go switch in
      the gate executor. The finite thing backstop owns is output formats, not
      engines: the sole owned output parser is `parseSarif`, resolved through
      ISSUE-003's `formatParsers` registry and `lookupParser` fail-loud contract.
      Adding eslint/ruff/clippy/clj-kondo must be expressible as an EngineBinding
      declaration with no new Go in the dispatch path.
    supports: pluggable-pack-engines:REQ-005

  - id: REQ-006
    text: >
      The output contract for findings engines must be strict SARIF. The
      EngineBinding must carry no `format` selector. A SARIF-native engine declares
      `{command}` with an empty `convert`; a non-SARIF engine declares
      `{command, convert}`. The gate always parses an engine's normalized output
      via `parseSarif`, which fail-louds on non-SARIF input. Raw per-tool JSON must
      not be parsed in the dispatch path.
    supports: pluggable-pack-engines:REQ-006

  - id: REQ-007
    text: >
      When an EngineBinding declares a non-empty `convert`, the gate must run a
      two-process pipe in Go (no shell): engine stdout is fed to the `convert`
      executable's stdin, and the convert executable's stdout is the SARIF passed
      to `parseSarif`. The convert executable must be resolved relative to the pack
      directory and run inside the same sandbox trust model used by sandbox
      validators — no new trust model — but via a clean-stdout capture (per REQ-009),
      not `SandboxedRun`'s `CombinedOutput()`, so a converter's stderr cannot
      interleave into the SARIF bytes. When `convert` is empty, engine stdout goes
      directly to `parseSarif` with no pipe.
    supports: pluggable-pack-engines:REQ-007
    follows: STD-GO-001:GO-010

  - id: REQ-008
    text: >
      Backstop must build and host no converters in this dispatch path. SARIF-native
      engines and engines whose converter is a pack-declared script are the only two
      shapes. ast-grep, the lone genuine gap, ships its own stdin→SARIF converter
      script inside its pack directory referenced by the `convert` field; backstop
      embeds no transform engine. The bundle's companion obligation (DD-7) to point
      pack authors at existing per-known-tool converters via a documented pack.yml
      reference table is a pure docs deliverable and is deferred — it is not part of
      this spec's executable scope and is tracked as a documentation task to be
      assigned separately. This omission is intentional, not a coverage gap.
    supports: pluggable-pack-engines:REQ-008

  - id: REQ-009
    text: >
      Every command capture on the dispatch SARIF path must capture stdout cleanly
      and separately from stderr, replacing `CombinedOutput()` which merges stderr
      into stdout and corrupts SARIF parsing. This covers both capture points: (a)
      the engine command runner in `pkg/check/runner.go` gains a stdout-only method
      (`RunStdout`) returning stdout bytes uncontaminated by stderr; and (b) the
      sandboxed `convert` execution — whose stdout is the FINAL SARIF fed to
      `parseSarif` — must likewise capture stdout cleanly, not via
      `SandboxedRun`'s `CombinedOutput()`. A convert-capable sandbox capture
      (a clean-stdout variant of `SandboxedRun`, e.g. `SandboxedRunStdout`, used by
      the convert step) must return only the convert executable's stdout so that a
      converter writing a banner/warning to stderr still yields parseable SARIF on
      stdout. The existing `CombinedOutput`-based `SandboxedRun` is left in place for
      the non-SARIF sandbox-validator path (REQ-014), whose message body is allowed
      to include stderr; only the convert capture on the SARIF path switches to clean
      stdout.
    supports: pluggable-pack-engines:REQ-009
    follows: STD-GO-001:GO-010

  - id: REQ-010
    text: >
      ast-grep must be wired as the first new engine end-to-end through the gate.
      A registered ast-grep EngineBinding (input_mode rule-dir, a stdin→SARIF
      convert script, declared provisioning) plus a trivial proof rule in a test
      pack must produce a normalized violation through the full path: declaration →
      group-by-engine → gather rule-dir → run ast-grep → convert → parseSarif →
      namespaced violation in gate output.
    supports: pluggable-pack-engines:REQ-010

  - id: REQ-011
    text: >
      Gate-time engine dispatch must replace the semgrep-only `semgrepExecutor`
      feeder path (`mergePackRules` → `ExtraSemgrepConfigs`) with "group rules by
      declared engine, run each engine, normalize to SARIF," fed from the pack
      source. Pack rules of different engines installed together must each be
      dispatched to their own engine; ast-grep rules must not be fed into a semgrep
      invocation, and semgrep rules must not be fed into ast-grep.
    supports: pluggable-pack-engines:REQ-011
    follows: STD-GO-001:GO-010

  - id: REQ-013
    text: >
      The shared `EngineBinding` type must live in a package importable by
      `pkg/check`, `pkg/packval`, and `cmd/backstop` without an import cycle (a new
      `pkg/pack/engine` leaf package, importing none of those three). The
      EngineBinding shape is `{command, input_mode, input_flag, scope_kind,
      convert?, provision?}` with `format` fixed to sarif and absent from the
      struct. `scope_kind` is the engine package's own int type that *mirrors but
      does not import* `pkg/check.ScopeKind` — reusing the `pkg/check` type would
      reintroduce the import cycle this leaf-package placement exists to prevent, so
      the planner must declare a parallel type, not alias the shared one. The pack-side declaration (`pack.yml` → EngineBinding) is what this
      spec emits and consumes; the native toolchain container (per-stack
      `ToolchainEntry` registry) is a separate container that must not be merged
      into the EngineBinding registry. Converting the native `ToolchainEntry` into
      an EngineBinding is out of scope here (native code-check is owned elsewhere);
      this requirement asserts only the shared leaf-package placement, the absence
      of an import cycle, the EngineBinding field shape, and the do-not-merge
      boundary between the two containers.
    supports: pluggable-pack-engines:REQ-013
    follows: STD-GO-001:GO-010

  - id: REQ-014
    text: >
      Build and test passes plus the sandbox engine (the `input_mode: none` shape)
      are non-SARIF edges (exit-code / pass-fail, not located findings) and must not
      ride `parseSarif`. Within `dispatchPackEngines`, an engine whose binding output
      shape is exit-code/pass-fail — i.e. the sandbox/`none` engine — is dispatched on
      a branch that interprets the executable's exit code: a non-zero exit is a
      violation whose message is the captured output, and the path never enters
      `parseSarif`. This branch is the relocation of the existing gate-time layer-3
      sandbox path (cmd/backstop/pack_gate.go `runPackValidators`, currently keyed on
      `rule.Layer != 3`): because REQ-002 removes `rule.Layer`, that branch is re-keyed
      from `rule.Layer == 3` to `engine == sandbox` (input_mode `none`) and folded into
      the engine dispatch as the exit-code terminal step; its `input_scope`
      single-file/multi-file fan-out and `SandboxedRun` (CombinedOutput) capture are
      preserved unchanged. ISSUE-003's `ToolchainEntry.Format` survives for build/test
      (`go-build`, `go-test`, `regex-lines` retained). Only findings engines on the
      dispatch path parse findings, and they parse them exclusively via `parseSarif`;
      the per-tool lint JSON parsers `golangci-json` and `eslint-json` are not
      introduced into and never reachable from the pack engine dispatch path —
      pack-rule findings converge to SARIF only. Note: those two parsers live today
      only in the native toolchain registry (pkg/check/registry.go, the ISSUE-003
      TS/code-check path), which this spec disclaims as out of scope; retiring them
      from that native path is owned by SPEC-030, not this spec. This requirement is
      therefore scoped to asserting their absence from pack engine dispatch, not to
      removing them from pkg/check.
    supports: pluggable-pack-engines:REQ-014

  - id: REQ-015
    text: >
      Migration of the one existing `layer: 2` pack must be a single sibling edit:
      `backstop-go-pack`'s 14 rules bump to `engine: semgrep`. There must be no
      silent grandfather (`layer:2 → engine:semgrep`), no deprecation window, and
      no alias machinery. Core's reader (this spec) and the pack repo flip to
      `engine` together; a `layer: 2`-only rule reaching the migrated reader is a
      blocking config error per REQ-002.
    supports: pluggable-pack-engines:REQ-015

  - id: REQ-018
    text: >
      The engine model must be engine-shape-agnostic with the config-driven
      (Layer-0 native linter) shape as a first-class, supported invocation, not a
      deferred frontier. An EngineBinding with `input_mode: config-file` runs the
      tool's own built-in rules tuned by an optional pack-supplied config file; an
      EngineBinding with a rule-fed `input_mode` (`rule-flags`/`rule-dir`) supplies
      pack rule files. Adding a native config-driven linter must be an EngineBinding
      declaration with no new Go in the dispatch path.
    supports: pluggable-pack-engines:REQ-018
    follows: STD-GO-001:GO-010

  - id: REQ-019
    text: >
      Engine provisioning must split by tool ownership. Layer-0 native engines
      (`provision` empty) are assumed-present: a missing binary fails loud with a
      ConfigError naming the engine, and backstop must not attempt to install it.
      Backstop-introduced engines (semgrep, ast-grep) declare a pinned `provision`
      record and are auto-provisioned and verified through the existing
      `backstop.lock` / `VerifyLock` path — data-driven, with no per-engine Go.
      `EnsureSemgrep`'s bespoke install logic is retired into this declared
      mechanism.
    supports: pluggable-pack-engines:REQ-019
    follows: STD-GO-001:GO-010

  - id: REQ-020
    text: >
      The EngineBinding must declare input injection via a structured `input_mode`
      enum plus an `input_flag` string. The enum has exactly four values:
      `config-file` (a single optional pack-supplied config; the tool runs its own
      built-in rules), `rule-flags` (each rule file becomes a repeated `input_flag`
      occurrence, e.g. `--config X`), `rule-dir` (rule files collected into one
      directory passed once via `input_flag`), and `none` (no injection; the
      executable is the logic). An unrecognized `input_mode` value is a blocking
      config error. Gathering and shaping inputs from the declared mode must be
      data-driven with no per-engine Go.
    supports: pluggable-pack-engines:REQ-020
    follows: STD-GO-001:GO-010

  - id: REQ-021
    text: >
      Packs must follow an engine-organized layout convention: one directory per
      engine holding that engine's rules/config, with the directory corresponding
      to the engine's `input_mode` (`semgrep/` → rule-flags, `ast-grep/` →
      rule-dir, `linter/` → config-file, `scripts/` → none). The gate must resolve
      an engine's inputs relative to the pack directory using the rule's declared
      paths; a rule whose declared input path is missing on disk is a blocking
      broken-pack error naming the pack and the missing path.
    supports: pluggable-pack-engines:REQ-021
    follows: STD-GO-001:GO-010

claims:
  # REQ-001 — first-class engine + table dispatch
  - id: CLM-001
    requirement: REQ-001
    text: Rule struct exposes a first-class engine string field parsed from pack.yml
    tests:
      - TestManifest_EngineFieldParsed
  - id: CLM-002
    requirement: REQ-001
    text: Gate groups a pack's rules by declared engine and runs each engine once via the EngineBinding table
    tests:
      - TestGateDispatch_GroupsRulesByEngine
  - id: CLM-003
    requirement: REQ-001
    text: A newly registered EngineBinding dispatches without editing the gate executor switch
    tests:
      - TestGateDispatch_NewEngineNeedsNoExecutorEdit

  # REQ-002 — layer retires, engine required, loud+blocking
  - id: CLM-004
    requirement: REQ-002
    text: A rule with engine set and no layer parses and validates cleanly
    tests:
      - TestManifest_EngineSetNoLayerPasses
  - id: CLM-005
    requirement: REQ-002
    text: A rule with layer but no engine is a blocking ConfigError after cutover
    tests:
      - TestManifest_LayerWithoutEngineFailsLoud
  - id: CLM-006
    requirement: REQ-002
    text: The Rule struct no longer reads a layer YAML key into an execution selector
    tests:
      - TestManifest_LayerKeyNotReadAsSelector

  # REQ-003 — engine field-contract matrix (engine x fit)
  - id: CLM-007
    requirement: REQ-003
    text: semgrep rule with rule_path and standard and no input_scope passes the field-contract
    tests:
      - TestEngineFit_SemgrepValidFields
  - id: CLM-008
    requirement: REQ-003
    text: semgrep rule missing rule_path fails the field-contract
    tests:
      - TestEngineFit_SemgrepMissingRulePathFails
  - id: CLM-009
    requirement: REQ-003
    text: semgrep rule that defines input_scope fails the field-contract (forbidden)
    tests:
      - TestEngineFit_SemgrepForbidsInputScope
  - id: CLM-051
    requirement: REQ-003
    text: semgrep rule that defines category fails the field-contract (forbidden)
    tests:
      - TestEngineFit_SemgrepForbidsCategory
  - id: CLM-052
    requirement: REQ-003
    text: semgrep rule that defines validator fails the field-contract (forbidden)
    tests:
      - TestEngineFit_SemgrepForbidsValidator
  - id: CLM-010
    requirement: REQ-003
    text: ast-grep rule with rule_path and no forbidden fields passes the field-contract
    tests:
      - TestEngineFit_AstGrepValidFields
  - id: CLM-011
    requirement: REQ-003
    text: ast-grep rule missing rule_path fails the field-contract
    tests:
      - TestEngineFit_AstGrepMissingRulePathFails
  - id: CLM-053
    requirement: REQ-003
    text: ast-grep rule that defines input_scope fails the field-contract (forbidden)
    tests:
      - TestEngineFit_AstGrepForbidsInputScope
  - id: CLM-054
    requirement: REQ-003
    text: ast-grep rule that defines category fails the field-contract (forbidden)
    tests:
      - TestEngineFit_AstGrepForbidsCategory
  - id: CLM-055
    requirement: REQ-003
    text: ast-grep rule that defines validator fails the field-contract (forbidden)
    tests:
      - TestEngineFit_AstGrepForbidsValidator
  - id: CLM-012
    requirement: REQ-003
    text: sandbox rule with validator, input_scope, and category passes the field-contract
    tests:
      - TestEngineFit_SandboxValidFields
  - id: CLM-013
    requirement: REQ-003
    text: sandbox rule missing input_scope fails the field-contract
    tests:
      - TestEngineFit_SandboxMissingInputScopeFails
  - id: CLM-014
    requirement: REQ-003
    text: sandbox rule that defines rule_path fails the field-contract (forbidden)
    tests:
      - TestEngineFit_SandboxForbidsRulePath
  - id: CLM-056
    requirement: REQ-003
    text: sandbox rule missing validator fails the field-contract
    tests:
      - TestEngineFit_SandboxMissingValidatorFails
  - id: CLM-057
    requirement: REQ-003
    text: sandbox rule missing category fails the field-contract
    tests:
      - TestEngineFit_SandboxMissingCategoryFails
  - id: CLM-058
    requirement: REQ-003
    text: sandbox rule with an out-of-enum category (not presence/structural/other) fails the field-contract
    tests:
      - TestEngineFit_SandboxCategoryEnumEnforced
  - id: CLM-059
    requirement: REQ-003
    text: sandbox rule with category other and empty justification fails the field-contract
    tests:
      - TestEngineFit_SandboxOtherCategoryRequiresJustification
  - id: CLM-060
    requirement: REQ-003
    text: sandbox rule with an out-of-enum input_scope (not single-file/multi-file) fails the field-contract
    tests:
      - TestEngineFit_SandboxInputScopeEnumEnforced
  - id: CLM-015
    requirement: REQ-003
    text: config-file engine rule that defines rule_path fails the field-contract (forbidden)
    tests:
      - TestEngineFit_ConfigFileForbidsRulePath
  - id: CLM-061
    requirement: REQ-003
    text: config-file engine rule that defines category fails the field-contract (forbidden)
    tests:
      - TestEngineFit_ConfigFileForbidsCategory
  - id: CLM-062
    requirement: REQ-003
    text: config-file engine rule that defines input_scope fails the field-contract (forbidden)
    tests:
      - TestEngineFit_ConfigFileForbidsInputScope
  - id: CLM-063
    requirement: REQ-003
    text: config-file engine rule that defines validator fails the field-contract (forbidden)
    tests:
      - TestEngineFit_ConfigFileForbidsValidator
  - id: CLM-016
    requirement: REQ-003
    text: validateLayer and validateLayerFields are replaced by engine-keyed validation
    tests:
      - TestManifest_NoLayerKeyedValidationRemains

  # REQ-004 — verify not guide
  - id: CLM-017
    requirement: REQ-004
    text: Engine-fit validation accepts a declared engine without inspecting rule content
    tests:
      - TestEngineFit_VerifiesDoesNotGuide
  - id: CLM-018
    requirement: REQ-004
    text: Engine-fit validation never reclassifies a rule onto a different engine
    tests:
      - TestEngineFit_NeverReclassifies

  # REQ-005 — open engines, parseSarif sole parser
  - id: CLM-019
    requirement: REQ-005
    text: Engine dispatch resolves the SARIF parser via lookupParser and owns no engine enumeration
    tests:
      - TestGateDispatch_ParserIsSarifViaLookup
  - id: CLM-020
    requirement: REQ-005
    text: A declared engine name unknown to the binding table is a fail-loud config error, not a silent skip
    tests:
      - TestGateDispatch_UnknownEngineFailsLoud

  # REQ-006 — strict SARIF, no format selector
  - id: CLM-021
    requirement: REQ-006
    text: EngineBinding has no format field and findings output is always parsed as SARIF
    tests:
      - TestEngineBinding_NoFormatSelector
  - id: CLM-022
    requirement: REQ-006
    text: SARIF-native engine output parses directly with empty convert
    tests:
      - TestGateDispatch_SarifNativeNoConvert
  - id: CLM-023
    requirement: REQ-006
    text: Non-SARIF output reaching parseSarif without conversion fails loud
    tests:
      - TestGateDispatch_NonSarifWithoutConvertFails

  # REQ-007 — sandboxed convert pipe
  - id: CLM-024
    requirement: REQ-007
    text: Engine stdout is piped to the convert executable stdin and its stdout becomes the SARIF input
    tests:
      - TestGateDispatch_ConvertPipeProducesSarif
  - id: CLM-025
    requirement: REQ-007
    text: The convert executable is resolved relative to the pack dir and run via SandboxedRun
    tests:
      - TestGateDispatch_ConvertRunsSandboxed
  - id: CLM-026
    requirement: REQ-007
    text: An empty convert sends engine stdout directly to parseSarif with no pipe
    tests:
      - TestGateDispatch_EmptyConvertNoPipe

  # REQ-008 — backstop hosts no converters
  - id: CLM-027
    requirement: REQ-008
    text: The ast-grep pack supplies its own stdin-to-SARIF convert script referenced by the convert field
    tests:
      - TestAstGrepPack_ShipsOwnConverter

  # REQ-009 — clean stdout capture
  - id: CLM-028
    requirement: REQ-009
    text: The dispatch runner returns stdout uncontaminated by stderr
    tests:
      - TestRunner_StdoutSeparateFromStderr
  - id: CLM-029
    requirement: REQ-009
    text: An engine writing a banner to stderr still yields parseable SARIF on stdout
    tests:
      - TestGateDispatch_StderrBannerDoesNotCorruptSarif
  - id: CLM-064
    requirement: REQ-009
    text: A convert script writing a banner to stderr still yields parseable SARIF on its stdout via the clean-stdout sandbox capture
    tests:
      - TestGateDispatch_ConvertStderrBannerDoesNotCorruptSarif
  - id: CLM-065
    requirement: REQ-009
    text: The convert step captures stdout via the clean-stdout sandbox variant, never SandboxedRun's CombinedOutput
    tests:
      - TestGateDispatch_ConvertUsesCleanStdoutSandbox

  # REQ-010 — ast-grep end-to-end
  - id: CLM-030
    requirement: REQ-010
    text: A trivial ast-grep proof rule produces a namespaced violation through the full dispatch path
    tests:
      - TestGateDispatch_AstGrepProofRuleEndToEnd

  # REQ-011 — replace semgrep-only feeder
  - id: CLM-031
    requirement: REQ-011
    text: Gate dispatch replaces mergePackRules/ExtraSemgrepConfigs with group-by-engine execution
    tests:
      - TestGateDispatch_ReplacesSemgrepOnlyFeeder
  - id: CLM-032
    requirement: REQ-011
    text: Mixed semgrep and ast-grep rules each dispatch to their own engine, never cross-fed
    tests:
      - TestGateDispatch_MixedEnginesNotCrossFed

  # REQ-013 — shared EngineBinding placement
  - id: CLM-033
    requirement: REQ-013
    text: EngineBinding lives in pkg/pack/engine importable by check, packval, and cmd/backstop without an import cycle
    tests:
      - TestEngineBinding_NoImportCycle
  - id: CLM-034
    requirement: REQ-013
    text: EngineBinding shape carries command, input_mode, input_flag, scope_kind, optional convert and provision, and no format field
    tests:
      - TestEngineBinding_Shape

  # REQ-014 — non-SARIF carve-out
  - id: CLM-035
    requirement: REQ-014
    text: Build and test passes retain their ToolchainEntry.Format and do not route through parseSarif
    tests:
      - TestEngineDispatch_BuildTestKeepFormatParsers
  - id: CLM-036
    requirement: REQ-014
    text: Pack engine dispatch resolves only the SARIF parser and never references golangci-json or eslint-json
    tests:
      - TestEngineDispatch_PackFindingsParseSarifOnly
  - id: CLM-066
    requirement: REQ-014
    text: A sandbox/none engine producing a non-zero exit yields a violation via exit-code semantics without entering parseSarif
    tests:
      - TestEngineDispatch_SandboxNoneExitCodeViolationSkipsSarif
  - id: CLM-067
    requirement: REQ-014
    text: The retired layer-3 sandbox branch is re-keyed to engine sandbox (input_mode none) with no remaining rule.Layer reference
    tests:
      - TestEngineDispatch_SandboxBranchReKeyedFromLayer3

  # REQ-015 — flag-day migration
  - id: CLM-037
    requirement: REQ-015
    text: backstop-go-pack's rules carry engine semgrep and each retain a non-empty standard, satisfying the semgrep field-contract under the migrated reader
    tests:
      - TestMigration_GoPackEngineSemgrep
      - TestMigration_GoPackRulesRetainStandard
  - id: CLM-038
    requirement: REQ-015
    text: A legacy layer-2-only rule reaching the migrated reader fails loud with no grandfather alias
    tests:
      - TestMigration_NoSilentGrandfather

  # REQ-018 — config-file shape first-class
  - id: CLM-039
    requirement: REQ-018
    text: A config-file engine runs the tool's own rules tuned by an optional pack-supplied config
    tests:
      - TestGateDispatch_ConfigFileEngineRunsOwnRules
  - id: CLM-040
    requirement: REQ-018
    text: Adding a config-driven linter is an EngineBinding declaration with no new dispatch Go
    tests:
      - TestGateDispatch_ConfigFileEngineNeedsNoGo

  # REQ-019 — split provisioning
  - id: CLM-041
    requirement: REQ-019
    text: A Layer-0 engine with empty provision fails loud when its binary is absent and is never auto-installed
    tests:
      - TestProvision_NativeAssumedPresentFailsLoud
  - id: CLM-042
    requirement: REQ-019
    text: A backstop-introduced engine with a pinned provision record is provisioned and verified via the lock path
    tests:
      - TestProvision_IntroducedEngineAutoProvisioned
  - id: CLM-043
    requirement: REQ-019
    text: EnsureSemgrep's bespoke install logic is retired into the declared provision mechanism
    tests:
      - TestProvision_EnsureSemgrepRetired

  # REQ-020 — input_mode enum matrix (4 values + invalid)
  - id: CLM-044
    requirement: REQ-020
    text: input_mode config-file passes a single optional pack-supplied config and runs the tool's own rules
    tests:
      - TestInputMode_ConfigFile
  - id: CLM-045
    requirement: REQ-020
    text: input_mode rule-flags emits one input_flag occurrence per rule file
    tests:
      - TestInputMode_RuleFlags
  - id: CLM-046
    requirement: REQ-020
    text: input_mode rule-dir collects rule files into one directory passed once via input_flag
    tests:
      - TestInputMode_RuleDir
  - id: CLM-047
    requirement: REQ-020
    text: input_mode none injects no rules or config and runs the executable as the logic
    tests:
      - TestInputMode_None
  - id: CLM-048
    requirement: REQ-020
    text: An unrecognized input_mode value is a blocking config error
    tests:
      - TestInputMode_UnknownValueFailsLoud

  # REQ-021 — engine-organized layout
  - id: CLM-049
    requirement: REQ-021
    text: Engine inputs are resolved relative to the per-engine pack directory matching the input_mode
    tests:
      - TestPackLayout_EngineDirResolvesInputs
  - id: CLM-050
    requirement: REQ-021
    text: A rule whose declared input path is missing on disk is a blocking broken-pack error naming the pack and path
    tests:
      - TestPackLayout_MissingInputPathFailsLoud

contracts:
  - file: pkg/pack/engine/binding.go
    provides:
      - name: EngineBinding
        kind: type
        signature: "type EngineBinding struct"
        notes: "Fields: Command string, InputMode InputMode, InputFlag string, ScopeKind an engine-package int type mirroring (NOT importing) pkg/check.ScopeKind — importing it would reintroduce the cycle this leaf package avoids, Convert string (optional), Provision *Provision (optional). No Format field — output is always SARIF."
      - name: InputMode
        kind: type
        signature: "type InputMode string"
        notes: "Enum: config-file, rule-flags, rule-dir, none. ParseInputMode fail-louds on unknown."
      - name: Provision
        kind: type
        signature: "type Provision struct"
        notes: "Pinned install descriptor for backstop-introduced engines; empty for assumed-present Layer-0 engines."
      - name: Registry
        kind: type
        signature: "type Registry map[string]EngineBinding"
        notes: "engine name → binding. Built-in semgrep, ast-grep, sandbox, plus config-file linters; packs may contribute. Lookup fail-louds on unknown engine."
      - name: ParseInputMode
        kind: function
        signature: "func ParseInputMode(s string) (InputMode, error)"
    consumes: []

  - file: pkg/pack/manifest.go
    provides:
      - name: Rule
        kind: type
        signature: "type Rule struct"
        notes: "Adds Engine string field; removes Layer int field and its yaml key."
    consumes:
      - source: pkg/pack/engine/binding.go
        name: Registry
        kind: type

  - file: pkg/pack/validate_manifest.go
    provides:
      - name: validateEngineFields
        kind: function
        signature: "func validateEngineFields(m *Manifest) []ValidationError"
        notes: "Replaces validateLayerFields; verifies each rule's fields satisfy its engine's requires/forbids field-contract."
    consumes:
      - source: pkg/pack/engine/binding.go
        name: EngineBinding
        kind: type

  - file: cmd/backstop/pack_gate.go
    provides:
      - name: dispatchPackEngines
        kind: function
        signature: "func dispatchPackEngines(packs []*pack.Manifest, packDir, projectRoot string, runner check.CommandRunner) ([]gate.Violation, error)"
        notes: "Replaces mergePackRules feeder AND folds in runPackValidators' layer-3 sandbox branch (re-keyed to engine==sandbox): groups rules by engine, gathers inputs by input_mode, runs each engine; findings engines pipe through convert when declared and parse via parseSarif; the sandbox/none engine takes the exit-code terminal branch (non-zero exit ⇒ violation, no parseSarif). Namespaces violations."
    consumes:
      - source: pkg/pack/engine/binding.go
        name: Registry
        kind: type
      - source: pkg/check/parsers.go
        name: lookupParser
        kind: function
      - source: pkg/packval/sandbox.go
        name: SandboxedRunStdout
        kind: function
      - source: pkg/packval/sandbox.go
        name: SandboxedRun
        kind: function
      - source: pkg/pack/distribution/verify.go
        name: VerifyLock
        kind: function

  - file: pkg/check/runner.go
    provides:
      - name: CommandRunner
        kind: interface
        signature: "type CommandRunner interface { RunStdout(ctx context.Context, name string, args ...string) ([]byte, error) }"
        notes: "Adds a stdout-only capture method; ExecCommandRunner implements it via cmd.Output() / explicit stdout buffer rather than CombinedOutput()."
    consumes: []

  - file: pkg/packval/sandbox.go
    provides:
      - name: SandboxedRunStdout
        kind: function
        signature: "func SandboxedRunStdout(cmd string, args []string, packDir string) (stdout []byte, err error)"
        notes: "Clean-stdout variant of SandboxedRun for the convert step (REQ-007/REQ-009): same sandbox trust model, but captures only stdout (explicit stdout buffer, not CombinedOutput) so a converter's stderr cannot interleave into the SARIF bytes. SandboxedRun (CombinedOutput) is retained unchanged for the sandbox-validator exit-code path (REQ-014)."
    consumes: []
---

# SPEC-031: Pluggable Engine Dispatch

## Overview

This spec is the core engine machinery of BUNDLE-010 pillar 1 (Spec Seed 2). Today the gate decides which tool runs a pack rule implicitly: `mergePackRules` (cmd/backstop/pack_gate.go) filters `rule.Layer != 2` and dumps every layer-2 rule file into one `ExtraSemgrepConfigs` list consumed by the single `semgrepExecutor` (pkg/check/check.go). "Layer 2" literally means semgrep, and the merge assumes one tool consuming many `--config` files in one invocation. Adding ast-grep — which scans a rule directory, not a `--config` list, and emits JSON, not SARIF — is impossible without rewriting that executor.

**Without this spec:** a rule's engine is encoded by an integer (`layer`) and discovered by a hardcoded `== 2` check; there is exactly one gate-time findings tool; ast-grep files fed into the semgrep executor do nothing.

**With this spec:** a rule declares `engine` as a first-class field; the gate looks the engine up in an `EngineBinding` table, groups a pack's rules by engine, gathers each engine's inputs by its declared `input_mode`, runs the engine, pipes non-SARIF output through a sandboxed pack-declared `convert` executable, and parses the one output contract (`parseSarif`). ast-grep is wired as the first new engine end-to-end with a trivial proof rule. `layer` retires; its per-tier field requirements re-key onto each engine as a field-contract. `backstop-go-pack`'s 14 rules flag-day migrate to `engine: semgrep`.

**Scope boundary.** This spec covers the **gate-time locus (locus A)** only. It does **not** cover:
- the fixture-time locus (`pkg/packval`, `pack validate`) — that is SPEC-032 (Seed 3), which depends on this spec's shared `EngineBinding`;
- pillar-2 packs-only / native-standards removal — that is SPEC-030 (Seed 1);
- the BUNDLE-009 contract seam — that is SPEC-033 (Seed 4).

**Verification Level:** Integration (80% coverage)
**Source Bundle:** BUNDLE-010 (pluggable-pack-engines), Spec Seed 2.

## Requirements

Requirements and claims are defined in frontmatter. The two dependency matrices this spec must exhaustively cover are summarized below; the body tables must match the frontmatter requirement text exactly.

### Engine field-contract matrix (REQ-003)

Each engine's field-contract re-keys the retired per-layer requirements. Validation verifies a rule's populated fields satisfy its declared engine's contract (verify, not guide — REQ-004).

| Engine | `requires` | `forbids` |
|---|---|---|
| semgrep | `rule_path`, `standard` | `category`, `input_scope`, `validator` |
| ast-grep | `rule_path` | `category`, `input_scope`, `validator` |
| sandbox | `validator`, `input_scope`, `category` | `rule_path` |
| config-file (native linter) | — | `rule_path`, `category`, `input_scope`, `validator` |

These `requires`/`forbids` lists are transcribed faithfully from the live `validateLayerFields` (pkg/pack/validate_manifest.go): semgrep ← layer-2, sandbox ← layer-3, config-file ← layer-1. Every forbid present today is preserved — no per-layer check is dropped in the re-key. ast-grep is rule-fed like semgrep and inherits semgrep's forbids; it differs only in not requiring `standard`. The sandbox engine additionally enforces the existing value-enum rules: `category` ∈ {`presence`, `structural`, `other`} with `other` requiring a non-empty `justification`, and `input_scope` ∈ {`single-file`, `multi-file`}.

Claims cover, per engine, every cell: each required-field pass, each missing-required fail, and each forbidden-field fail, plus the sandbox value-enum and justification checks. The claims are enumerated in frontmatter; no forbidden-field cell is left untested.

### Input-mode enum matrix (REQ-020)

`input_mode` has exactly four values plus the fail-loud unknown case. Each value maps to a ladder tier and a pack directory (REQ-021).

| `input_mode` | Injection behavior | Engine example | Pack dir |
|---|---|---|---|
| `config-file` | one optional pack config; tool runs its own built-in rules | golangci-lint / eslint / tsc | `linter/` |
| `rule-flags` | each rule file becomes a repeated `input_flag` occurrence | semgrep | `semgrep/` |
| `rule-dir` | rule files collected into one directory passed once via `input_flag` | ast-grep | `ast-grep/` |
| `none` | no rule/config injection; the executable is the logic | sandbox | `scripts/` |
| (unknown value) | blocking config error | — | — |

Claims cover all four values plus the unknown-value fail-loud (CLM-044 … CLM-048).

## Implementation

The gate-time engine path is a thin deterministic harness: **discover declarations → group by engine → gather inputs by mode → provision → run command → convert → parseSarif → normalize**. Each step below is a distinct processing stage the planner can map tasks to.

### 1. First-class `engine` field and `layer` retirement (REQ-001, REQ-002, REQ-015)

`pkg/pack/manifest.go` `Rule` gains an `Engine string` (`yaml:"engine"`) field and **removes** `Layer int` and its YAML key. Manifest parse/validation returns a `ConfigError` for any rule whose `engine` is empty (a `layer:`-only rule under the migrated reader). There is no `layer:2 → engine:semgrep` aliasing. `backstop-go-pack`'s 14 rules are bumped to `engine: semgrep` in the same change (the reader and the pack flip together).

### 2. The `EngineBinding` table in a leaf package (REQ-005, REQ-006, REQ-013)

A new `pkg/pack/engine` package holds `EngineBinding {Command, InputMode, InputFlag, ScopeKind, Convert?, Provision?}` (no `Format` — output is always SARIF), the `InputMode` enum + `ParseInputMode`, the `Provision` descriptor, and a `Registry` (`map[string]EngineBinding`). This package imports none of `pkg/check`, `pkg/packval`, `cmd/backstop`, so all three import it without a cycle. The built-in registry seeds `semgrep`, `ast-grep`, `sandbox`, and config-file linters; packs may contribute additional bindings. `Registry` lookup of an unknown engine returns a fail-loud config error (REQ-005). The sole findings parser **in the pack engine dispatch path** is `parseSarif`, resolved via `pkg/check`'s `lookupParser` (REQ-005/REQ-006); the per-tool lint parsers `golangci-json`/`eslint-json` are never referenced from dispatch. Build/test keep their `ToolchainEntry.Format` parsers and never enter this path (REQ-014). Those two lint parsers continue to live only in the native toolchain registry (pkg/check/registry.go); removing them from that native path is out of scope here and is owned by SPEC-030.

### 3. Engine-keyed field-contract validation (REQ-003, REQ-004)

`validateLayer` and `validateLayerFields` (pkg/pack/validate_manifest.go, pkg/pack/manifest.go) are replaced by `validateEngineFields`, which reads each engine's `requires[]`/`forbids[]` from the binding and emits a `ValidationError` naming the offending field and engine when a rule's populated fields violate the contract. Validation is verify-only: it never inspects rule content, recommends an engine, or reclassifies a rule (REQ-004). The matrix is in the Requirements section.

### 4. Group-by-engine gate dispatch (REQ-001, REQ-009, REQ-011)

`mergePackRules` → `ExtraSemgrepConfigs` → `semgrepExecutor` is replaced by `dispatchPackEngines`, which:
1. groups installed-pack rules by declared `engine`;
2. for each engine, looks up its `EngineBinding`;
3. gathers inputs per `input_mode` (stage 5);
4. provisions the engine (stage 6);
5. runs the engine command via the clean-stdout runner (stage 7);
6. **branches on the engine's output shape:**
   - **findings engines** (semgrep, ast-grep, config-file linters): pipe through
     `convert` when declared (stage 8), then parse normalized output via `parseSarif`
     and namespace violations (`pack-name/rule-id`);
   - **exit-code engines** (the sandbox / `input_mode: none` engine): take the
     exit-code terminal branch (stage 10), which never enters `convert` or
     `parseSarif`.

Mixed-engine packs dispatch each engine independently; ast-grep rules are never fed into a semgrep invocation and vice versa (REQ-011).

### 5. Input gathering by declared `input_mode` (REQ-020, REQ-021)

`input_mode` is read from the binding and shapes the invocation with no per-engine Go:
- `config-file`: append `input_flag` + the single pack config path (or nothing if the pack supplies none); the tool runs its own rules.
- `rule-flags`: for each rule file, append `input_flag` + path (repeated).
- `rule-dir`: collect the engine's rule files into one directory, append `input_flag` + dir once.
- `none`: append nothing.

Inputs resolve relative to the per-engine pack directory (`semgrep/`, `ast-grep/`, `linter/`, `scripts/` — REQ-021). A missing declared input path is a blocking broken-pack error naming the pack and path. `ParseInputMode` fail-louds on an unrecognized value (REQ-020).

### 6. Split engine provisioning (REQ-019)

A binding with an **empty** `Provision` is a Layer-0 assumed-present engine: a missing binary fails loud with a `ConfigError` naming the engine; backstop never installs it. A binding with a **pinned** `Provision` (semgrep, ast-grep) is auto-provisioned and verified through the existing `backstop.lock` / `VerifyLock` infra — data-driven, no per-engine Go. `EnsureSemgrep`'s bespoke install logic (pkg/check/semgrep.go) is retired into this declared mechanism.

### 7. Clean stdout capture (REQ-009)

`pkg/check/runner.go` `CommandRunner` gains a stdout-only method (`RunStdout`) implemented via `cmd.Output()` / an explicit stdout buffer rather than `CombinedOutput()`. Engine dispatch uses it so a tool's stderr banner/progress does not corrupt SARIF on stdout.

### 8. Sandboxed `convert` pipe (REQ-006, REQ-007, REQ-008)

When a binding declares a non-empty `Convert`, dispatch runs a two-process pipe **in Go** (no shell): engine stdout → `convert` stdin → `convert` stdout (SARIF) → `parseSarif`. The convert executable resolves relative to the pack directory and runs inside the same sandbox trust model as sandbox validators, but through a **clean-stdout sandbox capture** (a `SandboxedRunStdout`-style variant, REQ-009) rather than `SandboxedRun`'s `CombinedOutput()` — so a converter's stderr banner/warning cannot interleave into the SARIF bytes. An empty `Convert` sends engine stdout directly to `parseSarif`. Backstop hosts no converters; ast-grep ships its own stdin→SARIF script inside its pack (REQ-008).

### 10. Exit-code / sandbox (`none`) terminal branch (REQ-014)

The sandbox engine (`input_mode: none`) is the exit-code/pass-fail edge and does **not** ride `convert` or `parseSarif`. This branch is the relocation of the existing gate-time layer-3 validator path (`runPackValidators`, cmd/backstop/pack_gate.go, currently `if rule.Layer != 3 { continue }` → `sandboxedRun` → exit-code-to-violation). Because REQ-002 removes `rule.Layer`, the branch is re-keyed from `rule.Layer == 3` to `engine == sandbox` and folded into `dispatchPackEngines` as the exit-code terminal step. Its existing behavior is preserved unchanged: the `input_scope` single-file/multi-file target fan-out, the `SandboxedRun` (CombinedOutput) capture — whose merged stderr is legitimately allowed to form the violation message body — and the namespaced-violation emission on non-zero exit. The clean-stdout capture (§7, §8) applies only to the SARIF findings path, not here.

### 9. ast-grep wired end-to-end + proof rule (REQ-010)

ast-grep is registered as `{Command: "ast-grep scan", InputMode: rule-dir, InputFlag: "--rule", Convert: "<pack>/ast-grep/to-sarif.sh", Provision: <pinned>}`. A test pack carries a trivial ast-grep proof rule and the converter script; an integration test drives the full path (declaration → group → gather rule-dir → run → convert → parseSarif → namespaced violation).

## Verification

Verification is defined in frontmatter. Integration-level testing at 80% coverage across `cmd/backstop` (dispatch), `pkg/check` (runner, parser lookup), `pkg/pack` (manifest, engine-fit validation), and the new `pkg/pack/engine` (binding, input_mode).

Tests use temporary project directories with pre-installed test packs (testdata fixtures), a fake `CommandRunner` returning canned engine stdout/stderr, and a stub `convert` script so the SARIF pipe is exercised without a live ast-grep binary. The ast-grep end-to-end proof rule (CLM-030) uses a fixture pack plus a stub converter to avoid a live-tool dependency in CI.

## Sharp Edges

1. **`convert` and sandbox are macOS-only.** `SandboxedRun` wraps macOS `sandbox-exec`, so the `convert` pipe (and any `none`/sandbox engine) only runs on macOS today. ast-grep dispatch therefore only fully runs on macOS. This is a known, bundle-acknowledged vision-gap (BUNDLE-010 Non-Goal 10), not something this spec closes — but the implementation must fail loud (not silently green) on a platform where the sandbox is unavailable.

2. **The flag-day migration has no safety net.** Removing `layer` and rejecting `layer:`-only rules means any pack on disk that hasn't been migrated breaks immediately. This is correct only because the installed base is N=1 (`backstop-go-pack`, user-owned). The reader change and the pack rule bump must land together; landing the reader first turns the existing pack into a hard gate failure.

3. **`config-file` vs `rule-flags` ambiguity for an engine that accepts both.** A linter that can take both its own config and external rule files could plausibly be declared either way. The field-contract resolves this: a `config-file` engine `forbids` `rule_path`, so a rule that supplies external rule files must declare a rule-fed engine. Authors must pick the mode that matches how they inject inputs, and validation enforces the consequence.

4. **Empty `config-file` config is legitimate, missing rule-fed input is not.** Under `config-file`, a pack supplying no config is valid (the tool runs its own defaults). Under `rule-flags`/`rule-dir`, a missing declared rule path is a broken-pack error. The gather step must distinguish "no config offered" (fine) from "declared path absent" (fail) and not collapse them.

5. **A converter emitting non-SARIF still fails — but late, and the harness must not be the cause.** Because there is no `format` selector, a buggy `convert` script that emits genuine garbage on stdout is only caught when `parseSarif` rejects it. That is intentional (the parser is the contract), and the error must attribute the failure to the engine/pack and its convert step, not surface as an opaque SARIF parse error with no provenance. But there is a sharper trap: the convert executable's stdout is the *final* SARIF, so capturing it via `SandboxedRun`'s `CombinedOutput()` would merge any stderr banner/warning from an otherwise-clean converter into the parsed bytes and corrupt them — the harness itself contaminating a correct converter. REQ-007/REQ-009 close this by mandating a clean-stdout sandbox capture (`SandboxedRunStdout`) for the convert step; the legacy `CombinedOutput` `SandboxedRun` is retained only for the exit-code sandbox-validator path (REQ-014), where merged stderr is a legitimate message body, not parsed SARIF.

6. **`scope_kind` on the engine vs the existing pass `ScopeKind`.** The EngineBinding carries a `scope_kind` analogous to `pkg/check`'s `ScopeKind`, but engine dispatch and the native pass executor are different call paths. The two must not be conflated into one shared mutable value; the EngineBinding's `scope_kind` governs how the engine attaches scoped files, mirroring but not importing the pass-level concept (to keep `pkg/pack/engine` a leaf).

7. **Sandbox/`none` engines must not enter `parseSarif`, and the layer-3 branch must move, not break.** The dispatch's terminal step branches on output shape: findings engines parse SARIF, but the sandbox (`input_mode: none`) engine is exit-code/pass-fail and takes a separate terminal branch (REQ-014). The existing gate-time sandbox path (`runPackValidators`, keyed on `rule.Layer != 3`) breaks the moment REQ-002 deletes `rule.Layer`; this spec re-keys that branch to `engine == sandbox` and folds it into `dispatchPackEngines` rather than leaving it stranded against a removed field. Failing to relocate it would either dangle a dead `rule.Layer` reference (compile break) or silently drop layer-3 enforcement (vacuous green).

## Review Questions

1. Does `dispatchPackEngines` group by the **declared** engine string only, or does it ever infer an engine when `engine` is empty? It must fail loud, never infer — confirm no fallback path exists.

2. Is the `convert` pipe implemented with Go `io` plumbing between two `exec.Cmd` processes (no `sh -c`)? A shell intermediary would re-introduce the injection surface the sandbox is meant to bound — confirm no shell is invoked.

3. After removal, does any code path still read a `layer` YAML key, default it, or branch on `rule.Layer`? Grep must show zero remaining `Layer` references in the dispatch and validation paths.

4. Does the pack engine dispatch path reference `golangci-json` or `eslint-json` anywhere, or does it resolve only `parseSarif`? Those two parsers must remain confined to the native toolchain registry (out of scope here, owned by SPEC-030); confirm dispatch neither imports nor falls back to them, rather than confirming they were "removed" (this spec does not touch pkg/check/registry.go).

5. Does the clean-stdout runner change (`RunStdout`) leave the existing `Run`/`CombinedOutput` callers (build/test executors) untouched, or does it regress their behavior? The change must be additive for the engine path, not a global swap.

6. For a backstop-introduced engine, is provisioning genuinely routed through `VerifyLock`/`backstop.lock` (pinned + verified), or does it shell out to an installer outside the lock path the way `EnsureSemgrep` did? The trust surface must equal the `convert` step's.

7. Does the convert step capture the convert executable's stdout via the clean-stdout sandbox variant (`SandboxedRunStdout`), or does it call `SandboxedRun` (`CombinedOutput`)? The latter would merge a converter's stderr into the final SARIF and corrupt parsing — confirm the convert capture is stdout-only and that the `CombinedOutput` `SandboxedRun` remains used only by the exit-code sandbox-validator branch.

8. Does the sandbox/`none` engine reach `parseSarif` or `convert` at all? It must take the exit-code terminal branch only — confirm no code path routes a `none`-mode engine's output into the SARIF parser, and that the folded-in layer-3 logic (single-file/multi-file fan-out, non-zero-exit-to-violation) is preserved.

## References

- **BUNDLE-010** (pluggable-pack-engines) — Spec Seed 2; DD-1 (one key, `layer` retires), DD-2 (open engines / strict SARIF / `convert`), DD-3 (two loci; locus A here), DD-5 (flag-day migration), DD-8 (escalation ladder / Layer-0 first), DD-9 (`input_mode` seam), DD-10 (pack layout).
- **SPEC-030** — Seed 1, packs-only native-standards removal (collapses locus A's input to a single source; lands first).
- **SPEC-032** — Seed 3, fixture-time engine execution (`pkg/packval`); depends on this spec's `EngineBinding`.
- **SPEC-033** — Seed 4, BUNDLE-009 contract seam.
- **ISSUE-003** — data-driven toolchain registry: `ToolchainEntry`/`ScopeKind`, `formatParsers`, `lookupParser` fail-loud, the generic `commandExecutor` (the substrate this spec converges onto).
- **SPEC-017** — pack gate integration: `mergePackRules`, `loadInstalledPacks`, `NamespacedRuleID`, `VerifyLock` (the path this spec generalizes).
- Code: pkg/pack/manifest.go (`Rule`, `validateLayer`); pkg/pack/validate_manifest.go (`validateLayerFields`); cmd/backstop/pack_gate.go (`mergePackRules`, and `runPackValidators` — the existing `rule.Layer != 3` sandbox branch this spec re-keys to `engine == sandbox`); cmd/backstop/gate.go + code_check.go (`ExtraSemgrepConfigs` consumers); pkg/check/check.go (`semgrepExecutor`, `EnsureSemgrep`); pkg/check/parsers.go (`parseSarif`, `lookupParser`, `formatParsers`); pkg/check/runner.go (`CommandRunner`, `CombinedOutput`); pkg/packval/sandbox.go (`SandboxedRun` — gains a clean-stdout variant for the convert step); pkg/pack/distribution/verify.go (`VerifyLock`).
