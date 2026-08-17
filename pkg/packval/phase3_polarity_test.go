package packval_test

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// This file states BUNDLE-005 REQ-011's fixture-polarity contract in ONE greppable
// place, so a reader does not have to reconstruct it from the mock configurations of a
// dozen scattered tests:
//
//	"Phase 3 fixture execution ... requiring 100% pass rate: every positive fixture
//	 must NOT trigger the rule, every negative fixture MUST trigger the rule."

// TestPackVal_P3_PositiveFixtureThatFiresIsAFailure (CLM-004): a positive fixture is
// the CLEAN example a rule must leave alone. A finding on it is a FALSE POSITIVE, and
// a false positive is a phase-3 failure.
func TestPackVal_P3_PositiveFixtureThatFiresIsAFailure(t *testing.T) {
	res := packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(true, true))

	if res.Status != "fail" {
		t.Fatalf("a positive fixture that FIRES is a false positive and must fail phase 3; got %q", res.Status)
	}
	if !hasCheck(res.Errors, "semgrep-positive") {
		t.Fatalf("expected the failure on the positive path; got %+v", res.Errors)
	}
}

// TestPackVal_P3_NegativeFixtureThatDoesNotFireIsAFailure (CLM-004): a negative fixture
// is the VIOLATING example a rule exists to catch. A negative that produces no finding
// means the claim is untested, which is the vacuous green ISSUE-092 is about — so it
// is a phase-3 failure, and it carries the engine-limitation fix hint.
func TestPackVal_P3_NegativeFixtureThatDoesNotFireIsAFailure(t *testing.T) {
	res := packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(false, false))

	if res.Status != "fail" {
		t.Fatalf("a negative fixture that does NOT fire must fail phase 3; got %q", res.Status)
	}
	hinted := false
	for _, e := range res.Errors {
		if e.Check == "semgrep-negative" && strings.Contains(e.FixHint, "engine limitation") {
			hinted = true
		}
	}
	if !hinted {
		t.Fatalf("the engine-limitation fix hint belongs on the non-firing negative branch; got %+v", res.Errors)
	}
}

// TestPackVal_P3_ValidatorPolarityUnchangedByFindingsFix (CLM-004) exists to stop a
// well-meaning sweep from "fixing" the whole file on the theory that all of it was
// backwards. The two seams read ExecutionResult.Passed OPPOSITELY:
//
//   - RunEngine.Passed means "the engine FIRED" (produced findings).
//   - RunValidator.Passed means "the validator EXITED ZERO".
//
// So on the validator path a positive fixture SHOULD pass and a negative SHOULD fail —
// the inverse of the findings path — and those branches were always correct. One
// conditional shape applied to two seams whose Passed means opposite things is what
// produced ISSUE-092 in the first place.
func TestPackVal_P3_ValidatorPolarityUnchangedByFindingsFix(t *testing.T) {
	// Findings fixtures held at correct polarity throughout, so any failure below is
	// attributable to the validator seam alone.
	clean := func() *packval.MockExecutor { return newFixtureMock(false, true) }

	t.Run("validator exiting zero on positives and non-zero on negatives passes", func(t *testing.T) {
		mock := clean()
		mock.ValidatorFn = func(_, _ string, fixtures []string) (packval.ExecutionResult, error) {
			for _, f := range fixtures {
				if strings.Contains(f, "n.") {
					return packval.ExecutionResult{Passed: false}, nil
				}
			}
			return packval.ExecutionResult{Passed: true}, nil
		}
		res := packval.RunFixtures(baseManifest(), makePackDir(t), mock)
		if res.Status != "pass" {
			t.Fatalf("the validator branches were flipped along with the findings branches: "+
				"exit-zero-on-positive / non-zero-on-negative is the CORRECT validator polarity; got %s: %+v",
				res.Status, res.Errors)
		}
	})

	t.Run("a validator that exits zero on a negative still fails", func(t *testing.T) {
		mock := clean()
		mock.ValidatorFn = func(_, _ string, _ []string) (packval.ExecutionResult, error) {
			return packval.ExecutionResult{Passed: true}, nil
		}
		res := packval.RunFixtures(baseManifest(), makePackDir(t), mock)
		if !hasCheck(res.Errors, "validator-negative") {
			t.Fatalf("a validator that accepts the violating fixture must still be a validator-negative failure; got %+v", res.Errors)
		}
	})
}

// TestPackVal_P3_EngineFiringIsReadFromTheSarifContract documents the mechanism the
// polarity contract rests on, so the next reader does not have to infer it: the
// findings seam decides "fired" from the engine's SARIF output, not from its exit code.
func TestPackVal_P3_EngineFiringIsReadFromTheSarifContract(t *testing.T) {
	var seen []bool
	mock := newFixtureMock(false, true)
	mock.EngineFn = func(_ string, _ engine.EngineBinding, targets []string) (packval.ExecutionResult, error) {
		fired := targetsAreNegative(targets)
		seen = append(seen, fired)
		// A findings engine legitimately exits NON-ZERO when it reports findings, so a
		// non-zero exit code alongside Passed=true must not itself be an error.
		if fired {
			return packval.ExecutionResult{Passed: true, ExitCode: 1}, nil
		}
		return packval.ExecutionResult{Passed: false, ExitCode: 0}, nil
	}

	res := packval.RunFixtures(baseManifest(), makePackDir(t), mock)

	if len(seen) == 0 {
		t.Fatal("no engine dispatch observed")
	}
	if res.Status != "pass" {
		t.Fatalf("a firing negative with a non-zero exit code is the healthy case; got %s: %+v", res.Status, res.Errors)
	}
}
