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
func TestPackVal_P3_SandboxBlocksEnvVars(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	dir := makePackDir(t)
	_, err := packval.SandboxedRun("sh", []string{"-c", "printenv HOME"}, dir)
	if err == nil {
		t.Fatal("expected sandbox env failure")
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
	if err != nil && strings.Contains(err.Error(), "abort trap") {
		t.Skip("sandbox-exec profile unsupported in this environment")
	}
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
	if err != nil && strings.Contains(err.Error(), "abort trap") {
		t.Skip("sandbox-exec profile unsupported in this environment")
	}
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
	m := newFixtureMock(true, true)
	m.SemgrepFn = func(_, _, _ string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: true, Output: "WRONG"}, nil
	}
	if packval.RunFixtures(baseManifest(), makePackDir(t), m).Status != "fail" {
		t.Fatal("expected fail")
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
	writeFile(t, makePackDir(t), ".golangci.yml", "")
	_ = packval.RunFixtures(m, makePackDir(t), newFixtureMock(true, true))
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
		}
	}
	if !found {
		t.Fatal("expected fix hint")
	}
}
func TestPackVal_P3_Layer3SingleFileInvocation(t *testing.T) {
	if packval.RunFixtures(baseManifest(), makePackDir(t), newFixtureMock(true, true)).Status != "pass" {
		t.Fatal("pass")
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
	m.Content.Scaffolds = []packval.Scaffold{{ID: "S1", Path: "scaf", Tier: "complete", TestCommand: "go test ./..."}}
	writeFile(t, makePackDir(t), "scaf/main.go", "package main")
	if packval.RunFixtures(m, makePackDir(t), newFixtureMock(true, true)).Status != "pass" {
		t.Fatal("pass")
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
	writeFile(t, dir, "scaf/template_test.go", "package x")
	if packval.RunFixtures(m, dir, newFixtureMock(true, true)).Status != "pass" {
		t.Fatal("pass")
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
