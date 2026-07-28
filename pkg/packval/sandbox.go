package packval

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
