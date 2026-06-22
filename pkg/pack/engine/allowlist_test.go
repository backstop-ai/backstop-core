package engine

import (
	"strings"
	"testing"
)

// realAllowlist returns a genuine (non stub-open) {tool -> pinned version}
// allowlist for the matrix tests: it contains a present-and-pinned tool and
// deliberately OMITS "acme-absent" so the un-allowlisted cell is real, not a
// stub-open hole (Sharp Edge 3 — an empty/open allowlist proves nothing).
func realAllowlist() map[string]string {
	return map[string]string{
		"acme-scan": "2.3.1",
		"acme-lint": "1.4.0",
	}
}

// TestAllowlist_AllowlistedPinnedToolRuns asserts CheckToolAllowed returns nil
// when the tool is on the allowlist AND the caller-supplied lockedVersion matches
// the allowlist's pinned version — the gate-passes half of the trust floor.
// CLM-005.
func TestAllowlist_AllowlistedPinnedToolRuns(t *testing.T) {
	al := realAllowlist()
	if err := CheckToolAllowed(al, "acme-scan", "2.3.1"); err != nil {
		t.Fatalf("allowlisted + lock-pinned tool must pass the gate, got: %v", err)
	}
}

// TestAllowlist_UnallowlistedToolFailsLoud asserts a tool ABSENT from the
// allowlist yields a non-nil error naming the tool — the command may never run.
// CLM-006.
func TestAllowlist_UnallowlistedToolFailsLoud(t *testing.T) {
	al := realAllowlist()
	// Guard the fixture: the tool must genuinely be absent so this is a real
	// un-allowlisted cell, not a stub-open hole.
	if _, present := al["acme-absent"]; present {
		t.Fatal("fixture invariant broken: acme-absent must be ABSENT from the allowlist")
	}
	err := CheckToolAllowed(al, "acme-absent", "9.9.9")
	if err == nil {
		t.Fatal("un-allowlisted tool must fail loud, got nil")
	}
	if !strings.Contains(err.Error(), "acme-absent") {
		t.Errorf("error must name the un-allowlisted tool, got: %v", err)
	}
}

// TestAllowlist_AllowlistedButUnpinnedToolFailsLoud asserts a tool on the
// allowlist whose lockedVersion does NOT match the pinned version yields a
// non-nil error — being allowlisted is not enough; the lock must pin the exact
// required version. CLM-007.
func TestAllowlist_AllowlistedButUnpinnedToolFailsLoud(t *testing.T) {
	al := realAllowlist()
	// acme-scan is allowlisted at 2.3.1; the lock pins a different version.
	err := CheckToolAllowed(al, "acme-scan", "2.0.0")
	if err == nil {
		t.Fatal("allowlisted-but-version-mismatched tool must fail loud, got nil")
	}
	if !strings.Contains(err.Error(), "acme-scan") {
		t.Errorf("error must name the tool, got: %v", err)
	}

	// The empty lockedVersion (tool present on the allowlist but absent from the
	// lock — not pinned at all) is also a failure: allowlisted is necessary but
	// not sufficient without a matching lock pin.
	errUnpinned := CheckToolAllowed(al, "acme-scan", "")
	if errUnpinned == nil {
		t.Fatal("allowlisted-but-unpinned (empty locked version) tool must fail loud, got nil")
	}
}

// TestAllowlist_VersionPinReadsFromLockNotSecondSource asserts CheckToolAllowed
// compares the caller-supplied lockedVersion (the lock value) against the
// allowlist pin: a tool allowlisted at vX whose lock pins vY fails loud — proving
// the pin rides the lockedVersion argument and is not silently re-read from a
// second literal source. CLM-029.
func TestAllowlist_VersionPinReadsFromLockNotSecondSource(t *testing.T) {
	al := realAllowlist()
	// Allowlisted at 1.4.0; the lock (the lockedVersion argument) pins 1.3.0.
	// If the gate consulted a second source instead of the argument, it could
	// spuriously pass — it must compare the ARGUMENT against the pin and fail.
	err := CheckToolAllowed(al, "acme-lint", "1.3.0")
	if err == nil {
		t.Fatal("allowlisted-at-vX / locked-at-vY must fail loud (pin must ride the lock argument), got nil")
	}
	if !strings.Contains(err.Error(), "acme-lint") {
		t.Errorf("error must name the tool, got: %v", err)
	}

	// Sanity: feeding the lock value that DOES match the pin passes — confirming
	// the gate genuinely reads the argument, not a hardcoded answer.
	if err := CheckToolAllowed(al, "acme-lint", "1.4.0"); err != nil {
		t.Fatalf("matching lock pin must pass, got: %v", err)
	}
}
