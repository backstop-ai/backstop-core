package gate

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// CommandRunner abstracts external command execution for testability.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// SpecVerification holds the verification block fields from a spec.
type SpecVerification struct {
	SpecID            string
	TestCommand       string
	CoverageThreshold int
}

// coverageRe matches the go test coverage summary line format.
var coverageRe = regexp.MustCompile(`coverage:\s+(\d+\.?\d*)%\s+of\s+statements`)

// parseCoverageLine extracts the coverage percentage from a line matching
// the "coverage: NN.N% of statements" format.
func parseCoverageLine(line string) (float64, bool) {
	matches := coverageRe.FindStringSubmatch(line)
	if len(matches) < 2 {
		return 0, false
	}
	pct, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, false
	}
	return pct, true
}

// StepCoverageThresholdFunc returns a StepFunc that runs the test suite with
// coverage profiling and compares against the spec-declared threshold.
func StepCoverageThresholdFunc(runner CommandRunner, specs []SpecVerification) StepFunc {
	return func(ctx context.Context) StepResult {
		var violations []Violation

		for _, spec := range specs {
			if spec.CoverageThreshold <= 0 {
				continue
			}

			// Parse the test command
			parts := strings.Fields(spec.TestCommand)
			if len(parts) == 0 {
				violations = append(violations, Violation{
					Rule:     "coverage_threshold",
					Message:  fmt.Sprintf("spec %s: empty test command", spec.SpecID),
					Severity: "error",
				})
				continue
			}

			// Append -coverprofile if not already present
			args := parts[1:]
			hasCoverprofile := false
			for _, arg := range args {
				if strings.HasPrefix(arg, "-coverprofile") {
					hasCoverprofile = true
					break
				}
			}
			if !hasCoverprofile {
				args = append(args, "-coverprofile=/dev/null")
			}

			// Execute the test command
			output, err := runner.Run(ctx, parts[0], args...)
			if err != nil {
				violations = append(violations, Violation{
					Rule:     "coverage_threshold",
					Message:  fmt.Sprintf("spec %s: test command failed: %v", spec.SpecID, err),
					Severity: "error",
				})
				continue
			}

			// Parse coverage from output
			lines := strings.Split(string(output), "\n")
			var pct float64
			found := false
			for _, line := range lines {
				if p, ok := parseCoverageLine(line); ok {
					pct = p
					found = true
				}
			}

			if !found {
				violations = append(violations, Violation{
					Rule:     "coverage_threshold",
					Message:  "coverage summary line not found in test output",
					Severity: "error",
				})
				continue
			}

			if pct < float64(spec.CoverageThreshold) {
				violations = append(violations, Violation{
					Rule:     "coverage_threshold",
					Message:  fmt.Sprintf("spec %s: coverage %.1f%% below threshold %d%%", spec.SpecID, pct, spec.CoverageThreshold),
					Severity: "error",
				})
			}
		}

		status := "pass"
		if len(violations) > 0 {
			status = "fail"
		}
		if violations == nil {
			violations = []Violation{}
		}
		return StepResult{
			StepName:   StepCoverageThreshold,
			Status:     status,
			Violations: violations,
		}
	}
}
