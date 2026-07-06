package check

// Violation represents a single validation finding. It is the shared finding
// carrier produced by the LIVE SARIF parser (ParsePackFindings) and consumed by
// the gate's dispatchPackEngines path. The in-process check ENGINE that once
// produced it (the `backstop code check` command's Run/Engine machinery) was
// deleted by ISSUE-018; only the shared types and the SARIF/coverage surface
// survive.
type Violation struct {
	Pass     CheckType
	File     string
	Line     int
	Message  string
	Severity string
	// Rule carries a structured rule identifier for the finding. For semgrep
	// findings this is the check_id, preserved verbatim including
	// pack-namespaced IDs (pack.NamespacedRuleID format, e.g.
	// "org/pack/rule-id") so violations are attributable to their source pack.
	// Empty for passes that have no per-rule identity (lint/build/test).
	Rule string
	// Fingerprint is a content-based, line-INDEPENDENT identity carried from the
	// SARIF result (partialFingerprints or region snippet). It flows to
	// gate.Violation.RegionHash so the baseline keeps multiple same-rule findings
	// in one file distinct and survives unrelated line shifts. Empty when the
	// engine emits neither, leaving the coarse message-level fallback.
	Fingerprint string
}

// PassResult holds the result of a single validation pass. It is retained as the
// carrier the output formatter (output.go) renders.
type PassResult struct {
	Pass       CheckType
	Violations []Violation
	Skipped    bool
	SkipReason string
}

// Result holds aggregated results from all validation passes. It is retained as
// the value FormatResult/DetermineExitCode (output.go) operate over.
type Result struct {
	PassResults []PassResult
	Warnings    []string
	ExitCode    int
}

// HasViolations returns true if any pass produced violations.
func (r *Result) HasViolations() bool {
	for _, pr := range r.PassResults {
		if len(pr.Violations) > 0 {
			return true
		}
	}
	return false
}

// ViolationCount returns the total number of violations across all passes.
func (r *Result) ViolationCount() int {
	count := 0
	for _, pr := range r.PassResults {
		count += len(pr.Violations)
	}
	return count
}

// AllViolations returns all violations flattened from all passes.
func (r *Result) AllViolations() []Violation {
	var all []Violation
	for _, pr := range r.PassResults {
		all = append(all, pr.Violations...)
	}
	return all
}
