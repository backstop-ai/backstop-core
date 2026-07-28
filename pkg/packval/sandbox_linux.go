//go:build linux

package packval

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"unsafe"

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
func MaybeRunSandboxHelper() error {
	spec, present := os.LookupEnv(sandboxHelperEnvVar)
	if !present {
		return nil
	}

	var request sandboxHelperRequest
	if err := json.Unmarshal([]byte(spec), &request); err != nil {
		return fmt.Errorf("decode the sandbox helper request: %w", err)
	}

	if err := applyRestrictionsAndExec(request); err != nil {
		return fmt.Errorf("install the sandbox restrictions: %w", err)
	}

	// UNREACHABLE BY CONTRACT: applyRestrictionsAndExec ends in unix.Exec, which
	// replaces the process image. Returning nil here would report "not a helper" to
	// a caller that IS one, which is the silent pass-through this whole mechanism
	// exists to prevent — so the impossible case is an error, not a success.
	return errors.New("the sandbox helper returned without exec'ing the command")
}

// applyRestrictionsAndExec installs the restrictions on the CURRENT THREAD and execs.
// On success it does not return.
func applyRestrictionsAndExec(request sandboxHelperRequest) error {
	// STEP 1 of the pair. Must precede the restriction syscalls and must not be
	// unlocked before the exec — see the file header.
	runtime.LockOSThread()

	restrictions := DeriveSandboxRestrictions(request.Capability)

	// PR_SET_NO_NEW_PRIVS is REQUIRED before landlock_restrict_self and before a
	// seccomp filter, absent CAP_SYS_ADMIN. Forget it and BOTH return EPERM, which
	// presents as "the sandbox is unavailable" rather than as a missing prctl.
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return fmt.Errorf("prctl(PR_SET_NO_NEW_PRIVS): %w", err)
	}

	if err := applyLandlock(restrictions); err != nil {
		return err
	}
	if err := applySeccomp(restrictions); err != nil {
		return err
	}

	if request.Dir != "" {
		if err := unix.Chdir(request.Dir); err != nil {
			return fmt.Errorf("chdir %s: %w", request.Dir, err)
		}
	}

	resolved, err := exec.LookPath(request.Command)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", request.Command, err)
	}

	// STEP 5 of the pair: exec FROM THIS LOCKED THREAD. execve replaces the process
	// image and kills sibling threads, so the new image inherits exactly the domain
	// installed above.
	argv := append([]string{resolved}, request.Args...)
	environment := filterHelperEnv(os.Environ())
	if err := unix.Exec(resolved, argv, environment); err != nil {
		return fmt.Errorf("exec %s: %w", resolved, err)
	}
	return nil
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

// applyLandlock creates the ruleset, adds the derived path rules, and restricts the
// calling thread to it.
func applyLandlock(restrictions SandboxRestrictionSpec) error {
	attr := unix.LandlockRulesetAttr{Access_fs: restrictions.HandledAccessFS}
	rulesetFD, _, errno := unix.Syscall(
		unix.SYS_LANDLOCK_CREATE_RULESET,
		uintptr(unsafe.Pointer(&attr)),
		unsafe.Sizeof(attr),
		0,
	)
	if errno != 0 {
		return fmt.Errorf("landlock_create_ruleset(handled_access_fs=0x%x): %w — an unrecognised "+
			"right returns EINVAL, so the mask must be narrowed to the running ABI",
			restrictions.HandledAccessFS, errno)
	}
	defer func() { _ = unix.Close(int(rulesetFD)) }()

	for _, rule := range restrictions.PathRules {
		pathFD, err := unix.Open(rule.Path, unix.O_PATH|unix.O_CLOEXEC, 0)
		if err != nil {
			if rule.Required {
				return fmt.Errorf("open required sandbox path %s: %w", rule.Path, err)
			}
			// A system library directory that does not exist on this distro is
			// normal (/usr/lib64 vs /lib64) and is not a sandbox failure.
			continue
		}
		beneath := unix.LandlockPathBeneathAttr{
			Allowed_access: rule.AllowedAccess,
			Parent_fd:      int32(pathFD),
		}
		_, _, errno := unix.Syscall6(
			unix.SYS_LANDLOCK_ADD_RULE,
			rulesetFD,
			unix.LANDLOCK_RULE_PATH_BENEATH,
			uintptr(unsafe.Pointer(&beneath)),
			0, 0, 0,
		)
		closeErr := unix.Close(pathFD)
		if errno != 0 {
			return fmt.Errorf("landlock_add_rule(%s, 0x%x): %w", rule.Path, rule.AllowedAccess, errno)
		}
		if closeErr != nil {
			return fmt.Errorf("close %s: %w", rule.Path, closeErr)
		}
	}

	if _, _, errno := unix.Syscall(unix.SYS_LANDLOCK_RESTRICT_SELF, rulesetFD, 0, 0); errno != 0 {
		return fmt.Errorf("landlock_restrict_self: %w (PR_SET_NO_NEW_PRIVS must precede this)", errno)
	}
	return nil
}

// seccompAuditArch returns the AUDIT_ARCH value for the running architecture.
//
// The filter validates this and KILLs on mismatch. Without the check, an i386 or x32
// entry point reaches the same kernel with DIFFERENT syscall numbers, so every
// comparison below would match the wrong call — the standard seccomp bypass.
func seccompAuditArch() (uint32, error) {
	switch runtime.GOARCH {
	case "amd64":
		return uint32(unix.AUDIT_ARCH_X86_64), nil
	case "arm64":
		return uint32(unix.AUDIT_ARCH_AARCH64), nil
	default:
		return 0, fmt.Errorf("no seccomp audit arch mapping for GOARCH %s", runtime.GOARCH)
	}
}

// seccompSyscallNumbers maps the derived syscall NAMES to numbers for this
// architecture. Names are used in the derivation so the spec is portable and
// assertable on darwin; the numbers only exist here.
func seccompSyscallNumbers(names []string) ([]uint32, error) {
	byName := map[string]int{
		"socket":         unix.SYS_SOCKET,
		"socketpair":     unix.SYS_SOCKETPAIR,
		"connect":        unix.SYS_CONNECT,
		"bind":           unix.SYS_BIND,
		"sendto":         unix.SYS_SENDTO,
		"sendmsg":        unix.SYS_SENDMSG,
		"sendmmsg":       unix.SYS_SENDMMSG,
		"recvfrom":       unix.SYS_RECVFROM,
		"recvmsg":        unix.SYS_RECVMSG,
		"recvmmsg":       unix.SYS_RECVMMSG,
		"io_uring_setup": unix.SYS_IO_URING_SETUP,
		"io_uring_enter": unix.SYS_IO_URING_ENTER,
	}
	numbers := make([]uint32, 0, len(names))
	for _, name := range names {
		number, known := byName[name]
		if !known {
			// Loud: a name the derivation produces but this table cannot resolve
			// would otherwise be silently un-denied, leaving a hole in the exact
			// surface CLM-030 rests on.
			return nil, fmt.Errorf("no syscall number for %q on %s", name, runtime.GOARCH)
		}
		numbers = append(numbers, uint32(number))
	}
	return numbers, nil
}

// applySeccomp installs the derived filter on the calling thread.
//
// The program is a TARGETED DENY with a default of ALLOW. That is not a style
// choice: between this install and the execve above, the Go runtime still makes
// syscalls, so a default-deny allowlist kills the helper before it can exec. It is
// also the correct parity call — darwin denies network, it does not enumerate
// permitted syscalls.
func applySeccomp(restrictions SandboxRestrictionSpec) error {
	if len(restrictions.SeccompDenied) == 0 {
		return nil
	}

	auditArch, err := seccompAuditArch()
	if err != nil {
		return err
	}
	numbers, err := seccompSyscallNumbers(restrictions.SeccompDenied)
	if err != nil {
		return err
	}

	const (
		loadWord    = unix.BPF_LD | unix.BPF_W | unix.BPF_ABS
		jumpEqual   = unix.BPF_JMP | unix.BPF_JEQ | unix.BPF_K
		returnValue = unix.BPF_RET | unix.BPF_K
		// Offsets into struct seccomp_data: nr at 0, arch at 4.
		offsetNR   = 0
		offsetArch = 4
	)
	denyEPERM := uint32(unix.SECCOMP_RET_ERRNO | uint32(unix.EPERM))

	program := []unix.SockFilter{
		// Validate the audit arch; KILL the process on mismatch.
		{Code: loadWord, K: offsetArch},
		{Code: jumpEqual, Jt: 1, Jf: 0, K: auditArch},
		{Code: returnValue, K: uint32(unix.SECCOMP_RET_KILL_PROCESS)},
		// Load the syscall number for the comparisons below.
		{Code: loadWord, K: offsetNR},
	}
	for _, number := range numbers {
		// On a match, jump over the remaining comparisons to the EPERM return.
		program = append(program, unix.SockFilter{Code: jumpEqual, Jt: 0, Jf: 1, K: number})
		program = append(program, unix.SockFilter{Code: returnValue, K: denyEPERM})
	}
	program = append(program, unix.SockFilter{Code: returnValue, K: uint32(unix.SECCOMP_RET_ALLOW)})

	fprog := unix.SockFprog{
		Len:    uint16(len(program)),
		Filter: &program[0],
	}
	if _, _, errno := unix.Syscall(
		unix.SYS_SECCOMP,
		uintptr(unix.SECCOMP_SET_MODE_FILTER),
		0,
		uintptr(unsafe.Pointer(&fprog)),
	); errno != 0 {
		return fmt.Errorf("seccomp(SECCOMP_SET_MODE_FILTER): %w (PR_SET_NO_NEW_PRIVS must precede this)", errno)
	}
	return nil
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
func newSandboxHelperCommand(command string, args []string, packDir string) (*exec.Cmd, error) {
	abi, err := resolveLandlockMechanism(probeLandlockABI)
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
	helper, err := newSandboxHelperCommand(command, args, packDir)
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
	helper, err := newSandboxHelperCommand(command, args, packDir)
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
