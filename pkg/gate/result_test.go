package gate

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestGate_JSONOutput_StructureComplete verifies that GateResult marshals to JSON
// with schema_version, pass boolean, steps array, and summary fields.
func TestGate_JSONOutput_StructureComplete(t *testing.T) {
	steps := []StepResult{
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
		{StepName: StepCodeCheck, Status: "fail", Violations: []Violation{
			{Rule: "lint", File: "foo.go", Message: "unused var", Severity: "error"},
		}},
		{StepName: StepTestVerification, Status: "skipped", Violations: []Violation{}, Reason: "not implemented"},
	}
	result := NewGateResult(steps)

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal GateResult: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	for _, field := range []string{"schema_version", "pass", "steps", "total_violations", "steps_passed", "steps_failed", "steps_skipped"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("missing field %q in JSON output", field)
		}
	}
}

// TestGate_JSONOutput_StepFieldsPresent verifies each StepResult in JSON
// has step_name, status, and violations fields.
func TestGate_JSONOutput_StepFieldsPresent(t *testing.T) {
	steps := []StepResult{
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
	}
	result := NewGateResult(steps)

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var envelope struct {
		Steps []map[string]json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(envelope.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(envelope.Steps))
	}

	for _, field := range []string{"step_name", "status", "violations"} {
		if _, ok := envelope.Steps[0][field]; !ok {
			t.Errorf("missing field %q in step JSON", field)
		}
	}
}

// TestGate_JSONOutput_SkippedStepHasReason verifies a skipped StepResult
// includes the reason field in JSON output.
func TestGate_JSONOutput_SkippedStepHasReason(t *testing.T) {
	steps := []StepResult{
		{StepName: StepBaselineComparison, Status: "skipped", Violations: []Violation{}, Reason: "baseline not implemented"},
	}
	result := NewGateResult(steps)

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var envelope struct {
		Steps []struct {
			Reason string `json:"reason"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(envelope.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(envelope.Steps))
	}
	if envelope.Steps[0].Reason != "baseline not implemented" {
		t.Errorf("expected reason %q, got %q", "baseline not implemented", envelope.Steps[0].Reason)
	}
}

// TestGate_JSONOutput_SchemaVersionGateV1 verifies GateResult.SchemaVersion is "gate/v1".
func TestGate_JSONOutput_SchemaVersionGateV1(t *testing.T) {
	result := NewGateResult(nil)
	if result.SchemaVersion != "gate/v1" {
		t.Errorf("expected schema_version %q, got %q", "gate/v1", result.SchemaVersion)
	}
}

// TestGate_JSONOutput_SummaryCounts verifies TotalViolations, StepsPassed,
// StepsFailed, and StepsSkipped are computed correctly.
func TestGate_JSONOutput_SummaryCounts(t *testing.T) {
	steps := []StepResult{
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
		{StepName: StepCodeCheck, Status: "fail", Violations: []Violation{
			{Rule: "lint", Message: "v1"},
			{Rule: "lint", Message: "v2"},
		}},
		{StepName: StepTestVerification, Status: "pass", Violations: []Violation{}},
		{StepName: StepTestSubstantiveness, Status: "fail", Violations: []Violation{
			{Rule: "hollow", Message: "v3"},
		}},
		{StepName: StepCoverageThreshold, Status: "skipped", Violations: []Violation{}, Reason: "not implemented"},
		{StepName: StepContractSignature, Status: "pass", Violations: []Violation{}},
		{StepName: StepBaselineComparison, Status: "skipped", Violations: []Violation{}, Reason: "baseline not implemented"},
		{StepName: StepWaiverResolution, Status: "skipped", Violations: []Violation{}, Reason: "waivers not implemented"},
		{StepName: StepLedgerIntegrity, Status: "skipped", Violations: []Violation{}, Reason: "ledger not implemented"},
	}
	result := NewGateResult(steps)

	if result.TotalViolations != 3 {
		t.Errorf("expected TotalViolations=3, got %d", result.TotalViolations)
	}
	if result.StepsPassed != 3 {
		t.Errorf("expected StepsPassed=3, got %d", result.StepsPassed)
	}
	if result.StepsFailed != 2 {
		t.Errorf("expected StepsFailed=2, got %d", result.StepsFailed)
	}
	if result.StepsSkipped != 4 {
		t.Errorf("expected StepsSkipped=4, got %d", result.StepsSkipped)
	}
}

// TestGate_JSONOutput_PassTrueWhenAllGreen verifies Pass is true when all steps pass.
func TestGate_JSONOutput_PassTrueWhenAllGreen(t *testing.T) {
	steps := []StepResult{
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
		{StepName: StepCodeCheck, Status: "pass", Violations: []Violation{}},
	}
	result := NewGateResult(steps)
	if !result.Pass {
		t.Error("expected Pass=true when all steps pass")
	}
}

// TestGate_JSONOutput_PassFalseWhenAnyFail verifies Pass is false when any step fails.
func TestGate_JSONOutput_PassFalseWhenAnyFail(t *testing.T) {
	steps := []StepResult{
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
		{StepName: StepCodeCheck, Status: "fail", Violations: []Violation{
			{Rule: "lint", Message: "err"},
		}},
	}
	result := NewGateResult(steps)
	if result.Pass {
		t.Error("expected Pass=false when any step fails")
	}
}

// TestGate_JSONOutput_PassTrueWithSkippedSteps verifies Pass is true when
// steps pass and remaining steps are skipped.
func TestGate_JSONOutput_PassTrueWithSkippedSteps(t *testing.T) {
	steps := []StepResult{
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
		{StepName: StepBaselineComparison, Status: "skipped", Violations: []Violation{}, Reason: "not implemented"},
	}
	result := NewGateResult(steps)
	if !result.Pass {
		t.Error("expected Pass=true when steps pass and remaining are skipped")
	}
}

// TestGate_JSONOutput_PassTrueWhenAllSkipped verifies Pass is true when all
// nine steps are skipped (no executed step failed).
func TestGate_JSONOutput_PassTrueWhenAllSkipped(t *testing.T) {
	var steps []StepResult
	for _, name := range AllStepNames {
		steps = append(steps, StepResult{StepName: name, Status: "skipped", Violations: []Violation{}, Reason: "not implemented"})
	}
	result := NewGateResult(steps)
	if !result.Pass {
		t.Error("expected Pass=true when all steps are skipped")
	}
}

// TestGate_CanonicalStepNames_AllPresent verifies the constant list of nine
// canonical step names.
func TestGate_CanonicalStepNames_AllPresent(t *testing.T) {
	if len(AllStepNames) != 9 {
		t.Fatalf("expected 9 canonical step names, got %d", len(AllStepNames))
	}
}

// TestGate_CanonicalStepNames_ExactMatch verifies step names match exact
// strings from REQ-011.
func TestGate_CanonicalStepNames_ExactMatch(t *testing.T) {
	expected := [9]string{
		"artifact_validation",
		"code_check",
		"test_verification",
		"test_substantiveness",
		"coverage_threshold",
		"contract_signature",
		"baseline_comparison",
		"waiver_resolution",
		"ledger_integrity",
	}
	if AllStepNames != expected {
		t.Errorf("canonical step names mismatch:\n  got:  %v\n  want: %v", AllStepNames, expected)
	}
}

func TestGateIntegration_JSONSourcePackField(t *testing.T) {
	withPack := Violation{
		Rule:       "test-org/test-pack/no-eval",
		Message:    "bad call",
		SourcePack: "test-org/test-pack",
	}
	data, err := json.Marshal(withPack)
	if err != nil {
		t.Fatalf("marshal with pack: %v", err)
	}
	if !strings.Contains(string(data), `"source_pack":"test-org/test-pack"`) {
		t.Fatalf("expected source_pack field in JSON, got: %s", string(data))
	}

	native := Violation{
		Rule:    "code_check",
		Message: "native violation",
	}
	nativeData, err := json.Marshal(native)
	if err != nil {
		t.Fatalf("marshal native: %v", err)
	}
	if strings.Contains(string(nativeData), `"source_pack"`) {
		t.Fatalf("did not expect source_pack for native violation, got: %s", string(nativeData))
	}
}

func TestGateResult_BaselineScopedDiff_FiltersOutOfScopeViolations(t *testing.T) {
	scope := newGateScope("", GateScopeModeDiff, []string{"changed.go"}, nil)
	current := []Violation{
		{Rule: "code_check/new", File: "changed.go", Message: "new changed-file issue", Severity: "error"},
		{Rule: "code_check/fixed", File: "unchanged.go", Message: "outside scope", Severity: "error"},
	}
	filtered := filterViolations(scope, current)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 scoped violation, got %d (%#v)", len(filtered), filtered)
	}
	if filtered[0].File != "changed.go" {
		t.Fatalf("expected changed-file violation to remain, got %#v", filtered[0])
	}
}

func TestGateResult_BaselineComparisonStep_AdditiveDiagnosticsContract(t *testing.T) {
	result := NewGateResult([]StepResult{{
		StepName: StepBaselineComparison,
		Status:   "fail",
		Violations: []Violation{{
			Rule:     "code_check/new",
			File:     "changed.go",
			Message:  "new baseline differential violation",
			Severity: "error",
		}},
	}})

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw struct {
		Steps []map[string]json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(raw.Steps) != 1 {
		t.Fatalf("expected one step, got %d", len(raw.Steps))
	}
	if _, ok := raw.Steps[0]["new_violations"]; !ok {
		t.Errorf("expected additive baseline diagnostics field %q", "new_violations")
	}
}

// TestGateViolation_CarriesProperties pins CLM-003: gate.Violation carries the
// structured Properties map additively — it round-trips through the struct and
// serializes under `properties` (omitempty) only when populated, so a consumer
// reading only rule/file/message/severity is unaffected (additive under gate/v1).
func TestGateViolation_CarriesProperties(t *testing.T) {
	v := Violation{
		Rule:     StepTestSubstantiveness,
		File:     "a_test.go",
		Message:  "test X has no assertions (hollow)",
		Severity: "error",
		Properties: map[string]string{
			"func":   "surfaces a plan spec_id in the response",
			"symbol": "readmodel",
		},
	}
	if v.Properties["func"] != "surfaces a plan spec_id in the response" {
		t.Fatalf("Properties[func] = %q, want the verbatim spaced value", v.Properties["func"])
	}

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back struct {
		Properties map[string]string `json:"properties"`
	}
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Properties["func"] != "surfaces a plan spec_id in the response" || back.Properties["symbol"] != "readmodel" {
		t.Errorf("properties did not round-trip through JSON: %v", back.Properties)
	}

	// omitempty: a violation with no properties omits the key entirely.
	bare, err := json.Marshal(Violation{Rule: "r", Message: "m", Severity: "error"})
	if err != nil {
		t.Fatalf("marshal bare: %v", err)
	}
	if strings.Contains(string(bare), "\"properties\"") {
		t.Errorf("empty Properties must be omitted (omitempty), got %s", string(bare))
	}
}

// TestProperties_ExcludedFromBaselineIdentity pins CLM-004: Properties is
// DELIBERATELY excluded from baseline identity and RegionHash, exactly like Trace —
// a violation gaining or losing Properties yields the SAME Identity/IdentityHash/
// RegionHash, so it never destabilizes baseline grandfathering.
func TestProperties_ExcludedFromBaselineIdentity(t *testing.T) {
	base := Violation{
		Rule:       "backstop/substantiveness/referenced-symbol-go",
		File:       "pkg/x/x_test.go",
		Message:    "test X has no assertions (hollow)",
		Severity:   "error",
		SourcePack: "backstop/substantiveness",
	}
	withProps := base
	withProps.Properties = map[string]string{"func": "TestX", "symbol": "x"}

	a := EnrichViolationIdentity(base)
	b := EnrichViolationIdentity(withProps)

	if a.IdentityHash != b.IdentityHash {
		t.Errorf("IdentityHash changed when Properties were added: %q vs %q", a.IdentityHash, b.IdentityHash)
	}
	if a.Identity != b.Identity {
		t.Errorf("Identity changed when Properties were added: %q vs %q", a.Identity, b.Identity)
	}
	if a.RegionHash != b.RegionHash {
		t.Errorf("RegionHash changed when Properties were added: %q vs %q", a.RegionHash, b.RegionHash)
	}

	// And a DIFFERENT properties map must not move identity either.
	withOther := base
	withOther.Properties = map[string]string{"func": "a b c", "symbol": "other"}
	c := EnrichViolationIdentity(withOther)
	if c.IdentityHash != a.IdentityHash {
		t.Errorf("IdentityHash moved with a different Properties map: %q vs %q", c.IdentityHash, a.IdentityHash)
	}
}
