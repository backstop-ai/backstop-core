---
title: "pkg/check diff scope misses untracked files — dual scope-resolver drift"
schema_version: issue/v1

issue:
  id: ISSUE-004
  title: "pkg/check diff scope misses untracked files — dual scope-resolver drift"
  type: bug
  status: open
  created: "2026-06-11"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# pkg/check diff scope misses untracked files — dual scope-resolver drift

## Problem

The repo has two independent diff-scope resolvers that have drifted out of parity:

1. **`pkg/gate/scope.go` — `resolveGateScopeDiff`** (used by `backstop gate`): correct. After each merge-base diff and after the local-changes fallback, it appends `git ls-files --others --exclude-standard` output (lines 115 and 129), so untracked files are included in scope.

2. **`pkg/check/scope.go` — `resolveScopeDiff`** (used by standalone `backstop check`): buggy. The function (lines 166–202) drives a merge-base cascade and a local-diff fallback using only `git diff --name-only`. There is no call to `git ls-files`. Untracked files never appear in the resolved scope.

The `GitExecutor` interface in `pkg/check/scope.go` (lines 29–34) exposes only `IsGitRepo`, `MergeBase`, `DiffNameOnly`, and `DiffLocal`. There is no `UntrackedFiles` method, so the gap is present in any test double that implements the interface as well.

`backstop gate` in diff mode is not affected. `gate.CheckScoped` (`cmd/backstop/gate.go` lines 449–465) receives the gate-computed scope — already including untracked files from `pkg/gate/scope.go` — and feeds it to `pkg/check` one file at a time via `ScopeModeFile`. The exposure is strictly the standalone `backstop check --scope diff` path.

## Impact

A brand-new uncommitted file completely escapes lint, semgrep, and pack-rule enforcement when running `backstop check` in its default diff mode. New files are precisely the highest-risk content for policy enforcement: they have no prior review, no baseline, and no existing violations to compare against. The tool gives a clean result while ignoring the most dangerous surface.

The fix already exists for the gate half (landed in backstop-core). The check half did not receive the corresponding change.

## Solution

**Minimum fix (parity):** Add an `UntrackedFiles` method to the `GitExecutor` interface in `pkg/check/scope.go` and implement it on `DefaultGitExecutor` using `git ls-files --others --exclude-standard`. In `resolveScopeDiff`, append the result at each return point that currently returns only tracked files — matching the pattern in `resolveGateScopeDiff`.

**Preferred remediation (consolidation):** The gate path already solves the problem correctly by computing scope once in `pkg/gate/scope.go` and then calling `check.CheckScoped` with an explicit file list. The standalone check CLI path should follow the same pattern: accept an injected file list, delegating all scope resolution to a single shared resolver. This eliminates the duplicated resolver entirely and prevents the same drift from recurring. The choice between (a) moving the canonical resolver into a shared package consumed by both `pkg/gate` and `pkg/check`, or (b) having `pkg/check`'s `resolveScopeDiff` delegate to `pkg/gate`'s resolver, depends on the desired package dependency direction — `pkg/check` currently has no dependency on `pkg/gate`, which is the correct layering to preserve.

The minimum fix ships the correctness; the consolidation closes the structural root cause.

## References

- `pkg/check/scope.go` lines 29–34 (`GitExecutor` interface), 166–202 (`resolveScopeDiff`)
- `pkg/gate/scope.go` lines 104–131 (`resolveGateScopeDiff`, correct implementation)
- `cmd/backstop/gate.go` lines 449–465 (`CheckScoped` — gate path, not affected)
- Related to ISSUE-002 (executor stubs) — both are enforcement-integrity findings from the same sweep; independent fixes
