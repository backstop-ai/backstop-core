package check

import (
	"context"
	"testing"
)

// findViolation returns the first violation in vs whose File and Line match,
// or nil. Used so assertions do not depend on ordering. Shared with
// semgrep_executor_test.go.
func findViolation(vs []Violation, file string, line int) *Violation {
	for i := range vs {
		if vs[i].File == file && vs[i].Line == line {
			return &vs[i]
		}
	}
	return nil
}

// SPEC-034 cutover: the bespoke Go lint/build/test executors and their tests were
// deleted — the Go build/test/lint passes now run through the go-toolchain pack
// engines (covered by cmd/backstop's pack_gate_gotoolchain / bridge / golint /
// convert tests). What remains here is the SURVIVING shared semgrepExecutor and
// the generic-executor cancellation contract.

// TestCodeCheck_Executors_EntryContextCancelled verifies that the surviving
// executors (the generic commandExecutor and the shared semgrepExecutor) return
// ctx.Err() immediately when the context is already cancelled at entry, without
// invoking the runner.
func TestCodeCheck_Executors_EntryContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner := &fakeRunner{outputs: map[string][]byte{}}
	parser, _ := lookupParser("regex-lines")

	execs := []PassExecutor{
		&commandExecutor{pass: CheckTypeLint, command: "tool", parser: parser, scopeKind: ScopeKindFileArgs, runner: runner},
		&semgrepExecutor{runner: runner, ensurer: &mockSemgrepEnsurer{}},
	}
	for _, ex := range execs {
		if _, err := ex.Execute(ctx, []string{"a.go"}); err == nil {
			t.Errorf("%T.Execute should return ctx error when cancelled at entry", ex)
		}
	}
	if len(runner.calls) != 0 {
		t.Errorf("no runner call expected on entry-cancellation, got %d", len(runner.calls))
	}
}

// TestCodeCheck_SemgrepExecutor_EnsureFailureErrors verifies that an
// EnsureSemgrep failure surfaces as an Execute error rather than a silent pass.
func TestCodeCheck_SemgrepExecutor_EnsureFailureErrors(t *testing.T) {
	runner := &fakeRunner{outputs: map[string][]byte{}}
	ensurer := &mockSemgrepEnsurer{fn: func(_, _ string) (string, error) {
		return "", &DegradedError{Message: "install failed"}
	}}
	e := &semgrepExecutor{runner: runner, ensurer: ensurer}
	if _, err := e.Execute(context.Background(), []string{"a.go"}); err == nil {
		t.Error("expected an error when EnsureSemgrep fails")
	}
	if len(runner.calls) != 0 {
		t.Error("runner should not be invoked when the binary cannot be resolved")
	}
}

// TestCodeCheck_SemgrepSeverity covers severity normalization, including the
// default (unknown) branch.
func TestCodeCheck_SemgrepSeverity(t *testing.T) {
	cases := map[string]string{
		"ERROR":   "error",
		"WARNING": "warning",
		"INFO":    "info", // default branch lowercases
		"":        "",
	}
	for in, want := range cases {
		if got := semgrepSeverity(in); got != want {
			t.Errorf("semgrepSeverity(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestCodeCheck_SemgrepExecutor_MalformedJSONErrors verifies a parse error is
// surfaced for unparseable semgrep output.
func TestCodeCheck_SemgrepExecutor_MalformedJSONErrors(t *testing.T) {
	const binPath = "semgrep"
	runner := &fakeRunner{outputs: map[string][]byte{binPath: []byte("{not json")}}
	ensurer := &mockSemgrepEnsurer{fn: func(_, _ string) (string, error) { return binPath, nil }}
	e := &semgrepExecutor{runner: runner, ensurer: ensurer}
	if _, err := e.Execute(context.Background(), []string{"a.go"}); err == nil {
		t.Error("expected a parse error for malformed semgrep output")
	}
}
