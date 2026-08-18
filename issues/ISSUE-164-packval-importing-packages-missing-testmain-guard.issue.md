---
title: "Packval Importing Packages Missing Testmain Guard"
schema_version: issue/v1

issue:
  id: ISSUE-164
  title: "Packval Importing Packages Missing Testmain Guard"
  type: question
  status: open
  created: "2026-08-18"

complexity:
  scope: isolated
  uncertainty: novel
  risk: safe
---

# Packval Importing Packages Missing Testmain Guard

## Problem

`ISSUE-163` fixed one collision between two mechanisms: on Linux, `pkg/packval`'s sandbox
dispatch re-execs the currently running binary as a "helper" subprocess, and `cmd/backstop`'s
test binary — because it lacked a `MaybeRunSandboxHelper()` guard at the top of `TestMain` —
would re-enter its own `TestMain`, try an unconditional `go build` in the wrong directory, and
die before the helper handoff ever happened. The fix added the guard to
`cmd/backstop/integration_test.go`'s `TestMain`, mirroring `pkg/packval/main_test.go`'s existing
correct pattern.

That fix, and its regression-pinning roster test
(`TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain` in
`cmd/backstop/sandbox_helper_testmain_guard_test.go`), both work by finding every `TestMain` that
exists in a package reaching `pkg/packval` and asserting the guard is present in it. That
approach has a structural blind spot: **it can only pin a guard onto a `TestMain` that already
exists.** A package that imports `pkg/packval` but declares no `TestMain` at all is invisible to
both the fix and the pin, and is therefore silently outside their coverage.

Two such packages exist today:

- **`pkg/pack/distribution`** — specifically `validator.go`, which runs the real packval
  pipeline (`packval.NewPipeline(packDir, ...).Run()`, confirmed at `validator.go:60` at time of
  writing). This package has no `TestMain`.
- **`pkg/pack/engine`** — `import_cycle_test.go` and `binding_test.go` both import
  `pkg/packval`. This package also has no `TestMain`.

Neither package has a `TestMain` at all, so if any test in either one can reach real sandboxed
dispatch on Linux, the mechanism is the same class of collision `ISSUE-163` describes, in a
quieter shape: the re-exec target becomes that package's test binary, Go's **default** generated
`TestMain` runs (there being no custom one to intercept), `m.Run()` executes the whole test
suite, and the sandbox's parent process reads the entire suite's output as if it were the
sandboxed command's own output — rather than the build-failure crash `ISSUE-163`'s defect
produced, which at least failed loudly and visibly. A silent misattribution of a whole suite's
output is arguably worse than a loud crash, precisely because it might not look broken.

## What is, and is not, known

**Known:** both packages import `pkg/packval` and neither declares a `TestMain`. This was
confirmed by direct inspection during the `ISSUE-163`/`PLAN-ISSUE-163` investigation
(`grep -rl "backstop-core/pkg/packval" --include="*.go" cmd pkg tests`, reduced to the
directories with no `func TestMain` declaration).

**Explicitly not known, and not measured by anyone as of this filing:** whether any test in
`pkg/pack/distribution` or `pkg/pack/engine` actually drives real sandboxed dispatch on Linux —
the way `TestSubstantivenessFixtures_RealPackTestPassesPhase3` does in `cmd/backstop` (the
concrete reproducing test `ISSUE-163` traced its collision through). `validator.go` running
`packval.NewPipeline(...).Run()` establishes that the *pipeline* is reachable from
`pkg/pack/distribution`'s code, not that any *test* in that package pushes it through a code path
that triggers the Linux sandbox's re-exec specifically (as opposed to, e.g., running against
fixtures with no rules that require sandboxed execution, or being skipped/short-circuited before
reaching that point). No one has traced a specific test in either package through to a real
sandboxed-dispatch call on Linux, in CI or otherwise. This is an **open question to investigate**,
not a confirmed defect, and it must not be treated as one until that tracing happens.

## Why this is filed as a question, not a bug

The gap in coverage (a `TestMain`-less packval-importing package is invisible to `ISSUE-163`'s
guard and its pin) is a fact. Whether that gap is *live* — whether it can actually be exploited by
a real test run on Linux, causing real misattributed output rather than staying a theoretical
shape — is unmeasured. Filing this as a confirmed `bug` would overstate certainty the
investigation doesn't have (per `ISSUE-161`'s precedent for the same distinction in this repo:
question when the underlying live-exploitability is genuinely unknown, bug when it's confirmed).
If investigation confirms a real test in either package does reach sandboxed dispatch on Linux,
this should be re-filed or promoted to `bug` at that point — not before.

## Impact if confirmed live

Same failure class as `ISSUE-163`, but quieter and arguably worse to diagnose: instead of a
crash with a legible `"failed to build binary"` signature, the sandbox's parent process would
read a whole test suite's stdout/stderr as if it were the sandboxed command's own output, and
whatever real command was supposed to run there would never execute — a misattribution that could
look like a confusing engine-output parse failure rather than a re-exec collision, with no
obvious signature to grep CI logs for the way `ISSUE-163`'s defect had one.

## References

- `ISSUE-163` — origin issue. This issue is the sibling gap its own investigation surfaced and
  explicitly declined to fix in-lane (`PLAN-ISSUE-163`, TASK-005, item 4: "A real, unmeasured
  adjacent hole — surface it, do not fix it here").
- `PLAN-ISSUE-163` — the plan whose TASK-005 mandated filing this issue, with the open question
  stated as an open question, not a confirmed defect.
- `cmd/backstop/sandbox_helper_testmain_guard_test.go` —
  `TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain`, the roster pin `ISSUE-163`
  added; its derivation ("imports `pkg/packval`, or is `pkg/packval`") is exactly the predicate
  that makes a `TestMain`-less importer invisible to it, since the roster only ever inspects
  packages that already declare a `TestMain`.
- `pkg/pack/distribution/validator.go:60` (at time of writing) — `packval.NewPipeline(packDir,
  ...).Run()`, the real pipeline invocation confirming the package is packval-reaching.
- `pkg/pack/engine/import_cycle_test.go`, `pkg/pack/engine/binding_test.go` — the two files
  confirming `pkg/pack/engine` imports `pkg/packval`.
- `pkg/packval/main_test.go` — the correct `TestMain` precedent both `ISSUE-163`'s fix and this
  issue's eventual fix (if the gap proves live) would mirror.
- `ISSUE-161` — precedent in this repo for filing `type: question` rather than `bug` when the
  underlying live-exploitability of a known structural gap is genuinely unmeasured.

### Existence-in-world check

Performed 2026-08-18 before authoring: `grep -ril` over `issues/` and `bundles/` for `TestMain`
and `packval` matched no open issue or bundle charter covering this specific gap (the sandbox
guard existing in packages that declare no `TestMain` at all). `ISSUE-163` itself was the closest
hit, and is this issue's named sibling/origin, not a duplicate — its scope is the one collision
it fixed, explicitly excluding this gap by its own plan's scope fences.
