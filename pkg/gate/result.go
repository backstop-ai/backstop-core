// Package gate implements the backstop gate command — the full reconciliation
// kill chain (ADR-0010). Gate orchestrates nine verification steps and produces
// a unified pass/fail result consumed by the GitHub Actions gate action.
package gate

import "context"

// Canonical step names (REQ-011). These are part of the JSON output contract
// and must not change without a schema version bump.
const (
	StepArtifactValidation  = "artifact_validation"
	StepCodeCheck           = "code_check"
	StepTestVerification    = "test_verification"
	StepTestSubstantiveness = "test_substantiveness"
	StepCoverageThreshold   = "coverage_threshold"
	StepContractSignature   = "contract_signature"
	StepBaselineComparison  = "baseline_comparison"
	StepWaiverResolution    = "waiver_resolution"
	StepLedgerIntegrity     = "ledger_integrity"
)

// AllStepNames is the ordered list of nine canonical step names for iteration.
var AllStepNames = [9]string{
	StepArtifactValidation,
	StepCodeCheck,
	StepTestVerification,
	StepTestSubstantiveness,
	StepCoverageThreshold,
	StepContractSignature,
	StepBaselineComparison,
	StepWaiverResolution,
	StepLedgerIntegrity,
}

// Violation represents a single gate violation.
type Violation struct {
	Rule       string `json:"rule"`
	File       string `json:"file,omitempty"`
	Message    string `json:"message"`
	Severity   string `json:"severity,omitempty"`
	SourcePack string `json:"source_pack,omitempty"`
	// ProjectWide marks a violation as originating from a project-wide pass
	// (build/typecheck), independent of its Rule string. Project-wide
	// violations are NEVER scope-filtered (Ratified Design Constraint 3): a
	// change to a.ts that breaks a type in unchanged b.ts must fail a
	// diff-scoped gate even though the violation references out-of-scope b.ts.
	// Baseline comparison (gate step 7) remains the suppression mechanism for
	// pre-existing project-wide violations. The field is intentionally NOT
	// serialized (json:"-"): baseline identity hashing ignores it and
	// LoadBaseline is a non-strict unmarshal, so omitting it keeps baseline
	// identity stable across this change.
	ProjectWide  bool   `json:"-"`
	Identity     string `json:"identity"`
	IdentityHash string `json:"identity_hash"`
	RegionHash   string `json:"region_hash"`
}

// StepResult holds the result of a single gate step.
type StepResult struct {
	StepName         string      `json:"step_name"`
	Status           string      `json:"status"` // "pass", "fail", "skipped"
	DurationMS       int64       `json:"duration_ms,omitempty"`
	Violations       []Violation `json:"violations"`
	NewViolations    []Violation `json:"new_violations"`
	FixedViolations  []Violation `json:"fixed_violations"`
	SeededViolations []Violation `json:"seeded_violations"`
	Reason           string      `json:"reason,omitempty"`
	ConfigErr        bool        `json:"-"` // internal flag for config error propagation
}

// GateResult holds the unified gate output including all step results.
type GateResult struct {
	SchemaVersion   string     `json:"schema_version"`
	Scope           *GateScope `json:"scope,omitempty"`
	Pass            bool       `json:"pass"`
	TotalViolations int        `json:"total_violations"`
	StepsPassed     int        `json:"steps_passed"`
	StepsFailed     int        `json:"steps_failed"`
	StepsSkipped    int        `json:"steps_skipped"`
	// StepsWarned counts steps with the non-failing "warning" status (SPEC-036
	// REQ-005). A warning is loud-but-passing: it is counted here and rendered in
	// the FormatHuman summary line so a class-2 capability-absent advisory cannot
	// vanish from the at-a-glance summary on a green run, but it does NOT flip
	// Pass.
	StepsWarned int          `json:"steps_warned"`
	Steps       []StepResult `json:"steps"`
}

// StepFunc is the common signature for gate step functions.
type StepFunc func(ctx context.Context) StepResult

// NewGateResult computes summary fields from the given step results.
// Pass is true if no step has status "fail". SchemaVersion is always "gate/v1".
func NewGateResult(steps []StepResult) GateResult {
	return NewGateResultWithScope(steps, nil)
}

// NewGateResultWithScope computes summary fields and attaches gate scope.
func NewGateResultWithScope(steps []StepResult, scope *GateScope) GateResult {
	r := GateResult{
		SchemaVersion: "gate/v1",
		Scope:         scope,
		Pass:          true,
		Steps:         []StepResult{},
	}

	for _, s := range steps {
		if s.Violations == nil {
			s.Violations = []Violation{}
		}
		for i := range s.Violations {
			s.Violations[i] = EnrichViolationIdentity(s.Violations[i])
		}
		if s.NewViolations == nil {
			s.NewViolations = []Violation{}
		}
		for i := range s.NewViolations {
			s.NewViolations[i] = EnrichViolationIdentity(s.NewViolations[i])
		}
		if s.FixedViolations == nil {
			s.FixedViolations = []Violation{}
		}
		for i := range s.FixedViolations {
			s.FixedViolations[i] = EnrichViolationIdentity(s.FixedViolations[i])
		}
		if s.SeededViolations == nil {
			s.SeededViolations = []Violation{}
		}
		for i := range s.SeededViolations {
			s.SeededViolations[i] = EnrichViolationIdentity(s.SeededViolations[i])
		}

		switch s.Status {
		case "pass":
			r.StepsPassed++
		case "fail":
			r.StepsFailed++
			r.Pass = false
		case "skipped":
			r.StepsSkipped++
		case "warning":
			// Non-failing (Sharp Edge 3): a warning is counted but MUST NOT flip
			// Pass — class 2 (capability-absent) warns-and-passes (exit 0).
			r.StepsWarned++
		}
		r.TotalViolations += len(s.Violations)
		r.Steps = append(r.Steps, s)
	}

	if len(steps) == 0 {
		r.Steps = []StepResult{}
	}

	return r
}
