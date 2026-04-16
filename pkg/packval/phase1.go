package packval

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.\-]+)?$`)

func RunStructural(pack *PackManifest, packDir string) *PhaseResult {
	res := &PhaseResult{
		Phase:  "phase1-structural",
		Status: "pass",
		Checks: 5, // yaml-parse, required-fields, valid-enums, file-existence, risk-class
	}
	if pack == nil {
		res.Status = "fail"
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "manifest", Message: "manifest is nil"})
		return res
	}

	if pack.Name == "" {
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "required", Message: "name is required", ManifestPath: "name"})
	}
	if pack.Version == "" {
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "required", Message: "version is required", ManifestPath: "version"})
	} else if !semverRe.MatchString(pack.Version) {
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "version", Message: "version must be semver", ManifestPath: "version"})
	}
	if pack.Language == "" {
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "required", Message: "language is required", ManifestPath: "language"})
	} else if pack.Language != "go" {
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "language", Message: "unsupported language", ManifestPath: "language"})
	}
	if pack.Archetype == "" {
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "required", Message: "archetype is required", ManifestPath: "archetype"})
	} else if pack.Archetype != "code" && pack.Archetype != "enforcement" {
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "archetype", Message: "invalid archetype", ManifestPath: "archetype"})
	}
	if len(pack.Content.Ruleset.Rules) == 0 && len(pack.Content.Scaffolds) == 0 && pack.Content.SDK == nil {
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "content", Message: "content is required", ManifestPath: "content"})
	}

	validRisk := map[string]bool{"security": true, "correctness": true, "style": true, "perf": true}
	for i, rule := range pack.Content.Ruleset.Rules {
		if rule.File != "" {
			if _, err := os.Stat(filepath.Join(packDir, rule.File)); err != nil {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "file-exists", Rule: rule.ID, Message: "referenced file not found", ManifestPath: "content.ruleset.rules[" + strconv.Itoa(i) + "].file"})
			}
		}
		if rule.RiskClass == "" {
			res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "risk-class", Rule: rule.ID, Message: "missing risk_class"})
		} else if !validRisk[rule.RiskClass] {
			res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "risk-class", Rule: rule.ID, Message: "invalid risk_class"})
		}
		for _, claim := range rule.Claims {
			for _, f := range append(claim.Fixtures.Positive, claim.Fixtures.Negative...) {
				if _, err := os.Stat(filepath.Join(packDir, f.Path)); err != nil {
					res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "file-exists", Rule: rule.ID, Claim: claim.ID, Message: "fixture file not found"})
				}
			}
		}
		if rule.Validator != "" {
			if _, err := os.Stat(filepath.Join(packDir, rule.Validator)); err != nil {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "file-exists", Rule: rule.ID, Message: "validator file not found"})
			}
		}
	}
	for _, s := range pack.Content.Scaffolds {
		if s.Path != "" {
			if _, err := os.Stat(filepath.Join(packDir, s.Path)); err != nil {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "file-exists", Rule: s.ID, Message: "scaffold path not found"})
			}
		}
	}
	for _, tc := range pack.ToolConfig {
		if tc.ID != "" {
			if tc.RiskClass == "" {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "risk-class", Rule: tc.ID, Message: "missing risk_class"})
			} else if !validRisk[tc.RiskClass] {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "risk-class", Rule: tc.ID, Message: "invalid risk_class"})
			}
		}
		// NOTE: tool_config.file intentionally excluded.
	}

	if len(res.Errors) > 0 {
		res.Status = "fail"
	}
	return res
}
