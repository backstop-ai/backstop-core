package packval

import (
	"fmt"
	"os/exec"
	"strings"
)

type FixtureExecutor interface {
	RunSemgrep(packDir, ruleFile, fixturePath string) (ExecutionResult, error)
	RunToolConfig(packDir, tool, configFile, fixturePath string) (ExecutionResult, error)
	RunValidator(packDir, validator string, fixturePaths []string) (ExecutionResult, error)
	RunScaffoldTest(packDir, scaffoldPath, testCommand string) (ExecutionResult, error)
}

type ExecutionResult struct {
	Passed      bool     `json:"passed"`
	Output      string   `json:"output,omitempty"`
	ExitCode    int      `json:"exit_code"`
	Diagnostics []string `json:"diagnostics,omitempty"`
}

type DefaultExecutor struct{}

func (d *DefaultExecutor) RunSemgrep(packDir, ruleFile, fixturePath string) (ExecutionResult, error) {
	cmd := exec.Command("semgrep", "--config", ruleFile, fixturePath)
	cmd.Dir = packDir
	out, err := cmd.CombinedOutput()
	return resultFromCmd(out, err), nil
}

func (d *DefaultExecutor) RunToolConfig(packDir, tool, configFile, fixturePath string) (ExecutionResult, error) {
	var cmd *exec.Cmd
	switch strings.ToLower(tool) {
	case "golangci-lint":
		cmd = exec.Command("golangci-lint", "run", "--config", configFile, fixturePath)
	default:
		return ExecutionResult{}, fmt.Errorf("unsupported tool %q", tool)
	}
	cmd.Dir = packDir
	out, err := cmd.CombinedOutput()
	return resultFromCmd(out, err), nil
}

func (d *DefaultExecutor) RunValidator(packDir, validator string, fixturePaths []string) (ExecutionResult, error) {
	out, err := SandboxedRun(validator, fixturePaths, packDir)
	if err != nil {
		return ExecutionResult{Passed: false, Output: string(out), ExitCode: 1}, nil
	}
	return ExecutionResult{Passed: true, Output: string(out), ExitCode: 0}, nil
}

func (d *DefaultExecutor) RunScaffoldTest(packDir, scaffoldPath, testCommand string) (ExecutionResult, error) {
	cmd := exec.Command("sh", "-c", testCommand)
	cmd.Dir = packDir + "/" + scaffoldPath
	out, err := cmd.CombinedOutput()
	return resultFromCmd(out, err), nil
}

type MockExecutor struct {
	SemgrepFn      func(packDir, ruleFile, fixturePath string) (ExecutionResult, error)
	ToolConfigFn   func(packDir, tool, configFile, fixturePath string) (ExecutionResult, error)
	ValidatorFn    func(packDir, validator string, fixturePaths []string) (ExecutionResult, error)
	ScaffoldTestFn func(packDir, scaffoldPath, testCommand string) (ExecutionResult, error)
}

func (m *MockExecutor) RunSemgrep(packDir, ruleFile, fixturePath string) (ExecutionResult, error) {
	if m.SemgrepFn != nil {
		return m.SemgrepFn(packDir, ruleFile, fixturePath)
	}
	return ExecutionResult{Passed: true, ExitCode: 0}, nil
}
func (m *MockExecutor) RunToolConfig(packDir, tool, configFile, fixturePath string) (ExecutionResult, error) {
	if m.ToolConfigFn != nil {
		return m.ToolConfigFn(packDir, tool, configFile, fixturePath)
	}
	return ExecutionResult{Passed: true, ExitCode: 0}, nil
}
func (m *MockExecutor) RunValidator(packDir, validator string, fixturePaths []string) (ExecutionResult, error) {
	if m.ValidatorFn != nil {
		return m.ValidatorFn(packDir, validator, fixturePaths)
	}
	return ExecutionResult{Passed: true, ExitCode: 0}, nil
}
func (m *MockExecutor) RunScaffoldTest(packDir, scaffoldPath, testCommand string) (ExecutionResult, error) {
	if m.ScaffoldTestFn != nil {
		return m.ScaffoldTestFn(packDir, scaffoldPath, testCommand)
	}
	return ExecutionResult{Passed: true, ExitCode: 0}, nil
}

func resultFromCmd(out []byte, err error) ExecutionResult {
	r := ExecutionResult{Output: string(out), ExitCode: 0, Passed: true}
	if err == nil {
		return r
	}
	r.Passed = false
	if exitErr, ok := err.(*exec.ExitError); ok {
		r.ExitCode = exitErr.ExitCode()
	} else {
		r.ExitCode = 1
	}
	return r
}
