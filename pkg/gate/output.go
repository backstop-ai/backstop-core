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

	// The producing binary and the root it scanned, read off the SAME GateResult fields
	// the JSON rendering serializes. They are emitted HERE, in the formatter, and
	// nowhere else: a Println beside the FormatHuman call site would be a SECOND
	// rendering path, and "both renderings read one resolved value" exists precisely to
	// forbid two surfaces that can disagree about one run.
	//
	// The configured flag is printed even when false, for the same reason its JSON tag
	// carries no omitempty: unconfigured is the default and the motivating state.
	if result.BinaryVersion != "" || result.SchemaCohort != "" || result.ArtifactRoot != "" {
		if result.BinaryVersion != "" {
			sb.WriteString(fmt.Sprintf("backstop version %s\n", result.BinaryVersion))
		}
		if result.SchemaCohort != "" {
			sb.WriteString(fmt.Sprintf("schema cohort: %s\n", result.SchemaCohort))
		}
		if result.ArtifactRoot != "" {
			sb.WriteString(fmt.Sprintf("artifact root: %s (configured: %t)\n", result.ArtifactRoot, result.ArtifactRootConfigured))
		}
		sb.WriteString(strings.Repeat("─", 60) + "\n")
	}

	if result.Scope != nil && result.Scope.Mode != GateScopeModeAll {
		switch result.Scope.Mode {
		case GateScopeModeFile:
			sb.WriteString(fmt.Sprintf("Gate running against %d explicit files.\n", len(result.Scope.Files)))
		default:
			if len(result.Scope.Files) == 0 {
				sb.WriteString("Gate found no changed files; scoped checks have no files to inspect.\n")
			} else {
				sb.WriteString(fmt.Sprintf("Gate running against %d changed files (use --all for full sweep).\n", len(result.Scope.Files)))
			}
		}
		// An EXPLICIT base gets its resolution spelled out. Without this a CI reader
		// cannot tell "green over 12 files since <sha>" from an unexplained green over
		// zero — and the zero case is the vacuous green --base exists to prevent, so
		// the number it checked has to be legible in the log itself.
		if result.Scope.RequestedBase != "" {
			sb.WriteString(fmt.Sprintf("Scope mode: %s | requested base: %s | resolved merge-base: %s | in-scope files: %d\n",
				result.Scope.Mode, result.Scope.RequestedBase, result.Scope.MergeBase, len(result.Scope.Files)))
		}
		sb.WriteString(strings.Repeat("─", 60) + "\n")
	}

	// Summary table
	for _, step := range result.Steps {
		statusStr := formatStatus(step.Status, noColor)
		violationCount := len(step.Violations)
		reason := step.Reason
		if step.StepName == StepBaselineComparison && reason == "" {
			if len(step.NewViolations) == 0 {
				if step.Status == "fail" && len(step.Violations) > 0 {
					reason = fmt.Sprintf("%d new violations beyond baseline", len(step.Violations))
				} else {
					reason = "0 new violations beyond baseline"
				}
			} else {
				reason = fmt.Sprintf("%d new violations beyond baseline", len(step.NewViolations))
			}
		}

		line := fmt.Sprintf("  %-25s %s", step.StepName, statusStr)
		if step.DurationMS > 0 {
			line += fmt.Sprintf("  (%dms)", step.DurationMS)
		}
		if violationCount > 0 {
			line += fmt.Sprintf("  (%d violations)", violationCount)
		}
		if reason != "" {
			line += fmt.Sprintf("  (%s)", reason)
		}
		sb.WriteString(line + "\n")
	}

	// Violation details
	for _, step := range result.Steps {
		if len(step.Violations) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("\n  %s violations:\n", step.StepName))
		for _, violation := range step.Violations {
			rule := violation.Rule
			if violation.SourcePack != "" && !strings.HasPrefix(rule, violation.SourcePack+"/") {
				rule = violation.SourcePack + "/" + rule
			}
			sb.WriteString(fmt.Sprintf("    - [%s] %s", rule, violation.Message))
			if violation.File != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", violation.File))
			}
			// REQ-014: on a still-blocked waivable finding, hand the author the
			// pre-filled @waiver token so acknowledging is one paste.
			if violation.WaiverHint != "" {
				sb.WriteString(fmt.Sprintf("\n      ↳ to waive: %s", violation.WaiverHint))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString(strings.Repeat("─", 60) + "\n")

	// Summary counts. The warned count is rendered alongside passed/failed/
	// skipped (SPEC-036 CLM-030) so a class-2 capability-absent advisory cannot
	// vanish from the at-a-glance summary on a green run.
	sb.WriteString(fmt.Sprintf("  Steps: %d passed, %d failed, %d skipped, %d warned\n",
		result.StepsPassed, result.StepsFailed, result.StepsSkipped, result.StepsWarned))
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
	case "warning":
		// Conspicuous, visually distinct from a silent pass (SPEC-036 REQ-005 /
		// CLM-015): bold yellow with a bang prefix so a reviewer notices the
		// class-2 advisory on an otherwise-green run.
		return fmt.Sprintf("%s%s⚠ %s%s", colorBold, colorYellow, status, colorReset)
	default:
		return status
	}
}

// NoColorFromEnv checks the NO_COLOR environment variable.
// Any non-empty value means no color.
func NoColorFromEnv() bool {
	return os.Getenv("NO_COLOR") != ""
}
