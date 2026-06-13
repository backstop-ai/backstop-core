---
title: "typo'd enforcement.toolchain pass keys are silently skipped"
schema_version: issue/v1

issue:
  id: ISSUE-008
  title: "typo'd enforcement.toolchain pass keys are silently skipped"
  type: bug
  status: open
  created: "2026-06-11"

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate
---

# typo'd enforcement.toolchain pass keys are silently skipped

## Problem

`declaredEntries` (pkg/check/registry.go) silently `continue`s over any
`enforcement.toolchain` key that doesn't parse as a known pass name. A
typo (`lnit:`) or out-of-vocabulary key (`typecheck:`) produces no
executor, no error, and no warning — that pass simply doesn't enforce.

Neither validation layer catches it: `Enforcement.Toolchain` is a
`map[string]ToolchainPass`, and Go's `KnownFields(true)` rejects unknown
struct fields but not unknown map keys; `validateAgainstSchema` doesn't
descend into the toolchain object (the schema enumerates lint/build/test
under `properties` but map decoding sidesteps it).

Found by the ISSUE-003 impl-review (finding F-1). Not blocking there
because no claim mandated fail-loud on this specific key, but it is the
silent-non-enforcement failure mode in spirit: a config typo quietly
disables a pass.

## Fix

Unrecognized pass keys in `enforcement.toolchain` should be a config
error (exit 2), consistent with the missing-toolchain and zero-routable
precedents. One guard in `declaredEntries` (return `*ConfigError` naming
the bad key and the allowed vocabulary) plus a test; optionally tighten
the schema's toolchain object with `additionalProperties: false` map
closure for documentation parity.

## References

- pkg/check/registry.go — declaredEntries silent continue
- ISSUE-003 impl-review finding F-1 (2026-06-11)
- Precedents: missing-toolchain ConfigError (ISSUE-003), zero-routable ConfigError (ISSUE-005)
