// Package gate implements the backstop gate command — the full reconciliation
// kill chain (ADR-0010). Gate orchestrates nine verification steps and produces
// a unified pass/fail result consumed by the GitHub Actions gate action.
package gate

import (
	"context"

	"github.com/bmanson/backstop-core/pkg/waiver"
)

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

	// StepArtifactStatusDrift is the canonical name of the ISSUE-042 native
	// status/reality drift dimension (CLM-009). Like pack_engines /
	// pack_lock_verification / toolchain_enforcement, it is a dynamically-WIRED gate
	// step, NOT one of the nine fixed contract dimensions, so it is registered as a
	// canonical StepName here but intentionally NOT added to the [9]string AllStepNames
	// contract array (which the JSON output schema pins at nine). It carries the BLOCK
	// direction (success-terminal artifact with an absent mandated test) and is the
	// dimension the backstop.yml enforcement.policy entry targets.
	StepArtifactStatusDrift = "artifact_status_drift"
	// StepArtifactStatusDriftAdvisory is the NON-policied advisory surface for the drift
	// dimension's intrinsic WARN direction (delivered-but-open: a non-terminal artifact
	// whose mandated tests are all present). It is a SEPARATE step name specifically so
	// no enforcement.policy entry (which targets StepArtifactStatusDrift) can ever upgrade
	// the WARN to a block — the heuristic-not-proof asymmetry is structural (CLM-006).
	StepArtifactStatusDriftAdvisory = "artifact_status_drift_advisory"
)

// StepRequirementTraceability is the BUNDLE-014 corpus-state invariant step. Like
// status drift, it is dynamically wired and intentionally absent from AllStepNames.
const StepRequirementTraceability = "requirement_traceability"

// StepRequirementTraceabilityAdvisory is the non-policied WARN-only twin for in-flight
// bundle requirement gaps; no enforcement.policy entry can upgrade it to blocking.
const StepRequirementTraceabilityAdvisory = "requirement_traceability_advisory"

// AllStepNames is the ordered list of nine canonical step names for iteration.
var AllStepNames = [9]string{ // nosemgrep: go.core.no-global-mutable-state — immutable canonical step-name contract array
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
	ProjectWide bool `json:"-"`
	// Line is the finding's 1-indexed start line at its reported source location.
	// It is carried from the engine's SARIF region (check.Violation.Line) so the
	// SPEC-049 waiver reconciliation pass can byte-scan the finding's own line for a
	// @waiver token. It is deliberately EXCLUDED from baseline identity/serialization
	// (json:"-") — identity stays line-INDEPENDENT (RegionHash) so a waiver-carrying
	// line number never destabilizes baseline grandfathering. Zero when the engine
	// reported no line (a locationless finding).
	Line int `json:"-"`
	// WaiverHint is a pre-filled `@waiver:<rule>:<reason>:<expiry>` token (SPEC-049
	// REQ-014) surfaced on a still-blocked WAIVABLE finding so acknowledging it is at
	// most as much friction as an engine-native //nolint. Presentation only: like
	// Line it is json:"-" and NOT part of baseline identity, so the expiry date it
	// embeds never destabilizes grandfathering. Empty for non-waivable/structural
	// findings and when no waiver step ran.
	WaiverHint string `json:"-"`
	// Trace is the ISSUE-059 structured companion to Message on requirement_traceability
	// violations: the machine-readable gap_kind/remedy and the bundle/req coordinates a
	// downstream consumer would otherwise have to parse out of the prose. Like Line,
	// ProjectWide, and WaiverHint it is presentation-only and DELIBERATELY EXCLUDED from
	// baseline identity/RegionHash (EnrichViolationIdentity folds only
	// Rule|File|RegionHash(Message|Severity|SourcePack)), so a violation gaining or losing
	// its trace never destabilizes baseline grandfathering. Additive under gate/v1 and
	// omitempty, so consumers reading only rule/file/message/severity are unaffected. Nil
	// for every violation outside the requirement_traceability step.
	Trace        *Trace `json:"trace,omitempty"`
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
	SchemaVersion string `json:"schema_version"`
	// GitSHA is the repository HEAD commit SHA at gate-run time and GeneratedAt the
	// RFC 3339 wall-clock time the run completed (ISSUE-059). Together they are provenance:
	// a corpus-parsing consumer can detect skew between a gate JSON blob and the corpus it
	// parsed it against, which is what makes it safe for per-violation Trace.BundleMaturity
	// to be denormalized rather than re-looked-up. Populated on the CLI runGate path (empty
	// on a non-repo, mirroring the baseline artifact); omitempty so they are additive under
	// the unchanged gate/v1 schema.
	GitSHA          string     `json:"git_sha,omitempty"`
	GeneratedAt     string     `json:"generated_at,omitempty"`
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
	// ActiveWaivers is the SPEC-049 waiver carrier: the set of ACTIVE (valid,
	// unexpired) waivers the reconciliation pass applied on this run. GateResult /
	// Violation carry no reason-code or expiry, so this dedicated field is what
	// feeds the step-9 audit surface (ActiveWaiverFeed) and answers "what are we
	// deliberately ignoring, why, and until when" (REQ-015). Populated by
	// computeWaiverResult via the Run loop; empty on a run with no active waivers.
	ActiveWaivers []waiver.Waiver `json:"active_waivers,omitempty"`
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
