package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// scopeTargetRunner is a CommandRunner that records every RunStdout invocation's
// trailing scan targets (the args after the last input flag value) so a scope
// test can assert exactly which paths a rule-fed findings engine was pointed at.
// It returns canned SARIF keyed by the file the engine was pointed at, letting a
// test prove that an untouched-file finding can only be produced when the engine
// is actually pointed at that file.
type scopeTargetRunner struct {
	// sarifByTarget maps a scan-target path to the SARIF the engine "finds" when
	// pointed at it. Targets not present here contribute no findings.
	sarifByTarget map[string][]byte
	calls         []scopeTargetCall
}

type scopeTargetCall struct {
	name string
	args []string
}

func (r *scopeTargetRunner) Run(context.Context, string, ...string) ([]byte, error) {
	return nil, nil
}

func (r *scopeTargetRunner) RunStdout(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, scopeTargetCall{name: name, args: append([]string(nil), args...)})
	// Synthesize a combined SARIF from whichever targets were passed that have a
	// canned finding. Real semgrep/ast-grep self-filter by the targets they are
	// pointed at, so a finding for an untouched file can ONLY appear when that
	// untouched file is among the scan targets.
	for _, arg := range args {
		if sarif, ok := r.sarifByTarget[arg]; ok {
			return sarif, nil
		}
	}
	return []byte(`{"version":"2.1.0","runs":[]}`), nil
}

// scanTargets returns the trailing positional scan targets of the single
// recorded invocation: the args that follow the last input-flag/value pair.
// For a semgrep binding the shape is `--config <rulefile> <target>...`, so the
// scan targets are everything after the last rule-file value.
func (r *scopeTargetRunner) scanTargets(t *testing.T) []string {
	t.Helper()
	if len(r.calls) != 1 {
		t.Fatalf("expected exactly one engine invocation, got %d: %#v", len(r.calls), r.calls)
	}
	args := r.calls[0].args
	// Find the last input flag (--config / --rule) and take everything after its
	// value as the scan targets.
	lastFlag := -1
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--config" || args[i] == "--rule" {
			lastFlag = i + 1
		}
	}
	return args[lastFlag+1:]
}

// semgrepScopeManifest builds a one-rule semgrep (rule-fed, ScopeKindFileArgs)
// manifest with its rule file written to disk under packsDir, returning the
// manifest and the packs dir. The rule file must exist because resolveRulePath
// fail-louds on a missing declared rule path.
func semgrepScopeManifest(t *testing.T) ([]*pack.Manifest, string) {
	t.Helper()
	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "org", "pack")
	mkDirAll(t, filepath.Join(packRoot, "semgrep"))
	writeFileStr(t, filepath.Join(packRoot, "semgrep", "rule.yml"), "rules: []\n")
	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "no-foo", Engine: "semgrep", RulePath: "semgrep/rule.yml", Standard: "x"},
		}}},
	}}
	return manifests, packsDir
}

// diffScope builds a GateScope in diff mode whose Files are the given
// project-relative paths (the same shape resolveGateScopeDiff yields). An empty
// file list yields a genuinely empty diff scope (the empty-intersection case),
// which must NOT fall back to a whole-repo scan.
func diffScope(projectRoot string, files ...string) *gate.GateScope {
	return &gate.GateScope{
		Mode:        gate.GateScopeModeDiff,
		Files:       append([]string(nil), files...),
		ProjectRoot: projectRoot,
	}
}

// TestPackEngines_DiffScope_FindingsEngineScansChangedFilesOnly (CLM-001):
// a rule-fed findings engine dispatched with a diff scope whose Files are the
// changed files must be pointed at EXACTLY those changed files, never at
// projectRoot.
func TestPackEngines_DiffScope_FindingsEngineScansChangedFilesOnly(t *testing.T) {
	manifests, packsDir := semgrepScopeManifest(t)
	projectRoot := t.TempDir()
	writeFileStr(t, filepath.Join(projectRoot, "changed_a.go"), "package p\n")
	writeFileStr(t, filepath.Join(projectRoot, "changed_b.go"), "package p\n")

	rec := &scopeTargetRunner{}
	scope := diffScope(projectRoot, "changed_a.go", "changed_b.go")

	if _, err := dispatchPackEngines(manifests, packsDir, projectRoot, scope, rec); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}

	targets := rec.scanTargets(t)
	want := map[string]bool{"changed_a.go": true, "changed_b.go": true}
	if len(targets) != len(want) {
		t.Fatalf("expected scan targets to be exactly the changed files %v, got %v", want, targets)
	}
	for _, tgt := range targets {
		if !want[tgt] {
			t.Errorf("unexpected scan target %q; expected only the changed files", tgt)
		}
		if tgt == projectRoot {
			t.Errorf("project root must NOT be a scan target under a diff scope; targets=%v", targets)
		}
	}
}

// TestPackEngines_DiffScope_ExcludesUntouchedFileViolations (CLM-002): with a
// diff scope limited to one changed file, the engine is pointed only at that
// file, so an untouched out-of-scope file's violation can never be produced.
func TestPackEngines_DiffScope_ExcludesUntouchedFileViolations(t *testing.T) {
	manifests, packsDir := semgrepScopeManifest(t)
	projectRoot := t.TempDir()
	writeFileStr(t, filepath.Join(projectRoot, "changed.go"), "package p\n")
	writeFileStr(t, filepath.Join(projectRoot, "untouched.go"), "package p\n")

	changedSarif := []byte(`{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"semgrep","rules":[{"id":"no-foo"}]}},"results":[{"ruleId":"no-foo","message":{"text":"foo on changed file"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"changed.go"}}}]}]}]}`)
	untouchedSarif := []byte(`{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"semgrep","rules":[{"id":"no-foo"}]}},"results":[{"ruleId":"no-foo","message":{"text":"foo on untouched file"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"untouched.go"}}}]}]}]}`)
	rec := &scopeTargetRunner{sarifByTarget: map[string][]byte{
		"changed.go":   changedSarif,
		"untouched.go": untouchedSarif,
	}}
	scope := diffScope(projectRoot, "changed.go")

	violations, err := dispatchPackEngines(manifests, packsDir, projectRoot, scope, rec)
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}

	targets := rec.scanTargets(t)
	for _, tgt := range targets {
		if tgt == "untouched.go" {
			t.Fatalf("untouched out-of-scope file was passed as a scan target: %v", targets)
		}
	}
	for _, v := range violations {
		if v.File == "untouched.go" {
			t.Errorf("violation reported for untouched out-of-scope file: %#v", v)
		}
	}
}

// TestPackEngines_DiffScope_EmptyIntersectionScansNothing (CLM-003): a non-empty
// diff scope with NO files in it means the engine is pointed at nothing — it
// must NOT silently fall back to scanning projectRoot. The full-repo scan is
// reachable only via --all.
func TestPackEngines_DiffScope_EmptyIntersectionScansNothing(t *testing.T) {
	manifests, packsDir := semgrepScopeManifest(t)
	projectRoot := t.TempDir()

	rec := &scopeTargetRunner{}
	// An empty diff scope: present (mode=diff) but with zero files.
	scope := diffScope(projectRoot)

	violations, err := dispatchPackEngines(manifests, packsDir, projectRoot, scope, rec)
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}

	targets := rec.scanTargets(t)
	if len(targets) != 0 {
		t.Fatalf("empty diff scope must yield zero scan targets, got %v", targets)
	}
	for _, tgt := range targets {
		if tgt == projectRoot {
			t.Fatalf("empty diff scope must NOT fall back to projectRoot")
		}
	}
	if len(violations) != 0 {
		t.Errorf("empty diff scope must yield zero findings, got %d: %#v", len(violations), violations)
	}
}

// TestPackEngines_AllScope_RestoresWholeRepoScan (CLM-004): scope==nil AND
// scope.Mode==GateScopeModeAll both restore the whole-repo projectRoot scan,
// the explicit --all escape hatch.
func TestPackEngines_AllScope_RestoresWholeRepoScan(t *testing.T) {
	manifests, packsDir := semgrepScopeManifest(t)
	projectRoot := t.TempDir()

	t.Run("nil scope", func(t *testing.T) {
		rec := &scopeTargetRunner{}
		if _, err := dispatchPackEngines(manifests, packsDir, projectRoot, nil, rec); err != nil {
			t.Fatalf("dispatchPackEngines: %v", err)
		}
		targets := rec.scanTargets(t)
		if len(targets) != 1 || targets[0] != projectRoot {
			t.Fatalf("nil scope must scan projectRoot exactly, got %v", targets)
		}
	})

	t.Run("all scope", func(t *testing.T) {
		rec := &scopeTargetRunner{}
		allScope, err := gate.ComputeGateScope(projectRoot, gate.GateScopeModeAll, nil)
		if err != nil {
			t.Fatalf("ComputeGateScope all: %v", err)
		}
		if _, err := dispatchPackEngines(manifests, packsDir, projectRoot, allScope, rec); err != nil {
			t.Fatalf("dispatchPackEngines: %v", err)
		}
		targets := rec.scanTargets(t)
		if len(targets) != 1 || targets[0] != projectRoot {
			t.Fatalf("GateScopeModeAll must scan projectRoot exactly, got %v", targets)
		}
	})
}

// TestPackEngines_ProjectWideToolchainStaysProjectWide (CLM-005): a
// ScopeKindProjectWide engine with a ProjectTarget keeps targeting "./..." under
// a narrow diff scope — neither projectRoot nor the changed-file list is
// appended, so unchanged-file breakage still fails the gate.
func TestPackEngines_ProjectWideToolchainStaysProjectWide(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-build")
	stubSandboxedRunStdout(t, nil)
	projectRoot := t.TempDir()
	writeFileStr(t, filepath.Join(projectRoot, "changed.go"), "package p\n")
	runner := &fixtureRunner{byCmd: map[string][]byte{"go build": readFixture(t, "go-build-errors.txt")}}

	scope := diffScope(projectRoot, "changed.go")
	if _, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), projectRoot, scope, runner); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected one build invocation, got %d", len(runner.calls))
	}
	call := runner.calls[0]
	for _, a := range call.args {
		if a == projectRoot {
			t.Errorf("project root must NOT be appended to a project-wide toolchain pass; args=%v", call.args)
		}
		if a == "changed.go" {
			t.Errorf("the changed-file list must NOT be appended to a project-wide toolchain pass; args=%v", call.args)
		}
	}
	if !strings.Contains(strings.Join(call.args, " "), "./...") {
		t.Errorf("project-wide build pass must target ./..., got args=%v", call.args)
	}
}

// TestPackEngines_DiffScope_IncludesUntrackedChangedFiles (CLM-007): a diff
// scope whose Files include an untracked changed file (mirroring
// resolveGateScopeDiff's untracked inclusion) must point the engine at that
// untracked file so its findings are still enforced.
func TestPackEngines_DiffScope_IncludesUntrackedChangedFiles(t *testing.T) {
	manifests, packsDir := semgrepScopeManifest(t)
	projectRoot := t.TempDir()
	writeFileStr(t, filepath.Join(projectRoot, "brand_new.go"), "package p\n")

	rec := &scopeTargetRunner{}
	// brand_new.go stands in for an untracked changed file the gate diff resolver
	// appends via `git ls-files --others`.
	scope := diffScope(projectRoot, "brand_new.go")

	if _, err := dispatchPackEngines(manifests, packsDir, projectRoot, scope, rec); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	targets := rec.scanTargets(t)
	found := false
	for _, tgt := range targets {
		if tgt == "brand_new.go" {
			found = true
		}
	}
	if !found {
		t.Fatalf("untracked changed file must be a scan target, got %v", targets)
	}
}

// astGrepLikeScopeManifest builds a one-rule manifest for a config-file findings
// engine (the ast-grep shape after ISSUE-028: InputModeConfigFile,
// ScopeKindFileArgs, no convert) registered under a custom engine name, with its
// pack-shipped config file on disk. It exercises gatherEngineInputs' config-file
// branch (which emits [--config, <resolved path>]) together with the diff-scope
// arg-shaping that applies to any file-args findings engine.
func astGrepLikeScopeManifest(t *testing.T) ([]*pack.Manifest, string) {
	t.Helper()
	orig := engineRegistry
	t.Cleanup(func() { engineRegistry = orig })
	engineRegistry = engine.DefaultRegistry()
	engineRegistry["configfile-lint"] = engine.EngineBinding{
		Command:   "configfile-lint scan",
		InputMode: engine.InputModeConfigFile,
		InputFlag: "--config",
		ScopeKind: engine.ScopeKindFileArgs,
	}

	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "org", "pack")
	mkDirAll(t, filepath.Join(packRoot, "rules", "go"))
	writeFileStr(t, filepath.Join(packRoot, "rules", "go", "sgconfig.yml"), "ruleDirs:\n  - .\n")
	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "r", Engine: "configfile-lint", RulePath: "rules/go/sgconfig.yml", Standard: "x"},
		}}},
	}}
	return manifests, packsDir
}

// TestPackEngines_DiffScope_RuleDirEngineScansChangedFilesOnly (CLM-001 for the
// config-file findings engine class): a config-file findings engine (the ast-grep
// shape after ISSUE-028) under a diff scope is pointed at EXACTLY the changed
// files after its --config input, never at projectRoot. This also exercises
// gatherEngineInputs' config-file branch.
func TestPackEngines_DiffScope_RuleDirEngineScansChangedFilesOnly(t *testing.T) {
	manifests, packsDir := astGrepLikeScopeManifest(t)
	projectRoot := t.TempDir()
	writeFileStr(t, filepath.Join(projectRoot, "changed_a.go"), "package p\n")

	rec := &scopeTargetRunner{}
	scope := diffScope(projectRoot, "changed_a.go")

	if _, err := dispatchPackEngines(manifests, packsDir, projectRoot, scope, rec); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("expected one engine invocation, got %d", len(rec.calls))
	}
	args := rec.calls[0].args
	// The --config input must be present (config-file gathering) AND the scan
	// target must be the changed file, not projectRoot.
	if !containsArg(args, "--config") {
		t.Errorf("config-file engine must receive its --config input; args=%v", args)
	}
	last := args[len(args)-1]
	if last != "changed_a.go" {
		t.Errorf("ast-grep (config-file) engine scan target must be the changed file, got %q (args=%v)", last, args)
	}
	for _, a := range args {
		if a == projectRoot {
			t.Errorf("projectRoot must not be a scan target under a diff scope; args=%v", args)
		}
	}
}

// TestPackEngines_DiffScope_RuleDirEngineAllScopeUsesProjectRoot (CLM-004 for the
// config-file findings engine class): with a nil scope, the config-file engine
// (ast-grep shape) restores the whole-repo projectRoot scan target.
func TestPackEngines_DiffScope_RuleDirEngineAllScopeUsesProjectRoot(t *testing.T) {
	manifests, packsDir := astGrepLikeScopeManifest(t)
	projectRoot := t.TempDir()

	rec := &scopeTargetRunner{}
	if _, err := dispatchPackEngines(manifests, packsDir, projectRoot, nil, rec); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	args := rec.calls[0].args
	if args[len(args)-1] != projectRoot {
		t.Errorf("nil scope must scan projectRoot for an ast-grep (config-file) engine, got %q", args[len(args)-1])
	}
}

// containsArg reports whether args contains target.
func containsArg(args []string, target string) bool {
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}
