package gate

import (
	"context"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
)

// bunClassifier classifies TypeScript-style sources (the non-Go stack the
// vacuous-green regression is reproduced over): source **/*.ts, test **/*.test.ts.
func bunClassifier() SourceClassifier {
	return NewSourceClassifier([]string{"**/*.ts"}, []string{"**/*.test.ts"})
}

func violationForRule(violations []Violation, rule string) (Violation, bool) {
	for _, v := range violations {
		if v.Rule == rule {
			return v, true
		}
	}
	return Violation{}, false
}

// TestCoverage_ChangedTSSourceFileNoRecordIsLoudBlockingNotVacuousGreen (CLM-012):
// the LOAD-BEARING regression — a changed NON-Go source file (app/foo.ts) matching
// the declared source glob, with NO coverage record and not excluded, yields a LOUD
// blocking violation (severity error, status fail), NOT a silent green.
func TestCoverage_ChangedTSSourceFileNoRecordIsLoudBlockingNotVacuousGreen(t *testing.T) {
	result := StepCoverageThresholdScopedFunc(
		nil, coverageSpecs(80), diffScopeFor("app/foo.ts"), bunClassifier(),
	)(context.Background())

	if result.Status != "fail" {
		t.Fatalf("a changed .ts source file with no record must FAIL loud, not pass vacuous-green; got %s: %#v", result.Status, result.Violations)
	}
	v, ok := violationForFile(result.Violations, "app/foo.ts")
	if !ok {
		t.Fatalf("expected a loud violation for the unmeasured .ts source file, got %#v", result.Violations)
	}
	if v.Severity != "error" {
		t.Errorf("the no-record state must be a blocking error, got severity %q", v.Severity)
	}
}

// TestCoverage_NoRecordDistinctFromBelowThreshold (CLM-013): a no-record changed
// file and a below-threshold changed file produce DISTINGUISHABLE violations
// (different rule), so the report never conflates "nothing measured" with
// "measured low".
func TestCoverage_NoRecordDistinctFromBelowThreshold(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "app/low.ts", Covered: 10, Total: 100, Measured: true, Excluded: false, Metric: "line"},
	}
	result := StepCoverageThresholdScopedFunc(
		records, coverageSpecs(80), diffScopeFor("app/foo.ts", "app/low.ts"), bunClassifier(),
	)(context.Background())

	noRecord, okNR := violationForFile(result.Violations, "app/foo.ts")
	belowThreshold, okBT := violationForFile(result.Violations, "app/low.ts")
	if !okNR || !okBT {
		t.Fatalf("expected both a no-record and a below-threshold violation, got %#v", result.Violations)
	}
	if noRecord.Rule == belowThreshold.Rule {
		t.Errorf("no-record and below-threshold must be DISTINCT rules, both were %q", noRecord.Rule)
	}
}

// TestCoverage_NoRecordGuardFiresWithoutNumericThreshold (CLM-014): a changed
// measurable-source file with no record REDs even when NO positive numeric
// threshold is declared in scope (here a spec declares the file with threshold 0,
// so the floor resolves to 0) — the declared source glob is itself the promise.
// This exercises the dismantled `if threshold <= 0 { return pass }` early return.
func TestCoverage_NoRecordGuardFiresWithoutNumericThreshold(t *testing.T) {
	specs := []SpecVerification{{SpecID: "SPEC-NOFLOOR", File: "app/foo.ts", CoverageThreshold: 0}}
	result := StepCoverageThresholdScopedFunc(
		nil, specs, diffScopeFor("app/foo.ts"), bunClassifier(),
	)(context.Background())

	if result.Status != "fail" {
		t.Fatalf("the no-record guard must fire even with threshold 0 (the old `threshold<=0` early return is dismantled); got %s: %#v", result.Status, result.Violations)
	}
	if _, ok := violationForFile(result.Violations, "app/foo.ts"); !ok {
		t.Fatalf("expected a loud no-record violation despite the absent numeric threshold, got %#v", result.Violations)
	}
}

// TestCoverage_ChangedSourceWithRecordNotFlaggedByNoRecordGuard (CLM-015): a
// changed measurable-source file WITH a record present is NOT flagged by the
// no-record guard — it is handed to the threshold verdict path.
func TestCoverage_ChangedSourceWithRecordNotFlaggedByNoRecordGuard(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "app/foo.ts", Covered: 90, Total: 100, Measured: true, Excluded: false, Metric: "line"},
	}
	result := StepCoverageThresholdScopedFunc(
		records, coverageSpecs(80), diffScopeFor("app/foo.ts"), bunClassifier(),
	)(context.Background())

	if _, ok := violationForRule(result.Violations, "coverage_unmeasured"); ok {
		t.Errorf("a measurable-source file WITH a record must NOT be flagged by the no-record guard, got %#v", result.Violations)
	}
	if result.Status != "pass" {
		t.Errorf("90/100 at threshold 80 with a record present must pass, got %s: %#v", result.Status, result.Violations)
	}
}

// TestCoverage_ChangedTestFileNoRecordNotFlagged (CLM-016): a changed TEST file
// (app/foo.test.ts matching the declared test glob) with no record does NOT
// trigger the loud guard — test/non-source files carry no coverage requirement.
func TestCoverage_ChangedTestFileNoRecordNotFlagged(t *testing.T) {
	result := StepCoverageThresholdScopedFunc(
		nil, coverageSpecs(80), diffScopeFor("app/foo.test.ts"), bunClassifier(),
	)(context.Background())

	if hasViolationForFile(result.Violations, "app/foo.test.ts") {
		t.Errorf("a changed TEST file with no record must NOT trigger the loud guard, got %#v", result.Violations)
	}
	if result.Status == "fail" {
		t.Errorf("a changed test file (no measurable source in scope) must not red, got %s: %#v", result.Status, result.Violations)
	}
}

// TestCoverage_NoRecordGuardChecksAnyMetricRecordPresence (CLM-017): the guard keys
// on "no record for the path AT ALL (any metric)" — a path carrying at least one
// record under any metric is NOT a no-record error (the SPEC-044 seam).
func TestCoverage_NoRecordGuardChecksAnyMetricRecordPresence(t *testing.T) {
	records := []check.CoverageRecord{
		// A record under an arbitrary metric label — presence under ANY metric must
		// take the path off the no-record guard, leaving the per-metric verdict to
		// SPEC-044's (path, metric) model.
		{Path: "app/foo.ts", Covered: 90, Total: 100, Measured: true, Excluded: false, Metric: "branch"},
	}
	result := StepCoverageThresholdScopedFunc(
		records, coverageSpecs(80), diffScopeFor("app/foo.ts"), bunClassifier(),
	)(context.Background())

	if _, ok := violationForRule(result.Violations, "coverage_unmeasured"); ok {
		t.Errorf("a path carrying a record under ANY metric must not be a no-record error, got %#v", result.Violations)
	}
}

// TestCoverage_NoDeclaredSourceGlobsIsVisibleCapabilityAbsentNotSilentPass
// (CLM-018): with NO source globs and in-scope changed files, the step returns a
// DISTINCT, VISIBLE "classification capability absent" status/reason — never an
// unqualified pass.
func TestCoverage_NoDeclaredSourceGlobsIsVisibleCapabilityAbsentNotSilentPass(t *testing.T) {
	noGlobs := NewSourceClassifier(nil, nil)
	result := StepCoverageThresholdScopedFunc(
		nil, coverageSpecs(80), diffScopeFor("app/foo.ts"), noGlobs,
	)(context.Background())

	if result.Status == "pass" && len(result.Violations) == 0 {
		t.Fatalf("no source globs with in-scope changed files must NOT be an unqualified silent pass, got %#v", result)
	}
	v, ok := violationForRule(result.Violations, "coverage_capability_absent")
	if !ok {
		t.Fatalf("expected a DISTINCT coverage_capability_absent advisory, got %#v", result.Violations)
	}
	if v.Severity != "warning" {
		t.Errorf("the capability-absent advisory must be a warning (the existing convention), got %q", v.Severity)
	}
	if result.Status != "warning" {
		t.Errorf("the capability-absent state must surface as a warning status, got %q", result.Status)
	}
}

// TestCoverage_ClassificationCapabilityAbsentDoesNotBlock (CLM-019): the
// capability-absent state is NON-blocking (status not fail solely because
// classification is absent).
func TestCoverage_ClassificationCapabilityAbsentDoesNotBlock(t *testing.T) {
	noGlobs := NewSourceClassifier(nil, nil)
	result := StepCoverageThresholdScopedFunc(
		nil, coverageSpecs(80), diffScopeFor("app/foo.ts"), noGlobs,
	)(context.Background())

	if result.Status == "fail" {
		t.Errorf("classification-absent must be loud but NON-blocking, got fail: %#v", result.Violations)
	}
	if result.ConfigErr {
		t.Errorf("classification-absent must not set ConfigErr (it is not a broken-promise block)")
	}
}
