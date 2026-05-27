package gate

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockCommandRunner implements CommandRunner for testing.
type mockCommandRunner struct {
	output []byte
	err    error
	runs   int
}

func (m *mockCommandRunner) Run(_ context.Context, _ string, _ ...string) ([]byte, error) {
	m.runs++
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

func TestGateSteps_FilterToChangedFiles_Coverage(t *testing.T) {
	runner := &mockCommandRunner{output: []byte("ok  \tpkg/gate\t1.234s\tcoverage: 85.0% of statements\n")}
	specs := []SpecVerification{
		{SpecID: "CHANGED", TestCommand: "go test ./pkg/gate/...", CoverageThreshold: 80, File: "specs/changed.spec.md"},
		{SpecID: "UNCHANGED", TestCommand: "go test ./pkg/gate/...", CoverageThreshold: 80, File: "specs/unchanged.spec.md"},
	}
	result := StepCoverageThresholdScopedFunc(runner, specs, newGateScope("", GateScopeModeDiff, []string{"specs/changed.spec.md"}, nil))(context.Background())
	if result.Status != "pass" || runner.runs != 1 {
		t.Fatalf("expected coverage to run only changed spec, status=%s runs=%d violations=%#v", result.Status, runner.runs, result.Violations)
	}
}

func TestGateSteps_FilterToChangedFiles_CoverageChangedPackage(t *testing.T) {
	runner := &mockCommandRunner{output: []byte("ok  \tpkg/gate\t1.234s\tcoverage: 85.0% of statements\n")}
	specs := []SpecVerification{
		{SpecID: "GATE", TestCommand: "go test ./pkg/gate/...", CoverageThreshold: 80, File: "specs/unchanged-gate.spec.md"},
		{SpecID: "OTHER", TestCommand: "go test ./pkg/other", CoverageThreshold: 80, File: "specs/unchanged-other.spec.md"},
	}
	result := StepCoverageThresholdScopedFunc(runner, specs, newGateScope("", GateScopeModeDiff, []string{"pkg/gate/step_coverage.go"}, nil))(context.Background())
	if result.Status != "pass" || runner.runs != 1 {
		t.Fatalf("expected coverage to run for changed package, status=%s runs=%d violations=%#v", result.Status, runner.runs, result.Violations)
	}
}

func TestGateSteps_FilterToChangedFiles_CoverageRootPackage(t *testing.T) {
	runner := &mockCommandRunner{output: []byte("ok  \tpkg/gate\t1.234s\tcoverage: 85.0% of statements\n")}
	result := StepCoverageThresholdScopedFunc(runner, []SpecVerification{
		{SpecID: "ROOT", TestCommand: "go test ./...", CoverageThreshold: 80, File: "specs/unchanged-root.spec.md"},
	}, newGateScope("", GateScopeModeDiff, []string{"pkg/gate/step_coverage.go"}, nil))(context.Background())
	if result.Status != "pass" || runner.runs != 1 {
		t.Fatalf("expected coverage to use changed package instead of spec test_command, status=%s runs=%d violations=%#v", result.Status, runner.runs, result.Violations)
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

func TestGate_CoverageThreshold_IgnoresSpecTestCommandForScheduling(t *testing.T) {
	var capturedArgs []string
	runner := &recordingCommandRunner{
		output: []byte("ok  \tpkg/gate\t1.234s\tcoverage: 90.0% of statements\n"),
		onRun: func(name string, args ...string) {
			capturedArgs = append(capturedArgs, append([]string{name}, args...)...)
		},
	}
	specs := []SpecVerification{
		{SpecID: "TEST-001", TestCommand: "go test ./pkg/gate/... -race", CoverageThreshold: 80},
	}
	step := StepCoverageThresholdFunc(runner, specs)
	_ = step(context.Background())

	if len(capturedArgs) == 0 {
		t.Fatal("expected command to be executed")
	}
	if capturedArgs[0] != "go" {
		t.Errorf("expected command %q, got %q", "go", capturedArgs[0])
	}
	if fmt.Sprint(capturedArgs) != "[go test . ./cmd/... ./pkg/... ./tests/... -coverprofile=/dev/null]" {
		t.Fatalf("expected gate-owned default coverage command, got %#v", capturedArgs)
	}
}

func TestGate_CoverageThreshold_StripsQuotedRunPattern(t *testing.T) {
	parts := commandFields("go test ./cmd/backstop ./pkg/gate/... -run 'TestGate|TestBackstopGate' -v")
	if len(parts) != 7 || parts[5] != "TestGate|TestBackstopGate" {
		t.Fatalf("expected quoted run pattern as one unquoted arg, got %#v", parts)
	}
}

func TestGate_CoverageTargets_DerivedFromScope(t *testing.T) {
	targets := coverageTargetsForScope(newGateScope("", GateScopeModeDiff, []string{"pkg/gate/step_coverage.go", "pkg/config/config.go", "README.md"}, nil))
	got := make([]string, 0, len(targets))
	for _, target := range targets {
		got = append(got, fmt.Sprintf("%s %s", target.Command, strings.Join(target.Args, " ")))
	}
	want := []string{"go test ./pkg/config -coverprofile=/dev/null", "go test ./pkg/gate -coverprofile=/dev/null"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("unexpected targets: got %#v want %#v", got, want)
	}
}

func TestGate_CoverageThreshold_CollapsesCodeScopeThresholds(t *testing.T) {
	runner := &mockCommandRunner{output: []byte("ok  \tpkg/gate\t1.234s\tcoverage: 75.0% of statements\n")}
	result := StepCoverageThresholdScopedFunc(runner, []SpecVerification{
		{SpecID: "ONE", CoverageThreshold: 80, File: "specs/one.spec.md"},
		{SpecID: "TWO", CoverageThreshold: 90, File: "specs/two.spec.md"},
	}, newGateScope("", GateScopeModeDiff, []string{"pkg/gate/step_coverage.go"}, nil))(context.Background())
	if len(result.Violations) != 1 {
		t.Fatalf("expected one collapsed code-scope violation, got %#v", result.Violations)
	}
	if result.Violations[0].Message != "changed Go package coverage 75.0% below maximum declared threshold 90%" {
		t.Fatalf("unexpected violation message: %q", result.Violations[0].Message)
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
	specs := []SpecVerification{{SpecID: "TEST-001", TestCommand: "go test ./pkg/gate/...", CoverageThreshold: 80}}
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
		if v.Message == "coverage summary line not found in test output for . ./cmd/... ./pkg/... ./tests/..." {
			found = true
		}
	}
	if !found {
		t.Error("expected violation about missing coverage summary line")
	}
}

func TestGate_CoverageTargets_AllScopeExcludesPrototypeFixtures(t *testing.T) {
	targets := coverageTargetsForScope(nil)
	got := make([]string, 0, len(targets))
	for _, target := range targets {
		got = append(got, fmt.Sprintf("%s %s", target.Command, strings.Join(target.Args, " ")))
	}
	want := []string{
		"go test . ./cmd/... ./pkg/... ./tests/... -coverprofile=/dev/null",
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("unexpected targets: got %#v want %#v", got, want)
	}
}

// TestGate_CoverageThreshold_ParsesCoverageSummaryLine verifies parsing of
// the "coverage: 82.5% of statements" format.
func TestGate_CoverageThreshold_ParsesCoverageSummaryLine(t *testing.T) {
	tests := []struct {
		line   string
		want   float64
		wantOK bool
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
