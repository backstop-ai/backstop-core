package gate

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestReportSurface_FormatHuman_RendersWarningConspicuously (CLM-015):
// FormatHuman renders a class-2 advisory conspicuously — a warning marker is
// present in the rendered per-step output, visually distinct from a silent
// "pass". Checked in no-color mode so the marker is a literal token, not just an
// ANSI escape.
func TestReportSurface_FormatHuman_RendersWarningConspicuously(t *testing.T) {
	steps := []StepResult{
		{StepName: StepCoverageThreshold, Status: "warning", Violations: []Violation{
			{Rule: "coverage", Message: "coverage capability absent for go; declare or waive", Severity: "warning"},
		}},
	}
	r := NewGateResultWithScope(steps, nil)
	out := FormatHuman(r, true)

	if !strings.Contains(out, "warning") {
		t.Errorf("human output must render a 'warning' marker for a class-2 step, got:\n%s", out)
	}
	// Conspicuous: the warning step's status token must be distinct from the
	// plain "pass" token. A pass renders "pass"; the warning must not silently
	// render as "pass".
	stepLine := ""
	for _, line := range strings.Split(out, "\n") {
		// The summary-table row is the first line naming the step (the later
		// occurrence is the "<step> violations:" details header).
		if strings.Contains(line, StepCoverageThreshold) {
			stepLine = line
			break
		}
	}
	if stepLine == "" {
		t.Fatalf("no step line for %s in output:\n%s", StepCoverageThreshold, out)
	}
	if !strings.Contains(stepLine, "warning") {
		t.Errorf("the %s step line must carry the warning marker, got %q", StepCoverageThreshold, stepLine)
	}
}

// TestReportSurface_FormatHuman_SummaryReflectsWarnedCount (CLM-030): a gate
// carrying class-2 warning steps reflects the warned count in the FormatHuman
// SUMMARY line (alongside passed/failed/skipped), not only as a per-step
// marker — so the advisory cannot vanish from the at-a-glance summary on a
// green run.
func TestReportSurface_FormatHuman_SummaryReflectsWarnedCount(t *testing.T) {
	steps := []StepResult{
		{StepName: StepTestSubstantiveness, Status: "pass", Violations: []Violation{}},
		{StepName: StepCoverageThreshold, Status: "warning", Violations: []Violation{}},
		{StepName: StepContractSignature, Status: "warning", Violations: []Violation{}},
	}
	r := NewGateResultWithScope(steps, nil)
	out := FormatHuman(r, true)

	var summaryLine string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Steps:") {
			summaryLine = line
		}
	}
	if summaryLine == "" {
		t.Fatalf("no 'Steps:' summary line in output:\n%s", out)
	}
	if !strings.Contains(summaryLine, "warned") {
		t.Errorf("summary line must render the warned count, got %q", summaryLine)
	}
	if !strings.Contains(summaryLine, "2 warned") {
		t.Errorf("summary line must report 2 warned, got %q", summaryLine)
	}
}

// TestReportSurface_FormatJSON_EmitsWarningTaggedEntry (CLM-016): FormatJSON
// emits a class-2 advisory as a machine-readable warning-tagged entry —
// distinguishable from a pass with no advisory via the step status "warning"
// and the violation severity "warning".
func TestReportSurface_FormatJSON_EmitsWarningTaggedEntry(t *testing.T) {
	steps := []StepResult{
		{StepName: StepCoverageThreshold, Status: "warning", Violations: []Violation{
			{Rule: "coverage", Message: "coverage capability absent; declare or waive", Severity: "warning"},
		}},
	}
	r := NewGateResultWithScope(steps, nil)
	data, err := FormatJSON(r)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}
	var envelope struct {
		StepsWarned int `json:"steps_warned"`
		Steps       []struct {
			Status     string `json:"status"`
			Violations []struct {
				Severity string `json:"severity"`
			} `json:"violations"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if envelope.StepsWarned != 1 {
		t.Errorf("steps_warned = %d, want 1", envelope.StepsWarned)
	}
	if len(envelope.Steps) != 1 || envelope.Steps[0].Status != "warning" {
		t.Fatalf("expected one step with status 'warning', got %#v", envelope.Steps)
	}
	if len(envelope.Steps[0].Violations) == 0 || envelope.Steps[0].Violations[0].Severity != "warning" {
		t.Errorf("warning step's violation must be tagged severity 'warning', got %#v", envelope.Steps[0].Violations)
	}
}

// TestReportSurface_PassingGateStillSurfacesAdvisories (CLM-017): a gate that
// passes (exit 0) while carrying class-2 advisories still surfaces those
// advisories in BOTH human and JSON output — the advisory is never representable
// by exit code alone.
func TestReportSurface_PassingGateStillSurfacesAdvisories(t *testing.T) {
	steps := []StepResult{
		{StepName: StepTestSubstantiveness, Status: "pass", Violations: []Violation{}},
		{StepName: StepCoverageThreshold, Status: "warning", Violations: []Violation{
			{Rule: "coverage", Message: "coverage capability absent for go; declare or waive", Severity: "warning"},
		}},
	}
	r := NewGateResultWithScope(steps, nil)
	if !r.Pass {
		t.Fatal("gate with only pass+warning steps must Pass (exit 0)")
	}

	human := FormatHuman(r, true)
	if !strings.Contains(human, "warning") || !strings.Contains(human, "declare or waive") {
		t.Errorf("passing-gate human output must still surface the advisory, got:\n%s", human)
	}

	data, err := FormatJSON(r)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}
	if !strings.Contains(string(data), "\"warning\"") {
		t.Errorf("passing-gate JSON output must still surface the warning-tagged entry, got:\n%s", string(data))
	}
}
