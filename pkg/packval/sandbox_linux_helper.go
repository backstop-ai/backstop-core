//go:build linux

package packval

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// THE EXEC SIDE OF THE SANDBOX — everything that runs inside the re-exec helper.
//
// WHY THIS FILE EXISTS, AND WHY IT IS A MEASUREMENT BOUNDARY RATHER THAN A TIDY-UP.
// Every function here executes in the helper process, which ends in unix.Exec.
// exec REPLACES THE PROCESS IMAGE, so Go never flushes the coverage counters for
// anything that ran first. The runner profile captured at
// pkg/packval/testdata/sandbox-linux-coverage-profile.txt shows the signature
// exactly: applyRestrictionsAndExec 0/19, applyLandlock 0/29, applySeccomp 0/19,
// seccompAuditArch 0/4, seccompSyscallNumbers 0/8 — five functions at EXACTLY
// zero, on a run where the sandbox demonstrably installed and held.
//
// The code is not untested; the instrument cannot observe it. That distinction is
// the whole justification for the pack-declared coverage exclusion this path
// carries, and it is why the exclusion names THIS FILE rather than the package:
// sandbox_linux.go next door stays measured and enforced.
//
// ⚠ DO NOT MOVE MEASURABLE CODE IN HERE. The exclusion is a statement about the
// EXEC BOUNDARY, not a convenient place to put code that is merely awkward to
// test. Anything that can run to completion in a normal process belongs in
// sandbox_linux.go, where it is measured.

// runSandboxHelper is the exec-side half of MaybeRunSandboxHelper: this process IS
// a helper, so decode the request and install the restrictions.
//
// Its counters never flush — a successful call never returns, and the failure
// paths run in a process the parent is about to reap.
func runSandboxHelper(spec string) error {
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
	//
	// Do NOT delete this guard to raise a percentage: it is excluded from
	// measurement precisely so that it can stay.
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
		// The inode type comes from the descriptor the rule REGISTERS, not from a
		// separate stat by path. Fstat on this fd is race-free by construction: a
		// path-based stat could describe a different inode than the one path_beneath
		// binds, and a filename heuristic could not describe it at all.
		//
		// The narrowing itself is required, not defensive. landlock_add_rule rejects
		// directory-only rights against a non-directory with EINVAL, and one rejected
		// rule aborts the whole install — run 30383453888 died on exactly this, mask
		// 0xc against the regular file /etc/ld.so.cache.
		var stat unix.Stat_t
		if statErr := unix.Fstat(pathFD, &stat); statErr != nil {
			_ = unix.Close(pathFD)
			// Guessing the type would reintroduce the defect, so a required path whose
			// type cannot be read is fatal rather than assumed.
			if rule.Required {
				return fmt.Errorf("stat required sandbox path %s: %w", rule.Path, statErr)
			}
			continue
		}
		isDir := stat.Mode&unix.S_IFMT == unix.S_IFDIR

		beneath := unix.LandlockPathBeneathAttr{
			Allowed_access: narrowRuleToInodeType(rule.AllowedAccess, isDir),
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
