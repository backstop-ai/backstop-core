---
title: "Packval Importing Packages Missing Testmain Guard"
schema_version: issue/v1

issue:
  id: ISSUE-164
  title: "Packval Importing Packages Missing Testmain Guard"
  type: question
  status: closed
  created: "2026-08-18"
  closed: "2026-08-19"

complexity:
  scope: isolated
  uncertainty: novel
  risk: safe

resolved-by: ISSUE-180
---

# Packval Importing Packages Missing Testmain Guard

## Resolution

Both halves of this open question are now settled, closed via `resolved-by: ISSUE-180` rather than
a backing plan of this issue's own (this issue never had a fix in-lane; `PLAN-ISSUE-180` answered
it as a side effect of fixing its sibling).

**The `pkg/pack/distribution` half — CONFIRMED LIVE, then FIXED.** `ISSUE-180` traced a real test
(`TestInstallContractsLocalPack_InstallsWithSuppliedCommand`) through to real sandboxed dispatch on
Linux via this package's missing `TestMain`, exactly the mechanism this issue predicted.
`PLAN-ISSUE-180` fixed it: a `TestMain` mirroring `pkg/packval/main_test.go`'s pattern, verified on
real Linux CI (run `32314302525`, commit `9f4763b`, `pass: true`, `total_violations: 0`).

**The `pkg/pack/engine` half — was NEVER LIVE, because the premise was FALSE.** This issue's
original claim that `pkg/pack/engine` imports `pkg/packval` (via `import_cycle_test.go` and
`binding_test.go`) does not hold up under real AST parsing (an actual `*ast.ImportSpec`, not a
grep). The `grep -rl "backstop-core/pkg/packval"` this issue's investigation used matched
FORBIDDEN-IMPORT STRING LITERALS inside those two files — leaf-invariant tests
(`TestEngineBinding_NoImportCycle`, `TestEngine_NoForbiddenImports`) that assert packval's
ABSENCE from `pkg/pack/engine`'s transitive dependencies, not its presence. The grep matched the
exact test proving the opposite of what it implied. `pkg/pack/engine` was never a packval-importing
package and this issue's premise for including it was wrong from the start — see the corrected
sites below, kept rather than deleted so the mistake (and how it was caught) stays legible.

**The structural-guard generalization this issue also raised is delivered.**
`cmd/backstop/sandbox_helper_testmain_guard_test.go`'s
`TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain` no longer skips a packval-reaching
package that declares no `TestMain` at all (the `if pkg.testMain == nil { continue }` skip this
issue's References section named is gone from membership derivation; absence is now a loud, named
failure at a new STEP 3a). Measured at zero cost and zero exemptions: the generalized predicate
flagged exactly `pkg/pack/distribution` and nothing else, including confirming `pkg/pack/engine`
is not a member — the same corrected fact stated above, re-derived independently by the check
itself rather than by argument. Full record: `PLAN-ISSUE-180`, "JUDGMENT CALL 1".

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

Two such packages were BELIEVED to exist today:

- **`pkg/pack/distribution`** — specifically `validator.go`, which runs the real packval
  pipeline (`packval.NewPipeline(packDir, ...).Run()`, confirmed at `validator.go:60` at time of
  writing). This package has no `TestMain`.
- **`pkg/pack/engine`** — `import_cycle_test.go` and `binding_test.go` both import
  `pkg/packval`. This package also has no `TestMain`.
  > **CORRECTED 2026-08-19 (`ISSUE-180`/`PLAN-ISSUE-180`): this claim is FALSE.** Real AST
  > parsing shows `pkg/pack/engine` imports no such thing. The two files named above are
  > leaf-invariant tests (`TestEngineBinding_NoImportCycle`, `TestEngine_NoForbiddenImports`)
  > asserting packval's ABSENCE from `pkg/pack/engine`'s transitive dependencies — the `grep -rl
  > "backstop-core/pkg/packval"` this claim traced to matched FORBIDDEN-IMPORT STRING LITERALS
  > inside those tests, not a real import declaration. See "## What is, and is not, known" below
  > for the methodology bug this came from.

Neither package has a `TestMain` at all, so if any test in either one can reach real sandboxed
dispatch on Linux, the mechanism is the same class of collision `ISSUE-163` describes, in a
quieter shape: the re-exec target becomes that package's test binary, Go's **default** generated
`TestMain` runs (there being no custom one to intercept), `m.Run()` executes the whole test
suite, and the sandbox's parent process reads the entire suite's output as if it were the
sandboxed command's own output — rather than the build-failure crash `ISSUE-163`'s defect
produced, which at least failed loudly and visibly. A silent misattribution of a whole suite's
output is arguably worse than a loud crash, precisely because it might not look broken.
> **This paragraph's scope was CORRECTED 2026-08-19: only `pkg/pack/distribution` was ever a real
> member of this set** — see the correction above. `ISSUE-180` confirmed it live and
> `PLAN-ISSUE-180` fixed it; `pkg/pack/engine` was never exposed to this mechanism at all.

## What is, and is not, known

**Known:** both packages import `pkg/packval` and neither declares a `TestMain`. This was
confirmed by direct inspection during the `ISSUE-163`/`PLAN-ISSUE-163` investigation
(`grep -rl "backstop-core/pkg/packval" --include="*.go" cmd pkg tests`, reduced to the
directories with no `func TestMain` declaration).

> **CORRECTED 2026-08-19 (`PLAN-ISSUE-180`): the methodology above is the bug.** A bare
> `grep -rl "backstop-core/pkg/packval"` matches the import path as a STRING LITERAL anywhere in a
> file — including inside a list of FORBIDDEN imports a test asserts must be ABSENT, which is
> exactly what `pkg/pack/engine`'s `import_cycle_test.go` and `binding_test.go` contain. It is not
> a substitute for parsing a real `*ast.ImportSpec`. Re-run through the real predicate
> (`cmd/backstop/sandbox_helper_testmain_guard_test.go`'s own `scanGoPackages`), the
> packval-reaching set is exactly `cmd/backstop`, `pkg/packval`, and `pkg/pack/distribution` —
> `pkg/pack/engine` is not a member. Only the `pkg/pack/distribution` half of "known" above was
> ever true. Never decide packval-reachability from a grep for the import path string again.

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

> **RESOLVED 2026-08-19.** `pkg/pack/distribution`: traced and confirmed live by `ISSUE-180`
> (`TestInstallContractsLocalPack_InstallsWithSuppliedCommand` reaches real sandboxed dispatch on
> Linux), then fixed by `PLAN-ISSUE-180` and confirmed on real Linux CI. `pkg/pack/engine`: there
> is no test to trace, because the package was never a packval importer — the premise was false
> (see the correction above). Both halves are closed; see "## Resolution" at the top of this issue.

## Why this is filed as a question, not a bug

The gap in coverage (a `TestMain`-less packval-importing package is invisible to `ISSUE-163`'s
guard and its pin) is a fact. Whether that gap is *live* — whether it can actually be exploited by
a real test run on Linux, causing real misattributed output rather than staying a theoretical
shape — is unmeasured. Filing this as a confirmed `bug` would overstate certainty the
investigation doesn't have (per `ISSUE-161`'s precedent for the same distinction in this repo:
question when the underlying live-exploitability is genuinely unknown, bug when it's confirmed).
If investigation confirms a real test in either package does reach sandboxed dispatch on Linux,
this should be re-filed or promoted to `bug` at that point — not before.

> **UPDATE 2026-08-19.** For `pkg/pack/distribution`, exactly that happened: `ISSUE-180` was that
> promotion to `bug`, filed once the investigation confirmed live exploitability. This issue
> stays `type: question` at close rather than being retroactively relabeled — the filing
> discipline it describes worked as designed.

## Impact if confirmed live

Same failure class as `ISSUE-163`, but quieter and arguably worse to diagnose: instead of a
crash with a legible `"failed to build binary"` signature, the sandbox's parent process would
read a whole test suite's stdout/stderr as if it were the sandboxed command's own output, and
whatever real command was supposed to run there would never execute — a misattribution that could
look like a confusing engine-output parse failure rather than a re-exec collision, with no
obvious signature to grep CI logs for the way `ISSUE-163`'s defect had one.

> **UPDATE 2026-08-19.** This impact materialized for `pkg/pack/distribution` exactly as
> described — `ISSUE-180` traced the misattributed-suite-output signature in real Linux CI — and
> was fixed by `PLAN-ISSUE-180`. It never applied to `pkg/pack/engine`, which was not exposed to
> this mechanism (see correction above).

## References

- `ISSUE-163` — origin issue. This issue is the sibling gap its own investigation surfaced and
  explicitly declined to fix in-lane (`PLAN-ISSUE-163`, TASK-005, item 4: "A real, unmeasured
  adjacent hole — surface it, do not fix it here").
- `PLAN-ISSUE-163` — the plan whose TASK-005 mandated filing this issue, with the open question
  stated as an open question, not a confirmed defect.
- `cmd/backstop/sandbox_helper_testmain_guard_test.go` —
  `TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain`, the roster pin `ISSUE-163`
  added; its derivation ("imports `pkg/packval`, or is `pkg/packval`") was exactly the predicate
  that made a `TestMain`-less importer invisible to it, since the roster only ever inspected
  packages that already declared a `TestMain`. **STALE, NOW DELIVERED (2026-08-19):**
  `PLAN-ISSUE-180` closed exactly this gap — the roster no longer skips a packval-reaching package
  with no `TestMain` at all (new STEP 3a). This reference described the defect this issue was
  tracking; the defect is fixed.
- `pkg/pack/distribution/validator.go:60` (at time of writing) — `packval.NewPipeline(packDir,
  ...).Run()`, the real pipeline invocation confirming the package is packval-reaching.
- `pkg/pack/engine/import_cycle_test.go`, `pkg/pack/engine/binding_test.go` — the two files
  confirming `pkg/pack/engine` imports `pkg/packval`. **FALSE AND INVERTED (corrected
  2026-08-19):** these two files confirm the OPPOSITE — `TestEngineBinding_NoImportCycle` and
  `TestEngine_NoForbiddenImports` assert packval's ABSENCE from `pkg/pack/engine`'s transitive
  dependencies. `pkg/pack/engine` does not import `pkg/packval`; this citation was misread from a
  grep hit on a forbidden-import string literal, not a real import declaration.
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
