---
title: "Requirement Traceability Gate Step"
number: SPEC-052
created: "2026-07-14"
status: implemented
schema_version: spec/v1
spec_version: 1.1.1

implementation:
  summary: >
    Seed 3 of BUNDLE-014 (requirement-traceability): the enforcement keystone —
    the `requirement_traceability` gate step that welds the COVERAGE direction of
    the bundle REQ -> spec requirement hop as a corpus-STATE invariant. Pure
    artifact-domain logic (zero language/tool knowledge). Five mechanically-
    separable pieces: (1) mint a NEW gate step `requirement_traceability` plus its
    paired non-policied advisory twin `requirement_traceability_advisory`
    (pkg/gate/result.go), NOT claiming the reserved `ledger_integrity` name;
    (2) a pure classifier `ClassifyRequirementTraceability` + `SplitTraceabilityResult`
    (pkg/gate/requirement_traceability.go, NEW) modeled EXACTLY on the status_drift
    `ClassifyStatusDrift`/`SplitDriftResult` block/advisory pattern — combined
    result carrying error-severity BLOCK violations and warning-severity ADVISORY
    violations, split by severity into a policied block surface and a non-policied
    advisory surface no enforcement.policy entry can upgrade; (3) the delivered-gate
    coverage rules — every LIVE citing spec `implemented` (REQ-008), every bundle
    REQ supported by >=1 requirement in an `implemented` spec, per-REQ not aggregate
    (REQ-009), `replaced`/retired specs never flow through (REQ-010), issue
    requirements are queryable lineage but never coverage (REQ-011); (4) the state
    invariant — the step blocks any corpus STATE where a downstream artifact outruns
    its upstream chain (a spec advanced past the depth its chain supports, a plan
    for a spec whose chain does not verify, a `delivered` bundle without full-depth
    coverage), with dispatch-time refusal exposed as a SEAM hooks/runtime consume
    (the gate verdict), core judging states never intercepting events (REQ-012);
    (5) the lifecycle-keyed, semver-gated stale-pin model (REQ-014) reading the
    CURRENT version off each bundle REQ per SPEC-050's schema. To reach the corpus
    it EXTENDS the existing `ResolveArtifactStatus` (pkg/gate/artifact_status.go) to
    also walk `bundles/` (a KindBundle case in the existing walker, NOT a second
    walker) surfacing bundle maturity + each REQ's current version, and REUSES
    SPEC-050's `CollectSupportRefs` for the citing supports refs; the shipped
    `cmd/backstop/gate.go` wires the two surfaces into the step list mirroring the
    drift steps. Covers BUNDLE-014 REQ-006, REQ-008, REQ-009, REQ-010, REQ-011,
    REQ-012, REQ-013, REQ-014 ONLY. Depends on SPEC-050 (Seed 1: version/log schema +
    resolution + `CollectSupportRefs`) and SPEC-051 (Seed 2: the reconciled, `1.0.0`-
    backfilled green corpus); it MUST land after both — sequencing is the guard, so
    the step never has to reason about an unbackfilled corpus (see Sharp Edges).
  subject: pkg/gate

verification:
  level: integration
  test_command: go test ./pkg/gate/... ./cmd/backstop/... -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      The coverage/traceability enforcement must be a NEW gate step named
      `requirement_traceability`, with a paired NON-policied advisory twin named
      `requirement_traceability_advisory` (mirroring `artifact_status_drift` /
      `artifact_status_drift_advisory`). Both names are minted as canonical gate
      step constants in pkg/gate/result.go and are part of the gate's JSON output
      contract. The reserved `ledger_integrity` step name (pkg/gate/result.go)
      must NOT be claimed, renamed, or reused — it stays reserved for SPEC-010's
      provenance ledger hash chain. The step is built on the status_drift full-
      corpus block/advisory pattern: one classifier emits a combined StepResult
      carrying both severities, and `SplitTraceabilityResult` partitions it by
      severity into the policied block surface (`requirement_traceability`,
      error-severity) and the non-policied advisory surface
      (`requirement_traceability_advisory`, warning-severity). Minting this gate
      step realizes the GATE-SIDE of the DD-8 placement split (bundle REQ-007):
      coverage is enforced in the gate step while SPEC-050 owns the other half,
      resolution-in-`artifact validate`.
    supports:
      - requirement-traceability:REQ-006@1.0.0
      - requirement-traceability:REQ-007@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-002
    text: >
      The `requirement_traceability` step must block a `delivered` bundle unless
      EVERY LIVE spec citing it (via a `supports` ref) is success-terminal
      (`implemented`). A `draft` or `ready-for-implementation` spec hanging off a
      `delivered` bundle is a broken-promise BLOCK. Terminal-RETIRED citing specs
      (`replaced`, `canceled`, `deprecated`, `obsoleted`) are EXCLUDED from this
      live-citing check (a retired spec is not an unfinished promise) — they simply
      provide no coverage (REQ-004); in the production pipeline they are also absent
      from the collected ref set (SPEC-050 v1.2.0's terminal-citer exemption skips
      them), consistent with a terminal spec being neither coverage nor resolution-
      checked. The delivered-gate applies ONLY to bundles whose maturity is
      `delivered`; terminal-RETIRED bundles (`replaced`, `canceled`, `deprecated`)
      are excluded from the delivered-gate entirely, consistent with the gate's
      existing terminal-exclusion (do NOT conflate a retired BUNDLE's exclusion with
      a retired SPEC's non-support).
    supports: requirement-traceability:REQ-008@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-003
    text: >
      The step must block a `delivered` bundle unless EVERY one of its declared
      REQs is supported by >=1 requirement in an `implemented` spec. Coverage is
      PER-REQ, not aggregate: each REQ needs its own live `implemented` supporter,
      and a delivered bundle with one covered REQ and one uncovered REQ blocks,
      with the block naming the specific uncovered REQ. A requirement in an
      `implemented` spec whose `supports` ref names the bundle REQ is what closes
      that REQ.
    supports: requirement-traceability:REQ-009@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-004
    text: >
      Support must NOT flow through a spec that is not live-and-`implemented`. A
      bundle REQ supported ONLY by a `replaced` spec counts as UNCOVERED for
      REQ-003, and the same holds for the other retired-terminal spec statuses
      (`canceled`, `deprecated`, `obsoleted`) and for the not-yet-implemented live
      statuses (`draft`, `ready-for-implementation`). Only an `implemented` spec
      requirement provides coverage; the supports claim must be re-stated on the
      replacing spec or another live `implemented` spec for the REQ to be covered.
      This aligns with SPEC-050 v1.2.0's terminal-citer exemption (a terminal citing
      spec is skipped by `CollectSupportRefs`, so it is neither resolution-checked
      NOR coverage); the classifier additionally treats a retired-terminal citing
      artifact as non-coverage defensively, so the verdict holds even if such a ref
      reaches it.
    supports: requirement-traceability:REQ-010@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      Issue requirements tracing to a bundle REQ must be first-class lineage links
      — retained and queryable in the traceability corpus — but must NEVER satisfy
      REQ-003 coverage. An issue `supports` ref neither adds nor removes coverage:
      a bundle REQ supported only by an issue requirement is UNCOVERED, and a REQ
      already covered by an `implemented` spec is unaffected by any co-citing issue
      ref. Only `implemented` SPEC requirements close a bundle REQ; issue support
      is lineage, not coverage.
    supports: requirement-traceability:REQ-011@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-006
    text: >
      Chain-of-custody must be enforced as a corpus-STATE invariant, not an event
      handler: the step must BLOCK any corpus state in which a downstream artifact
      exists whose upstream chain does not verify — a spec advanced (`implemented`
      or `ready-for-implementation`) past the depth its bundle chain supports (a
      stale pin against an unimplemented bundle, per REQ-008), a plan existing for
      a spec whose bundle chain does not verify, or a `delivered` bundle without
      full-depth coverage (REQ-002 + REQ-003). Core judges STATES, never events:
      because promotions land as edits and all gate dimensions run at every
      verification step, every transition is judged by construction. Dispatch-time
      REFUSAL — blocking an `/implement` run before it starts — is explicitly an
      enforcement SEAM consumed by hooks/runtime (the gate-on-implement hook today,
      the opencode runtime later): core supplies the deterministic verdict on the
      corpus state via the block StepResult surfaced in the gate output, and MUST
      NOT intercept, subscribe to, or refuse any promotion/dispatch event itself.
    supports: requirement-traceability:REQ-012@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-007
    text: >
      Enforcement posture must follow loud!=blocking as THREE tiers. (a) DEFECTS —
      a dangling, unpinned, or fabricated-version ref — are validate's domain
      (SPEC-050, re-surfaced through the gate's `artifact_validation` step) and
      this step MUST NOT re-run resolution or emit ref-resolution findings. (b)
      BROKEN PROMISES — a coverage failure on a `delivered` bundle, or any corpus
      state where a downstream artifact's upstream chain does not verify (REQ-002,
      REQ-003, REQ-006, REQ-008) — BLOCK on the `requirement_traceability` surface.
      (c) IN-FLIGHT GAPS — an uncovered REQ on a NON-`delivered` bundle whose
      downstream has not advanced past what its chain supports — an ADVISORY
      WARNING on the `requirement_traceability_advisory` surface: visible, never
      blocking, never policy-upgradable. The advisory surface carries NO
      enforcement.policy entry so no policy can upgrade its WARN to a block
      (structurally, exactly like `artifact_status_drift_advisory`). The
      `requirement_traceability` block surface is neither WAIVABLE (it is not in
      the gate's code-located waivable dimension set — `@waiver` tokens cannot
      suppress it) NOR baseline-grandfathered (it carries no `applies-to: new-code`
      policy entry): an artifact-state broken promise must be resolved, never
      snapshotted, so it blocks whether or not a baseline is present.
    supports: requirement-traceability:REQ-013@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-008
    text: >
      The stale-pin model must be lifecycle-keyed and semver-gated, reading each
      bundle REQ's CURRENT version off the version log per SPEC-050's schema (the
      REQ's top-level `version:`, which equals the newest log entry). A PATCH bump
      is wording-only and FREE: a downstream pin auto-satisfies as long as it shares
      the current version's MAJOR.MINOR (pin.major==current.major &&
      pin.minor==current.minor). A MAJOR or MINOR bump means the meaning changed: a
      downstream pin whose major.minor differs from the current major.minor is
      STALE, and the outcome is keyed on the bundle's lifecycle — for an
      UNIMPLEMENTED (non-`delivered`) bundle every downstream spec/plan pinned to
      the old version is a stale-pin BLOCK (must re-pin and re-work before
      proceeding); for a `delivered` bundle the old specs stay IMMUTABLE at their
      (still-log-resolvable) pin, the new version requires a NEW spec, and the
      bundle BLOCKS as not-delivered until a spec pinned to the current version is
      `implemented`. Version comparison is numeric per semver component (so
      `1.10.x` shares major.minor with `1.10.y` but not `1.9.z`).
    supports: requirement-traceability:REQ-014@1.0.0
    follows: STD-GO-001:GO-010

claims:
  # REQ-001 — step mint + advisory twin + status_drift block/advisory pattern
  - id: CLM-001
    requirement: REQ-001
    subject: pkg/gate
    text: The StepRequirementTraceability constant equals "requirement_traceability"
    tests:
      - TestTraceStep_BlockStepNameConstant
  - id: CLM-002
    requirement: REQ-001
    subject: pkg/gate
    text: The StepRequirementTraceabilityAdvisory constant equals "requirement_traceability_advisory"
    tests:
      - TestTraceStep_AdvisoryStepNameConstant
  - id: CLM-003
    requirement: REQ-001
    subject: pkg/gate
    kind: absence
    text: The reserved ledger_integrity step name is neither claimed nor reused by the traceability step (it still names the provenance ledger)
    tests:
      - TestTraceStep_LedgerIntegrityStaysReserved
  - id: CLM-004
    requirement: REQ-001
    subject: pkg/gate
    text: SplitTraceabilityResult partitions a combined result's error-severity violations onto the block surface and warning-severity violations onto the advisory surface
    tests:
      - TestTraceStep_SplitPartitionsBySeverity
  - id: CLM-005
    requirement: REQ-001
    subject: cmd/backstop
    text: The shipped backstop gate step list wires BOTH the requirement_traceability block surface and the requirement_traceability_advisory surface
    tests:
      - TestGateCLI_RequirementTraceabilityStepsWired

  # REQ-002 — delivered requires every LIVE citing spec implemented; bundle-maturity + retired-spec scope
  - id: CLM-006
    requirement: REQ-002
    subject: pkg/gate
    text: A delivered bundle whose every citing spec is implemented passes the live-citing check
    tests:
      - TestTrace_Delivered_ImplementedCitingSpecOK
  - id: CLM-007
    requirement: REQ-002
    subject: pkg/gate
    text: A delivered bundle with a draft spec citing it is a broken-promise block
    tests:
      - TestTrace_Delivered_DraftCitingSpecBlocks
  - id: CLM-008
    requirement: REQ-002
    subject: pkg/gate
    text: A delivered bundle with a ready-for-implementation spec citing it is a broken-promise block
    tests:
      - TestTrace_Delivered_ReadyForImplCitingSpecBlocks
  - id: CLM-009
    requirement: REQ-002
    subject: pkg/gate
    text: A replaced spec citing a delivered bundle is EXCLUDED from the live-citing check (its retired status is not an unfinished promise)
    tests:
      - TestTrace_Delivered_ReplacedCitingSpecExcludedFromLiveCheck
  - id: CLM-010
    requirement: REQ-002
    subject: pkg/gate
    text: A replaced bundle is excluded from the delivered-gate entirely (no coverage obligation, no block)
    tests:
      - TestTrace_Bundle_ReplacedBundleExcluded
  - id: CLM-011
    requirement: REQ-002
    subject: pkg/gate
    text: A canceled bundle is excluded from the delivered-gate entirely
    tests:
      - TestTrace_Bundle_CanceledBundleExcluded
  - id: CLM-012
    requirement: REQ-002
    subject: pkg/gate
    text: A deprecated bundle is excluded from the delivered-gate entirely
    tests:
      - TestTrace_Bundle_DeprecatedBundleExcluded
  - id: CLM-048
    requirement: REQ-002
    subject: pkg/gate
    text: A delivered bundle declaring a bundle.name but no number field still produces a KindBundle record and is joined by name — its citing refs resolve and its coverage is evaluated (the agent-definitions shape)
    tests:
      - TestTrace_Bundle_NameWithoutNumberJoinsByName

  # REQ-003 — per-REQ coverage by an implemented spec (positive direction + per-REQ granularity)
  - id: CLM-013
    requirement: REQ-003
    subject: pkg/gate
    text: A delivered bundle whose every REQ has an implemented-spec supporter passes
    tests:
      - TestTrace_Delivered_FullCoveragePasses
  - id: CLM-014
    requirement: REQ-003
    subject: pkg/gate
    text: A delivered bundle with a REQ supported by no spec at all blocks as uncovered
    tests:
      - TestTrace_Delivered_UncoveredReqBlocks
  - id: CLM-015
    requirement: REQ-003
    subject: pkg/gate
    text: Coverage is per-REQ not aggregate — a delivered bundle with one covered and one uncovered REQ blocks, naming the uncovered REQ
    tests:
      - TestTrace_Delivered_PerReqNotAggregate
  - id: CLM-016
    requirement: REQ-003
    subject: pkg/gate
    text: An implemented spec requirement whose supports ref names a bundle REQ closes that REQ (SupportsCoverage true for spec + success-terminal)
    tests:
      - TestTrace_Coverage_ImplementedSpecCovers
  - id: CLM-049
    requirement: REQ-003
    subject: cmd/backstop
    text: Over a real fixture corpus through the shipped wiring, a delivered bundle with an uncovered REQ actually resolves to a KindBundle record and actually blocks the requirement_traceability step end-to-end (not just a classifier unit test over hand-built records)
    tests:
      - TestGateCLI_RequirementTraceability_DeliveredUncoveredBlocksOverFixtureCorpus
  - id: CLM-050
    requirement: REQ-003
    subject: cmd/backstop
    text: A citing ref whose path differs from the record path only by absolute/relative form still joins after both are gate.NormalizePath'd, so coverage is not silently dropped by a path mismatch
    tests:
      - TestGateCLI_RequirementTraceability_PathNormalizationJoins

  # REQ-004 — coverage-source status matrix: which spec statuses do NOT flow through
  - id: CLM-017
    requirement: REQ-004
    subject: pkg/gate
    text: A REQ supported ONLY by a replaced spec is uncovered (no flow-through through the tombstone)
    tests:
      - TestTrace_Coverage_ReplacedSpecDoesNotCover
  - id: CLM-018
    requirement: REQ-004
    subject: pkg/gate
    text: A REQ supported ONLY by a canceled spec is uncovered
    tests:
      - TestTrace_Coverage_CanceledSpecDoesNotCover
  - id: CLM-019
    requirement: REQ-004
    subject: pkg/gate
    text: A REQ supported ONLY by a deprecated spec is uncovered
    tests:
      - TestTrace_Coverage_DeprecatedSpecDoesNotCover
  - id: CLM-020
    requirement: REQ-004
    subject: pkg/gate
    text: A REQ supported ONLY by an obsoleted spec is uncovered
    tests:
      - TestTrace_Coverage_ObsoletedSpecDoesNotCover
  - id: CLM-021
    requirement: REQ-004
    subject: pkg/gate
    text: A REQ supported ONLY by a draft spec is uncovered (not yet implemented)
    tests:
      - TestTrace_Coverage_DraftSpecDoesNotCover
  - id: CLM-022
    requirement: REQ-004
    subject: pkg/gate
    text: A REQ supported ONLY by a ready-for-implementation spec is uncovered (not yet implemented)
    tests:
      - TestTrace_Coverage_ReadyForImplSpecDoesNotCover

  # REQ-005 — issue lineage never coverage (queryable but non-coverage)
  - id: CLM-023
    requirement: REQ-005
    subject: pkg/gate
    text: A delivered bundle REQ supported only by an issue requirement is UNCOVERED and blocks (issue support is not coverage)
    tests:
      - TestTrace_Coverage_IssueRefNeverCovers
  - id: CLM-024
    requirement: REQ-005
    subject: pkg/gate
    text: An issue supports ref is retained as a queryable lineage link in the traceability corpus
    tests:
      - TestTrace_Lineage_IssueRefQueryable
  - id: CLM-025
    requirement: REQ-005
    subject: pkg/gate
    text: A bundle REQ already covered by an implemented spec is unaffected by a co-citing issue ref (issue neither adds nor removes coverage)
    tests:
      - TestTrace_Coverage_IssueRefDoesNotAffectSpecCoverage

  # REQ-006 — corpus-STATE invariant + dispatch-refusal-as-seam
  - id: CLM-026
    requirement: REQ-006
    subject: pkg/gate
    text: A spec advanced to implemented whose bundle chain does not verify (stale pin on an unimplemented bundle) blocks as a downstream-outruns-chain state
    tests:
      - TestTrace_State_SpecOutrunsChainBlocks
  - id: CLM-027
    requirement: REQ-006
    subject: pkg/gate
    text: A plan existing for a spec whose bundle chain does not verify blocks, via the plan.SpecID -> spec record -> spec TraceRefs two-hop (the plan carries no supports of its own)
    tests:
      - TestTrace_State_PlanForUnverifiedSpecBlocks
  - id: CLM-028
    requirement: REQ-006
    subject: pkg/gate
    text: A delivered bundle without full-depth coverage blocks as a state (REQ-002 + REQ-003 at full depth)
    tests:
      - TestTrace_State_DeliveredWithoutCoverageBlocks
  - id: CLM-029
    requirement: REQ-006
    subject: pkg/gate
    text: A downstream artifact at a depth its chain fully supports produces no state block (negative)
    tests:
      - TestTrace_State_SpecWithinChainNoBlock
  - id: CLM-030
    requirement: REQ-006
    subject: pkg/gate
    kind: absence
    text: The step supplies its verdict only as the block StepResult and subscribes to / intercepts / refuses no promotion or dispatch event (dispatch refusal stays a hook/runtime seam)
    tests:
      - TestTrace_State_VerdictIsStateNoEventInterception
  - id: CLM-031
    requirement: REQ-006
    subject: cmd/backstop
    text: The block verdict is surfaced under the requirement_traceability step name in the gate output so the gate-on-implement hook can consume it
    tests:
      - TestGateCLI_RequirementTraceabilityVerdictInOutput

  # REQ-007 — three-tier posture: defects (validate), broken promises block, in-flight advisory; non-waivable, non-baselineable, non-upgradable
  - id: CLM-032
    requirement: REQ-007
    subject: pkg/gate
    text: An uncovered REQ on a non-delivered (ready) bundle surfaces as a warning-severity advisory, not a block
    tests:
      - TestTrace_Posture_InFlightUncoveredAdvisoryWarn
  - id: CLM-033
    requirement: REQ-007
    subject: pkg/gate
    text: An uncovered REQ on an exploring or defined bundle surfaces on the advisory surface (never blocks)
    tests:
      - TestTrace_Posture_NonDeliveredUncoveredAdvisory
  - id: CLM-034
    requirement: REQ-007
    subject: pkg/gate
    text: A delivered-bundle coverage failure is an error-severity violation on the block surface (broken promise blocks)
    tests:
      - TestTrace_Posture_BrokenPromiseBlocks
  - id: CLM-035
    requirement: REQ-007
    subject: pkg/gate
    kind: absence
    text: The requirement_traceability step emits no ref-resolution / dangling-ref findings — resolution defects are validate's domain (tier a)
    tests:
      - TestTrace_Posture_ResolutionDefectsNotInThisStep
  - id: CLM-036
    requirement: REQ-007
    subject: pkg/gate
    text: No enforcement.policy entry can upgrade the requirement_traceability_advisory surface to a block (it carries no policy entry, mirroring the drift advisory)
    tests:
      - TestTrace_Posture_AdvisoryNotPolicyUpgradable
  - id: CLM-037
    requirement: REQ-007
    subject: pkg/gate
    kind: absence
    text: requirement_traceability is NOT in the gate's waivable dimension set, so a @waiver token cannot suppress a traceability block finding
    tests:
      - TestTrace_Posture_NotWaivable
  - id: CLM-038
    requirement: REQ-007
    subject: pkg/gate
    text: A requirement_traceability block finding blocks with or without a baseline entry (not baseline-grandfathered — no applies-to new-code path)
    tests:
      - TestTrace_Posture_NotBaselineGrandfathered

  # REQ-008 — stale-pin model: bump size x bundle lifecycle matrix
  - id: CLM-039
    requirement: REQ-008
    subject: pkg/gate
    text: A pin equal to the REQ's current version is satisfied (no bump)
    tests:
      - TestTrace_StalePin_SameVersionSatisfied
  - id: CLM-040
    requirement: REQ-008
    subject: pkg/gate
    text: A PATCH bump on an unimplemented bundle auto-satisfies a downstream pin that shares major.minor (free)
    tests:
      - TestTrace_StalePin_PatchUnimplementedFree
  - id: CLM-041
    requirement: REQ-008
    subject: pkg/gate
    text: A PATCH bump on a delivered bundle auto-satisfies a downstream pin that shares major.minor (free)
    tests:
      - TestTrace_StalePin_PatchDeliveredFree
  - id: CLM-042
    requirement: REQ-008
    subject: pkg/gate
    text: A MINOR bump on an unimplemented bundle makes a downstream pin to the old version a stale-pin block (must re-pin)
    tests:
      - TestTrace_StalePin_MinorUnimplementedRepinBlocks
  - id: CLM-043
    requirement: REQ-008
    subject: pkg/gate
    text: A MAJOR bump on an unimplemented bundle makes a downstream pin to the old version a stale-pin block (must re-pin)
    tests:
      - TestTrace_StalePin_MajorUnimplementedRepinBlocks
  - id: CLM-044
    requirement: REQ-008
    subject: pkg/gate
    text: A MINOR bump on a delivered bundle blocks the bundle as not-delivered until a spec pinned to the new version is implemented (new spec required)
    tests:
      - TestTrace_StalePin_MinorDeliveredNewSpecRequiredBlocks
  - id: CLM-045
    requirement: REQ-008
    subject: pkg/gate
    text: A MAJOR bump on a delivered bundle blocks the bundle as not-delivered until a spec pinned to the new version is implemented
    tests:
      - TestTrace_StalePin_MajorDeliveredNewSpecRequiredBlocks
  - id: CLM-046
    requirement: REQ-008
    subject: pkg/gate
    text: A delivered bundle whose bumped REQ has a NEW implemented spec pinned to the current version is satisfied (old spec stays immutable at its pin)
    tests:
      - TestTrace_StalePin_DeliveredNewVersionNewSpecImplementedSatisfied
  - id: CLM-047
    requirement: REQ-008
    subject: pkg/gate
    text: The current version is read as the REQ's top-level version (newest log entry) and comparison is numeric per component (1.10 shares major.minor with 1.10 but not 1.9)
    tests:
      - TestTrace_StalePin_CurrentVersionNumericMajorMinor

contracts:
  - file: pkg/gate/result.go
    provides:
      - name: StepRequirementTraceability
        kind: constant
        signature: "const StepRequirementTraceability = \"requirement_traceability\""
        notes: "Canonical block-surface step name (REQ-001). Part of the gate JSON output contract; a dynamically-wired step like artifact_status_drift, NOT added to the fixed AllStepNames[9] array. It is the policied broken-promise surface and the deterministic verdict the gate-on-implement hook consumes (REQ-012)."
      - name: StepRequirementTraceabilityAdvisory
        kind: constant
        signature: "const StepRequirementTraceabilityAdvisory = \"requirement_traceability_advisory\""
        notes: "Canonical NON-policied advisory-surface step name (REQ-001/REQ-007). A SEPARATE step name specifically so no enforcement.policy entry can upgrade the in-flight-gap WARN to a block, exactly as StepArtifactStatusDriftAdvisory does for drift."
  - file: pkg/gate/artifact_status.go
    provides:
      - name: BundleReqVersion
        kind: type
        signature: "type BundleReqVersion struct { ReqID string; CurrentVersion string }"
        notes: "One declared bundle REQ and its CURRENT version (the REQ's top-level version: per SPEC-050's schema = the newest log entry). Populated for KindBundle records only; the stale-pin model needs only the current version, never the full log (log MEMBERSHIP is SPEC-050/validate's concern)."
      - name: ArtifactStatusRecord.BundleName
        kind: type
        signature: "type ArtifactStatusRecord struct { ID string; Kind ArtifactKind; Status string; Class StatusClass; Path string; MandatedTests []MandatedTest; SpecID string; BundleName string; BundleReqs []BundleReqVersion }"
        notes: "Additive field carrying `bundle.name` for a KindBundle record (empty for other kinds). This is the JOIN KEY: supports refs cite a bundle by NAME (e.g. `requirement-traceability`), NOT by number — and a delivered bundle may have NO `number:` field at all (agent-definitions.bundle.md is exactly this), so ID = fm.Number is unusable as the join key. TraceRef.BundleName joins against this, mirroring SPEC-050's BuildBundleReqCatalog keying by bundle name."
      - name: ArtifactStatusRecord.BundleReqs
        kind: type
        signature: "type ArtifactStatusRecord struct { ID string; Kind ArtifactKind; Status string; Class StatusClass; Path string; MandatedTests []MandatedTest; SpecID string; BundleName string; BundleReqs []BundleReqVersion }"
        notes: "Additive field on the existing ArtifactStatusRecord: the declared REQs + current versions for a KindBundle record (nil for other kinds). Existing fields are untouched."
    consumes:
      - source: pkg/gate
        name: MandatedTest
        kind: type
  - file: pkg/gate/requirement_traceability.go
    provides:
      - name: TraceRef
        kind: type
        signature: "type TraceRef struct { BundleName string; ReqID string; PinVersion string; Pinned bool; CitingPath string }"
        notes: "A gate-local citing supports ref, converted from validate.SupportRef by the cmd/backstop wiring (pkg/gate does not import pkg/validate). BundleName joins to a KindBundle record's BundleName; CitingPath joins to an ArtifactStatusRecord.Path so the classifier reads the citing artifact's kind + status class. Both CitingPath and the record Path MUST be normalized identically (gate.NormalizePath, the same normalizer the drift wiring uses) BEFORE joining — a raw SupportRef.File (validate-relative) vs a filepath.Join'd record Path (gate-side) would otherwise silently fail every join and drop all coverage."
      - name: SupportsCoverage
        kind: function
        signature: "func SupportsCoverage(kind ArtifactKind, class StatusClass) bool"
        notes: "The coverage-source predicate: true ONLY for a spec (KindSpec) that is success-terminal (ClassSuccessTerminal == implemented). Every other kind/class — issues (REQ-005), draft/ready-for-implementation specs, and retired-terminal specs (REQ-004) — is false. Reuses the existing ClassifyArtifactStatus machinery. Terminal-retired citing specs are additionally pre-excluded upstream by SPEC-050 v1.2.0's CollectSupportRefs; SupportsCoverage returning false for them is defensive-in-depth."
      - name: StalePinVerdict
        kind: function
        signature: "func StalePinVerdict(pin string, current string) (bump string, stale bool)"
        notes: "Semver-gated stale-pin comparison (REQ-008): bump is none/patch/minor/major from numeric per-component comparison of well-formed pins (guaranteed by SPEC-050); stale is true iff pin.major!=current.major OR pin.minor!=current.minor (a MINOR/MAJOR bump). A patch-only difference is not stale (free within major.minor)."
      - name: ClassifyRequirementTraceability
        kind: function
        signature: "func ClassifyRequirementTraceability(records []ArtifactStatusRecord, refs []TraceRef) StepResult"
        notes: "The pure full-corpus classifier, MODELED ON ClassifyStatusDrift. Joins bundle records to refs by BUNDLE NAME (ref.BundleName -> record.BundleName; NOT by number) and refs to citing records by NORMALIZED path (ref.CitingPath -> record.Path, both pre-normalized by the wiring). Emits ONE combined StepResult: error-severity BLOCK violations (delivered coverage failures REQ-002/003/004/005, downstream-outruns-chain states REQ-006 including the plan two-hop, stale-pin broken promises REQ-008) and warning-severity ADVISORY violations (in-flight uncovered REQs on non-delivered bundles REQ-007). Emits NO ref-resolution findings (tier a is validate's, REQ-007)."
      - name: SplitTraceabilityResult
        kind: function
        signature: "func SplitTraceabilityResult(combined StepResult) (block StepResult, advisory StepResult)"
        notes: "Partitions the combined result by severity into the policied block surface (StepRequirementTraceability, error-severity) and the non-policied advisory surface (StepRequirementTraceabilityAdvisory, warning-severity), exactly as SplitDriftResult does for drift."
    consumes:
      - source: pkg/gate
        name: ArtifactStatusRecord
        kind: type
      - source: pkg/gate
        name: StatusClass
        kind: type
  - file: cmd/backstop/gate.go
    provides:
      - name: buildRequirementTraceabilitySteps
        kind: function
        signature: "func buildRequirementTraceabilitySteps(projectRoot string) (gate.StepFunc, gate.StepFunc)"
        notes: "Wires the two surfaces into the shipped gate, MIRRORING buildStatusDriftSteps: one lazy (sync.Once) resolution that reuses gate.ResolveArtifactStatus (now bundle-aware) for statuses + bundle states and validate.CollectSupportRefs (v1.2.0, which exempts terminal citers) for the citing refs (converted to []gate.TraceRef). It NORMALIZES both the converted TraceRef.CitingPath and each record.Path via gate.NormalizePath(projectRoot, ...) so the join keys agree (exactly as computeDriftSurfaces normalizes violation.File), then runs ClassifyRequirementTraceability + SplitTraceabilityResult and returns the block + advisory StepFuncs inserted into the step list next to the drift steps. A resolve failure is a config-error under the block name; the advisory stays a clean pass. This shipped path — not just classifier unit tests over hand-built records — is proven by a real fixture-corpus integration test (a delivered bundle with an uncovered REQ actually resolves to a KindBundle record and actually blocks)."
    consumes:
      - source: pkg/gate
        name: ResolveArtifactStatus
        kind: function
      - source: pkg/gate
        name: ClassifyRequirementTraceability
        kind: function
      - source: pkg/gate
        name: SplitTraceabilityResult
        kind: function
      - source: pkg/gate
        name: TraceRef
        kind: type
      - source: pkg/validate
        name: CollectSupportRefs
        kind: function
      - source: pkg/validate
        name: SupportRef
        kind: type
---

# SPEC-052: Requirement Traceability Gate Step

## Overview

Backstop's central promise is that a green gate on an `implemented` spec is
mechanical proof of every requirement in that spec. BUNDLE-014 welds the one hop
that promise did not cover — bundle REQ -> spec requirement. SPEC-050 (Seed 1)
built the versioning schema and the both-direction RESOLUTION of `supports` refs
in `artifact validate`; SPEC-051 (Seed 2) reconciled and `1.0.0`-backfilled the
legacy corpus so mandatory-pin validation turns on green. This spec is Seed 3 —
the enforcement keystone: the `requirement_traceability` gate step that welds the
COVERAGE direction ("is every bundle REQ supported by a real, `implemented` spec
requirement?") as a corpus-STATE invariant.

Resolution (a ref points at a real bundle/REQ/version) is a per-artifact DEFECT
check and lives in validate (SPEC-050). Coverage is the OTHER direction, is
corpus-spanning, and is judged as a STATE — so it lives in a gate step built on
the exact block/advisory pattern the status-drift dimension already uses
(`ClassifyStatusDrift` / `SplitDriftResult`, pkg/gate/status_drift.go). The step
does five things, all pure artifact-domain logic with zero language or tool
knowledge:

1. **Mint the step + advisory twin (REQ-001).** A new `requirement_traceability`
   block-surface step and its non-policied `requirement_traceability_advisory`
   twin, NOT the reserved `ledger_integrity`.
2. **Delivered-gate coverage (REQ-002, REQ-003, REQ-004, REQ-005).** A `delivered`
   bundle needs every LIVE citing spec `implemented` and every REQ covered by >=1
   `implemented` spec requirement, per-REQ; retired specs never flow through;
   issue requirements are queryable lineage, never coverage.
3. **State invariant (REQ-006).** The step blocks any corpus STATE where a
   downstream artifact outruns its upstream chain; dispatch-time refusal is a
   hook/runtime SEAM consuming the verdict, never core intercepting events.
4. **Three-tier posture (REQ-007).** Defects (validate's), broken promises, and
   in-flight gaps map to block / block / advisory-warn.
5. **Stale-pin model (REQ-008).** Lifecycle-keyed, semver-gated: PATCH free,
   MAJOR/MINOR forces downstream re-pin (unimplemented) or a new spec + drop-out-
   of-`delivered` (delivered).

This spec covers BUNDLE-014 REQ-006, REQ-008, REQ-009, REQ-010, REQ-011, REQ-012,
REQ-013, REQ-014 ONLY. The version/log schema + resolution (Seed 1, SPEC-050) and
the corpus reconciliation + backfill (Seed 2, SPEC-051) are out of scope and are
hard dependencies — this step lands AFTER both.

## Requirements

Requirements REQ-001 through REQ-008 are defined in frontmatter and trace to
BUNDLE-014 REQ-006/008/009/010/011/012/013/014 via `supports`. Each requirement
has at least one claim; claims are defined in frontmatter. (This spec's own
`supports` refs are written UNPINNED because SPEC-050's mandatory-pin enforcement
is not yet live at authoring time; SPEC-051's backfill stamps `@1.0.0` onto them
when the flip lands — same convention as SPEC-050. See Sharp Edges.)

### Step surfaces (REQ-001)

| Step name | Surface | Severity it carries | Policied? |
|-----------|---------|---------------------|-----------|
| `requirement_traceability` | block | error | yes (but no `applies-to: new-code`, see below) |
| `requirement_traceability_advisory` | advisory | warning | NO — no policy can upgrade it to a block |

`ledger_integrity` is NOT claimed — it stays reserved for SPEC-010's provenance
ledger hash chain.

### Coverage-source matrix (REQ-003, REQ-004, REQ-005)

Which citing artifact/status closes (covers) a `delivered` bundle REQ. This is the
complete matrix — every cell has a claim.

| Citing artifact | Status | Covers a bundle REQ? |
|-----------------|--------|----------------------|
| spec | `implemented` | YES |
| spec | `draft` | no |
| spec | `ready-for-implementation` | no |
| spec | `replaced` | no |
| spec | `canceled` | no |
| spec | `deprecated` | no |
| spec | `obsoleted` | no |
| issue | any | no (queryable lineage only) |

### Delivered-gate by bundle maturity (REQ-002, REQ-003)

| Bundle maturity | Delivered-gate applies? | Uncovered REQ outcome |
|-----------------|-------------------------|-----------------------|
| `exploring` | no | advisory warn |
| `defined` | no | advisory warn |
| `ready` | no | advisory warn |
| `delivered` | YES | BLOCK (broken promise) |
| `replaced` | no (terminal-excluded) | — |
| `canceled` | no (terminal-excluded) | — |
| `deprecated` | no (terminal-excluded) | — |

For a `delivered` bundle, additionally every LIVE (`draft` / `ready-for-implementation`
/ `implemented`) citing spec must be `implemented` (REQ-002); a `draft` or
`ready-for-implementation` citing spec blocks. Terminal-retired citing specs are
excluded from the live-citing check (they are not unfinished promises) but still
provide no coverage.

### Three-tier posture (REQ-007)

| Tier | Example | Surface | Blocks? |
|------|---------|---------|---------|
| (a) defect | dangling / unpinned / fabricated-version ref | `artifact_validation` (validate, SPEC-050) | yes — not this step |
| (b) broken promise | delivered coverage failure; downstream outruns chain | `requirement_traceability` | yes |
| (c) in-flight gap | uncovered REQ on a non-`delivered` bundle | `requirement_traceability_advisory` | never |

The block surface is neither WAIVABLE (not in the gate's code-located waivable
dimension set) nor baseline-grandfathered (no `applies-to: new-code` policy entry).

### Stale-pin matrix (REQ-008)

Bump size (downstream pin vs the REQ's CURRENT version) x bundle lifecycle.

| Bump | Bundle lifecycle | Outcome |
|------|------------------|---------|
| none (pin == current) | any | satisfied |
| PATCH (same major.minor) | unimplemented | satisfied (free) |
| PATCH (same major.minor) | delivered | satisfied (free) |
| MINOR | unimplemented | stale-pin BLOCK (re-pin + re-work) |
| MAJOR | unimplemented | stale-pin BLOCK (re-pin + re-work) |
| MINOR | delivered | new spec required; bundle BLOCKS as not-delivered until a spec pinned to the new version is `implemented` |
| MAJOR | delivered | new spec required; bundle BLOCKS as not-delivered |

"Same major.minor" means `pin.major == current.major && pin.minor == current.minor`,
compared numerically per component. The current version is the REQ's top-level
`version:` (the newest log entry) per SPEC-050's schema.

## Implementation

### Package layout

- `pkg/gate/result.go` — the two new step-name constants (REQ-001).
- `pkg/gate/artifact_status.go` — EXTEND `ResolveArtifactStatus` to also resolve
  `bundles/` with a DEDICATED bundle frontmatter struct + walk (mirroring
  `walkPlanDir`/`planFrontmatter`), NOT by reusing `artifactFrontmatter`: a bundle's
  `status:` is a YAML map (`status.maturity`) and `bundle:` is a map (`bundle.name`),
  so routing a `.bundle.md` through the string-`Status` `artifactFrontmatter` would
  `yaml.TypeError` and be SILENTLY SKIPPED by `walkArtifactDir` (zero bundle records,
  vacuous green). The dedicated reader captures `bundle.name` (join key),
  `status.maturity` (nested — reuse the `extractMaturity`-style read), and
  `requirements[]` (id + current version). It surfaces `BundleName`, the extracted
  maturity in `Status` (fed to `ClassifyArtifactStatus` KindBundle -> `delivered` ->
  `ClassSuccessTerminal`), and `BundleReqs []BundleReqVersion`. It shares the corpus
  resolution, not a parallel repo walk.
- `pkg/gate/requirement_traceability.go` (NEW) — the pure classifier
  (`ClassifyRequirementTraceability`), the severity split
  (`SplitTraceabilityResult`), the coverage-source predicate (`SupportsCoverage`),
  and the semver stale-pin gate (`StalePinVerdict`). No filesystem access, no
  language knowledge — operates on already-resolved records + refs.
- `cmd/backstop/gate.go` — `buildRequirementTraceabilitySteps` wires the two
  surfaces into the shipped step list, mirroring `buildStatusDriftSteps`, reusing
  `ResolveArtifactStatus` and SPEC-050's `CollectSupportRefs`.

### Classification passes (the mechanical steps a planner maps tasks to)

`ClassifyRequirementTraceability(records, refs)` runs these passes over the
resolved corpus; each is independently testable:

1. **Corpus join.** Index bundle records (`KindBundle`) by `BundleName` — supports
   refs cite a bundle by NAME, not number, and a delivered bundle may carry no
   `number:` at all, so the name is the only viable key — and index citing records
   (spec/issue/plan) by their NORMALIZED `Path`. Both `TraceRef.CitingPath` and each
   record `Path` are normalized identically (via `gate.NormalizePath` in the wiring)
   before the join, so a validate-relative ref path and a gate-side record path
   agree rather than silently missing. Build, per bundle, the set of its declared
   REQs + current versions (REQ-003 target, REQ-008 comparison source).
2. **Coverage resolution (REQ-003, REQ-004, REQ-005).** For each `delivered`
   bundle REQ, scan the refs that cite it; a REQ is COVERED iff some citing ref's
   artifact satisfies `SupportsCoverage` (spec + success-terminal). Issue refs and
   non-`implemented`/retired spec refs never cover. Per-REQ, not aggregate.
   (Citing refs arrive from SPEC-050 v1.2.0's `CollectSupportRefs`, which already
   exempts terminal citers; `SupportsCoverage` is nonetheless the authority and
   returns false for any non-`implemented` citing artifact, defensive of that
   upstream filter.)
3. **Live-citing check (REQ-002).** For each `delivered` bundle, every LIVE citing
   spec (`draft` / `ready-for-implementation` / `implemented`) must be
   `implemented`; a live non-`implemented` citing spec is a broken-promise BLOCK.
   Terminal-retired citing specs are skipped here (and, per pass 2's note, are
   absent from the production ref set anyway).
4. **Stale-pin resolution (REQ-008).** For each citing ref, compare its
   `PinVersion` to the cited REQ's current version via `StalePinVerdict`. A stale
   pin (MINOR/MAJOR) on a non-`delivered` bundle is a downstream stale-pin BLOCK;
   on a `delivered` bundle it makes the bundle not-delivered until a spec pinned to
   the current version is `implemented`. PATCH / equal pins are free.
5. **State invariant (REQ-006).** A downstream artifact whose upstream chain does
   not verify is a BLOCK. For a spec advanced to `ready-for-implementation`/
   `implemented`, "chain does not verify" is a stale pin against an unimplemented
   bundle (pass 4). For a PLAN, the plan carries no `supports` of its own — its link
   is the TWO-HOP `ArtifactStatusRecord.SpecID` -> the spec record it implements ->
   that spec's own `TraceRef`s; the plan verifies iff that spec's chain verifies, so
   a plan existing for a spec whose bundle chain does not verify blocks (CLM-027).
   Delivered-without-coverage is the full-depth instance of the same invariant. The
   plan-for-unverified-spec block is DISTINCT from the coverage/stale-pin block that
   may already stand on the spec itself: the spec's block reports the spec's own
   broken chain, while this one reports that a plan was allowed to exist atop it —
   the SpecID hop is what attributes the finding to the plan, so it is not
   double-counting the spec's finding.
6. **Posture assignment (REQ-007).** Broken-promise findings (delivered coverage
   failure, downstream-outruns-chain, stale-pin) get `Severity: "error"`. In-flight
   gaps (uncovered REQ on a non-`delivered` bundle whose downstream has not
   advanced past what its chain supports) get `Severity: "warning"`. No ref-
   resolution finding is emitted — that is validate's tier (a).

`SplitTraceabilityResult` then partitions the combined result by severity into the
policied block surface and the non-policied advisory surface, exactly as
`SplitDriftResult` does — a single classification, two derived surfaces.

### Where the step runs, and why STATE not events

The step is wired next to the drift steps in the assembled `cmd/backstop/gate.go`
step list (before `waiver_resolution` / `baseline_comparison`). Its block surface
carries `Status: "fail"` when any error-severity violation exists, which flips
`GateResult.Pass` in `NewGateResult` — so it blocks intrinsically, with NO
`applies-to: new-code` policy entry, meaning findings are never baseline-
grandfathered (an artifact-state broken promise must be fixed, not snapshotted).
The advisory surface carries `Status: "warning"`, counted in `StepsWarned`, never
flipping `Pass`, and — like `artifact_status_drift_advisory` — carries no policy
entry so it can never be upgraded.

Core judges the resulting STATE, never a promotion/dispatch EVENT: because
promotions land as edits and every gate dimension runs at every verification step
(the no-check-filtering razor), every transition is judged by construction.
Dispatch-time refusal (blocking `/implement` before it starts) is a downstream
SEAM — the gate-on-implement hook reads the `requirement_traceability` verdict out
of the gate output today, the opencode runtime later. Core supplies the verdict; it
does not subscribe to, intercept, or refuse the event.

## Verification

Verification is defined in frontmatter: integration level, 80% coverage threshold,
targeting `pkg/gate` and `cmd/backstop`. Integration level is chosen because the
load-bearing behavior is cross-package wiring — the pure classifier in `pkg/gate`
is inert until `buildRequirementTraceabilitySteps` in `cmd/backstop` reuses
`ResolveArtifactStatus` (now bundle-aware) and SPEC-050's `CollectSupportRefs` to
assemble the corpus and runs it, and the requirement is precisely that the two
surfaces land in the shipped gate step list. A unit-only verification of `pkg/gate`
would prove the classifier correct while leaving the wiring — the exact place this
could ship dark (a `.bundle.md` silently skipped, a path-mismatched join dropped) —
unproven. A REAL fixture-corpus integration test is therefore MANDATORY (CLM-049):
a fixture with a delivered bundle carrying an uncovered REQ must, through the
shipped `buildRequirementTraceabilitySteps` path, actually resolve to a bundle
record and actually block — the classifier unit tests over hand-built records do
NOT substitute for it, given the vacuous-green stakes of the bundle-walk and join.
Claims are defined in frontmatter; every requirement has at least one claim, and
the matrices are covered cell by cell: coverage-source (8 cells, CLM-016..023),
delivered-gate-by-bundle-maturity (7 maturities, CLM-006..012 plus the
name-without-number join CLM-048), and stale-pin (the 7 outcome rows via
CLM-039..045, plus the delivered-recovery CLM-046 and the numeric major.minor read
CLM-047).

## Sharp Edges

- **Depends on SPEC-050 AND SPEC-051 — sequencing is the guard, not a runtime
  branch.** This step reads bundle REQ versions and citing pins that only EXIST
  after SPEC-050's schema + SPEC-051's `1.0.0` backfill land. It MUST land after
  both. Because the corpus is uniformly versioned and green by then, the step
  never has to reason about an unbackfilled / unpinned corpus — there is no
  "if unpinned" branch to write. If it landed early, every `delivered` bundle
  would red at once. State the ordering; do not build a tolerance mode.

- **Terminal citers are exempted UPSTREAM (SPEC-050 v1.2.0) — the classifier is
  defensive, not the sole filter.** SPEC-050 v1.2.0's `CollectSupportRefs` skips
  `replaced`/`canceled`/`deprecated`/`obsoleted` citing artifacts (mirroring
  pkg/validate/spec.go's terminal-citer exemption), so a terminal spec is neither
  resolution-checked NOR coverage — its refs never reach this step in production.
  `SupportsCoverage` still returns false for a retired-terminal citing artifact so
  the verdict is correct even in a unit test that constructs such a ref directly
  (defensive-in-depth). Do NOT rely on the classifier to re-derive the exemption,
  and do NOT let the classifier resolution-check a citer (tier a is validate's).

- **Retired BUNDLE exclusion vs retired SPEC non-support are DIFFERENT exclusions
  — do not conflate them.** A `replaced`/`canceled`/`deprecated` BUNDLE is excluded
  from the delivered-gate entirely (no coverage obligation), consistent with the
  gate's existing terminal-exclusion. A `replaced` (or otherwise retired) SPEC is
  the opposite case: the delivered gate is fully in force for its (live) bundle,
  and the retired spec simply provides NO coverage (REQ-004) and is skipped by the
  live-citing check (REQ-002). One exclusion is "the bundle is retired, skip it";
  the other is "the spec is a tombstone, it covers nothing." Collapsing them would
  let a delivered bundle claim coverage from a replaced spec (the BUNDLE-011
  REQ-007/010 bug SPEC-051 fixes) or would wrongly block a retired bundle.

- **`implemented` is the ONLY coverage-closing spec status — draft/ready are not
  partial credit.** A `ready-for-implementation` spec is planned but unproven; its
  requirement is not mechanically verified down to code, so it must NOT count as
  coverage. The temptation to accept "ready-for-implementation" as "close enough"
  reintroduces the assumed-not-proven hole the whole bundle closes. `SupportsCoverage`
  keys off `ClassSuccessTerminal` (== `implemented`) exclusively.

- **The advisory surface must be a SEPARATE step name, not a severity flag on the
  block step.** If in-flight gaps rode the `requirement_traceability` step at
  warning severity, a future enforcement.policy entry targeting that step could
  upgrade them to a block (the drift dimension hit exactly this, which is why
  `artifact_status_drift_advisory` exists as its own name). The non-policied twin
  is structural non-upgradability. Emit in-flight gaps on
  `requirement_traceability_advisory` only.

- **Non-waivable and non-baselineable are deliberate, and asymmetric to the code-
  debt dimensions.** `pack_engines` / `test_substantiveness` are waivable and
  baseline-grandfathered because code debt accretes per-file and needs an
  incremental-adoption ramp. An artifact-state broken promise (a `delivered`
  bundle with an uncovered REQ) does not accrete and must never be snapshotted as
  acceptable — so `requirement_traceability` is left OUT of the waivable dimension
  set and carries no `applies-to: new-code` policy entry. Adding either would let a
  broken delivered promise hide behind a waiver token or a stale baseline. (NB:
  `artifact_status_drift`'s block surface DOES grandfather via its policy entry —
  this dimension deliberately does not, because a delivered-coverage promise is not
  accreting debt.)

- **Stale-pin comparison is by NUMERIC semver component, and only needs the CURRENT
  version.** `1.10.0` shares major.minor with `1.10.9` but NOT with `1.9.0`; string
  comparison gets this wrong. The stale-pin gate needs only the REQ's current
  version (top-level `version:`), never the full log — log MEMBERSHIP (does the pin
  resolve to a real historical entry) is SPEC-050's validate-side concern (REQ-003
  there), already enforced before this step runs. Do not re-resolve the log here.

- **The bundle reader is DEDICATED — do NOT route `.bundle.md` through the string-
  `Status` `artifactFrontmatter`.** A bundle's `status:` is a YAML MAP
  (`status.maturity`) and `bundle:` is a map (`bundle.name`); the shared
  `artifactFrontmatter.Status` is a `string`, so unmarshalling a bundle through it
  is a `yaml.TypeError`, and `walkArtifactDir` (artifact_status.go) SILENTLY SKIPS a
  file whose unmarshal errors — every `.bundle.md` would vanish, giving zero bundle
  records and a vacuously-GREEN coverage step. Extend `ResolveArtifactStatus` with a
  dedicated bundle frontmatter struct + walk (mirroring `walkPlanDir`/`planFrontmatter`)
  reading `bundle.name`, nested `status.maturity`, and `requirements[]` (id +
  version). This is still the one canonical resolution (citing refs from SPEC-050's
  `CollectSupportRefs`), not a parallel repo walk — but it is a NEW reader, not a
  reuse of the string-status one.

- **Join on bundle NAME and on NORMALIZED paths — both are silent-drop hazards.**
  Supports refs cite a bundle by NAME (`bundle.name`), and a delivered bundle may
  have no `number:` at all (agent-definitions), so `ID = fm.Number` is not the join
  key — join `TraceRef.BundleName` to the record's `BundleName`. And
  `TraceRef.CitingPath` (from `validate.SupportRef.File`) vs the record `Path` (from
  `filepath.Join`) can differ by absolute/relative form; normalize BOTH via
  `gate.NormalizePath` before joining (the drift wiring does exactly this for
  `violation.File`). Either mismatch silently drops every join — which reads as
  green, the worst failure mode. Both are guarded by claims (CLM-048, CLM-050) and
  the mandatory fixture-corpus integration test (CLM-049).

- **Dispatch refusal is a SEAM, not core behavior — resist adding an event hook.**
  The founder's model is that core judges STATES and the runtime/hook layer
  consumes the verdict. It is tempting to have the step "refuse" a promotion; it
  must not. It emits a StepResult; the gate-on-implement hook (and later the
  opencode runtime) decides what to block. Keep `pkg/gate` free of any
  promotion-event subscription.

## Review Questions

1. Does `SupportsCoverage` return true ONLY for `KindSpec` + `ClassSuccessTerminal`
   (`implemented`), so that draft/ready-for-implementation specs, all retired
   spec statuses, and every issue ref return false — covering the full coverage-
   source matrix? (REQ-003/REQ-004/REQ-005.)

2. Is a retired BUNDLE (`replaced`/`canceled`/`deprecated`) excluded from the
   delivered-gate, while a retired SPEC citing a live delivered bundle is treated
   as non-support (not a bundle exclusion) — the two exclusions kept distinct?
   (REQ-002/REQ-004.)

3. Is coverage evaluated PER-REQ (a delivered bundle with one covered and one
   uncovered REQ blocks and names the uncovered REQ), not as an aggregate count?
   (REQ-003.)

4. Does an issue `supports` ref appear in the traceability corpus as a queryable
   lineage link yet never close a bundle REQ, and does a co-citing issue ref leave
   an already-`implemented`-covered REQ's coverage unchanged? (REQ-005.)

5. Does the block surface flip `GateResult.Pass` intrinsically with NO
   `applies-to: new-code` policy entry (so it is never baseline-grandfathered), and
   is `requirement_traceability` absent from the gate's waivable dimension set (so
   `@waiver` cannot suppress it)? (REQ-007.)

6. Is the in-flight-gap WARN emitted ONLY on the separate
   `requirement_traceability_advisory` step name (carrying no policy entry), so no
   enforcement.policy can upgrade it to a block, exactly like
   `artifact_status_drift_advisory`? (REQ-007/REQ-001.)

7. Does `StalePinVerdict` treat a PATCH-only difference as free (not stale) and a
   MAJOR/MINOR difference as stale by NUMERIC per-component comparison, and does the
   stale outcome fork correctly on bundle lifecycle (re-pin block for unimplemented,
   new-spec-required not-delivered block for delivered)? (REQ-008.)

8. Does the step emit NO ref-resolution / dangling-ref findings (tier a stays
   validate's), and does `pkg/gate` subscribe to / intercept NO promotion or
   dispatch event (dispatch refusal stays a hook/runtime seam)? (REQ-006/REQ-007.)

9. Is bundle maturity + REQ current-version sourced from a `KindBundle` case in the
   EXISTING `ResolveArtifactStatus` walker (not a second walker), and are citing
   refs sourced from SPEC-050 v1.2.0's `CollectSupportRefs` (terminal citers
   already exempted)? (Implementation reuse.)

## References

- BUNDLE-014 (requirement-traceability): source bundle; Seed 3 of 3. Resolved
  decisions RDQ-1/RDQ-3/RDQ-4/RDQ-7/RDQ-8/RDQ-10; DD-1/DD-2/DD-3/DD-6/DD-7/DD-8/
  DD-9/DD-11/DD-12; requirements REQ-006, REQ-008..REQ-014.
- SPEC-050 (Seed 1, now v1.2.0): the version/log schema, mandatory pin syntax,
  both-direction resolution in `artifact validate`, and `CollectSupportRefs` (whose
  v1.2.0 terminal-citer exemption skips replaced/canceled/deprecated/obsoleted
  citers) — DEFINES what this step consumes (current version per REQ, citing refs).
  Hard dependency.
- SPEC-051 (Seed 2): the one-time reconciliation + `1.0.0` backfill that makes the
  corpus uniformly versioned and green before this step turns on. Hard dependency.
- `pkg/gate/status_drift.go` (`ClassifyStatusDrift`, `SplitDriftResult`,
  `driftStepResult`): the block/advisory pattern this step is modeled on.
- `pkg/gate/artifact_status.go` (`ResolveArtifactStatus`, `ClassifyArtifactStatus`,
  `ArtifactStatusRecord`, `StatusClass`): the corpus walker extended with a
  `KindBundle` case (bundle maturity + REQ versions).
- `cmd/backstop/gate.go` (`buildStatusDriftSteps`, the assembled `steps` list): the
  wiring this step mirrors and is inserted into.
- `pkg/gate/result.go` (`StepLedgerIntegrity`, `AllStepNames`): `ledger_integrity`
  stays reserved; the new step names are dynamically-wired, not added to the fixed
  nine-name array.
- `pkg/gate/policy.go` (`ApplyPolicy`, `policyMetaStep`) and `pkg/gate/step_waiver.go`
  (`waivableDimension`): why the block surface is non-upgradable-from-advisory,
  non-baselineable, and non-waivable.
- [[feedback_loud_not_blocking]] — the three-tier posture (block defects + broken
  promises, advisory-warn in-flight gaps).
- [[project_artifact_terminal_states]] — `replaced`/`delivered` semantics behind
  the retired-spec non-support and delivered-gate scope.
- [[project_gate_scoped_to_implemented]] — DIR-015 scoped the lower dimensions to
  `implemented` specs; this step is the layer above, extending "mechanical proof"
  one hop up as a state invariant.

## Version History

- **1.0.0 (2026-07-14, draft)** — Initial spec. The `requirement_traceability` gate
  step (Seed 3 of BUNDLE-014): 8 requirements mapping to bundle REQ-006/008..014, 47
  claims, the coverage-source / delivered-gate-by-maturity / stale-pin matrices
  covered cell by cell, and the status_drift block/advisory mirror. Wove in SPEC-050
  v1.2.0's terminal-citer exemption (terminal citers neither coverage nor
  resolution-checked).
- **1.1.0 (2026-07-14, draft)** — spec-reviewer fixes (verdict FAIL -> resolved),
  no scope change. Two blockers: (1) the ref->bundle join must key on `bundle.name`
  (not `number:`, which agent-definitions lacks) — added the `ArtifactStatusRecord.BundleName`
  join key, the name-join in the classifier/passes, and CLM-048 (name-without-number).
  (2) A bundle's `status:`/`bundle:` are YAML maps, so the walker must use a DEDICATED
  bundle frontmatter reader (not the string-`Status` `artifactFrontmatter`, which
  `yaml.TypeError`s and silently skips) — corrected the contract, package layout, and
  Sharp Edges, and mandated a real fixture-corpus INTEGRATION test (CLM-049). Two
  should-fixes: CitingPath<->Path `gate.NormalizePath` normalization (CLM-050 + wiring
  note) and the plan two-hop `SpecID` -> spec -> spec TraceRefs made explicit in pass 5
  and CLM-027. Nits: precise matrix/claim counts in Verification; a sentence
  distinguishing the plan-for-unverified-spec block from the spec's own block
  (no double-count). Claims 47 -> 50 (CLM-048/049/050). `spec_version` 1.0.0 -> 1.1.0.
- **1.1.1 (2026-07-14, draft)** — Consistency-pass fix F2: bundle REQ-007 (the DD-8
  placement split) was orphaned (no spec in the 050/051/052/053 set cited it), so a
  delivered BUNDLE-014 would block its own REQ-007. REQ-001 embodies the gate-side of
  that split, so its `supports` is now a LIST co-citing `requirement-traceability:REQ-006`
  and `:REQ-007`, with a one-line note (SPEC-050 owns the resolution-in-validate half).
  No claim/scope change. `spec_version` 1.1.0 -> 1.1.1.
