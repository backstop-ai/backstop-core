---
title: "Pack-Declared File-Classification Globs + De-Go'd Coverage Measurable-Path"
number: SPEC-043
created: "2026-06-28"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    BUNDLE-012 Spec Seed 1 — the FOUNDATION contract the bundle's other consumer
    specs depend on. It does TWO coupled things. (1) DEFINES the pack-declared
    file-CLASSIFICATION contract: a toolchain pack declares, as DATA in its
    `pack.yml`, which files are SOURCE and which are TEST via two glob lists
    (`classification.source`, `classification.test`), parsed onto a new
    `pack.Manifest.Classification` field. This is the keystone the rest of
    BUNDLE-012's de-Go work (REQ-002/003/005/006) consumes; the binary gains ZERO
    baked language knowledge (DD-1, the thin-executor first principle). (2) APPLIES
    that contract to the single highest-correctness-impact site: `pkg/gate/
    step_coverage.go` `coverageMeasurablePath` (~L232) today returns false for any
    path not ending in `.go`, so in the DEFAULT diff scope a changed `.ts` file is
    SILENTLY SKIPPED and coverage passes VACUOUS-GREEN for non-Go. The baked
    `.go`/`_test.go`/`testdata` literals are replaced with a language-neutral
    classifier built from the MERGED UNION of the declared source/test globs across
    ALL declared toolchain packs (a polyglot repo declares several). A path is
    MEASURABLE SOURCE iff it matches some declared source glob and no declared test
    glob (test-wins-on-overlap). The single most important behavior in the bundle
    is the anti-vacuous-green guard: an in-scope CHANGED file classified as
    measurable source with NO coverage record (and not pack-declared excluded) is a
    LOUD blocking error, DISTINCT from below-threshold, and fires regardless of
    whether a numeric threshold is declared in scope (a declared source glob is
    itself the measurement promise). When NO toolchain pack declares any source
    globs, the step reports a DISTINCT, VISIBLE "classification capability absent"
    state (non-blocking warn) — never a silent pass. SEAM with SPEC-044 (authored
    in parallel; shares this file): SPEC-044 owns the coverage RECORD model
    (multiple metrics per file keyed by `(path, metric)`, line+branch, per-metric
    thresholds); THIS spec's no-record guard is expressed as "no record for the
    path AT ALL" (any metric) so it composes cleanly with SPEC-044's
    `(path, metric)` index, and the per-metric threshold verdict stays SPEC-044's.
    Per the integration-gap lesson, the merged classifier MUST be threaded into the
    LIVE gate coverage step (cmd/backstop wiring), proven end-to-end, not only
    unit-tested over a constructed classifier.
  package: pkg/gate

verification:
  level: integration
  test_command: go test ./pkg/gate/ ./pkg/pack/ ./cmd/backstop/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      DEFINE the pack-declared file-CLASSIFICATION contract as DATA in the pack
      manifest. A toolchain pack declares two OPTIONAL glob lists under a top-level
      `classification:` block: `classification.source` (patterns whose matches are
      SOURCE files coverage is expected for) and `classification.test` (patterns
      whose matches are TEST/non-source files — including a stack's fixture/testdata
      convention, e.g. `**/testdata/**`, which is folded into the test list, NOT a
      separate baked dimension). These parse onto a new `pack.Manifest.Classification`
      field (`type Classification struct { Source []string; Test []string }`). The
      block is OPTIONAL: a manifest with no `classification:` yields a zero-value
      Classification and NO parse error. Glob semantics are doublestar-capable and
      path-segment-aware: `**` crosses path separators, `*` matches within a single
      segment, matching is on the project-relative slash path. The contract is
      DATA — backstop bakes no language-specific source/test convention; every
      stack supplies its own globs (DD-1). The reference shape is the
      `backstop/go-toolchain` pack declaring `source: ["**/*.go"]`,
      `test: ["**/*_test.go", "**/testdata/**"]`.
    supports: language-neutral-consumer-ts-toolchain:REQ-001
  - id: REQ-002
    text: >
      The coverage consumer MUST derive its in-scope MEASURABLE SOURCE set from the
      pack-declared globs, NOT a baked `.go` extension. The classifier reads the
      MERGED UNION of every declared toolchain pack's `classification.source` and
      `classification.test` globs (a polyglot repo declares several packs; their
      glob sets are unioned). A path is MEASURABLE SOURCE iff it matches at least
      one declared SOURCE glob AND matches NO declared TEST glob — TEST-WINS-ON-
      OVERLAP (a test glob is normally a more-specific subset of a source glob, e.g.
      `**/*_test.go` ⊂ `**/*.go`, so a test/fixture file is never measured). It is
      PROHIBITED for `coverageMeasurablePath` (or its successor) to retain any baked
      `.go`, `_test.go`, or `testdata` string literal: with a classifier that
      declares only non-Go globs, a `.go` file MUST NOT be measurable, proving the
      baked Go literal is gone (DD-1). Files matching only a test glob, or matching
      no declared glob, are NOT measurable.
    supports: language-neutral-consumer-ts-toolchain:REQ-001
  - id: REQ-003
    text: >
      The anti-vacuous-green guard (load-bearing). An in-scope CHANGED file that the
      declared globs classify as MEASURABLE SOURCE and that has NO coverage record
      for the path AT ALL — and that is NOT pack-declared excluded — MUST produce a
      LOUD blocking error (severity error, status fail), NEVER a silent pass. This
      state MUST be DISTINCT from below-threshold (a different rule/message so the
      two are not conflated on the report). The guard MUST fire REGARDLESS of whether
      a numeric coverage threshold is declared in scope: a declared source glob is
      itself the promise that the file is measurable, so "matched the source glob but
      nothing measured it" cannot be silenced by the mere absence of a threshold
      number. SEAM: the guard keys on "no record for the path at all" (ANY metric); a
      path carrying at least one record (any metric) is NOT a no-record error — the
      PER-METRIC threshold verdict for a present record is owned by SPEC-044 and is
      out of scope here. A changed file classified as TEST (or unclassified) is NOT
      subject to this guard. A pack-declared-excluded path is skipped from the guard
      (its exclusion loudness is SPEC-041 behavior, unchanged).
    supports: language-neutral-consumer-ts-toolchain:REQ-001
    follows: STD-GO-001:GO-010
  - id: REQ-004
    text: >
      When the merged classifier declares NO source globs at all (no declared
      toolchain pack carries a `classification.source` list), the coverage step MUST
      report a DISTINCT, VISIBLE "classification capability absent" state rather than
      a silent pass with nothing measured. Consistent with loud-not-blocking, this
      state is NON-blocking (status MUST NOT be fail solely because classification is
      absent) but MUST be surfaced on the report (a distinct reason/status), so the
      inability to classify is visible and never masquerades as a green coverage
      result. It is PROHIBITED to return an unqualified `pass` when no source globs
      are declared and in-scope changed files exist.
    supports: language-neutral-consumer-ts-toolchain:REQ-001
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      The merged classifier MUST be threaded into the LIVE gate coverage step
      (`cmd/backstop` wiring around `buildCoverageStep`), built from the UNION of the
      bridged and declared toolchain packs, and CONSUMED by the real
      `StepCoverageThresholdScopedFunc`. This MUST be proven END-TO-END (an executed
      gate over a declared toolchain pack whose source globs cover a non-Go file),
      not only by unit tests over a hand-constructed classifier — closing the
      integration gap where a correct unit is never actually wired into the gate. The
      merge MUST be a UNION across all declared toolchain packs (two declared
      toolchain packs contribute both glob sets to the live step).
    supports: language-neutral-consumer-ts-toolchain:REQ-001

claims:
  # REQ-001 — the pack-declared classification contract (DATA shape)
  - id: CLM-001
    requirement: REQ-001
    text: A pack.yml top-level `classification:` block with `source:` and `test:` glob lists parses onto pack.Manifest.Classification (Source/Test string slices) in declared order
    tests:
      - TestManifest_ParsesClassificationGlobs
  - id: CLM-002
    requirement: REQ-001
    text: A manifest with NO `classification:` block yields a zero-value Classification (empty Source and Test) and NO parse error — the block is optional
    tests:
      - TestManifest_AbsentClassificationIsEmptyNotError
  - id: CLM-003
    requirement: REQ-001
    text: The go-toolchain reference shape round-trips — source `["**/*.go"]`, test `["**/*_test.go", "**/testdata/**"]` parse intact, with the fixture/testdata convention folded into the test list (no separate baked testdata dimension)
    tests:
      - TestManifest_GoToolchainClassificationReferenceShapeRoundTrips
  # REQ-002 — measurable-source classification matrix over merged globs
  - id: CLM-004
    requirement: REQ-002
    text: A path matching a declared SOURCE glob and NO declared test glob is MEASURABLE SOURCE (classifier returns true) — e.g. `pkg/x/foo.go` under source `**/*.go`
    tests:
      - TestClassifier_SourceOnlyIsMeasurable
  - id: CLM-005
    requirement: REQ-002
    text: A path matching BOTH a source glob AND a test glob is NOT measurable — TEST-WINS-ON-OVERLAP (e.g. `pkg/x/foo_test.go` matches source `**/*.go` and test `**/*_test.go`, so it is excluded)
    tests:
      - TestClassifier_SourceAndTestOverlapNotMeasurable
  - id: CLM-006
    requirement: REQ-002
    text: A path matching ONLY a declared TEST glob (no source glob) is NOT measurable
    tests:
      - TestClassifier_TestOnlyNotMeasurable
  - id: CLM-007
    requirement: REQ-002
    text: A path matching NO declared glob (neither source nor test) is NOT measurable
    tests:
      - TestClassifier_UnclassifiedNotMeasurable
  - id: CLM-008
    requirement: REQ-002
    text: The classifier is the MERGED UNION across multiple toolchain packs — with a go pack (source `**/*.go`) and a bun pack (source `**/*.ts`) declared, BOTH a `.go` and a `.ts` source file are measurable from the one merged classifier (polyglot)
    tests:
      - TestClassifier_UnionAcrossMultipleToolchainPacks
  - id: CLM-009
    requirement: REQ-002
    text: No baked Go literal survives — given a classifier declaring ONLY non-Go globs (e.g. bun `**/*.ts`), a `.go` file is NOT measurable, proving `coverageMeasurablePath` no longer hard-codes `.go` as source
    tests:
      - TestClassifier_NoBakedGoLiteral_GoNotMeasurableWithoutGoGlobs
  - id: CLM-010
    requirement: REQ-002
    text: Glob semantics are doublestar/segment-aware — `**/*.ts` matches a nested `app/a/b.ts`; a non-doublestar `*.ts` does not match across separators; matching is on the project-relative slash path
    tests:
      - TestClassifier_GlobSemanticsDoublestarAndSegmentAware
  - id: CLM-011
    requirement: REQ-002
    text: The measurable-path implementation contains NO baked `.go`/`_test.go`/`testdata` string literal — a source guard over pkg/gate asserts the classification path keys only on the declared globs, so a reintroduced baked extension fails
    tests:
      - TestClassifier_NoBakedExtensionLiteralsInMeasurablePath
  # REQ-003 — anti-vacuous-green loud guard (keystone)
  - id: CLM-012
    requirement: REQ-003
    text: VACUOUS-GREEN REGRESSION — a changed NON-Go source file (`app/foo.ts`) matching the declared source glob, with NO coverage record and not excluded, yields a LOUD blocking violation (severity error, status fail), NOT a silent green; this is the exact default-diff-scope hole the bundle exists to close
    tests:
      - TestCoverage_ChangedTSSourceFileNoRecordIsLoudBlockingNotVacuousGreen
  - id: CLM-013
    requirement: REQ-003
    text: The no-record loud state is DISTINCT from below-threshold — a no-record changed file and a below-threshold changed file produce distinguishable violations (different rule/message), so the report never conflates "nothing measured" with "measured low"
    tests:
      - TestCoverage_NoRecordDistinctFromBelowThreshold
  - id: CLM-014
    requirement: REQ-003
    text: The no-record guard fires even when NO numeric coverage threshold is declared in scope — a changed measurable-source file with no record REDs (loud blocking) with the threshold absent, because the declared source glob is itself the measurement promise
    tests:
      - TestCoverage_NoRecordGuardFiresWithoutNumericThreshold
  - id: CLM-015
    requirement: REQ-003
    text: A changed measurable-source file WITH a coverage record present is NOT flagged by the no-record guard — it is handed to the threshold verdict path instead (the guard is strictly about absence)
    tests:
      - TestCoverage_ChangedSourceWithRecordNotFlaggedByNoRecordGuard
  - id: CLM-016
    requirement: REQ-003
    text: A changed TEST file (e.g. `app/foo.test.ts` matching the declared test glob) with no record does NOT trigger the loud guard — test/non-source files carry no coverage requirement
    tests:
      - TestCoverage_ChangedTestFileNoRecordNotFlagged
  - id: CLM-017
    requirement: REQ-003
    text: SEAM with SPEC-044 — the guard keys on "no record for the path AT ALL" (any metric); a path carrying at least one record under any metric is NOT a no-record error, leaving the per-metric threshold verdict to SPEC-044's `(path, metric)` model
    tests:
      - TestCoverage_NoRecordGuardChecksAnyMetricRecordPresence
  # REQ-004 — no declared source globs is visible, not a silent pass
  - id: CLM-018
    requirement: REQ-004
    text: When the merged classifier has NO source globs and in-scope changed files exist, the coverage step returns a DISTINCT, VISIBLE "classification capability absent" status/reason — never an unqualified `pass` with nothing measured
    tests:
      - TestCoverage_NoDeclaredSourceGlobsIsVisibleCapabilityAbsentNotSilentPass
  - id: CLM-019
    requirement: REQ-004
    text: The classification-capability-absent state is NON-blocking (status is not fail solely because classification is absent) — loud, but not a block; the absence is surfaced rather than punished
    tests:
      - TestCoverage_ClassificationCapabilityAbsentDoesNotBlock
  # REQ-005 — the merged classifier is wired into the live gate (integration gap)
  - id: CLM-020
    requirement: REQ-005
    text: END-TO-END — the live gate threads the merged classifier into the coverage step; with a declared toolchain pack whose source globs cover `.ts`, a changed `.ts` file with no record REDs the real gate, proving the classifier is actually consumed (not bypassed)
    tests:
      - TestGate_CoverageStepConsumesMergedClassifierEndToEnd
  - id: CLM-021
    requirement: REQ-005
    text: The live merge is a UNION across declared toolchain packs — with two toolchain packs declared (go + bun), the coverage step measures BOTH a changed `.go` and a changed `.ts` file from the merged glob set
    tests:
      - TestGate_ClassifierMergesAcrossDeclaredToolchainPacks

contracts:
  - file: pkg/pack/manifest.go
    provides:
      - name: Classification
        kind: type
        signature: "type Classification struct { Source []string `yaml:\"source\"`; Test []string `yaml:\"test\"` }"
        notes: "NEW (REQ-001/CLM-001): the pack-declared file-classification DATA. Source globs are patterns whose matches are SOURCE files coverage is expected for; Test globs are patterns whose matches are TEST/non-source files (a stack folds its fixture/testdata convention into Test, e.g. `**/testdata/**`). Parsed from the OPTIONAL top-level `classification:` yaml block via a `yaml:\"classification\"`-tagged field on Manifest. Absent block => zero value, no error (CLM-002). The binary holds no baked source/test convention — every stack supplies its own globs (DD-1)."
      - name: Manifest.Classification
        kind: variable
        signature: "Classification Classification `yaml:\"classification\"`"
        notes: "NEW field on pack.Manifest carrying the parsed classification block. Optional; zero-value when absent (CLM-002). The go-toolchain reference declares Source `[\"**/*.go\"]`, Test `[\"**/*_test.go\", \"**/testdata/**\"]` (CLM-003)."
    consumes: []
  - file: pkg/gate/classification.go
    provides:
      - name: SourceClassifier
        kind: type
        signature: "type SourceClassifier struct"
        notes: "NEW (REQ-002): the language-neutral merged classifier the coverage consumer reads instead of a baked `.go` literal. Holds the UNION of declared source and test globs (compiled doublestar matchers). Carries no language knowledge — it is data + match logic only."
      - name: NewSourceClassifier
        kind: function
        signature: "func NewSourceClassifier(source, test []string) SourceClassifier"
        notes: "NEW (REQ-002/CLM-008): constructs a classifier from MERGED source/test glob lists (the union across all declared toolchain packs is assembled by the caller in cmd/backstop and passed here). Compiles doublestar/segment-aware matchers (CLM-010)."
      - name: SourceClassifier.IsMeasurableSource
        kind: method
        signature: "func (c SourceClassifier) IsMeasurableSource(path string) bool"
        notes: "NEW (REQ-002/CLM-004..CLM-009): true iff path matches some declared SOURCE glob AND no declared TEST glob (test-wins-on-overlap). With only non-Go globs declared, a `.go` path returns false (CLM-009 — no baked Go literal). Replaces the baked-literal `coverageMeasurablePath`."
      - name: SourceClassifier.HasSourceGlobs
        kind: method
        signature: "func (c SourceClassifier) HasSourceGlobs() bool"
        notes: "NEW (REQ-004/CLM-018): reports whether any source globs are declared, so the coverage step can surface the DISTINCT 'classification capability absent' state instead of a silent pass."
    consumes: []
  - file: pkg/gate/step_coverage.go
    provides:
      - name: StepCoverageThresholdScopedFunc
        kind: function
        signature: "func StepCoverageThresholdScopedFunc(coverage []check.CoverageRecord, specs []SpecVerification, scope *GateScope, classifier SourceClassifier) StepFunc"
        notes: "MODIFIED (REQ-002/REQ-003/REQ-004): gains a SourceClassifier parameter. The in-scope measurable set is derived from classifier.IsMeasurableSource instead of the baked coverageMeasurablePath (REQ-002). An in-scope changed measurable-source path with NO record for the path AT ALL (any metric) and not pack-declared excluded is a LOUD blocking error, DISTINCT from below-threshold, fired even when no numeric threshold is in scope (REQ-003/CLM-012..CLM-017). When classifier.HasSourceGlobs() is false and in-scope changed files exist, the step returns a DISTINCT non-blocking 'classification capability absent' status, never an unqualified pass (REQ-004/CLM-018/CLM-019). SEAM: the no-record predicate is 'any record for the path' so it composes with SPEC-044's (path, metric) index; per-metric threshold verdict is SPEC-044's. The existing SPEC-041 exclusion-loudness and per-file threshold semantics are retained."
      - name: StepCoverageThresholdFunc
        kind: function
        signature: "func StepCoverageThresholdFunc(coverage []check.CoverageRecord, specs []SpecVerification, classifier SourceClassifier) StepFunc"
        notes: "MODIFIED convenience wrapper: delegates to StepCoverageThresholdScopedFunc with nil scope, now carrying the SourceClassifier."
      - name: coverageMeasurablePath
        kind: function
        signature: "func coverageMeasurablePath(path string) bool"
        absent: true
        notes: "DELETED (REQ-002/CLM-009/CLM-011): the baked `.go`/`_test.go`/`testdata`-literal measurable-path predicate. Replaced by SourceClassifier.IsMeasurableSource over pack-declared globs. Declared absent so reintroducing a baked extension literal is caught as a regression."
    consumes:
      - source: pkg/gate
        name: SourceClassifier
        kind: type
      - source: pkg/gate
        name: GateScope
        kind: type
      - source: pkg/check
        name: CoverageRecord
        kind: type
  - file: cmd/backstop/gate.go
    provides:
      - name: mergeSourceClassifier
        kind: function
        signature: "func mergeSourceClassifier(packSets ...[]*pack.Manifest) gate.SourceClassifier"
        notes: "NEW (REQ-005/CLM-021): unions the classification.source and classification.test globs across the bridged and declared toolchain packs into one gate.SourceClassifier. Built where the manifests are visible (cmd/backstop) so pkg/gate takes no pkg/pack dependency."
      - name: buildCoverageStep
        kind: function
        signature: "func buildCoverageStep(specDir, projectRoot string, scope *gate.GateScope, classifier gate.SourceClassifier, records coverageRecordsFn) gate.StepFunc"
        notes: "MODIFIED (REQ-005/CLM-020): gains the merged classifier and threads it into StepCoverageThresholdScopedFunc so the LIVE gate consumes the declared globs end-to-end (closing the integration gap). The gate-assembly caller builds the classifier via mergeSourceClassifier(bridged, packs)."
    consumes:
      - source: pkg/gate
        name: SourceClassifier
        kind: type
      - source: pkg/pack
        name: Manifest
        kind: type
---

# SPEC-043: Pack-Declared File-Classification Globs + De-Go'd Coverage Measurable-Path

## Overview

This spec is **Seed 1 of BUNDLE-012** (language-neutral gate consumer + TypeScript
toolchain pack) — the **foundation contract** the bundle's other consumer specs
(REQ-002/003/005/006) depend on. It is authored in **parallel with SPEC-044** (the
multi-metric coverage **record** model); the two share `pkg/gate/step_coverage.go`,
so the seam between them is defined explicitly below.

It does two coupled things:

1. **Defines the pack-declared file-classification contract** — the keystone. A
   toolchain pack declares, as DATA in `pack.yml`, which files are SOURCE and which
   are TEST, via two optional glob lists (`classification.source`,
   `classification.test`). The binary gains ZERO baked language knowledge: every
   stack supplies its own globs (**DD-1**, the thin-executor first principle).

2. **De-Go's the coverage measurable-path (BUNDLE-012 REQ-001)** — the single
   highest-correctness-impact site. `pkg/gate/step_coverage.go`
   `coverageMeasurablePath` (~L232) today returns `false` for any path not ending in
   `.go`, so in the **default diff scope** a changed `.ts` file is **silently
   skipped** and coverage passes **vacuous-green** for non-Go — *worse than an honest
   red, because it looks like coverage passed*. The baked `.go`/`_test.go`/`testdata`
   literals are replaced with a language-neutral classifier built from the **merged
   union** of declared source/test globs across all declared toolchain packs.

The **single most important behavior in the bundle** is the anti-vacuous-green guard
(REQ-003): a changed measurable-source file with no coverage record is a **loud
blocking error**, distinct from below-threshold, never a silent pass
([[feedback_loud_not_blocking]]).

**In scope:** the classification contract DATA shape; the merged-glob measurable-set
derivation; the loud "no record for a measurable-source path" guard; the
"no-source-globs-declared is visible, not silent" state; and wiring the merged
classifier into the live gate end-to-end.

**Out of scope (fenced to sibling seeds):** the coverage RECORD model — multiple
metrics per file keyed by `(path, metric)`, line+branch, per-metric thresholds →
**SPEC-044** (the parallel sibling). De-Go'ing test-verification discovery + the
go-package/`./...` matchers → **Seed 2**. Deleting the `language:` bridge + retiring
the `language:` field + the traceability-classifier rehome → **Seed 3**. The
`backstop/bun-toolchain` pack + the ratchet→block flip + the external executed proof
→ **Seed 5**.

## Requirements

Requirements are enumerated in the `requirements:` frontmatter (REQ-001 … REQ-005),
each tracing to BUNDLE-012 REQ-001 via `supports`. Summary:

| Spec REQ | Commits to |
| --- | --- |
| REQ-001 | The pack-declared classification contract: optional `classification.source` / `classification.test` glob lists parsed onto `pack.Manifest.Classification`; absent block => zero value, no error. |
| REQ-002 | The consumer derives the measurable-source set from the MERGED UNION of declared globs; measurable ⟺ matches some source glob AND no test glob (test-wins-on-overlap); no baked `.go`/`_test.go`/`testdata` literal survives. |
| REQ-003 | The anti-vacuous-green guard: an in-scope changed measurable-source file with NO record (any metric) and not excluded is a LOUD blocking error, DISTINCT from below-threshold, fired even with no numeric threshold in scope. |
| REQ-004 | No declared source globs => a DISTINCT, VISIBLE non-blocking "classification capability absent" state, never a silent pass. |
| REQ-005 | The merged classifier is threaded into the LIVE gate coverage step (cmd/backstop wiring) and proven end-to-end — not only unit-tested. |

### The classification contract (REQ-001)

The contract is **two glob lists** under a top-level `classification:` block — the
minimal language-neutral shape that lets the consumer answer "is this changed file a
source file coverage is expected for?":

```yaml
# backstop/go-toolchain pack.yml (reference shape)
classification:
  source:
    - "**/*.go"
  test:
    - "**/*_test.go"
    - "**/testdata/**"
```

```yaml
# backstop/bun-toolchain pack.yml (Seed 5; shown here only to fix the contract shape)
classification:
  source:
    - "**/*.ts"
    - "**/*.tsx"
  test:
    - "**/*.test.ts"
    - "**/*.spec.ts"
```

- The block is **optional**: a manifest with no `classification:` parses to a
  zero-value `Classification` with no error (it simply contributes nothing to the
  merged classifier).
- A stack's **fixture/testdata convention** is folded into the `test` list (a
  non-source, non-measured set) rather than a separate baked dimension — this is how
  the old baked `testdata` literal is retired (Go declares `**/testdata/**`).
- **Glob semantics:** doublestar-capable and path-segment-aware — `**` crosses path
  separators, `*` matches within a single segment; matching is on the
  project-relative slash path.

### Classification matrix (REQ-002)

`SourceClassifier.IsMeasurableSource(path)`, over the merged (unioned) source set `S`
and test set `T`:

| path matches source glob? | path matches test glob? | measurable source? | Claim |
| --- | --- | --- | --- |
| yes | no | **YES** | CLM-004 |
| yes | yes (overlap) | **NO** (test wins) | CLM-005 |
| no | yes | **NO** | CLM-006 |
| no | no | **NO** | CLM-007 |

Plus: the classifier is the **union** across packs, so a polyglot repo measures both
`.go` and `.ts` (CLM-008); with only non-Go globs declared, a `.go` file is **not**
measurable, proving no baked Go literal survives (CLM-009); glob matching is
doublestar/segment-aware (CLM-010); and the measurable-path implementation holds **no**
baked `.go`/`_test.go`/`testdata` string literal (CLM-011).

### The anti-vacuous-green guard (REQ-003) and the SPEC-044 seam

The guard fires for an in-scope **changed** file that is **measurable source** and has
**no coverage record for the path at all** and is **not** pack-declared excluded →
**loud blocking error**, distinct from below-threshold, fired even when no numeric
threshold is in scope.

**Seam with SPEC-044 (shared file `pkg/gate/step_coverage.go`):**

| Owned by SPEC-043 (this) | Owned by SPEC-044 (parallel sibling) |
| --- | --- |
| The pack-declared globs contract (`classification.source`/`.test`). | The coverage RECORD model — multiple metrics per file. |
| The in-scope measurable PATH SET derived from the globs. | The `(path, metric)`-keyed index (successor to `indexCoverageByPath`). |
| The "changed source file with **no record for the path at all** → loud error" guard. | The **per-metric threshold verdict** for a record that IS present. |

The guard is expressed in terms of **"no record for the path at all" (any metric)** so
it composes with SPEC-044's `(path, metric)` index without redefining the record schema
or thresholding. Both specs add to `StepCoverageThresholdScopedFunc`'s signature — this
spec adds the `classifier SourceClassifier` parameter; SPEC-044 evolves the record
consumption. **This shared signature is flagged for the cross-consistency pass.**

## Implementation

Target package: **`pkg/gate`** (the classifier + the de-Go'd coverage step), with the
classification field in **`pkg/pack`** and the merge/wiring in **`cmd/backstop`**. The
processing steps the planner must map tasks to:

1. **Add the classification contract to the manifest (REQ-001).** Add
   `type Classification struct { Source []string; Test []string }` and a
   `Classification Classification \`yaml:"classification"\`` field to `pack.Manifest`
   in `pkg/pack/manifest.go`. The yaml block is optional (non-strict `yaml.Unmarshal`
   already ignores absence — a missing block yields the zero value). Add the
   reference globs to the `backstop/go-toolchain` pack's `pack.yml`
   (`source: ["**/*.go"]`, `test: ["**/*_test.go", "**/testdata/**"]`) so the dogfood
   gate keeps classifying `.go` once the baked literal is gone.

2. **Build the language-neutral classifier (REQ-002).** Add
   `pkg/gate/classification.go` with `SourceClassifier`, `NewSourceClassifier(source,
   test []string)`, `IsMeasurableSource(path) bool` (matches some source glob AND no
   test glob — test-wins-on-overlap), and `HasSourceGlobs() bool`. Use a
   doublestar/segment-aware matcher (a `**`-capable glob, e.g. the
   already-in-module-graph `github.com/gobwas/glob` compiled with `/` as separator);
   compile each glob once at construction.

3. **De-Go the measurable-path (REQ-002).** DELETE `coverageMeasurablePath` (the baked
   `.go`/`_test.go`/`testdata` literals) from `pkg/gate/step_coverage.go`. Re-key
   `coveragePathsInScope` to filter in-scope changed files via
   `classifier.IsMeasurableSource`. No baked extension literal remains.

4. **The anti-vacuous-green guard (REQ-003).** Re-shape
   `StepCoverageThresholdScopedFunc` (new `classifier SourceClassifier` parameter) so
   the measurable-source no-record scan runs **before/independent of** the numeric
   threshold short-circuit: an in-scope changed measurable-source path with no record
   for the path **at all** and not pack-declared excluded emits a loud blocking
   `coverage_threshold`-distinct violation (a distinct rule, e.g.
   `coverage_unmeasured`, so it is not conflated with below-threshold). The "any
   record present for the path" predicate is the seam point — it asks only whether
   *some* record exists for the path, leaving per-metric verdicts to SPEC-044. The
   existing SPEC-041 exclusion-loudness and per-file threshold behavior are retained.

5. **Classification-capability-absent state (REQ-004).** When
   `classifier.HasSourceGlobs()` is false and in-scope changed files exist, return a
   DISTINCT non-blocking status/reason (e.g. status `pass` is PROHIBITED here; surface
   a `warning`/`capability_absent`-style reason naming that no toolchain pack declared
   source classification) so the inability to classify is visible, never a silent
   green.

6. **Wire the merged classifier into the live gate (REQ-005).** Add
   `mergeSourceClassifier(packSets ...[]*pack.Manifest) gate.SourceClassifier` in
   `cmd/backstop/gate.go` (unions the source/test globs across the bridged and
   declared toolchain packs), give `buildCoverageStep` the classifier parameter, and
   thread it into `StepCoverageThresholdScopedFunc`. Prove end-to-end that the live
   gate measures a changed non-Go file via the declared globs (CLM-020) and unions
   across multiple declared toolchain packs (CLM-021).

## Verification

- **Level:** `integration` (threshold 80). The classifier and guard are unit-testable
  in `pkg/gate`, but REQ-005 is a cross-package wiring guarantee (`cmd/backstop` →
  `pkg/gate`), so the spec is verified at integration level with a mandated
  end-to-end gate test that the merged classifier is actually consumed
  ([[feedback_integration_gap]]).
- **Command:** `go test ./pkg/gate/ ./pkg/pack/ ./cmd/backstop/ -race
  -coverprofile=cover.out`.
- **Mandated tests:** every test named in the `claims[]` `tests:` fields. The
  load-bearing one is `TestCoverage_ChangedTSSourceFileNoRecordIsLoudBlockingNotVacuousGreen`
  (CLM-012) — the vacuous-green regression test: a changed non-Go source file matching
  the declared source glob with no record MUST yield a loud blocking violation, not a
  silent green.

## Sharp Edges

- **The vacuous-green hole is the whole point — do not re-open it.** The fix is not
  "measure `.ts` too"; it is "any changed file the declared globs call source must be
  measured or loudly fail." If the no-record guard is left gated behind the
  `threshold <= 0` short-circuit (as the current code is, lines 57-59), a TS project
  with no spec-declared threshold passes vacuous-green again. REQ-003 explicitly
  requires the guard to fire **independent of** a numeric threshold being declared.
- **Test-wins-on-overlap is load-bearing.** A test glob is normally a more-specific
  subset of a source glob (`**/*_test.go` ⊂ `**/*.go`, `**/*.test.ts` ⊂ `**/*.ts`). If
  source-wins, every test file becomes a measurable source file with no record and the
  gate reds on every test file. Measurable ⟺ source ∧ ¬test.
- **The go-toolchain pack must declare globs or the dogfood regresses.** Once the baked
  `.go` literal is deleted, a `.go` file is measurable ONLY if a declared pack supplies
  `**/*.go`. The `backstop/go-toolchain` pack (its own repo; `.backstop/packs/` is
  gitignored — see [[packs_always_external]]) must add the `classification` block, and
  the in-repo dogfood copy must carry it, or backstop-core's own coverage gate stops
  classifying `.go`. Flag this as a wiring dependency for Seed 5 / the dogfood lock.
- **"No source globs declared" must be visible, not silent — but not a hard block.**
  Blocking outright would break every repo that declares a non-toolchain pack with no
  classification. The state is a DISTINCT visible warn (REQ-004), honoring
  loud≠blocking ([[feedback_loud_not_blocking]]). The trap is returning an unqualified
  `pass` (silent green) — explicitly prohibited.
- **Shared-file signature drift with SPEC-044.** Both specs change
  `StepCoverageThresholdScopedFunc`. If the parameter order/record-model evolves in
  SPEC-044 while this spec adds `classifier`, the two can collide. The guard here is
  framed as "no record for the path at all" so it is independent of SPEC-044's
  `(path, metric)` index — but the final signature is co-owned and MUST be reconciled
  in the cross-consistency pass.
- **Glob matcher semantics must be `**`-aware.** Go's stdlib `path.Match`/`filepath.Match`
  do NOT support `**` crossing separators. Using stdlib match would silently make
  `**/*.ts` behave like `*.ts` and miss nested files — re-opening the vacuous-green hole
  by under-matching. A doublestar-capable matcher is required.
- **Path normalization parity.** The classifier matches on the project-relative slash
  path; it must use the same `normalizeScopePath` form the scope and record index
  already use, or a changed file's scope path won't match its glob and the guard won't
  fire. Mismatched normalization is a silent under-match.

## Review Questions

- Does the no-record guard fire when NO numeric coverage threshold is declared in
  scope, or is it still gated behind the `threshold <= 0` early return? (REQ-003 — the
  exact vacuous-green vector.)
- Is `coverageMeasurablePath` (and every baked `.go`/`_test.go`/`testdata` string
  literal) actually DELETED from `pkg/gate/step_coverage.go`, not merely bypassed?
  (REQ-002/CLM-011.)
- Is the no-record predicate "no record for the path AT ALL (any metric)", so it
  composes with SPEC-044's `(path, metric)` index and does not silently swallow a
  present-but-wrong-metric record? (REQ-003/CLM-017 — the seam.)
- Does the LIVE gate (not just a unit test) build the merged classifier from the union
  of bridged + declared toolchain packs and consume it in the coverage step? Is there
  an end-to-end test proving a changed `.ts` file reds the real gate? (REQ-005/CLM-020.)
- When no toolchain pack declares source globs, does the step return a DISTINCT visible
  state, or an unqualified `pass`? (REQ-004 — silent green is prohibited.)
- Does the glob matcher support `**` crossing separators (doublestar), and does it
  match on the same normalized project-relative path the scope/record index use?
  (Sharp edges — under-matching re-opens the hole.)

## References

- BUNDLE-012 (`language-neutral-consumer-ts-toolchain`) — this is Seed 1, the
  foundation contract; REQ-001 is the bundle requirement this spec implements.
- SPEC-044 (`multi-metric-coverage-records`) — the parallel sibling owning the coverage
  RECORD model; shares `pkg/gate/step_coverage.go`. The seam is defined above.
- SPEC-041 / SPEC-042 — the BUNDLE-011 coverage consumer/producer (`check.CoverageRecord`,
  the per-file threshold + exclusion-loudness logic this spec extends, not replaces).
- [[feedback_loud_not_blocking]] — governs REQ-003/REQ-004: "nothing measured" must be
  loud, never a silent green; capability-absent is loud but non-blocking.
- [[feedback_zero_baked_checks]] / DD-1 — the thin-executor first principle: the
  consumer reads pack-declared globs as DATA; zero baked language knowledge.
- [[feedback_integration_gap]] — REQ-005's end-to-end wiring guard against a correct
  unit never reaching the live gate.
- [[packs_always_external]] — the go-toolchain/bun-toolchain packs live in their own
  repos; the classification block must land there (and in the gitignored dogfood copy).
- Code (this branch): `pkg/gate/step_coverage.go` `coverageMeasurablePath` (~L232),
  `coveragePathsInScope` (~L205), `StepCoverageThresholdScopedFunc` (~L53);
  `cmd/backstop/gate.go` `buildCoverageStep` (~L987); `pkg/pack/manifest.go` `Manifest`
  (~L14); reference `.backstop/packs/backstop/go-toolchain/pack.yml`.
