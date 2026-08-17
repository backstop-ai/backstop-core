package engine

import "fmt"

// TrustedToolAllowlist returns the backstop-OWNED {tool -> pinned version}
// allowlist: the trust floor for tools BACKSTOP ITSELF downloads and pins on the
// user's behalf (REQ-002). It ships ATOMICALLY with the pack-declared bindings
// (Sharp Edge 1) and lives in pkg/pack/engine beside the binding so the
// validate-time, provision-time, and dispatch-time checks all consume ONE source.
//
// SCOPE — this map is consulted ONLY for engine bindings carrying a non-nil
// Provision block. Every call site of CheckToolAllowed returns early when a
// binding declares no provision, so a pack-declared command run against a
// Layer-0/runtime tool the pack invokes directly is outside this gate's reach by
// construction; the Provision/lock path is the whole of what this map governs.
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
		"semgrep":  "1.156.0",
		"ast-grep": "0.43.0",
		// grep is the text-presence engine the contracts absence probe rides
		// (SPEC-038 REQ-005/CLM-016). It is pack-DECLARED (no DefaultRegistry entry)
		// and provisioned by the contracts packs, so the allowlist is what lets a
		// pack-declared grep command clear the trust floor. A version of "*" pins to
		// "present" — grep is a POSIX/Layer-0 tool whose version backstop does not
		// introduce (unlike semgrep/ast-grep, which ride the lock pin), so the trust
		// requirement is presence rather than a concrete version.
		"grep": "*",
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
