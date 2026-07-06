---
title: "Retire Language Toolchain Bridge"
number: SPEC-046
created: "2026-06-28"
status: implemented
schema_version: spec/v1
spec_version: 1.2.0

implementation:
  summary: >
    BUNDLE-012 Spec Seed 3 — Pillar B. Finishes the thin-executor mission on the
    gate's TOOLCHAIN-SELECTION spine by deleting the last place backstop bakes the
    false assumption "a project has ONE language." Three coupled moves: (1) DELETE
    the `language:`-derived toolchain BRIDGE (`cmd/backstop/gate.go`
    `loadBridgedToolchainPacks`, `toolchainPackName`, `gateLanguage`, and the
    auto-load call site ~L602). Toolchain packs become ORDINARY packs: loaded ONLY
    via the declared-pack path (`loadInstalledPacks` over `backstop.yml packs:`) and
    dispatched UNIFORMLY through `dispatchPackEngines` like every other pack — a
    polyglot repo simply declares more than one. (2) FULLY RETIRE the `language:`
    field: delete `Config.Language` from `pkg/config/config.go`, strip `language: go`
    from the dogfood `backstop.yml`, and update every compile-time reader. (3) REHOME
    the traceability classifier off `language:` and RESOLVE the bundle's SQ-1. The
    classifier's capability/polarity DECISIONS already key on installed-pack presence
    (SPEC-037/038/041) — `language:` only ever fed the cosmetic STACK LABEL in
    fail-loud messages (`pkg/gate/traceability_polarity.go` `stackLabel`; the
    coverage-arm fallback message in `deriveCapabilityState`). That label is
    re-derived from the SET of declared toolchain-pack NAMES, with the polyglot glob
    UNION read ONLY through SPEC-043's single `SourceClassifier` (reused, never forking a
    parallel classifier). SQ-1 RESOLUTION: a polyglot repo's stack label is the joined
    SET of its declared toolchain-pack names, NOT a per-pack or single-precedence read;
    there is no overlap-precedence winner (overlap for measurability is already governed
    by SPEC-043's test-wins rule); when no toolchain pack is declared the label is
    "unspecified", driven by a SINGLE authoritative signal — the empty declared
    toolchain-pack-NAME set (the label is name-derived) — never by an empty `language:`
    field. SPEC-043's `SourceClassifier.HasSourceGlobs()` is CORROBORATING only, NOT the
    authoritative label driver (it can diverge: a toolchain pack with no `classification`
    source globs has a non-empty pack-name set but `HasSourceGlobs()==false`). The
    stack label is threaded `buildGateSteps → wrapTraceabilityStep → deriveCapabilityState`
    (the wrapper gains a `stack` param; its three call sites pass the label). The dogfood
    MUST
    NOT regress: backstop-core already declares `backstop/go-toolchain` in `packs:`,
    so deleting the bridge keeps go-toolchain in its own gate via the declared-pack
    path. SHARED-FILE SEAM: `cmd/backstop/gate.go` is co-edited by SPEC-045
    (`goFilePackageMatchesTarget`), SPEC-043 (`buildCoverageStep` classifier param),
    and SPEC-044 (coverage records) — this seed's edits (delete the bridge + remove
    `bridged` from `coverageRecordsProducer`/`toolchainEnforcementStatus`/
    `countToolchainPacks`/the dispatch list/the early return) are disjoint from those
    but share the file; flagged for the cross-consistency pass. B2 RECONCILIATION: the
    deletion is SCOPED to the bridge auto-load + the `bridged` threading ONLY —
    SPEC-043's `mergeSourceClassifier`/`SourceClassifier` SURVIVE; SPEC-043 itself
    repoints them onto the DECLARED toolchain-pack set (`backstop.yml packs:`), NOT the
    deleted `bridged` set, and this seed only FENCES that plumbing (does not edit
    SPEC-043's call site), so no classifier call site is orphaned. The
    `coverageRecordsProducer` signature change (dropping `bridged`) composes with
    SPEC-043's classifier-param addition and SPEC-044's record-model change on the
    coverage call chain (orthogonal axes).
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/ ./pkg/gate/ ./pkg/config/ ./pkg/check/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      The `language:`-derived toolchain BRIDGE MUST be DELETED. Remove
      `loadBridgedToolchainPacks`, `toolchainPackName`, and `gateLanguage` from
      `cmd/backstop/gate.go`, and remove the bridge call site in `buildGateSteps`
      (`loadBridgedToolchainPacks(projectRoot, gateLanguage(projectRoot), packs)`
      ~L602 plus its `bridgeErr` config-error branch). After deletion a toolchain
      pack MAY be acquired ONLY via the DECLARED-pack path — `loadInstalledPacks`
      over `backstop.yml packs:` — and MUST be dispatched UNIFORMLY through the same
      `dispatchPackEngines`/`pack_engines` step every other declared pack uses. It is
      PROHIBITED for any code path to synthesize, derive, or auto-load a toolchain
      pack from a single language field or a name computed from one: a project that
      does NOT declare a toolchain pack in `packs:` MUST get ZERO toolchain packs (the
      bridge never invents one). The `bridged` manifest list is removed as an input
      to `coverageRecordsProducer`, the dispatch set
      (`excludeDedicatedStepRules(... bridged ... packs ...)`), and the
      `len(packs)==0 && len(bridged)==0` early return — each now keys on the declared
      `packs` alone. The deletion is SCOPED to the bridge auto-load and the `bridged`
      threading ONLY. SPEC-043's `mergeSourceClassifier`/`SourceClassifier` are NOT
      deleted and MUST NOT be orphaned: they SURVIVE the bridge deletion. SPEC-043 itself
      performs the repoint of those symbols onto the DECLARED-pack manifest set (the
      ordinary `backstop.yml packs:` set returned by `loadInstalledPacks`); this seed
      does NOT edit SPEC-043's classifier call site — it only FENCES the bridge-deletion
      edit-set so the declared-pack manifest set remains in scope after `bridged` is
      removed, so no `mergeSourceClassifier`/classifier call site is left dangling by the
      `bridged` removal. The EXISTING behavioral tests whose SUBJECT is the deleted
      bridge MUST be DELETED or REWRITTEN, not kept green via a shim: delete
      `cmd/backstop/gate_bridge_load_test.go` (exercises `loadBridgedToolchainPacks` /
      `gateLanguage`), delete `cmd/backstop/gate_bridge_agnostic_test.go` (asserts one
      bridge call resolves two languages), and delete or rewrite the
      `loadBridgedToolchainPacks` cases in `cmd/backstop/cutover_noregress_test.go`.
      After this seed it is PROHIBITED for ANY `_test.go` file in `cmd/backstop` to
      reference `loadBridgedToolchainPacks`, `gateLanguage`, or `toolchainPackName` — a
      surviving reference means the bridge was shimmed rather than deleted.
    supports: language-neutral-consumer-ts-toolchain:REQ-004
  - id: REQ-002
    text: >
      Deleting the bridge MUST NOT regress the dogfood gate and MUST support polyglot
      declaration. Because backstop-core declares `backstop/go-toolchain` in
      `backstop.yml packs:`, its gate MUST still dispatch go-toolchain via the
      declared-pack path with NO `language:` field present. The no-toolchain-pack
      WARN-only state (`toolchainEnforcementStatus` / `countToolchainPacks`) MUST be
      re-keyed to count ONLY declared toolchain packs — the `bridged` parameter is
      removed from both signatures — emitting the "enforcement not configured (0
      toolchain packs)" warning iff ZERO toolchain packs are DECLARED. A polyglot repo
      declaring two toolchain packs (e.g. `backstop/go-toolchain` and
      `backstop/bun-toolchain`) MUST dispatch BOTH through the uniform declared-pack
      path. `coverageRecordsProducer` MUST source records from the declared toolchain
      packs alone. The EXISTING `cmd/backstop/gate_no_toolchain_pack_test.go` MUST be
      REWRITTEN to the 1-argument `toolchainEnforcementStatus(declared)` signature — its
      `toolchainEnforcementStatus(bridged, declared)` two-argument call sites (the
      `bridged`-passing cases) are DROPPED — so the test no longer pins the removed
      `bridged` parameter.
    supports: language-neutral-consumer-ts-toolchain:REQ-004
  - id: REQ-003
    text: >
      The `language:` config field MUST be FULLY removed with NO remnant. Delete the
      `Language` field from `config.Config` in `pkg/config/config.go`; remove
      `language: go` from the dogfood `backstop.yml`; and update EVERY compile-time
      reader of `cfg.Language` — the `gateConfig` zero-value fallback (drop the
      `Language: "go"` seed), the `deriveCapabilityState` coverage-arm fallback message
      (de-language the string), the legacy `backstop code check` command's
      `check.Options{ ... Language: cfg.Language ... }` assignment in
      `cmd/backstop/code_check.go` (drop the config-sourced assignment; `check.Options`
      already has an empty-language default), and the `language: go` string literal in
      the `cmd/backstop/gate_substantiveness_e2e.go` test-config helper — plus every
      test fixture or `config.Config{Language: ...}` construction. It is PROHIBITED for
      any `cfg.Language` read or any `Config.Language` field to remain in
      `cmd/backstop` or `pkg/gate`. Because `backstop.yml` is parsed with non-strict
      YAML, a config file that still carries a `language:` key MUST parse cleanly
      (the unknown key is ignored, never an error) and MUST have NO effect on the gate
      verdict — the field is simply gone. The EXISTING tests that encode the retired
      single-language thesis MUST be updated, not preserved: the `cfg.Language == "go"`
      assertion at `pkg/config/config_test.go:26` MUST be DROPPED, and the
      `cmd/backstop/gate_capability_test.go` tests
      `TestCapabilityState_NonGoProject_DerivesAbsentClass2` and
      `TestCapabilityState_NonGoUndeclared_NeverAutoPromotes` (which assert the OLD
      language-keyed capability thesis — capability-absence derived from a non-Go
      `language:` field rather than installed-pack presence) MUST be DELETED or REWRITTEN
      onto installed-pack presence. The COLLATERAL `config.Config.Language` readers MUST
      ALSO be updated — but the COMPLETENESS CONTRACT for that sweep is NOT a
      hand-maintained file list (which has repeatedly proven to be whack-a-mole). It is the
      SOURCE GUARD `TestConfig_NoTestAssertsConfigLanguage` (NO `_test.go` in
      `cmd/backstop`, `pkg/config`, `pkg/gate`, OR `pkg/check` reads
      `config.Config.Language`) PLUS the `test_command` compiling and running those four
      packages — together they MECHANICALLY catch EVERY reader (a stale construction or
      read either trips the guard or breaks compilation), and those four packages are
      provably exhaustive because they are the only packages that read the field. The
      implementer MUST therefore fix WHATEVER the guard and `test_command` flag, NOT only
      the files named here; the enumeration below is ILLUSTRATIVE (the known readers at
      authoring time), not the task boundary. Known readers, by treatment:
      (a) MECHANICAL STRIPS — drop the inert `Language:` field, no behavioral change:
      `cmd/backstop/cutover_deletion_test.go`,
      `cmd/backstop/cutover_consistency_test.go`,
      `cmd/backstop/gate_capability_rekey_test.go`, and
      `cmd/backstop/gate_capability_contracts_rekey_test.go`.
      (b) `check.Options.Language`-BRIDGE FIXES — both `pkg/check/registry_test.go` and
      `pkg/check/ts_executor_test.go` build `Options{Language: cfg.Language, Config: cfg}`
      from a `config.Config`; each MUST STOP reading the deleted `config.Config.Language`
      and pass the language LITERAL directly into the SEPARATE, fenced
      `check.Options.Language` field, which SURVIVES.
      (c) REHOME-COUPLED BEHAVIORAL REWRITES — NOT token strips: drop the field AND
      re-source the rendered stack-label assertions from `CapabilityState.Stack` (since
      `language:` no longer drives the label), with capability/class verdicts driven by
      installed-pack presence/absence: `cmd/backstop/gate_wiring_test.go` (its
      `Language: "typescript"` fixtures DRIVE the class-2/class-3 capability-absent
      verdicts) and `pkg/gate/traceability_polarity_step_test.go` (its `cfgDeclaring`
      helper feeds `PolarityStepResult` message/stack assertions). After this seed it is
      PROHIBITED for ANY `_test.go` file in `cmd/backstop`,
      `pkg/config`, `pkg/gate`, or `pkg/check` to read `cfg.Language` /
      `config.Config.Language`. (NOTE: `pack.Manifest.Language` and
      `check.Options.Language` are SEPARATE fields on other structs, are NOT deleted, and
      MUST remain — the guard fences the look-alikes and trips ONLY on
      `config.Config.Language`.)
    supports: language-neutral-consumer-ts-toolchain:REQ-005
  - id: REQ-004
    text: >
      The traceability classifier MUST be rehomed off `language:` onto the
      pack-declared globs, resolving the bundle's SQ-1. The classifier's
      capability/polarity DECISIONS already key on installed-pack presence
      (SPEC-037/038/041) and MUST remain unchanged by the `language:` removal — the
      rehome touches ONLY the cosmetic STACK LABEL. `stackLabel` MUST stop reading
      `cfg.Language`; the label is instead derived from the SET of DECLARED toolchain
      packs and carried to `PolarityStepResult` via a new `CapabilityState.Stack`
      field stamped in `cmd/backstop`. SQ-1 RESOLUTION (precedence is explicit): the
      label is the joined SET of the DECLARED toolchain-pack NAMES, with NO single-pack
      precedence and NO overlap winner — a polyglot repo's label names every declared
      stack (measurability overlap is already governed by SPEC-043's test-wins-on-overlap
      rule for the source/test glob UNION, not re-decided here). The label's empty
      fallback MUST be driven by a SINGLE authoritative signal: the SET of DECLARED
      toolchain-pack NAMES — the same set `declaredToolchainStackLabel` derives the label
      from. An empty declared-toolchain-pack-name set ⇒ "unspecified"; a non-empty set ⇒
      the joined names; never an empty `language:` field. SPEC-043's
      `SourceClassifier.HasSourceGlobs()` is NOT the authoritative driver of the label and
      MUST NOT be conflated with the pack-name set — the two can DIVERGE (a declared
      toolchain pack with no `classification` source globs has a NON-empty pack-name set
      but `HasSourceGlobs()==false`). `HasSourceGlobs` remains the source/measurability
      signal SPEC-043 owns and is at most CORROBORATING here, never authoritative for the
      label. It is PROHIBITED to introduce a second, parallel glob classifier in this
      seed: the polyglot glob UNION continues to be read ONLY through SPEC-043's single
      `SourceClassifier` (reused, never forked).
    supports: language-neutral-consumer-ts-toolchain:REQ-005

claims:
  # REQ-001 — delete the bridge; toolchain packs only via the declared path
  - id: CLM-001
    requirement: REQ-001
    text: The `loadBridgedToolchainPacks` symbol is DELETED from cmd/backstop/gate.go — a source guard over the file asserts it is absent, so a reintroduced language-derived bridge loader is caught as a regression
    tests:
      - TestGate_BridgeLoaderSymbolDeleted
  - id: CLM-002
    requirement: REQ-001
    text: The `toolchainPackName` deriver (the `"backstop/"+language+"-toolchain"` name function) is DELETED — a source guard asserts the symbol is absent, proving no pack name is computed from a language field
    tests:
      - TestGate_ToolchainPackNameDeriverDeleted
  - id: CLM-003
    requirement: REQ-001
    text: The `gateLanguage` reader is DELETED from cmd/backstop/gate.go — a source guard asserts the symbol is absent
    tests:
      - TestGate_GateLanguageReaderDeleted
  - id: CLM-004
    requirement: REQ-001
    text: A project that declares a toolchain pack ONLY in `packs:` (no `language:`) has that pack dispatched uniformly through the pack_engines/dispatchPackEngines step — proving toolchain packs flow through the ordinary declared-pack path
    tests:
      - TestGate_DeclaredToolchainPackDispatchedUniformly
  - id: CLM-005
    requirement: REQ-001
    text: A project that declares NO toolchain pack and has NO `language:` field gets ZERO toolchain packs and the WARN-only "enforcement not configured" state — the deleted bridge no longer synthesizes a pack from a derived name (prohibited-path negative)
    tests:
      - TestGate_NoDeclaredToolchainPackYieldsWarnNotSynthesized
  - id: CLM-006
    requirement: REQ-001
    text: The dispatch set, coverage producer, and zero-pack early return key on the declared `packs` alone — `coverageRecordsProducer`, `toolchainEnforcementStatus`, and the early-return guard no longer reference a `bridged` list (signature/source guard)
    tests:
      - TestGate_BridgedInputRemovedFromGateWiring
  - id: CLM-022
    requirement: REQ-001
    text: The bridge deletion does NOT orphan SPEC-043's classifier plumbing — `mergeSourceClassifier`/`SourceClassifier` SURVIVE (a source guard asserts both symbols are still present and still called), and the declared-pack manifest set (`loadInstalledPacks` over `backstop.yml packs:`) remains in scope at the classifier call site after `bridged` is removed, so the classifier sources from the DECLARED set and no call site is left dangling (no compile break, no orphaned reference)
    tests:
      - TestGate_ClassifierPlumbingSurvivesBridgeDeletion
  # REQ-002 — dogfood no-regress + polyglot + count keys on declared only
  - id: CLM-007
    requirement: REQ-002
    text: MANDATED — the gate still dispatches `backstop/go-toolchain` for backstop-core via the DECLARED-pack path with NO `language:` field present in backstop.yml, proving the dogfood does not regress after the bridge deletion
    tests:
      - TestGate_DispatchesGoToolchainViaDeclaredPackPathWithoutLanguageField
  - id: CLM-008
    requirement: REQ-002
    text: POLYGLOT — a repo declaring BOTH `backstop/go-toolchain` and `backstop/bun-toolchain` in `packs:` dispatches BOTH toolchain packs through the uniform declared-pack path (more than one toolchain pack per repo)
    tests:
      - TestGate_PolyglotDeclaredToolchainPacksBothDispatched
  - id: CLM-009
    requirement: REQ-002
    text: The `countToolchainPacks` helper counts DECLARED toolchain packs only — its signature no longer takes a `bridged` argument, and it returns the count of `packs:` entries whose normalized name ends in `-toolchain`
    tests:
      - TestGate_CountToolchainPacksCountsDeclaredOnly
  - id: CLM-010
    requirement: REQ-002
    text: The `toolchainEnforcementStatus` helper emits the WARN-only state iff ZERO toolchain packs are DECLARED (no `bridged` input) and is suppressed when at least one toolchain pack is declared
    tests:
      - TestGate_ToolchainEnforcementKeysOnDeclaredOnly
  # REQ-003 — fully remove the language: field
  - id: CLM-011
    requirement: REQ-003
    text: The `config.Config` struct has NO `Language` field — a struct/source guard asserts the field is gone, so any `cfg.Language` reader fails to compile (compile-time regression guard)
    tests:
      - TestConfig_LanguageFieldRemoved
  - id: CLM-012
    requirement: REQ-003
    text: MANDATED — a backstop.yml that still carries a `language:` key parses cleanly with NO error and the field is ignored (non-strict YAML), proving the field is fully gone rather than rejected
    tests:
      - TestConfig_LanguageKeyIgnoredCleanlyFieldRemoved
  - id: CLM-013
    requirement: REQ-003
    text: The dogfood `backstop.yml` carries NO `language:` key after the retirement
    tests:
      - TestDogfood_BackstopYmlHasNoLanguageKey
  - id: CLM-014
    requirement: REQ-003
    text: No `cfg.Language` reader remains anywhere in cmd/backstop or pkg/gate — a source guard over those packages asserts the de-language'd readers (gateConfig fallback, deriveCapabilityState message, code_check.go assignment, the e2e helper yml literal) are all updated
    tests:
      - TestGate_NoConfigLanguageReaderRemains
  - id: CLM-015
    requirement: REQ-003
    text: The gate verdict is IDENTICAL whether or not a `language:` key is present in backstop.yml — the field is inert, so adding/removing it changes nothing about the gate result
    tests:
      - TestGate_LanguageKeyPresenceDoesNotChangeVerdict
  # REQ-004 — rehome the traceability classifier (SQ-1)
  - id: CLM-016
    requirement: REQ-004
    text: The traceability STACK LABEL for a repo declaring `backstop/go-toolchain` is derived from the declared toolchain pack SET (the rendered fail-loud message names the "go" stack) with NO `language:` field present — proving the label is rehomed off `language:`
    tests:
      - TestPolarity_StackLabelFromDeclaredToolchainPacksNotLanguage
  - id: CLM-017
    requirement: REQ-004
    text: SQ-1 POLYGLOT — a repo declaring `backstop/go-toolchain` AND `backstop/bun-toolchain` yields a SET-valued stack label naming BOTH stacks (merged union), with NO single-pack precedence or overlap winner
    tests:
      - TestPolarity_PolyglotStackLabelIsUnionNoPrecedence
  - id: CLM-018
    requirement: REQ-004
    text: SQ-1 EMPTY — a repo declaring NO toolchain pack yields the "unspecified" stack label, driven by the SINGLE authoritative signal (the empty DECLARED toolchain-pack-NAME set that `declaredToolchainStackLabel` reads), NOT by an empty `language:` field and NOT by `SourceClassifier.HasSourceGlobs()` (which is corroborating only and can diverge from the pack-name set)
    tests:
      - TestPolarity_NoToolchainPackStackLabelUnspecified
  - id: CLM-019
    requirement: REQ-004
    text: SQ-1 NO-FORK — the rehome introduces NO parallel/forked glob classifier in this seed; SPEC-043's single `gate.SourceClassifier` remains the ONLY glob classifier for the polyglot source/test glob union (source guard asserts no new glob-classifier type is defined here)
    tests:
      - TestPolarity_RehomeIntroducesNoForkedGlobClassifier
  - id: CLM-020
    requirement: REQ-004
    text: Capability/polarity classification VERDICTS (ClassNone / ClassCapabilityAbsent / ClassBrokenDeclared) are UNCHANGED by the presence or absence of `language:` — the rehome touches only the label, not the decision, so removing `language:` does not alter which class a dimension lands in
    tests:
      - TestPolarity_ClassificationVerdictUnaffectedByLanguageRemoval
  - id: CLM-021
    requirement: REQ-004
    text: The `stackLabel` helper no longer reads `cfg.Language` — `PolarityStepResult` renders the declared-pack-derived stack from the new `CapabilityState.Stack` carrier (source guard asserts the `cfg.Language` read in stackLabel is gone)
    tests:
      - TestPolarity_StackLabelNoLongerReadsConfigLanguage
  # REQ-001 — existing bridge-subject tests deleted, not shimmed green
  - id: CLM-023
    requirement: REQ-001
    text: The behavioral tests whose SUBJECT is the deleted bridge are DELETED/REWRITTEN, not shimmed — `cmd/backstop/gate_bridge_load_test.go` and `cmd/backstop/gate_bridge_agnostic_test.go` are DELETED and the `loadBridgedToolchainPacks` cases in `cmd/backstop/cutover_noregress_test.go` are DELETED/REWRITTEN; a source guard asserts NO `_test.go` file in cmd/backstop references `loadBridgedToolchainPacks`, `gateLanguage`, or `toolchainPackName`, so the bridge cannot survive as a green-keeping shim
    tests:
      - TestGate_NoTestReferencesDeletedBridgeSymbols
  # REQ-002 — existing no-toolchain-pack test rewritten to the 1-arg signature
  - id: CLM-024
    requirement: REQ-002
    text: The existing `cmd/backstop/gate_no_toolchain_pack_test.go` is REWRITTEN to the 1-argument `toolchainEnforcementStatus(declared)` signature — its `toolchainEnforcementStatus(bridged, declared)` two-argument call sites are dropped, so no test pins the removed `bridged` parameter
    tests:
      - TestNoToolchainPack_EnforcementStatusRewrittenToDeclaredOnlyArg
  # REQ-003 — existing language-keyed tests/fixtures updated, not preserved
  - id: CLM-025
    requirement: REQ-003
    text: >
      The COMPLETENESS CONTRACT for the language-reader sweep is the SOURCE GUARD
      `TestConfig_NoTestAssertsConfigLanguage` (NO `_test.go` in cmd/backstop, pkg/config,
      pkg/gate, OR pkg/check reads `cfg.Language`/`config.Config.Language`) PLUS the
      `test_command` compiling/running those four packages — together they mechanically
      catch EVERY reader (those four packages are the only ones that read the field, so the
      guard scope is exhaustive), making the file list ILLUSTRATIVE (the known readers),
      NOT the task boundary; the implementer fixes WHATEVER the guard/test_command flags,
      not only the named files. Known readers, by treatment — mechanical STRIPS (drop the
      inert `Language` field): `pkg/config/config_test.go:26` (`cfg.Language == "go"`
      assertion DROPPED), `cmd/backstop/cutover_deletion_test.go`,
      `cmd/backstop/cutover_consistency_test.go`,
      `cmd/backstop/gate_capability_rekey_test.go`,
      `cmd/backstop/gate_capability_contracts_rekey_test.go`, plus
      `cmd/backstop/gate_capability_test.go`'s
      `TestCapabilityState_NonGoProject_DerivesAbsentClass2`/`TestCapabilityState_NonGoUndeclared_NeverAutoPromotes`
      (the language-keyed capability thesis) DELETED/REWRITTEN onto installed-pack
      presence; `check.Options.Language`-BRIDGE fixes — `pkg/check/registry_test.go` AND
      `pkg/check/ts_executor_test.go` STOP reading the deleted `config.Config.Language` and
      pass the language LITERAL directly into the SEPARATE, surviving
      `check.Options.Language`; REHOME-COUPLED behavioral REWRITES (drop the field AND
      re-source the rendered stack-label assertions from `CapabilityState.Stack`, since
      `language` no longer drives the label) — `cmd/backstop/gate_wiring_test.go`
      (class-2/class-3 fixtures rehomed onto installed-pack presence) and
      `pkg/gate/traceability_polarity_step_test.go` (its `cfgDeclaring` helper feeds
      `PolarityStepResult` message/stack assertions). The guard fences the look-alike
      `check.Options.Language`/`pack.Manifest.Language` (separate structs, NOT deleted) and
      trips ONLY on `config.Config.Language`
    tests:
      - TestConfig_NoTestAssertsConfigLanguage

contracts:
  - file: pkg/config/config.go
    provides:
      - name: Config.Language
        kind: variable
        signature: "Language string `yaml:\"language\" json:\"language\"`"
        absent: true
        notes: "DELETED (REQ-003/CLM-011/CLM-012): the single-language field is FULLY retired — a single language field is wrong for a polyglot repo. Declared absent so reintroducing it is caught as a regression. backstop.yml is parsed non-strict, so an existing `language:` key parses cleanly and is ignored (CLM-012). pack.Manifest.Language and check.Options.Language are SEPARATE fields on other structs and are unaffected."
    consumes: []
  - file: cmd/backstop/gate.go
    provides:
      - name: gateLanguage
        kind: function
        signature: "func gateLanguage(projectRoot string) string"
        absent: true
        notes: "DELETED (REQ-001/CLM-003): read the retired `language:` field to feed the bridge. Declared absent."
      - name: toolchainPackName
        kind: function
        signature: "func toolchainPackName(language string) string"
        absent: true
        notes: "DELETED (REQ-001/CLM-002): derived a `backstop/<language>-toolchain` pack name from the single language field. Declared absent so no toolchain pack name is ever computed from a language."
      - name: loadBridgedToolchainPacks
        kind: function
        signature: "func loadBridgedToolchainPacks(projectRoot, language string, declared []*pack.Manifest) ([]*pack.Manifest, error)"
        absent: true
        notes: "DELETED (REQ-001/CLM-001): the language-derived bridge that auto-loaded a toolchain pack from a single language field. Declared absent. Toolchain packs are now loaded ONLY via loadInstalledPacks over backstop.yml packs:."
      - name: countToolchainPacks
        kind: function
        signature: "func countToolchainPacks(declared []*pack.Manifest) int"
        notes: "MODIFIED (REQ-002/CLM-009): the `bridged` parameter is removed — it counts ONLY declared toolchain packs (normalized name ends in `-toolchain`)."
      - name: toolchainEnforcementStatus
        kind: function
        signature: "func toolchainEnforcementStatus(declared []*pack.Manifest) (gate.StepResult, bool)"
        notes: "MODIFIED (REQ-002/CLM-010): the `bridged` parameter is removed — emits the WARN-only 'enforcement not configured (0 toolchain packs)' state iff ZERO toolchain packs are DECLARED."
      - name: coverageRecordsProducer
        kind: function
        signature: "func coverageRecordsProducer(declared []*pack.Manifest, projectRoot string) coverageRecordsFn"
        notes: "MODIFIED (REQ-001/REQ-002/CLM-006/CLM-010): the `bridged` parameter is removed — coverage records source from the declared toolchain packs alone. SEAM: composes with SPEC-043 (`buildCoverageStep` gains a `classifier SourceClassifier` param) and SPEC-044 (the `(path, metric)` coverage record model) on the same coverage call chain — the three edits are orthogonal axes (input-set narrowing here, classifier-param addition in 043, record-model change in 044) and reconcile in the cross-consistency pass."
      - name: mergeSourceClassifier
        kind: function
        signature: "func mergeSourceClassifier(packs []*pack.Manifest) gate.SourceClassifier"
        notes: "PRESERVED — SPEC-043 symbol, NOT introduced or deleted here (B2 reconciliation/CLM-022). SPEC-043 performs the repoint of this symbol onto the DECLARED toolchain-pack set (`backstop.yml packs:`); this seed does NOT edit its call site. The bridge deletion must NOT orphan it — this seed only FENCES the `bridged` removal so the declared-pack manifest set remains in scope when SPEC-043's call site receives it. Declared here only to fence the bridge-deletion edit-set against accidentally removing or stranding it. (Param name `packs` matches SPEC-043's canonical signature.)"
      - name: gateConfig
        kind: function
        signature: "func gateConfig(projectRoot string) *config.Config"
        notes: "MODIFIED (REQ-003/CLM-014): the unreadable-config fallback no longer seeds `&config.Config{Language: \"go\"}` — it returns a zero-value-safe config with no Language field."
      - name: wrapTraceabilityStep
        kind: function
        signature: "func wrapTraceabilityStep(cfg *config.Config, dim gate.TraceabilityDimension, stepName string, stack string, delegate gate.StepFunc) gate.StepFunc"
        notes: "MODIFIED (REQ-004/CLM-016/CLM-021): gains a `stack string` parameter and threads it into `deriveCapabilityState(cfg, dim, stack)` (the real call graph is `buildGateSteps → wrapTraceabilityStep → deriveCapabilityState`, NOT `buildGateSteps → deriveCapabilityState` directly). Its THREE call sites in `buildGateSteps` (testSubstantiveness ~L653, coverage ~L654, contract ~L655) each pass the label computed once via `declaredToolchainStackLabel(packs)`. The classifier-intercept logic (the `ClassifyDimension` switch) is otherwise unchanged."
      - name: deriveCapabilityState
        kind: function
        signature: "func deriveCapabilityState(cfg *config.Config, dim gate.TraceabilityDimension, stack string) gate.CapabilityState"
        notes: "MODIFIED (REQ-003/REQ-004/CLM-014/CLM-016): gains a `stack string` parameter (passed by its sole caller `wrapTraceabilityStep`); the coverage-arm fallback message is de-language'd (no `cfg.Language` read), and the function stamps CapabilityState.Stack with the passed declared-toolchain stack label so the rehomed classifier renders it. Capability classification keys remain installed-pack-presence (unchanged)."
      - name: declaredToolchainStackLabel
        kind: function
        signature: "func declaredToolchainStackLabel(packs []*pack.Manifest) string"
        notes: "NEW (REQ-004/CLM-016/CLM-017/CLM-018): derives the cosmetic stack label from the SET of declared toolchain packs (normalized names with the `-toolchain` suffix stripped, joined). Returns \"unspecified\" when the declared toolchain-pack-NAME set is empty — the SINGLE authoritative empty-fallback signal (the label is name-derived). The label is the merged SET (no precedence). `SourceClassifier.HasSourceGlobs()` is corroborating only and is NOT the authoritative driver (it can diverge from the pack-name set when a toolchain pack declares no `classification` source globs)."
    consumes:
      - source: pkg/pack
        name: Manifest
        kind: type
      - source: pkg/gate
        name: SourceClassifier
        kind: type
      - source: pkg/gate
        name: CapabilityState
        kind: type
  - file: pkg/gate/traceability_polarity.go
    provides:
      - name: CapabilityState
        kind: type
        signature: "type CapabilityState struct { Present bool; Working bool; PackOrCommand string; Detail string; Stack string }"
        notes: "MODIFIED (REQ-004/CLM-016/CLM-021): gains a `Stack` field carrying the declared-toolchain stack label (assembled in cmd/backstop's deriveCapabilityState), replacing the `cfg.Language`-derived label in PolarityStepResult."
      - name: stackLabel
        kind: function
        signature: "func stackLabel(cfg *config.Config) string"
        absent: true
        notes: "DELETED (REQ-004/CLM-021): read `cfg.Language` for the fail-loud stack label. Replaced by CapabilityState.Stack, which is derived from the declared toolchain pack set in cmd/backstop. Declared absent so no `cfg.Language` read returns to pkg/gate."
      - name: PolarityStepResult
        kind: function
        signature: "func PolarityStepResult(stepName string, dim TraceabilityDimension, class PolarityClass, cfg *config.Config, cap CapabilityState) StepResult"
        notes: "MODIFIED (REQ-004/CLM-021): renders the stack label from cap.Stack instead of stackLabel(cfg). Classification verdicts (the class switch) are unchanged (CLM-020)."
    consumes: []
---

# SPEC-046: Retire Language Toolchain Bridge

## Overview

This spec is **Seed 3 of BUNDLE-012** (language-neutral gate consumer + TypeScript
toolchain pack) — **Pillar B**. It finishes the thin-executor mission on the gate's
**toolchain-selection** spine: the last baked place backstop assumes "a project has
ONE language, and that language implies its toolchain." It is authored in parallel
with siblings **SPEC-045** (de-Go test-verification discovery) and **SPEC-047** (the
`backstop/bun-toolchain` pack + external proof), and **builds on the fixed contract
SPEC-043** (the pack-declared `classification` globs block + the `pkg/gate`
`SourceClassifier`). The bundle's **SQ-1** (the polyglot traceability-classifier
precedence) is **owned and resolved here** per Seed 3's assignment — reusing SPEC-043's
single `SourceClassifier` for the glob union, not a parallel mechanism.

Three coupled moves:

1. **Delete the `language:`-derived toolchain bridge (REQ-001 → bundle REQ-004).**
   Remove `loadBridgedToolchainPacks`, `toolchainPackName`, `gateLanguage`, and the
   auto-load call site. Toolchain packs become **ordinary packs** — loaded only via
   the declared-pack path (`loadInstalledPacks` over `backstop.yml packs:`) and
   dispatched uniformly through `dispatchPackEngines`. A polyglot repo declares more
   than one. The dogfood **must not regress**: backstop-core already declares
   `backstop/go-toolchain` in `packs:`, so deleting the bridge keeps go-toolchain in
   its own gate via the declared path.

2. **Fully retire the `language:` field (REQ-003 → bundle REQ-005).** Delete
   `Config.Language`, strip `language: go` from the dogfood `backstop.yml`, and
   update every compile-time reader. No remnant.

3. **Rehome the traceability classifier + resolve SQ-1 (REQ-004 → bundle REQ-005).**
   The classifier's capability/polarity **decisions** already key on installed-pack
   presence (SPEC-037/038/041); `language:` only ever fed the cosmetic **stack
   label**. Re-derive that label from the declared toolchain-pack-name **set** (the
   single authoritative empty-fallback signal), reusing SPEC-043's single
   `SourceClassifier` for the polyglot glob union (never forking one).

**In scope:** the bridge deletion + uniform declared-pack dispatch; the full
`language:` retirement across schema, dogfood config, and every reader; the
traceability stack-label rehome and the SQ-1 resolution.

**Out of scope (fenced to siblings):** the coverage measurable-path /
`classification` contract → **SPEC-043** (consumed here, not redefined). De-Go'ing
test-verification discovery + `goFilePackageMatchesTarget` → **SPEC-045** (co-edits
`cmd/backstop/gate.go` — see the seam below). The coverage record model → **SPEC-044**.
The `backstop/bun-toolchain` pack + ratchet→block + external proof → **SPEC-047**.
`pack.Manifest.Language` and `check.Options.Language` (separate fields on other
structs) are **not** touched.

## Requirements

Requirements are enumerated in the `requirements:` frontmatter (REQ-001 … REQ-004),
tracing to BUNDLE-012 REQ-004/REQ-005 via `supports`. Summary:

| Spec REQ | Commits to | Bundle REQ |
| --- | --- | --- |
| REQ-001 | Delete the bridge (`loadBridgedToolchainPacks` / `toolchainPackName` / `gateLanguage` / the auto-load); toolchain packs load ONLY via the declared-pack path and dispatch uniformly; the bridge never synthesizes a pack; `bridged` removed from the gate wiring. | REQ-004 |
| REQ-002 | Dogfood does NOT regress (go-toolchain dispatched via `packs:` with no `language:`); polyglot declares 2+ toolchain packs; `countToolchainPacks` / `toolchainEnforcementStatus` / `coverageRecordsProducer` key on declared packs only. | REQ-004 |
| REQ-003 | Fully remove `Config.Language`, the dogfood `language: go`, and every `cfg.Language` reader; a stray `language:` key parses cleanly and is inert. | REQ-005 |
| REQ-004 | Rehome the stack label off `cfg.Language` onto the declared toolchain-pack-NAME set (the SINGLE authoritative signal, name-derived); reuse SPEC-043's single `SourceClassifier` for the polyglot glob union, with `HasSourceGlobs` CORROBORATING only (can diverge from the pack-name set — NOT the authoritative label driver); SQ-1 = merged union, set-valued label, no precedence; capability/polarity verdicts unchanged. | REQ-005 |

### Toolchain-pack acquisition matrix (REQ-001 / REQ-002)

The single allowlist this seed enforces: **a toolchain pack may be acquired ONLY via
the declared-pack path.** Every acquisition path, allowed and prohibited:

| Acquisition path | After this seed | Claim |
| --- | --- | --- |
| Declared in `backstop.yml packs:` (single) | **LOADED** + dispatched uniformly | CLM-004, CLM-007 |
| Declared in `backstop.yml packs:` (multiple / polyglot) | **ALL LOADED** + dispatched | CLM-008 |
| Not declared (zero toolchain packs) | **NONE** — WARN-only "enforcement not configured" | CLM-005, CLM-010 |
| Synthesized/derived from `language:` (the bridge) | **PROHIBITED** — path deleted | CLM-001, CLM-002, CLM-003, CLM-006 |

The dogfood is the single-declared cell (CLM-007): backstop-core's gate keeps
dispatching `backstop/go-toolchain` from `packs:` with no `language:` field — the
no-regression guarantee.

### SQ-1 resolution — the traceability classifier rehome (REQ-004)

SQ-1 ("how does the classifier read the glob set across a polyglot repo — per-pack?
the merged union 043 already builds? precedence on overlap?") is owned and resolved
here per Seed 3's assignment. **Resolution, stated explicitly:**

- **The polyglot glob UNION is read through SPEC-043's single `SourceClassifier`** — the
  same union `mergeSourceClassifier`/`SourceClassifier` already builds. It is **reused,
  not forked** (CLM-019): introducing a second glob classifier in this seed is
  prohibited.
- **B2 reconciliation — the classifier plumbing survives the bridge deletion.** The
  bridge deletion is **scoped to the bridge auto-load and the `bridged` threading
  only**. SPEC-043's `mergeSourceClassifier`/`SourceClassifier` are **not** this seed's
  symbols and **must not be deleted or orphaned**. **SPEC-043 itself performs the
  repoint** of those symbols onto the DECLARED toolchain-pack set (`backstop.yml
  packs:`), not the deleted `bridged` set; this seed does **not** edit SPEC-043's
  classifier call site — it only **fences** the `bridged` removal so the declared-pack
  manifest set (`loadInstalledPacks`) stays in scope at the classifier call site, so no
  call site is left dangling (CLM-022).
- **No per-pack read and no single-pack precedence.** The stack label is a SET — a
  polyglot repo's label names every declared stack (CLM-017). There is **no
  overlap-precedence winner** at the classifier level: overlap between source and
  test globs is already governed by SPEC-043's **test-wins-on-overlap** rule for
  *measurability*, and that rule is not re-decided here.
- **The "unspecified" fallback is driven by ONE authoritative signal: the empty
  declared toolchain-pack-NAME set** (the set `declaredToolchainStackLabel` reads), since
  the label is name-derived (CLM-018) — **never** by an empty `language:` field.
  SPEC-043's `SourceClassifier.HasSourceGlobs()` is **corroborating only**, NOT the
  authoritative driver: the two can **diverge** (a declared toolchain pack with no
  `classification` source globs has a NON-empty pack-name set but
  `HasSourceGlobs()==false`), so the label keys on the pack-name set, not on
  `HasSourceGlobs`.
- **Decisions are unchanged.** Capability/polarity classification (which `Class*` a
  dimension lands in) already keys on installed-pack presence (SPEC-037/038/041); the
  rehome touches **only the cosmetic label** (CLM-020). Removing `language:` cannot
  change any verdict.

### Shared-file seam (`cmd/backstop/gate.go`) — for the cross-consistency pass

This file is co-edited by four BUNDLE-012 seeds. The edits are disjoint but share the
file; flag for reconciliation:

| Seed | Edit on `cmd/backstop/gate.go` |
| --- | --- |
| **SPEC-043** | `buildCoverageStep` gains a `classifier SourceClassifier` param; adds `mergeSourceClassifier` (built over the **declared** toolchain-pack set, per B2). |
| **SPEC-044** | Coverage record consumption (`(path, metric)` model). |
| **SPEC-045** | Edits `goFilePackageMatchesTarget`. |
| **SPEC-046 (this)** | Deletes ONLY the bridge auto-load + the `bridged` threading: removes `bridged` from `coverageRecordsProducer` / `toolchainEnforcementStatus` / `countToolchainPacks` / the dispatch set / the early return; adds `declaredToolchainStackLabel`; threads `stack` through `wrapTraceabilityStep` → `deriveCapabilityState`. **Preserves** SPEC-043's `mergeSourceClassifier` / `SourceClassifier` (SPEC-043 repoints them onto the declared-pack set; this seed only fences, never deletes). |

This seed **reuses** SPEC-043's single `SourceClassifier` (for the polyglot glob union)
and **does not widen its interface** — SQ-1's stack label keys on the declared
toolchain-pack-NAME set (the single authoritative signal), with `HasSourceGlobs`
corroborating only; neither requires a `SourceClassifier` API change.

**B2 reconciliation (the blocking conflict resolved here).** The bridge deletion must
not orphan SPEC-043's classifier plumbing. `mergeSourceClassifier`/`SourceClassifier`
are SPEC-043 symbols; this seed neither introduces nor deletes them. **SPEC-043 itself
performs the repoint** onto the **declared** toolchain-pack manifest set (`backstop.yml
packs:`); this seed does not edit SPEC-043's call site — it only **fences** the
`bridged` removal so that declared-pack set stays in scope at the classifier call site,
leaving nothing dangling (CLM-022). The `coverageRecordsProducer` signature change
(dropping `bridged`) **composes** with SPEC-043's `buildCoverageStep` classifier-param
addition and SPEC-044's `(path, metric)` record-model change on the coverage call
chain — three orthogonal axes (input-set narrowing, classifier param, record model);
reconcile the call chain in the final pass.

## Implementation

Target package: **`cmd/backstop`** (the bridge + `language:` readers + the stack
label), with the field deletion in **`pkg/config`** and the `CapabilityState`/label
change in **`pkg/gate`**. Processing steps the planner must map tasks to:

1. **Delete the bridge (REQ-001).** Remove `loadBridgedToolchainPacks`,
   `toolchainPackName`, and `gateLanguage` from `cmd/backstop/gate.go`. In
   `buildGateSteps`, remove the `bridged, bridgeErr := loadBridgedToolchainPacks(...)`
   call and its `bridgeErr` config-error branch. Rewrite the downstream uses to key on
   `packs` alone: `coverageRecordsProducer(packs, projectRoot)`; the early return
   `if len(packs) == 0`; the dispatch set `excludeDedicatedStepRules(packs)`; the warn
   step `toolchainEnforcementStatus(packs)`. **Do NOT touch SPEC-043's
   `mergeSourceClassifier`/`SourceClassifier`** — they survive the bridge deletion;
   verify their call site is fed the DECLARED `packs` manifest set (not `bridged`) so
   nothing is orphaned (CLM-022).

2. **Re-key the count/warn helpers (REQ-002).** Change `countToolchainPacks(declared
   []*pack.Manifest) int` and `toolchainEnforcementStatus(declared []*pack.Manifest)`
   to drop the `bridged` parameter. The WARN-only "enforcement not configured (0
   toolchain packs)" message and `isToolchainPack` convention are retained verbatim —
   only the input set narrows to declared packs.

3. **Delete the `Config.Language` field (REQ-003).** Remove the `Language` field from
   `config.Config` in `pkg/config/config.go`. Update `gateConfig` to return a
   zero-value-safe config without the `Language: "go"` seed. De-language the
   `deriveCapabilityState` coverage-arm fallback message. Drop the
   `Language: cfg.Language` assignment in `cmd/backstop/code_check.go`. Remove the
   `language: go` line from the `cmd/backstop/gate_substantiveness_e2e.go` yml string
   literal. Strip `language: go` from the dogfood `backstop.yml`. Then update the
   EXISTING language-keyed tests (see the "Existing tests to delete or rewrite" table
   below): drop the `cfg.Language == "go"` assertion at `pkg/config/config_test.go:26`,
   and delete/rewrite `gate_capability_test.go`'s
   `TestCapabilityState_NonGoProject_DerivesAbsentClass2` /
   `TestCapabilityState_NonGoUndeclared_NeverAutoPromotes` onto installed-pack presence.
   Then sweep the COLLATERAL `config.Config.Language` readers. The completeness contract
   is NOT the file list below (a hand-list has proven to be whack-a-mole) — it is the
   CLM-025 source guard PLUS the `test_command` over `cmd/backstop` / `pkg/config` /
   `pkg/gate` / `pkg/check`, which together compile-break or trip on EVERY reader (those
   four are the only packages that read the field, so the guard scope is exhaustive); fix
   WHATEVER they flag, not only the enumerated files. Mechanical STRIPS (drop the inert
   `Language:` field): `cutover_deletion_test.go`, `cutover_consistency_test.go`,
   `gate_capability_rekey_test.go`, `gate_capability_contracts_rekey_test.go`. Two
   `check.Options.Language`-BRIDGE fixes — `pkg/check/registry_test.go` and
   `pkg/check/ts_executor_test.go` (both build `Options{Language: cfg.Language, Config: cfg}`
   from a `config.Config`) — STOP reading the deleted `config.Config.Language` and pass the
   language literal directly into the SEPARATE, surviving `check.Options.Language`. Two
   REHOME-COUPLED behavioral REWRITES (not token strips): `gate_wiring_test.go` (its
   class-2/class-3 capability-absent fixtures rehomed onto installed-pack presence/absence)
   and `pkg/gate/traceability_polarity_step_test.go` (its `cfgDeclaring` helper feeds
   `PolarityStepResult` message/stack assertions) — both drop the field AND re-source the
   rendered stack-label assertions from `CapabilityState.Stack`, since `language:` no
   longer drives the label. The CLM-025 source guard runs over `cmd/backstop`, `pkg/config`,
   `pkg/gate`, AND `pkg/check` (so the `pkg/check` readers, outside the original
   three-package scope, cannot escape as a late compile break) and fences the look-alike
   `check.Options.Language`/`pack.Manifest.Language` (NOT deleted), so NO `_test.go` in
   those packages reads `cfg.Language`/`config.Config.Language` (CLM-025).

4. **Add the declared-toolchain stack label and thread it through the wrapper
   (REQ-004 / SQ-1).** Add `declaredToolchainStackLabel(packs []*pack.Manifest) string`
   in `cmd/backstop`: collect the declared toolchain packs (`isToolchainPack`), strip
   the `backstop/` prefix and `-toolchain` suffix from each normalized name, join the
   SET; return `"unspecified"` when the declared toolchain-pack-NAME set is empty — the
   SINGLE authoritative empty-fallback signal. (`SourceClassifier.HasSourceGlobs()` is
   corroborating only; it MUST NOT be the authoritative driver, because it can diverge
   from the pack-name set.) Add a `Stack string` field to `gate.CapabilityState`. Thread
   the label along the REAL call graph
   `buildGateSteps → wrapTraceabilityStep(cfg, dim, stepName, stack, delegate) →
   deriveCapabilityState(cfg, dim, stack)` (NOT `buildGateSteps → deriveCapabilityState`
   directly — `deriveCapabilityState` is reached only through `wrapTraceabilityStep`).
   `wrapTraceabilityStep` gains a `stack string` parameter; compute the label once via
   `declaredToolchainStackLabel(packs)` in `buildGateSteps` and pass it at all THREE
   `wrapTraceabilityStep` call sites (testSubstantiveness ~L653, coverage ~L654, contract
   ~L655) so each `CapabilityState` carries it.

5. **Rehome `stackLabel` (REQ-004).** Delete `stackLabel(cfg)` from
   `pkg/gate/traceability_polarity.go` and change `PolarityStepResult` to read
   `cap.Stack` for the rendered stack label. The class switch (the verdict) is
   untouched (CLM-020).

6. **Prove the dogfood and the polyglot/empty cases end-to-end (REQ-002/REQ-004).**
   The mandated `TestGate_DispatchesGoToolchainViaDeclaredPackPathWithoutLanguageField`
   exercises the live gate over a config declaring `backstop/go-toolchain` with no
   `language:` field; the polyglot and empty-set claims exercise the label and dispatch
   over two-pack and zero-pack configs.

### Existing tests to delete or rewrite

These EXISTING tests take the DELETED bridge / `language:` code as their SUBJECT.
Leaving them green would force an implementer to PRESERVE the bridge (a shim),
defeating the spec. Each MUST be deleted or rewritten as listed (CLM-023/024/025); the
source guards assert no surviving reference to the deleted symbols.

**The `language:`-reader rows are ILLUSTRATIVE, not the task boundary.** The
COMPLETENESS CONTRACT for the `config.Config.Language` sweep is the CLM-025 source guard
(`TestConfig_NoTestAssertsConfigLanguage`) plus the `test_command` over the four packages
that read the field (`cmd/backstop` / `pkg/config` / `pkg/gate` / `pkg/check`) — together
they compile-break or trip on EVERY reader, so the implementer fixes WHATEVER they flag,
not only the files tabled here. Hand-maintaining this list has repeatedly missed readers
(it grew across review rounds); the guard + `test_command` is the durable, mechanical
backstop. The four-package guard scope is provably exhaustive because those are the only
packages that construct or read `config.Config.Language`.

| Existing test file / location | Subject (deleted code) | Action | Claim |
| --- | --- | --- | --- |
| `cmd/backstop/gate_bridge_load_test.go` | `loadBridgedToolchainPacks` / `gateLanguage` (5×) | **DELETE** | CLM-023 |
| `cmd/backstop/gate_bridge_agnostic_test.go` | one bridge call resolving two languages | **DELETE** | CLM-023 |
| `cmd/backstop/cutover_noregress_test.go` | `loadBridgedToolchainPacks` cases | **DELETE / REWRITE** | CLM-023 |
| `cmd/backstop/gate_no_toolchain_pack_test.go` | `toolchainEnforcementStatus(bridged, declared)` 2-arg | **REWRITE** to 1-arg `toolchainEnforcementStatus(declared)`; drop `bridged` cases | CLM-024 |
| `cmd/backstop/gate_capability_test.go` | `NonGoProject_DerivesAbsentClass2` / `NonGoUndeclared_NeverAutoPromotes` (language-keyed capability thesis) | **DELETE / REWRITE** onto installed-pack presence | CLM-025 |
| `pkg/config/config_test.go:26` | `cfg.Language == "go"` assertion | **STRIP** — drop the assertion | CLM-025 |
| `cmd/backstop/cutover_deletion_test.go` | `config.Config{Language: "go"}` site | **STRIP** — drop the inert field | CLM-025 |
| `cmd/backstop/cutover_consistency_test.go` | `config.Config{... Language: "go"}` (L28, L65) | **STRIP** — drop the inert field | CLM-025 |
| `cmd/backstop/gate_capability_rekey_test.go` | `config.Config{... Language: "go"}` constructions / `cfg.Language` | **STRIP** — drop the inert field | CLM-025 |
| `cmd/backstop/gate_capability_contracts_rekey_test.go` | `config.Config{... Language: "go"}` constructions | **STRIP** — drop the inert field | CLM-025 |
| `pkg/check/registry_test.go` | builds `Options{Language: cfg.Language, Config: cfg}` from a `config.Config` | **BRIDGE-FIX** — stop reading the deleted `config.Config.Language`; pass the language literal directly into the surviving, fenced `check.Options.Language` | CLM-025 |
| `pkg/check/ts_executor_test.go` | builds `Options{Language: noCmdCfg.Language ...}` / `Options{Language: cmdCfg.Language ...}` (L20, L30) | **BRIDGE-FIX** — stop reading the deleted `config.Config.Language`; pass the language literal directly into the surviving, fenced `check.Options.Language` | CLM-025 |
| `cmd/backstop/gate_wiring_test.go` | `Language: "typescript"` DRIVES the class-2/class-3 capability-absent verdict fixtures | **REHOME REWRITE** (not a token strip) — rehome the verdicts onto installed-pack presence/absence AND re-source the rendered stack-label assertions from `CapabilityState.Stack` | CLM-025 |
| `pkg/gate/traceability_polarity_step_test.go` | `cfgDeclaring(language ...)` helper (L12–15) feeds `PolarityStepResult` message/stack tests | **REHOME REWRITE** (not a token strip) — drop the field AND re-source the rendered stack-label assertions from `CapabilityState.Stack` | CLM-025 |

The two **BRIDGE-FIX** files live in `pkg/check`, OUTSIDE the original guard scope
(`cmd/backstop` / `pkg/config` / `pkg/gate`), so a stale `config.Config.Language` reader
there would escape the guard and surface only as a late module-wide compile break — the
guard scope is therefore EXTENDED to `pkg/check` (CLM-025). The two **REHOME REWRITE**
files (`gate_wiring_test.go`, `traceability_polarity_step_test.go`) genuinely need design
attention, NOT mechanical deletion: their assertions are about the rendered stack label,
which `language:` used to drive and which is now sourced from `CapabilityState.Stack` —
so they must re-source those assertions, not just delete the field token. The look-alike
`check.Options.Language` and `pack.Manifest.Language` (e.g. `pkg/packval`'s
`PackManifest.Language`) are SEPARATE fields on other structs, are NOT deleted, and MUST
remain — the guard fences them and trips ONLY on `config.Config.Language`.

After this seed, NO `_test.go` in `cmd/backstop` may reference
`loadBridgedToolchainPacks` / `gateLanguage` / `toolchainPackName` (CLM-023), and NO
`_test.go` in `cmd/backstop` / `pkg/config` / `pkg/gate` / `pkg/check` may read
`cfg.Language` / `config.Config.Language` (CLM-025).

## Verification

- **Level:** `integration` (threshold 80). The deletions and the label rehome span
  `cmd/backstop` → `pkg/gate` → `pkg/config`, and the no-regression guarantee
  (CLM-007) is a live-gate wiring fact, so the spec is verified at integration level
  with end-to-end gate tests ([[feedback_integration_gap]]).
- **Command:** `go test ./cmd/backstop/ ./pkg/gate/ ./pkg/config/ ./pkg/check/ -race
  -coverprofile=cover.out` (matching the `test_command` frontmatter). `./pkg/check/` is
  load-bearing: it is one of the four packages the CLM-025 completeness contract compiles
  to catch every `config.Config.Language` reader, so omitting it would let a stale
  `pkg/check` reader escape as a late module-wide compile break.
- **Mandated tests** (named in `claims[]`):
  - `TestGate_DispatchesGoToolchainViaDeclaredPackPathWithoutLanguageField` (CLM-007)
    — the gate STILL dispatches go-toolchain for backstop-core via the declared-pack
    path with NO `language:` field present.
  - `TestConfig_LanguageKeyIgnoredCleanlyFieldRemoved` (CLM-012) — a config carrying a
    `language:` key is parsed cleanly and ignored (the field is gone).

## Sharp Edges

- **Dogfood regression is the load-bearing risk.** Once `gateLanguage` and the bridge
  are gone, a `.go` toolchain runs ONLY if `backstop/go-toolchain` is in `packs:`.
  Backstop-core declares it (and `.backstop/packs/` is gitignored — the pack lives in
  its own repo, see [[packs_always_external]]), so the declared path covers it — but
  if a future config drops the declaration, the gate silently loses go enforcement and
  falls to the WARN-only state. CLM-007 pins this; do not let it rot.
- **`bridged` is threaded into FOUR downstream uses.** It is easy to delete the
  bridge call and miss `coverageRecordsProducer(bridged, ...)`,
  `excludeDedicatedStepRules(append(bridged, packs...))`,
  `toolchainEnforcementStatus(bridged, packs)`, and the
  `len(packs)==0 && len(bridged)==0` early return. Missing any one leaves a dangling
  reference (compile break) or a stale double-count. CLM-006 guards the set.
- **Deleting `Config.Language` is a compile fence — find every reader first.** The
  field has five non-test readers (`gateLanguage`, `gateConfig` fallback,
  `deriveCapabilityState` message, `code_check.go` assignment, `stackLabel`) plus the
  e2e yml literal and many test fixtures. `pack.Manifest.Language` and
  `check.Options.Language` are LOOK-ALIKES on other structs — do NOT delete those; the
  grep must distinguish `config.Config.Language` from the pack/check fields.
- **The reader list is whack-a-mole — the guard + `test_command`, not the enumeration,
  is the completeness contract.** Successive review rounds kept surfacing
  `config.Config.Language` readers the hand-list missed (e.g. `cutover_consistency_test.go`,
  `ts_executor_test.go`, `traceability_polarity_step_test.go` were all found late). The
  durable fix is to STOP treating the file list as authoritative: CLM-025's source guard
  (`TestConfig_NoTestAssertsConfigLanguage` over `cmd/backstop` / `pkg/config` / `pkg/gate`
  / `pkg/check`) plus the `test_command` compiling those four packages mechanically catch
  EVERY reader. The guard scope is provably exhaustive because those four are the ONLY
  packages that read the field. The implementer must fix WHATEVER the guard/`test_command`
  flag — the enumeration is illustrative, not the boundary — so CLM-025 is satisfiable by
  the mechanical contract rather than by a list that drifts.
- **Collateral `config.Config.Language` readers live OUTSIDE the original guard scope —
  most dangerously in `pkg/check`.** The first cut of CLM-025's guard covered only
  `cmd/backstop` / `pkg/config` / `pkg/gate`, but `config.Config{Language: ...}` is also
  constructed/read in `cmd/backstop/gate_wiring_test.go`,
  `cmd/backstop/gate_capability_rekey_test.go`,
  `cmd/backstop/gate_capability_contracts_rekey_test.go`,
  `cmd/backstop/cutover_deletion_test.go`, `cmd/backstop/cutover_consistency_test.go`,
  `pkg/gate/traceability_polarity_step_test.go`, and — outside the original three guarded
  packages — `pkg/check/registry_test.go` and `pkg/check/ts_executor_test.go`, which
  bridge `cfg.Language` → `check.Options.Language`. A stale reader in `pkg/check` would
  slip past the guard and surface only as a late module-wide compile break, the exact gap
  the PLAN review caught. The guard scope is EXTENDED to `pkg/check` and the `test_command`
  runs `./pkg/check/`. The `pkg/check` rewrite is SUBTLE: `check.Options.Language` is a
  SEPARATE, fenced field that SURVIVES — the fix is to pass the language LITERAL straight
  into `check.Options.Language`, NOT to delete that field. Over-zealous deletion of the
  look-alike is its own regression.
- **Two test files are REHOME REWRITES, not token strips — they need design attention.**
  `gate_wiring_test.go`'s `Language: "typescript"` fixtures DRIVE the class-2/class-3
  capability-absent verdicts, and `pkg/gate/traceability_polarity_step_test.go`'s
  `cfgDeclaring` helper feeds `PolarityStepResult` message/stack assertions. For BOTH,
  merely deleting the field token would change what the cases test or strand a now-empty
  label assertion. The verdicts must be rehomed onto installed-pack presence/absence — the
  same language-keyed-thesis retirement REQ-005 mandates — AND the rendered stack-label
  assertions must be re-sourced from the new `CapabilityState.Stack` carrier (since
  `language:` no longer drives the label). These two are categorically distinct from the
  mechanical strips (drop the inert field) and the `check.Options.Language` bridge-fixes
  (re-target the literal): they alone require behavioral rework, so flag them for the
  implementer as design-bearing, not deletion candidates.
- **Inert vs rejected.** REQ-003 requires a stray `language:` key to parse cleanly and
  be ignored (non-strict YAML), NOT to error. Adding strict-mode rejection would be a
  scope violation and would break older configs in the wild. The field is *gone*, not
  *forbidden*.
- **SQ-1 precedence is "no precedence" — resist inventing one.** The temptation is to
  pick a "primary" stack on overlap. The resolution is a SET (union); a polyglot
  repo's label names every declared stack. Overlap is a measurability concern already
  owned by SPEC-043's test-wins rule, not a label concern. Do not reintroduce a
  language-like single-winner.
- **The "unspecified" fallback keys on ONE signal — do not conflate two.** The label is
  name-derived, so its empty fallback MUST key on the declared toolchain-pack-NAME set
  being empty (the authoritative signal). SPEC-043's `HasSourceGlobs()` is corroborating
  ONLY: the two DIVERGE when a declared toolchain pack ships no `classification` source
  globs (non-empty pack-name set, yet `HasSourceGlobs()==false`). Driving the label off
  `HasSourceGlobs` would mislabel that repo "unspecified" despite a declared stack.
  CLM-018 pins the pack-name-set signal.
- **The rehome must REUSE, not FORK, SPEC-043's `SourceClassifier`.** Building a
  second glob classifier here would resurrect a parallel language-classification path
  — exactly the thing the bundle exists to delete. CLM-019 asserts no new
  glob-classifier type is defined in this seed; the only new symbol is
  `declaredToolchainStackLabel` (a name-set helper, not a glob matcher).
- **The stack label threads through `wrapTraceabilityStep`, not straight into
  `deriveCapabilityState`.** The real call graph is `buildGateSteps →
  wrapTraceabilityStep(cfg, dim, stepName, stack, delegate) →
  deriveCapabilityState(cfg, dim, stack)`. `deriveCapabilityState` is reached ONLY
  through the wrapper, so the `stack` param must be added to `wrapTraceabilityStep` and
  passed at ALL THREE call sites (testSubstantiveness/coverage/contract ~L653-655).
  Adding `stack` to `deriveCapabilityState` while computing the label inside it (or
  threading it past the wrapper) would either drop the label or duplicate the
  computation. CLM-016/CLM-021 pin the label render.
- **The bridge deletion must not orphan SPEC-043's classifier plumbing (B2).** SPEC-043
  adds `mergeSourceClassifier`/`SourceClassifier` on this same file. Naively deleting
  `bridged` could strand that call site (compile break or empty classifier). The
  resolution: those symbols are NOT this seed's to delete — they survive, and SPEC-043
  itself repoints them onto the DECLARED toolchain-pack set (`backstop.yml packs:`).
  This seed does NOT edit SPEC-043's call site; it only FENCES the `bridged` removal so
  the declared-pack manifest set stays in scope and reaches the classifier after
  `bridged` is gone. CLM-022 pins that both symbols survive and the call site is fed the
  declared set; the repoint is SPEC-043's half, the fence is this seed's half.
- **The label change must not leak into the verdict.** `stackLabel` feeds only
  message strings. If the rehome accidentally routes the label into a class decision,
  a polyglot/empty repo could flip class. CLM-020 pins that the class switch is
  unchanged by `language:` removal.

## Review Questions

- Are `loadBridgedToolchainPacks`, `toolchainPackName`, and `gateLanguage` actually
  DELETED (not merely unused), and is `bridged` removed from `coverageRecordsProducer`,
  `excludeDedicatedStepRules`, `toolchainEnforcementStatus`, and the zero-pack early
  return? (REQ-001/CLM-001..006.)
- Does the live gate STILL dispatch `backstop/go-toolchain` for backstop-core with NO
  `language:` field present — i.e. via the declared-pack path, not a synthesized one?
  (REQ-002/CLM-007 — the dogfood no-regression guarantee.)
- Is `Config.Language` removed from `pkg/config`, and does a `backstop.yml` carrying a
  `language:` key parse cleanly and inertly (non-strict YAML, not rejected)? Were the
  look-alike `pack.Manifest.Language` / `check.Options.Language` fields left intact?
  (REQ-003/CLM-011/CLM-012.)
- Is the SQ-1 "unspecified" fallback driven by the SINGLE authoritative signal — the
  empty declared toolchain-pack-NAME set — rather than `SourceClassifier.HasSourceGlobs()`
  (corroborating only, and divergent when a pack ships no source globs)? Is the polyglot
  glob union read through SPEC-043's single `SourceClassifier` (no fork), and is the
  polyglot label a SET with no single-pack precedence? (REQ-004/CLM-017/CLM-018/CLM-019.)
- Are the EXISTING bridge/`language:` behavioral tests DELETED or REWRITTEN (not kept
  green via a shim) — `gate_bridge_load_test.go` / `gate_bridge_agnostic_test.go` /
  `cutover_noregress_test.go` deleted, `gate_no_toolchain_pack_test.go` rewritten to the
  1-arg `toolchainEnforcementStatus`, `gate_capability_test.go`'s NonGo tests + the
  `config_test.go:26` `cfg.Language` assertion dropped — with no `_test.go` referencing
  the deleted symbols? (REQ-001/002/003/CLM-023/CLM-024/CLM-025.)
- Is CLM-025 satisfied by the MECHANICAL completeness contract — the
  `TestConfig_NoTestAssertsConfigLanguage` source guard over `cmd/backstop` / `pkg/config`
  / `pkg/gate` / `pkg/check` PLUS the `test_command` compiling those four packages — rather
  than by matching a hand-list? Does the guard PASS with NO surviving
  `config.Config.Language` reader anywhere in those four packages (every reader the
  guard/`test_command` flags is fixed, not only the enumerated ones)? Are BOTH `pkg/check`
  bridge-fix files (`registry_test.go` AND `ts_executor_test.go`) — OUTSIDE the original
  `cmd/backstop`/`pkg/config`/`pkg/gate` scope, hence the `./pkg/check/` extension —
  updated to pass the language literal into the SEPARATE, surviving `check.Options.Language`,
  with that look-alike (and `pack.Manifest.Language`) left intact rather than deleted?
  (REQ-003/CLM-025.)
- Were BOTH rehome-coupled rewrites — `gate_wiring_test.go` AND
  `pkg/gate/traceability_polarity_step_test.go` — actually REWRITTEN (their
  capability/class verdicts rehomed onto installed-pack presence/absence AND their rendered
  stack-label assertions re-sourced from `CapabilityState.Stack`) rather than
  token-stripped, so neither depends on the deleted `Language:` value? (REQ-003/CLM-025.)
- Is the stack label threaded along `buildGateSteps → wrapTraceabilityStep →
  deriveCapabilityState` (the wrapper gains `stack`, all three call sites pass it), NOT
  injected straight into `deriveCapabilityState` past the wrapper? (REQ-004/CLM-016/CLM-021.)
- Did the bridge deletion leave SPEC-043's `mergeSourceClassifier`/`SourceClassifier`
  intact AND fed the DECLARED toolchain-pack manifest set (not the deleted `bridged`
  set), with no orphaned call site, no compile break, and no empty classifier?
  (REQ-001/CLM-022 — the B2 reconciliation.)
- Are capability/polarity classification VERDICTS provably unchanged by removing
  `language:` — i.e. only the cosmetic label moved? (REQ-004/CLM-020.)

## References

- BUNDLE-012 (`language-neutral-consumer-ts-toolchain`) — this is Seed 3 (Pillar B);
  bundle REQ-004 (delete the bridge) and REQ-005 (retire `language:` + rehome the
  classifier) are the requirements this spec implements, and SQ-1 is resolved here.
- SPEC-043 (`pack-declared-globs-coverage-consumer`) — the **fixed contract** this
  spec builds on and reuses: the `classification` globs block, the `SourceClassifier`,
  and `mergeSourceClassifier` (which SPEC-043 repoints onto the declared toolchain-pack
  set). SQ-1 is owned and resolved in this seed per the bundle's Seed 3 assignment.
- SPEC-045 (`de-go-test-verification-discovery`) — parallel sibling co-editing
  `cmd/backstop/gate.go` (`goFilePackageMatchesTarget`); disjoint edits, flagged for
  the cross-consistency pass.
- SPEC-047 (`bun-toolchain-pack-and-proof`) — parallel sibling; declares a SECOND
  toolchain pack via `packs:`, exercising the uniform declared-pack dispatch this
  seed makes the only path.
- SPEC-037 / SPEC-038 / SPEC-041 — re-keyed the substantiveness/contracts/coverage
  capabilities onto installed-pack presence; this spec removes the last `language:`
  dependency (the cosmetic label), confirming those verdicts never needed it.
- SPEC-040 — the bridge's origin (`loadBridgedToolchainPacks`, the WARN-only
  no-toolchain-pack state); this spec deletes the bridge while keeping the WARN state
  keyed on declared packs.
- [[feedback_zero_baked_checks]] — the thin-executor first principle this seed
  finishes on the toolchain-selection spine: a toolchain is just another pack; zero
  baked language knowledge.
- [[feedback_integration_gap]] — the no-regression and dispatch guarantees are
  proven over the live gate, not only a unit.
- [[packs_always_external]] — `backstop/go-toolchain` lives in its own repo; the
  declared-pack path (not the deleted bridge) is what keeps it in the dogfood gate.
- Code (this branch): `cmd/backstop/gate.go` `gateLanguage` (~L263),
  `toolchainPackName` (~L283), `gateConfig` (~L291), `deriveCapabilityState` (~L322),
  `loadBridgedToolchainPacks` (~L486), `countToolchainPacks` (~L543),
  `toolchainEnforcementStatus` (~L563), `buildGateSteps` bridge call (~L602),
  `coverageRecordsProducer` (~L969); `pkg/gate/traceability_polarity.go` `stackLabel`
  (~L188), `CapabilityState` (~L81), `PolarityStepResult` (~L217); `pkg/config/config.go`
  `Config.Language` (~L22); `cmd/backstop/code_check.go` (~L134);
  `cmd/backstop/gate_substantiveness_e2e.go` (~L50); dogfood `backstop.yml` (`language: go`).

## Version History

- **1.2.0** (2026-07-05) — Retired the stale `cmd/backstop/code_check.go` provides
  `newCodeCheckCommand`: ISSUE-018 (authorized thin-executor eradication) deleted the
  `backstop code check` command and its file entirely, so the present-signature promise was a
  stale red under `contract_signature`. The whole `code_check.go` contract block was removed
  (deleted file). Contract-only realignment (align-predating-artifacts); no requirement, claim,
  or design change.
- **1.1.0** (2026-06-30) — Status → `implemented`. The BUNDLE-012 Seed 3 (Pillar B) code
  shipped and passed impl-review PASS; the `language:`-derived toolchain bridge is deleted,
  `Config.Language` is fully retired, and the traceability stack label is rehomed onto the
  declared toolchain-pack-NAME set (SQ-1 resolved). Also corrected the REQ-004
  summary-table row to match the REQ-004 body: `HasSourceGlobs` is CORROBORATING only, not
  the authoritative label driver — the label keys on the declared toolchain-pack-NAME set.
  No requirement, claim, or contract text changed — lifecycle transition plus one
  cosmetic table-consistency fix.
- **1.0.0** (2026-06-28) — Initial spec authored from BUNDLE-012 Seed 3 (Pillar B); SQ-1
  owned and resolved here.
