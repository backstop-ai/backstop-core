package spine

// Negative fixture for Family B7 (no-rule-id-keyed-routing, ISSUE-064 REQ-004): finding
// routing/selection keyed on a baked NAME literal — a hardcoded namespaced rule-id
// compared against a finding's Rule field, and an engine-KEY literal used to select a
// dispatch — instead of the DECLARED role property. Both forms MUST be flagged. The
// correct source is the finding's declared role (see role-property-routing.go).

type violation struct {
	Rule       string
	Properties map[string]string
}

type manifest struct {
	Engines map[string]string
}

// routeByRuleID partitions findings by matching a hardcoded NAMESPACED rule-id literal —
// the baked-routing-identity defect: a consuming pack must name its rule EXACTLY this to
// route (why a TS pack was forced to name its rule `hollow-test-go`).
func routeByRuleID(vs []violation) (hollow []violation) {
	for _, v := range vs {
		if v.Rule == "backstop/substantiveness/hollow-test-go" {
			hollow = append(hollow, v)
		}
	}
	return hollow
}

// selectContractsEngine selects a dispatch on a baked engine-KEY literal (the ISSUE-065
// contracts-signature site this rule flags but is not activated against yet).
func selectContractsEngine(m manifest) string {
	if _, ok := m.Engines["ast-grep-contracts"]; ok {
		return "ast-grep-contracts"
	}
	return "ast-grep"
}
