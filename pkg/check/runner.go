package check

import (
	"context"
	"os/exec"
)

// CommandRunner abstracts external command execution so executors are
// unit-testable against fixture output without shelling out to live tools.
//
// pkg/check must not depend on pkg/gate, so this is a local equivalent of the
// CommandRunner / ExecCommandRunner pair in pkg/gate/step_coverage.go.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
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
	cmd := exec.CommandContext(ctx, name, args...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	return cmd.CombinedOutput()
}
