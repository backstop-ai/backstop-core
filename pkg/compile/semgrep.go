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

	if exceptions, ok := rule.Detection["exceptions"].([]interface{}); ok {
		for _, e := range exceptions {
			if s, ok := e.(string); ok {
				sr.PatternNotRegex = append(sr.PatternNotRegex, s)
			}
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

		if len(rule.PatternNotRegex) > 0 {
			// Use composite patterns with exclusions
			patterns := []map[string]interface{}{}
			if rule.Pattern != "" {
				patterns = append(patterns, map[string]interface{}{"pattern": rule.Pattern})
			} else if rule.PatternRegex != "" {
				patterns = append(patterns, map[string]interface{}{"pattern-regex": rule.PatternRegex})
			}
			for _, exc := range rule.PatternNotRegex {
				patterns = append(patterns, map[string]interface{}{"pattern-not-regex": exc})
			}
			ruleMap["patterns"] = patterns
		} else if rule.Pattern != "" {
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
