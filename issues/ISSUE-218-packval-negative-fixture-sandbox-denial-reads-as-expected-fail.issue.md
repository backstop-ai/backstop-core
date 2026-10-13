---
title: "Packval Negative Fixture Sandbox Denial Reads As Expected Fail"
schema_version: issue/v1

issue:
  id: ISSUE-218
  title: "Packval Negative Fixture Sandbox Denial Reads As Expected Fail"
  type: bug
  status: open
  created: "2026-10-12"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Packval Negative Fixture Sandbox Denial Reads As Expected Fail

## Problem

In `backstop pack test` phase3, a validator run against a NEGATIVE fixture (one the validator must
flag by exiting non-zero) treats every non-zero exit as the expected failure. A run in which the
validator never evaluated the fixture, because the sandbox denied it, the interpreter was missing,
or the script crashed, therefore passes silently: a vacuous green on the negative side.

Mechanism, verified in code:

- `DefaultExecutor.RunValidator` (`pkg/packval/executor.go`, ~line 262) calls `SandboxedRun` and, on
  any error, returns `ExecutionResult{Passed: false, ExitCode: 1}` with a nil error. The real exit
  status (126 sandbox denial or non-executable, 127 missing interpreter, signal or crash) is
  flattened to 1 and the failure kind is lost.
- The negative branch in `pkg/packval/phase3.go` (~lines 164-168) only errors when `err != nil`
  (never true for this executor) or when `r.Passed` is true ("unexpectedly passed"). `!r.Passed`
  is the success path, so a run that could not execute is indistinguishable from "validator ran and
  flagged the fixture".
- The positive branch (~lines 139-142) fails on the same flattened result, so a broken validator
  surfaces only on the positive side and only if the pack declares a positive fixture.

Concrete instance: ISSUE-216 (darwin relative pack dir) makes `/bin/sh` exit 126 `Operation not
permitted` on every validator run. The positive side reports a false red, while the negative side
reports green for the wrong reason.

### Reproduction

Pack with a validator rule and a negative fixture, validator path made unrunnable (for example
`chmod -x`, a bad shebang interpreter, or the ISSUE-216 relative-dir case on macOS): `backstop pack
test` reports no `validator-negative` error even though the validator never examined the fixture.

## Impact

- A negative fixture that was never evaluated counts as proof that the validator catches the bad
  case. The claim is mechanically "verified" with no evidence behind it.
- Same family as ISSUE-206/207/209: a gate path where a tool that did not run reads as a clean
  result. Pack authors ship validators whose negative-side coverage is unproven.
- Diagnosability: the discarded exit code and output (also noted in ISSUE-216/147) means even a
  visible failure cannot be traced to its cause.

## Root cause

`RunValidator`'s result type carries only a boolean "exited zero", and phase3 interprets the negative
side as "not zero is good". There is no channel for "the validator did not run" versus "the
validator ran and flagged", and no declared contract for what a flag exit looks like.

## Direction

(Direction, not a plan; options to be settled in the plan.)

1. Have `RunValidator` / `SandboxedRun` return the real exit code and output, and classify run
   failures (exit 126/127, sandbox or exec errors, signals) separately from a flagging exit; phase3
   negative treats a run failure as a `validator-negative` error with the output surfaced.
2. Alternatively or additionally, require a specific flag exit code or marker for a validator to
   count as having fired, reusing ISSUE-207's declared-exit-code concept for validators.
3. Add tests: a negative fixture whose validator is unrunnable (non-executable, missing
   interpreter) must produce an error, while a validator that runs and exits with the flag code
   still passes.

Do not alter the validator polarity documented at the seam in `phase3.go` (positive passes, negative
fails); only add the ran-versus-could-not-run distinction. This changes phase3 judging semantics, so
it needs its own plan and must not be folded into the ISSUE-216 fix.

## Notes

- Cross-references: ISSUE-216 (darwin relative pack dir; the instance that exposed this), ISSUE-207
  (declared exit codes for engines; candidate mechanism), ISSUE-147 (relative packDir on darwin
  convert steps, stderr discarded), ISSUE-206/209 (same vacuous-green family), ISSUE-092 (phase3
  validator polarity; keep pinned by `TestPackVal_P3_ValidatorPolarityUnchangedByFindingsFix`).
- Found by the PLAN-ISSUE-216 planner.
