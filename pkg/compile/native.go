package compile

import (
	"encoding/json"
	"os"
)

// EmitNativeCheck converts a Rule to a NativeCheck for native enforcement.
func EmitNativeCheck(rule Rule) NativeCheck {
	nc := NativeCheck{
		ID:       rule.ID,
		Message:  rule.Description,
		Severity: rule.Severity,
	}

	if len(rule.Languages) > 0 {
		nc.Language = rule.Languages[0]
	}

	if metric, ok := rule.Detection["metric"].(string); ok {
		nc.Metric = metric
	}
	if operator, ok := rule.Detection["operator"].(string); ok {
		nc.Operator = operator
	}
	nc.Threshold = rule.Detection["threshold"]

	if excludeRaw, ok := rule.Detection["exclude"].([]interface{}); ok {
		exclude := make([]string, 0, len(excludeRaw))
		for _, e := range excludeRaw {
			if s, ok := e.(string); ok {
				exclude = append(exclude, s)
			}
		}
		if len(exclude) > 0 {
			nc.Exclude = exclude
		}
	}

	return nc
}

// WriteNativeChecksFile serializes native checks to JSON.
func WriteNativeChecksFile(checks []NativeCheck, path string) error {
	if len(checks) == 0 {
		return nil
	}

	wrapper := struct {
		Checks []NativeCheck `json:"checks"`
	}{Checks: checks}

	data, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
