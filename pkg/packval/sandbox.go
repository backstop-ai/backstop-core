package packval

import (
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
