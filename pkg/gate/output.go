package gate

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ANSI color codes for human output.
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBold   = "\033[1m"
)

// FormatJSON marshals a GateResult to indented JSON.
func FormatJSON(result GateResult) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}

// FormatHuman formats a GateResult as human-readable text.
func FormatHuman(result GateResult, noColor bool) string {
	var sb strings.Builder

	// Header
	if noColor {
		sb.WriteString("Gate Results\n")
	} else {
		sb.WriteString(fmt.Sprintf("%s%sGate Results%s\n", colorBold, colorReset, colorReset))
	}
	sb.WriteString(strings.Repeat("─", 60) + "\n")

	// Summary table
	for _, step := range result.Steps {
		statusStr := formatStatus(step.Status, noColor)
		violationCount := len(step.Violations)

		line := fmt.Sprintf("  %-25s %s", step.StepName, statusStr)
		if violationCount > 0 {
			line += fmt.Sprintf("  (%d violations)", violationCount)
		}
		if step.Status == "skipped" && step.Reason != "" {
			line += fmt.Sprintf("  (%s)", step.Reason)
		}
		sb.WriteString(line + "\n")
	}

	sb.WriteString(strings.Repeat("─", 60) + "\n")

	// Summary counts
	sb.WriteString(fmt.Sprintf("  Steps: %d passed, %d failed, %d skipped\n",
		result.StepsPassed, result.StepsFailed, result.StepsSkipped))
	sb.WriteString(fmt.Sprintf("  Total violations: %d\n", result.TotalViolations))

	// Overall verdict
	sb.WriteString("\n")
	if result.Pass {
		if noColor {
			sb.WriteString("PASS\n")
		} else {
			sb.WriteString(fmt.Sprintf("%sPASS%s\n", colorGreen, colorReset))
		}
	} else {
		if noColor {
			sb.WriteString("FAIL\n")
		} else {
			sb.WriteString(fmt.Sprintf("%sFAIL%s\n", colorRed, colorReset))
		}
	}

	return sb.String()
}

// formatStatus formats a step status with optional color.
func formatStatus(status string, noColor bool) string {
	if noColor {
		return status
	}
	switch status {
	case "pass":
		return fmt.Sprintf("%s%s%s", colorGreen, status, colorReset)
	case "fail":
		return fmt.Sprintf("%s%s%s", colorRed, status, colorReset)
	case "skipped":
		return fmt.Sprintf("%s%s%s", colorYellow, status, colorReset)
	default:
		return status
	}
}

// NoColorFromEnv checks the NO_COLOR environment variable.
// Any non-empty value means no color.
func NoColorFromEnv() bool {
	return os.Getenv("NO_COLOR") != ""
}
