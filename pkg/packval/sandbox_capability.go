package packval

import (
	"fmt"
	"path/filepath"
)

// Landlock filesystem access rights, declared here rather than taken from
// golang.org/x/sys/unix because those constants live in zerrors_linux.go and so do
// not exist on darwin. The derivation below must compile and be TESTABLE on the
// development machine, which is the only place this mechanism can be reviewed before
// it reaches a runner.
//
// The values are the kernel ABI's and are verified against
// golang.org/x/sys/unix@v0.31.0 by TestSandboxCapability_ConstantsMatchKernelABI
// under GOOS=linux — a drift here would be silent and total, since a wrong bit means
// a right other than the one intended.
//
// The trailing comment on each is the Landlock ABI level that introduced it.
const (
	AccessFSExecute    uint64 = 0x1    // ABI 1
	AccessFSWriteFile  uint64 = 0x2    // ABI 1
	AccessFSReadFile   uint64 = 0x4    // ABI 1
	AccessFSReadDir    uint64 = 0x8    // ABI 1
	AccessFSRemoveDir  uint64 = 0x10   // ABI 1
	AccessFSRemoveFile uint64 = 0x20   // ABI 1
	AccessFSMakeChar   uint64 = 0x40   // ABI 1
	AccessFSMakeDir    uint64 = 0x80   // ABI 1
	AccessFSMakeReg    uint64 = 0x100  // ABI 1
	AccessFSMakeSock   uint64 = 0x200  // ABI 1
	AccessFSMakeFifo   uint64 = 0x400  // ABI 1
	AccessFSMakeBlock  uint64 = 0x800  // ABI 1
	AccessFSMakeSym    uint64 = 0x1000 // ABI 1
	AccessFSRefer      uint64 = 0x2000 // ABI 2
	AccessFSTruncate   uint64 = 0x4000 // ABI 3
	AccessFSIoctlDev   uint64 = 0x8000 // ABI 5
)

// SandboxCapability is the INTERNAL description of what a sandboxed command may do.
//
// ─── BUNDLE-021 OQ-2 ────────────────────────────────────────────────────────────
// This struct is leg 2 of ISSUE-020's triple-tracked hedge, and its existence is a
// deliberate bet rather than an incidental refactor.
//
// BUNDLE-021 OQ-2 asks whether PACKS should DECLARE their behaviour — reads project
// files, writes files, needs network — instead of backstop matching on a pack name
// or a platform. That question is NOT RESOLVED HERE, and this struct does not
// resolve it. BUNDLE-021 owns it, together with its paired OQ-4 on consumer trust
// semantics: a capability struct is only as trustworthy as whoever gets to populate
// it, and today the only thing that populates this one is backstop itself.
//
// What this IS: an internal representation chosen so that a future OQ-2 resolution
// can slot a pack-declared value in without a rewrite. The alternative — a second
// hardcoded profile literal sitting beside darwinSandboxProfile — would have to be
// unwound instead of extended. Everything the Linux mechanism installs is DERIVED
// from this struct (see DeriveSandboxRestrictions), which is what makes the
// mechanism graduated-profile-capable: widening the boundary later is a policy
// change, not a rebuild.
// ────────────────────────────────────────────────────────────────────────────────
type SandboxCapability struct {
	// ReadablePaths may be read. Anything outside this set is denied.
	ReadablePaths []string
	// WritablePaths may be written. EMPTY for convert/validator work: darwin denies
	// file-write* outright and parity is the spec, so a convenience /tmp would be a
	// real divergence from the macOS trust model rather than an ergonomic detail.
	//
	// AN EMPTY SET STILL MEANS "THIS POLICY GRANTS NO WRITABLE PATH", and it stays
	// true after ISSUE-168. The one write the MECHANISM grants is the /dev/null sink,
	// emitted unconditionally by DeriveSandboxRestrictions for every capability and
	// matched one-for-one by darwin's `(allow file-write* (literal "/dev/null"))`. It
	// is deliberately NOT expressed here — see the rationale on the appended rule in
	// DeriveSandboxRestrictions — so do not "restore parity" by moving it into this
	// field or by deleting the rule.
	WritablePaths []string
	// Network reports whether the command may use the socket family at all.
	Network bool
	// PackDir is the pack directory, recorded so the derivation can mark its rule
	// Required. It is already symlink-resolved by ConvertValidatorCapability.
	PackDir string
	// LandlockABI is the ABI level ACHIEVED on the running kernel, not one this code
	// wishes for. The handled-access mask is narrowed to it, because
	// landlock_create_ruleset returns EINVAL for a right the kernel does not know —
	// an un-narrowed mask does not degrade gracefully, it fails outright.
	LandlockABI int
}

// SandboxPathRule is one derived Landlock path_beneath rule.
type SandboxPathRule struct {
	Path          string
	AllowedAccess uint64
	// Required marks a rule whose path MUST exist. A missing system library
	// directory is normal (distros disagree about /usr/lib64 vs /lib64) and is
	// skipped, but a missing packDir means the convert script is not reachable
	// inside the sandbox — skipping that silently would produce a sandbox that
	// denies the pack its own files and reports nothing.
	Required bool
}

// SandboxRestrictionSpec is the fully-derived description of what the trampoline
// installs. It is DATA, computed by a pure function, so the mechanism can be
// asserted on any platform — including the darwin machine where it is written.
type SandboxRestrictionSpec struct {
	// HandledAccessFS is the Landlock handled-access mask. Landlock restricts ONLY
	// what this mask covers, so a right absent here is FULLY PERMITTED regardless of
	// what the rules below grant. That asymmetry is the easiest way to ship a sandbox
	// that confines nothing.
	HandledAccessFS uint64
	// PathRules are the read/write grants, each scoped beneath one path.
	PathRules []SandboxPathRule
	// SeccompDenied names the syscalls the filter rejects with EPERM.
	SeccompDenied []string
	// SeccompDefaultAllow records that the program's default action is ALLOW.
	SeccompDefaultAllow bool
	// SeccompValidatesAuditArch records that the program checks the audit arch and
	// KILLs on mismatch.
	SeccompValidatesAuditArch bool
}

// linuxSystemReadPaths are the paths a dynamically linked interpreter must read to
// start at all: the loader, its cache, and the shared libraries themselves.
//
// THIS LIST IS THE LINUX FACE OF ISSUE-029. On macOS, an over-tight read allowlist
// meant dyld could not open the interpreter's dylibs, so EVERY convert script died
// at launch — and it presented as a broken converter rather than as a sandbox
// decision, which is what made it expensive to find. The Linux equivalent is a
// ld.so that cannot resolve libc. Narrow this list only with a real convert script
// running under it as evidence.
//
// It is a function rather than a package-level var because a package-level slice is
// mutable global state (go-standards no-global-mutable-state), and a caller could
// otherwise widen the sandbox by appending to it.
func linuxSystemReadPaths() []string {
	return []string{
		"/usr/lib",       // primary shared-library tree on merged-/usr systems
		"/usr/lib64",     // 64-bit tree on the distros that split it
		"/lib",           // legacy path, usually a symlink into /usr
		"/lib64",         // legacy 64-bit path, and where ld-linux lives
		"/usr/local/lib", // locally installed interpreters' libraries
		"/usr/bin",       // the interpreter binary itself
		"/bin",           // legacy binary path
		"/usr/share",     // interpreter runtime data (python stdlib, icu, ...)
		"/etc/ld.so.cache",
		"/etc/ld.so.conf.d",
	}
}

// ConvertValidatorCapability is the darwin-parity default for pack convert scripts
// and sandbox validators: read packDir plus the system paths an interpreter needs,
// write NOTHING, no network.
//
// packDir is symlink-RESOLVED for the same reason the darwin profile resolves it
// (sandbox.go): a Landlock path_beneath rule matches the KERNEL-resolved path, so an
// unresolved rule silently fails to match and denies legitimate reads inside packDir.
func ConvertValidatorCapability(packDir string, landlockABI int) SandboxCapability {
	resolved := packDir
	if r, err := filepath.EvalSymlinks(packDir); err == nil {
		resolved = r
	}
	return SandboxCapability{
		PackDir:       resolved,
		ReadablePaths: append([]string{resolved}, linuxSystemReadPaths()...),
		WritablePaths: nil,
		Network:       false,
		LandlockABI:   landlockABI,
	}
}

// seccompDeniedSyscalls is the network denial surface, and the claim rests on it
// rather than on Landlock. Two reasons, the second decisive:
//
//  1. Landlock's network rights are TCP-ONLY (LANDLOCK_ACCESS_NET_BIND_TCP and
//     _CONNECT_TCP are the entire surface). UDP, unix, netlink and raw sockets are
//     untouched, so DNS — which is UDP — would still resolve. A "denies network"
//     claim resting on that would be false in the most obvious case anyone tests.
//  2. Landlock gained those rights at ABI 4. Ubuntu 22.04 ships kernel 5.15 = ABI 1,
//     which has NO network rights at all, so a consumer there would get filesystem
//     confinement and SILENTLY no network denial. seccomp has been universally
//     available since ~3.5 and is ABI-independent.
//
// The set, and why each member is here:
//   - socket, socketpair — creating a socket at all.
//   - connect, bind — acting on one.
//   - sendto, sendmsg, sendmmsg — transmitting, including on an INHERITED fd, which
//     is why blocking creation alone is insufficient.
//   - recvfrom, recvmsg, recvmmsg — the RECEIVE half. Omitting it leaves an
//     inherited socket fully readable; exfiltration is not the only threat, and a
//     filter that blocks sending while permitting receiving is not a network denial.
//   - io_uring_setup, io_uring_enter — io_uring can submit every operation above
//     WITHOUT issuing the syscalls being filtered, so denying only the classic
//     entry points leaves the whole surface reachable through it.
//
// KNOWING EXCLUSIONS, recorded so a future reader tightening this under time
// pressure does not have to re-derive them:
//   - io_uring_register is ABSENT because it operates on a ring that
//     io_uring_setup must already have created; denying setup makes register
//     unreachable, and listing it would imply the ring can exist.
//   - accept, accept4, listen are ABSENT because they only act on a LISTENING
//     socket, which requires socket + bind — both denied above. They are
//     unreachable, and denying unreachable calls makes the surface look larger
//     than the reasoning behind it.
//
// It is a function rather than a package-level var for the same
// no-global-mutable-state reason as linuxSystemReadPaths, and here it matters more:
// a mutable denial list is a hole anything in the process could widen.
func seccompDeniedSyscalls() []string {
	return []string{
		"socket", "socketpair",
		"connect", "bind",
		"sendto", "sendmsg", "sendmmsg",
		"recvfrom", "recvmsg", "recvmmsg",
		"io_uring_setup", "io_uring_enter",
	}
}

// landlockWriteRights is the write family the handled mask must cover. Every one of
// these is a way to modify the filesystem, and an unhandled one is permitted.
func landlockWriteRights() uint64 {
	return AccessFSWriteFile |
		AccessFSRemoveDir | AccessFSRemoveFile |
		AccessFSMakeChar | AccessFSMakeDir | AccessFSMakeReg |
		AccessFSMakeSock | AccessFSMakeFifo | AccessFSMakeBlock | AccessFSMakeSym
}

// landlockReadRights is the read family. Handling reads is what makes a path OUTSIDE
// the granted set unreadable — without it the filesystem stays world-readable to the
// sandboxed process and the confinement is write-only.
func landlockReadRights() uint64 {
	return AccessFSReadFile | AccessFSReadDir
}

// sandboxDevNullPath is the null device — the ONE path this mechanism grants a write
// on, on both platforms.
//
// A const rather than a var or a slice: the neighbours above are functions
// specifically to avoid mutable package state a caller could widen (go-standards
// no-global-mutable-state), and for a single path an immutable const gets the same
// guarantee more directly.
const sandboxDevNullPath = "/dev/null"

// landlockDevNullRights is the exact grant for the null device, and every one of the
// four rights is here on purpose.
//
//   - WRITE_FILE — the point. `command -v foo >/dev/null 2>&1` is a universal shell
//     idiom that a pack-supplied convert or validator script has every right to use,
//     and without this right Landlock refuses the redirect: the shell reports
//     `cannot create /dev/null: Permission denied` and exits 127, which reads as a
//     broken converter rather than as a sandbox decision (ISSUE-168).
//   - TRUNCATE — required, not decoration. The shell's `>` opens with O_TRUNC, and
//     TRUNCATE is a right this mechanism HANDLES from ABI 3 onward. Granting
//     WRITE_FILE alone therefore risks the redirect still being refused on exactly
//     the modern kernels CI runs (ubuntu-latest reports ABI 4+). Whether the kernel
//     demands it for a character device is version-dependent; it costs nothing on a
//     sink that discards data, and omitting it risks shipping a fix that fixes
//     nothing.
//   - IOCTL_DEV — /dev/null is a CHARACTER DEVICE, and IOCTL_DEV is handled from
//     ABI 5. Programs routinely isatty()/TCGETS a redirected fd, and an un-granted
//     ioctl on an ABI-5 kernel returns EACCES. Inert on the null device, and it
//     removes a whole class of future ABI-5-only breakage.
//   - READ_FILE — `</dev/null`, feeding a command empty stdin, is the same idiom
//     family and is just as safe: reading the null device reads nothing.
//
// EXECUTE IS DELIBERATELY ABSENT. The mechanism leaves EXECUTE UNHANDLED (see
// DeriveSandboxRestrictions), so granting it back on any rule would be contradictory;
// TestSandboxCapability_SystemPathsKeepInterpreterReadable asserts no rule does.
//
// Every right here is in Landlock's file-applicable class, which is what makes the
// rule safe against a character device — see the appended rule in
// DeriveSandboxRestrictions for why that matters so much.
func landlockDevNullRights() uint64 {
	return AccessFSReadFile | AccessFSWriteFile | AccessFSTruncate | AccessFSIoctlDev
}

// landlockDirectoryOnlyRights is the class of access rights the kernel accepts ONLY on
// a directory. It exists as one named classification so that adding a right means
// classifying it in exactly one place, rather than open-coding a bitmask wherever the
// distinction happens to matter.
//
// Getting this wrong is not a soft failure. landlock_add_rule rejects a path_beneath
// rule whose allowed_access carries any of these against a non-directory, and ONE
// rejected rule aborts the ENTIRE restriction install — there is no partial
// application. CI run 30383453888 is the recorded case: mask 0xc
// (READ_FILE|READ_DIR) against /etc/ld.so.cache, a regular file, returned EINVAL and
// took the whole sandbox down.
//
// REFER is included where the ABI has it (2+): it governs moving files BETWEEN
// directories and is meaningless on a file. narrowToABI drops it on older kernels.
func landlockDirectoryOnlyRights() uint64 {
	return AccessFSReadDir |
		AccessFSRemoveDir | AccessFSRemoveFile |
		AccessFSMakeChar | AccessFSMakeDir | AccessFSMakeReg |
		AccessFSMakeSock | AccessFSMakeFifo | AccessFSMakeBlock | AccessFSMakeSym |
		AccessFSRefer
}

// narrowRuleToInodeType returns the rights valid for a path_beneath rule against an
// inode of the given kind: a directory keeps the mask intact, a non-directory loses
// every directory-only right and keeps every file-applicable one.
//
// isDir is a PARAMETER rather than something this function discovers, and that is a
// deliberate testability choice, not an inconvenience. The capability -> restriction
// -spec derivation is a pure function of the capability struct (CLM-012), which is the
// only reason it can be tested on darwin at all; a stat inside it would make the whole
// capability untestable off-Linux. Both defects this lane has found were correct as a
// spec and wrong against a real host, so keeping the ARITHMETIC portable — and letting
// the caller supply the one fact only a Linux host knows — is what keeps them
// catchable here.
//
// Narrow only path RULES. The ruleset's handled_access_fs mask is NOT subject to the
// kernel's inode-type check, so narrowing it would fix nothing and would silently
// un-restrict those operations everywhere.
func narrowRuleToInodeType(mask uint64, isDir bool) uint64 {
	if isDir {
		return mask
	}
	return mask &^ landlockDirectoryOnlyRights()
}

// narrowToABI masks rights down to what the running kernel knows.
//
// This is a correctness requirement, not tidiness: landlock_create_ruleset returns
// EINVAL when the mask carries an unrecognised right, so an un-narrowed mask fails
// the ruleset creation outright on an older kernel rather than degrading.
func narrowToABI(rights uint64, abi int) uint64 {
	if abi < 5 {
		rights &^= AccessFSIoctlDev // ABI 5
	}
	if abi < 3 {
		rights &^= AccessFSTruncate // ABI 3
	}
	if abi < 2 {
		rights &^= AccessFSRefer // ABI 2
	}
	return rights
}

// DeriveSandboxRestrictions computes the full restriction spec from a capability.
// It is PURE — no syscalls, no host lookups — which is what lets the mechanism be
// asserted on darwin and regression-locked away from a Linux runner.
func DeriveSandboxRestrictions(capability SandboxCapability) SandboxRestrictionSpec {
	// EXECUTE is deliberately NOT handled. Landlock restricts only what the mask
	// covers, so leaving EXECUTE out is precisely how darwin's `(allow process*)`
	// (sandbox.go) translates: the convert step's whole job is to RUN an interpreter,
	// and handling EXECUTE would confine the very thing being invoked. Adding it here
	// without also granting it on every interpreter path would break every convert
	// script — the ISSUE-029 failure mode, in a different right.
	handled := narrowToABI(
		landlockReadRights()|landlockWriteRights()|AccessFSRefer|AccessFSTruncate|AccessFSIoctlDev,
		capability.LandlockABI,
	)

	// +1 for the unconditional /dev/null rule appended after both loops.
	rules := make([]SandboxPathRule, 0, len(capability.ReadablePaths)+len(capability.WritablePaths)+1)
	for _, path := range capability.ReadablePaths {
		rules = append(rules, SandboxPathRule{
			Path:          path,
			AllowedAccess: narrowToABI(landlockReadRights(), capability.LandlockABI),
			Required:      path == capability.PackDir && capability.PackDir != "",
		})
	}
	for _, path := range capability.WritablePaths {
		rules = append(rules, SandboxPathRule{
			Path:     path,
			Required: true,
			AllowedAccess: narrowToABI(
				landlockReadRights()|landlockWriteRights()|AccessFSTruncate,
				capability.LandlockABI,
			),
		})
	}

	// THE NULL-DEVICE SINK (ISSUE-168), granted for EVERY capability.
	//
	// WHY IT IS UNCONDITIONAL. The rationale is universal rather than a property of any
	// one capability: /dev/null discards everything written to it, so nothing persists,
	// nothing leaks and nothing can be corrupted. That makes the grant a FLOOR of the
	// mechanism, which is also exact parity with darwin — whose profile literal carries
	// `(allow file-write* (literal "/dev/null"))` unconditionally and does not derive it
	// from a capability struct at all.
	//
	// WHY NOT `WritablePaths: []string{"/dev/null"}`, THE OBVIOUS ALTERNATIVE. The
	// writable loop above grants landlockReadRights|landlockWriteRights|TRUNCATE — i.e.
	// MAKE_REG, MAKE_DIR, MAKE_SYM, REMOVE_FILE, READ_DIR and the rest. Those are inert
	// on a character device ONLY because narrowRuleToInodeType strips them at install
	// time on the Linux host. The DERIVED DATA would still claim them, and the derived
	// data is the only artefact through which this mechanism can be reviewed off-Linux.
	// A spec asserting "you may create directories under /dev/null" is a lie in exactly
	// the place the reviewing happens.
	//
	// CHARACTER-DEVICE SAFETY IS WHY THE MASK IS EXACT. landlock_add_rule REJECTS a
	// path_beneath rule carrying a directory-only right against a non-directory, and ONE
	// rejected rule aborts the ENTIRE restriction install — CI run 30383453888 is this
	// repo's recorded case. Every right in landlockDevNullRights is file-applicable, so
	// narrowRuleToInodeType(mask, false) returns it unchanged and the kernel cannot
	// EINVAL on it. TestSandboxCapability_DevNullGrantIsCharDeviceSafe pins that as an
	// intended property rather than an accident.
	//
	// Required: true is the loud-over-silent choice. A host with no /dev/null refuses by
	// name instead of silently reproducing ISSUE-168 there.
	//
	// ORDERING CONTRAST, SO NOBODY MIRRORS THE WRONG RULE ACROSS PLATFORMS: Landlock
	// rules UNION and have no precedence — there is no deny rule and no last-match-wins,
	// so this rule's position in the slice is irrelevant to the kernel and it is
	// appended last only for readability. On darwin the opposite holds: the Seatbelt
	// clause's position AFTER `(deny file-write*)` is load-bearing.
	rules = append(rules, SandboxPathRule{
		Path:          sandboxDevNullPath,
		AllowedAccess: narrowToABI(landlockDevNullRights(), capability.LandlockABI),
		Required:      true,
	})

	denied := []string{}
	if !capability.Network {
		denied = seccompDeniedSyscalls()
	}

	return SandboxRestrictionSpec{
		HandledAccessFS: handled,
		PathRules:       rules,
		SeccompDenied:   denied,
		// A TARGETED DENY with a default of ALLOW. Between filter install and execve
		// the Go runtime still makes syscalls, so a default-deny allowlist kills the
		// helper before it can exec — and enumerating permitted syscalls is not what
		// darwin does either: it denies network, it does not allowlist.
		SeccompDefaultAllow: true,
		// The audit-arch check is the standard guard against an i386/x32 entry point
		// reaching the same kernel with different syscall numbers, which would make
		// every rule above match the wrong call.
		SeccompValidatesAuditArch: true,
	}
}

// LandlockABIProbe reports the Landlock ABI level and the running kernel release.
// It is injected so the refusal path can be proven without depending on the
// developer's kernel — a test that passes only on the right host proves nothing on
// the wrong one.
type LandlockABIProbe func() (abi int, kernelRelease string, err error)

// resolveLandlockMechanism negotiates the ABI explicitly and REFUSES when Landlock
// is unavailable.
//
// ABI >= 1 is a fully functional configuration for this plan's parity claim, because
// network denial rests on seccomp rather than on Landlock's ABI-4 network rights. So
// the only thing that degrades on an older kernel is filesystem granularity — never
// the network guarantee.
//
// ABI 0 / ENOSYS / Landlock absent from the LSM list is a REFUSAL, not a best-effort
// downgrade. ISSUE-020's acceptance item 4 is explicit: an unavailable mechanism
// fails loudly, because a silent pass-through to an unsandboxed exec IS the defect
// being fixed.
func resolveLandlockMechanism(probe LandlockABIProbe) (int, error) {
	abi, kernelRelease, err := probe()
	if err != nil {
		return 0, fmt.Errorf("kernel %s does not provide Landlock (%v); backstop refuses to run "+
			"pack-supplied code unsandboxed — see ISSUE-020", kernelRelease, err)
	}
	if abi < 1 {
		return 0, fmt.Errorf("kernel %s reports Landlock ABI %d, so no filesystem confinement is "+
			"available; backstop refuses to run pack-supplied code unsandboxed — see ISSUE-020",
			kernelRelease, abi)
	}
	return abi, nil
}

// sandboxPlatformSupported reports whether a GOOS has a real sandbox implementation.
//
// It exists as a named predicate rather than living inline in the dispatch switch so
// the refusal is testable: the point of the default branch is that a THIRD platform
// does not silently inherit the Linux path the moment someone adds a case above it,
// and an untestable default branch is how that regression ships.
func sandboxPlatformSupported(goos string) error {
	switch goos {
	case "darwin", "linux":
		return nil
	default:
		return fmt.Errorf("sandbox unsupported platform: %s", goos)
	}
}
