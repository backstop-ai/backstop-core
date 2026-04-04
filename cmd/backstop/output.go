package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Formatter is the output formatting contract for JSON and human modes.
type Formatter interface {
	FormatResult(result interface{}) (string, error)
}

// JSONFormatter implements Formatter for structured JSON output.
// It includes a schema_version field in every response for contract evolution.
type JSONFormatter struct{}

// FormatResult serializes the result to indented JSON with a schema_version field.
func (f *JSONFormatter) FormatResult(result interface{}) (string, error) {
	// Wrap result in envelope with schema_version
	envelope := make(map[string]interface{})

	switch v := result.(type) {
	case map[string]interface{}:
		for k, val := range v {
			envelope[k] = val
		}
	default:
		envelope["data"] = result
	}

	envelope["schema_version"] = "cli/v1"

	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return "", fmt.Errorf("formatting JSON output: %w", err)
	}
	return string(data), nil
}

// HumanFormatter implements Formatter for human-readable terminal output.
// It respects the NO_COLOR environment variable.
type HumanFormatter struct{}

// FormatResult formats the result as human-readable text.
func (f *HumanFormatter) FormatResult(result interface{}) (string, error) {
	useColor := os.Getenv("NO_COLOR") == ""

	var sb strings.Builder

	switch v := result.(type) {
	case map[string]interface{}:
		f.formatMap(&sb, v, useColor)
	default:
		sb.WriteString(fmt.Sprintf("%v\n", result))
	}

	return sb.String(), nil
}

// formatMap formats a map as human-readable text.
func (f *HumanFormatter) formatMap(sb *strings.Builder, m map[string]interface{}, useColor bool) {
	// Check for violations
	if violations, ok := m["violations"]; ok {
		f.formatViolations(sb, violations, useColor)
	}

	// Check for pass/fail status
	if pass, ok := m["pass"]; ok {
		if passBool, ok := pass.(bool); ok {
			if passBool {
				if useColor {
					sb.WriteString("\033[32m✓ All checks passed\033[0m\n")
				} else {
					sb.WriteString("✓ All checks passed\n")
				}
			} else {
				if useColor {
					sb.WriteString("\033[31m✗ Checks failed\033[0m\n")
				} else {
					sb.WriteString("✗ Checks failed\n")
				}
			}
		}
	}

	// Format other fields
	for key, val := range m {
		if key == "violations" || key == "pass" {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s: %v\n", key, val))
	}
}

// formatViolations formats a list of violations.
func (f *HumanFormatter) formatViolations(sb *strings.Builder, violations interface{}, useColor bool) {
	switch v := violations.(type) {
	case []map[string]string:
		for _, viol := range v {
			rule := viol["rule"]
			msg := viol["message"]
			severity := viol["severity"]
			if useColor && severity == "error" {
				sb.WriteString(fmt.Sprintf("\033[31m  ✗ [%s] %s\033[0m\n", rule, msg))
			} else {
				sb.WriteString(fmt.Sprintf("  ✗ [%s] %s\n", rule, msg))
			}
		}
	case []interface{}:
		for _, item := range v {
			if viol, ok := item.(map[string]interface{}); ok {
				rule := fmt.Sprintf("%v", viol["rule"])
				msg := fmt.Sprintf("%v", viol["message"])
				severity := fmt.Sprintf("%v", viol["severity"])
				if useColor && severity == "error" {
					sb.WriteString(fmt.Sprintf("\033[31m  ✗ [%s] %s\033[0m\n", rule, msg))
				} else {
					sb.WriteString(fmt.Sprintf("  ✗ [%s] %s\n", rule, msg))
				}
			} else {
				sb.WriteString(fmt.Sprintf("  ✗ %v\n", item))
			}
		}
	case []string:
		for _, s := range v {
			sb.WriteString(fmt.Sprintf("  ✗ %s\n", s))
		}
	}
}
