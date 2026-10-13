package gate

// ISSUE-217: the code-scope fallback coverage threshold (the "floor") is a
// configurable value threaded in through WithCoverageFloor, not a core constant.

import (
	"context"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
)

// floorRecord is one measured line-coverage record for path at covered/100.
func floorRecord(path string, covered int) []check.CoverageRecord {
	return []check.CoverageRecord{{Path: path, Covered: covered, Total: 100, Measured: true, Metric: "line"}}
}

// belowBarViolations returns the coverage_threshold (below-bar) violations for file.
func belowBarViolations(violations []Violation, file string) []Violation {
	var out []Violation
	for _, v := range violations {
		if v.Rule == "coverage_threshold" && v.File == file {
			out = append(out, v)
		}
	}
	return out
}

// TestCoverageFloorConfiguredUsedForFilesWithoutSpec (CLM-002): for a changed
// file with no qualifying spec threshold, the configured floor is the bar — a
// floor below the measured value passes, one above it fails at that floor, 0
// means no floor, and no option keeps today's 90.
func TestCoverageFloorConfiguredUsedForFilesWithoutSpec(t *testing.T) {
	const file = "app/legacy.ts"
	scope := diffScopeFor(file)

	t.Run("floor below measured passes", func(t *testing.T) {
		res := StepCoverageThresholdScopedFunc(floorRecord(file, 60), nil, scope, bunClassifier(), WithCoverageFloor(50))(context.Background())
		if res.Status != "pass" {
			t.Fatalf("a 60%% file with floor 50 must pass, got %s: %#v", res.Status, res.Violations)
		}
		if got := belowBarViolations(res.Violations, file); len(got) != 0 {
			t.Errorf("no below-bar violation expected at floor 50, got %#v", got)
		}
	})

	t.Run("floor above measured fails at the configured bar", func(t *testing.T) {
		res := StepCoverageThresholdScopedFunc(floorRecord(file, 60), nil, scope, bunClassifier(), WithCoverageFloor(70))(context.Background())
		if res.Status != "fail" {
			t.Fatalf("a 60%% file with floor 70 must fail, got %s: %#v", res.Status, res.Violations)
		}
		got := belowBarViolations(res.Violations, file)
		if len(got) != 1 {
			t.Fatalf("want exactly one below-bar violation for %s, got %#v", file, res.Violations)
		}
		if !strings.Contains(got[0].Message, "below threshold 70%") {
			t.Errorf("the violation must name the configured bar 70, got %q", got[0].Message)
		}
	})

	t.Run("floor zero means no floor", func(t *testing.T) {
		res := StepCoverageThresholdScopedFunc(floorRecord(file, 10), nil, scope, bunClassifier(), WithCoverageFloor(0))(context.Background())
		if got := belowBarViolations(res.Violations, file); len(got) != 0 {
			t.Errorf("coverage_floor 0 must apply no bar, but a 10%% file produced %#v", got)
		}
		if res.Status != "pass" {
			t.Errorf("a measured file with no floor must pass, got %s: %#v", res.Status, res.Violations)
		}
	})

	t.Run("no option keeps the default 90", func(t *testing.T) {
		res := StepCoverageThresholdScopedFunc(floorRecord(file, 89), nil, scope, bunClassifier())(context.Background())
		got := belowBarViolations(res.Violations, file)
		if res.Status != "fail" || len(got) != 1 || !strings.Contains(got[0].Message, "below threshold 90%") {
			t.Fatalf("with no floor option an 89%% file must fail below 90, got %s: %#v", res.Status, res.Violations)
		}
		pass := StepCoverageThresholdScopedFunc(floorRecord(file, 90), nil, scope, bunClassifier())(context.Background())
		if pass.Status != "pass" {
			t.Errorf("with no floor option a 90%% file must pass, got %s: %#v", pass.Status, pass.Violations)
		}
	})

	t.Run("project-wide spec with no threshold still uses the floor", func(t *testing.T) {
		specs := []SpecVerification{{SpecID: "SPEC-NOTHRESH", CoverageThreshold: 0}}
		res := StepCoverageThresholdScopedFunc(floorRecord(file, 60), specs, scope, bunClassifier(), WithCoverageFloor(70))(context.Background())
		got := belowBarViolations(res.Violations, file)
		if len(got) != 1 || !strings.Contains(got[0].Message, "below threshold 70%") {
			t.Fatalf("a spec declaring no threshold must fall back to the configured floor 70, got %#v", res.Violations)
		}
	})

	t.Run("gate --all does not apply the floor", func(t *testing.T) {
		all := newGateScope("", GateScopeModeAll, []string{file}, nil)
		res := StepCoverageThresholdScopedFunc(floorRecord(file, 10), nil, all, bunClassifier(), WithCoverageFloor(70))(context.Background())
		if got := belowBarViolations(res.Violations, file); len(got) != 0 {
			t.Errorf("--all keeps today's behavior (no floor reached), got %#v", got)
		}
	})
}

// TestCoverageFloorSpecThresholdStillWins (CLM-003): a qualifying spec-declared
// coverage_threshold governs even when it is BELOW the configured floor, on both
// resolution paths.
func TestCoverageFloorSpecThresholdStillWins(t *testing.T) {
	const file = "app/widget/foo.ts"

	t.Run("spec file in diff", func(t *testing.T) {
		specs := []SpecVerification{{SpecID: "SPEC-IN-DIFF", File: "specs/SPEC-IN-DIFF.spec.md", CoverageThreshold: 60}}
		scope := diffScopeFor(file, "specs/SPEC-IN-DIFF.spec.md")
		res := StepCoverageThresholdScopedFunc(floorRecord(file, 70), specs, scope, bunClassifier(), WithCoverageFloor(95))(context.Background())
		if res.Status != "pass" {
			t.Fatalf("spec threshold 60 must win over floor 95 for a 70%% file, got %s: %#v", res.Status, res.Violations)
		}
		// Contrast: the same spec still bites below its own value.
		low := StepCoverageThresholdScopedFunc(floorRecord(file, 50), specs, scope, bunClassifier(), WithCoverageFloor(95))(context.Background())
		got := belowBarViolations(low.Violations, file)
		if len(got) != 1 || !strings.Contains(got[0].Message, "below threshold 60%") {
			t.Errorf("a 50%% file must fail at the spec's 60 (not the floor 95), got %#v", low.Violations)
		}
	})

	t.Run("spec matched by implementation package", func(t *testing.T) {
		specs := []SpecVerification{{SpecID: "SPEC-PKG", ImplementationPackage: "app/widget", CoverageThreshold: 60}}
		scope := diffScopeFor(file)
		res := StepCoverageThresholdScopedFunc(floorRecord(file, 70), specs, scope, bunClassifier(), WithCoverageFloor(95))(context.Background())
		if res.Status != "pass" {
			t.Fatalf("package-matched spec threshold 60 must win over floor 95 for a 70%% file, got %s: %#v", res.Status, res.Violations)
		}
		low := StepCoverageThresholdScopedFunc(floorRecord(file, 50), specs, scope, bunClassifier(), WithCoverageFloor(95))(context.Background())
		got := belowBarViolations(low.Violations, file)
		if len(got) != 1 || !strings.Contains(got[0].Message, "below threshold 60%") {
			t.Errorf("a 50%% file must fail at the package spec's 60 (not the floor 95), got %#v", low.Violations)
		}
	})

	t.Run("non-matching package spec falls to the floor", func(t *testing.T) {
		specs := []SpecVerification{{SpecID: "SPEC-OTHER", ImplementationPackage: "app/other", CoverageThreshold: 60}}
		res := StepCoverageThresholdScopedFunc(floorRecord(file, 70), specs, diffScopeFor(file), bunClassifier(), WithCoverageFloor(95))(context.Background())
		got := belowBarViolations(res.Violations, file)
		if len(got) != 1 || !strings.Contains(got[0].Message, "below threshold 95%") {
			t.Errorf("a spec for another package does not qualify; the floor 95 must apply, got %#v", res.Violations)
		}
	})
}
