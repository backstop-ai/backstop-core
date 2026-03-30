package compile

import (
	"os"

	"gopkg.in/yaml.v3"
)

// EmitSemgrepRule converts a Rule to a SemgrepRule for semgrep output.
func EmitSemgrepRule(rule Rule, languages []string) SemgrepRule {
	sr := SemgrepRule{
		ID:        rule.ID,
		Message:   rule.Description,
		Severity:  MapSeverity(rule.Severity),
		Languages: languages,
	}

	switch rule.Strategy() {
	case "pattern":
		if semgrep, ok := rule.Detection["semgrep"].(string); ok {
			sr.Pattern = semgrep
		}
	case "regex":
		if pattern, ok := rule.Detection["pattern"].(string); ok {
			sr.PatternRegex = pattern
		}
	}

	return sr
}

// WriteSemgrepFile writes semgrep rules to a YAML file.
func WriteSemgrepFile(rules []SemgrepRule, path string) error {
	if len(rules) == 0 {
		return nil
	}

	outputRules := make([]map[string]interface{}, 0, len(rules))
	for _, rule := range rules {
		ruleMap := map[string]interface{}{
			"id":        rule.ID,
			"message":   rule.Message,
			"severity":  rule.Severity,
			"languages": rule.Languages,
		}

		if rule.Pattern != "" {
			ruleMap["pattern"] = rule.Pattern
		} else if rule.PatternRegex != "" {
			ruleMap["pattern-regex"] = rule.PatternRegex
		}

		outputRules = append(outputRules, ruleMap)
	}

	doc := map[string]interface{}{
		"rules": outputRules,
	}

	data, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}
