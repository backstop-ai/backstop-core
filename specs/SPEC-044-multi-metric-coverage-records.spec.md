---
title: "Multi Metric Coverage Records"
number: SPEC-044
created: "2026-06-28"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    BUNDLE-012 Spec Seed 4 (REQ-007, SQ-2) — the FOUNDATION record-model contract that
    Seed 5's `backstop/bun-toolchain` coverage engine (SPEC-047) targets. Today the
    coverage CONSUMER (`pkg/gate/step_coverage.go`) indexes coverage ONE-RECORD-PER-PATH:
    `indexCoverageByPath` builds `map[string]check.CoverageRecord` keyed by normalized
    path, so a file carrying TWO metrics (line AND branch) silently collapses — the second
    record overwrites the first. That is fine for the only producer that exists today
    (`go-toolchain` emits a SINGLE `statement` metric per file), but the bun coverage
    producer will emit BOTH `line` and `branch` per file, and they MUST coexist and be
    thresholded INDEPENDENTLY. This spec EVOLVES the SPEC-042 coverage record model on the
    CONSUMER side: re-key the index by `(path, metric)`, threshold each `(path, metric)`
    record independently, and RESOLVE the bundle's deferred SQ-2 — line vs branch get
    PER-METRIC declared thresholds (branch is normally held to a lower bar than line), with
    the existing scalar `coverage_threshold` as the DEFAULT for any metric without an
    explicit override. The canonical `check.CoverageRecord`
    (`{Path, Covered, Total, Measured, Excluded, Metric}`, raw counts) already carries the
    `Metric` label and is NOT changed — only the consumer's indexing and threshold
    selection evolve, plus a per-metric threshold surface on the spec verification block.
    The gate stays metric-BLIND in spirit: it computes `Covered/Total >= threshold` per
    `(path, metric)` from raw counts and uses the `Metric` label ONLY as (a) a map key to
    look up the declared per-metric threshold and (b) a report label — it NEVER orders,
    ranks, or semantically interprets a metric (it does not "know" branch < line).
    Backward-compat with the live `go-toolchain` `statement`-only producer is a HARD
    requirement: a single-`statement`-metric record set with no per-metric override must
    yield verdicts IDENTICAL to today's one-record-per-path behavior.
    SEAM with SPEC-043 (parallel sibling, shares `pkg/gate/step_coverage.go`): SPEC-043
    owns the pack-declared source/test glob contract, the in-scope measurable PATH SET, and
    the PATH-LEVEL "changed source file with NO record at all → loud error" guard. THIS
    spec owns the RECORD model: `(path, metric)` indexing, line+branch coexistence,
    per-metric thresholding, and the NARROWER metric-granular anti-vacuous-green guard
    ("a path HAS records but is missing an EXPLICITLY declared metric → loud error"), which
    is DISTINCT from SPEC-043's path-level guard and does not redefine it.
  package: pkg/gate

verification:
  level: unit
  test_command: go test ./pkg/gate/ -race -coverprofile=cover.out
  coverage_threshold: 90

requirements:
  - id: REQ-001
    text: >
      The coverage index MUST be keyed by `(path, metric)`, not by path alone. The
      one-record-per-path `indexCoverageByPath` (`map[string]check.CoverageRecord`) MUST be
      replaced by a `(path, metric)`-keyed structure
      (`map[string]map[string]check.CoverageRecord` — outer key the normalized
      project-relative path, inner key the `Metric` label) so a single file carrying
      MULTIPLE metrics (e.g. `line` AND `branch`) retains ALL of them with NO collision and
      NO last-write-wins overwrite. The path-normalization (`normalizeScopePath`) and the
      module/namespace-qualified suffix fallback (`resolveCoverageRecord`'s
      `HasSuffix("/"+path)` reconciliation, unique-match-required) MUST be preserved, now
      applied per `(path, metric)`. It is PROHIBITED for two records sharing the SAME
      `(path, metric)` to silently collapse: a duplicate `(path, metric)` from the producer
      is a LOUD blocking error (a producer defect — two measurements of the same file under
      the same metric), never a silent last-wins.
    supports: language-neutral-consumer-ts-toolchain:REQ-007
    follows: STD-GO-001:GO-010
  - id: REQ-002
    text: >
      For an in-scope path that HAS at least one record, EACH `(path, metric)` record MUST
      be thresholded INDEPENDENTLY. A single file MAY therefore produce MULTIPLE verdicts —
      e.g. `line` passes while `branch` fails — and a below-threshold metric MUST red the
      gate regardless of a PASSING sibling metric on the SAME file (no aggregation or
      sibling-metric rescue, mirroring the existing per-FILE no-aggregation rule, now at
      `(path, metric)` granularity). The verdict MUST be computed metric-BLIND from RAW
      COUNTS as `Covered*100 >= Total*threshold` per record (no floating-point, no
      pre-computed percent). The pre-existing per-record conventions MUST be preserved at
      `(path, metric)` granularity: a `Total == 0` record is N/A and is SKIPPED (never a
      0%-fail) for THAT metric while OTHER metrics on the same file are still thresholded; a
      pack-declared `Excluded` record is skipped from the threshold check and its
      suppression of an in-scope CHANGED file is loudly surfaced. The gate MUST use the
      `Metric` label ONLY as a threshold-lookup key and a report label — it is PROHIBITED to
      order, rank, compare, or otherwise semantically interpret a metric (the gate does not
      "know" branch is normally lower than line).
    supports: language-neutral-consumer-ts-toolchain:REQ-007
    follows: STD-GO-001:GO-010
  - id: REQ-003
    text: >
      SQ-2 IS RESOLVED HERE as PER-METRIC declared thresholds with a scalar default. The
      spec verification block MUST gain an OPTIONAL per-metric threshold surface
      `coverage_metric_thresholds` (a map of metric-label → integer threshold) carried onto
      `SpecVerification` as `MetricThresholds map[string]int`. For a given metric the
      applicable threshold is: the per-metric override for that metric if declared, ELSE the
      existing scalar `coverage_threshold` (the DEFAULT for any metric without an explicit
      override). The "strictest in-scope spec governs" rule MUST be preserved PER METRIC:
      when multiple in-scope specs apply, the governing threshold for a metric is the MAX of
      each spec's applicable (override-or-default) threshold for that metric. A metric whose
      resolved threshold is `<= 0` has NO threshold declared and its records are SKIPPED from
      the threshold check (preserving the existing "no threshold declared in scope ⇒ pass"
      behavior, now per metric). It is PROHIBITED for a per-metric override declared for
      metric M to alter the threshold applied to any OTHER metric N.
    supports: language-neutral-consumer-ts-toolchain:REQ-007
    follows: STD-GO-001:GO-010
  - id: REQ-004
    text: >
      Backward-compatibility with the live single-metric `go-toolchain` producer (which
      emits ONE `statement` metric per file — SPEC-042 REQ-007) MUST be preserved EXACTLY.
      A record set with one `statement` record per path consumed against a spec that
      declares ONLY the scalar `coverage_threshold` (no `coverage_metric_thresholds` map)
      MUST produce verdicts IDENTICAL to the pre-existing one-record-per-path behavior: a
      measured-and-passing `statement` file passes, a measured-and-below-threshold
      `statement` file reds, a `Total == 0` `statement` file is N/A. Such a record set MUST
      NOT trigger a `(path, metric)` collision (one metric per path) and MUST NOT trigger
      the declared-metric-not-measured guard (REQ-005), because no per-metric threshold is
      explicitly declared. The scalar-only declaration path MUST remain fully supported.
    supports: language-neutral-consumer-ts-toolchain:REQ-007
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      A metric-granular anti-vacuous-green guard MUST exist, DISTINCT from SPEC-043's
      path-level no-record guard. When a per-metric threshold is EXPLICITLY declared for a
      metric M (an entry in `coverage_metric_thresholds`) and an in-scope CHANGED path HAS
      at least one record but NONE with `Metric == M`, that MUST be a LOUD blocking error
      (a dedicated rule, e.g. `coverage_metric_missing`, severity error) — refusing to pass
      with a declared metric silently unmeasured — never a silent pass. This guard MUST fire
      ONLY for metrics that are EXPLICITLY declared (so a scalar-only spec, REQ-004, never
      triggers it) and ONLY for in-scope CHANGED paths (an unchanged/all-mode path missing a
      declared metric stays quiet, parallel to the existing exclusion-loudness scoping). It
      is PROHIBITED for this guard to redefine or duplicate SPEC-043's path-level guard:
      SPEC-043 owns "the path has ZERO records at all"; THIS guard owns "the path has
      records but is missing an explicitly-required metric." The two are different
      granularities and both are loud.
    supports: language-neutral-consumer-ts-toolchain:REQ-007
    follows: STD-GO-001:GO-010
  - id: REQ-006
    text: >
      This spec MUST define the RECORD-SHAPE contract the bun coverage producer (SPEC-047 /
      Seed 5) targets, so that later spec can be authored against it: for each measured
      source file the bun producer's lcov convert MUST emit TWO canonical
      `check.CoverageRecord`s sharing the SAME `Path` — one with `Metric: "line"` and one
      with `Metric: "branch"` — each carrying raw `{Covered, Total, Measured, Excluded}`
      counts. Both records MUST flow through the EXISTING canonical
      `check.CoverageRecord` type and the EXISTING `check.ParsePackCoverage` parser with NO
      new type, NO new field, and NO schema fork; the only consumer-side change is the
      `(path, metric)` indexing + per-metric thresholding this spec adds. A bun-shaped
      record set (line + branch per file) MUST index without collision and threshold each
      metric independently through the same consumer path the `go-toolchain` `statement`
      records flow through.
    supports: language-neutral-consumer-ts-toolchain:REQ-007
    follows: STD-GO-001:GO-010

claims:
  # REQ-001 — (path, metric)-keyed index; multi-metric coexistence; loud duplicate collision
  - id: CLM-001
    requirement: REQ-001
    text: A file carrying TWO records under the same Path with distinct metrics (line and branch) is indexed under (path, metric) so BOTH records are retained — neither overwrites the other; the index for that path exposes exactly two metric entries
    tests:
      - TestIndexByPathMetric_LineAndBranchCoexistNoCollision
  - id: CLM-002
    requirement: REQ-001
    text: A single-metric file (one statement record per path) indexes to exactly one (path, metric) entry — the backward-compatible shape, proving the (path, metric) index is a strict generalization of the old one-record-per-path index
    tests:
      - TestIndexByPathMetric_SingleMetricIndexesToOneEntry
  - id: CLM-003
    requirement: REQ-001
    text: Two records sharing the SAME (path, metric) is a LOUD blocking error (a duplicate-measurement producer defect), never a silent last-wins collapse — feeding two line records for the same path yields a coverage error, not one survivor
    tests:
      - TestIndexByPathMetric_DuplicatePathMetricFailsLoud
  - id: CLM-004
    requirement: REQ-001
    text: The module/namespace-qualified suffix fallback is preserved per (path, metric) — a record emitted under a qualified path (e.g. "github.com/org/repo/pkg/x/f.go") with metrics line and branch resolves for the repo-relative scope path "pkg/x/f.go" for EACH metric, and an ambiguous suffix is treated as no-match so the loud guards fire
    tests:
      - TestIndexByPathMetric_QualifiedPathSuffixResolvesPerMetric
  # REQ-002 — independent per-metric thresholding; no sibling-metric rescue; per-metric N/A; metric-blind ratio
  - id: CLM-005
    requirement: REQ-002
    text: A file carrying line 95/100 (line threshold 90, pass) AND branch 60/100 (branch threshold 70, fail) produces EXACTLY ONE violation — for the branch metric — and the passing line metric is NOT flagged and did NOT collide; the two metrics are thresholded independently and (path, metric)-keyed with no collision
    tests:
      - TestCoverage_SameFileLineAndBranchThresholdedIndependentlyNoCollision
  - id: CLM-006
    requirement: REQ-002
    text: A file whose line AND branch records BOTH pass their thresholds produces NO violation
    tests:
      - TestCoverage_BothMetricsPassNoViolation
  - id: CLM-007
    requirement: REQ-002
    text: A file whose line AND branch records BOTH fail their thresholds produces TWO violations — one per (path, metric) — proving each metric reds independently
    tests:
      - TestCoverage_BothMetricsFailTwoViolations
  - id: CLM-008
    requirement: REQ-002
    text: A below-threshold metric reds the gate even when a SIBLING metric on the SAME file passes — there is no aggregation or sibling-metric rescue (the per-(path,metric) no-aggregation rule)
    tests:
      - TestCoverage_BelowThresholdMetricNotRescuedByPassingSibling
  - id: CLM-009
    requirement: REQ-002
    text: A Total==0 record for ONE metric on a file is N/A (skipped, never a 0%-fail) while the OTHER metric on the same file is still thresholded normally — per-(path,metric) N/A
    tests:
      - TestCoverage_TotalZeroPerMetricIsNAOtherMetricStillThresholded
  - id: CLM-010
    requirement: REQ-002
    text: The gate stays metric-blind in the verdict computation — two records with DIFFERENT metric labels but IDENTICAL Covered/Total evaluated against the SAME applicable threshold produce IDENTICAL pass/fail verdicts; the metric label is never ordered, ranked, or semantically interpreted in the ratio
    tests:
      - TestCoverage_VerdictMetricBlindOnRawCounts
  # REQ-003 — per-metric declared thresholds (SQ-2) + scalar default + strictest-governs-per-metric
  - id: CLM-011
    requirement: REQ-003
    text: An explicit per-metric override (coverage_metric_thresholds {branch 70}) is applied to the branch record while the line record (no override) uses the scalar coverage_threshold default 90 — branch 65/100 fails against 70, line 92/100 passes against 90
    tests:
      - TestThreshold_PerMetricOverrideAppliedScalarDefaultForUnlisted
  - id: CLM-012
    requirement: REQ-003
    text: A metric with NO per-metric override falls back to the scalar coverage_threshold default — a branch record with no branch override is thresholded at the scalar value, identical to how the metric-blind scalar path behaves today
    tests:
      - TestThreshold_UnlistedMetricUsesScalarDefault
  - id: CLM-013
    requirement: REQ-003
    text: The strictest-in-scope-spec-governs rule is preserved PER METRIC — with two in-scope specs declaring different branch overrides (70 and 80), the governing branch threshold is the MAX (80); the per-metric selection takes the max of each spec's applicable override-or-default per metric
    tests:
      - TestThreshold_StrictestSpecGovernsPerMetric
  - id: CLM-014
    requirement: REQ-003
    text: A per-metric override declared for metric M does NOT alter the threshold applied to metric N — declaring branch 70 leaves the line threshold at the scalar default, proving overrides are isolated per metric
    tests:
      - TestThreshold_OverrideForOneMetricDoesNotAffectAnother
  - id: CLM-015
    requirement: REQ-003
    text: A metric whose resolved threshold is <= 0 (no scalar and no override declared in scope) is SKIPPED from the threshold check rather than failed — preserving the "no threshold declared in scope ⇒ pass" behavior at metric granularity
    tests:
      - TestThreshold_MetricWithNoDeclaredThresholdSkipped
  # REQ-004 — backward-compat with the single-statement-metric go-toolchain producer
  - id: CLM-016
    requirement: REQ-004
    text: A go-toolchain-shaped record set (one statement record per path) consumed against a scalar-only spec yields verdicts IDENTICAL to the pre-existing one-record-per-path behavior — a measured-and-passing statement file passes, a below-threshold statement file reds, a Total==0 statement file is N/A
    tests:
      - TestBackwardCompat_StatementOnlyRecordsIdenticalToScalarBehavior
  - id: CLM-017
    requirement: REQ-004
    text: A statement-only record set with no coverage_metric_thresholds map triggers NEITHER a (path, metric) collision (one metric per path) NOR the declared-metric-not-measured guard (no metric is explicitly declared) — the scalar-only path is free of the new guards
    tests:
      - TestBackwardCompat_StatementOnlyNoCollisionNoMetricMissingGuard
  # REQ-005 — metric-granular declared-metric-not-measured guard, distinct from SPEC-043's path-level guard
  - id: CLM-018
    requirement: REQ-005
    text: With branch explicitly declared in coverage_metric_thresholds, an in-scope CHANGED path that has a line record but NO branch record is a LOUD blocking coverage_metric_missing error — refusing to pass with the declared branch metric silently unmeasured
    tests:
      - TestMetricMissing_DeclaredBranchUnmeasuredOnChangedPathFailsLoud
  - id: CLM-019
    requirement: REQ-005
    text: With NO per-metric override declared (scalar-only spec), an in-scope changed path that has only a line record does NOT trigger the declared-metric-not-measured guard — the guard fires only for EXPLICITLY declared metrics, so the scalar path has no false positive
    tests:
      - TestMetricMissing_NoFalsePositiveWhenNoMetricDeclared
  - id: CLM-020
    requirement: REQ-005
    text: The metric-missing guard is DISTINCT from SPEC-043's path-level no-record guard — a path with ZERO records is the path-level guard's case (not coverage_metric_missing), while a path WITH records missing an explicitly-declared metric is coverage_metric_missing; the two rules are emitted distinctly and do not duplicate
    tests:
      - TestMetricMissing_DistinctFromPathLevelNoRecordGuard
  - id: CLM-021
    requirement: REQ-005
    text: An UNCHANGED / all-mode path missing an explicitly-declared metric stays QUIET (the metric-missing guard is scoped to in-scope CHANGED paths, parallel to the existing exclusion-loudness scoping) — it does not red an all-mode sweep
    tests:
      - TestMetricMissing_UnchangedPathMissingMetricStaysQuiet
  # REQ-006 — bun producer record-shape contract flows through the canonical type/parser unchanged
  - id: CLM-022
    requirement: REQ-006
    text: A bun-shaped record set (two records per file sharing one Path, metrics line and branch, raw counts) parses through the EXISTING check.ParsePackCoverage into two canonical check.CoverageRecords and indexes under (path, metric) without collision — no new type, no new field, no schema fork
    tests:
      - TestBunShape_LineAndBranchParseAndIndexThroughCanonicalType
  - id: CLM-023
    requirement: REQ-006
    text: A bun-shaped line+branch record set thresholds each metric independently through the SAME consumer path the go-toolchain statement records flow through — proving the consumer is producer-agnostic and the only change needed is (path, metric) indexing + per-metric thresholds
    tests:
      - TestBunShape_LineAndBranchThresholdedThroughSharedConsumerPath

contracts:
  - file: pkg/gate/step_coverage.go
    provides:
      - name: indexCoverageByPathMetric
        kind: function
        signature: "func indexCoverageByPathMetric(coverage []check.CoverageRecord) (map[string]map[string]check.CoverageRecord, []string)"
        notes: "REPLACES indexCoverageByPath (REQ-001/CLM-001..CLM-003). Builds a (path, metric)-keyed index: outer key the normalized project-relative path (via normalizeScopePath), inner key the Metric label, value the canonical record. A file carrying line AND branch retains BOTH (no last-wins). The second return value is the list of (path, metric) keys seen MORE THAN ONCE (duplicate-measurement producer defects) so the step can surface them as loud coverage errors rather than silently collapsing them. Preserves the path-normalization the old index applied."
      - name: resolveCoverageRecordsForPath
        kind: function
        signature: "func resolveCoverageRecordsForPath(byPathMetric map[string]map[string]check.CoverageRecord, path string) (map[string]check.CoverageRecord, bool)"
        notes: "REPLACES resolveCoverageRecord (REQ-001/CLM-004). Returns ALL metric records for a repo-relative scope path: exact-match first, else the unique module/namespace-qualified suffix fallback (HasSuffix(\"/\"+path), unique-match-required, separator-anchored) the old resolver used — now returning the whole per-metric map for that path. An ambiguous suffix (two qualified paths ending the same way) is no-match so the loud guards fire rather than silently picking one."
      - name: coverageThresholdForMetric
        kind: function
        signature: "func coverageThresholdForMetric(sel coverageThresholdSelection, metric string) int"
        notes: "NEW (REQ-003/CLM-011..CLM-015, SQ-2 resolution). Resolves the governing threshold for a single metric from the selected specs: for each spec, its applicable threshold for the metric is MetricThresholds[metric] if declared, else CoverageThreshold (scalar default); the governing value is the MAX across selected specs (strictest governs, per metric). Returns 0 when no threshold is declared for the metric in scope (the metric is then skipped). Generalizes coverageFloorForScope from a single floor to a per-metric floor."
      - name: SpecVerification
        kind: type
        signature: "type SpecVerification struct { SpecID string; TestCommand string; CoverageThreshold int; MetricThresholds map[string]int; File string; ImplementationPackage string }"
        notes: "EXTENDED (REQ-003): gains MetricThresholds map[string]int — the per-metric declared threshold surface (SQ-2). nil/empty means scalar-only (backward-compatible, REQ-004). A metric absent from the map uses CoverageThreshold as its default."
      - name: StepCoverageThresholdScopedFunc
        kind: function
        signature: "func StepCoverageThresholdScopedFunc(coverage []check.CoverageRecord, specs []SpecVerification, scope *GateScope, classifier SourceClassifier) StepFunc"
        notes: "EVOLVED (REQ-001/REQ-002/REQ-005). The 4-arg signature (with the trailing `classifier SourceClassifier` parameter) is the CANONICAL form OWNED BY SPEC-043, which adds that parameter and threads the classifier in. This spec ADOPTS that signature unchanged and does NOT own the classifier parameter, the measurable in-scope PATH SET, or the path-level no-record guard — those are SPEC-043's half. THIS spec owns only the INNER record model REACHED through that signature: it replaces the path-keyed index with the (path, metric) index via indexCoverageByPathMetric, resolves per-path metric records via resolveCoverageRecordsForPath, iterates EVERY metric record for each in-scope path, thresholds each independently via coverageThresholdForMetric (metric-blind Covered*100>=Total*threshold), preserves per-record Total==0 N/A and Excluded loud-surfacing at (path, metric) granularity, surfaces duplicate-(path,metric) collisions loudly, and emits the coverage_metric_missing guard for an explicitly-declared metric absent on an in-scope changed path that has records. SEAM: this spec's (path, metric) index and per-metric verdict loop MUST COMPOSE with SPEC-043's classifier-derived measurable set and its path-level 'zero records for the path (any metric)' guard — the two halves share this one function and must merge without either re-deriving the other's surface."
    consumes:
      - source: pkg/check
        name: CoverageRecord
        kind: type
  - file: pkg/gate/step_testverify.go
    provides:
      - name: ExtractSpecVerifications
        kind: function
        signature: "func ExtractSpecVerifications(specDir string) ([]SpecVerification, error)"
        notes: "EVOLVED (REQ-003): the spec-frontmatter Verification struct gains coverage_metric_thresholds (map[string]int, yaml: coverage_metric_thresholds); the extractor populates SpecVerification.MetricThresholds from it. The existing gate (TestCommand != \"\" && CoverageThreshold > 0) is loosened so a spec declaring per-metric thresholds without a scalar is still extracted. nil map preserves today's scalar-only behavior. CO-EDIT SEAM: SPEC-045 (de-go-test-verification-discovery) also edits this file (step_testverify.go) — replacing the baked funcPattern / collectTestFuncNamesScoped test-name discovery with a pack-declared TestNameMatcher. The two edits are disjoint in concern (frontmatter/extraction here; test-name discovery there) but must be reconciled on merge so neither clobbers the other; this loosened extraction gate must survive. Flagged for the cross-consistency pass."
    consumes: []
---

# SPEC-044: Multi-Metric Coverage Records

## Overview

This spec is **Seed 4 of BUNDLE-012** (language-neutral gate consumer + TypeScript/Bun
toolchain), owning **REQ-007** and **resolving the bundle's deferred sub-question SQ-2**. It
is a **FOUNDATION contract**: the `backstop/bun-toolchain` coverage engine (Seed 5 /
SPEC-047) emits BOTH `line` and `branch` coverage per file, and this spec evolves the
coverage CONSUMER record model so those two metrics **coexist on the same file and are
thresholded independently**.

**The problem.** Today `pkg/gate/step_coverage.go` indexes coverage **one record per
path**: `indexCoverageByPath` returns `map[string]check.CoverageRecord` keyed by normalized
path. That is correct for the only producer that exists — `go-toolchain` emits a SINGLE
`statement` metric per file (SPEC-042 REQ-007) — but the moment a producer emits two metrics
for one file, the second record **silently overwrites the first**. A bun producer emitting
`line` and `branch` for `src/foo.ts` would have one of them vanish, and there is no way to
hold `branch` to a different bar than `line`.

**What this spec changes (consumer-side only).** The canonical `check.CoverageRecord`
(`{Path, Covered, Total, Measured, Excluded, Metric}`, raw counts — SPEC-042) already carries
the `Metric` label and is **NOT modified**. This spec:

1. **Re-keys the index by `(path, metric)`** (REQ-001) so line + branch coexist with no
   collision, and makes a duplicate `(path, metric)` a **loud producer-defect error** rather
   than a silent last-wins.
2. **Thresholds each `(path, metric)` record independently** (REQ-002) — one file can yield a
   `line` pass and a `branch` fail at once; a below-threshold metric reds regardless of a
   passing sibling metric; the verdict stays metric-blind (`Covered*100 >= Total*threshold`
   from raw counts).
3. **Resolves SQ-2 as PER-METRIC declared thresholds with a scalar default** (REQ-003): the
   spec verification block gains an optional `coverage_metric_thresholds` map; the existing
   scalar `coverage_threshold` is the **default** for any metric without an override (so
   branch can be held to a lower bar than line). Strictest-in-scope-spec-governs is preserved
   per metric.
4. **Preserves go-toolchain backward-compat exactly** (REQ-004): single-`statement`-metric
   records + a scalar-only spec produce verdicts identical to today.
5. **Adds a metric-granular anti-vacuous-green guard** (REQ-005): an explicitly-declared
   metric that is unmeasured on an in-scope changed path that HAS records is a loud
   `coverage_metric_missing` error — distinct from SPEC-043's path-level no-record guard.
6. **Defines the bun producer record-shape contract** (REQ-006) so Seed 5 can target it:
   two canonical records per file (`line`, `branch`, same `Path`), flowing through the
   existing `check.CoverageRecord` + `check.ParsePackCoverage` with no new type.

### SQ-2 resolution (recorded)

> **SQ-2 — do line vs branch get the SAME threshold or PER-METRIC thresholds?**
> **RESOLVED: per-metric declared thresholds, with the scalar `coverage_threshold` as the
> default.** Branch coverage is normally held to a lower bar than line, so the gate supports
> a per-metric override surface (`coverage_metric_thresholds`) on the spec verification block.
> Any metric without an explicit override falls back to the scalar `coverage_threshold` — so a
> spec that declares only the scalar behaves exactly as today (REQ-004), and a spec that wants
> a looser branch bar declares `coverage_metric_thresholds: {branch: 70}`. The gate remains
> metric-blind: it uses the metric label ONLY as a threshold-lookup key and a report label, and
> never orders or ranks metrics (it does not "know" branch < line — the policy lives in the
> declared numbers, not in the binary).

### Seam with SPEC-043 (parallel sibling, shared file)

SPEC-043 and this spec both touch `pkg/gate/step_coverage.go` AND both reach the SAME function
`StepCoverageThresholdScopedFunc`. The ownership split:

| Concern | Owner |
| --- | --- |
| `StepCoverageThresholdScopedFunc` **signature** — the canonical 4-arg form `(coverage, specs, scope, classifier SourceClassifier)`; adds the trailing `classifier SourceClassifier` parameter | **SPEC-043** |
| Pack-declared source/test glob contract | **SPEC-043** |
| The in-scope measurable PATH SET (derived from `classifier.IsMeasurableSource`) | **SPEC-043** |
| Path-level guard: a changed source path with **NO record at all** (any metric) → loud error | **SPEC-043** |
| `(path, metric)` indexing; line+branch coexistence; duplicate-`(path,metric)` collision | **SPEC-044 (this spec)** |
| Per-metric independent thresholding + per-metric declared thresholds (SQ-2) | **SPEC-044 (this spec)** |
| Metric-granular guard: a path **with records** missing an **explicitly-declared** metric → loud error | **SPEC-044 (this spec)** |

**Function-signature seam (B1).** This spec does NOT keep `StepCoverageThresholdScopedFunc`'s
signature unchanged. SPEC-043 adds the trailing `classifier SourceClassifier` parameter and owns
threading the classifier in plus deriving the measurable in-scope path set from it. This spec
ADOPTS that 4-arg signature verbatim and owns only the INNER half reached through it: replacing
the path-keyed index with the `(path, metric)` index (`indexCoverageByPathMetric` /
`resolveCoverageRecordsForPath`), the per-metric threshold selection
(`coverageThresholdForMetric`), and the per-`(path, metric)` verdict loop. The two halves must
COMPOSE inside the one function — this spec's verdict loop iterates over the metric records of
each path SPEC-043's classifier admits to the measurable set; neither half re-derives the other's
surface.

The guards compose at **different granularities and never overlap**: SPEC-043's guard fires
when a path has **zero** records; this spec's `coverage_metric_missing` guard fires only when a
path **has** records but is missing a metric the spec **explicitly declared** a threshold for.
A scalar-only spec (no `coverage_metric_thresholds`) never triggers this spec's guard, so the
two guards cannot both fire on the same condition. This residual is flagged for the
cross-consistency pass (see Sharp Edges and the seam item at the end).

## Requirements

Requirements are enumerated in the `requirements:` frontmatter (REQ-001 … REQ-006), each
tracing to bundle `REQ-007` via `supports`. Summary:

| Spec REQ | What it commits to |
| --- | --- |
| REQ-001 | Index keyed by `(path, metric)`; line+branch coexist with no collision; the qualified-suffix fallback is preserved per metric; a duplicate `(path, metric)` is a LOUD error, never last-wins. |
| REQ-002 | Each `(path, metric)` record is thresholded INDEPENDENTLY (one file → multiple verdicts); a below-threshold metric reds despite a passing sibling (no aggregation rescue); `Total==0` and `Excluded` conventions preserved per `(path, metric)`; verdict metric-blind from raw counts. |
| REQ-003 | SQ-2 resolution — per-metric declared thresholds via `coverage_metric_thresholds`, with scalar `coverage_threshold` as the default; strictest-in-scope-spec-governs preserved PER METRIC; an undeclared (`<=0`) metric threshold is skipped, not failed; an override for M never alters N. |
| REQ-004 | Backward-compat: single-`statement`-metric records + scalar-only spec produce verdicts IDENTICAL to today; no collision, no metric-missing guard. |
| REQ-005 | Metric-granular anti-vacuous-green guard `coverage_metric_missing` (explicitly-declared metric unmeasured on an in-scope changed path that HAS records) — DISTINCT from SPEC-043's path-level no-record guard; fires only for explicitly-declared metrics and only on in-scope changed paths. |
| REQ-006 | Bun producer record-shape contract: two canonical records per file (`line`, `branch`, same `Path`), through the EXISTING `check.CoverageRecord` + `check.ParsePackCoverage`, no new type/field; indexes + thresholds through the same consumer path as `statement` records. |

### The threshold-resolution rule (REQ-003)

For an in-scope path, for each metric `M` present in that path's records, the applicable
threshold is resolved as:

| Condition | Applicable threshold for metric `M` |
| --- | --- |
| spec declares `coverage_metric_thresholds[M]` | that override |
| spec declares only the scalar `coverage_threshold` | the scalar (default) |
| multiple in-scope specs apply | the MAX across specs of each spec's applicable (override-or-default) value for `M` |
| no scalar and no override in scope ⇒ resolved `<= 0` | metric is SKIPPED (not failed) |

The gate computes `Covered*100 >= Total*threshold` for that `(path, metric)` record. The
metric label is consumed ONLY as the lookup key into `coverage_metric_thresholds` and as a
report label — never ranked or compared.

## Implementation

Target package: **`pkg/gate`** (the consumer index + threshold selection), with a small
frontmatter-extraction change in `pkg/gate/step_testverify.go`. The canonical record/parser
in `pkg/check` are **unchanged**. Processing steps the planner must map tasks to:

1. **Re-key the index by `(path, metric)` (REQ-001).** Replace `indexCoverageByPath`
   (`map[string]check.CoverageRecord`) with `indexCoverageByPathMetric` returning a nested
   `map[string]map[string]check.CoverageRecord` (path → metric → record) PLUS the set of
   `(path, metric)` keys seen more than once. Preserve `normalizeScopePath` on the outer key.
   Replace `resolveCoverageRecord` with `resolveCoverageRecordsForPath`, which returns the
   whole per-metric map for a scope path using the SAME exact-then-unique-suffix fallback
   (separator-anchored, ambiguous ⇒ no-match) the old resolver used.

2. **Surface duplicate `(path, metric)` loudly (REQ-001).** When the producer emitted two
   records under the same `(path, metric)`, the step emits a blocking `coverage_threshold`
   (or dedicated `coverage_metric_collision`) violation for that path+metric — a
   duplicate-measurement producer defect — rather than silently keeping one.

3. **Thread the per-metric threshold surface (REQ-003).** Add
   `MetricThresholds map[string]int` to `SpecVerification`. In `pkg/gate/step_testverify.go`,
   add `coverage_metric_thresholds` (`map[string]int`) to the spec-frontmatter Verification
   struct and populate `SpecVerification.MetricThresholds` in `ExtractSpecVerifications`;
   loosen the extraction gate so a spec declaring only per-metric thresholds (no scalar) is
   still extracted. Add `coverageThresholdForMetric(sel, metric)` generalizing
   `coverageFloorForScope` from a single floor to a per-metric floor (override-or-default,
   MAX across selected specs, `0` when none).

4. **Evolve the verdict loop to per-`(path, metric)` (REQ-002).** In
   `StepCoverageThresholdScopedFunc`, for each in-scope path that HAS records, iterate EVERY
   metric record (`resolveCoverageRecordsForPath`). For each, resolve the metric's threshold
   via `coverageThresholdForMetric`; skip when `<= 0`; preserve `Total==0` ⇒ N/A and the
   `Excluded` loud-surfacing of an in-scope changed file — all at `(path, metric)`
   granularity. Emit a below-threshold violation per failing metric with a message naming the
   metric, its raw counts, and the metric-specific threshold. A single file may thus emit
   several violations (one per failing metric); a passing sibling metric never rescues a
   failing one.

5. **Add the metric-granular missing-metric guard (REQ-005).** For each metric `M`
   EXPLICITLY declared in any in-scope spec's `coverage_metric_thresholds`, if an in-scope
   CHANGED path has at least one record but none with `Metric == M`, emit a loud blocking
   `coverage_metric_missing` violation. Fire ONLY for explicitly-declared metrics and ONLY on
   in-scope changed paths. Do NOT touch the path-level "zero records at all" guard owned by
   SPEC-043 (a path with zero records is not this guard's concern).

6. **Preserve backward-compat (REQ-004).** A `statement`-only record set against a scalar-only
   spec exercises exactly the path above with one metric per path, no override map, no
   explicitly-declared metric — so it produces identical verdicts to today, no collision, and
   no metric-missing guard. This is a regression boundary, not new behavior.

7. **Record the bun shape contract (REQ-006).** No code emits bun records in THIS spec (that
   is Seed 5), but the consumer + parser this spec evolves MUST accept and correctly threshold
   a bun-shaped record set (two records per file, `line` + `branch`, same `Path`, through the
   existing `check.CoverageRecord` + `check.ParsePackCoverage`). The mandated bun-shape tests
   (CLM-022/CLM-023) lock the contract Seed 5 targets.

## Verification

- **Level:** `unit` — the evolution is entirely within the `pkg/gate` consumer (the index,
  the per-metric threshold selection, the verdict loop, and the frontmatter extraction); the
  canonical record/parser are unchanged and the producer is out of scope (Seed 5). No
  cross-package dispatch is exercised here.
- **Test command:** `go test ./pkg/gate/ -race -coverprofile=cover.out`
- **Coverage threshold:** 90 (unit level).

Claims (CLM-NNN) are enumerated in the `claims:` frontmatter, each mapping a REQ to mandated
test names. The mandated independence test
(`TestCoverage_SameFileLineAndBranchThresholdedIndependentlyNoCollision`, CLM-005) is the
keystone: one file carrying `line` 95/100 (passes 90) AND `branch` 60/100 (fails 70) yields
exactly one violation — branch — with the line metric neither flagged nor collided. Every
requirement carries both positive and negative/edge claims; backward-compat (REQ-004) and the
metric-missing guard's no-false-positive condition (CLM-019) guard against regressions and
over-firing respectively.

## Sharp Edges

1. **Last-write-wins collapse is the silent failure this spec exists to kill.** Today
   `indexCoverageByPath` keys by path, so two records for one file overwrite. The cheap-but-wrong
   "fix" is to keep keying by path and just iterate `coverage` — but any path-keyed map still
   collapses line+branch. The index MUST key by `(path, metric)` (REQ-001/CLM-001) AND a
   duplicate `(path, metric)` must be LOUD (CLM-003), not last-wins — otherwise a producer that
   double-emits a metric silently drops half its measurements.

2. **A passing sibling metric must NOT rescue a failing one.** With per-`(path, metric)`
   verdicts, the tempting bug is to treat the file as "covered" if ANY metric passes (or to
   average them). That re-introduces the aggregation-rescue the per-FILE model already forbids,
   now at metric granularity. `branch` 60/100 must red even though `line` 95/100 passes on the
   same file (CLM-005/CLM-008). No averaging across metrics, ever.

3. **The metric label must never be semantically interpreted — only matched.** It is tempting
   to bake "branch is normally lower than line" into the gate (e.g. auto-discount branch). That
   re-bakes language/tool knowledge the thin-executor principle forbids. The policy lives in the
   DECLARED numbers (`coverage_metric_thresholds`), not the binary; the gate uses the label only
   as a map key + report label and never ranks metrics (CLM-010). A future `region`/`function`
   metric works with zero new gate code.

4. **`Total == 0` and `Excluded` are now PER METRIC, not per file.** A file might have a
   `Total==0` `branch` (no branchable code) but a real `line` measurement. The N/A skip and the
   `Excluded` loud-surfacing must apply at `(path, metric)` granularity (CLM-009), or a legitimate
   N/A on one metric will either red the file or suppress the other metric.

5. **Backward-compat with the single-`statement` producer is a hard regression boundary.** The
   live `go-toolchain` producer emits one `statement` metric per file. If the new code requires a
   per-metric override, or fires the metric-missing guard on it, or collides it, the existing Go
   gate breaks. A scalar-only spec over `statement`-only records MUST behave EXACTLY as today
   (REQ-004/CLM-016/CLM-017). This is the most likely thing to regress.

6. **The metric-missing guard can over-fire into a false positive.** If `coverage_metric_missing`
   fired for ANY metric not present (rather than only EXPLICITLY-declared metrics), then every
   `statement`-only file would red for "missing branch/line." The guard MUST be scoped to metrics
   that appear in `coverage_metric_thresholds` AND to in-scope changed paths (CLM-019/CLM-021),
   or it punishes producers for not measuring metrics nobody asked for.

7. **Seam overlap with SPEC-043's path-level guard (cross-consistency item).** Both specs add a
   "loud, refuse-to-pass" coverage guard to the same file. They must remain at different
   granularities: SPEC-043 = "path has ZERO records"; this spec = "path HAS records but is missing
   an explicitly-declared metric" (CLM-020). The risk is a merge where one guard shadows the other
   (e.g. SPEC-043's guard `continue`s before this spec's metric loop runs, hiding a
   `coverage_metric_missing`, or both fire and double-report). The implementer must order them so a
   zero-record path yields SPEC-043's guard ALONE and a has-records-missing-declared-metric path
   yields this spec's guard ALONE. Flagged for the cross-consistency pass.

8. **Empty / absent metric label still fails loud upstream — do not re-handle it here.** The
   parser (`check.ParsePackCoverage`, SPEC-042 REQ-005) already fail-louds on a measured record
   with an empty `Metric`. The consumer must not silently bucket an empty-metric record under
   `("", record)` and thereby resurrect the unlabeled-measurement hazard; it relies on the parser
   having rejected it. If a record reaches the index with an empty `Metric`, that is a contract
   violation, not a metric to threshold.

9. **`step_testverify.go` is co-edited with SPEC-045 (shared-extraction seam).** This spec edits
   `pkg/gate/step_testverify.go` to extend the spec-frontmatter `Verification` struct and
   `ExtractSpecVerifications` with the `coverage_metric_thresholds` → `MetricThresholds`
   extraction (REQ-003). SPEC-045 ALSO edits the same file — it replaces the baked `funcPattern` /
   `collectTestFuncNamesScoped` test-name discovery with a pack-declared `TestNameMatcher`. The
   two edits are DISJOINT in concern (frontmatter/extraction here; test-name discovery there) but
   land in the same file and both touch the extraction/verification surface, so they will merge-
   conflict if implemented independently. The implementer MUST reconcile the two on merge — neither
   spec's `step_testverify.go` change may clobber the other's; in particular this spec's loosening
   of the extraction gate (so a spec declaring only per-metric thresholds without a scalar is still
   extracted) must survive alongside SPEC-045's `TestNameMatcher`/discovery rework. Flagged for the
   cross-consistency pass and the implementer.

## Review Questions

These probe risks not fully pinned by the claims; the impl-reviewer should check each against
the diff.

1. Is the index keyed by `(path, metric)` such that two records with the same `Path` but
   different `Metric` BOTH survive, and would the build FAIL (loudly) if a producer emitted two
   records with the same `(path, metric)` rather than silently keeping one? (REQ-001 /
   CLM-001/CLM-003.)
2. Does a single file with a passing `line` and a failing `branch` produce EXACTLY ONE
   violation (branch), and would averaging or "any-metric-passes" logic make the test fail?
   (REQ-002 / CLM-005/CLM-008.)
3. Is the per-metric threshold resolved as `MetricThresholds[metric]` else scalar
   `CoverageThreshold`, with the MAX taken across in-scope specs PER METRIC — and does declaring
   a `branch` override leave the `line` threshold untouched? (REQ-003 / CLM-011/CLM-013/CLM-014.)
4. Does a `statement`-only record set against a scalar-only spec produce verdicts byte-for-byte
   equivalent to the pre-existing one-record-per-path behavior, with NO collision and NO
   metric-missing guard firing? (REQ-004 / CLM-016/CLM-017.)
5. Does `coverage_metric_missing` fire ONLY for metrics that appear in
   `coverage_metric_thresholds` and ONLY for in-scope changed paths that HAVE records — and
   would it NOT fire on a scalar-only spec or on an all-mode/unchanged path? (REQ-005 /
   CLM-019/CLM-021.)
6. Is this spec's metric-missing guard kept DISTINCT from SPEC-043's path-level "zero records"
   guard — a zero-record path yields SPEC-043's guard alone, a has-records-missing-declared-metric
   path yields this spec's guard alone, never both double-reporting and never one shadowing the
   other? (REQ-005 / CLM-020, Sharp Edge 7.)
7. Does the gate compute the verdict from RAW COUNTS (`Covered*100 >= Total*threshold`) per
   `(path, metric)` with no floating-point and no pre-computed percent, and is the metric label
   used ONLY as a lookup key + report label (never ranked/compared)? (REQ-002 / CLM-010.)
8. Does a bun-shaped record set (two records per file, `line` + `branch`, same `Path`) flow
   through the EXISTING `check.CoverageRecord` and `check.ParsePackCoverage` with no new type or
   field, and threshold each metric independently through the same consumer path the
   `statement` records use? (REQ-006 / CLM-022/CLM-023.)

## References

- BUNDLE-012 (language-neutral-consumer-ts-toolchain) — **Seed 4**, REQ-007, and the
  resolution of **SQ-2** (per-metric thresholds). This spec is the FOUNDATION record-model
  contract Seed 5 (the bun coverage engine) depends on.
- SPEC-042 (coverage-production-engine) — defines the canonical `check.CoverageRecord`
  (`{Path, Covered, Total, Measured, Excluded, Metric}`, raw counts) and
  `check.ParsePackCoverage` this spec EVOLVES on the consumer side (re-opened by REQ-007). The
  `Metric` label this spec keys on is SPEC-042's pack-declared label; the empty-metric
  fail-loud lives there (Sharp Edge 8).
- SPEC-043 (pack-declared-globs-coverage-consumer) — **parallel sibling**, shares
  `pkg/gate/step_coverage.go` AND the `StepCoverageThresholdScopedFunc` signature. Owns the
  pack-declared glob contract, the in-scope measurable PATH SET, the PATH-LEVEL no-record guard,
  AND the CANONICAL 4-arg `StepCoverageThresholdScopedFunc(coverage, specs, scope, classifier
  SourceClassifier)` signature — it adds the trailing `classifier SourceClassifier` parameter.
  THIS spec adopts that 4-arg signature unchanged and owns only the inner record model reached
  through it (the `(path, metric)` index + per-metric thresholding + the inner verdict loop).
  Two seams are flagged for the cross-consistency pass: (a) this spec's per-metric guard vs
  043's path-level no-record guard, and (b) this spec's `(path, metric)` index + per-metric
  verdict loop composing with 043's classifier-derived measurable set inside the shared function.
- SPEC-047 (bun-toolchain-pack-and-proof, Seed 5 — later) — the bun coverage producer that
  emits `line` + branch into the record shape REQ-006 defines; this spec's consumer is its
  target.
- SPEC-045 (de-go-test-verification-discovery, Seed 5-track sibling) — **co-edits
  `pkg/gate/step_testverify.go`** (it replaces the baked `funcPattern` / `collectTestFuncNamesScoped`
  test-name discovery with a pack-declared `TestNameMatcher`). THIS spec also edits
  `step_testverify.go` — extending the spec-frontmatter `Verification` struct and
  `ExtractSpecVerifications` to thread `coverage_metric_thresholds` → `SpecVerification.MetricThresholds`
  (REQ-003). The two specs touch DISJOINT concerns in that file (extraction/frontmatter here;
  test-name discovery there), but the shared-file / shared-extraction co-edit is flagged for the
  cross-consistency pass and for the implementer to reconcile on merge (see Sharp Edges).
- Code (verified on this branch 2026-06-28): `pkg/gate/step_coverage.go` —
  `indexCoverageByPath` (one-record-per-path, ~L164), `resolveCoverageRecord` (~L181),
  `coverageFloorForScope`/`coverageThresholdsForScope` (~L268/291), `StepCoverageThresholdScopedFunc`
  (~L53); `pkg/gate/step_testverify.go` — the spec-frontmatter `Verification` struct (~L37) and
  `ExtractSpecVerifications` (~L316); `pkg/check/coverage.go` — the canonical `CoverageRecord` +
  `ParsePackCoverage` (unchanged).
- [[feedback_loud_not_blocking]] — the anti-vacuous-green philosophy the metric-granular
  missing-metric guard (REQ-005) serves at metric granularity.
- [[feedback_zero_baked_checks]] — the metric-blindness in REQ-002/REQ-003: policy lives in
  declared numbers, not in the binary; the gate never ranks metrics.
