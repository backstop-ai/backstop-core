package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bmanson/backstop-core/pkg/validate"
)

// ArtifactNewFormatter formats artifact new results for output.
type ArtifactNewFormatter interface {
	FormatNewResult(result ArtifactNewResult) (string, error)
}

// JSONArtifactNewFormatter formats artifact new results as JSON.
type JSONArtifactNewFormatter struct{}

// FormatNewResult serializes the result to indented JSON.
func (f *JSONArtifactNewFormatter) FormatNewResult(result ArtifactNewResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("formatting JSON output: %w", err)
	}
	return string(data) + "\n", nil
}

// HumanArtifactNewFormatter formats artifact new results as human-readable text.
type HumanArtifactNewFormatter struct{}

// FormatNewResult formats the result for human consumption.
func (f *HumanArtifactNewFormatter) FormatNewResult(result ArtifactNewResult) (string, error) {
	return fmt.Sprintf("Created %s (ID: %s)\n", result.FilePath, result.ID), nil
}

// Formatter is the output formatting contract for JSON and human modes.
type Formatter interface {
	FormatResult(result validate.ValidationResult) (string, error)
}

// JSONFormatter implements Formatter for structured JSON output.
// It includes a schema_version field in every response for contract evolution.
type JSONFormatter struct{}

// jsonEnvelope is the wire format for JSON output.
type jsonEnvelope struct {
	SchemaVersion   string          `json:"schema_version"`
	Pass            bool            `json:"pass"`
	ViolationsCount int             `json:"violations_count"`
	Violations      []jsonViolation `json:"violations"`
}

// jsonViolation is the JSON representation of a single violation.
type jsonViolation struct {
	Rule     string `json:"rule"`
	File     string `json:"file,omitempty"`
	Message  string `json:"message"`
	Severity string `json:"severity,omitempty"`
}

// FormatResult serializes the result to indented JSON with a schema_version field.
func (f *JSONFormatter) FormatResult(result validate.ValidationResult) (string, error) {
	env := jsonEnvelope{
		SchemaVersion:   "cli/v1",
		Pass:            result.Pass(),
		ViolationsCount: len(result.Violations),
		Violations:      make([]jsonViolation, 0, len(result.Violations)),
	}

	for _, v := range result.Violations {
		env.Violations = append(env.Violations, jsonViolation{
			Rule:     v.Rule,
			File:     v.File,
			Message:  v.Message,
			Severity: v.Severity,
		})
	}

	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return "", fmt.Errorf("formatting JSON output: %w", err)
	}
	return string(data), nil
}

// HumanFormatter implements Formatter for human-readable terminal output.
// It respects the NO_COLOR environment variable.
type HumanFormatter struct{}

// FormatResult formats the result as human-readable text, grouping
// violations by file for readability (REQ-009).
func (f *HumanFormatter) FormatResult(result validate.ValidationResult) (string, error) {
	useColor := os.Getenv("NO_COLOR") == ""

	var sb strings.Builder

	// Group violations by file
	fileOrder := make([]string, 0)
	byFile := make(map[string][]validate.Violation)
	for _, v := range result.Violations {
		file := v.File
		if file == "" {
			file = "(no file)"
		}
		if _, seen := byFile[file]; !seen {
			fileOrder = append(fileOrder, file)
		}
		byFile[file] = append(byFile[file], v)
	}

	// Format violations grouped by file
	for _, file := range fileOrder {
		if useColor {
			sb.WriteString(fmt.Sprintf("\033[1m%s\033[0m\n", file))
		} else {
			sb.WriteString(fmt.Sprintf("%s\n", file))
		}
		for _, v := range byFile[file] {
			if useColor && v.Severity == "error" {
				sb.WriteString(fmt.Sprintf("\033[31m  ✗ [%s] %s\033[0m\n", v.Rule, v.Message))
			} else {
				sb.WriteString(fmt.Sprintf("  ✗ [%s] %s\n", v.Rule, v.Message))
			}
		}
	}

	// Format pass/fail status
	if result.Pass() {
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

	return sb.String(), nil
}
