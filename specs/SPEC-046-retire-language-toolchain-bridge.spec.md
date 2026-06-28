---
title: "Retire Language Toolchain Bridge"
number: SPEC-046
created: "2026-06-28"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

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
    re-derived from the SET of declared toolchain packs, with the "any source
    declared" signal CONSUMING SPEC-043's merged-union `SourceClassifier`
    (`HasSourceGlobs`) rather than forking a parallel classifier. SQ-1 RESOLUTION: the
    classifier reads the MERGED UNION across all declared toolchain packs (the same
    union SPEC-043's `mergeSourceClassifier` builds), NOT a per-pack or
    single-precedence read; a polyglot repo's label is the joined SET of its declared
    stacks; there is no overlap-precedence winner (overlap for measurability is
    already governed by SPEC-043's test-wins rule); when no toolchain pack is declared
    the label is "unspecified", driven by the empty declared-toolchain set /
    `HasSourceGlobs()==false`, never by an empty `language:` field. The dogfood MUST
    NOT regress: backstop-core already declares `backstop/go-toolchain` in `packs:`,
    so deleting the bridge keeps go-toolchain in its own gate via the declared-pack
    path. SHARED-FILE SEAM: `cmd/backstop/gate.go` is co-edited by SPEC-045
    (`goFilePackageMatchesTarget`), SPEC-043 (`buildCoverageStep` classifier param),
    and SPEC-044 (coverage records) — this seed's edits (delete the bridge + remove
    `bridged` from `coverageRecordsProducer`/`toolchainEnforcementStatus`/
    `countToolchainPacks`/the dispatch list/the early return) are disjoint from those
    but share the file; flagged for the cross-consistency pass. B2 RECONCILIATION: the
    deletion is SCOPED to the bridge auto-load + the `bridged` threading ONLY —
    SPEC-043's `mergeSourceClassifier`/`SourceClassifier` SURVIVE and are repointed onto
    the DECLARED toolchain-pack set (`backstop.yml packs:`), NOT the deleted `bridged`
    set (SPEC-043 is updated to match), so no classifier call site is orphaned. The
    `coverageRecordsProducer` signature change (dropping `bridged`) composes with
    SPEC-043's classifier-param addition and SPEC-044's record-model change on the
    coverage call chain (orthogonal axes).
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/ ./pkg/gate/ ./pkg/config/ -race -coverprofile=cover.out
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
      deleted and MUST NOT be orphaned: they SURVIVE the bridge deletion, repointed
      onto the DECLARED-pack manifest set (the ordinary `backstop.yml packs:` set
      returned by `loadInstalledPacks`) rather than the deleted `bridged` set. The
      declared-pack manifest set MUST remain in scope at every classifier call site
      after the bridge is gone, so no `mergeSourceClassifier`/classifier call site is
      left dangling.
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
      packs alone.
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
      verdict — the field is simply gone. (NOTE: `pack.Manifest.Language` and
      `check.Options.Language` are SEPARATE fields on other structs and are out of
      scope.)
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
      classifier reads the MERGED UNION across ALL declared toolchain packs — the SAME
      union SPEC-043's `mergeSourceClassifier`/`SourceClassifier` builds, REUSED not
      forked, and (post-reconciliation) sourced from the DECLARED toolchain-pack set
      (`backstop.yml packs:`), NOT the deleted `bridged` set — so a polyglot repo's
      label is the joined SET of its declared stacks
      with NO single-pack precedence and NO overlap winner (measurability overlap is
      already governed by SPEC-043's test-wins-on-overlap rule, not re-decided here).
      The "any source declared at all" signal that selects the "unspecified" fallback
      MUST consume SPEC-043's `SourceClassifier.HasSourceGlobs()` (empty declared set
      ⇒ `false` ⇒ "unspecified"), NOT an empty `language:` field. It is PROHIBITED to
      introduce a second, parallel glob classifier in this seed.
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
    text: SQ-1 EMPTY — a repo declaring NO toolchain pack yields the "unspecified" stack label, driven by the empty declared-toolchain set / SPEC-043 `SourceClassifier.HasSourceGlobs()==false`, NOT by an empty `language:` field
    tests:
      - TestPolarity_NoToolchainPackStackLabelUnspecified
  - id: CLM-019
    requirement: REQ-004
    text: SQ-1 REUSE — the rehome consumes SPEC-043's `gate.SourceClassifier` (`HasSourceGlobs`) for the "any source declared" signal and introduces NO parallel/forked glob classifier in this seed (source guard asserts no new glob-classifier type is defined here)
    tests:
      - TestPolarity_RehomeReusesSourceClassifierNotForked
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
        signature: "func mergeSourceClassifier(declared []*pack.Manifest) gate.SourceClassifier"
        notes: "PRESERVED — SPEC-043 symbol, NOT introduced or deleted here (B2 reconciliation/CLM-022). The bridge deletion must NOT orphan it: it survives and is repointed onto the DECLARED toolchain-pack set (`backstop.yml packs:`) rather than the deleted `bridged` set. Its call site still receives the declared-pack manifest set after `bridged` is removed. Declared here only to fence the bridge-deletion edit-set against accidentally removing or stranding it."
      - name: gateConfig
        kind: function
        signature: "func gateConfig(projectRoot string) *config.Config"
        notes: "MODIFIED (REQ-003/CLM-014): the unreadable-config fallback no longer seeds `&config.Config{Language: \"go\"}` — it returns a zero-value-safe config with no Language field."
      - name: deriveCapabilityState
        kind: function
        signature: "func deriveCapabilityState(cfg *config.Config, dim gate.TraceabilityDimension, stack string) gate.CapabilityState"
        notes: "MODIFIED (REQ-003/REQ-004/CLM-014/CLM-016): the coverage-arm fallback message is de-language'd (no `cfg.Language` read), and the function stamps CapabilityState.Stack with the declared-toolchain stack label so the rehomed classifier renders it. Capability classification keys remain installed-pack-presence (unchanged)."
      - name: declaredToolchainStackLabel
        kind: function
        signature: "func declaredToolchainStackLabel(packs []*pack.Manifest) string"
        notes: "NEW (REQ-004/CLM-016/CLM-017/CLM-018): derives the cosmetic stack label from the SET of declared toolchain packs (normalized names with the `-toolchain` suffix stripped, joined). Returns \"unspecified\" when no toolchain pack is declared. The label is the merged SET (no precedence); the empty-set fallback aligns with SPEC-043 SourceClassifier.HasSourceGlobs()==false."
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
  - file: cmd/backstop/code_check.go
    provides:
      - name: newCodeCheckCommand
        kind: function
        signature: "func newCodeCheckCommand(jsonFlag *bool) *cobra.Command"
        notes: "MODIFIED (REQ-003/CLM-014): the `check.Options{ ... Language: cfg.Language ... }` assignment drops the config-sourced `Language` (the retired field). check.Options.Language is left at its empty-language default — a behavior-preserving change for the empty case (gateLanguage's old comment notes check.Options' empty-language default). check.Options.Language itself is out of scope."
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
`SourceClassifier`). SPEC-043 explicitly **deferred the bundle's SQ-1** (the polyglot
traceability-classifier precedence) to this spec, so **SQ-1 is owned and resolved
here** — reusing SPEC-043's `SourceClassifier`, not a parallel mechanism.

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
   label**. Re-derive that label from the declared toolchain pack **set**, with the
   "any source declared" signal consuming SPEC-043's merged `SourceClassifier`.

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
| REQ-004 | Rehome the stack label off `cfg.Language` onto the declared-toolchain set; consume SPEC-043's `SourceClassifier` (`HasSourceGlobs`); SQ-1 = merged union, set-valued label, no precedence; capability/polarity verdicts unchanged. | REQ-005 |

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

SPEC-043 deferred SQ-1 ("how does the classifier read the glob set across a polyglot
repo — per-pack? the merged union 043 already builds? precedence on overlap?") to
this spec. **Resolution, stated explicitly:**

- **The classifier reads the MERGED UNION across all declared toolchain packs** — the
  same union SPEC-043's `mergeSourceClassifier`/`SourceClassifier` already builds. It
  is **reused, not forked** (CLM-019): introducing a second glob classifier in this
  seed is prohibited.
- **B2 reconciliation — the classifier plumbing survives the bridge deletion.** The
  bridge deletion is **scoped to the bridge auto-load and the `bridged` threading
  only**. SPEC-043's `mergeSourceClassifier`/`SourceClassifier` are **not** this seed's
  symbols and **must not be deleted or orphaned**: they survive and are **repointed
  onto the DECLARED toolchain-pack set** (`backstop.yml packs:`), not the deleted
  `bridged` set (SPEC-043 is being updated to match). The declared-pack manifest set
  (`loadInstalledPacks`) stays in scope at every classifier call site after the bridge
  is gone, so no call site is left dangling (CLM-022).
- **No per-pack read and no single-pack precedence.** The stack label is a SET — a
  polyglot repo's label names every declared stack (CLM-017). There is **no
  overlap-precedence winner** at the classifier level: overlap between source and
  test globs is already governed by SPEC-043's **test-wins-on-overlap** rule for
  *measurability*, and that rule is not re-decided here.
- **The "unspecified" fallback is driven by the empty declared-toolchain set**, i.e.
  SPEC-043's `SourceClassifier.HasSourceGlobs() == false` (CLM-018) — **never** by an
  empty `language:` field.
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
| **SPEC-046 (this)** | Deletes ONLY the bridge auto-load + the `bridged` threading: removes `bridged` from `coverageRecordsProducer` / `toolchainEnforcementStatus` / `countToolchainPacks` / the dispatch set / the early return; adds `declaredToolchainStackLabel`. **Preserves** SPEC-043's `mergeSourceClassifier` / `SourceClassifier` (repointed onto the declared-pack set, not deleted). |

This seed **consumes** SPEC-043's `SourceClassifier` (for the `HasSourceGlobs`
signal) and **does not widen its interface** — SQ-1 is resolved with `HasSourceGlobs`
plus the declared-pack-name set, neither of which requires a SourceClassifier API
change.

**B2 reconciliation (the blocking conflict resolved here).** The bridge deletion must
not orphan SPEC-043's classifier plumbing. `mergeSourceClassifier`/`SourceClassifier`
are SPEC-043 symbols; this seed neither introduces nor deletes them — it only ensures
they are fed the **declared** toolchain-pack manifest set (`backstop.yml packs:`)
after `bridged` is removed, so the classifier sources from the declared set and no
call site is left dangling (CLM-022). The `coverageRecordsProducer` signature change
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
   literal. Strip `language: go` from the dogfood `backstop.yml`. Sweep every test
   fixture and `config.Config{Language: ...}` construction.

4. **Add the declared-toolchain stack label (REQ-004 / SQ-1).** Add
   `declaredToolchainStackLabel(packs []*pack.Manifest) string` in `cmd/backstop`:
   collect the declared toolchain packs (`isToolchainPack`), strip the `backstop/`
   prefix and `-toolchain` suffix from each normalized name, join the SET; return
   `"unspecified"` when the set is empty. Use SPEC-043's
   `SourceClassifier.HasSourceGlobs()` as the corroborating "any source declared"
   signal for the empty-set fallback. Add a `Stack string` field to
   `gate.CapabilityState`. Thread the label through `buildGateSteps` →
   `deriveCapabilityState(cfg, dim, stack)` so each `CapabilityState` carries it.

5. **Rehome `stackLabel` (REQ-004).** Delete `stackLabel(cfg)` from
   `pkg/gate/traceability_polarity.go` and change `PolarityStepResult` to read
   `cap.Stack` for the rendered stack label. The class switch (the verdict) is
   untouched (CLM-020).

6. **Prove the dogfood and the polyglot/empty cases end-to-end (REQ-002/REQ-004).**
   The mandated `TestGate_DispatchesGoToolchainViaDeclaredPackPathWithoutLanguageField`
   exercises the live gate over a config declaring `backstop/go-toolchain` with no
   `language:` field; the polyglot and empty-set claims exercise the label and dispatch
   over two-pack and zero-pack configs.

## Verification

- **Level:** `integration` (threshold 80). The deletions and the label rehome span
  `cmd/backstop` → `pkg/gate` → `pkg/config`, and the no-regression guarantee
  (CLM-007) is a live-gate wiring fact, so the spec is verified at integration level
  with end-to-end gate tests ([[feedback_integration_gap]]).
- **Command:** `go test ./cmd/backstop/ ./pkg/gate/ ./pkg/config/ -race
  -coverprofile=cover.out`.
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
- **Inert vs rejected.** REQ-003 requires a stray `language:` key to parse cleanly and
  be ignored (non-strict YAML), NOT to error. Adding strict-mode rejection would be a
  scope violation and would break older configs in the wild. The field is *gone*, not
  *forbidden*.
- **SQ-1 precedence is "no precedence" — resist inventing one.** The temptation is to
  pick a "primary" stack on overlap. The resolution is a SET (union); a polyglot
  repo's label names every declared stack. Overlap is a measurability concern already
  owned by SPEC-043's test-wins rule, not a label concern. Do not reintroduce a
  language-like single-winner.
- **The rehome must REUSE, not FORK, SPEC-043's `SourceClassifier`.** Building a
  second glob classifier here would resurrect a parallel language-classification path
  — exactly the thing the bundle exists to delete. CLM-019 asserts no new
  glob-classifier type is defined in this seed; the only new symbol is
  `declaredToolchainStackLabel` (a name-set helper, not a glob matcher).
- **The bridge deletion must not orphan SPEC-043's classifier plumbing (B2).** SPEC-043
  adds `mergeSourceClassifier`/`SourceClassifier` on this same file, and in its
  pre-reconciliation form the classifier was fed off the toolchain-pack set that
  included `bridged`. Naively deleting `bridged` would strand that call site (compile
  break or empty classifier). The resolution: those symbols are NOT this seed's to
  delete — they survive and are repointed onto the DECLARED toolchain-pack set
  (`backstop.yml packs:`). The declared-pack manifest set MUST remain in scope and be
  passed to the classifier after `bridged` is gone. CLM-022 pins that both symbols
  survive and the call site is fed the declared set. SPEC-043 is updated to match (the
  classifier sources declared, not `bridged`); this is the cross-spec half of the
  reconciliation.
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
- Does the SQ-1 rehome read the MERGED UNION via SPEC-043's `SourceClassifier`
  (`HasSourceGlobs`) rather than a forked classifier, and is the polyglot label a SET
  with no single-pack precedence? (REQ-004/CLM-017/CLM-018/CLM-019.)
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
  and `mergeSourceClassifier`. SPEC-043 deferred SQ-1 to this seed.
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
