package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/bmanson/backstop-core/pkg/gate"
)

// spyDelegate records whether the underlying analyzer step was reached.
type spyDelegate struct {
	reached bool
}

func (s *spyDelegate) step(_ context.Context) gate.StepResult {
	s.reached = true
	return gate.StepResult{StepName: "analyzer", Status: "pass", Violations: []gate.Violation{}}
}

// TestWiring_ClassifierInterceptsClass123_AndFallsThroughWhenWorking (CLM-028):
// the classifier wrapper INTERCEPTS for an assigned class — for class 1/2/3 the
// wrapping step returns PolarityStepResult and the underlying analyzer is NOT
// reached (spy not invoked); for declared-and-working (and undeclared-but-
// present) the wrapper FALLS THROUGH unchanged and the analyzer IS reached (spy
// invoked). An UNWIRED classifier (analyzer always reached) MUST fail this test.
func TestWiring_ClassifierInterceptsClass123_AndFallsThroughWhenWorking(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		dim         gate.TraceabilityDimension
		wantReached bool
		wantStatus  string
	}{
		{
			// Class 2: typescript undeclared coverage -> capability-absent, intercept.
			name:        "class2_capability_absent_intercepts",
			cfg:         &config.Config{Project: "p", Language: "typescript"},
			dim:         gate.DimensionCoverage,
			wantReached: false,
			wantStatus:  "warning",
		},
		{
			// Class 3: go declared coverage but capability forced missing -> intercept.
			// We force class 3 by declaring coverage on a stack whose baked analyzer
			// derivation marks it absent (typescript declared coverage = declared +
			// absent = class 3).
			name: "class3_declared_intent_unmet_intercepts",
			cfg: &config.Config{Project: "p", Language: "typescript", Enforcement: config.Enforcement{
				Toolchain: map[string]config.ToolchainPass{
					"test": {Command: "c", Format: "f", GateType: string(gate.DimensionCoverage)},
				},
			}},
			dim:         gate.DimensionCoverage,
			wantReached: false,
			wantStatus:  "fail",
		},
		{
			// Class 1: unknown gate_type -> broken-declared, intercept.
			name: "class1_unknown_key_intercepts",
			cfg: &config.Config{Project: "p", Language: "go", Enforcement: config.Enforcement{
				Toolchain: map[string]config.ToolchainPass{
					"lint": {Command: "c", Format: "f", GateType: "bogus"},
				},
			}},
			dim:         gate.DimensionCoverage,
			wantReached: false,
			wantStatus:  "fail",
		},
		{
			// None/proceed: go undeclared coverage, baked analyzer present -> fall through.
			name:        "undeclared_present_falls_through",
			cfg:         &config.Config{Project: "p", Language: "go"},
			dim:         gate.DimensionCoverage,
			wantReached: true,
			wantStatus:  "pass",
		},
		{
			// None/proceed: go declared-and-working substantiveness -> fall through.
			// SPEC-037 re-key: substantiveness capability is now INSTALLED-pack-keyed,
			// so "working" requires the substantiveness pack installed (packs map), not
			// the deleted baked analyzer. With the pack installed AND declared, the
			// dimension is declared+present+working -> none/proceed -> fall through.
			name: "declared_working_falls_through",
			cfg: &config.Config{Project: "p", Language: "go",
				Packs: config.Packs{"backstop/substantiveness": "local"},
				Enforcement: config.Enforcement{
					Toolchain: map[string]config.ToolchainPass{
						"test": {Command: "go test ./...", Format: "go-test", GateType: string(gate.DimensionSubstantiveness)},
					},
				}},
			dim:         gate.DimensionSubstantiveness,
			wantReached: true,
			wantStatus:  "pass",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spy := &spyDelegate{}
			wrapped := wrapTraceabilityStep(tc.cfg, tc.dim, "step", spy.step)
			res := wrapped(context.Background())
			if spy.reached != tc.wantReached {
				t.Errorf("analyzer reached = %v, want %v (intercept/fall-through wiring)", spy.reached, tc.wantReached)
			}
			if res.Status != tc.wantStatus {
				t.Errorf("status = %q, want %q", res.Status, tc.wantStatus)
			}
		})
	}
}

// TestNoAnalyzerChange_Substantiveness_VerdictPreserved (CLM-025): the
// classification layer leaves the substantiveness analyzer's verdict unchanged
// when the dimension is declared-and-working — the wrapper falls through to the
// unchanged analyzer, which returns its own verdict.
func TestNoAnalyzerChange_Substantiveness_VerdictPreserved(t *testing.T) {
	// SPEC-037 re-key: substantiveness is INSTALLED-pack-keyed, so "working" requires
	// the pack installed (packs map) — declared+present+working falls through to the
	// delegate, preserving its verdict.
	cfg := &config.Config{Project: "p", Language: "go",
		Packs: config.Packs{"backstop/substantiveness": "local"},
		Enforcement: config.Enforcement{
			Toolchain: map[string]config.ToolchainPass{
				"test": {Command: "go test ./...", Format: "go-test", GateType: string(gate.DimensionSubstantiveness)},
			},
		}}
	// Analyzer that returns a FAIL verdict (a hollow test). The wrapper must
	// preserve it verbatim when falling through.
	delegate := func(_ context.Context) gate.StepResult {
		return gate.StepResult{StepName: gate.StepTestSubstantiveness, Status: "fail", Violations: []gate.Violation{{Rule: "hollow", Message: "hollow test"}}}
	}
	wrapped := wrapTraceabilityStep(cfg, gate.DimensionSubstantiveness, gate.StepTestSubstantiveness, delegate)
	res := wrapped(context.Background())
	if res.Status != "fail" || len(res.Violations) == 0 {
		t.Errorf("declared-and-working must fall through and preserve the analyzer's fail verdict, got %#v", res)
	}
}

// TestNoAnalyzerChange_Contracts_VerdictPreserved (CLM-026). UPDATED FOR SPEC-038
// (align-predating-artifacts): the contracts dimension is now INSTALLED-pack-keyed
// (the go/parser analyzer is deleted), so "working" requires the contracts pack
// installed (packs map) — declared+present+working falls through to the delegate,
// preserving its verdict, exactly as the substantiveness arm does post-Seed-3.
func TestNoAnalyzerChange_Contracts_VerdictPreserved(t *testing.T) {
	cfg := &config.Config{Project: "p", Language: "go",
		Packs: config.Packs{"backstop/contracts": "local"},
		Enforcement: config.Enforcement{
			Toolchain: map[string]config.ToolchainPass{
				"test": {Command: "go test ./...", Format: "go-test", GateType: string(gate.DimensionContracts)},
			},
		}}
	delegate := func(_ context.Context) gate.StepResult {
		return gate.StepResult{StepName: gate.StepContractSignature, Status: "pass", Violations: []gate.Violation{}}
	}
	wrapped := wrapTraceabilityStep(cfg, gate.DimensionContracts, gate.StepContractSignature, delegate)
	res := wrapped(context.Background())
	if res.Status != "pass" {
		t.Errorf("declared-and-working contracts must fall through to the analyzer's pass verdict, got %#v", res)
	}
}

// TestNoAnalyzerChange_Coverage_VerdictPreserved (CLM-027).
func TestNoAnalyzerChange_Coverage_VerdictPreserved(t *testing.T) {
	cfg := &config.Config{Project: "p", Language: "go", Enforcement: config.Enforcement{
		Toolchain: map[string]config.ToolchainPass{
			"test": {Command: "go test ./...", Format: "go-test", GateType: string(gate.DimensionCoverage)},
		},
	}}
	delegate := func(_ context.Context) gate.StepResult {
		return gate.StepResult{StepName: gate.StepCoverageThreshold, Status: "fail", Violations: []gate.Violation{{Rule: "coverage", Message: "below threshold"}}}
	}
	wrapped := wrapTraceabilityStep(cfg, gate.DimensionCoverage, gate.StepCoverageThreshold, delegate)
	res := wrapped(context.Background())
	if res.Status != "fail" {
		t.Errorf("declared-and-working coverage must fall through to the analyzer's below-threshold fail verdict, got %#v", res)
	}
}

// TestBuildGateSteps_WiresTraceabilityWrappers asserts the wiring is actually
// installed in buildGateSteps (not just that the wrapper exists): on a non-Go
// project with no declared traceability dimensions, the three traceability
// steps come back as non-failing warning advisories (class 2), proving the
// classifier runs IN FRONT OF the analyzers in the real step list.
func TestBuildGateSteps_WiresTraceabilityWrappers(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(
		"project: rt\nlanguage: typescript\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".backstop"), 0o755); err != nil {
		t.Fatal(err)
	}

	steps := buildGateSteps(projectRoot)
	traceabilitySteps := map[string]bool{
		gate.StepTestSubstantiveness: false,
		gate.StepCoverageThreshold:   false,
		gate.StepContractSignature:   false,
	}
	for _, step := range steps {
		res := step(context.Background())
		if _, ok := traceabilitySteps[res.StepName]; ok {
			traceabilitySteps[res.StepName] = true
			if res.Status != "warning" {
				t.Errorf("step %s on a non-Go undeclared project: status = %q, want warning (class 2)", res.StepName, res.Status)
			}
			if res.ConfigErr {
				t.Errorf("step %s class-2 advisory must not set ConfigErr", res.StepName)
			}
		}
	}
	for name, seen := range traceabilitySteps {
		if !seen {
			t.Errorf("traceability step %s not present in the built gate step list", name)
		}
	}
}
