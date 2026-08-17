package packval_test

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// TestPackVal_P3_ValidatorDispatchDoesNotGateOnRetiredLayerField (CLM-007): the
// layer-3 validator branches used to be guarded on `rule.Layer == 3`. `Layer` was
// RETIRED from the gate-runtime model (SPEC-031 REQ-002 removed the field and its yaml
// key), and phase 5 already exempts engine-model rules from the layer 1..3 requirement
// precisely because they carry an engine instead — so `rule.Layer` is 0 for every pack
// written against the current model and RunValidator was never dispatched at all. That
// is the same dead-dispatch defect as the `File` guard, one field over.
//
// Stated plainly, because CLM-007's wording could imply otherwise: NO in-repo pack
// declares `validator:` today. This change removes a dead gate keyed on a retired
// field; it does not light up a real pack's sandbox rule.
func TestPackVal_P3_ValidatorDispatchDoesNotGateOnRetiredLayerField(t *testing.T) {
	m := baseManifest()
	// The shape a pack written against the CURRENT runtime model has: an engine, a
	// validator, and NO layer.
	m.Content.Ruleset.Rules[0].Layer = 0

	var calls [][]string
	mock := newFixtureMock(false, true)
	mock.ValidatorFn = func(_, _ string, fixturePaths []string) (packval.ExecutionResult, error) {
		calls = append(calls, append([]string(nil), fixturePaths...))
		for _, p := range fixturePaths {
			if p == "fixtures/n.go" {
				return packval.ExecutionResult{Passed: false}, nil
			}
		}
		return packval.ExecutionResult{Passed: true}, nil
	}

	res := packval.RunFixtures(m, makePackDir(t), mock)

	if len(calls) == 0 {
		t.Fatal("a rule declaring a validator and NO layer never dispatched RunValidator — " +
			"the dispatch is still gated on the retired Layer field")
	}
	if res.Status != "pass" {
		t.Fatalf("a correctly-behaving validator must leave the phase green; got %s: %+v", res.Status, res.Errors)
	}
}

// TestPackVal_P3_ValidatorDispatchStillSkipsRulesWithoutAValidator is the
// discriminating half: the new guard keys on the rule DECLARING a validator, so a rule
// that declares none must still dispatch nothing. Without this, an implementation that
// dispatched unconditionally would satisfy the test above.
func TestPackVal_P3_ValidatorDispatchStillSkipsRulesWithoutAValidator(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Layer = 0
	m.Content.Ruleset.Rules[0].Validator = ""

	calls := 0
	mock := newFixtureMock(false, true)
	mock.ValidatorFn = func(_, _ string, _ []string) (packval.ExecutionResult, error) {
		calls++
		return packval.ExecutionResult{Passed: true}, nil
	}

	packval.RunFixtures(m, makePackDir(t), mock)

	if calls != 0 {
		t.Fatalf("a rule declaring no validator must dispatch none; got %d calls", calls)
	}
}

// TestPackVal_P3_MultiFileValidatorDispatchDoesNotGateOnLayer covers the third
// validator branch: the multi-file batch call. `input_scope` is still a LIVE field and
// stays on the guard; only the retired `Layer` condition comes off.
func TestPackVal_P3_MultiFileValidatorDispatchDoesNotGateOnLayer(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Layer = 0
	m.Content.Ruleset.Rules[0].InputScope = "multi-file"

	var batched bool
	mock := newFixtureMock(false, true)
	mock.ValidatorFn = func(_, _ string, fixturePaths []string) (packval.ExecutionResult, error) {
		if len(fixturePaths) > 1 {
			batched = true
			return packval.ExecutionResult{Passed: true}, nil
		}
		if len(fixturePaths) == 1 && fixturePaths[0] == "fixtures/n.go" {
			return packval.ExecutionResult{Passed: false}, nil
		}
		return packval.ExecutionResult{Passed: true}, nil
	}

	res := packval.RunFixtures(m, makePackDir(t), mock)

	if !batched {
		t.Fatal("a multi-file rule declaring a validator and NO layer never received the batched call")
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s: %+v", res.Status, res.Errors)
	}
}
