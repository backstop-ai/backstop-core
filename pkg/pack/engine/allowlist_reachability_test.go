package engine

import (
	"os"
	"strings"
	"testing"
)

// allowlist_reachability_test.go (ISSUE-082): the allowlist may carry ONLY tools
// that are actually reachable. An entry is reachable only through
// CheckToolAllowed, and every call site of that function returns early on a nil
// Provision block — so a tool no pack.yml provisions is never asked about, and an
// entry for it is dead weight rather than a trust guarantee. These tests hold the
// removal (and the narrowed doc comment) in place: four of the five removed
// entries were originally added "to be safe" against a pack that never consulted
// them, and a deletion with nothing asserting it regresses the moment someone
// repeats that reasoning.
//
// SPEC-038 Sharp Edge 16 assigns the absence assertions to ISSUE-082 (this file)
// so allowlist_grep_test.go and this file never mandate duplicate tests over the
// same map.

// TestAllowlist_UnreachableToolsAbsent asserts the real TrustedToolAllowlist()
// carries none of the five unreachable tools, and still carries the three with a
// live Provision-bearing consumer. Both directions are asserted deliberately: a
// bare absence check would pass against an empty map, which is a vacuous green
// (CLM-001).
func TestAllowlist_UnreachableToolsAbsent(t *testing.T) {
	al := TrustedToolAllowlist()

	for _, tool := range []string{"rg", "oxlint", "bun", "tsc", "prettier"} {
		if _, present := al[tool]; present {
			t.Errorf("tool %q must NOT be on the trusted-tool allowlist: no pack.yml declares a provision: block for it, so no runtime path ever consults the allowlist about it (ISSUE-082)", tool)
		}
	}

	for _, tool := range []string{"semgrep", "ast-grep", "grep"} {
		pinned, present := al[tool]
		if !present {
			t.Errorf("tool %q must REMAIN on the trusted-tool allowlist — it has a live Provision-bearing consumer", tool)
			continue
		}
		if strings.TrimSpace(pinned) == "" {
			t.Errorf("tool %q is allowlisted but its pinned version is empty — the pin must be concrete", tool)
		}
	}

	if len(al) != 3 {
		t.Errorf("the trusted-tool allowlist must carry exactly the three reachable tools (semgrep, ast-grep, grep), got %d: %v", len(al), al)
	}
}

// TestAllowlist_FileCarriesNoSuppression reads allowlist.go from disk and asserts
// backstop/self green is achieved BY REMOVAL, not by a suppression: the file
// carries no semgrep directive, no backstop `@waiver:` marker, and no surviving
// doc-comment overclaim. A green self-pack run alone cannot distinguish those
// worlds; this test can (CLM-002, CLM-003).
//
// Both suppression mechanisms are closed. Checking only for a semgrep directive
// would leave the backstop-native one open: pkg/waiver/waiver.go defines
// "@waiver:" as a byte-scanned inline marker and backstop.yml declares no
// non-waivable rules, so a waiver comment would suppress the self-pack finding
// just as effectively while sailing past a directive-only assertion.
func TestAllowlist_FileCarriesNoSuppression(t *testing.T) {
	data, err := os.ReadFile("allowlist.go")
	if err != nil {
		t.Fatalf("allowlist.go must be readable — an unread file makes every does-not-contain assertion below trivially pass: %v", err)
	}
	src := string(data)

	// "nosem" is the shorter form, so this catches semgrep's `nosem` directive as
	// well as `nosemgrep`.
	if strings.Contains(src, "nosem") {
		t.Errorf("allowlist.go must carry NO semgrep suppression: the self-pack rule that fired here was CORRECT, and removing the entry — not suppressing the rule — is the honest resolution (ISSUE-082)")
	}
	if strings.Contains(src, "@waiver:") {
		t.Errorf("allowlist.go must carry NO backstop waiver marker: a waiver suppresses the self-pack finding exactly as a semgrep directive would, and switching mechanisms is not a fix (ISSUE-082)")
	}
	if strings.Contains(src, "no matter what a pack declares") {
		t.Errorf("allowlist.go's doc comment must NOT claim coverage of any pack-declared command regardless of what a pack declares: every CheckToolAllowed call site returns early on a nil Provision, so the real guarantee is the narrower Provision/lock-pin scope")
	}
}
