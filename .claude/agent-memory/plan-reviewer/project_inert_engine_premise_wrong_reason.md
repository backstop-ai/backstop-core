---
name: inert-engine-premise-wrong-reason
description: "A plan's 'no live in-repo victim' premise usually cites the wrong reason — check rule bindings AND fixture presence separately; packval phase3 only dispatches RunEngine from inside fixture loops"
metadata:
  type: project
---

When a plan justifies "there is no dogfood repro / no live in-repo victim today" for a
packval or engine-dispatch defect, verify the STATED REASON separately from the
CONCLUSION. The conclusion is often right while the reason is false.

**Why:** PLAN-ISSUE-144 (2026-08-17) wrote "exactly ONE installed binding declares
`stdout_artifact` — `go-coverage` — and no rule declares that engine." A rule DOES
declare it (`.backstop/packs/backstop-ai/go-toolchain/pack.yml`, `- id: go-coverage /
engine: go-coverage`). What actually makes it inert is that go-toolchain declares NO
fixtures at all, and `pkg/packval/phase3.go` only reaches `executor.RunEngine` from
inside `claim.Fixtures.Positive` / `claim.Fixtures.Negative` loops. The correct reason
is also a FRAGILE one — adding a single fixture to any rule bound to such an engine
makes packval dispatch it immediately — which is a different (and stronger) argument
than "nothing binds it."

**How to apply:** for any "no victim / no repro" premise, run two independent greps —
(a) does any rule bind the engine, (b) does that rule declare fixtures — and check the
dispatch call sites to see which one is actually load-bearing. Same discipline as
[[base-registry-binding-overrides-fixture]] and [[census-through-real-parser]]: the
plan's label is not the measurement.

Related mechanic worth remembering: `pkg/packval`'s `RunEngine` honors far less of
`engine.EngineBinding` than `cmd/backstop/pack_gate.go`'s `runFindingsEngine` does.
As of 2026-08-17 packval had zero references to `CrashGuard` or `StrictSarif` (ISSUE-140
covered never-started, ISSUE-141 Convert, ISSUE-144 StdoutArtifact, ISSUE-143 the
consolidation). When reviewing any packval-vs-gate drift plan, diff the two functions'
honored-field sets and ask which unhonored fields the residuals section records.
