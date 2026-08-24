//go:build !linux

package packval

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/check"
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
// returns an error to match the linux signature. Nil means this process is not a
// helper; Darwin target completion is returned as an unexported typed error so
// process guards can propagate its status without exporting another API.
type sandboxHelperCompletionError struct {
	exitCode int
}

func (e sandboxHelperCompletionError) Error() string {
	return fmt.Sprintf("sandbox helper target completed with exit code %d", e.exitCode)
}

func (e sandboxHelperCompletionError) ExitCode() int { return e.exitCode }

func MaybeRunSandboxHelper() error {
	spec, present := os.LookupEnv(sandboxHelperEnvVar)
	if !present {
		return nil
	}
	if runtime.GOOS != "darwin" {
		return sandboxPlatformSupported(runtime.GOOS)
	}
	var request sandboxHelperRequest
	if err := json.Unmarshal([]byte(spec), &request); err != nil {
		return fmt.Errorf("decode the sandbox helper request: %w", err)
	}
	if err := writeDarwinSandboxAcknowledgement(request.AckFD); err != nil {
		return fmt.Errorf("acknowledge darwin sandbox installation: %w", err)
	}
	resolved, err := exec.LookPath(request.Command)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", request.Command, err)
	}
	target := exec.Command(resolved, request.Args...)
	target.Dir = request.Dir
	target.Env = append([]string(nil), request.Environment...)
	target.Stdin = os.Stdin
	target.Stdout = os.Stdout
	target.Stderr = os.Stderr
	if err := target.Start(); err != nil {
		return fmt.Errorf("start %s: %w", resolved, err)
	}
	return sandboxHelperCompletionError{exitCode: sandboxCompletionExitCode(target.Wait())}
}

const sandboxCompletionFallbackExitCode = 125

func sandboxCompletionExitCode(waitErr error) int {
	if waitErr == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if !errors.As(waitErr, &exitErr) {
		return sandboxCompletionFallbackExitCode
	}
	if code := exitErr.ExitCode(); code >= 0 {
		return code
	}

	// syscall.WaitStatus is platform-specific. Reflection keeps this shared
	// !linux file compilable on Windows while using its Signal method on Darwin.
	if state := exitErr.ProcessState; state != nil {
		value := reflect.ValueOf(state.Sys())
		if !value.IsValid() {
			return sandboxCompletionFallbackExitCode
		}
		method := value.MethodByName("Signal")
		if method.IsValid() && method.Type().NumIn() == 0 && method.Type().NumOut() == 1 {
			result := method.Call(nil)[0]
			if result.Kind() >= reflect.Int && result.Kind() <= reflect.Int64 && result.Int() > 0 {
				return 128 + int(result.Int())
			}
		}
	}
	return sandboxCompletionFallbackExitCode
}

func writeDarwinSandboxAcknowledgement(fd int) error {
	file := os.NewFile(uintptr(fd), "sandbox-ack")
	if file == nil {
		return fmt.Errorf("open sandbox acknowledgement descriptor")
	}
	if _, err := file.Write([]byte{sandboxAckByte}); err != nil {
		_ = file.Close()
		return fmt.Errorf("write sandbox acknowledgement: %w", err)
	}
	return file.Close()
}

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
//
// ─── THE ONE SCOPED WRITE EXCEPTION: /dev/null (ISSUE-168) ──────────────────────
// The write denial above is otherwise total, and it stays total for every path that
// can hold state. /dev/null is a write-only sink: nothing written to it persists,
// leaks or can be corrupted, so the exception costs the trust model nothing. It
// exists because `command -v foo >/dev/null 2>&1` is a universal shell idiom that a
// pack-supplied convert or validator script has every right to use, and because the
// Linux half of this sandbox grants the same single path through a Landlock rule in
// sandbox_capability.go — the two platforms must say the same thing.
//
// It is a `literal`, never a `subpath`: `(subpath "/dev")` would grant write to every
// device node on the system. It is placed AFTER `(deny file-write*)` because Seatbelt
// evaluates LAST-MATCH-WINS, so a preceding allow would simply be overridden.
//
// ★ THE MEASURED FACT, FIRST: ON DARWIN THIS CLAUSE CHANGES NO OBSERVABLE BEHAVIOUR.
// Writes to /dev/null ALREADY succeeded under the profile without it — measured
// 2026-08-18 through real sandbox-exec — as an emergent Seatbelt property of device
// nodes generally (/dev/zero behaves identically) that the profile text never stated.
// Linux's Landlock, meanwhile, enforced the same stated intent literally and broke
// the idiom. THAT ASYMMETRY IS THE WHOLE DEFECT.
//
// ★ AND THEN THE FRAMING: what the clause buys is that darwin's stated intent now
// matches its actual behaviour, and that both platforms' profiles say the same thing
// — a guarantee where there was an accident. That is what stops a future macOS
// tightening, or a reader deleting the clause as decorative, from silently
// re-opening ISSUE-168 on the platform where it currently happens to work.
func darwinSandboxProfile(packDir string) (string, error) {
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
		literal, err := seatbeltStringLiteral(p)
		if err != nil {
			return "", fmt.Errorf("encode sandbox read path %q: %w", p, err)
		}
		fmt.Fprintf(&b, " (subpath %s)", literal)
	}
	return fmt.Sprintf(
		"(version 1)(import \"bsd.sb\")(deny default)(allow process*)(allow file-read*%s)(deny network*)(deny file-write*)(allow file-write* (literal \"/dev/null\"))",
		b.String(),
	), nil
}

func seatbeltStringLiteral(value string) (string, error) {
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("control character U+%04X is not allowed in a Seatbelt literal", r)
		}
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"`, nil
}

// sandboxExecCommand builds the sandbox-exec invocation both arms share, so the
// profile and the argv construction have exactly one definition.
func sandboxExecCommand(cmd string, args []string, packDir string) (*exec.Cmd, error) {
	profile, err := darwinSandboxProfile(packDir)
	if err != nil {
		return nil, fmt.Errorf("build darwin sandbox profile: %w", err)
	}
	fullArgs := []string{"-p", profile, cmd}
	fullArgs = append(fullArgs, args...)
	c := exec.Command("sandbox-exec", fullArgs...)
	c.Dir = packDir
	c.Env = check.WithoutEnvironment(os.Environ(), PackSandboxEnvVar)
	return c, nil
}

func newDarwinSandboxInvocation(command string, args []string, packDir string) (*exec.Cmd, *os.File, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, nil, fmt.Errorf("resolve this executable for the sandbox trampoline: %w", err)
	}
	environment := check.WithoutEnvironment(os.Environ(), sandboxHelperEnvVar, PackSandboxEnvVar)
	request := sandboxHelperRequest{Command: command, Args: args, Dir: packDir, Environment: environment, AckFD: sandboxAckFD}
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, nil, fmt.Errorf("encode sandbox helper request: %w", err)
	}
	cmd, err := sandboxExecCommand(self, nil, packDir)
	if err != nil {
		return nil, nil, fmt.Errorf("construct sandbox-exec command: %w", err)
	}
	cmd.Env = append(check.WithoutEnvironment(os.Environ(), sandboxHelperEnvVar, PackSandboxEnvVar), sandboxHelperEnvVar+"="+string(encoded))
	ackRead, ackWrite, err := os.Pipe()
	if err != nil {
		return nil, nil, fmt.Errorf("create sandbox acknowledgement pipe: %w", err)
	}
	cmd.ExtraFiles = []*os.File{ackWrite}
	return cmd, ackRead, nil
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
	result, err := platformSandboxedExecute(cmd, args, packDir, nil, false)
	return result.Output, err
}

// platformSandboxedRunStdout is the non-Linux arm of SandboxedRunStdout: the same
// profile, trust model and refusal, but stdout is captured through an explicit
// buffer so a converter's stderr banner cannot interleave into the SARIF bytes,
// and the optional stdin is fed to the command. On a non-zero exit it returns the
// stdout captured so far alongside the error.
func platformSandboxedRunStdout(cmd string, args []string, packDir string, stdin []byte) ([]byte, error) {
	result, err := platformSandboxedExecute(cmd, args, packDir, stdin, true)
	return result.Output, err
}

func platformSandboxedExecute(command string, args []string, packDir string, stdin []byte, stdoutOnly bool) (SandboxRunResult, error) {
	if err := sandboxPlatformSupported(runtime.GOOS); err != nil {
		return SandboxRunResult{}, fmt.Errorf("sandboxed run: %w", err)
	}
	c, ackRead, err := newDarwinSandboxInvocation(command, args, packDir)
	if err != nil {
		return SandboxRunResult{}, fmt.Errorf("prepare the darwin sandbox: %w", err)
	}
	if stdin != nil {
		c.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	if stdoutOnly {
		c.Stderr = &stderr
	} else {
		c.Stderr = &stdout
	}
	if err := c.Start(); err != nil {
		_ = ackRead.Close()
		_ = c.ExtraFiles[0].Close()
		return SandboxRunResult{}, fmt.Errorf("sandboxed run failed: %w", err)
	}
	_ = c.ExtraFiles[0].Close()
	ack, ackErr := io.ReadAll(ackRead)
	_ = ackRead.Close()
	runErr := c.Wait()
	applied := ackErr == nil && len(ack) == 1 && ack[0] == sandboxAckByte
	result := sandboxRunResult(stdout.Bytes(), applied)
	if runErr != nil {
		if stdoutOnly && stderr.Len() > 0 {
			return result, fmt.Errorf("sandboxed run (stdout) failed: %w: %s", runErr, strings.TrimSpace(stderr.String()))
		}
		return result, fmt.Errorf("sandboxed run failed: %w", runErr)
	}
	return result, nil
}
