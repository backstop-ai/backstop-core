package gate

import (
	"context"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
)

// step_coverage_skip_reason_test.go pins CLM-007: the coverage dimension's
// "nothing in scope to measure" early return stays a NON-BLOCKING pass, but says
// what it did not do.
//
// ISSUE-118 was filed against that line on the belief that it skipped a TEST RUN.
// It does not — this step holds no test invocation at all — but its bare `pass`
// read as verification to a human and to an agent, and that misreading is the whole
// reason the issue named the wrong mechanism. The Reason is the fix; the verdict,
// the scoring, the guards and the skip CONDITION are all unchanged.

// coverageSkipClassifier declares source globs (so the REQ-004
// classification-capability guard does not intercept) and test globs (so a test
// file is classified as a test rather than as measurable source).
func coverageSkipClassifier() SourceClassifier {
	return NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go"})
}

// coverageSkipSpec is one in-scope spec declaring a positive floor.
func coverageSkipSpec() []SpecVerification {
	return []SpecVerification{{
		SpecID:            "SPEC-118",
		TestCommand:       "go test ./...",
		CoverageThreshold: 80,
		File:              "specs/SPEC-118.spec.md",
	}}
}

// TestCoverageThreshold_NoInScopeFilesSkipIsAttributed (CLM-007): with a diff scope
// whose files are entirely test files, there is no measurable source to score. The
// step must still be a clean non-blocking pass — the skip is legitimate — but its
// Reason must name that it measured NOTHING and that this is a measurement skip,
// never a test verdict.
func TestCoverageThreshold_NoInScopeFilesSkipIsAttributed(t *testing.T) {
	scope := newGateScope("", GateScopeModeDiff,
		[]string{"pkg/gate/widget_test.go", "pkg/gate/gadget_test.go"}, nil)

	result := StepCoverageThresholdScopedFunc(nil, coverageSkipSpec(), scope, coverageSkipClassifier())(context.Background())

	// The verdict is UNCHANGED. Attribution is a Reason string, not a new failure.
	if result.Status != "pass" {
		t.Fatalf("Status = %q, want \"pass\" — the skip is legitimate and must stay non-blocking; violations: %#v", result.Status, result.Violations)
	}
	if len(result.Violations) != 0 {
		t.Fatalf("the skip reported %d violations, want 0: %#v", len(result.Violations), result.Violations)
	}

	reason := strings.ToLower(result.Reason)
	if reason == "" {
		t.Fatalf("the skip reported no Reason at all; a bare pass here is what read as verification")
	}
	// It must still convey the ORIGINAL condition — no in-scope files to measure.
	if !strings.Contains(reason, "no in-scope files to measure") {
		t.Fatalf("the Reason lost the no-in-scope-files condition: %q", result.Reason)
	}
	// And it must now convey the NOT-A-VERDICT qualification. Asserted on meaning,
	// not on punctuation: the reason must say it scored/measured nothing AND that
	// this dimension is not a test-pass verdict.
	if !strings.Contains(reason, "nothing") && !strings.Contains(reason, "scored no") {
		t.Fatalf("the Reason does not say that nothing was measured: %q", result.Reason)
	}
	if !strings.Contains(reason, "not a test-pass verdict") {
		t.Fatalf("the Reason does not qualify that this is NOT a test-pass verdict: %q", result.Reason)
	}
	// And it should point the reader at the dimension that DOES answer that.
	if !strings.Contains(reason, "test_verification") {
		t.Fatalf("the Reason does not name the dimension that reports test outcomes: %q", result.Reason)
	}
}

// TestCoverageThreshold_ScoredPathsUnaffectedBySkipReason (CLM-007, CLM-008): a
// scope WITH measurable source still scores exactly as before. The new Reason must
// never appear on a run that actually measured something — otherwise a real
// measurement would start claiming it measured nothing.
func TestCoverageThreshold_ScoredPathsUnaffectedBySkipReason(t *testing.T) {
	scope := newGateScope("", GateScopeModeDiff, []string{"pkg/gate/widget.go"}, nil)
	coverage := []check.CoverageRecord{{
		Path:     "pkg/gate/widget.go",
		Covered:  95,
		Total:    100,
		Measured: true,
		Metric:   "statement",
	}}

	result := StepCoverageThresholdScopedFunc(coverage, coverageSkipSpec(), scope, coverageSkipClassifier())(context.Background())

	if result.Status != "pass" {
		t.Fatalf("a 95%% file against an 80%% floor must pass, got %q: %#v", result.Status, result.Violations)
	}
	if strings.Contains(strings.ToLower(result.Reason), "no in-scope files to measure") {
		t.Fatalf("a run that MEASURED something reported the skip Reason: %q", result.Reason)
	}
	if strings.Contains(strings.ToLower(result.Reason), "not a test-pass verdict") {
		t.Fatalf("the skip qualification leaked onto a scored run: %q", result.Reason)
	}
}
