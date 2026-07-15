---
title: "Gate Trace Structured Fields"
schema_version: issue/v1

issue:
  id: ISSUE-059
  title: "Gate Trace Structured Fields"
  type: enhancement
  status: ready
  created: "2026-07-15"

complexity:
  scope: contained
  uncertainty: known
  risk: safe

verification:
  level: unit
  coverage_threshold: 90
  test_command: "go test ./pkg/gate/... ./cmd/backstop/... -run Trace"

implementation:
  summary: >
    Add an additive `trace` object to every violation emitted by the
    `requirement_traceability` gate step (block + advisory) and add top-level
    `git_sha`/`generated_at` provenance fields to the gate JSON output, without
    touching identity/baseline computation.
  package: pkg/gate, cmd/backstop

requirements:
  - id: REQ-001
    supports: requirement-traceability:REQ-006@1.0.0
    text: >
      Every `Violation` emitted by `ClassifyRequirementTraceability`
      (`pkg/gate/requirement_traceability.go`) — both the block
      (`requirement_traceability`) and advisory (`requirement_traceability_advisory`)
      steps — must carry an additive `trace` object alongside the existing
      `rule`/`file`/`message`/`severity` fields. `trace` is present on every
      traceability violation, no exceptions carved out per gap kind.
  - id: REQ-002
    supports: requirement-traceability:REQ-006@1.0.0
    text: >
      The `trace` object's unconditional fields are `bundle` (bundle name),
      `bundle_maturity` (the bundle's maturity status at classification time,
      denormalized deliberately — see REQ-005), `req_id`, `req_version` (the
      bundle REQ's current strict MAJOR.MINOR.PATCH version), `gap_kind`
      (enum, see REQ-003), and `remedy` (enum, see REQ-003).
  - id: REQ-003
    supports: requirement-traceability:REQ-006@1.0.0
    text: >
      `gap_kind` is a closed enum mapped one-to-one to the BUNDLE-014 design
      decisions that produce each violation shape: `uncovered` (DD-2 — a bundle
      REQ that never had implemented-spec support; remedy `author_spec`),
      `coverage_lapsed` (DD-3 — a bundle REQ whose only citer is a
      replaced/retired-terminal spec, detected via a retired-terminal citer
      existing for the REQ; remedy `restate_supports`; distinct from `uncovered`
      because the fix is re-pinning an existing spec's supports ref, not
      authoring a new one), `citing_spec_not_implemented` (DD-1 — a delivered
      bundle cited by a non-implemented spec; remedy `implement_spec`),
      `stale_pin` (DD-12 — a supports ref pins an older major/minor than the
      REQ's current version; remedy is lifecycle-keyed per REQ-004),
      `chain_outrun` (DD-11 family — any state where a downstream artifact's
      upstream chain does not verify, e.g. a plan targeting a spec whose bundle
      chain doesn't verify; remedy `resolve_upstream`).
  - id: REQ-004
    supports: requirement-traceability:REQ-006@1.0.0
    text: >
      For `gap_kind: stale_pin`, `trace` additionally carries `pinned_version`
      and `bump` (`none`/`patch`/`minor`/`major`/`unknown`, per
      `StalePinVerdict`), and `remedy` is lifecycle-keyed rather than fixed:
      `re_pin` when the bundle is not yet `delivered` (DD-12 unimplemented
      branch — rework/re-pin against the current version), `new_spec` when the
      bundle is `delivered` (DD-12 delivered branch — existing specs are
      immutable; a new implemented spec is required and the bundle drops out
      of `delivered` until it lands). The JSON must tell the operator which
      DD-12 branch applies without them re-deriving it from bundle maturity.
  - id: REQ-005
    supports: requirement-traceability:REQ-006@1.0.0
    text: >
      For `gap_kind` values `stale_pin` and `citing_spec_not_implemented`,
      `trace` additionally carries `citing_artifact` (the ID of the spec/issue
      whose supports ref produced the violation). `bundle_maturity` stays on
      every violation (not gated to these two kinds) — it is denormalized
      deliberately so a corpus-parsing consumer can assert bundle-maturity
      consistency per violation without a second lookup; this is only a safe
      simplification because of the top-level provenance added by REQ-007.
  - id: REQ-006
    supports: requirement-traceability:REQ-006@1.0.0
    text: >
      For `gap_kind: chain_outrun`, `trace` additionally carries `via` — a
      pointer identifying the broken upstream link the outrunning artifact
      builds on (e.g. the spec ID a stale/non-verifying plan targets). The
      pointer is the actionable content: an operator following `via` lands on
      the artifact that must be fixed first, not the one the violation fired on.
  - id: REQ-007
    supports: requirement-traceability:REQ-006@1.0.0
    text: >
      The gate JSON output (`GateResult`, `pkg/gate/result.go`) gains two
      top-level provenance fields: `git_sha` (the repository HEAD commit SHA
      at gate-run time) and `generated_at` (RFC 3339 timestamp of gate-run
      completion). These make gate-run-vs-corpus-parse skew detectable by a
      downstream consumer and are the reason `bundle_maturity` can safely stay
      denormalized per-violation (REQ-005) instead of requiring a second,
      time-skewed lookup.
  - id: REQ-008
    supports: requirement-traceability:REQ-006@1.0.0
    text: >
      All new fields (`trace` on violations; `git_sha`/`generated_at` on
      `GateResult`) are additive under the current gate `schema_version`
      (`gate/v1`) — existing consumers parsing only `rule`/`file`/`message`/
      `severity`/`identity`/`identity_hash`/`region_hash` are unaffected.
      Consumers encountering an unrecognized `gap_kind` or `remedy` value MUST
      degrade to rendering the violation's prose `message` and must not error
      or crash; a future breaking change to this vocabulary requires a gate
      `schema_version` bump, not a silent field reinterpretation. This
      contract is stated in the emitted docs/contract, not only in this issue.
  - id: REQ-009
    supports: requirement-traceability:REQ-006@1.0.0
    text: >
      `Identity`, `IdentityHash`, and `RegionHash` computation
      (`EnrichViolationIdentity` and callers) is unchanged by this work —
      `trace` is excluded from identity computation the same way `Line`,
      `ProjectWide`, and `WaiverHint` already are (`pkg/gate/result.go`).
      Existing baselines and waivers keyed on identity/identity_hash remain
      valid with zero re-baselining forced by this change. Existing envelope
      fields (`rule`/`file`/`message`/`severity`) are unchanged in name, type,
      and meaning.
  - id: REQ-010
    supports: requirement-traceability:REQ-006@1.0.0
    text: >
      All docs, field descriptions, and violation messages introduced or
      touched by this work say "covered" (a bundle REQ proven by an
      implemented spec) — never "delivered", which names bundle maturity
      exclusively and must not be used to describe REQ-level coverage state.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: >
      Every violation returned by `ClassifyRequirementTraceability`, across
      both the block and advisory steps, has a non-nil `trace` field with no
      gap_kind excluded.
    tests:
      - TestTrace_Fields_EveryViolationCarriesTraceObject
      - TestTrace_Fields_AdvisoryStepAlsoCarriesTraceObject
  - id: CLM-002
    requirement: REQ-002
    text: >
      `trace.bundle`, `trace.bundle_maturity`, `trace.req_id`, `trace.req_version`,
      `trace.gap_kind`, and `trace.remedy` are populated correctly for each of
      the five existing violation-producing code paths.
    tests:
      - TestTrace_Fields_UncoveredCarriesBundleReqAndVersion
      - TestTrace_Fields_BundleMaturityMatchesRecordAtClassification
  - id: CLM-003
    requirement: REQ-003
    text: >
      `gap_kind`/`remedy` are correctly assigned per code path: `uncovered`→
      `author_spec`, `coverage_lapsed`→`restate_supports`,
      `citing_spec_not_implemented`→`implement_spec`, `stale_pin`→(lifecycle-keyed,
      see CLM-004), `chain_outrun`→`resolve_upstream`. `coverage_lapsed` is
      distinguished from `uncovered` by detecting a retired-terminal citer for
      the REQ, and produces a different `gap_kind`/`remedy` than the previous
      generic uncovered message even though the underlying violation still fires.
    tests:
      - TestTrace_GapKind_UncoveredMapsToAuthorSpec
      - TestTrace_GapKind_CoverageLapsedDistinctFromUncovered
      - TestTrace_GapKind_CitingSpecNotImplementedMapsToImplementSpec
      - TestTrace_GapKind_ChainOutrunMapsToResolveUpstream
  - id: CLM-004
    requirement: REQ-004
    text: >
      `stale_pin` violations carry `pinned_version` and `bump`, and `remedy` is
      `re_pin` when the bundle is not `delivered` and `new_spec` when the
      bundle is `delivered`, for the same underlying stale-pin detection.
    tests:
      - TestTrace_StalePin_RemedyRePinWhenBundleUnimplemented
      - TestTrace_StalePin_RemedyNewSpecWhenBundleDelivered
      - TestTrace_StalePin_PinnedVersionAndBumpPopulated
  - id: CLM-005
    requirement: REQ-005
    text: >
      `stale_pin` and `citing_spec_not_implemented` violations carry
      `citing_artifact` set to the citing spec/issue ID; other gap kinds omit it.
    tests:
      - TestTrace_CitingArtifact_PresentOnStalePinAndCitingSpecNotImplemented
      - TestTrace_CitingArtifact_AbsentOnUncoveredAndChainOutrun
  - id: CLM-006
    requirement: REQ-006
    text: >
      `chain_outrun` violations carry `via` pointing at the specific
      non-verifying upstream artifact (e.g. the spec ID a stale plan targets),
      not merely repeating the violating artifact's own ID.
    tests:
      - TestTrace_ChainOutrun_ViaPointsAtNonVerifyingSpec
  - id: CLM-007
    requirement: REQ-007
    text: >
      `gate --json` output includes top-level `git_sha` and `generated_at`
      fields on every invocation, sourced from the actual repository HEAD and
      wall-clock run time.
    tests:
      - TestGateCLI_JSONOutputIncludesGitSHAAndGeneratedAt
      - TestGateCLI_GitSHAMatchesRepoHEAD
  - id: CLM-008
    requirement: REQ-008
    text: >
      A JSON fixture with an unrecognized `gap_kind`/`remedy` value round-trips
      through the consumer-facing contract without error, degrading to the
      prose `message`; `schema_version` on `GateResult` stays `gate/v1` for
      this additive change.
    tests:
      - TestTrace_ForwardCompat_UnknownGapKindDegradesToMessage
      - TestGateCLI_SchemaVersionUnchangedByTraceFields
  - id: CLM-009
    requirement: REQ-009
    text: >
      `identity`/`identity_hash`/`region_hash` values for a fixed violation
      fixture are byte-identical before and after this change; a JSON snapshot
      of a pre-change baseline artifact still validates against the
      post-change `CompareBaseline`.
    tests:
      - TestTrace_Identity_UnchangedByTraceFieldAddition
      - TestTrace_Baseline_PreExistingBaselineStillCompares
  - id: CLM-010
    requirement: REQ-010
    text: >
      Grep-level check that no violation message, field doc, or contract text
      introduced by this issue uses "delivered" to describe REQ coverage state.
    tests:
      - TestTrace_Vocabulary_NoDeliveredUsedForCoverageState

contracts:
  - file: pkg/gate/requirement_traceability.go
    provides:
      - name: Trace
        kind: type
        signature: "type Trace struct { Bundle string; BundleMaturity string; ReqID string; ReqVersion string; GapKind string; Remedy string; PinnedVersion string; Bump string; CitingArtifact string; Via string }"
  - file: pkg/gate/result.go
    provides:
      - name: Violation.Trace
        kind: variable
        signature: "Trace *Trace `json:\"trace,omitempty\"`"
      - name: GateResult.GitSHA
        kind: variable
        signature: "GitSHA string `json:\"git_sha,omitempty\"`"
      - name: GateResult.GeneratedAt
        kind: variable
        signature: "GeneratedAt string `json:\"generated_at,omitempty\"`"
---

# ISSUE-059: Gate Trace Structured Fields

## Problem

The `requirement_traceability` gate step (SPEC-052, shipped 2026-07-15, the
BUNDLE-014 enforcement keystone) is mechanically sound — `ClassifyRequirementTraceability`
(`pkg/gate/requirement_traceability.go:50-155`) correctly detects every gap
class the bundle designed: uncovered requirements, stale pins, non-implemented
citing specs on delivered bundles, and chain-outrun plans. But every one of
those facts is encoded **only** in a free-text `Message` string on `Violation`
(`pkg/gate/result.go:64-99`):

```go
// pkg/gate/requirement_traceability.go:145-150
violations = append(violations, Violation{
    Rule:     rule,
    File:     bundle.Path,
    Message:  fmt.Sprintf("bundle %s requirement %s has no implemented-spec coverage", bundle.BundleName, req.ReqID),
    Severity: severity,
})
```

`Rule` only distinguishes block vs advisory (`requirement_traceability` vs
`requirement_traceability_advisory`) — every gap kind within a step shares one
rule string, differentiated solely by parsing prose. A downstream consumer
(bclabs client portal) building a dashboard on top of `gate --json` reviewed a
design proposal for structured fields and returned change requests; the design
below is the agreed result of that review, not a fresh proposal.

Two related gaps found during that review:

1. **No distinction between "never covered" and "used to be covered."** The
   ref-processing loop (`pkg/gate/requirement_traceability.go:77-79`) skips
   refs whose citer is retired-terminal, so a REQ whose only citer was a
   `replaced` spec falls through to the exact same generic uncovered message
   (line 145-150) as a REQ that was never cited at all — even though the
   remedy differs enormously (minutes of ref paperwork to re-point at a
   surviving spec, vs authoring a new spec from scratch).
2. **No provenance on the gate JSON output itself.** `GateResult`
   (`pkg/gate/result.go:114-137`) has a `SchemaVersion` field hardcoded to
   `"gate/v1"` but no `git_sha` or `generated_at` — a corpus-parsing consumer
   snapshotting gate output over time cannot tell whether a given JSON blob
   and the corpus it was parsed against are from the same commit, which
   matters more once per-violation fields (like `bundle_maturity`) are
   denormalized rather than re-looked-up.

## Solution

Add an additive `trace` object to every `requirement_traceability` violation
and top-level provenance to `GateResult`, per the agreed design:

**1. `trace` object, unconditional fields** (every violation, both steps):
`bundle`, `bundle_maturity` (denormalized deliberately — see below),
`req_id`, `req_version`, `gap_kind`, `remedy`.

**2. `gap_kind` enum, one-to-one with the BUNDLE-014 design decisions that
produce each violation shape:**

| `gap_kind` | Design decision | Detection | `remedy` |
|---|---|---|---|
| `uncovered` | DD-2 | REQ never had implemented-spec support | `author_spec` |
| `coverage_lapsed` | DD-3 | REQ's only support is a replaced/retired-terminal citer | `restate_supports` |
| `citing_spec_not_implemented` | DD-1 | delivered bundle cited by non-implemented spec | `implement_spec` |
| `stale_pin` | DD-12 | supports ref pins an older major/minor | lifecycle-keyed: `re_pin` / `new_spec` |
| `chain_outrun` | DD-11 family | downstream artifact's upstream chain doesn't verify | `resolve_upstream` |

`coverage_lapsed` is new detection work, not purely a relabel: today, the
retired-terminal-citer case and the never-covered case both fall through to
the identical generic message at line 145-150. This issue requires telling
them apart — a retired-terminal citer existing for the REQ is the detectable
signal — because the fix is categorically different (re-pin paperwork vs.
spec authoring).

**3. Conditional fields:**
- `stale_pin` adds `pinned_version` + `bump` (`StalePinVerdict`'s existing
  output), and its `remedy` is lifecycle-keyed per DD-12: `re_pin` when the
  bundle is unimplemented, `new_spec` when the bundle is `delivered`.
- `stale_pin` / `citing_spec_not_implemented` add `citing_artifact` (the
  spec/issue ID whose ref produced the violation).
- `chain_outrun` adds `via` — the broken upstream link (e.g. the spec ID a
  stale plan targets). The pointer is the actionable content: it tells the
  operator which artifact to fix first, not just which one alarmed.

**4. Gate-JSON provenance:** `GateResult` gains top-level `git_sha` and
`generated_at`. This is CR-1 from the consumer review, and it is what makes
keeping `bundle_maturity` denormalized on every violation (rather than
requiring a second, time-skewed corpus lookup) a safe simplification instead
of a latent skew bug.

**5. Vocabulary (CR-3):** every doc, field description, and message this issue
touches says "covered" (proven by an implemented spec) for REQ-level state —
never "delivered", which names bundle maturity exclusively.

**6. Forward-compat contract:** additive under `gate/v1`. Consumers seeing an
unrecognized `gap_kind`/`remedy` must degrade to the prose `message`, never
error. A vocabulary-breaking change requires a `schema_version` bump.

**7. Stability:** `identity`/`identity_hash`/`region_hash` computation is
untouched — `trace` is excluded from identity the same way `Line`,
`ProjectWide`, and `WaiverHint` already are (`pkg/gate/result.go`). Existing
baselines and waivers remain valid with zero forced re-baselining.

### Out of scope

Proof attribution — which spec/tests prove a covered REQ, or issue lineage
leading to it — is explicitly deferred to the future `backstop trace` command
(ISSUE-060, backlogged separately). This issue is only about the shape of
data the existing `requirement_traceability` step already computes.

## Verification

Unit-level, in `pkg/gate` and `cmd/backstop`: exercise
`ClassifyRequirementTraceability` directly against constructed
`ArtifactStatusRecord`/`TraceRef` fixtures for `trace` field population per
gap kind, plus a `cmd/backstop` CLI-level test asserting `git_sha`/
`generated_at` appear in real `gate --json` output.

```
go test ./pkg/gate/... ./cmd/backstop/... -run Trace
```

### Mandated tests

- `TestTrace_Fields_EveryViolationCarriesTraceObject` — every violation from
  `ClassifyRequirementTraceability` across a fixture corpus covering all five
  gap kinds has non-nil `trace`.
- `TestTrace_Fields_AdvisoryStepAlsoCarriesTraceObject` — the advisory
  (warning-severity, non-delivered-bundle) branch also populates `trace`, not
  just the block branch.
- `TestTrace_Fields_UncoveredCarriesBundleReqAndVersion` — `trace.bundle`,
  `trace.req_id`, `trace.req_version` match the bundle fixture exactly.
- `TestTrace_Fields_BundleMaturityMatchesRecordAtClassification` —
  `trace.bundle_maturity` reflects the bundle's status at classification time.
- `TestTrace_GapKind_UncoveredMapsToAuthorSpec`
- `TestTrace_GapKind_CoverageLapsedDistinctFromUncovered` — a REQ whose only
  citer is a `replaced` spec produces `gap_kind: coverage_lapsed` /
  `remedy: restate_supports`, NOT `gap_kind: uncovered`.
- `TestTrace_GapKind_CitingSpecNotImplementedMapsToImplementSpec`
- `TestTrace_GapKind_ChainOutrunMapsToResolveUpstream`
- `TestTrace_StalePin_RemedyRePinWhenBundleUnimplemented`
- `TestTrace_StalePin_RemedyNewSpecWhenBundleDelivered`
- `TestTrace_StalePin_PinnedVersionAndBumpPopulated`
- `TestTrace_CitingArtifact_PresentOnStalePinAndCitingSpecNotImplemented`
- `TestTrace_CitingArtifact_AbsentOnUncoveredAndChainOutrun`
- `TestTrace_ChainOutrun_ViaPointsAtNonVerifyingSpec` — `via` names the
  specific non-verifying spec ID, distinct from the violating plan's own ID.
- `TestGateCLI_JSONOutputIncludesGitSHAAndGeneratedAt`
- `TestGateCLI_GitSHAMatchesRepoHEAD`
- `TestTrace_ForwardCompat_UnknownGapKindDegradesToMessage`
- `TestGateCLI_SchemaVersionUnchangedByTraceFields`
- `TestTrace_Identity_UnchangedByTraceFieldAddition` — `identity`/
  `identity_hash`/`region_hash` are byte-identical for a fixed violation
  fixture before and after this change.
- `TestTrace_Baseline_PreExistingBaselineStillCompares` — a pre-change
  baseline artifact still `CompareBaseline`s correctly post-change.
- `TestTrace_Vocabulary_NoDeliveredUsedForCoverageState`

## References

- **BUNDLE-014** (Requirement Traceability) — DD-1 (`citing_spec_not_implemented`),
  DD-2 (`uncovered`), DD-3 (`coverage_lapsed`), DD-11 family (`chain_outrun`),
  DD-12 (`stale_pin` lifecycle-keyed remedy) are the design decisions this
  issue's `gap_kind`/`remedy` enum encodes one-to-one.
- **REQ-006** (`requirement-traceability:REQ-006@1.0.0`) — the bundle
  requirement establishing the `requirement_traceability` gate step as the
  enforcement surface this issue adds structured fields to.
- `pkg/gate/requirement_traceability.go:50-155`
  (`ClassifyRequirementTraceability`) — the five violation-emission sites this
  issue instruments with `trace`.
- `pkg/gate/result.go:64-99` (`Violation`), `pkg/gate/result.go:114-137`
  (`GateResult`) — the structs this issue extends additively.
- **ISSUE-060** (backstop trace command) — proof attribution (which spec/tests
  prove a covered REQ) is explicitly out of scope here and belongs there.
- CLAUDE.md "Loud ≠ blocking" — this is additive instrumentation on an
  existing block/warn step, not a new enforcement surface; severity and gating
  behavior are unchanged.
