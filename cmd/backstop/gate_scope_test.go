package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
)

// recordingRunner is a check.CommandRunner that records every invocation
// (command name + args) and returns empty output (a clean, finding-free run).
// It never shells out to a live tool.
type recordingRunner struct {
	calls []recordedCall
}

type recordedCall struct {
	name string
	args []string
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, recordedCall{name: name, args: append([]string(nil), args...)})
	return nil, nil
}

// RunStdout records the call like Run; the recordingRunner only needs to
// satisfy the CommandRunner interface for executors that call RunStdout.
func (r *recordingRunner) RunStdout(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, recordedCall{name: name, args: append([]string(nil), args...)})
	return nil, nil
}

func (r *recordingRunner) callsFor(name string) []recordedCall {
	var out []recordedCall
	for _, c := range r.calls {
		if c.name == name {
			out = append(out, c)
		}
	}
	return out
}

// TestGate_PackEngines_DiffScopeExcludesUntouchedFindings (CLM-006/CLM-007) pins
// the GATE-WIRING integration seam: the packValidatorStep closure built by
// buildGateSteps must thread the activeScope through to dispatchPackEngines so a
// rule-fed findings engine pointed at projectRoot would report untouched-file
// findings, but pointed at the diff scope reports only in-scope findings.
//
// The dispatchPackEnginesFn seam is overridden to stand in for the real engine:
// it simulates what semgrep/ast-grep do — emit a finding ONLY for the files it
// is actually pointed at (scope.Files when scoped, the whole repo otherwise).
// The assertion is that the gate passes the narrow diff scope, so the untouched
// file's finding can never be produced, while the in-scope changed file
// (including an untracked one) is still reported (no vacuous green).
func TestGate_PackEngines_DiffScopeExcludesUntouchedFindings(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(
		"project: p\nlanguage: go\npacks:\n  org/pack: \"1.0.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Install a minimal valid pack so loadInstalledPacks succeeds and the gate
	// builds the pack_engines step. The seam override below stands in for the
	// engine run, so the pack only needs to load.
	packRoot := filepath.Join(projectRoot, ".backstop", "packs", "org", "pack")
	if err := os.MkdirAll(filepath.Join(packRoot, "semgrep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packRoot, "semgrep", "rule.yml"), []byte("rules: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `name: org/pack
version: 1.0.0
language: go
archetype: enforcement
description: Pack with a rule-fed semgrep engine
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: no-foo
        standard: standards/go/no-foo.standard.md
        rule_path: semgrep/rule.yml
        risk_class: security
        engine: semgrep
        claims:
          - id: c-no-foo
            text: No foo.
            fixtures:
              positive:
                - fixtures/positive.go
              negative:
                - fixtures/negative.go
`
	if err := os.WriteFile(filepath.Join(packRoot, "pack.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	// The diff scope: one tracked changed file plus one untracked changed file
	// (CLM-007 parity). An untouched file is deliberately NOT in scope. Names are
	// taken from this map (not package-level consts) so the test owns no global
	// mutable state.
	files := map[string]string{
		"changedTracked":   "changed.go",
		"changedUntracked": "brand_new.go",
		"untouched":        "untouched.go",
	}
	changedTracked := files["changedTracked"]
	changedUntracked := files["changedUntracked"]
	untouched := files["untouched"]
	scope := &gate.GateScope{
		Mode:        gate.GateScopeModeDiff,
		Files:       []string{changedTracked, changedUntracked},
		ProjectRoot: projectRoot,
	}

	// Override the dispatch seam to simulate a rule-fed engine: it produces a
	// finding for each file it is POINTED AT. When the gate threads the diff
	// scope, scope.Files are the only targets, so the untouched file is never a
	// target and never produces a finding. If the wiring failed to pass the
	// scope (nil), the engine would scan the whole repo and surface the untouched
	// finding — failing the exclusion assertion.
	var gotScope *gate.GateScope
	orig := dispatchPackEnginesFn
	t.Cleanup(func() { dispatchPackEnginesFn = orig })
	dispatchPackEnginesFn = func(_ []*pack.Manifest, _, root string, s *gate.GateScope, _ check.CommandRunner) ([]gate.Violation, error) {
		gotScope = s
		// Targets the engine is pointed at: the in-scope changed files when a diff
		// scope is present, else the whole repo (which would include untouched.go).
		var targets []string
		if s == nil || s.Mode == gate.GateScopeModeAll {
			targets = []string{changedTracked, changedUntracked, untouched}
		} else {
			targets = append(targets, s.Files...)
		}
		var vs []gate.Violation
		for _, f := range targets {
			vs = append(vs, gate.Violation{
				Rule:       "org/pack/no-foo",
				File:       f,
				Message:    "foo found in " + f,
				Severity:   "error",
				SourcePack: "org/pack",
			})
		}
		return vs, nil
	}

	steps := buildGateSteps(projectRoot, scope)
	var packResult *gate.StepResult
	for _, step := range steps {
		res := step(context.Background())
		if res.StepName == "pack_engines" {
			r := res
			packResult = &r
		}
	}
	if packResult == nil {
		t.Fatal("expected a pack_engines step in the gate step list")
	}

	// The diff scope must have reached the engine through the gate wiring.
	if gotScope == nil {
		t.Fatal("activeScope did not reach dispatchPackEngines — gate wiring dropped the scope")
	}
	if gotScope.Mode != gate.GateScopeModeDiff {
		t.Errorf("engine received scope mode %q, want diff", gotScope.Mode)
	}

	// No out-of-scope untouched-file finding may appear.
	for _, v := range packResult.Violations {
		if v.File == untouched {
			t.Errorf("untouched out-of-scope finding leaked through the gate: %#v", v)
		}
	}

	// No-vacuous-green: the in-scope changed files (including the untracked one)
	// MUST still be reported.
	got := map[string]bool{}
	for _, v := range packResult.Violations {
		got[v.File] = true
	}
	if !got[changedTracked] {
		t.Errorf("in-scope tracked changed file finding was dropped; violations=%#v", packResult.Violations)
	}
	if !got[changedUntracked] {
		t.Errorf("in-scope UNTRACKED changed file finding was dropped (CLM-007); violations=%#v", packResult.Violations)
	}
}
