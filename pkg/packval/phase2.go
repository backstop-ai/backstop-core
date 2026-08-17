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
		Checks: 9, // claims, fixtures, unique-ids, tool-config-traceability, pairs-with, orphans, exempt-scope-decision, path-scope-dispatch, path-scope-fixture-mask
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
			// Mechanism engine-model rules (go-toolchain's build/test/coverage) carry an
			// engine and no claims. Exempt a claimless rule from the no-claims error ONLY
			// when its declared engine RESOLVES to a real binding. An unknown engine fails
			// LOUD naming it, and an EMPTY engine is never exempted — the exemption is not
			// a free escape hatch a bogus engine can dodge (B-GUARD / CLM-013). This
			// resolve gate lives in phase2, which CHECK mode runs, so it is not deferred to
			// phase3.
			if rule.Engine != "" {
				if _, err := resolveEngine(pack, rule.Engine); err != nil {
					res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "engine-resolve", Rule: rule.ID, Message: err.Error()})
				}
			} else {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "claims", Rule: rule.ID, Message: "rule has no claims"})
			}
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

	// A project-wide engine that omits exempt_from_scope_filter has never had its
	// scope-filter decision recorded — the absent key is indistinguishable from an
	// oversight. Prompt the author to make the call; never make it for them. This
	// reads scope_kind and key presence only, and rewrites no binding value.
	for _, name := range ExemptDecisionPending(pack) {
		res.Warnings = append(res.Warnings, ValidationWarning{
			Phase: res.Phase,
			Check: "exempt-scope-decision",
			Message: "engine " + name + " is scope_kind: project-wide but does not declare exempt_from_scope_filter, " +
				"so its scope-filter behaviour is an unrecorded default: true means its violations still RED a " +
				"diff-scoped gate when they land on a file the change never touched, false means they are filtered out",
			FixHint: "add `exempt_from_scope_filter: true` or `exempt_from_scope_filter: false` to the " + name + " engine block, recording the decision either way",
			Files:   []string{"pack.yml"},
		})
	}

	// A slash-bearing paths.include/paths.exclude pattern is unsatisfied under the
	// gate's explicit-file dispatch in every spelling, so the rule it scopes is dark:
	// an include silently scans nothing, an exclude fails open. This lives in phase 2
	// rather than phase 3 because EXECUTION CANNOT SEE IT — a pack whose rule keeps
	// slash-free "fixture hooks" alongside its inert live scope passes its own
	// fixtures, the fixture being the one file the rule can still match. Phase 2 also
	// runs in CHECK mode, which skips fixture execution entirely. Advisory only: for
	// several real patterns there is no lossless slash-free rewrite, so the remedy is a
	// pack-authoring judgement call the validator does not get to force.
	for _, f := range pathScopeDispatchFindings(pack, packDir) {
		res.Warnings = append(res.Warnings, ValidationWarning{
			Phase:   res.Phase,
			Check:   pathScopeDispatchCheckName,
			Message: pathScopeDispatchMessage(f),
			FixHint: pathScopeFixHint,
			Files:   []string{f.RuleSource},
		})
	}
	// The SECOND, distinct advisory: a rule whose live scope is inert but whose declared
	// fixtures are still matched by its own slash-free patterns. Those patterns are
	// fixture HOOKS — they keep phase 3 green while every live-scope pattern is dark, so
	// this is the reason a broken pack looks healthy. Reported separately from
	// path-scope-dispatch because it answers a different question: not "which pattern is
	// dark" but "why did nothing tell you".
	for _, m := range pathScopeMaskFindings(pack, packDir) {
		res.Warnings = append(res.Warnings, ValidationWarning{
			Phase:   res.Phase,
			Check:   pathScopeMaskCheckName,
			Message: pathScopeMaskMessage(m),
			FixHint: pathScopeFixHint,
			Files:   []string{m.RuleSource},
		})
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
