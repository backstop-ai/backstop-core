package gate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

// loadCoverageRecords reads a canonical []check.CoverageRecord fixture from
// pkg/gate/testdata. The fixtures encode the records SPEC-042's producer emits;
// the consumer tests drive the re-implemented step against them (no stub that
// can't go vacuous).
func loadCoverageRecords(t *testing.T, name string) []check.CoverageRecord {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read coverage fixture %s: %v", name, err)
	}
	var records []check.CoverageRecord
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatalf("parse coverage fixture %s: %v", name, err)
	}
	return records
}

// coverageSpecs is a single spec verification declaring the given threshold,
// applied to all in-scope files (the per-FILE consumer derives the floor from it).
func coverageSpecs(threshold int) []SpecVerification {
	return []SpecVerification{{SpecID: "SPEC-X", TestCommand: "go test ./...", CoverageThreshold: threshold}}
}

// runCoverage runs the re-implemented per-FILE coverage step over the given
// records/specs/scope and returns the StepResult.
func runCoverage(records []check.CoverageRecord, threshold int, scope *GateScope) StepResult {
	return StepCoverageThresholdScopedFunc(records, coverageSpecs(threshold), scope)(context.Background())
}

// diffScopeFor builds a diff-mode GateScope over the given project-relative files.
func diffScopeFor(files ...string) *GateScope {
	return newGateScope("", GateScopeModeDiff, files, nil)
}

func hasViolationForFile(violations []Violation, file string) bool {
	for _, v := range violations {
		if v.File == file {
			return true
		}
	}
	return false
}

func violationForFile(violations []Violation, file string) (Violation, bool) {
	for _, v := range violations {
		if v.File == file {
			return v, true
		}
	}
	return Violation{}, false
}

// TestCoverage_ConsumesDeclaredPerFileCoverageRecord proves the step consumes the
// canonical []check.CoverageRecord (Path/Covered/Total/Measured/Excluded/Metric,
// raw counts, file granularity) and applies the threshold per FILE — verdict
// asserted by feeding declared per-file records, with NO in-binary `go test`
// executed and no package-level modeling (CLM-003).
func TestCoverage_ConsumesDeclaredPerFileCoverageRecord(t *testing.T) {
	records := loadCoverageRecords(t, "coverage-record-rawcounts-metric.json")
	// pkg/raw/full.go is 80/100 = 80%, at the threshold ⇒ pass at 80.
	result := runCoverage(records, 80, diffScopeFor("pkg/raw/full.go"))
	if result.StepName != StepCoverageThreshold {
		t.Errorf("expected step %q, got %q", StepCoverageThreshold, result.StepName)
	}
	if result.Status != "pass" {
		t.Fatalf("80/100 at threshold 80 must pass, got %s: %#v", result.Status, result.Violations)
	}
}

// TestCoverage_RealPerFileShortfallStillReds proves a changed/new file measured
// below its declared threshold produces a blocking coverage_threshold violation
// with status fail (CLM-007) — the load-bearing non-vacuousness invariant.
func TestCoverage_RealPerFileShortfallStillReds(t *testing.T) {
	records := loadCoverageRecords(t, "coverage-record-perfile-floor.json")
	// pkg/widget/helper.go is 2/100 = 2%, far below 90.
	result := runCoverage(records, 90, diffScopeFor("pkg/widget/helper.go"))
	if result.Status != "fail" {
		t.Fatalf("a 2%% file at threshold 90 must FAIL, got %s: %#v", result.Status, result.Violations)
	}
	v, ok := violationForFile(result.Violations, "pkg/widget/helper.go")
	if !ok {
		t.Fatalf("expected a coverage_threshold violation for the shortfall file, got %#v", result.Violations)
	}
	if v.Rule != "coverage_threshold" || v.Severity != "error" {
		t.Errorf("shortfall must be a blocking coverage_threshold error, got rule=%q severity=%q", v.Rule, v.Severity)
	}
}

// TestCoverage_UnmeasuredNonExcludedPathIsLoudError proves an in-scope changed
// PATH with NO record that is NOT Excluded produces a LOUD blocking error
// (severity error, status fail), never a silent pass (CLM-008).
func TestCoverage_UnmeasuredNonExcludedPathIsLoudError(t *testing.T) {
	records := loadCoverageRecords(t, "coverage-record-unmeasured-and-excluded.json")
	// pkg/unmeasured/new.go is in scope (changed) but has NO record and is not excluded.
	result := runCoverage(records, 80, diffScopeFor("pkg/unmeasured/new.go"))
	if result.Status != "fail" {
		t.Fatalf("an unmeasured non-excluded changed path must FAIL loud, got %s: %#v", result.Status, result.Violations)
	}
	v, ok := violationForFile(result.Violations, "pkg/unmeasured/new.go")
	if !ok {
		t.Fatalf("expected a loud error for the unmeasured path, got %#v", result.Violations)
	}
	if v.Severity != "error" {
		t.Errorf("unmeasured-non-excluded path must be severity error, got %q", v.Severity)
	}
	if !strings.Contains(v.Message, "no coverage measurement") {
		t.Errorf("the loud error must name the unmeasured condition, got %q", v.Message)
	}
}

// TestCoverage_DeclaredExcludedPathIsSkippedNotErrored proves a pack-DECLARED-
// excluded path with no measurement is SKIPPED from the threshold check, not
// errored (CLM-008). The exclusion of an UNCHANGED file stays quiet.
func TestCoverage_DeclaredExcludedPathIsSkippedNotErrored(t *testing.T) {
	records := loadCoverageRecords(t, "coverage-record-unmeasured-and-excluded.json")
	// pkg/excluded/generated.go is Excluded:true; pkg/measured/ok.go is 90/100.
	// Neither is the unmeasured-non-excluded vector, so the step passes and the
	// excluded UNCHANGED path produces NO violation.
	result := runCoverage(records, 80, diffScopeFor("pkg/measured/ok.go"))
	if result.Status != "pass" {
		t.Fatalf("a declared-excluded path must not error; expected pass, got %s: %#v", result.Status, result.Violations)
	}
	if hasViolationForFile(result.Violations, "pkg/excluded/generated.go") {
		t.Errorf("an UNCHANGED declared-excluded path must stay quiet, got %#v", result.Violations)
	}
}

// TestCoverage_PerChangedFile_OverFloorPassesRegardlessOfSiblings proves an
// over-floor changed FILE PASSES even when a sibling file in its directory is
// below floor (CLM-009) — per-changed-file, no aggregation.
func TestCoverage_PerChangedFile_OverFloorPassesRegardlessOfSiblings(t *testing.T) {
	records := loadCoverageRecords(t, "coverage-record-perfile-floor.json")
	// widget.go (95/100) and helper.go (2/100) are siblings in pkg/widget. Scope to
	// ONLY widget.go: it passes regardless of helper.go's shortfall.
	result := runCoverage(records, 90, diffScopeFor("pkg/widget/widget.go"))
	if result.Status != "pass" {
		t.Fatalf("an over-floor file must pass regardless of a below-floor sibling, got %s: %#v", result.Status, result.Violations)
	}
}

// TestCoverage_PerChangedFile_UnderFloorFailsHiddenByPackageAggregate proves an
// under-floor (2%) changed FILE in an otherwise-95% directory FAILS — never
// rescued by a directory/package roll-up (CLM-010).
func TestCoverage_PerChangedFile_UnderFloorFailsHiddenByPackageAggregate(t *testing.T) {
	records := loadCoverageRecords(t, "coverage-record-perfile-floor.json")
	// Scope BOTH siblings: a package aggregate would be (95+2)/200 ~= 48.5% (fail)
	// OR if averaged differently could mask; the point is helper.go ALONE reds.
	result := runCoverage(records, 90, diffScopeFor("pkg/widget/widget.go", "pkg/widget/helper.go"))
	if result.Status != "fail" {
		t.Fatalf("a 2%% file in an otherwise-95%% directory must FAIL per-file, got %s: %#v", result.Status, result.Violations)
	}
	if !hasViolationForFile(result.Violations, "pkg/widget/helper.go") {
		t.Errorf("the under-floor file must be the one flagged, got %#v", result.Violations)
	}
	if hasViolationForFile(result.Violations, "pkg/widget/widget.go") {
		t.Errorf("the over-floor sibling must NOT be flagged, got %#v", result.Violations)
	}
}

// TestCoverage_DeclaredExclusionOfChangedFileIsLoudlySurfaced proves a pack-
// declared exclusion of an IN-SCOPE CHANGED file surfaces the excluded path AND
// its declared reason on the report, never silently dropped (CLM-025).
func TestCoverage_DeclaredExclusionOfChangedFileIsLoudlySurfaced(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "pkg/gen/changed.go", Covered: 0, Total: 0, Measured: false, Excluded: true, Metric: "generated"},
	}
	result := runCoverage(records, 80, diffScopeFor("pkg/gen/changed.go"))
	v, ok := violationForFile(result.Violations, "pkg/gen/changed.go")
	if !ok {
		t.Fatalf("a declared exclusion of an IN-SCOPE CHANGED file must be surfaced, got %#v", result.Violations)
	}
	if !strings.Contains(v.Message, "suppressed") || !strings.Contains(v.Message, "generated") {
		t.Errorf("the surfaced exclusion must name the path AND the declared reason, got %q", v.Message)
	}
	// Loud-but-not-blocking: the changed-file exclusion is surfaced as a warning,
	// not a blocking error (the threshold check is skipped but the suppression is
	// visible). It must NOT be silently dropped.
	if v.Severity == "error" {
		t.Errorf("changed-file exclusion is surfaced as a warning, not a blocking error; got %q", v.Severity)
	}
}

// TestCoverage_DeclaredExclusionOfUnchangedFileMayStayQuiet proves an exclusion of
// an UNCHANGED file MAY stay quiet — it is not the suppression vector (CLM-025).
func TestCoverage_DeclaredExclusionOfUnchangedFileMayStayQuiet(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "pkg/changed/ok.go", Covered: 90, Total: 100, Measured: true, Excluded: false, Metric: "statement"},
		{Path: "pkg/gen/unchanged.go", Covered: 0, Total: 0, Measured: false, Excluded: true, Metric: "generated"},
	}
	// Scope to ONLY the changed measured file; the excluded UNCHANGED file is not in scope.
	result := runCoverage(records, 80, diffScopeFor("pkg/changed/ok.go"))
	if result.Status != "pass" {
		t.Fatalf("expected pass, got %s: %#v", result.Status, result.Violations)
	}
	if hasViolationForFile(result.Violations, "pkg/gen/unchanged.go") {
		t.Errorf("an UNCHANGED-file exclusion may stay quiet, got %#v", result.Violations)
	}
}

// TestCoverage_GateComputesRatioFromRawCountsMetricBlind proves the gate computes
// Covered/Total >= threshold from RAW COUNTS (not a pre-computed percent),
// metric-blind (CLM-026).
func TestCoverage_GateComputesRatioFromRawCountsMetricBlind(t *testing.T) {
	// 7/10 = 70% — below 80 ⇒ fail; the gate derives this from raw counts, never a
	// supplied percent, and never interprets the Metric label.
	below := []check.CoverageRecord{{Path: "pkg/raw/seven.go", Covered: 7, Total: 10, Measured: true, Excluded: false, Metric: "branch"}}
	if got := runCoverage(below, 80, diffScopeFor("pkg/raw/seven.go")); got.Status != "fail" {
		t.Fatalf("7/10 at threshold 80 must fail from raw counts, got %s: %#v", got.Status, got.Violations)
	}
	// 8/10 = 80% — exactly at threshold ⇒ pass (boundary computed from raw counts).
	at := []check.CoverageRecord{{Path: "pkg/raw/eight.go", Covered: 8, Total: 10, Measured: true, Excluded: false, Metric: "statement"}}
	if got := runCoverage(at, 80, diffScopeFor("pkg/raw/eight.go")); got.Status != "pass" {
		t.Fatalf("8/10 at threshold 80 must pass from raw counts, got %s: %#v", got.Status, got.Violations)
	}
}

// TestCoverage_TotalZeroIsNANotZeroPercentFail proves a record with Total==0 (no
// executable lines) is N/A (skipped), NEVER a 0%-fail (CLM-026).
func TestCoverage_TotalZeroIsNANotZeroPercentFail(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "pkg/raw/decls.go", Covered: 0, Total: 0, Measured: true, Excluded: false, Metric: "statement"},
	}
	result := runCoverage(records, 80, diffScopeFor("pkg/raw/decls.go"))
	if result.Status != "pass" {
		t.Fatalf("Total==0 must be N/A (skipped), never a 0%%-fail; got %s: %#v", result.Status, result.Violations)
	}
	if hasViolationForFile(result.Violations, "pkg/raw/decls.go") {
		t.Errorf("a Total==0 file must produce NO violation, got %#v", result.Violations)
	}
}

// TestCoverage_MetricLabelSurfacedNeverInterpreted proves the pack-declared Metric
// label is surfaced on the report and never interpreted/compared/branched on; a
// polyglot set with differing Metric values surfaces each file's metric (CLM-027).
func TestCoverage_MetricLabelSurfacedNeverInterpreted(t *testing.T) {
	// Two files below floor with DIFFERING metrics (statement vs branch). Both must
	// red, each surfacing its OWN metric label — the gate never collapses them under
	// one number nor branches on the metric.
	records := []check.CoverageRecord{
		{Path: "a.go", Covered: 50, Total: 100, Measured: true, Excluded: false, Metric: "statement"},
		{Path: "b.go", Covered: 50, Total: 100, Measured: true, Excluded: false, Metric: "branch"},
	}
	result := runCoverage(records, 90, diffScopeFor("a.go", "b.go"))
	if result.Status != "fail" {
		t.Fatalf("both below-floor files must red, got %s: %#v", result.Status, result.Violations)
	}
	va, _ := violationForFile(result.Violations, "a.go")
	vb, _ := violationForFile(result.Violations, "b.go")
	if !strings.Contains(va.Message, "statement") {
		t.Errorf("a.go's report must surface its statement metric, got %q", va.Message)
	}
	if !strings.Contains(vb.Message, "branch") {
		t.Errorf("b.go's report must surface its branch metric, got %q", vb.Message)
	}
}

// TestCoverage_PackagePathMatches exercises the RETAINED, language-agnostic
// threshold-derivation helper (re-keyed from the deleted per-package model).
func TestCoverage_PackagePathMatches(t *testing.T) {
	cases := []struct {
		changedDir, specPkg string
		want                bool
	}{
		{"pkg/gate", "pkg/gate", true},
		{"pkg/gate/sub", "pkg/gate", true},
		{"pkg/gate", "./pkg/gate", true},
		{"pkg/other", "pkg/gate", false},
		{"", "", true},
	}
	for _, c := range cases {
		if got := packagePathMatches(c.changedDir, c.specPkg); got != c.want {
			t.Errorf("packagePathMatches(%q,%q)=%v want %v", c.changedDir, c.specPkg, got, c.want)
		}
	}
}

// TestCoverage_BelowThresholdRawCountArithmetic pins the integer ratio comparison
// (no floating-point drift at the boundary).
func TestCoverage_BelowThresholdRawCountArithmetic(t *testing.T) {
	if coverageBelowThreshold(80, 100, 80) {
		t.Error("80/100 must NOT be below threshold 80")
	}
	if !coverageBelowThreshold(79, 100, 80) {
		t.Error("79/100 must be below threshold 80")
	}
	if coverageBelowThreshold(1, 3, 33) {
		t.Error("1/3 (~33.3%) must NOT be below threshold 33")
	}
	if !coverageBelowThreshold(1, 3, 34) {
		t.Error("1/3 (~33.3%) must be below threshold 34")
	}
}
