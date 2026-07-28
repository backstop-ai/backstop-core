//go:build darwin

package packval

import (
	"runtime"
	"strings"
	"testing"
)

// TestSandboxProfileAllowsDyldLibraries lives here rather than in
// sandbox_realconvert_test.go because darwinSandboxProfile is now build-tagged
// (Phase 3b, TASK-023: it sits in sandbox_nonlinux.go, `//go:build !linux`), so
// an untagged caller would break `GOOS=linux go vet ./pkg/packval/`. This file's
// name carries an implicit darwin constraint, which is the narrower and more
// honest scope for an assertion about a macOS profile.
//
// Its body is UNCHANGED — including the runtime.GOOS guard, which the build
// constraint makes redundant but which is left exactly as written so the test is
// byte-identical to the one CLM-016 requires to pass unmodified. The helpers it
// shares with the linux exec tests (mustEvalSymlinks, copyFixtureInto, locateJQ)
// stay in the untagged file.

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
