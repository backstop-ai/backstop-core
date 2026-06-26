package gate

// Per-dimension enforcement policy. A consumer's backstop.yml declares, per gate
// dimension (the step/gate_type name), an enforcement level and whether pre-existing
// findings are grandfathered against the baseline. This is the incremental-adoption
// surface: turn some gates to block now, leave others at warn/off, and ratchet up —
// all keyed on backstop's universal dimension vocabulary, never a tool or language.

const (
	PolicyOff   = "off"   // don't enforce: the step is reported skipped
	PolicyWarn  = "warn"  // surface findings, never fail the gate (loud-not-blocking)
	PolicyBlock = "block" // fail the gate on findings (the default)
)

// DimensionPolicy is the resolved policy for one gate dimension: the enforcement
// level and whether the baseline grandfathers pre-existing findings (so only net-new
// findings count). Empty Level defaults to block.
type DimensionPolicy struct {
	Level    string
	Baseline bool
}

// policyMetaStep reports whether a step is a meta/deferred step that the policy table
// never targets (its status is bookkeeping, not a dimension a consumer enforces).
func policyMetaStep(name string) bool {
	switch name {
	case StepWaiverResolution, StepLedgerIntegrity, "pack_lock_verification":
		return true
	}
	return false
}

// ApplyPolicy rewrites step results per the per-dimension enforcement policy. For each
// step with a policy entry: level off -> skipped; level warn -> warning (never fails);
// level block -> fail only when findings remain after baseline grandfathering. When
// baseline is true the step's pre-existing findings are subtracted (per-dimension), so
// the step acts only on net-new. A step with NO policy entry is left unchanged, so the
// feature is opt-in and backward compatible. When any policy is configured the aggregate
// baseline_comparison step is neutralized — per-dimension baseline supersedes it.
//
// Config errors and capability-absent skips are preserved untouched: policy sets how
// strictly real findings gate, it never masks a config error or fabricates a capability.
func ApplyPolicy(steps []StepResult, baseline *BaselineArtifact, policy map[string]DimensionPolicy, scope *GateScope) []StepResult {
	if len(policy) == 0 {
		return steps
	}
	out := make([]StepResult, 0, len(steps))
	for _, s := range steps {
		if s.StepName == StepBaselineComparison {
			s.Status = "skipped"
			s.Reason = "superseded by per-dimension enforcement policy"
			s.Violations = []Violation{}
			out = append(out, s)
			continue
		}
		p, ok := policy[s.StepName]
		if !ok || policyMetaStep(s.StepName) || s.ConfigErr || s.Status == "skipped" {
			out = append(out, s)
			continue
		}

		level := p.Level
		if level == "" {
			level = PolicyBlock
		}
		if level == PolicyOff {
			s.Status = "skipped"
			s.Reason = "disabled by enforcement policy (level: off)"
			s.Violations = []Violation{}
			s.NewViolations = []Violation{}
			out = append(out, s)
			continue
		}

		// Findings that still count after baseline grandfathering.
		counted := s.Violations
		if p.Baseline && baseline != nil {
			cmp := CompareBaseline(s.Violations, baseline, BaselineCompareOptions{Scope: scope})
			counted = cmp.NewViolations
			s.NewViolations = cmp.NewViolations
			s.FixedViolations = cmp.FixedViolations
		}

		switch {
		case level == PolicyWarn:
			if len(counted) > 0 {
				s.Status = "warning"
			} else {
				s.Status = "pass"
			}
		default: // block
			if len(counted) > 0 {
				s.Status = "fail"
			} else {
				s.Status = "pass"
			}
		}
		out = append(out, s)
	}
	return out
}
