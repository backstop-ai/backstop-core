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

// TestCodeCheck_ScopeSemantics_LintFileArgsBuildProjectWide pins CLM-008 against
// the GATE CheckScoped path (not an in-process check.Run): driving
// realCodeChecker.CheckScoped with a MULTI-FILE diff scope must invoke the lint
// command EXACTLY ONCE with ALL scoped files as arguments (not once per file),
// and the build command EXACTLY ONCE project-wide (scoped files NOT appended).
// Asserting against the gate path is load-bearing: it is what fails against the
// current per-file loop and passes only after the loop is removed.
func TestCodeCheck_ScopeSemantics_LintFileArgsBuildProjectWide(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: scope-test\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	rulesDir := filepath.Join(dir, ".backstop", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules: %v", err)
	}
	manifest := `{"rules": [{"extensions": [".go"], "check_types": ["lint", "build", "test"]}]}`
	if err := os.WriteFile(filepath.Join(rulesDir, "routing.manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	// Two scoped source files in the same package.
	files := []string{"a.go", "b.go"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}

	runner := &recordingRunner{}
	checker := &realCodeChecker{
		projectRoot:  dir,
		runnerForTest: runner,
		ensurerForTest: stubEnsurer{},
	}

	scope := &gate.GateScope{Mode: gate.GateScopeModeDiff, Files: files}
	_, err := checker.CheckScoped(context.Background(), scope)
	if err != nil {
		t.Fatalf("CheckScoped: %v", err)
	}

	// (a) lint RUN invoked EXACTLY ONCE with ALL scoped files as args. The
	// executor also issues a `golangci-lint version` probe (ISSUE-006
	// version-aware flag selection); filter to the `run` subcommand so this
	// assertion still pins the scope semantics (one run, all files, not
	// once-per-file) the way the build assertion below filters to `build`.
	var lintRuns []recordedCall
	for _, c := range runner.callsFor("golangci-lint") {
		if len(c.args) > 0 && c.args[0] == "run" {
			lintRuns = append(lintRuns, c)
		}
	}
	if len(lintRuns) != 1 {
		t.Fatalf("golangci-lint run invoked %d times, want exactly 1 (not once per file)", len(lintRuns))
	}
	gotA, gotB := false, false
	for _, a := range lintRuns[0].args {
		if a == "a.go" || filepath.Base(a) == "a.go" {
			gotA = true
		}
		if a == "b.go" || filepath.Base(a) == "b.go" {
			gotB = true
		}
	}
	if !gotA || !gotB {
		t.Errorf("lint args %v must include BOTH scoped files a.go and b.go in one invocation", lintRuns[0].args)
	}

	// (b) build invoked EXACTLY ONCE project-wide; scoped files NOT appended.
	buildCalls := runner.callsFor("go")
	buildRuns := 0
	for _, c := range buildCalls {
		if len(c.args) > 0 && c.args[0] == "build" {
			buildRuns++
			for _, a := range c.args {
				if filepath.Base(a) == "a.go" || filepath.Base(a) == "b.go" {
					t.Errorf("go build args %v wrongly include a scoped file; build is project-wide", c.args)
				}
			}
		}
	}
	if buildRuns != 1 {
		t.Fatalf("go build invoked %d times, want exactly 1 project-wide run (not once per file)", buildRuns)
	}
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
