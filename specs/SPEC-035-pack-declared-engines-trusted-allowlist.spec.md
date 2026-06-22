---
title: "Pack Declared Engines Trusted Allowlist"
number: SPEC-035
created: "2026-06-20"
status: draft
schema_version: spec/v1
spec_version: 1.1.1

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
    Open Question, NOT pre-decided. The allowlist gates the tool COMMAND only; the
    two other pack-supplied executables that reach execution — the pack `convert`
    script and the sandbox `validator` — are arbitrary pack code that CANNOT be
    tool-allowlisted, are deliberately EXEMPT from the tool-allowlist, and rest their
    trust SOLELY on the sandbox (`SandboxedRun`/`SandboxedRunStdout`). That residual,
    platform-conditional trust boundary (the sandbox is macOS `sandbox-exec`, a Linux
    no-op per BUNDLE-010 Non-Goal 10) is stated LOUDLY as a requirement, not silently
    bypassed.
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
      pack-declared engine TOOL command — at the point `runFindingsEngine` would
      `splitCommand`(binding.Command) → `RunStdout` it — the engine's `tool` (the
      command's executable name) MUST be present on the allowlist AND its required
      version MUST be lock-pinned (verified through the existing
      backstop.lock / VerifyLock / Provision machinery). A tool that is NOT on the
      allowlist, OR is on the allowlist but not lock-pinned to its required version,
      MUST produce a LOUD config error (a *check.ConfigError, exit 2) naming the tool
      and the pack, and backstop MUST NOT run the command. There is no silent run, no
      silent skip, and no "run it anyway." This closes the arbitrary-command-execution
      hole that opens the instant packs can declare commands (today
      `runFindingsEngine` runs `splitCommand` → `RunStdout` with NO trust check). The
      allowlist gate covers the TOOL command only; the pack `convert` script and the
      sandbox `validator` are arbitrary pack code governed by REQ-008's sandbox
      posture, not this tool-allowlist.

      This allowlist's `{tool → pinned version}` map is the REPLACEMENT mechanism for
      per-tool version pinning: the old single-tool `SemgrepVersion`
      (pkg/config/config.go) → `opts.PinnedSemgrepVersion` →
      `semgrepExecutor.pinnedVersion` (pkg/check/registry.go) plumbing is superseded by
      this general allowlist pin. That old config + plumbing is NOT removed by this spec —
      it dies with the in-process `semgrepExecutor` that ISSUE-018 deletes (it has no
      other consumer). This spec adds the allowlist as the going-forward pin; ISSUE-018
      removes the superseded `SemgrepVersion`/`PinnedSemgrepVersion` path. See the Sharp
      Edges sequencing note.
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
      tool-allowlist check must hold at EVERY non-test caller of `resolveEngineRegistry`
      so an un-allowlisted tool cannot reach execution by any path: validation time
      (`validateEngine`, here), the EARLIEST tool-resolution walk
      (`provisionEngines`, cmd/backstop/pack_gate_provision.go — the natural
      chokepoint, where the allowlist check is routed so an un-allowlisted tool
      fails loud BEFORE provisioning), and dispatch time (`runFindingsEngine`,
      REQ-002). The pack-separation reader (`pack_separation.go`) only looks up a
      binding's Category and never runs a command, so it needs no allowlist gate; that
      exemption is explicit, not an oversight.

      ADDITIONALLY, the engine FIELD-CONTRACT (the Requires/Forbids Rule-field lists)
      is baked engine knowledge of exactly the same shape as the hardcoded registry
      this spec moves to pack declaration, and must fold into the SAME pack-declared
      binding model. `engine.DefaultFieldContracts()`
      (pkg/pack/engine/fieldcontract.go) returns a `map[string]FieldContract` keyed by
      engine NAME — and `engineFieldClaim` (pkg/pack/validate_manifest.go, the
      `"semgrep|rule_path|requires"` → CLM-007 / `"semgrep|standard|requires"` →
      CLM-008 / etc. map consulted by `claimFor`) is a SECOND name-keyed map keyed on
      `"semgrep"` / `"ast-grep"` / `"sandbox"` / `"config-file"`. Both are contracts
      keyed by engine name, structurally identical to `DefaultRegistry()`. This spec
      makes the FieldContract (its Requires/Forbids field lists) a DECLARED property of
      the pack-declared `EngineBinding` — an engine's field-contract is parsed from the
      pack `engines:` block alongside its Command/InputMode/GateType, NOT looked up by
      engine name in a baked map. The validator verifies a rule's populated fields
      satisfy its declared engine's DECLARED field-contract (verify, not guide). The
      backstop-owned-DEFAULT disposition of `DefaultFieldContracts()` AND the
      name-keyed CLM-id mapping `engineFieldClaim` falls under the SAME fallback-vs-
      eradicate Open Question as `DefaultRegistry()` (OQ-1): under option (i) they
      survive as an incremental fallback the pack overrides; under option (ii) they are
      eradicated and shipped as default-pack data. They are NOT left as an unscoped
      baked map — they move with the registry under one resolution.
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
      `CheckTypeSemgrep` (pkg/check) must be RENAMED to a neutral type (e.g.
      `CheckTypeFindings`) so no gate-type identifier — AND no gate-type STRING
      surface — names a specific tool. The rename's real non-test footprint is ~11
      references across five files and INCLUDES the serialized string surface: the
      identifier sites (pkg/check/check.go passOrder / the `delete(opts.Executors,
      CheckTypeSemgrep)` site / the two `PassResult{Pass: CheckTypeSemgrep}` sites;
      pkg/check/manifest.go the const decl, the `String()` case, the `parseCheckType`
      case, and the three `[]CheckType{...CheckTypeSemgrep}` sites; pkg/check/parsers.go
      the stamp comment + `parser(out, CheckTypeSemgrep)`; pkg/check/registry.go the
      `execs[CheckTypeSemgrep]` assignment; cmd/backstop/code_check.go the two
      `Pass: check.CheckTypeSemgrep` sites), and CRUCIALLY the tool-named STRINGS
      `String()` returning the literal "semgrep" (pkg/check/manifest.go) and
      `parseCheckType` accepting the literal "semgrep" — both must change to the
      neutral "findings" so no serialized/config value names a tool. This rename
      touches serialized/config surfaces and the tests that assert them; those tests
      migrate with it. An engine's declared gate-type is data on the binding, parsed
      from the pack `engines:` block; an unrecognized gate-type value is a blocking
      config error (fail-loud, parallel to ParseInputMode).

      SEQUENCING (ISSUE-018 lands FIRST): two of the sites enumerated above — the
      `delete(opts.Executors, CheckTypeSemgrep)` site (pkg/check/check.go) and the
      `execs[CheckTypeSemgrep] = &semgrepExecutor{...}` assignment (pkg/check/registry.go)
      — are INSIDE the in-process `semgrepExecutor` that ISSUE-018 DELETES. ISSUE-018 is
      sequenced ahead of this spec; after it lands those two sites are GONE, so this
      spec's rename applies only to the `CheckTypeSemgrep` references that REMAIN
      (the const decl, `String()`/`parseCheckType`, the `[]CheckType` slices, `passOrder`,
      the parsers.go stamp, the code_check.go `PassResult` sites). The ~11-site count is
      the CURRENT full grep; the post-ISSUE-018 rename footprint is smaller. This is an
      ORDERING dependency, not a conflict — see the Sharp Edges sequencing note.
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
      The seven built-in engine bindings currently hardcoded in
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
  - id: REQ-008
    text: >
      The SECURITY POSTURE of the two OTHER pack-supplied executables that reach
      execution must be stated EXPLICITLY and LOUDLY, not left as a silent bypass of
      the tool-allowlist. The pack `convert` script (binding.Convert, run via
      `resolveSandboxedRunStdout` in `runFindingsEngine`) and the sandbox `validator`
      (rule.Validator, run via `resolveSandboxedRun` in `runSandboxEngine`) are
      ARBITRARY PACK CODE — pack-authored scripts, not named tools — so they CANNOT be
      tool-allowlisted and are DELIBERATELY EXEMPT from the REQ-002 tool-allowlist.
      Their trust rests SOLELY on the sandbox (`SandboxedRun` / `SandboxedRunStdout`,
      pkg/packval). This reconciles BUNDLE-010 REQ-019 ("trust surface equals the
      convert step's: pinned and verified"): the TOOL command is trusted by the
      ALLOWLIST + lock-pin (REQ-002), and the convert/validator scripts are trusted by
      the SANDBOX — together these two mechanisms ARE the trust surface, and a
      pack-declared command cannot escape both. The RESIDUAL gap must be recorded: the
      sandbox is platform-conditional — macOS `sandbox-exec`, a Linux NO-OP until the
      deferred sandbox-portability work (BUNDLE-010 Non-Goal 10) — so on a
      non-sandboxing platform the convert/validator scripts run with NO confinement.
      This is a KNOWN, STATED residual trust boundary documented in Sharp Edges, NOT a
      silent bypass; the spec must not over-claim the convert/validator paths as
      gated when they are sandbox-trusted.
    supports: pluggable-pack-engines:REQ-019
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
  - id: CLM-029
    requirement: REQ-002
    text: The allowlist's pinned-version comparison reads the locked version through the existing backstop.lock/VerifyLock path, not from a literal baked into TrustedToolAllowlist — a tool allowlisted at vX whose lock pins vY fails loud, proving the pin rides the lock and cannot drift from a second source
    tests:
      - TestAllowlist_VersionPinReadsFromLockNotSecondSource

  # REQ-003 — validateEngine + provisionEngines + dispatch: every resolveEngineRegistry caller gated
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
  - id: CLM-030
    requirement: REQ-003
    text: provisionEngines (the earliest tool-resolution walk, the second resolveEngineRegistry caller) fails loud on an un-allowlisted tool BEFORE provisioning, so an un-allowlisted tool is rejected at the earliest chokepoint as well as at validate and dispatch
    tests:
      - TestProvisionEngines_UnallowlistedToolFailsLoudBeforeProvision
  - id: CLM-031
    requirement: REQ-003
    text: The pack-separation reader (pack_separation.go, the third resolveEngineRegistry caller) looks up only a binding's Category and runs no command, so it is explicitly exempt from the allowlist gate — an un-allowlisted engine there is not an execution path
    tests:
      - TestPackSeparation_ReaderRunsNoCommandNoAllowlistNeeded
  - id: CLM-036
    requirement: REQ-003
    text: An engine's field-contract (Requires/Forbids Rule-field lists) is a DECLARED property of the pack-declared EngineBinding parsed from the pack engines block — a pack-declared engine carries its own field-contract and the validator verifies a rule's fields against the DECLARED contract, NOT against a map keyed by engine name
    tests:
      - TestFieldContract_DeclaredOnBindingNotNameKeyed
  - id: CLM-037
    requirement: REQ-003
    text: DefaultFieldContracts() and the name-keyed engineFieldClaim CLM-id map (keyed on "semgrep"/"ast-grep"/"sandbox"/"config-file") fall under the SAME OQ-1 fallback-vs-eradicate disposition as DefaultRegistry — under the incremental-fallback option a pack-declared field-contract overrides the baked default for the same engine name; they are not an independent unscoped baked map
    tests:
      - TestFieldContract_DefaultsFollowRegistryDisposition

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
    text: The tool-named CheckTypeSemgrep is renamed to the neutral CheckTypeFindings across all ~11 non-test sites in the five files; no gate-type IDENTIFIER names a tool and the build compiles with zero references to the old identifier
    tests:
      - TestCheckType_SemgrepRenamedToNeutralFindings
  - id: CLM-032
    requirement: REQ-005
    text: The gate-type STRING surface is neutralized too — CheckType.String() returns "findings" (not the literal "semgrep") and parseCheckType accepts "findings" (not "semgrep"), so no serialized/config gate-type value names a tool
    tests:
      - TestCheckType_StringAndParseUseNeutralFindingsNotSemgrep

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

  # REQ-008 — convert/validator sandbox posture: tool-allowlist-EXEMPT, sandbox-trusted, platform-conditional
  - id: CLM-033
    requirement: REQ-008
    text: The pack convert script (binding.Convert) runs via the sandbox (SandboxedRunStdout) and is NOT subjected to the tool-allowlist — a non-SARIF findings engine whose tool is allowlisted runs its convert script through the sandbox without an allowlist check on the script
    tests:
      - TestConvert_ScriptIsSandboxTrustedNotToolAllowlisted
  - id: CLM-034
    requirement: REQ-008
    text: The sandbox validator (rule.Validator) runs via the sandbox (SandboxedRun) and is NOT subjected to the tool-allowlist — it is arbitrary pack code trusted by the sandbox, not a named tool
    tests:
      - TestValidator_IsSandboxTrustedNotToolAllowlisted
  - id: CLM-035
    requirement: REQ-008
    text: On a platform where the sandbox is unavailable (non-macOS, sandbox-exec absent), the convert/validator path surfaces the platform limitation loudly rather than silently running unconfined or silently passing — the residual trust boundary is stated, not bypassed
    tests:
      - TestSandbox_PlatformUnavailableSurfacedNotSilent

contracts:
  - file: pkg/pack/engine/binding.go
    provides:
      - name: InputModePatternArg
        kind: constant
        signature: "const InputModePatternArg InputMode = \"pattern-arg\""
        notes: "Fifth InputMode value (REQ-004/CLM-014). ParseInputMode adds it to the accepted switch and keeps failing loud on any other value (CLM-015). gatherEngineInputs emits [InputFlag, pattern] for this mode (CLM-016) — no filesystem rule-path resolution (CLM-018)."
      - name: EngineBinding
        kind: type
        signature: "type EngineBinding struct { Command string; InputMode InputMode; InputFlag string; ScopeKind ScopeKind; Convert string; Provision *Provision; CrashGuard bool; Category EngineCategory; ProjectTarget string; GateType GateType; StrictSarif bool; PackageScoped bool; FieldContract FieldContract }"
        notes: "EXTENDED, not rewritten (constraint: do not churn the struct). Adds GateType (REQ-005), StrictSarif (REQ-006a — declared output-contract flag replacing the isNativeSarifLintEngine name-sniff; the pack_gate.go comment already references binding.StrictSarif), PackageScoped (REQ-006b — declared file-mode package-scope capability replacing the isNativeGoTestEngine name-sniff), and FieldContract (REQ-003/CLM-036 — the engine's declared Requires/Forbids Rule-field lists, folding DefaultFieldContracts off a name-keyed map onto the binding). All existing fields keep their meaning. yaml tags are added to every field so the pack engines block parses into this struct (REQ-001/CLM-001)."
      - name: ParseInputMode
        kind: function
        signature: "func ParseInputMode(s string) (InputMode, error)"
        notes: "Accepts pattern-arg as the fifth value; unchanged fail-loud contract on unknown values (REQ-004/CLM-014/CLM-015)."
    consumes: []
  - file: pkg/pack/engine/gatetype.go
    provides:
      - name: GateType
        kind: type
        signature: "type GateType int"
        notes: "Backstop-owned, tool-NEUTRAL gate-type enum (REQ-005). Exactly seven values: lint, build, test, findings, coverage, substantiveness, contracts (CLM-019). Defined in pkg/pack/engine so it carries no pkg/check import (leaf-package placement, mirrors EngineCategory/ScopeKind). An unrecognized declared gate_type fails loud (CLM-021)."
      - name: ParseGateType
        kind: function
        signature: "func ParseGateType(s string) (GateType, error)"
        notes: "Fail-loud parser for the declared gate_type (CLM-021), mirroring ParseInputMode — no silent default."
    consumes: []
  - file: pkg/pack/engine/fieldcontract.go
    provides:
      - name: FieldContract
        kind: type
        signature: "type FieldContract struct { Requires []string; Forbids []string }"
        notes: "The engine's declared field-contract (REQ-003/CLM-036). Becomes a DECLARED property of the pack-declared EngineBinding — its Requires/Forbids Rule-field lists are parsed from the pack engines block, NOT looked up by engine name. DefaultFieldContracts() (pkg/pack/engine/fieldcontract.go), the existing name-keyed map of these, falls under the same OQ-1 fallback-vs-eradicate disposition as DefaultRegistry (CLM-037)."
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
        notes: "Returns a non-nil error (wrapped by callers into a *check.ConfigError) when tool is not on the allowlist OR lockedVersion does not match the allowlist's pinned version (REQ-002/CLM-006/CLM-007). The lockedVersion argument is read by callers from the existing backstop.lock / VerifyLock path, NOT from a literal in TrustedToolAllowlist, so the pin cannot drift from a second source (CLM-029). Pure function so all three resolveEngineRegistry callers that run a command — validateEngine (REQ-003), provisionEngines (CLM-030), and the dispatch gate (REQ-002) — call the SAME check."
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
        notes: "Gains the trust gate (REQ-002): BEFORE splitCommand(binding.Command) -> runner.RunStdout, it resolves the engine's tool from the command and calls CheckToolAllowed against the allowlist + the lock-pinned version; a failure returns a *check.ConfigError naming tool+pack and the command is never run (CLM-005..008). The generic gather/run/convert/parseSarif flow is otherwise unchanged. The pack convert step (binding.Convert via resolveSandboxedRunStdout) is NOT tool-allowlisted — it is arbitrary pack code trusted by the sandbox (REQ-008/CLM-033); the trust gate covers the TOOL command only. The strict-SARIF guard now keys off binding.StrictSarif (REQ-006a)."
      - name: runSandboxEngine
        kind: function
        signature: "func runSandboxEngine(manifest *pack.Manifest, packRoot, projectRoot string, rules []pack.Rule) ([]gate.Violation, error)"
        notes: "UNCHANGED by the trust gate: the sandbox validator (rule.Validator via resolveSandboxedRun) is arbitrary pack code trusted by the sandbox, NOT a named tool, so it is deliberately exempt from the tool-allowlist (REQ-008/CLM-034). Its trust rests solely on SandboxedRun; the platform-conditional sandbox is the residual boundary (CLM-035)."
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

  - file: cmd/backstop/pack_gate_provision.go
    provides:
      - name: provisionEngines
        kind: function
        signature: "func provisionEngines(packs []*pack.Manifest) error"
        notes: "The EARLIEST tool-resolution walk (it already iterates each pack's engines via resolveEngineRegistry). The allowlist check is routed HERE so an un-allowlisted tool fails loud with a *check.ConfigError BEFORE provisioning (REQ-003/CLM-030), the natural chokepoint ahead of validate+dispatch. provisionEngines already resolves each engine's tool name via engineToolName(binding.Command), so it has the tool in hand for the CheckToolAllowed call."
    consumes:
      - source: pkg/pack/engine/allowlist.go
        name: TrustedToolAllowlist
        kind: function
      - source: pkg/pack/engine/allowlist.go
        name: CheckToolAllowed
        kind: function

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
      - name: claimFor
        kind: function
        signature: "func claimFor(engineName, field, kind string) string"
        notes: "Resolves the field-contract CLM-id via the engineFieldClaim map keyed on engine name (\"semgrep|rule_path|requires\" => CLM-007, etc.). Under REQ-003/CLM-037 this name-keyed map folds onto the pack-declared binding's FieldContract under the SAME OQ-1 fallback-vs-eradicate disposition as DefaultRegistry — it is NOT an independent baked map. A pack-declared engine's field-contract drives validation from the binding, not this name-keyed lookup."
    consumes:
      - source: pkg/pack/engine/binding.go
        name: InputMode
        kind: type
      - source: pkg/pack/engine/binding.go
        name: FieldContract
        kind: type

  - file: pkg/check/manifest.go
    provides:
      - name: CheckTypeFindings
        kind: constant
        signature: "const CheckTypeFindings CheckType = ..."
        notes: "RENAME of the tool-named CheckTypeSemgrep to a neutral type (REQ-005/CLM-022). The ~11 non-test identifier sites across pkg/check (manifest.go: const decl, String() case, parseCheckType case, three []CheckType slices; check.go: passOrder, the delete(opts.Executors,...) site, two PassResult{Pass:...} sites; parsers.go: the stamp comment + parser(out,...) call; registry.go: the execs[...] assignment) and cmd/backstop/code_check.go (two Pass: check.CheckType... sites) all update. CRUCIALLY the gate-type STRING surface neutralizes too: String() returns \"findings\" (was the literal \"semgrep\", manifest.go) and parseCheckType accepts \"findings\" (was \"semgrep\") — no serialized/config value names a tool (CLM-032). The rename touches serialized/config surfaces + their tests, which migrate with it."
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
   (binding.go) carries seven built-in bindings — their commands and pinned
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
- **DefaultRegistry migration** (REQ-007) — move the seven built-ins toward pack
  declaration; fallback-vs-eradicate left as an **Open Question** below.
- **Convert/validator sandbox posture** (REQ-008, security completeness) — the
  allowlist gates the TOOL command only; the pack `convert` script and the sandbox
  `validator` are arbitrary pack code, deliberately EXEMPT from the tool-allowlist
  and trusted SOLELY by the sandbox. The tool-allowlist (REQ-002) and the sandbox
  TOGETHER form BUNDLE-010 REQ-019's trust surface; the residual, platform-conditional
  gap (the sandbox is a Linux no-op) is stated loudly, not silently bypassed.

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

### `resolveEngineRegistry` callers and the allowlist gate (REQ-003)

The tool-allowlist must hold at every non-test caller that leads to running a command.

| Caller | Runs a command? | Allowlist gate |
|---|---|---|
| `validateEngine` (pkg/pack/manifest.go) | no (validation) | YES — reject un-allowlisted (CLM-013) |
| `provisionEngines` (pack_gate_provision.go) | leads to it; earliest walk | YES — fail loud BEFORE provisioning (CLM-030) |
| `runFindingsEngine` (pack_gate.go) | yes (dispatch) | YES — gate before RunStdout (CLM-008) |
| `pack_separation.go` | no (reads Category only) | EXEMPT — no execution path (CLM-031) |

### Engine field-contract folds onto the binding (REQ-003)

REQ-003 also folds the engine FIELD-CONTRACT off its name-keyed baked maps onto the
pack-declared binding. The `FieldContract` (Requires/Forbids Rule-field lists) becomes a
DECLARED property of the `EngineBinding`; the validator verifies against the DECLARED
contract, not a name-keyed lookup. Both name-keyed maps travel under OQ-1 with
`DefaultRegistry`.

| Baked name-keyed map | Location | Keyed on | Disposition |
|---|---|---|---|
| `DefaultFieldContracts()` | pkg/pack/engine/fieldcontract.go | engine name (`"semgrep"`, …) | declared on binding; default follows OQ-1 (CLM-036/CLM-037) |
| `engineFieldClaim` (via `claimFor`) | pkg/pack/validate_manifest.go | `"semgrep\|rule_path\|requires"` → CLM-007, … | follows OQ-1 with `DefaultRegistry` (CLM-037) |

### Pack-supplied executables and their trust mechanism (REQ-002, REQ-008)

Three pack-supplied executables can reach execution; each has exactly one trust
mechanism. The tool-allowlist and the sandbox TOGETHER are BUNDLE-010 REQ-019's trust
surface.

| Executable | Source | Trust mechanism | Allowlist-gated? |
|---|---|---|---|
| tool command | `binding.Command` | trusted-tool allowlist + lock-pin (REQ-002) | YES |
| pack `convert` script | `binding.Convert` | sandbox (`SandboxedRunStdout`) (REQ-008) | NO — exempt (CLM-033) |
| sandbox `validator` | `rule.Validator` | sandbox (`SandboxedRun`) (REQ-008) | NO — exempt (CLM-034) |

Residual gap: the sandbox is macOS `sandbox-exec`, a Linux no-op until the deferred
sandbox-portability work (BUNDLE-010 Non-Goal 10), so the convert/validator scripts run
unconfined on a non-sandboxing platform — surfaced loudly, not silently (CLM-035).

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
`CheckToolAllowed` against the allowlist and the lock-pinned version. The
`lockedVersion` argument MUST be read from the existing backstop.lock / `VerifyLock` /
`Provision` path — NOT from a literal baked into `TrustedToolAllowlist` — so the pin
cannot drift from a second source (CLM-029). A failure returns a `*check.ConfigError`
(exit 2) naming the tool and pack; the command is NEVER handed to the runner
(CLM-005..008). The sandbox engine (no command) is exempt (CLM-009). The version pin
RIDES the existing lock machinery — no new trust model is invented. This gate covers
the TOOL command only; the convert/validator scripts are governed by stage 8.

### 3. Allowlist at every `resolveEngineRegistry` caller (REQ-003)

`resolveEngineRegistry` has three non-test callers; the tool-allowlist check must hold
at every one that leads to running a command, so no path reaches execution
un-allowlisted:

- **`validateEngine`** (pkg/pack/manifest.go) — takes the manifest's declared engine
  bindings and checks the rule's engine against the pack-declared registry UNION the
  fallback registry, AND that the engine's tool is allowlisted. Unknown engine →
  blocking config error (existing). Known engine, un-allowlisted tool → blocking config
  error (new) (CLM-010..013).
- **`provisionEngines`** (cmd/backstop/pack_gate_provision.go) — the EARLIEST
  tool-resolution walk: it already iterates each pack's engines and resolves the tool
  name via `engineToolName(binding.Command)`. Route the `CheckToolAllowed` call here so
  an un-allowlisted tool fails loud BEFORE provisioning — the natural chokepoint ahead
  of dispatch (CLM-030).
- **`pack_separation.go`** — looks up only a binding's `Category` and runs NO command,
  so it is explicitly EXEMPT; an un-allowlisted engine there is not an execution path
  (CLM-031).

All callers that run a command call the SAME `CheckToolAllowed`.

This stage also FOLDS the engine field-contract off its name-keyed baked maps and onto
the pack-declared binding (REQ-003/CLM-036). The `FieldContract` (Requires/Forbids
Rule-field lists) becomes a DECLARED property of the `EngineBinding`, parsed from the
pack `engines:` block; the validator verifies a rule's populated fields against its
declared engine's DECLARED contract rather than looking the contract up by engine name.
`engine.DefaultFieldContracts()` (pkg/pack/engine/fieldcontract.go) and the name-keyed
CLM-id map `engineFieldClaim` (pkg/pack/validate_manifest.go, consulted by `claimFor`,
keyed `"semgrep|rule_path|requires"` → CLM-007, etc.) are exactly the same shape of
baked engine knowledge as `DefaultRegistry()`, so they travel under the SAME OQ-1
fallback-vs-eradicate disposition (CLM-037) — under option (i) a pack-declared
field-contract overrides the baked default for the same engine name; under option (ii)
they are eradicated into default-pack data. They are NOT left as an unscoped baked map.

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
pack `engines:` block. Rename the tool-named `CheckTypeSemgrep` to `CheckTypeFindings`
across its ~11 non-test sites in five files (pkg/check/manifest.go, check.go,
parsers.go, registry.go; cmd/backstop/code_check.go). The rename is NOT just the
identifier: the gate-type STRING surface must neutralize too — `CheckType.String()`
returns `"findings"` (was the literal `"semgrep"`) and `parseCheckType` accepts
`"findings"` (was `"semgrep"`), so no serialized/config value names a tool (CLM-032).
This touches serialized/config surfaces and the tests asserting them, which migrate
with the rename (CLM-019..022, CLM-032).

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

Move the seven built-in bindings (commands + pinned versions) toward pack
declaration. The disposition (incremental fallback vs full eradication) is an OPEN
QUESTION below and is NOT pre-decided here. Whichever option is chosen, the stage-1
merge semantics and the stage-2 allowlist gate hold identically: the built-ins'
tools are subject to the SAME allowlist as any pack-declared tool (CLM-027/CLM-028).

### 8. Convert/validator sandbox posture (REQ-008) — security completeness

State the trust posture of the two OTHER pack-supplied executables LOUDLY. The pack
`convert` script (binding.Convert, run via `resolveSandboxedRunStdout` in
`runFindingsEngine`) and the sandbox `validator` (rule.Validator, run via
`resolveSandboxedRun` in `runSandboxEngine`) are ARBITRARY PACK CODE — not named tools
— so they CANNOT be tool-allowlisted and are DELIBERATELY EXEMPT from the stage-2
tool-allowlist; their trust rests SOLELY on the sandbox (CLM-033/CLM-034). This is how
the spec reconciles BUNDLE-010 REQ-019: the TOOL command is trusted by the allowlist +
lock-pin, the convert/validator scripts are trusted by the sandbox, and TOGETHER they
are the trust surface — a pack-declared command escapes neither. The RESIDUAL gap is
recorded: the sandbox is `macOS sandbox-exec`, a Linux NO-OP until the deferred
sandbox-portability work (BUNDLE-010 Non-Goal 10), so on a non-sandboxing platform the
convert/validator scripts run unconfined; the implementation must surface that platform
limitation LOUDLY rather than silently run unconfined or silently pass (CLM-035). No
code change is mandated to the sandbox itself here — the deliverable is the stated,
tested posture plus the loud platform-limitation surface; the spec must not over-claim
the convert/validator paths as allowlist-gated.

### Explicitly OUT OF SCOPE

- **The compiled-standards-manifest semgrep ROUTING is NOT this spec's concern.**
  `pkg/check/manifest.go`'s `hasSemgrepSignal()` / `deriveRules()` / `compiledManifestFile`
  is the legacy compiled-standards-manifest RUNTIME routing — the
  `.standard.md` → compiler → manifest native-standards path slated for removal under the
  packs-only directive. It is a DIFFERENT subsystem from this spec's pack-declared engine
  bindings, and it belongs to the native-standards-path eradication (the ISSUE-018 family
  / standards removal), NOT SPEC-035. This spec does not touch, gate, or over-claim that
  routing.

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
    that a pack `engines:` block OVERRIDES/EXTENDS via the stage-1 merge. The seven
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

  **OQ-1 covers ALL the name-keyed baked engine maps, not just `DefaultRegistry()`.**
  The engine FIELD-CONTRACT defaults are the same shape of baked engine knowledge and
  resolve under the SAME option (REQ-003/CLM-037): `engine.DefaultFieldContracts()`
  (pkg/pack/engine/fieldcontract.go, keyed `"semgrep"`/`"ast-grep"`/`"sandbox"`/
  `"config-file"`) and the name-keyed CLM-id map `engineFieldClaim`
  (pkg/pack/validate_manifest.go, keyed `"semgrep|rule_path|requires"` → CLM-007, etc.)
  travel with `DefaultRegistry`. Under option (i) they survive as the incremental
  fallback a pack-declared field-contract overrides; under option (ii) they are
  eradicated and shipped as default-pack data. They are NOT a separate decision and
  NOT left as an unscoped baked map.

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
   Reuse the lock path (REQ-002); do not invent a parallel pin. This is claim-backed:
   CLM-029 drives a tool allowlisted at vX whose lock pins vY and asserts a loud
   failure, proving the comparison reads the lock and not a literal.

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

8. **`CheckTypeSemgrep` rename is ~11 sites across five files AND a STRING surface.**
   The rename (REQ-005) is not a 3-site change. The real non-test footprint is ~11
   identifier references: pkg/check/check.go (`passOrder`, the `delete(opts.Executors,
   CheckTypeSemgrep)` site, two `PassResult{Pass: CheckTypeSemgrep}` sites);
   pkg/check/manifest.go (the const decl, the `String()` case, the `parseCheckType`
   case, three `[]CheckType{...}` slices); pkg/check/parsers.go (the stamp comment +
   `parser(out, CheckTypeSemgrep)`); pkg/check/registry.go (`execs[CheckTypeSemgrep]`);
   and cmd/backstop/code_check.go (two `Pass: check.CheckTypeSemgrep` sites).
   CRUCIALLY two of these are the tool-named STRING surface: `String()` returns the
   literal `"semgrep"` and `parseCheckType` accepts `"semgrep"` — serialized/config
   values that must become `"findings"`, or the gate-type still names a tool on the
   wire even after the identifier is renamed. A rename that fixes the identifier but
   leaves the strings (or misses a slice site) leaves a tool name alive or breaks the
   build; CLM-022 confirms no tool-named identifier survives and CLM-032 confirms the
   string surface is neutral. This rename touches serialized/config surfaces and their
   tests, which migrate with it.

9. **Merge ambiguity if a pack redeclares a built-in engine name.** A pack
   `engines:` block declaring `semgrep` with a different command collides with the
   built-in. The merge semantics (CLM-004) must be deterministic — the spec says the
   resolved binding is well-defined (pack-declared wins, or built-ins are
   reserved-and-rejected; either is fine, but it must be DECIDED and tested, not left
   to map-iteration order). This same knob is what OQ-1 turns.

10. **The convert/validator paths are the OTHER execution surface — allowlist-EXEMPT,
    sandbox-trusted, and platform-conditional (the residual trust boundary).** The
    tool-allowlist (REQ-002) gates the TOOL command, but TWO other pack-supplied
    executables reach execution: the pack `convert` script (binding.Convert via
    `resolveSandboxedRunStdout`, on every non-SARIF findings engine) and the sandbox
    `validator` (rule.Validator via `resolveSandboxedRun`). These are arbitrary pack
    code, NOT named tools, so they CANNOT be tool-allowlisted — they are deliberately
    EXEMPT (REQ-008/CLM-033/CLM-034). Their trust rests SOLELY on the sandbox. This is
    how the spec reconciles BUNDLE-010 REQ-019 ("trust surface equals the convert
    step's: pinned and verified"): the tool command is trusted by the allowlist+lock,
    the convert/validator scripts by the sandbox, and TOGETHER they are the trust
    surface. The RESIDUAL gap is real and must be STATED, not silently bypassed: the
    sandbox is macOS `sandbox-exec` and a Linux NO-OP until the deferred
    sandbox-portability work (BUNDLE-010 Non-Goal 10), so on a non-sandboxing platform
    the convert/validator scripts run UNCONFINED. The implementation must surface that
    loudly (CLM-035), and the spec must NOT over-claim these paths as allowlist-gated.
    The danger is asserting "every pack-supplied executable is allowlisted" — false;
    only the tool command is, and the rest is the sandbox's job.

11. **ISSUE-018 lands FIRST — REQ-005's rename surface is "post-ISSUE-018," not the
    current full grep.** REQ-005's `CheckTypeSemgrep` rename and ISSUE-018's in-process
    `semgrepExecutor` deletion both touch the SAME sites: `pkg/check/check.go`'s
    `delete(opts.Executors, CheckTypeSemgrep)` and `pkg/check/registry.go`'s
    `execs[CheckTypeSemgrep] = &semgrepExecutor{...}` are INSIDE the in-process executor
    ISSUE-018 deletes. This is an ORDERING dependency, not a conflict: **ISSUE-018 lands
    first and deletes that executor**, so this spec's REQ-005 rename then applies ONLY to
    the `CheckTypeSemgrep` references that REMAIN after that deletion (the const decl,
    `String()`/`parseCheckType` cases, the `[]CheckType` slices, `passOrder`, the
    parsers.go stamp, the cmd/backstop/code_check.go `PassResult` sites). The "~11
    non-test sites" enumeration in REQ-005 is the CURRENT full grep; after ISSUE-018
    deletes the in-process executor, the `delete(opts.Executors,...)` and
    `execs[CheckTypeSemgrep]` sites are GONE and the rename footprint shrinks
    accordingly. A planner must sequence ISSUE-018 ahead of this spec's rename and treat
    the executor-internal sites as already-removed, not re-rename them. Likewise (Sharp
    Edge note for REQ-002): the old `SemgrepVersion`/`PinnedSemgrepVersion` version-pin
    plumbing this spec's allowlist supersedes is deleted by ISSUE-018 with that same
    executor, NOT by this spec.

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

12. Are the pack `convert` script and the sandbox `validator` correctly treated as
    allowlist-EXEMPT, sandbox-trusted pack code (REQ-008/CLM-033/CLM-034) — NOT
    silently run through the tool-allowlist and NOT over-claimed as gated? And is the
    platform-conditional residual gap (sandbox is a Linux no-op) surfaced LOUDLY rather
    than silently unconfined (CLM-035)?

13. Is the tool-allowlist check routed through `provisionEngines` — the earliest
    `resolveEngineRegistry` caller — so an un-allowlisted tool fails loud BEFORE
    provisioning (CLM-030), in addition to validate and dispatch? And is the third
    caller (`pack_separation.go`) correctly exempt because it runs no command
    (CLM-031)?

14. Does the `CheckTypeSemgrep` → `CheckTypeFindings` rename neutralize the STRING
    surface — `String()` returning `"findings"` and `parseCheckType` accepting
    `"findings"`, not the literal `"semgrep"` (CLM-032) — and cover all ~11
    identifier sites across the five files, with the affected serialized-surface tests
    migrated?

15. Does the allowlist's pinned-version comparison genuinely read from backstop.lock /
    `VerifyLock` rather than a literal in `TrustedToolAllowlist`, proven by the
    allowlisted-at-vX / locked-at-vY fail-loud case (CLM-029)?

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
- **ISSUE-018** (in-process executor deletion) — the SEQUENCING dependency. ISSUE-018
  lands FIRST and DELETES the in-process `semgrepExecutor`. That deletion (a) removes the
  superseded `SemgrepVersion` (pkg/config/config.go) / `opts.PinnedSemgrepVersion` /
  `semgrepExecutor.pinnedVersion` (pkg/check/registry.go) version-pin plumbing that this
  spec's REQ-002 trusted-tool allowlist REPLACES — it is NOT removed by this spec; and
  (b) removes the two REQ-005 rename sites that live inside that executor
  (`delete(opts.Executors, CheckTypeSemgrep)` in pkg/check/check.go and
  `execs[CheckTypeSemgrep] = &semgrepExecutor{...}` in pkg/check/registry.go), so this
  spec's rename applies only to the references that REMAIN post-ISSUE-018. The
  compiled-standards-manifest semgrep ROUTING (`hasSemgrepSignal()` / `deriveRules()` /
  `compiledManifestFile`, pkg/check/manifest.go) is the legacy native-standards path and
  belongs to the ISSUE-018 family / standards removal — explicitly OUT OF SCOPE for this
  spec.
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
  pkg/check/semgrep.go (`ConfigError` — the exit-2 fail-loud type);
  pkg/pack/engine/fieldcontract.go (`FieldContract` / `DefaultFieldContracts` — the
  name-keyed engine field-contract folded onto the binding by REQ-003);
  pkg/pack/validate_manifest.go (`engineFieldClaim` / `claimFor` — the name-keyed
  CLM-id map that travels under OQ-1 with `DefaultRegistry`).

## Version History

- **1.1.1** (2026-06-22) — Contract file-path correction (Phase 7 verification). `GateType` and `ParseGateType` moved to their own `pkg/pack/engine/gatetype.go` contract entry, and `FieldContract` to a `pkg/pack/engine/fieldcontract.go` entry, matching the actual source layout; the file-scoped `contract_signature` extractor was reporting "symbol not found in binding.go" false-positives because these three symbols live in sibling files of the same package. No requirement, claim, or signature changed.
- **1.1.0** (2026-06-21) — Corrective pass closing the re-review FAIL. (1) REQ-003
  extended to fold the engine FIELD-CONTRACT (`DefaultFieldContracts()` +
  `engineFieldClaim`) onto the pack-declared `EngineBinding` under the SAME OQ-1
  fallback-vs-eradicate disposition as `DefaultRegistry()` — no longer an unscoped baked
  map (CLM-036/CLM-037 added). (2) REQ-002 notes its `{tool → pinned version}` allowlist
  REPLACES the per-tool `SemgrepVersion`/`PinnedSemgrepVersion` pin, which ISSUE-018 (not
  this spec) deletes with the in-process executor. (3) REQ-005 + a Sharp Edge state the
  ISSUE-018→SPEC-035 SEQUENCING dependency: ISSUE-018 lands first and deletes the two
  rename sites inside the in-process executor, so the rename footprint is "post-ISSUE-018."
  (4) Added an explicit OUT-OF-SCOPE exclusion for the compiled-standards-manifest
  semgrep routing (`hasSemgrepSignal`/`deriveRules`/`compiledManifestFile`), which belongs
  to the native-standards-path eradication. The pack-declared-engines + trusted-allowlist
  core (REQ-001/REQ-002) and the generic dispatch + `EngineBinding` struct are unchanged.
- **1.0.0** (2026-06-20) — Initial spec authored from BUNDLE-010 (pluggable-pack-engines).
