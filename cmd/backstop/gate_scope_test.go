package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
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

	// (a) lint invoked EXACTLY ONCE with ALL scoped files as args.
	lintCalls := runner.callsFor("golangci-lint")
	if len(lintCalls) != 1 {
		t.Fatalf("golangci-lint invoked %d times, want exactly 1 (not once per file)", len(lintCalls))
	}
	gotA, gotB := false, false
	for _, a := range lintCalls[0].args {
		if a == "a.go" || filepath.Base(a) == "a.go" {
			gotA = true
		}
		if a == "b.go" || filepath.Base(a) == "b.go" {
			gotB = true
		}
	}
	if !gotA || !gotB {
		t.Errorf("lint args %v must include BOTH scoped files a.go and b.go in one invocation", lintCalls[0].args)
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
