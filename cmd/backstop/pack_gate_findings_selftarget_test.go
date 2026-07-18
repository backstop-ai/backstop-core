package main

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// SPEC-048 REQ-001 — the project-wide arg-shaping matrix over the REAL
// runFindingsEngine, capturing the dispatched command args via a recording
// check.CommandRunner (capturingRunner). These tests pin every cell of the
// ScopeKind × ProjectTarget matrix so the self-target fix (DEFECT-1) cannot
// regress and does not leak into the go-toolchain "./..." engines, the file-mode
// go-test package-selector edge, or the file-args branch.

// selfTargetManifest is a minimal in-memory manifest for driving runFindingsEngine
// directly: its NormalizedName namespaces produced violations and its (absent)
// convert means requireLintSarifShape runs as a no-op (StrictSarif unset).
func selfTargetManifest() *pack.Manifest {
	return &pack.Manifest{NormalizedName: "test-org/selftarget"}
}

// emptySarifCapturingRunner returns a capturingRunner whose RunStdout yields an
// empty SARIF log (zero findings) so runFindingsEngine's parse succeeds and the
// recorded args are all the test needs.
func emptySarifCapturingRunner() *capturingRunner {
	return &capturingRunner{out: []byte(`{"version":"2.1.0","runs":[]}`)}
}

// argsContain reports whether the recorded args include target.
func argsContain(args []string, target string) bool {
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}

// TestRunFindingsEngine_ProjectWideEmptyTargetSelfTargetsNoRootAppended proves the
// DEFECT-1 fix (CLM-001): a project-wide findings engine with an EMPTY ProjectTarget
// appends NOTHING — the dispatched args carry no projectRoot and no scan target, so
// the engine self-targets. RED pre-fix, where the empty-target engine falls to the
// file-args else branch and gets projectRoot bolted on.
func TestRunFindingsEngine_ProjectWideEmptyTargetSelfTargetsNoRootAppended(t *testing.T) {
	projectRoot := t.TempDir()
	binding := engine.EngineBinding{
		Command:       "fake-tool run",
		InputMode:     engine.InputModeNone,
		ScopeKind:     engine.ScopeKindProjectWide,
		ProjectTarget: "", // self-target: append nothing
	}
	runner := emptySarifCapturingRunner()

	if _, err := runFindingsEngine(selfTargetManifest(), t.TempDir(), projectRoot, nil, binding, nil, runner, newSharedRunCache()); err != nil {
		t.Fatalf("runFindingsEngine: %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("expected exactly one engine invocation, got %d", runner.calls)
	}
	if argsContain(runner.lastArgs, projectRoot) {
		t.Errorf("DEFECT-1: projectRoot must NOT be appended to a project-wide empty-target engine; args=%v", runner.lastArgs)
	}
	// The ONLY args are the command's own leading subcommand ("run") — nothing is
	// bolted on. Any additional arg is a scan target the self-targeting engine must
	// not receive.
	if len(runner.lastArgs) != 1 || runner.lastArgs[0] != "run" {
		t.Errorf("a self-targeting engine must receive NO appended scan target; want args=[run], got %v", runner.lastArgs)
	}
}

// TestRunFindingsEngine_ProjectWideWithTargetAppendsProjectTarget proves the allowed
// cell (CLM-002): a project-wide engine with a NON-EMPTY ProjectTarget ("./...")
// appends that ProjectTarget unchanged, so the go-toolchain engines are UNAFFECTED
// by the self-target change.
func TestRunFindingsEngine_ProjectWideWithTargetAppendsProjectTarget(t *testing.T) {
	projectRoot := t.TempDir()
	// A neutral command (not a baked "go build" literal): the arg-shaping keys on
	// ScopeKind + a non-empty ProjectTarget, NOT the command string — this cell
	// models any go-toolchain-shaped engine (ProjectTarget "./...").
	binding := engine.EngineBinding{
		Command:       "neutral-build run",
		InputMode:     engine.InputModeNone,
		ScopeKind:     engine.ScopeKindProjectWide,
		ProjectTarget: "./...",
	}
	runner := emptySarifCapturingRunner()

	if _, err := runFindingsEngine(selfTargetManifest(), t.TempDir(), projectRoot, nil, binding, nil, runner, newSharedRunCache()); err != nil {
		t.Fatalf("runFindingsEngine: %v", err)
	}
	if !argsContain(runner.lastArgs, "./...") {
		t.Errorf("a project-wide engine with ProjectTarget ./... must append ./... unchanged; args=%v", runner.lastArgs)
	}
	if argsContain(runner.lastArgs, projectRoot) {
		t.Errorf("projectRoot must NOT be appended to a project-wide toolchain pass; args=%v", runner.lastArgs)
	}
}

// TestRunFindingsEngine_FileArgsScopeAppendsScopeFilesUnchanged proves the file-args
// branch is untouched (CLM-003): a non-project-wide rule-fed engine still appends the
// gate's in-scope changed files — the empty-target self-target change did not leak
// into the file-args branch.
func TestRunFindingsEngine_FileArgsScopeAppendsScopeFilesUnchanged(t *testing.T) {
	projectRoot := t.TempDir()
	binding := engine.EngineBinding{
		Command:   "semgrep scan",
		InputMode: engine.InputModeNone,
		ScopeKind: engine.ScopeKindFileArgs,
	}
	scope := &gate.GateScope{
		Mode:  gate.GateScopeModeDiff,
		Files: []string{"pkg/a/x.go", "pkg/b/y.go"},
	}
	runner := emptySarifCapturingRunner()

	if _, err := runFindingsEngine(selfTargetManifest(), t.TempDir(), projectRoot, scope, binding, nil, runner, newSharedRunCache()); err != nil {
		t.Fatalf("runFindingsEngine: %v", err)
	}
	if !argsContain(runner.lastArgs, "pkg/a/x.go") || !argsContain(runner.lastArgs, "pkg/b/y.go") {
		t.Errorf("a file-args engine must append the in-scope changed files; args=%v", runner.lastArgs)
	}
	if argsContain(runner.lastArgs, projectRoot) {
		t.Errorf("with a diff scope, projectRoot must NOT be appended to a file-args engine; args=%v", runner.lastArgs)
	}
}

// TestRunFindingsEngine_FileModeGoTestPackageScopingPreserved proves the SPEC-034
// file-mode package-scoping edge survives the branch restructure (CLM-004): under a
// file-mode scope the native go-test project-wide engine (ProjectTarget "./...",
// PackageScoped) receives its changed file's PACKAGE selector via fileModeTestTarget,
// not projectRoot and not ./....
func TestRunFindingsEngine_FileModeGoTestPackageScopingPreserved(t *testing.T) {
	projectRoot := t.TempDir()
	// A neutral command (not a baked "go test" literal): fileModeTestTarget keys on
	// the DECLARED PackageScoped flag, NOT a "go test" command sniff (SPEC-034 Sharp
	// Edge 5), so this faithfully models the native package-scoped test engine.
	binding := engine.EngineBinding{
		Command:       "neutral-test run",
		InputMode:     engine.InputModeNone,
		ScopeKind:     engine.ScopeKindProjectWide,
		ProjectTarget: "./...",
		PackageScoped: true,
	}
	scope := &gate.GateScope{
		Mode:  gate.GateScopeModeFile,
		Files: []string{"pkg/foo/bar.go"},
	}
	runner := emptySarifCapturingRunner()

	if _, err := runFindingsEngine(selfTargetManifest(), t.TempDir(), projectRoot, scope, binding, nil, runner, newSharedRunCache()); err != nil {
		t.Fatalf("runFindingsEngine: %v", err)
	}
	if !argsContain(runner.lastArgs, "./pkg/foo") {
		t.Errorf("file-mode go-test must receive the changed file's package selector ./pkg/foo; args=%v", runner.lastArgs)
	}
	if argsContain(runner.lastArgs, "./...") {
		t.Errorf("file-mode go-test must be package-scoped, NOT run the whole module ./...; args=%v", runner.lastArgs)
	}
	if argsContain(runner.lastArgs, projectRoot) {
		t.Errorf("projectRoot must NOT be appended to the file-mode go-test engine; args=%v", runner.lastArgs)
	}
}
