---
title: "go-toolchain go-test Engine Reports Test Failures As an Opaque Crash, Not Parseable Findings"
schema_version: issue/v1

issue:
  id: ISSUE-067
  title: "go-toolchain go-test Engine Reports Test Failures As an Opaque Crash, Not Parseable Findings"
  type: technical-debt
  status: open
  created: "2026-07-17"

complexity:
  scope: contained
  uncertainty: known
  risk: critical
---

# go-toolchain go-test Engine Reports Test Failures As an Opaque Crash, Not Parseable Findings

## Problem

When `go test` exits non-zero because a test FAILED (not because the toolchain failed to run), the
`backstop/go-toolchain` go-test engine surfaces it as an opaque dispatch error —
`dispatching findings engine "go-test" for pack backstop/go-toolchain: pack backstop/go-toolchain
engine "go test" crashed: non-zero exit with no parseable findings: exit status 1` — instead of a
finding that names the failing test(s). A genuine test regression is therefore indistinguishable
from an environmental tool crash on the gate surface, and reads as the latter.

Discovered in ISSUE-064: a real test regression appeared at the gate ONLY as this "crash." It was
dismissed (by the implementer, by the coordinator, and reasoned-around by the impl-reviewer) as a
known/environmental go-test crash — the exact failure mode this reporting invites. The truth was
only established by running `go test` directly, then re-running the gate after the fix and watching
the "crash" disappear. A gate whose test failures masquerade as environmental noise cannot be
trusted to fail loud, which is the whole point of the gate.

## Root cause

The engine's non-zero-exit handling treats "process exited non-zero AND I parsed zero findings" as a
crash, when for `go test` a non-zero exit with failing tests is the NORMAL, expected signal. The
converter (`scripts/test-to-sarif.sh`) is not extracting the `--- FAIL:` / failure output into
findings before the exit code is judged, so the exit code wins and the real failures are discarded.

## Direction (to be specified)

Run `go test` in a machine-readable mode (`-json`) and have the go-test engine emit a per-failure
finding (test name, package, message) so a failing test becomes a legible gate finding. Distinguish
a genuine tool CRASH (compile error, panic before any test output, toolchain missing → engine error
is correct) from TEST FAILURES (tests ran, some failed → findings, not a crash). Fold in with
ISSUE-066: a full-package run that fails must produce legible findings, not an opaque crash.

## Notes / references

- Surfaced by ISSUE-064's impl-review. Sibling to ISSUE-066 (narrow -run filter). Marked `critical`
  risk because this defect silently converts real test failures into dismissible "environmental"
  noise — a trust hole in the gate's loud-failure guarantee, worse than a mere ergonomics gap.
- Lives in the `backstop/go-toolchain` pack (engine binding + `scripts/test-to-sarif.sh`), tracked
  here in backstop-core's issues.
