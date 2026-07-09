package packval

import (
	"os"
	"path/filepath"
	"strings"
)

func RunLayer(pack *PackManifest, packDir string) *PhaseResult {
	res := &PhaseResult{
		Phase:  "phase5-layer",
		Status: "pass",
		Checks: 4, // layer, category, input-scope, validator
	}
	if pack == nil {
		res.Status = "fail"
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "manifest", Message: "manifest is nil"})
		return res
	}
	validCategory := map[string]bool{"presence": true, "structural": true, "other": true}
	for _, rule := range pack.Content.Ruleset.Rules {
		if rule.Layer < 1 || rule.Layer > 3 {
			// Engine-model rules carry an engine, not a layer (go-standards rules
			// declare `engine: semgrep` and no layer). Exempt such a rule from the
			// layer 1..3 requirement ONLY when its declared engine RESOLVES to a real
			// binding (base registry merged with the pack's engines: block). An unknown
			// engine fails LOUD naming it, and an EMPTY engine is never exempted — the
			// exemption is not a free escape hatch a bogus engine can dodge (B-GUARD /
			// CLM-013). This resolve gate lives in phase5, which CHECK mode runs, so it
			// is not deferred to phase3.
			if rule.Engine != "" {
				if _, err := resolveEngine(pack, rule.Engine); err != nil {
					res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "engine-resolve", Rule: rule.ID, Message: err.Error()})
				}
				continue
			}
			res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "layer", Rule: rule.ID, Message: "invalid layer"})
			continue
		}
		if rule.Layer == 1 || rule.Layer == 2 {
			if rule.Category != "" {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "category", Rule: rule.ID, Message: "category forbidden for layer 1/2"})
			}
		}
		if rule.Layer == 3 {
			if !validCategory[rule.Category] {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "category", Rule: rule.ID, Message: "invalid category"})
			}
			if rule.Category == "other" && strings.TrimSpace(rule.Justification) == "" {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "justification", Rule: rule.ID, Message: "other requires non-empty justification"})
			}
			if rule.InputScope != "single-file" && rule.InputScope != "multi-file" {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "input_scope", Rule: rule.ID, Message: "invalid input_scope"})
			}
			if strings.TrimSpace(rule.Validator) == "" {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "validator", Rule: rule.ID, Message: "missing validator"})
			} else if _, err := os.Stat(filepath.Join(packDir, rule.Validator)); err != nil {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "validator", Rule: rule.ID, Message: "validator file not found"})
			}
		}
	}
	if len(res.Errors) > 0 {
		res.Status = "fail"
	}
	return res
}
