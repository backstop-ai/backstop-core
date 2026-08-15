package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/schema"
	"github.com/backstop-ai/backstop-core/pkg/validate"
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
//
// OMITEMPTY IS NOT UNIFORM HERE AND THE EXCEPTION IS DELIBERATE. ArtifactsAsserted
// carries NO omitempty: encoding/json drops a zero int under omitempty, and ZERO is
// exactly the value that must stay legible — an omitted count on an empty pass is the
// illegibility REQ-002 names. The string and slice fields keep omitempty because their
// empty value means genuinely absent.
type jsonEnvelope struct {
	SchemaVersion     string               `json:"schema_version"`
	BinaryVersion     string               `json:"binary_version,omitempty"`
	SchemaCohort      string               `json:"schema_cohort,omitempty"`
	Pass              bool                 `json:"pass"`
	ViolationsCount   int                  `json:"violations_count"`
	ArtifactsAsserted int                  `json:"artifacts_asserted"`
	ScannedRoot       string               `json:"scanned_root,omitempty"`
	Artifacts         []jsonArtifactRecord `json:"artifacts,omitempty"`
	Violations        []jsonViolation      `json:"violations"`
}

// jsonArtifactRecord is one validated artifact's record on the wire.
//
// This type exists rather than reusing the gate result's flat SchemaIdentities list
// because REQ-003 requires an identity for EACH validated artifact, and a per-SCHEMA
// list cannot express that binding.
type jsonArtifactRecord struct {
	Path           string `json:"path"`
	Type           string `json:"type"`
	SchemaIdentity string `json:"schema_identity,omitempty"`
	Schemaless     bool   `json:"schemaless,omitempty"`
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

// FormatValidateResult serializes the WIDENED validate result: the violations the
// narrow renderer already carried, plus what the run asserted (the per-artifact record
// array, the asserted count, the scanned root) and the identity of the binary that
// asserted it.
//
// It reads ONE ValidateResult. The human renderer below reads the same one, which is
// what makes the two renderings incapable of reporting different identities.
func (f *JSONFormatter) FormatValidateResult(result ValidateResult) (string, error) {
	identity := effectiveBuildIdentity()
	cohort, err := schema.ComputeCohort(SchemaFS)
	if err != nil {
		return "", fmt.Errorf("computing schema cohort for output: %w", err)
	}

	env := jsonEnvelope{
		SchemaVersion:     "cli/v1",
		BinaryVersion:     identity.Version,
		SchemaCohort:      cohort.ID,
		Pass:              result.Pass,
		ViolationsCount:   len(result.Violations),
		ArtifactsAsserted: result.ArtifactsAsserted,
		ScannedRoot:       result.ScannedRoot,
		Artifacts:         make([]jsonArtifactRecord, 0, len(result.Records)),
		Violations:        make([]jsonViolation, 0, len(result.Violations)),
	}

	for _, r := range result.Records {
		env.Artifacts = append(env.Artifacts, jsonArtifactRecord{
			Path:           r.Path,
			Type:           r.Type,
			SchemaIdentity: r.SchemaIdentity,
			Schemaless:     r.Schemaless,
		})
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

// FormatValidateResult renders the WIDENED validate result for humans: the violation
// report the narrow renderer already produced, then what the run asserted and the
// identity of the binary that asserted it.
//
// It delegates the violation half to FormatResult so there is one violation-rendering
// implementation, and reads the SAME ValidateResult the JSON renderer reads — the two
// cannot report different identities because neither recomputes.
func (f *HumanFormatter) FormatValidateResult(result ValidateResult) (string, error) {
	violations, err := f.FormatResult(validate.ValidationResult{Violations: result.Violations})
	if err != nil {
		return "", fmt.Errorf("formatting the violation report: %w", err)
	}

	identity := effectiveBuildIdentity()
	cohort, err := schema.ComputeCohort(SchemaFS)
	if err != nil {
		return "", fmt.Errorf("computing schema cohort for output: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(violations)

	// The asserted count is printed even when it is ZERO. An empty pass that reports
	// nothing reads as verified when it means empty.
	sb.WriteString(fmt.Sprintf("artifacts asserted: %d\n", result.ArtifactsAsserted))
	if result.ScannedRoot != "" {
		sb.WriteString(fmt.Sprintf("scanned root: %s\n", result.ScannedRoot))
	}
	for _, r := range result.Records {
		if r.Schemaless {
			sb.WriteString(fmt.Sprintf("  %s [%s] schema-less\n", r.Path, r.Type))
			continue
		}
		sb.WriteString(fmt.Sprintf("  %s [%s] %s\n", r.Path, r.Type, r.SchemaIdentity))
	}
	sb.WriteString(fmt.Sprintf("backstop version %s\n", identity.Version))
	sb.WriteString(fmt.Sprintf("schema cohort: %s\n", cohort.ID))

	return sb.String(), nil
}
