---
title: "Pack-Declared File-Classification Globs + De-Go'd Coverage Measurable-Path"
number: SPEC-043
created: "2026-06-28"
status: implemented
schema_version: spec/v1
spec_version: 1.1.0

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
      Classification and NO parse error. Glob semantics MUST be TRUE doublestar with
      ZERO-LEADING-SEGMENT matching: `**` crosses path separators AND matches zero
      directories, so `**/*.go` MUST match a repo-ROOT file (e.g. `embed.go`) as well
      as a nested `dir/x.go`; `*` matches within a single segment; matching is on the
      project-relative slash path. The matcher MUST be `github.com/bmatcuk/doublestar`
      (whose `**/*.go` matches root files) — NOT `github.com/gobwas/glob`, whose
      `**/*.go` requires a leading directory segment and SILENTLY DROPS repo-root
      source files (re-opening the exact vacuous-green hole this spec closes). The
      contract is DATA — backstop bakes no language-specific source/test convention;
      every stack supplies its own globs (DD-1). The reference shape is the
      `backstop/go-toolchain` pack declaring `source: ["**/*.go"]`,
      `test: ["**/*_test.go", "**/testdata/**"]` — and under the mandated
      zero-leading-segment semantics this single `**/*.go` source glob measures the
      live repo-root `embed.go`, no separate `*.go` entry needed.
    supports: language-neutral-consumer-ts-toolchain:REQ-001@1.0.0
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
      PROHIBITED for the CLASSIFICATION implementation — `coverageMeasurablePath`'s
      successor `SourceClassifier.IsMeasurableSource` in the NEW `pkg/gate/
      classification.go` — to retain any baked `.go`, `_test.go`, or `testdata` string
      literal: with a classifier that declares only non-Go globs, a `.go` file MUST NOT
      be measurable, proving the baked Go literal is gone (DD-1). This prohibition is
      SCOPED to the classification implementation, NOT package-wide `pkg/gate`: the
      sibling relevance helpers `coverageSpecRelevantToFile` / `packagePathMatches` in
      `step_coverage.go` legitimately RETAIN their `.go` / `./...` literals (the spec
      test-command/package-relevance model) until SPEC-045 (Seed 2) de-Go's
      test-verification discovery — they are out of scope here. Files matching only a
      test glob, or matching no declared glob, are NOT measurable.
    supports: language-neutral-consumer-ts-toolchain:REQ-001@1.0.0
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
      number. CONCRETELY, this requires DISMANTLING the live early-return short-circuit
      `if threshold <= 0 { return pass }` (`pkg/gate/step_coverage.go:56-57`): the
      measurable-source no-record scan MUST run BEFORE/INDEPENDENT of any numeric
      threshold resolution, so the guard is reached even when `coverageFloorForScope`
      yields no positive threshold. This restructure is a STATED DELIVERABLE of this
      spec (not merely an internal detail). NOTE: SPEC-044's REQ-003/REQ-005 (per-metric
      thresholds) depend on this SAME early-return being dismantled — a spec that
      declares only a per-metric threshold (no scalar floor) must NOT vacuously-pass
      through the old `threshold <= 0` exit; the two specs share the restructured
      step entry. SEAM: the guard keys on "no record for the path at all" (ANY metric); a
      path carrying at least one record (any metric) is NOT a no-record error — the
      PER-METRIC threshold verdict for a present record is owned by SPEC-044 and is
      out of scope here. A changed file classified as TEST (or unclassified) is NOT
      subject to this guard. A pack-declared-excluded path is skipped from the guard
      (its exclusion loudness is SPEC-041 behavior, unchanged).
    supports: language-neutral-consumer-ts-toolchain:REQ-001@1.0.0
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
      are declared and in-scope changed files exist. This state MUST follow the
      EXISTING capability-absent convention, not invent a new status string: reuse the
      shape `pkg/gate/traceability_polarity.go` `ClassCapabilityAbsent` /
      `PolarityStepResult` already emit — a `warning`-status `StepResult` carrying a
      `<dim>_capability_absent`-style Violation (Severity `warning`, ConfigErr false).
      Emit it as a `coverage`-dimension capability-absent warning in that same shape so
      the report renders it consistently with the traceability-polarity absent states.
    supports: language-neutral-consumer-ts-toolchain:REQ-001@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      The merged classifier MUST be threaded into the LIVE gate coverage step. The
      REAL wiring SEAM is `buildGateSteps` (the caller of `buildCoverageStep` in
      `cmd/backstop/gate.go`): `buildGateSteps` already has the resolved `packs`
      (`loadInstalledPacks` over `backstop.yml packs:`) in hand, calls
      `mergeSourceClassifier(packs)` there, and passes the resulting classifier through
      `buildCoverageStep` into `StepCoverageThresholdScopedFunc`. The classifier is
      built SOLELY from the DECLARED packs — and `mergeSourceClassifier` MUST be given
      the FULL declared-manifest set (every `*pack.Manifest` `loadInstalledPacks`
      returns), NOT a pre-filtered toolchain-only subset: a manifest with no
      `classification:` block contributes a zero-value Classification (empty globs =
      ZERO contribution to the union), so no toolchain-pack filter is needed or wanted.
      There is NO separate `bridged` toolchain-pack input: a toolchain is just an
      ordinary declared pack (SPEC-046 / Seed 3 DELETES the `language:`-derived bridge
      `loadBridgedToolchainPacks`/`toolchainPackName`, so the `bridged` set ceases to
      exist), and `mergeSourceClassifier` MUST take NO `bridged` argument so it is NOT
      orphaned when that bridge is removed. This MUST be proven END-TO-END by driving
      the FULL gate assembly path (`buildGateSteps`/`runGate`) so the
      `mergeSourceClassifier(packs)` call site itself is on the proven path — NOT by a
      test that hand-merges a classifier and calls `buildCoverageStep` (or
      `StepCoverageThresholdScopedFunc`) directly, which would leave the live
      `mergeSourceClassifier` call unexercised (the exact integration gap that bit
      SPEC-035/037). The merge MUST be a UNION across all declared toolchain packs (two
      declared toolchain packs contribute both glob sets to the live step).
    supports: language-neutral-consumer-ts-toolchain:REQ-001@1.0.0

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
    text: The CLASSIFICATION implementation (`pkg/gate/classification.go` `SourceClassifier.IsMeasurableSource`) contains NO baked `.go`/`_test.go`/`testdata` string literal — a source guard scoped to `classification.go` (NOT package-wide pkg/gate, since `step_coverage.go`'s `coverageSpecRelevantToFile`/`packagePathMatches` legitimately keep `.go`/`./...` until SPEC-045) asserts the classifier keys only on the declared globs, so a reintroduced baked extension fails
    tests:
      - TestClassifier_NoBakedExtensionLiteralsInClassificationGo
  - id: CLM-023
    requirement: REQ-002
    text: ZERO-LEADING-SEGMENT — under the declared source glob `**/*.go`, a repo-ROOT source file (e.g. `embed.go`, which exists live at backstop-core's root) IS measurable (the `**` matches zero directories), AND a repo-ROOT test file `foo_test.go` matches the declared test glob `**/*_test.go` so it is NOT measurable; this proves the matcher does not silently drop root-level source files (the gobwas under-match regression)
    tests:
      - TestClassifier_RootFileMeasurableUnderDoublestarSourceGlob
      - TestClassifier_RootTestFileMatchesTestGlobNotMeasurable
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
    text: END-TO-END over the REAL wiring seam — the test drives the FULL gate assembly (`buildGateSteps`/`runGate`), so the live `mergeSourceClassifier(packs)` call site in `buildGateSteps` is exercised (NOT a hand-merged classifier passed to `buildCoverageStep`/`StepCoverageThresholdScopedFunc` directly); with a declared toolchain pack whose source globs cover `.ts`, a changed `.ts` file with no record REDs the real assembled gate, proving the classifier is actually built-and-consumed on the production path (the SPEC-035/037 integration-gap closure)
    tests:
      - TestGate_BuildGateStepsConsumesMergedClassifierEndToEnd
  - id: CLM-021
    requirement: REQ-005
    text: The live merge is a UNION across declared toolchain packs — with two toolchain packs declared (go + bun), the coverage step measures BOTH a changed `.go` and a changed `.ts` file from the merged glob set
    tests:
      - TestGate_ClassifierMergesAcrossDeclaredToolchainPacks
  - id: CLM-022
    requirement: REQ-005
    text: NOT ORPHANED BY THE BRIDGE DELETION — `mergeSourceClassifier` builds the classifier from the FULL DECLARED-pack manifest set (`loadInstalledPacks` over `backstop.yml packs:`, passed wholesale with NO toolchain-only pre-filter — a manifest with no `classification:` block contributes empty globs = zero to the union), taking NO `bridged` argument; given declared manifests where only a toolchain pack carries source globs it produces a classifier whose source globs measure the in-scope changed files, proving the classifier survives SPEC-046's removal of the `language:`-derived bridge
    tests:
      - TestGate_MergeSourceClassifierSourcesFromDeclaredPacksNotBridge

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
        notes: "NEW (REQ-002): the language-neutral merged classifier the coverage consumer reads instead of a baked `.go` literal. Holds the UNION of declared source AND test globs (compiled doublestar matchers) — it MUST store BOTH sets, not just source. The stored TEST-glob set is load-bearing for SPEC-045 (Seed 2), which adds `IsTestFile`/`HasTestGlobs` reading exactly this test set; dropping it would orphan SPEC-045. Carries no language knowledge — it is data + match logic only."
      - name: NewSourceClassifier
        kind: function
        signature: "func NewSourceClassifier(source, test []string) SourceClassifier"
        notes: "NEW (REQ-002/CLM-008): constructs a classifier from MERGED source/test glob lists (the union across all declared toolchain packs is assembled by the caller in cmd/backstop and passed here). Stores and compiles BOTH the source AND test glob sets using github.com/bmatcuk/doublestar (TRUE doublestar, zero-leading-segment: `**/*.go` matches a repo-ROOT file as well as nested — CLM-010/CLM-023; NOT gobwas/glob, which drops root files); the test set is retained on the struct so SPEC-045 can read it via IsTestFile/HasTestGlobs."
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
        notes: "MODIFIED (REQ-002/REQ-003/REQ-004): this 4-arg form — adding a trailing `classifier SourceClassifier` parameter — is the CANONICAL signature for the shared `pkg/gate/step_coverage.go` step. SPEC-043 OWNS threading the classifier in; SPEC-044 ADOPTS this exact 4-arg signature and owns the internal `(path, metric)` record model + per-metric inner loop (it does NOT keep the prior 3-arg form). The in-scope measurable set is derived from classifier.IsMeasurableSource instead of the baked coverageMeasurablePath (REQ-002). The existing `if threshold <= 0 { return pass }` early-return (step_coverage.go:56-57) MUST be DISMANTLED so the no-record scan runs before/independent of threshold resolution (REQ-003) — SPEC-044's per-metric path shares this restructured entry. An in-scope changed measurable-source path with NO record for the path AT ALL (any metric) and not pack-declared excluded is a LOUD blocking error under a DISTINCT rule (e.g. coverage_unmeasured), DISTINCT from below-threshold, fired even when no numeric threshold is in scope (REQ-003/CLM-012..CLM-017/CLM-023). When classifier.HasSourceGlobs() is false and in-scope changed files exist, the step returns a DISTINCT non-blocking 'classification capability absent' status following the EXISTING capability-absent convention (the warning-status `<dim>_capability_absent` shape PolarityStepResult/ClassCapabilityAbsent already emit), never an unqualified pass (REQ-004/CLM-018/CLM-019). SEAM: the no-record predicate is 'any record for the path' so it composes with SPEC-044's (path, metric) index; per-metric threshold verdict is SPEC-044's. The existing SPEC-041 exclusion-loudness and per-file threshold semantics are retained."
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
        signature: "func mergeSourceClassifier(packs []*pack.Manifest) gate.SourceClassifier"
        notes: "NEW (REQ-005/CLM-021/CLM-022): unions the classification.source and classification.test globs across the DECLARED packs into one gate.SourceClassifier. Takes the FULL `[]*pack.Manifest` set `loadInstalledPacks` resolves over `backstop.yml packs:` — NOT a toolchain-only pre-filter: a manifest with no `classification:` block contributes empty globs (zero to the union), so no filter is needed. Consumes the declared-pack manifest set ONLY — it takes NO `bridged` argument, so it is NOT orphaned when SPEC-046 deletes the `language:`-derived bridge (`loadBridgedToolchainPacks`/`toolchainPackName`). Built where the manifests are visible (cmd/backstop) so pkg/gate takes no pkg/pack dependency. Called from buildGateSteps (the live wiring seam)."
      - name: buildCoverageStep
        kind: function
        signature: "func buildCoverageStep(specDir, projectRoot string, scope *gate.GateScope, classifier gate.SourceClassifier, records coverageRecordsFn) gate.StepFunc"
        notes: "MODIFIED (REQ-005/CLM-020): gains the merged classifier and threads it into StepCoverageThresholdScopedFunc so the LIVE gate consumes the declared globs end-to-end (closing the integration gap). The classifier is built by the caller buildGateSteps via mergeSourceClassifier(packs), where `packs` is the FULL DECLARED manifest set (loadInstalledPacks over backstop.yml) — no `bridged` set is passed. The mandated e2e test drives buildGateSteps/runGate (not buildCoverageStep directly) so the live mergeSourceClassifier call is exercised."
      - name: buildGateSteps
        kind: function
        signature: "func buildGateSteps(projectRoot string, scope ...*gate.GateScope) []gate.StepFunc"
        notes: "MODIFIED (REQ-005/CLM-020): the LIVE wiring SEAM. It already resolves `packs, _ := loadInstalledPacks(projectRoot)`; it now calls `mergeSourceClassifier(packs)` (full declared-manifest set, no toolchain-only filter, no `bridged` arg) and passes the resulting gate.SourceClassifier through buildCoverageStep into StepCoverageThresholdScopedFunc. This call site is what the REQ-005 end-to-end test MUST exercise (via buildGateSteps/runGate), closing the SPEC-035/037 integration gap where a correct unit never reaches the assembled gate."
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
- **Glob semantics:** TRUE doublestar with **zero-leading-segment** matching — `**`
  crosses path separators **and matches zero directories**, so `**/*.go` matches a
  repo-**root** file (`embed.go`) as well as a nested `dir/x.go`; `*` matches within a
  single segment; matching is on the project-relative slash path. The mandated matcher
  is `github.com/bmatcuk/doublestar` — **not** `github.com/gobwas/glob`, whose `**/*.go`
  requires a leading directory segment and silently drops repo-root source files (the
  exact under-match this spec forbids; see Sharp Edges).

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
doublestar/segment-aware (CLM-010); a repo-**root** source file (`embed.go`) is
measurable under `**/*.go` via zero-leading-segment matching while a root `foo_test.go`
matches the test glob and is not (CLM-023); and the **classification implementation**
(`pkg/gate/classification.go`, scoped — *not* package-wide `pkg/gate`, since
`step_coverage.go`'s `coverageSpecRelevantToFile`/`packagePathMatches` keep their
`.go`/`./...` literals until SPEC-045) holds **no** baked `.go`/`_test.go`/`testdata`
string literal (CLM-011).

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
or thresholding.

**Canonical signature (cross-consistency resolution).** The shared
`StepCoverageThresholdScopedFunc` takes the **4-arg form** with a trailing
`classifier SourceClassifier` parameter:

```go
func StepCoverageThresholdScopedFunc(
    coverage []check.CoverageRecord, specs []SpecVerification,
    scope *GateScope, classifier SourceClassifier) StepFunc
```

This 4-arg classifier-param form is **canonical**. **SPEC-043 (this spec) owns
threading the classifier in** (the measurable PATH SET + the no-record guard);
**SPEC-044 adopts this exact signature** and owns the internal `(path, metric)` record
model and per-metric inner loop. SPEC-044 does **not** retain the prior 3-arg form — the
two compose on one signature: 043 contributes the `classifier` parameter, 044 contributes
the record-consumption internals behind it.

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
   test glob — test-wins-on-overlap), and `HasSourceGlobs() bool`. Use
   `github.com/bmatcuk/doublestar` (add it to the module graph) — its `**` matches
   **zero or more** directories, so `**/*.go` matches a repo-root `embed.go` as well as
   nested files. Do **NOT** use `github.com/gobwas/glob`: its `**/*.go` requires a
   leading segment and silently drops root files (CLM-023's regression). Compile each
   glob once at construction; match on the project-relative slash path.

3. **De-Go the measurable-path (REQ-002).** DELETE `coverageMeasurablePath` (the baked
   `.go`/`_test.go`/`testdata` literals) from `pkg/gate/step_coverage.go`. Re-key
   `coveragePathsInScope` to filter in-scope changed files via
   `classifier.IsMeasurableSource`. The no-baked-literal guard (CLM-011) is scoped to
   the NEW `pkg/gate/classification.go`; the sibling helpers `coverageSpecRelevantToFile`
   / `packagePathMatches` in `step_coverage.go` KEEP their `.go` / `./...` literals (the
   spec test-command/package-relevance model) — de-Go'ing those is SPEC-045 (Seed 2),
   out of scope here. Do not delete or pressure-delete them.

4. **The anti-vacuous-green guard (REQ-003) — DISMANTLE the early return.** Re-shape
   `StepCoverageThresholdScopedFunc` (new `classifier SourceClassifier` parameter). The
   stated deliverable: **REMOVE the `if threshold <= 0 { return pass }` short-circuit**
   at `step_coverage.go:56-57` and restructure so the measurable-source no-record scan
   runs **before/independent of** any numeric threshold resolution. An in-scope changed
   measurable-source path with no record for the path **at all** and not pack-declared
   excluded emits a loud blocking violation under a DISTINCT rule (e.g.
   `coverage_unmeasured`, so it is not conflated with below-threshold). The "any record
   present for the path" predicate is the seam point — it asks only whether *some*
   record exists for the path, leaving per-metric verdicts to SPEC-044. **Shared-step
   note:** SPEC-044's per-metric work depends on this SAME early-return removal (a
   per-metric-only spec must not slip through the old `threshold <= 0` exit); both specs
   compose on the restructured entry. The existing SPEC-041 exclusion-loudness and
   per-file threshold behavior are retained.

5. **Classification-capability-absent state (REQ-004).** When
   `classifier.HasSourceGlobs()` is false and in-scope changed files exist, return a
   DISTINCT non-blocking status/reason (unqualified `pass` is PROHIBITED here). Reuse
   the EXISTING capability-absent convention rather than inventing a status string:
   `pkg/gate/traceability_polarity.go` `ClassCapabilityAbsent` / `PolarityStepResult`
   already emit a `warning`-status `StepResult` with a `<dim>_capability_absent`
   Violation (Severity `warning`, ConfigErr false). Emit the no-source-globs state in
   that same shape (a `coverage`-dimension capability-absent warning) so the report
   renders it consistently and the inability to classify is visible, never a silent
   green.

6. **Wire the merged classifier into the live gate (REQ-005).** Add
   `mergeSourceClassifier(packs []*pack.Manifest) gate.SourceClassifier` in
   `cmd/backstop/gate.go` that unions the source/test globs across the **declared**
   packs. The **call site is `buildGateSteps`** (`cmd/backstop/gate.go:575`, the caller
   of `buildCoverageStep`): it already holds `packs, _ := loadInstalledPacks(...)`, so
   call `mergeSourceClassifier(packs)` there and thread the classifier through
   `buildCoverageStep` into `StepCoverageThresholdScopedFunc`. Pass the **FULL** `packs`
   set — NOT a toolchain-only pre-filter — because a manifest with no `classification:`
   block contributes empty globs (zero to the union), so no filter is needed. It takes
   **no `bridged` argument**: a toolchain is just an ordinary declared pack, and SPEC-046
   (Seed 3) deletes the `language:`-derived bridge
   (`loadBridgedToolchainPacks`/`toolchainPackName`), so consuming the declared-pack
   manifest set is what keeps `mergeSourceClassifier` from being orphaned by that
   deletion (CLM-022). Prove end-to-end by driving the **full gate assembly**
   (`buildGateSteps`/`runGate`) so the live `mergeSourceClassifier(packs)` call is on the
   exercised path (NOT a hand-merged classifier handed to `buildCoverageStep` directly —
   the SPEC-035/037 integration gap): the assembled gate measures a changed non-Go file
   via the declared globs (CLM-020) and unions across multiple declared toolchain packs
   (CLM-021).

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
  measured or loudly fail." The current code returns `pass` at the `threshold <= 0`
  short-circuit (`step_coverage.go:56-57`), so a TS project with no spec-declared
  threshold passes vacuous-green. **Dismantling that early return is a STATED
  DELIVERABLE of REQ-003** (not an internal detail): the no-record scan must run before
  any threshold resolution. The same removal is load-bearing for SPEC-044 (a
  per-metric-only spec must not slip through the old exit) — both specs share the
  restructured step entry.
- **Test-wins-on-overlap is load-bearing.** A test glob is normally a more-specific
  subset of a source glob (`**/*_test.go` ⊂ `**/*.go`, `**/*.test.ts` ⊂ `**/*.ts`). If
  source-wins, every test file becomes a measurable source file with no record and the
  gate reds on every test file. Measurable ⟺ source ∧ ¬test.
- **go-toolchain `pack.yml` lockstep — deleting the baked `.go` literal and adding the
  `classification` block MUST land together, or the dogfood regresses.** Once the baked
  `.go` literal is deleted from `coverageMeasurablePath`, a `.go` file is measurable
  ONLY if a declared pack supplies `**/*.go`. So the SAME change set that deletes the
  literal MUST add the `classification` block (`source: ["**/*.go"]`,
  `test: ["**/*_test.go", "**/testdata/**"]`) to `backstop/go-toolchain`'s `pack.yml`.
  This regression is NOT silent — with no source globs declared, REQ-004's
  `HasSourceGlobs() == false` fires the LOUD `coverage`-dimension capability-absent
  warn rather than a vacuous green — but it IS a real dogfood regression: backstop-core
  stops actually measuring its own `.go` coverage and downgrades to a capability-absent
  advisory. The lockstep keeps the dogfood measuring, not merely non-silent.
  **Durability:** the authoritative
  edit must land in the `backstop/go-toolchain` pack's OWN repo (`.backstop/packs/` is
  gitignored — editing the installed copy is non-durable; see [[packs_always_external]]
  and [[project_pack_distribution]]); the gitignored in-repo dogfood copy must also
  carry it so the local gate stays green until the pack-repo edit is pulled. Flag this
  as a wiring dependency for Seed 5 / the dogfood lock.
- **"No source globs declared" must be visible, not silent — but not a hard block.**
  Blocking outright would break every repo that declares a non-toolchain pack with no
  classification. The state is a DISTINCT visible warn (REQ-004), honoring
  loud≠blocking ([[feedback_loud_not_blocking]]). The trap is returning an unqualified
  `pass` (silent green) — explicitly prohibited.
- **Shared-file signature drift with SPEC-044 (resolved).** Both specs change
  `StepCoverageThresholdScopedFunc`. The cross-consistency resolution fixes the
  **4-arg classifier-param form as canonical**: this spec owns threading the
  `classifier SourceClassifier` parameter in; SPEC-044 adopts that exact signature and
  owns the `(path, metric)` record model + per-metric inner loop behind it. The guard
  here is framed as "no record for the path at all" so it is independent of SPEC-044's
  `(path, metric)` index. The trap is a planner regenerating the prior 3-arg form for
  044 — they must compose on the one 4-arg signature, not fork it.
- **`mergeSourceClassifier` must source from DECLARED packs, or it is orphaned by
  SPEC-046.** SPEC-046 (Seed 3) DELETES the `language:`-derived toolchain bridge
  (`loadBridgedToolchainPacks`/`toolchainPackName`), so the `bridged` toolchain-pack
  set ceases to exist. If `mergeSourceClassifier` were built from that `bridged` input
  it would compile against a deleted symbol and the live coverage classifier would go
  dark. It MUST consume the declared-pack manifest set (`loadInstalledPacks` over
  `backstop.yml packs:`) — a toolchain is just an ordinary declared pack (the bundle
  thesis). CLM-022 pins this: the merge takes no `bridged` argument and produces a
  working classifier from declared manifests alone.
- **The classifier MUST store the test-glob set, not just source.** `SourceClassifier`
  holds BOTH the source and test glob sets. SPEC-045 (Seed 2) adds `IsTestFile`/
  `HasTestGlobs` reading exactly the stored test set; if this spec dropped the test
  globs after computing `IsMeasurableSource` (e.g. kept only a precomputed source
  matcher), SPEC-045 would have nothing to read and would be orphaned.
  `NewSourceClassifier(source, test)` retains both sets on the struct.
- **Glob matcher must be TRUE doublestar with zero-leading-segment matching — use
  `bmatcuk/doublestar`, NOT `gobwas/glob`.** Two distinct under-match traps re-open the
  vacuous-green hole. (1) Go's stdlib `path.Match`/`filepath.Match` do NOT support `**`
  crossing separators — `**/*.ts` would behave like `*.ts` and miss nested files. (2)
  Subtler and the real regression caught in review: `github.com/gobwas/glob`'s `**/*.go`
  requires a LEADING directory segment, so it matches `dir/x.go` but SILENTLY DROPS a
  repo-ROOT file like the live `embed.go` — root source files fall out of the measurable
  set, the exact vacuous-green this spec condemns. `github.com/bmatcuk/doublestar` treats
  `**` as "zero or more directories," so `**/*.go` matches both root and nested files.
  The spec MANDATES `bmatcuk/doublestar`; CLM-023 pins the root-file behavior with a test
  over `embed.go`.
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
- Does the end-to-end test drive the FULL gate assembly (`buildGateSteps`/`runGate`) so
  the live `mergeSourceClassifier(packs)` call site is exercised — rather than
  hand-merging a classifier and calling `buildCoverageStep`/`StepCoverageThresholdScopedFunc`
  directly (the SPEC-035/037 integration gap)? Does it build from the FULL declared
  manifest set (`loadInstalledPacks`, no toolchain-only pre-filter, NO `bridged` input)
  and prove a changed `.ts` file reds the real gate? (REQ-005/CLM-020/CLM-022.)
- Does `mergeSourceClassifier` source from the declared-pack manifest set rather than
  the `language:`-derived bridge, so it does not reference the symbols SPEC-046 deletes
  (`loadBridgedToolchainPacks`/`toolchainPackName`)? (REQ-005/CLM-022 — the SPEC-046 seam.)
- Does `SourceClassifier` retain the test-glob set (not only a source matcher), so
  SPEC-045 can add `IsTestFile`/`HasTestGlobs` over it? (Contract — the SPEC-045 seam.)
- When no toolchain pack declares source globs, does the step return a DISTINCT visible
  state, or an unqualified `pass`? (REQ-004 — silent green is prohibited.)
- Is the matcher `github.com/bmatcuk/doublestar` (NOT `gobwas/glob`), and does `**/*.go`
  match a repo-ROOT file like `embed.go` (zero-leading-segment), not only nested files?
  Is there a test asserting a root source file IS measurable and a root `_test.go` is
  not? (CLM-023 — gobwas silently drops root files, re-opening the hole.)
- Does the no-source-globs state reuse the EXISTING capability-absent convention (the
  `<dim>_capability_absent` warning shape from `PolarityStepResult`/`ClassCapabilityAbsent`),
  rather than inventing a new status string? (REQ-004 — NIT.)
- Does the glob matcher match on the same normalized project-relative path the
  scope/record index use? (Sharp edges — mismatched normalization is a silent under-match.)

## References

- BUNDLE-012 (`language-neutral-consumer-ts-toolchain`) — this is Seed 1, the
  foundation contract; REQ-001 is the bundle requirement this spec implements.
- SPEC-044 (`multi-metric-coverage-records`) — the parallel sibling owning the coverage
  RECORD model; shares `pkg/gate/step_coverage.go`. It ADOPTS this spec's canonical
  4-arg `StepCoverageThresholdScopedFunc` signature (classifier param). The seam is
  defined above.
- SPEC-045 (`de-go-test-verification-discovery`, Seed 2) — adds
  `SourceClassifier.IsTestFile`/`HasTestGlobs` reading the TEST-glob set this spec
  stores on the classifier; depends on the classifier staying in `pkg/gate` with its
  test set populated.
- SPEC-046 (`retire-language-toolchain-bridge`, Seed 3) — DELETES the `language:`-derived
  toolchain bridge (`loadBridgedToolchainPacks`/`toolchainPackName`), removing the
  `bridged` toolchain-pack set. This spec's `mergeSourceClassifier` consumes the
  DECLARED-pack manifest set (not `bridged`) so it survives that deletion (CLM-022).
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

## Version History

- **1.1.0** (2026-06-30) — Status → `implemented`. The BUNDLE-012 Seed 1 code shipped
  and passed impl-review PASS; the pack-declared `classification` globs block and the
  `pkg/gate` `SourceClassifier` (with the de-Go'd coverage measurable-path consumer) are
  live. No requirement, claim, or contract text changed — lifecycle transition only.
- **1.0.0** (2026-06-28) — Initial spec authored from BUNDLE-012 Seed 1.
