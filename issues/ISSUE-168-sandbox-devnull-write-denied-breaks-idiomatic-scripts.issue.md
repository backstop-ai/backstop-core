---
title: "Sandbox Devnull Write Denied Breaks Idiomatic Scripts"
schema_version: issue/v1

issue:
  id: ISSUE-168
  title: "Sandbox Devnull Write Denied Breaks Idiomatic Scripts"
  type: bug
  status: open
  created: "2026-08-18"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate
---

# Sandbox Devnull Write Denied Breaks Idiomatic Scripts

## Problem

On real Linux CI (run `32108003542`, `gate-report.json`, `pack_engines` step, confirmed at
commit `970512b`), two `pkg/packval` tests fail because a sandboxed script's write to
`/dev/null` is denied by the Landlock sandbox:

- **`TestLinuxSandbox_RealInterpreterRunsUnderTheFilter`**
  (`pkg/packval/sandbox_linux_exec_test.go:437`) — its convert-script fixture,
  `pkg/packval/testdata/sandbox/convert-jq.sh`, fails outright:
  ```
  a real interpreter could not run a real convert script under the sandbox: sandboxed run
  (stdout) failed: exit status 127: .../convert-jq.sh: 14: cannot create /dev/null: Permission
  denied
  ```
  Line 14 of that fixture is `if command -v jq >/dev/null 2>&1; then` — a completely standard,
  idiomatic POSIX shell existence-check pattern.
- **`TestLinuxSandbox_NetworkAllowedControlLegSucceeds`**
  (`pkg/packval/sandbox_linux_exec_test.go:310`) — its dynamically-generated
  `network-probe.sh` fixture hits `/dev/null: Permission denied` on two lines, visible in its
  captured stderr:
  ```
  .../network-probe.sh: line 4: /dev/null: Permission denied
  TCP_BLOCKED
  .../network-probe.sh: line 5: /dev/null: Permission denied
  UDP_BLOCKED
  PROBE_COMPLETED
  ```
  This test's actual failure mode, distinct from the `/dev/null` denials, is that TCP and UDP
  were both blocked under a capability that PERMITS the network (`TCP_OPEN`/`UDP_OPEN` markers
  absent from the report — the test's `t.Errorf` fires on that condition, not on the `/dev/null`
  lines). The `/dev/null` denials are visible in the captured output and are a real, independent
  instance of this same defect, but they are NOT what makes this particular test fail; that
  distinction matters for whoever fixes the network-blocking failure separately.

### Root cause, fully traced

Both sandbox profiles deny ALL file writes with no exception carved out for `/dev/null`, by
deliberate design, confirmed directly by reading the source:

- `pkg/packval/sandbox_capability.go` — `ConvertValidatorCapability` (used by both tests above)
  sets `WritablePaths: nil` unconditionally for convert/validator work, with the comment: "EMPTY
  for convert/validator work: darwin denies file-write* outright and parity is the spec."
- `pkg/packval/sandbox_nonlinux.go` — the darwin `sandbox-exec` profile literal is:
  ```
  (version 1)(import "bsd.sb")(deny default)(allow process*)(allow file-read*%s)(deny network*)(deny file-write*)
  ```
  a blanket `(deny file-write*)` with no `/dev/null` carve-out, structurally identical in intent
  to Linux's `WritablePaths: nil`.

So this is **not** a Linux-specific regression or a platform asymmetry in what the profiles were
designed to allow — both platforms encode the identical "deny all writes" decision, and neither
profile literal has an explicit `/dev/null` exception. It was simply never exercised by CI
before tonight (`ISSUE-163`'s `TestMain` fix is what let Linux CI reach these tests for the
first time; it did not cause this defect).

### Whether darwin is also, silently, broken the same way — checked directly

`sandbox_linux_exec_test.go:422-423` names
`TestLinuxSandbox_RealInterpreterRunsUnderTheFilter` as "the Linux analogue of darwin's
`TestSandboxConvertWithRealInterpreter`" (`pkg/packval/sandbox_realconvert_test.go:52`) — and
both tests drive the exact same fixture file, `convert-jq.sh`, containing the same
`>/dev/null 2>&1` idiom on line 14. Run directly on this darwin machine:

```
$ go test ./pkg/packval/ -run TestSandboxConvertWithRealInterpreter -v
=== RUN   TestSandboxConvertWithRealInterpreter
--- PASS: TestSandboxConvertWithRealInterpreter (0.04s)
PASS
```

It passes. So the design gap — a blanket "deny all writes" profile literal with no explicit
`/dev/null` exception on either platform — is real and identical on both platforms at the
SOURCE level, but only Linux's Landlock actually enforces it against `/dev/null` writes in
practice; macOS's `sandbox-exec`, despite having no explicit carve-out for it either, evidently
lets writes to `/dev/null` through as an emergent property of how Seatbelt treats the character
device (not something asserted anywhere in the profile text). This was confirmed empirically,
not assumed — state this precisely: the profile-literal gap is symmetric, the OBSERVED breakage
is Linux-only.

## Impact

`command -v foo >/dev/null 2>&1` and equivalents (existence checks, suppressing expected stderr
noise with `2>/dev/null`) are an extremely common, idiomatic pattern in real-world shell
scripts. Any real pack author's convert or validator script that uses this idiom will break
under backstop's Linux sandbox, with a confusing `exit status 127` / "Permission denied"
signature rather than anything that points at the sandbox as the cause. Writing to `/dev/null`
is universally considered safe and harmless — unlike writing to an arbitrary file, granting it
would not weaken the sandbox's actual security properties (no data escapes, no state persists,
no side effect is observable). This is a genuine capability gap worth closing, not a
security-relevant restriction working as intended.

## References

- `pkg/packval/sandbox_linux_exec_test.go:437` — `TestLinuxSandbox_RealInterpreterRunsUnderTheFilter`.
- `pkg/packval/sandbox_linux_exec_test.go:310` — `TestLinuxSandbox_NetworkAllowedControlLegSucceeds`.
- `pkg/packval/testdata/sandbox/convert-jq.sh:14` — `if command -v jq >/dev/null 2>&1; then`,
  the failing line, shared verbatim by both the Linux and darwin real-interpreter tests.
- `pkg/packval/sandbox_capability.go` — `ConvertValidatorCapability`, `WritablePaths: nil`.
- `pkg/packval/sandbox_nonlinux.go` — the darwin `sandbox-exec` profile literal,
  `(deny file-write*)` with no `/dev/null` exception.
- `pkg/packval/sandbox_realconvert_test.go:52` — `TestSandboxConvertWithRealInterpreter`, the
  darwin analogue; confirmed passing locally against the same fixture, establishing the
  platform-asymmetric OBSERVED behavior despite symmetric profile design.
- CI run `32108003542`, `gate-report.json` (`git_sha: 970512b`), `pack_engines` step — source of
  both failure messages quoted above.
- `ISSUE-163` / commit `970512b` — the `TestMain` fix that let Linux CI reach these tests for
  the first time; not the cause.

A fix likely needs an explicit, narrow allow-rule for `/dev/null` specifically on both
platforms — a Landlock path rule scoped to `/dev/null` on Linux, and a corresponding
`sandbox-exec` profile addition (e.g. an `(allow file-write* (literal "/dev/null"))` clause) on
darwin, for genuine platform parity between the profiles' stated intent and their actual
behavior. The specific mechanism is left to this issue's plan.

### Existence-in-world check

Performed 2026-08-18 before authoring: `grep -ril` over `issues/` and `bundles/` for "dev/null"
and "devnull" matched no open issue or bundle charter — only this issue's own not-yet-authored
file. No duplicate ownership of this surface exists.
