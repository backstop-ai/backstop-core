package gate

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Gate orchestrates the nine-step verification kill chain.
type Gate struct {
	steps                       []StepFunc
	configErr                   error
	scope                       *GateScope
	baseline                    *BaselineArtifact
	baselineEnabled             bool
	baselineWarning             string
	baselinePath                string
	baselineTTL                 time.Duration
	baselineModified            time.Time
	ruleSetChangeSeedingAllowed bool
	ruleSetChangeFiles          map[string]struct{}
	policy                      map[string]DimensionPolicy
}

// Option is a functional option for configuring a Gate.
type Option func(*Gate)

// WithSteps sets the ordered step functions for the gate.
func WithSteps(steps []StepFunc) Option {
	return func(g *Gate) {
		g.steps = steps
	}
}

// WithConfigError sets a config error that causes immediate exit code 2.
func WithConfigError(err error) Option {
	return func(g *Gate) {
		g.configErr = err
	}
}

// WithScope attaches the gate scope computed once at command startup.
func WithScope(scope *GateScope) Option {
	return func(g *Gate) {
		g.scope = scope
	}
}

// WithBaseline attaches a loaded baseline artifact for step 7 comparison.
func WithBaseline(artifact *BaselineArtifact) Option {
	return func(g *Gate) {
		g.baselineEnabled = true
		g.baseline = artifact
	}
}

// WithBaselineWarning sets baseline warning text for diagnostics.
func WithBaselineWarning(warning string) Option {
	return func(g *Gate) {
		g.baselineEnabled = true
		g.baselineWarning = strings.TrimSpace(warning)
	}
}

// WithBaselineCacheMeta sets cache metadata used in output diagnostics.
func WithBaselineCacheMeta(path string, ttl time.Duration, modified time.Time) Option {
	return func(g *Gate) {
		g.baselinePath = path
		g.baselineTTL = ttl
		g.baselineModified = modified
	}
}

// WithRuleSetChangeSeedingAllowed enables REQ-013's narrow seeding exception.
func WithRuleSetChangeSeedingAllowed(allowed bool) Option {
	return func(g *Gate) {
		g.ruleSetChangeSeedingAllowed = allowed
	}
}

// WithRuleSetChangeFiles marks files changed in the seeding context.
// WithPolicy sets the per-dimension enforcement policy (level + baseline grandfathering),
// keyed by gate dimension name. An empty/nil map leaves all dimensions at the default
// (block, no baseline), so the gate behaves exactly as before.
func WithPolicy(policy map[string]DimensionPolicy) Option {
	return func(g *Gate) { g.policy = policy }
}

func WithRuleSetChangeFiles(files []string) Option {
	return func(g *Gate) {
		if len(files) == 0 {
			g.ruleSetChangeFiles = nil
			return
		}
		g.ruleSetChangeFiles = map[string]struct{}{}
		for _, file := range files {
			trimmed := strings.TrimSpace(file)
			if trimmed == "" {
				continue
			}
			g.ruleSetChangeFiles[trimmed] = struct{}{}
		}
	}
}

// New creates a Gate with the given options.
func New(opts ...Option) *Gate {
	g := &Gate{}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Run executes all gate steps in order and returns the aggregated result
// and exit code. Exit codes: 0 (all green), 1 (failures), 2 (config error).
//
// If a config error was set at construction time, Run returns immediately
// with exit code 2. If a delegated step signals a config error (ConfigErr
// flag), Run halts remaining steps and returns exit code 2.
func (g *Gate) Run(ctx context.Context) (GateResult, int) {
	// Config error before any steps → exit 2 immediately.
	if g.configErr != nil {
		return GateResult{
			SchemaVersion: "gate/v1",
			Scope:         g.scope,
			Pass:          false,
			Steps:         []StepResult{},
		}, 2
	}

	var results []StepResult
	configErrHalt := false

	for _, stepFn := range g.steps {
		started := time.Now()
		result := stepFn(ctx)
		if g.baselineEnabled && result.StepName == StepBaselineComparison {
			result = g.computeBaselineResult(results)
		}
		result.DurationMS = time.Since(started).Milliseconds()
		results = append(results, result)

		// Check for config error from delegated steps.
		if result.ConfigErr {
			configErrHalt = true
			break
		}
	}

	results = ApplyPolicy(results, g.baseline, g.policy, g.scope)

	gateResult := NewGateResultWithScope(results, g.scope)

	if configErrHalt {
		return gateResult, 2
	}

	return gateResult, ExitCode(gateResult, nil)
}

func (g *Gate) computeBaselineResult(accumulated []StepResult) StepResult {
	if g.baseline == nil {
		reason := "baseline unavailable: no cached baseline found at .backstop/baseline.json; run CI baseline publication or backstop baseline pull"
		if g.baselineWarning != "" {
			reason = g.baselineWarning
		}
		return StepResult{StepName: StepBaselineComparison, Status: "skipped", Violations: []Violation{}, NewViolations: []Violation{}, FixedViolations: []Violation{}, Reason: reason}
	}
	warningSuffix := ""
	if g.baselineWarning != "" {
		warningSuffix = "; " + g.baselineWarning
	}
	comparison := CompareBaseline(accumulatedViolations(accumulated), g.baseline, BaselineCompareOptions{
		Scope:                     g.scope,
		AllowRuleSetChangeSeeding: g.ruleSetChangeSeedingAllowed,
		ChangedFiles:              g.ruleSetChangeFiles,
	})
	if len(comparison.NewViolations) == 0 {
		reason := "0 new violations beyond baseline"
		if len(comparison.SeededViolations) > 0 {
			reason = fmt.Sprintf("0 new violations beyond baseline; %d violations seeded due to explicit rule-set change", len(comparison.SeededViolations))
		}
		reason += warningSuffix
		return StepResult{StepName: StepBaselineComparison, Status: "pass", Violations: []Violation{}, NewViolations: []Violation{}, FixedViolations: comparison.FixedViolations, SeededViolations: comparison.SeededViolations, Reason: reason}
	}
	return StepResult{StepName: StepBaselineComparison, Status: "fail", Violations: comparison.NewViolations, NewViolations: comparison.NewViolations, FixedViolations: comparison.FixedViolations, SeededViolations: comparison.SeededViolations, Reason: fmt.Sprintf("%d new violations beyond baseline%s", len(comparison.NewViolations), warningSuffix)}
}

func accumulatedViolations(steps []StepResult) []Violation {
	violations := []Violation{}
	for _, step := range steps {
		if step.StepName == StepBaselineComparison || step.StepName == StepWaiverResolution || step.StepName == StepLedgerIntegrity {
			continue
		}
		violations = append(violations, step.Violations...)
	}
	if violations == nil {
		return []Violation{}
	}
	return violations
}

// ExitCode determines the exit code from a GateResult.
// 2 if configErr != nil, 1 if any step failed, 0 otherwise.
func ExitCode(result GateResult, configErr error) int {
	if configErr != nil {
		return 2
	}
	if !result.Pass {
		return 1
	}
	return 0
}
