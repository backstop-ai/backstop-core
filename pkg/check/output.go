package check

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// OutputMode determines the output format.
type OutputMode int

const (
	// OutputModeHuman formats output for terminal reading.
	OutputModeHuman OutputMode = iota
	// OutputModeJSON formats output as structured JSON.
	OutputModeJSON
)

// JSONOutput is the wire format for JSON output.
type JSONOutput struct {
	SchemaVersion string          `json:"schema_version"`
	Pass          bool            `json:"pass"`
	Violations    []JSONViolation `json:"violations"`
	Warnings      []string        `json:"warnings"`
	PassResults   []JSONPassInfo  `json:"pass_results"`
	ExitCode      int             `json:"exit_code"`
}

// JSONViolation is the JSON representation of a single violation.
type JSONViolation struct {
	Pass     string `json:"pass"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message"`
	Severity string `json:"severity,omitempty"`
	// Rule carries the structured rule identifier (e.g. a pack-namespaced
	// semgrep check_id) so the namespaced ID is not dropped from output.
	Rule string `json:"rule,omitempty"`
}

// JSONPassInfo summarizes a pass result for JSON output.
type JSONPassInfo struct {
	Pass       string `json:"pass"`
	Skipped    bool   `json:"skipped"`
	SkipReason string `json:"skip_reason,omitempty"`
	Violations int    `json:"violations"`
}

// FormatResult formats a Result for output in the given mode.
func FormatResult(result *Result, mode OutputMode) (string, error) {
	switch mode {
	case OutputModeJSON:
		return formatJSON(result)
	case OutputModeHuman:
		return formatHuman(result), nil
	default:
		return "", fmt.Errorf("unknown output mode: %d", mode)
	}
}

func formatJSON(result *Result) (string, error) {
	out := JSONOutput{
		SchemaVersion: "check/v1",
		Pass:          !result.HasViolations(),
		Violations:    make([]JSONViolation, 0),
		Warnings:      result.Warnings,
		PassResults:   make([]JSONPassInfo, 0, len(result.PassResults)),
		ExitCode:      DetermineExitCode(result, nil, false),
	}

	if out.Warnings == nil {
		out.Warnings = []string{}
	}

	for _, pr := range result.PassResults {
		out.PassResults = append(out.PassResults, JSONPassInfo{
			Pass:       pr.Pass.String(),
			Skipped:    pr.Skipped,
			SkipReason: pr.SkipReason,
			Violations: len(pr.Violations),
		})
		for _, v := range pr.Violations {
			out.Violations = append(out.Violations, JSONViolation{
				Pass:     v.Pass.String(),
				File:     v.File,
				Line:     v.Line,
				Message:  v.Message,
				Severity: v.Severity,
				Rule:     v.Rule,
			})
		}
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling JSON output: %w", err)
	}
	return string(data), nil
}

func formatHuman(result *Result) string {
	_, noColorSet := os.LookupEnv("NO_COLOR")
	useColor := !noColorSet
	var sb strings.Builder

	violations := result.AllViolations()

	if len(violations) > 0 {
		// Group by file
		fileOrder := []string{}
		byFile := map[string][]Violation{}
		for _, v := range violations {
			file := v.File
			if file == "" {
				file = "(no file)"
			}
			if _, seen := byFile[file]; !seen {
				fileOrder = append(fileOrder, file)
			}
			byFile[file] = append(byFile[file], v)
		}

		for _, file := range fileOrder {
			if useColor {
				sb.WriteString(fmt.Sprintf("\033[1m%s\033[0m\n", file))
			} else {
				sb.WriteString(fmt.Sprintf("%s\n", file))
			}
			for _, v := range byFile[file] {
				prefix := fmt.Sprintf("  [%s]", v.Pass)
				if v.Rule != "" {
					// Surface the structured rule ID (e.g. a pack-namespaced
					// semgrep check_id) alongside the pass prefix.
					prefix = fmt.Sprintf("  [%s:%s]", v.Pass, v.Rule)
				}
				if useColor && v.Severity == "error" {
					sb.WriteString(fmt.Sprintf("\033[31m%s %s\033[0m\n", prefix, v.Message))
				} else {
					sb.WriteString(fmt.Sprintf("%s %s\n", prefix, v.Message))
				}
			}
		}
	}

	// Warnings
	if len(result.Warnings) > 0 {
		sb.WriteString("\nWarnings:\n")
		for _, w := range result.Warnings {
			sb.WriteString(fmt.Sprintf("  - %s\n", w))
		}
	}

	// Summary
	if result.HasViolations() {
		count := result.ViolationCount()
		if useColor {
			sb.WriteString(fmt.Sprintf("\n\033[31m✗ %d violation(s) found\033[0m\n", count))
		} else {
			sb.WriteString(fmt.Sprintf("\n✗ %d violation(s) found\n", count))
		}
	} else {
		if useColor {
			sb.WriteString("\033[32m✓ All checks passed\033[0m\n")
		} else {
			sb.WriteString("✓ All checks passed\n")
		}
	}

	// Skipped passes
	for _, pr := range result.PassResults {
		if pr.Skipped && pr.SkipReason != "" {
			sb.WriteString(fmt.Sprintf("  [skip] %s: %s\n", pr.Pass, pr.SkipReason))
		}
	}

	return sb.String()
}

// DetermineExitCode computes the exit code from result state.
// Exit code 2 (config error or flag conflict) takes precedence over 1 (violations).
func DetermineExitCode(result *Result, configErr error, flagConflict bool) int {
	if configErr != nil || flagConflict {
		return 2
	}
	if result != nil && result.HasViolations() {
		return 1
	}
	return 0
}

// ValidateBackstopDir checks that .backstop/ directory exists at the given root.
func ValidateBackstopDir(projectRoot string) error {
	backstopDir := fmt.Sprintf("%s/.backstop", projectRoot)
	info, err := os.Stat(backstopDir)
	if err != nil || !info.IsDir() {
		return &ConfigError{
			Message: fmt.Sprintf(".backstop directory not found at %s", projectRoot),
		}
	}
	return nil
}
