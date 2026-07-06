package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// exemptBinding builds an EngineBinding for the exempt-matrix tests: a
// SARIF-native findings-shaped engine (no Convert, so the runner's stdout is
// parsed directly as SARIF) carrying the given gate_type and the explicit
// ExemptFromScopeFilter value. ScopeKind is project-wide for parity with the Go
// toolchain engines but DECOUPLED from the exempt decision (CLM-017): the exempt
// value alone drives ProjectWide on the engine path.
func exemptBinding(gt engine.GateType, exempt bool) engine.EngineBinding {
	return engine.EngineBinding{
		Command:               "matrix-tool scan",
		InputMode:             engine.InputModeRuleFlags,
		InputFlag:             "--config",
		ScopeKind:             engine.ScopeKindProjectWide,
		ProjectTarget:         "./...",
		GateType:              gt,
		ExemptFromScopeFilter: exempt,
	}
}

// installExemptRegistry installs a DefaultRegistry overlay carrying the named
// custom bindings for the duration of the test, restoring the prior registry on
// cleanup. It returns the registry so a test can introspect declared bindings.
func installExemptRegistry(t *testing.T, bindings map[string]engine.EngineBinding) engine.Registry {
	t.Helper()
	orig := engineRegistry
	t.Cleanup(func() { engineRegistry = orig })
	reg := builtinTestRegistry(t)
	for name, b := range bindings {
		reg[name] = b
	}
	engineRegistry = reg
	return reg
}

// exemptManifest builds a one-rule manifest binding the given custom engine
// name, with a pack-shipped rule file on disk under packsDir so resolveRulePath
// does not fail-loud. Returns the manifest list and the packs dir.
func exemptManifest(t *testing.T, engineName string) ([]*pack.Manifest, string) {
	t.Helper()
	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "org", "pack")
	mkDirAll(t, filepath.Join(packRoot, "rules"))
	writeFileStr(t, filepath.Join(packRoot, "rules", "rule.yml"), "rules: []\n")
	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "r", Engine: engineName, RulePath: "rules/rule.yml", Standard: "x"},
		}}},
	}}
	return manifests, packsDir
}

// sarifForFile returns a minimal SARIF document carrying one finding pinned to
// the given file path and rule id, which the matrix runner feeds so a dispatched
// violation references that (possibly out-of-scope) file.
func sarifForFile(file, ruleID, message string) []byte {
	return []byte(`{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"matrix-tool","rules":[{"id":"` + ruleID + `"}]}},"results":[{"ruleId":"` + ruleID + `","message":{"text":"` + message + `"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"` + file + `"}}}]}]}]}`)
}

// matrixRunner is a CommandRunner whose RunStdout always returns the canned
// SARIF, regardless of args — used to feed a single dispatched finding pinned to
// a chosen file through the real dispatch + scope filter.
type matrixRunner struct {
	sarif []byte
}

func (r *matrixRunner) Run(context.Context, string, ...string) ([]byte, error) { return nil, nil }

func (r *matrixRunner) RunStdout(context.Context, string, ...string) ([]byte, error) {
	return r.sarif, nil
}

// filterThroughGate drives the dispatched violations through the REAL
// pkg/gate/scope.go filterViolations via StepCodeCheckScopedFunc — the genuine
// end-to-end gate scope-filter path (NOT a stamp-site-only assertion). It
// returns the violations that SURVIVE diff-scope filtering for the given scope.
func filterThroughGate(t *testing.T, scope *gate.GateScope, violations []gate.Violation) []gate.Violation {
	t.Helper()
	step := gate.StepCodeCheckScopedFunc(&fixedChecker{violations: violations}, scope)
	res := step(context.Background())
	return res.Violations
}

// fixedChecker is a gate.ScopedCodeChecker that returns a fixed violation set so
// the real filterViolations (invoked inside StepCodeCheckScopedFunc) can be
// exercised end-to-end on engine-dispatched violations.
type fixedChecker struct {
	violations []gate.Violation
}

func (c *fixedChecker) CheckAll(context.Context) ([]gate.Violation, error) {
	return c.violations, nil
}

func (c *fixedChecker) CheckScoped(context.Context, *gate.GateScope) ([]gate.Violation, error) {
	return c.violations, nil
}
