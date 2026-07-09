package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// buildBackstopBinary builds the REAL backstop binary from this package into a temp
// path and returns it. The authoring-loop acceptance drives this binary as a
// subprocess — real binaries, no in-process stubs (ISSUE-032 CLM-011).
func buildBackstopBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "backstop-e2e")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building backstop binary: %v\n%s", err, out)
	}
	return bin
}

// runBackstop runs the built binary with the given working dir and args, returning
// combined output and the process exit code.
func runBackstop(t *testing.T, bin, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			t.Fatalf("running %s %v: %v\n%s", bin, args, err, out)
		}
	}
	return string(out), code
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// TestPackAuthoringLoop_EndToEnd is the load-bearing acceptance (ISSUE-032 CLM-011 /
// CLM-006): the REAL authoring loop against a TEMP project — pack new -> pack check
// passes -> pack test passes -> pack add installs -> backstop gate RUN FROM INSIDE THE
// TEMP PROJECT consumes the newly-installed pack GREEN. Running the gate with the temp
// project as cwd is what genuinely proves the scaffolded pack is consumed: backstop-
// core's own manifest does not list the throwaway pack. Real binaries, no stubs.
func TestPackAuthoringLoop_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("authoring-loop e2e builds the binary and runs the gate; skipped in -short")
	}
	if runtime.GOOS != "darwin" {
		t.Skip("the scaffolded sandbox-validator engine runs under the darwin sandbox (sandbox-exec)")
	}

	bin := buildBackstopBinary(t)
	proj := t.TempDir()

	// A minimal but real project: a git repo (the gate derives its diff scope from
	// git), a backstop.yml, and an (empty) specs/ dir for the mandated-test extraction
	// step. No toolchain pack is declared — those dimensions warn capability_absent
	// (non-blocking), so the gate's greenness turns on the scaffolded pack's own engine.
	runGit(t, proj, "init")
	runGit(t, proj, "config", "user.email", "e2e@backstop.test")
	runGit(t, proj, "config", "user.name", "e2e")
	if err := os.WriteFile(filepath.Join(proj, "backstop.yml"), []byte("project: loop\npacks: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(proj, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}

	// 1. pack new — scaffold a fresh engine pack.
	if out, code := runBackstop(t, bin, proj, "pack", "new", "--type", "engine", "--language", "go", "--slug", "loop-pack"); code != 0 {
		t.Fatalf("pack new failed (exit %d): %s", code, out)
	}
	packDir := filepath.Join(proj, "loop-pack")

	// 2. pack check — passes on the freshly scaffolded pack.
	if out, code := runBackstop(t, bin, packDir, "pack", "check"); code != 0 {
		t.Fatalf("pack check failed (exit %d): %s", code, out)
	}

	// 3. pack test — passes (adds phase3).
	if out, code := runBackstop(t, bin, packDir, "pack", "test"); code != 0 {
		t.Fatalf("pack test failed (exit %d): %s", code, out)
	}

	// 4. pack add — installs the local pack into the temp project.
	if out, code := runBackstop(t, bin, proj, "pack", "add", "./loop-pack"); code != 0 {
		t.Fatalf("pack add failed (exit %d): %s", code, out)
	}
	runGit(t, proj, "add", "-A")
	runGit(t, proj, "commit", "-qm", "install scaffolded pack")

	// 5. gate FROM INSIDE the temp project — consumes the installed pack GREEN.
	out, code := runBackstop(t, bin, proj, "gate")
	if code != 0 {
		t.Fatalf("gate from inside temp project was not green (exit %d):\n%s", code, out)
	}
	// The pack's own engine must have run and passed — the load-bearing proof the
	// scaffolded pack is actually consumed, not silently skipped.
	if !strings.Contains(out, "pack_engines") {
		t.Fatalf("gate output did not reference pack_engines dispatch:\n%s", out)
	}
	if strings.Contains(out, "FAIL") {
		t.Fatalf("gate reported a failure while consuming the scaffolded pack:\n%s", out)
	}
}
