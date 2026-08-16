---
title: "Allowlist Overclaim Recurs Outside Issue082 Scope"
schema_version: issue/v1

issue:
  id: ISSUE-131
  title: "Allowlist Overclaim Recurs Outside Issue082 Scope"
  type: technical-debt
  status: open
  created: "2026-08-15"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# Allowlist Overclaim Recurs Outside Issue082 Scope

## Problem

ISSUE-082 (`issues/ISSUE-082-tool-allowlist-unreachable-entries.issue.md`, implemented
via commit `d534e2a`) corrected `engine.TrustedToolAllowlist()`'s doc comment
(`pkg/pack/engine/allowlist.go`), which had falsely claimed the allowlist governs "any
pack-declared command." The real, narrower guarantee — verified in ISSUE-082 and still
true today — is that the allowlist is consulted **only** for engine bindings carrying a
non-nil `Provision` block; a pack-declared command invoking a tool directly, with no
`Provision`, was never covered. That is the intended design, not a gap.

The identical false claim survives, unfixed, in three other files that were outside
ISSUE-082's declared file scope:

1. **`cmd/backstop/pack_gate.go:44-45`** — `resolveTrustedToolAllowlist`'s doc comment
   reads: "the backstop-owned trust floor every pack-declared command's tool must satisfy
   before backstop runs it." Same false "every pack-declared command" claim, in the same
   package whose own `checkEngineToolAllowed` (`pack_gate.go:796-812`) correctly states
   the Provision precondition elsewhere in the same file — confirmed accurate and NOT
   part of this issue.

2. **`cmd/backstop/recipe_apply.go:53-54`** — user-facing. The `Long` help text for
   `backstop recipe apply` (surfaced via `--help`) states: "that engine's tool must clear
   the same trusted-tool allowlist gate every pack-declared enforcement command clears —
   an un-allowlisted or wrongly-pinned tool is refused before any command is built." A
   user reading `--help` output is told something false about a security-adjacent
   guarantee: a recipe's transform op dispatched to an engine with no `Provision` block
   never touches this gate at all.

3. **`cmd/backstop/allowlist_test_helpers_test.go:44-46`** — a test-helper doc comment on
   the `Allowlist` field: "A tool absent from this map may not be run by any
   pack-declared command (CLM-006)." Same overclaim, in test code.

All three were verified directly against current source (not taken on faith from the
report that surfaced them) on 2026-08-15.

## Solution

Apply the same correction ISSUE-082 already applied to `allowlist.go`'s own doc comment
to each of the three locations above: state the real, narrower guarantee (trust floor
for `Provision`/lock-pinned tools only — a nil-`Provision` binding is exempt by
construction) instead of "any pack-declared command" / "every pack-declared command."
This is comment/help-text-only, no behavior change — same as ISSUE-082 itself was. Exact
wording is not prescribed; each site's phrasing should match its own context (internal
doc comment vs. user-facing CLI help vs. test-helper comment).

### Acceptance criteria

- None of the three files claims the allowlist governs "any"/"every" pack-declared
  command; each states the Provision-gated scope instead.
- `backstop recipe apply --help` no longer implies a blanket allowlist guarantee over
  every pack-declared enforcement command.
- No behavior change — `go test ./...` green, no assertions altered beyond comment text.

## Priority note

The user-facing CLI help text (`recipe_apply.go`) is the one a real user could actually
read and be misled by — prioritize it first. The other two (`pack_gate.go`,
`allowlist_test_helpers_test.go`) are internal/test comments a contributor might read;
lower urgency but same defect shape.

## References

- ISSUE-082 (`issues/ISSUE-082-tool-allowlist-unreachable-entries.issue.md`) — the
  original fix this issue extends to the three sibling files outside its scope.
