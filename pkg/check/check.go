package check

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/bmanson/backstop-core/pkg/config"
)

// passOrder defines the fixed execution order for validation passes.
var passOrder = []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeFindings}

// Options configures a check run.
type Options struct {
	Mode        ScopeMode
	FilePath    string
	BackstopDir string
	Timeout     time.Duration
	ProjectDir  string
	// Language selects the toolchain stack (go, typescript, or a declared
	// stack). Empty defaults to the Go stack, preserving prior behavior.
	Language string
	// Config carries the loaded backstop.yml so the registry can read
	// enforcement.toolchain declarations and enforcement.test_command. The
	// contract (pkg/check consumes pkg/config.Config) is satisfied here; the
	// relevant fields are read in registry construction. Nil is treated as an
	// empty config (Go-default stack with no declarations).
	Config *config.Config
	// Files, when non-empty, is an EXPLICIT scoped file list resolved directly
	// (its own branch in resolveExplicitFiles), bypassing git scope resolution.
	// It carries a whole diff-scope file set through a SINGLE Run so the gate
	// does not loop per file — per-pass ScopeKind then shapes args (lint: all
	// files in one invocation; build/typecheck: project-wide ignoring the list;
	// test: dependency-mapped). This is DISTINCT from ScopeModeFile, whose
	// single-file semantics and 2s standalone-hook timeout must stay untouched.
	Files []string
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
	// Fingerprint is a content-based, line-INDEPENDENT identity carried from the
	// SARIF result (partialFingerprints or region snippet). It flows to
	// gate.Violation.RegionHash so the baseline keeps multiple same-rule findings
	// in one file distinct and survives unrelated line shifts. Empty when the
	// engine emits neither, leaving the coarse message-level fallback.
	Fingerprint string
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
	Git       GitExecutor
	Executors map[CheckType]PassExecutor
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

	switch {
	case len(opts.Files) > 0:
		// Explicit multi-file scope: a whole diff-scope file set carried through
		// ONE Run (its own branch — never overloads ScopeModeFile). Per-pass
		// ScopeKind shapes the arg list downstream.
		files, scopeWarnings, err = resolveExplicitFiles(opts.Files)
	case opts.Git != nil:
		files, scopeWarnings, err = resolveScopeWithGit(opts.Mode, opts.FilePath, opts.Git, withProjectDir(opts.ProjectDir))
	default:
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

	// Load manifest for file-type ROUTING only. The directory is derived from
	// BackstopDir (.backstop/rules); a routing-schema .manifest.json there is
	// honored, and an absent/empty dir falls back to defaultManifest(). The
	// compiled-standards *.semgrep.yml files in that directory are NOT a semgrep
	// rule-config source after SPEC-030 — routing and rule-config were always
	// distinct uses of the same directory.
	manifest, err := LoadManifest(filepath.Join(opts.BackstopDir, "rules"))
	if err != nil {
		return nil, err
	}

	// Build engine. Registry construction selects the toolchain by language; a
	// declared language with no built-in stack and no enforcement.toolchain
	// declaration (or an unknown format, or a TS stack missing its test command)
	// is a *ConfigError that must propagate verbatim — exit 2, never a silent
	// skip or green pass (CLM-004, CLM-007).
	executors := opts.Executors
	if executors == nil {
		built, buildErr := buildExecutorsForConfigErr(opts.Options, opts.Runner)
		if buildErr != nil {
			return nil, buildErr
		}
		executors = built
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
// runner override for hermetic tests of the default-construction path. It
// delegates to the stack-keyed registry, which selects the toolchain by
// Options.Language (Go-default when empty).
func buildDefaultExecutorsWithRunner(opts Options, runner CommandRunner) map[CheckType]PassExecutor {
	execs, _ := buildExecutorsForConfigErr(opts, runner)
	return execs
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
