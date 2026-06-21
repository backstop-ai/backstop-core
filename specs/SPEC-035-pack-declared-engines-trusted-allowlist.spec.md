---
title: "Pack Declared Engines Trusted Allowlist"
number: SPEC-035
created: "2026-06-20"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Finish BUNDLE-010's "first-class / pluggable engines" intent. SPEC-031/SPEC-034
    shipped the generic engine substrate — the `EngineBinding` struct
    (pkg/pack/engine/binding.go), the strict-SARIF output contract, and the generic
    group-by-engine dispatch (`dispatchPackEngines` / `runFindingsEngine` /
    `gatherEngineInputs` in cmd/backstop/pack_gate.go) — but the engine BINDINGS
    themselves still live HARDCODED in backstop Go (`DefaultRegistry()`,
    binding.go), and there is NO trust gate: the moment a pack could declare a
    command, `runFindingsEngine` would `splitCommand` → `RunStdout` it UNCHECKED.
    This spec moves the engine bindings out of hardcoded Go into a pack-declared
    `engines:` block on the manifest, and — SHIPPING IN THE SAME SPEC, as a
    non-negotiable security gate — introduces a backstop-owned trusted-tool
    ALLOWLIST (`{tool → pinned version}`) that every pack-declared command's `tool`
    must satisfy (allowlisted AND lock-pinned) before backstop will run it; an
    un-allowlisted or unpinned tool is a LOUD config error (exit 2), never a silent
    run. It also adds the `pattern-arg` input mode (a rule-level `pattern` field
    feeding `[input_flag, pattern]` for parameterized contract/absence queries
    BUNDLE-009 rides on), a backstop-owned tool-NEUTRAL gate-TYPE enum on the
    binding (lint/build/test/findings/coverage/substantiveness/contracts) replacing
    the tool-named `CheckTypeSemgrep`, and retires the three engine-NAME
    special-cases (`isNativeSarifLintEngine` sniffing "golangci-lint",
    `isNativeGoTestEngine` sniffing "go test", and validate_manifest's layout switch
    on "semgrep"/"ast-grep") into DECLARED binding flags + InputMode/ScopeKind-derived
    layout, so backstop knows zero tool names. The generic dispatch and the
    `EngineBinding` struct are ALREADY correct and MUST NOT be churned — this spec
    feeds them from pack data + a trust gate, it does not rewrite them. The
    DefaultRegistry-eradication-vs-incremental-fallback choice is left as an explicit
    Open Question, NOT pre-decided.
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/ ./pkg/pack/ ./pkg/pack/engine/ ./pkg/check/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      Engine bindings must be PACK-DECLARED, not hardcoded in backstop Go. The pack
      manifest schema (pkg/pack/manifest.go) must gain a top-level `engines:` block
      that maps an engine NAME to a binding spec carrying the EngineBinding fields
      that already exist (Command, InputMode, InputFlag, Convert, Provision,
      ScopeKind, Category, plus the new declarative flags from REQ-005/REQ-006):
      these fields gain yaml tags and are parsed at pack load and MERGED into the
      registry the gate resolves through (`resolveEngineRegistry`, pack_gate.go). A
      pack that declares an engine in its `engines:` block makes that binding
      available to its rules with NO change to backstop Go. The generic dispatch
      (`runFindingsEngine` / `gatherEngineInputs`) already reads binding fields and
      must require ZERO change to consume a pack-declared binding.
    supports: pluggable-pack-engines:REQ-001
    follows: STD-GO-001:GO-010
  - id: REQ-002
    text: >
      A backstop-owned TRUSTED-TOOL ALLOWLIST must ship IN THE SAME SPEC as REQ-001
      (it is a non-negotiable security gate, not a follow-up). The allowlist is a
      backstop-owned `{tool → pinned version}` map. Before backstop runs ANY
      pack-declared engine command — at the point `runFindingsEngine` would
      `splitCommand`(binding.Command) → `RunStdout` it — the engine's `tool` (the
      command's executable name) MUST be present on the allowlist AND its required
      version MUST be lock-pinned (verified through the existing
      backstop.lock / VerifyLock / Provision machinery). A tool that is NOT on the
      allowlist, OR is on the allowlist but not lock-pinned to its required version,
      MUST produce a LOUD config error (a *check.ConfigError, exit 2) naming the tool
      and the pack, and backstop MUST NOT run the command. There is no silent run, no
      silent skip, and no "run it anyway." This closes the arbitrary-command-execution
      hole that opens the instant packs can declare commands (today
      `runFindingsEngine` runs `splitCommand` → `RunStdout` with NO trust check).
    supports: pluggable-pack-engines:REQ-019
    follows: STD-GO-001:GO-010
  - id: REQ-003
    text: >
      `validateEngine` (pkg/pack/manifest.go) must change from "the engine name is
      known to engine.DefaultRegistry()" to "the engine name is known to the
      pack-declared registry (the manifest's own `engines:` block) UNION the
      built-in/fallback registry, AND the engine's tool is on the trusted-tool
      allowlist." A rule whose declared engine is unknown to that combined registry
      remains a blocking config error (the existing unknown-engine fail-loud), and a
      rule whose engine's tool is not allowlisted is ALSO a blocking config error. The
      allowlist check sits at BOTH validation time (here) and dispatch time (REQ-002),
      so an un-allowlisted tool cannot reach execution by any path.
    supports: pluggable-pack-engines:REQ-001
    follows: STD-GO-001:GO-010
  - id: REQ-004
    text: >
      A `pattern-arg` input mode must be added to the InputMode enum
      (`InputModePatternArg`, pkg/pack/engine/binding.go) alongside a rule-level
      `pattern` field on the Rule struct (pkg/pack/manifest.go). When an engine's
      input_mode is `pattern-arg`, `gatherEngineInputs` (cmd/backstop/pack_gate.go)
      must emit `[binding.InputFlag, rule.Pattern]` for each rule — the literal
      pattern string passed as a flag value — INSTEAD of resolving a rule file path on
      disk. A `pattern-arg` rule with an empty `pattern` is a blocking broken-pack
      config error naming the pack and rule (parallel to the missing-rule-path
      fail-loud). `ParseInputMode` must accept `pattern-arg` as a fifth valid value
      and continue to fail loud on any unrecognized value. This mode feeds the
      parameterized contract/absence queries BUNDLE-009 rides on.
    supports: pluggable-pack-engines:REQ-020
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      The EngineBinding must carry a backstop-owned, tool-NEUTRAL gate-TYPE enum
      declared per engine, so a pack declares "this engine fills the LINT type" (or
      build/test/findings/coverage/substantiveness/contracts) WITHOUT naming a tool.
      The enum has exactly these seven values and is defined in pkg/pack/engine
      (alongside the binding) so it carries no import of pkg/check. The tool-NAMED
      `CheckTypeSemgrep` (pkg/check/manifest.go / pkg/check/check.go) must be RENAMED
      to a neutral type (e.g. `CheckTypeFindings`) so no gate-type identifier names a
      specific tool. An engine's declared gate-type is data on the binding, parsed
      from the pack `engines:` block; an unrecognized gate-type value is a blocking
      config error (fail-loud, parallel to ParseInputMode).
    supports: pluggable-pack-engines:REQ-001
    follows: STD-GO-001:GO-010
  - id: REQ-006
    text: >
      The three engine-NAME special-cases must retire into DECLARATIVE binding flags
      so backstop knows zero tool names. (a) `isNativeSarifLintEngine`
      (cmd/backstop/pack_gate_golint.go), which sniffs a command prefix of
      "golangci-lint", must be replaced by a declared boolean output-contract flag on
      the binding (a `StrictSarif` field — the comment in pack_gate.go already
      references a non-existent `binding.StrictSarif`); the strict-SARIF shape guard
      keys off that declared flag, not the tool name. (b) `isNativeGoTestEngine`
      (cmd/backstop/pack_gate_filemode.go), which sniffs a command prefix of
      "go test", must be replaced by a declared package-scope capability flag on the
      binding (a `PackageScoped` field); the file-mode package scoping keys off that
      declared flag, not the tool name. (c) The layout switch in
      `ExpectedLayout`/`validateEngineFields` (pkg/pack/validate_manifest.go) that
      branches on engine names "semgrep"/"ast-grep" must DERIVE the expected pack
      layout from the binding's InputMode/ScopeKind, not from the engine name. After
      this requirement, a grep of cmd/backstop and pkg/pack non-test source for the
      literal tool names "golangci-lint", "go test", "semgrep", and "ast-grep" used
      as a DISPATCH/LAYOUT discriminator returns zero matches (string literals in
      default-pack data or documentation are exempt; the prohibition is on Go control
      flow keyed off a tool name).
    supports: pluggable-pack-engines:REQ-001
    follows: STD-GO-001:GO-010
  - id: REQ-007
    text: >
      The eight built-in engine bindings currently hardcoded in
      `DefaultRegistry()` (semgrep, ast-grep, sandbox, config-file, golangci,
      go-build, go-test — the commands and pinned versions) must be MIGRATED toward
      pack declaration. The DefaultRegistry-vs-default-pack disposition is an OPEN
      QUESTION (recorded below) and MUST NOT be pre-decided by this spec: the two
      stageable options are (i) keep DefaultRegistry as an incremental fallback that a
      pack `engines:` block OVERRIDES/EXTENDS (so go-toolchain / go-standards do not
      churn), or (ii) fully eradicate DefaultRegistry, ship the built-ins as a default
      pack, and have backstop know ZERO engines. Whichever option the user selects,
      the merge semantics in REQ-001 (`resolveEngineRegistry` reads pack-declared
      bindings) and the allowlist gate in REQ-002 (every resolved binding's tool must
      be allowlisted before it runs) MUST hold identically — the allowlist is the
      trust floor regardless of where the binding is declared.
    supports: pluggable-pack-engines:REQ-001
    follows: STD-GO-001:GO-010

claims:
  # REQ-001 — pack-declared engine bindings merged into the resolved registry
  - id: CLM-001
    requirement: REQ-001
    text: A pack manifest with a top-level engines block parses each binding spec into an EngineBinding (Command, InputMode, InputFlag, Convert, Provision, ScopeKind, Category) via its yaml tags
    tests:
      - TestManifest_EnginesBlockParsesBindingFields
  - id: CLM-002
    requirement: REQ-001
    text: A pack-declared engine binding is merged into the registry resolveEngineRegistry returns, so a rule declaring that engine resolves to the pack-declared binding
    tests:
      - TestResolveRegistry_PackDeclaredBindingMerged
  - id: CLM-003
    requirement: REQ-001
    text: A rule bound to a pack-declared engine dispatches through the EXISTING runFindingsEngine/gatherEngineInputs with no change to the generic dispatch — the binding fields drive it
    tests:
      - TestDispatch_PackDeclaredEngineRunsViaGenericDispatch
  - id: CLM-004
    requirement: REQ-001
    text: A pack engines block declaring an engine name that collides with a built-in resolves to the pack-declared binding under the spec's merge semantics (the resolved binding is well-defined, not ambiguous)
    tests:
      - TestResolveRegistry_PackBindingResolutionIsDeterministic

  # REQ-002 — trusted-tool allowlist: allowed (pass) and prohibited (fail) cells
  - id: CLM-005
    requirement: REQ-002
    text: A pack-declared command whose tool is on the allowlist AND lock-pinned to the required version runs through dispatch (the trust gate passes)
    tests:
      - TestAllowlist_AllowlistedPinnedToolRuns
  - id: CLM-006
    requirement: REQ-002
    text: A pack-declared command whose tool is NOT on the allowlist produces a loud ConfigError (exit 2) naming the tool and pack, and the command is never run
    tests:
      - TestAllowlist_UnallowlistedToolFailsLoud
  - id: CLM-007
    requirement: REQ-002
    text: A pack-declared command whose tool IS on the allowlist but is NOT lock-pinned to its required version produces a loud ConfigError (exit 2) and the command is never run
    tests:
      - TestAllowlist_AllowlistedButUnpinnedToolFailsLoud
  - id: CLM-008
    requirement: REQ-002
    text: The trust gate sits BEFORE splitCommand/RunStdout in runFindingsEngine — an un-allowlisted tool's command is never handed to the runner (no silent run, no partial execution)
    tests:
      - TestAllowlist_GateBlocksBeforeRunStdout
  - id: CLM-009
    requirement: REQ-002
    text: The sandbox engine (no command, input_mode none) is not subject to the command-allowlist tool check — it has no tool to allowlist and runs as before
    tests:
      - TestAllowlist_SandboxEngineNotSubjectToToolAllowlist

  # REQ-003 — validateEngine: pack-declared ∪ fallback ∪ allowlist (allowed + prohibited)
  - id: CLM-010
    requirement: REQ-003
    text: validateEngine accepts a rule whose engine is declared in the manifest's own engines block (not in the built-in registry) when that engine's tool is allowlisted
    tests:
      - TestValidateEngine_PackDeclaredEngineKnown
  - id: CLM-011
    requirement: REQ-003
    text: validateEngine accepts a rule whose engine is a built-in/fallback engine whose tool is allowlisted
    tests:
      - TestValidateEngine_BuiltinEngineKnown
  - id: CLM-012
    requirement: REQ-003
    text: validateEngine rejects (blocking config error) a rule whose engine is unknown to both the pack-declared block and the fallback registry
    tests:
      - TestValidateEngine_UnknownEngineRejected
  - id: CLM-013
    requirement: REQ-003
    text: validateEngine rejects (blocking config error) a rule whose engine is known but whose tool is not on the trusted-tool allowlist
    tests:
      - TestValidateEngine_KnownEngineUnallowlistedToolRejected

  # REQ-004 — pattern-arg input mode (parse-accept, emit, empty-fail) + enum exhaustiveness
  - id: CLM-014
    requirement: REQ-004
    text: ParseInputMode accepts "pattern-arg" as a valid InputMode value
    tests:
      - TestParseInputMode_PatternArgAccepted
  - id: CLM-015
    requirement: REQ-004
    text: ParseInputMode still fails loud on an unrecognized input_mode value after pattern-arg is added (no silent default)
    tests:
      - TestParseInputMode_UnknownStillFailsLoud
  - id: CLM-016
    requirement: REQ-004
    text: For a pattern-arg engine, gatherEngineInputs emits [InputFlag, rule.Pattern] for each rule instead of resolving a rule file path on disk
    tests:
      - TestGatherInputs_PatternArgEmitsFlagAndPattern
  - id: CLM-017
    requirement: REQ-004
    text: A pattern-arg rule with an empty pattern field is a blocking broken-pack config error naming the pack and rule
    tests:
      - TestGatherInputs_PatternArgEmptyPatternFailsLoud
  - id: CLM-018
    requirement: REQ-004
    text: A pattern-arg engine does NOT touch the filesystem for rule-path resolution — a rule with a pattern but no rule_path still gathers inputs successfully
    tests:
      - TestGatherInputs_PatternArgIgnoresRulePath

  # REQ-005 — neutral gate-type enum + CheckTypeSemgrep rename (all 7 values + fail-loud)
  - id: CLM-019
    requirement: REQ-005
    text: The gate-type enum is defined in pkg/pack/engine with exactly the seven neutral values lint, build, test, findings, coverage, substantiveness, contracts and carries no pkg/check import
    tests:
      - TestGateType_SevenNeutralValuesNoCheckImport
  - id: CLM-020
    requirement: REQ-005
    text: A pack engines block parses a declared gate_type into the binding's gate-type field for each of the seven valid values
    tests:
      - TestManifest_GateTypeParsesAllSevenValues
  - id: CLM-021
    requirement: REQ-005
    text: An unrecognized gate_type value in a pack engines block is a blocking config error (fail-loud, no silent default)
    tests:
      - TestManifest_UnknownGateTypeFailsLoud
  - id: CLM-022
    requirement: REQ-005
    text: The tool-named CheckTypeSemgrep is renamed to the neutral CheckTypeFindings and no gate-type identifier names a specific tool
    tests:
      - TestCheckType_SemgrepRenamedToNeutralFindings

  # REQ-006 — three name special-cases retire into declared flags (declared vs name-sniff)
  - id: CLM-023
    requirement: REQ-006
    text: The strict-SARIF shape guard fires based on the binding's declared StrictSarif flag, NOT a "golangci-lint" command-prefix sniff — a non-golangci command with StrictSarif true is guarded, and a golangci-named command with StrictSarif false is not
    tests:
      - TestStrictSarif_GuardKeyedOnDeclaredFlagNotName
  - id: CLM-024
    requirement: REQ-006
    text: File-mode package scoping fires based on the binding's declared PackageScoped flag, NOT a "go test" command-prefix sniff — a non-"go test" command with PackageScoped true is package-scoped, and a "go test"-named command with PackageScoped false is not
    tests:
      - TestPackageScoped_KeyedOnDeclaredFlagNotName
  - id: CLM-025
    requirement: REQ-006
    text: ExpectedLayout derives the expected pack layout from the binding's InputMode/ScopeKind, not from the engine names "semgrep"/"ast-grep"
    tests:
      - TestExpectedLayout_DerivedFromInputModeNotEngineName
  - id: CLM-026
    requirement: REQ-006
    text: A grep of cmd/backstop and pkg/pack non-test source finds no tool-name literal ("golangci-lint", "go test", "semgrep", "ast-grep") used as a dispatch/layout discriminator in Go control flow
    tests:
      - TestEndState_NoToolNameUsedAsDispatchDiscriminator

  # REQ-007 — DefaultRegistry migration: merge + allowlist invariant holds under either OQ option
  - id: CLM-027
    requirement: REQ-007
    text: Under the resolved merge semantics, the built-in engine bindings (their commands and pinned versions) are available to dispatch — whether sourced from DefaultRegistry fallback or a default pack — so go-toolchain/go-standards rules still dispatch
    tests:
      - TestMigration_BuiltinBindingsRemainAvailable
  - id: CLM-028
    requirement: REQ-007
    text: The allowlist trust gate (REQ-002) applies to a built-in/fallback binding's tool exactly as it applies to a pack-declared binding's tool — the allowlist is the trust floor regardless of binding source
    tests:
      - TestMigration_AllowlistAppliesToBuiltinBindingsToo

contracts:
  - file: pkg/pack/engine/binding.go
    provides:
      - name: InputModePatternArg
        kind: constant
        signature: "const InputModePatternArg InputMode = \"pattern-arg\""
        notes: "Fifth InputMode value (REQ-004/CLM-014). ParseInputMode adds it to the accepted switch and keeps failing loud on any other value (CLM-015). gatherEngineInputs emits [InputFlag, pattern] for this mode (CLM-016) — no filesystem rule-path resolution (CLM-018)."
      - name: GateType
        kind: type
        signature: "type GateType int"
        notes: "Backstop-owned, tool-NEUTRAL gate-type enum (REQ-005). Exactly seven values: lint, build, test, findings, coverage, substantiveness, contracts (CLM-019). Defined in pkg/pack/engine so it carries no pkg/check import (leaf-package placement, mirrors EngineCategory/ScopeKind). An unrecognized declared gate_type fails loud (CLM-021)."
      - name: ParseGateType
        kind: function
        signature: "func ParseGateType(s string) (GateType, error)"
        notes: "Fail-loud parser for the declared gate_type (CLM-021), mirroring ParseInputMode — no silent default."
      - name: EngineBinding
        kind: type
        signature: "type EngineBinding struct { Command string; InputMode InputMode; InputFlag string; ScopeKind ScopeKind; Convert string; Provision *Provision; CrashGuard bool; Category EngineCategory; ProjectTarget string; GateType GateType; StrictSarif bool; PackageScoped bool }"
        notes: "EXTENDED, not rewritten (constraint: do not churn the struct). Adds GateType (REQ-005), StrictSarif (REQ-006a — declared output-contract flag replacing the isNativeSarifLintEngine name-sniff; the pack_gate.go comment already references binding.StrictSarif), and PackageScoped (REQ-006b — declared file-mode package-scope capability replacing the isNativeGoTestEngine name-sniff). All existing fields keep their meaning. yaml tags are added to every field so the pack engines block parses into this struct (REQ-001/CLM-001)."
      - name: ParseInputMode
        kind: function
        signature: "func ParseInputMode(s string) (InputMode, error)"
        notes: "Accepts pattern-arg as the fifth value; unchanged fail-loud contract on unknown values (REQ-004/CLM-014/CLM-015)."
    consumes: []

  - file: pkg/pack/engine/allowlist.go
    provides:
      - name: TrustedToolAllowlist
        kind: function
        signature: "func TrustedToolAllowlist() map[string]string"
        notes: "Backstop-OWNED {tool -> pinned version} allowlist (REQ-002). The trust floor: a tool absent from this map may not be run by any pack-declared command. Lives in pkg/pack/engine beside the binding so both validate-time and dispatch-time checks consume one source. The pinned version rides the existing backstop.lock / Provision verification (CLM-007)."
      - name: CheckToolAllowed
        kind: function
        signature: "func CheckToolAllowed(allowlist map[string]string, tool string, lockedVersion string) error"
        notes: "Returns a non-nil error (wrapped by callers into a *check.ConfigError) when tool is not on the allowlist OR lockedVersion does not match the allowlist's pinned version (REQ-002/CLM-006/CLM-007). Pure function so both validateEngine (REQ-003) and the dispatch gate (REQ-002) call the SAME check."
    consumes: []

  - file: pkg/pack/manifest.go
    provides:
      - name: Manifest
        kind: type
        signature: "type Manifest struct { Name string; NormalizedName string; Version string; Language string; Archetype string; Description string; Content Content; ToolConfig []ToolConfigEntry; Engines map[string]EngineSpec }"
        notes: "Gains a top-level Engines map[string]EngineSpec parsed from the pack `engines:` block (REQ-001/CLM-001). EngineSpec carries the yaml-tagged binding fields (Command, InputMode, InputFlag, Convert, Provision, ScopeKind, Category, GateType, StrictSarif, PackageScoped) and converts to engine.EngineBinding at load."
      - name: Rule
        kind: type
        signature: "type Rule struct { ID string; NamespacedID string; Engine string; Standard string; RulePath string; Pattern string; RiskClass string; Claims []Claim; Category string; Justification string; Validator string; InputScope string; PairsWith PairsWith }"
        notes: "Gains a Pattern string field (yaml: pattern) for pattern-arg engines (REQ-004). Empty Pattern under a pattern-arg engine is a broken-pack config error (CLM-017)."
      - name: validateEngine
        kind: function
        signature: "func validateEngine(name string, declared map[string]engine.EngineBinding) error"
        notes: "Changed from a DefaultRegistry-only lookup to: known in the pack-declared registry UNION the fallback registry, AND the engine's tool is allowlisted (REQ-003/CLM-010..013). Unknown engine OR un-allowlisted tool is a blocking config error."
    consumes:
      - source: pkg/pack/engine/binding.go
        name: EngineBinding
        kind: type
      - source: pkg/pack/engine/allowlist.go
        name: CheckToolAllowed
        kind: function

  - file: cmd/backstop/pack_gate.go
    provides:
      - name: gatherEngineInputs
        kind: function
        signature: "func gatherEngineInputs(manifest *pack.Manifest, packRoot string, binding engine.EngineBinding, rules []pack.Rule) ([]string, error)"
        notes: "Gains the InputModePatternArg case emitting [binding.InputFlag, rule.Pattern] per rule, with an empty-pattern fail-loud (REQ-004/CLM-016/CLM-017/CLM-018). The four existing cases are unchanged."
      - name: runFindingsEngine
        kind: function
        signature: "func runFindingsEngine(manifest *pack.Manifest, packRoot, projectRoot string, scope *gate.GateScope, binding engine.EngineBinding, rules []pack.Rule, runner check.CommandRunner) ([]gate.Violation, error)"
        notes: "Gains the trust gate (REQ-002): BEFORE splitCommand(binding.Command) -> runner.RunStdout, it resolves the engine's tool from the command and calls CheckToolAllowed against the allowlist + the lock-pinned version; a failure returns a *check.ConfigError naming tool+pack and the command is never run (CLM-005..008). The generic gather/run/convert/parseSarif flow is otherwise unchanged. The strict-SARIF guard now keys off binding.StrictSarif (REQ-006a)."
      - name: resolveEngineRegistry
        kind: function
        signature: "func resolveEngineRegistry(manifest *pack.Manifest) engine.Registry"
        notes: "Resolves a rule's engine against the manifest's pack-declared engines block merged with the fallback registry (REQ-001/CLM-002/CLM-004). The merge semantics are the contract the DefaultRegistry OQ resolution must preserve (REQ-007)."
    consumes:
      - source: pkg/pack/engine/allowlist.go
        name: TrustedToolAllowlist
        kind: function
      - source: pkg/pack/engine/allowlist.go
        name: CheckToolAllowed
        kind: function
      - source: pkg/check/semgrep.go
        name: ConfigError
        kind: type

  - file: cmd/backstop/pack_gate_golint.go
    provides:
      - name: requireLintSarifShape
        kind: function
        signature: "func requireLintSarifShape(manifest *pack.Manifest, binding engine.EngineBinding, stdout []byte) error"
        notes: "isNativeSarifLintEngine's command-prefix sniff of \"golangci-lint\" is DELETED; the strict-SARIF shape guard now keys off binding.StrictSarif (REQ-006a/CLM-023). Behavior preserved: a binding with StrictSarif true whose stdout is not v2 SARIF fails loud, engine-attributed."
    consumes:
      - source: pkg/pack/engine/binding.go
        name: EngineBinding
        kind: type

  - file: cmd/backstop/pack_gate_filemode.go
    provides:
      - name: fileModeTestTarget
        kind: function
        signature: "func fileModeTestTarget(binding engine.EngineBinding, scope *gate.GateScope) (string, bool)"
        notes: "isNativeGoTestEngine's command-prefix sniff of \"go test\" is DELETED; file-mode package scoping now keys off binding.PackageScoped (REQ-006b/CLM-024). Behavior preserved: a PackageScoped project-wide engine under a file-mode scope scopes to the changed file's package."
    consumes:
      - source: pkg/pack/engine/binding.go
        name: EngineBinding
        kind: type

  - file: pkg/pack/validate_manifest.go
    provides:
      - name: ExpectedLayout
        kind: function
        signature: "func ExpectedLayout(m *Manifest) []string"
        notes: "The engine-NAME switch on \"semgrep\"/\"ast-grep\" is DELETED; the expected rules/ vs validators/ layout is DERIVED from each rule's resolved binding InputMode/ScopeKind (rule-fed input modes => rules/; input_mode none => validators/) (REQ-006c/CLM-025)."
    consumes:
      - source: pkg/pack/engine/binding.go
        name: InputMode
        kind: type

  - file: pkg/check/manifest.go
    provides:
      - name: CheckTypeFindings
        kind: constant
        signature: "const CheckTypeFindings CheckType = ..."
        notes: "RENAME of the tool-named CheckTypeSemgrep to a neutral type (REQ-005/CLM-022). The String() method and passOrder (pkg/check/check.go) update accordingly; no gate-type identifier names a tool."
    consumes: []
---

# SPEC-035: Pack-Declared Engines + Trusted-Tool Allowlist

## Overview

This spec **finishes BUNDLE-010's "first-class / pluggable engines" intent.**
SPEC-031 and SPEC-034 shipped the generic engine substrate — the `EngineBinding`
struct (pkg/pack/engine/binding.go), the strict-SARIF output contract, and the
generic group-by-engine dispatch (`dispatchPackEngines` / `runFindingsEngine` /
`gatherEngineInputs`, cmd/backstop/pack_gate.go). But two gaps remain between the
substrate and the target model:

1. **The engine bindings still live HARDCODED in backstop Go.** `DefaultRegistry()`
   (binding.go) carries eight built-in bindings — their commands and pinned
   versions — baked into the binary. A pack cannot declare its own engine; adding a
   stack's toolchain is still a Go edit, the opposite of the pack thesis.
2. **There is NO trust gate.** The dispatch is already generic enough to run any
   command a binding carries — `runFindingsEngine` does `splitCommand(binding.Command)`
   → `runner.RunStdout(...)` with **no trust check**. The instant packs can declare
   commands (gap 1's fix), an arbitrary command would execute UNCHECKED. The trust
   gate is therefore **non-negotiable and ships in the same spec** as the
   pack-declared bindings — they cannot land apart without opening an
   arbitrary-command-execution hole.

**The target model.** Backstop = a fixed set of gate **TYPES** + their semantics +
a trusted-tool **ALLOWLIST**. A **PACK** is self-describing: it declares its engines
(each: tool/command, input_mode, input_flag, convert, provision, scope_kind,
category, and which gate TYPE it fills) and its rules. Backstop runs a pack-declared
command **only if its `tool` is on the pinned allowlist**; it otherwise knows nothing
about how to run any tool. The "how to run each tool/stack" detail lives in packs;
backstop's surface stays **flat** as stacks (typescript / java / rust / ruby / …)
multiply.

**What this spec changes, against the EXISTING generic substrate it must not churn:**

- **Pack-declared `engines:` block** (REQ-001) — a top-level manifest map of engine
  name → binding spec; the EngineBinding fields gain yaml tags + parse + merge into
  the resolved registry. The generic dispatch needs ZERO change (it already reads
  binding fields).
- **Trusted-tool allowlist** (REQ-002, security gate) — a backstop-owned
  `{tool → pinned version}` map; every pack-declared command's tool must be
  allowlisted AND lock-pinned before it runs, else a loud config error.
- **`validateEngine` widening** (REQ-003) — from "known to DefaultRegistry" to "known
  to pack-declared ∪ fallback registry AND tool allowlisted."
- **`pattern-arg` input mode** (REQ-004) — a fifth InputMode + a rule-level `pattern`
  field; `gatherEngineInputs` emits `[input_flag, pattern]`. BUNDLE-009's
  parameterized contract/absence queries ride on this.
- **Neutral gate-TYPE enum** (REQ-005) — a tool-neutral
  lint/build/test/findings/coverage/substantiveness/contracts enum on the binding;
  rename the tool-named `CheckTypeSemgrep` → `CheckTypeFindings`.
- **Retire three name special-cases** (REQ-006) — `isNativeSarifLintEngine`
  ("golangci-lint" sniff) → declared `StrictSarif`; `isNativeGoTestEngine` ("go test"
  sniff) → declared `PackageScoped`; validate_manifest layout switch on engine names
  → InputMode/ScopeKind-derived layout. Backstop ends up knowing zero tool names.
- **DefaultRegistry migration** (REQ-007) — move the eight built-ins toward pack
  declaration; fallback-vs-eradicate left as an **Open Question** below.

**Constraints (from the bundle / prompt).** The generic dispatch
(`runFindingsEngine` / `gatherEngineInputs`) and the `EngineBinding` struct are
ALREADY correct — keep them, extend (do not rewrite) the struct, and feed them from
pack data + a trust gate. SARIF normalization stays. The Provision/lock machinery is
generic — reuse it (the allowlist's version-pin rides the lock). This is
**enforcement-path + security work**: no vacuous green, fail-loud on un-allowlisted
tools and unknown engines.

**Verification Level:** Integration (80% coverage).
**Source Bundle:** BUNDLE-010 (pluggable-pack-engines) — discharges REQ-001
(pack-declared engines), REQ-019 (the allowlist as the trust floor of split
provisioning), and REQ-020 (the `input_mode` family, extended with `pattern-arg`).

## Requirements

Requirements and claims are defined in frontmatter. The two summary tables below
must match the requirement text exactly.

### Allowlist trust gate: every cell (REQ-002, REQ-003)

The trust gate is an allowlist with a version pin. A pack-declared command runs
**only** when both conditions hold; either failing is a loud `ConfigError` (exit 2),
never a silent run or skip. The gate is checked at BOTH validation time (REQ-003) and
dispatch time (REQ-002).

| Tool on allowlist? | Lock-pinned to required version? | Outcome |
|---|---|---|
| yes | yes | command RUNS (CLM-005) |
| yes | no | loud ConfigError, command NEVER runs (CLM-007) |
| no | — | loud ConfigError, command NEVER runs (CLM-006) |

The sandbox engine (no command, `input_mode: none`) carries no tool and is exempt
from the command-allowlist check (CLM-009).

### `validateEngine` resolution matrix (REQ-003)

| Engine known in pack `engines:` block? | Known in fallback registry? | Tool allowlisted? | validateEngine |
|---|---|---|---|
| yes | — | yes | accepts (CLM-010) |
| — | yes | yes | accepts (CLM-011) |
| no | no | — | rejects — unknown engine (CLM-012) |
| yes or fallback | — | no | rejects — un-allowlisted tool (CLM-013) |

### InputMode values after this spec (REQ-004)

`gatherEngineInputs` dispatches on the binding's `input_mode`. The five values and
how each gathers inputs:

| input_mode | how inputs are gathered |
|---|---|
| `config-file` | one optional pack config via `input_flag` (tool runs its own rules) |
| `rule-flags` | repeated `input_flag <rule-file>` per rule file |
| `rule-dir` | rule files collected into a directory passed once via `input_flag` |
| `pattern-arg` | `[input_flag, rule.Pattern]` per rule — literal pattern, NO file resolution (REQ-004) |
| `none` | no injection; the executable is the logic (sandbox) |

### Neutral gate-TYPE enum (REQ-005)

The gate-TYPE enum is backstop-owned and **tool-neutral**, declared per engine on the
binding. Exactly seven values; an unrecognized declared value fails loud.

| gate_type value | meaning |
|---|---|
| `lint` | the lint gate type |
| `build` | the build gate type |
| `test` | the test gate type |
| `findings` | located findings (the type the renamed `CheckTypeSemgrep` → `CheckTypeFindings` fills) |
| `coverage` | the coverage gate type |
| `substantiveness` | the test-substantiveness gate type |
| `contracts` | the contract-signature gate type |

### Three name special-cases → declared flags (REQ-006)

| Retired name special-case | Sniffs | Replaced by declared flag / derivation |
|---|---|---|
| `isNativeSarifLintEngine` (pack_gate_golint.go) | command prefix `"golangci-lint"` | `binding.StrictSarif` (declared bool) |
| `isNativeGoTestEngine` (pack_gate_filemode.go) | command prefix `"go test"` | `binding.PackageScoped` (declared bool) |
| `ExpectedLayout`/`validateEngineFields` switch (validate_manifest.go) | engine names `"semgrep"`/`"ast-grep"` | derive layout from `InputMode`/`ScopeKind` |

## Implementation

The work is a set of distinct, planner-mappable stages. Stages 1 and 2
(pack-declared bindings + the allowlist) MUST land together — the allowlist is the
trust gate that makes pack-declared commands safe, and they cannot ship apart
without opening the arbitrary-command-execution hole. The `EngineBinding` struct and
the generic dispatch are EXTENDED/fed, never rewritten.

### 1. Pack-declared `engines:` block (REQ-001)

Add a top-level `Engines map[string]EngineSpec` to `Manifest` (pkg/pack/manifest.go),
where `EngineSpec` carries the yaml-tagged binding fields (Command, InputMode,
InputFlag, Convert, Provision, ScopeKind, Category, plus the new GateType /
StrictSarif / PackageScoped). At load, each spec converts to an
`engine.EngineBinding`. `resolveEngineRegistry` (cmd/backstop/pack_gate.go) is
changed to take the manifest and return the pack-declared bindings MERGED with the
fallback registry, so a rule's declared engine resolves to the pack-declared binding
when present. The generic dispatch (`runFindingsEngine` / `gatherEngineInputs`) is
UNCHANGED — it already reads binding fields, so a pack-declared binding dispatches
with zero executor edits (CLM-003). The merge resolution must be deterministic
(CLM-004): the spec defines whether a pack binding overrides or coexists with a
same-named built-in (this is the same knob the REQ-007 Open Question turns).

### 2. Trusted-tool allowlist (REQ-002) — ships with stage 1

Add `pkg/pack/engine/allowlist.go`: a backstop-owned `TrustedToolAllowlist()`
returning `{tool → pinned version}`, plus a pure `CheckToolAllowed(allowlist, tool,
lockedVersion)` returning an error when the tool is absent or the locked version does
not match. In `runFindingsEngine`, BEFORE `splitCommand(binding.Command)` →
`runner.RunStdout`, resolve the engine's tool (the command's executable name) and run
`CheckToolAllowed` against the allowlist and the lock-pinned version (read through the
existing backstop.lock / `VerifyLock` / `Provision` path). A failure returns a
`*check.ConfigError` (exit 2) naming the tool and pack; the command is NEVER handed to
the runner (CLM-005..008). The sandbox engine (no command) is exempt (CLM-009). The
version pin RIDES the existing lock machinery — no new trust model is invented.

### 3. `validateEngine` widening (REQ-003)

`validateEngine` (pkg/pack/manifest.go) takes the manifest's declared engine bindings
and checks the rule's engine against the pack-declared registry UNION the fallback
registry, AND that the engine's tool is allowlisted. Unknown engine → blocking config
error (existing behavior). Known engine, un-allowlisted tool → blocking config error
(new). Both `validateEngine` and the dispatch gate call the SAME `CheckToolAllowed`,
so no path reaches execution with an un-allowlisted tool (CLM-010..013).

### 4. `pattern-arg` input mode (REQ-004)

Add `InputModePatternArg` to the InputMode enum (binding.go) and accept it in
`ParseInputMode` (still failing loud on any other value). Add `Pattern string`
(`yaml: pattern`) to `Rule` (manifest.go). In `gatherEngineInputs`
(cmd/backstop/pack_gate.go), add the `InputModePatternArg` case: emit
`[binding.InputFlag, rule.Pattern]` per rule, with an empty-pattern fail-loud naming
the pack and rule — and NO filesystem rule-path resolution (CLM-014..018).

### 5. Neutral gate-TYPE enum + `CheckTypeSemgrep` rename (REQ-005)

Add a `GateType` enum + `ParseGateType` in pkg/pack/engine (leaf-package placement, no
pkg/check import) with the seven tool-neutral values; an unrecognized declared
`gate_type` fails loud. Add a `GateType` field to `EngineBinding`, parsed from the
pack `engines:` block. Rename the tool-named `CheckTypeSemgrep` (pkg/check/manifest.go
/ check.go) to `CheckTypeFindings`, updating `String()` and `passOrder`
(CLM-019..022).

### 6. Retire the three name special-cases (REQ-006)

(a) Add `StrictSarif bool` to `EngineBinding`; delete `isNativeSarifLintEngine`'s
"golangci-lint" sniff and key `requireLintSarifShape` off `binding.StrictSarif`
(pack_gate_golint.go). (b) Add `PackageScoped bool`; delete `isNativeGoTestEngine`'s
"go test" sniff and key `fileModeTestTarget` off `binding.PackageScoped`
(pack_gate_filemode.go). (c) In `ExpectedLayout` (validate_manifest.go), delete the
`"semgrep"`/`"ast-grep"` switch and derive the expected `rules/` vs `validators/`
layout from each rule's resolved binding `InputMode`/`ScopeKind` (rule-fed modes →
`rules/`; `input_mode: none` → `validators/`). End state: no tool-name literal drives
Go control flow (CLM-023..026).

### 7. DefaultRegistry migration (REQ-007) — Open Question, staged

Move the eight built-in bindings (commands + pinned versions) toward pack
declaration. The disposition (incremental fallback vs full eradication) is an OPEN
QUESTION below and is NOT pre-decided here. Whichever option is chosen, the stage-1
merge semantics and the stage-2 allowlist gate hold identically: the built-ins'
tools are subject to the SAME allowlist as any pack-declared tool (CLM-027/CLM-028).

## Verification

Verification is defined in frontmatter. Integration-level testing at 80% coverage
across `cmd/backstop` (the dispatch trust gate, pattern-arg gather, the retired name
special-cases), `pkg/pack` (manifest `engines:` parsing, `validateEngine` widening,
layout derivation), `pkg/pack/engine` (the binding fields, InputMode/GateType
parsing, the allowlist), and `pkg/check` (the `CheckTypeSemgrep` → `CheckTypeFindings`
rename).

Tests use a fake `CommandRunner` so the trust gate is exercised without live tools: a
test asserts an un-allowlisted tool's command is NEVER passed to `RunStdout`
(CLM-008). The allowlist matrix (CLM-005..007) injects a stub allowlist and a stub
lock-pinned version to drive each cell — allowed+pinned (run), allowed+unpinned
(fail), absent (fail). The pattern-arg claims (CLM-016..018) assert the gathered args
are exactly `[input_flag, pattern]` with no `os.Stat` on a rule path. The
name-special-case claims (CLM-023..025) assert the guard/scoping/layout fire off the
DECLARED flag or InputMode, proven by a binding whose flag and whose command-name
DISAGREE (a non-golangci command with `StrictSarif: true`; a non-"go test" command
with `PackageScoped: true`). CLM-026 greps non-test source for tool-name literals used
as discriminators. The DefaultRegistry-migration claims (CLM-027/CLM-028) hold under
whichever Open-Question option the user selects — they assert the merge + allowlist
INVARIANTS, not a specific binding source.

No test may stub the allowlist OPEN (returning "allowed" for everything) on the
dispatch path under test — that would re-introduce the arbitrary-exec hole the
allowlist closes; the trust-gate tests must drive a real allowlist with a real
absent-tool cell.

## Open Questions

This Open Question is left for the USER to resolve and is NOT pre-decided. The
requirements and claims above are written to hold under EITHER resolution.

- **OQ-1 — Keep `DefaultRegistry` as an incremental fallback, or fully eradicate it
  into a default pack? (REQ-007)** Two stageable options:
  - **(i) Incremental fallback.** Keep `DefaultRegistry()` as a built-in registry
    that a pack `engines:` block OVERRIDES/EXTENDS via the stage-1 merge. The eight
    built-ins (semgrep, ast-grep, sandbox, config-file, golangci, go-build, go-test)
    stay available with no pack churn, so `go-toolchain` / `go-standards` keep
    working unchanged. Backstop still ships SOME baked-in engine knowledge.
  - **(ii) Full eradication.** Delete `DefaultRegistry()`; ship the built-ins as a
    DEFAULT PACK that backstop installs/consumes like any other; backstop knows ZERO
    engines. Cleaner end-state ("backstop knows no tools"), but a larger change that
    churns the toolchain packs and the default-install path.

  Either way the merge semantics (REQ-001) and the allowlist trust floor (REQ-002)
  are identical — the allowlist gates every binding's tool regardless of source. The
  choice is stageable: option (i) first to land the allowlist + pack-declared
  bindings safely, option (ii) later as a follow-up if "zero baked-in engines" is
  wanted. Recording it here so the spec does not silently assume one.

## Sharp Edges

1. **The allowlist is the trust boundary — and it must SHIP WITH the pack-declared
   bindings, not after.** The instant a pack can declare a command (REQ-001),
   `runFindingsEngine` would `splitCommand` → `RunStdout` it. If the allowlist
   (REQ-002) lands in a later commit, there is a window where ANY pack-declared
   command executes UNCHECKED — arbitrary command execution from pack data. The two
   requirements are one atomic change; a planner that sequences them apart, or a
   reviewer that approves stage 1 without stage 2, ships the hole. This is the
   cardinal sin of this spec.

2. **A trusted tool can still be weaponized through its OWN config/plugins (honest
   caveat).** The allowlist shrinks "run any command a pack declares" down to "run a
   pinned set of trusted tools" — but a trusted tool (semgrep, golangci-lint,
   ast-grep) can itself be driven to do harm via a malicious pack-supplied config,
   custom rule, or plugin it loads. The allowlist reduces the attack surface to the
   normal **"trust your toolchain"** surface every project already accepts; it does
   NOT reduce it to zero. This is a documented limitation, not a defect — recorded so
   no one over-claims the allowlist as a sandbox. (Cross-platform sandboxing of the
   convert/validator steps remains BUNDLE-010 Non-Goal 10's known gap.)

3. **Allowlist-stubbed-OPEN in tests re-opens the hole.** A test fake that returns
   "allowed" for every tool makes the trust-gate tests pass vacuously while proving
   nothing. The trust-gate tests MUST drive a real allowlist with a genuine
   absent-tool cell (CLM-006) and a genuine unpinned cell (CLM-007). The Verification
   section forbids stubbing the allowlist open on the dispatch path under test.

4. **The version pin must ride the LOCK, not a second source of truth.** The
   allowlist carries a pinned version, and the existing backstop.lock / `VerifyLock` /
   `Provision` machinery already pins+verifies tool versions. If the allowlist check
   reads a version from somewhere OTHER than the lock, the two can drift and a tool
   could be "allowlisted at vX" while the lock pins vY — a silent inconsistency.
   Reuse the lock path (REQ-002); do not invent a parallel pin.

5. **Name special-cases hide as "harmless string checks."** Retiring
   `isNativeSarifLintEngine`/`isNativeGoTestEngine`/the layout switch (REQ-006) is
   easy to half-do: leave the `strings.HasPrefix(binding.Command, "go test")` sniff
   "as a fallback" beside the new flag. That defeats the point — backstop must know
   ZERO tool names in dispatch/layout control flow. CLM-026's grep is the guard; a
   declared flag that merely SHADOWS a still-present name sniff fails the intent even
   if tests pass.

6. **Declared flag vs command name disagreement is the real test.** CLM-023/CLM-024
   are only meaningful if the binding's declared flag and its command name DISAGREE
   (a non-golangci command with `StrictSarif: true`; a non-"go test" command with
   `PackageScoped: true`). A test that uses the historical golangci/go-test commands
   would pass under BOTH the old name-sniff and the new declared flag, proving
   nothing. The fixtures must force the divergence.

7. **`pattern-arg` must not silently fall through to a file-path resolution.** A
   `pattern-arg` rule has a `pattern`, not a `rule_path`. If `gatherEngineInputs`'
   new case is missing or mis-ordered, the rule could fall into a rule-path branch and
   fail-loud on a "missing rule file" — a confusing wrong error — or worse, an empty
   pattern could emit `[input_flag, ""]` and pass an empty arg to the tool. The
   empty-pattern fail-loud (CLM-017) and the no-filesystem-touch assertion (CLM-018)
   pin both edges.

8. **`CheckTypeSemgrep` rename ripples through `passOrder` and `String()`.** The
   rename (REQ-005) is not a single-line change: `passOrder` (pkg/check/check.go),
   the `String()` switch (pkg/check/manifest.go), and the `delete(opts.Executors,
   CheckTypeSemgrep)` site (check.go:322) all reference it. A rename that misses one
   leaves a tool-named identifier alive or breaks the build; the neutral-name claim
   (CLM-022) must confirm no tool-named gate-type identifier survives.

9. **Merge ambiguity if a pack redeclares a built-in engine name.** A pack
   `engines:` block declaring `semgrep` with a different command collides with the
   built-in. The merge semantics (CLM-004) must be deterministic — the spec says the
   resolved binding is well-defined (pack-declared wins, or built-ins are
   reserved-and-rejected; either is fine, but it must be DECIDED and tested, not left
   to map-iteration order). This same knob is what OQ-1 turns.

## Review Questions

1. Do the pack-declared `engines:` block (REQ-001) and the trusted-tool allowlist
   (REQ-002) land as ONE atomic change? Confirm there is no commit or stage where a
   pack-declared command can run before the allowlist gate exists — the
   arbitrary-exec window must never open.

2. Is the allowlist trust gate placed BEFORE `splitCommand`/`RunStdout` in
   `runFindingsEngine`, so an un-allowlisted tool's command is never handed to the
   runner (CLM-008)? Confirm it is not a post-hoc check after the command already ran.

3. Are BOTH the allowed and prohibited allowlist cells tested — allowed+pinned (runs),
   allowed+unpinned (fails), absent (fails) (CLM-005/006/007)? Confirm no test stubs
   the allowlist open on the dispatch path under test.

4. Does the version pin read from the EXISTING backstop.lock / `VerifyLock` /
   `Provision` path, or did the implementation introduce a second version source that
   can drift from the lock (Sharp Edge 4)?

5. Does `validateEngine` reject a known engine whose tool is NOT allowlisted
   (CLM-013), in addition to rejecting an unknown engine (CLM-012)? Both rejections
   must hold so no path reaches dispatch with an un-allowlisted tool.

6. For `pattern-arg`: does `gatherEngineInputs` emit exactly `[input_flag, pattern]`
   without touching the filesystem (CLM-016/CLM-018), and does an empty `pattern`
   fail loud (CLM-017) rather than emitting an empty arg?

7. Do the strict-SARIF guard and the file-mode package scoping fire off the DECLARED
   `StrictSarif`/`PackageScoped` flags, proven by a binding whose flag DISAGREES with
   its command name (CLM-023/CLM-024)? Confirm no `strings.HasPrefix` tool-name sniff
   survives as a fallback.

8. Does `ExpectedLayout` derive layout from `InputMode`/`ScopeKind`, with the
   `"semgrep"`/`"ast-grep"` name switch deleted (CLM-025)? And does the
   non-test-source grep for tool-name literals as discriminators return zero
   (CLM-026)?

9. Is the gate-TYPE enum tool-neutral with exactly the seven values, defined in
   pkg/pack/engine with no pkg/check import (CLM-019), and is `CheckTypeSemgrep`
   genuinely renamed everywhere it is referenced (passOrder, String(), the delete
   site) (CLM-022)?

10. Is the `EngineBinding` struct EXTENDED (new fields appended) rather than
    rewritten, and is the generic dispatch (`runFindingsEngine`/`gatherEngineInputs`)
    left intact apart from the new pattern-arg case and the trust gate? The bundle
    constraint is "do not churn the correct substrate."

11. Is the `DefaultRegistry` disposition (OQ-1) left UNRESOLVED for the user, with the
    spec holding under either fallback or eradication, and the allowlist gating
    built-in tools the same as pack-declared tools (CLM-028)?

## References

- **BUNDLE-010** (pluggable-pack-engines) — discharges **REQ-001** (engine is
  implicit-via-Go; should be first-class/pack-declared), **REQ-019** (split
  provisioning; the allowlist is its trust floor — "trust surface equals the convert
  step's: pinned and verified"), and **REQ-020** (the structured `input_mode` family,
  here extended with `pattern-arg`). This spec is BUNDLE-010's "first-class /
  pluggable engines" finisher: the engine MODEL shipped, the engine DATA and the
  TRUST GATE did not.
- **SPEC-031** (pluggable engine dispatch) — built the `EngineBinding` table, the
  generic `dispatchPackEngines` / `runFindingsEngine` / `gatherEngineInputs`, the
  strict-SARIF contract, and the `input_mode` enum + `ParseInputMode` fail-loud. This
  spec MOVES the bindings out of `DefaultRegistry` into pack declaration and adds the
  allowlist + `pattern-arg`; it consumes (does not rewrite) the dispatch.
- **SPEC-034** (native toolchain engine cutover) — added the `golangci` / `go-build` /
  `go-test` built-in bindings, `CrashGuard`, `ProjectTarget`, the
  `isNativeSarifLintEngine` / `isNativeGoTestEngine` name special-cases this spec
  retires into declared flags, and the split-provisioning fail-loud the allowlist
  builds on.
- **BUNDLE-009** (stack-aware traceability) — the **consumer** of `pattern-arg`
  (REQ-004): its parameterized contract/absence queries pass a literal pattern as a
  flag value. Locked as a seam by BUNDLE-010 REQ-017; this spec ships the mode, not
  the rule packs.
- Code: pkg/pack/engine/binding.go (`EngineBinding`, `DefaultRegistry`, `InputMode` /
  `ParseInputMode`, `ScopeKind`, `EngineCategory`, `Provision`, `Registry.Lookup`);
  pkg/pack/manifest.go (`Manifest`, `Rule`, `validateEngine`); cmd/backstop/pack_gate.go
  (`dispatchPackEngines`, `runFindingsEngine` — the `splitCommand`→`RunStdout` site the
  trust gate guards, the comment already referencing `binding.StrictSarif`;
  `gatherEngineInputs`, `resolveEngineRegistry`); cmd/backstop/pack_gate_golint.go
  (`isNativeSarifLintEngine`, `requireLintSarifShape`); cmd/backstop/pack_gate_filemode.go
  (`isNativeGoTestEngine`, `fileModeTestTarget`); cmd/backstop/pack_gate_provision.go
  (`provisionEngines`, `engineToolName` — the per-pack engine walk the allowlist check
  rides); pkg/pack/validate_manifest.go (`ExpectedLayout`, `validateEngineFields` — the
  engine-name layout switch); pkg/check/manifest.go (`CheckType`, `CheckTypeSemgrep` →
  `CheckTypeFindings`); pkg/check/check.go (`passOrder`); pkg/pack/distribution/verify.go
  + lockfile.go (`VerifyLock` / `Lockfile` — the version-pin the allowlist rides);
  pkg/check/semgrep.go (`ConfigError` — the exit-2 fail-loud type).
