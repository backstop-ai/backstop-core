package packval

func RunRiskClass(pack *PackManifest) *PhaseResult {
	res := &PhaseResult{
		Phase:  "phase6-risk-class",
		Status: "pass",
		Checks: []string{"bypass-attempt", "fixture-independence"},
	}
	if pack == nil {
		res.Status = "fail"
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "manifest", Message: "manifest is nil"})
		return res
	}

	for _, rule := range pack.Content.Ruleset.Rules {
		if rule.RiskClass != "security" {
			continue
		}
		seen := map[string]bool{}
		for _, claim := range rule.Claims {
			hasBypass := false
			for _, f := range append(claim.Fixtures.Positive, claim.Fixtures.Negative...) {
				if f.BypassAttempt {
					hasBypass = true
				}
				if seen[f.Path] {
					res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "independent-fixtures", Rule: rule.ID, Claim: claim.ID, Message: "security claims must not share fixtures"})
				}
				seen[f.Path] = true
			}
			if !hasBypass {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "bypass-attempt", Rule: rule.ID, Claim: claim.ID, Message: "security claim missing bypass_attempt fixture"})
			}
		}
	}

	if len(res.Errors) > 0 {
		res.Status = "fail"
	}
	return res
}
