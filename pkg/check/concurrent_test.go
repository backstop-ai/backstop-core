package check

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestCodeCheck_FileMode_CompletesWithinBudget verifies --file mode completes
// within 2-second budget under normal conditions. (CLM-011)
func TestCodeCheck_FileMode_CompletesWithinBudget(t *testing.T) {
	executors := map[CheckType]PassExecutor{
		CheckTypeLint:    &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) { return &PassResult{Pass: CheckTypeLint}, nil }},
		CheckTypeBuild:   &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) { return &PassResult{Pass: CheckTypeBuild}, nil }},
		CheckTypeTest:    &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) { return &PassResult{Pass: CheckTypeTest}, nil }},
		CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) { return &PassResult{Pass: CheckTypeSemgrep}, nil }},
	}

	engine := &Engine{
		Executors: executors,
		Manifest:  defaultManifest(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	result, err := engine.RunPasses(ctx, []string{"main.go"})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed > 2*time.Second {
		t.Errorf("execution took %v, exceeds 2-second budget", elapsed)
	}
	if result.HasViolations() {
		t.Error("expected no violations for fast passes")
	}
}

// TestCodeCheck_FileMode_TimeoutReturnsViolation verifies --file mode cancels
// and returns a timeout violation when budget is exceeded. (CLM-012)
func TestCodeCheck_FileMode_TimeoutReturnsViolation(t *testing.T) {
	executors := map[CheckType]PassExecutor{
		CheckTypeLint: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) {
			// Simulate a slow pass that blocks until context is cancelled
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(10 * time.Second):
				return &PassResult{Pass: CheckTypeLint}, nil
			}
		}},
		CheckTypeBuild:   &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) { return &PassResult{Pass: CheckTypeBuild}, nil }},
		CheckTypeTest:    &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) { return &PassResult{Pass: CheckTypeTest}, nil }},
		CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) { return &PassResult{Pass: CheckTypeSemgrep}, nil }},
	}

	engine := &Engine{
		Executors: executors,
		Manifest:  defaultManifest(),
	}

	// Short timeout to trigger cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := engine.RunPasses(ctx, []string{"main.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.HasViolations() {
		t.Error("expected timeout violation, but no violations found")
	}

	// Check that a timeout violation was recorded
	foundTimeout := false
	for _, v := range result.AllViolations() {
		if contains(v.Message, "timeout") {
			foundTimeout = true
			break
		}
	}
	if !foundTimeout {
		t.Errorf("no timeout violation found in: %v", result.AllViolations())
	}
}

// TestCodeCheck_FileMode_TimeoutExitCode1 verifies timeout produces exit code 1
// (violations), not exit code 2 (config error). (CLM-013)
func TestCodeCheck_FileMode_TimeoutExitCode1(t *testing.T) {
	result := &Result{
		PassResults: []PassResult{
			{
				Pass: CheckTypeLint,
				Violations: []Violation{{
					Pass:     CheckTypeLint,
					Message:  "timeout: lint pass cancelled",
					Severity: "error",
				}},
			},
		},
		Warnings: []string{"execution timed out during lint pass"},
	}

	code := DetermineExitCode(result, nil, false)
	if code != 1 {
		t.Errorf("exit code = %d, want 1 (violations, not config error)", code)
	}
}

// TestCodeCheck_Concurrent_NoSharedStateCorruption launches 10 concurrent
// invocations and verifies no data races or corrupted results. (CLM-031)
func TestCodeCheck_Concurrent_NoSharedStateCorruption(t *testing.T) {
	const goroutines = 10

	manifest := defaultManifest()

	var wg sync.WaitGroup
	results := make([]*Result, goroutines)
	errors := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			executors := map[CheckType]PassExecutor{
				CheckTypeLint:    &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) { return &PassResult{Pass: CheckTypeLint, Violations: []Violation{{Pass: CheckTypeLint, File: "a.go", Message: "lint"}}}, nil }},
				CheckTypeBuild:   &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) { return &PassResult{Pass: CheckTypeBuild}, nil }},
				CheckTypeTest:    &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) { return &PassResult{Pass: CheckTypeTest}, nil }},
				CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) { return &PassResult{Pass: CheckTypeSemgrep}, nil }},
			}

			engine := &Engine{
				Executors: executors,
				Manifest:  manifest,
			}

			results[idx], errors[idx] = engine.RunPasses(context.Background(), []string{"a.go"})
		}(i)
	}

	wg.Wait()

	for i := 0; i < goroutines; i++ {
		if errors[i] != nil {
			t.Errorf("goroutine %d: unexpected error: %v", i, errors[i])
		}
		if results[i] == nil {
			t.Errorf("goroutine %d: nil result", i)
			continue
		}
		if results[i].ViolationCount() != 1 {
			t.Errorf("goroutine %d: got %d violations, want 1", i, results[i].ViolationCount())
		}
	}
}

// TestCodeCheck_Concurrent_IndependentOutput launches concurrent invocations
// and verifies each gets its own complete result. (CLM-032)
func TestCodeCheck_Concurrent_IndependentOutput(t *testing.T) {
	const goroutines = 5

	var wg sync.WaitGroup
	outputs := make([]string, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			executors := map[CheckType]PassExecutor{
				CheckTypeLint:    &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) { return &PassResult{Pass: CheckTypeLint}, nil }},
				CheckTypeBuild:   &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) { return &PassResult{Pass: CheckTypeBuild}, nil }},
				CheckTypeTest:    &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) { return &PassResult{Pass: CheckTypeTest}, nil }},
				CheckTypeSemgrep: &mockPassExecutor{fn: func(ctx context.Context, files []string) (*PassResult, error) { return &PassResult{Pass: CheckTypeSemgrep}, nil }},
			}

			engine := &Engine{
				Executors: executors,
				Manifest:  defaultManifest(),
			}

			result, _ := engine.RunPasses(context.Background(), []string{"main.go"})
			out, _ := FormatResult(result, OutputModeJSON)
			outputs[idx] = out
		}(i)
	}

	wg.Wait()

	for i := 0; i < goroutines; i++ {
		if outputs[i] == "" {
			t.Errorf("goroutine %d: empty output", i)
			continue
		}
		// Each output should be valid JSON and contain schema_version
		if !contains(outputs[i], "schema_version") {
			t.Errorf("goroutine %d: output missing schema_version", i)
		}
	}
}

// SPEC-034 cutover: TestCodeCheck_Concurrent_SemgrepInstallIdempotent was deleted
// with the bespoke install ladder it exercised. semgrep is now provisioned through
// the declared provisioning model (provisionEngines), not an ad hoc pip-install +
// lock in pkg/check, so there is no in-package concurrent-install path to test.
