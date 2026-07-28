package gate

import (
	"context"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
)

// perMetricSpec is an in-scope spec carrying a scalar default and an optional
// per-metric override map, declared over the given file so coverageThresholdsForScope
// selects it (and thereby brings its MetricThresholds into scope).
func perMetricSpec(file string, scalar int, overrides map[string]int) []SpecVerification {
	return []SpecVerification{{
		SpecID:            "SPEC-PM",
		TestCommand:       "go test ./...",
		CoverageThreshold: scalar,
		MetricThresholds:  overrides,
		File:              file,
	}}
}

// runPerMetric drives the per-(path, metric) verdict loop directly.
func runPerMetric(records []check.CoverageRecord, specs []SpecVerification, scope *GateScope, classifier SourceClassifier) StepResult {
	return StepCoverageThresholdScopedFunc(records, specs, scope, classifier)(context.Background())
}

func metricViolations(violations []Violation, rule string) []Violation {
	var out []Violation
	for _, v := range violations {
		if v.Rule == rule {
			out = append(out, v)
		}
	}
	return out
}

// TestCoverage_SameFileLineAndBranchThresholdedIndependentlyNoCollision (CLM-005):
// the keystone — one file carrying line 95/100 (line threshold 90, pass) AND branch
// 60/100 (branch threshold 70, fail) produces EXACTLY ONE violation (branch); the
// passing line metric is NOT flagged and did NOT collide.
func TestCoverage_SameFileLineAndBranchThresholdedIndependentlyNoCollision(t *testing.T) {
	// branch FIRST, the passing line LAST: a path-keyed last-write-wins index would
	// keep line (95/100, pass) and go vacuous-green; the (path, metric) index must
	// retain BOTH and red the branch.
	records := []check.CoverageRecord{
		{Path: "src/foo.ts", Covered: 60, Total: 100, Measured: true, Metric: "branch"},
		{Path: "src/foo.ts", Covered: 95, Total: 100, Measured: true, Metric: "line"},
	}
	result := runPerMetric(records, perMetricSpec("src/foo.ts", 90, map[string]int{"branch": 70}),
		diffScopeFor("src/foo.ts"), bunClassifier())

	if result.Status != "fail" {
		t.Fatalf("a below-threshold branch metric must red, got %s: %#v", result.Status, result.Violations)
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected EXACTLY ONE violation (branch), got %d: %#v", len(result.Violations), result.Violations)
	}
	v := result.Violations[0]
	if v.Rule != "coverage_threshold" {
		t.Errorf("expected a coverage_threshold violation, got rule %q", v.Rule)
	}
	if !strings.Contains(v.Message, "branch") {
		t.Errorf("the lone violation must be the branch metric, got %q", v.Message)
	}
	if strings.Contains(v.Message, "line") {
		t.Errorf("the passing line metric must NOT be flagged, got %q", v.Message)
	}
}

// TestCoverage_DuplicatePathMetricSurfacedLoudlyInVerdict (CLM-003, verdict level):
// the verdict loop surfaces a duplicate (path, metric) from the index's dup-key
// return as a LOUD blocking coverage_metric_collision — a producer that double-emits
// one metric for one file is a defect, never silently collapsed to one survivor.
func TestCoverage_DuplicatePathMetricSurfacedLoudlyInVerdict(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "src/foo.ts", Covered: 95, Total: 100, Measured: true, Metric: "line"},
		{Path: "src/foo.ts", Covered: 40, Total: 100, Measured: true, Metric: "line"},
	}
	result := runPerMetric(records, perMetricSpec("src/foo.ts", 90, nil),
		diffScopeFor("src/foo.ts"), bunClassifier())

	if result.Status != "fail" {
		t.Fatalf("a duplicate (path, metric) must red loudly, got %s: %#v", result.Status, result.Violations)
	}
	v, ok := violationForRule(result.Violations, "coverage_metric_collision")
	if !ok {
		t.Fatalf("expected a coverage_metric_collision violation, got %#v", result.Violations)
	}
	if v.File != "src/foo.ts" || v.Severity != "error" {
		t.Errorf("the collision must be a blocking error naming the file, got %#v", v)
	}
	if !strings.Contains(v.Message, "line") || !strings.Contains(v.Message, "duplicate") {
		t.Errorf("the collision message must name the duplicated metric, got %q", v.Message)
	}
}

// TestCoverage_BothMetricsPassNoViolation (CLM-006): a file whose line AND branch
// both pass their thresholds produces NO violation.
func TestCoverage_BothMetricsPassNoViolation(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "src/foo.ts", Covered: 95, Total: 100, Measured: true, Metric: "line"},
		{Path: "src/foo.ts", Covered: 80, Total: 100, Measured: true, Metric: "branch"},
	}
	result := runPerMetric(records, perMetricSpec("src/foo.ts", 90, map[string]int{"branch": 70}),
		diffScopeFor("src/foo.ts"), bunClassifier())

	if result.Status != "pass" {
		t.Fatalf("line 95/100 (>=90) and branch 80/100 (>=70) must both pass, got %s: %#v", result.Status, result.Violations)
	}
	if len(result.Violations) != 0 {
		t.Errorf("both-pass must produce NO violation, got %#v", result.Violations)
	}
}

// TestCoverage_BothMetricsFailTwoViolations (CLM-007): a file whose line AND branch
// both fail produces TWO violations — one per (path, metric) — each metric reds
// independently.
func TestCoverage_BothMetricsFailTwoViolations(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "src/foo.ts", Covered: 50, Total: 100, Measured: true, Metric: "line"},
		{Path: "src/foo.ts", Covered: 40, Total: 100, Measured: true, Metric: "branch"},
	}
	result := runPerMetric(records, perMetricSpec("src/foo.ts", 90, map[string]int{"branch": 70}),
		diffScopeFor("src/foo.ts"), bunClassifier())

	if result.Status != "fail" {
		t.Fatalf("both below threshold must red, got %s: %#v", result.Status, result.Violations)
	}
	thresh := metricViolations(result.Violations, "coverage_threshold")
	if len(thresh) != 2 {
		t.Fatalf("expected TWO coverage_threshold violations (line and branch), got %d: %#v", len(thresh), result.Violations)
	}
	var sawLine, sawBranch bool
	for _, v := range thresh {
		if strings.Contains(v.Message, "line") {
			sawLine = true
		}
		if strings.Contains(v.Message, "branch") {
			sawBranch = true
		}
	}
	if !sawLine || !sawBranch {
		t.Errorf("each metric must red independently (one line, one branch), got %#v", thresh)
	}
}

// TestCoverage_BelowThresholdMetricNotRescuedByPassingSibling (CLM-008): a
// below-threshold metric reds even when a SIBLING metric on the SAME file passes — no
// aggregation or sibling-metric rescue.
func TestCoverage_BelowThresholdMetricNotRescuedByPassingSibling(t *testing.T) {
	// branch (failing) FIRST, line (passing) LAST: last-write-wins would keep the
	// passing line and rescue the file — exactly the bug this guards against.
	records := []check.CoverageRecord{
		{Path: "src/foo.ts", Covered: 40, Total: 100, Measured: true, Metric: "branch"},
		{Path: "src/foo.ts", Covered: 99, Total: 100, Measured: true, Metric: "line"},
	}
	result := runPerMetric(records, perMetricSpec("src/foo.ts", 90, map[string]int{"branch": 70}),
		diffScopeFor("src/foo.ts"), bunClassifier())

	if result.Status != "fail" {
		t.Fatalf("a passing line sibling must NOT rescue a failing branch, got %s: %#v", result.Status, result.Violations)
	}
	thresh := metricViolations(result.Violations, "coverage_threshold")
	if len(thresh) != 1 || !strings.Contains(thresh[0].Message, "branch") {
		t.Errorf("exactly the failing branch metric must red, got %#v", result.Violations)
	}
}

// TestCoverage_TotalZeroPerMetricIsNAOtherMetricStillThresholded (CLM-009): a
// Total==0 record for ONE metric is N/A (skipped, never a 0%-fail) while the OTHER
// metric on the same file is still thresholded.
func TestCoverage_TotalZeroPerMetricIsNAOtherMetricStillThresholded(t *testing.T) {
	records := []check.CoverageRecord{
		// line is a real measurement below its threshold ⇒ still reds.
		{Path: "src/foo.ts", Covered: 50, Total: 100, Measured: true, Metric: "line"},
		// No branchable code on this file: branch Total==0 ⇒ N/A. Placed LAST so a
		// last-write-wins index would keep this N/A record and skip the whole file,
		// hiding the line shortfall.
		{Path: "src/foo.ts", Covered: 0, Total: 0, Measured: true, Metric: "branch"},
	}
	result := runPerMetric(records, perMetricSpec("src/foo.ts", 90, map[string]int{"branch": 70}),
		diffScopeFor("src/foo.ts"), bunClassifier())

	thresh := metricViolations(result.Violations, "coverage_threshold")
	if len(thresh) != 1 {
		t.Fatalf("the Total==0 branch must be N/A while line is thresholded — expected exactly one violation (line), got %#v", result.Violations)
	}
	if !strings.Contains(thresh[0].Message, "line") {
		t.Errorf("the surviving thresholded metric must be line, got %q", thresh[0].Message)
	}
	for _, v := range result.Violations {
		if strings.Contains(v.Message, "branch") {
			t.Errorf("a Total==0 branch must never produce a 0%%-fail, got %q", v.Message)
		}
	}
}

// TestCoverage_VerdictMetricBlindOnRawCounts (CLM-010): two records with DIFFERENT
// metric labels but IDENTICAL Covered/Total against the SAME applicable threshold
// produce IDENTICAL verdicts — the label is never ordered/ranked/interpreted in the
// ratio.
func TestCoverage_VerdictMetricBlindOnRawCounts(t *testing.T) {
	// Same file, scalar-only spec ⇒ line and branch both resolve to the same scalar
	// threshold 70. Identical counts ⇒ identical verdicts.
	failRecords := []check.CoverageRecord{
		{Path: "src/foo.ts", Covered: 60, Total: 100, Measured: true, Metric: "line"},
		{Path: "src/foo.ts", Covered: 60, Total: 100, Measured: true, Metric: "branch"},
	}
	failResult := runPerMetric(failRecords, perMetricSpec("src/foo.ts", 70, nil),
		diffScopeFor("src/foo.ts"), bunClassifier())
	if len(metricViolations(failResult.Violations, "coverage_threshold")) != 2 {
		t.Fatalf("identical 60/100 counts at threshold 70 must red IDENTICALLY for both labels, got %#v", failResult.Violations)
	}

	passRecords := []check.CoverageRecord{
		{Path: "src/foo.ts", Covered: 80, Total: 100, Measured: true, Metric: "line"},
		{Path: "src/foo.ts", Covered: 80, Total: 100, Measured: true, Metric: "branch"},
	}
	passResult := runPerMetric(passRecords, perMetricSpec("src/foo.ts", 70, nil),
		diffScopeFor("src/foo.ts"), bunClassifier())
	if passResult.Status != "pass" || len(passResult.Violations) != 0 {
		t.Fatalf("identical 80/100 counts at threshold 70 must pass IDENTICALLY for both labels, got %s: %#v", passResult.Status, passResult.Violations)
	}
}

// TestBackwardCompat_StatementOnlyRecordsIdenticalToScalarBehavior (CLM-016): a
// go-toolchain-shaped record set (one statement record per path) against a
// scalar-only spec yields verdicts IDENTICAL to the pre-existing one-record-per-path
// behavior — measured-passing passes, below-threshold reds, Total==0 is N/A.
func TestBackwardCompat_StatementOnlyRecordsIdenticalToScalarBehavior(t *testing.T) {
	// measured-and-passing
	pass := []check.CoverageRecord{{Path: "pkg/a/ok.go", Covered: 95, Total: 100, Measured: true, Metric: "statement"}}
	if got := runCoverage(pass, 90, diffScopeFor("pkg/a/ok.go")); got.Status != "pass" {
		t.Fatalf("statement 95/100 at 90 must pass, got %s: %#v", got.Status, got.Violations)
	}
	// measured-and-below-threshold
	low := []check.CoverageRecord{{Path: "pkg/a/low.go", Covered: 10, Total: 100, Measured: true, Metric: "statement"}}
	lowResult := runCoverage(low, 90, diffScopeFor("pkg/a/low.go"))
	if lowResult.Status != "fail" {
		t.Fatalf("statement 10/100 at 90 must fail, got %s: %#v", lowResult.Status, lowResult.Violations)
	}
	if v, ok := violationForFile(lowResult.Violations, "pkg/a/low.go"); !ok || v.Rule != "coverage_threshold" {
		t.Errorf("the below-threshold statement file must red as coverage_threshold, got %#v", lowResult.Violations)
	}
	// Total==0 ⇒ N/A
	decls := []check.CoverageRecord{{Path: "pkg/a/decls.go", Covered: 0, Total: 0, Measured: true, Metric: "statement"}}
	if got := runCoverage(decls, 90, diffScopeFor("pkg/a/decls.go")); got.Status != "pass" || len(got.Violations) != 0 {
		t.Fatalf("statement Total==0 must be N/A (pass, no violation), got %s: %#v", got.Status, got.Violations)
	}
}

// TestBunShape_LineAndBranchThresholdedThroughSharedConsumerPath (CLM-023): a
// bun-shaped line+branch record set thresholds each metric independently through the
// SAME consumer path the statement records flow through — the consumer is
// producer-agnostic.
func TestBunShape_LineAndBranchThresholdedThroughSharedConsumerPath(t *testing.T) {
	records, err := check.ParsePackCoverage(readTestdata(t, "coverage-bun-line-branch.json"))
	if err != nil {
		t.Fatalf("bun fixture must parse: %v", err)
	}
	// line threshold 90 (scalar default), branch override 70:
	//   src/foo.ts: line 95 pass, branch 60 FAIL
	//   src/bar.ts: line 98 pass, branch 85 pass
	result := runPerMetric(records, perMetricSpec("src/foo.ts", 90, map[string]int{"branch": 70}),
		diffScopeFor("src/foo.ts", "src/bar.ts"), bunClassifier())

	if result.Status != "fail" {
		t.Fatalf("src/foo.ts branch 60/100 (<70) must red, got %s: %#v", result.Status, result.Violations)
	}
	thresh := metricViolations(result.Violations, "coverage_threshold")
	if len(thresh) != 1 {
		t.Fatalf("expected exactly one below-threshold violation (foo branch), got %#v", result.Violations)
	}
	v := thresh[0]
	if v.File != "src/foo.ts" || !strings.Contains(v.Message, "branch") {
		t.Errorf("the violation must be src/foo.ts branch, got %#v", v)
	}
	if hasViolationForFile(result.Violations, "src/bar.ts") {
		t.Errorf("src/bar.ts (line 98, branch 85 both pass) must NOT be flagged, got %#v", result.Violations)
	}
}
