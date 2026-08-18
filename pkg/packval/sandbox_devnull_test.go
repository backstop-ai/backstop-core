package packval

import (
	"testing"
)

// ISSUE-168's derivation pins: the /dev/null write grant, and — just as important —
// the proof that it is the ONLY write anywhere in the derived spec.
//
// NO BUILD TAG, on purpose. DeriveSandboxRestrictions is a pure function of the
// capability struct, so this is the one part of the LINUX fix that is genuinely
// falsifiable on the darwin machine where it is written. What these tests prove is
// the ARITHMETIC of the derivation — the mask, its narrowing, its uniqueness. They
// prove NOTHING about the kernel's verdict on the resulting rule; that is confirmed
// only on a Linux runner (see this file's neighbours in sandbox_linux_exec_test.go,
// which do not compile here).
//
// WHY THE GRANT EXISTS AT ALL. `command -v foo >/dev/null 2>&1` is a universal shell
// idiom, and a pack-supplied convert or validator script has every right to use it.
// Landlock enforced the profile's stated "no writes" intent literally, so the
// redirect was refused and the shell reported `cannot create /dev/null: Permission
// denied`, exit 127 — which reads as a broken converter rather than as a sandbox
// decision. /dev/null is a write-only sink: nothing written to it persists, leaks or
// can be corrupted, so granting write on it and NOTHING else costs the confinement
// nothing.

// devNullRule returns the single derived /dev/null rule, failing loudly when the
// derivation emits none or more than one.
//
// The COUNT is asserted here rather than in one caller because a duplicate is a
// specific, silent defect: it means someone also put /dev/null into ReadablePaths or
// WritablePaths, which would emit a SECOND rule for the same inode carrying a mask
// this lane deliberately refuses. A findPathRule-style first-match lookup cannot see
// that, so it is checked once, centrally, for every caller.
func devNullRule(t *testing.T, spec SandboxRestrictionSpec) SandboxPathRule {
	t.Helper()
	var found []SandboxPathRule
	for _, rule := range spec.PathRules {
		if rule.Path == "/dev/null" {
			found = append(found, rule)
		}
	}
	switch len(found) {
	case 0:
		t.Fatalf("the derived spec carries NO rule for /dev/null, so `command -v foo >/dev/null 2>&1` "+
			"is refused by Landlock and the script dies reporting `cannot create /dev/null: Permission "+
			"denied` (ISSUE-168); derived paths = %v", derivedPaths(spec))
	case 1:
		return found[0]
	default:
		t.Fatalf("the derived spec carries %d rules for /dev/null and there must be exactly ONE; a "+
			"duplicate means /dev/null was also added to ReadablePaths or WritablePaths, which emits a "+
			"second rule with a mask this grant deliberately refuses. rules = %#v", len(found), found)
	}
	return SandboxPathRule{}
}

// derivedPaths lists the derived rule paths, so a failure message names what WAS
// derived instead of only what was missing.
func derivedPaths(spec SandboxRestrictionSpec) []string {
	paths := make([]string, 0, len(spec.PathRules))
	for _, rule := range spec.PathRules {
		paths = append(paths, rule.Path)
	}
	return paths
}

// TestSandboxCapability_DevNullCarriesANarrowWriteGrant is the primary pin for the
// ISSUE-168 grant: exactly one rule, an EXACT mask, Required — and no write right
// anywhere else in the spec.
//
// The mask is asserted by EQUALITY rather than by containment on purpose. A
// containment check ("it has WRITE_FILE") passes just as happily on a mask that also
// carries MAKE_DIR, MAKE_REG and REMOVE_FILE — which is precisely the over-grant this
// lane refuses. The generic writable-path loop in DeriveSandboxRestrictions produces
// exactly that wider mask, and it is inert on a character device ONLY because the
// Linux host's inode-type narrowing strips it at install time. The DERIVED DATA would
// still claim those rights, and the derived data is the only artefact through which
// this mechanism can be reviewed off-Linux.
func TestSandboxCapability_DevNullCarriesANarrowWriteGrant(t *testing.T) {
	spec := DeriveSandboxRestrictions(convertCapabilityForTest(t, 7))
	rule := devNullRule(t, spec)

	want := AccessFSReadFile | AccessFSWriteFile | AccessFSTruncate | AccessFSIoctlDev
	if rule.AllowedAccess != want {
		t.Errorf("the /dev/null grant is 0x%x and must be EXACTLY 0x%x "+
			"(READ_FILE|WRITE_FILE|TRUNCATE|IOCTL_DEV); the difference is 0x%x. An equality mismatch here "+
			"is either a missing right (the redirect is still refused) or an over-grant (rights nobody "+
			"intends, claimed in the one artefact this mechanism can be reviewed through)",
			rule.AllowedAccess, want, rule.AllowedAccess^want)
	}

	if !rule.Required {
		t.Error("the /dev/null rule must be Required. A non-required rule is SKIPPED when the path is " +
			"absent, which silently reproduces ISSUE-168 on that host: the script's `>/dev/null` is " +
			"refused and it reports a broken converter. Required makes it refuse loudly, naming the path")
	}

	// The anti-widening half: /dev/null is the ONLY rule carrying a write right.
	writeFamily := map[string]uint64{
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
		"TRUNCATE":    AccessFSTruncate,
	}
	for _, other := range spec.PathRules {
		if other.Path == "/dev/null" {
			continue
		}
		for name, right := range writeFamily {
			if other.AllowedAccess&right != 0 {
				t.Errorf("path rule %q grants write right %s (0x%x); /dev/null is the ONLY path the "+
					"mechanism may grant a write on, because it is the only one where a write cannot "+
					"persist, leak or corrupt anything. Full mask on that rule: 0x%x",
					other.Path, name, right, other.AllowedAccess)
			}
		}
	}
}

// TestSandboxCapability_DevNullGrantIsCharDeviceSafe is the test that prevents a
// WHOLE-INSTALL ABORT, and it is checked against the classification rather than
// eyeballed for exactly that reason.
//
// /dev/null is S_IFCHR, so applyLandlock's Fstat computes isDir == false for it and
// passes the rule's mask through narrowRuleToInodeType. landlock_add_rule REJECTS a
// path_beneath rule carrying a directory-only right against a non-directory, and ONE
// rejected rule aborts the ENTIRE restriction install — there is no partial
// application. This repository's recorded instance is CI run 30383453888: mask 0xc
// (READ_FILE|READ_DIR) against /etc/ld.so.cache, a regular file, returned EINVAL and
// took the whole sandbox down.
//
// So the property being pinned is INTENTIONAL, not incidental: every right in the
// grant is in Landlock's file-applicable class, which is what makes the narrowing a
// no-op and the kernel unable to EINVAL on it.
func TestSandboxCapability_DevNullGrantIsCharDeviceSafe(t *testing.T) {
	spec := DeriveSandboxRestrictions(convertCapabilityForTest(t, 7))
	rule := devNullRule(t, spec)

	// The REASON, stated first: no right in the grant belongs to the directory-only
	// class. This is the mechanism; the assertion below is its consequence.
	if overlap := rule.AllowedAccess & landlockDirectoryOnlyRights(); overlap != 0 {
		t.Errorf("the /dev/null grant (0x%x) carries directory-only right(s) 0x%x, and /dev/null is a "+
			"CHARACTER DEVICE. landlock_add_rule returns EINVAL for that combination, and one rejected "+
			"rule aborts the entire restriction install — the sandbox then confines nothing "+
			"(CI run 30383453888 is the recorded case)", rule.AllowedAccess, overlap)
	}

	if got := narrowRuleToInodeType(rule.AllowedAccess, false); got != rule.AllowedAccess {
		t.Errorf("non-directory narrowing changed the /dev/null grant from 0x%x to 0x%x (lost 0x%x). "+
			"The grant must survive intact, or the rule installed on the host is not the rule this "+
			"derivation describes and the redirect may still be refused",
			rule.AllowedAccess, got, rule.AllowedAccess&^got)
	}
}

// TestSandboxCapability_DevNullGrantNarrowsWithTheRunningABI asserts the grant is
// narrowed to the kernel's ABI like every other mask in the mechanism.
//
// This is a correctness requirement and not tidiness: landlock_create_ruleset returns
// EINVAL for a right the kernel does not recognise, so an un-narrowed grant does not
// degrade gracefully — it fails the whole ruleset creation. Ubuntu 22.04 ships kernel
// 5.15, which is ABI 1: it knows neither TRUNCATE (ABI 3) nor IOCTL_DEV (ABI 5).
// WRITE_FILE and READ_FILE are ABI 1 rights and must survive everywhere, because they
// are what the fix actually needs.
func TestSandboxCapability_DevNullGrantNarrowsWithTheRunningABI(t *testing.T) {
	for _, tc := range []struct {
		abi          int
		wantTruncate bool
		wantIoctlDev bool
	}{
		{abi: 1, wantTruncate: false, wantIoctlDev: false}, // kernel 5.15 / Ubuntu 22.04
		{abi: 3, wantTruncate: true, wantIoctlDev: false},
		{abi: 7, wantTruncate: true, wantIoctlDev: true}, // what ubuntu-latest reports
	} {
		spec := DeriveSandboxRestrictions(convertCapabilityForTest(t, tc.abi))
		rule := devNullRule(t, spec)

		if rule.AllowedAccess&AccessFSWriteFile == 0 {
			t.Errorf("ABI %d: the /dev/null grant (0x%x) lost WRITE_FILE (0x%x). WRITE_FILE is an ABI 1 "+
				"right and is the entire point of the grant — without it ISSUE-168 is unfixed on this kernel",
				tc.abi, rule.AllowedAccess, AccessFSWriteFile)
		}
		if rule.AllowedAccess&AccessFSReadFile == 0 {
			t.Errorf("ABI %d: the /dev/null grant (0x%x) lost READ_FILE (0x%x), an ABI 1 right; `</dev/null` "+
				"is the same idiom family and would be refused", tc.abi, rule.AllowedAccess, AccessFSReadFile)
		}
		if got := rule.AllowedAccess&AccessFSTruncate != 0; got != tc.wantTruncate {
			t.Errorf("ABI %d: TRUNCATE (0x%x, introduced at ABI 3) present=%v, want %v; grant=0x%x. "+
				"Carrying a right the kernel does not know makes landlock_create_ruleset EINVAL and the "+
				"whole ruleset fails", tc.abi, AccessFSTruncate, got, tc.wantTruncate, rule.AllowedAccess)
		}
		if got := rule.AllowedAccess&AccessFSIoctlDev != 0; got != tc.wantIoctlDev {
			t.Errorf("ABI %d: IOCTL_DEV (0x%x, introduced at ABI 5) present=%v, want %v; grant=0x%x",
				tc.abi, AccessFSIoctlDev, got, tc.wantIoctlDev, rule.AllowedAccess)
		}
	}
}

// TestSandboxCapability_DevNullGrantIsIndependentOfTheCapabilityFields pins the grant
// as a FLOOR of the mechanism rather than a property of one capability.
//
// The rationale for granting it — a sink that discards everything written to it
// cannot leak or persist anything — is universal, so it does not vary with what a
// caller happens to declare. That is also exact parity with the darwin side, where
// the `(allow file-write* (literal "/dev/null"))` clause is unconditional and is not
// derived from the capability struct at all.
//
// The empty-everything capability is the sharpest case: it has no packDir, no
// readable paths and no writable paths, so if the grant were being smuggled in
// through one of those lists this is where it would vanish.
func TestSandboxCapability_DevNullGrantIsIndependentOfTheCapabilityFields(t *testing.T) {
	networked := convertCapabilityForTest(t, 7)
	networked.Network = true

	for name, capability := range map[string]SandboxCapability{
		"convert/validator default": convertCapabilityForTest(t, 7),
		"network permitted":         networked,
		"empty capability": {
			ReadablePaths: nil,
			WritablePaths: nil,
			Network:       false,
			PackDir:       "",
			LandlockABI:   7,
		},
	} {
		spec := DeriveSandboxRestrictions(capability)
		rule := devNullRule(t, spec)

		want := AccessFSReadFile | AccessFSWriteFile | AccessFSTruncate | AccessFSIoctlDev
		if rule.AllowedAccess != want {
			t.Errorf("capability %q: the /dev/null grant is 0x%x, want 0x%x. The grant is a FLOOR of the "+
				"mechanism and must not vary with the capability's fields — a capability-dependent grant "+
				"means some sandboxed script somewhere still cannot redirect to /dev/null",
				name, rule.AllowedAccess, want)
		}
		if !rule.Required {
			t.Errorf("capability %q: the /dev/null rule is not Required; a missing /dev/null must refuse "+
				"loudly rather than silently reproducing ISSUE-168", name)
		}
	}
}
