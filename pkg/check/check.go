package check

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// passOrder defines the fixed execution order for validation passes.
var passOrder = []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeSemgrep}

// Options configures a check run.
type Options struct {
	Mode                  ScopeMode
	FilePath              string
	ManifestDir           string
	BackstopDir           string
	PinnedSemgrepVersion  string
	Timeout               time.Duration
	ProjectDir            string
	GolangciLintAvailable bool
	ExtraSemgrepConfigs   []string // Additional --config paths from installed packs
}

// Violation represents a single validation finding.
type Violation struct {
	Pass     CheckType
	File     string
	Line     int
	Message  string
	Severity string
	// Rule carries a structured rule identifier for the finding. For semgrep
	// findings this is the check_id, preserved verbatim including
	// pack-namespaced IDs (pack.NamespacedRuleID format, e.g.
	// "org/pack/rule-id") so violations are attributable to their source pack.
	// Empty for passes that have no per-rule identity (lint/build/test).
	Rule string
}

// PassResult holds the result of a single validation pass.
type PassResult struct {
	Pass       CheckType
	Violations []Violation
	Skipped    bool
	SkipReason string
}

// Result holds aggregated results from all validation passes.
type Result struct {
	PassResults []PassResult
	Warnings    []string
	ExitCode    int
}

// HasViolations returns true if any pass produced violations.
func (r *Result) HasViolations() bool {
	for _, pr := range r.PassResults {
		if len(pr.Violations) > 0 {
			return true
		}
	}
	return false
}

// ViolationCount returns the total number of violations across all passes.
func (r *Result) ViolationCount() int {
	count := 0
	for _, pr := range r.PassResults {
		count += len(pr.Violations)
	}
	return count
}

// AllViolations returns all violations flattened from all passes.
func (r *Result) AllViolations() []Violation {
	var all []Violation
	for _, pr := range r.PassResults {
		all = append(all, pr.Violations...)
	}
	return all
}

// PassExecutor runs a single validation pass against a set of files.
type PassExecutor interface {
	Execute(ctx context.Context, files []string) (*PassResult, error)
	IsAvailable() (bool, string)
}

// Engine orchestrates validation passes.
type Engine struct {
	Executors map[CheckType]PassExecutor
	Manifest  *Manifest
}

// RunPasses executes all applicable validation passes in order against the
// given files. It collects all violations before returning — passes do not
// short-circuit on earlier failures.
func (e *Engine) RunPasses(ctx context.Context, files []string) (*Result, error) {
	result := &Result{
		PassResults: make([]PassResult, 0, len(passOrder)),
	}

	// Determine which check types are applicable to the given files
	applicableChecks := e.applicableChecks(files)

	for _, ct := range passOrder {
		// Check context cancellation between passes
		if ctx.Err() != nil {
			result.PassResults = append(result.PassResults, PassResult{
				Pass: ct,
				Violations: []Violation{{
					Pass:     ct,
					Message:  fmt.Sprintf("timeout: %s pass cancelled", ct),
					Severity: "error",
				}},
			})
			result.Warnings = append(result.Warnings, fmt.Sprintf("execution timed out during %s pass", ct))
			break
		}

		// Check if this pass is applicable
		if !applicableChecks[ct] {
			result.PassResults = append(result.PassResults, PassResult{
				Pass:       ct,
				Skipped:    true,
				SkipReason: "not applicable to files in scope",
			})
			continue
		}

		// Check if the executor exists
		executor, ok := e.Executors[ct]
		if !ok {
			result.PassResults = append(result.PassResults, PassResult{
				Pass:       ct,
				Skipped:    true,
				SkipReason: "no executor configured",
			})
			continue
		}

		// Check if the tool is available
		available, msg := executor.IsAvailable()
		if !available {
			result.PassResults = append(result.PassResults, PassResult{
				Pass:       ct,
				Skipped:    true,
				SkipReason: msg,
			})
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s skipped: %s", ct, msg))
			continue
		}

		// Execute the pass
		pr, err := executor.Execute(ctx, files)
		if err != nil {
			// Check if this was a context cancellation (timeout)
			if ctx.Err() != nil {
				pr = &PassResult{
					Pass: ct,
					Violations: []Violation{{
						Pass:     ct,
						Message:  fmt.Sprintf("timeout: %s pass cancelled", ct),
						Severity: "error",
					}},
				}
				result.Warnings = append(result.Warnings, fmt.Sprintf("execution timed out during %s pass", ct))
			} else {
				// Non-timeout error — record as violation
				pr = &PassResult{
					Pass: ct,
					Violations: []Violation{{
						Pass:     ct,
						Message:  fmt.Sprintf("%s error: %v", ct, err),
						Severity: "error",
					}},
				}
			}
		}

		result.PassResults = append(result.PassResults, *pr)

		// Check context after pass execution
		if ctx.Err() != nil {
			break
		}
	}

	return result, nil
}

// applicableChecks determines which check types apply to the given files
// based on the manifest.
func (e *Engine) applicableChecks(files []string) map[CheckType]bool {
	applicable := make(map[CheckType]bool)
	for _, f := range files {
		for _, ct := range e.Manifest.RouteFile(f) {
			applicable[ct] = true
		}
	}
	return applicable
}

// RunOptions extends Options with injectable dependencies for testing.
type RunOptions struct {
	Options
	Git             GitExecutor
	Executors       map[CheckType]PassExecutor
	SemgrepEnsurer  SemgrepEnsurer
	// Runner, if set, replaces the ExecCommandRunner inside the default
	// executors when Executors is nil. Lets tests exercise the
	// default-executor construction path without shelling out to live tools.
	Runner CommandRunner
}

// Run executes validation passes against the given scope and returns
// aggregated results. This is the top-level entry point.
func Run(ctx context.Context, opts Options) (*Result, error) {
	return RunWith(ctx, RunOptions{Options: opts})
}

// SemgrepEnsurer abstracts semgrep availability checking for testability.
type SemgrepEnsurer interface {
	EnsureSemgrep(backstopDir string, pinnedVersion string) (string, error)
}

// DefaultSemgrepEnsurer uses the real EnsureSemgrep function.
type DefaultSemgrepEnsurer struct{}

// EnsureSemgrep calls the package-level EnsureSemgrep function.
func (d *DefaultSemgrepEnsurer) EnsureSemgrep(backstopDir string, pinnedVersion string) (string, error) {
	return EnsureSemgrep(backstopDir, pinnedVersion)
}

// RunWith is the testable entry point with injectable dependencies.
func RunWith(ctx context.Context, opts RunOptions) (*Result, error) {
	// Apply timeout if set
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Resolve scope
	var files []string
	var scopeWarnings []string
	var err error

	if opts.Git != nil {
		files, scopeWarnings, err = resolveScopeWithGit(opts.Mode, opts.FilePath, opts.Git, withProjectDir(opts.ProjectDir))
	} else {
		files, scopeWarnings, err = ResolveScope(opts.Mode, opts.FilePath)
	}
	if err != nil {
		return nil, err
	}

	result := &Result{
		Warnings: scopeWarnings,
	}

	if len(files) == 0 {
		return result, nil
	}

	// Load manifest
	manifest, err := LoadManifest(opts.ManifestDir)
	if err != nil {
		return nil, err
	}

	// Ensure semgrep is available before running passes (Fix #1)
	ensurer := opts.SemgrepEnsurer
	if ensurer == nil {
		ensurer = &DefaultSemgrepEnsurer{}
	}

	if opts.BackstopDir != "" {
		_, semgrepErr := ensurer.EnsureSemgrep(opts.BackstopDir, opts.PinnedSemgrepVersion)
		if semgrepErr != nil {
			switch semgrepErr.(type) {
			case *ConfigError:
				// Version mismatch — hard stop, propagate
				return nil, semgrepErr
			case *DegradedError:
				// Install failure — skip semgrep with warning
				result.Warnings = append(result.Warnings, semgrepErr.Error())
				// Remove semgrep executor so the engine skips it
				if opts.Executors != nil {
					delete(opts.Executors, CheckTypeSemgrep)
				}
			}
		}
	}

	// Build engine
	executors := opts.Executors
	if executors == nil {
		executors = buildDefaultExecutorsWithRunner(opts.Options, opts.Runner)
	}

	engine := &Engine{
		Executors: executors,
		Manifest:  manifest,
	}

	engineResult, err := engine.RunPasses(ctx, files)
	if err != nil {
		return nil, err
	}

	// Merge all accumulated warnings (scope + semgrep) with engine warnings
	engineResult.Warnings = append(result.Warnings, engineResult.Warnings...)

	return engineResult, nil
}

// buildDefaultExecutors constructs pass executors for real tool invocations.
// Each executor is wired with an ExecCommandRunner rooted at the project
// directory so go build / go test resolve the project's module; tests inject a
// fake CommandRunner instead.
func buildDefaultExecutors(opts Options) map[CheckType]PassExecutor {
	return buildDefaultExecutorsWithRunner(opts, nil)
}

// buildDefaultExecutorsWithRunner is buildDefaultExecutors with an optional
// runner override for hermetic tests of the default-construction path.
func buildDefaultExecutorsWithRunner(opts Options, runner CommandRunner) map[CheckType]PassExecutor {
	if runner == nil {
		runner = &ExecCommandRunner{Dir: opts.ProjectDir}
	}
	return map[CheckType]PassExecutor{
		CheckTypeLint:  &lintExecutor{runner: runner},
		CheckTypeBuild: &buildExecutor{runner: runner},
		CheckTypeTest:  &testExecutor{runner: runner, fileMode: opts.Mode == ScopeModeFile},
		CheckTypeSemgrep: &semgrepExecutor{
			runner:              runner,
			ensurer:             &DefaultSemgrepEnsurer{},
			backstopDir:         opts.BackstopDir,
			pinnedVersion:       opts.PinnedSemgrepVersion,
			manifestDir:         opts.ManifestDir,
			extraSemgrepConfigs: opts.ExtraSemgrepConfigs,
		},
	}
}

// lintExecutor runs golangci-lint over the scoped files and converts its JSON
// findings into lint violations.
type lintExecutor struct {
	runner CommandRunner
}

func (e *lintExecutor) Execute(ctx context.Context, files []string) (*PassResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// golangci-lint run --out-format json <files...>. golangci-lint exits
	// non-zero when it finds issues, so a non-nil error is not fatal — we still
	// parse stdout. Only treat the error as fatal when there is no parseable
	// output (e.g. the binary failed to start), which the engine then records.
	args := append([]string{"run", "--out-format", "json"}, files...)
	out, runErr := e.runner.Run(ctx, "golangci-lint", args...)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	violations, parseErr := parseGolangciJSON(out)
	if parseErr != nil {
		if runErr != nil {
			return nil, fmt.Errorf("golangci-lint: %w", runErr)
		}
		return nil, fmt.Errorf("parsing golangci-lint output: %w", parseErr)
	}

	return &PassResult{Pass: CheckTypeLint, Violations: violations}, nil
}

func (e *lintExecutor) IsAvailable() (bool, string) {
	_, err := findExecutable("golangci-lint")
	if err != nil {
		return false, "golangci-lint not found on PATH"
	}
	return true, ""
}

// buildExecutor runs go build and converts compiler errors into violations.
type buildExecutor struct {
	runner CommandRunner
}

func (e *buildExecutor) Execute(ctx context.Context, files []string) (*PassResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// go build ./... in the project root. A non-zero exit normally means
	// compile errors, which we parse from the combined output.
	out, runErr := e.runner.Run(ctx, "go", "build", "./...")
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	violations := parseGoBuildErrors(out)
	// A failed run that yields no parseable compiler errors is a tool crash
	// or unparseable output, not a finding-free pass — surface it instead of
	// returning a silent green.
	if runErr != nil && len(violations) == 0 {
		return nil, fmt.Errorf("go build failed without parseable compiler errors: %v: %s", runErr, firstOutputLine(out))
	}
	return &PassResult{Pass: CheckTypeBuild, Violations: violations}, nil
}

func (e *buildExecutor) IsAvailable() (bool, string) {
	return true, "" // go is assumed available
}

// testExecutor runs go test and converts test failures into violations,
// honoring file-mode package scoping.
type testExecutor struct {
	runner   CommandRunner
	fileMode bool
}

func (e *testExecutor) Execute(ctx context.Context, files []string) (*PassResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// In file mode, scope go test to the single scoped file's package directory
	// rather than the whole module (REQ-003). Otherwise test the whole module.
	pkg := "./..."
	if e.fileMode && len(files) > 0 {
		pkg = goPackageSelector(files[0])
	}

	// A non-zero exit normally means test failures, which we parse from the
	// output.
	out, runErr := e.runner.Run(ctx, "go", "test", pkg)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	violations := parseGoTestFailures(out)
	// Same crash-vs-findings distinction as the build pass: a failed run with
	// no parseable test failures must not read as a clean pass.
	if runErr != nil && len(violations) == 0 {
		return nil, fmt.Errorf("go test failed without parseable test failures: %v: %s", runErr, firstOutputLine(out))
	}
	return &PassResult{Pass: CheckTypeTest, Violations: violations}, nil
}

func (e *testExecutor) IsAvailable() (bool, string) {
	return true, "" // go is assumed available
}

// semgrepExecutor runs semgrep rules against the compiled project rules plus
// every pack-provided ExtraSemgrepConfigs path.
type semgrepExecutor struct {
	runner              CommandRunner
	ensurer             SemgrepEnsurer
	backstopDir         string
	pinnedVersion       string
	manifestDir         string
	extraSemgrepConfigs []string
}

func (e *semgrepExecutor) Execute(ctx context.Context, files []string) (*PassResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Resolve the semgrep binary. Run() already ensures semgrep before passes,
	// but resolving here yields the concrete binary path to invoke.
	ensurer := e.ensurer
	if ensurer == nil {
		ensurer = &DefaultSemgrepEnsurer{}
	}
	binPath, ensureErr := ensurer.EnsureSemgrep(e.backstopDir, e.pinnedVersion)
	if ensureErr != nil {
		return nil, ensureErr
	}

	// Assemble --config flags: the compiled project rules dir (where the
	// standards compiler emits <number>.semgrep.yml — semgrep accepts a
	// directory) plus every pack-provided ExtraSemgrepConfigs path.
	args := []string{"--json"}
	if e.manifestDir != "" {
		args = append(args, "--config", e.manifestDir)
	}
	for _, cfg := range e.extraSemgrepConfigs {
		args = append(args, "--config", cfg)
	}
	args = append(args, files...)

	out, _ := e.runner.Run(ctx, binPath, args...)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	violations, parseErr := parseSemgrepJSON(out)
	if parseErr != nil {
		return nil, fmt.Errorf("parsing semgrep output: %w", parseErr)
	}
	return &PassResult{Pass: CheckTypeSemgrep, Violations: violations}, nil
}

func (e *semgrepExecutor) IsAvailable() (bool, string) {
	return true, "" // handled by EnsureSemgrep
}

// findExecutable checks if a binary is on PATH.
func findExecutable(name string) (string, error) {
	installer := &DefaultSemgrepInstaller{}
	return installer.LookPath(name)
}

// golangciJSON is the subset of golangci-lint `--out-format json` output the
// lint executor consumes.
type golangciJSON struct {
	Issues []struct {
		FromLinter string `json:"FromLinter"`
		Text       string `json:"Text"`
		Severity   string `json:"Severity"`
		Pos        struct {
			Filename string `json:"Filename"`
			Line     int    `json:"Line"`
			Column   int    `json:"Column"`
		} `json:"Pos"`
	} `json:"Issues"`
}

// parseGolangciJSON unmarshals golangci-lint JSON output into lint violations.
// Empty input yields zero violations (a clean run can produce no stdout).
func parseGolangciJSON(out []byte) ([]Violation, error) {
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 {
		return nil, nil
	}
	var parsed golangciJSON
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		return nil, err
	}
	violations := make([]Violation, 0, len(parsed.Issues))
	for _, issue := range parsed.Issues {
		severity := issue.Severity
		if severity == "" {
			severity = "error"
		}
		violations = append(violations, Violation{
			Pass:     CheckTypeLint,
			File:     issue.Pos.Filename,
			Line:     issue.Pos.Line,
			Message:  issue.Text,
			Severity: severity,
		})
	}
	return violations, nil
}

// semgrepJSON is the subset of semgrep `--json` output the executor consumes.
type semgrepJSON struct {
	Results []struct {
		CheckID string `json:"check_id"`
		Path    string `json:"path"`
		Start   struct {
			Line int `json:"line"`
		} `json:"start"`
		Extra struct {
			Message  string `json:"message"`
			Severity string `json:"severity"`
		} `json:"extra"`
	} `json:"results"`
}

// parseSemgrepJSON unmarshals semgrep JSON output into semgrep violations,
// preserving each finding's check_id verbatim (including pack-namespaced IDs)
// on Violation.Rule and mapping ERROR->error, WARNING->warning.
func parseSemgrepJSON(out []byte) ([]Violation, error) {
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 {
		return nil, nil
	}
	var parsed semgrepJSON
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		return nil, err
	}
	violations := make([]Violation, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		violations = append(violations, Violation{
			Pass:     CheckTypeSemgrep,
			File:     r.Path,
			Line:     r.Start.Line,
			Message:  r.Extra.Message,
			Severity: semgrepSeverity(r.Extra.Severity),
			Rule:     r.CheckID,
		})
	}
	return violations, nil
}

// semgrepSeverity normalizes a semgrep severity to the check severity vocabulary.
func semgrepSeverity(s string) string {
	switch s {
	case "ERROR":
		return "error"
	case "WARNING":
		return "warning"
	default:
		return strings.ToLower(s)
	}
}

// goPackageSelector returns the `go test` package selector for a single file:
// the file's directory as a module-relative ./-prefixed path. Files at the
// module root resolve to ".".
func goPackageSelector(file string) string {
	dir := filepath.Dir(file)
	if dir == "" || dir == "." {
		return "."
	}
	// Normalize to forward slashes and a ./ prefix so it is a local package
	// path, not an import path resolved against GOPATH/modcache.
	dir = filepath.ToSlash(dir)
	if strings.HasPrefix(dir, "/") || strings.HasPrefix(dir, "./") {
		return dir
	}
	return "./" + dir
}

// goTestFailRe matches a `--- FAIL: TestName (0.00s)` line.
var goTestFailRe = regexp.MustCompile(`^\s*--- FAIL: (\S+)`)

// goTestPosRe matches a `file_test.go:NN: message` line from a failure body.
var goTestPosRe = regexp.MustCompile(`^\s*(\S+\.go):(\d+):\s*(.*)$`)

// parseGoTestFailures parses `--- FAIL: TestName` blocks from go test output,
// pairing each failing test with the first file:line position reported in its
// failure body, into test violations.
func parseGoTestFailures(out []byte) []Violation {
	var violations []Violation
	lines := strings.Split(string(out), "\n")
	for i := 0; i < len(lines); i++ {
		m := goTestFailRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		testName := m[1]
		// Scan forward for the first file:line position before the next FAIL.
		file, lineNo, detail := "", 0, ""
		for j := i + 1; j < len(lines); j++ {
			if goTestFailRe.MatchString(lines[j]) {
				break
			}
			if pos := goTestPosRe.FindStringSubmatch(lines[j]); pos != nil {
				file = pos[1]
				if n, err := strconv.Atoi(pos[2]); err == nil {
					lineNo = n
				}
				detail = strings.TrimSpace(pos[3])
				break
			}
		}
		message := testName + " failed"
		if detail != "" {
			message = testName + ": " + detail
		}
		violations = append(violations, Violation{
			Pass:     CheckTypeTest,
			File:     file,
			Line:     lineNo,
			Message:  message,
			Severity: "error",
		})
	}
	return violations
}

// firstOutputLine returns the first non-empty line of tool output, for use in
// crash-path error messages. Empty output reads as "(no output)".
func firstOutputLine(out []byte) string {
	for _, raw := range strings.Split(string(out), "\n") {
		if line := strings.TrimSpace(raw); line != "" {
			return line
		}
	}
	return "(no output)"
}

// goBuildErrorRe matches a go build compiler error line: file:line:col: message.
// Column is optional (some diagnostics omit it).
var goBuildErrorRe = regexp.MustCompile(`^(.+?\.go):(\d+):(?:(\d+):)?\s*(.+)$`)

// parseGoBuildErrors parses `file:line:col: message` lines from go build output
// into build violations. The "# package" header lines and other non-positional
// notes are ignored.
func parseGoBuildErrors(out []byte) []Violation {
	var violations []Violation
	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := goBuildErrorRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		lineNo, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		violations = append(violations, Violation{
			Pass:     CheckTypeBuild,
			File:     m[1],
			Line:     lineNo,
			Message:  m[4],
			Severity: "error",
		})
	}
	return violations
}
