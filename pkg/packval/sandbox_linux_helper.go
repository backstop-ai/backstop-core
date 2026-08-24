//go:build linux

package packval

import (
	"fmt"
	"os/exec"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// This excluded file contains exactly the three production functions whose
// successful path ends in process replacement. Pure decisions, mappings, request
// decoding, and injected orchestration stay measured in sandbox_linux.go.

// applyRestrictionsAndExec installs irreversible process state and delegates to
// measured orchestration with the real production dependencies.
func applyRestrictionsAndExec(request sandboxHelperRequest) error {
	runtime.LockOSThread()
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return fmt.Errorf("prctl(PR_SET_NO_NEW_PRIVS): %w", err)
	}
	return applyRestrictionsAndExecWith(
		request,
		applyLandlock,
		applySeccomp,
		writeSandboxAcknowledgement,
		unix.Chdir,
		exec.LookPath,
		unix.Exec,
	)
}

// applyLandlock installs the real filesystem restriction domain.
func applyLandlock(restrictions SandboxRestrictionSpec) (returnErr error) {
	attr := unix.LandlockRulesetAttr{Access_fs: restrictions.HandledAccessFS}
	rulesetFD, _, errno := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr), 0)
	if errno != 0 {
		return fmt.Errorf("landlock_create_ruleset(handled_access_fs=0x%x): %w — an unrecognised right returns EINVAL, so the mask must be narrowed to the running ABI", restrictions.HandledAccessFS, errno)
	}
	defer func() {
		if err := unix.Close(int(rulesetFD)); err != nil && returnErr == nil {
			returnErr = fmt.Errorf("close Landlock ruleset: %w", err)
		}
	}()
	for _, rule := range restrictions.PathRules {
		pathFD, err := unix.Open(rule.Path, unix.O_PATH|unix.O_CLOEXEC, 0)
		if err != nil {
			if rule.Required {
				return fmt.Errorf("open required sandbox path %s: %w", rule.Path, err)
			}
			continue
		}
		var stat unix.Stat_t
		if statErr := unix.Fstat(pathFD, &stat); statErr != nil {
			closeErr := unix.Close(pathFD)
			if rule.Required {
				return fmt.Errorf("stat required sandbox path %s: %w", rule.Path, statErr)
			}
			if closeErr != nil {
				return fmt.Errorf("close optional sandbox path %s after stat failure: %w", rule.Path, closeErr)
			}
			continue
		}
		beneath := unix.LandlockPathBeneathAttr{
			Allowed_access: narrowRuleToInodeType(rule.AllowedAccess, stat.Mode&unix.S_IFMT == unix.S_IFDIR),
			Parent_fd:      int32(pathFD),
		}
		_, _, errno := unix.Syscall6(unix.SYS_LANDLOCK_ADD_RULE, rulesetFD, unix.LANDLOCK_RULE_PATH_BENEATH, uintptr(unsafe.Pointer(&beneath)), 0, 0, 0)
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

// applySeccomp installs a real filter for a policy already proven non-empty by
// measured applySeccompPolicy.
func applySeccomp(restrictions SandboxRestrictionSpec) error {
	auditArch, err := seccompAuditArch(runtime.GOARCH)
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
		offsetNR    = 0
		offsetArch  = 4
	)
	denyEPERM := uint32(unix.SECCOMP_RET_ERRNO | uint32(unix.EPERM))
	program := []unix.SockFilter{
		{Code: loadWord, K: offsetArch},
		{Code: jumpEqual, Jt: 1, K: auditArch},
		{Code: returnValue, K: uint32(unix.SECCOMP_RET_KILL_PROCESS)},
		{Code: loadWord, K: offsetNR},
	}
	for _, number := range numbers {
		program = append(program, unix.SockFilter{Code: jumpEqual, Jf: 1, K: number})
		program = append(program, unix.SockFilter{Code: returnValue, K: denyEPERM})
	}
	program = append(program, unix.SockFilter{Code: returnValue, K: uint32(unix.SECCOMP_RET_ALLOW)})
	fprog := unix.SockFprog{Len: uint16(len(program)), Filter: &program[0]}
	if _, _, errno := unix.Syscall(unix.SYS_SECCOMP, uintptr(unix.SECCOMP_SET_MODE_FILTER), 0, uintptr(unsafe.Pointer(&fprog))); errno != 0 {
		return fmt.Errorf("seccomp(SECCOMP_SET_MODE_FILTER): %w (PR_SET_NO_NEW_PRIVS must precede this)", errno)
	}
	return nil
}
