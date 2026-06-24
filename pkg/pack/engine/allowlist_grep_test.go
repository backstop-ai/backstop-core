package engine

import (
	"strings"
	"testing"
)

// allowlist_grep_test.go (SPEC-038 TASK-005, REQ-005): the grep/rg allowlist
// gating matrix. The grep engine is pack-declared (no DefaultRegistry entry), so
// the ONLY thing that lets a pack-declared grep command clear the trust floor is
// grep/rg being on the backstop-owned TrustedToolAllowlist at a pinned version
// (CLM-016). The allowlist gating is the tested invariant: allowlisted+pinned →
// runs (CLM-017); removed from the allowlist → loud un-allowlisted ConfigError
// (CLM-018). This lands ATOMICALLY with the grep engine declaration (Sharp Edge 6).

// TestAllowlist_GrepAndRgPresentPinned asserts grep and rg are both present on the
// backstop-owned trusted-tool allowlist at a pinned (non-empty) version — the
// trust floor a pack-declared grep command must clear (CLM-016).
func TestAllowlist_GrepAndRgPresentPinned(t *testing.T) {
	al := TrustedToolAllowlist()
	for _, tool := range []string{"grep", "rg"} {
		pinned, ok := al[tool]
		if !ok {
			t.Fatalf("tool %q must be on the trusted-tool allowlist (the grep engine cannot run otherwise)", tool)
		}
		if strings.TrimSpace(pinned) == "" {
			t.Errorf("tool %q is allowlisted but its pinned version is empty — the pin must be a concrete version", tool)
		}
	}
}

// TestAllowlist_GrepAllowlistedPinnedRuns asserts a pack-declared grep command
// passes CheckToolAllowed when grep is on the allowlist AND lock-pinned to its
// allowlisted version — the trust gate passes, so the engine runs (CLM-017).
func TestAllowlist_GrepAllowlistedPinnedRuns(t *testing.T) {
	al := TrustedToolAllowlist()
	pinned := al["grep"]
	if pinned == "" {
		t.Fatal("fixture invariant: grep must be allowlisted with a pinned version")
	}
	if err := CheckToolAllowed(al, "grep", pinned); err != nil {
		t.Fatalf("allowlisted + lock-pinned grep must pass the trust gate, got: %v", err)
	}
}

// TestAllowlist_GrepUnallowlistedFailsLoud asserts that when grep/rg are removed
// from the allowlist, a pack-declared grep command is NOT run — CheckToolAllowed
// returns the loud un-allowlisted error naming the tool, so the engine cannot run
// before it is allowlisted (CLM-018). The cmd layer wraps this into a ConfigError
// (exit 2) naming tool+pack; this asserts the leaf error it wraps.
func TestAllowlist_GrepUnallowlistedFailsLoud(t *testing.T) {
	// Build an allowlist WITHOUT grep/rg to model the removed-from-allowlist state.
	stripped := map[string]string{}
	for k, v := range TrustedToolAllowlist() {
		if k == "grep" || k == "rg" {
			continue
		}
		stripped[k] = v
	}
	if _, present := stripped["grep"]; present {
		t.Fatal("fixture invariant broken: grep must be absent from the stripped allowlist")
	}
	err := CheckToolAllowed(stripped, "grep", "any")
	if err == nil {
		t.Fatal("un-allowlisted grep must fail loud, got nil — the engine would run un-trusted")
	}
	if !strings.Contains(err.Error(), "grep") {
		t.Errorf("error must name the un-allowlisted tool grep, got: %v", err)
	}
	if !strings.Contains(err.Error(), "allowlist") {
		t.Errorf("error must explain the allowlist trust floor, got: %v", err)
	}
}
