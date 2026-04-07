package gate

import (
	"context"
	"fmt"
	"testing"
)

// mockCommandRunner implements CommandRunner for testing.
type mockCommandRunner struct {
	output []byte
	err    error
}

func (m *mockCommandRunner) Run(_ context.Context, _ string, _ ...string) ([]byte, error) {
	return m.output, m.err
}

// TestGate_CoverageThreshold_MeetsThreshold verifies pass when coverage meets threshold.
func TestGate_CoverageThreshold_MeetsThreshold(t *testing.T) {
	runner := &mockCommandRunner{
		output: []byte("ok  \tpkg/gate\t1.234s\tcoverage: 85.0% of statements\n"),
	}
	specs := []SpecVerification{
		{SpecID: "TEST-001", TestCommand: "go test ./pkg/gate/...", CoverageThreshold: 80},
	}
	step := StepCoverageThresholdFunc(runner, specs)
	result := step(context.Background())

	if result.StepName != StepCoverageThreshold {
		t.Errorf("expected step_name %q, got %q", StepCoverageThreshold, result.StepName)
	}
	if result.Status != "pass" {
		t.Errorf("expected status %q, got %q; violations: %v", "pass", result.Status, result.Violations)
	}
}

// TestGate_CoverageThreshold_BelowThreshold verifies fail when coverage is below threshold.
func TestGate_CoverageThreshold_BelowThreshold(t *testing.T) {
	runner := &mockCommandRunner{
		output: []byte("ok  \tpkg/gate\t1.234s\tcoverage: 65.0% of statements\n"),
	}
	specs := []SpecVerification{
		{SpecID: "TEST-001", TestCommand: "go test ./pkg/gate/...", CoverageThreshold: 80},
	}
	step := StepCoverageThresholdFunc(runner, specs)
	result := step(context.Background())

	if result.Status != "fail" {
		t.Errorf("expected status %q, got %q", "fail", result.Status)
	}
	if len(result.Violations) == 0 {
		t.Error("expected at least one violation for below-threshold coverage")
	}
}

// TestGate_CoverageThreshold_UsesSpecTestCommand verifies step uses the
// test_command from the spec verification block.
func TestGate_CoverageThreshold_UsesSpecTestCommand(t *testing.T) {
	var capturedArgs []string
	runner := &recordingCommandRunner{
		output: []byte("ok  \tpkg/gate\t1.234s\tcoverage: 90.0% of statements\n"),
		onRun: func(name string, args ...string) {
			capturedArgs = append([]string{name}, args...)
		},
	}
	specs := []SpecVerification{
		{SpecID: "TEST-001", TestCommand: "go test ./pkg/gate/... -race", CoverageThreshold: 80},
	}
	step := StepCoverageThresholdFunc(runner, specs)
	_ = step(context.Background())

	// The test command should have been split and passed to the runner.
	// We expect "go" as the command name and ["test", "./pkg/gate/...", "-race", "-coverprofile=..."] as args.
	if len(capturedArgs) == 0 {
		t.Fatal("expected command to be executed")
	}
	if capturedArgs[0] != "go" {
		t.Errorf("expected command %q, got %q", "go", capturedArgs[0])
	}
}

// recordingCommandRunner records the args passed to Run.
type recordingCommandRunner struct {
	output []byte
	onRun  func(name string, args ...string)
}

func (r *recordingCommandRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	if r.onRun != nil {
		r.onRun(name, args...)
	}
	return r.output, nil
}

// TestGate_CoverageThreshold_TestCommandNotFound verifies fail when test command
// cannot be executed.
func TestGate_CoverageThreshold_TestCommandNotFound(t *testing.T) {
	runner := &mockCommandRunner{
		err: fmt.Errorf("exec: \"nonexistent\": executable file not found in $PATH"),
	}
	specs := []SpecVerification{
		{SpecID: "TEST-001", TestCommand: "nonexistent --test", CoverageThreshold: 80},
	}
	step := StepCoverageThresholdFunc(runner, specs)
	result := step(context.Background())

	if result.Status != "fail" {
		t.Errorf("expected status %q, got %q", "fail", result.Status)
	}
	if len(result.Violations) == 0 {
		t.Error("expected violation for failed command execution")
	}
}

// TestGate_CoverageThreshold_NoCoverageSummaryLine verifies fail when coverage
// summary line is not present in test output.
func TestGate_CoverageThreshold_NoCoverageSummaryLine(t *testing.T) {
	runner := &mockCommandRunner{
		output: []byte("ok  \tpkg/gate\t1.234s\n"), // no coverage line
	}
	specs := []SpecVerification{
		{SpecID: "TEST-001", TestCommand: "go test ./pkg/gate/...", CoverageThreshold: 80},
	}
	step := StepCoverageThresholdFunc(runner, specs)
	result := step(context.Background())

	if result.Status != "fail" {
		t.Errorf("expected status %q, got %q", "fail", result.Status)
	}
	found := false
	for _, v := range result.Violations {
		if v.Message == "coverage summary line not found in test output" {
			found = true
		}
	}
	if !found {
		t.Error("expected violation about missing coverage summary line")
	}
}

// TestGate_CoverageThreshold_ParsesCoverageSummaryLine verifies parsing of
// the "coverage: 82.5% of statements" format.
func TestGate_CoverageThreshold_ParsesCoverageSummaryLine(t *testing.T) {
	tests := []struct {
		line    string
		want    float64
		wantOK  bool
	}{
		{"coverage: 82.5% of statements", 82.5, true},
		{"coverage: 100.0% of statements", 100.0, true},
		{"coverage: 0.0% of statements", 0.0, true},
		{"ok  \tpkg/gate\t1.234s", 0, false},
		{"", 0, false},
	}

	for _, tt := range tests {
		pct, ok := parseCoverageLine(tt.line)
		if ok != tt.wantOK {
			t.Errorf("parseCoverageLine(%q): got ok=%v, want %v", tt.line, ok, tt.wantOK)
		}
		if ok && pct != tt.want {
			t.Errorf("parseCoverageLine(%q): got pct=%v, want %v", tt.line, pct, tt.want)
		}
	}
}
