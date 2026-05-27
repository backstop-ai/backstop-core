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

func TestGate_CoverageTargets_ProductionFilesTakePrecedenceOverTestFiles(t *testing.T) {
	targets := coverageTargetsForScope(newGateScope("", GateScopeModeDiff, []string{"cmd/backstop/gate_test.go", "pkg/gate/step_coverage.go"}, nil))
	if len(targets) != 1 {
		t.Fatalf("expected only production package target, got %#v", targets)
	}
	if targets[0].Label != "./pkg/gate" {
		t.Fatalf("expected pkg/gate target, got %q", targets[0].Label)
	}
}

func TestGate_CoverageTargets_TestOnlyScopeStillRunsPackageCoverage(t *testing.T) {
	targets := coverageTargetsForScope(newGateScope("", GateScopeModeDiff, []string{"cmd/backstop/gate_test.go"}, nil))
	if len(targets) != 1 {
		t.Fatalf("expected test-only package target, got %#v", targets)
	}
	if targets[0].Label != "./cmd/backstop" {
		t.Fatalf("expected cmd/backstop target, got %q", targets[0].Label)
	}
}

func TestGate_CoverageThreshold_CodeScopeUsesRelevantSpecThreshold(t *testing.T) {
	runner := &mockCommandRunner{output: []byte("ok  \tpkg/gate\t1.234s\tcoverage: 75.0% of statements\n")}
	result := StepCoverageThresholdScopedFunc(runner, []SpecVerification{
		{SpecID: "RELEVANT", TestCommand: "go test ./pkg/gate/...", CoverageThreshold: 80, File: "specs/relevant.spec.md", ImplementationPackage: "pkg/gate"},
		{SpecID: "UNRELATED", TestCommand: "semgrep --test standards/go/testdata", CoverageThreshold: 100, File: "specs/unrelated.spec.md", ImplementationPackage: "standards/go"},
	}, newGateScope("", GateScopeModeDiff, []string{"pkg/gate/step_coverage.go"}, nil))(context.Background())
	if len(result.Violations) != 1 {
		t.Fatalf("expected one collapsed code-scope violation, got %#v", result.Violations)
	}
	if result.Violations[0].Message != "changed Go package coverage 75.0% below threshold 80%" {
		t.Fatalf("unexpected violation message: %q", result.Violations[0].Message)
	}
}

func TestGate_CoverageThreshold_CodeScopeUsesUnitFloorWhenNoSpecRelevant(t *testing.T) {
	runner := &mockCommandRunner{output: []byte("ok  \tpkg/gate\t1.234s\tcoverage: 85.0% of statements\n")}
	result := StepCoverageThresholdScopedFunc(runner, []SpecVerification{
		{SpecID: "UNRELATED", TestCommand: "semgrep --test standards/go/testdata", CoverageThreshold: 100, File: "specs/unrelated.spec.md", ImplementationPackage: "standards/go"},
	}, newGateScope("", GateScopeModeDiff, []string{"pkg/gate/step_coverage.go"}, nil))(context.Background())
	if len(result.Violations) != 1 {
		t.Fatalf("expected one default-floor violation, got %#v", result.Violations)
	}
	if result.Violations[0].Message != "changed Go package coverage 85.0% below threshold 90%" {
		t.Fatalf("unexpected violation message: %q", result.Violations[0].Message)
	}
}

func TestGate_CoverageThreshold_CodeScopeUsesRootCommandWhenNoSpecificSpec(t *testing.T) {
	runner := &mockCommandRunner{output: []byte("ok  \tpkg/gate\t1.234s\tcoverage: 85.0% of statements\n")}
	result := StepCoverageThresholdScopedFunc(runner, []SpecVerification{
		{SpecID: "ROOT", TestCommand: "go test ./...", CoverageThreshold: 80, File: "specs/root.spec.md"},
		{SpecID: "UNRELATED", TestCommand: "semgrep --test standards/go/testdata", CoverageThreshold: 100, File: "specs/unrelated.spec.md", ImplementationPackage: "standards/go"},
	}, newGateScope("", GateScopeModeDiff, []string{"pkg/gate/step_coverage.go"}, nil))(context.Background())
	if len(result.Violations) != 0 {
		t.Fatalf("expected root command threshold 80 to pass, got %#v", result.Violations)
	}
}

func TestGate_CoverageThreshold_CodeScopePrefersSpecificSpecOverRootCommand(t *testing.T) {
	runner := &mockCommandRunner{output: []byte("ok  \tpkg/gate\t1.234s\tcoverage: 85.0% of statements\n")}
	result := StepCoverageThresholdScopedFunc(runner, []SpecVerification{
		{SpecID: "SPECIFIC", TestCommand: "go test ./pkg/gate", CoverageThreshold: 90, File: "specs/specific.spec.md", ImplementationPackage: "pkg/gate"},
		{SpecID: "ROOT", TestCommand: "go test ./...", CoverageThreshold: 80, File: "specs/root.spec.md"},
	}, newGateScope("", GateScopeModeDiff, []string{"pkg/gate/step_coverage.go"}, nil))(context.Background())
	if len(result.Violations) != 1 {
		t.Fatalf("expected specific spec threshold violation, got %#v", result.Violations)
	}
	if result.Violations[0].Message != "changed Go package coverage 85.0% below threshold 90%" {
		t.Fatalf("unexpected violation message: %q", result.Violations[0].Message)
	}
}

func TestGate_CoverageThreshold_RelevanceIgnoresNonGoAndTestdataFiles(t *testing.T) {
	spec := SpecVerification{SpecID: "GATE", TestCommand: "go test ./pkg/gate", CoverageThreshold: 90, ImplementationPackage: "pkg/gate"}
	if coverageSpecRelevantToCodeScope(spec, nil, false) {
		t.Fatal("expected nil scope not to be relevant")
	}
	if coverageSpecRelevantToCodeScope(spec, newGateScope("", GateScopeModeDiff, nil, nil), false) {
		t.Fatal("expected empty scope not to be relevant")
	}
	if coverageSpecRelevantToFile(spec, "pkg/gate/foo_testdata.go", false) {
		t.Fatal("expected _testdata.go file to be ignored")
	}
	if coverageSpecRelevantToFile(spec, "pkg/gate/README.md", false) {
		t.Fatal("expected non-Go file to be ignored")
	}
	if !coverageSpecRelevantToFile(SpecVerification{TestCommand: "go test ./pkg/gate", CoverageThreshold: 90}, "pkg/gate/step_coverage.go", false) {
		t.Fatal("expected test_command package to make spec relevant")
	}
	if coverageSpecRelevantToFile(SpecVerification{CoverageThreshold: 90}, "pkg/gate/step_coverage.go", false) {
		t.Fatal("expected spec without implementation package or test_command not to be relevant")
	}
}

func TestGate_CoverageThreshold_PackagePathMatchesNestedPackages(t *testing.T) {
	if !packagePathMatches("", "") {
		t.Fatal("expected empty changed package to match empty implementation package")
	}
	if packagePathMatches("pkg/gate", "") {
		t.Fatal("expected non-empty changed package not to match empty implementation package")
	}
	if !packagePathMatches("pkg/gate/internal", "pkg/gate") {
		t.Fatal("expected changed nested package to match parent implementation package")
	}
	if !packagePathMatches("pkg/gate", "pkg/gate/internal") {
		t.Fatal("expected parent changed package to match nested implementation package")
	}
	if packagePathMatches("pkg/gate", "pkg/config") {
		t.Fatal("expected unrelated packages not to match")
	}
}

func TestGate_CoverageOutputExcerpt_SelectsFailureLines(t *testing.T) {
	output := []byte(strings.Join([]string{
		"ok  \tpkg/gate\t1.234s\tcoverage: 85.0% of statements",
		"--- FAIL: TestOne (0.00s)",
		"step_coverage.go:99: failed",
		"FAIL",
		"pkg/gate: error detail",
		"extra: ignored after cap",
	}, "\n"))
	got := coverageOutputExcerpt(output)
	if !strings.Contains(got, "--- FAIL: TestOne") || !strings.Contains(got, "step_coverage.go:99: failed") {
		t.Fatalf("expected failure details in excerpt, got %q", got)
	}
	if strings.Contains(got, "extra: ignored") {
		t.Fatalf("expected excerpt to cap selected lines, got %q", got)
	}
}

func TestGate_CoverageOutputExcerpt_EmptyWhenNoFailureDetails(t *testing.T) {
	if got := coverageOutputExcerpt([]byte("ok  \tpkg/gate\t1.234s\tcoverage: 85.0% of statements\n")); got != "" {
		t.Fatalf("expected empty excerpt, got %q", got)
	}
}

func TestGate_CoverageSpecInScope(t *testing.T) {
	spec := SpecVerification{File: "specs/SPEC-010-gate.spec.md"}
	if !coverageSpecInScope(spec, nil) {
		t.Fatal("expected nil scope to include spec")
	}
	if !coverageSpecInScope(spec, newGateScope("", GateScopeModeAll, nil, nil)) {
		t.Fatal("expected all scope to include spec")
	}
	if coverageSpecInScope(spec, newGateScope("", GateScopeModeDiff, nil, nil)) {
		t.Fatal("expected empty diff scope to exclude spec")
	}
	if !coverageSpecInScope(spec, newGateScope("", GateScopeModeDiff, []string{"specs/SPEC-010-gate.spec.md"}, nil)) {
		t.Fatal("expected matching spec file scope to include spec")
	}
	if !coverageSpecInScope(spec, newGateScope("", GateScopeModeDiff, []string{"pkg/gate/step_coverage.go"}, nil)) {
		t.Fatal("expected Go code scope to include coverage spec")
	}
}

func TestGate_NormalizeCoveragePackageAndContainsFile(t *testing.T) {
	cases := map[string]string{
		"./pkg/gate/...": "pkg/gate",
		"/pkg/gate/":     "pkg/gate",
		"...":            "",
	}
	for input, want := range cases {
		if got := normalizeCoveragePackage(input); got != want {
			t.Fatalf("normalizeCoveragePackage(%q) = %q, want %q", input, got, want)
		}
	}
	if !coveragePackageContainsFile("./pkg/gate/...", "pkg/gate/step_coverage.go") {
		t.Fatal("expected package to contain file")
	}
	if !coveragePackageContainsFile("...", "pkg/anything/file.go") {
		t.Fatal("expected root wildcard package to contain any file")
	}
	if coveragePackageContainsFile("./pkg/gate", "pkg/config/config.go") {
		t.Fatal("expected unrelated package not to contain file")
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
