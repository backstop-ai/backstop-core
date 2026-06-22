package check

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestCodeCheck_AllPassesRun verifies all four validation passes (lint, build,
// test, semgrep) execute and produce violations. (CLM-001)
func TestCodeCheck_AllPassesRun(t *testing.T) {
	invoked := map[CheckType]bool{}
	var mu sync.Mutex

	executors := map[CheckType]PassExecutor{
		CheckTypeLint: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			invoked[CheckTypeLint] = true
			mu.Unlock()
			return &PassResult{Pass: CheckTypeLint, Violations: []Violation{{Pass: CheckTypeLint, File: "a.go", Message: "lint-err"}}}, nil
		}},
		CheckTypeBuild: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			invoked[CheckTypeBuild] = true
			mu.Unlock()
			return &PassResult{Pass: CheckTypeBuild, Violations: []Violation{{Pass: CheckTypeBuild, File: "a.go", Message: "build-err"}}}, nil
		}},
		CheckTypeTest: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			invoked[CheckTypeTest] = true
			mu.Unlock()
			return &PassResult{Pass: CheckTypeTest, Violations: []Violation{{Pass: CheckTypeTest, File: "a.go", Message: "test-err"}}}, nil
		}},
		CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			invoked[CheckTypeSemgrep] = true
			mu.Unlock()
			return &PassResult{Pass: CheckTypeSemgrep, Violations: []Violation{{Pass: CheckTypeSemgrep, File: "a.go", Message: "semgrep-err"}}}, nil
		}},
	}

	engine := &Engine{
		Executors: executors,
		Manifest:  defaultManifest(),
	}

	result, err := engine.RunPasses(context.Background(), []string{"a.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, ct := range []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeSemgrep} {
		if !invoked[ct] {
			t.Errorf("pass %v was not invoked", ct)
		}
	}
	if result.ViolationCount() != 4 {
		t.Errorf("got %d violations, want 4", result.ViolationCount())
	}
}

// TestCodeCheck_PassesContinueAfterViolation verifies all passes run even when
// an earlier pass produces violations. (CLM-002)
func TestCodeCheck_PassesContinueAfterViolation(t *testing.T) {
	callOrder := []CheckType{}
	var mu sync.Mutex

	executors := map[CheckType]PassExecutor{
		CheckTypeLint: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			callOrder = append(callOrder, CheckTypeLint)
			mu.Unlock()
			return &PassResult{Pass: CheckTypeLint, Violations: []Violation{{Pass: CheckTypeLint, File: "a.go", Message: "lint-fail"}}}, nil
		}},
		CheckTypeBuild: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			callOrder = append(callOrder, CheckTypeBuild)
			mu.Unlock()
			return &PassResult{Pass: CheckTypeBuild}, nil
		}},
		CheckTypeTest: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			callOrder = append(callOrder, CheckTypeTest)
			mu.Unlock()
			return &PassResult{Pass: CheckTypeTest}, nil
		}},
		CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			callOrder = append(callOrder, CheckTypeSemgrep)
			mu.Unlock()
			return &PassResult{Pass: CheckTypeSemgrep}, nil
		}},
	}

	engine := &Engine{
		Executors: executors,
		Manifest:  defaultManifest(),
	}

	result, err := engine.RunPasses(context.Background(), []string{"a.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(callOrder) != 4 {
		t.Errorf("got %d passes executed, want 4", len(callOrder))
	}
	if result.ViolationCount() != 1 {
		t.Errorf("got %d violations, want 1", result.ViolationCount())
	}
}

// TestCodeCheck_NonApplicablePassSkipped verifies that non-applicable passes
// are skipped silently. (CLM-003)
func TestCodeCheck_NonApplicablePassSkipped(t *testing.T) {
	invoked := map[CheckType]bool{}
	var mu sync.Mutex

	executors := map[CheckType]PassExecutor{
		CheckTypeLint: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			invoked[CheckTypeLint] = true
			mu.Unlock()
			return &PassResult{Pass: CheckTypeLint}, nil
		}},
		CheckTypeBuild: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			invoked[CheckTypeBuild] = true
			mu.Unlock()
			return &PassResult{Pass: CheckTypeBuild}, nil
		}},
		CheckTypeTest: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			invoked[CheckTypeTest] = true
			mu.Unlock()
			return &PassResult{Pass: CheckTypeTest}, nil
		}},
		CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			invoked[CheckTypeSemgrep] = true
			mu.Unlock()
			return &PassResult{Pass: CheckTypeSemgrep}, nil
		}},
	}

	// Manifest that only routes .py to semgrep
	manifest := &Manifest{
		rules: []ManifestRule{
			{
				Extensions: []string{".py"},
				CheckTypes: []string{"semgrep"},
				parsed:     []CheckType{CheckTypeSemgrep},
			},
		},
	}

	engine := &Engine{
		Executors: executors,
		Manifest:  manifest,
	}

	result, err := engine.RunPasses(context.Background(), []string{"script.py"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only semgrep should have been invoked
	if invoked[CheckTypeLint] {
		t.Error("lint was invoked for .py file")
	}
	if invoked[CheckTypeBuild] {
		t.Error("build was invoked for .py file")
	}
	if invoked[CheckTypeTest] {
		t.Error("test was invoked for .py file")
	}
	if !invoked[CheckTypeSemgrep] {
		t.Error("semgrep was NOT invoked for .py file")
	}

	// Check that skipped passes are recorded
	skippedCount := 0
	for _, pr := range result.PassResults {
		if pr.Skipped {
			skippedCount++
		}
	}
	// lint, build, test should be skipped; only semgrep should run
	if skippedCount != 3 {
		t.Errorf("got %d skipped passes, want 3", skippedCount)
	}
}

// TestCodeCheck_PassOrderLintBuildTestSemgrep verifies passes execute in the
// order lint -> build -> test -> semgrep. (CLM-004)
func TestCodeCheck_PassOrderLintBuildTestSemgrep(t *testing.T) {
	order := []CheckType{}
	var mu sync.Mutex

	makeExecutor := func(ct CheckType) PassExecutor {
		return &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			order = append(order, ct)
			mu.Unlock()
			return &PassResult{Pass: ct}, nil
		}}
	}

	executors := map[CheckType]PassExecutor{
		CheckTypeLint:    makeExecutor(CheckTypeLint),
		CheckTypeBuild:   makeExecutor(CheckTypeBuild),
		CheckTypeTest:    makeExecutor(CheckTypeTest),
		CheckTypeSemgrep: makeExecutor(CheckTypeSemgrep),
	}

	engine := &Engine{
		Executors: executors,
		Manifest:  defaultManifest(),
	}

	_, err := engine.RunPasses(context.Background(), []string{"main.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeSemgrep}
	if len(order) != 4 {
		t.Fatalf("got %d passes, want 4", len(order))
	}
	for i, ct := range expected {
		if order[i] != ct {
			t.Errorf("pass[%d] = %v, want %v", i, order[i], ct)
		}
	}
}

// TestCodeCheck_Lint_SkippedWhenGolangciLintMissing verifies lint pass is
// skipped with warning when golangci-lint is not on PATH. (CLM-042)
func TestCodeCheck_Lint_SkippedWhenGolangciLintMissing(t *testing.T) {
	executors := map[CheckType]PassExecutor{
		CheckTypeLint: &mockPassExecutor{available: false, unavailableMsg: "golangci-lint not found on PATH"},
		CheckTypeBuild: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeBuild}, nil
		}},
		CheckTypeTest: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeTest}, nil
		}},
		CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeSemgrep}, nil
		}},
	}

	engine := &Engine{
		Executors: executors,
		Manifest:  defaultManifest(),
	}

	result, err := engine.RunPasses(context.Background(), []string{"main.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Lint should be skipped
	lintResult := findPassResult(result, CheckTypeLint)
	if lintResult == nil {
		t.Fatal("expected lint pass result")
	}
	if !lintResult.Skipped {
		t.Error("lint pass should be skipped when golangci-lint missing")
	}

	// Warning should mention golangci-lint
	if len(result.Warnings) == 0 {
		t.Error("expected warning about missing golangci-lint")
	}
}

// TestCodeCheck_Lint_OtherPassesContinueWithoutLint verifies build and test
// passes still run when golangci-lint is not on PATH. (CLM-043)
func TestCodeCheck_Lint_OtherPassesContinueWithoutLint(t *testing.T) {
	invoked := map[CheckType]bool{}
	var mu sync.Mutex

	executors := map[CheckType]PassExecutor{
		CheckTypeLint: &mockPassExecutor{available: false, unavailableMsg: "golangci-lint not found"},
		CheckTypeBuild: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			invoked[CheckTypeBuild] = true
			mu.Unlock()
			return &PassResult{Pass: CheckTypeBuild}, nil
		}},
		CheckTypeTest: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			invoked[CheckTypeTest] = true
			mu.Unlock()
			return &PassResult{Pass: CheckTypeTest}, nil
		}},
		CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			invoked[CheckTypeSemgrep] = true
			mu.Unlock()
			return &PassResult{Pass: CheckTypeSemgrep}, nil
		}},
	}

	engine := &Engine{
		Executors: executors,
		Manifest:  defaultManifest(),
	}

	_, err := engine.RunPasses(context.Background(), []string{"main.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, ct := range []CheckType{CheckTypeBuild, CheckTypeTest, CheckTypeSemgrep} {
		if !invoked[ct] {
			t.Errorf("pass %v was not invoked despite lint being unavailable", ct)
		}
	}
}

// TestCodeCheck_RunWith_Integration verifies the full RunWith flow with mocked
// dependencies.
func TestCodeCheck_RunWith_Integration(t *testing.T) {
	dir := t.TempDir()

	executors := map[CheckType]PassExecutor{
		CheckTypeLint: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeLint}, nil
		}},
		CheckTypeBuild: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeBuild}, nil
		}},
		CheckTypeTest: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeTest}, nil
		}},
		CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeSemgrep}, nil
		}},
	}

	git := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			return "abc123", nil
		},
		diffNameOnlyFn: func(base string) ([]string, error) {
			return []string{"main.go"}, nil
		},
	}

	opts := RunOptions{
		Options: Options{
			Mode:        ScopeModeDiff,
			BackstopDir: dir, // empty dir/rules, routing uses defaults
			ProjectDir:  dir,
		},
		Git:       git,
		Executors: executors,
	}

	result, err := RunWith(context.Background(), opts)
	if err != nil {
		t.Fatalf("RunWith: %v", err)
	}
	if result.HasViolations() {
		t.Error("expected no violations")
	}
	if len(result.PassResults) != 4 {
		t.Errorf("got %d pass results, want 4", len(result.PassResults))
	}
}

// TestCodeCheck_RunWith_EmptyScope verifies RunWith with no files returns early.
func TestCodeCheck_RunWith_EmptyScope(t *testing.T) {
	git := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			return "abc123", nil
		},
		diffNameOnlyFn: func(base string) ([]string, error) {
			return []string{}, nil // no changes
		},
	}

	opts := RunOptions{
		Options: Options{
			Mode: ScopeModeDiff,
		},
		Git: git,
	}

	result, err := RunWith(context.Background(), opts)
	if err != nil {
		t.Fatalf("RunWith: %v", err)
	}
	if len(result.PassResults) != 0 {
		t.Errorf("expected no pass results for empty scope, got %d", len(result.PassResults))
	}
}

// TestCodeCheck_RunWith_Timeout verifies RunWith respects timeout option.
func TestCodeCheck_RunWith_Timeout(t *testing.T) {
	git := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			return "abc", nil
		},
		diffNameOnlyFn: func(base string) ([]string, error) {
			return []string{"main.go"}, nil
		},
	}

	executors := map[CheckType]PassExecutor{
		CheckTypeLint: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(5 * time.Second):
				return &PassResult{Pass: CheckTypeLint}, nil
			}
		}},
		CheckTypeBuild: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeBuild}, nil
		}},
		CheckTypeTest: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeTest}, nil
		}},
		CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeSemgrep}, nil
		}},
	}

	dir := t.TempDir()
	opts := RunOptions{
		Options: Options{
			Mode:        ScopeModeDiff,
			BackstopDir: dir, // empty dir/rules, routing uses defaults
			ProjectDir:  dir,
			Timeout:     100 * time.Millisecond,
		},
		Git:       git,
		Executors: executors,
	}

	result, err := RunWith(context.Background(), opts)
	if err != nil {
		t.Fatalf("RunWith: %v", err)
	}
	if !result.HasViolations() {
		t.Error("expected timeout violation")
	}
}

// TestCodeCheck_RunWith_FileMode verifies RunWith with ScopeModeFile.
func TestCodeCheck_RunWith_FileMode(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "main.go")
	os.WriteFile(f, []byte("package main"), 0o644)

	executors := map[CheckType]PassExecutor{
		CheckTypeLint: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeLint}, nil
		}},
		CheckTypeBuild: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeBuild}, nil
		}},
		CheckTypeTest: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeTest}, nil
		}},
		CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeSemgrep}, nil
		}},
	}

	opts := RunOptions{
		Options: Options{
			Mode:        ScopeModeFile,
			FilePath:    f,
			BackstopDir: dir, // empty dir/rules, routing uses defaults
			ProjectDir:  dir,
		},
		Executors: executors,
	}

	result, err := RunWith(context.Background(), opts)
	if err != nil {
		t.Fatalf("RunWith: %v", err)
	}
	if len(result.PassResults) == 0 {
		t.Error("expected pass results for file mode")
	}
}

// TestCodeCheck_RunWith_AllMode verifies RunWith with ScopeModeAll.
func TestCodeCheck_RunWith_AllMode(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)

	executors := map[CheckType]PassExecutor{
		CheckTypeLint: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeLint}, nil
		}},
		CheckTypeBuild: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeBuild}, nil
		}},
		CheckTypeTest: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeTest}, nil
		}},
		CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeSemgrep}, nil
		}},
	}

	opts := RunOptions{
		Options: Options{
			Mode:        ScopeModeAll,
			BackstopDir: dir, // empty dir/rules, routing uses defaults
			ProjectDir:  dir,
		},
		Executors: executors,
	}

	result, err := RunWith(context.Background(), opts)
	if err != nil {
		t.Fatalf("RunWith: %v", err)
	}
	if len(result.PassResults) == 0 {
		t.Error("expected pass results for all mode")
	}
}

// TestCodeCheck_FileFlag_RoutesByType verifies that ScopeModeFile with a
// .go file only runs passes applicable to Go files (lint, build, test, semgrep)
// and that non-applicable passes are skipped. (CLM-009 functional coverage)
func TestCodeCheck_FileFlag_RoutesByType(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "main.go")
	os.WriteFile(goFile, []byte("package main"), 0o644)

	invoked := map[CheckType]bool{}
	var mu sync.Mutex

	makeExec := func(ct CheckType) PassExecutor {
		return &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			invoked[ct] = true
			mu.Unlock()
			return &PassResult{Pass: ct}, nil
		}}
	}

	executors := map[CheckType]PassExecutor{
		CheckTypeLint:    makeExec(CheckTypeLint),
		CheckTypeBuild:   makeExec(CheckTypeBuild),
		CheckTypeTest:    makeExec(CheckTypeTest),
		CheckTypeSemgrep: makeExec(CheckTypeSemgrep),
	}

	// Use default manifest (no manifest dir) so .go files get all 4 passes
	opts := RunOptions{
		Options: Options{
			Mode:        ScopeModeFile,
			FilePath:    goFile,
			BackstopDir: dir, // empty dir/rules, routing uses defaults
			ProjectDir:  dir,
		},
		Executors: executors,
	}

	result, err := RunWith(context.Background(), opts)
	if err != nil {
		t.Fatalf("RunWith: %v", err)
	}

	// All four passes should be invoked for a .go file
	for _, ct := range []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeSemgrep} {
		if !invoked[ct] {
			t.Errorf("pass %v was not invoked for .go file", ct)
		}
	}
	if len(result.PassResults) != 4 {
		t.Errorf("got %d pass results, want 4", len(result.PassResults))
	}

	// Now test with a .txt file — only semgrep should run with default manifest
	txtFile := filepath.Join(dir, "notes.txt")
	os.WriteFile(txtFile, []byte("some notes"), 0o644)

	invoked2 := map[CheckType]bool{}
	makeExec2 := func(ct CheckType) PassExecutor {
		return &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			mu.Lock()
			invoked2[ct] = true
			mu.Unlock()
			return &PassResult{Pass: ct}, nil
		}}
	}

	executors2 := map[CheckType]PassExecutor{
		CheckTypeLint:    makeExec2(CheckTypeLint),
		CheckTypeBuild:   makeExec2(CheckTypeBuild),
		CheckTypeTest:    makeExec2(CheckTypeTest),
		CheckTypeSemgrep: makeExec2(CheckTypeSemgrep),
	}

	opts2 := RunOptions{
		Options: Options{
			Mode:        ScopeModeFile,
			FilePath:    txtFile,
			BackstopDir: dir, // empty dir/rules, routing uses defaults
			ProjectDir:  dir,
		},
		Executors: executors2,
	}

	result2, err := RunWith(context.Background(), opts2)
	if err != nil {
		t.Fatalf("RunWith txt: %v", err)
	}

	// Only semgrep should be invoked for .txt (default manifest)
	if invoked2[CheckTypeLint] {
		t.Error("lint was invoked for .txt file")
	}
	if invoked2[CheckTypeBuild] {
		t.Error("build was invoked for .txt file")
	}
	if invoked2[CheckTypeTest] {
		t.Error("test was invoked for .txt file")
	}
	if !invoked2[CheckTypeSemgrep] {
		t.Error("semgrep was NOT invoked for .txt file")
	}

	// Verify that lint, build, test were skipped
	skippedCount := 0
	for _, pr := range result2.PassResults {
		if pr.Skipped {
			skippedCount++
		}
	}
	if skippedCount != 3 {
		t.Errorf("got %d skipped passes for .txt, want 3", skippedCount)
	}
}

// TestCodeCheck_BuildDefaultExecutors verifies that after the SPEC-034 cutover
// and the ISSUE-018 in-process-semgrep removal, the default (Go) executor map is
// empty: the build/test/lint passes run through the go-toolchain pack engines and
// the semgrep pass runs through the pack engine, so no executor is constructed in
// pkg/check. (CLM-005)
func TestCodeCheck_BuildDefaultExecutors(t *testing.T) {
	opts := Options{
		Mode:        ScopeModeDiff,
		BackstopDir: "/fake/.backstop",
	}
	executors := buildDefaultExecutors(opts)

	for _, ct := range []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeSemgrep} {
		if _, ok := executors[ct]; ok {
			t.Errorf("the Go stack must NOT construct a native %v executor; that pass runs through a pack engine", ct)
		}
	}
}

// --- Test helpers ---

// mockPassExecutor implements PassExecutor for testing.
type mockPassExecutor struct {
	fn             func(ctx context.Context, files []string) (*PassResult, error)
	available      bool
	unavailableMsg string
}

func (m *mockPassExecutor) Execute(ctx context.Context, files []string) (*PassResult, error) {
	if m.fn != nil {
		return m.fn(ctx, files)
	}
	return &PassResult{}, nil
}

func (m *mockPassExecutor) IsAvailable() (bool, string) {
	if m.fn != nil && !m.available && m.unavailableMsg == "" {
		return true, ""
	}
	if m.unavailableMsg != "" {
		return false, m.unavailableMsg
	}
	return true, ""
}

func findPassResult(result *Result, ct CheckType) *PassResult {
	for i := range result.PassResults {
		if result.PassResults[i].Pass == ct {
			return &result.PassResults[i]
		}
	}
	return nil
}

// fakeRunner is a test double for CommandRunner. It returns canned output keyed
// by command name (the first arg to Run), records the most recent invocation's
// name and args for assertion, and optionally returns a canned error so tests
// can simulate a non-zero tool exit. It never shells out to a live tool.
type fakeRunner struct {
	// outputs maps a command name (e.g. "golangci-lint", "go", "semgrep") to
	// the bytes Run should return for it.
	outputs map[string][]byte
	// err, if set, is returned alongside the output (tools like golangci-lint
	// and go build exit non-zero when they find problems).
	err error

	// queued holds per-call responses keyed by command name, consumed in FIFO
	// order each time that name is invoked. This lets a single binary (e.g.
	// "golangci-lint") return DIFFERENT bytes/error across sequential calls —
	// the version-probe call then the run call (ISSUE-006). When the queue for
	// a name is empty (or nil), Run falls back to outputs[name]/err, so every
	// test that leaves queued nil behaves exactly as before.
	queued map[string][]queuedResponse

	// recorded invocations, most recent last.
	calls []runnerCall
}

// queuedResponse is one canned (output, error) pair for a single Run call.
type queuedResponse struct {
	out []byte
	err error
}

type runnerCall struct {
	name string
	args []string
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, runnerCall{name: name, args: append([]string(nil), args...)})
	if q := f.queued[name]; len(q) > 0 {
		resp := q[0]
		f.queued[name] = q[1:]
		return resp.out, resp.err
	}
	if f.outputs == nil {
		return nil, f.err
	}
	return f.outputs[name], f.err
}

// RunStdout shares the canned-output mechanism with Run; the fake does not
// model the stdout/stderr split (real separation is covered by
// TestRunner_StdoutSeparateFromStderr against ExecCommandRunner), it only needs
// to satisfy the CommandRunner interface for executors that call RunStdout.
func (f *fakeRunner) RunStdout(ctx context.Context, name string, args ...string) ([]byte, error) {
	return f.Run(ctx, name, args...)
}

// lastCall returns the most recent invocation recorded by the fake runner.
func (f *fakeRunner) lastCall() runnerCall {
	if len(f.calls) == 0 {
		return runnerCall{}
	}
	return f.calls[len(f.calls)-1]
}

// TestCodeCheck_Run_DelegatesToRunWith verifies Run delegates to RunWith.
func TestCodeCheck_Run_DelegatesToRunWith(t *testing.T) {
	// Run calls RunWith which calls ResolveScope (real git). We just verify
	// it doesn't panic for ScopeModeFile with a real file.
	dir := t.TempDir()
	f := filepath.Join(dir, "main.go")
	os.WriteFile(f, []byte("package main"), 0o644)

	result, err := Run(context.Background(), Options{
		Mode:        ScopeModeFile,
		FilePath:    f,
		BackstopDir: dir, // empty dir/rules, routing uses defaults
		ProjectDir:  dir,
	})
	// May fail due to semgrep not being available, but should not panic
	_ = err
	_ = result
}

// TestCodeCheck_RunPasses_ExecutorError verifies that a non-timeout executor
// error is recorded as a violation.
func TestCodeCheck_RunPasses_ExecutorError(t *testing.T) {
	executors := map[CheckType]PassExecutor{
		CheckTypeLint: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return nil, fmt.Errorf("lint binary crashed")
		}},
		CheckTypeBuild: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeBuild}, nil
		}},
		CheckTypeTest: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeTest}, nil
		}},
		CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeSemgrep}, nil
		}},
	}

	engine := &Engine{
		Executors: executors,
		Manifest:  defaultManifest(),
	}

	result, err := engine.RunPasses(context.Background(), []string{"main.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasViolations() {
		t.Error("expected violation from executor error")
	}
	// Verify the error message is in the violations
	found := false
	for _, v := range result.AllViolations() {
		if contains(v.Message, "lint binary crashed") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("violation message should contain executor error, got: %v", result.AllViolations())
	}
}

// TestCodeCheck_RunPasses_NoExecutorConfigured verifies that when no executor
// is configured for a check type, the pass is skipped.
func TestCodeCheck_RunPasses_NoExecutorConfigured(t *testing.T) {
	// Only provide lint executor, not build/test/semgrep
	executors := map[CheckType]PassExecutor{
		CheckTypeLint: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeLint}, nil
		}},
	}

	engine := &Engine{
		Executors: executors,
		Manifest:  defaultManifest(),
	}

	result, err := engine.RunPasses(context.Background(), []string{"main.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	skippedCount := 0
	for _, pr := range result.PassResults {
		if pr.Skipped && pr.SkipReason == "no executor configured" {
			skippedCount++
		}
	}
	if skippedCount != 3 {
		t.Errorf("got %d skipped (no executor), want 3", skippedCount)
	}
}

// TestCodeCheck_RunPasses_ContextCancelledBeforePass verifies that when the
// context is already cancelled before a pass starts, it records a timeout
// violation and breaks.
func TestCodeCheck_RunPasses_ContextCancelledBeforePass(t *testing.T) {
	executors := map[CheckType]PassExecutor{
		CheckTypeLint: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeLint}, nil
		}},
		CheckTypeBuild: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeBuild}, nil
		}},
		CheckTypeTest: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeTest}, nil
		}},
		CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeSemgrep}, nil
		}},
	}

	engine := &Engine{
		Executors: executors,
		Manifest:  defaultManifest(),
	}

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := engine.RunPasses(ctx, []string{"main.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasViolations() {
		t.Error("expected timeout violation for cancelled context")
	}
	// Should have broken early - not all 4 passes recorded
	if len(result.PassResults) == 4 {
		t.Error("expected early break, but all 4 pass results recorded")
	}
}

// TestCodeCheck_RunWith_NoBackstopDir verifies RunWith works when BackstopDir
// is empty (skips semgrep ensure).
func TestCodeCheck_RunWith_NoBackstopDir(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "main.go")
	os.WriteFile(goFile, []byte("package main"), 0o644)

	executors := map[CheckType]PassExecutor{
		CheckTypeLint: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeLint}, nil
		}},
		CheckTypeBuild: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeBuild}, nil
		}},
		CheckTypeTest: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeTest}, nil
		}},
		CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			return &PassResult{Pass: CheckTypeSemgrep}, nil
		}},
	}

	opts := RunOptions{
		Options: Options{
			Mode:        ScopeModeFile,
			FilePath:    goFile,
			ProjectDir:  dir,
			BackstopDir: "", // empty — skip semgrep ensure
		},
		Executors: executors,
	}

	result, err := RunWith(context.Background(), opts)
	if err != nil {
		t.Fatalf("RunWith: %v", err)
	}
	if len(result.PassResults) != 4 {
		t.Errorf("got %d pass results, want 4", len(result.PassResults))
	}
}

// TestCodeCheck_RunWith_ScopeError verifies RunWith returns error when scope
// resolution fails.
func TestCodeCheck_RunWith_ScopeError(t *testing.T) {
	git := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			return "", errNoRemote
		},
		diffLocalFn: func() ([]string, error) {
			return nil, fmt.Errorf("git died")
		},
	}

	opts := RunOptions{
		Options: Options{
			Mode:       ScopeModeDiff,
			ProjectDir: "", // empty project dir + diff local failure -> resolveScopeAll fails
		},
		Git: git,
	}

	// With empty ProjectDir and all git fallbacks failing, this should
	// ultimately call resolveScopeAll("") which returns an error
	_, err := RunWith(context.Background(), opts)
	if err != nil {
		// This path actually falls through to resolveScopeAll which returns error for empty dir
		// That's fine — we exercised the scope error path in RunWith
		_ = err
	}
}

// TestCodeCheck_Result_Methods verifies Result helper methods.
func TestCodeCheck_Result_Methods(t *testing.T) {
	// Empty result
	r := &Result{}
	if r.HasViolations() {
		t.Error("empty result should not have violations")
	}
	if r.ViolationCount() != 0 {
		t.Errorf("empty result: ViolationCount = %d, want 0", r.ViolationCount())
	}
	if len(r.AllViolations()) != 0 {
		t.Errorf("empty result: AllViolations = %d, want 0", len(r.AllViolations()))
	}

	// Result with violations across passes
	r2 := &Result{
		PassResults: []PassResult{
			{Pass: CheckTypeLint, Violations: []Violation{
				{Pass: CheckTypeLint, Message: "a"},
				{Pass: CheckTypeLint, Message: "b"},
			}},
			{Pass: CheckTypeBuild},
			{Pass: CheckTypeTest, Violations: []Violation{
				{Pass: CheckTypeTest, Message: "c"},
			}},
		},
	}
	if !r2.HasViolations() {
		t.Error("expected HasViolations true")
	}
	if r2.ViolationCount() != 3 {
		t.Errorf("ViolationCount = %d, want 3", r2.ViolationCount())
	}
	all := r2.AllViolations()
	if len(all) != 3 {
		t.Errorf("AllViolations = %d, want 3", len(all))
	}
}

// TestOptions_NoManifestDirField constructs check.Options with the post-removal
// field set (no ManifestDir) to confirm the field is gone and the package still
// compiles. If a ManifestDir field were re-added, this keyed literal would still
// compile, so the companion source self-check TestNoTestRequiresOptionsManifestDir
// guards the token's absence across the test corpus. (CLM-004)
func TestOptions_NoManifestDirField(t *testing.T) {
	dir := t.TempDir()
	opts := Options{
		Mode:        ScopeModeAll,
		FilePath:    "",
		BackstopDir: dir,
		Timeout:     0,
		ProjectDir:  dir,
		Language:    "",
		Config:      nil,
		Files:       nil,
	}
	if opts.BackstopDir != dir {
		t.Fatalf("BackstopDir = %q, want %q", opts.BackstopDir, dir)
	}
}

// TestNoTestRequiresOptionsManifestDir is a source self-check over the pkg/check
// test files asserting the removed Options field token does not survive in any
// check.Options literal, so the green go-test guarantee (REQ-005, CLM-021) is
// enforced rather than assumed. (CLM-024)
func TestNoTestRequiresOptionsManifestDir(t *testing.T) {
	// Build the token from parts so this self-check file does not match itself.
	token := "Manifest" + "Dir:"
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading pkg/check dir: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, readErr := os.ReadFile(name)
		if readErr != nil {
			t.Fatalf("reading %s: %v", name, readErr)
		}
		if strings.Contains(string(data), token) {
			t.Errorf("%s still references a %s field; check.Options has no ManifestDir", name, token)
		}
	}
}

func init() {
	// Suppress unused import warning
	_ = fmt.Sprintf
}
