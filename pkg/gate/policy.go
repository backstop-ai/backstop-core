package gate

import "strings"

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

const (
	// AppliesToNewCode grandfathers pre-existing findings against the baseline, so only
	// net-new findings count (the ratchet).
	AppliesToNewCode = "new-code"
	// AppliesToAllCode blocks on the TOTAL — every finding counts regardless of the
	// baseline (zero tolerance). An ABSENT/empty AppliesTo defaults to this (the strict
	// floor): a bare dimension is never silently grandfathered.
	AppliesToAllCode = "all-code"
)

// DimensionPolicy is the resolved policy for one gate dimension: the enforcement level
// and which violations count (AppliesTo). AppliesTo == "new-code" grandfathers
// pre-existing findings against the baseline (only net-new counts); "all-code" — and
// an ABSENT/empty AppliesTo — counts every finding (block on total). Empty Level
// defaults to block. Level and applies-to are orthogonal knobs.
//
// Sources is the OPTIONAL per-PACK / per-rule-SOURCE scoping (SPEC-047 REQ-007),
// keyed by gate.Violation.SourcePack. A source-scoped override's level+applies-to apply
// ONLY to findings from that pack within the dimension; every OTHER pack's findings
// keep the dimension default (or their own scoped override). This is the mechanism
// the REQ-006 flip is expressed through — `backstop/self` flips to block + all-code
// (zero baseline) on the shared `pack_engines` dimension while go-standards/go-toolchain
// keep their new-code-grandfathered style debt. Absent Sources ⇒ the dimension-only path
// runs unchanged (backward compatible).
type DimensionPolicy struct {
	Level     string
	AppliesTo string
	Sources   map[string]DimensionPolicy
}

// policyMetaStep reports whether a step is a meta/deferred step that the policy table
// never targets (its status is bookkeeping, not a dimension a consumer enforces).
// blocksVerdict reports whether a violation should count toward FAILING a step.
//
// SEVERITY, NOT COUNT. Recorded founder law is "loud != blocking": block defects and
// broken promises, warn-with-guidance for capability signals. A warning that fails the
// gate is a contradiction in terms — and it shipped: CI run 30395875188 failed
// coverage_threshold whose ONLY violation was the severity=warning coverage-exclusion
// NOTICE, because both verdict paths here counted entries without reading this field.
//
// AN UNSET SEVERITY BLOCKS. Only an explicit "warning" is exempt, so a producer that
// omits the field fails closed rather than silently escaping enforcement.
func blocksVerdict(v Violation) bool {
	return !strings.EqualFold(strings.TrimSpace(v.Severity), "warning")
}

// blockingViolations returns only the entries that count toward a failing verdict.
// It NEVER filters what is REPORTED — StepResult.Violations keeps every entry, because
// non-blocking must not mean invisible.
func blockingViolations(violations []Violation) []Violation {
	blocking := make([]Violation, 0, len(violations))
	for _, v := range violations {
		if blocksVerdict(v) {
			blocking = append(blocking, v)
		}
	}
	return blocking
}

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
// AppliesTo is "new-code" the step's pre-existing findings are subtracted (per-dimension),
// so the step acts only on net-new; "all-code" (and an absent/empty AppliesTo) counts
// every finding (block on total — the strict default). A step with NO policy entry is
// left unchanged, so the feature is opt-in and backward compatible. When any policy is
// configured the aggregate baseline_comparison step is neutralized — per-dimension
// applies-to supersedes it.
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

		// PER-PACK / per-rule-SOURCE scoping (SPEC-047 REQ-007): when the entry carries
		// source overrides, each violation's effective level+baseline is resolved from
		// its SourcePack, so a scoped entry (e.g. backstop/self → block + zero baseline)
		// applies ONLY to that pack's findings and every OTHER pack keeps the dimension
		// default. The dimension-only path (no Sources) below is byte-for-byte the
		// prior behavior (backward compatible, CLM-036).
		if len(p.Sources) > 0 {
			out = append(out, applyScopedPolicy(s, p, baseline, scope))
			continue
		}

		// Findings that still count after baseline grandfathering.
		counted := s.Violations
		if p.AppliesTo == AppliesToNewCode && baseline != nil {
			cmp := CompareBaseline(s.Violations, baseline, BaselineCompareOptions{Scope: scope})
			counted = cmp.NewViolations
			s.NewViolations = cmp.NewViolations
			s.FixedViolations = cmp.FixedViolations
		}

		// Only severity-blocking entries decide the verdict; the full list is still
		// reported (see blocksVerdict).
		countedBlocking := blockingViolations(counted)
		if level == PolicyWarn {
			// warn: surface everything, fail nothing.
			s.Status = "pass"
			if len(counted) > 0 {
				s.Status = "warning"
			}
		} else {
			// block: only severity-blocking entries fail; a surfaced-but-non-blocking
			// entry still reports "warning" so the notice is not silently swallowed.
			s.Status = "pass"
			if len(counted) > 0 {
				s.Status = "warning"
			}
			if len(countedBlocking) > 0 {
				s.Status = "fail"
			}
		}
		out = append(out, s)
	}
	return out
}

// applyScopedPolicy resolves a step's verdict when the dimension policy carries
// per-PACK/per-rule-SOURCE overrides (SPEC-047 REQ-007). Each violation's effective
// level+applies-to is chosen by its SourcePack — p.Sources[SourcePack] when present,
// else the dimension default {p.Level, p.AppliesTo}. An applies-to:new-code source
// counts a violation only when it is net-new (grandfathering preserved); an
// applies-to:all-code source (or an absent/empty applies-to) counts EVERY one of its
// findings (block on total). The step fails on any counted block-level violation, warns
// on counted warn-level violations, else passes. s.NewViolations is set to the blocking
// counted set so the report attributes the verdict. The net-new set is computed ONCE
// over the whole step; a new-code source reads it, an all-code source ignores it.
//
// NIL-BASELINE FAIL-LOUD: with NO baseline (a fresh checkout before the CI-pulled
// baseline is present), an applies-to:new-code source CANNOT grandfather — every finding
// counts, exactly MIRRORING the unscoped path (which also counts all findings when
// baseline is nil). This is the anti-vacuous-green invariant: a missing baseline must
// never SILENTLY grandfather a whole dimension to green — the degraded case blocks,
// it does not pass. (Verified by TestPolicy_ScopedNilBaselineBlocksFailLoudNotSilentGreen.)
func applyScopedPolicy(s StepResult, p DimensionPolicy, baseline *BaselineArtifact, scope *GateScope) StepResult {
	newSet := map[string]bool{}
	if baseline != nil {
		cmp := CompareBaseline(s.Violations, baseline, BaselineCompareOptions{Scope: scope})
		for _, v := range cmp.NewViolations {
			newSet[EnrichViolationIdentity(v).IdentityHash] = true
		}
		s.FixedViolations = cmp.FixedViolations
	}

	blocking := []Violation{}
	warned := 0
	for _, v := range s.Violations {
		eff := p
		if override, ok := p.Sources[v.SourcePack]; ok {
			eff = override
		}
		lvl := eff.Level
		if lvl == "" {
			lvl = PolicyBlock
		}
		if lvl == PolicyOff {
			continue
		}

		counts := true
		if eff.AppliesTo == AppliesToNewCode && baseline != nil {
			// Grandfathering preserved for this source: only net-new counts. Gated on a
			// PRESENT baseline — with no baseline (baseline == nil) grandfathering is
			// impossible, so counts stays true and the finding blocks (fail-loud),
			// mirroring the unscoped path. A nil baseline must never silently grandfather.
			counts = newSet[EnrichViolationIdentity(v).IdentityHash]
		}
		if !counts {
			continue
		}
		if lvl == PolicyWarn || !blocksVerdict(v) {
			warned++
			continue
		}
		blocking = append(blocking, v)
	}

	s.NewViolations = blocking
	switch {
	case len(blocking) > 0:
		s.Status = "fail"
	case warned > 0:
		s.Status = "warning"
	default:
		s.Status = "pass"
	}
	return s
}
