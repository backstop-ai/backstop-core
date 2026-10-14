---
title: "Go Toolchain 1.10 Lint Debt"
schema_version: issue/v1

issue:
  id: ISSUE-226
  title: "Go Toolchain 1.10 Lint Debt"
  type: bug
  status: open
  created: "2026-10-13"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Go Toolchain 1.10 Lint Debt

## Problem

After adopting `backstop-ai/go-toolchain` 1.10.0, `backstop gate --all` reports 33 lint findings in
files that recent work did not touch. The pack's staticcheck, errcheck and unused checks now cover
these files. The findings are pre-existing debt, not regressions:

- staticcheck SA9009 in `cmd/backstop/embed.go`: a prose comment that begins `// go:embed` reads
  as a malformed compiler directive.
- staticcheck QF1011 in `embed_test.go`: a typed blank declaration used as a compile-time type
  assertion.
- errcheck in tests: unchecked `os.Setenv`/`os.Unsetenv` (`cmd/backstop/output_test.go`,
  `pkg/check/output_test.go`, `pkg/gate/output_test.go`), unchecked `os.WriteFile`/`os.MkdirAll`
  (15 sites in `pkg/check/scope_test.go`), unchecked `os.RemoveAll`
  (`pkg/pack/distribution/list_test.go`) and unchecked `os.Chdir` (7 sites in
  `pkg/scaffold/git_executor_real_test.go`).
- errcheck in production code: unchecked `os.RemoveAll` in `pkg/pack/distribution/remove.go`.
- unused: `allSkippedSteps` in `pkg/gate/gate_test.go`.

## Impact

- The full gate cannot pass on the repository while these findings stand.
- In tests, an unchecked setup error surfaces later as a confusing assertion failure, or not at all.
- In `pack remove`, a failed directory delete is silently ignored. The command then drops the pack
  from `backstop.yml`, `backstop.lock` and provenance and reports success, while the pack's files
  stay in `.backstop/packs/`.

## Direction

Mechanical fixes. Check test setup errors (`t.Setenv`, `t.Chdir`, `t.Fatal`). Reword the comment.
Keep the type assertion as a real check. Delete the unused helper. In `pack remove`, return the
delete error, wrapped, before any manifest or lockfile change.

## Scope

1. The 33 go-toolchain 1.10.0 findings listed under Problem.
2. Five more findings in the same files. The `--all` run above could not show them, because its
   semgrep engines never started (argument list too long) and later steps did not run. Checking
   each changed file whole with `gate --file` showed them:
   - go-standards no-ignored-errors in `pkg/pack/distribution/remove.go` (`lf, _ := ReadLockfile`).
   - go-standards no-global-mutable-state in `embed_test.go`, matching a `var` declared inside the
     test function.
   - test_substantiveness: `TestRealGitExecutor_FetchTags_InRepo` had no assertions.
   - coverage_threshold below 90% for `cmd/backstop/embed.go` (0/1) and
     `pkg/pack/distribution/remove.go`.

   The four go-standards error-wrapping warnings in `Remove` were also fixed.

## Fix

- `cmd/backstop/embed.go`: the comment no longer begins with `go:embed`. No directive changed.
- `embed_test.go`: the typed blank declaration was the test's only check, so it is replaced with
  real assertions instead of being dropped. The test now checks `reflect.TypeOf(SchemaFS)` against
  `embed.FS`, and runs `fs.Stat` on its root.
- NO_COLOR tests: `t.Setenv` restores the original value. The unset is then error-checked. Before,
  two tests left NO_COLOR unset for later tests.
- `pkg/check/scope_test.go`: setup goes through `mustWriteFile` and `mustMkdirAll` helpers that fail
  the test on error.
- `pkg/scaffold/git_executor_real_test.go`: `t.Chdir`. `TestRealGitExecutor_FetchTags_InRepo` now
  fetches from a local origin repository that holds a tag, and asserts the tag arrives.
- `pkg/gate/gate_test.go`: the unused `allSkippedSteps` is deleted. Nothing in the repository
  referenced it.
- `pkg/pack/distribution/list_test.go`: the `os.RemoveAll` error is checked.
- `cmd/backstop/embed_test.go` (new): checks that the `ListSchemas` re-export returns the same paths
  as the module root.

### Behavior change in `pack remove`

- A failed pack directory delete now returns `removing pack directory <dir>: <cause>`.
  `backstop.yml`, `backstop.lock` and provenance are left unchanged.
- `backstop.lock` is now read before anything changes. A missing lockfile is still allowed. An
  unreadable one now returns an error with nothing changed. Before, it was ignored and the stale
  lock entry stayed.
- The errors returned from `Remove` are wrapped with the pack name and the step that failed.
- New tests: `TestPackRemove_DeleteFailureLeavesManifestsUntouched`,
  `TestPackRemove_UnreadableLockfileChangesNothing` and
  `TestPackRemove_LockfileWriteFailureReturnsError`. The delete-failure test fails against the old
  code. The two permission-based tests skip when run as root, as
  `pkg/initialize/step_config_test.go` already does, because a read-only path does not refuse root.

## Verification

- `go test` passes for `.`, `./cmd/backstop`, `./pkg/check`, `./pkg/gate`,
  `./pkg/pack/distribution` and `./pkg/scaffold`.
- The go-standards core rules report zero findings on `remove.go`, against five at the previous
  commit.
- `backstop gate --file` over the changed files: `Steps: 8 passed, 2 failed, 2 skipped, 2 warned`,
  `Total violations: 5`. `coverage_threshold` passes. That run was before the error-wrapping change
  in `remove.go`, which removed four of its warnings.

The 5 blocking findings that remain were there before this change. A follow-up issue tracks them:

- go-standards no-global-mutable-state on `var SchemaFS embed.FS` in `cmd/backstop/embed.go`. This
  is a SPEC-005 contract signature.
- test_substantiveness "does not call package backstop" for
  `TestGate_BaselineComparison_WiredAfterAccumulatedChecks`,
  `TestGate_BaselineComparison_MissingBaselineSkipsWithReason`
  (`pkg/gate/gate_test.go`), `TestGateOutput_BaselineDifferential_HumanMessaging` and
  `TestGateOutput_BaselineDifferential_JSONDiagnostics` (`pkg/gate/output_test.go`). SPEC-019
  declares `package: cmd/backstop`, but these tests are in `pkg/gate`.

## Notes
