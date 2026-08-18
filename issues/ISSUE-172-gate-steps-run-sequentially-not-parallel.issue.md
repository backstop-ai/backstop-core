---
title: "Gate Steps Run Sequentially Not Parallel"
schema_version: issue/v1

issue:
  id: ISSUE-172
  title: "Gate Steps Run Sequentially Not Parallel"
  type: technical-debt
  status: open
  created: "2026-08-18"

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
