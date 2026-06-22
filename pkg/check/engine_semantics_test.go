package check

import (
	"context"
	"strings"
	"testing"
)

// SPEC-034 cutover: these engine-semantics tests previously drove the bespoke
// lint/build/test executors as vehicles. Those executors were deleted, but the
// Engine.RunPasses semantics under test (timeout-surfacing, skip-with-warning,
// no-short-circuit) live in the engine, not the executors, and survive. They are
// re-driven here through the SURVIVING generic commandExecutor, plus a small
// test-only unavailable executor.

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

// RunStdout shares the cancellation behavior with Run so the fake satisfies the
// CommandRunner interface.
func (r *cancelingRunner) RunStdout(ctx context.Context, name string, args ...string) ([]byte, error) {
	return r.Run(ctx, name, args...)
}

// unavailableExecutor is a test-only PassExecutor that reports itself
// unavailable, standing in for a tool missing from PATH so the engine's
// skip-with-warning path can be exercised without a deleted bespoke executor.
type unavailableExecutor struct {
	pass   CheckType
	reason string
}

func (e *unavailableExecutor) Execute(context.Context, []string) (*PassResult, error) {
	return &PassResult{Pass: e.pass}, nil
}

func (e *unavailableExecutor) IsAvailable() (bool, string) { return false, e.reason }

// regexLinesExecutor builds a generic commandExecutor for the given pass that
// runs `tool` and parses generic regex-lines output — a stand-in for a real Go
// toolchain pass in these engine-semantics tests.
func regexLinesExecutor(pass CheckType, runner CommandRunner) PassExecutor {
	parser, _ := lookupParser("regex-lines")
	return &commandExecutor{pass: pass, command: "tool", parser: parser, scopeKind: ScopeKindFileArgs, runner: runner}
}

// TestCodeCheck_Executors_EntryContextCancelled verifies that the surviving
// generic commandExecutor returns ctx.Err() immediately when the context is
// already cancelled at entry, without invoking the runner.
func TestCodeCheck_Executors_EntryContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner := &fakeRunner{outputs: map[string][]byte{}}

	ex := regexLinesExecutor(CheckTypeLint, runner)
	if _, err := ex.Execute(ctx, []string{"a.go"}); err == nil {
		t.Errorf("%T.Execute should return ctx error when cancelled at entry", ex)
	}
	if len(runner.calls) != 0 {
		t.Errorf("no runner call expected on entry-cancellation, got %d", len(runner.calls))
	}
}

// TestCodeCheck_Executors_ContextCancellationSurfacesTimeout verifies that when
// the context is cancelled, running the executors through Engine.RunPasses
// surfaces a timeout Violation ("timeout: <pass> pass cancelled") plus the
// "execution timed out during <pass> pass" warning. A real Execute body must not
// swallow ctx.Err(): the cancelingRunner trips cancellation DURING the lint
// Execute, so the executor's own post-Run ctx.Err() check (not just the engine's
// top-of-loop guard) must return the error for the engine to render the timeout.
// (CLM-007)
func TestCodeCheck_Executors_ContextCancellationSurfacesTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := &cancelingRunner{cancel: cancel, output: []byte("")}

	engine := &Engine{
		Executors: map[CheckType]PassExecutor{
			CheckTypeLint:    regexLinesExecutor(CheckTypeLint, runner),
			CheckTypeBuild:   regexLinesExecutor(CheckTypeBuild, runner),
			CheckTypeTest:    regexLinesExecutor(CheckTypeTest, runner),
			CheckTypeFindings: regexLinesExecutor(CheckTypeFindings, runner),
		},
		Manifest: defaultManifest(),
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

// TestCodeCheck_Executors_UnavailableToolSkipsWithWarning verifies that when the
// lint tool is unavailable, the lint pass is Skipped with the skip reason and a
// "<pass> skipped: <msg>" warning, while later passes (build/test/semgrep) still
// run — no short-circuit. (CLM-007)
func TestCodeCheck_Executors_UnavailableToolSkipsWithWarning(t *testing.T) {
	runner := &fakeRunner{outputs: map[string][]byte{
		"tool": []byte(""), // build/test/semgrep generic passes: clean
	}}
	executors := map[CheckType]PassExecutor{
		CheckTypeLint:    &unavailableExecutor{pass: CheckTypeLint, reason: "golangci-lint not found on PATH"},
		CheckTypeBuild:   regexLinesExecutor(CheckTypeBuild, runner),
		CheckTypeTest:    regexLinesExecutor(CheckTypeTest, runner),
		CheckTypeFindings: regexLinesExecutor(CheckTypeFindings, runner),
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
		t.Errorf("lint pass should be skipped when the tool is unavailable")
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
	for _, ct := range []CheckType{CheckTypeBuild, CheckTypeTest, CheckTypeFindings} {
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
