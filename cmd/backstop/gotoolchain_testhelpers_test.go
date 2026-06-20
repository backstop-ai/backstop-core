package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

// runConvertScriptDirect runs the pack convert script at path with stdin via
// /bin/sh, capturing only stdout. It is the test stand-in for the sandboxed
// convert capture: the real transform runs, but without sandbox-exec, so the
// build/test findings-engine path is exercised on any platform in tests while
// production still routes through SandboxedRunStdout.
func runConvertScriptDirect(scriptPath string, stdin []byte) ([]byte, error) {
	cmd := exec.Command("/bin/sh", scriptPath)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	return stdout.Bytes(), err
}

// fakeExitError is a stand-in non-zero process error for the crash-vs-findings
// guard tests: a runner returns it alongside captured stdout to simulate a tool
// exiting non-zero.
type fakeExitError struct {
	code int
}

func (e *fakeExitError) Error() string { return fmt.Sprintf("exit status %d", e.code) }
