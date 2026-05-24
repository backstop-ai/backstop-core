package gate

import (
	"context"
	"fmt"
	"os/exec"
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
	File              string
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

// ExecCommandRunner is a CommandRunner that uses os/exec to run commands.
type ExecCommandRunner struct {
	Dir string // working directory for commands
}

// Run executes the named command with args and returns combined output.
func (r *ExecCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	return cmd.CombinedOutput()
}

// StepCoverageThresholdFunc returns a StepFunc that runs the test suite with
// coverage profiling and compares against the spec-declared threshold.
func StepCoverageThresholdFunc(runner CommandRunner, specs []SpecVerification) StepFunc {
	return StepCoverageThresholdScopedFunc(runner, specs, nil)
}

// StepCoverageThresholdScopedFunc runs coverage only for scoped specs or changed packages.
func StepCoverageThresholdScopedFunc(runner CommandRunner, specs []SpecVerification, scope *GateScope) StepFunc {
	return func(ctx context.Context) StepResult {
		var violations []Violation

		for _, spec := range specs {
			if !coverageSpecInScope(spec, scope) {
				continue
			}
			if spec.CoverageThreshold <= 0 {
				continue
			}

			// Parse the test command
			parts := commandFields(spec.TestCommand)
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

func coverageSpecInScope(spec SpecVerification, scope *GateScope) bool {
	if scope == nil || scope.Mode == GateScopeModeAll {
		return true
	}
	if scope.Empty() {
		return false
	}
	if spec.File != "" && scope.Contains(spec.File) {
		return true
	}
	for _, pkg := range coverageCommandPackages(spec.TestCommand) {
		if pkg == "" {
			continue
		}
		for _, file := range scope.Files {
			if coveragePackageContainsFile(pkg, file) {
				return true
			}
		}
	}
	return false
}

func coverageCommandPackages(command string) []string {
	var packages []string
	for _, part := range commandFields(command) {
		if strings.HasPrefix(part, "-") || strings.Contains(part, "=") {
			continue
		}
		if strings.HasPrefix(part, "./") || strings.HasPrefix(part, "/") {
			packages = append(packages, normalizeCoveragePackage(part))
		}
	}
	return packages
}

func commandFields(command string) []string {
	var fields []string
	var current strings.Builder
	var quote rune
	for _, r := range command {
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' {
			if current.Len() > 0 {
				fields = append(fields, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}
	return fields
}

func normalizeCoveragePackage(pkg string) string {
	trimmed := strings.TrimPrefix(strings.TrimPrefix(pkg, "./"), "/")
	if trimmed == "..." {
		return ""
	}
	return strings.TrimSuffix(strings.TrimSuffix(trimmed, "/..."), "/")
}

func coveragePackageContainsFile(pkg, file string) bool {
	cleanFile := normalizeScopePath("", file)
	cleanPkg := normalizeCoveragePackage(pkg)
	if cleanPkg == "" {
		return true
	}
	return cleanFile == cleanPkg || strings.HasPrefix(cleanFile, cleanPkg+"/")
}
