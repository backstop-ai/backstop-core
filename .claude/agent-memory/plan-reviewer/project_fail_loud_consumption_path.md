---
name: fail-loud-consumption-path
description: When reviewing pkg/check config-error (fail-loud) plans, trace every executor-construction path, not just the guard site
metadata:
  type: project
---

The ISSUE-003 (missing-toolchain), ISSUE-005 (zero-routable), ISSUE-008
(typo'd toolchain key) family all turn a silent skip into a `*check.ConfigError`
(exit 2). Reviewing these correctly requires tracing the FULL consumption path,
because the guard's effectiveness depends on which constructor every consumer uses.

Key facts in pkg/check (verified 2026-06-12):
- `buildExecutorsForConfigErr` (registry.go) is the error-RETURNING constructor.
- `buildExecutorsForConfig` / `buildDefaultExecutors` / `buildDefaultExecutorsWithRunner`
  are error-DISCARDING wrappers (`execs, _ := ...`). A guard placed in
  buildExecutorsForConfigErr is INVISIBLE through these wrappers.
- BUT those discarding wrappers have ZERO production callers — only tests use them.
- The only production executor-construction is check.go RunWith: when
  `opts.Executors == nil` it calls buildExecutorsForConfigErr and propagates the error.
- Both production entry points (check.Run, and gate.go's check.RunWith) enter with
  Executors==nil, so the error path is always hit.
- Standalone CLI translation: cmd/backstop/code_check.go type-asserts
  `runErr.(*check.ConfigError)` → ExitConfigError(2), carrying Message verbatim.
  So a registry-level ConfigError reaches exit 2 with NO CLI-layer change.

**Why:** ISSUE-008's go-language early-return (registry.go:204) means a guard placed
only in declaredEntries/resolveToolchain never runs for go projects. The correct fix
hoists the guard to the TOP of buildExecutorsForConfigErr, before the language branch.

**How to apply:** For any fail-loud pkg/check plan, confirm (1) the guard sits where
EVERY language reaches it, and (2) no production path constructs executors via an
error-discarding wrapper. If a future change adds a production caller of
buildDefaultExecutors*, the guard would be silently bypassed — flag it.
