---
title: "SPEC-033: Engine ↔ BUNDLE-009 Contract Seam"
number: SPEC-033
created: "2026-06-16"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Lock the contract seam between BUNDLE-010 (pluggable pack engines) and
    BUNDLE-009 (stack-aware traceability). This is a boundary-definition /
    contract-lock deliverable, NOT engine implementation code. BUNDLE-010 ships
    the engine machinery (SPEC-031), the fixture-time generalization (SPEC-032),
    ast-grep wired as the first new engine, a single trivial proof rule, and the
    reusable ast-grep→SARIF converter authored once here. BUNDLE-009 authors the
    real substantiveness/contract rule packs on top of that machinery and reuses
    the converter. This spec pins exactly which artifacts cross the seam, asserts
    they exist and are reusable, and asserts what this bundle explicitly does NOT
    deliver (the real rules) so neither side absorbs the other's work.
  package: specs

verification:
  level: static
  test_command: go run ./tools/seam-check ./specs/SPEC-033-engine-bundle-009-seam.spec.md

requirements:
  - id: REQ-001
    text: >
      The BUNDLE-009 contract seam must be explicitly locked as a named set of
      hand-off artifacts. BUNDLE-010 delivers, and BUNDLE-009 consumes, exactly
      this set: (1) a first-class `engine` field on a pack rule dispatched through
      the engine table; (2) ast-grep wired end-to-end through the gate as a
      registered engine; (3) the reusable ast-grep stdin→SARIF `convert`
      executable; (4) the `parseSarif` output contract that every findings engine
      normalizes to; (5) the structured `input_mode`/`input_flag` rule-injection
      seam and engine-organized pack layout convention. No additional artifact may
      be claimed as crossing this seam.
    supports: pluggable-pack-engines:REQ-017
    follows: prompts-are-vibes-recipe
  - id: REQ-002
    text: >
      The seam must include exactly one trivial proof rule, and that proof rule's
      sole purpose is to demonstrate engine dispatch + the `convert` step working
      from declaration through to normalized violations. The proof rule is NOT a
      real substantiveness or contract check and must not be presented, reused, or
      extended as one. It is a wiring witness, not domain logic.
    supports: pluggable-pack-engines:REQ-017
  - id: REQ-003
    text: >
      The seam must explicitly enumerate what BUNDLE-010 does NOT deliver, and
      assign that work to BUNDLE-009: (a) the real substantiveness rule packs;
      (b) the real contract/traceability rule packs; (c) wiring those packs into
      the gate's traceability steps; (d) migrating the baked-in Go `go/parser`
      substantiveness analyzer onto the pack model — BUNDLE-009-assigned but
      explicitly deferrable ("later," per the bundle's Non-Goal #3 / DD-6 dogfood
      note), i.e. owned by BUNDLE-009 but not a near-term commitment.
      BUNDLE-010 ships none of these.
      A finding that this spec or any sibling BUNDLE-010 spec contains a real
      substantiveness/contract rule is a seam violation.
    supports: pluggable-pack-engines:REQ-017
  - id: REQ-004
    text: >
      The ast-grep→SARIF `convert` executable must be authored exactly once, by
      BUNDLE-010, as a standalone stdin→SARIF script, and must be reusable by
      BUNDLE-009's ast-grep rule packs without modification. BUNDLE-009 must not
      author a second ast-grep converter. The converter is the one ast-grep gap
      BUNDLE-010 owns (DD-7); all other engines reach SARIF via native output or
      existing third-party converters.
    supports: pluggable-pack-engines:REQ-017
  - id: REQ-005
    text: >
      The seam must declare directional ownership and the dependency edge: this
      spec (and the BUNDLE-010 engine machinery) is the producer; BUNDLE-009 is the
      consumer. BUNDLE-009's query-pack layer is blocked on ast-grep being wired
      here; BUNDLE-010 carries no reciprocal dependency on BUNDLE-009 and must not
      block on it. Neither bundle's specs may absorb the other's requirements.
    supports: pluggable-pack-engines:REQ-017

claims:
  # REQ-001 — the delivered hand-off artifact set (every "Delivered" ledger row)
  - id: CLM-001
    requirement: REQ-001
    text: Seam enumerates the first-class engine field + dispatch table as a delivered artifact
    tests:
      - TestSeam_Delivers_EngineFieldAndDispatchTable
  - id: CLM-002
    requirement: REQ-001
    text: Seam enumerates ast-grep wired end-to-end as a delivered artifact
    tests:
      - TestSeam_Delivers_AstGrepWired
  - id: CLM-003
    requirement: REQ-001
    text: Seam enumerates the ast-grep stdin-to-SARIF convert executable as a delivered artifact
    tests:
      - TestSeam_Delivers_AstGrepConverter
  - id: CLM-004
    requirement: REQ-001
    text: Seam enumerates the parseSarif output contract as a delivered artifact
    tests:
      - TestSeam_Delivers_ParseSarifContract
  - id: CLM-005
    requirement: REQ-001
    text: Seam enumerates the input_mode/input_flag injection seam + pack layout as a delivered artifact
    tests:
      - TestSeam_Delivers_InputModeAndLayout
  - id: CLM-006
    requirement: REQ-001
    text: Seam declares the delivered set closed — no artifact outside the enumerated list crosses as delivered
    tests:
      - TestSeam_DeliveredSet_IsClosed

  # REQ-002 — exactly one proof rule, marked non-domain (pass + negative)
  - id: CLM-007
    requirement: REQ-002
    text: Seam declares exactly one trivial proof rule as the wiring witness
    tests:
      - TestSeam_ProofRule_ExactlyOne
  - id: CLM-008
    requirement: REQ-002
    text: Seam labels the proof rule as a wiring witness, not a substantiveness or contract rule
    tests:
      - TestSeam_ProofRule_LabeledNonDomain
  - id: CLM-009
    requirement: REQ-002
    text: A second proof rule, or a proof rule presented as a real rule, is a seam violation
    tests:
      - TestSeam_ProofRule_RejectsExtraOrPromoted

  # REQ-003 — the NOT-delivered set, each row assigned to BUNDLE-009 (every "NOT delivered" ledger row)
  - id: CLM-010
    requirement: REQ-003
    text: Real substantiveness rule packs are assigned to BUNDLE-009, not delivered by BUNDLE-010
    tests:
      - TestSeam_NotDelivered_SubstantivenessPacks
  - id: CLM-011
    requirement: REQ-003
    text: Real contract/traceability rule packs are assigned to BUNDLE-009, not delivered by BUNDLE-010
    tests:
      - TestSeam_NotDelivered_ContractPacks
  - id: CLM-012
    requirement: REQ-003
    text: Wiring rule packs into the gate's traceability steps is assigned to BUNDLE-009
    tests:
      - TestSeam_NotDelivered_TraceabilityStepWiring
  - id: CLM-013
    requirement: REQ-003
    text: Migrating the Go go/parser substantiveness analyzer to packs is assigned to BUNDLE-009
    tests:
      - TestSeam_NotDelivered_GoParserMigration
  - id: CLM-014
    requirement: REQ-003
    text: A real substantiveness or contract rule appearing in any BUNDLE-010 spec is flagged as a seam violation
    tests:
      - TestSeam_RealRuleInBundle010_IsViolation

  # REQ-004 — converter authored once, reused, never re-authored (pass + negative)
  - id: CLM-015
    requirement: REQ-004
    text: The ast-grep converter is declared author-once by BUNDLE-010
    tests:
      - TestSeam_Converter_AuthoredOnceByBundle010
  - id: CLM-016
    requirement: REQ-004
    text: BUNDLE-009 reuses the converter without modification
    tests:
      - TestSeam_Converter_ReusedUnmodified
  - id: CLM-017
    requirement: REQ-004
    text: A second / BUNDLE-009-authored ast-grep converter is a seam violation
    tests:
      - TestSeam_Converter_RejectsReauthoring

  # REQ-005 — one-way producer->consumer dependency edge (pass + negative)
  - id: CLM-018
    requirement: REQ-005
    text: Seam declares BUNDLE-010 as producer and BUNDLE-009 as consumer of ast-grep wiring
    tests:
      - TestSeam_Direction_ProducerConsumerDeclared
  - id: CLM-019
    requirement: REQ-005
    text: Seam records that BUNDLE-009's query-pack layer is blocked on ast-grep wired here
    tests:
      - TestSeam_Direction_ConsumerBlockedOnAstGrep
  - id: CLM-020
    requirement: REQ-005
    text: A BUNDLE-010 dependency on BUNDLE-009 (reciprocal edge) is a seam violation
    tests:
      - TestSeam_Direction_RejectsReciprocalDependency

contracts:
  - file: specs/SPEC-033-engine-bundle-009-seam.spec.md
    provides:
      - name: BUNDLE-009-engine-seam
        kind: constant
        signature: "contract-lock: BUNDLE-010 producer → BUNDLE-009 consumer hand-off artifact set"
        notes: >
          This spec is the authoritative, citable boundary contract between
          BUNDLE-010 (producer) and BUNDLE-009 (consumer). It declares no
          package symbols; its "provides" is the frozen seam ledger itself.
    consumes:
      # Each name below resolves to a real provides.name in the cited source
      # (SPEC-031/032 provides blocks) or a concrete symbol/file. The seam cites
      # the symbols that actually carry the hand-off, not aspirational labels.
      - source: specs/SPEC-031-pluggable-engine-dispatch.spec.md
        name: dispatchPackEngines
        kind: function
        notes: >
          The engine dispatch path (group-by-engine → gather inputs → run →
          convert → parseSarif → namespace). This is the "first-class engine
          field + dispatch table" hand-off artifact; BUNDLE-009 rules ride it by
          declaring `engine:`.
      - source: specs/SPEC-031-pluggable-engine-dispatch.spec.md
        name: Registry
        kind: type
        notes: >
          The engine binding registry that seeds ast-grep (REQ-010). "ast-grep
          wired end-to-end" is delivered as the ast-grep entry in this Registry
          plus its dispatch through dispatchPackEngines; there is no separate
          named ast-grep symbol to cite.
      - source: specs/SPEC-031-pluggable-engine-dispatch.spec.md
        name: EngineBinding
        kind: type
        notes: >
          The per-engine binding ({command, input_mode, input_flag, convert?,
          provision?}). The ast-grep stdin→SARIF converter is carried by this
          binding's `Convert` field (a pack-resident script per REQ-008, DD-7),
          not a separately exported symbol — SPEC-031 declares the converter in
          prose, so the seam cites the binding field that references it.
      - source: specs/SPEC-032-pack-fixture-engine-execution.spec.md
        name: RunEngine
        kind: method
        notes: >
          The fixture-time engine execution method (FixtureExecutor.RunEngine).
          This is the "fixture-engine-execution" hand-off; it normalizes to the
          same parseSarif contract as the gate path.
      - source: pkg/check/parsers.go
        name: parseSarif
        kind: function
---

# SPEC-033: Engine ↔ BUNDLE-009 Contract Seam

## Overview

This spec is a **contract-lock / boundary-definition** deliverable. It produces
no new package code. Its job is to pin the seam between two bundles so that the
work decomposes cleanly and neither side silently absorbs the other's scope.

- **BUNDLE-010 (pluggable pack engines)** is the *producer*. It ships the engine
  machinery (SPEC-031), the fixture-time generalization (SPEC-032), ast-grep
  wired as the first new engine, a single *trivial proof rule*, and the reusable
  ast-grep→SARIF converter authored once.
- **BUNDLE-009 (stack-aware traceability)** is the *consumer*. It authors the
  *real* substantiveness and contract/traceability rule packs on top of that
  machinery, reusing the converter and the `parseSarif` output contract.

**Without this spec:** the two bundles' scopes blur. SPEC-031/032 could quietly
grow a "real" substantiveness rule (because the machinery is right there), or
BUNDLE-009 could re-author the ast-grep converter (because nobody declared it
already exists). Both are integration-gap drift.

**With this spec:** the hand-off artifact set is enumerated and frozen, the proof
rule is explicitly marked as a non-domain wiring witness, the converter is
declared author-once / reuse-many, and the work BUNDLE-010 does NOT do is
assigned to BUNDLE-009 by name.

**Verification Level:** Static — this is a contract; verification asserts artifact
presence and seam consistency, not runtime behavior. The substantive runtime
behavior is verified by SPEC-031 (dispatch + convert + ast-grep proof) and
SPEC-032 (fixture-time execution); this spec must not re-verify it.

**Is a full spec the right shape here?** A contract-lock is a thin spec, but a
spec is still the correct artifact: the id is reserved, BUNDLE-009's planning
depends on a stable, traceable reference for the seam, and the "what we do NOT
deliver" allowlist needs a citable home that the impl-reviewer and the BUNDLE-009
author can both point at. A section buried inside SPEC-031 would couple the
boundary contract to the machinery's lifecycle and make it easy to edit away.
Keeping it a standalone `draft` spec keeps the boundary independently citable.

## Requirements

Requirements and claims are defined in frontmatter. Each requirement traces to
`pluggable-pack-engines:REQ-017` (the bundle's Seed 4 requirement) and the OQ-6 /
DD-6 resolution that scoped this bundle as "two pillars + a locked BUNDLE-009
seam."

### The Seam Ledger

The seam is a two-column ledger. The left column is everything BUNDLE-010
delivers across the seam; the right column is everything BUNDLE-010 explicitly
does NOT deliver, which BUNDLE-009 owns. Every artifact lives in exactly one
column — no artifact is ambiguous, none is shared-by-omission.

| Artifact | Side | Owner |
|---|---|---|
| First-class `engine` field + dispatch table | Delivered | BUNDLE-010 (SPEC-031) |
| ast-grep wired end-to-end as a registered engine | Delivered | BUNDLE-010 (SPEC-031) |
| ast-grep stdin→SARIF `convert` executable (author-once) | Delivered | BUNDLE-010 (SPEC-031) |
| `parseSarif` output contract for findings engines | Delivered | BUNDLE-010 (existing, ISSUE-003) |
| `input_mode`/`input_flag` injection seam + pack layout | Delivered | BUNDLE-010 (SPEC-031) |
| Exactly one trivial proof rule (wiring witness) | Delivered | BUNDLE-010 (SPEC-031) |
| Real substantiveness rule packs | NOT delivered | BUNDLE-009 |
| Real contract/traceability rule packs | NOT delivered | BUNDLE-009 |
| Wiring rule packs into the gate's traceability steps | NOT delivered | BUNDLE-009 |
| Migrating the Go `go/parser` substantiveness analyzer to packs | NOT delivered | BUNDLE-009 (deferrable / "later") |
| A second / BUNDLE-009-authored ast-grep converter | NOT delivered | (forbidden — reuse BUNDLE-010's) |

This table is the authoritative expression of the allowlist. The frontmatter
requirements and claims must match it exactly; if they diverge, the table is the
bug.

## Implementation

There is no package implementation. The "implementation" of a contract-lock is
the hand-off artifact set and the boundary assertions, enumerated below. The
sibling BUNDLE-010 specs are where the corresponding *code* lands; this spec only
declares the boundary they must respect.

### Hand-off artifacts produced by BUNDLE-010 (delivered across the seam)

1. **First-class `engine` field + dispatch table.** A pack rule declares its
   engine; the gate dispatches through the explicit
   `engine → {command, convert?, requires[], forbids[]}` table. Authored in
   SPEC-031. BUNDLE-009 consumes this by declaring `engine: ast-grep` on its
   rules.
2. **ast-grep wired end-to-end.** ast-grep is a registered engine reaching the
   gate from declaration through to normalized violations, via the `rule-dir`
   `input_mode`. Authored in SPEC-031. BUNDLE-009's query-pack layer is unblocked
   the moment this lands.
3. **ast-grep stdin→SARIF `convert` executable.** Authored exactly once here as a
   standalone script (DD-7), resolved relative to the pack dir and run via the
   existing `SandboxedRun` layer. BUNDLE-009 reuses it verbatim; it does not
   author its own.
4. **`parseSarif` output contract.** Every findings engine normalizes to SARIF;
   `parseSarif` (pkg/check/parsers.go, from ISSUE-003) is the sole owned parser.
   BUNDLE-009's rule packs inherit this contract for free.
5. **`input_mode`/`input_flag` injection seam + engine-organized pack layout.**
   The declared mechanism (`config-file` / `rule-flags` / `rule-dir` / `none`)
   and the directory-per-engine layout convention. BUNDLE-009 lays out its packs
   per this convention.

### The single proof rule (wiring witness)

BUNDLE-010 ships **one** trivial ast-grep proof rule. Its only job is to prove
dispatch + convert + parseSarif work end-to-end: it fires deterministically on a
fixture, flows through the converter, and surfaces as a normalized violation. It
encodes no real substantiveness or contract semantics. It is a witness that the
machinery works, deletable the moment BUNDLE-009's real rules land, and must
never be cited as a substantiveness/contract rule.

### Work NOT delivered by BUNDLE-010 (assigned to BUNDLE-009)

The following are out of scope for *every* BUNDLE-010 spec and belong to
BUNDLE-009. Their presence in this or any sibling BUNDLE-010 spec is a seam
violation:

- the real substantiveness rule packs;
- the real contract/traceability rule packs;
- wiring those packs into the gate's traceability steps;
- migrating the baked-in Go `go/parser` substantiveness analyzer
  (step_testverify.go) onto the pack model — BUNDLE-009-owned but deferrable
  ("later," per the bundle's Non-Goal #3 / DD-6 dogfood note); the seam fixes
  ownership, not timing, so this is not a near-term BUNDLE-009 commitment.

### Directional ownership

BUNDLE-010 → BUNDLE-009 is a one-way producer→consumer edge. BUNDLE-009 depends
on ast-grep being wired here; BUNDLE-010 carries no reciprocal dependency and
must not block on BUNDLE-009. This direction is the load-bearing reason the
engine work was pulled *ahead* of BUNDLE-009 (DD-6).

## Verification

Verification is defined in frontmatter at the `static` level — appropriate for a
contract-lock with no runtime behavior of its own. Claims assert seam
consistency: that the delivered artifacts are enumerated, that exactly one proof
rule is declared, that the converter is author-once, that the not-delivered set
is assigned to BUNDLE-009, and that no sibling BUNDLE-010 spec contains a real
substantiveness/contract rule.

Runtime behavior of the delivered machinery (dispatch, convert, ast-grep proof,
fixture execution) is verified by **SPEC-031** and **SPEC-032** and must not be
re-verified here — doing so would mean this spec absorbed their work, which is
itself the seam violation this spec exists to prevent.

## Sharp Edges

1. **Proof rule masquerading as a real rule.** The single trivial proof rule is
   structurally identical to a real ast-grep rule (it dispatches, converts,
   normalizes). The temptation is to "just make it check something useful" and
   call substantiveness done. That silently moves BUNDLE-009 work into BUNDLE-010
   and leaves the seam undocumented. The proof rule must stay trivial and be
   explicitly labeled a wiring witness; a useful-looking rule here is a defect.

2. **Converter re-authoring.** Because the ast-grep converter is "just a script,"
   BUNDLE-009 could re-write its own instead of reusing BUNDLE-010's, producing
   two diverging converters and two SARIF dialects. The seam must name the
   converter as author-once / reuse-many so the BUNDLE-009 author knows it
   already exists.

3. **Scope osmosis from the machinery being right there.** SPEC-031/032 hold the
   engine machinery; once it works, authoring a real rule is one file away. The
   gravitational pull is to keep going past the proof rule. The "NOT delivered"
   allowlist exists precisely to make that boundary a citable hard stop.

4. **Bidirectional-dependency drift.** If anyone adds a BUNDLE-010 dependency on
   BUNDLE-009 (e.g., "wait for the real rules to validate the engine"), the
   producer→consumer edge inverts and both bundles deadlock. The proof rule, not
   the real rules, is what validates the engine — that is the whole reason it
   exists.

5. **`go/parser` analyzer mis-assignment.** The baked-in Go substantiveness
   analyzer (step_testverify.go) is a "stop baking rules into the binary" anomaly
   that *looks* like pillar-2's standards-compiler removal but is distinct code
   and belongs to BUNDLE-009. Migrating it under a BUNDLE-010 spec because both
   are "unbaking" is a classification error this seam forbids.

## Review Questions

1. Does any sibling BUNDLE-010 spec (SPEC-030/031/032) declare a rule that
   performs a real substantiveness or contract check, rather than the single
   trivial proof rule? If so, that work has crossed the seam into BUNDLE-009.

2. Is the ast-grep→SARIF converter declared exactly once (in SPEC-031) and
   referenced — not re-authored — anywhere BUNDLE-009 needs it? Two converter
   implementations is a defect even if both work.

3. Is there exactly one proof rule, and is it labeled as a wiring witness (not a
   substantiveness/contract rule) at every place it is referenced?

4. Does any BUNDLE-010 artifact declare a dependency on BUNDLE-009? The edge must
   be one-way (BUNDLE-009 depends on BUNDLE-010), or the bundles deadlock.

5. Is the `go/parser` substantiveness analyzer migration assigned to BUNDLE-009
   and absent from every BUNDLE-010 spec's scope?

## References

- **BUNDLE-010** (pluggable-pack-engines) — REQ-017, DD-6 (two pillars + locked
  seam), DD-7 (own only the ast-grep converter), Out of Scope / Non-Goals items
  1–3 (the BUNDLE-009-owned work).
- **BUNDLE-009** (stack-aware-traceability) — the consumer; its query-pack layer
  is gated on ast-grep wired here.
- **SPEC-031** (pluggable-engine-dispatch) — the engine machinery, ast-grep
  wiring, proof rule, and converter this seam hands off.
- **SPEC-032** (pack-fixture-engine-execution) — the fixture-time generalization;
  also bound by this seam's "no real rules" allowlist.
- **ISSUE-003** — `parseSarif` / `formatParsers` / `lookupParser`, the SARIF
  output contract inherited across the seam.
