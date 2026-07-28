package packval

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

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

func SandboxedRun(cmd string, args []string, packDir string) ([]byte, error) {
	switch runtime.GOOS {
	case "darwin":
		profile := darwinSandboxProfile(packDir)
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
		return linuxSandboxedRun(cmd, args, packDir)
	default:
		return nil, sandboxPlatformSupported(runtime.GOOS)
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
		profile := darwinSandboxProfile(packDir)
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
		return linuxSandboxedRunStdout(cmd, args, packDir, stdin)
	default:
		return nil, sandboxPlatformSupported(runtime.GOOS)
	}
}
