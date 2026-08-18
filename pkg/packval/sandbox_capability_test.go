package packval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests cover the PURE derivation of the Linux restriction spec from the
// capability struct. They carry NO build tag on purpose: the derivation is a pure
// function, so it is reviewable and regression-locked on the machine where
// development happens — and it is the only part of the mechanism that CAN be proven
// there. The execution half lives in sandbox_linux_exec_test.go and is proven in CI.
//
// MECHANISM PROVENANCE: this is Landlock + seccomp, NOT bubblewrap. The record is
// pkg/packval/testdata/ubuntu-runner-probe.txt — on stock ubuntu-24.04 the probe
// measured apparmor_restrict_unprivileged_userns=1 and SEVEN failing bwrap
// invocations INCLUDING BOTH POSITIVE CONTROLS, while Landlock came back live at
// ABI 7. Read that file's "READ THIS FIRST" section, particularly point 3, before
// writing any assertion about denial: it is this repository's recorded instance of
// non-zero exits that meant "the mechanism never started" rather than "access was
// denied". A denial assertion without a positive control proves nothing.

// convertCapabilityForTest builds the convert/validator capability at a given ABI.
func convertCapabilityForTest(t *testing.T, abi int) SandboxCapability {
	t.Helper()
	return ConvertValidatorCapability("/tmp/packdir", abi)
}

// findPathRule returns the derived rule for path, and whether one exists.
func findPathRule(spec SandboxRestrictionSpec, path string) (SandboxPathRule, bool) {
	for _, rule := range spec.PathRules {
		if rule.Path == path {
			return rule, true
		}
	}
	return SandboxPathRule{}, false
}

func containsSyscall(denied []string, name string) bool {
	for _, syscallName := range denied {
		if syscallName == name {
			return true
		}
	}
	return false
}

// TestSandboxCapability_HandlesWriteRightsAndGrantsNone is the single most
// important test in this file.
//
// Landlock restricts ONLY what the handled-access mask covers. A mask carrying just
// the read rights leaves every write FULLY PERMITTED while the ruleset installs
// cleanly, the syscalls all return 0, and the sandbox looks correct from every angle
// except the one that matters. So: the write family must be HANDLED, and NO
// path_beneath rule may grant any of it. (CLM-011, CLM-034)
//
// ─── THE ONE EXEMPTION, AND WHY IT DOES NOT SOFTEN THIS TEST (ISSUE-168) ────────
// The write family is still HANDLED in full, and it is still granted NOWHERE except
// the null device. /dev/null is a write-only sink: nothing written to it persists,
// leaks or can be corrupted, so a write right on that one inode does not weaken the
// confinement this test exists to protect. It is exempted BY PATH, precisely — never
// by loosening the assertion, which would stop this test noticing a genuine grant on
// a real file.
//
// The exemption's SHAPE is deliberately not checked here, where nothing would notice
// it widening. TestSandboxCapability_DevNullCarriesANarrowWriteGrant
// (sandbox_devnull_test.go) asserts the exact mask by EQUALITY, that the rule is
// unique, and that no OTHER rule carries a write right. Widen the grant and that test
// fails; this one would not.
func TestSandboxCapability_HandlesWriteRightsAndGrantsNone(t *testing.T) {
	spec := DeriveSandboxRestrictions(convertCapabilityForTest(t, 7))

	writeRights := map[string]uint64{
		"WRITE_FILE":  AccessFSWriteFile,
		"MAKE_REG":    AccessFSMakeReg,
		"MAKE_DIR":    AccessFSMakeDir,
		"MAKE_SYM":    AccessFSMakeSym,
		"MAKE_SOCK":   AccessFSMakeSock,
		"MAKE_FIFO":   AccessFSMakeFifo,
		"MAKE_BLOCK":  AccessFSMakeBlock,
		"MAKE_CHAR":   AccessFSMakeChar,
		"REMOVE_FILE": AccessFSRemoveFile,
		"REMOVE_DIR":  AccessFSRemoveDir,
		"TRUNCATE":    AccessFSTruncate, // ABI 3+, and we derived at 7
	}

	for name, right := range writeRights {
		if spec.HandledAccessFS&right == 0 {
			t.Errorf("handled_access_fs must HANDLE %s (0x%x) — an unhandled right is fully permitted, "+
				"which is how a sandbox ships confining nothing", name, right)
		}
	}

	// And nothing may GRANT them back — except the /dev/null sink (see this test's
	// docstring). Exempted by exact path, so any other rule acquiring a write right
	// still fails here.
	for _, rule := range spec.PathRules {
		if rule.Path == "/dev/null" {
			continue
		}
		for name, right := range writeRights {
			if rule.AllowedAccess&right != 0 {
				t.Errorf("path rule %q grants write right %s (0x%x); the convert/validator capability "+
					"has an EMPTY writable set — darwin denies file-write* outright and parity is the spec",
					rule.Path, name, right)
			}
		}
	}
}

// TestSandboxCapability_ReadRulesCoverPackDirOnly asserts the derived read rules
// cover packDir and the declared system paths and NOTHING else — no project root,
// no parent of packDir. (CLM-011)
func TestSandboxCapability_ReadRulesCoverPackDirOnly(t *testing.T) {
	const packDir = "/home/runner/work/project/.backstop/packs/acme/pack"
	spec := DeriveSandboxRestrictions(ConvertValidatorCapability(packDir, 7))

	if _, ok := findPathRule(spec, packDir); !ok {
		t.Fatalf("the pack directory %q must carry a read rule, or the convert script does not exist "+
			"inside the sandbox; rules=%#v", packDir, spec.PathRules)
	}

	// Every ancestor of packDir, and the project root, must be absent. Granting a
	// parent would silently grant the whole project tree beneath it.
	forbidden := []string{
		"/home/runner/work/project",
		"/home/runner/work/project/.backstop",
		"/home/runner/work/project/.backstop/packs",
		"/home/runner/work/project/.backstop/packs/acme",
		"/home/runner/work",
		"/home/runner",
		"/",
	}
	for _, ancestor := range forbidden {
		if _, ok := findPathRule(spec, ancestor); ok {
			t.Errorf("a read rule covers %q, an ancestor of packDir — that grants the whole tree "+
				"beneath it, including the project sources the sandbox exists to protect", ancestor)
		}
	}

	// Restated structurally: every rule is packDir itself or a system path, never a
	// project path outside packDir.
	for _, rule := range spec.PathRules {
		if rule.Path == packDir {
			continue
		}
		if strings.HasPrefix(rule.Path, "/home/") {
			t.Errorf("read rule %q is a project path outside packDir", rule.Path)
		}
	}
}

// TestSandboxCapability_SeccompDeniesSocketFamily pins the denial surface CLM-030
// rests on.
//
// The RECEIVE half and the io_uring pair are asserted BY NAME because they are the
// two holes the first draft of this surface had: a filter denying only socket/
// connect/send leaves an inherited fd fully readable, and io_uring lets a program
// submit those same operations without ever issuing the syscalls being filtered. A
// test checking only the send calls would have passed over both.
//
// The default-allow shape is asserted too, and it is not a style preference: between
// filter install and execve the Go runtime still makes syscalls, so a default-deny
// program kills the helper before it can exec. It is also the correct parity call —
// darwin denies network, it does not enumerate permitted syscalls. (CLM-030)
func TestSandboxCapability_SeccompDeniesSocketFamily(t *testing.T) {
	spec := DeriveSandboxRestrictions(convertCapabilityForTest(t, 7))

	// The full denied set, verbatim.
	for _, name := range []string{
		"socket", "socketpair",
		"connect", "bind",
		"sendto", "sendmsg", "sendmmsg",
		"recvfrom", "recvmsg", "recvmmsg",
		"io_uring_setup", "io_uring_enter",
	} {
		if !containsSyscall(spec.SeccompDenied, name) {
			t.Errorf("the seccomp filter must deny %q; denied set = %v", name, spec.SeccompDenied)
		}
	}

	if !spec.SeccompDefaultAllow {
		t.Error("the seccomp program must be a TARGETED DENY with a default-ALLOW action: " +
			"the Go runtime makes syscalls between filter install and execve, so a default-deny " +
			"program kills the helper before it can exec")
	}

	if !spec.SeccompValidatesAuditArch {
		t.Error("the seccomp program must validate the audit arch and KILL on mismatch — " +
			"without it an i386/x32 entry point bypasses the filter entirely")
	}
}

// TestSandboxCapability_SystemPathsKeepInterpreterReadable is the Linux regression
// lock for the ISSUE-029 defect class, and it lives HERE rather than in TASK-013
// precisely so it runs on the development machine instead of waiting for a runner.
//
// It is the twin of darwin's TestSandboxProfileAllowsDyldLibraries. On macOS,
// over-confining interpreter and loader reads broke EVERY convert script, and it
// presented as a broken converter rather than as a sandbox decision — which is what
// made it expensive to diagnose. The Linux face of that bug is a dynamic loader that
// cannot open the interpreter's shared libraries. (CLM-034)
func TestSandboxCapability_SystemPathsKeepInterpreterReadable(t *testing.T) {
	spec := DeriveSandboxRestrictions(convertCapabilityForTest(t, 7))

	systemRules := 0
	for _, rule := range spec.PathRules {
		if strings.HasPrefix(rule.Path, "/tmp/packdir") {
			continue
		}
		// The /dev/null grant (ISSUE-168) is not a system READ path and this loop's
		// requirements do not apply to it: it is a CHARACTER DEVICE, and READ_DIR is a
		// directory-only right that the kernel rejects against a non-directory — one
		// rejected rule aborts the entire restriction install. Its absence there is
		// deliberate, and the rule's real shape is asserted in sandbox_devnull_test.go.
		if rule.Path == "/dev/null" {
			continue
		}
		systemRules++
		if rule.AllowedAccess&AccessFSReadFile == 0 {
			t.Errorf("system path %q lacks READ_FILE — the dynamic loader cannot open the "+
				"interpreter's shared libraries and every convert script dies at load time", rule.Path)
		}
		if rule.AllowedAccess&AccessFSReadDir == 0 {
			t.Errorf("system path %q lacks READ_DIR — the loader enumerates library directories "+
				"during resolution", rule.Path)
		}
	}
	if systemRules == 0 {
		t.Fatal("no system library paths are readable at all; a dynamically linked interpreter " +
			"cannot start, and the assertions above would hold vacuously")
	}

	// EXECUTE must be absent from the HANDLED MASK, not merely permitted in practice.
	// Landlock restricts only what the mask handles, so leaving EXECUTE unhandled is
	// exactly how darwin's `(allow process*)` (sandbox.go:57) translates. Asserting
	// absence-from-the-mask rather than "execution works" is deliberate: they are
	// different facts, and only the first survives someone later "tightening" the
	// mask by adding EXECUTE and granting it back on packDir.
	if spec.HandledAccessFS&AccessFSExecute != 0 {
		t.Error("LANDLOCK_ACCESS_FS_EXECUTE must NOT appear in the handled mask — darwin allows " +
			"process* and parity is the spec; handling EXECUTE would confine the very interpreter " +
			"the convert step runs")
	}
	for _, rule := range spec.PathRules {
		if rule.AllowedAccess&AccessFSExecute != 0 {
			t.Errorf("path rule %q grants EXECUTE; the right is deliberately unhandled, so granting "+
				"it back is contradictory and signals the mask was tightened without updating this", rule.Path)
		}
	}
}

// TestSandboxCapability_NetworkAllowedDropsSocketDenial is half the
// graduated-profile proof: the spec is DERIVED from the capability struct, so
// flipping Network changes the derived filter. If this fails while the others pass,
// the restrictions are hardcoded and the struct is decorative. (CLM-012)
func TestSandboxCapability_NetworkAllowedDropsSocketDenial(t *testing.T) {
	denied := convertCapabilityForTest(t, 7)
	allowed := denied
	allowed.Network = true

	deniedSpec := DeriveSandboxRestrictions(denied)
	allowedSpec := DeriveSandboxRestrictions(allowed)

	if !containsSyscall(deniedSpec.SeccompDenied, "socket") {
		t.Fatal("PRECONDITION FAILED: the default capability must deny socket, or the contrast below is meaningless")
	}
	if containsSyscall(allowedSpec.SeccompDenied, "socket") {
		t.Errorf("a network-ALLOWED capability must not deny socket; denied set = %v", allowedSpec.SeccompDenied)
	}
	if len(allowedSpec.SeccompDenied) != 0 {
		t.Errorf("a network-allowed capability denies nothing at the socket layer; denied set = %v",
			allowedSpec.SeccompDenied)
	}
}

// TestSandboxCapability_AddedReadablePathAddsPathBeneathRule is the other half of
// the graduated-profile proof: adding a readable path adds a rule. (CLM-012)
func TestSandboxCapability_AddedReadablePathAddsPathBeneathRule(t *testing.T) {
	base := convertCapabilityForTest(t, 7)
	baseSpec := DeriveSandboxRestrictions(base)

	const extra = "/opt/extra-readable"
	if _, ok := findPathRule(baseSpec, extra); ok {
		t.Fatalf("PRECONDITION FAILED: %q is already readable, so adding it proves nothing", extra)
	}

	widened := base
	widened.ReadablePaths = append(append([]string{}, base.ReadablePaths...), extra)
	widenedSpec := DeriveSandboxRestrictions(widened)

	rule, ok := findPathRule(widenedSpec, extra)
	if !ok {
		t.Fatalf("adding %q to the capability's readable set must add a path_beneath rule; rules=%#v",
			extra, widenedSpec.PathRules)
	}
	if rule.AllowedAccess&AccessFSReadFile == 0 {
		t.Errorf("the added path rule must grant READ_FILE, got access 0x%x", rule.AllowedAccess)
	}
	if len(widenedSpec.PathRules) != len(baseSpec.PathRules)+1 {
		t.Errorf("adding one readable path must add exactly one rule; before=%d after=%d",
			len(baseSpec.PathRules), len(widenedSpec.PathRules))
	}
}

// TestSandboxCapability_AbiMaskNarrowsToRunningKernel asserts the handled mask is
// narrowed to what the RUNNING ABI knows.
//
// This is a correctness requirement, not tidiness: landlock_create_ruleset returns
// EINVAL when the mask carries a right the kernel does not recognise, so an
// un-narrowed mask does not "degrade gracefully" — it fails outright on Ubuntu 22.04
// (kernel 5.15 = ABI 1). It is also how CLM-031's explicit negotiation is proven
// without a Linux host. (CLM-031)
func TestSandboxCapability_AbiMaskNarrowsToRunningKernel(t *testing.T) {
	abi1 := DeriveSandboxRestrictions(convertCapabilityForTest(t, 1))
	abi7 := DeriveSandboxRestrictions(convertCapabilityForTest(t, 7))

	// ABI 1 knows none of these.
	for name, right := range map[string]uint64{
		"REFER":     AccessFSRefer,    // ABI 2
		"TRUNCATE":  AccessFSTruncate, // ABI 3
		"IOCTL_DEV": AccessFSIoctlDev, // ABI 5
	} {
		if abi1.HandledAccessFS&right != 0 {
			t.Errorf("ABI 1 mask must not carry %s (0x%x) — the kernel returns EINVAL for an "+
				"unrecognised right, so the ruleset would fail to create at all", name, right)
		}
		if abi7.HandledAccessFS&right == 0 {
			t.Errorf("ABI 7 mask must carry %s (0x%x); narrowing must not discard rights the "+
				"running kernel does support", name, right)
		}
	}

	// The write family ABI 1 DOES know must survive narrowing, or an old kernel gets
	// a sandbox that confines nothing rather than one that confines a little less.
	for name, right := range map[string]uint64{
		"WRITE_FILE": AccessFSWriteFile,
		"MAKE_REG":   AccessFSMakeReg,
		"MAKE_DIR":   AccessFSMakeDir,
	} {
		if abi1.HandledAccessFS&right == 0 {
			t.Errorf("ABI 1 mask must still handle %s (0x%x)", name, right)
		}
	}

	// Network denial is seccomp's job, so it must be IDENTICAL at both ABIs. This is
	// the payoff of that decision: on ABI 1, which has no Landlock net rights at all,
	// the network guarantee is unchanged and only filesystem granularity degrades.
	if strings.Join(abi1.SeccompDenied, ",") != strings.Join(abi7.SeccompDenied, ",") {
		t.Errorf("the seccomp denial surface must not vary with the Landlock ABI — network denial "+
			"rests on seccomp precisely so it does not degrade on an older kernel;\n  abi1=%v\n  abi7=%v",
			abi1.SeccompDenied, abi7.SeccompDenied)
	}
}

// TestSandboxUnavailableMechanismIsLoudError asserts that when Landlock is
// unavailable the mechanism REFUSES loudly and runs nothing.
//
// The probe is INJECTED rather than read from the host: depending on the developer's
// kernel would make this test pass or fail for reasons unrelated to the code. ABI 0 /
// ENOSYS is the LinuxKit case the 2026-07-27 spike hit. ISSUE-020's acceptance item 4
// is explicit that an unavailable mechanism fails loudly, because a silent
// pass-through to an unsandboxed exec IS the defect being fixed. (CLM-015)
func TestSandboxUnavailableMechanismIsLoudError(t *testing.T) {
	const kernel = "6.10.14-linuxkit"
	probe := func() (int, string, error) { return 0, kernel, nil }

	abi, err := resolveLandlockMechanism(probe)
	if err == nil {
		t.Fatalf("an ABI of 0 must be a hard refusal, got abi=%d and no error", abi)
	}

	// The message has to be actionable on a machine nobody is sitting at.
	for label, want := range map[string]string{
		"the mechanism": "andlock", // matches Landlock/landlock without pinning case
		"the kernel":    kernel,
		"the issue":     "ISSUE-020",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must name %s (%q); got %q", label, want, err)
		}
	}
}

// TestSandboxUnsupportedPlatformStillErrors asserts the default branch keeps
// refusing, so a third platform does not silently inherit the Linux path the moment
// someone adds a GOOS case above it.
func TestSandboxUnsupportedPlatformStillErrors(t *testing.T) {
	for _, goos := range []string{"windows", "plan9", "js"} {
		if err := sandboxPlatformSupported(goos); err == nil {
			t.Errorf("platform %q must be refused; an unsupported platform silently inheriting a "+
				"sandbox path is how an unsandboxed exec ships", goos)
		} else if !strings.Contains(err.Error(), goos) {
			t.Errorf("the refusal for %q must name the platform; got %q", goos, err)
		}
	}
	for _, goos := range []string{"darwin", "linux"} {
		if err := sandboxPlatformSupported(goos); err != nil {
			t.Errorf("platform %q is supported and must not be refused; got %q", goos, err)
		}
	}
}

// TestSandboxCapability_PackDirIsSymlinkResolved mirrors the darwin profile's
// EvalSymlinks step (sandbox.go:38). A Landlock path_beneath rule is matched against
// the KERNEL-resolved path, so an unresolved rule silently fails to match and denies
// legitimate reads inside packDir — the same trap ISSUE-029 hit on macOS with
// /var vs /private/var.
func TestSandboxCapability_PackDirIsSymlinkResolved(t *testing.T) {
	realDir := t.TempDir()
	linkDir := filepath.Join(t.TempDir(), "link-to-pack")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("create symlink %s -> %s: %v", linkDir, realDir, err)
	}

	spec := DeriveSandboxRestrictions(ConvertValidatorCapability(linkDir, 7))

	resolved, err := filepath.EvalSymlinks(realDir)
	if err != nil {
		t.Fatalf("resolve %s: %v", realDir, err)
	}
	if _, ok := findPathRule(spec, resolved); !ok {
		t.Errorf("the packDir rule must use the symlink-RESOLVED path %q, or the kernel rule does "+
			"not match and reads inside packDir are denied; rules=%#v", resolved, spec.PathRules)
	}
}
