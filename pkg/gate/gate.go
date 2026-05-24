package gate

import "context"

// Gate orchestrates the nine-step verification kill chain.
type Gate struct {
	steps     []StepFunc
	configErr error
	scope     *GateScope
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
		result := stepFn(ctx)
		results = append(results, result)

		// Check for config error from delegated steps.
		if result.ConfigErr {
			configErrHalt = true
			break
		}
	}

	gateResult := NewGateResultWithScope(results, g.scope)

	if configErrHalt {
		return gateResult, 2
	}

	return gateResult, ExitCode(gateResult, nil)
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
