package gate

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// CommandRunner abstracts external command execution for testability.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// SpecVerification holds the verification block fields from a spec.
type SpecVerification struct {
	SpecID                string
	TestCommand           string
	CoverageThreshold     int
	File                  string
	ImplementationPackage string
}

const defaultCodeScopeCoverageFloor = 90

// CoverageTarget is one concrete coverage command selected by the gate. The
// spec documents thresholds; target selection belongs to the gate so different
// stacks can plug in their own schedulers without spec-authored commands
// becoming execution plans.
type CoverageTarget struct {
	Stack   string
	Label   string
	Command string
	Args    []string
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
		targets := coverageTargetsForScope(scope)
		if len(targets) == 0 {
			return StepResult{StepName: StepCoverageThreshold, Status: "pass", Violations: []Violation{}, Reason: "no Go package coverage targets in scope"}
		}

		coveragePct, coverageAvailable, runViolations := runCoverageTargets(ctx, runner, targets)
		violations = append(violations, runViolations...)
		if !coverageAvailable {
			return StepResult{StepName: StepCoverageThreshold, Status: "fail", Violations: violations}
		}

		thresholds := coverageThresholdsForScope(specs, scope)
		for _, spec := range thresholds.Specs {
			if !coverageSpecInScope(spec, scope) {
				continue
			}
			if spec.CoverageThreshold <= 0 {
				continue
			}

			if coveragePct < float64(spec.CoverageThreshold) {
				violations = append(violations, Violation{
					Rule:     "coverage_threshold",
					Message:  fmt.Sprintf("spec %s: coverage %.1f%% below threshold %d%%", spec.SpecID, coveragePct, spec.CoverageThreshold),
					Severity: "error",
				})
			}
		}
		if thresholds.CollapsedCodeScope && coveragePct < float64(thresholds.MaxThreshold) {
			violations = append(violations, Violation{
				Rule:     "coverage_threshold",
				Message:  fmt.Sprintf("changed Go package coverage %.1f%% below threshold %d%%", coveragePct, thresholds.MaxThreshold),
				Severity: "error",
			})
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

type coverageThresholdSelection struct {
	Specs              []SpecVerification
	CollapsedCodeScope bool
	MaxThreshold       int
}

func coverageThresholdsForScope(specs []SpecVerification, scope *GateScope) coverageThresholdSelection {
	if scope == nil || scope.Mode == GateScopeModeAll {
		return coverageThresholdSelection{Specs: specs}
	}
	selected := []SpecVerification{}
	for _, spec := range specs {
		if spec.File != "" && scope.Contains(spec.File) {
			selected = append(selected, spec)
		}
	}
	if len(selected) > 0 {
		return coverageThresholdSelection{Specs: selected}
	}
	maxThreshold := 0
	hasSpecific := false
	for _, spec := range specs {
		if !coverageSpecRelevantToCodeScope(spec, scope, false) {
			continue
		}
		hasSpecific = true
		if spec.CoverageThreshold > maxThreshold {
			maxThreshold = spec.CoverageThreshold
		}
	}
	if !hasSpecific {
		for _, spec := range specs {
			if !coverageSpecRelevantToCodeScope(spec, scope, true) {
				continue
			}
			if spec.CoverageThreshold > maxThreshold {
				maxThreshold = spec.CoverageThreshold
			}
		}
	}
	if maxThreshold == 0 {
		maxThreshold = defaultCodeScopeCoverageFloor
	}
	return coverageThresholdSelection{CollapsedCodeScope: true, MaxThreshold: maxThreshold}
}

func coverageSpecRelevantToCodeScope(spec SpecVerification, scope *GateScope, includeRootCommand bool) bool {
	if scope == nil || scope.Empty() {
		return false
	}
	for _, file := range scope.Files {
		if coverageSpecRelevantToFile(spec, normalizeScopePath("", file), includeRootCommand) {
			return true
		}
	}
	return false
}

func coverageSpecRelevantToFile(spec SpecVerification, file string, includeRootCommand bool) bool {
	if !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_testdata.go") {
		return false
	}
	dir := filepath.Dir(file)
	if dir == "." {
		dir = ""
	}
	if spec.ImplementationPackage != "" && packagePathMatches(dir, spec.ImplementationPackage) {
		return true
	}
	if spec.TestCommand == "" {
		return false
	}
	return includeRootCommand && strings.Contains(spec.TestCommand, "./...") || strings.Contains(spec.TestCommand, "./"+dir)
}

func packagePathMatches(changedDir string, specPackage string) bool {
	trimmed := strings.TrimPrefix(strings.Trim(specPackage, "/"), "./")
	if changedDir == "" || trimmed == "" {
		return changedDir == trimmed
	}
	return changedDir == trimmed || strings.HasPrefix(changedDir, trimmed+"/") || strings.HasPrefix(trimmed, changedDir+"/")
}

func runCoverageTargets(ctx context.Context, runner CommandRunner, targets []CoverageTarget) (float64, bool, []Violation) {
	var lowest float64
	foundAny := false
	var violations []Violation

	for _, target := range targets {
		output, err := runner.Run(ctx, target.Command, target.Args...)
		if err != nil {
			violations = append(violations, Violation{Rule: "coverage_threshold", Message: fmt.Sprintf("coverage command failed for %s: %v%s", target.Label, err, coverageOutputExcerpt(output)), Severity: "error"})
		}

		found := false
		for _, line := range strings.Split(string(output), "\n") {
			if pct, ok := parseCoverageLine(line); ok {
				if !foundAny || pct < lowest {
					lowest = pct
				}
				found = true
				foundAny = true
			}
		}
		if !found && !strings.Contains(string(output), "coverage: [no statements]") {
			violations = append(violations, Violation{Rule: "coverage_threshold", Message: fmt.Sprintf("coverage summary line not found in test output for %s", target.Label), Severity: "error"})
		}
	}

	return lowest, foundAny, violations
}

func coverageTargetsForScope(scope *GateScope) []CoverageTarget {
	return goCoverageTargetsForScope(scope)
}

func goCoverageTargetsForScope(scope *GateScope) []CoverageTarget {
	if scope == nil || scope.Mode == GateScopeModeAll {
		return []CoverageTarget{
			goCoveragePackagesTarget(". ./cmd/... ./pkg/... ./tests/...", ".", "./cmd/...", "./pkg/...", "./tests/..."),
		}
	}
	if scope.Empty() {
		return nil
	}
	packages := map[string]struct{}{}
	testPackages := map[string]struct{}{}
	for _, file := range scope.Files {
		clean := normalizeScopePath("", file)
		if strings.HasSuffix(clean, ".spec.md") {
			packages[". ./cmd/... ./pkg/... ./tests/..."] = struct{}{}
			continue
		}
		if !strings.HasSuffix(clean, ".go") || strings.HasSuffix(clean, "_testdata.go") {
			continue
		}
		dir := filepath.Dir(clean)
		selected := packages
		if strings.HasSuffix(clean, "_test.go") {
			selected = testPackages
		}
		if dir == "." {
			selected["."] = struct{}{}
			continue
		}
		selected["./"+dir] = struct{}{}
	}
	if len(packages) == 0 {
		for pkg := range testPackages {
			packages[pkg] = struct{}{}
		}
	}
	result := make([]string, 0, len(packages))
	for pkg := range packages {
		result = append(result, pkg)
	}
	sort.Strings(result)
	targets := make([]CoverageTarget, 0, len(result))
	for _, pkg := range result {
		if pkg == ". ./cmd/... ./pkg/... ./tests/..." {
			targets = append(targets, goCoveragePackagesTarget(pkg, ".", "./cmd/...", "./pkg/...", "./tests/..."))
			continue
		}
		targets = append(targets, goCoverageTarget(pkg))
	}
	return targets
}

func goCoverageTarget(pkg string) CoverageTarget {
	return goCoveragePackagesTarget(pkg, pkg)
}

func goCoveragePackagesTarget(label string, packages ...string) CoverageTarget {
	return CoverageTarget{Stack: "go", Label: label, Command: "go", Args: append(append([]string{"test"}, packages...), "-coverprofile=/dev/null")}
}

func coverageOutputExcerpt(output []byte) string {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	selected := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "--- FAIL:") || strings.HasPrefix(trimmed, "FAIL") || strings.Contains(trimmed, ": ") && !strings.HasPrefix(trimmed, "ok  ") {
			selected = append(selected, trimmed)
		}
		if len(selected) == 4 {
			break
		}
	}
	if len(selected) == 0 {
		return ""
	}
	return ": " + strings.Join(selected, " | ")
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
	return coverageTargetsForScope(scope) != nil
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
