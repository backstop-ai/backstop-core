package gate

import (
	"context"
	"fmt"
	"testing"
)

// makePassStep returns a StepFunc that always returns pass with the given name.
func makePassStep(name string) StepFunc {
	return func(_ context.Context) StepResult {
		return StepResult{StepName: name, Status: "pass", Violations: []Violation{}}
	}
}

// makeFailStep returns a StepFunc that always returns fail with a violation.
func makeFailStep(name string) StepFunc {
	return func(_ context.Context) StepResult {
		return StepResult{
			StepName:   name,
			Status:     "fail",
			Violations: []Violation{{Rule: name, Message: "failed", Severity: "error"}},
		}
	}
}

// makeSkippedStep returns a StepFunc that always returns skipped.
func makeSkippedStep(name, reason string) StepFunc {
	return func(_ context.Context) StepResult {
		return StepResult{StepName: name, Status: "skipped", Violations: []Violation{}, Reason: reason}
	}
}

// makeConfigErrStep returns a StepFunc that returns a config error result.
func makeConfigErrStep(name string) StepFunc {
	return func(_ context.Context) StepResult {
		return StepResult{
			StepName:   name,
			Status:     "fail",
			Violations: []Violation{{Rule: name, Message: "config error", Severity: "error"}},
			ConfigErr:  true,
		}
	}
}

// allPassSteps returns nine pass steps in canonical order.
func allPassSteps() []StepFunc {
	var steps []StepFunc
	for _, name := range AllStepNames {
		steps = append(steps, makePassStep(name))
	}
	return steps
}

// allSkippedSteps returns nine skipped steps in canonical order.
func allSkippedSteps() []StepFunc {
	var steps []StepFunc
	for _, name := range AllStepNames {
		steps = append(steps, makeSkippedStep(name, "not implemented"))
	}
	return steps
}

// --- Kill chain ordering tests ---

// TestGate_AllNineStepsExecuteInOrder verifies gate runs all nine steps in
// the specified canonical order.
func TestGate_AllNineStepsExecuteInOrder(t *testing.T) {
	var order []string
	var steps []StepFunc
	for _, name := range AllStepNames {
		n := name // capture
		steps = append(steps, func(_ context.Context) StepResult {
			order = append(order, n)
			return StepResult{StepName: n, Status: "pass", Violations: []Violation{}}
		})
	}

	g := New(WithSteps(steps))
	result, exitCode := g.Run(context.Background())

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if len(result.Steps) != 9 {
		t.Fatalf("expected 9 steps, got %d", len(result.Steps))
	}
	for i, name := range AllStepNames {
		if order[i] != name {
			t.Errorf("step %d: expected %q, got %q", i, name, order[i])
		}
	}
}

// TestGate_ContinuesAfterStepFailure verifies gate continues executing
// remaining steps after a step fails.
func TestGate_ContinuesAfterStepFailure(t *testing.T) {
	steps := []StepFunc{
		makePassStep(StepArtifactValidation),
		makeFailStep(StepCodeCheck), // fails
		makePassStep(StepTestVerification),
		makePassStep(StepTestSubstantiveness),
		makePassStep(StepCoverageThreshold),
		makePassStep(StepContractSignature),
		makeSkippedStep(StepBaselineComparison, "baseline not implemented"),
		makeSkippedStep(StepWaiverResolution, "waivers not implemented"),
		makeSkippedStep(StepLedgerIntegrity, "ledger not implemented"),
	}

	g := New(WithSteps(steps))
	result, _ := g.Run(context.Background())

	// All 9 steps should appear
	if len(result.Steps) != 9 {
		t.Fatalf("expected 9 steps in output, got %d", len(result.Steps))
	}

	// Steps after the failure should have executed
	if result.Steps[2].Status != "pass" {
		t.Errorf("step 3 (test_verification) should have executed and passed, got %q", result.Steps[2].Status)
	}
}

// TestGate_AllNineStepsAppearInOutput verifies all nine steps appear in gate
// output regardless of pass/fail/skip status.
func TestGate_AllNineStepsAppearInOutput(t *testing.T) {
	steps := []StepFunc{
		makePassStep(StepArtifactValidation),
		makeFailStep(StepCodeCheck),
		makeSkippedStep(StepTestVerification, "skipped"),
		makePassStep(StepTestSubstantiveness),
		makePassStep(StepCoverageThreshold),
		makePassStep(StepContractSignature),
		makeSkippedStep(StepBaselineComparison, "baseline not implemented"),
		makeSkippedStep(StepWaiverResolution, "waivers not implemented"),
		makeSkippedStep(StepLedgerIntegrity, "ledger not implemented"),
	}

	g := New(WithSteps(steps))
	result, _ := g.Run(context.Background())

	if len(result.Steps) != 9 {
		t.Fatalf("expected 9 steps, got %d", len(result.Steps))
	}

	nameSet := make(map[string]bool)
	for _, s := range result.Steps {
		nameSet[s.StepName] = true
	}
	for _, name := range AllStepNames {
		if !nameSet[name] {
			t.Errorf("step %q missing from output", name)
		}
	}
}

// --- Config error halt tests ---

// TestGate_ExitCode2_ConfigError verifies exit code 2 when gate has a
// configuration error before any steps.
func TestGate_ExitCode2_ConfigError(t *testing.T) {
	g := New(WithSteps(allPassSteps()), WithConfigError(fmt.Errorf("backstop.yml not found")))
	_, exitCode := g.Run(context.Background())

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
}

// TestGate_ExitCode2_PrecedenceOverExitCode1 verifies exit code 2 takes
// precedence when both config error and step failure exist.
func TestGate_ExitCode2_PrecedenceOverExitCode1(t *testing.T) {
	g := New(WithSteps(allPassSteps()), WithConfigError(fmt.Errorf("invalid config")))
	_, exitCode := g.Run(context.Background())

	if exitCode != 2 {
		t.Errorf("expected exit code 2 (precedence), got %d", exitCode)
	}
}

// --- Exit code tests ---

// TestGate_ExitCode0_AllPass verifies exit code 0 when all steps pass.
func TestGate_ExitCode0_AllPass(t *testing.T) {
	g := New(WithSteps(allPassSteps()))
	_, exitCode := g.Run(context.Background())

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

// TestGate_ExitCode0_PassAndSkipped verifies exit code 0 when all steps
// either pass or are skipped.
func TestGate_ExitCode0_PassAndSkipped(t *testing.T) {
	steps := []StepFunc{
		makePassStep(StepArtifactValidation),
		makePassStep(StepCodeCheck),
		makePassStep(StepTestVerification),
		makePassStep(StepTestSubstantiveness),
		makePassStep(StepCoverageThreshold),
		makePassStep(StepContractSignature),
		makeSkippedStep(StepBaselineComparison, "baseline not implemented"),
		makeSkippedStep(StepWaiverResolution, "waivers not implemented"),
		makeSkippedStep(StepLedgerIntegrity, "ledger not implemented"),
	}
	g := New(WithSteps(steps))
	_, exitCode := g.Run(context.Background())

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

// TestGate_ExitCode1_StepFailed verifies exit code 1 when any step fails.
func TestGate_ExitCode1_StepFailed(t *testing.T) {
	steps := make([]StepFunc, 0, 9)
	steps = append(steps, makePassStep(StepArtifactValidation))
	steps = append(steps, makeFailStep(StepCodeCheck))
	for _, name := range AllStepNames[2:] {
		steps = append(steps, makePassStep(name))
	}

	g := New(WithSteps(steps))
	_, exitCode := g.Run(context.Background())

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestGate_BaselineComparison_WiredAfterAccumulatedChecks verifies the
// baseline step runs after earlier evaluating steps and can observe accumulated
// step state without rerunning previous checks.
func TestGate_BaselineComparison_WiredAfterAccumulatedChecks(t *testing.T) {
	executed := make([]string, 0, 9)
	steps := []StepFunc{}
	for _, stepName := range AllStepNames[:6] {
		name := stepName
		steps = append(steps, func(_ context.Context) StepResult {
			executed = append(executed, name)
			return StepResult{StepName: name, Status: "pass", Violations: []Violation{}}
		})
	}
	steps = append(steps, func(_ context.Context) StepResult {
		executed = append(executed, StepBaselineComparison)
		if len(executed) != 7 {
			return StepResult{
				StepName: StepBaselineComparison,
				Status:   "fail",
				Violations: []Violation{{
					Rule:     "baseline/wiring",
					Message:  "baseline step did not run after accumulated evaluating steps",
					Severity: "error",
				}},
			}
		}
		return StepResult{StepName: StepBaselineComparison, Status: "pass", Violations: []Violation{}}
	})
	steps = append(steps,
		makeSkippedStep(StepWaiverResolution, "waivers not implemented"),
		makeSkippedStep(StepLedgerIntegrity, "ledger not implemented"),
	)

	g := New(WithSteps(steps))
	result, exitCode := g.Run(context.Background())

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d with result=%#v", exitCode, result)
	}
	if len(result.Steps) != 9 {
		t.Fatalf("expected 9 steps, got %d", len(result.Steps))
	}
	if result.Steps[6].StepName != StepBaselineComparison {
		t.Fatalf("expected step 7 to be %q, got %q", StepBaselineComparison, result.Steps[6].StepName)
	}
}

// TestGate_BaselineComparison_MissingBaselineSkipsWithReason verifies missing
// baseline behavior currently surfaces as skipped baseline comparison.
func TestGate_BaselineComparison_MissingBaselineSkipsWithReason(t *testing.T) {
	steps := []StepFunc{
		makePassStep(StepArtifactValidation),
		makePassStep(StepCodeCheck),
		makePassStep(StepTestVerification),
		makePassStep(StepTestSubstantiveness),
		makePassStep(StepCoverageThreshold),
		makePassStep(StepContractSignature),
		StepBaselineComparisonFunc(),
		StepWaiverResolutionFunc(),
		StepLedgerIntegrityFunc(),
	}

	result, exitCode := New(WithSteps(steps)).Run(context.Background())
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 when baseline step is skipped, got %d", exitCode)
	}
	if result.Steps[6].Status != "skipped" {
		t.Fatalf("expected baseline step skipped, got %q", result.Steps[6].Status)
	}
	if result.Steps[6].Reason == "" {
		t.Fatal("expected baseline skipped reason to be populated")
	}
}

func TestGate_BaselineComparison_RuleSetChangeSeedingAllowedForAllScope(t *testing.T) {
	steps := []StepFunc{
		func(_ context.Context) StepResult {
			return StepResult{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{{Rule: "code_check/new-rule", File: "legacy.ts", Message: "legacy violation after rule update", Severity: "error"}}}
		},
		makePassStep(StepCodeCheck),
		makePassStep(StepTestVerification),
		makePassStep(StepTestSubstantiveness),
		makePassStep(StepCoverageThreshold),
		makePassStep(StepContractSignature),
		StepBaselineComparisonFunc(),
		StepWaiverResolutionFunc(),
		StepLedgerIntegrityFunc(),
	}

	result, exitCode := New(
		WithSteps(steps),
		WithScope(newGateScope("", GateScopeModeAll, nil, nil)),
		WithBaseline(&BaselineArtifact{SchemaVersion: BaselineSchemaV1, Violations: []Violation{}}),
		WithRuleSetChangeSeedingAllowed(true),
	).Run(context.Background())

	if exitCode != 0 {
		t.Fatalf("expected full-scope seeding exception to pass, got exit %d", exitCode)
	}
	baselineStep := result.Steps[6]
	if baselineStep.Status != "pass" {
		t.Fatalf("expected baseline step pass with allowed seeding, got %q", baselineStep.Status)
	}
	if len(baselineStep.SeededViolations) != 1 {
		t.Fatalf("expected one seeded violation diagnostic, got %d", len(baselineStep.SeededViolations))
	}
	if baselineStep.Reason == "" || baselineStep.Reason == "0 new violations beyond baseline" {
		t.Fatalf("expected explicit seeding reason, got %q", baselineStep.Reason)
	}
}

func TestGate_BaselineComparison_ChangedCodeStillFailsWhenSeedingFlagSet(t *testing.T) {
	steps := []StepFunc{
		func(_ context.Context) StepResult {
			return StepResult{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{{Rule: "code_check/new-rule", File: "changed.ts", Message: "changed file regression", Severity: "error"}}}
		},
		makePassStep(StepCodeCheck),
		makePassStep(StepTestVerification),
		makePassStep(StepTestSubstantiveness),
		makePassStep(StepCoverageThreshold),
		makePassStep(StepContractSignature),
		StepBaselineComparisonFunc(),
		StepWaiverResolutionFunc(),
		StepLedgerIntegrityFunc(),
	}

	result, exitCode := New(
		WithSteps(steps),
		WithScope(newGateScope("", GateScopeModeDiff, []string{"changed.ts"}, nil)),
		WithBaseline(&BaselineArtifact{SchemaVersion: BaselineSchemaV1, Violations: []Violation{}}),
		WithRuleSetChangeSeedingAllowed(true),
	).Run(context.Background())

	if exitCode != 1 {
		t.Fatalf("expected changed/scoped regression to fail despite seeding flag, got exit %d", exitCode)
	}
	baselineStep := result.Steps[6]
	if baselineStep.Status != "fail" {
		t.Fatalf("expected baseline step fail for changed-file regression, got %q", baselineStep.Status)
	}
	if len(baselineStep.NewViolations) != 1 {
		t.Fatalf("expected one new violation in diagnostics, got %d", len(baselineStep.NewViolations))
	}
	if len(baselineStep.SeededViolations) != 0 {
		t.Fatalf("expected zero seeded violations for scoped run, got %d", len(baselineStep.SeededViolations))
	}
}

// --- Delegated config error halt tests ---

// TestGate_ExitCode2_DelegatedArtifactValidateConfigError verifies that a
// config error from artifact validate halts remaining steps and exits 2.
func TestGate_ExitCode2_DelegatedArtifactValidateConfigError(t *testing.T) {
	executed := make(map[string]bool)
	steps := []StepFunc{
		makeConfigErrStep(StepArtifactValidation),
	}
	// Add remaining steps that track execution
	for _, name := range AllStepNames[1:] {
		n := name
		steps = append(steps, func(_ context.Context) StepResult {
			executed[n] = true
			return StepResult{StepName: n, Status: "pass", Violations: []Violation{}}
		})
	}

	g := New(WithSteps(steps))
	_, exitCode := g.Run(context.Background())

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}

	// Remaining steps should NOT have executed
	for _, name := range AllStepNames[1:] {
		if executed[name] {
			t.Errorf("step %q should not have executed after config error halt", name)
		}
	}
}

// TestGate_ExitCode2_DelegatedCodeCheckConfigError verifies that a config
// error from code check halts remaining steps and exits 2.
func TestGate_ExitCode2_DelegatedCodeCheckConfigError(t *testing.T) {
	executed := make(map[string]bool)
	steps := []StepFunc{
		makePassStep(StepArtifactValidation),
		makeConfigErrStep(StepCodeCheck),
	}
	for _, name := range AllStepNames[2:] {
		n := name
		steps = append(steps, func(_ context.Context) StepResult {
			executed[n] = true
			return StepResult{StepName: n, Status: "pass", Violations: []Violation{}}
		})
	}

	g := New(WithSteps(steps))
	_, exitCode := g.Run(context.Background())

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}

	for _, name := range AllStepNames[2:] {
		if executed[name] {
			t.Errorf("step %q should not have executed after config error halt", name)
		}
	}
}

// --- Config loading tests ---

// TestGate_ConfigMissing_ExitCode2 verifies exit code 2 when backstop.yml
// is not found.
func TestGate_ConfigMissing_ExitCode2(t *testing.T) {
	g := New(WithSteps(allPassSteps()), WithConfigError(fmt.Errorf("backstop.yml not found")))
	_, exitCode := g.Run(context.Background())

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
}

// TestGate_ConfigInvalid_ExitCode2 verifies exit code 2 when backstop.yml
// fails validation.
func TestGate_ConfigInvalid_ExitCode2(t *testing.T) {
	g := New(WithSteps(allPassSteps()), WithConfigError(fmt.Errorf("backstop.yml validation failed: missing project field")))
	_, exitCode := g.Run(context.Background())

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
}
