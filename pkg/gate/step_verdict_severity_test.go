package gate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// step_verdict_severity_test.go — ISSUE-105. Step builders decided PASS/FAIL by RAW
// COUNT and never read Violation.Severity, so the ratified pack severity contract (a
// SARIF `level: warning` is NON-BLOCKING BY CONTRACT, see blocksVerdict) held only for
// consumers who happened to declare an enforcement.policy entry for that dimension.
// These lock the STEP-LEVEL half: the step is the first and default authority on its
// own verdict, and it now decides severity-aware whether or not a policy layer runs
// afterwards.

// TestStepVerdict_WarningOnlyFindingsReportWarningNotFail pins the helper's tri-state
// (CLM-001/CLM-003).
//
// The "not pass" half is the load-bearing one. Returning "pass" for a warning-only step
// would make a surfaced non-blocking finding indistinguishable from a clean run: it
// would stop incrementing GateResult.StepsWarned, vanish from the human summary line,
// and turn "loud, non-blocking" into "silent" — the inverse defect, and no better.
func TestStepVerdict_WarningOnlyFindingsReportWarningNotFail(t *testing.T) {
	warningOnly := []Violation{warningViolation("notice-a"), warningViolation("notice-b")}
	if got := StepVerdict(warningOnly); got != "warning" {
		t.Errorf("a step whose findings are ALL severity=warning must report %q, got %q; "+
			"a warning that fails the gate is a contradiction in terms", "warning", got)
	}
	if got := StepVerdict(warningOnly); got == "pass" {
		t.Error("warning-only reported \"pass\": the finding would drop out of StepsWarned and " +
			"the summary line, making a surfaced notice invisible")
	}

	if got := StepVerdict(nil); got != "pass" {
		t.Errorf("no findings at all must report %q, got %q", "pass", got)
	}
	if got := StepVerdict([]Violation{}); got != "pass" {
		t.Errorf("an empty (non-nil) finding slice must report %q, got %q", "pass", got)
	}
}

// TestStepVerdict_BlockingSeverityFailsIncludingUnsetSeverity guards the other
// direction, so "severity-aware" cannot quietly become "relaxed" (CLM-004).
//
// The unset-severity case is the fail-closed floor inherited unchanged from
// blocksVerdict: SARIF makes `level` OPTIONAL, so a producer that declares nothing must
// block rather than silently escape enforcement.
func TestStepVerdict_BlockingSeverityFailsIncludingUnsetSeverity(t *testing.T) {
	if got := StepVerdict([]Violation{errorViolation("a-real-defect")}); got != "fail" {
		t.Errorf("an error-severity finding must report %q, got %q", "fail", got)
	}

	unset := []Violation{{Rule: "no-declared-severity", Message: "severity omitted", File: "c.go"}}
	if got := StepVerdict(unset); got != "fail" {
		t.Errorf("a finding with NO declared severity must FAIL CLOSED and report %q, got %q; "+
			"a pack must not be able to disable enforcement by declaring nothing", "fail", got)
	}

	mixed := []Violation{warningViolation("notice"), errorViolation("defect")}
	if got := StepVerdict(mixed); got != "fail" {
		t.Errorf("a mixed set must FAIL on its blocking entry, got %q", got)
	}
}

// TestStepArtifactValidation_DeclaredWarningDoesNotFailWithoutPolicy drives the real
// StepArtifactValidationScopedFunc with NO policy layer involved at all (CLM-002/003).
//
// The delegate is stubbed to return exactly one severity=warning violation — the shape
// ISSUE-105 measured. RED before the fix: status "fail", because the step counted.
func TestStepArtifactValidation_DeclaredWarningDoesNotFailWithoutPolicy(t *testing.T) {
	validator := &mockValidator{violations: []Violation{warningViolation("advisory")}}
	result := StepArtifactValidationScopedFunc(validator, nil)(context.Background())

	if result.Status != "warning" {
		t.Errorf("a declared-warning artifact finding must report %q with NO policy entry, got %q; "+
			"the severity contract belongs to the FINDING, not to adopter configuration",
			"warning", result.Status)
	}
	if len(result.Violations) != 1 {
		t.Errorf("non-blocking must not mean invisible: expected the warning to still be REPORTED, "+
			"got %d violations", len(result.Violations))
	}
	assertNonBlockingResult(t, result)

	// The error twin: narrowing by SEVERITY, not simply blocking less.
	blocking := &mockValidator{violations: []Violation{errorViolation("schema")}}
	blockingResult := StepArtifactValidationScopedFunc(blocking, nil)(context.Background())
	if blockingResult.Status != "fail" {
		t.Errorf("an error-severity artifact finding must still report %q, got %q",
			"fail", blockingResult.Status)
	}
}

// TestStepCodeCheck_DeclaredWarningDoesNotFailWithoutPolicy is the same probe over the
// second delegate site (CLM-002/003). Both live in step_delegate.go and both counted.
func TestStepCodeCheck_DeclaredWarningDoesNotFailWithoutPolicy(t *testing.T) {
	checker := &mockChecker{violations: []Violation{warningViolation("style-advice")}}
	result := StepCodeCheckScopedFunc(checker, nil)(context.Background())

	if result.Status != "warning" {
		t.Errorf("a declared-warning code finding must report %q with NO policy entry, got %q",
			"warning", result.Status)
	}
	if len(result.Violations) != 1 {
		t.Errorf("expected the warning to still be REPORTED, got %d violations", len(result.Violations))
	}
	assertNonBlockingResult(t, result)

	blocking := &mockChecker{violations: []Violation{errorViolation("lint")}}
	blockingResult := StepCodeCheckScopedFunc(blocking, nil)(context.Background())
	if blockingResult.Status != "fail" {
		t.Errorf("an error-severity code finding must still report %q, got %q", "fail", blockingResult.Status)
	}
}

// TestDelegateSteps_ConfigErrorStillFailsRegardlessOfSeverity is the guard against
// "severity-aware" quietly becoming "relaxed" (CLM-006).
//
// A ConfigError is a BROKEN EXECUTION, not a severity question. Those branches build
// their StepResult directly with Status "fail" and must never route through StepVerdict
// — ApplyPolicy already refuses to relax a ConfigErr step, and the step level must not
// open a hole underneath it.
func TestDelegateSteps_ConfigErrorStillFailsRegardlessOfSeverity(t *testing.T) {
	cfgErr := &ConfigError{Err: errors.New("backstop.yml is unreadable")}

	validatorResult := StepArtifactValidationScopedFunc(&mockValidator{err: cfgErr}, nil)(context.Background())
	if validatorResult.Status != "fail" || !validatorResult.ConfigErr {
		t.Errorf("a config error from the artifact validator must report fail+ConfigErr, got status=%q ConfigErr=%v",
			validatorResult.Status, validatorResult.ConfigErr)
	}

	checkerResult := StepCodeCheckScopedFunc(&mockChecker{err: cfgErr}, nil)(context.Background())
	if checkerResult.Status != "fail" || !checkerResult.ConfigErr {
		t.Errorf("a config error from the code checker must report fail+ConfigErr, got status=%q ConfigErr=%v",
			checkerResult.Status, checkerResult.ConfigErr)
	}
}

// TestStepContractSignature_DeclaredWarningDoesNotFailWithoutPolicy drives the real
// StepContractSignatureScopedFunc (CLM-002).
//
// MEASURED WHILE IMPLEMENTING, AND IT CONTRADICTS THE PLAN'S SITE CLASSIFICATION: this
// site cannot be handed a declared warning at all. Its violations have exactly ONE
// source — VerifyContractVerdict (contract_verdict.go) — and all three of its violation
// returns HARDCODE Severity "error"; ContractEngineResult carries no severity field for
// a pack to populate. So contract_signature is structurally warning-free (the plan's
// CLASS 3) rather than a slice carrying a pack-resolved severity (CLASS 1), and routing
// it through StepVerdict is behavior-PRESERVING single-authority maintenance today, not
// a behavior change.
//
// The assertion is therefore the invariant the conversion establishes rather than a
// severity flip that cannot occur: the reported Status must EQUAL StepVerdict over the
// step's own reported violations. That is false for a raw-count implementation the
// instant any severity reaches this slice, which is precisely how ISSUE-105 arrived at
// the other sites. The contingent half — that a warning here WOULD be non-blocking once
// severity flows — is asserted directly on the helper, since no input can produce it.
func TestStepContractSignature_DeclaredWarningDoesNotFailWithoutPolicy(t *testing.T) {
	raising := ContractEngineResult{
		Entry:   ContractEntry{File: "pkg/gate/policy.go", Name: "StepVerdict", Kind: "function", Signature: "func StepVerdict(violations []Violation) string"},
		Matched: false,
		Scanned: true,
	}
	result := StepContractSignatureScopedFunc([]ContractEngineResult{raising}, nil)(context.Background())

	if len(result.Violations) != 1 {
		t.Fatalf("expected the unmatched present-contract to raise exactly 1 violation, got %d", len(result.Violations))
	}
	if result.Status != StepVerdict(result.Violations) {
		t.Errorf("the contract step's status (%q) is not the severity-aware verdict over its own "+
			"reported violations (%q); a raw count here would discard any severity that ever "+
			"reaches this slice", result.Status, StepVerdict(result.Violations))
	}
	if result.Status != "fail" {
		t.Errorf("an error-severity contract violation must still report %q, got %q", "fail", result.Status)
	}
	if result.Violations[0].Severity != "error" {
		t.Errorf("this test's premise is that VerifyContractVerdict hardcodes error severity; it "+
			"returned %q. If severity now flows here, the CLASS-3 reading above is stale and this "+
			"site needs a genuine warning-input test", result.Violations[0].Severity)
	}

	// The contingent half, asserted where it is constructible: were a declared warning
	// ever to reach this slice, the converted step reports it non-blocking.
	if got := StepVerdict([]Violation{warningViolation("contract-advisory")}); got != "warning" {
		t.Errorf("a declared-warning contract finding must be non-blocking, got %q", got)
	}
}

// TestStepContractSignature_ScopeSkipAndEmptyResults covers the branches the violation
// test does not reach (CLM-002/CLM-009).
//
// This is NOT padding. StepContractSignatureScopedFunc sat at 0.0% statement coverage
// (13 statements, zero covered) and it is the ONLY function in step_contract.go, so
// touching that file activates all of it against the per-file threshold. Between the two
// tests this covers: a raised violation, a SATISFIED result, a result whose Entry.File is
// outside a non-all scope (the `continue`), and a call with no results (the pass +
// non-nil empty slice return).
func TestStepContractSignature_ScopeSkipAndEmptyResults(t *testing.T) {
	// No results at all: pass, and a non-nil empty slice rather than nil.
	empty := StepContractSignatureScopedFunc(nil, nil)(context.Background())
	if empty.Status != "pass" {
		t.Errorf("no contract results must report %q, got %q", "pass", empty.Status)
	}
	if empty.Violations == nil {
		t.Error("the step must normalize nil violations to an empty slice; a nil slice leaks into " +
			"the report shape")
	}
	if len(empty.Violations) != 0 {
		t.Errorf("expected no violations, got %d", len(empty.Violations))
	}

	// A SATISFIED contract raises nothing.
	satisfied := ContractEngineResult{
		Entry:   ContractEntry{File: "pkg/gate/policy.go", Name: "blocksVerdict"},
		Matched: true,
		Scanned: true,
	}
	satisfiedResult := StepContractSignatureScopedFunc([]ContractEngineResult{satisfied}, nil)(context.Background())
	if satisfiedResult.Status != "pass" || len(satisfiedResult.Violations) != 0 {
		t.Errorf("a matched present-contract must be satisfied, got status=%q violations=%d",
			satisfiedResult.Status, len(satisfiedResult.Violations))
	}

	// SCOPE SKIP: a would-be violation whose Entry.File is OUTSIDE a non-all scope is
	// skipped entirely (the `continue`), so the step passes.
	root := t.TempDir()
	outOfScope := ContractEngineResult{
		Entry:   ContractEntry{File: "pkg/gate/untouched.go", Name: "Untouched", Signature: "func Untouched()"},
		Matched: false,
		Scanned: true,
	}
	scope := newGateScope(root, GateScopeModeDiff, []string{"pkg/gate/something_else.go"}, nil)
	skipped := StepContractSignatureScopedFunc([]ContractEngineResult{outOfScope}, scope)(context.Background())
	if skipped.Status != "pass" || len(skipped.Violations) != 0 {
		t.Errorf("a contract result outside a diff scope must be skipped, got status=%q violations=%d",
			skipped.Status, len(skipped.Violations))
	}

	// The same result INSIDE the scope still raises — otherwise the skip above would be
	// indistinguishable from the verdict being broken.
	inScope := newGateScope(root, GateScopeModeDiff, []string{"pkg/gate/untouched.go"}, nil)
	raised := StepContractSignatureScopedFunc([]ContractEngineResult{outOfScope}, inScope)(context.Background())
	if raised.Status != "fail" || len(raised.Violations) != 1 {
		t.Errorf("the same result INSIDE scope must raise, got status=%q violations=%d",
			raised.Status, len(raised.Violations))
	}
}

// TestClass3Sites_ViolationsAreErrorSeverityByConstruction is the severity lock for
// three sites whose blocking violations must stay error-severity (CLM-007).
//
// WHY IT EXISTS: each site's blocking severity is an invariant someone can break in ONE
// LINE. A one-line edit making it false would silently change what those sites block on,
// at a site nobody is watching — which is exactly how ISSUE-105 arrived elsewhere. It is
// asserted as close to the deciding code as the site allows, not over hand-built inputs,
// which is what keeps it from rotting into a tautology.
//
// WHAT THIS TEST DOES *NOT* TRACK, stated because an earlier version of this docstring
// claimed it and was wrong: whether each site's CONSUMER still reaches its verdict by raw
// count or now calls StepVerdict. That has drifted per site and is not this test's
// subject. DO NOT reintroduce a count of raw-count sites here — a fresh number is just a
// fresh thing to go stale, and a claim that outlived its truth is precisely the defect
// class this file exists to guard.
func TestClass3Sites_ViolationsAreErrorSeverityByConstruction(t *testing.T) {
	// SITE 1 — the waiver step's blocking diagnostics (step_waiver.go).
	//
	// ASSERTED AT THE CALL SITE, NOT AT THE CONVERTER, AND THAT IS THE POINT.
	// waiverDiagToViolation no longer hardcodes a severity: ISSUE-097 added an UNBOUND
	// diagnostic kind that is deliberately non-blocking, so severity became a parameter
	// and asserting it on the converter would now assert nothing but the argument just
	// passed in. What still has to hold is that the step's two BLOCKING kinds — malformed
	// and non-waivable — reach the report at severity "error", and that the step's verdict
	// is derived from those severities rather than from a raw count of them. The raw count
	// this block once protected is gone; a count would report the new warning kind as a
	// hard gate failure.
	malformedRes := (&Gate{}).computeWaiverResult(
		[]StepResult{packEnginesStep(Violation{Rule: "pkg/rule-a", File: "app.go", Line: 5, Severity: "error"})},
		memLineReader("app.go", map[int]string{
			5: "risky() // @waiver:pkg/rule-a:not-a-reason-code:2999-01-01",
		}),
		nil, waiverTestNow,
	)
	if len(malformedRes.Violations) != 1 {
		t.Fatalf("a malformed token must surface as exactly one waiver_resolution violation, got %d (%#v)",
			len(malformedRes.Violations), malformedRes.Violations)
	}
	if got := malformedRes.Violations[0].Severity; got != "error" {
		t.Errorf("a MALFORMED waiver diagnostic reached the report at severity %q, want \"error\"; "+
			"malformed is a broken promise, not rot, and softening it here would let a broken token "+
			"pass unnoticed", got)
	}
	if malformedRes.Status != StepVerdict(malformedRes.Violations) {
		t.Errorf("the waiver step's status (%q) is not the severity-aware verdict over its own "+
			"reported violations (%q)", malformedRes.Status, StepVerdict(malformedRes.Violations))
	}
	if malformedRes.Status != "fail" {
		t.Errorf("a malformed token must still FAIL the step, got %q", malformedRes.Status)
	}

	nonWaivableRes := (&Gate{}).computeWaiverResult(
		[]StepResult{packEnginesStep(Violation{Rule: "some/pack/protected", File: "app.go", Line: 5, Severity: "error"})},
		memLineReader("app.go", map[int]string{
			5: "risky() // @waiver:some/pack/protected:accepted-risk:2999-01-01",
		}),
		newTestPolicyNonWaivable("some/pack/protected"), waiverTestNow,
	)
	if len(nonWaivableRes.Violations) != 1 {
		t.Fatalf("a token on a declared non-waivable rule must surface as exactly one violation, got %d (%#v)",
			len(nonWaivableRes.Violations), nonWaivableRes.Violations)
	}
	if got := nonWaivableRes.Violations[0].Severity; got != "error" {
		t.Errorf("a NON-WAIVABLE waiver diagnostic reached the report at severity %q, want \"error\"; "+
			"a waiver on a protected rule is a gate error by REQ-006", got)
	}
	if nonWaivableRes.Status != "fail" {
		t.Errorf("a non-waivable token must still FAIL the step, got %q", nonWaivableRes.Status)
	}

	// SITE 2 — the substantiveness JOIN (substantiveness_join.go).
	noTarget, raised := NoTargetViolation("TestSomething", "pkg/target", ReferencedSymbolSet{}, false)
	if !raised {
		t.Fatal("NoTargetViolation must raise for a test that references nothing")
	}
	if noTarget.Severity != "error" {
		t.Errorf("NoTargetViolation returned severity %q, want \"error\". This value is fixed BY "+
			"DESIGN (ISSUE-106): the violation is SYNTHESIZED from a presence-only set-membership "+
			"test, so there is no contributing pack finding whose severity could be forwarded. It "+
			"is a ratified decision, not the same defect surviving — see NoTargetViolation's "+
			"docstring and TestNoTarget_SynthesizedSeverityIsFixedByDesign", noTarget.Severity)
	}

	// NOT ASSERTED HERE, DELIBERATELY: HollowFindingsToViolations. That sibling USED to
	// overwrite the pack's declared severity with a hardcoded "error" (ISSUE-106, hop 3 of
	// the ISSUE-104/ISSUE-105 chain) and this block used to lock that behavior. It now
	// FORWARDS the source finding's severity, so it is a pass-through rather than a
	// by-construction site, and asserting it here would be asserting the defect. The single
	// authority on the forwarding behavior is
	// TestQ1_HollowFindingsToViolations_ForwardsPackDeclaredSeverity
	// (pkg/gate/substantiveness_severity_test.go) — a pointer, not a duplicate assertion.

	// SITE 3 — StepTestVerificationScopedFunc's missing-mandated-test violations.
	root := t.TempDir()
	specDir := filepath.Join(root, "specs")
	codeDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatalf("creating spec dir: %v", err)
	}
	if err := os.MkdirAll(codeDir, 0o755); err != nil {
		t.Fatalf("creating code dir: %v", err)
	}
	writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestGate_MandatedButAbsent"},
	})
	verification := StepTestVerificationScopedFunc(
		specDir, codeDir,
		newGateScope(root, GateScopeModeDiff, []string{"specs/test.spec.md"}, nil),
		goTestClassifier(), goTestMatcher(t),
	)(context.Background())
	if len(verification.Violations) != 1 {
		t.Fatalf("expected exactly one missing-mandated-test violation, got %d (%#v)",
			len(verification.Violations), verification.Violations)
	}
	if !strings.Contains(verification.Violations[0].Message, "TestGate_MandatedButAbsent") {
		t.Fatalf("expected the violation to name the missing test, got %#v", verification.Violations[0])
	}
	if verification.Violations[0].Severity != "error" {
		t.Errorf("a missing mandated test returned severity %q, want \"error\"; this site's violations "+
			"are error-severity BY CONSTRUCTION, that invariant is breakable in one line, and the "+
			"missing-mandated-test signal depends on it", verification.Violations[0].Severity)
	}
}

// assertNonBlockingResult checks the exit-relevant semantics of a non-blocking step
// through the REAL summary computation: NewGateResult must keep Pass true, count the
// step in StepsWarned, and leave StepsFailed at zero. Asserting only the status string
// would miss a status that renders correctly but still flips the gate.
func assertNonBlockingResult(t *testing.T, result StepResult) {
	t.Helper()
	summary := NewGateResult([]StepResult{result})
	if !summary.Pass {
		t.Error("a warning-only step flipped GateResult.Pass; the gate would exit 1 on a finding " +
			"its own pack declared non-blocking")
	}
	if summary.StepsWarned != 1 {
		t.Errorf("expected the step to be counted in StepsWarned, got %d", summary.StepsWarned)
	}
	if summary.StepsFailed != 0 {
		t.Errorf("expected no failed steps, got %d", summary.StepsFailed)
	}
}
