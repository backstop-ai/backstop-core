//go:build linux

package packval

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/check"
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
// ─── THE PARENT-SIDE STATEMENTS THAT STAY MEASURED ────────────────────────────
// Measured on run 30389988184: kernelRelease's Uname failure, probeLandlockABI's
// errno branch, and newSandboxHelperInvocation's json.Marshal and os.Executable
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
// one seam judged worth its risk, and it ships with a wiring guard. Returning
// decode, policy, acknowledgement, and injected orchestration logic remains in this
// measured file; only the successful non-returning wrapper is excluded.

// sandboxHelperRequest is what the parent hands the helper.
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

func runSandboxHelper(spec string) error {
	var request sandboxHelperRequest
	if err := json.Unmarshal([]byte(spec), &request); err != nil {
		return fmt.Errorf("decode the sandbox helper request: %w", err)
	}
	if err := applyRestrictionsAndExec(request); err != nil {
		return fmt.Errorf("install the sandbox restrictions: %w", err)
	}
	return errors.New("the sandbox helper returned without exec'ing the command")
}

func applyRestrictionsAndExecWith(
	request sandboxHelperRequest,
	applyFilesystem func(SandboxRestrictionSpec) error,
	applySyscalls func(SandboxRestrictionSpec) error,
	acknowledge func(int) error,
	changeDir func(string) error,
	lookPath func(string) (string, error),
	execTarget func(string, []string, []string) error,
) error {
	restrictions := DeriveSandboxRestrictions(request.Capability)
	if err := applyFilesystem(restrictions); err != nil {
		return err
	}
	if err := applySeccompPolicy(restrictions, applySyscalls); err != nil {
		return err
	}
	if err := acknowledge(request.AckFD); err != nil {
		return err
	}
	if request.Dir != "" {
		if err := changeDir(request.Dir); err != nil {
			return fmt.Errorf("chdir %s: %w", request.Dir, err)
		}
	}
	resolved, err := lookPath(request.Command)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", request.Command, err)
	}
	argv := append([]string{resolved}, request.Args...)
	if err := execTarget(resolved, argv, request.Environment); err != nil {
		return fmt.Errorf("exec %s: %w", resolved, err)
	}
	return nil
}

func applySeccompPolicy(restrictions SandboxRestrictionSpec, install func(SandboxRestrictionSpec) error) error {
	if len(restrictions.SeccompDenied) == 0 {
		return nil
	}
	return install(restrictions)
}

func seccompAuditArch(goarch string) (uint32, error) {
	switch goarch {
	case "amd64":
		return uint32(unix.AUDIT_ARCH_X86_64), nil
	case "arm64":
		return uint32(unix.AUDIT_ARCH_AARCH64), nil
	default:
		return 0, fmt.Errorf("no seccomp audit arch mapping for GOARCH %s", goarch)
	}
}

func seccompSyscallNumbers(names []string) ([]uint32, error) {
	byName := map[string]int{
		"socket": unix.SYS_SOCKET, "socketpair": unix.SYS_SOCKETPAIR,
		"connect": unix.SYS_CONNECT, "bind": unix.SYS_BIND,
		"sendto": unix.SYS_SENDTO, "sendmsg": unix.SYS_SENDMSG, "sendmmsg": unix.SYS_SENDMMSG,
		"recvfrom": unix.SYS_RECVFROM, "recvmsg": unix.SYS_RECVMSG, "recvmmsg": unix.SYS_RECVMMSG,
		"io_uring_setup": unix.SYS_IO_URING_SETUP, "io_uring_enter": unix.SYS_IO_URING_ENTER,
	}
	numbers := make([]uint32, 0, len(names))
	for _, name := range names {
		number, known := byName[name]
		if !known {
			return nil, fmt.Errorf("no syscall number for %q", name)
		}
		numbers = append(numbers, uint32(number))
	}
	return numbers, nil
}

// filterHelperEnv strips the helper variable so the exec'd command — which may
// itself be backstop, or may spawn it — does not re-enter helper mode and loop.
func filterHelperEnv(environment []string) []string {
	return check.WithoutEnvironment(environment, sandboxHelperEnvVar, PackSandboxEnvVar)
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

// newSandboxHelperInvocation builds the parent-side command that re-execs this binary
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
type sandboxHelperInvocation struct {
	command *exec.Cmd
	ackRead *os.File
}

func newSandboxHelperInvocation(command string, args []string, packDir string, probeABI LandlockABIProbe) (*sandboxHelperInvocation, error) {
	abi, err := resolveLandlockMechanism(probeABI)
	if err != nil {
		return nil, fmt.Errorf("negotiate the Landlock mechanism: %w", err)
	}

	request := sandboxHelperRequest{
		Capability:  ConvertValidatorCapability(packDir, abi),
		Command:     command,
		Args:        args,
		Dir:         packDir,
		Environment: filterHelperEnv(os.Environ()),
		AckFD:       sandboxAckFD,
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
	ackRead, ackWrite, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create sandbox acknowledgement pipe: %w", err)
	}
	helper.ExtraFiles = []*os.File{ackWrite}
	return &sandboxHelperInvocation{command: helper, ackRead: ackRead}, nil
}

// platformSandboxedRun is the linux arm of SandboxedRun. It preserves that function's
// CombinedOutput contract exactly.
//
// The name is the build-tagged dispatch seam it shares with sandbox_nonlinux.go:
// sandbox.go calls it unqualified and the linker resolves whichever file the build
// tags admitted.
// The dispatch seam keeps its PLATFORM-NEUTRAL signature and delegates. The prober
// is not a parameter here on purpose: platformSandboxedRun is defined identically in
// sandbox_nonlinux.go, and threading a Landlock-only dependency through it would leak
// a linux concept into a contract darwin also implements — forcing the darwin arm to
// accept an argument it can never use.
func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	result, err := linuxSandboxedExecuteWith(command, args, packDir, nil, false, probeLandlockABI)
	return result.Output, err
}

// linuxSandboxedRunWith is platformSandboxedRun's body with the ABI prober injected,
// which is what makes the refusal wrap below reachable: on a healthy host the real
// probe always succeeds, so without this seam the wrap could never execute.
func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) ([]byte, error) {
	result, err := linuxSandboxedExecuteWith(command, args, packDir, nil, false, probeABI)
	return result.Output, err
}

// platformSandboxedRunStdout is the linux arm of SandboxedRunStdout. It preserves that
// function's contract exactly: an explicit stdout buffer, the stdin pipe, and the
// stdout-captured-so-far returned alongside the error on a non-zero exit. Those are
// what keep a converter's stderr banner out of the SARIF the gate parses, so the
// trampoline has to be transparent to them.
// Same neutral-signature-plus-delegation shape as platformSandboxedRun above, and for
// the same reason.
func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	result, err := linuxSandboxedExecuteWith(command, args, packDir, stdin, true, probeLandlockABI)
	return result.Output, err
}

// linuxSandboxedRunStdoutWith is platformSandboxedRunStdout's body with the ABI prober
// injected. See linuxSandboxedRunWith for why the seam exists.
func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	invocation, err := newSandboxHelperInvocation(command, args, packDir, probeABI)
	if err != nil {
		return nil, fmt.Errorf("prepare the linux sandbox: %w", err)
	}
	helper := invocation.command
	if stdin != nil {
		helper.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	helper.Stdout = &stdout
	helper.Stderr = &stderr
	if err := helper.Start(); err != nil {
		_ = invocation.ackRead.Close()
		_ = helper.ExtraFiles[0].Close()
		return nil, fmt.Errorf("sandboxed run (stdout) failed: %w", err)
	}
	_ = helper.ExtraFiles[0].Close()
	if _, err := io.ReadAll(invocation.ackRead); err != nil {
		_ = invocation.ackRead.Close()
		return nil, fmt.Errorf("read sandbox acknowledgement: %w", err)
	}
	_ = invocation.ackRead.Close()
	return foldHelperStderrIntoError(stdout.Bytes(), stderr.Bytes(), helper.Wait())
}

func writeSandboxAcknowledgement(fd int) error {
	file := os.NewFile(uintptr(fd), "sandbox-ack")
	if file == nil {
		return fmt.Errorf("open sandbox acknowledgement descriptor")
	}
	if _, err := file.Write([]byte{sandboxAckByte}); err != nil {
		_ = file.Close()
		return fmt.Errorf("write sandbox acknowledgement: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close sandbox acknowledgement: %w", err)
	}
	return nil
}

func platformSandboxedExecute(command string, args []string, packDir string, stdin []byte, stdoutOnly bool) (SandboxRunResult, error) {
	return linuxSandboxedExecuteWith(command, args, packDir, stdin, stdoutOnly, probeLandlockABI)
}

func linuxSandboxedExecuteWith(command string, args []string, packDir string, stdin []byte, stdoutOnly bool, probeABI LandlockABIProbe) (SandboxRunResult, error) {
	invocation, err := newSandboxHelperInvocation(command, args, packDir, probeABI)
	if err != nil {
		return SandboxRunResult{}, fmt.Errorf("prepare the linux sandbox: %w", err)
	}
	helper := invocation.command
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
	if stdoutOnly {
		helper.Stdout = &stdout
		helper.Stderr = &stderr
	} else {
		helper.Stdout = &stdout
		helper.Stderr = &stdout
	}
	if err := helper.Start(); err != nil {
		_ = invocation.ackRead.Close()
		_ = helper.ExtraFiles[0].Close()
		return SandboxRunResult{}, fmt.Errorf("sandboxed run failed: %w", err)
	}
	_ = helper.ExtraFiles[0].Close()
	ack, ackErr := io.ReadAll(invocation.ackRead)
	_ = invocation.ackRead.Close()
	runErr := helper.Wait()
	applied := ackErr == nil && len(ack) == 1 && ack[0] == sandboxAckByte
	if stdoutOnly {
		out, foldedErr := foldHelperStderrIntoError(stdout.Bytes(), stderr.Bytes(), runErr)
		return SandboxRunResult{Output: out, NativeSandboxApplied: applied}, foldedErr
	}
	if runErr != nil {
		return SandboxRunResult{Output: stdout.Bytes(), NativeSandboxApplied: applied}, fmt.Errorf("sandboxed run failed: %w", runErr)
	}
	return SandboxRunResult{Output: stdout.Bytes(), NativeSandboxApplied: applied}, nil
}
