---
title: "Legacy Reconciliation Version Backfill"
number: SPEC-051
created: "2026-07-14"
status: implemented
schema_version: spec/v1
spec_version: 1.3.1

implementation:
  summary: >
    Seed 2 of BUNDLE-014 (requirement-traceability): the ONE-TIME legacy corpus
    reconciliation + `1.0.0` version-backfill sweep that makes the artifact corpus
    uniformly explicit and GREEN at the exact moment SPEC-050 (Seed 1) flips
    mandatory-pin + both-direction resolution ON. This spec is CORPUS work, not a
    new code path: it enumerates and mandates five reconciliation actions plus an
    ordering constraint, and proves the end state with corpus-assertion tests that
    run SPEC-050's `pkg/validate` resolution + version-log validation over the REAL
    project corpus. The five actions: (1) DEPRECATE the delivered `agent-definitions`
    bundle (maturity → `deprecated`), DROPPING the once-planned requirements[] backfill
    — keeping it `delivered` would demand 18 REQs of implemented-spec coverage on thin
    anchors (fake rigor) that SPEC-052's delivered-gate would block, whereas a terminal
    bundle is exempt from the delivered/requirements[] gates and creates zero coverage
    debt (it is scaffolding-era history superseded by the living `.claude` agent roster
    + the agent-guard hook + the ISSUE-044 roster-consistency check); (2) RE-STATE
    BUNDLE-011's REQ-004 (today supported by nothing) onto the live `implemented`
    SPEC-040, and REQ-007/REQ-010 (today supported only by the `replaced` SPEC-039) onto
    the new `implemented` retroactive spec SPEC-053 — SPEC-039 stays byte-for-byte
    untouched; (3) terminally RETIRE the rest of the legacy scaffolding cluster to
    `deprecated` alongside the bundle — the stale draft specs SPEC-002/003/004 (they
    cite only `agent-definitions:REQ-004..018`; retiring them AND the now-terminal bundle
    clears those dangling refs from live validation), NOT promote them to `implemented`;
    (4) STAMP `version: "1.0.0"` on every REQ of every non-terminal bundle declaring
    `requirements[]`; (5) STAMP
    the exact current-version pin onto every existing `supports` ref on every live
    (non-terminal) spec/issue requirement — `@1.0.0` for the initial backfill, and the
    REQ's amended current version where one shipped (SPEC-051's own refs to
    `requirement-traceability:REQ-015` pin `@1.1.0`, that REQ having been amended via
    RDQ-2, per the RDQ-7 stale-pin model) — the Seed specs SPEC-050/051/052 included
    (self-referential corpus). The whole sweep lands WITH or immediately before
    SPEC-050's enforcement flip (a merge-sequencing constraint, not a runtime toggle)
    so the corpus never sits red, and every edit is routed through the artifact
    agents / CLI — never hand-edited. Covers BUNDLE-014 REQ-015 ONLY; the versioning
    schema + resolution mechanism (Seed 1 / SPEC-050) and the coverage gate step +
    stale-pin model (Seed 3 / SPEC-052) are out of scope. Depends on SPEC-050's
    `pkg/validate` resolution + version-log functions existing (the assertion tests
    call them).
  subject: pkg/validate

verification:
  level: integration
  test_command: go test ./pkg/validate/... -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      The delivered `agent-definitions` bundle must be moved to the terminal maturity
      `deprecated` (an optional free-text reason MAY cite its supersession by the living
      `.claude` agent roster + the agent-guard hook + the ISSUE-044 roster-consistency
      check), and the once-planned `requirements[]` backfill must be DROPPED — the
      bundle carries no `requirements[]` array. Rationale: keeping it `delivered` under
      SPEC-050 REQ-005 (requirements[] required at `delivered`) plus SPEC-052's
      delivered-coverage gate (every bundle REQ covered by ≥1 `implemented` spec
      requirement) would demand 18 backfilled REQs each with an `implemented` supporter,
      but its only citers (SPEC-002/003/004) retire, so the backfill would be delivered
      with ZERO implemented-spec coverage and SPEC-052 would block it. A terminal bundle
      is exempt from BOTH the requirements[] gate (pkg/validate/bundle.go terminal
      exemption) and the delivered-coverage gate (terminal artifacts are excluded), so
      deprecating creates zero coverage debt — honest recording, not grandfathering.
    supports: requirement-traceability:REQ-015@1.1.0
    follows: STD-GO-001:GO-010
  - id: REQ-002
    text: >
      BUNDLE-011's REQ-004 (currently supported by NOTHING) and REQ-007 and REQ-010
      (currently supported ONLY by the `replaced` SPEC-039) must each be re-homed so
      that after the sweep each is supported by at least one requirement in an
      `implemented` spec, per DD-2/DD-3 (support cannot flow through a `replaced` spec,
      and an issue can never satisfy coverage). Two DISTINCT homes, named by exact
      target requirement id: (a) REQ-004 (one `<lang>-toolchain` pack; route
      lint/build/test through declared pack passes) is APPENDED to the `supports` LIST
      of SPEC-040 REQ-001 (the implemented keystone cutover's toolchain-routing
      requirement) — that trace is faithful. (b) REQ-007 (delete the dead non-Go
      `routeFileDefaults` semgrep catch-all) and REQ-010 (delete the dead
      standards-manifest reader) are NOT re-homed on SPEC-040 — SPEC-040's scope fence
      explicitly DISOWNS those deletions (they were SPEC-039 / Seed-1 scope) and the
      work actually landed via ISSUE-018 (SPEC-039's `replaced-by`), which is an issue
      and cannot provide coverage. They are instead covered by SPEC-053 REQ-007 and
      SPEC-053 REQ-010 (whose ids deliberately mirror the BUNDLE-011 ids they support) —
      the new `implemented` retroactive spec
      SPEC-053-retroactive-codecheck-deadcode-deletions, authored in parallel, whose
      claims anchor to the ISSUE-018 removal tests on `main`. The SPEC-040 change must
      ADD a ref to SPEC-040 REQ-001's `supports` list only (no new requirement, no new
      claim, no code change on SPEC-040), and the `replaced` SPEC-039 must be left
      byte-for-byte unchanged (its refs are not re-pinned and not re-homed).
    supports: requirement-traceability:REQ-015@1.1.0
    follows: STD-GO-001:GO-010
  - id: REQ-003
    text: >
      The stale draft specs SPEC-002, SPEC-003, and SPEC-004 (all `draft`, each citing
      only `agent-definitions:REQ-004..REQ-018`) must be moved to the terminal end-of-life
      status `deprecated`, retired together with the `agent-definitions` bundle (REQ-001)
      as one legacy scaffolding cluster (retired without a single named successor spec —
      the scope shipped via the bundle's direct delivery, so `canceled`/work-abandoned
      would be dishonest and `replaced`/`obsoleted` would require a successor pointer none
      of them has), matching the ISSUE-031 terminal-state model and the corpus precedent
      (SPEC-001, SPEC-011 are `deprecated`). `deprecated` requires no retirement pointer
      field; an optional free-text reason MAY be recorded. Retiring the specs removes
      their dangling `agent-definitions` refs from live validation via the terminal-citer
      exemption, and — since the bundle they cite is itself now terminal — those refs have
      no live resolution obligation either. They must NOT be promoted to `implemented`.
    supports: requirement-traceability:REQ-015@1.1.0
    follows: STD-GO-001:GO-010
  - id: REQ-004
    text: >
      Every requirement in every NON-TERMINAL bundle that declares a `requirements[]`
      array must carry `version: "1.0.0"` (its initial version-log entry) so that
      SPEC-050 REQ-004's per-REQ version + version-log validation passes corpus-wide.
      The affected bundles are BUNDLE-004, BUNDLE-005, BUNDLE-006, BUNDLE-007,
      BUNDLE-009, BUNDLE-010, BUNDLE-011, BUNDLE-012, BUNDLE-013, and the `cli` bundle
      (ten bundles); BUNDLE-014 already carries `version: "1.0.0"` on every REQ (it
      dogfooded the convention) and needs no stamp. The now-`deprecated`
      `agent-definitions` bundle (REQ-001) is terminal and carries no `requirements[]`,
      so it is NOT stamped. Terminal bundles (`agent-definitions`, and the `deprecated`
      `runtime-hooks` and `standards-compiler`) are exempt via the terminal-state
      exemption and must be left untouched.
    supports: requirement-traceability:REQ-015@1.1.0
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      Every existing `supports` ref on a LIVE (non-terminal) spec or issue
      requirement must gain an exact version pin (`bundle-name:REQ-NNN@MAJOR.MINOR.PATCH`)
      matching the CURRENT version of the bundle REQ it cites — `@1.0.0` for the initial
      `1.0.0` backfill (the overwhelming majority), and the REQ's amended current version
      where one has already shipped (e.g. `requirement-traceability:REQ-015`, amended to
      `1.1.0` via RDQ-2, so its citers pin `@1.1.0` per the RDQ-7 stale-pin model) — so
      that SPEC-050 REQ-002's mandatory-pin format check and REQ-003's log-match
      resolution both pass corpus-wide — roughly 370 refs across ~33 live specs as the
      corpus stands with all Seed specs authored (the exact set is defined by the rule
      below, not a frozen count). The affected set is every spec carrying `supports`
      refs EXCEPT (a) the retiring SPEC-002/003/004 (whose refs cease to be validated
      on retirement, REQ-003) and (b) ALL already-terminal specs — SPEC-008 and
      SPEC-034 (`replaced`), SPEC-039 (`replaced`, whose refs stay untouched per
      REQ-002), and SPEC-011 (`deprecated`) — whose refs are exempt from both the
      supports-format check and resolution (SPEC-050's terminal exclusion). No issue
      currently carries a `supports` ref. The Seed specs SPEC-050, SPEC-051 (this
      spec), SPEC-052, and SPEC-053 are themselves in the live set — the sweep stamps
      their own refs too (self-referential corpus).
    supports: requirement-traceability:REQ-015@1.1.0
    follows: STD-GO-001:GO-010
  - id: REQ-006
    text: >
      The entire sweep must land WITH or immediately before SPEC-050's mandatory-pin
      enforcement flip so the corpus never sits red at any committed point: because
      the tightened `supportsRe` rejects an UNPINNED ref while the CURRENT `supportsRe`
      rejects a PINNED (`@1.0.0`) ref, the corpus edits (REQ-001..REQ-005) and
      SPEC-050's regex/resolution change are a single atomic change-train — a
      merge-sequencing constraint, NOT a runtime config toggle and NOT a grandfather
      path. The end state must be: `./bin/backstop artifact validate` GREEN with
      SPEC-050's resolution + mandatory-pin enforcement unconditionally ON, every
      `supports` ref resolving (real bundle, declared REQ, pin in the version log),
      zero grandfathering. All edits must be routed through the artifact agents / CLI
      conventions (never hand-edited).
    supports: requirement-traceability:REQ-015@1.1.0
    follows: STD-GO-001:GO-010

claims:
  # REQ-001 — deprecate the agent-definitions bundle (backfill dropped)
  - id: CLM-001
    requirement: REQ-001
    text: The agent-definitions bundle is at terminal maturity deprecated after the sweep
    tests:
      - TestReconcile_AgentDefsDeprecated
  - id: CLM-002
    requirement: REQ-001
    text: The deprecated agent-definitions validates clean with NO requirements[] array — the terminal exemption skips the requirements-required gate (no bundle requirements-missing violation)
    tests:
      - TestReconcile_AgentDefsTerminalExemptFromRequirementsGate
  - id: CLM-003
    requirement: REQ-001
    text: The agent-definitions bundle carries no backfilled requirements[] — the once-planned 18-REQ backfill was dropped (negative guard)
    tests:
      - TestReconcile_AgentDefsNoBackfilledRequirements
  - id: CLM-004
    requirement: REQ-001
    text: The deprecated agent-definitions is recognized as terminal (isTerminalStatus), so it is outside the delivered-coverage evaluation and creates zero coverage debt — the F1 seam with SPEC-052 is resolved
    tests:
      - TestReconcile_AgentDefsTerminalOutsideCoverageEval

  # REQ-002 — BUNDLE-011 re-statement onto live implemented SPEC-040; SPEC-039 untouched
  - id: CLM-005
    requirement: REQ-002
    text: BUNDLE-011 REQ-004 is covered by SPEC-040 REQ-001 (implemented) after the append to its supports list
    tests:
      - TestReconcile_Bundle011Req004CoveredBySpec040Req001
  - id: CLM-006
    requirement: REQ-002
    text: BUNDLE-011 REQ-007 is covered by an implemented spec requirement (SPEC-053 REQ-007), not by SPEC-040 and not by the replaced SPEC-039
    tests:
      - TestReconcile_Bundle011Req007CoveredBySpec053
  - id: CLM-007
    requirement: REQ-002
    text: BUNDLE-011 REQ-010 is covered by an implemented spec requirement (SPEC-053 REQ-010), not by SPEC-040 and not by the replaced SPEC-039
    tests:
      - TestReconcile_Bundle011Req010CoveredBySpec053
  - id: CLM-008
    requirement: REQ-002
    text: The replaced SPEC-039 is left unchanged — still status replaced, its two supports refs neither re-pinned nor re-homed (negative guard)
    tests:
      - TestReconcile_Spec039LeftUntouched

  # REQ-003 — terminal retirement of the stale draft specs
  - id: CLM-009
    requirement: REQ-003
    text: SPEC-002 is at terminal status deprecated after the sweep
    tests:
      - TestReconcile_Spec002Deprecated
  - id: CLM-010
    requirement: REQ-003
    text: SPEC-003 is at terminal status deprecated after the sweep
    tests:
      - TestReconcile_Spec003Deprecated
  - id: CLM-011
    requirement: REQ-003
    text: SPEC-004 is at terminal status deprecated after the sweep
    tests:
      - TestReconcile_Spec004Deprecated
  - id: CLM-012
    requirement: REQ-003
    text: The retired drafts validate clean under the terminal exemption — no completeness or supports-format violation is raised for their dangling refs
    tests:
      - TestReconcile_RetiredDraftsExemptFromDanglingRefs
  - id: CLM-013
    requirement: REQ-003
    text: None of SPEC-002/003/004 was promoted to implemented (negative guard — they were retired, not promoted)
    tests:
      - TestReconcile_RetiredDraftsNotImplemented

  # REQ-004 — corpus-wide 1.0.0 version stamp on bundle REQs
  - id: CLM-014
    requirement: REQ-004
    text: Every requirement of every non-terminal bundle declaring requirements[] carries a well-formed semver version after the sweep
    tests:
      - TestReconcile_AllNonTerminalBundleReqsVersioned
  - id: CLM-015
    requirement: REQ-004
    text: Running the version-log validation over the whole corpus yields zero version-log violations
    tests:
      - TestReconcile_NoBundleVersionLogViolationsCorpusWide

  # REQ-005 — corpus-wide @1.0.0 pin stamp on live supports refs
  - id: CLM-016
    requirement: REQ-005
    text: Every supports ref on a live (non-terminal) spec requirement carries an exact @MAJOR.MINOR.PATCH pin after the sweep
    tests:
      - TestReconcile_AllLiveSupportsRefsPinned
  - id: CLM-017
    requirement: REQ-005
    text: The tightened supportsRe raises zero pin-format violations across live specs (no unpinned ref survives)
    tests:
      - TestReconcile_NoUnpinnedSupportsRefViolations
  - id: CLM-018
    requirement: REQ-005
    text: The Seed specs SPEC-050, SPEC-051, and SPEC-052 carry their own supports refs pinned at the CURRENT version of each bundle REQ they implement — @1.0.0 for refs to requirement-traceability:REQ-001..014, and @1.1.0 for SPEC-051's refs to requirement-traceability:REQ-015 (amended via RDQ-2; an in-flight bundle's downstream citers rev to the latest on a minor bump per the RDQ-7 stale-pin model) — self-referential corpus
    tests:
      - TestReconcile_SeedSpecsSelfPinnedAt100

  # REQ-006 — atomic co-landing + green end state under enforcement ON
  - id: CLM-019
    requirement: REQ-006
    text: Both-direction resolution over the real corpus with mandatory-pin enforcement ON raises zero unresolved-ref violations (bundle exists, REQ declared, pin in log)
    tests:
      - TestReconcile_CorpusResolvesGreenUnderEnforcement
  - id: CLM-020
    requirement: REQ-006
    text: A still-unpinned ref and a fabricated-version ref each fail resolution under enforcement ON, proving the flip is genuinely on rather than grandfathered (negative)
    tests:
      - TestReconcile_UnpinnedAndFabricatedRefsStillRedUnderEnforcement
contracts:
  - file: pkg/validate/reconciliation_backfill_test.go
    consumes:
      - source: pkg/validate
        name: BuildBundleReqCatalog
        kind: function
      - source: pkg/validate
        name: CollectSupportRefs
        kind: function
      - source: pkg/validate
        name: ResolveSupports
        kind: function
      - source: pkg/artifact
        name: ParseFile
        kind: function
---

# SPEC-051: Legacy Reconciliation Version Backfill

## Overview

BUNDLE-014's Seed 1 (SPEC-050) tightens `supports` to a mandatory exact-semver pin
and adds a corpus-wide both-direction resolution pass. The instant that enforcement
turns on, the EXISTING corpus goes red: ~370 `supports` refs across ~33 live specs
are unpinned, the delivered `agent-definitions` bundle declares no `requirements[]`
(so 10 refs from SPEC-002/003/004 dangle), BUNDLE-011's REQ-004 is supported by
nothing and REQ-007/REQ-010 only by the `replaced` SPEC-039, and most bundle REQs
carry no `version:` field for the version log to resolve pins against.

This spec is Seed 2: the ONE-TIME reconciliation + `1.0.0` version-backfill sweep
(BUNDLE-014 REQ-015) that makes the corpus uniformly explicit and GREEN at the exact
moment SPEC-050's enforcement flips on. It is CORPUS work — it changes no gate/validate
code path (SPEC-050 owns the regex and the resolution functions); its deliverable is a
reconciled corpus plus corpus-assertion tests that prove the end state by running
SPEC-050's `pkg/validate` resolution + version-log validation over the REAL project
artifacts.

The sweep performs five actions and honours one ordering constraint:

1. **Deprecate the `agent-definitions` bundle (REQ-001).** Move it to terminal maturity
   `deprecated` and DROP the once-planned requirements[] backfill — a terminal bundle is
   exempt from the delivered/requirements[] gates, so it creates zero coverage debt
   (keeping it delivered with a backfilled-but-uncovered requirements[] would be blocked
   by SPEC-052). It is superseded by the living `.claude` roster + agent-guard + ISSUE-044.
2. **Re-home BUNDLE-011's orphaned REQs (REQ-002).** Append
   `collapse-legacy-codecheck-into-packs:REQ-004` onto SPEC-040 REQ-001 (faithful);
   REQ-007 and REQ-010 are covered by the new `implemented` retroactive spec SPEC-053
   (REQ-007, REQ-010) rather than SPEC-040 — SPEC-040 disowns those deletions and the
   work landed via ISSUE-018 (an issue, which can't cover). Leave the `replaced`
   SPEC-039 untouched.
3. **Terminally retire the rest of the cluster (REQ-003).** Move SPEC-002/003/004 to
   `deprecated` alongside the bundle (step 1); do not promote them.
4. **Stamp `1.0.0` on bundle REQs (REQ-004).** Add `version: "1.0.0"` to every REQ of
   every non-terminal bundle that declares `requirements[]`.
5. **Stamp the current-version pin on supports refs (REQ-005).** Append the pin matching
   each cited bundle REQ's current version to every `supports` ref on every live spec/issue
   requirement — `@1.0.0` for the initial backfill, `@1.1.0` for SPEC-051's own refs to
   `requirement-traceability:REQ-015` (amended via RDQ-2; per the RDQ-7 stale-pin model an
   in-flight bundle's downstream citers rev to the latest on a minor bump).
6. **Co-land atomically (REQ-006).** Ship the sweep in the same change-train as
   SPEC-050's flip so the corpus never sits red; end state is `artifact validate`
   green with enforcement unconditionally on.

This spec covers BUNDLE-014 REQ-015 ONLY. The versioning schema + resolution mechanism
(Seed 1, SPEC-050) and the `requirement_traceability` coverage gate step + stale-pin
model (Seed 3, SPEC-052) are explicitly out of scope. This spec's own `supports` ref was
written UNPINNED at first (like SPEC-050's) because the then-live `supportsRe` would have
rejected the pinned form; the sweep this spec describes stamps it to `@1.1.0` — the current
version of `requirement-traceability:REQ-015`, which was amended from `1.0.0` to `1.1.0`
via RDQ-2 (backfill → deprecate), so its downstream pins rev to the latest per the RDQ-7
stale-pin model (see Sharp Edges, "self-referential corpus").

## Requirements

Requirements REQ-001 through REQ-006 are defined in frontmatter and all trace to
BUNDLE-014 REQ-015 via `supports` (they decompose its five actions plus the ordering
constraint). Each has at least one claim; claims are defined in frontmatter.

### Reconciliation action map

| Action | What changes | Where | Requirement |
|--------|--------------|-------|-------------|
| Deprecate bundle | `delivered` → `deprecated`, backfill dropped (no `requirements[]`) | `agent-definitions.bundle.md` | REQ-001 |
| Re-home | `collapse…:REQ-004` → SPEC-040 REQ-001 (append); `:REQ-007`/`:REQ-010` → SPEC-053 REQ-007/REQ-010 | SPEC-040 + SPEC-053 | REQ-002 |
| Retire specs | `draft` → `deprecated` (cluster, with the bundle) | SPEC-002, SPEC-003, SPEC-004 | REQ-003 |
| Version stamp | `version: "1.0.0"` on every REQ | ten non-terminal bundles with `requirements[]` (agent-definitions excluded — terminal) | REQ-004 |
| Pin stamp | `@1.0.0` on every `supports` ref | all live specs/issues carrying `supports` | REQ-005 |
| Co-land | one atomic change-train with SPEC-050's flip | merge sequencing | REQ-006 |

### BUNDLE-011 re-home targets (REQ-002)

Current coverage (audited): REQ-004 → nothing; REQ-007, REQ-010 → the `replaced`
SPEC-039 only. Because support does not flow through a `replaced` spec (BUNDLE-014
DD-3) and issues can never satisfy coverage (DD-9), each must be re-homed onto a live
`implemented` spec requirement — but the two homes DIFFER, because SPEC-040 delivered
REQ-004's toolchain-routing while explicitly disowning the REQ-007/REQ-010 deletions
(SPEC-039 / Seed-1 scope, landed via ISSUE-018).

| BUNDLE-011 REQ | What it demanded | Home (exact requirement id) | Mechanism |
|----------------|------------------|-----------------------------|-----------|
| REQ-004 | one `<lang>-toolchain` pack; lint/build/test via declared pack passes | SPEC-040 REQ-001 (implemented) | append to its `supports` list |
| REQ-007 | delete the dead non-Go `routeFileDefaults` semgrep catch-all | SPEC-053 REQ-007 (implemented, retroactive) | carried by SPEC-053, authored in parallel |
| REQ-010 | delete the dead standards-manifest reader in `manifest.go` | SPEC-053 REQ-010 (implemented, retroactive) | carried by SPEC-053, authored in parallel |

The REQ-004 home is a metadata append to SPEC-040 REQ-001's `supports` list — `supports`
accepts a string OR a list (pkg/validate/spec.go), so no new requirement, no new claim,
no code touched on the implemented SPEC-040. REQ-007/REQ-010's home is SPEC-053, a NEW
`implemented` retroactive spec whose two requirements support
`collapse-legacy-codecheck-into-packs:REQ-007` (its REQ-007) and `:REQ-010` (its
REQ-010) with claims anchored to the ISSUE-018 removal tests on `main`; SPEC-051 depends
on SPEC-053 landing (see Sharp Edges). The `replaced` SPEC-039 is left byte-for-byte
unchanged in both cases.

### Legacy cluster retirement (REQ-001 + REQ-003)

The whole `agent-definitions` scaffolding cluster goes terminal together:

| Artifact | Before | After | Retirement field | Requirement |
|----------|--------|-------|------------------|-------------|
| `agent-definitions.bundle.md` | `delivered` | `deprecated` | none required | REQ-001 |
| SPEC-002 | `draft` (cites `agent-definitions:REQ-004..007`) | `deprecated` | none required | REQ-003 |
| SPEC-003 | `draft` (cites `agent-definitions:REQ-008`) | `deprecated` | none required | REQ-003 |
| SPEC-004 | `draft` (cites `agent-definitions:REQ-014..018`) | `deprecated` | none required | REQ-003 |

`deprecated` = retired without a single named successor; per pkg/validate/terminal.go it
requires no pointer field (unlike `replaced`/`obsoleted`), and a terminal artifact is
exempt from the live-work completeness, requirements[], and supports-format checks
(pkg/validate terminal exemption). Retiring the three specs takes their dangling
`agent-definitions` refs out of validation via the terminal-citer exemption; deprecating
the bundle removes it from the delivered/requirements[] gates AND the SPEC-052
delivered-coverage evaluation, so no coverage debt is created for the dropped backfill.

## Implementation

### This spec adds no production code

The regex tightening and the resolution/version-log functions belong to SPEC-050
(`supportsRe`, `BuildBundleReqCatalog`, `CollectSupportRefs`, `ResolveSupports`,
`validateBundleRequirements`). SPEC-051 delivers (a) the reconciled corpus — a set of
artifact edits routed through the artifact agents / CLI — and (b) corpus-assertion tests
in `pkg/validate` that exercise SPEC-050's functions over the REAL project artifacts to
prove the end state. Because those tests CONSUME SPEC-050's public resolution functions
but EXPORT no new symbol of their own, the `contracts[]` block declares a single
`consumes`-only contract for the assertion-test file (no `provides`).

### Reconciliation steps (the mechanical steps a planner maps to)

1. **Deprecate the `agent-definitions` bundle (REQ-001).** Set `status.maturity:
   deprecated` on `agent-definitions.bundle.md` (optionally a free-text reason citing the
   `.claude` roster / agent-guard / ISSUE-044 supersession); do NOT add a `requirements:`
   array — the backfill is dropped. `deprecated` needs no retirement pointer.
2. **Re-home BUNDLE-011 (REQ-002).** Append `collapse-legacy-codecheck-into-packs:REQ-004`
   to SPEC-040 REQ-001's `supports` list. Do NOT put REQ-007/REQ-010 on SPEC-040 — they
   are covered by SPEC-053 REQ-007 (`:REQ-007`) and SPEC-053 REQ-010 (`:REQ-010`), the
   parallel-authored `implemented` retroactive spec; SPEC-051 only depends on SPEC-053
   landing. Do not edit SPEC-039.
3. **Retire the drafts (REQ-003).** Set `status: deprecated` on SPEC-002/003/004
   (optionally a free-text `reason`), leaving their bodies otherwise intact — these retire
   as one cluster with the bundle (step 1). Do not set `implemented`.
4. **Version-stamp bundle REQs (REQ-004).** Add `version: "1.0.0"` to every REQ of
   BUNDLE-004/005/006/007/009/010/011/012/013 and the `cli` bundle (ten bundles). Skip
   BUNDLE-014 (already stamped), the now-terminal `agent-definitions` (no requirements[]),
   and all other terminal bundles.
5. **Pin-stamp supports refs (REQ-005).** Append the cited REQ's current-version pin to
   every `supports` ref on every live spec/issue requirement — `@1.0.0` for the initial
   backfill, `@1.1.0` for SPEC-051's own refs to `requirement-traceability:REQ-015` (amended
   via RDQ-2, revved per the RDQ-7 stale-pin model) — the specs carrying `supports` minus the
   retiring SPEC-002/003/004 and ALL terminal specs (SPEC-008/011/034/039), including the Seed
   specs SPEC-050/051/052/053.
6. **Atomic co-land (REQ-006).** Sequence all of the above into the same change-train as
   SPEC-050's `supportsRe` tightening + resolution wiring, so no intermediate committed
   state has the old regex facing pinned refs or the new regex facing unpinned refs.

### Verification passes (what the assertion tests exercise)

The `pkg/validate` tests parse the real corpus with `pkg/artifact` and drive SPEC-050's
functions: `BuildBundleReqCatalog` over the parsed bundles, `CollectSupportRefs` over the
parsed live specs/issues, and `ResolveSupports` for the both-direction + log-match verdict;
`validate.Bundle`/`validateBundleRequirements` for the version-log well-formedness on the
stamped bundles AND the terminal exemption on the deprecated `agent-definitions` bundle
(no requirements-missing violation despite carrying no `requirements[]`); `validate.Spec`
for the terminal exemption on the retired drafts. Each claim asserts a specific
corpus-state fact (a status terminal, a version present, a ref pinned, resolution clean)
so the whole set is the mechanical proof that the sweep landed and the corpus is green
under enforcement.

## Verification

Verification is defined in frontmatter: integration level, 80% coverage threshold,
targeting `pkg/validate`. Integration is chosen because the load-bearing behaviour is
corpus-spanning — the assertion tests reason across every bundle and spec at once by
running SPEC-050's resolution + version-log validation over the real project artifacts,
not over a single-file fixture. Because this spec adds only assertion tests over the
corpus (no new production `pkg/validate` code), its own coverage contribution is
effectively N/A; the substantive proof is the green corpus the tests assert, not a
coverage delta (see Sharp Edges). Claims are defined in frontmatter; every requirement
has at least one claim, and both positive assertions and negative guards (SPEC-039
untouched, drafts not promoted, unpinned/fabricated refs still red) are present.

## Sharp Edges

- **Atomic co-landing is non-negotiable — the flip and the sweep cannot be separated.**
  The current `supportsRe` (`^[a-z0-9-]+:REQ-\d{3}$`) REJECTS a pinned `@1.0.0` ref, and
  SPEC-050's tightened `supportsRe` REJECTS an unpinned ref. So a commit that stamps
  pins before the regex changes goes red, AND a commit that changes the regex before the
  stamp goes red. The sweep (REQ-001..REQ-005) and SPEC-050's regex/resolution change
  MUST be one change-train. This is a merge-sequencing constraint, NOT a runtime toggle
  and NOT a grandfather path — the end state is unconditionally on (mirror of SPEC-050's
  own enforcement-flip edge).

- **Terminal retirement only clears the dangling refs IF the resolution pass skips
  terminal citers.** Retiring SPEC-002/003/004 removes their refs from the PER-ARTIFACT
  supports-format check (pkg/validate/spec.go skips terminal specs). For the CORPUS
  resolution pass (SPEC-050) to likewise not red their dangling/unpinned refs,
  `CollectSupportRefs`/`ResolveSupports` must EXCLUDE terminal-status citing specs —
  mirroring that per-artifact exemption. If SPEC-050's resolution harvested refs from
  terminal specs, the retired drafts would still red. REQ-003 depends on this exclusion;
  it is flagged to Seed 1 so the two seeds agree.

- **Deprecating `agent-definitions` resolves the F1 seam with SPEC-052 — backfilling it
  would NOT.** The earlier plan (backfill the bundle's 18 prose REQs into `requirements[]`
  and keep it `delivered`) is a trap: the moment SPEC-052's delivered-coverage gate lands,
  a `delivered` bundle needs every REQ supported by ≥1 `implemented` spec requirement, but
  the only citers (SPEC-002/003/004) retire — so the 18 backfilled REQs would sit
  `delivered` with ZERO implemented coverage and SPEC-052 would BLOCK, silently
  contradicting this spec's "corpus green" claim. Deprecating the bundle instead puts it
  OUTSIDE the delivered-gate and the requirements[] gate by design — the same terminal
  exclusion the gate already honors — creating zero coverage debt. This is honest
  recording (the scaffolding-era roster is superseded by the living `.claude` roster +
  agent-guard + ISSUE-044), not grandfathering ([[feedback_align_predating_artifacts]]).

- **Re-home targets must be faithful, not merely structurally-satisfying — and SPEC-040
  is the WRONG home for REQ-007/REQ-010.** REQ-004 is faithfully covered by SPEC-040
  REQ-001 (toolchain-routing). But SPEC-040's scope fence explicitly DISOWNS the
  REQ-007/REQ-010 deletions (SPEC-039 / Seed-1 scope; the work landed via ISSUE-018,
  SPEC-039's `replaced-by`). Appending REQ-007/REQ-010 to SPEC-040 would satisfy the
  coverage check while making the trace LIE — a false green. Their honest home is
  SPEC-053 REQ-007/REQ-010 (implemented, retroactive, claims anchored to the ISSUE-018
  removal tests); an issue (ISSUE-018) can't itself provide coverage, which is why a
  retroactive SPEC is minted.

- **SPEC-053 must land `implemented` before the flip, or BUNDLE-011 blocks.** REQ-007 and
  REQ-010's only live coverage is SPEC-053 (REQ-007/REQ-010), the retroactive spec
  authored in parallel. If SPEC-053 is not `implemented` and landed in the same
  change-train as the enforcement flip, BUNDLE-011's REQ-007/REQ-010 have zero implemented
  coverage and the coverage gate (Seed 3 / SPEC-052) blocks. SPEC-051's sweep therefore
  has a hard co-landing dependency on SPEC-053; its `@1.0.0` pin on SPEC-053's own refs
  (REQ-005) is part of the same sweep.

- **Re-homing onto an IMPLEMENTED spec must not re-open its gate.** The REQ-004 → SPEC-040
  REQ-001 append adds a ref to an already-`implemented` spec; adding a NEW requirement or
  claim there would demand new tests and re-open its contracts/tests/coverage gate.
  Re-statement therefore APPENDS to the existing requirement's `supports` LIST only — a
  metadata edit with zero code and zero new claims, leveraging `supports` being
  list-valued (pkg/validate/spec.go). SPEC-053, by contrast, is authored fresh as
  `implemented` with its own claims/tests, so it carries REQ-007/REQ-010 without touching
  any existing gate.

- **SPEC-039 stays a tombstone — do not rewrite history.** BUNDLE-011 REQ-007/REQ-010
  were originally cited by SPEC-039, now `replaced`. The re-statement adds LIVE coverage
  on SPEC-040; SPEC-039's own refs are neither re-pinned (`@1.0.0`) nor re-homed. Editing
  a `replaced` spec would rewrite delivered history and is out of scope; its terminal
  exemption already keeps its refs out of validation.

- **Self-referential corpus — and the corpus's first live version-bump event.** SPEC-050,
  SPEC-051, and SPEC-052 are draft specs carrying `supports` refs, so they are in the
  stamped set. This spec's own `supports: requirement-traceability:REQ-015` was written
  UNPINNED at first (the pinned form would have been rejected by the then-live `supportsRe`);
  the sweep stamps it to `@1.1.0`, NOT `@1.0.0`. That is the load-bearing subtlety: BUNDLE-014
  REQ-015 was amended from `1.0.0` to `1.1.0` via RDQ-2 (a real minor meaning-change,
  backfill → deprecate) and SPEC-051 v1.3.0 implements the AMENDED meaning, so per the RDQ-7
  stale-pin model an in-flight bundle's downstream citers rev to the latest on a minor bump.
  Pinning `@1.0.0` here would falsely claim SPEC-051 implements the retired backfill text and
  would fail log-match resolution against REQ-015's version log (which carries 1.0.0 and
  1.1.0). Every OTHER seed-spec self-pin stays `@1.0.0` because it cites a REQ still at its
  initial version (requirement-traceability:REQ-001..014).

- **Coverage vacuity is expected here and is not a defect.** SPEC-051 adds only
  corpus-assertion tests, no new production lines, so there is no meaningful coverage
  delta to measure. The anti-vacuous-green concern is answered by the assertion tests
  proving a genuinely green corpus, not by a coverage percentage; a reviewer must not
  read the N/A coverage as a hollow gate ([[feedback_loud_not_blocking]]).

- **`deprecated` vs `canceled` is a deliberate, honest choice — for the whole cluster.**
  The `agent-definitions` bundle's work shipped (its agent roster is in active use, now via
  the living `.claude` roster), and SPEC-002/003/004's scope shipped via that bundle's
  direct delivery — so `canceled` (work abandoned) would misrepresent history and
  `replaced`/`obsoleted` would demand a successor pointer none has. `deprecated` (retired,
  no single named successor) is the faithful terminal state for the bundle AND the three
  specs, and matches corpus precedent (SPEC-001, SPEC-011, and the `deprecated` bundles
  `runtime-hooks`/`standards-compiler`).

## Review Questions

1. Is the `agent-definitions` bundle moved to `deprecated` (terminal) with NO
   `requirements[]` array added — the once-planned backfill dropped — so it is exempt from
   SPEC-052's delivered-coverage gate and the requirements[] gate, creating zero coverage
   debt? (REQ-001.)

2. Is `collapse-legacy-codecheck-into-packs:REQ-004` appended to SPEC-040 REQ-001 (its
   faithful home), while `:REQ-007` and `:REQ-010` are covered by SPEC-053 REQ-007 and
   SPEC-053 REQ-010 (NOT SPEC-040, which disowns those deletions), and is SPEC-039 left
   byte-for-byte unchanged (no `@1.0.0`, still `replaced`)? (REQ-002.)

3. Are SPEC-002/003/004 moved to `deprecated` (a terminal status) rather than `implemented`,
   and does each validate clean under the terminal exemption with no retirement-field
   violation? (REQ-003.)

4. Does every LIVE `supports` ref carry an exact pin matching its cited REQ's current version
   after the sweep — `@1.0.0` for the initial backfill, and `@1.1.0` for SPEC-051's own refs to
   `requirement-traceability:REQ-015` (amended via RDQ-2, revved per RDQ-7) — while the retiring
   SPEC-002/003/004 and the terminal SPEC-039 are excluded from the stamp? (REQ-005.)

5. Do the corpus edits and SPEC-050's regex/resolution flip land in the SAME change-train,
   so `artifact validate` is never red at any committed point and the end state has no
   config toggle and no grandfather path? (REQ-006.)

6. Do the assertion tests run SPEC-050's resolution + version-log validation over the REAL
   project corpus (not a fixture), so a green result is a genuine end-to-end proof that the
   sweep landed? (REQ-006 / CLM-019.)

## References

- BUNDLE-014 (requirement-traceability): source bundle; Seed 2 of 3. Covers REQ-015; the
  reconciliation actions descend from RDQ-2 (legacy corpus reconciliation) and RDQ-9
  (explicit `1.0.0` backfill sweep).
- SPEC-050 (Seed 1): defines the version-log schema, the mandatory `@X.Y.Z` pin, and the
  `pkg/validate` resolution functions this sweep's assertion tests call and whose
  enforcement flip this sweep co-lands with. Its "enforcement-flip ordering" and
  "self-referential corpus" sharp edges are the counterparts of this spec's.
- SPEC-052 (Seed 3): the `requirement_traceability` coverage gate step + stale-pin model;
  depends on a green corpus this sweep produces.
- `bundles/agent-definitions.bundle.md`: the delivered bundle DEPRECATED by this sweep
  (the requirements[] backfill is dropped), superseded by the living `.claude` agent
  roster + the agent-guard hook + the ISSUE-044 roster-consistency check (REQ-001).
- `bundles/BUNDLE-011-collapse-legacy-codecheck-into-packs.bundle.md`, SPEC-040 REQ-001
  (implemented, REQ-004 home), SPEC-053 REQ-007/REQ-010 (implemented retroactive,
  REQ-007/REQ-010 homes), SPEC-039 (`replaced`, untouched), ISSUE-018 (where the
  deletions actually landed): the re-home sources and targets (REQ-002).
- SPEC-053 (retroactive-codecheck-deadcode-deletions): the new `implemented` spec authored
  in parallel that carries `collapse…:REQ-007` (its REQ-007) and `:REQ-010` (its REQ-010);
  SPEC-051 depends on it co-landing before the enforcement flip.
- `specs/SPEC-002/003/004`: the stale `draft` specs retired to `deprecated` (REQ-003).
- `pkg/validate/spec.go` (`supportsRe`, terminal exemption at the completeness gate),
  `pkg/validate/bundle.go` (`validateBundleRequirements`, terminal exemption),
  `pkg/validate/terminal.go` (`isTerminalStatus`, retirement-field rules): the validation
  surfaces the sweep must turn green.
- [[feedback_align_predating_artifacts]] — governs the backfill/re-statement/retirement
  choices (update delivered artifacts, retire abandoned drafts, openly).
- [[project_artifact_terminal_states]] — the ISSUE-031 `deprecated`/`replaced` semantics the
  retirement (REQ-003) and the SPEC-039 no-flow-through (REQ-002) build on.
- [[feedback_scaffold_via_cli]], [[feedback_loud_not_blocking]] — routing edits through the
  agents/CLI, and the coverage-vacuity framing.

## Version History

- **1.0.0 (2026-07-14, draft)** — Initial authoring of Seed 2: the reconciliation +
  `1.0.0` backfill sweep (6 requirements, 20 claims, 1 consumes-only contract). At this
  version REQ-004/REQ-007/REQ-010 all re-homed onto SPEC-040, and REQ-005's live set
  excluded only SPEC-039.
- **1.1.0 (2026-07-14, draft)** — spec-reviewer fixes. BLOCKER: REQ-007/REQ-010 re-homed
  off SPEC-040 (a false trace — SPEC-040's scope fence disowns those deletions; they
  landed via ISSUE-018, an issue that can't cover) onto the new `implemented` retroactive
  spec SPEC-053 REQ-007/REQ-010; REQ-004 stays on SPEC-040 REQ-001. Every home now named
  by exact requirement id (REQ-002, Overview, homes table, Impl step 2, claims
  CLM-005..007). Fixed the Implementation prose to match the `consumes`-only `contracts[]`
  block. Trued up REQ-005's live set to exclude ALL terminal specs
  (SPEC-008/011/034/039), not just SPEC-039, and the count to ~370 refs / ~33 live specs.
  Added a Sharp Edge: SPEC-053 must land `implemented` before the flip or BUNDLE-011
  blocks. `spec_version` 1.0.0 → 1.1.0.
- **1.2.0 (2026-07-14, draft)** — SPEC-053 landed with requirement ids REQ-007/REQ-010
  (deliberately mirroring the BUNDLE-011 ids they support), not REQ-001/REQ-002. Swept
  every SPEC-053 id reference in the file to REQ-007 (← collapse:REQ-007) and REQ-010
  (← collapse:REQ-010). `spec_version` 1.1.0 → 1.2.0.
- **1.2.1 (2026-07-14, draft)** — Trued up Review Question 2, which still asked whether all
  three of REQ-004/007/010 were appended to SPEC-040; it now matches the corrected homes
  (REQ-004 → SPEC-040 REQ-001; REQ-007/010 → SPEC-053 REQ-007/REQ-010). Prose-only.
- **1.3.0 (2026-07-14, draft)** — Consistency-pass BLOCKER F1 + founder decision. The
  agent-definitions requirements[] BACKFILL is DROPPED; instead the whole legacy
  scaffolding cluster is deprecated — `agent-definitions.bundle.md` goes `delivered` →
  `deprecated` (REQ-001, rewritten from backfill to deprecation) TOGETHER with
  SPEC-002/003/004 (REQ-003 expanded to the cluster). Rationale: a `delivered` bundle with
  a backfilled-but-uncovered requirements[] would be blocked by SPEC-052's
  delivered-coverage gate the moment it lands (the "corpus green" claim and SPEC-052
  silently disagreed on this bundle); a terminal bundle is exempt from that gate and the
  requirements[] gate, so deprecating creates zero coverage debt — honest recording, not
  grandfathering. REQ-004's stamp list drops `agent-definitions` (now terminal, ten
  bundles). Reworked claims CLM-001..004 (deprecation, not backfill), the Overview action
  list + action map, the cluster-retirement table, Impl steps 1/3/4, the F1-resolution +
  `deprecated`-vs-`canceled` sharp edges (replacing the backfill-scope edge), Review
  Question 1, and References. `spec_version` 1.2.0 → 1.3.0.
- **1.3.1 (2026-07-14, draft)** — First LIVE version-bump event, resolved by the RDQ-7
  stale-pin model. BUNDLE-014 REQ-015 was amended `1.0.0` → `1.1.0` via RDQ-2 (backfill →
  deprecate), and SPEC-051 v1.3.0 implements the AMENDED meaning; per RDQ-7 an in-flight
  bundle's downstream citers rev to the latest on a minor bump. Re-pinned SPEC-051's six own
  `supports` refs from `requirement-traceability:REQ-015@1.0.0` to `@1.1.0`. Amended CLM-018
  and the seed-self-pin prose (implementation summary action 5, Overview step 5 + the
  self-referential note, Impl step 5, and the "self-referential corpus" sharp edge) so they
  state the current-version rule: `@1.0.0` for refs to requirement-traceability:REQ-001..014,
  `@1.1.0` for REQ-015. Nuanced REQ-005's blanket-`@1.0.0` language to the current-version
  pin rule. bundle-author-sweep adds REQ-015's `versions:` log (1.0.0 + 1.1.0) in parallel so
  the `@1.1.0` pins resolve. The mandated test `TestReconcile_SeedSpecsSelfPinnedAt100` is
  implementer territory; its assertion is aligned there, not here. `spec_version` 1.3.0 → 1.3.1.
