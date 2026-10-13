package main

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// ISSUE-209: a pack_engines config-error step can now carry PRESERVED FINDINGS next
// to its engine-error violations. The baseline-generate refusal diagnostic must name
// what actually broke — the engine failures — not concatenate every finding message
// into the refusal line. Steps whose config-error violations use other rule names
// keep today's rendering exactly.

// diagTestEngineError is the D-1 engine-error violation shape.
func diagTestEngineError(msg string) gate.Violation {
	return gate.Violation{Rule: gate.StepPackEngines, Severity: "error", File: "", ProjectWide: true, Message: msg}
}

// TestConfigErrorDiagnostic_RendersOnlyStepRuleViolationsWhenPresent (CLM-008).
func TestConfigErrorDiagnostic_RendersOnlyStepRuleViolationsWhenPresent(t *testing.T) {
	e1 := `dispatching findings engine "eng-1" for pack org/pack: ENGINE-ONE-BROKE`
	e2 := `pack org/other: unknown engine "eng-2": no EngineBinding registered`
	violations := []gate.Violation{
		{Rule: "org/pack/no-foo", File: "a.go", Severity: "error", SourcePack: "org/pack", Message: "FINDING-ONE"},
		diagTestEngineError(e1),
		{Rule: "org/pack/no-foo", File: "b.go", Severity: "error", SourcePack: "org/pack", Message: "FINDING-TWO"},
		diagTestEngineError(e2),
	}

	cases := []struct {
		name   string
		reason string
		want   string
	}{
		{name: "no reason", reason: "", want: "pack_engines: " + e1 + " " + e2},
		{name: "reason leads", reason: "engine dispatch failed", want: "pack_engines: engine dispatch failed " + e1 + " " + e2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			steps := []gate.StepResult{{StepName: gate.StepPackEngines, Status: "fail", ConfigErr: true, Reason: tc.reason, Violations: violations}}
			got := configErrorDiagnostic(steps)
			if !strings.HasPrefix(got, "pack_engines: ") {
				t.Errorf("the diagnostic must lead with the step name, got: %s", got)
			}
			if tc.reason != "" && !strings.HasPrefix(got, "pack_engines: "+tc.reason) {
				t.Errorf("a non-empty Reason must still lead, got: %s", got)
			}
			i1, i2 := strings.Index(got, e1), strings.Index(got, e2)
			if i1 < 0 || i2 < 0 || i1 >= i2 {
				t.Errorf("both engine-error messages must appear in dispatch order, got: %s", got)
			}
			for _, finding := range []string{"FINDING-ONE", "FINDING-TWO"} {
				if strings.Contains(got, finding) {
					t.Errorf("a preserved finding (%s) is not the config error and must not be rendered, got: %s", finding, got)
				}
			}
			if got != tc.want {
				t.Errorf("diagnostic mismatch:\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

// TestConfigErrorDiagnostic_RendersAllMessagesWithoutStepRuleViolation (CLM-008;
// control): a traceability-polarity-shaped ConfigErr step, whose violations use a
// "<dim>_broken_declared" rule rather than the step name, renders every message
// exactly as before.
func TestConfigErrorDiagnostic_RendersAllMessagesWithoutStepRuleViolation(t *testing.T) {
	steps := []gate.StepResult{{
		StepName:  "traceability_contracts",
		Status:    "fail",
		ConfigErr: true,
		Reason:    "declared dimension broken",
		Violations: []gate.Violation{
			{Rule: "contracts_broken_declared", Severity: "error", Message: "MESSAGE-ONE"},
			{Rule: "contracts_declared_intent_unmet", Severity: "error", Message: "  MESSAGE-TWO  "},
			{Rule: "contracts_broken_declared", Severity: "error", Message: "   "},
		},
	}}

	got := configErrorDiagnostic(steps)
	want := "traceability_contracts: declared dimension broken MESSAGE-ONE MESSAGE-TWO"
	if got != want {
		t.Errorf("a step without a Rule == StepName violation must render every message as before:\n got: %s\nwant: %s", got, want)
	}
}
