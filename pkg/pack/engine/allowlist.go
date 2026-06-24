package engine

import "fmt"

// TrustedToolAllowlist returns the backstop-OWNED {tool -> pinned version}
// allowlist: the trust floor every pack-declared command must satisfy before
// backstop will run it (REQ-002). A tool ABSENT from this map may not be run by
// any pack-declared command, no matter what a pack declares — this is the
// non-negotiable security gate that ships ATOMICALLY with the pack-declared
// bindings (Sharp Edge 1). It lives in pkg/pack/engine beside the binding so the
// validate-time, provision-time, and dispatch-time checks all consume ONE source.
//
// The pinned version here is the REQUIRED version; the gate compares it against
// the version the caller reads from the existing backstop.lock / Provision
// verification path (CheckToolAllowed's lockedVersion argument), so the pin
// cannot drift from a second literal source (CLM-029).
func TrustedToolAllowlist() map[string]string {
	return map[string]string{
		// The backstop-introduced findings/opinion engines (pinned via the lock
		// path). These mirror the DefaultRegistry Provision pins — the allowlist is
		// the trust floor, the lock is the proof a build is actually pinned there.
		"semgrep":  "1.96.0",
		"ast-grep": "0.43.0",
		// grep / ripgrep are the text-presence engines the contracts absence probe
		// rides (SPEC-038 REQ-005/CLM-016). They are pack-DECLARED (no
		// DefaultRegistry entry); the allowlist is what lets a pack-declared grep
		// command clear the trust floor. A version of "*" pins to "present" — grep
		// is a POSIX/Layer-0 tool whose version backstop does not introduce or
		// auto-provision (unlike semgrep/ast-grep, which ride the lock pin), so the
		// trust requirement is presence on the allowlist, matched by the assumed
		// "*" lock value the dispatch gate supplies for an un-provisioned tool.
		"grep": "*",
		"rg":   "*",
	}
}

// CheckToolAllowed is the PURE trust-floor check every command-running caller
// shares (validateEngine, provisionEngines, the dispatch gate). It returns a
// non-nil error — wrapped by callers into a *check.ConfigError (exit 2) naming
// tool+pack — when:
//
//   - the tool is NOT on the allowlist (CLM-006), or
//   - the tool IS on the allowlist but lockedVersion does not match its pinned
//     version (CLM-007).
//
// lockedVersion is the caller's read of the existing backstop.lock / VerifyLock
// path — NOT a second literal — so the version pin rides the lock and cannot
// drift from a second source (CLM-029). This package returns a plain error (no
// pkg/check import — leaf placement); the cmd layer wraps it into a ConfigError.
func CheckToolAllowed(allowlist map[string]string, tool string, lockedVersion string) error {
	pinned, ok := allowlist[tool]
	if !ok {
		return fmt.Errorf("tool %q is not on the trusted-tool allowlist: backstop will not run a pack-declared command for an un-allowlisted tool", tool)
	}
	if lockedVersion != pinned {
		return fmt.Errorf("tool %q is allowlisted at version %q but the lock pins %q: a pack-declared command's tool must be lock-pinned to its allowlisted version", tool, pinned, lockedVersion)
	}
	return nil
}
