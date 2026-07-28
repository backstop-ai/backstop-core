package main

import (
	"context"
	"testing"
	"time"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// stepNameOrder runs each assembled step and records its StepName, giving the
// real shipped pipeline order.
func stepNameOrder(t *testing.T, steps []gate.StepFunc) []string {
	t.Helper()
	names := make([]string, 0, len(steps))
	for _, s := range steps {
		names = append(names, s(context.Background()).StepName)
	}
	return names
}

// stepRuleInViolations reports whether a step's Violations carry a rule.
func stepRuleInViolations(steps []gate.StepResult, stepName, rule string) bool {
	for _, s := range steps {
		if s.StepName != stepName {
			continue
		}
		for _, v := range s.Violations {
			if v.Rule == rule {
				return true
			}
		}
	}
	return false
}

func indexOf(names []string, target string) int {
	for i, n := range names {
		if n == target {
			return i
		}
	}
	return -1
}

// TestGateCLI_StepOrder_WaiverBeforeBaseline proves the assembled shipped step
// list places StepWaiverResolution BEFORE StepBaselineComparison (CLM-072), so the
// accumulated set is waiver-subtracted before baseline captures NewViolations.
func TestGateCLI_StepOrder_WaiverBeforeBaseline(t *testing.T) {
	temp := t.TempDir()
	copyTree(t, waiverE2EFixtureRoot(t), temp)
	t.Setenv("WAIVER_E2E_SCENARIO", "waivable")
	scope, err := gate.ComputeGateScope(temp, gate.GateScopeModeAll, nil)
	if err != nil {
		t.Fatalf("ComputeGateScope: %v", err)
	}
	names := stepNameOrder(t, buildGateSteps(temp, scope))

	waiverIdx := indexOf(names, gate.StepWaiverResolution)
	baselineIdx := indexOf(names, gate.StepBaselineComparison)
	if waiverIdx < 0 || baselineIdx < 0 {
		t.Fatalf("both waiver and baseline steps must be present, got order %v", names)
	}
	if waiverIdx > baselineIdx {
		t.Fatalf("StepWaiverResolution (%d) must precede StepBaselineComparison (%d); order=%v", waiverIdx, baselineIdx, names)
	}
}

// TestGateCLI_Ratchet_ActiveWaiverSubtractedBeforeBaseline proves that against the
// REAL pipeline order, an active waiver subtracts its finding BEFORE
// baseline_comparison captures NewViolations, so the waived finding does NOT count
// against the ISSUE-050 ratchet (CLM-073). With an EMPTY baseline, an unsubtracted
// finding would be a NEW violation and fail baseline; the waiver running first
// makes baseline see zero. RED while waiver is ordered after baseline.
func TestGateCLI_Ratchet_ActiveWaiverSubtractedBeforeBaseline(t *testing.T) {
	temp := t.TempDir()
	copyTree(t, waiverE2EFixtureRoot(t), temp)
	t.Setenv("WAIVER_E2E_SCENARIO", "waivable")

	scope, err := gate.ComputeGateScope(temp, gate.GateScopeModeAll, nil)
	if err != nil {
		t.Fatalf("ComputeGateScope: %v", err)
	}
	policy, err := buildWaiverPolicy(temp)
	if err != nil {
		t.Fatalf("buildWaiverPolicy: %v", err)
	}
	steps := buildGateSteps(temp, scope)
	// An EMPTY baseline: any surviving finding is NEW and fails baseline.
	emptyBaseline := &gate.BaselineArtifact{SchemaVersion: "baseline/v1", Violations: []gate.Violation{}}
	g := gate.New(
		gate.WithSteps(steps),
		gate.WithScope(scope),
		gate.WithWaiver(buildWaiverLineReader(temp, scope), policy, time.Now()),
		gate.WithBaseline(emptyBaseline),
	)
	res, _ := g.Run(context.Background())

	const waivedRule = "backstop/waiver-e2e/waivable-defect"
	for _, s := range res.Steps {
		if s.StepName != gate.StepBaselineComparison {
			continue
		}
		for _, v := range s.NewViolations {
			if v.Rule == waivedRule {
				t.Fatalf("the active-waived finding must be subtracted BEFORE baseline captures NewViolations, but it counted as a NEW violation against the ratchet: %+v", v)
			}
		}
		// The finding must also be gone from the accumulated pack_engines set.
		if stepRuleInViolations(res.Steps, gate.StepPackEngines, waivedRule) {
			t.Fatal("the active-waived pack_engines finding must be subtracted from the accumulated set")
		}
		return
	}
	t.Fatal("baseline_comparison step not present in the assembled pipeline")
}
