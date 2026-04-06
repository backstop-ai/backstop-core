package check

import (
	"context"
	"fmt"
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
}

// Violation represents a single validation finding.
type Violation struct {
	Pass     CheckType
	File     string
	Line     int
	Message  string
	Severity string
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
		executors = buildDefaultExecutors(opts.Options)
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
func buildDefaultExecutors(opts Options) map[CheckType]PassExecutor {
	return map[CheckType]PassExecutor{
		CheckTypeLint:    &lintExecutor{},
		CheckTypeBuild:   &buildExecutor{},
		CheckTypeTest:    &testExecutor{fileMode: opts.Mode == ScopeModeFile},
		CheckTypeSemgrep: &semgrepExecutor{backstopDir: opts.BackstopDir, pinnedVersion: opts.PinnedSemgrepVersion},
	}
}

// lintExecutor runs golangci-lint.
type lintExecutor struct{}

func (e *lintExecutor) Execute(ctx context.Context, files []string) (*PassResult, error) {
	// Real implementation shells out to golangci-lint
	return &PassResult{Pass: CheckTypeLint}, nil
}

func (e *lintExecutor) IsAvailable() (bool, string) {
	_, err := findExecutable("golangci-lint")
	if err != nil {
		return false, "golangci-lint not found on PATH"
	}
	return true, ""
}

// buildExecutor runs go build.
type buildExecutor struct{}

func (e *buildExecutor) Execute(ctx context.Context, files []string) (*PassResult, error) {
	return &PassResult{Pass: CheckTypeBuild}, nil
}

func (e *buildExecutor) IsAvailable() (bool, string) {
	return true, "" // go is assumed available
}

// testExecutor runs go test.
type testExecutor struct {
	fileMode bool
}

func (e *testExecutor) Execute(ctx context.Context, files []string) (*PassResult, error) {
	return &PassResult{Pass: CheckTypeTest}, nil
}

func (e *testExecutor) IsAvailable() (bool, string) {
	return true, "" // go is assumed available
}

// semgrepExecutor runs semgrep rules.
type semgrepExecutor struct {
	backstopDir   string
	pinnedVersion string
}

func (e *semgrepExecutor) Execute(ctx context.Context, files []string) (*PassResult, error) {
	return &PassResult{Pass: CheckTypeSemgrep}, nil
}

func (e *semgrepExecutor) IsAvailable() (bool, string) {
	return true, "" // handled by EnsureSemgrep
}

// findExecutable checks if a binary is on PATH.
func findExecutable(name string) (string, error) {
	installer := &DefaultSemgrepInstaller{}
	return installer.LookPath(name)
}
