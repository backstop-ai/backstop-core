//go:build darwin

package packval

import (
	"os"
	"path/filepath"
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

// TestSandboxProfileGrantsDevNullWriteAndNothingElse pins the ISSUE-168 carve-out in
// the darwin profile literal: /dev/null is writable, and the blanket write denial is
// otherwise completely intact.
//
// ★ THIS IS THE DARWIN RED. It is the one assertion in this lane that is false before
// the fix and true after it. The BEHAVIOURAL darwin test below is not — see its
// docstring — because Seatbelt already permitted the write as an emergent property
// the profile text never stated. What this test locks is that the profile now SAYS
// what the platform does, which is what stops a future macOS tightening (or a reader
// deleting the clause as decorative) from silently re-opening ISSUE-168 here.
func TestSandboxProfileGrantsDevNullWriteAndNothingElse(t *testing.T) {
	packDir := mustEvalSymlinks(t, t.TempDir())
	profile := darwinSandboxProfile(packDir)

	const (
		allowDevNull = "(allow file-write* (literal \"/dev/null\"))"
		denyWrites   = "(deny file-write*)"
	)

	if !strings.Contains(profile, allowDevNull) {
		t.Fatalf("profile is missing the scoped /dev/null write allow %q — a sandboxed convert or "+
			"validator script cannot use `command -v foo >/dev/null 2>&1` (ISSUE-168):\n%s",
			allowDevNull, profile)
	}

	// The carve-out is an ADDITION, never a replacement: the blanket deny must survive.
	if !strings.Contains(profile, denyWrites) {
		t.Errorf("profile no longer carries the blanket %q; the /dev/null allow is a scoped exception "+
			"to that deny and is meaningless without it — every path in the filesystem would be "+
			"writable:\n%s", denyWrites, profile)
	}

	// ORDERING IS LOAD-BEARING. Seatbelt evaluates LAST-MATCH-WINS, so the scoped allow
	// must follow the blanket deny or the deny overrides it. This is the arrangement
	// measured working in a production-shaped profile through real sandbox-exec
	// (2026-08-18). The reverse order also worked in an authoring probe — recorded so
	// nobody re-derives it, NOT as licence to move the clause, because the measured
	// arrangement is the one that ships.
	allowAt := strings.Index(profile, allowDevNull)
	denyAt := strings.Index(profile, denyWrites)
	if allowAt < denyAt {
		t.Errorf("the /dev/null allow is at index %d, BEFORE the blanket deny at %d. Seatbelt is "+
			"last-match-wins, so a preceding allow is overridden by the deny that follows it and the "+
			"carve-out silently does nothing:\n%s", allowAt, denyAt, profile)
	}

	// Guard against the security hole this carve-out could be mistaken for. Note the
	// guard is not self-defeating: the new clause is `(allow file-write* (literal ...`,
	// which does not contain the substring below — the closing paren differs.
	if strings.Contains(profile, "(allow file-write*)") {
		t.Errorf("profile contains a BLANKET (allow file-write*) — the carve-out must be scoped to a "+
			"single literal path:\n%s", profile)
	}

	// `literal`, never `subpath`. (subpath "/dev") would grant write to every device
	// node on the system, which is a genuine widening rather than a null sink.
	if strings.Contains(profile, "(subpath \"/dev\")") {
		t.Errorf("profile grants a subpath under /dev — that is every device node, not the one "+
			"write-only sink this carve-out is scoped to:\n%s", profile)
	}
}

// TestSandboxDarwin_DevNullIdiomRunsUnderTheRealProfile drives the ISSUE-168 idiom
// through the PRODUCTION path — real sandbox-exec, the real profile, no stub — and
// pairs it with the anti-widening counterweight in the same test.
//
// ★ CLASSIFY THIS HONESTLY: IT IS A REGRESSION LOCK, NOT A RED→GREEN. Measured on a
// real macOS host on 2026-08-18, by running the CURRENT (pre-fix) profile literal
// through real sandbox-exec: `command -v jq >/dev/null 2>&1` already succeeded and
// printed its marker, while `touch` inside packDir was still refused with "Operation
// not permitted". Darwin permits writes to device nodes as an emergent Seatbelt
// property that the profile text never stated (/dev/zero behaves identically), while
// Linux's Landlock enforced the same stated intent literally and broke the idiom.
// THAT ASYMMETRY IS THE WHOLE DEFECT. So this test PASSES BEFORE THE FIX; it goes red
// only if a future macOS starts enforcing the blanket deny, or if someone removes the
// clause believing it decorative. The genuine darwin red is
// TestSandboxProfileGrantsDevNullWriteAndNothingElse above.
//
// The completion marker is what distinguishes "the write was permitted" from "the
// sandbox never started" — a bare exit code cannot tell those apart, which is this
// package's standing rule (see the sandbox_linux_exec_test.go header on
// ubuntu-runner-probe.txt). jq's presence is deliberately NOT asserted: the idiom is
// under test, not the tool it happens to look for.
func TestSandboxDarwin_DevNullIdiomRunsUnderTheRealProfile(t *testing.T) {
	packDir := mustEvalSymlinks(t, t.TempDir())

	script := filepath.Join(packDir, "devnull-idiom.sh")
	body := "#!/bin/sh\n" +
		"if command -v jq >/dev/null 2>&1; then echo IDIOM_OK; else echo IDIOM_OK_NO_JQ; fi\n" +
		"echo probe > /dev/null 2>&1\n" +
		"echo DEVNULL_PROBE_COMPLETED\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatalf("write the idiom probe %q: %v", script, err)
	}

	out, err := SandboxedRunStdout("/bin/sh", []string{script}, packDir, nil)
	if err != nil {
		t.Fatalf("the /dev/null idiom failed under the production sandbox profile: %v\n"+
			"This is ISSUE-168 on darwin: `command -v foo >/dev/null 2>&1` is a universal shell idiom "+
			"and a pack convert/validator script must be able to use it.\noutput: %q", err, string(out))
	}
	if !strings.Contains(string(out), "DEVNULL_PROBE_COMPLETED") {
		t.Fatalf("the probe never reached its completion marker, so nothing above can be read as a "+
			"permission — the sandbox may simply have failed to start: %q", string(out))
	}

	// THE ANTI-WIDENING LEG. Without it this test cannot tell a narrow carve-out from a
	// profile that stopped denying writes altogether: both would let the idiom through.
	target := filepath.Join(packDir, "should-not-exist.txt")
	writeOut, writeErr := SandboxedRun("/usr/bin/touch", []string{target}, packDir)
	if writeErr == nil {
		t.Fatalf("creating an ordinary file inside packDir was ALLOWED under the same profile that "+
			"grants /dev/null; the carve-out widened the write surface instead of scoping it to one "+
			"sink: output %q", string(writeOut))
	}
	if _, statErr := os.Stat(target); statErr == nil {
		t.Fatalf("%q exists despite the write denial, so the profile is not denying writes at all", target)
	}
}
