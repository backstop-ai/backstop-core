package gate

import (
	"context"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

// TestMetricMissing_DeclaredBranchUnmeasuredOnChangedPathFailsLoud (CLM-018): with
// branch EXPLICITLY declared in coverage_metric_thresholds, an in-scope CHANGED path
// that has a line record but NO branch record is a LOUD blocking
// coverage_metric_missing error — refusing to pass with the declared branch metric
// silently unmeasured.
func TestMetricMissing_DeclaredBranchUnmeasuredOnChangedPathFailsLoud(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "src/foo.ts", Covered: 95, Total: 100, Measured: true, Metric: "line"},
	}
	result := runPerMetric(records, perMetricSpec("src/foo.ts", 90, map[string]int{"branch": 70}),
		diffScopeFor("src/foo.ts"), bunClassifier())

	if result.Status != "fail" {
		t.Fatalf("a declared branch metric unmeasured on a changed path must red, got %s: %#v", result.Status, result.Violations)
	}
	v, ok := violationForRule(result.Violations, "coverage_metric_missing")
	if !ok {
		t.Fatalf("expected a coverage_metric_missing violation, got %#v", result.Violations)
	}
	if v.File != "src/foo.ts" || v.Severity != "error" {
		t.Errorf("the guard must be a blocking error for the changed file, got %#v", v)
	}
	if !strings.Contains(v.Message, "branch") {
		t.Errorf("the guard must name the missing declared metric (branch), got %q", v.Message)
	}
}

// TestMetricMissing_NoFalsePositiveWhenNoMetricDeclared (CLM-019): with NO per-metric
// override (scalar-only spec), an in-scope changed path with only a line record does
// NOT trigger the guard — it fires ONLY for explicitly-declared metrics.
func TestMetricMissing_NoFalsePositiveWhenNoMetricDeclared(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "src/foo.ts", Covered: 95, Total: 100, Measured: true, Metric: "line"},
	}
	// Scalar-only spec: no coverage_metric_thresholds ⇒ no explicitly-declared metric.
	result := runPerMetric(records, perMetricSpec("src/foo.ts", 90, nil),
		diffScopeFor("src/foo.ts"), bunClassifier())

	if _, ok := violationForRule(result.Violations, "coverage_metric_missing"); ok {
		t.Errorf("a scalar-only spec must NOT trigger the metric-missing guard, got %#v", result.Violations)
	}
	if result.Status != "pass" {
		t.Errorf("line 95/100 at scalar 90 with no declared metric must pass, got %s: %#v", result.Status, result.Violations)
	}
}

// TestMetricMissing_DistinctFromPathLevelNoRecordGuard (CLM-020): a path with ZERO
// records is SPEC-043's path-level guard's case (coverage_unmeasured), while a path
// WITH records missing an explicitly-declared metric is coverage_metric_missing — the
// two rules emit distinctly and do not duplicate or shadow.
func TestMetricMissing_DistinctFromPathLevelNoRecordGuard(t *testing.T) {
	records := []check.CoverageRecord{
		// src/has.ts HAS a line record but is missing the declared branch metric.
		{Path: "src/has.ts", Covered: 95, Total: 100, Measured: true, Metric: "line"},
		// src/none.ts has ZERO records (not present at all) — SPEC-043's case.
	}
	result := runPerMetric(records, perMetricSpec("src/has.ts", 90, map[string]int{"branch": 70}),
		diffScopeFor("src/none.ts", "src/has.ts"), bunClassifier())

	none, okNone := violationForFile(result.Violations, "src/none.ts")
	has, okHas := violationForFile(result.Violations, "src/has.ts")
	if !okNone || !okHas {
		t.Fatalf("expected one violation for EACH path, got %#v", result.Violations)
	}
	if none.Rule != "coverage_unmeasured" {
		t.Errorf("a ZERO-record path is SPEC-043's path-level guard (coverage_unmeasured), got %q", none.Rule)
	}
	if has.Rule != "coverage_metric_missing" {
		t.Errorf("a path WITH records missing a declared metric is coverage_metric_missing, got %q", has.Rule)
	}
	// No double-report: each path emits exactly one violation, neither shadowing the
	// other's rule.
	if countViolationsForFile(result.Violations, "src/none.ts") != 1 {
		t.Errorf("the zero-record path must NOT also emit coverage_metric_missing, got %#v", result.Violations)
	}
	if countViolationsForFile(result.Violations, "src/has.ts") != 1 {
		t.Errorf("the has-records path must NOT also emit coverage_unmeasured, got %#v", result.Violations)
	}
}

// TestMetricMissing_UnchangedPathMissingMetricStaysQuiet (CLM-021): an UNCHANGED /
// all-mode path missing an explicitly-declared metric stays QUIET — the guard is
// scoped to in-scope CHANGED paths, parallel to the exclusion-loudness scoping.
func TestMetricMissing_UnchangedPathMissingMetricStaysQuiet(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "src/foo.ts", Covered: 95, Total: 100, Measured: true, Metric: "line"},
	}
	allMode := newGateScope("", GateScopeModeAll, nil, nil)
	result := StepCoverageThresholdScopedFunc(records,
		perMetricSpec("src/foo.ts", 90, map[string]int{"branch": 70}), allMode, bunClassifier())(context.Background())

	if _, ok := violationForRule(result.Violations, "coverage_metric_missing"); ok {
		t.Errorf("an all-mode/unchanged path missing a declared metric must stay quiet, got %#v", result.Violations)
	}
	if result.Status == "fail" {
		t.Errorf("an all-mode sweep must not red solely on a missing declared metric, got %#v", result.Violations)
	}
}

// TestBackwardCompat_StatementOnlyNoCollisionNoMetricMissingGuard (CLM-017): a
// statement-only record set with no coverage_metric_thresholds triggers NEITHER a
// (path, metric) collision NOR the declared-metric-not-measured guard — the
// scalar-only path is free of the new guards.
func TestBackwardCompat_StatementOnlyNoCollisionNoMetricMissingGuard(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "pkg/a/ok.go", Covered: 95, Total: 100, Measured: true, Metric: "statement"},
	}
	result := runCoverage(records, 90, diffScopeFor("pkg/a/ok.go"))

	if _, ok := violationForRule(result.Violations, "coverage_metric_collision"); ok {
		t.Errorf("a statement-only set must NOT trigger a (path, metric) collision, got %#v", result.Violations)
	}
	if _, ok := violationForRule(result.Violations, "coverage_metric_missing"); ok {
		t.Errorf("a scalar-only spec must NOT trigger the metric-missing guard, got %#v", result.Violations)
	}
	if result.Status != "pass" {
		t.Errorf("statement 95/100 at 90 must pass cleanly, got %s: %#v", result.Status, result.Violations)
	}
}

// countViolationsForFile counts violations whose File matches.
func countViolationsForFile(violations []Violation, file string) int {
	n := 0
	for _, v := range violations {
		if v.File == file {
			n++
		}
	}
	return n
}
