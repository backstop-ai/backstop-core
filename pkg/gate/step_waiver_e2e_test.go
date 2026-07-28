package gate

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
)

// waiverE2EFixtureDir resolves the committed installed-pack waiver e2e fixture.
func waiverE2EFixtureDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", "waiver-e2e"))
	if err != nil {
		t.Fatalf("resolving waiver-e2e fixture dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "fake-engine.sh")); err != nil {
		t.Fatalf("committed fake-engine.sh must exist: %v", err)
	}
	return dir
}

// setupWaiverE2ETemp copies the fake engine + source into a fresh temp project so
// the fake self-targets there (writing findings.sarif into cwd) without polluting
// the committed testdata tree, and sets the scenario env the fake reads.
func setupWaiverE2ETemp(t *testing.T, scenario string) string {
	t.Helper()
	fixture := waiverE2EFixtureDir(t)
	temp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(temp, "src"), 0o755); err != nil {
		t.Fatalf("mkdir temp src: %v", err)
	}
	copyFileMode(t, filepath.Join(fixture, "fake-engine.sh"), filepath.Join(temp, "fake-engine.sh"), 0o755)
	copyFileMode(t, filepath.Join(fixture, "src", "app.go"), filepath.Join(temp, "src", "app.go"), 0o644)
	t.Setenv("WAIVER_E2E_SCENARIO", scenario)
	return temp
}

func copyFileMode(t *testing.T, src, dst string, mode os.FileMode) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("reading %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, mode); err != nil {
		t.Fatalf("writing %s: %v", dst, err)
	}
}

// realPackEnginesStep runs the committed fake engine via a REAL check.CommandRunner
// and builds a pack_engines StepResult from its SARIF findings — the LIVE
// pack_engines stream, not a stub.
func realPackEnginesStep(t *testing.T, tempRoot string) StepResult {
	t.Helper()
	runner := &check.ExecCommandRunner{Dir: tempRoot}
	if _, err := runner.RunStdout(context.Background(), "sh", "fake-engine.sh"); err != nil {
		t.Fatalf("running the committed fake engine: %v", err)
	}
	sarif, err := os.ReadFile(filepath.Join(tempRoot, "findings.sarif"))
	if err != nil {
		t.Fatalf("reading the engine's declared stdout_artifact: %v", err)
	}
	findings, err := check.ParsePackFindings(sarif)
	if err != nil {
		t.Fatalf("parsing pack findings SARIF: %v", err)
	}
	vs := make([]Violation, 0, len(findings))
	for _, f := range findings {
		vs = append(vs, Violation{
			Rule:     "backstop/waiver-e2e/" + f.Rule,
			File:     f.File,
			Line:     f.Line,
			Severity: f.Severity,
		})
	}
	return StepResult{StepName: StepPackEngines, Status: statusFor(vs), Violations: vs}
}

// fileLineReader builds a waiver LineReader that reads real source lines from the
// temp project root, keyed by the repo-relative file path the finding carries.
func fileLineReader(root string) func(string, int) (string, bool) {
	return func(file string, line int) (string, bool) {
		f, err := os.Open(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return "", false
		}
		defer func() { _ = f.Close() }()
		scanner := bufio.NewScanner(f)
		n := 0
		for scanner.Scan() {
			n++
			if n == line {
				return scanner.Text(), true
			}
		}
		return "", false
	}
}

// TestGateWaiver_E2E_WaiverSuppressesRealPackEnginesFinding proves a @waiver
// SUPPRESSES a real pack_engines finding in a full gate Run over the committed
// fake-engine fixture, terminating in the distinct PASS·N-waivers state — the
// finding is really removed from the accumulated pack_engines Violations
// (CLM-067, CLM-041). Fails against the deferred stub (empty waivable surface).
func TestGateWaiver_E2E_WaiverSuppressesRealPackEnginesFinding(t *testing.T) {
	temp := setupWaiverE2ETemp(t, "waivable")
	pe := realPackEnginesStep(t, temp)
	if len(pe.Violations) != 1 || pe.Violations[0].Rule != "backstop/waiver-e2e/waivable-defect" {
		t.Fatalf("fixture must emit exactly the waivable finding, got %#v", pe.Violations)
	}

	steps := []StepFunc{
		func(context.Context) StepResult { return pe },
		StepWaiverResolutionScopedFunc(nil),
	}
	reader := fileLineReader(temp)
	g := New(WithSteps(steps), WithWaiver(reader, nil, waiverTestNow))
	res, _ := g.Run(context.Background())

	if stepHasViolation(res.Steps, StepPackEngines, "src/app.go", 5, "backstop/waiver-e2e/waivable-defect") {
		t.Fatal("the real pack_engines finding was NOT suppressed by the co-located @waiver")
	}
	if len(res.ActiveWaivers) != 1 {
		t.Fatalf("expected 1 active waiver over the real dispatch, got %d", len(res.ActiveWaivers))
	}
	for _, s := range res.Steps {
		if s.StepName == StepWaiverResolution && !strings.Contains(s.Reason, "PASS · 1 waivers") {
			t.Fatalf("waiver step must render PASS·N-waivers, got Reason=%q", s.Reason)
		}
	}
}

// TestGateWaiver_E2E_NonWaivableSelfRuleWaiverIsGateError proves a @waiver on the
// policy-declared non-waivable rule produces a gate ERROR (not a suppression)
// over the real pack_engines stream (CLM-025). Fails against the deferred stub.
func TestGateWaiver_E2E_NonWaivableSelfRuleWaiverIsGateError(t *testing.T) {
	temp := setupWaiverE2ETemp(t, "protected")
	pe := realPackEnginesStep(t, temp)
	if len(pe.Violations) != 1 || pe.Violations[0].Rule != "backstop/waiver-e2e/protected-defect" {
		t.Fatalf("fixture must emit exactly the protected finding, got %#v", pe.Violations)
	}

	steps := []StepFunc{
		func(context.Context) StepResult { return pe },
		StepWaiverResolutionScopedFunc(nil),
	}
	reader := fileLineReader(temp)
	// The declared non-waivable rule (extracted in production from the pack manifest).
	policy := newTestPolicyNonWaivable("backstop/waiver-e2e/protected-defect")
	g := New(WithSteps(steps), WithWaiver(reader, policy, waiverTestNow))
	res, _ := g.Run(context.Background())

	if !res.hasWaiverNonWaivableError("backstop/waiver-e2e/protected-defect") {
		t.Fatal("a @waiver on a declared non-waivable rule must raise a gate ERROR over the real dispatch")
	}
	if !stepHasViolation(res.Steps, StepPackEngines, "src/app.go", 10, "backstop/waiver-e2e/protected-defect") {
		t.Fatal("a non-waivable finding must NOT be suppressed; it must stand")
	}
}

// hasWaiverNonWaivableError reports whether the waiver_resolution step raised a
// non-waivable gate ERROR naming the rule.
func (r GateResult) hasWaiverNonWaivableError(rule string) bool {
	for _, s := range r.Steps {
		if s.StepName != StepWaiverResolution {
			continue
		}
		if s.Status != "fail" {
			return false
		}
		for _, v := range s.Violations {
			if v.Rule == rule {
				return true
			}
		}
	}
	return false
}
