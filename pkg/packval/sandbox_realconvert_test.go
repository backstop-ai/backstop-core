package packval

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// locateJQ returns the absolute path to a real jq binary, searching PATH first
// then the Intel and Apple-Silicon Homebrew prefixes. On a darwin host jq is a
// hard requirement for the real-sandbox tests: callers MUST t.Fatal (never
// t.Skip) when it returns "" — a skip-on-missing-jq is the vacuous green
// ISSUE-029 exists to kill.
func locateJQ() string {
	if p, err := exec.LookPath("jq"); err == nil {
		return p
	}
	for _, cand := range []string{"/usr/local/bin/jq", "/opt/homebrew/bin/jq"} {
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			return cand
		}
	}
	return ""
}

// requireJQ fails loud (NOT skip) on darwin when jq is absent, naming jq and how
// to install it, per ISSUE-029's anti-vacuous-green rule.
func requireJQ(t *testing.T) string {
	t.Helper()
	jq := locateJQ()
	if jq == "" {
		t.Fatalf("jq is required for the real-sandbox convert tests on darwin but was not found on PATH, /usr/local/bin/jq, or /opt/homebrew/bin/jq; install it with `brew install jq`")
	}
	return jq
}

// TestSandboxConvertWithRealInterpreter is the OPERATIONAL half of ISSUE-029's
// both-halves acceptance (CLM-003). A real convert script pipes its stdin
// through real jq — a dynamically-linked interpreter that SIGABRTs at dyld load
// under the unfixed packDir-only profile — through the PRODUCTION
// SandboxedRunStdout / sandbox-exec path (NO /bin/sh stub). It must FAIL against
// the unfixed profile (abort trap) and SUCCEED once the dyld read paths land,
// producing the jq-transformed bytes.
//
// VACUOUS-GREEN RULES: the runtime.GOOS guard is a genuine platform N/A
// (sandbox-exec is macOS-only). On darwin jq is REQUIRED (requireJQ → Fatal, not
// Skip). The test MUST NOT swallow "abort trap"/SIGABRT as a skip or pass: on a
// darwin host with jq present, abort trap is a HARD failure.
func TestSandboxConvertWithRealInterpreter(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("sandbox-exec is macOS-only; this is a genuine platform N/A (linux no-op is ISSUE-020)")
	}
	jq := requireJQ(t)
	_ = jq // the convert script locates jq itself; requireJQ asserts presence/fail-loud

	// packDir is the only project path the sandbox may read. It must contain the
	// convert script so the sandbox can exec it. Resolve symlinks so the profile
	// subpath matches the kernel-resolved path (/var -> /private/var).
	packDir := mustEvalSymlinks(t, t.TempDir())
	script := filepath.Join(packDir, "convert-jq.sh")
	copyFixtureInto(t, "convert-jq.sh", script)

	stdin := []byte(`{"a":41}`)
	out, err := SandboxedRunStdout("/bin/sh", []string{script}, packDir, stdin)
	if err != nil {
		// On darwin-with-jq an abort trap / any sandbox failure is a HARD failure
		// (the red phase). Never downgrade to Skip — that is the bug-hiding pattern.
		t.Fatalf("real convert through sandbox-exec failed (this is the unfixed-profile dyld abort the fix must eliminate): %v\noutput: %q", err, string(out))
	}
	got := strings.TrimSpace(string(out))
	if got != "42" {
		t.Fatalf("convert produced wrong output: got %q want %q (stdin %q)", got, "42", string(stdin))
	}
}

// TestSandboxProfileAllowsDyldLibraries MOVED to sandbox_darwin_test.go
// (`//go:build darwin`) in Phase 3b, unchanged. darwinSandboxProfile is now
// compiled only on darwin, so a caller in this untagged file would break
// `GOOS=linux go vet ./pkg/packval/`. The helpers below stay here: the linux exec
// tests use mustEvalSymlinks, copyFixtureInto and locateJQ, so this file must
// remain untagged.

// mustEvalSymlinks resolves symlinks (e.g. macOS /var -> /private/var) so a
// sandbox subpath rule matches the kernel-resolved path.
func mustEvalSymlinks(t *testing.T, p string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", p, err)
	}
	return resolved
}

// copyFixtureInto copies a testdata/sandbox fixture into dst (executable),
// because the sandbox may only read files under packDir.
func copyFixtureInto(t *testing.T, name, dst string) {
	t.Helper()
	src := filepath.Join("testdata", "sandbox", name)
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read fixture %q: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		t.Fatalf("write fixture to %q: %v", dst, err)
	}
}
