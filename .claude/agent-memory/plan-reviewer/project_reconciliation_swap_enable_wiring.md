---
name: reconciliation-swap-enable-wiring
description: Run-loop reconciliation passes (baseline/waiver) need enable+feed wiring at the cmd/backstop construction site, not just the pkg/gate swap; plans that scope only pkg/gate leave it dark in the real CLI
metadata:
  type: project
---
A gate reconciliation pass (computeBaselineResult / computeWaiverResult) has TWO
seams, and plans routinely scope only the first:
1. The SWAP in pkg/gate/gate.go Run loop (`if g.xEnabled && result.StepName==X { result = g.computeX(results) }`) — pkg/gate scope.
2. The ENABLE + FEED at the construction site cmd/backstop/gate.go: the `WithX(...)`
   Option is CALLED there (baseline: gate.go:114 `WithBaseline(...)` sets
   `g.baselineEnabled=true`), plus any runtime inputs the pass needs (for waivers:
   a LineReader over scope, `now`, and a Policy BUILT from installed packs'
   declared non-waivable sets — the CLM-027 "declared not hardcoded" mechanism).

**Why:** if a plan scopes only pkg/gate, the shipped `backstop gate` runs the step
as its DISABLED placeholder (emits "skipped"), so the subsystem is dark in the real
CLI even with every pkg/gate unit + a pkg/gate e2e green. A pkg/gate e2e test
constructs gate.New itself (can't import cmd/backstop), so it CANNOT prove the CLI
enables the pass. Phase-E `backstop gate` on the repo is vacuously green if the repo
has no committed token+finding to suppress.

**How to apply:** for any reconciliation-pass plan, require a task that file-scopes
cmd/backstop/gate.go (call the WithX option, construct the runtime inputs) AND a
test that drives the CLI-construction path. Also watch PIPELINE ORDER: waiver step
sits AFTER baseline in the step list (cmd/backstop/gate.go:707 baseline, 708 waiver),
so a REQ-013-style "active waiver satisfies the ratchet" claim only holds if waiver
subtraction precedes/feeds the baseline+ratchet eval — ratchet tests on synthetic
StepResults pass in isolation while the real order defeats them. Relates to
[[pack_provisioning_integration_gap]] family.
