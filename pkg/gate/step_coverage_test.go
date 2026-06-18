package gate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

// perPackageCommandRunner returns a distinct coverage output for each changed
// package, keyed by the package label that appears in the `go test <pkg>` args.
// It lets per-changed-package coverage tests assert that each package is checked
// against the threshold independently rather than as a single aggregate.
type perPackageCommandRunner struct {
	coverageByPkg map[string]float64 // pkg label -> coverage percent
	runs          int
	ranPkgs       []string
}

func (r *perPackageCommandRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	r.runs++
	pkg := ""
	for _, arg := range args {
		if arg == "test" || strings.HasPrefix(arg, "-") {
			continue
		}
		pkg = arg
		break
	}
	r.ranPkgs = append(r.ranPkgs, pkg)
	pct, ok := r.coverageByPkg[pkg]
	if !ok {
		return nil, fmt.Errorf("no Go files in %s", pkg)
	}
	return []byte(fmt.Sprintf("ok  \t%s\t1.0s\tcoverage: %.1f%% of statements\n", pkg, pct)), nil
}

// TestGate_CoverageThreshold_CollapsedScope_PerChangedPackage verifies that the
// collapsed-code-scope path checks EACH changed Go package against the threshold
// independently — not as a whole-repo aggregate — and that a deleted package is
// excluded from measurement rather than failing on a missing directory.
func TestGate_CoverageThreshold_CollapsedScope_PerChangedPackage(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"pkg/alpha", "pkg/beta"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// pkg/gone is intentionally NOT created on disk: it models a deleted package
	// whose .go file still appears in the diff.

	// Spec is unrelated to the changed Go packages, so the collapsed path uses
	// the default unit floor (90%).
	specs := []SpecVerification{
		{SpecID: "UNRELATED", TestCommand: "semgrep --test standards", CoverageThreshold: 100, File: "specs/unrelated.spec.md", ImplementationPackage: "standards/go"},
	}

	cases := []struct {
		name           string
		coverageByPkg  map[string]float64
		wantStatus     string
		wantViolations []string
		wantRanPkgs    []string
	}{
		{
			name:          "all changed packages above floor",
			coverageByPkg: map[string]float64{"./pkg/alpha": 95.0, "./pkg/beta": 91.0},
			wantStatus:    "pass",
			wantRanPkgs:   []string{"./pkg/alpha", "./pkg/beta"},
		},
		{
			name:          "one changed package below floor",
			coverageByPkg: map[string]float64{"./pkg/alpha": 95.0, "./pkg/beta": 80.0},
			wantStatus:    "fail",
			wantViolations: []string{
				"changed Go package ./pkg/beta coverage 80.0% below threshold 90%",
			},
			wantRanPkgs: []string{"./pkg/alpha", "./pkg/beta"},
		},
		{
			name:          "both changed packages below floor each reported",
			coverageByPkg: map[string]float64{"./pkg/alpha": 70.0, "./pkg/beta": 80.0},
			wantStatus:    "fail",
			wantViolations: []string{
				"changed Go package ./pkg/alpha coverage 70.0% below threshold 90%",
				"changed Go package ./pkg/beta coverage 80.0% below threshold 90%",
			},
			wantRanPkgs: []string{"./pkg/alpha", "./pkg/beta"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &perPackageCommandRunner{coverageByPkg: tc.coverageByPkg}
			scope := newGateScope(root, GateScopeModeDiff, []string{
				"pkg/alpha/a.go",
				"pkg/beta/b.go",
				"pkg/gone/gone.go", // deleted package: excluded, never measured
			}, nil)
			result := StepCoverageThresholdScopedFunc(runner, specs, scope)(context.Background())

			if result.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q; violations=%#v", result.Status, tc.wantStatus, result.Violations)
			}
			if !slices.Equal(runner.ranPkgs, tc.wantRanPkgs) {
				t.Fatalf("measured packages = %v, want %v (deleted pkg/gone must be excluded)", runner.ranPkgs, tc.wantRanPkgs)
			}
			gotMsgs := make([]string, 0, len(result.Violations))
			for _, v := range result.Violations {
				gotMsgs = append(gotMsgs, v.Message)
			}
			if len(tc.wantViolations) == 0 {
				if len(gotMsgs) != 0 {
					t.Fatalf("expected no violations, got %v", gotMsgs)
				}
				return
			}
			for _, want := range tc.wantViolations {
				if !slices.Contains(gotMsgs, want) {
					t.Fatalf("missing expected violation %q in %v", want, gotMsgs)
				}
			}
			if len(gotMsgs) != len(tc.wantViolations) {
				t.Fatalf("violation count = %d (%v), want %d (%v)", len(gotMsgs), gotMsgs, len(tc.wantViolations), tc.wantViolations)
			}
		})
	}
}

// TestGate_CoverageThreshold_CollapsedScope_NotAggregateAcrossPackages proves the
// fix: a changed package over the floor must NOT be failed by an unrelated low
// package via averaging. Under the old whole-repo aggregate this would have
// measured one combined number; per-package, the high package passes on its own.
func TestGate_CoverageThreshold_CollapsedScope_NotAggregateAcrossPackages(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"pkg/high", "pkg/alsohigh"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	runner := &perPackageCommandRunner{coverageByPkg: map[string]float64{
		"./pkg/high":     90.2,
		"./pkg/alsohigh": 91.1,
	}}
	specs := []SpecVerification{
		{SpecID: "UNRELATED", TestCommand: "semgrep --test standards", CoverageThreshold: 100, File: "specs/unrelated.spec.md", ImplementationPackage: "standards/go"},
	}
	scope := newGateScope(root, GateScopeModeDiff, []string{
		"pkg/high/h.go",
		"pkg/alsohigh/a.go",
		"specs/some.spec.md", // must NOT trigger a whole-repo sweep in this path
	}, nil)

	result := StepCoverageThresholdScopedFunc(runner, specs, scope)(context.Background())

	if result.Status != "pass" {
		t.Fatalf("expected pass when every changed package is over the floor, got %q: %#v", result.Status, result.Violations)
	}
	for _, pkg := range runner.ranPkgs {
		if strings.Contains(pkg, "...") {
			t.Fatalf("collapsed per-package path must not run a whole-repo sweep, ran %v", runner.ranPkgs)
		}
	}
	if !slices.Equal(runner.ranPkgs, []string{"./pkg/alsohigh", "./pkg/high"}) {
		t.Fatalf("expected exactly the two changed packages measured, got %v", runner.ranPkgs)
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
	if result.Violations[0].Message != "changed Go package ./pkg/gate coverage 75.0% below threshold 80%" {
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
	if result.Violations[0].Message != "changed Go package ./pkg/gate coverage 85.0% below threshold 90%" {
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
	if result.Violations[0].Message != "changed Go package ./pkg/gate coverage 85.0% below threshold 90%" {
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

// TestGate_RepoSweepCoverageTarget_UnknownRoot verifies that an empty
// projectRoot yields the full canonical sweep across all four package roots
// unchanged, since nothing can be pruned without a root to stat against.
func TestGate_RepoSweepCoverageTarget_UnknownRoot(t *testing.T) {
	target := repoSweepCoverageTarget("")

	if target.Stack != "go" {
		t.Errorf("expected stack %q, got %q", "go", target.Stack)
	}
	wantArgs := []string{"test", ".", "./cmd/...", "./pkg/...", "./tests/...", "-coverprofile=/dev/null"}
	if !slices.Equal(target.Args, wantArgs) {
		t.Errorf("expected args %v, got %v", wantArgs, target.Args)
	}
}

// TestGate_RepoSweepCoverageTarget_PrunesMissingRoots verifies that top-level
// roots absent on disk are pruned, while present ones are kept, and that a
// module root holding buildable .go files contributes the "." package.
func TestGate_RepoSweepCoverageTarget_PrunesMissingRoots(t *testing.T) {
	root := t.TempDir()
	// Module root has a buildable .go file -> "." included.
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Only ./pkg exists; cmd and tests are absent and must be pruned.
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}

	target := repoSweepCoverageTarget(root)

	wantArgs := []string{"test", ".", "./pkg/...", "-coverprofile=/dev/null"}
	if !slices.Equal(target.Args, wantArgs) {
		t.Errorf("expected args %v, got %v", wantArgs, target.Args)
	}
}

// TestGate_RepoSweepCoverageTarget_NoModuleGoFiles verifies that a module root
// that nests packages but holds no buildable .go files of its own omits the "."
// package, keeping only the present top-level roots.
func TestGate_RepoSweepCoverageTarget_NoModuleGoFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}

	target := repoSweepCoverageTarget(root)

	wantArgs := []string{"test", "./cmd/...", "./pkg/...", "-coverprofile=/dev/null"}
	if !slices.Equal(target.Args, wantArgs) {
		t.Errorf("expected args %v, got %v", wantArgs, target.Args)
	}
}

// TestGate_RepoSweepCoverageTarget_NothingResolvesFallsBack verifies that when
// no buildable package root resolves under projectRoot the function falls back
// to the canonical sweep so the step surfaces a real failure instead of
// silently measuring nothing.
func TestGate_RepoSweepCoverageTarget_NothingResolvesFallsBack(t *testing.T) {
	root := t.TempDir() // empty: no .go at root, no cmd/pkg/tests dirs.

	target := repoSweepCoverageTarget(root)

	wantArgs := []string{"test", ".", "./cmd/...", "./pkg/...", "./tests/...", "-coverprofile=/dev/null"}
	if !slices.Equal(target.Args, wantArgs) {
		t.Errorf("expected canonical fallback args %v, got %v", wantArgs, target.Args)
	}
}

// TestGate_DirHasGoFiles verifies detection of a directly-contained .go source
// file, ignoring subdirectories (even .go-suffixed ones) and missing dirs.
func TestGate_DirHasGoFiles(t *testing.T) {
	t.Run("with go file", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if !dirHasGoFiles(dir) {
			t.Error("expected dirHasGoFiles to report true when a .go file is present")
		}
	})

	t.Run("only non-go files", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if dirHasGoFiles(dir) {
			t.Error("expected dirHasGoFiles to report false with no .go file present")
		}
	})

	t.Run("go-suffixed subdir ignored", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "nested.go"), 0o755); err != nil {
			t.Fatal(err)
		}
		if dirHasGoFiles(dir) {
			t.Error("expected dirHasGoFiles to ignore directories named like .go files")
		}
	})

	t.Run("missing directory", func(t *testing.T) {
		if dirHasGoFiles(filepath.Join(t.TempDir(), "does-not-exist")) {
			t.Error("expected dirHasGoFiles to report false for a missing directory")
		}
	})
}

// TestGate_IsTestdataPath verifies that paths under a testdata directory at any
// depth are recognized, while unrelated paths and testdata substrings within a
// segment name are not.
func TestGate_IsTestdataPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"testdata/fixture.go", true},
		{"pkg/gate/testdata/contract-target.go", true},
		{"testdata", true},
		{"pkg/gate/step_coverage.go", false},
		{"pkg/mytestdata/file.go", false},
		{"pkg/testdataish/file.go", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isTestdataPath(tc.path); got != tc.want {
			t.Errorf("isTestdataPath(%q): got %v, want %v", tc.path, got, tc.want)
		}
	}
}

// TestGate_CoveragePackageDirMissing verifies the package-directory existence
// probe: unknown roots and the module root are never missing, an absent scoped
// dir is missing, a present dir is not, and a path that is a file (not a dir) is
// treated as missing.
func TestGate_CoveragePackageDirMissing(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg", "present"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "afile"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name        string
		projectRoot string
		dir         string
		want        bool
	}{
		{name: "unknown root", projectRoot: "", dir: "pkg/present", want: false},
		{name: "empty dir", projectRoot: root, dir: "", want: false},
		{name: "module root dot", projectRoot: root, dir: ".", want: false},
		{name: "present dir", projectRoot: root, dir: "pkg/present", want: false},
		{name: "absent dir", projectRoot: root, dir: "pkg/gone", want: true},
		{name: "path is a file not a dir", projectRoot: root, dir: "pkg/afile", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := coveragePackageDirMissing(tc.projectRoot, tc.dir); got != tc.want {
				t.Errorf("coveragePackageDirMissing(%q, %q): got %v, want %v", tc.projectRoot, tc.dir, got, tc.want)
			}
		})
	}
}
