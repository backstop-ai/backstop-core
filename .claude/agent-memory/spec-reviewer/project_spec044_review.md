---
name: spec044-review
description: SPEC-044 (multi-metric (path,metric) coverage records, BUNDLE-012 Seed 4) per-spec review — PASS
metadata:
  type: project
---

SPEC-044 (multi-metric coverage records, BUNDLE-012 Seed 4, owns REQ-007 + resolves SQ-2 per-metric thresholds) per-spec review = **PASS** (validator PASS; no blockers). Verified vs live code 2026-06-29.

**B1 reconciliation (post-cross-pass) verified sound:** 044's adopted `StepCoverageThresholdScopedFunc(coverage, specs, scope, classifier SourceClassifier)` 4-arg signature is byte-identical to SPEC-043's canonical contract — closes cross-pass blocker #1 ([[bundle012-crosspass]] #1, the prior 3-arg "UNCHANGED" conflict). Ownership split clean: 043 owns classifier+measurable-set+path-level no-record guard; 044 owns the `(path,metric)` index + per-metric thresholding + the metric-granular `coverage_metric_missing` guard. SPEC-045→047 citation fix applied. step_testverify.go co-edit flag is REAL + disjoint (045 deletes funcPattern→TestNameMatcher in collectTestFuncNamesScoped; 044 adds coverage_metric_thresholds→MetricThresholds in ExtractSpecVerifications; verified 045 does not touch the extractor).

**REQ-006 central thesis VERIFIED TRUE vs live:** `check.CoverageRecord` already carries `Metric`; `check.ParsePackCoverage` returns a `[]CoverageRecord` slice with NO path-dedupe → two records sharing a Path (line+branch) both survive parse. Collapse is consumer-only (`indexCoverageByPath` map[string]Record last-wins). So "no new type/field/schema fork" is accurate; producer-side schema evolution was already delivered by SPEC-042 — the only Seed-4 producer deliverable is the record-SHAPE contract (REQ-006), bun convert script is genuinely Seed 5/047. Not a scope gap.

**One awareness note (NOT a 044 blocker — seam, out of scope):** live `StepCoverageThresholdScopedFunc` opens with `if coverageFloorForScope(...) <= 0 { return pass }` keyed on the SCALAR floor. A per-metric-only spec (scalar absent) would short-circuit to vacuous pass before 044's per-metric loop + metric-missing guard run. Governed by the shared function's OUTER control flow which SPEC-043 owns + explicitly commits to firing guards "even when no numeric threshold is in scope." Cross-check at implementation that 043's restructure generalizes that early-return, else 044 REQ-003 per-metric-only + REQ-005 break silently.
