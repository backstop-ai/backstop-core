package packval

func RunArchetype(pack *PackManifest) *PhaseResult {
	res := &PhaseResult{
		Phase:  "phase4-archetype",
		Status: "pass",
		Checks: 2, // archetype-rules, co-occurrence
	}
	if pack == nil {
		res.Status = "fail"
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "manifest", Message: "manifest is nil"})
		return res
	}
	if pack.Archetype == "code" && len(pack.Content.Ruleset.Rules) == 0 {
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "code-rules", Message: "code pack requires rules"})
	}
	if pack.Archetype == "code" {
		ruleMap := map[string]bool{}
		for _, r := range pack.Content.Ruleset.Rules {
			ruleMap[r.ID] = true
			if len(r.PairsWith.Scaffolds) == 0 && r.PairsWith.SDK == "" {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "pairs_with", Rule: r.ID, Message: "rule missing pairs_with"})
			}
		}
		for _, s := range pack.Content.Scaffolds {
			if len(s.PairsWith.Rules) == 0 {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "pairs_with", Rule: s.ID, Message: "scaffold missing pairs_with rules"})
				continue
			}
			for _, rid := range s.PairsWith.Rules {
				if !ruleMap[rid] {
					res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "pairs_with", Rule: s.ID, Message: "scaffold pairs_with unresolved rule"})
				}
			}
		}
	}
	if pack.Archetype == "enforcement" {
		if len(pack.Content.Scaffolds) > 0 {
			res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "enforcement-content", Message: "enforcement pack must not include scaffolds"})
		}
		if pack.Content.SDK != nil {
			res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "enforcement-content", Message: "enforcement pack must not include sdk"})
		}
	}
	if len(res.Errors) > 0 {
		res.Status = "fail"
	}
	return res
}
