package packval_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

func TestPackVal_DefaultExecutorImplementsInterface(t *testing.T) {
	// Compile-time assertion that DefaultExecutor satisfies the interface:
	// this blank assignment fails to compile if it ever stops satisfying it.
	var _ packval.FixtureExecutor = (*packval.DefaultExecutor)(nil) // nosemgrep: go.core.no-global-mutable-state — blank compile-time interface assertion, not mutable state
	// Runtime behavior check: the validator seam returns a non-fired result for a
	// missing validator rather than panicking.
	d := &packval.DefaultExecutor{}
	r, err := d.RunValidator(t.TempDir(), "/nonexistent/v.sh", []string{"x"})
	if err != nil {
		t.Fatalf("RunValidator should absorb exec failure, got err: %v", err)
	}
	if r.Passed {
		t.Fatal("a missing validator must not report a passing result")
	}
}
func TestPackVal_MockExecutorEngine(t *testing.T) {
	m := &packval.MockExecutor{EngineFn: func(_ string, _ engine.EngineBinding, _ []string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: true}, nil
	}}
	r, err := m.RunEngine("", engine.EngineBinding{}, nil)
	if err != nil || !r.Passed {
		t.Fatalf("expected pass, err=%v", err)
	}
}
func TestPackVal_MockExecutorValidator(t *testing.T) {
	m := &packval.MockExecutor{ValidatorFn: func(_, _ string, _ []string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: true}, nil
	}}
	r, err := m.RunValidator("", "", nil)
	if err != nil || !r.Passed {
		t.Fatalf("expected pass, err=%v", err)
	}
}
func TestPackVal_MockExecutorScaffoldTest(t *testing.T) {
	m := &packval.MockExecutor{ScaffoldTestFn: func(_, _, _ string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: true}, nil
	}}
	r, err := m.RunScaffoldTest("", "", "")
	if err != nil || !r.Passed {
		t.Fatalf("expected pass, err=%v", err)
	}
}

func TestPackVal_P3_SandboxBlocksFilesystemWrite(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	dir := makePackDir(t)
	_, err := packval.SandboxedRun("sh", []string{"-c", "echo x > new.txt"}, dir)
	if err == nil {
		t.Fatal("expected sandbox write failure")
	}
}
func TestPackVal_P3_SandboxBlocksNetwork(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	dir := makePackDir(t)
	_, err := packval.SandboxedRun("sh", []string{"-c", "nc -z 127.0.0.1 80"}, dir)
	if err == nil {
		t.Fatal("expected sandbox network failure")
	}
}

// TestPackVal_P3_SandboxIsFilesystemNetworkScopedNotEnvJail documents the true
// scope of the macOS sandbox: sandbox-exec confines FILESYSTEM and NETWORK
// access, it does NOT scrub the inherited environment. The earlier
// "BlocksEnvVars" assertion was vacuous green — `printenv HOME` only "failed"
// because the dynamically-linked `sh` SIGABRT'd at dyld load under the old
// packDir-only profile (the exact ISSUE-029 bug). With the dyld read paths in
// place, `sh` runs and `printenv HOME` succeeds; env vars were never sandboxed.
// The genuine confinement (write/network denied) is covered by
// TestPackVal_P3_SandboxBlocksFilesystemWrite / ...BlocksNetwork.
func TestPackVal_P3_SandboxIsFilesystemNetworkScopedNotEnvJail(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	dir := makePackDir(t)
	out, err := packval.SandboxedRun("sh", []string{"-c", "printenv HOME"}, dir)
	if err != nil {
		t.Fatalf("sh must run under the sandbox (no dyld abort) so env passthrough is observable: %v\noutput: %q", err, string(out))
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		t.Fatalf("expected HOME to pass through the sandbox (env is not jailed), got empty output")
	}
}
func TestPackVal_P3_SandboxViolationIsHardError(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	dir := makePackDir(t)
	_, err := packval.SandboxedRun("sh", []string{"-c", "touch x"}, dir)
	if err == nil {
		t.Fatal("expected error")
	}
}
func TestPackVal_P3_SandboxAllowsReadInPackDir(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	dir := makePackDir(t)
	out, err := packval.SandboxedRun("cat", []string{"pack.yml"}, dir)
	if err != nil || !strings.Contains(string(out), "name:") {
		t.Fatalf("expected read allowed: %v", err)
	}
}
func TestPackVal_P3_SandboxAllowsExecution(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	dir := makePackDir(t)
	out, err := packval.SandboxedRun("sh", []string{"-c", "echo ok"}, dir)
	if err != nil || !strings.Contains(string(out), "ok") {
		t.Fatalf("expected exec allowed: %v", err)
	}
}

// targetsAreNegative reports whether the dispatched targets reference a negative
// fixture (named with an "n." segment). The generic RunEngine seam receives the
// rule/config file plus the fixture path as targets, so the mock keys on the fixture.
func targetsAreNegative(targets []string) bool {
	for _, t := range targets {
		if strings.Contains(t, "n.") {
			return true
		}
	}
	return false
}

// newFixtureMock configures whether each fixture's engine run FIRES. ExecutionResult
// .Passed on the findings seam means "the engine produced findings" — see RunEngine's
// own doc comment — so the parameters say `Fires`, not `passes`. The old spelling
// (`passPos`, `failNeg`, with a `Passed: !failNeg` double negative) is a large part of
// why the inverted polarity survived review for so long (ISSUE-092).
//
// Under BUNDLE-005 REQ-011 the HEALTHY configuration is therefore
// newFixtureMock(false, true): a positive fixture is the clean example and must NOT
// fire, a negative fixture is the violating one and MUST fire.
func newFixtureMock(posFires bool, negFires bool) *packval.MockExecutor {
	return &packval.MockExecutor{
		EngineFn: func(_ string, _ engine.EngineBinding, targets []string) (packval.ExecutionResult, error) {
			if targetsAreNegative(targets) {
				return packval.ExecutionResult{Passed: negFires, Output: "R1"}, nil
			}
			return packval.ExecutionResult{Passed: posFires, Output: "R1"}, nil
		},
		ValidatorFn: func(_, _ string, fixtures []string) (packval.ExecutionResult, error) {
			for _, f := range fixtures {
				if strings.Contains(f, "n.") {
					return packval.ExecutionResult{Passed: false}, nil
				}
			}
			return packval.ExecutionResult{Passed: true}, nil
		},
		ScaffoldTestFn: func(_, _, _ string) (packval.ExecutionResult, error) {
			return packval.ExecutionResult{Passed: true}, nil
		},
	}
}

// hasCheck reports whether any phase error carries the given Check identifier.
func hasCheck(errs []packval.ValidationError, check string) bool {
	for _, e := range errs {
		if e.Check == check {
			return true
		}
	}
	return false
}

// The four tests below state BUNDLE-005 REQ-011's fixture contract. Their NAMES always
// said it correctly; before ISSUE-092 their BODIES asserted its inverse, and all four
// were configured with byte-identical mock calls so none of them discriminated. Each
// now carries the mock configuration its own name describes.

// TestPackVal_P3_SemgrepPositivePass: a positive fixture is the CLEAN example. It must
// not trigger the rule, and a run in which it does not is a pass.
func TestPackVal_P3_SemgrepPositivePass(t *testing.T) {
	res := packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(false, true))
	if res.Status != "pass" {
		t.Fatalf("a positive fixture that does not fire must pass; got %s: %+v", res.Status, res.Errors)
	}
}

// TestPackVal_P3_SemgrepPositiveFalsePositive: a positive fixture that DOES trigger the
// rule is a FALSE POSITIVE and must fail, identified as the positive path.
func TestPackVal_P3_SemgrepPositiveFalsePositive(t *testing.T) {
	res := packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(true, true))
	if res.Status != "fail" {
		t.Fatalf("a positive fixture that fires is a false positive and must fail; got %s", res.Status)
	}
	if !hasCheck(res.Errors, "semgrep-positive") {
		t.Fatalf("the failure must be identified as the positive path; got %+v", res.Errors)
	}
}

// TestPackVal_P3_SemgrepNegativeAllTrigger: a negative fixture is the VIOLATING
// example. Every negative triggering its rule is the passing case.
func TestPackVal_P3_SemgrepNegativeAllTrigger(t *testing.T) {
	res := packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(false, true))
	if res.Status != "pass" {
		t.Fatalf("negative fixtures that all trigger must pass; got %s: %+v", res.Status, res.Errors)
	}
}

// TestPackVal_P3_SemgrepNegativeNotTriggered: a negative fixture that does NOT trigger
// its rule is a failure — the case the shipped FixHint was always written for.
func TestPackVal_P3_SemgrepNegativeNotTriggered(t *testing.T) {
	res := packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(false, false))
	if res.Status != "fail" {
		t.Fatalf("a negative fixture that does not fire must fail; got %s", res.Status)
	}
	if !hasCheck(res.Errors, "semgrep-negative") {
		t.Fatalf("the failure must be identified as the negative path; got %+v", res.Errors)
	}
}

// TestPackVal_P3_SemgrepRuleIDMismatch runs on a POLARITY-CORRECT base mock, so its
// fail verdict is provably produced by the rule-ID mismatch and not incidentally by
// broken fixtures. That distinction is load-bearing: with a polarity-inverted base
// mock BOTH fixtures also fail, and this test would keep reporting "fail" — and keep
// looking green — even if the rule-ID cross-check stopped working entirely.
func TestPackVal_P3_SemgrepRuleIDMismatch(t *testing.T) {
	m := baseManifest()
	dir := makePackDir(t)
	writeFile(t, dir, "rules/r1.yml", "rules:\n  - id: WRONG_ID\n")
	r := packval.RunFixtures(m, dir, newFixtureMock(false, true))
	if r.Status == "pass" {
		t.Fatal("expected fail when rule ID doesn't match file")
	}
	if !hasCheck(r.Errors, "semgrep-rule-id") {
		t.Fatalf("expected semgrep-rule-id error; got %+v", r.Errors)
	}
	if hasCheck(r.Errors, "semgrep-positive") || hasCheck(r.Errors, "semgrep-negative") {
		t.Fatalf("the fixtures must be clean, or this test is failing for the polarity reason "+
			"rather than the rule-ID reason and has stopped discriminating; got %+v", r.Errors)
	}
}

// TestPackVal_P3_SemgrepRuleIDMatch: the same manifest whose rule file DOES carry the
// declared id passes. This is the control for the mismatch test above.
func TestPackVal_P3_SemgrepRuleIDMatch(t *testing.T) {
	res := packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(false, true))
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s: %+v", res.Status, res.Errors)
	}
}
func TestPackVal_P3_ToolConfigDispatchesGeneric(t *testing.T) {
	// The tool_config fixture path routes through the generic engine dispatch, no
	// baked Go module-tidy pre-flight (ISSUE-019).
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{ID: "T1", Engine: "config-file", Tool: "golangci-lint", File: ".golangci.yml", RiskClass: "style", Claims: m.Content.Ruleset.Rules[0].Claims}}
	dir := makePackDir(t)
	writeFile(t, dir, ".golangci.yml", "")
	r := packval.RunFixtures(m, dir, newFixtureMock(false, true))
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s: %+v", r.Status, r.Errors)
	}
}
func TestPackVal_P3_ToolConfigPositiveClean(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{ID: "T1", Engine: "config-file", Tool: "golangci-lint", File: ".golangci.yml", RiskClass: "style", Claims: m.Content.Ruleset.Rules[0].Claims}}
	dir := makePackDir(t)
	writeFile(t, dir, ".golangci.yml", "")
	if packval.RunFixtures(m, dir, newFixtureMock(false, true)).Status != "pass" {
		t.Fatal("pass")
	}
}

// TestPackVal_P3_ToolConfigNegativeNotTriggered: the tool_config loop carries the same
// BUNDLE-005 REQ-011 contract as the ruleset loop. The base mock stays polarity-CORRECT
// so the failure is provably produced by the tool_config negative alone, not by an
// incidentally-broken ruleset fixture.
func TestPackVal_P3_ToolConfigNegativeNotTriggered(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{ID: "T1", Engine: "config-file", Tool: "golangci-lint", File: ".golangci.yml", RiskClass: "style", Claims: m.Content.Ruleset.Rules[0].Claims}}
	dir := makePackDir(t)
	writeFile(t, dir, ".golangci.yml", "")
	mock := newFixtureMock(false, true)
	base := mock.EngineFn
	// The tool_config entry dispatches with its CONFIG file as the first target; the
	// ruleset rule dispatches with its rule file. Keying on that keeps the ruleset
	// fixtures at correct polarity while forcing the tool_config negative not to fire.
	mock.EngineFn = func(packDir string, b engine.EngineBinding, targets []string) (packval.ExecutionResult, error) {
		if len(targets) > 0 && targets[0] == ".golangci.yml" {
			return packval.ExecutionResult{Passed: false}, nil
		}
		return base(packDir, b, targets)
	}
	res := packval.RunFixtures(m, dir, mock)
	if res.Status != "fail" {
		t.Fatalf("a tool_config negative fixture that does not fire must fail; got %s", res.Status)
	}
	if !hasCheck(res.Errors, "tool-config-negative") {
		t.Fatalf("the failure must be the tool_config negative path; got %+v", res.Errors)
	}
	if hasCheck(res.Errors, "semgrep-positive") || hasCheck(res.Errors, "semgrep-negative") {
		t.Fatalf("the ruleset fixtures must stay clean, or this test is not discriminating; got %+v", res.Errors)
	}
}
func TestPackVal_P3_ToolConfigNegativeTriggered(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{ID: "T1", Engine: "config-file", Tool: "golangci-lint", File: ".golangci.yml", RiskClass: "style", Claims: m.Content.Ruleset.Rules[0].Claims}}
	dir := makePackDir(t)
	writeFile(t, dir, ".golangci.yml", "")
	if packval.RunFixtures(m, dir, newFixtureMock(false, true)).Status != "pass" {
		t.Fatal("pass")
	}
}

// The go-mod-tidy pre-flight tests are retired (ISSUE-019): packval bakes no Go
// module-tidy step, so removing go.mod no longer fails a tool_config fixture run.
// TestPackVal_P3_NegativeFixtureEngineLimitationHint: the engine-limitation hint
// belongs on the branch its own wording describes — a negative fixture that did NOT
// trigger the rule. Before ISSUE-092 the hint hung off the opposite condition.
func TestPackVal_P3_NegativeFixtureEngineLimitationHint(t *testing.T) {
	r := packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(false, false))
	found := false
	for _, e := range r.Errors {
		if strings.Contains(e.FixHint, "engine limitation") {
			found = true
			if !strings.Contains(e.FixHint, "removing") || !strings.Contains(e.FixHint, "documenting") {
				t.Fatalf("fix hint missing removal/documentation guidance: %s", e.FixHint)
			}
		}
	}
	if !found {
		t.Fatal("expected fix hint")
	}
}
func TestPackVal_P3_Layer3SingleFileInvocation(t *testing.T) {
	m := baseManifest()
	var calls [][]string
	mock := newFixtureMock(false, true)
	mock.ValidatorFn = func(_, _ string, fixturePaths []string) (packval.ExecutionResult, error) {
		captured := append([]string(nil), fixturePaths...)
		calls = append(calls, captured)
		for _, p := range fixturePaths {
			if strings.Contains(p, "n.") {
				return packval.ExecutionResult{Passed: false}, nil
			}
		}
		return packval.ExecutionResult{Passed: true}, nil
	}
	r := packval.RunFixtures(m, makePackDir(t), mock)
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s", r.Status)
	}
	if len(calls) == 0 {
		t.Fatal("validator was never called")
	}
	for i, c := range calls {
		if len(c) != 1 {
			t.Fatalf("call %d: expected 1 fixture path for single-file, got %d", i, len(c))
		}
	}
}
func TestPackVal_P3_Layer3MultiFileInvocation(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].InputScope = "multi-file"
	mock := newFixtureMock(false, true)
	mock.ValidatorFn = func(_, _ string, fixtures []string) (packval.ExecutionResult, error) {
		if len(fixtures) > 1 {
			return packval.ExecutionResult{Passed: true}, nil
		}
		if len(fixtures) == 1 && strings.Contains(fixtures[0], "n.") {
			return packval.ExecutionResult{Passed: false}, nil
		}
		return packval.ExecutionResult{Passed: true}, nil
	}
	if packval.RunFixtures(m, makePackDir(t), mock).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P3_Layer3PositiveExitZero(t *testing.T) {
	if packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(false, true)).Status != "pass" {
		t.Fatal("pass")
	}
}

// TestPackVal_P3_Layer3PositiveExitNonZero: the fail must come from this test's OWN
// ValidatorFn override (a validator exiting non-zero on a positive fixture), not from
// the findings fixtures. The base mock is polarity-CORRECT for exactly that reason —
// with an inverted base both fixtures would also fail and the assertion would be
// satisfied incidentally whether or not the override still worked.
func TestPackVal_P3_Layer3PositiveExitNonZero(t *testing.T) {
	mock := newFixtureMock(false, true)
	mock.ValidatorFn = func(_, _ string, _ []string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: false}, nil
	}
	res := packval.RunFixtures(baseManifest(), makePackDir(t), mock)
	if res.Status != "fail" {
		t.Fatalf("a layer-3 validator exiting non-zero on a positive fixture must fail; got %s", res.Status)
	}
	if !hasCheck(res.Errors, "validator-positive") {
		t.Fatalf("the fail must come from the validator override; got %+v", res.Errors)
	}
}
func TestPackVal_P3_Layer3NegativeExitNonZero(t *testing.T) {
	if packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(false, true)).Status != "pass" {
		t.Fatal("pass")
	}
}

// TestPackVal_P3_Layer3NegativeExitZero: same re-verification discipline as
// Layer3PositiveExitNonZero — the fail must come from this test's own ValidatorFn
// override (a validator exiting ZERO on a negative fixture), against a
// polarity-correct base mock.
func TestPackVal_P3_Layer3NegativeExitZero(t *testing.T) {
	mock := newFixtureMock(false, true)
	mock.ValidatorFn = func(_, _ string, _ []string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: true}, nil
	}
	res := packval.RunFixtures(baseManifest(), makePackDir(t), mock)
	if res.Status != "fail" {
		t.Fatalf("a layer-3 validator exiting zero on a negative fixture must fail; got %s", res.Status)
	}
	if !hasCheck(res.Errors, "validator-negative") {
		t.Fatalf("the fail must come from the validator override; got %+v", res.Errors)
	}
}
func TestPackVal_P3_CompleteScaffoldRenderAndTest(t *testing.T) {
	m := baseManifest()
	m.Content.Scaffolds = []packval.Scaffold{{
		ID:           "S1",
		Path:         "scaf",
		Tier:         "complete",
		TestCommand:  "go test ./...",
		SampleConfig: map[string]string{"config.yml": "key: value"},
	}}
	dir := makePackDir(t)
	writeFile(t, dir, "scaf/main.go", "package main")
	r := packval.RunFixtures(m, dir, newFixtureMock(false, true))
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s: %+v", r.Status, r.Errors)
	}
	data, err := os.ReadFile(filepath.Join(dir, "scaf", "config.yml"))
	if err != nil {
		t.Fatalf("sample_config not rendered: %v", err)
	}
	if string(data) != "key: value" {
		t.Fatalf("sample_config content wrong: %s", data)
	}
}

// TestPackVal_P3_CompleteScaffoldTestFails: the fail must come from this test's own
// ScaffoldTestFn override, against a polarity-correct base mock — see
// Layer3PositiveExitNonZero for why the base configuration is load-bearing.
func TestPackVal_P3_CompleteScaffoldTestFails(t *testing.T) {
	m := baseManifest()
	m.Content.Scaffolds = []packval.Scaffold{{ID: "S1", Path: "scaf", Tier: "complete", TestCommand: "go test ./..."}}
	mock := newFixtureMock(false, true)
	mock.ScaffoldTestFn = func(_, _, _ string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: false}, nil
	}
	res := packval.RunFixtures(m, makePackDir(t), mock)
	if res.Status != "fail" {
		t.Fatalf("a failing complete-scaffold test command must fail the phase; got %s", res.Status)
	}
	if !hasCheck(res.Errors, "scaffold-complete") {
		t.Fatalf("the fail must come from the scaffold test override; got %+v", res.Errors)
	}
}
func TestPackVal_P3_SkeletonScaffoldStructure(t *testing.T) {
	m := baseManifest()
	m.Content.Scaffolds = []packval.Scaffold{{ID: "S1", Path: "scaf", Tier: "skeleton"}}
	dir := makePackDir(t)
	writeFile(t, dir, "scaf/README.md", "x")
	if packval.RunFixtures(m, dir, newFixtureMock(false, true)).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P3_SkeletonScaffoldTestIndicatorSatisfied(t *testing.T) {
	m := baseManifest()
	m.Content.Scaffolds = []packval.Scaffold{{ID: "S1", Path: "scaf", Tier: "skeleton", TestIndicator: "func Test"}}
	dir := makePackDir(t)
	writeFile(t, dir, "scaf/template_test.go", "package x\n\nfunc TestExample(t *testing.T) {}\n")
	r := packval.RunFixtures(m, dir, newFixtureMock(false, true))
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s", r.Status)
	}
	for _, w := range r.Warnings {
		if w.Check == "scaffold-skeleton-test-indicator" {
			t.Fatal("should not warn when the declared test indicator is present")
		}
	}
}
func TestPackVal_P3_SkeletonScaffoldNoTestExecution(t *testing.T) {
	m := baseManifest()
	m.Content.Scaffolds = []packval.Scaffold{{ID: "S1", Path: "scaf", Tier: "skeleton"}}
	dir := makePackDir(t)
	writeFile(t, dir, "scaf/README.md", "x")
	if packval.RunFixtures(m, dir, newFixtureMock(false, true)).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P3_SDKProvidesDeclared(t *testing.T) {
	m := baseManifest()
	m.Content.SDK = &packval.SDK{Provides: []string{"client"}}
	if packval.RunFixtures(m, makePackDir(t), newFixtureMock(false, true)).Status != "pass" {
		t.Fatal("pass")
	}
}

// TestPackVal_P3_SDKProvidesMissing: the fail must come from this test's own manifest
// edit (the empty entry in SDK.Provides), against a polarity-correct base mock — see
// Layer3PositiveExitNonZero for why the base configuration is load-bearing.
func TestPackVal_P3_SDKProvidesMissing(t *testing.T) {
	m := baseManifest()
	m.Content.SDK = &packval.SDK{Provides: []string{""}}
	res := packval.RunFixtures(m, makePackDir(t), newFixtureMock(false, true))
	if res.Status != "fail" {
		t.Fatalf("an empty sdk provides entry must fail the phase; got %s", res.Status)
	}
	if !hasCheck(res.Errors, "sdk-provides") {
		t.Fatalf("the fail must come from the empty SDK.Provides entry; got %+v", res.Errors)
	}
}
