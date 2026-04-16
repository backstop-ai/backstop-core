package packval

import (
	"os"
	"path/filepath"
	"strings"
)

func RunCoherence(pack *PackManifest, packDir string) *PhaseResult {
	res := &PhaseResult{
		Phase:  "phase2-coherence",
		Status: "pass",
		Checks: []string{"claims", "fixtures", "unique-ids", "tool-config-traceability", "pairs-with", "orphans"},
	}
	if pack == nil {
		res.Status = "fail"
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "manifest", Message: "manifest is nil"})
		return res
	}

	claimIDs := map[string]bool{}
	ruleIDs := map[string]bool{}
	referencedFixtures := map[string]bool{}

	for _, rule := range pack.Content.Ruleset.Rules {
		if ruleIDs[rule.ID] {
			res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "unique-rule-id", Rule: rule.ID, Message: "duplicate rule id"})
		}
		ruleIDs[rule.ID] = true
		if len(rule.Claims) == 0 {
			res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "claims", Rule: rule.ID, Message: "rule has no claims"})
		}
		for _, claim := range rule.Claims {
			if claimIDs[claim.ID] {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "unique-claim-id", Claim: claim.ID, Message: "duplicate claim id"})
			}
			claimIDs[claim.ID] = true
			if len(claim.Fixtures.Positive) == 0 {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "fixtures-positive", Rule: rule.ID, Claim: claim.ID, Message: "no positive fixtures"})
			}
			if len(claim.Fixtures.Negative) == 0 {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "fixtures-negative", Rule: rule.ID, Claim: claim.ID, Message: "no negative fixtures"})
			}
			for _, f := range append(claim.Fixtures.Positive, claim.Fixtures.Negative...) {
				full := filepath.Join(packDir, f.Path)
				referencedFixtures[f.Path] = true
				data, err := os.ReadFile(full)
				if err != nil {
					res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "fixture-exists", Rule: rule.ID, Claim: claim.ID, Message: "fixture file not found"})
					continue
				}
				if strings.TrimSpace(string(data)) == "" {
					res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "fixture-empty", Rule: rule.ID, Claim: claim.ID, Message: "fixture file is empty"})
				}
			}
		}
		for _, pw := range append(rule.PairsWith.Rules, rule.PairsWith.Scaffolds...) {
			if pw == "" {
				continue
			}
			// resolved after all IDs known.
		}
	}

	for _, tc := range pack.ToolConfig {
		if tc.ID != "" {
			if ruleIDs[tc.ID] {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "unique-rule-id", Rule: tc.ID, Message: "duplicate rule id"})
			}
			ruleIDs[tc.ID] = true
			if len(tc.Claims) == 0 {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "tool-config-claims", Rule: tc.ID, Message: "tool_config id has no claims"})
			}
			for _, claim := range tc.Claims {
				if claimIDs[claim.ID] {
					res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "unique-claim-id", Claim: claim.ID, Message: "duplicate claim id"})
				}
				claimIDs[claim.ID] = true
				if len(claim.Fixtures.Positive) == 0 || len(claim.Fixtures.Negative) == 0 {
					res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "tool-config-fixtures", Rule: tc.ID, Claim: claim.ID, Message: "tool_config claim missing fixtures"})
				}
				for _, f := range append(claim.Fixtures.Positive, claim.Fixtures.Negative...) {
					referencedFixtures[f.Path] = true
				}
			}
		}
	}

	knownRules := map[string]bool{}
	for _, id := range AllRuleIDs(pack) {
		knownRules[id] = true
	}
	for _, rule := range pack.Content.Ruleset.Rules {
		for _, rid := range rule.PairsWith.Rules {
			if !knownRules[rid] {
				res.Warnings = append(res.Warnings, ValidationWarning{Phase: res.Phase, Check: "pairs_with", Message: "dangling pairs_with reference", Files: []string{rid}})
			}
		}
	}

	fixturesDir := filepath.Join(packDir, "fixtures")
	_ = filepath.Walk(fixturesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(packDir, path)
		if relErr != nil {
			return nil
		}
		if !referencedFixtures[filepath.ToSlash(rel)] {
			res.Warnings = append(res.Warnings, ValidationWarning{Phase: res.Phase, Check: "orphan-fixture", Message: "orphan fixture", Files: []string{filepath.ToSlash(rel)}})
		}
		return nil
	})

	if len(res.Errors) > 0 {
		res.Status = "fail"
	}
	return res
}
