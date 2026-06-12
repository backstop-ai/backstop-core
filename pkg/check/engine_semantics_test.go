package check

import (
	"context"
	"strings"
	"testing"
)

// realExecutorsWithRunner builds the four real executors wired to a fake
// CommandRunner (and fake SemgrepEnsurer) so engine semantics can be exercised
// against real Execute bodies without shelling out to live tools.
func realExecutorsWithRunner(runner CommandRunner) map[CheckType]PassExecutor {
	ensurer := &mockSemgrepEnsurer{}
	return map[CheckType]PassExecutor{
		CheckTypeLint:    &lintExecutor{runner: runner},
		CheckTypeBuild:   &buildExecutor{runner: runner},
		CheckTypeTest:    &testExecutor{runner: runner},
		CheckTypeSemgrep: &semgrepExecutor{runner: runner, ensurer: ensurer},
	}
}

// cancelingRunner cancels the supplied context the first time Run is invoked,
// then returns the canned output. This forces a real executor's post-Run
// ctx.Err() check to fire — exercising that the real Execute body surfaces
// cancellation rather than swallowing it.
type cancelingRunner struct {
	cancel context.CancelFunc
	output []byte
	calls  int
}

func (r *cancelingRunner) Run(_ context.Context, _ string, _ ...string) ([]byte, error) {
	r.calls++
	if r.cancel != nil {
		r.cancel()
	}
	return r.output, nil
}

// TestCodeCheck_Executors_ContextCancellationSurfacesTimeout verifies that when
// the context is cancelled, running the real executors through
// Engine.RunPasses surfaces a timeout Violation ("timeout: <pass> pass
// cancelled") plus the "execution timed out during <pass> pass" warning,
// unchanged from stub-era engine behavior. A real Execute body must not swallow
// ctx.Err(): the cancelingRunner trips cancellation DURING the lint Execute, so
// the executor's own post-Run ctx.Err() check (not just the engine's
// top-of-loop guard) must return the error for the engine to render the
// timeout. (CLM-007)
func TestCodeCheck_Executors_ContextCancellationSurfacesTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := &cancelingRunner{cancel: cancel, output: []byte(`{"Issues":[]}`)}

	engine := &Engine{
		Executors: realExecutorsWithRunner(runner),
		Manifest:  defaultManifest(),
	}

	result, err := engine.RunPasses(ctx, []string{"main.go"})
	if err != nil {
		t.Fatalf("RunPasses returned error: %v", err)
	}

	// The lint executor must have actually been invoked (it ran, then ctx was
	// cancelled mid-run) — proving the real body, not the engine's pre-loop
	// guard, drove the cancellation path.
	if runner.calls == 0 {
		t.Fatal("expected the lint executor to invoke the runner before cancellation surfaced")
	}

	// A timeout violation must be present.
	var sawTimeoutViolation bool
	for _, v := range result.AllViolations() {
		if strings.HasPrefix(v.Message, "timeout:") && strings.Contains(v.Message, "cancelled") {
			sawTimeoutViolation = true
		}
	}
	if !sawTimeoutViolation {
		t.Errorf("expected a timeout violation, got violations: %+v", result.AllViolations())
	}

	// The timeout warning must be present.
	var sawTimeoutWarning bool
	for _, w := range result.Warnings {
		if strings.Contains(w, "execution timed out during") {
			sawTimeoutWarning = true
		}
	}
	if !sawTimeoutWarning {
		t.Errorf("expected an 'execution timed out during ...' warning, got: %v", result.Warnings)
	}
}

// TestCodeCheck_Executors_UnavailableToolSkipsWithWarning verifies that when
// the lint tool is unavailable (golangci-lint missing from PATH), the lint pass
// is Skipped with the skip reason and a "<pass> skipped: <msg>" warning, while
// later passes (build/test/semgrep) still run — no short-circuit. (CLM-007)
func TestCodeCheck_Executors_UnavailableToolSkipsWithWarning(t *testing.T) {
	// Make golangci-lint unavailable: emptying PATH makes the real
	// lintExecutor.IsAvailable (findExecutable on PATH) report it missing.
	t.Setenv("PATH", "")

	runner := &fakeRunner{outputs: map[string][]byte{
		"go":      []byte(""),         // go build / go test: clean
		"semgrep": []byte(`{"results":[]}`), // semgrep clean (binary path "" maps via ensurer)
	}}
	// Wire the semgrep executor to a fake ensurer returning a path the fake
	// runner produces clean output for.
	executors := map[CheckType]PassExecutor{
		CheckTypeLint:    &lintExecutor{runner: runner},
		CheckTypeBuild:   &buildExecutor{runner: runner},
		CheckTypeTest:    &testExecutor{runner: runner},
		CheckTypeSemgrep: &semgrepExecutor{runner: runner, ensurer: &mockSemgrepEnsurer{fn: func(_, _ string) (string, error) { return "semgrep", nil }}},
	}
	engine := &Engine{Executors: executors, Manifest: defaultManifest()}

	result, err := engine.RunPasses(context.Background(), []string{"main.go"})
	if err != nil {
		t.Fatalf("RunPasses returned error: %v", err)
	}

	lintPR := findPassResult(result, CheckTypeLint)
	if lintPR == nil {
		t.Fatal("no lint pass result")
	}
	if !lintPR.Skipped {
		t.Errorf("lint pass should be skipped when golangci-lint is unavailable")
	}
	if lintPR.SkipReason == "" {
		t.Error("skipped lint pass should carry a skip reason")
	}

	var sawSkipWarning bool
	for _, w := range result.Warnings {
		if strings.Contains(w, "lint skipped:") {
			sawSkipWarning = true
		}
	}
	if !sawSkipWarning {
		t.Errorf("expected a 'lint skipped: ...' warning, got: %v", result.Warnings)
	}

	// No short-circuit: build, test, and semgrep passes must still have run.
	for _, ct := range []CheckType{CheckTypeBuild, CheckTypeTest, CheckTypeSemgrep} {
		pr := findPassResult(result, ct)
		if pr == nil {
			t.Errorf("%v pass result missing — earlier skip must not short-circuit later passes", ct)
			continue
		}
		if pr.Skipped {
			t.Errorf("%v pass should not be skipped after lint skip", ct)
		}
	}
}
