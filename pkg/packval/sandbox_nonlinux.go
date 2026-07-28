//go:build !linux

package packval

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// The non-Linux half of the sandbox: the darwin sandbox-exec implementation, the
// refusal for every other platform, and the trampoline's entry gate.
//
// ⚠ THIS TAG IS `!linux` AND MUST STAY `!linux`. Narrowing it to `!linux &&
// !darwin` — which looks tidy once platform dispatch is build-tagged — costs the
// darwin build MaybeRunSandboxHelper, and the obvious fix at that point is to
// delete the call site in cmd/backstop. That makes the build green and SILENTLY
// DISARMS THE LINUX SANDBOX: nothing in the darwin test suite would notice, and
// the shipped Linux binary would run pack-supplied code with no confinement.
// TestNonLinuxSandboxHelperTagIsNotNarrowed asserts the constraint structurally.
//
// WHY THE DARWIN ARM AND THE REFUSAL SHARE ONE FILE. The obvious split is
// sandbox_darwin.go (`darwin`) plus a `!linux && !darwin` file for the refusal,
// and Phase 3b built it that way first. It is unshippable for a reason that has
// nothing to do with taste: `coverage_unmeasured` fires per FILE, and a file no
// CI platform compiles never produces a coverage record at all. A `!linux &&
// !darwin` file resolves NOWHERE in a darwin-development, linux-CI matrix, so it
// is permanently RED — measured 2026-07-28 — which would have put CLM-028's
// unconditional green permanently out of reach. Folding both into `!linux` costs
// exactly two dead statements on darwin (the guard's error returns below) and
// nothing at all on Linux, which compiles none of this file.
//
// What the phase set out to delete is deleted either way: sandbox.go carries no
// platform branch, and neither linux nor darwin compiles the other's arms.

// MaybeRunSandboxHelper is the non-Linux entry gate for the re-exec trampoline.
//
// It is NOT a dispatch arm, which is why this file's tag is wider than dispatch
// alone would need. cmd/backstop's run() calls it UNCONDITIONALLY on every
// platform as the first thing it does, so the symbol must exist everywhere. It
// returns an error to match the linux signature, and nil is the only value it can
// produce: "this process is not a sandbox helper" is the permanent truth on a
// platform with no helper mode.
func MaybeRunSandboxHelper() error { return nil }

// darwinSandboxProfile builds the macOS sandbox-exec profile shared by
// SandboxedRun and SandboxedRunStdout — ONE maintenance point for the
// read-allowlist (ISSUE-029, CLM-001). It keeps the trust model HARD —
// (deny default), (deny file-write*), (deny network*) — while granting the
// MINIMAL file-read* set a dynamically-linked convert interpreter (jq, python3,
// node, ...) needs at dyld load.
//
// Two non-obvious, empirically-established requirements (ISSUE-029):
//   - (import "bsd.sb"): the base system profile. Without it, ANY restricted
//     file-read* profile SIGABRTs at launch because dyld cannot read the shared
//     cache; with it dyld reaches a real, debuggable denial. bsd.sb does NOT
//     grant arbitrary project-file reads, file writes, or network — the deny
//     rules below still hold (verified by TestSandboxSecurityDenialsHold).
//   - packDir is symlink-resolved (filepath.EvalSymlinks): a sandbox subpath
//     rule matches the KERNEL-resolved path, so an unresolved /var/... subpath
//     would silently fail to match the real /private/var/... and deny legit
//     reads inside packDir.
//
// The added system/runtime read subpaths (alongside packDir) are scoped, NOT a
// blanket (allow file-read*) — that would be a security hole. They cover the
// dyld shared cache and the dirs a Homebrew interpreter's dylibs live in on both
// Intel (/usr/local/...) and Apple-Silicon (/opt/homebrew) hosts. NO project /
// non-pack / non-system path is readable.
func darwinSandboxProfile(packDir string) string {
	resolved := packDir
	if r, err := filepath.EvalSymlinks(packDir); err == nil {
		resolved = r
	}
	readSubpaths := []string{
		resolved,                            // the pack directory itself (the only project path)
		"/usr/lib",                          // system dylibs
		"/System/Library",                   // system frameworks / libraries
		"/usr/local/lib",                    // Intel Homebrew libs
		"/usr/local/Cellar",                 // Intel Homebrew keg-only installs (e.g. libjq)
		"/usr/local/opt",                    // Intel Homebrew opt symlinks (e.g. oniguruma)
		"/opt/homebrew",                     // Apple-Silicon Homebrew prefix
		"/private/var/db/dyld",              // dyld shared cache (classic location)
		"/System/Volumes/Preboot/Cryptexes", // dyld shared cache (Cryptexes location)
	}
	var b strings.Builder
	for _, p := range readSubpaths {
		fmt.Fprintf(&b, " (subpath \"%s\")", p)
	}
	return fmt.Sprintf(
		"(version 1)(import \"bsd.sb\")(deny default)(allow process*)(allow file-read*%s)(deny network*)(deny file-write*)",
		b.String(),
	)
}

// sandboxExecCommand builds the sandbox-exec invocation both arms share, so the
// profile and the argv construction have exactly one definition.
func sandboxExecCommand(cmd string, args []string, packDir string) *exec.Cmd {
	fullArgs := []string{"-p", darwinSandboxProfile(packDir), cmd}
	fullArgs = append(fullArgs, args...)
	c := exec.Command("sandbox-exec", fullArgs...)
	c.Dir = packDir
	return c
}

// platformSandboxedRun is the non-Linux arm of SandboxedRun. It preserves that
// function's CombinedOutput contract exactly.
//
// The guard is what makes this file's wide tag safe. On a platform with no
// sandbox at all the answer is an ERROR, never an empty success: a (nil, nil)
// here would hand the caller zero bytes and no error, which the gate reads as a
// convert step that produced no findings — the vacuous green ISSUE-020 exists to
// eliminate, arriving through a platform arm nobody looks at. The message is the
// one the retired `switch runtime.GOOS` default branch produced.
func platformSandboxedRun(cmd string, args []string, packDir string) ([]byte, error) {
	if err := sandboxPlatformSupported(runtime.GOOS); err != nil {
		return nil, fmt.Errorf("sandboxed run: %w", err)
	}
	out, err := sandboxExecCommand(cmd, args, packDir).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("sandboxed run failed: %w", err)
	}
	return out, nil
}

// platformSandboxedRunStdout is the non-Linux arm of SandboxedRunStdout: the same
// profile, trust model and refusal, but stdout is captured through an explicit
// buffer so a converter's stderr banner cannot interleave into the SARIF bytes,
// and the optional stdin is fed to the command. On a non-zero exit it returns the
// stdout captured so far alongside the error.
func platformSandboxedRunStdout(cmd string, args []string, packDir string, stdin []byte) ([]byte, error) {
	if err := sandboxPlatformSupported(runtime.GOOS); err != nil {
		return nil, fmt.Errorf("sandboxed run (stdout): %w", err)
	}
	c := sandboxExecCommand(cmd, args, packDir)
	if stdin != nil {
		c.Stdin = bytes.NewReader(stdin)
	}
	var stdout bytes.Buffer
	c.Stdout = &stdout
	if err := c.Run(); err != nil {
		return stdout.Bytes(), fmt.Errorf("sandboxed run (stdout) failed: %w", err)
	}
	return stdout.Bytes(), nil
}
