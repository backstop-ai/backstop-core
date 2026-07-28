package gate

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/backstop-ai/backstop-core/pkg/waiver"
)

// StepPackEngines is the canonical step name of the LIVE pack code-rule
// dimension. Pack code-rule findings (including the flagship non-waivable
// backstop/self rules) flow through this dimension — NOT the removed code_check
// dimension. It is one of the two WAIVABLE (code-located) surfaces (REQ-010).
const StepPackEngines = "pack_engines"

// coverageAnnotationLine is the FILE's first line — the fixed location the
// locationless coverage_threshold dimension's @waiver:coverage_threshold
// annotation convention lives on (REQ-010).
const coverageAnnotationLine = 1

// WithWaiver enables the SPEC-049 waiver reconciliation pass and attaches its
// runtime inputs (REQ-016), mirroring WithBaseline: it sets g.waiverEnabled = true
// so the Run-loop swap fires (guarded on the flag), and attaches the LineReader
// (raw source bytes), the declared Policy (non-waivable tier), and now. Without
// this Option AND the cmd/backstop construction-site call, the shipped gate stays
// dark and suppresses nothing.
func WithWaiver(read waiver.LineReader, policy waiver.Policy, now time.Time) Option {
	return func(g *Gate) {
		g.waiverEnabled = true
		g.waiverRead = read
		g.waiverPolicy = policy
		g.waiverNow = now
	}
}

// StepWaiverResolutionFunc returns the registered placeholder waiver step. When
// waivers are DISABLED it reports skipped; when enabled the Run loop replaces its
// output with computeWaiverResult (mirroring the baseline swap).
func StepWaiverResolutionFunc() StepFunc {
	return StepWaiverResolutionScopedFunc(nil)
}

// StepWaiverResolutionScopedFunc is the scope-aware placeholder waiver step,
// mirroring StepBaselineComparisonScopedFunc. It emits the skipped/pending
// StepResult when waivers are disabled; when g.waiverEnabled the Run loop swaps
// its output for the real computeWaiverResult reconciliation pass.
func StepWaiverResolutionScopedFunc(_ *GateScope) StepFunc {
	return func(_ context.Context) StepResult {
		return StepResult{
			StepName:   StepWaiverResolution,
			Status:     "skipped",
			Violations: []Violation{},
			Reason:     "waivers not enabled",
		}
	}
}

// waivableDimension reports whether a gate dimension is code-located and thus
// suppressible by an inline @waiver token (REQ-010). The waivable surface is
// EXACTLY pack_engines + test_substantiveness. coverage_threshold is handled
// separately via the first-line annotation convention; every other dimension is
// structural and NOT waivable.
func waivableDimension(step string) bool {
	return step == StepPackEngines || step == StepTestSubstantiveness
}

// computeWaiverResult is the Run-loop waiver reconciliation pass, MODELED ON
// computeBaselineResult (gate.go). It sees ALL accumulated StepResults, collects
// ONLY the waivable-surface findings, calls the pure pkg/waiver.Adjudicate,
// REMOVES the suppressed findings from the accumulated results' Violation slices
// (the actual subtraction — CLM-067), adds malformed / non-waivable diagnostics
// as gate findings, persists the active waivers onto GateResult.ActiveWaivers via
// g.activeWaivers, and returns the distinct PASS·N-waivers waiver_resolution
// StepResult. It MUTATES accumulated in place.
func (g *Gate) computeWaiverResult(accumulated []StepResult, read waiver.LineReader, policy waiver.Policy, now time.Time) StepResult {
	// Collect findings from the waivable surface only (REQ-010). Structural
	// dimensions (artifact_status_drift, contract_signature, test_verification,
	// artifact_validation) are excluded; coverage_threshold uses the first-line
	// annotation convention (locationless -> line 1).
	var findings []waiver.Finding
	for i := range accumulated {
		s := accumulated[i]
		switch {
		case waivableDimension(s.StepName):
			for _, v := range s.Violations {
				findings = append(findings, waiver.Finding{RuleID: v.Rule, File: v.File, Line: v.Line, Severity: v.Severity})
			}
		case s.StepName == StepCoverageThreshold:
			for _, v := range s.Violations {
				findings = append(findings, waiver.Finding{RuleID: v.Rule, File: v.File, Line: coverageAnnotationLine, Severity: v.Severity})
			}
		}
	}

	res := waiver.Adjudicate(findings, read, policy, now)

	sup := map[string]bool{}
	for _, f := range res.Suppressed {
		sup[findingKey(f.File, f.Line, f.RuleID)] = true
	}

	// The actual subtraction (CLM-067): remove suppressed findings from the
	// accumulated waivable-surface Violation slices, in place.
	for i := range accumulated {
		s := &accumulated[i]
		var keyLine func(v Violation) int
		switch {
		case waivableDimension(s.StepName):
			keyLine = func(v Violation) int { return v.Line }
		case s.StepName == StepCoverageThreshold:
			keyLine = func(_ Violation) int { return coverageAnnotationLine }
		default:
			continue
		}
		kept := make([]Violation, 0, len(s.Violations))
		for _, v := range s.Violations {
			if sup[findingKey(v.File, keyLine(v), v.Rule)] {
				continue
			}
			// REQ-014: hand the author a pre-filled @waiver token for every finding
			// that REMAINS blocked on the waivable surface, so acknowledging is at
			// most one paste. Presentation only — WaiverHint is a non-identity field,
			// so the embedded expiry date never perturbs baseline grandfathering.
			if tok, ok := PrefilledWaiverToken(s.StepName, v, policy, now); ok {
				v.WaiverHint = tok
			}
			kept = append(kept, v)
		}
		s.Violations = kept
		if len(kept) == 0 && s.Status == "fail" {
			s.Status = "pass"
		}
	}

	g.activeWaivers = res.Active

	// Malformed and non-waivable tokens are themselves gate findings (REQ-007 /
	// REQ-006): a @waiver on a declared non-waivable rule is a gate ERROR.
	wv := make([]Violation, 0, len(res.Malformed)+len(res.NonWaivable))
	for _, d := range res.Malformed {
		wv = append(wv, waiverDiagToViolation(d))
	}
	for _, d := range res.NonWaivable {
		wv = append(wv, waiverDiagToViolation(d))
	}

	status := "pass"
	if len(wv) > 0 {
		status = "fail"
	}
	return StepResult{
		StepName:   StepWaiverResolution,
		Status:     status,
		Violations: wv,
		Reason:     waiverReason(res),
	}
}

// findingKey uniquely identifies a code-located finding for suppression matching.
func findingKey(file string, line int, rule string) string {
	return file + "\x00" + strconv.Itoa(line) + "\x00" + rule
}

// waiverDiagToViolation converts a waiver Diagnostic (malformed / non-waivable)
// into a gate Violation so it surfaces as a gate finding.
func waiverDiagToViolation(d waiver.Diagnostic) Violation {
	return Violation{
		Rule:     d.RuleID,
		File:     d.File,
		Line:     d.Line,
		Message:  d.Message,
		Severity: "error",
	}
}

// waiverReason renders the never-silent waiver report (REQ-012): the distinct
// PASS·N-waivers state with an always-on active summary, and the actionable
// subset (expiring-soon + unused) surfaced inline. It also names any malformed /
// non-waivable diagnostics.
func waiverReason(res waiver.Result) string {
	var b strings.Builder
	if len(res.Active) > 0 {
		fmt.Fprintf(&b, "PASS · %d waivers (%s)", len(res.Active), joinWaiverRules(res.Active))
	} else {
		b.WriteString("clean — no active waivers")
	}
	if len(res.Expiring) > 0 {
		fmt.Fprintf(&b, "; %d expiring soon (%s)", len(res.Expiring), joinWaiverRules(res.Expiring))
	}
	if len(res.Unused) > 0 {
		fmt.Fprintf(&b, "; %d unused/dangling (%s)", len(res.Unused), joinWaiverRules(res.Unused))
	}
	if len(res.Malformed) > 0 {
		fmt.Fprintf(&b, "; %d malformed", len(res.Malformed))
	}
	if len(res.NonWaivable) > 0 {
		fmt.Fprintf(&b, "; %d non-waivable (gate error)", len(res.NonWaivable))
	}
	return b.String()
}

// joinWaiverRules joins the rule-ids of a waiver set for the summary line.
func joinWaiverRules(ws []waiver.Waiver) string {
	ids := make([]string, 0, len(ws))
	for _, w := range ws {
		ids = append(ids, w.RuleID)
	}
	return strings.Join(ids, ", ")
}

// ActiveWaiverFeed is the step-9 audit data-feed accessor (REQ-015). It reads the
// GateResult.ActiveWaivers carrier populated by computeWaiverResult so "what are
// we deliberately ignoring, why, and until when" is a first-class auditable
// question. Each entry carries at least rule-id, reason-code, and expiry.
func ActiveWaiverFeed(res GateResult) []waiver.Waiver {
	return res.ActiveWaivers
}

// PrefilledWaiverToken returns a pre-filled, neutral @waiver token for a BLOCKED
// waivable finding (REQ-014), so acknowledging is at most as much friction as an
// engine-native //nolint. It returns ok=false for a NON-WAIVABLE finding (it
// cannot be waived) and for a structural/non-code finding (outside the waivable
// surface). The token carries the finding's own rule-id and a reason-code
// defaulted expiry.
func PrefilledWaiverToken(dimension string, v Violation, policy waiver.Policy, now time.Time) (string, bool) {
	// coverage_threshold is waivable via the first-line annotation, so it is
	// eligible for a pre-filled token too; structural dimensions are not.
	if !waivableDimension(dimension) && dimension != StepCoverageThreshold {
		return "", false
	}
	if policy != nil && !policy.Waivable(v.Rule, v.Severity) {
		return "", false
	}
	reason := waiver.ReasonAcceptedRisk
	dur, ok := waiver.DefaultDuration(reason)
	if !ok {
		dur = 90 * 24 * time.Hour
	}
	expiry := now.Add(dur).Format("2006-01-02")
	return fmt.Sprintf("@waiver:%s:%s:%s", v.Rule, reason, expiry), true
}
