package waiver

import "testing"

// TestWaiver_NonWaivable_PolicyDeclaredNotHardcoded proves the non-waivable set
// is data-driven: the declared Policy reports non-waivable for exactly the rules
// / severities it was CONSTRUCTED with, and core contains no hardcoded list — a
// Policy built with an EMPTY declared set reports every rule waivable (CLM-027).
func TestWaiver_NonWaivable_PolicyDeclaredNotHardcoded(t *testing.T) {
	// Empty declared set: everything is waivable — no core-hardcoded protected list.
	empty := NewDeclaredPolicy(nil, nil)
	for _, rule := range []string{"backstop/self/no-baked-language", "secrets/aws-key", "anything"} {
		if !empty.Waivable(rule, "critical") {
			t.Errorf("empty-declared Policy reported %q non-waivable; there must be NO hardcoded list", rule)
		}
	}

	// Constructed set: exactly the supplied rules/severities are non-waivable.
	p := NewDeclaredPolicy([]string{"backstop/self/no-baked-language"}, []string{"critical"})
	if p.Waivable("backstop/self/no-baked-language", "error") {
		t.Error("declared rule reported waivable; the constructed set must govern")
	}
	if p.Waivable("secrets/aws-key", "critical") {
		t.Error("declared critical severity reported waivable; the constructed severity set must govern")
	}
	if !p.Waivable("go-standards/line-length", "warning") {
		t.Error("an undeclared rule/severity must remain waivable")
	}
}
