---
title: "Gate Steps Run Sequentially Not Parallel"
schema_version: issue/v1
delivered_by: PLAN-ISSUE-172

issue:
  id: ISSUE-172
  title: "Gate Steps Run Sequentially Not Parallel"
  type: technical-debt
  status: closed
  created: "2026-08-18"
  closed: "2026-08-19"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# Gate Steps Run Sequentially Not Parallel

## Problem

`(*Gate).Run` (`pkg/gate/gate.go:136`) dispatches every gate step through a plain sequential Go
`for` loop:

```go
for _, stepFn := range g.steps {
    started := time.Now()
    result := stepFn(ctx)
    ...
    results = append(results, result)
    ...
}
```

(`pkg/gate/gate.go:150-170`). No goroutines, no `errgroup`, no `WaitGroup` — each `stepFn(ctx)`
call at line 152 blocks the loop from starting the next step until it returns. The kill chain
(`pack_lock_verification → artifact_validation → pack_engines → test_verification →
substantiveness → coverage → contracts → status_drift → requirement_traceability →
waiver_resolution → baseline_comparison → ledger_integrity`, per `docs/CODEBASE-MAP.md`) therefore
pays the FULL SUM of every step's wall-clock cost, not the max.

Two steps dominate that sum. Measured on a real CI run tonight (run 32151610956, its uploaded
`gate-report.json`): `pack_engines` took 629797ms (~10.5 min) and `coverage_threshold` took
612148ms (~10.2 min) in a single gate invocation — every other step combined took under 10
seconds. Run sequentially, those two alone account for essentially the entire gate's wall-clock
cost, back to back, when — per the founder tonight — they could in principle overlap.

## Structural evidence steps look independent (an inference, not a proven trace)

`StepFunc` (`pkg/gate/result.go:238`) has the signature `func(ctx context.Context) StepResult` —
each step closure takes only `ctx`, never the accumulated `results` from steps that ran before it.
That call shape means most steps, INCLUDING `pack_engines` and `coverage_threshold` specifically,
do not appear to consume each other's output.

The two steps that genuinely need earlier results are explicitly carved out of the main loop as
post-processing, not run inline with it:

- `waiver_resolution` — `g.computeWaiverResult(results, g.waiverRead, g.waiverPolicy,
  g.waiverNow)` (`pkg/gate/gate.go:157`), gated on `result.StepName == StepWaiverResolution`.
- `baseline_comparison` — `g.computeBaselineResult(results)` (`pkg/gate/gate.go:160`), gated on
  `result.StepName == StepBaselineComparison`.

Both read the accumulated `results` slice AFTER the ordinary `stepFn(ctx)` call returns, swapping
in a recomputed `StepResult` for that one step (`pkg/gate/gate.go:153-161`). That these two are
the ONLY steps special-cased this way is the evidence: every other step in `g.steps`, including
the two expensive ones, is dispatched through the identical `ctx`-only call at line 152 with no
comparable data dependency visible in `Run` itself.

**This is an inference from the call signature, not a full trace of every step's internal logic.**
A step could still have a hidden dependency `Run` never sees — shared filesystem state (e.g. two
steps reading/writing the same scratch file), shared external tool/process state, or an
order-sensitive side effect inside a step's own implementation. Confirming there is none of that
for `pack_engines` and `coverage_threshold` specifically is real investigation work this issue
does not do; it belongs to whatever plan takes this on.

`results = append(results, result)` (`pkg/gate/gate.go:163`) is itself not goroutine-safe. Running
steps concurrently would require real synchronization around that accumulation — a mutex-guarded
slice, a channel-based collector, or dispatching into a pre-sized indexed slice and only doing the
final ordered append single-threaded — not a bare parallel loop with the append left as-is.

## Impact

The gate is, per `CLAUDE.md`, "the primary enforcement checkpoint" — every push through CI runs
it (see ISSUE-099, which already tracks a related but distinct defect: CI invoking `backstop
gate` twice per push because there is no single-invocation way to get both the human table and
the `--json` output). The two issues compound multiplicatively rather than being alternatives:

- Today: two full sequential gate runs, each paying the full `pack_engines` +
  `coverage_threshold` sum (~20.7 min per run) plus the other steps — roughly 42 min worst case
  across both invocations.
- Fixing ISSUE-099's duplicate-invocation gap alone: one gate run, ~21 min.
- ALSO parallelizing the two dominant steps (this issue): one gate run's critical path drops from
  ~20.7 min (sum) to roughly max(10.5, 10.2) ≈ 10.5 min plus the sub-10-second remainder — call it
  ~11 min total.

Neither issue subsumes the other: ISSUE-099 is about invocation count, this issue is about
per-invocation critical path. Fixing both is what gets CI from the current ~42 min worst case down
to roughly ~11 min.

## Alternative worth weighing: CI-level parallel jobs instead of in-process concurrency

Changing `pkg/gate/gate.go`'s internal orchestration is not the only way to get most of this
speedup, and it is the harder, riskier change (it touches the gate orchestrator directly). Since
`pack_engines` and `coverage_threshold` are the two dominant steps, and CI already gates via
scoped `backstop gate` invocations (base-scoped today; file-scoped and all-scoped both exist as
flags), it may be possible to run two separate `backstop gate` invocations in PARALLEL CI JOBS —
one oriented at whatever dimensions `pack_engines` covers, one at coverage — instead of touching
`Run`'s dispatch loop at all.

This is flagged as an alternative for the plan to weigh, not a decision made here. It is unclear
whether the gate's dimensions can be cleanly split that way without also splitting or duplicating
whatever underlying `go test` dispatch the two steps may share (per ISSUE-068's finding that a
project's test and coverage paths can be two distinct engines each re-running the same suite,
resolved there by a PACK-side combined-run convention rather than a core mechanism) — a CI-level
split that does not also address that sharing could reintroduce the double-suite-run cost ISSUE-068
paid down. Whichever approach the plan picks needs to weigh implementation risk (in-process
concurrency in the enforcement checkpoint) against verification cost (splitting gate dimensions
across CI jobs without silently narrowing what gates).

## Investigation (2026-08-19)

This issue asked whether `pack_engines` and `coverage_threshold` — the two dominant gate steps —
are truly independent and therefore safe to parallelize. They are not. The independence inference
in the section above is **false**, and there is a second, separate finding underneath it about
what the two steps are actually doing.

### The independence inference is false: a hidden cross-step dependency

`cmd/backstop/gate.go:956-957` declares package-level `var collectedVerdicts []gate.Violation` and
`verdictEngineDeclared bool` inside `buildGateSteps`. They are **written** inside the `pack_engines`
step's own implementation, `packValidatorStep`, at `cmd/backstop/gate.go:1133-1134`. They are
**read** by a `TestVerdictSupplier` closure invoked by a **different** step, `test_verification`,
via `pkg/gate/step_testverify.go:391` (the supplier itself documented at `:365-377`, invoked at
`:424-427`).

This channel's safety today rests entirely on the two steps running in the current sequential
assembly order — documented only in a comment at `cmd/backstop/gate.go:947-950`, never enforced by
any test or synchronization primitive. If the two steps were naively parallelized (approach A from
the Problem section above, with only the `results` slice append made safe), the failure mode is
**not a crash** — it is silent enforcement loss. The two writes at `gate.go:1133-1134` are separate
statements (the slice, then the bool), so an unsynchronized concurrent reader can observe either of
two distinct bad states:

- **Path 1 (clean reorder, both zero-valued):** `verdictEngineDeclared == false` routes
  `test_verification` to a non-blocking `test_verification_verdict_capability_absent` advisory
  instead of the correct `critical` `mandated_test_failed` violation. The gate goes green over a
  genuinely failing mandated test, but it at least says something.
- **Path 2 (the race artifact — worse):** the bool is read `true` but the slice is read empty,
  since the two writes are unsynchronized and the Go memory model permits observing one without the
  other. This routes to an unqualified **pass**, with not even the advisory — indistinguishable
  from a healthy green.

### The two dominant steps are the same test suite, run twice

Separately, and more fundamentally: `pack_engines` and `coverage_threshold` are not two genuinely
independent ~10-minute workloads — they are the **same** full `go test ./...` suite, run twice. The
installed `go-toolchain` pack's `go-test` engine runs `go test` with no coverage flags; the
`go-coverage` engine's producer script independently re-runs `go test -coverprofile=cover.out ./...`
from scratch, with no freshness or reuse check anywhere in the script. The measured durations from
the Problem section (629797ms for `pack_engines` vs. 612148ms for `coverage_threshold`, a ~17.6s
delta attributable to `go build` and `golangci-lint` riding along in `pack_engines`) are consistent
with two whole-module test runs back to back, not two genuinely different workloads.

This is literally the same defect class `ISSUE-068` already found and fixed for other toolchains
— a "single-run convention" where the test command produces a coverage profile that the coverage
step then reuses instead of re-running the suite from scratch. `ISSUE-068` explicitly parked the
Go-specific follow-on as "not scoped here" ("the same combined-run convention... can be adopted by
`go-toolchain`... Not filed as a separate issue; noted for whoever next touches go-toolchain's
coverage path") and it was never picked up — `go-toolchain` sat at v1.6.0 across six releases with
no reuse check. A reference implementation already ships in production for `typescript-toolchain`
(v1.3.0): adopting the same convention there dropped `coverage_threshold` from ~174s to ~0.75s with
zero change in verdicts.

## Direction

Both original candidate approaches from the Problem/Alternative sections above are **declined**:

- **(A) in-process concurrency in the gate orchestrator** — declined for two independent reasons.
  First, it is unsafe as originally imagined: the hidden `collectedVerdicts`/`verdictEngineDeclared`
  dependency above is real, and neither a mutex-guarded `results` append nor a channel collector
  addresses it — both synchronize the wrong variable, leaving the silent-enforcement-loss race
  intact. Second, the `max(10.5, 10.2)` arithmetic that motivated the estimated speedup is unsound
  on its own terms: the two steps are not resource-independent — both saturate CPU running the same
  whole-module suite on a shared runner — so parallelizing them would not actually yield anywhere
  close to the naive `max()` estimate.
- **(B) splitting into two parallel CI jobs** — declined because it would reintroduce the
  double-suite-run "by construction," at roughly 2x today's compute (two runners, two independent
  whole-module `go test` invocations), and would require a dimension-subset flag core does not have
  — especially now that ISSUE-099 just consolidated CI down to exactly one gate invocation per push.

**(C) is recommended**: apply ISSUE-068's proven single-run convention to the external
`backstop-ai/go-toolchain` pack, so the `go-test` engine's command carries `-coverprofile=cover.out`
and the `go-coverage` producer reuses that profile when fresh instead of re-running the suite. This
reaches the same end-state speedup the issue originally projected for approach (A) — roughly the
gate's ~20.7-minute critical path dropping to ~10-11 minutes — at **half** today's compute (one
suite run instead of two), with zero change to the gate orchestrator itself (`pkg/gate/gate.go`
stays untouched), which is lower risk than either original option given the gate is "the primary
enforcement checkpoint."

This decision was pending founder sign-off as of the investigation above. Sign-off has since been
**granted** ("Yes I assumed that was what we would do anyway"), and implementation of (C) is now
in progress.

## Resolution

This issue is **not resolved by either approach it originally proposed.** Neither (A) in-process
concurrency in `(*Gate).Run` nor (B) parallel CI jobs shipped. `pkg/gate/gate.go` is untouched and
`.github/workflows/ci.yml` is untouched.

What actually shipped is approach (C) from the Direction section above: `backstop-ai/go-toolchain`
was republished at v1.7.0 (commit `fb2b947`), adopting a single-run convention — the `go-test`
engine's command now carries `-coverprofile=cover.out`, so the same `go test` invocation that used
to run once for `pack_engines` now also produces the coverage profile that `coverage_threshold`
reuses (via a stamp-and-reuse-if-fresh check in the producer scripts), instead of the `go-coverage`
engine re-running the full suite from scratch. This discharges the go-toolchain follow-on that
`ISSUE-068` explicitly parked and nobody had picked up.

**Measured result (local).** A before/after gate run on this workstation:

| Step | Before | After | Change |
|---|---|---|---|
| `pack_engines` | 318359ms | 405797ms | same order of magnitude |
| `coverage_threshold` | 294530ms | 2211ms | ~133x collapse |

`coverage_threshold` was 92% of `pack_engines`'s duration; it is now 0.5% of it. `pack_engines`
itself stayed roughly flat (its rise is attributable to a wider gate scope on the "after" run plus
the coverage instrumentation the combined run now carries, not a regression). The collapse is
confirmed non-vacuous: the 2211ms run produced real per-file coverage measurements (e.g. `49/55`,
`112/133`, `19/24`), i.e. it parsed a genuine whole-module profile rather than an empty one.

**Measurement scope — read before citing these numbers.** These are LOCAL numbers only, and the two
readings are not scope-identical: the "before" reading is diff-scoped (`14 changed files`) taken
while the installed pack was still v1.6.0; the "after" reading is a `gate --all` run at installed
v1.7.0. The ratio is the load-bearing signal, not the absolute deltas. **Real CI confirmation is
still outstanding as of this close** — no post-merge CI run has been observed at v1.7.0. The only
CI-grade measurement in evidence remains the pre-fix run this issue was authored against (CI run
32151610956: `pack_engines` 629797ms, `coverage_threshold` 612148ms). This issue closes on a
measured local collapse and a projected CI win, nothing more; capturing the next push's
`gate-report.json` and recording its two durations is an open obligation for whoever watches it.

**Standing constraint for any future in-process-concurrency attempt (CLM-002/CLM-003).** The
investigation found a real hidden cross-step ordering dependency that a naive parallelization of
(A)'s shape would not have caught: `cmd/backstop/gate.go` (around lines 956-957, written around
1133-1134) declares a package-level `collectedVerdicts`/`verdictEngineDeclared` channel, written by
the `pack_engines` step and read by a `TestVerdictSupplier` inside `test_verification`, with no
synchronization. Its safety today rests entirely on `pack_engines` running before
`test_verification` in the fixed sequential assembly order. If the two steps were ever run
concurrently, the failure mode would not be a crash — it is silent enforcement loss: either a
`critical` `mandated_test_failed` violation silently downgrades to a non-blocking advisory, or
(worse, under partial visibility of the unsynchronized writes) the step returns an unqualified pass
with not even the advisory, indistinguishable from a healthy green. This is now pinned by an
executable guard, `TestGateStepOrdering_PackEnginesPrecedesItsDependentSteps`, so this constraint
does not need rediscovering if in-process concurrency is revisited later.

**Recorded without absorbing:**

- `ISSUE-068` was updated with a short note recording this discharge (already done, as a separate
  edit) — it stays `closed` and carries no `resolved-by`/`delivered-by` pointer back to this issue,
  since the work was delivered as a byproduct of this issue's plan rather than tracked by ISSUE-068
  itself.
- `PLAN-ISSUE-066` carries `status: canceled` with `phases: []`, which reds `plan/phases-empty` in
  `artifact_validation` repo-wide. This is a known, already-tracked issue — `ISSUE-154`, filed
  2026-08-17, confirmed via existence-in-world check before this close. Not a duplicate filing.
- Two smaller items surfaced during implementation are owed follow-ons, judged too minor to warrant
  their own artifact right now, recorded here for visibility instead:
  - No guard exists asserting the `backstop-ai/go-toolchain` pack source and backstop-core's own
    go-toolchain test fixture stay in semantic agreement — the fixture is a deliberately frozen
    older snapshot (different name, version, and feature set), and "deliberate divergence" vs.
    "silent drift" are indistinguishable to the corpus today without a hand check.
  - A documented, bounded edge case in the coverage producer's stamp: a crash between the stamp
    write and the coverage step could leave a stamp on disk that a later coverage-only invocation
    honors against a matching-age `cover.out`. The window is one gate invocation and the
    consequence is a slightly-stale coverage number, never a suppression.

## References

- `pkg/gate/gate.go:136-170` (`(*Gate).Run`) — the sequential dispatch loop; line 150 is the `for`
  itself, line 152 the blocking `stepFn(ctx)` call, line 163 the non-synchronized `append`.
- `pkg/gate/gate.go:153-161` — the only two steps that read the accumulated `results` slice
  (`waiver_resolution`, `baseline_comparison`), both handled as post-loop swaps rather than inline
  with the ordinary dispatch, which is the structural basis for inferring the rest are independent.
- `pkg/gate/result.go:238` — `type StepFunc func(ctx context.Context) StepResult`, the call
  signature every step (including `pack_engines` and `coverage_threshold`) shares.
- `docs/CODEBASE-MAP.md` — the gate flow / kill-chain step ordering
  (`pack_lock_verification → artifact_validation → pack_engines → test_verification →
  substantiveness → coverage → contracts → status_drift → requirement_traceability →
  waiver_resolution → baseline_comparison → ledger_integrity`).
- CI run 32151610956, `gate-report.json` (downloaded artifact) — `pack_engines` 629797ms,
  `coverage_threshold` 612148ms, everything else combined under 10 seconds.
- `ISSUE-099-gate-cannot-emit-json-and-human-together.issue.md` — related but distinct: tracks CI
  invoking `backstop gate` twice per push (invocation count), not the per-invocation step
  ordering this issue tracks. The two compound multiplicatively; see Impact.
- `ISSUE-068-gate-double-runs-test-suite.issue.md` — prior finding that the gate's step loop is
  fully sequential with zero concurrency (corroborates the "Root cause" description reused here)
  and the precedent for why a naive shared-run mechanism at the core level can fail to fit real
  toolchains — relevant caution for the CI-level-parallel-jobs alternative above.
- `CLAUDE.md` — names the gate "the primary enforcement checkpoint," the basis for this issue's
  `risk: moderate` complexity rating.

### Existence-in-world check

Performed 2026-08-18 before authoring: `grep -ril` over `issues/` and `bundles/` for
"goroutine", "concurren", "errgroup", and "parallel" found no open issue or bundle charter
proposing gate-step parallelization. ISSUE-068 discusses the gate's sequential loop only as
corroborating evidence for a DIFFERENT defect (duplicate full-suite test runs from two engines,
already resolved pack-side) and does not propose parallelizing step dispatch. ISSUE-099 owns
invocation-count duplication, not per-invocation step ordering. No open issue or bundle already
owns this surface.
