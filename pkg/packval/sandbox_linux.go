//go:build linux

package packval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"
)

// The Linux sandbox: Landlock for the filesystem, seccomp for the network.
//
// ─── WHY NOT BUBBLEWRAP ─────────────────────────────────────────────────────────
// This plan was written against bubblewrap. A probe on a REAL ubuntu-latest runner
// falsified that choice and the founder ruled on 2026-07-28. The evidence is
// pkg/packval/testdata/ubuntu-runner-probe.txt (GitHub Actions run 30364880850,
// ubuntu-24.04, kernel 6.17.0-1020-azure, unprivileged uid 1001):
//
//	apparmor_restrict_unprivileged_userns = 1        RESTRICTED
//	all SEVEN bwrap invocations exited 1, INCLUDING BOTH POSITIVE CONTROLS
//	  ("loopback: Failed RTM_NEWADDR" / "setting up uid map: Permission denied")
//	`unshare -Urn /bin/true` failed identically -> host policy, not a bwrap defect
//	LANDLOCK_ABI = 7, LSM list carries landlock
//
// The restriction is stock consumer Ubuntu 24.04, not a CI quirk. The sandbox has to
// work UNPRIVILEGED on hosts we do not control, and "run sudo first" is not a promise
// backstop makes — a mechanism that needs a host policy change is not a mechanism.
//
// ─── WHY A RE-EXEC TRAMPOLINE ───────────────────────────────────────────────────
// Bubblewrap is a wrapper binary: you build an argv and exec it. Landlock and seccomp
// are neither. They are SELF-APPLIED, PER-THREAD and IRREVOCABLE — a process installs
// them on ITSELF, and they are then inherited across fork and execve. Applying them
// in-process inside SandboxedRun would permanently confine the backstop process: no
// further project reads, no report writing, no subsequent gate steps, and no way to
// undo it.
//
// So the parent spawns /proc/self/exe in a hidden helper mode, and the helper
// restricts itself and then execs the real command.
//
// STEPS 1 AND 5 BELOW ARE A PAIR, AND SPLITTING THEM IS A SILENT HOLE. Both
// restrictions attach to the CALLING THREAD. Go's scheduler migrates goroutines
// across OS threads at will, so restricting on one thread and exec'ing from another
// yields an UNCONFINED CHILD while every syscall returns 0 and every unit test still
// passes. runtime.LockOSThread pins the goroutine, and execve from that thread
// replaces the process image, so the new image inherits that thread's Landlock domain
// and seccomp filter. TestLinuxSandbox_ConfinementCarriesIntoTheExecdChild asserts
// this from INSIDE the exec'd process, which is the only place it is observable.

// sandboxHelperEnvVar carries the marshalled helper request. Its PRESENCE is what
// puts the binary into helper mode.
//
// It is an env var rather than a Cobra subcommand deliberately: /proc/self/exe is the
// backstop binary, so the gate has to run before Cobra parses argv, and a real
// subcommand would also be a user-facing surface promising something it is not.
const sandboxHelperEnvVar = "BACKSTOP_SANDBOX_HELPER_SPEC"

// ─── THE FOUR STATEMENTS THAT STAY UNCOVERED, AND WHY THAT IS CORRECT ──────────
// Measured on run 30389988184: kernelRelease's Uname failure, probeLandlockABI's
// errno branch, and newSandboxHelperCommand's json.Marshal and os.Executable
// failures. Every one is the error return of a call that CANNOT fail on a healthy
// host — Uname succeeds, the Landlock probe succeeds, os.Executable succeeds, and
// marshalling a struct with no channels or funcs cannot fail.
//
// They are LOUD FAILURE HANDLERS, not untested logic: they are what makes a broken
// host legible instead of silent. Reaching them would need a syscall-function struct
// and a marshal seam, and that is the thick-seam direction this file deliberately
// avoids — every indirection between this code and the kernel is a place the test
// and production paths can diverge, which is what produced two of this lane's runner
// failures. Do NOT delete them to raise the number; the ABI-prober seam above is the
// one seam judged worth its risk, and it ships with a wiring guard.

// sandboxHelperRequest is what the parent hands the helper.
type sandboxHelperRequest struct {
	Capability SandboxCapability `json:"capability"`
	Command    string            `json:"command"`
	Args       []string          `json:"args"`
	Dir        string            `json:"dir"`
}

// MaybeRunSandboxHelper runs the sandbox helper. On success it NEVER RETURNS,
// because the helper's last act is an execve that replaces the process image. When
// this process was not spawned as a helper it returns nil immediately, having done
// nothing.
//
// A NON-NIL RETURN IS FATAL AND THE CALLER MUST TREAT IT SO. It means this process
// IS a helper and the restrictions could NOT be installed, so there is no sandbox
// and nothing has been exec'd. Continuing would either run pack-supplied code
// unsandboxed — the exact defect ISSUE-020 exists to fix — or drop into the CLI
// while still in helper mode. Exit non-zero; do not fall through.
//
// The error is RETURNED rather than exited on because pkg/packval is a library:
// exit policy belongs to main(), which is also what the cobra-cli-standards rule
// cli.core.exit-code-discipline requires. Both call sites therefore exit
// themselves, and an ignored return is caught by the no-ignored-errors rule.
//
// It must be the FIRST statement in main() and in pkg/packval's TestMain. Under
// `go test` the exe is the TEST binary, so both entry points need it; either one
// missing is a silent hole — the shipped binary sandboxes nothing, or the suite
// cannot reach the helper at all.
//
// ─── THE SPLIT ──────────────────────────────────────────────────────────────────
// This function is SPLIT DELIBERATELY, and the seam is a measurement boundary
// rather than a design preference.
//
// Everything below runs in EVERY backstop process: the env lookup and the
// not-a-helper return are the overwhelmingly common path, and they are honestly
// measurable. Everything past the dispatch runs only inside the re-exec helper,
// which ends in unix.Exec — the process image is replaced and Go never flushes
// those counters (evidence: pkg/packval/testdata/sandbox-linux-coverage-profile
// .txt). Left whole, this function stranded ~6 permanently-unmeasurable statements
// in a file that IS measured and IS enforced, where they could neither be covered
// nor excluded. The exec-side half now lives in sandbox_linux_helper.go, which the
// pack declares excluded.
func MaybeRunSandboxHelper() error {
	spec, present := os.LookupEnv(sandboxHelperEnvVar)
	if !present {
		return nil
	}
	return runSandboxHelper(spec)
}

// filterHelperEnv strips the helper variable so the exec'd command — which may
// itself be backstop, or may spawn it — does not re-enter helper mode and loop.
func filterHelperEnv(environment []string) []string {
	kept := make([]string, 0, len(environment))
	for _, entry := range environment {
		if strings.HasPrefix(entry, sandboxHelperEnvVar+"=") {
			continue
		}
		kept = append(kept, entry)
	}
	return kept
}

// kernelRelease reports the running kernel release for diagnostics.
func kernelRelease() string {
	var uname unix.Utsname
	if err := unix.Uname(&uname); err != nil {
		return "unknown"
	}
	return strings.TrimRight(string(uname.Release[:]), "\x00")
}

// probeLandlockABI asks the kernel which Landlock ABI it implements.
func probeLandlockABI() (int, string, error) {
	release := kernelRelease()
	abi, _, errno := unix.Syscall(
		unix.SYS_LANDLOCK_CREATE_RULESET,
		0, 0,
		uintptr(unix.LANDLOCK_CREATE_RULESET_VERSION),
	)
	if errno != 0 {
		return 0, release, errno
	}
	return int(abi), release, nil
}

// newSandboxHelperCommand builds the parent-side command that re-execs this binary
// in helper mode. It negotiates the Landlock ABI first and REFUSES loudly when the
// mechanism is unavailable, executing nothing.
// probeABI is a PARAMETER so the refusal path is reachable from a test: on a healthy
// host the probe always succeeds, so the "mechanism unavailable" branch — and the two
// callers' error wraps that depend on it — could never execute. The seam is
// deliberately THIN, matching the shape resolveLandlockMechanism already takes one
// level down: it substitutes the ABI ANSWER, never what a Landlock rule is or how it
// is applied. TestSandboxLinux_ProductionPathUsesTheRealABIProbe asserts both
// production call sites hand it the real probeLandlockABI, so the seam cannot become a
// place where test and production diverge.
func newSandboxHelperCommand(command string, args []string, packDir string, probeABI LandlockABIProbe) (*exec.Cmd, error) {
	abi, err := resolveLandlockMechanism(probeABI)
	if err != nil {
		return nil, fmt.Errorf("negotiate the Landlock mechanism: %w", err)
	}

	request := sandboxHelperRequest{
		Capability: ConvertValidatorCapability(packDir, abi),
		Command:    command,
		Args:       args,
		Dir:        packDir,
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encode sandbox helper request: %w", err)
	}

	self, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve this executable for the sandbox trampoline: %w", err)
	}

	helper := exec.Command(self)
	helper.Dir = packDir
	helper.Env = append(filterHelperEnv(os.Environ()), sandboxHelperEnvVar+"="+string(encoded))
	return helper, nil
}

// platformSandboxedRun is the linux arm of SandboxedRun. It preserves that function's
// CombinedOutput contract exactly.
//
// The name is the build-tagged dispatch seam it shares with sandbox_nonlinux.go:
// sandbox.go calls it unqualified and the linker resolves whichever file the build
// tags admitted.
func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeLandlockABI)
	if err != nil {
		return nil, fmt.Errorf("prepare the linux sandbox: %w", err)
	}
	out, err := helper.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("sandboxed run failed: %w", err)
	}
	return out, nil
}

// platformSandboxedRunStdout is the linux arm of SandboxedRunStdout. It preserves that
// function's contract exactly: an explicit stdout buffer, the stdin pipe, and the
// stdout-captured-so-far returned alongside the error on a non-zero exit. Those are
// what keep a converter's stderr banner out of the SARIF the gate parses, so the
// trampoline has to be transparent to them.
func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeLandlockABI)
	if err != nil {
		return nil, fmt.Errorf("prepare the linux sandbox: %w", err)
	}
	if stdin != nil {
		helper.Stdin = bytes.NewReader(stdin)
	}
	// TWO SEPARATE BUFFERS, and they must never be aliased to each other. stdout is
	// what the gate parses as SARIF; stderr is where the helper writes the reason it
	// could not install the sandbox. Pointing both at one buffer would satisfy every
	// test here and reintroduce the stderr-in-SARIF corruption this arm was built to
	// prevent — surfacing far from here as unparseable SARIF from any converter that
	// writes a banner. Leaving stderr NIL is the other trap, and the one that
	// actually shipped: os/exec sends a nil Stderr to /dev/null, which is how the
	// helper's CLM-015 diagnostic was lost in CI run 30381252600.
	var stdout, stderr bytes.Buffer
	helper.Stdout = &stdout
	helper.Stderr = &stderr

	runErr := helper.Run()
	return foldHelperStderrIntoError(stdout.Bytes(), stderr.Bytes(), runErr)
}
