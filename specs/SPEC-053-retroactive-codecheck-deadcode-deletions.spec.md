---
title: "Retroactive Codecheck Deadcode Deletions"
number: SPEC-053
created: "2026-07-14"
status: implemented
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    RETROACTIVE ARCHIVAL SPEC — it documents work that ALREADY SHIPPED and is verified
    on `main` today; it mandates NO new code and NO new tests. It carries status
    `implemented` because it describes verified reality, not future work. It exists to
    close a TRACEABILITY gap surfaced by review on 2026-07-14: BUNDLE-011 REQ-007 (delete
    the `routeFileDefaults` non-Go semgrep catch-all — backstop must not route files to a
    findings/semgrep pass by baked default) and BUNDLE-011 REQ-010 (delete the legacy
    standards-manifest reader) were originally owned by SPEC-039, which was retired
    terminal (`replaced-by: ISSUE-018`). Under the BUNDLE-014 rule that only `implemented`
    SPECS provide requirement coverage (issues are lineage, not coverage), those two
    BUNDLE-011 requirements were left with no implemented spec carrying them. This spec
    restores that coverage by mirroring the two requirements and anchoring every claim to
    deletion-assertion / survival tests that exist and pass on `main` today. Recorded
    openly per the align-predating-artifacts principle: the record is being brought into
    alignment with what shipped, not the code being changed to fit a record.
    WHAT SHIPPED (verified on `main`, 2026-07-14):
    (1) REQ-007 — the baked file-extension ROUTING is GONE. `pkg/check`'s in-process
    routing manifest (`LoadManifest`, `(*Manifest).RouteFile`, `routeFileDefaults`,
    `defaultManifest`) was deleted with the `backstop code check` engine by ISSUE-018
    (commit d5efd5b). This SUBSUMES and EXCEEDS REQ-007's narrower "delete the `default`
    catch-all branch" scope: with the whole routing layer gone, no baked default sends any
    file to a findings/semgrep pass. Findings enforcement now runs ONLY opt-in through the
    pack engine (`dispatchPackEngines` → the live SARIF surface `ParsePackFindings` /
    `parseSarif`, which survives untouched).
    (2) REQ-010 — the legacy `.manifest.json` standards-manifest READER is GONE. The
    compiled-standards reader (`compiledManifestFile` and its methods, the `.manifest.json`
    decode branch of `LoadManifest`) was removed during the legacy-codecheck collapse
    (commit 824530e), and the surrounding compiled-standards pipeline was eradicated by the
    packs-only removal (SPEC-030 / ISSUE-018): `pkg/compile` is deleted, no production file
    imports it, the compiled-standards artifacts are absent from `.backstop/rules/`, and no
    residual compiled-standards `--config` feed (`ExtraSemgrepConfigs` / `manifestDir`)
    survives. In production, backstop reads no baked standards manifest at all.
    The ONLY thing deliberately KEPT from `pkg/check/manifest.go` is the neutral `CheckType`
    pass-identity enum (the gate's tool-neutral pass vocabulary, stamped onto findings by
    the live SARIF parser) — its survival is pinned as a guard against over-deletion.
  package: pkg/check

verification:
  level: unit
  test_command: go test ./pkg/check/ ./cmd/backstop/ -race
  coverage_threshold: 90

requirements:
  - id: REQ-007
    text: >
      The baked non-Go semgrep catch-all in `pkg/check`'s file-extension routing —
      `routeFileDefaults`'s `default` arm, which routed every non-`.go`/`.ts`/`.tsx` file
      to a findings/semgrep pass — MUST be absent from production source: backstop MUST NOT
      route any file to a findings (semgrep) pass by a baked default. On `main` this is
      satisfied more broadly than the original narrow scope: the entire baked routing
      manifest (`LoadManifest`, `(*Manifest).RouteFile`, `routeFileDefaults`,
      `defaultManifest`) was deleted with the in-process check engine (ISSUE-018), so no
      baked default routing of any kind survives. It is PROHIBITED for any `pkg/check`
      production file to reintroduce `routeFileDefaults`, `(*Manifest).RouteFile`,
      `LoadManifest`, or `defaultManifest`. Findings enforcement is PRESERVED as an OPT-IN
      pack capability: the live SARIF surface (`ParsePackFindings`, `parseSarif`) that the
      gate's `dispatchPackEngines` path consumes MUST survive and continue to stamp
      `CheckTypeFindings`. (Mirrors BUNDLE-011 REQ-007 / DD-3.)
    supports: collapse-legacy-codecheck-into-packs:REQ-007
  - id: REQ-010
    text: >
      The legacy standards-manifest READER MUST be absent from production source: the
      compiled-standards `.manifest.json` reader (`compiledManifestFile` and its methods,
      and the `.manifest.json` decode branch that lived inside `LoadManifest`) MUST NOT be
      present in any `pkg/check` production file, and in production backstop MUST read no
      baked standards manifest to route files. On `main` this is satisfied along with the
      surrounding compiled-standards pipeline: the manifest producer (`pkg/compile`) is
      deleted and no production file imports it, the compiled-standards artifacts
      (`STD-GO-001.manifest.json`, `STD-GO-001.native.json`, `STD-GO-001.semgrep.yml`) are
      absent from `.backstop/rules/`, and no residual compiled-standards `--config` feed
      (`ExtraSemgrepConfigs`, `manifestDir`, `ManifestDir`) survives on any semgrep path. It
      is PROHIBITED to reintroduce the reader, its `.backstop/rules/` artifacts, the
      `pkg/compile` producer, or a residual `--config` feed. The neutral `CheckType`
      pass-identity enum (deliberately kept) MUST survive. (Mirrors BUNDLE-011 REQ-010 /
      DD-5.)
    supports: collapse-legacy-codecheck-into-packs:REQ-010

claims:
  # REQ-007 — baked non-Go catch-all / default routing deleted; findings preserved opt-in
  - id: CLM-001
    requirement: REQ-007
    kind: absence
    text: >
      The baked `routeFileDefaults` routing function — which carried the non-Go semgrep
      catch-all — is absent from every non-test `pkg/check` production source file.
    tests:
      - TestInProcessCheckEngine_Removed
  - id: CLM-002
    requirement: REQ-007
    kind: absence
    text: >
      The routing machinery the catch-all lived inside — `LoadManifest`,
      `(*Manifest).RouteFile`, and `defaultManifest` — is absent from every non-test
      `pkg/check` production source file, so no baked default routes any file to a
      findings/semgrep pass.
    tests:
      - TestInProcessCheckEngine_Removed
  - id: CLM-003
    requirement: REQ-007
    text: >
      GUARD AGAINST OVER-DELETION — findings enforcement is not lost, only made opt-in via
      packs: the live SARIF surface (`ParsePackFindings`, `parseSarif`, `lookupParser`,
      `sarifFingerprint`, `sarifSeverity`) survives the engine deletion and parses a
      minimal SARIF document into a `CheckTypeFindings`-stamped violation, which is how the
      gate's `dispatchPackEngines` path now runs findings.
    tests:
      - TestSARIFSurface_Preserved
  # REQ-010 — legacy standards-manifest reader + its pipeline deleted; CheckType preserved
  - id: CLM-004
    requirement: REQ-010
    kind: absence
    text: >
      The `.manifest.json` standards-manifest reader is gone: `LoadManifest` — the function
      that housed the compiled-standards `.manifest.json` decode branch — is absent from
      every non-test `pkg/check` production source file, so no baked standards manifest is
      read in production.
    tests:
      - TestInProcessCheckEngine_Removed
  - id: CLM-005
    requirement: REQ-010
    kind: absence
    text: >
      No residual compiled-standards `--config` feed survives: `ExtraSemgrepConfigs`,
      `manifestDir`, and `ManifestDir` are absent from `pkg/check` production source, so
      with zero packs there is no leftover standards-dir manifest assembled into a semgrep
      `--config`.
    tests:
      - TestPkgCheck_NoResidualStandardsConfigWhenNoPacks
  - id: CLM-006
    requirement: REQ-010
    kind: absence
    text: >
      The compiled-standards manifest PRODUCER pipeline is gone: no non-test file under
      `cmd/backstop` or `pkg/check` imports `github.com/bmanson/backstop-core/pkg/compile`,
      and the `pkg/compile` package directory is absent from the tree — so nothing produces
      the `.manifest.json` the deleted reader consumed.
    tests:
      - TestNoProductionImportOfCompile
      - TestPkgCompileDirectoryAbsent
  - id: CLM-007
    requirement: REQ-010
    kind: absence
    text: >
      The compiled-standards artifacts (`STD-GO-001.manifest.json`,
      `STD-GO-001.native.json`, `STD-GO-001.semgrep.yml`) are absent from `.backstop/rules/`
      in the repository tree — the reader's on-disk inputs are gone.
    tests:
      - TestCompiledStandardsArtifactsAbsent
  - id: CLM-008
    requirement: REQ-010
    text: >
      GUARD AGAINST OVER-DELETION — the ONE thing deliberately kept from
      `pkg/check/manifest.go`, the neutral `CheckType` pass-identity enum, survives:
      `CheckTypeFindings.String()` returns `"findings"`, and the shared `Violation` /
      `ConfigError` / `CoverageRecord` types remain constructible.
    tests:
      - TestSharedTypes_Preserved

contracts:
  - file: pkg/check/manifest.go
    provides:
      - name: routeFileDefaults
        kind: function
        signature: "func (m *Manifest) routeFileDefaults(path string) []CheckType"
        absent: true
        notes: "DELETED (REQ-007/CLM-001) by ISSUE-018 with the in-process check engine. Carried the non-Go semgrep catch-all (`default` → findings). Declared absent so reintroducing baked default routing is caught as a regression. Absence pinned by TestInProcessCheckEngine_Removed."
      - name: RouteFile
        kind: method
        signature: "func (m *Manifest) RouteFile(path string) []CheckType"
        absent: true
        notes: "DELETED (REQ-007/CLM-002) by ISSUE-018. The sole caller of routeFileDefaults; gone with the routing layer. Declared absent. Absence pinned by TestInProcessCheckEngine_Removed."
      - name: LoadManifest
        kind: function
        signature: "func LoadManifest(dir string) (*Manifest, error)"
        absent: true
        notes: "DELETED (REQ-007/CLM-002 + REQ-010/CLM-004). Housed the `.manifest.json` compiled-standards decode branch (the legacy standards-manifest reader) AND produced the *Manifest that RouteFile routed against. Compiled-standards reader removed in commit 824530e; the routing tail removed by ISSUE-018 (d5efd5b). Declared absent. Absence pinned by TestInProcessCheckEngine_Removed."
      - name: defaultManifest
        kind: function
        signature: "func defaultManifest() *Manifest"
        absent: true
        notes: "DELETED (REQ-007/CLM-002) by ISSUE-018. The always-taken production fallback that produced the baked default-routing Manifest. Declared absent. Absence pinned by TestInProcessCheckEngine_Removed."
      - name: compiledManifestFile
        kind: type
        signature: "type compiledManifestFile struct{}"
        absent: true
        notes: "DELETED (REQ-010/CLM-004) in commit 824530e (legacy-codecheck collapse). The compiled-standards `.manifest.json` reader type. No current test names this symbol directly; its absence is covered transitively by LoadManifest's removal (which housed its decode branch) and by the compiled-standards pipeline-absence tests (CLM-006/CLM-007) — see Sharp Edges. Declared absent so a symbol-level reintroduction is caught by the contract-signature gate."
      - name: CheckType
        kind: type
        signature: "type CheckType int"
        notes: "PRESERVED (REQ-007/REQ-010, CLM-003/CLM-008). The neutral pass-identity enum deliberately kept as the gate's tool-neutral pass vocabulary, stamped onto findings by the live SARIF parser (ParsePackFindings → CheckTypeFindings). NOT deleted; guarded against over-deletion by TestSharedTypes_Preserved."
    consumes: []
---

# SPEC-053: Retroactive Codecheck Deadcode Deletions

## Overview

This is a **retroactive archival spec**. It documents deletions that **already shipped**
and are **verified on `main` today** (2026-07-14). It mandates **no new code and no new
tests** — every claim points at a deletion-assertion or survival test that already exists
and passes.

**Why it exists.** BUNDLE-011 (delivered) declared **REQ-007** (delete the
`routeFileDefaults` non-Go semgrep catch-all — backstop must not route files to a
findings/semgrep pass by baked default) and **REQ-010** (delete the legacy
standards-manifest reader). Those two requirements were originally owned by **SPEC-039**,
which was retired terminal (`status: replaced`, `replaced-by: ISSUE-018`) once the
deletions were subsumed by the wider thin-executor eradication. Under the **BUNDLE-014
rule that only `implemented` SPECS provide requirement coverage** — issues are *lineage*,
not coverage — BUNDLE-011 REQ-007 and REQ-010 were left with no implemented spec carrying
them. This spec closes that traceability gap by mirroring the two requirements and
anchoring them to the tests that prove the shipped reality.

Recorded openly per **align-predating-artifacts**: the record is being brought into
alignment with what shipped, not the code being altered to fit a record.

**What shipped (verified on `main`):**

- **REQ-007** — the baked file-extension **routing is gone**. `pkg/check`'s in-process
  routing manifest (`LoadManifest`, `(*Manifest).RouteFile`, `routeFileDefaults`,
  `defaultManifest`) was deleted with the `backstop code check` engine by **ISSUE-018**
  (commit `d5efd5b`). This **subsumes and exceeds** REQ-007's narrower original scope
  ("delete only the `default` catch-all branch"): with the whole routing layer gone, no
  baked default sends any file to a findings/semgrep pass. Findings enforcement now runs
  **only opt-in** through the pack engine (`dispatchPackEngines` → the live SARIF surface).
- **REQ-010** — the legacy `.manifest.json` **standards-manifest reader is gone**. The
  compiled-standards reader (`compiledManifestFile` + methods, the `.manifest.json` decode
  branch of `LoadManifest`) was removed in commit `824530e` (legacy-codecheck collapse),
  and the surrounding compiled-standards pipeline was eradicated by the packs-only removal
  (SPEC-030 / ISSUE-018): `pkg/compile` deleted, no production importer, artifacts absent
  from `.backstop/rules/`, no residual `--config` feed.

The **only** thing deliberately kept from `pkg/check/manifest.go` is the neutral
`CheckType` pass-identity enum; its survival is pinned as a guard against over-deletion.

## Requirements

Formal requirements are enumerated in the `requirements:` frontmatter (REQ-007, REQ-010),
each tracing to its BUNDLE-011 source via `supports`. The summary below matches the
frontmatter exactly.

| Req | Shipped deletion (verified on `main`) | Anchoring evidence | Preserved (guard) |
|-----|----------------------------------------|--------------------|-------------------|
| REQ-007 | The baked non-Go semgrep catch-all — and, more broadly, the entire routing manifest (`routeFileDefaults`, `(*Manifest).RouteFile`, `LoadManifest`, `defaultManifest`) that housed it — is absent from `pkg/check` production source (ISSUE-018, `d5efd5b`). No baked default routes any file to a findings/semgrep pass. | `TestInProcessCheckEngine_Removed` | Findings run opt-in via the pack engine: live SARIF surface (`ParsePackFindings`/`parseSarif`) survives — `TestSARIFSurface_Preserved` |
| REQ-010 | The legacy `.manifest.json` standards-manifest reader (`compiledManifestFile`, the reader branch of `LoadManifest`) is gone (`824530e`), along with its pipeline: `pkg/compile` producer deleted + no importer, `.backstop/rules/` artifacts absent, no residual compiled-standards `--config` feed (`ExtraSemgrepConfigs`/`manifestDir`). | `TestInProcessCheckEngine_Removed`, `TestPkgCheck_NoResidualStandardsConfigWhenNoPacks`, `TestNoProductionImportOfCompile`, `TestPkgCompileDirectoryAbsent`, `TestCompiledStandardsArtifactsAbsent` | The neutral `CheckType` enum is kept — `TestSharedTypes_Preserved` |

**Explicit prohibition (REQ-007):** no `pkg/check` production file may reintroduce
`routeFileDefaults`, `(*Manifest).RouteFile`, `LoadManifest`, or `defaultManifest`. Findings
enforcement is permitted **only** as an opt-in pack capability through the surviving SARIF
surface.

**Explicit prohibition (REQ-010):** no `pkg/check` production file may reintroduce the
`compiledManifestFile` reader or a `.manifest.json` decode branch; the `pkg/compile`
producer, the `.backstop/rules/` compiled-standards artifacts, and any residual
compiled-standards `--config` feed (`ExtraSemgrepConfigs`/`manifestDir`/`ManifestDir`) may
not return. The neutral `CheckType` enum must be preserved.

## Implementation

**No production change is authored by this spec — the work already shipped.** This section
records the single production package the deletions touched and the mechanical evidence, so
the record is self-contained.

- **Package:** `pkg/check` (with corroborating tree/import checks in `cmd/backstop`).
- **REQ-007 shipped via ISSUE-018 (`d5efd5b`):** deleted the `backstop code check` engine
  and, with it, the entire in-process routing manifest — `routeFileDefaults` (the non-Go
  semgrep catch-all lived in its `default` arm), `(*Manifest).RouteFile`, `LoadManifest`,
  and `defaultManifest`. The live SARIF surface consumed by `dispatchPackEngines`
  (`ParsePackFindings`, `parseSarif`, `lookupParser`, `sarifFingerprint`, `sarifSeverity`)
  was preserved.
- **REQ-010 shipped across the thin-executor eradication:** the compiled-standards reader
  (`compiledManifestFile` + methods, the `.manifest.json` decode branch of `LoadManifest`)
  was removed in commit `824530e` (legacy-codecheck collapse); the compiled-standards
  producer/pipeline (`pkg/compile`, its `.backstop/rules/` artifacts, and the residual
  `--config` feed) was eradicated by the packs-only removal (SPEC-030 / ISSUE-018).
- **Deliberately kept:** the neutral `CheckType` enum in `pkg/check/manifest.go` — the
  gate's tool-neutral pass vocabulary.

No validation pass or gate step is added or removed; this spec adds only traceability.

## Verification

`go test ./pkg/check/ ./cmd/backstop/ -race`, unit level. Claims are defined in the
frontmatter `claims:` array; each maps to a named test that **already exists and passes on
`main`**. Two test shapes:

1. **Deletion-assertion (`kind: absence`) tests** — scan non-test `pkg/check` production
   source (and, for the producer pipeline, `cmd/backstop`) and assert the deleted symbols /
   artifacts are gone: `TestInProcessCheckEngine_Removed` (routing machinery absent),
   `TestPkgCheck_NoResidualStandardsConfigWhenNoPacks` (compiled-standards `--config` feed
   absent), `TestNoProductionImportOfCompile` / `TestPkgCompileDirectoryAbsent` (producer
   pipeline absent), `TestCompiledStandardsArtifactsAbsent` (on-disk artifacts absent).
2. **Survival / over-deletion guards** — `TestSARIFSurface_Preserved` (the live findings
   SARIF path outlives the engine deletion) and `TestSharedTypes_Preserved` (the neutral
   `CheckType` enum and shared types remain constructible).

**Coverage note:** this spec adds no production code, so its `coverage_threshold` is
inherited from the already-green `pkg/check` / `cmd/backstop` packages; it is not a new
per-file coverage obligation (see Sharp Edges).

## Sharp Edges

- **`compiledManifestFile` has no direct symbol-level absence test — this is a real, honest
  verification thinness, not a gap papered over.** No current test names
  `compiledManifestFile` in a deletion assertion (the symbol was deleted in `824530e`,
  before the ISSUE-018 removal-test cluster was written to name the *routing* symbols). Its
  absence is covered **transitively**: `LoadManifest` — which housed the reader's
  `.manifest.json` decode branch — is pinned absent by `TestInProcessCheckEngine_Removed`,
  and the reader's producer/inputs are pinned absent by `TestNoProductionImportOfCompile`,
  `TestPkgCompileDirectoryAbsent`, and `TestCompiledStandardsArtifactsAbsent`. The
  `compiledManifestFile` contract entry (`absent: true`) adds a symbol-level reintroduction
  guard via the contract-signature gate. A vacuous claim naming a test that does not assert
  this symbol was deliberately NOT invented.
- **REQ-007's shipped scope is BROADER than BUNDLE-011 REQ-007's original text.** BUNDLE-011
  REQ-007 scoped only deleting the `default` catch-all *branch* of `routeFileDefaults`. On
  `main` the WHOLE routing layer (`routeFileDefaults` and its callers) is gone, because
  ISSUE-018 deleted the entire in-process engine. The requirement here is stated as the
  superset that actually holds ("the catch-all — and the routing that housed it — is
  absent"); it does not claim the narrow branch-only deletion, which never landed in
  isolation. Anyone reconciling REQ-007's exact original wording against `main` must expect
  the broader deletion, not a surgically trimmed `routeFileDefaults`.
- **Lineage spans multiple commits, not just ISSUE-018.** REQ-007's routing deletion landed
  in ISSUE-018 (`d5efd5b`); REQ-010's compiled-standards reader was already removed in
  `824530e` (the legacy-codecheck collapse) and its pipeline in the packs-only removal
  (SPEC-030). Attributing both cleanly to a single commit would be inaccurate; the claims
  anchor to tests (present-state truth), and the commit attributions are recorded per
  deletion.
- **No live standard governs these deletions, so no `follows` binding is asserted.** SPEC-039
  bound REQ-007/REQ-010 to `STD-GO-001:GO-010`, but STD-GO-001 was itself deleted in the same
  thin-executor eradication (`standards/` now holds only `core` and `typescript`; a
  tree-check test pins the STD-GO-001 source absent). Per escalation-over-guessing, this spec
  does NOT invent a replacement standard-rule mapping for already-shipped Go-code deletions —
  see Review Questions.
- **Retroactive + `implemented` interacts with the gate's coverage/contracts enforcement.**
  The gate enforces contracts/tests/coverage on `implemented` specs. This spec touches no new
  production file, so its coverage obligation is inherited from the already-green packages
  rather than a fresh per-file target. If the gate flags a coverage obligation here, that is a
  signal about retroactive-spec handling, not about missing tests — the anchor tests all pass.

## Review Questions

- Does every claim anchor to a test that exists and passes on `main` **today**, with no
  claim mandating new code or a new test? (This spec is retroactive; a claim pointing at a
  not-yet-written test would be a defect.)
- Is the `compiledManifestFile` verification thinness (no direct symbol-level test) disclosed
  honestly in Sharp Edges rather than masked by a claim that names a test which does not
  actually assert that symbol?
- Does REQ-007's text correctly state the BROADER shipped reality (the whole routing layer is
  gone) without claiming the narrow branch-only deletion that never landed in isolation?
- Is the absence of a `follows` standard-rule binding justified (STD-GO-001 deleted) rather
  than an oversight — and should any live `core`/`typescript` standard rule instead be bound,
  or is escalation-over-guessing the correct call for already-shipped Go-code deletions?
- Do the survival guards (`TestSARIFSurface_Preserved`, `TestSharedTypes_Preserved`) actually
  pin that findings enforcement and the neutral `CheckType` enum OUTLIVE the deletions, so the
  spec proves "made opt-in / deliberately kept" rather than "silently removed"?

## References

- **BUNDLE-011** (`bundles/BUNDLE-011-collapse-legacy-codecheck-into-packs.bundle.md`,
  delivered) — the source bundle. This spec mirrors REQ-007 (DD-3 / RDQ-3) and REQ-010
  (DD-5 / RDQ-5).
- **SPEC-039** (`specs/SPEC-039-codecheck-deadcode-prelude.spec.md`, `replaced`,
  `replaced-by: ISSUE-018`) — the original owner of REQ-007/REQ-010, retired terminal once
  the deletions were subsumed. The "why retroactive" lineage.
- **ISSUE-018** (`issues/ISSUE-018-remove-vestigial-baked-in-code.issue.md`, closed) — the
  work item under which the routing layer was actually deleted (commit `d5efd5b`); carries
  the ISSUE-018 removal-test cluster. Lineage, NOT coverage (per the BUNDLE-014 rule that only
  implemented SPECS provide coverage — the gap this spec closes).
- **SPEC-030** (`specs/SPEC-030-packs-only-native-standards-removal.spec.md`, `implemented`)
  — eradicated the compiled-standards pipeline (`pkg/compile`, `.backstop/rules/` artifacts);
  the `kind: absence` claim convention this spec follows.
- **SPEC-046** (`specs/SPEC-046-retire-language-toolchain-bridge.spec.md`, `implemented`) —
  the pure-deletion contract shape (`absent: true` provides) this spec mirrors.
- Anchor tests (verified passing on `main`, 2026-07-14):
  `pkg/check/code_check_engine_removal_test.go` (`TestInProcessCheckEngine_Removed`,
  `TestSARIFSurface_Preserved`, `TestSharedTypes_Preserved`);
  `pkg/check/semgrep_removal_test.go` (`TestPkgCheck_NoResidualStandardsConfigWhenNoPacks`);
  `cmd/backstop/standards_removal_test.go` (`TestNoProductionImportOfCompile`,
  `TestPkgCompileDirectoryAbsent`, `TestCompiledStandardsArtifactsAbsent`).
- Code (verified on `main`, 2026-07-14): `pkg/check/manifest.go` (now only the neutral
  `CheckType` enum + `String()`; routing machinery deleted); commit `d5efd5b` (ISSUE-018,
  routing/engine deletion); commit `824530e` (legacy-codecheck collapse, compiled-standards
  reader removed).

## Version History

- **1.0.0** (2026-07-14) — Initial retroactive archival spec. Mirrors BUNDLE-011 REQ-007
  (baked non-Go semgrep catch-all / routing manifest deleted — no baked default findings
  routing) and REQ-010 (legacy `.manifest.json` standards-manifest reader + its pipeline
  deleted), each anchored to deletion-assertion and survival tests already passing on `main`.
  Authored `implemented` to close the traceability gap left when SPEC-039 was retired
  `replaced-by: ISSUE-018` and the two requirements lost an implemented-spec carrier under the
  BUNDLE-014 issues-are-lineage-not-coverage rule. Recorded openly per align-predating-artifacts.
