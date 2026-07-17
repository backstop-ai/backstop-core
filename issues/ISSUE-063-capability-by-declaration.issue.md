---
title: "Capability By Declaration — Detect Traceability Capabilities By Declared gate_type, Not Pack Name"
schema_version: issue/v1

issue:
  id: ISSUE-063
  title: "Capability By Declaration — Detect Traceability Capabilities By Declared gate_type, Not Pack Name"
  type: technical-debt
  status: closed
  created: "2026-07-17"
  closed: "2026-07-17"

delivered_by: PLAN-ISSUE-063

complexity:
  scope: contained
  uncertainty: known
  risk: moderate

verification:
  level: unit
  coverage_threshold: 90
  test_command: "go test ./cmd/backstop/... -run 'Capability|Dimension|PackDeclares'"

implementation:
  summary: >
    Determine each traceability dimension's capability presence by whether an INSTALLED
    pack declares an engine with that dimension's gate_type (contracts / substantiveness /
    coverage), not by the pack's name. Replace the three name-keyed detectors —
    `contractsPackInstalled` / `substantivenessPackInstalled` (exact `backstop/<name>`) and
    `coverageToolchainPackInstalled` (`-toolchain` suffix) — with one declaration-keyed
    check, and resolve WHICH pack runs (e.g. the contracts compile-signature) by the same
    gate_type declaration rather than the hardcoded `backstop/contracts` name. Add a
    backstop/self rule family flagging capability/behavior detection keyed on a pack-name
    literal or name-shape instead of a declared capability.
  package: cmd/backstop, pkg/gate, backstop-self-pack

requirements:
  - id: REQ-001
    text: >
      A traceability dimension's capability (contracts, substantiveness, coverage) must be
      present iff an INSTALLED pack (declared in `cfg.Packs`) declares an engine whose
      `gate_type` equals the dimension name — determined by loading the installed pack
      manifests and inspecting `manifest.Engines[].GateType`, NOT by matching a pack name.
      `manifest.Engines[].GateType` (pkg/pack/manifest.go) and the resolved manifest set
      (the `resolveContractsPacks` load path) already exist; capability detection must key
      on them.
  - id: REQ-002
    text: >
      The name-keyed detectors must be removed from the capability path:
      `contractsPackInstalled` / `substantivenessPackInstalled` must no longer match
      `cfg.Packs["backstop/contracts"]` / `["backstop/substantiveness"]` (via
      `contractsPackName()` / `substantivenessPackName()`), and
      `coverageToolchainPackInstalled` must no longer match a `-toolchain` name suffix. The
      hardcoded `backstop/<name>` literals and the suffix heuristic are deleted from the
      capability decision (a pack NAME may remain only as a human display label, never as the
      capability key).
  - id: REQ-003
    text: >
      Capability detection must be ORG-AGNOSTIC and convention-free. Pack names are GitHub
      distribution coordinates — `org/pack-name` resolves to `github.com/org/pack-name`
      (SPEC-015), so keying capability on the name binds it to one org (`backstop/*`) or one
      naming convention (`*-toolchain`). After this change, a pack from ANY org
      (`acme/ts-contracts`, `bclabs/contracts`, a local pack) that declares a
      `gate_type: contracts` engine provides the contracts capability, with zero dependence
      on its name or org.
  - id: REQ-004
    text: >
      The pack whose script runs for a dimension (e.g. the contracts pack whose
      `compile-signature.sh` compiles a signature, resolved today by `resolveContractsPacks`
      matching `NormalizedName == "backstop/contracts"`) must likewise be selected by its
      declared `gate_type: contracts` engine, not by name — so the SAME by-declaration
      resolution drives both capability presence (REQ-001) and engine dispatch. If more than
      one installed pack declares the dimension's gate_type, resolution must be deterministic
      and fail-loud on ambiguity rather than silently pick one.
  - id: REQ-005
    text: >
      `backstop/self` must gain a rule family flagging capability or behavior detection keyed
      on a pack-name literal or name-shape — an exact `org/pack` string literal used as a map
      key against `cfg.Packs`, or a `strings.HasSuffix`/`HasPrefix`/`Contains` test on a pack
      name — where a declared `gate_type`/capability check is the correct source. This is the
      baked-distribution-identity class (analogous to ISSUE-062's baked-name-extraction and
      the existing baked-language-token rules): a specific coordinate/convention baked into
      capability logic that should be derived from declarations. Ships with a positive fixture
      (a `cfg.Packs["backstop/contracts"]`-style detector) and a negative fixture (a
      gate_type-declaration scan).
  - id: REQ-006
    text: >
      Existing behavior must be preserved for the current corpus: with the backstop-first
      packs installed under their present names, every dimension resolves exactly as today
      (present/working), and the class-2 (undeclared+absent) warn / class-3 (declared+absent)
      block classifications are unchanged. The change is a detection-mechanism swap, not a
      policy change — no dimension flips present<->absent for an already-correct install.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: A dimension's capability is present when an installed pack declares an engine with the matching gate_type.
    tests:
      - TestCapability_PresentWhenPackDeclaresGateType
  - id: CLM-002
    requirement: REQ-001
    text: A dimension's capability is absent when no installed pack declares an engine with the matching gate_type.
    tests:
      - TestCapability_AbsentWhenNoPackDeclaresGateType
  - id: CLM-003
    requirement: REQ-002
    kind: absence
    text: The hardcoded backstop/contracts and backstop/substantiveness capability-key literals are removed.
    tests:
      - TestHardcodedCapabilityPackNamesRemoved
  - id: CLM-004
    requirement: REQ-002
    kind: absence
    text: The -toolchain name-suffix heuristic is removed from coverage capability detection.
    tests:
      - TestCoverageToolchainSuffixHeuristicRemoved
  - id: CLM-005
    requirement: REQ-003
    text: A pack under a non-backstop org declaring a contracts gate_type engine provides the contracts capability.
    tests:
      - TestCapability_OrgAgnosticProvider
  - id: CLM-006
    requirement: REQ-004
    text: The dispatched contracts pack is selected by its declared gate_type engine, not by name.
    tests:
      - TestResolveContractsPack_ByGateTypeNotName
  - id: CLM-007
    requirement: REQ-004
    text: Multiple installed packs declaring the same dimension gate_type resolve deterministically or fail loud.
    tests:
      - TestResolveCapabilityPack_AmbiguityFailsLoud
  - id: CLM-008
    requirement: REQ-005
    text: The self rule flags a cfg.Packs pack-name-literal capability detector.
    tests:
      - TestSelfRule_FlagsPackNameKeyedCapabilityDetection
  - id: CLM-009
    requirement: REQ-006
    text: With the current backstop-first install, every dimension resolves present exactly as before.
    tests:
      - TestCapability_CurrentInstallUnchanged

contracts:
  - file: cmd/backstop/gate.go
    provides:
      - name: contractsPackName
        kind: function
        signature: "func contractsPackName() string"
        absent: true
      - name: substantivenessPackName
        kind: function
        signature: "func substantivenessPackName() string"
        absent: true
---

# Capability By Declaration — Detect Traceability Capabilities By Declared gate_type, Not Pack Name

## Problem

Traceability-dimension capability presence is decided by the pack's NAME, not by what the
pack declares. Three detectors in `cmd/backstop/gate.go`:

- `contractsPackInstalled` → `cfg.Packs["backstop/contracts"]` (exact, via `contractsPackName()`)
- `substantivenessPackInstalled` → `cfg.Packs["backstop/substantiveness"]` (exact)
- `coverageToolchainPackInstalled` → `strings.HasSuffix(name, "-toolchain")` (name suffix)

Pack names are GitHub distribution coordinates: `pack add` resolves `org/pack-name` to
`github.com/org/pack-name` (SPEC-015, git-native, no central registry). So keying a
capability on the name `backstop/contracts` binds the contracts capability to a pack
published by the **`backstop` GitHub org** — no other org can provide it. `coverage` is
less strict (any `*-toolchain` pack, org-agnostic) but still keys on a naming CONVENTION
rather than the capability itself.

This does not scale to a multi-publisher ecosystem — third-party, community, or a
consumer's OWN private packs. A TypeScript contracts pack authored by anyone other than the
`backstop` org cannot fill the contracts slot except by claiming the `backstop/contracts`
coordinate (only possible via a local install, which skips github resolution — the exact
workaround used to unblock bclabs-portal). The capability is conflated with a distribution
coordinate.

## Root cause

Capability identity ("this pack provides the contracts dimension") is keyed on distribution
identity (`org/pack-name`). The two should be independent: capability is declared —
`manifest.Engines[].GateType == "contracts"` — while the name is only where the pack lives.
The detection infrastructure already exists (`resolveContractsPacks` loads the manifest set;
`contractSignatureEngine` already inspects `manifest.Engines`; `manifest.Engines[].GateType`
is a first-class field), so the fix is a mechanism swap, not new plumbing.

## Fix

1. **Detect by declared gate_type (REQ-001/REQ-002/REQ-003).** Replace all three name-keyed
   detectors with one check: is any installed pack declaring an engine whose `gate_type`
   equals the dimension? Delete the `backstop/<name>` literals and the `-toolchain` suffix.
2. **Resolve dispatch by the same declaration (REQ-004).** Select the pack whose script runs
   (contracts `compile-signature.sh`, substantiveness rules) by its declared gate_type
   engine, deterministic + fail-loud on ambiguity.
3. **self rule closes the class (REQ-005).** Flag capability detection keyed on a pack-name
   literal or name-shape — the baked-distribution-identity class, sibling to ISSUE-062's
   baked-name-extraction rule and the existing baked-language-token rules.
4. **Behavior-preserving (REQ-006).** The current backstop-first install resolves every
   dimension exactly as today; this is a detection swap, not a policy change.

## Out of scope

- ISSUE-062 (structured finding properties / substantiveness join). Independent; may land
  in either order.
- Any change to the `org/pack-name` distribution scheme itself (SPEC-015). Names stay GitHub
  coordinates; this issue only stops capability logic from keying on them.

## Notes / references

- Discovered while building bclabs-portal (a TypeScript consumer): the TS contracts pack had
  to be named `backstop/contracts` and local-installed to be detected, which surfaced the
  org-coupling.
- `coverageToolchainPackInstalled` (name-suffix) is the least-bad current detector but still
  convention-keyed; fold it into the same by-declaration check for consistency.
- Sibling to ISSUE-062: both replace baked knowledge (a Go-shaped name assumption there; a
  distribution coordinate here) with a declaration-derived source, and both add the matching
  backstop/self rule so the class stays caught in core's own suite.

## Resolution

Delivered by PLAN-ISSUE-063.

- **REQ-001/002/003 (detect by declared gate_type).** A new `packDeclaresGateType(packs,
  dim)` (`cmd/backstop/gate.go`) reports a dimension present iff some installed pack manifest
  declares an engine whose `Engines[].Binding.GateType` equals the dimension (resolved via
  `engine.ParseGateType`, so it fails closed on drift). The three name-keyed detectors are
  rewritten as thin delegators to it: `contractsPackInstalled`, `substantivenessPackInstalled`,
  and `coverageToolchainPackInstalled` now take `[]*pack.Manifest` and call
  `packDeclaresGateType`. The hardcoded `contractsPackName()`/`substantivenessPackName()`
  accessors (returning `backstop/contracts` / `backstop/substantiveness`) and the
  `strings.HasSuffix(name, "-toolchain")` coverage heuristic are DELETED from the capability
  path (contract `absent: true` honored). `capabilityStateForDimension` collapses to one
  by-declaration check; its `PackOrCommand` string is a human display label only, never a
  capability key. The installed manifests are threaded from `buildGateSteps` through
  `wrapTraceabilityStep` → `deriveCapabilityState`. Detection is now org-agnostic: a pack
  under any org (e.g. `acme/ts-contracts`) declaring `gate_type: contracts` fills the slot.
- **REQ-004 (dispatch by the same declaration).** A shared `packsDeclaringGateType` (deduped
  per pack, deterministically sorted by NormalizedName) backs both capability presence and a
  new `resolveCapabilityPack(installed, dim)` that returns the single provider, nil when
  absent, and a fail-loud config error naming the ambiguous packs when more than one declares
  the dimension. `resolveContractsPacks` and `resolveSubstantivenessPacks` now select by
  gate_type instead of `NormalizedName == backstop/<name>`. Substantiveness routing rule IDs
  are derived from the RESOLVED pack's NormalizedName (rule-name constants
  `hollow-test-go`/`referenced-symbol-go` are rule identities, not coordinates), so no pack
  coordinate is baked.
- **REQ-005 (self rule).** `backstop/self` gained Family B6 (`no-pack-name-keyed-capability`)
  flagging capability logic keyed on a baked distribution identity — an `org/pack` literal
  used as a `cfg.Packs` map key, or a `HasSuffix`/`HasPrefix`/`Contains` test whose literal is
  a pack coordinate or the `-toolchain` convention — with a positive (`cfg.Packs` +
  `-toolchain` detector) and negative (gate_type-declaration scan) fixture, `*testdata*`
  excluded from live scope. Its regexes use the exact pack-name grammar (alnum+hyphen segments)
  so a file-path prefix is not a false hit. NOTE (cascade guard): scanning core with B6
  surfaces ONE remaining site beyond the three rewritten detectors — `isToolchainPack`
  (`strings.HasSuffix(m.NormalizedName, "-toolchain")` in `cmd/backstop/gate.go`), which is the
  toolchain-DISPATCH routing / cosmetic stack label, NOT a traceability-capability detector and
  out of this issue's scope. The rule is authored, fixture-tested (`backstop pack test`), and
  Go-tested (`TestSelfRule_FlagsPackNameKeyedCapabilityDetection`, `pkg/gate/self_rule_test.go`)
  but NOT installed into `.backstop/packs` — activation held for separate triage, as B5 was.
- **REQ-006 (behavior-preserving).** The current backstop-first install declares
  `gate_type: contracts` (contracts pack), `gate_type: substantiveness` (substantiveness pack),
  and `gate_type: coverage` (go-toolchain), each by exactly one pack, so every dimension
  resolves present and unambiguous exactly as before — pinned by
  `TestCapability_CurrentInstallUnchanged` over the real installed manifests. The classifier-e2e
  toolchain fixtures, which had relied on the `-toolchain` suffix for coverage capability, now
  declare an (unbound, never-executed) `gate_type: coverage` engine to stay coverage-capable
  hermetically.
