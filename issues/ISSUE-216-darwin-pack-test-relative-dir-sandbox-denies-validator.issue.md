---
title: "Darwin Pack Test Relative Dir Sandbox Denies Validator"
schema_version: issue/v1

issue:
  id: ISSUE-216
  title: "Darwin Pack Test Relative Dir Sandbox Denies Validator"
  type: bug
  status: open
  created: "2026-10-12"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Darwin Pack Test Relative Dir Sandbox Denies Validator

## Problem

On macOS, `backstop pack test` with no path argument (or `pack test .`) reports a false red for a
pack whose validator is correct, and the message gives no way to diagnose it.

Mechanism, verified in code:

- `cmd/backstop/pack_check.go` (~line 32) defaults `packDir := "."` when no path argument is given.
- `darwinSandboxProfile` (`pkg/packval/sandbox_nonlinux.go`, ~line 188) runs
  `filepath.EvalSymlinks(packDir)`. For `"."` that stays `"."`, so the Seatbelt profile receives
  `(subpath ".")`. Seatbelt matches kernel-resolved absolute paths, so nothing under the pack
  directory is readable.
- `/bin/sh` is then denied reading the validator script and exits 126 with
  `/bin/sh: validators/<x>.sh: Operation not permitted`.
- `phase3` (`pkg/packval/phase3.go`, ~lines 139-142) reports only
  `ERROR [phase3-fixtures/validator-positive] layer3 positive failed`, because
  `DefaultExecutor.RunValidator` (`pkg/packval/executor.go`, ~line 262) discards the sandboxed run's
  error and output (it returns `Output` but phase3 never surfaces it, and the exit status is flattened
  to `ExitCode: 1`).

`backstop pack test <absolute path>` and `backstop pack test "$PWD"` pass.

### Reproduction

macOS only. From any pack directory whose rules declare a validator that reads its fixture:

```
cd /tmp/some-pack && backstop pack test      # false red: layer3 positive failed
cd /tmp/some-pack && backstop pack test "$PWD"   # passes
```

`TestPackAuthoringLoop_EndToEnd` (`cmd/backstop/pack_authoring_loop_test.go`, darwin-only, skipped on
Linux CI) fails on main for this reason.

## Impact

- Any macOS user running `backstop pack test` or `pack test .` from inside a pack gets a false red
  on a correct pack, with an undiagnosable message that points at the pack rather than the sandbox.
- The darwin-only authoring-loop test is red on main, and Linux-only CI cannot see it.
- Evidence integrity: a failure whose real cause (sandbox denial) is dropped on the floor is the
  kind of error the gate should never present as a pack defect.

## Root cause

Two defects compound:

1. A relative pack directory reaches the Seatbelt profile unresolved (no `filepath.Abs` before
   `EvalSymlinks`), producing a `(subpath ".")` rule that matches nothing.
2. `RunValidator` discards the sandboxed run's error and output, so the denial is invisible.

Timing: the relative-dir behavior predates bc532ac. It was likely surfaced by bec4a9e (ISSUE-146,
which added a sample validator that actually reads its fixtures), but that is unconfirmed. The Linux
Landlock path is likely unaffected; this has not been tested.

## Direction

(Direction, not a plan.)

1. Make the pack directory absolute before the sandbox sees it: `filepath.Abs` before
   `EvalSymlinks` in `darwinSandboxProfile` / `sandboxExecCommand`, keeping `cmd.Dir` and the helper
   request `Dir` consistent with the resolved path. Optionally also normalize in `pack check` /
   `pack test`.
2. Add a regression test that calls `SandboxedRun` with a relative `"."` pack dir on darwin and
   asserts a validator inside the pack is readable and runs.
3. Include the sandboxed run's output and exit status in the `ValidationError` message for
   `validator-positive` (and the sibling negative seam), so a sandbox denial is diagnosable.

Keep the change small and contained; do not alter the validator polarity documented at the seam in
`phase3.go`.

## Notes

- Related but distinct from ISSUE-214 (macOS BSD userland portability in a repo test script): both
  are darwin-only defects invisible to Linux-only CI.
