---
name: shared-cache-seam-wiring
description: cross-step cache/state plans in cmd/backstop must thread through the dispatchPackEnginesFn seam, not a direct dispatchPackEngines call
metadata:
  type: project
---
The gate's pack_engines (test/findings) dispatch does NOT call dispatchPackEngines
directly — it goes through `resolveDispatchPackEngines()` / the package-level
`dispatchPackEnginesFn` seam (cmd/backstop/pack_gate.go:256, called at gate.go:820),
which ~15 tests spy on. The coverage path is different: coverageRecordsProducer calls
dispatchPackCoverage DIRECTLY (no seam).

**Why:** any plan that threads per-gate state (e.g. ISSUE-068 shared-run cache to
de-dup test+coverage runs) into "both dispatch call sites" must reckon with this
asymmetry. The cache reaches the coverage step cleanly (direct call) but the test
step only via the fixed-signature seam. A plan claiming "keep seam signatures stable"
AND "thread the cache into the pack_engines step" is self-contradictory: the cache
can't reach the seam-routed test dispatch without either widening the seam type
(breaks ~15 spies + 3 call sites) or bypassing the seam (loses the spy point). If it
never reaches the test step, the test step still runs twice — the fix is vacuous.

**How to apply:** when reviewing a cmd/backstop plan that injects per-invocation
state into engine dispatch, grep for resolveDispatchPackEngines/dispatchPackEnginesFn.
If the plan names only dispatchPackEngines/dispatchPackCoverage as direct calls,
flag the missing seam. Instance of [[integration_gap]] / [[project_dispatch_consumer_edges]].
