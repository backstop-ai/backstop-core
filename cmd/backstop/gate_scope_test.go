package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

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
	// Build the scope via the real constructor so its unexported fileSet is
	// populated — packValidatorStep now applies scope.FilterViolations (ISSUE-070),
	// and Contains() keys on fileSet. A raw struct literal leaves fileSet nil, so
	// Contains would drop every finding. File mode takes the explicit files without a
	// git diff (Diff mode ignores the files arg and shells git, which a TempDir lacks);
	// the FilterViolations path is mode-agnostic (both File and Diff filter via fileSet),
	// so this exercises the same filter, plus the diff-scope THREADING assertions below.
	scope, scopeErr := gate.ComputeGateScope(projectRoot, gate.GateScopeModeFile, []string{changedTracked, changedUntracked})
	if scopeErr != nil {
		t.Fatal(scopeErr)
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
	if gotScope.Mode != gate.GateScopeModeFile {
		t.Errorf("engine received scope mode %q, want file", gotScope.Mode)
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

// gitInitCommitAll initializes a git repo in dir and commits every file already on
// disk, so a subsequently-planted untracked file is the SOLE entry the diff-scope
// resolver appends via `git ls-files --others`.
func gitInitCommitAll(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test User"},
		{"add", "-A"},
		{"commit", "-m", "baseline"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// newDiffScopedPackGateProject builds a projectRoot with backstop.yml + a minimal
// installed pack (so buildGateSteps builds the pack_engines step) inside a real git
// repo, commits the baseline, then plants an UNTRACKED changed file so
// ComputeGateScope(diff) yields a genuine GateScopeModeDiff scope containing ONLY
// that changed file. The dispatchPackEnginesFn seam stands in for the engine, so the
// pack only needs to load; the scope's populated fileSet is what packValidatorStep's
// FilterViolations keys on.
func newDiffScopedPackGateProject(t *testing.T, changedFile string) (string, *gate.GateScope) {
	t.Helper()
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(
		"project: p\nlanguage: go\npacks:\n  org/pack: \"1.0.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
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
	gitInitCommitAll(t, projectRoot)
	// Plant the changed file AFTER the baseline commit so it is the sole untracked
	// entry the diff resolver appends.
	if err := os.WriteFile(filepath.Join(projectRoot, changedFile), []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	scope, err := gate.ComputeGateScope(projectRoot, gate.GateScopeModeDiff, nil)
	if err != nil {
		t.Fatalf("ComputeGateScope diff: %v", err)
	}
	if scope.Mode != gate.GateScopeModeDiff {
		t.Fatalf("expected a diff-mode scope, got %q", scope.Mode)
	}
	if !scope.Contains(changedFile) {
		t.Fatalf("the changed file %q must be in the diff scope; got %#v", changedFile, scope.Files)
	}
	return projectRoot, scope
}

// setDispatchSeam overrides the dispatchPackEnginesFn seam to return a fixed
// violation set (restored on cleanup), so the pack_engines step's post-dispatch
// diff-scope FILTER is exercised hermetically without a live engine.
func setDispatchSeam(t *testing.T, violations []gate.Violation) {
	t.Helper()
	orig := dispatchPackEnginesFn
	t.Cleanup(func() { dispatchPackEnginesFn = orig })
	dispatchPackEnginesFn = func(_ []*pack.Manifest, _, _ string, _ *gate.GateScope, _ check.CommandRunner) ([]gate.Violation, error) {
		return append([]gate.Violation(nil), violations...), nil
	}
}

// runPackEnginesStep builds the gate steps and returns the pack_engines StepResult.
func runPackEnginesStep(t *testing.T, projectRoot string, scope *gate.GateScope) gate.StepResult {
	t.Helper()
	steps := buildGateSteps(projectRoot, scope)
	for _, step := range steps {
		res := step(context.Background())
		if res.StepName == "pack_engines" {
			return res
		}
	}
	t.Fatal("expected a pack_engines step in the gate step list")
	return gate.StepResult{}
}

// TestGate_PackEngines_DiffScope_FiltersOutOfScopeNonExemptViolation (CLM-001,
// CLM-002 at the step layer): the packValidatorStep closure filters project-wide
// NON-exempt engine violations on files NOT in the change unit. With ONLY
// out-of-scope non-exempt findings (the changed file is clean) — including one whose
// File arrives in "./"-prefixed textual form — the step drops them all and PASSES.
func TestGate_PackEngines_DiffScope_FiltersOutOfScopeNonExemptViolation(t *testing.T) {
	changed := "changed.go"
	projectRoot, scope := newDiffScopedPackGateProject(t, changed)

	outOfScope := "pkg/untouched/other.go"
	dotSlashOut := "./pkg/untouched/dotslash.go" // "./"-prefixed out-of-scope form (CLM-002)
	setDispatchSeam(t, []gate.Violation{
		{Rule: "org/pack/no-foo", File: outOfScope, Message: "foo on untouched file", Severity: "error", SourcePack: "org/pack"},
		{Rule: "org/pack/no-foo", File: dotSlashOut, Message: "foo on ./ untouched file", Severity: "error", SourcePack: "org/pack"},
	})

	res := runPackEnginesStep(t, projectRoot, scope)
	for _, v := range res.Violations {
		if v.File == outOfScope || v.File == dotSlashOut {
			t.Errorf("out-of-scope non-exempt violation leaked past the diff-scope filter: %#v", v)
		}
	}
	if res.Status != "pass" {
		t.Errorf("with only out-of-scope non-exempt findings (all filtered), the pack_engines step must PASS; got status=%q violations=%#v", res.Status, res.Violations)
	}
}

// TestGate_PackEngines_DiffScope_RedsOnChangedFileViolation (CLM-004): a non-exempt
// violation on a CHANGED (in-scope) file survives the filter and REDs the step —
// you-touch-it-you-fix-it. No-vacuous-green: real in-scope findings are never dropped.
func TestGate_PackEngines_DiffScope_RedsOnChangedFileViolation(t *testing.T) {
	changed := "changed.go"
	projectRoot, scope := newDiffScopedPackGateProject(t, changed)

	setDispatchSeam(t, []gate.Violation{
		{Rule: "org/pack/no-foo", File: changed, Message: "foo on the changed file", Severity: "error", SourcePack: "org/pack"},
	})

	res := runPackEnginesStep(t, projectRoot, scope)
	found := false
	for _, v := range res.Violations {
		if v.File == changed {
			found = true
		}
	}
	if !found {
		t.Errorf("the in-scope changed-file violation must survive the filter (you-touch-it-you-fix-it); violations=%#v", res.Violations)
	}
	if res.Status != "fail" {
		t.Errorf("an in-scope changed-file violation must RED the pack_engines step; got status=%q", res.Status)
	}
}

// TestGate_PackEngines_DiffScope_ExemptProjectWideViolationStillReds (CLM-005): a
// ProjectWide==true (go-build/exempt) violation on an OUT-of-scope, unchanged file is
// RETAINED and reds the step — the exempt path is not regressed, an unchanged-file
// build break still fails.
func TestGate_PackEngines_DiffScope_ExemptProjectWideViolationStillReds(t *testing.T) {
	changed := "changed.go"
	projectRoot, scope := newDiffScopedPackGateProject(t, changed)

	exemptFile := "pkg/unchanged/build_break.go"
	setDispatchSeam(t, []gate.Violation{
		{Rule: "backstop/go-toolchain/go-build", File: exemptFile, Message: "undefined: Frobnicate", Severity: "error", SourcePack: "backstop/go-toolchain", ProjectWide: true},
	})

	res := runPackEnginesStep(t, projectRoot, scope)
	found := false
	for _, v := range res.Violations {
		if v.File == exemptFile {
			found = true
		}
	}
	if !found {
		t.Errorf("an out-of-scope ProjectWide (exempt, e.g. go-build) violation must be RETAINED — the exempt path must not regress; violations=%#v", res.Violations)
	}
	if res.Status != "fail" {
		t.Errorf("an unchanged-file build break (ProjectWide=true) must still RED the gate; got status=%q", res.Status)
	}
}

// TestGate_PackEngines_DiffScope_FilterKeyedStructurallyNotByToolName (CLM-003): the
// filter keys on the STRUCTURAL ProjectWide field + scope membership, NOT on a
// tool/rule-name string. A differently-named (non-"golangci") out-of-scope non-exempt
// engine's violation is ALSO filtered, while an out-of-scope ProjectWide==true
// violation from yet ANOTHER engine name is retained — differentiation is by the
// ProjectWide field, never the engine name.
func TestGate_PackEngines_DiffScope_FilterKeyedStructurallyNotByToolName(t *testing.T) {
	changed := "changed.go"
	projectRoot, scope := newDiffScopedPackGateProject(t, changed)

	golangciOut := "pkg/a/lint_a.go"
	otherEngineOut := "pkg/b/lint_b.go"
	exemptOut := "pkg/c/build_c.go"
	setDispatchSeam(t, []gate.Violation{
		{Rule: "org/golangci/errcheck", File: golangciOut, Message: "unchecked error", Severity: "error", SourcePack: "org/golangci"},
		{Rule: "acme/other-engine/some-rule", File: otherEngineOut, Message: "some finding", Severity: "error", SourcePack: "acme/other-engine"},
		{Rule: "vendor/builder/build", File: exemptOut, Message: "build break", Severity: "error", SourcePack: "vendor/builder", ProjectWide: true},
	})

	res := runPackEnginesStep(t, projectRoot, scope)
	var sawGolangci, sawOther, sawExempt bool
	for _, v := range res.Violations {
		switch v.File {
		case golangciOut:
			sawGolangci = true
		case otherEngineOut:
			sawOther = true
		case exemptOut:
			sawExempt = true
		}
	}
	if sawGolangci {
		t.Errorf("the out-of-scope golangci-shaped non-exempt violation must be filtered; violations=%#v", res.Violations)
	}
	if sawOther {
		t.Errorf("the out-of-scope DIFFERENTLY-NAMED non-exempt violation must ALSO be filtered — the filter keys on the structural ProjectWide field, not a tool/rule name; violations=%#v", res.Violations)
	}
	if !sawExempt {
		t.Errorf("the out-of-scope ProjectWide (exempt) violation must be retained regardless of its engine name; violations=%#v", res.Violations)
	}
}
