---
title: "Distribution Testmain Missing Sandbox Guard"
schema_version: issue/v1

issue:
  id: ISSUE-180
  title: "Distribution Testmain Missing Sandbox Guard"
  type: bug
  status: open
  created: "2026-08-19"

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate
---

# Distribution Testmain Missing Sandbox Guard

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
- **`pkg/pack/engine` — still unconfirmed.** Nothing in tonight's investigation traced a specific
  test in that package through to real sandboxed dispatch on Linux. `ISSUE-164`'s open question
  stands for this package; this issue does not extend to it and should not be read as closing that
  half.
- **`pkg/gate` — checked tonight, found NOT exposed.** The task that produced this issue asked
  whether `pkg/gate` (also `TestMain`-less) carries the same risk. `grep -rl
  "backstop-core/pkg/packval" --include="*.go"` across the whole module returns exactly four
  directories: `cmd/backstop`, `pkg/pack/distribution`, `pkg/pack/engine`, and `pkg/packval` itself.
  `pkg/gate` does not import `pkg/packval` at all, by any file, so it cannot become a re-exec
  target through this mechanism — not "no evidence found," but confirmed absent from the import
  graph this defect requires.

Recommend narrowing `ISSUE-164` to just `pkg/pack/engine` (or closing it with a note pointing here
for the `pkg/pack/distribution` half) — left as a recommendation, not done here, since this session
is scoped to authoring this issue, not hand-editing another one.

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
  confirmation for `pkg/pack/distribution` specifically; `pkg/pack/engine` remains open under
  `ISSUE-164`, unconfirmed by this investigation.
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
- `cmd/backstop/sandbox_helper_testmain_guard_test.go` — the existing structural guard;
  `scanGoPackages` + `TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain`'s
  `if pkg.testMain == nil { continue }` is the blind spot this issue's "Structural facet" section
  describes and hands to `ISSUE-164`.
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
