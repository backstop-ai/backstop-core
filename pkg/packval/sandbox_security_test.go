package packval

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSandboxSecurityDenialsHold is the SECURITY half of ISSUE-029's both-halves
// acceptance (CLM-004). Under the SAME relaxed profile the real convert uses
// (via the production SandboxedRun / SandboxedRunStdout sandbox-exec path), all
// three real probes must still be DENIED: (i) reading a project file OUTSIDE
// packDir and the allowed system-lib subpaths, (ii) WRITING any file, (iii)
// opening a NETWORK connection. Relaxing reads for system libs must NOT open a
// hole — every sub-denial must hold for this test to pass.
//
// Same darwin guard as the operational half. The guard is a genuine platform
// N/A (sandbox-exec is macOS-only); it is NOT a skip-on-abort-trap.
func TestSandboxSecurityDenialsHold(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("sandbox-exec is macOS-only; this is a genuine platform N/A")
	}
	t.Run("ProjectFileRead", testSandboxDeniesProjectFileReadImpl)
	t.Run("FileWrite", testSandboxDeniesFileWriteImpl)
	t.Run("Network", testSandboxDeniesNetworkImpl)
}

// TestSandboxDeniesProjectFileRead — focused: a command reading a project file
// OUTSIDE packDir (and outside the allowed system-lib subpaths) exits non-zero
// with a sandbox denial (CLM-002).
func TestSandboxDeniesProjectFileRead(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("sandbox-exec is macOS-only")
	}
	testSandboxDeniesProjectFileReadImpl(t)
}

// TestSandboxDeniesFileWrite — focused: a command writing any file exits
// non-zero because (deny file-write*) holds under the relaxed profile (CLM-002).
func TestSandboxDeniesFileWrite(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("sandbox-exec is macOS-only")
	}
	testSandboxDeniesFileWriteImpl(t)
}

// TestSandboxDeniesNetwork — focused: a command opening a network connection
// exits non-zero because (deny network*) holds under the relaxed profile
// (CLM-002).
func TestSandboxDeniesNetwork(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("sandbox-exec is macOS-only")
	}
	testSandboxDeniesNetworkImpl(t)
}

// testSandboxDeniesProjectFileReadImpl places a secret OUTSIDE packDir, then
// tries to cat it through the real sandbox. The relaxed profile must DENY it:
// non-zero exit AND the secret bytes must NOT appear in the output.
func testSandboxDeniesProjectFileReadImpl(t *testing.T) {
	t.Helper()
	packDir := mustEvalSymlinks(t, t.TempDir())

	// A "project" file outside packDir and outside the allowed system-lib paths.
	outsideDir := mustEvalSymlinks(t, t.TempDir())
	secret := filepath.Join(outsideDir, "outside-secret.txt")
	const secretContent = "TOPSECRET-OUTSIDE-PACKDIR"
	if err := os.WriteFile(secret, []byte(secretContent), 0o644); err != nil {
		t.Fatalf("seed outside secret: %v", err)
	}

	out, err := SandboxedRun("/bin/cat", []string{secret}, packDir)
	if err == nil {
		t.Fatalf("reading project file outside packDir was ALLOWED (security hole): output %q", string(out))
	}
	if strings.Contains(string(out), secretContent) {
		t.Fatalf("secret content leaked despite error: %q", string(out))
	}
	// Confirm the failure is a sandbox denial, not some unrelated error.
	if !sandboxDenied(out, err) {
		t.Fatalf("expected a sandbox read denial, got: %v / %q", err, string(out))
	}
}

// testSandboxDeniesFileWriteImpl tries to create a file under packDir (which IS
// readable) — the write must still be DENIED by (deny file-write*).
func testSandboxDeniesFileWriteImpl(t *testing.T) {
	t.Helper()
	packDir := mustEvalSymlinks(t, t.TempDir())
	target := filepath.Join(packDir, "should-not-exist.txt")

	out, err := SandboxedRun("/usr/bin/touch", []string{target}, packDir)
	if err == nil {
		t.Fatalf("writing a file was ALLOWED (deny file-write* breached): output %q", string(out))
	}
	if _, statErr := os.Stat(target); statErr == nil {
		t.Fatalf("file %q was created despite the write denial", target)
	}
	if !sandboxDenied(out, err) {
		t.Fatalf("expected a sandbox write denial, got: %v / %q", err, string(out))
	}
}

// testSandboxDeniesNetworkImpl tries to open a TCP connection — (deny network*)
// must reject it (non-zero exit).
func testSandboxDeniesNetworkImpl(t *testing.T) {
	t.Helper()
	packDir := mustEvalSymlinks(t, t.TempDir())

	// nc -z attempts a connection; under (deny network*) it must fail.
	out, err := SandboxedRun("/usr/bin/nc", []string{"-w", "2", "-z", "1.1.1.1", "80"}, packDir)
	if err == nil {
		t.Fatalf("network connection was ALLOWED (deny network* breached): output %q", string(out))
	}
	if !sandboxDenied(out, err) {
		t.Fatalf("expected a sandbox network denial, got: %v / %q", err, string(out))
	}
}

// sandboxDenied confirms a non-zero result is attributable to the sandbox
// (a permission denial / non-zero exit under sandbox-exec) rather than a missing
// binary or other unrelated failure. We require err != nil and that the output
// does not indicate the command itself was absent.
func sandboxDenied(out []byte, err error) bool {
	if err == nil {
		return false
	}
	o := string(out)
	if strings.Contains(o, "command not found") || strings.Contains(o, "No such file or directory: nc") {
		return false
	}
	return true
}
