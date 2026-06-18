package packval

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
)

func SandboxedRun(cmd string, args []string, packDir string) ([]byte, error) {
	switch runtime.GOOS {
	case "darwin":
		profile := fmt.Sprintf("(version 1)(deny default)(allow process*)(allow file-read* (subpath \"%s\"))(deny network*)(deny file-write*)", packDir)
		fullArgs := []string{"-p", profile, cmd}
		fullArgs = append(fullArgs, args...)
		c := exec.Command("sandbox-exec", fullArgs...)
		c.Dir = packDir
		out, err := c.CombinedOutput()
		if err != nil {
			return out, fmt.Errorf("sandboxed run failed: %w", err)
		}
		return out, nil
	case "linux":
		return nil, errors.New("sandbox unavailable on linux in this build")
	default:
		return nil, errors.New("sandbox unsupported platform")
	}
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
	switch runtime.GOOS {
	case "darwin":
		profile := fmt.Sprintf("(version 1)(deny default)(allow process*)(allow file-read* (subpath \"%s\"))(deny network*)(deny file-write*)", packDir)
		fullArgs := []string{"-p", profile, cmd}
		fullArgs = append(fullArgs, args...)
		c := exec.Command("sandbox-exec", fullArgs...)
		c.Dir = packDir
		if stdin != nil {
			c.Stdin = bytes.NewReader(stdin)
		}
		var stdout bytes.Buffer
		c.Stdout = &stdout
		err := c.Run()
		if err != nil {
			return stdout.Bytes(), fmt.Errorf("sandboxed run (stdout) failed: %w", err)
		}
		return stdout.Bytes(), nil
	case "linux":
		return nil, errors.New("sandbox unavailable on linux in this build")
	default:
		return nil, errors.New("sandbox unsupported platform")
	}
}
