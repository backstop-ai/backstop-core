package main

import (
	"strings"
	"testing"
)

// substantiveness_routing_test.go pins the ISSUE-064 absence contract (REQ-002 / CLM-003):
// the language-suffixed rule-NAME routing constants are gone from cmd/backstop and the
// substantiveness call site no longer feeds namespaced rule ids into the routing function.
// Routing is now by the pack-declared substantiveness_role property (pkg/gate), not a baked
// rule-name literal.

// stripGoLineComments removes // line-comment text so the absence assertions match actual
// CODE, not a symbol name that might appear in a deletion-marker comment.
func stripGoLineComments(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

// TestSubstantivenessRuleNameConstantsRemoved (CLM-003, kind: absence) — the
// substantivenessHollowRuleName / substantivenessExtractionRuleName constants no longer
// exist in cmd/backstop, and the substantiveness call site no longer passes namespaced
// rule ids into gate.RouteSubstantivenessFindings.
func TestSubstantivenessRuleNameConstantsRemoved(t *testing.T) {
	code := stripGoLineComments(readFileStr(t, "gate.go"))

	for _, sym := range []string{
		"substantivenessHollowRuleName",
		"substantivenessExtractionRuleName",
	} {
		if strings.Contains(code, sym) {
			t.Errorf("rule-name routing constant %q must be REMOVED from cmd/backstop (CLM-003); still present", sym)
		}
	}

	// The call site must route by the declared role — a bare, single-argument route — and
	// must not thread a namespaced rule id into the routing function.
	if !strings.Contains(code, "gate.RouteSubstantivenessFindings(flat)") {
		t.Error("the substantiveness call site must route the flat stream by declared role (RouteSubstantivenessFindings(flat)); the by-role single-arg call was not found")
	}
	if strings.Contains(code, "RouteSubstantivenessFindings(\n") ||
		strings.Contains(code, "RouteSubstantivenessFindings(flat,") {
		t.Error("the call site must NOT pass namespaced rule ids into RouteSubstantivenessFindings (CLM-003)")
	}
}
