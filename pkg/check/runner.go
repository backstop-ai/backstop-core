package check

import (
	"bytes"
	"context"
	"os/exec"
)

// CommandRunner abstracts external command execution so executors are
// unit-testable against fixture output without shelling out to live tools.
//
// pkg/check must not depend on pkg/gate, so this is a local equivalent of the
// CommandRunner / ExecCommandRunner pair in pkg/gate/step_coverage.go.
type CommandRunner interface {
	// Run returns combined stdout+stderr, used by the build/test executors
	// whose violation messages may legitimately include stderr.
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
	// RunStdout returns ONLY stdout, uncontaminated by stderr (REQ-009). The
	// engine dispatch SARIF path uses it so a tool's stderr banner/progress
	// cannot corrupt the SARIF bytes on stdout.
	RunStdout(ctx context.Context, name string, args ...string) ([]byte, error)
}

// ExecCommandRunner is a CommandRunner that uses os/exec to run commands. Dir
// is set to Options.ProjectDir by callers so go build / go test resolve the
// project's module, mirroring pkg/gate/step_coverage.go without the gate
// dependency.
type ExecCommandRunner struct {
	Dir string // working directory for commands
}

// Run executes the named command with args and returns combined output. A
// cancelled or expired context aborts the underlying process via
// exec.CommandContext, so the engine's timeout-violation path fires.
func (r *ExecCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	// name/args are the declared engine command from a verified pack manifest
	// (not user input); running it dynamically is this runner's whole purpose.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, name, args...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	return cmd.CombinedOutput()
}

// RunStdout executes the named command and returns ONLY its stdout, captured
// via an explicit stdout buffer so stderr cannot interleave into the bytes
// (REQ-009 / CLM-028). On a non-zero exit it returns the stdout captured so far
// alongside the error so the caller can attribute the failure to the engine's
// output. The existing Run (CombinedOutput) method is intentionally left
// untouched for the build/test executors (Review Question 5).
func (r *ExecCommandRunner) RunStdout(ctx context.Context, name string, args ...string) ([]byte, error) {
	// See Run: declared engine command from a verified manifest, not user input.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, name, args...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	return stdout.Bytes(), err
}
