package packval_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/packval"
)

func TestPackVal_DefaultExecutorImplementsInterface(t *testing.T) {
	var _ packval.FixtureExecutor = &packval.DefaultExecutor{}
}
func TestPackVal_MockExecutorSemgrep(t *testing.T) {
	m := &packval.MockExecutor{SemgrepFn: func(_, _, _ string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: true}, nil
	}}
	r, _ := m.RunSemgrep("", "", "")
	if !r.Passed {
		t.Fatal("expected pass")
	}
}
func TestPackVal_MockExecutorToolConfig(t *testing.T) {
	m := &packval.MockExecutor{ToolConfigFn: func(_, _, _, _ string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: true}, nil
	}}
	r, _ := m.RunToolConfig("", "", "", "")
	if !r.Passed {
		t.Fatal("expected pass")
	}
}
func TestPackVal_MockExecutorValidator(t *testing.T) {
	m := &packval.MockExecutor{ValidatorFn: func(_, _ string, _ []string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: true}, nil
	}}
	r, _ := m.RunValidator("", "", nil)
	if !r.Passed {
		t.Fatal("expected pass")
	}
}
func TestPackVal_MockExecutorScaffoldTest(t *testing.T) {
	m := &packval.MockExecutor{ScaffoldTestFn: func(_, _, _ string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: true}, nil
	}}
	r, _ := m.RunScaffoldTest("", "", "")
	if !r.Passed {
		t.Fatal("expected pass")
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

func newFixtureMock(passPos bool, failNeg bool) *packval.MockExecutor {
	return &packval.MockExecutor{
		SemgrepFn: func(_, _, fixture string) (packval.ExecutionResult, error) {
			isNeg := strings.Contains(fixture, "n.")
			if isNeg {
				return packval.ExecutionResult{Passed: !failNeg, Output: "R1"}, nil
			}
			return packval.ExecutionResult{Passed: passPos, Output: "R1"}, nil
		},
		ToolConfigFn: func(_, _, _, fixture string) (packval.ExecutionResult, error) {
			isNeg := strings.Contains(fixture, "n.")
			if isNeg {
				return packval.ExecutionResult{Passed: false}, nil
			}
			return packval.ExecutionResult{Passed: true}, nil
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

func TestPackVal_P3_SemgrepPositivePass(t *testing.T) {
	if packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(true, true)).Status != "pass" {
		t.Fatal("expected pass")
	}
}
func TestPackVal_P3_SemgrepPositiveFalsePositive(t *testing.T) {
	if packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(false, true)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P3_SemgrepNegativeAllTrigger(t *testing.T) {
	if packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(true, true)).Status != "pass" {
		t.Fatal("expected pass")
	}
}
func TestPackVal_P3_SemgrepNegativeNotTriggered(t *testing.T) {
	if packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(true, false)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P3_SemgrepRuleIDMismatch(t *testing.T) {
	m := baseManifest()
	dir := makePackDir(t)
	writeFile(t, dir, "rules/r1.yml", "rules:\n  - id: WRONG_ID\n")
	r := packval.RunFixtures(m, dir, newFixtureMock(true, true))
	if r.Status == "pass" {
		t.Fatal("expected fail when rule ID doesn't match file")
	}
	found := false
	for _, e := range r.Errors {
		if e.Check == "semgrep-rule-id" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected semgrep-rule-id error")
	}
}
func TestPackVal_P3_SemgrepRuleIDMatch(t *testing.T) {
	if packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(true, true)).Status != "pass" {
		t.Fatal("expected pass")
	}
}
func TestPackVal_P3_ToolConfigTempModule(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{ID: "T1", Tool: "golangci-lint", File: ".golangci.yml", RiskClass: "style", Claims: m.Content.Ruleset.Rules[0].Claims}}
	dir := makePackDir(t)
	writeFile(t, dir, ".golangci.yml", "")
	goModBefore, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod before: %v", err)
	}
	r := packval.RunFixtures(m, dir, newFixtureMock(true, true))
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s: %+v", r.Status, r.Errors)
	}
	goModAfter, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod after: %v", err)
	}
	if string(goModBefore) != string(goModAfter) {
		t.Fatal("go.mod was modified — temp copy not used")
	}
}
func TestPackVal_P3_ToolConfigPositiveClean(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{ID: "T1", Tool: "golangci-lint", File: ".golangci.yml", RiskClass: "style", Claims: m.Content.Ruleset.Rules[0].Claims}}
	dir := makePackDir(t)
	writeFile(t, dir, ".golangci.yml", "")
	if packval.RunFixtures(m, dir, newFixtureMock(true, true)).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P3_ToolConfigNegativeNotTriggered(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{ID: "T1", Tool: "golangci-lint", File: ".golangci.yml", RiskClass: "style", Claims: m.Content.Ruleset.Rules[0].Claims}}
	dir := makePackDir(t)
	writeFile(t, dir, ".golangci.yml", "")
	mock := newFixtureMock(true, true)
	mock.ToolConfigFn = func(_, _, _, _ string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: true}, nil
	}
	if packval.RunFixtures(m, dir, mock).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P3_ToolConfigNegativeTriggered(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{ID: "T1", Tool: "golangci-lint", File: ".golangci.yml", RiskClass: "style", Claims: m.Content.Ruleset.Rules[0].Claims}}
	dir := makePackDir(t)
	writeFile(t, dir, ".golangci.yml", "")
	if packval.RunFixtures(m, dir, newFixtureMock(true, true)).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P3_GoModTidyRuns(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{ID: "T1", Tool: "golangci-lint", File: ".golangci.yml", RiskClass: "style", Claims: m.Content.Ruleset.Rules[0].Claims}}
	dir := makePackDir(t)
	writeFile(t, dir, ".golangci.yml", "")
	if packval.RunFixtures(m, dir, newFixtureMock(true, true)).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P3_GoModTidyFails(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{ID: "T1", Tool: "golangci-lint", File: ".golangci.yml", RiskClass: "style", Claims: m.Content.Ruleset.Rules[0].Claims}}
	dir := makePackDir(t)
	_ = os.Remove(filepath.Join(dir, "go.mod"))
	if packval.RunFixtures(m, dir, newFixtureMock(true, true)).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P3_NegativeFixtureEngineLimitationHint(t *testing.T) {
	r := packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(true, false))
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
	mock := newFixtureMock(true, true)
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
	mock := newFixtureMock(true, true)
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
	if packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(true, true)).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P3_Layer3PositiveExitNonZero(t *testing.T) {
	mock := newFixtureMock(true, true)
	mock.ValidatorFn = func(_, _ string, _ []string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: false}, nil
	}
	if packval.RunFixtures(baseManifest(), makePackDir(t), mock).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P3_Layer3NegativeExitNonZero(t *testing.T) {
	if packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(true, true)).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P3_Layer3NegativeExitZero(t *testing.T) {
	mock := newFixtureMock(true, true)
	mock.ValidatorFn = func(_, _ string, _ []string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: true}, nil
	}
	if packval.RunFixtures(baseManifest(), makePackDir(t), mock).Status != "fail" {
		t.Fatal("fail")
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
	r := packval.RunFixtures(m, dir, newFixtureMock(true, true))
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
func TestPackVal_P3_CompleteScaffoldTestFails(t *testing.T) {
	m := baseManifest()
	m.Content.Scaffolds = []packval.Scaffold{{ID: "S1", Path: "scaf", Tier: "complete", TestCommand: "go test ./..."}}
	mock := newFixtureMock(true, true)
	mock.ScaffoldTestFn = func(_, _, _ string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: false}, nil
	}
	if packval.RunFixtures(m, makePackDir(t), mock).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P3_SkeletonScaffoldStructure(t *testing.T) {
	m := baseManifest()
	m.Content.Scaffolds = []packval.Scaffold{{ID: "S1", Path: "scaf", Tier: "skeleton"}}
	dir := makePackDir(t)
	writeFile(t, dir, "scaf/README.md", "x")
	if packval.RunFixtures(m, dir, newFixtureMock(true, true)).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P3_SkeletonScaffoldTestNames(t *testing.T) {
	m := baseManifest()
	m.Content.Scaffolds = []packval.Scaffold{{ID: "S1", Path: "scaf", Tier: "skeleton"}}
	dir := makePackDir(t)
	writeFile(t, dir, "scaf/template_test.go", "package x\n\nfunc TestExample(t *testing.T) {}\n")
	r := packval.RunFixtures(m, dir, newFixtureMock(true, true))
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s", r.Status)
	}
	for _, w := range r.Warnings {
		if w.Check == "scaffold-skeleton-test-names" {
			t.Fatal("should not warn when test function names are present")
		}
	}
}
func TestPackVal_P3_SkeletonScaffoldNoTestExecution(t *testing.T) {
	m := baseManifest()
	m.Content.Scaffolds = []packval.Scaffold{{ID: "S1", Path: "scaf", Tier: "skeleton"}}
	dir := makePackDir(t)
	writeFile(t, dir, "scaf/README.md", "x")
	if packval.RunFixtures(m, dir, newFixtureMock(true, true)).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P3_SDKProvidesDeclared(t *testing.T) {
	m := baseManifest()
	m.Content.SDK = &packval.SDK{Provides: []string{"client"}}
	if packval.RunFixtures(m, makePackDir(t), newFixtureMock(true, true)).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P3_SDKProvidesMissing(t *testing.T) {
	m := baseManifest()
	m.Content.SDK = &packval.SDK{Provides: []string{""}}
	if packval.RunFixtures(m, makePackDir(t), newFixtureMock(true, true)).Status != "fail" {
		t.Fatal("fail")
	}
}
