package packval_test

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// engineErrorCheck is the dedicated Check identifier a broken engine run must carry.
// It is deliberately NOT one of the fixture-verdict checks: an engine that never
// produced a usable answer has no fixture verdict to report, and laundering it into
// one sends a pack author to inspect a fixture that was never the problem (CLM-005).
const engineErrorCheck = "engine-error"

// erroringEngineMock returns a mock whose engine run ERRORS for every target whose
// path contains the marker, and behaves at correct BUNDLE-005 REQ-011 polarity
// otherwise — so any failure is attributable to the error, not to fixture polarity.
func erroringEngineMock(marker string, underlying error) *packval.MockExecutor {
	mock := newFixtureMock(false, true)
	base := mock.EngineFn
	mock.EngineFn = func(packDir string, b engine.EngineBinding, targets []string) (packval.ExecutionResult, error) {
		for _, tgt := range targets {
			if strings.Contains(tgt, marker) {
				return packval.ExecutionResult{}, underlying
			}
		}
		return base(packDir, b, targets)
	}
	return mock
}

// TestPackVal_P3_EngineErrorIsDistinctFromFixtureVerdict (CLM-005): when the engine
// run for a POSITIVE fixture returns an error, phase 3 must report it under a
// dedicated engine-error Check carrying the underlying message — never fold it into
// "positive fixture ..." as `if err != nil || r.Passed` used to.
func TestPackVal_P3_EngineErrorIsDistinctFromFixtureVerdict(t *testing.T) {
	underlying := errors.New("engine \"semgrep --sarif --quiet\" never started: no such file")
	res := packval.RunFixtures(baseManifest(), makePackDir(t), erroringEngineMock("p.go", underlying))

	if res.Status != "fail" {
		t.Fatalf("a broken engine run must fail the phase; got %s", res.Status)
	}
	var found *packval.ValidationError
	for i, e := range res.Errors {
		if e.Check == engineErrorCheck {
			found = &res.Errors[i]
		}
		if e.Check == "semgrep-positive" {
			t.Errorf("a broken engine run must NOT masquerade as a positive fixture verdict; got %+v", e)
		}
	}
	if found == nil {
		t.Fatalf("expected an error under the dedicated %q check; got %+v", engineErrorCheck, res.Errors)
	}
	if !strings.Contains(found.Message, "never started") {
		t.Errorf("the engine-error message must carry the underlying engine error text; got %q", found.Message)
	}
	if found.Rule != "R1" || found.Claim != "C1" {
		t.Errorf("the engine error must name the rule and claim being processed; got Rule=%q Claim=%q", found.Rule, found.Claim)
	}
}

// TestPackVal_P3_EngineErrorIsNotAPassingNegative (CLM-005) is the VACUOUS-GREEN
// direction and the one that matters most: a NEGATIVE fixture whose engine run errored
// must not be absorbed into a pass, and the error text must survive rather than being
// discarded as the negative loop used to discard it.
func TestPackVal_P3_EngineErrorIsNotAPassingNegative(t *testing.T) {
	underlying := errors.New("engine \"semgrep --sarif --quiet\" produced no parseable SARIF: unexpected end of JSON input")
	res := packval.RunFixtures(baseManifest(), makePackDir(t), erroringEngineMock("n.go", underlying))

	if res.Status != "fail" {
		t.Fatalf("an engine error on a negative fixture must never read as a clean negative; got %s with errors %+v",
			res.Status, res.Errors)
	}
	var found *packval.ValidationError
	for i, e := range res.Errors {
		if e.Check == engineErrorCheck {
			found = &res.Errors[i]
		}
	}
	if found == nil {
		t.Fatalf("expected an error under the dedicated %q check; got %+v", engineErrorCheck, res.Errors)
	}
	if !strings.Contains(found.Message, "no parseable SARIF") {
		t.Errorf("the underlying engine error text must survive into the phase error; got %q", found.Message)
	}
	if found.Rule != "R1" || found.Claim != "C1" {
		t.Errorf("the engine error must name the rule and claim; got Rule=%q Claim=%q", found.Rule, found.Claim)
	}
}

// TestPackVal_P3_EngineSchemaErrorOnRealEngine (CLM-005) drives the REAL
// DefaultExecutor over testdata/engine-error-pack, whose rule file is structurally
// invalid to the engine.
//
// WHAT THE ENGINE ACTUALLY DOES, captured in TASK-001 rather than assumed (semgrep
// 1.156.0): it exits 7 and writes a COMPLETE, PARSEABLE SARIF document to stdout,
// carrying InvalidRuleSchemaError / SemgrepError notifications and "results": [].
// check.ParsePackFindings therefore succeeds and RunEngine returns Passed:false with
// a NIL error — indistinguishable from a clean, non-firing run.
//
// So the engine-error DISTINCTION CANNOT BE DRAWN FROM INSIDE phase 3 for this class:
// phase 3 is never handed an error to re-home. Closing that half means teaching the
// executor to notice a SARIF document whose invocation reported failure, which lives
// in pkg/packval/executor.go and belongs to ISSUE-140/ISSUE-141 — NOT to this lane
// (PLAN-ISSUE-092 F6). This test therefore asserts what IS reachable here: with the
// corrected polarity, the broken run's non-firing NEGATIVE fixture still turns the
// phase red, so a broken engine cannot produce a green pack test.
func TestPackVal_P3_EngineSchemaErrorOnRealEngine(t *testing.T) {
	if _, err := exec.LookPath("semgrep"); err != nil {
		t.Skipf("SKIPPING: the semgrep binary is not on PATH (%v); this test must actually run the engine "+
			"and is NOT being reported as a pass", err)
	}
	m, dir := testdataPack(t, "engine-error-pack")

	res := packval.RunFixtures(m, dir, &packval.DefaultExecutor{})

	if res.Status == "pass" {
		t.Fatalf("a pack whose rule file is structurally invalid to its engine must not reach phase3-fixtures: pass")
	}
	if len(res.Errors) == 0 {
		t.Fatal("expected at least one phase-3 error for a broken engine run")
	}
}
