---
title: "Test Verification Fails Without Specs Dir"
schema_version: issue/v1

issue:
  id: ISSUE-225
  title: "Test Verification Fails Without Specs Dir"
  type: bug
  status: open
  created: "2026-10-13"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: moderate
---

# Test Verification Fails Without Specs Dir

## Problem

`backstop gate` fails the `test_verification` step in a repository that has no `specs/` directory:

```
[test_verification] failed to extract mandated tests: reading spec dir .../specs: no such file or directory
```

Observed 2026-10-13 in the `backstop-ai/bash-toolchain` pack repository. Its `main` fails the same
way, so no recent change caused it.

Mechanism: `ExtractMandatedTests`, `CountTerminalSpecs` and the two other spec readers in
`pkg/gate/step_testverify.go` (~lines 116, 238, 690, 735) call `os.ReadDir(specDir)` and return
any error, including `fs.ErrNotExist`.

## Impact

- The gate can never pass in a repository that legitimately has no specs, such as a pack repository
  or a project that has adopted packs but not the artifact workflow.
- The failure message is accurate, but it treats an ordinary layout as an error.

## Root cause

An absent spec directory is treated the same as an unreadable one.

## Direction

(Direction, not a plan. The choice is a design decision.)

The step must not turn vacuously green. "No specs dir" must not let a project with specs elsewhere
skip mandated-test verification. Options:

1. Treat `fs.ErrNotExist` as "zero specs" and report it as an explicit `skipped` with the reason,
   not as `pass`.
2. Keep failing closed, but only when the project declares an artifact layout (or has any other
   artifacts), and skip with a reason otherwise.
3. Make the expectation explicit in `backstop.yml` (for example, the project declares it has no
   specs).

## Notes

- Related: the vacuous-green family (a step must never pass on missing input).
