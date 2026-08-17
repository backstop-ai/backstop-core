---
name: shared-helper-collapse-blast-radius
description: A plan editing a branch inside a shared helper (runFindingsEngine, a *Fn seam) changes EVERY production caller — grep the seam's call sites; plans name only the one the issue mentions
metadata:
  type: project
---

When a plan's fix is "collapse/alter a branch inside function F", grep every PRODUCTION
call site of F (and of any `resolveXFn()` seam that returns it) before accepting the
plan's blast-radius statement. Plans routinely say "that is the entire behavioral change"
while naming only the caller the source issue happened to mention.

**Why:** Caught in PLAN-ISSUE-091 (2026-08-16). The fix collapsed the scope branch in
`cmd/backstop/pack_gate.go:runFindingsEngine` so any non-nil scope gets an explicit file
list. `resolveDispatchPackEngines()` has THREE production callers in `cmd/backstop/gate.go`:
the `pack_engines` step (~:978, the one the plan addressed), `buildTestSubstantivenessStep`
(~:1292, **same `activeScope`** — so its `--all` dispatch also flipped, and newly acquired
`excludeTestdataPaths` pruning that drops the substantiveness pack's own
`pkg/gate/testdata/substantiveness-pack/fixtures/**`), and the contract engine (~:1647,
`GateScopeModeFile`, genuinely unaffected). The plan predicted nothing for the second, so
its final `gate --all` task would have hit the flip blind with no expected-vs-stop-and-report
guidance.

**How to apply:** `grep -n "<funcName>\|<funcName>Fn" <pkg>/*.go | grep -v _test.go`.
For each caller, determine which scope/mode/arg it passes — callers already on the
"new" side of the branch are unaffected; callers sharing the arg with the named consumer
are in blast radius and need a predicted direction in the plan. Complements
[[project-dispatch-consumer-edges]] (which is about DAG build edges, not runtime fanout)
and [[project-fail-loud-consumption-path]].
