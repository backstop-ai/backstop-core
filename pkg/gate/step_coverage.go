package gate

import (
	"context"
	"fmt"
	"os"
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
	// name/args are the gate's own coverage commands (go test ...), not user input.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
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
		thresholds := coverageThresholdsForScope(specs, scope)
		if thresholds.CollapsedCodeScope {
			return collapsedCodeScopeCoverageResult(ctx, runner, scope, thresholds.MaxThreshold)
		}

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

// collapsedCodeScopeCoverageResult measures coverage PER CHANGED Go PACKAGE and
// checks each package against the threshold independently, rather than folding
// the changed packages into a single whole-repo aggregate. This is a correction
// of over-broad scoping: the diff touches a known set of packages, and the gate
// invariant is "coverage per-changed-package, not aggregate". A whole-repo sweep
// would let an over-the-floor changed package fail because unrelated, untouched
// packages drag the average down (the 87.9% vs. per-package >90% case).
func collapsedCodeScopeCoverageResult(ctx context.Context, runner CommandRunner, scope *GateScope, threshold int) StepResult {
	targets := changedGoCoverageTargets(scope)
	if len(targets) == 0 {
		return StepResult{StepName: StepCoverageThreshold, Status: "pass", Violations: []Violation{}, Reason: "no Go package coverage targets in scope"}
	}

	// Derive per-package coverage from ONE whole-module `go test ./...` run,
	// which a shared runner serves from code_check's already-executed suite
	// (dedup — the suite is the gate's dominant cost). Only used on the green
	// path (run err == nil); on the red path wholeCov is empty and every target
	// falls back to a dedicated per-package run, reproducing prior behavior
	// exactly (including the "coverage command failed" violations).
	wholeCov, wholeOK := wholeModulePackageCoverage(ctx, runner, scope)

	var violations []Violation
	for _, target := range targets {
		var pct float64
		var available bool
		if wholeOK {
			pct, available = wholeCov[target.Label]
		}
		if !available {
			var runViolations []Violation
			pct, available, runViolations = runCoverageTargets(ctx, runner, []CoverageTarget{target})
			violations = append(violations, runViolations...)
		}
		if !available {
			continue
		}
		if threshold > 0 && pct < float64(threshold) {
			violations = append(violations, Violation{
				Rule:     "coverage_threshold",
				Message:  fmt.Sprintf("changed Go package %s coverage %.1f%% below threshold %d%%", target.Label, pct, threshold),
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
	return StepResult{StepName: StepCoverageThreshold, Status: status, Violations: violations}
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

// wholeModulePackageCoverage runs `go test ./... -coverprofile=/dev/null` once
// (served from the shared runner's cache when code_check already ran it) and
// returns a map of package label (e.g. "./pkg/gate", "." for root) to its
// self-coverage percentage. It returns (nil, false) when the run errored (a
// failing test or compile error) so callers fall back to dedicated per-package
// runs and reproduce prior behavior exactly on the red path. The green-path map
// is what removes the duplicate whole-suite execution.
func wholeModulePackageCoverage(ctx context.Context, runner CommandRunner, scope *GateScope) (map[string]float64, bool) {
	var projectRoot string
	if scope != nil {
		projectRoot = scope.ProjectRoot
	}
	modulePath := goModulePath(projectRoot)
	if modulePath == "" {
		return nil, false
	}
	out, err := runner.Run(ctx, "go", "test", "./...", "-coverprofile=/dev/null")
	if err != nil {
		return nil, false
	}
	result := map[string]float64{}
	for _, line := range strings.Split(string(out), "\n") {
		pct, ok := parseCoverageLine(line)
		if !ok {
			continue
		}
		label := packageLabelFromLine(line, modulePath)
		if label == "" {
			continue
		}
		result[label] = pct
	}
	if len(result) == 0 {
		return nil, false
	}
	return result, true
}

// packageLabelFromLine maps a `go test ./...` per-package output line's import
// path to its coverage-target label: modulePath -> ".", modulePath+"/pkg/x" ->
// "./pkg/x". Returns "" when no field on the line is under the module.
func packageLabelFromLine(line, modulePath string) string {
	for _, f := range strings.Fields(line) {
		if f == modulePath {
			return "."
		}
		if strings.HasPrefix(f, modulePath+"/") {
			return "./" + strings.TrimPrefix(f, modulePath+"/")
		}
	}
	return ""
}

// goModulePath reads the module path from the project's go.mod, or "" if it
// cannot be read (callers then fall back to per-package coverage runs).
func goModulePath(projectRoot string) string {
	data, err := os.ReadFile(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func coverageTargetsForScope(scope *GateScope) []CoverageTarget {
	return goCoverageTargetsForScope(scope)
}

func goCoverageTargetsForScope(scope *GateScope) []CoverageTarget {
	var projectRoot string
	if scope != nil {
		projectRoot = scope.ProjectRoot
	}
	if scope == nil || scope.Mode == GateScopeModeAll {
		return []CoverageTarget{repoSweepCoverageTarget(projectRoot)}
	}
	if scope.Empty() {
		return nil
	}
	result := scopeCoveragePackages(scope, true)
	targets := make([]CoverageTarget, 0, len(result))
	for _, pkg := range result {
		if pkg == ". ./cmd/... ./pkg/... ./tests/..." {
			targets = append(targets, repoSweepCoverageTarget(projectRoot))
			continue
		}
		targets = append(targets, goCoverageTarget(pkg))
	}
	return targets
}

// scopeCoveragePackages derives the sorted set of Go coverage package labels
// (e.g. "./pkg/gate", "." or the whole-repo sweep label) from the scope's
// changed files. Each derived label corresponds to a single changed Go package;
// deleted directories and testdata fixtures are excluded so they never become
// spurious 0%-coverage targets.
//
// When includeSpecSweep is true a changed *.spec.md file contributes the
// whole-repo sweep label, matching the all-scope sweep semantics. When it is
// false (the per-changed-package coverage path) the sweep is omitted: that path
// checks each changed Go package against the threshold independently rather than
// folding everything into one aggregate whole-repo measurement.
func scopeCoveragePackages(scope *GateScope, includeSpecSweep bool) []string {
	if scope == nil || scope.Empty() {
		return nil
	}
	packages := map[string]struct{}{}
	// testPackages collects test-only package targets; it is iterated below only
	// under a len() > 0 guard, so the "iterate over possibly empty map" warning
	// is a false positive. Suppression anchored at the declaration (the rule's
	// match start).
	// nosemgrep: trailofbits.go.iterate-over-empty-map.iterate-over-empty-map
	testPackages := map[string]struct{}{}
	for _, file := range scope.Files {
		clean := normalizeScopePath("", file)
		if strings.HasSuffix(clean, ".spec.md") {
			if includeSpecSweep {
				packages[". ./cmd/... ./pkg/... ./tests/..."] = struct{}{}
			}
			continue
		}
		if !strings.HasSuffix(clean, ".go") || strings.HasSuffix(clean, "_testdata.go") {
			continue
		}
		// Skip .go files that live under a testdata/ directory. The Go toolchain
		// excludes testdata from `./...` expansion, so the all-scope sweep never
		// measures these fixture packages; the diff-scoped builder must match that
		// convention. A testdata fixture (e.g. cmd/.../testdata/dogfood_enforcement)
		// has source but no tests, so measuring it yields 0.0% coverage and — since
		// runCoverageTargets takes the lowest across targets — would sink the whole
		// step to a spurious failure.
		if isTestdataPath(clean) {
			continue
		}
		dir := filepath.Dir(clean)
		// Skip a derived package whose directory no longer exists on disk: a
		// DELETED .go file still appears in `git diff --name-only`, but its
		// package can't be coverage-measured (`go test ./pkg/gone` errors on a
		// missing dir). Without this guard, any change that removes a package
		// makes the coverage step fail spuriously.
		if coveragePackageDirMissing(scope.ProjectRoot, dir) {
			continue
		}
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
	// Intentional: when a diff touches only *_test.go files, fall back to their
	// (test-only) packages as coverage targets. The len(testPackages) > 0 guard
	// makes the iteration provably non-empty.
	// When a diff touches only *_test.go files, their test-only packages become
	// the coverage targets. The len() > 0 guard keeps the iteration non-empty.
	if len(packages) == 0 && len(testPackages) > 0 {
		for pkg := range testPackages {
			packages[pkg] = struct{}{}
		}
	}
	result := make([]string, 0, len(packages))
	for pkg := range packages {
		result = append(result, pkg)
	}
	sort.Strings(result)
	return result
}

// changedGoCoverageTargets builds one coverage target per changed Go package in
// scope, excluding the whole-repo spec.md sweep. It is the scheduling input for
// the per-changed-package coverage check in the collapsed-code-scope path, where
// each changed package must clear the threshold on its own rather than being
// averaged into a single whole-repo aggregate.
func changedGoCoverageTargets(scope *GateScope) []CoverageTarget {
	pkgs := scopeCoveragePackages(scope, false)
	targets := make([]CoverageTarget, 0, len(pkgs))
	for _, pkg := range pkgs {
		targets = append(targets, goCoverageTarget(pkg))
	}
	return targets
}

// repoSweepCoverageTarget builds the whole-repo coverage sweep target, pruning
// package roots that would make `go test` setup-fail under projectRoot:
//   - the module root "." is pruned when it holds no buildable .go files, since
//     `go test .` errors with "no Go files in <dir>" for a root that only nests
//     packages under ./pkg etc. (the common case for a minimal project).
//   - the top-level roots ./cmd/..., ./pkg/..., ./tests/... are pruned when the
//     corresponding directory does not exist, since `go test ./cmd/...` errors
//     with "no such file or directory".
//
// This keeps the sweep correct for projects that lack one of those directories
// (e.g. a minimal project with sources only under ./pkg). When projectRoot is
// unknown the full canonical sweep is returned unchanged.
func repoSweepCoverageTarget(projectRoot string) CoverageTarget {
	const label = ". ./cmd/... ./pkg/... ./tests/..."
	if projectRoot == "" {
		return goCoveragePackagesTarget(label, ".", "./cmd/...", "./pkg/...", "./tests/...")
	}
	var packages []string
	if dirHasGoFiles(projectRoot) {
		packages = append(packages, ".")
	}
	for _, root := range []string{"cmd", "pkg", "tests"} {
		info, err := os.Stat(filepath.Join(projectRoot, root))
		if err == nil && info.IsDir() {
			packages = append(packages, "./"+root+"/...")
		}
	}
	if len(packages) == 0 {
		// Nothing buildable resolved; fall back to the canonical sweep so the
		// step surfaces a real failure rather than silently measuring nothing.
		return goCoveragePackagesTarget(label, ".", "./cmd/...", "./pkg/...", "./tests/...")
	}
	return goCoveragePackagesTarget(label, packages...)
}

// isTestdataPath reports whether the slash-separated path lies under a
// "testdata" directory at any depth. The Go toolchain treats testdata as a
// reserved fixture directory and excludes it from `./...` package expansion, so
// such paths must not be turned into coverage targets.
func isTestdataPath(path string) bool {
	for _, segment := range strings.Split(path, "/") {
		if segment == "testdata" {
			return true
		}
	}
	return false
}

// dirHasGoFiles reports whether dir directly contains at least one .go source
// file (test files included). It is used to decide whether `go test .` on the
// module root would build, vs. setup-fail with "no Go files in <dir>".
func dirHasGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			return true
		}
	}
	return false
}

// coveragePackageDirMissing reports whether the package directory dir is known
// to be absent on disk under projectRoot — i.e. the scoped .go file was deleted.
// It returns false when projectRoot is unknown (so callers without a resolved
// root, e.g. unit tests, keep their prior behavior) or when dir is the module
// root ("."), which always exists.
func coveragePackageDirMissing(projectRoot, dir string) bool {
	if projectRoot == "" || dir == "" || dir == "." {
		return false
	}
	info, err := os.Stat(filepath.Join(projectRoot, dir))
	if err != nil {
		return os.IsNotExist(err)
	}
	return !info.IsDir()
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
