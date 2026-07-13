package gate

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// scopeSpec renders a minimal spec frontmatter whose status is the given value,
// declaring a test_command + coverage_threshold (for the coverage extraction path)
// and exactly one claim mandating the named test. Used by the ISSUE-054
// implemented-only scope tests to drive the full status vocabulary through
// ExtractSpecVerifications / ExtractMandatedTests / StepTestVerificationScopedFunc.
func scopeSpec(status, testName string) string {
	return fmt.Sprintf(`---
title: "Scope Spec"
number: SPEC-054
created: "2026-07-13"
status: %s
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: Scope test
  package: pkg/gate

verification:
  level: unit
  test_command: go test ./pkg/gate/... -race
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: Test req
    supports: cli:REQ-001

claims:
  - id: CLM-001
    requirement: REQ-001
    text: Test claim
    tests:
      - %s
---

# Scope Spec
`, status, testName)
}

// writeScopeSpec writes a scopeSpec of the given status/testName to dir.
func writeScopeSpec(t *testing.T, dir, filename, status, testName string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(scopeSpec(status, testName)), 0o644); err != nil {
		t.Fatalf("writing scope spec: %v", err)
	}
}

// TestExtractSpecVerifications_OnlyImplementedSpecsExtracted (CLM-005) proves the
// coverage_threshold feed extracts a spec's verification ONLY when the spec is
// `implemented`. Table-driven over the full status vocabulary: each spec declares
// the SAME test_command + coverage_threshold.
//   - implemented                  → 1 SpecVerification (coverage is DUE).
//   - draft / ready-for-implementation → 0 (pre-implementation false pressure removed).
//   - replaced/canceled/deprecated → 0 (terminal, unbroken).
//   - obsoleted                    → 0 (terminal per schema; previously LEAKED because
//                                    isTerminalSpecStatus omits it — now excluded via
//                                    contractsAreDue's implemented-only test).
func TestExtractSpecVerifications_OnlyImplementedSpecsExtracted(t *testing.T) {
	cases := []struct {
		status  string
		wantLen int
	}{
		{"implemented", 1},
		{"draft", 0},
		{"ready-for-implementation", 0},
		{"replaced", 0},
		{"canceled", 0},
		{"deprecated", 0},
		{"obsoleted", 0},
	}

	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			specDir := t.TempDir()
			writeScopeSpec(t, specDir, "scope.spec.md", tc.status, "TestSomething")

			specs, err := ExtractSpecVerifications(specDir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(specs) != tc.wantLen {
				t.Fatalf("status %q: expected %d SpecVerification(s), got %d: %v",
					tc.status, tc.wantLen, len(specs), specs)
			}
			if tc.wantLen == 1 && specs[0].SpecID != "SPEC-054" {
				t.Errorf("status %q: expected SpecID SPEC-054, got %q", tc.status, specs[0].SpecID)
			}
		})
	}
}

// TestTestVerification_EnforcesOnlyImplementedSpecMandatedTests (CLM-003, CLM-007)
// proves the test_verification consumer enforces a mandated test ONLY when its
// spec is `implemented`. Each case stages a spec whose single mandated test does
// NOT exist in the (empty) codeDir; discovery capability is fully present. For
// `implemented` the step raises a test_verification violation (enforced); for
// draft / ready-for-implementation / terminal statuses it raises NONE (not
// enforced — the false pressure removed; an all-draft set is a clean pass, not a
// capability warning).
func TestTestVerification_EnforcesOnlyImplementedSpecMandatedTests(t *testing.T) {
	cases := []struct {
		status          string
		wantEnforcement bool
	}{
		{"implemented", true},
		{"draft", false},
		{"ready-for-implementation", false},
		{"replaced", false},
		{"canceled", false},
		{"deprecated", false},
		{"obsoleted", false},
	}

	classifier := NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go", "**/testdata/**"})
	matcher, err := NewTestNameMatcher([]string{`^\s*func\s+(Test\w+)\s*\(`})
	if err != nil {
		t.Fatalf("NewTestNameMatcher: %v", err)
	}

	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			specDir := t.TempDir()
			codeDir := t.TempDir() // no test files — the mandated test is genuinely absent
			writeScopeSpec(t, specDir, "scope.spec.md", tc.status, "TestMandatedButMissing")

			step := StepTestVerificationScopedFunc(specDir, codeDir, nil, classifier, matcher)
			result := step(nil)

			var raised bool
			for _, v := range result.Violations {
				if v.Rule == "test_verification" {
					raised = true
				}
			}
			if raised != tc.wantEnforcement {
				t.Fatalf("status %q: expected test_verification enforcement=%v, got %v (status=%q, violations=%#v)",
					tc.status, tc.wantEnforcement, raised, result.Status, result.Violations)
			}
		})
	}
}

// TestExtractMandatedTests_PopulatesStatusAndStaysUnfiltered (CLM-002) proves that
// ExtractMandatedTests populates MandatedTest.Status from the spec frontmatter AND
// stays UNFILTERED by implementation status — a `draft` spec's mandated test is
// STILL returned (so artifact_status_drift keeps full visibility), carrying
// Status == "draft".
func TestExtractMandatedTests_PopulatesStatusAndStaysUnfiltered(t *testing.T) {
	specDir := t.TempDir()
	writeScopeSpec(t, specDir, "scope.spec.md", "draft", "TestDraftMandated")

	tests, err := ExtractMandatedTests(specDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tests) != 1 {
		t.Fatalf("expected a draft spec's mandated test to STILL be extracted (unfiltered), got %d: %v", len(tests), tests)
	}
	if tests[0].Status != "draft" {
		t.Fatalf("expected MandatedTest.Status == %q, got %q", "draft", tests[0].Status)
	}
	if tests[0].FuncName != "TestDraftMandated" {
		t.Fatalf("expected mandated test TestDraftMandated, got %q", tests[0].FuncName)
	}
}

// TestStatusDriftAdvisory_DraftLooksDeliveredStillFires (CLM-006) is the GUARD. It
// drives the REAL ResolveArtifactStatus → ExtractMandatedTests path over a temp
// projectRoot (NOT a hand-built ClassifyStatusDrift record) so it exercises the
// SHARED extractor — the only variant that catches a wrongly-scoped
// ExtractMandatedTests. A `draft` (non-terminal) spec whose single mandated test IS
// present must still raise the "looks delivered" advisory. This is the regression
// proof that the ISSUE-054 filter lives at the consumer, not the shared extractor:
// a red here after the impl means ExtractMandatedTests was wrongly scoped.
func TestStatusDriftAdvisory_DraftLooksDeliveredStillFires(t *testing.T) {
	root := t.TempDir()
	specDir := filepath.Join(root, "specs")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatalf("mkdir specs: %v", err)
	}
	writeScopeSpec(t, specDir, "scope.spec.md", "draft", "TestPresentMandated")

	res, err := ResolveArtifactStatus(root)
	if err != nil {
		t.Fatalf("ResolveArtifactStatus: %v", err)
	}

	// The mandated test IS present in the code tree.
	present := map[string]bool{"TestPresentMandated": true}
	result := ClassifyStatusDrift(res.Records, present)

	var advisoryFired bool
	for _, v := range result.Violations {
		if v.Rule == StepArtifactStatusDriftAdvisory && v.Severity == "warning" {
			advisoryFired = true
		}
	}
	if !advisoryFired {
		t.Fatalf("expected the draft-looks-delivered advisory to STILL fire after ISSUE-054 "+
			"(shared ExtractMandatedTests must stay unfiltered), got status=%q violations=%#v",
			result.Status, result.Violations)
	}
}
