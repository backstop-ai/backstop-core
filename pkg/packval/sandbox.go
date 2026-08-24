package packval

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/backstop-ai/backstop-core/pkg/check"
)

// The platform-neutral face of the sandbox.
//
// SandboxedRun and SandboxedRunStdout are the ONLY sandbox entry points outside
// this package (pkg/packval/executor.go, cmd/backstop/pack_gate.go), and their
// signatures and output contracts are fixed — CombinedOutput semantics for the
// first, an explicit stdout buffer plus stdin pipe and "stdout captured so far
// alongside the error" for the second.
//
// Everything platform-specific is selected at BUILD time, not run time:
//
//	sandbox_linux.go     //go:build linux    Landlock + seccomp
//	sandbox_nonlinux.go  //go:build !linux   sandbox-exec on darwin, refusal elsewhere
//
// This file previously dispatched on `switch runtime.GOOS`, which meant every
// build compiled the OTHER platform's arms as unreachable code — measured, never
// executable, and impossible to cover on either host. Build tags delete that
// class of dead code outright: each platform compiles only what it can run.
// TestSandboxDispatch_NoRuntimeGOOSSwitchRemains asserts structurally that the
// switch does not come back.

const PackSandboxEnvVar = "BACKSTOP_PACK_SANDBOX"

const (
	sandboxHelperEnvVar = "BACKSTOP_SANDBOX_HELPER_SPEC"
	sandboxAckFD        = 3
	sandboxAckByte      = byte(0xa5)
)

type sandboxHelperRequest struct {
	Capability  SandboxCapability `json:"capability"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Dir         string            `json:"dir"`
	Environment []string          `json:"environment"`
	AckFD       int               `json:"ack_fd"`
}

type SandboxMode string

const (
	SandboxModeNative   SandboxMode = "native"
	SandboxModeExternal SandboxMode = "external"
)

type SandboxRunResult struct {
	Output               []byte
	NativeSandboxApplied bool
}

type SandboxRunner interface {
	Mode() SandboxMode
	Run(cmd string, args []string, packDir string) (SandboxRunResult, error)
	RunStdout(cmd string, args []string, packDir string, stdin []byte) (SandboxRunResult, error)
}

type sandboxExecution func(string, []string, string, []byte, bool) (SandboxRunResult, error)

type sandboxRunner struct {
	mode     SandboxMode
	native   sandboxExecution
	external sandboxExecution
}

func NewSandboxRunner(mode SandboxMode) (SandboxRunner, error) {
	return newSandboxRunnerWithExecution(mode, platformSandboxedExecute, externalSandboxedExecute)
}

func newSandboxRunnerWithExecution(mode SandboxMode, native, external sandboxExecution) (SandboxRunner, error) {
	if mode != SandboxModeNative && mode != SandboxModeExternal {
		return nil, fmt.Errorf("invalid pack sandbox mode %q: must be exactly native or external", mode)
	}
	return &sandboxRunner{mode: mode, native: native, external: external}, nil
}

func (r *sandboxRunner) Mode() SandboxMode { return r.mode }

func (r *sandboxRunner) Run(cmd string, args []string, packDir string) (SandboxRunResult, error) {
	return r.execute(cmd, args, packDir, nil, false)
}

func (r *sandboxRunner) RunStdout(cmd string, args []string, packDir string, stdin []byte) (SandboxRunResult, error) {
	return r.execute(cmd, args, packDir, stdin, true)
}

func (r *sandboxRunner) execute(cmd string, args []string, packDir string, stdin []byte, stdoutOnly bool) (SandboxRunResult, error) {
	if r.mode == SandboxModeExternal {
		return r.external(cmd, args, packDir, stdin, stdoutOnly)
	}
	return r.native(cmd, args, packDir, stdin, stdoutOnly)
}

func externalSandboxedExecute(command string, args []string, packDir string, stdin []byte, stdoutOnly bool) (SandboxRunResult, error) {
	// command and args originate in a verified pack manifest.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.Command(command, args...)
	cmd.Dir = packDir
	cmd.Env = check.WithoutEnvironment(os.Environ(), PackSandboxEnvVar)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	if !stdoutOnly {
		out, err := cmd.CombinedOutput()
		return SandboxRunResult{Output: out}, err
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	return SandboxRunResult{Output: stdout.Bytes()}, err
}

// SandboxedRun runs cmd under the platform's sandbox from packDir and returns
// its combined stdout+stderr. A non-zero exit is returned as an error alongside
// whatever output was produced.
func SandboxedRun(cmd string, args []string, packDir string) ([]byte, error) {
	return platformSandboxedRun(cmd, args, packDir)
}

// SandboxedRunStdout is the clean-stdout variant of SandboxedRun used by the
// convert step (REQ-007/REQ-009/CLM-065). It runs cmd under the same sandbox
// trust model as SandboxedRun, but captures ONLY stdout via an explicit buffer
// so a converter writing a banner/warning to stderr cannot interleave into the
// SARIF bytes the gate parses. The optional stdin is fed to the command's
// standard input, implementing the engine-stdout -> convert-stdin pipe in Go
// (no shell). On a non-zero exit it returns the stdout captured so far
// alongside the error so the caller can attribute the failure to the convert
// step. The CombinedOutput-based SandboxedRun is retained unchanged for the
// exit-code sandbox-validator path (REQ-014), whose merged stderr is a
// legitimate message body.
func SandboxedRunStdout(cmd string, args []string, packDir string, stdin []byte) ([]byte, error) {
	return platformSandboxedRunStdout(cmd, args, packDir, stdin)
}
