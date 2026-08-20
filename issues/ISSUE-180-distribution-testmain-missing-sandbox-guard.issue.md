---
title: "Distribution Testmain Missing Sandbox Guard"
schema_version: issue/v1

issue:
  id: ISSUE-180
  title: "Distribution Testmain Missing Sandbox Guard"
  type: bug
  status: closed
  created: "2026-08-19"
  closed: "2026-08-19"

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate

delivered_by: PLAN-ISSUE-180
---

# Distribution Testmain Missing Sandbox Guard

## Resolution

Delivered by `PLAN-ISSUE-180` (commit `9f4763b`). Added `pkg/pack/distribution/main_test.go`, a
`TestMain(m *testing.M)` in `package distribution_test` identical in shape to
`pkg/packval/main_test.go`'s existing pattern: its first statement is the error-checked
`if err := packval.MaybeRunSandboxHelper(); err != nil { ...; os.Exit(126) }`, followed by
`os.Exit(m.Run())`. This closes the gap directly — `pkg/pack/distribution`'s test binary now
recognizes `BACKSTOP_SANDBOX_HELPER_SPEC` on Linux instead of falling through to Go's default
generated test main and recursively rerunning the whole suite in the pack's scratch directory.

The same lane also generalized `cmd/backstop/sandbox_helper_testmain_guard_test.go`'s
`TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain` (name kept byte-identical — it is
`PLAN-ISSUE-163`'s mandated name). The roster's membership predicate no longer skips a
packval-reaching package that declares no `TestMain` at all: the `if pkg.testMain == nil {
continue }` skip moved out of membership derivation and became a new STEP 3a inside the loop
(membership itself now gates on `if !pkg.hasTestFile { continue }`), so absence of a `TestMain` is
now a loud, named failure instead of silent exclusion. This is the disposition reversal recorded
in the retraction note under "## Structural facet" below.

Pre-fix, the pins produced exactly ONE failure — STEP 3a only; the anti-vacuous floor at STEP 2
was green, because membership there is derived independent of `TestMain` presence:

    sandbox_helper_testmain_guard_test.go:226: pkg/pack/distribution reaches
    github.com/backstop-ai/backstop-core/pkg/packval and compiles a test binary but
    declares NO `func TestMain(m *testing.M)` (ISSUE-180)

A corroborating finding fell out of the generalized check's own run, unplanned: `pkg/pack/engine`
also has no `TestMain`, and STEP 3a did NOT flag it — because real AST-based membership (an actual
`*ast.ImportSpec` for `pkg/packval`, not a grep) shows `pkg/pack/engine` does not import
`pkg/packval` at all. Had it been a member, the pre-fix run would have red on TWO packages, not
one; it red on one. This re-derives "engine is not packval-reaching" from the predicate itself,
independent of any prior assertion — see the roster-claim corrections below.

Verified on real Linux CI, not locally — the Linux sandbox trampoline (`//go:build linux`) does
not compile on darwin, so this defect is structurally unreproducible here (see the plan's
verification ceiling). Run `32314302525`, commit `9f4763b`, `pass: true`, `total_violations: 0` —
the whole gate green. Scope was independently confirmed to include the fix files:
`pkg/pack/distribution/main_test.go`, `pkg/pack/distribution/sandbox_helper_gate_test.go`,
`cmd/backstop/sandbox_helper_testmain_guard_test.go`. The immediately-prior main commit (`22c7574`,
run `32307615655`) carries the pre-fix failure verbatim, making the green a one-commit-delta
measurement rather than an inference.

## Problem

`pkg/pack/distribution/contracts_local_install_test.go:
TestInstallContractsLocalPack_InstallsWithSuppliedCommand` fails on real Linux CI with `pack test
for .../packs/contracts failed: ... in phase3-fixtures: 14 validation error(s)` — byte-identical
before and after `PLAN-ISSUE-166`'s `-H -I` fix landed (`ISSUE-177`'s finding). Investigation
tonight (throwaway debug branch, real Linux CI, since cleaned up and deleted) itemized all 14
errors and found they are **not** `ISSUE-166`'s mechanism at all: they are a second, independent
re-exec collision in the same family as `ISSUE-163`, in a package `ISSUE-164` had already named
as at-risk but left unconfirmed.

### Root cause

`pkg/packval`'s Linux sandbox is a re-exec trampoline (`pkg/packval/sandbox_linux.go`): it spawns
`os.Executable()` — which under `go test` is the calling package's own compiled test binary —
with `BACKSTOP_SANDBOX_HELPER_SPEC` set. The re-exec'd process is expected to recognize the env
var via `packval.MaybeRunSandboxHelper()`, called as the FIRST STATEMENT of that package's own
`TestMain`, then install Landlock/seccomp and `execve` the real convert script
(`runSandboxHelper`, `pkg/packval/sandbox_linux_helper.go`).

That wiring is a documented family of exactly two correct members today — `cmd/backstop/main.go`'s
`runWith` (the shipped binary's half) and `pkg/packval/main_test.go`'s `TestMain` (the packval
test binary's half) — per `cmd/backstop/sandbox_helper_testmain_guard_test.go`'s own doc comment.
`pkg/pack/distribution` was never made a member: **it declares no `TestMain` at all.** Confirmed
by direct inspection — the package is `distribution_test`
(`pkg/pack/distribution/contracts_local_install_test.go:1`), and `grep -n "^func TestMain"` across
every `*_test.go` in the package returns nothing.

**Consequence.** When `TestInstallContractsLocalPack_InstallsWithSuppliedCommand` triggers the
real sandboxed convert via `distribution.NewPackvalValidator()` (`pkg/pack/distribution/
validator.go`, `packval.NewPipeline(packDir, ...).Run()`), the re-exec'd child is just
`distribution.test` with no custom `TestMain`. Go's **default** generated test main doesn't
recognize `BACKSTOP_SANDBOX_HELPER_SPEC` — it just runs `m.Run()` — so the re-exec'd process
**reruns the entire `pkg/pack/distribution` test suite from scratch**, in the scratch-copy
directory the sandbox trampoline set as its `cwd`. That recursive run's own tests fail fast
(`module root not found` and similar — cwd is off any `go.mod` ancestry), and the whole recursive
`go test` process exits 1. Go's testing framework writes all of that to **stdout**, not stderr —
so `foldHelperStderrIntoError` (`pkg/packval/sandbox_diagnostic.go`) sees empty stderr and reports
"wrote no diagnostic," while the actual (voluminous) recursive-suite output silently vanishes on
the discarded stdout side. This exactly reproduces the observed exit-status-1 / empty-diagnostic
signature — 14 times, once per fixture (12 ast-grep-dispatched signature fixtures + 2
grep-dispatched absence fixtures — i.e. every fixture in `packs/contracts`, confirming the
mechanism is fixture-shape-agnostic, not specific to grep or ast-grep dispatch).

### Why every `ISSUE-166`-named sibling cleared instead

Confirmed by inspection: every one of the roughly dozen structurally similar sibling tests
`ISSUE-177` named as having cleared with the `-H -I` fix lives in `cmd/backstop` or
`pkg/packval` — the exact two packages that DO carry the correct `TestMain` gate.
`TestInstallContractsLocalPack_InstallsWithSuppliedCommand` is the only one of that set living in
the one package missing it.

### Why invisible on darwin

`pkg/packval/sandbox_nonlinux.go`'s `MaybeRunSandboxHelper()` is an unconditional no-op stub;
`sandbox_linux.go` and `sandbox_linux_helper.go` are both `//go:build linux`-gated and never
compile on darwin. darwin's `sandbox-exec` needs no re-exec trampoline at all, so this defect
class is structurally undetectable off Linux — same darwin-invisible/Linux-only shape as
`ISSUE-166` and `ISSUE-168`.

### The fix (stated for context only — not implemented here)

Add a `TestMain(m *testing.M)` to `pkg/pack/distribution`'s test package, identical in shape to
`pkg/packval/main_test.go`'s existing one:

```go
func TestMain(m *testing.M) {
	if err := packval.MaybeRunSandboxHelper(); err != nil {
		fmt.Fprintf(os.Stderr, "backstop sandbox helper: %v\n", err)
		os.Exit(126)
	}
	os.Exit(m.Run())
}
```

Issues never carry the fix — this belongs in a plan (`PLAN-ISSUE-180`).

## Relationship to `ISSUE-164` — confirms, does not duplicate

`ISSUE-164` (`type: question`, still open) named exactly this gap in advance: it identified both
`pkg/pack/distribution` and `pkg/pack/engine` as packval-importing packages with no `TestMain`,
explicitly stated the live-exploitability was **unmeasured**, and said the right move on
confirmation was to "re-file or promote to bug at that point" rather than silently upgrade its own
status. This issue is that promotion, for one of its two named packages, done through direct
evidence rather than inference:

- **`pkg/pack/distribution` — confirmed live** by this investigation. A real test
  (`TestInstallContractsLocalPack_InstallsWithSuppliedCommand`) does reach real sandboxed dispatch
  on Linux through this package's missing `TestMain`, with the exact recursive-rerun mechanism
  `ISSUE-164` predicted and a real CI signature traced to it.
- **`pkg/pack/engine` — CORRECTED 2026-08-19 (`PLAN-ISSUE-180` review): confirmed NOT
  packval-reaching, not merely unconfirmed.** This issue originally left it as an open question
  inherited from `ISSUE-164`. Real AST parsing (an actual `*ast.ImportSpec` for `pkg/packval`, not
  a grep) shows `pkg/pack/engine` imports no such thing — its exposure is dead by construction,
  doubly pinned by `TestEngineBinding_NoImportCycle` and `TestEngine_NoForbiddenImports`. The
  original framing here, and the four-directory grep count below, both trace to `grep -rl
  "backstop-core/pkg/packval"` matching FORBIDDEN-IMPORT STRING LITERALS inside
  `import_cycle_test.go` and `binding_test.go` — tests that assert packval's ABSENCE from engine's
  transitive dependencies, not its presence.
- **`pkg/gate` — checked tonight, found NOT exposed.** The task that produced this issue asked
  whether `pkg/gate` (also `TestMain`-less) carries the same risk. **CORRECTED 2026-08-19: the
  grep-derived count below was wrong.** The real, AST-derived packval-reaching set is exactly
  THREE directories — `cmd/backstop`, `pkg/packval`, and `pkg/pack/distribution` — not the four
  originally reported (a `grep -rl "backstop-core/pkg/packval" --include="*.go"` also matched
  `pkg/pack/engine`, which is not actually a member; see above). `pkg/gate`'s absence stands
  either way: it does not import `pkg/packval` at all, by any file, so it cannot become a re-exec
  target through this mechanism — confirmed absent from the import graph this defect requires.

> **RETRACTED — 2026-08-19, `PLAN-ISSUE-180` review round 1.** The recommendation above (narrow
> `ISSUE-164` to just `pkg/pack/engine`, or close it with a note pointing here for the
> `pkg/pack/distribution` half) is superseded. `pkg/pack/engine` is not open at all — see the
> corrected bullet above — and the roster generalization `ISSUE-164` was tracking landed in
> `PLAN-ISSUE-180` itself (see the retraction note under "## Structural facet" below). `ISSUE-164`
> is recommended for closure via `resolved-by: ISSUE-180`, not narrowing.
>
> ~~Recommend narrowing `ISSUE-164` to just `pkg/pack/engine` (or closing it with a note pointing
> here for the `pkg/pack/distribution` half) — left as a recommendation, not done here, since this
> session is scoped to authoring this issue, not hand-editing another one.~~

## Structural facet — the guard's blind spot, deliberately left out of this issue's fix

`ISSUE-164` also already names the deeper generalization: `cmd/backstop/
sandbox_helper_testmain_guard_test.go`'s `TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain`
roster is hand-derived as "every package that imports `pkg/packval` **and already has a
`TestMain`**" (`scanGoPackages`, then `if pkg.testMain == nil { continue }` before the roster is
built) — confirmed by direct reading of that test tonight. A packval-importing package with **no**
`TestMain` at all is invisible to it by construction, which is exactly the shape of this defect:
`pkg/pack/distribution` had no `TestMain` for the roster to inspect, so nothing caught it.

This issue's fix (add the correct `TestMain` to `pkg/pack/distribution`) does not touch that
blind spot — a future new packval-importing package with no `TestMain` would still slip past the
roster silently. Generalizing the roster to flag a **missing** `TestMain` on any packval-importing
package (not just a malformed one on a package that already has one) is `ISSUE-164`'s territory,
not this issue's: `ISSUE-164` already exists as the tracking artifact for exactly this structural
gap, named it before this defect was confirmed, and covers `pkg/pack/engine` too, which this issue
does not touch. Folding the generalization in here would scope-creep this issue past its own
confirmed evidence. Left as a recommendation for `ISSUE-164`'s eventual plan, alongside verifying
`pkg/pack/engine`.

> **RETRACTED (partial) — 2026-08-19, `PLAN-ISSUE-180` review round 1.** The blind-spot
> DESCRIPTION above (the `if pkg.testMain == nil { continue }` skip, and its consequence for a
> `TestMain`-less packval-importing package) stays accurate — it is the exact defect this issue's
> fix closes. Only the DISPOSITION — scoping the generalization out to `ISSUE-164`, on the grounds
> that fixing it here would "scope-creep" — is retracted. `PLAN-ISSUE-180`'s first draft reasoned
> the same way and was corrected in review: re-measuring the packval-reaching set with real AST
> parsing (an actual `*ast.ImportSpec`, not `grep -rl` for the import path string) showed the
> generalization costs ~3 lines and flags exactly zero packages beyond `pkg/pack/distribution`
> itself — no forced unmandated fix, no exemption list; the cost/exemption concern this section
> raised turned out not to exist. The generalization landed in this lane
> (`cmd/backstop/sandbox_helper_testmain_guard_test.go`'s STEP 3a) rather than being deferred. This
> was a founder-visible decision made during the plan's review, not a silent reversal — see
> `PLAN-ISSUE-180`'s "JUDGMENT CALL 1" for the full record, including the false premise (the bad
> grep) that produced this section's original recommendation.

## Impact

`TestInstallContractsLocalPack_InstallsWithSuppliedCommand` (CLM-092,
`pkg/pack/distribution/contracts_local_install_test.go`) fails on every real Linux CI run that
exercises it — has done so since the test was introduced, independent of and unaffected by
`ISSUE-166`'s fix. Any other `pkg/pack/distribution` test that reaches real sandboxed dispatch
(none currently confirmed beyond this one, but not ruled out) would hit the identical collision.

## References

- `ISSUE-177` (`contracts-local-install-phase3-anomaly`) — origin/parent context. Its
  investigation is superseded by this issue's root cause: the 14 errors are not `ISSUE-166`
  residue, they are this defect. `ISSUE-177` should be left open with a note pointing here rather
  than closed outright — it opened as an unexplained CI anomaly with its own evidence trail
  (the two CI run IDs, the before/after comparison), and this issue's diagnosis is new evidence to
  attach to it, not a substitute for its own closure ceremony. Whether/when to close `ISSUE-177` is
  better decided once `PLAN-ISSUE-180` actually lands the fix and CI confirms the test goes green.
- `ISSUE-163` (`cmd-backstop-testmain-missing-sandbox-helper-check`) — the origin defect and fix
  pattern this issue's fix mirrors exactly (same collision shape, different package).
- `ISSUE-164` (`packval-importing-packages-missing-testmain-guard`) — named this exact gap in
  `pkg/pack/distribution` in advance, as an unconfirmed question. This issue is that question's
  confirmation for `pkg/pack/distribution` specifically. **Updated 2026-08-19:** `pkg/pack/engine`,
  `ISSUE-164`'s other named package, is no longer an open question — real AST parsing during
  `PLAN-ISSUE-180` confirmed it is not packval-reaching at all (dead by construction), and
  `ISSUE-164`'s roster-generalization half is delivered by this issue's fix (see "## Structural
  facet" above). `ISSUE-164` is recommended for closure via `resolved-by: ISSUE-180`.
- `ISSUE-166` (`contracts-pack-phase3-fixtures-fail-on-linux-ci`) — the `-H -I` fix whose landing
  is what `ISSUE-177` measured this test against; ruled out as this test's mechanism tonight.
- `ISSUE-168` (`sandbox-devnull-write-denied-breaks-idiomatic-scripts`) — same darwin-invisible,
  Linux-only sandbox-mechanism shape, different specific defect.
- `pkg/packval/main_test.go` — the correct `TestMain` precedent this issue's eventual fix mirrors.
- `pkg/pack/distribution/validator.go` — `packval.NewPipeline(packDir, ...).Run()`, the real
  pipeline invocation confirming the package is packval-reaching.
- `pkg/pack/distribution/contracts_local_install_test.go` — the failing test
  (`TestInstallContractsLocalPack_InstallsWithSuppliedCommand`, CLM-092) and its package
  declaration (`package distribution_test`, no `TestMain`).
- `cmd/backstop/sandbox_helper_testmain_guard_test.go` — the structural guard; `scanGoPackages` +
  `TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain`'s former `if pkg.testMain == nil {
  continue }` was the blind spot this issue's "Structural facet" section describes. **Updated
  2026-08-19:** that gap is now closed inside this same guard (STEP 3a, `PLAN-ISSUE-180`) rather
  than handed to `ISSUE-164` — see the retraction note on "## Structural facet" above.
- `pkg/packval/sandbox_linux.go` (`newSandboxHelperCommand`), `pkg/packval/
  sandbox_linux_helper.go` (`runSandboxHelper`), `pkg/packval/sandbox_diagnostic.go`
  (`foldHelperStderrIntoError`) — the mechanism: re-exec trampoline, helper takeover, and the
  stdout/stderr split that makes the recursive rerun's failure look like "wrote no diagnostic."

### Existence-in-world check

Performed 2026-08-19 before filing: searched `issues/` and `bundles/` for "TestMain", "sandbox
helper", "distribution", and "re-exec trampoline". Confirmed distinct from `ISSUE-163` (fixed a
different package, `cmd/backstop`, already landed), `ISSUE-165` (a sandbox ABI-probe test-naming
mismatch, unrelated mechanism), and `ISSUE-166` (a different validation-flag defect, already ruled
out as this test's cause by `ISSUE-177`'s own before/after CI comparison). Confirmed this is the
promotion `ISSUE-164` itself calls for on confirmation, not a duplicate of it — see "Relationship
to `ISSUE-164`" above for the specific boundary. No bundle charter references this surface.
