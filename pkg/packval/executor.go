package packval

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// FixtureExecutor runs a pack's declared fixtures against their resolved engines.
// The findings path is GENERIC: RunEngine takes a resolved engine.EngineBinding
// (pack DATA) and the fixture targets — there is no tool-named method and no baked
// tool switch. RunValidator/RunScaffoldTest remain the sandbox/scaffold seams.
type FixtureExecutor interface {
	// RunEngine dispatches a resolved engine binding (command from pack DATA) at the
	// given targets. It is the single generic replacement for the retired tool-named
	// fixture methods (ISSUE-019).
	RunEngine(packDir string, binding engine.EngineBinding, targets []string) (ExecutionResult, error)
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

// buildEngineArgv constructs the command + args from the engine binding DATA: the
// binding.Command supplies the executable and its baked-in flags, the InputFlag (when
// set) is injected once, and the resolved targets (rule/config file + fixture) are
// appended. No tool name is ever a literal here — it comes entirely from the binding.
func buildEngineArgv(binding engine.EngineBinding, targets []string) (string, []string) {
	fields := strings.Fields(binding.Command)
	if len(fields) == 0 {
		return "", nil
	}
	name := fields[0]
	args := append([]string{}, fields[1:]...)
	if binding.InputFlag != "" {
		args = append(args, binding.InputFlag)
	}
	args = append(args, targets...)
	return name, args
}

// RunEngine runs the resolved engine binding at the targets and reports whether the
// engine FIRED (produced findings). The trust floor is enforced FIRST: a provisioned
// tool that is not on the trusted-tool allowlist (or not pinned to its allowlisted
// version) is NEVER executed — engine.CheckToolAllowed fail-louds before the command
// is handed to the runner (SPEC-035 REQ-002). "Fired" is decided from the engine's
// SARIF output — the universal contract backstop speaks — so the signal is
// tool/language-blind: a positive fixture that the rule matches yields >=1 finding
// (Passed=true), a clean negative yields zero (Passed=false).
func (d *DefaultExecutor) RunEngine(packDir string, binding engine.EngineBinding, targets []string) (ExecutionResult, error) {
	if binding.Provision != nil {
		if err := engine.CheckToolAllowed(
			engine.TrustedToolAllowlist(),
			binding.Provision.Tool,
			binding.Provision.Version,
		); err != nil {
			return ExecutionResult{}, fmt.Errorf("engine %q failed the trusted-tool allowlist gate: %w", binding.Command, err)
		}
	}
	name, args := buildEngineArgv(binding, targets)
	if name == "" {
		return ExecutionResult{}, fmt.Errorf("engine binding has no command to run")
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = packDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	// A findings engine legitimately exits non-zero WHEN it reports findings, so the
	// exit code is not the contract — the SARIF on stdout is. A run whose PROCESS
	// NEVER STARTED, however, is a broken run and not a finding-free pass — fail loud
	// so a missing engine never reads as a clean negative (vacuous green).
	// check.NeverStarted is the single authority for that class, and it is a CLASS,
	// not `runErr != nil`: binding.Command is pack DATA that may carry a path
	// separator, and a path-ful command that cannot be exec'd never reaches LookPath,
	// so it reports an *fs.PathError rather than the *exec.Error this check once
	// matched alone (ISSUE-140).
	runErr := cmd.Run()
	if check.NeverStarted(runErr) {
		return ExecutionResult{Passed: false, Output: stdout.String(), ExitCode: 1},
			fmt.Errorf("engine %q never started: the process could not be executed — this is a broken run, not a finding-free pass; ensure the command is present and executable: %w", binding.Command, runErr)
	}
	// Apply the binding's declared convert BEFORE parsing (ISSUE-141). packval's
	// executor previously handed raw stdout to the parser, so a pack whose engine
	// emits non-SARIF output and ships a reshaper died at the parse on output the
	// convert would have made parseable.
	//
	// ORDER: strictly after the never-started refusal above. A process that never
	// started produced no bytes, so running a convert over its empty stdout would
	// manufacture a convert-step failure to describe an engine that never ran.
	//
	// NO IMPORT: SandboxedRunStdout is same-package. cmd/backstop's convert reaches
	// it through resolveSandboxedRunStdout, whose production value IS this function
	// — the gate side has always called INTO this package for its convert, so both
	// paths carry the same sandboxing guarantee rather than two approximations.
	payload := stdout.Bytes()
	if binding.Convert != "" {
		convertPath := filepath.Join(packDir, filepath.FromSlash(binding.Convert))
		if info, statErr := os.Stat(convertPath); statErr != nil || info.IsDir() {
			return ExecutionResult{Passed: false, Output: stdout.String(), ExitCode: 1},
				fmt.Errorf("engine %q declares a convert script that is missing or not a file: %s", binding.Command, convertPath)
		}
		converted, convErr := SandboxedRunStdout(convertPath, nil, packDir, payload)
		if convErr != nil {
			return ExecutionResult{Passed: false, Output: stdout.String(), ExitCode: 1},
				fmt.Errorf("engine %q: convert step (%s) failed: %w", binding.Command, binding.Convert, convErr)
		}
		payload = converted
	}
	findings, parseErr := check.ParsePackFindings(payload)
	if parseErr != nil {
		return ExecutionResult{Passed: false, Output: stdout.String(), ExitCode: 1},
			fmt.Errorf("engine %q produced no parseable SARIF: %w", binding.Command, parseErr)
	}
	return ExecutionResult{Passed: len(findings) > 0, Output: stdout.String(), ExitCode: 0}, nil
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
	EngineFn       func(packDir string, binding engine.EngineBinding, targets []string) (ExecutionResult, error)
	ValidatorFn    func(packDir, validator string, fixturePaths []string) (ExecutionResult, error)
	ScaffoldTestFn func(packDir, scaffoldPath, testCommand string) (ExecutionResult, error)
}

func (m *MockExecutor) RunEngine(packDir string, binding engine.EngineBinding, targets []string) (ExecutionResult, error) {
	if m.EngineFn != nil {
		return m.EngineFn(packDir, binding, targets)
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
