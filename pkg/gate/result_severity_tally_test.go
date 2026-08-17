package gate

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// motivatingTallyStep builds the ISSUE-100 instance: ONE coverage_threshold step
// carrying a blocking shortfall alongside a non-blocking coverage-exclusion
// notice. Pre-fix this step tallied as 2 violations; exactly 1 of its entries
// blocks the verdict.
func motivatingTallyStep() StepResult {
	return StepResult{
		StepName: StepCoverageThreshold,
		Status:   "fail",
		Violations: []Violation{
			{Rule: "coverage_threshold", File: "pkg/gate/step_coverage.go", Message: "coverage 71.2% below threshold 80.0%", Severity: "error"},
			{Rule: "coverage_exclusion", File: "pkg/gate/output.go", Message: "file excluded from coverage measurement", Severity: "warning"},
		},
	}
}

// TestSeverityTally_TotalViolationsCountsBlockingOnly (CLM-001): total_violations
// counts only entries that BLOCK the verdict. On the ISSUE-100 motivating
// instance — one severity:error shortfall plus one severity:warning exclusion
// notice — that is 1, where the pre-fix tree reported 2.
func TestSeverityTally_TotalViolationsCountsBlockingOnly(t *testing.T) {
	r := NewGateResultWithScope([]StepResult{motivatingTallyStep()}, nil)

	if r.TotalViolations != 1 {
		t.Errorf("TotalViolations = %d, want 1 — only the blocking entry counts; a warning-severity finding is non-blocking by the pack severity contract (blocksVerdict)", r.TotalViolations)
	}
}

// TestSeverityTally_TotalWarningsCountsExplicitWarnings (CLM-002): the envelope
// gains total_warnings, counting explicitly warning-severity entries. It carries
// NO omitempty, so a consumer can read the zero on a warning-free run.
func TestSeverityTally_TotalWarningsCountsExplicitWarnings(t *testing.T) {
	r := NewGateResultWithScope([]StepResult{motivatingTallyStep()}, nil)

	if r.TotalWarnings != 1 {
		t.Errorf("TotalWarnings = %d, want 1", r.TotalWarnings)
	}

	data, err := FormatJSON(r)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	raw, ok := envelope["total_warnings"]
	if !ok {
		t.Fatalf("JSON envelope must carry the key total_warnings, got keys %v", sortedKeys(envelope))
	}
	if strings.TrimSpace(string(raw)) != "1" {
		t.Errorf("total_warnings = %s, want 1", string(raw))
	}

	// The zero must remain READABLE — no omitempty.
	clean := NewGateResultWithScope([]StepResult{
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
		{StepName: StepCodeCheck, Status: "fail", Violations: []Violation{
			{Rule: "lint", Message: "blocking", Severity: "error"},
		}},
	}, nil)
	if clean.TotalWarnings != 0 {
		t.Errorf("warning-free result TotalWarnings = %d, want 0", clean.TotalWarnings)
	}
	cleanData, err := FormatJSON(clean)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}
	var cleanEnvelope map[string]json.RawMessage
	if err := json.Unmarshal(cleanData, &cleanEnvelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	rawZero, ok := cleanEnvelope["total_warnings"]
	if !ok {
		t.Fatalf("total_warnings must be PRESENT (not omitted) with value 0 on a warning-free run; the field must carry no omitempty. keys: %v", sortedKeys(cleanEnvelope))
	}
	if strings.TrimSpace(string(rawZero)) != "0" {
		t.Errorf("warning-free total_warnings = %s, want 0", string(rawZero))
	}
}

// TestSeverityTally_PartitionIsTotal (CLM-002): the split is a PARTITION —
// total_violations + total_warnings accounts for every entry present in every
// step's Violations slice. Nothing dropped, nothing double-counted.
func TestSeverityTally_PartitionIsTotal(t *testing.T) {
	steps := []StepResult{
		{StepName: StepCodeCheck, Status: "fail", Violations: []Violation{
			{Rule: "lint", Message: "a", Severity: "error"},
			{Rule: "lint", Message: "b", Severity: "warning"},
			{Rule: "lint", Message: "c"},
		}},
		{StepName: StepCoverageThreshold, Status: "warning", Violations: []Violation{
			{Rule: "coverage", Message: "d", Severity: "warning"},
			{Rule: "coverage", Message: "e", Severity: "Warning"},
		}},
		{StepName: StepContractSignature, Status: "pass", Violations: []Violation{}},
		{StepName: StepLedgerIntegrity, Status: "fail", Violations: []Violation{
			{Rule: "ledger", Message: "f"},
		}},
	}

	// Expected total is SUMMED FROM THE FIXTURE, never a literal — a literal
	// stops falsifying the moment someone edits the fixture above.
	expected := 0
	for _, s := range steps {
		expected += len(s.Violations)
	}

	r := NewGateResultWithScope(steps, nil)
	if got := r.TotalViolations + r.TotalWarnings; got != expected {
		t.Errorf("TotalViolations(%d) + TotalWarnings(%d) = %d, want %d — the split must be a total partition of every reported entry",
			r.TotalViolations, r.TotalWarnings, got, expected)
	}
}

// TestSeverityTally_UnsetSeverityCountsAsBlocking (CLM-003): THE FAIL-CLOSED
// RULE. A violation with no Severity set BLOCKS the verdict (blocksVerdict
// exempts only an explicit "warning"), so it must count toward total_violations
// and NOT toward total_warnings. A `Severity == "error"` tally would classify it
// as a notice while the gate blocks on it — fail-OPEN, and worse than the
// over-reporting defect this lane fixes.
func TestSeverityTally_UnsetSeverityCountsAsBlocking(t *testing.T) {
	steps := []StepResult{
		{StepName: StepCodeCheck, Status: "fail", Violations: []Violation{
			{Rule: "lint", Message: "no severity declared"},
		}},
	}
	r := NewGateResultWithScope(steps, nil)

	if r.TotalViolations != 1 {
		t.Errorf("TotalViolations = %d, want 1 — an UNSET severity blocks (fail-closed), so it must be counted as blocking", r.TotalViolations)
	}
	if r.TotalWarnings != 0 {
		t.Errorf("TotalWarnings = %d, want 0 — an unset severity is not a warning; counting it in both classes would double-count", r.TotalWarnings)
	}
	// The tally's classification must agree with the verdict it sits beside.
	if v := StepVerdict(steps[0].Violations); v != "fail" {
		t.Fatalf("StepVerdict = %q, want \"fail\" — the fixture's premise is that an unset severity BLOCKS", v)
	}
}

// TestSeverityTally_CaseAndWhitespaceNormalizedLikeVerdict (CLM-003, CLM-004):
// the tally normalizes severity EXACTLY as blocksVerdict does (EqualFold over a
// TrimSpace'd value), so tally and verdict can never disagree about a given
// violation. A hand-rolled `== "warning"` copy fails this.
func TestSeverityTally_CaseAndWhitespaceNormalizedLikeVerdict(t *testing.T) {
	variants := []string{"Warning", "WARNING", " warning ", "warning"}

	violations := make([]Violation, 0, len(variants))
	for _, sev := range variants {
		violations = append(violations, Violation{Rule: "coverage", Message: "notice " + sev, Severity: sev})
	}

	r := NewGateResultWithScope([]StepResult{
		{StepName: StepCoverageThreshold, Status: "warning", Violations: violations},
	}, nil)

	if r.TotalWarnings != len(variants) {
		t.Errorf("TotalWarnings = %d, want %d — every case/whitespace variant of \"warning\" must tally as a warning, matching blocksVerdict's EqualFold+TrimSpace", r.TotalWarnings, len(variants))
	}
	if r.TotalViolations != 0 {
		t.Errorf("TotalViolations = %d, want 0 — no variant of \"warning\" blocks", r.TotalViolations)
	}

	// AGREEMENT, not merely case-insensitivity: the verdict must classify these
	// identical values the same way.
	for _, sev := range variants {
		single := []Violation{{Rule: "coverage", Message: "notice", Severity: sev}}
		if v := StepVerdict(single); v != "warning" {
			t.Errorf("StepVerdict for severity %q = %q, want \"warning\" — the tally and the verdict must agree", sev, v)
		}
	}
}

// TestSeverityTally_NoSecondBlockingDefinition (CLM-004): THE DRIFT GUARD.
// Exactly one definition of "blocking" exists in pkg/gate — blocksVerdict in
// policy.go. Neither the JSON envelope (result.go) nor the human renderer
// (output.go) may hand-roll a severity comparison of its own: a copy silently
// drifts from the verdict the next time the pack severity contract moves, which
// reintroduces ISSUE-100 one layer over.
//
// The forbidding predicate is the FIELD-ACCESS form ".Severity" (leading dot).
// It is deliberately NOT `Severity ==` / `Severity !=`: an inlined
// strings.EqualFold(strings.TrimSpace(v.Severity), "warning") — the natural
// in-repo idiom, see pkg/gate/baseline.go — contains neither operator but is
// exactly the second definition this claim forbids. Any hand-rolled approach
// must read the field to compare it, so ".Severity" catches all of them. The
// bare token "Severity" (no dot) is NOT forbidden: result.go legitimately
// declares the Violation.Severity struct field and comments name it.
//
// CONTENT scan only — no git status, no diff, no tree-state assertion: this tree
// is shared and a tree-state check blames whoever happens to run it.
func TestSeverityTally_NoSecondBlockingDefinition(t *testing.T) {
	const authority = "blocksVerdict (pkg/gate/policy.go) is the single authority on blockingness; " +
		"call it — via tallyBySeverity or directly — instead of comparing the Severity field here"

	// Sharper diagnostics when a naive copy is what matched; the field-access
	// form below is what decides pass/fail.
	sharperForms := []string{"Severity ==", "Severity !="}

	for _, path := range []string{"result.go", "output.go"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		src := string(data)

		for _, form := range sharperForms {
			if strings.Contains(src, form) {
				t.Errorf("%s contains a hand-rolled severity comparison %q. %s", path, form, authority)
			}
		}
		if strings.Contains(src, ".Severity") {
			t.Errorf("%s reads the .Severity field directly, creating a SECOND definition of blocking. %s", path, authority)
		}
	}
}

// TestSeverityTally_VerdictSurfacesUnchanged (CLM-009): REPORTING ONLY. The
// tally split leaves Pass, per-step Status, and the
// StepsPassed/Failed/Skipped/Warned counts exactly as they were — the ratified
// severity-aware verdict in policy.go is neither reopened nor duplicated.
func TestSeverityTally_VerdictSurfacesUnchanged(t *testing.T) {
	steps := []StepResult{
		motivatingTallyStep(),
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
		{StepName: StepContractSignature, Status: "warning", Violations: []Violation{
			{Rule: "contracts", Message: "capability absent", Severity: "warning"},
		}},
		{StepName: StepLedgerIntegrity, Status: "skipped", Violations: []Violation{}},
	}
	r := NewGateResultWithScope(steps, nil)

	if r.Pass {
		t.Error("Pass = true, want false — the fixture carries a failing step and the tally must not change the verdict")
	}
	if r.Steps[0].Status != "fail" {
		t.Errorf("step status = %q, want \"fail\" — per-step Status is untouched by the tally", r.Steps[0].Status)
	}
	if r.StepsPassed != 1 {
		t.Errorf("StepsPassed = %d, want 1", r.StepsPassed)
	}
	if r.StepsFailed != 1 {
		t.Errorf("StepsFailed = %d, want 1", r.StepsFailed)
	}
	if r.StepsWarned != 1 {
		t.Errorf("StepsWarned = %d, want 1", r.StepsWarned)
	}
	if r.StepsSkipped != 1 {
		t.Errorf("StepsSkipped = %d, want 1", r.StepsSkipped)
	}
}

// sortedKeys renders an envelope's keys for a readable failure message.
func sortedKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
