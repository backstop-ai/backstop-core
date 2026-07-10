package waiver

import "strings"

// DeclaredPolicy is a concrete, DATA-DRIVEN Policy for the declared non-waivable
// tier (REQ-006). It reports non-waivable for exactly the rule-ids and
// severities it was CONSTRUCTED with — core holds NO hardcoded list of protected
// rules (CLM-027). The shipped configuration (built at the cmd/backstop
// construction site by EXTRACTING the declared non-waivable sets from the
// installed pack manifests — REQ-016) supplies the backstop/self rule-ids and
// the critical severity; this type is only the mechanism, never the list.
type DeclaredPolicy struct {
	rules      map[string]bool
	severities map[string]bool
}

// NewDeclaredPolicy constructs a DeclaredPolicy from a supplied non-waivable
// rule-id set and a supplied non-waivable severity set. An empty/nil pair yields
// an all-waivable Policy — proving there is no core-hardcoded protected list
// (CLM-027). Severities are matched case-insensitively.
func NewDeclaredPolicy(nonWaivableRuleIDs []string, nonWaivableSeverities []string) *DeclaredPolicy {
	p := &DeclaredPolicy{
		rules:      make(map[string]bool, len(nonWaivableRuleIDs)),
		severities: make(map[string]bool, len(nonWaivableSeverities)),
	}
	for _, r := range nonWaivableRuleIDs {
		if r = strings.TrimSpace(r); r != "" {
			p.rules[r] = true
		}
	}
	for _, s := range nonWaivableSeverities {
		if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
			p.severities[s] = true
		}
	}
	return p
}

// Waivable reports whether a rule at a given severity may be waived. It is false
// when the rule-id is in the declared non-waivable set OR the severity is in the
// declared non-waivable severity set; otherwise true. A nil receiver treats
// everything as waivable.
func (p *DeclaredPolicy) Waivable(ruleID string, severity string) bool {
	if p == nil {
		return true
	}
	if p.rules[ruleID] {
		return false
	}
	if p.severities[strings.ToLower(strings.TrimSpace(severity))] {
		return false
	}
	return true
}
