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
		Checks: []string{"layer", "category", "input-scope", "validator"},
	}
	if pack == nil {
		res.Status = "fail"
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "manifest", Message: "manifest is nil"})
		return res
	}
	validCategory := map[string]bool{"presence": true, "structural": true, "other": true}
	for _, rule := range pack.Content.Ruleset.Rules {
		if rule.Layer < 1 || rule.Layer > 3 {
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
