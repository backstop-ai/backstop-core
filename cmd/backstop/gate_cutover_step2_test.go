package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// gateStepNames runs buildGateSteps over projectRoot with an EMPTY diff scope
// (so scoped steps are cheap no-ops) and collects the StepName of every step in
// the built list — the observable for the Step-2 cutover assertions.
func gateStepNames(t *testing.T, projectRoot string, scope *gate.GateScope) []string {
	t.Helper()
	steps := buildGateSteps(projectRoot, scope)
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		names = append(names, step(context.Background()).StepName)
	}
	return names
}

// emptyDiffScope returns a diff-mode scope with no files, so StepCodeCheckScoped
// (and other scoped steps) short-circuit to a cheap pass without shelling out.
func emptyDiffScope() *gate.GateScope {
	return &gate.GateScope{Mode: gate.GateScopeModeDiff, Files: []string{}}
}

// noToolchainProject writes a minimal Go project root with no toolchain pack —
// the post-cutover step list must contain no code_check Step-2 entry.
func noToolchainProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "backstop.yml"), []byte("project: rt\nlanguage: go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".backstop"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestCutover_NoCodeCheckStepInGateStepList proves the step list built by
// buildGateSteps contains no realCodeChecker / StepCodeCheckScopedFunc Step-2
// entry after the cutover (CLM-001). RED until TASK-014 un-wires it.
func TestCutover_NoCodeCheckStepInGateStepList(t *testing.T) {
	names := gateStepNames(t, noToolchainProject(t), emptyDiffScope())
	for _, n := range names {
		if n == gate.StepCodeCheck {
			t.Fatalf("buildGateSteps still wires a %q Step-2 entry; lint/build/test must run only through dispatchPackEngines after the cutover. steps=%v", gate.StepCodeCheck, names)
		}
	}
}

// TestCutover_LintBuildTestRunThroughDispatchPackEngines proves a gate over a
// project with an installed toolchain pack dispatches its lint/build/test through
// the engine path (pack_engines step), not pkg/check.Run (CLM-002).
func TestCutover_LintBuildTestRunThroughDispatchPackEngines(t *testing.T) {
	// The go-toolchain testdata project has the toolchain pack installed; its
	// step list must include the pack_engines dispatch step and NOT a code_check
	// Step-2 entry.
	root := goToolchainProjectRoot(t)
	names := gateStepNames(t, root, emptyDiffScope())
	hasPackEngines := false
	for _, n := range names {
		if n == "pack_engines" {
			hasPackEngines = true
		}
		if n == gate.StepCodeCheck {
			t.Fatalf("lint/build/test must run through dispatchPackEngines, but a %q Step-2 entry is still wired. steps=%v", gate.StepCodeCheck, names)
		}
	}
	if !hasPackEngines {
		t.Fatalf("expected a pack_engines dispatch step for a project with an installed toolchain pack; steps=%v", names)
	}
}

// TestCutover_NoCheckToEngineImportAndNoParallelDispatcher proves pkg/check does
// NOT import pkg/pack/engine and no second dispatcher is introduced — the gate
// reuses dispatchPackEngines (CLM-003). Source/import scan.
func TestCutover_NoCheckToEngineImportAndNoParallelDispatcher(t *testing.T) {
	// pkg/check must not import pkg/pack/engine.
	checkDir := filepath.Join(repoRoot(t), "pkg", "check")
	for _, p := range nonTestGoSources(t, checkDir) {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("reading %s: %v", p, err)
		}
		if strings.Contains(string(b), "pkg/pack/engine") {
			t.Errorf("%s imports pkg/pack/engine; pkg/check must stay a disjoint leaf (CLM-003)", p)
		}
	}

	// gate.go must reference dispatchPackEngines and introduce no parallel
	// native dispatcher.
	gateSrc := readFileStr(t, "gate.go")
	if !strings.Contains(gateSrc, "dispatchPackEngines") {
		t.Error("gate.go must route lint/build/test through the existing dispatchPackEngines, not a new dispatcher")
	}
	cmdDir := filepath.Join(repoRoot(t), "cmd", "backstop")
	banned := []string{"dispatchNativeEngines", "dispatchToolchain", "runNativeDispatch", "dispatchGoToolchain", "dispatchLintBuildTest"}
	for _, p := range nonTestGoSources(t, cmdDir) {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("reading %s: %v", p, err)
		}
		for _, fn := range banned {
			if strings.Contains(string(b), "func "+fn) {
				t.Errorf("%s defines a parallel dispatcher %q; the cutover must reuse dispatchPackEngines", p, fn)
			}
		}
	}
}

// TestCutover_NoDualRunOfLintBuildTest proves after the cutover lint/build/test
// run EXACTLY ONCE (through dispatchPackEngines), not through both pkg/check.Run
// and the engine path — no standing dual-run window (CLM-008). Asserting the
// step list has the engine dispatch but no code_check Step-2 entry establishes
// there is a single dispatch path.
func TestCutover_NoDualRunOfLintBuildTest(t *testing.T) {
	root := goToolchainProjectRoot(t)
	names := gateStepNames(t, root, emptyDiffScope())
	codeCheckCount, packEnginesCount := 0, 0
	for _, n := range names {
		switch n {
		case gate.StepCodeCheck:
			codeCheckCount++
		case "pack_engines":
			packEnginesCount++
		}
	}
	if codeCheckCount != 0 {
		t.Fatalf("a code_check Step-2 entry survives alongside the engine path — that is a dual-run window. steps=%v", names)
	}
	if packEnginesCount != 1 {
		t.Fatalf("expected exactly one pack_engines dispatch step (the single lint/build/test path), got %d. steps=%v", packEnginesCount, names)
	}
}
