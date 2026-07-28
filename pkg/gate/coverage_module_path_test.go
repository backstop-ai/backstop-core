package gate

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
)

// TestCoverage_MatchesModuleQualifiedRecordToRepoRelativeScope proves the
// consumer reconciles a coverage record whose Path is module-qualified
// (e.g. "github.com/org/repo/pkg/validate/terminal.go", the form a Go
// -coverprofile yields) against a diff scope expressed in repo-relative paths
// (e.g. "pkg/validate/terminal.go"). Before the fix, exact-path indexing missed
// the module-qualified record, so a measured, well-covered changed file falsely
// red-flagged as "no coverage measurement". The record is 40/42 (95%), so at
// threshold 90 it must be a clean PASS — not a missing-record fail and not a
// below-threshold fail.
func TestCoverage_MatchesModuleQualifiedRecordToRepoRelativeScope(t *testing.T) {
	records := []check.CoverageRecord{
		{
			Path:     "github.com/backstop-ai/backstop-core/pkg/validate/terminal.go",
			Covered:  40,
			Total:    42,
			Measured: true,
			Metric:   "statement",
		},
	}
	result := runCoverage(records, 90, diffScopeFor("pkg/validate/terminal.go"))

	if hasViolationForFile(result.Violations, "pkg/validate/terminal.go") {
		v, _ := violationForFile(result.Violations, "pkg/validate/terminal.go")
		t.Fatalf("module-qualified record must satisfy the repo-relative scope path; got violation [%s] %s", v.Rule, v.Message)
	}
	if result.Status != "pass" {
		t.Errorf("expected pass (40/42 = 95%% >= 90%%), got %q with violations %v", result.Status, result.Violations)
	}
}

// TestCoverage_ModuleQualifiedBelowThresholdStillFails proves the suffix
// reconciliation does NOT swallow a genuine shortfall: a module-qualified record
// that is below threshold still fails for the repo-relative scope path (the fix
// reconciles path STYLE, it does not weaken the verdict).
func TestCoverage_ModuleQualifiedBelowThresholdStillFails(t *testing.T) {
	records := []check.CoverageRecord{
		{
			Path:     "github.com/backstop-ai/backstop-core/pkg/validate/terminal.go",
			Covered:  10,
			Total:    100,
			Measured: true,
			Metric:   "statement",
		},
	}
	result := runCoverage(records, 90, diffScopeFor("pkg/validate/terminal.go"))

	v, ok := violationForFile(result.Violations, "pkg/validate/terminal.go")
	if !ok {
		t.Fatalf("a below-threshold module-qualified record must still fail for its repo-relative scope path; got %v", result.Violations)
	}
	if v.Rule != "coverage_threshold" {
		t.Errorf("expected coverage_threshold rule, got %q", v.Rule)
	}
}
