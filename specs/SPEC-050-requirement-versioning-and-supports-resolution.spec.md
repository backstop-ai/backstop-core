---
title: "Requirement Versioning And Supports Resolution"
number: SPEC-050
created: "2026-07-14"
status: implemented
schema_version: spec/v1
spec_version: 1.2.1

implementation:
  summary: >
    Seed 1 of BUNDLE-014 (requirement-traceability): the foundational versioning
    schema + both-direction supports resolution that welds the bundle REQ -> spec
    requirement hop. Four mechanically-separable pieces, all inside core's own
    artifact-validation domain (zero language/tool knowledge): (1) each bundle REQ
    in `requirements[]` carries a semver `version:` AND a version LOG that retains
    every version of that REQ with that version's full text, validated for
    well-formedness in `validateBundleRequirements` (pkg/validate/bundle.go);
    (2) the `supports` ref format is tightened to REQUIRE a mandatory exact-semver
    pin `bundle-name:REQ-NNN@MAJOR.MINOR.PATCH` on spec AND issue requirements
    (pkg/validate/spec.go, pkg/validate/issue.go); (3) a NEW corpus-wide resolution
    pass — built as pure pkg/validate functions and wired into `ValidateArtifacts`
    (cmd/backstop/artifact_validate.go) so `backstop artifact validate` (CLI) and
    the gate's `realArtifactValidator.ValidateAll` (cmd/backstop/gate.go) agree —
    resolves every ref both directions (named bundle exists, REQ id is declared)
    and checks the pinned version against a real entry in that REQ's version log;
    (4) `requirements[]` is required at `delivered` maturity, closing the empty-top
    hole while keeping replaced/canceled/deprecated terminal-exempt. Covers
    BUNDLE-014 REQ-001..REQ-005 ONLY; the corpus backfill (Seed 2/SPEC-051) and the
    coverage gate step (Seed 3/SPEC-052) are out of scope. The mandatory-pin
    enforcement flip must ship in the same change-train as SPEC-051's backfill or
    the unversioned/unpinned corpus goes red (see Sharp Edges). Implementation must
    migrate the existing tests that assert the OLD unpinned format — see the
    "Coupled existing tests" implementation note.
  subject: pkg/validate

verification:
  level: integration
  test_command: go test ./pkg/validate/... ./cmd/backstop/... -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      `backstop artifact validate` must RESOLVE every `supports` ref on a spec or
      issue requirement in BOTH directions: the named bundle must exist in the
      corpus AND the cited REQ id must be declared in that bundle's
      `requirements[]` array. A ref naming a missing bundle, or a REQ the bundle
      never declared, is an error-severity defect and blocks. Resolution is
      INDEPENDENT of the citing artifact's status — a `draft` spec citing a bundle
      REQ is normal pipeline flow, and the ref must still resolve. Resolution
      applies at any LIVE citing status (draft, ready-for-implementation,
      implemented); TERMINAL/retired citers (replaced, canceled, deprecated,
      obsoleted) are EXEMPT — the harvest must skip them, mirroring the per-artifact
      terminal exemption already in pkg/validate/spec.go:175, pkg/validate/issue.go:349,
      and `isTerminalStatus`, so a retired spec's stale ref is not resolution-checked
      (this is what lets SPEC-051 clear the SPEC-002/003/004 dangling refs by
      deprecating those drafts, and keeps the `replaced` SPEC-039 out of resolution).
      Format-shape (`supportsRe`) is NOT resolution. The resolution pass must run inside the
      shared `ValidateArtifacts` corpus walk so that the CLI and the gate's
      `ValidateAll` surface the identical verdict, AND the resolution catalog must
      be built from the FULL artifact corpus regardless of any type-scoping filter
      (`--spec`, `--issue`, etc.): because `ValidateArtifacts` passes the type
      filter to discovery, a scoped run that built the catalog from the filtered
      set alone would see zero bundles and falsely red every ref as "missing
      bundle." The bundles for the catalog must therefore be loaded independently
      of the per-artifact validation scope, so a `--spec`-scoped run resolves
      identically to an unscoped run and to the gate.
    supports: requirement-traceability:REQ-001@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-002
    text: >
      Every `supports` ref must carry a MANDATORY exact-semver version pin of the
      form `bundle-name:REQ-NNN@MAJOR.MINOR.PATCH`. The `supportsRe` format check
      (pkg/validate/spec.go, shared by pkg/validate/issue.go) must be tightened to
      require the `@X.Y.Z` segment. Pinning is mandatory on every ref — no optional
      pinning, no default-latest — so an unpinned ref (`bundle-name:REQ-NNN`) and a
      ref with a non-semver pin (e.g. `@1.0` or `@v1.0.0`) are each an
      error-severity defect. This applies identically to spec requirements and
      issue requirements. The pin's semver notion is the same single strict
      MAJOR.MINOR.PATCH used by REQ-004's version-field validation — three
      dot-separated integers, no prerelease or build metadata.
    supports: requirement-traceability:REQ-002@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-003
    text: >
      Resolution (REQ-001) must additionally require the ref's pinned version to
      match a REAL entry in the cited REQ's version LOG. A pin to a version that
      never existed in that REQ's log is a validate-time defect and blocks, so a
      fabricated version dies at `artifact validate` rather than surviving to the
      gate. A pin to an OLDER version that IS present in a multi-entry log resolves
      clean (historical pins stay resolvable).
    supports: requirement-traceability:REQ-003@1.0.0
    follows: STD-GO-001:GO-011
  - id: REQ-004
    text: >
      Each bundle REQ in `requirements[]` must carry a per-REQ semver `version:`
      (MAJOR.MINOR.PATCH) AND a version LOG that retains EVERY version of that REQ
      together with that version's text. The log is expressed as an OPTIONAL
      `versions:` list of `{version, text}` entries; when `versions:` is ABSENT the
      REQ's effective log is the single implicit entry `{version, text}` taken from
      the REQ's own `version:` and `text:` (backward-compatible with a REQ that has
      only ever existed at one version). Well-formedness is validated: `version:`
      must be well-formed semver; when `versions:` is PRESENT it must be non-empty
      (an explicit but empty list is an error), every entry's `version` must be
      well-formed semver and `text` non-empty, the entries must be strictly
      monotonically ascending by semver with no duplicate versions, and the REQ's
      top-level `version:` AND `text:` must each equal the newest (highest, last)
      log entry's `version` and `text`. The semver notion is the same single strict
      MAJOR.MINOR.PATCH used by REQ-002's pin (no prerelease/build metadata). This
      makes the differences between versions recoverable from the artifact itself,
      not git archaeology.
    supports: requirement-traceability:REQ-004@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      `requirements[]` must be required from `ready` maturity onward INCLUDING
      `delivered`: `validateBundleRequirements` (pkg/validate/bundle.go) must treat
      a `delivered` bundle the same as `defined`/`ready` and error when the array is
      absent or empty. It remains required at `defined`/`ready` exactly as today —
      this only EXTENDS the requirement, never relaxes it. The retirement-terminal
      statuses `replaced`, `canceled`, and `deprecated` stay exempt (they are
      skipped by the terminal-state exemption at pkg/validate/bundle.go and are not
      brought into scope by this change).
    supports: requirement-traceability:REQ-005@1.0.0
    follows: STD-GO-001:GO-010

claims:
  # REQ-001 — both-direction resolution, status-independent, full-corpus, shared pass
  - id: CLM-001
    requirement: REQ-001
    text: A supports ref naming an existing bundle and a REQ declared in that bundle's requirements[] resolves clean
    tests:
      - TestResolveSupports_DeclaredReqResolves
  - id: CLM-002
    requirement: REQ-001
    text: A supports ref naming a bundle that does not exist in the corpus is an error
    tests:
      - TestResolveSupports_MissingBundleErrors
  - id: CLM-003
    requirement: REQ-001
    text: A supports ref to an existing bundle but a REQ id it never declared is an error
    tests:
      - TestResolveSupports_UndeclaredReqErrors
  - id: CLM-004
    requirement: REQ-001
    text: Resolution fires regardless of the citing artifact's LIVE status — a draft spec citing a real REQ resolves clean (live-status-independence)
    tests:
      - TestResolveSupports_DraftCitingStatusIndependent
  - id: CLM-005
    requirement: REQ-001
    text: An issue requirement's supports ref is resolution-checked the same way — a missing bundle is an error
    tests:
      - TestResolveSupports_IssueRefMissingBundleErrors
  - id: CLM-006
    requirement: REQ-001
    subject: cmd/backstop
    text: The corpus resolution pass runs inside ValidateArtifacts, so backstop artifact validate and the gate's ValidateAll surface the identical resolution violation
    tests:
      - TestValidateArtifacts_ResolutionPassSharedByCLIAndGate
  - id: CLM-031
    requirement: REQ-001
    subject: cmd/backstop
    text: A type-scoped run (e.g. --spec) still builds the resolution catalog from the FULL corpus, so a real ref resolves clean instead of false-redding as a missing bundle
    tests:
      - TestResolveSupports_TypeScopedRunUsesFullCorpusCatalog
  - id: CLM-034
    requirement: REQ-001
    text: BuildBundleReqCatalog gracefully skips a cited bundle whose requirements[] is absent/non-list/malformed — resolution surfaces the ref violation without panicking
    tests:
      - TestBuildBundleReqCatalog_MalformedBundleSkippedGracefully
  - id: CLM-035
    requirement: REQ-001
    text: A deprecated (terminal) spec's dangling supports ref is NOT resolution-checked — the retired spec resolves clean (terminal exemption mirrors spec.go:175)
    tests:
      - TestResolveSupports_DeprecatedCiterExempt
  - id: CLM-036
    requirement: REQ-001
    text: A replaced (terminal) spec's supports ref is NOT resolution-checked (keeps SPEC-039 out of resolution)
    tests:
      - TestResolveSupports_ReplacedCiterExempt

  # REQ-002 — mandatory exact-semver pin on spec AND issue refs
  - id: CLM-007
    requirement: REQ-002
    text: A pinned well-formed spec supports ref bundle-name:REQ-001@1.0.0 passes the format check
    tests:
      - TestSpecSupportsFormat_PinnedRefValid
  - id: CLM-008
    requirement: REQ-002
    text: An unpinned spec supports ref bundle-name:REQ-001 is an error (pin mandatory)
    tests:
      - TestSpecSupportsFormat_UnpinnedRefErrors
  - id: CLM-009
    requirement: REQ-002
    text: A spec supports ref with a non-semver pin (e.g. @1.0) is an error
    tests:
      - TestSpecSupportsFormat_NonSemverPinErrors
  - id: CLM-010
    requirement: REQ-002
    text: A pinned well-formed issue supports ref passes the format check
    tests:
      - TestIssueSupportsFormat_PinnedRefValid
  - id: CLM-011
    requirement: REQ-002
    text: An unpinned issue supports ref is an error (pin mandatory, applies to issues too)
    tests:
      - TestIssueSupportsFormat_UnpinnedRefErrors

  # REQ-003 — pinned version must match a real version-log entry
  - id: CLM-012
    requirement: REQ-003
    text: A pin matching a real entry in the cited REQ's version log resolves clean
    tests:
      - TestResolveSupports_PinMatchesLogEntry
  - id: CLM-013
    requirement: REQ-003
    text: A pin to a version absent from the cited REQ's log (fabricated) is an error
    tests:
      - TestResolveSupports_FabricatedVersionErrors
  - id: CLM-014
    requirement: REQ-003
    text: A pin to an older version present in a multi-entry log resolves clean (historical pins stay resolvable)
    tests:
      - TestResolveSupports_OlderLoggedVersionResolves

  # REQ-004 — per-REQ version + version-log well-formedness (shape matrix)
  - id: CLM-015
    requirement: REQ-004
    text: A REQ with a well-formed version and no explicit versions list (implicit single-entry log) is valid
    tests:
      - TestBundleReqVersion_ImplicitSingleEntryValid
  - id: CLM-016
    requirement: REQ-004
    text: A REQ missing its version field is an error
    tests:
      - TestBundleReqVersion_MissingVersionErrors
  - id: CLM-017
    requirement: REQ-004
    text: A REQ with a malformed (non-semver) version is an error
    tests:
      - TestBundleReqVersion_MalformedVersionErrors
  - id: CLM-018
    requirement: REQ-004
    text: A REQ with a well-formed explicit versions log (ascending, unique, current == newest) is valid
    tests:
      - TestBundleReqVersionLog_WellFormedValid
  - id: CLM-019
    requirement: REQ-004
    text: A versions-log entry with a non-semver version is an error
    tests:
      - TestBundleReqVersionLog_NonSemverEntryErrors
  - id: CLM-020
    requirement: REQ-004
    text: A versions log not strictly monotonically ascending is an error
    tests:
      - TestBundleReqVersionLog_NonMonotonicErrors
  - id: CLM-021
    requirement: REQ-004
    text: A versions log with a duplicate version is an error
    tests:
      - TestBundleReqVersionLog_DuplicateVersionErrors
  - id: CLM-022
    requirement: REQ-004
    text: A top-level version not matching the newest versions-log entry is an error
    tests:
      - TestBundleReqVersionLog_CurrentNotNewestErrors
  - id: CLM-023
    requirement: REQ-004
    text: A versions-log entry with empty text is an error
    tests:
      - TestBundleReqVersionLog_EmptyEntryTextErrors
  - id: CLM-032
    requirement: REQ-004
    text: An explicit but EMPTY versions list is an error (a present log must have at least one entry)
    tests:
      - TestBundleReqVersionLog_EmptyVersionsListErrors
  - id: CLM-033
    requirement: REQ-004
    text: A top-level text not equal to the newest versions-log entry's text is an error
    tests:
      - TestBundleReqVersionLog_CurrentTextNotNewestErrors

  # REQ-005 — requirements[] required at delivered; retirement-terminal stay exempt (maturity matrix)
  - id: CLM-024
    requirement: REQ-005
    text: A delivered bundle with no requirements[] array is an error (the empty-top hole closes)
    tests:
      - TestBundleRequirements_DeliveredMissingErrors
  - id: CLM-025
    requirement: REQ-005
    text: A delivered bundle with a valid non-empty requirements[] validates clean
    tests:
      - TestBundleRequirements_DeliveredPresentValid
  - id: CLM-026
    requirement: REQ-005
    text: A defined bundle with no requirements[] is still an error (unchanged, regression guard)
    tests:
      - TestBundleRequirements_DefinedMissingStillErrors
  - id: CLM-027
    requirement: REQ-005
    text: A ready bundle with no requirements[] is still an error (unchanged, regression guard)
    tests:
      - TestBundleRequirements_ReadyMissingStillErrors
  - id: CLM-028
    requirement: REQ-005
    text: A replaced bundle with no requirements[] stays exempt (no error — terminal exemption preserved)
    tests:
      - TestBundleRequirements_ReplacedStaysExempt
  - id: CLM-029
    requirement: REQ-005
    text: A canceled bundle with no requirements[] stays exempt (no error)
    tests:
      - TestBundleRequirements_CanceledStaysExempt
  - id: CLM-030
    requirement: REQ-005
    text: A deprecated bundle with no requirements[] stays exempt (no error)
    tests:
      - TestBundleRequirements_DeprecatedStaysExempt

contracts:
  - file: pkg/validate/supports_resolution.go
    provides:
      - name: SupportRef
        kind: type
        signature: "type SupportRef struct { Raw string; BundleName string; ReqID string; Version string; Pinned bool; File string; Label string }"
        notes: "One parsed supports pin harvested from a citing spec/issue requirement; Version is the @X.Y.Z pin, Pinned false when the pin is absent (REQ-002)."
      - name: BundleReqCatalog
        kind: type
        signature: "type BundleReqCatalog struct { /* bundle name -> REQ id -> ordered set of logged versions */ }"
        notes: "Corpus index of every bundle REQ's effective version log, built from parsed bundles; the resolution target for REQ-001/REQ-003."
      - name: BuildBundleReqCatalog
        kind: function
        signature: "func BuildBundleReqCatalog(bundles []*artifact.ParsedArtifact) *BundleReqCatalog"
        notes: "Extracts each bundle REQ's effective version log (implicit single-entry when versions: absent, else the explicit list) into the catalog. Built from the FULL bundle corpus regardless of any type-scoping filter (REQ-001). GRACEFULLY skips a bundle whose requirements[] is absent, non-list, or malformed — that bundle's own shape violations are reported by validateBundleRequirements; the builder never panics (CLM-034). Well-formedness is NOT re-checked here — it lives in validateBundleRequirements; the builder reads what is present."
      - name: CollectSupportRefs
        kind: function
        signature: "func CollectSupportRefs(citing []*artifact.ParsedArtifact) []SupportRef"
        notes: "Harvests every supports ref from the requirements[] of the given spec/issue artifacts, parsing bundle name, REQ id, and pin into SupportRef. SKIPS terminal-status citers (replaced/canceled/deprecated/obsoleted) so a retired artifact's stale ref is not resolution-checked — mirrors the per-artifact terminal exemption at spec.go:175 / issue.go:349 (CLM-035/CLM-036)."
      - name: ResolveSupports
        kind: function
        signature: "func ResolveSupports(catalog *BundleReqCatalog, refs []SupportRef) []Violation"
        notes: "Both-direction resolution + version-log match: emits error-severity Violations for a missing bundle (REQ-001), an undeclared REQ (REQ-001), and a pin absent from the REQ's log (REQ-003), at ANY citing status."
    consumes:
      - source: pkg/artifact
        name: ParsedArtifact
        kind: type
  - file: cmd/backstop/artifact_validate.go
    provides:
      - name: ValidateArtifacts
        kind: function
        signature: "func ValidateArtifacts(cfg ValidateConfig) (ValidateResult, error)"
        notes: "Unchanged signature; now ALSO runs the corpus resolution pass (BuildBundleReqCatalog + CollectSupportRefs + ResolveSupports) after the per-artifact loop and appends its violations. The catalog is built from a FULL-corpus bundle discovery independent of cfg's type filter (REQ-001), so a --spec-scoped run does not false-red. Because the gate's realArtifactValidator.ValidateAll delegates here, the CLI and gate share one verdict."
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
---

# SPEC-050: Requirement Versioning And Supports Resolution

## Overview

Backstop's central promise is that a green gate on an `implemented` spec is
mechanical proof of every requirement in that spec. The one hop that promise does
NOT yet cover is the topmost one: bundle REQ -> spec requirement, carried by the
`supports:` field on a spec or issue requirement. Today that field is
regex-FORMAT-checked only (`supportsRe`, pkg/validate/spec.go) and never RESOLVED —
a spec can cite `nonexistent-bundle:REQ-999` and validate green — and bundle REQs
carry no version, so nothing records WHICH understanding of a REQ a spec pinned.

This spec is Seed 1 of BUNDLE-014 (requirement-traceability): the foundational
mechanism the coverage gate step (Seed 3) later builds on. It does four things,
all inside core's own artifact-validation domain — no language or tool knowledge
enters:

1. **Per-REQ version + version log (REQ-004).** Each bundle REQ carries a semver
   `version:` and a version LOG retaining every version with its text, so
   "what did @1.1.0 say versus @2.0.0" is answerable from the artifact.
2. **Mandatory exact pin on supports refs (REQ-002).** The `supports` format is
   tightened to require `bundle-name:REQ-NNN@MAJOR.MINOR.PATCH` on spec AND issue
   requirements; an unpinned or malformed-pin ref is a defect.
3. **Both-direction resolution + log match (REQ-001, REQ-003).** A new corpus-wide
   pass resolves every ref — the named bundle exists, the REQ id is declared, and
   the pinned version matches a real log entry — at any citing status.
4. **`requirements[]` required at `delivered` (REQ-005).** The top of the chain can
   no longer be structurally empty at the moment a bundle claims success.

This spec covers BUNDLE-014 REQ-001 through REQ-005 ONLY. The one-time corpus
reconciliation and `1.0.0` backfill (Seed 2, SPEC-051) and the
`requirement_traceability` coverage gate step + stale-pin model (Seed 3, SPEC-052)
are explicitly out of scope.

## Requirements

Requirements REQ-001 through REQ-005 are defined in frontmatter and trace to
BUNDLE-014 REQ-001..REQ-005 via `supports`. Each requirement has at least one
claim; claims are defined in frontmatter. (This spec's own `supports` refs are
written UNPINNED because the mandatory-pin enforcement they introduce is not yet
live; SPEC-051's backfill stamps `@1.0.0` onto them when it flips on — see Sharp
Edges, "self-referential corpus.")

### Version-log schema (REQ-004)

Each entry in a bundle's `requirements[]` array carries:

| Field | Required | Constraint |
|-------|----------|------------|
| `version` | yes | Well-formed semver `MAJOR.MINOR.PATCH`; equals the newest log entry's version |
| `text` | yes | Equals the newest log entry's text |
| `versions` | no | Ordered list of `{version, text}` entries; when absent, the effective log is the single implicit entry `{version, text}`; when present, must be non-empty |

The **effective version log** is the explicit `versions:` list when present, else
the single implicit entry built from the REQ's own `version:` + `text:`. This keeps
a REQ that has only ever existed at one version (the shape BUNDLE-014 already uses)
backward-compatible while satisfying "retain every version."

Well-formedness of an explicit `versions:` list:

| Rule | Violation if broken |
|------|---------------------|
| Non-empty (an explicit empty list is an error) | error |
| Every entry `version` well-formed semver | error |
| Every entry `text` non-empty | error |
| Strictly monotonically ascending by semver | error |
| No duplicate versions | error |
| Top-level `version:` equals the newest (last) entry's `version` | error |
| Top-level `text:` equals the newest entry's `text` | error |

### Supports pin format (REQ-002)

The `supports` ref format tightens from `bundle-name:REQ-NNN` to
`bundle-name:REQ-NNN@MAJOR.MINOR.PATCH`. The pin is mandatory — there is no
optional pinning and no default-latest.

| Ref form | Valid? | Applies to |
|----------|--------|-----------|
| `bundle-name:REQ-001@1.0.0` | yes | spec + issue |
| `bundle-name:REQ-001` (unpinned) | no — error | spec + issue |
| `bundle-name:REQ-001@1.0` (non-semver pin) | no — error | spec + issue |
| `bundle-name:REQ-001@v1.0.0` (non-semver pin) | no — error | spec + issue |

### Resolution (REQ-001, REQ-003)

A ref resolves iff ALL THREE hold; any failure is an error-severity block, at ANY
citing artifact status:

| Check | REQ | Fails when |
|-------|-----|-----------|
| Named bundle exists | REQ-001 | ref names a bundle absent from the corpus |
| REQ id declared in that bundle's `requirements[]` | REQ-001 | ref cites a REQ the bundle never declared |
| Pinned version present in that REQ's effective version log | REQ-003 | pin names a version that never existed in the log |

### requirements[] gate by maturity (REQ-005)

| Maturity | `requirements[]` required? | Change |
|----------|----------------------------|--------|
| `defined` | yes | unchanged |
| `ready` | yes | unchanged |
| `delivered` | yes | NEW — the empty-top hole closes |
| `replaced` | no (terminal-exempt) | unchanged |
| `canceled` | no (terminal-exempt) | unchanged |
| `deprecated` | no (terminal-exempt) | unchanged |

## Implementation

### Package layout

- `pkg/validate/bundle.go` — version + version-log well-formedness inside
  `validateBundleRequirements` (REQ-004), and the `delivered` extension of the
  requirements-required gate (REQ-005). The maturity gate is `requiresDefined` at
  bundle.go:531 (inside `validateBundleRequirements`), NOT the terminal exemption
  at bundle.go:94 — `delivered` is not a terminal status, so
  `validateBundleRequirements` already runs for it; the fix is to make its internal
  requirement condition also fire for `delivered`.
- `pkg/validate/spec.go` + `pkg/validate/issue.go` — the tightened `supports`
  format regex requiring the `@X.Y.Z` pin (REQ-002). Both validators share the one
  compiled regex.
- `pkg/validate/supports_resolution.go` (NEW) — the pure corpus-resolution
  functions: `BuildBundleReqCatalog`, `CollectSupportRefs`, `ResolveSupports`
  (REQ-001, REQ-003). No filesystem access, no language knowledge — operates on
  already-parsed artifacts.
- `cmd/backstop/artifact_validate.go` — `ValidateArtifacts` invokes the resolution
  pass over the discovered corpus after the per-artifact loop (REQ-001, REQ-003).

### Semver: one strict M.M.P notion for BOTH the version field and the pin (REQ-002/REQ-004)

REQ-002's pin regex and REQ-004's `version:`/log-entry validation MUST share ONE
strict MAJOR.MINOR.PATCH semver notion: three dot-separated integers only, matching
`^\d+\.\d+\.\d+$` — the same strict shape as the existing `semverRe`
(pkg/validate/bundle.go:14, already used for `bundle.version`) and the `\d+\.\d+\.\d+`
tail of the pin regex. The version field, the log entries, and the pin all key off
that one strict form so they cannot drift apart.

DO NOT reuse `pkg/pack/manifest.go`'s `validateSemver` for the version-field check:
its pattern PERMITS `-prerelease`/`+build` suffixes, so it would ACCEPT `1.0.0-rc1`
as a version-field value while the pin regex rejects `@1.0.0-rc1` — producing a
version no pin could ever match and breaking the one-strict-M.M.P seam SPEC-052's
stale-pin comparison relies on. The strict `semverRe` shape (or a shared strict
regex) is the single source of truth here; the version field and the pin validate
against it, not against the looser manifest helper.

### Validation passes (the mechanical steps a planner maps tasks to)

1. **Version-log well-formedness (REQ-004).** In `validateBundleRequirements`, for
   each REQ: check `version:` is present and well-formed semver; if a `versions:`
   list is present, check non-empty (an explicit empty list is an error), each
   entry semver + non-empty text, strictly ascending, no duplicates, and top-level
   `version:` AND `text:` equal to the newest entry. Absent `versions:` is valid
   (implicit single-entry log).
2. **Pin-format tightening (REQ-002).** Replace the `supportsRe` pattern
   (`^[a-z0-9-]+:REQ-\d{3}$`) with one that requires the pin
   (`^[a-z0-9-]+:REQ-\d{3}@\d+\.\d+\.\d+$`). Both `spec.go` and `issue.go` use it;
   an unpinned or malformed-pin ref emits the existing supports-format violation.
3. **Catalog build (REQ-001, REQ-003).** `BuildBundleReqCatalog` walks the parsed
   bundles and records, per bundle name, per REQ id, the set of versions in that
   REQ's effective log. It gracefully skips any bundle whose `requirements[]` is
   absent, non-list, or malformed (that bundle's shape errors are reported by the
   per-artifact bundle validator) — it never panics.
4. **Ref harvest (REQ-001).** `CollectSupportRefs` walks the parsed spec + issue
   artifacts and parses every `supports` ref into a `SupportRef` (bundle name, REQ
   id, pin).
5. **Resolution (REQ-001, REQ-003).** `ResolveSupports` checks each ref against the
   catalog: missing bundle -> error; undeclared REQ -> error; pin absent from the
   REQ's version set -> error. Emits error-severity violations; passes clean at any
   citing status.
6. **Corpus wiring (REQ-001).** `ValidateArtifacts` builds the resolution catalog
   from a FULL-corpus bundle set independent of `cfg`'s type filter, harvests the
   refs from the discovered spec/issue artifacts, runs `ResolveSupports`, and
   appends the violations. Because the gate's `realArtifactValidator.ValidateAll`
   (cmd/backstop/gate.go) delegates to `ValidateArtifacts`, the CLI and the gate
   share one resolution verdict — no second copy of the pass.

### Full-corpus catalog under type-scoping (REQ-001)

`ValidateArtifacts` (cmd/backstop/artifact_validate.go:112) passes the type filter
to `DiscoverArtifacts`, so a `--spec`-scoped run discovers ONLY specs. If the
catalog were built from that filtered set it would contain zero bundles and
`ResolveSupports` would red every ref as "missing bundle." The resolution pass must
therefore load the bundles for the catalog from a SEPARATE, unfiltered discovery
(or discover all types once and filter only for the per-artifact validation loop),
so a scoped run resolves identically to an unscoped run and to the gate. CLM-031
guards this exact regression.

### Coupled existing tests to migrate (REQ-002)

Tightening `supportsRe` to require the pin breaks tests and fixtures that assert
the OLD unpinned format. These are IN SCOPE for the implementer and must be
migrated to the pinned `@1.0.0` form as part of this work — do NOT weaken the regex
to make them pass:

- `pkg/validate/spec_test.go:988` `TestSpec_RequirementValidSupports` asserts the
  unpinned `my-feature:REQ-001` produces no supports-format violation. Migrate the
  fixture value to `my-feature:REQ-001@1.0.0` and keep the assertion.
- `pkg/validate/issue_test.go:23` `validIssueArtifact()` — the shared issue fixture
  carries an unpinned `my-feature:REQ-003` (issue_test.go:78) that ~20+ tests ride
  through the `Pass()` assertion at issue_test.go:127. Migrate the fixture ref to
  `my-feature:REQ-003@1.0.0` so the shared fixture stays green.

### Where resolution lives, and why

`ValidateArtifacts` is the single corpus walk that both `backstop artifact
validate` and the gate's `ValidateAll` already flow through. Placing the
resolution pass there (rather than in a per-artifact validator, which sees only one
file) is what lets a ref be resolved against a bundle in a DIFFERENT file, and is
what keeps the CLI and gate verdicts identical. Format-shape (REQ-002) stays
per-artifact because it needs no corpus view; resolution (REQ-001/REQ-003) is
corpus-wide by necessity.

## Verification

Verification is defined in frontmatter: integration level, 80% coverage threshold,
targeting `pkg/validate` and `cmd/backstop`. Integration level is chosen because
the load-bearing behavior is cross-package wiring — the pure resolution functions
in `pkg/validate` are inert until `ValidateArtifacts` in `cmd/backstop` builds the
catalog and runs them over the discovered corpus, and the requirement is precisely
that the CLI and the gate share that one wired pass, including under type-scoping
(REQ-001). A unit-only verification of `pkg/validate` would prove the functions
correct while leaving the wiring — the exact place this could ship dark or
false-red — unproven. Claims are defined in frontmatter; every requirement has at
least one claim, and the REQ-002, REQ-004, and REQ-005 matrices are covered cell by
cell.

## Sharp Edges

- **Enforcement-flip ordering — do not turn the pin mandate on before the backfill
  lands.** The moment `supportsRe` requires the `@X.Y.Z` pin and the resolution
  pass runs error-severity, the ENTIRE existing corpus (every spec citing an
  unpinned `bundle:REQ-NNN`, every bundle REQ with no version FIELD) goes red. This
  spec's tightening MUST ship in the same change-train as SPEC-051's corpus backfill
  (Seed 2), which stamps `version: 1.0.0` and `@1.0.0` pins across the corpus, so
  validation tightens and the corpus turns green in one move. The END state is
  unconditionally on — there is NO permanent config toggle and no grandfather path;
  the ordering is a merge-sequencing constraint, not a runtime switch. This spec's
  own unit tests operate on fixtures, so they pass green in isolation regardless of
  the live corpus; the live-corpus green depends on Seed 2 landing together.

- **Type-scoped validation must not starve the catalog.** `ValidateArtifacts` passes
  the type filter to discovery, so a naive catalog built from the scoped set has zero
  bundles on a `--spec` run and reds every ref. The catalog MUST be built from a
  full-corpus bundle discovery independent of the scope, or `backstop artifact
  validate --spec X` false-fails while the unscoped run and the gate pass — a verdict
  divergence that breaks the very "one verdict" property CLM-006 asserts. CLM-031 is
  the guard.

- **Version identity is only consumed at first pin — never demand log ceremony for
  never-pinned churn.** An unpinned REQ (one no spec cites yet) may evolve freely at
  its current version with no bump obligation; pre-pin evolution is captured by the
  bundle's Version History / ledger, not the version log. The validator must NOT
  demand a multi-entry `versions:` list for a REQ that has churned before its first
  pin — well-formedness of whatever log is present (including the implicit
  single-entry form) is the only obligation. A REQ with just `version:` + `text:`
  and no `versions:` is valid by construction.

- **Self-referential corpus — this spec's own shape and refs are subject to the new
  rules.** BUNDLE-014's own `requirements[]` already carries `version: "1.0.0"` per
  REQ with no `versions:` list; the schema MUST accept that shape (it is the implicit
  single-entry log), which is why absent `versions:` is valid. Symmetrically, this
  spec's own `supports` refs are written UNPINNED (`requirement-traceability:REQ-NNN`)
  because the pinned form would be rejected by the CURRENTLY-LIVE `supportsRe`; once
  SPEC-051 backfills and the flip lands, this spec's refs are themselves rewritten to
  `@1.0.0`. Writing them pinned now would fail `artifact validate` today.

- **Backward-compatibility hinge: absent `versions:` must be a valid implicit log,
  not a missing-log error.** If REQ-004 well-formedness treated an absent `versions:`
  list as "no log -> error," it would red the entire existing corpus (every REQ ever
  written) and contradict the never-pinned-churn edge above. The implicit-single-entry
  rule is the load-bearing compatibility bridge; get it wrong and Seed 2's backfill
  becomes mandatory for every REQ rather than only for refs that need a pin target.
  (Note the asymmetry with an EXPLICIT empty `versions: []`, which IS an error — a
  present log must have at least one entry.)

- **Resolution must be status-independent — a draft citing is normal flow, not a
  defect.** The temptation is to only resolve refs on `implemented` or non-draft
  specs. That is wrong: a dangling or fabricated-version ref is a defect the moment
  it is written, and a `draft` spec citing a real, declared, logged REQ is the normal
  pipeline. `ResolveSupports` keys off the ref's target, never the citing artifact's
  status. (Coverage — the OTHER direction, "is every bundle REQ supported" — is Seed
  3's gate step and is deliberately NOT in this spec.)

- **One pass, two callers — never fork the resolution.** The resolution must live in
  the shared `ValidateArtifacts` so the CLI and the gate's `ValidateAll` inherit it
  identically. A second copy (e.g. a gate-only resolution step) would drift: a ref
  could resolve clean on the CLI and red in the gate, or vice versa. CLM-006 guards
  that the pass is reached through the shared path.

- **A malformed cited bundle must not crash resolution.** A ref can name a bundle
  whose `requirements[]` is absent, a non-list, or otherwise malformed.
  `BuildBundleReqCatalog` must skip that bundle's contribution gracefully (its own
  shape errors are surfaced by `validateBundleRequirements`) so resolution still
  emits the ref's own violation and the run reports BOTH problems without a panic.
  CLM-034 guards the no-panic path.

- **Terminal citers are exempt from resolution (SPEC-051 seam).** The per-artifact
  validators ALREADY skip the whole completeness block — including the
  supports-format check — for terminal specs/issues (pkg/validate/spec.go:175,
  pkg/validate/issue.go:349, `isTerminalStatus`). The corpus resolution pass MUST
  honor the same exemption: harvesting refs from a `replaced`/`canceled`/`deprecated`/
  `obsoleted` citer would be an incoherent format-exempt-but-resolution-checked
  split, and would pin a retired artifact's stale ref red forever. This is the
  load-bearing seam SPEC-051 (Seed 2) depends on: it retires the stale drafts
  SPEC-002/003/004 (draft -> deprecated) to clear their 10 dangling
  `agent-definitions:REQ-004..018` refs, and re-states BUNDLE-011's REQ-007/010 on
  the `implemented` SPEC-040 rather than the `replaced` SPEC-039 — both of which
  only turn the corpus green if resolution skips terminal citers. CLM-035/CLM-036
  guard it.

- **Monotonic-ascending is by SEMVER, not string order.** `versions:` ordering and
  duplicate detection must compare semver components numerically — `1.10.0` is newer
  than `1.9.0`, which string comparison gets wrong. "Newest entry" for the
  current-version and current-text cross-checks is the semver-max, which for a
  well-formed ascending list is the last element.

## Review Questions

1. Does `ResolveSupports` key entirely off the ref's TARGET (bundle existence, REQ
   declaration, version-log membership) and never off the citing artifact's status,
   so a `draft` spec citing a real REQ resolves clean while a fabricated-version ref
   reds regardless of who cites it? (REQ-001/REQ-003.)

2. Is the resolution pass invoked exactly once, inside `ValidateArtifacts`, so the
   CLI (`backstop artifact validate`) and the gate's `realArtifactValidator.ValidateAll`
   surface the identical verdict with no forked second copy? (REQ-001 / CLM-006.)

3. Is the resolution catalog built from a FULL-corpus bundle discovery independent of
   `cfg`'s type filter, so `backstop artifact validate --spec X` resolves a real ref
   clean instead of false-redding it as a missing bundle? (REQ-001 / CLM-031.)

4. Does an ABSENT `versions:` list validate as the implicit single-entry log (valid),
   while an EXPLICIT empty `versions: []` is an error — and does the existing corpus
   plus BUNDLE-014's own `version: "1.0.0"` REQs stay green? (REQ-004 backward-compat
   hinge + CLM-032.)

5. Is `versions:` ordering and duplicate detection done by numeric SEMVER comparison
   (so `1.10.0 > 1.9.0`), and are BOTH the current-`version:`-equals-newest and
   current-`text:`-equals-newest checks made against the semver-max? (REQ-004 /
   CLM-033.)

6. Is the tightened `supportsRe` shared by BOTH `spec.go` and `issue.go`, and does
   the implementer migrate the coupled fixtures (`TestSpec_RequirementValidSupports`
   at spec_test.go:988; `validIssueArtifact()` at issue_test.go:23/78) to the pinned
   form rather than weakening the regex? (REQ-002.)

7. Does `BuildBundleReqCatalog` skip a malformed cited bundle without panicking, so
   resolution still surfaces the ref violation alongside the bundle's own shape
   violations? (REQ-001 / CLM-034.)

8. Does the ref harvest SKIP terminal-status citers (replaced/canceled/deprecated/
   obsoleted), mirroring the per-artifact terminal exemption (spec.go:175 /
   issue.go:349), so a retired spec's stale ref is not resolution-checked — the seam
   SPEC-051 relies on to green the corpus? (REQ-001 / CLM-035 / CLM-036.)

9. Does the `delivered` extension of the requirements-required gate leave the
   `replaced`/`canceled`/`deprecated` terminal exemption untouched (they stay exempt),
   and does it not relax the existing `defined`/`ready` requirement? (REQ-005.)

10. Is the mandatory-pin regex change plus the resolution pass structured so they can
   ship in the SAME change-train as SPEC-051's backfill (no permanent toggle), and do
   this spec's own tests pass on fixtures independent of the live corpus state?
   (Enforcement-flip ordering.)

## References

- BUNDLE-014 (requirement-traceability): source bundle; Seed 1 of 3. Resolved
  decisions RDQ-5/RDQ-6/RDQ-10 (versioning, mandatory pin, log-resolved), DD-4/DD-5/
  DD-10; requirements REQ-001..REQ-005.
- SPEC-051 (Seed 2): the one-time reconciliation + `1.0.0` backfill sweep that must
  land with this spec's enforcement flip.
- SPEC-052 (Seed 3): the `requirement_traceability` coverage gate step + stale-pin
  model that builds on this spec's version log and resolution.
- `pkg/validate/spec.go` (`supportsRe`, requirement-supports-format) and
  `pkg/validate/issue.go` (issue twin): the format check tightened by REQ-002.
- `pkg/validate/bundle.go` (`validateBundleRequirements` at :528, `requiresDefined`
  at :531, `semverRe` at :14): the version-log validation home (REQ-004) and the
  `delivered` extension (REQ-005).
- `pkg/validate/bundle.go` `semverRe` (:14, `^\d+\.\d+\.\d+$`): the strict M.M.P
  shape REQ-002's pin tail and REQ-004's version field share. NOTE: do NOT use
  `pkg/pack/manifest.go`'s `validateSemver` (:571) for the version field — it permits
  `-prerelease`/`+build` suffixes the pin regex rejects, breaking the one-strict-notion
  seam with SPEC-052.
- `cmd/backstop/artifact_validate.go` (`ValidateArtifacts` at :101, type filter to
  discovery at :112) and `cmd/backstop/gate.go` (`realArtifactValidator.ValidateAll`
  at :1442): the shared corpus walk the resolution pass wires into (REQ-001).

## Version History

- **1.0.0 (2026-07-14, draft)** — Initial spec authored from BUNDLE-014 Seed 1:
  version-log schema (REQ-004), mandatory pin (REQ-002), both-direction resolution
  (REQ-001/REQ-003), and `requirements[]`-at-`delivered` (REQ-005). 5 requirements,
  30 claims, 8 sharp edges.
- **1.1.0 (2026-07-14, draft)** — Spec-reviewer FAIL fixes. B1: made full-corpus
  catalog under type-scoping a REQ-001 behavior (not a note) + added CLM-031 and the
  "Full-corpus catalog under type-scoping" implementation section. B2: named the
  coupled existing tests (`TestSpec_RequirementValidSupports` spec_test.go:988;
  `validIssueArtifact()` issue_test.go:23/78) as in-scope fixture migration. S1:
  added CLM-032 (explicit empty `versions:` list = error) and CLM-033 (top-level
  `text:` != newest entry text = error). S2: specified `BuildBundleReqCatalog`
  graceful-skip of malformed cited bundles + CLM-034. Nits: N1 "no version LOG" ->
  "no version FIELD" in Sharp Edge 1; N2 pointed the implementer at the semver helper;
  N3 stated REQ-002 and REQ-004 share one strict M.M.P
  semver notion. Now 5 requirements, 34 claims, 9 sharp edges.
- **1.2.0 (2026-07-14, draft)** — Cross-seam coordination with SPEC-051 (Seed 2).
  Made the corpus resolution pass EXEMPT terminal-status citers
  (replaced/canceled/deprecated/obsoleted) from harvest, mirroring the existing
  per-artifact terminal exemption (pkg/validate/spec.go:175, issue.go:349,
  `isTerminalStatus`) — a consistency fix (format-exempt artifacts must not be
  resolution-checked) that is also the seam letting SPEC-051 retire SPEC-002/003/004
  and keep `replaced` SPEC-039 out of resolution. Refined REQ-001 text + CLM-004
  ("live status"), added CLM-035 (deprecated citer exempt) and CLM-036 (replaced
  citer exempt), a "Terminal citers are exempt" sharp edge, a contract note on
  CollectSupportRefs, and Review Question 8. Now 5 requirements, 36 claims, 10 sharp
  edges.
- **1.2.1 (2026-07-14, draft)** — Consistency-pass fix F3. Corrected the semver-helper
  note: `pkg/pack/manifest.go`'s `validateSemver` PERMITS `-prerelease`/`+build`
  suffixes and is NOT the same notion as the strict `semverRe` (bundle.go:14,
  `^\d+\.\d+\.\d+$`). Reusing it for the version field would accept `1.0.0-rc1` while
  the pin regex rejects `@1.0.0-rc1` — a version no pin could match, breaking the
  one-strict-M.M.P seam SPEC-052's stale-pin comparison depends on. Pointed the version
  field AND the pin at the strict `^\d+\.\d+\.\d+$` shape as the single source of truth;
  corrected the References entry and the 1.1.0 N2 note. No requirement/claim count
  change.
