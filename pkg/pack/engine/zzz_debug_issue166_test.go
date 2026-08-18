package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestZZZDebugIssue166(t *testing.T) {
	root := repoRoot(t)
	file := filepath.Join(root, "pkg", "gate", "testdata", "contract-absence-present.go")

	if out, err := exec.Command("grep", "--version").CombinedOutput(); err == nil {
		fmt.Fprintf(os.Stderr, "DEBUG166: grep --version: %s\n", out)
	} else {
		fmt.Fprintf(os.Stderr, "DEBUG166: grep --version failed: %v\n", err)
	}

	info, statErr := os.Stat(file)
	fmt.Fprintf(os.Stderr, "DEBUG166: stat %s err=%v info=%v\n", file, statErr, info)

	data, readErr := os.ReadFile(file)
	fmt.Fprintf(os.Stderr, "DEBUG166: read err=%v len=%d\n", readErr, len(data))
	fmt.Fprintf(os.Stderr, "DEBUG166: content=%q\n", string(data))

	cmd := exec.Command("grep", "-rn", "-e", "legacyProbeSymbol", file)
	out, runErr := cmd.CombinedOutput()
	fmt.Fprintf(os.Stderr, "DEBUG166: grep runErr=%v exitCode=%d output=%q\n", runErr, cmd.ProcessState.ExitCode(), out)

	// Also try without -r, plain grep on the file directly.
	cmd2 := exec.Command("grep", "-n", "-e", "legacyProbeSymbol", file)
	out2, runErr2 := cmd2.CombinedOutput()
	fmt.Fprintf(os.Stderr, "DEBUG166: plain grep runErr=%v output=%q\n", runErr2, out2)

	cwd, _ := os.Getwd()
	fmt.Fprintf(os.Stderr, "DEBUG166: cwd=%q root=%q\n", cwd, root)
}
