package gate

import "testing"

// stepWith builds a synthetic StepResult of the given dimension with one violation.
func stepWith(stepName string, v Violation) StepResult {
	return StepResult{StepName: stepName, Status: "fail", Violations: []Violation{v}}
}

// remainsAfterReconcile runs computeWaiverResult over a single-violation step with
// a co-located @waiver token on the violation's line and reports whether the
// violation SURVIVES (i.e. was NOT suppressed).
func remainsAfterReconcile(t *testing.T, stepName, rule string) bool {
	t.Helper()
	v := Violation{Rule: rule, File: "app.go", Line: 5, Severity: "error"}
	accumulated := []StepResult{stepWith(stepName, v)}
	read := memLineReader("app.go", map[int]string{
		5: "code() // @waiver:" + rule + ":accepted-risk:2999-01-01",
	})
	g := &Gate{}
	g.computeWaiverResult(accumulated, read, nil, waiverTestNow)
	return stepHasViolation(accumulated, stepName, "app.go", 5, rule)
}

// TestGateWaiver_Scope_PackEnginesWaivable proves a pack_engines finding (the LIVE
// pack code-rule dimension) is waivable (CLM-041).
func TestGateWaiver_Scope_PackEnginesWaivable(t *testing.T) {
	if remainsAfterReconcile(t, StepPackEngines, "pkg/rule") {
		t.Fatal("a pack_engines finding must be waivable (suppressed by a co-located token)")
	}
}

// TestGateWaiver_Scope_SubstantivenessWaivable proves a test_substantiveness
// (source-located) finding is waivable (CLM-064).
func TestGateWaiver_Scope_SubstantivenessWaivable(t *testing.T) {
	if remainsAfterReconcile(t, StepTestSubstantiveness, "test_substantiveness") {
		t.Fatal("a test_substantiveness finding must be waivable")
	}
}

// TestGateWaiver_Scope_StatusDriftNotWaivable proves an artifact_status_drift
// finding is NOT waivable by a @waiver token (CLM-042).
func TestGateWaiver_Scope_StatusDriftNotWaivable(t *testing.T) {
	if !remainsAfterReconcile(t, StepArtifactStatusDrift, "artifact_status_drift") {
		t.Fatal("artifact_status_drift is structural and must NOT be waivable")
	}
}

// TestGateWaiver_Scope_ContractSignatureNotWaivable proves a contract_signature
// finding is NOT waivable (CLM-043).
func TestGateWaiver_Scope_ContractSignatureNotWaivable(t *testing.T) {
	if !remainsAfterReconcile(t, StepContractSignature, "contract_signature") {
		t.Fatal("contract_signature is structural and must NOT be waivable")
	}
}

// TestGateWaiver_Scope_TestVerificationNotWaivable proves a test_verification
// finding is NOT waivable (CLM-044).
func TestGateWaiver_Scope_TestVerificationNotWaivable(t *testing.T) {
	if !remainsAfterReconcile(t, StepTestVerification, "test_verification") {
		t.Fatal("test_verification is structural and must NOT be waivable")
	}
}

// TestGateWaiver_Scope_ArtifactValidationNotWaivable proves an artifact_validation
// finding is NOT waivable (CLM-065).
func TestGateWaiver_Scope_ArtifactValidationNotWaivable(t *testing.T) {
	if !remainsAfterReconcile(t, StepArtifactValidation, "artifact_validation") {
		t.Fatal("artifact_validation is structural and must NOT be waivable")
	}
}

// TestGateWaiver_Scope_CoverageFirstLineAnnotation proves a locationless
// coverage_threshold finding is waived by a @waiver:coverage_threshold token on
// the file's FIRST line (annotation convention), not a per-line source token
// (CLM-045).
func TestGateWaiver_Scope_CoverageFirstLineAnnotation(t *testing.T) {
	v := Violation{Rule: "coverage_threshold", File: "pkg/foo.go", Line: 0, Severity: "error"}
	accumulated := []StepResult{stepWith(StepCoverageThreshold, v)}
	// The token lives on the FILE's FIRST line — not co-located with a source line.
	read := memLineReader("pkg/foo.go", map[int]string{
		1: "// @waiver:coverage_threshold:deferred:2999-01-01",
	})
	g := &Gate{}
	g.computeWaiverResult(accumulated, read, nil, waiverTestNow)
	if stepHasViolation(accumulated, StepCoverageThreshold, "pkg/foo.go", 0, "coverage_threshold") {
		t.Fatal("a coverage_threshold finding must be waivable via a first-line @waiver:coverage_threshold annotation")
	}
}
