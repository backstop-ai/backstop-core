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

// TestSandboxProfileAllowsDyldLibraries asserts the shared darwin profile builder
// (CLM-001) emits the runtime/system read subpaths a dynamically-linked
// interpreter needs at dyld load — in ADDITION to packDir — proving both
// SandboxedRun and SandboxedRunStdout source the SAME extended profile. It also
// asserts the deny rules and the bsd.sb base import that lets dyld read the
// shared cache survive.
func TestSandboxProfileAllowsDyldLibraries(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("sandbox-exec / darwinSandboxProfile is macOS-only")
	}
	packDir := mustEvalSymlinks(t, t.TempDir())
	profile := darwinSandboxProfile(packDir)

	// packDir read must be present and symlink-resolved.
	if !strings.Contains(profile, "(subpath \""+packDir+"\")") {
		t.Fatalf("profile missing packDir read subpath %q:\n%s", packDir, profile)
	}

	// The system/runtime library read subpaths an interpreter's dyld load needs.
	// These are scoped subpaths, NOT a blanket (allow file-read*).
	for _, want := range []string{
		"/usr/lib",
		"/System/Library",
		"/usr/local/lib",
		"/opt/homebrew",
		"/private/var/db/dyld",
	} {
		if !strings.Contains(profile, "(subpath \""+want+"\")") {
			t.Errorf("profile missing required dyld/system read subpath %q:\n%s", want, profile)
		}
	}

	// bsd.sb base import — empirically required so dyld can read the shared cache;
	// without it every restricted file-read* profile SIGABRTs at launch.
	if !strings.Contains(profile, "(import \"bsd.sb\")") {
		t.Errorf("profile missing (import \"bsd.sb\") base — restricted file-read* profiles SIGABRT without it:\n%s", profile)
	}

	// Deny rules must remain HARD and must NOT be a blanket read allow.
	for _, want := range []string{"(deny default)", "(deny file-write*)", "(deny network*)"} {
		if !strings.Contains(profile, want) {
			t.Errorf("profile missing hard deny rule %q:\n%s", want, profile)
		}
	}
	// Guard against the security hole: an unscoped (allow file-read*) with no
	// filter would read every project file.
	if strings.Contains(profile, "(allow file-read*)") {
		t.Errorf("profile contains a BLANKET (allow file-read*) — must be subpath-scoped:\n%s", profile)
	}
}

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
