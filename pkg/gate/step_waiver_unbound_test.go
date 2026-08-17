package gate

import (
	"context"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/waiver"
)

// step_waiver_unbound_test.go — ISSUE-097. The surfacing half: an unbound waiver token
// is ROT, not a broken promise, so it must be LOUD and NON-BLOCKING. Blocking on one
// would make a routine pack rename un-landable; passing silently is how five dead
// tokens survived three weeks in this repository.
//
// Fixtures are synthetic. The one test that reads the live tree lives in cmd/backstop.

// unboundTokenFor builds a harvested token keyed to a pack namespace no lock records.
func unboundTokenFor(ruleID, file string, line int) waiver.Waiver {
	return waiver.Waiver{
		RuleID: ruleID,
		Reason: waiver.ReasonFalsePositive,
		Expiry: waiverTestNow.AddDate(1, 0, 0),
		File:   file,
		Line:   line,
	}
}

// knownNamespaces returns the namespace set a populated backstop.lock would yield.
func knownNamespaces() []string { return []string{"backstop-ai/backstop-self"} }

// TestGate_WaiverUnbound_SurfacesAsNonBlockingWarning is CLM-008: the diagnostic
// reaches the waiver_resolution step as a Violation at severity exactly "warning", and
// the REAL blocksVerdict predicate returns false for it.
//
// The predicate is asserted directly rather than against a string literal, so a future
// change to the severity vocabulary breaks this test instead of silently escaping it.
func TestGate_WaiverUnbound_SurfacesAsNonBlockingWarning(t *testing.T) {
	stale := "backstop/self/backstop.packs.backstop.self.rules.no-structural-name-split-on-spine"
	g := &Gate{}
	WithUnboundWaiverScan(
		[]waiver.Waiver{unboundTokenFor(stale, "cmd/backstop/pack_gate.go", 1015)},
		knownNamespaces(),
	)(g)

	res := g.computeWaiverResult([]StepResult{packEnginesStep()}, memLineReader("app.go", nil), nil, waiverTestNow)

	if len(res.Violations) != 1 {
		t.Fatalf("an unbound token must surface as exactly one waiver_resolution violation, got %d (%#v)",
			len(res.Violations), res.Violations)
	}
	v := res.Violations[0]
	if v.Severity != "warning" {
		t.Errorf("an unbound-token violation must carry severity %q, got %q; rot is loud, not blocking",
			"warning", v.Severity)
	}
	if v.Rule != stale {
		t.Errorf("the violation must carry the token's FULL rule-id (the string the reader edits), got %q", v.Rule)
	}
	if v.File != "cmd/backstop/pack_gate.go" || v.Line != 1015 {
		t.Errorf("the violation must carry the token's own location, got %s:%d", v.File, v.Line)
	}
	if blocksVerdict(v) {
		t.Error("blocksVerdict returned true for the unbound-token violation; a routine pack rename " +
			"would become un-landable")
	}
}

// TestGate_WaiverUnbound_StepStatusIsWarningNotFail is CLM-012, the StepVerdict
// correction. computeWaiverResult derived its status from a RAW COUNT, which was only
// ACCIDENTALLY correct while every diagnostic it emitted was severity "error" — the
// first warning-severity kind turns that count into a hard gate failure.
func TestGate_WaiverUnbound_StepStatusIsWarningNotFail(t *testing.T) {
	stale := "backstop/self/backstop.packs.backstop.self.rules.no-baked-tool-exec"
	g := &Gate{}
	WithUnboundWaiverScan(
		[]waiver.Waiver{unboundTokenFor(stale, "tests/smoke/smoke_test.go", 33)},
		knownNamespaces(),
	)(g)

	res := g.computeWaiverResult([]StepResult{packEnginesStep()}, memLineReader("app.go", nil), nil, waiverTestNow)

	if res.Status == "fail" {
		t.Errorf("with ONLY unbound diagnostics the step must not report %q — that is the raw-count "+
			"behavior, and it turns a warning into a hard gate failure", "fail")
	}
	if res.Status == "pass" {
		t.Errorf("with unbound diagnostics present the step must not report %q either — a surfaced "+
			"warning that reports pass drops out of StepsWarned and the summary line, which is "+
			"invisibility by another name", "pass")
	}
	if res.Status != "warning" {
		t.Errorf("step Status = %q, want %q", res.Status, "warning")
	}

	// THE REGRESSION LEG: the existing error-severity path must not have been softened.
	// A MALFORMED token still fails the step.
	malformed := &Gate{}
	malformedRes := malformed.computeWaiverResult(
		[]StepResult{packEnginesStep(Violation{Rule: "pkg/rule-a", File: "app.go", Line: 5, Severity: "error"})},
		memLineReader("app.go", map[int]string{
			5: "risky() // @waiver:pkg/rule-a:not-a-reason-code:2999-01-01",
		}),
		nil, waiverTestNow,
	)
	if malformedRes.Status != "fail" {
		t.Errorf("a malformed token must still FAIL the step, got %q; the switch to a severity-derived "+
			"verdict must not have relaxed the error-severity kinds", malformedRes.Status)
	}
}

// TestGate_WaiverUnbound_DoesNotFlipGateVerdict is the claim that a pack rename stays
// landable (CLM-008): a full Gate.Run whose only finding is an unbound-waiver warning
// still passes, so the exit code stays 0.
func TestGate_WaiverUnbound_DoesNotFlipGateVerdict(t *testing.T) {
	stale := "backstop/self/backstop.packs.backstop.self.rules.no-baked-language-token"
	steps := []StepFunc{
		func(context.Context) StepResult {
			return StepResult{StepName: StepPackEngines, Status: "pass", Violations: []Violation{}}
		},
		StepWaiverResolutionScopedFunc(nil),
	}
	g := New(
		WithSteps(steps),
		WithWaiver(memLineReader("app.go", nil), nil, waiverTestNow),
		WithUnboundWaiverScan(
			[]waiver.Waiver{unboundTokenFor(stale, "tests/smoke/smoke_test.go", 53)},
			knownNamespaces(),
		),
	)
	res, exitCode := g.Run(context.Background())

	if !res.Pass {
		t.Errorf("an unbound-waiver warning flipped GateResult.Pass to false; rot must not block")
	}
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if res.StepsFailed != 0 {
		t.Errorf("StepsFailed = %d, want 0", res.StepsFailed)
	}

	// Non-blocking must not mean invisible: the warning is still REPORTED.
	var found bool
	for _, s := range res.Steps {
		if s.StepName != StepWaiverResolution {
			continue
		}
		for _, v := range s.Violations {
			if v.Rule == stale {
				found = true
			}
		}
	}
	if !found {
		t.Error("the unbound token must still be reported on the waiver_resolution step; a " +
			"non-blocking finding that is not reported is silent green")
	}
}

// TestGate_WaiverUnbound_ReasonNamesEachUnboundRuleID is CLM-008's report half: the
// waiverReason string gains a `; N unbound (...)` clause naming EVERY unbound rule-id,
// beside the existing active / expiring / unused clauses, which are unchanged.
//
// TWO distinct tokens are used deliberately: a single-token assertion cannot
// distinguish a real list from a hardcoded singular.
func TestGate_WaiverUnbound_ReasonNamesEachUnboundRuleID(t *testing.T) {
	first := "backstop/self/backstop.packs.backstop.self.rules.no-structural-name-split-on-spine"
	second := "backstop/self/backstop.packs.backstop.self.rules.no-baked-tool-exec"

	// An active waiver AND an unused/dangling one, so the existing clauses are exercised
	// in the same run the new clause is asserted on.
	accumulated := []StepResult{
		packEnginesStep(Violation{Rule: "pkg/rule-a", File: "app.go", Line: 5, Severity: "error"}),
	}
	read := memLineReader("app.go", map[int]string{
		5: "risky() // @waiver:pkg/rule-a:accepted-risk:2999-01-01",
		7: "// @waiver:pkg/ghost:deferred:2999-01-01",
		8: "other()",
	})

	g := &Gate{}
	WithUnboundWaiverScan([]waiver.Waiver{
		unboundTokenFor(first, "cmd/backstop/pack_gate.go", 1015),
		unboundTokenFor(second, "tests/smoke/smoke_test.go", 33),
	}, knownNamespaces())(g)

	res := g.computeWaiverResult(accumulated, read, nil, waiverTestNow)

	if !strings.Contains(res.Reason, "2 unbound") {
		t.Errorf("the reason must report the unbound COUNT, got %q", res.Reason)
	}
	for _, id := range []string{first, second} {
		if !strings.Contains(res.Reason, id) {
			t.Errorf("the reason must name unbound rule-id %q, got %q", id, res.Reason)
		}
	}

	// The existing clauses are unchanged in the same run.
	if !strings.Contains(res.Reason, "PASS · 1 waivers") {
		t.Errorf("the existing active clause must be unchanged, got %q", res.Reason)
	}
	if !strings.Contains(res.Reason, "pkg/rule-a") {
		t.Errorf("the existing active clause must still name its rule, got %q", res.Reason)
	}
}
