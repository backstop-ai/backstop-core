---
title: "Doctor Findings Engine Tool Blindspot"
schema_version: issue/v1

issue:
  id: ISSUE-134
  title: "Doctor Findings Engine Tool Blindspot"
  type: bug
  status: open
  created: "2026-08-15"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# Doctor findings-engine tool blindspot

## Problem

`backstop doctor`'s toolchain-runs check (`checkToolchainRuns`, `cmd/backstop/doctor_checks.go:227`)
verifies toolchain health by executing pack-declared entrypoints through the shared
`packEntrypointProber` (`cmd/backstop/pack_entrypoint_prober.go`). `Probe` selects bindings by walking
`manifest.Engines` directly and keeping only `binding.GateType == engine.GateTypeTest || GateTypeBuild`
(`pack_entrypoint_prober.go:126`) — by the prober's own doc comment, "Selection is by STAGE, never by
tool," naming only backstop's test/build kill-chain stages.

Findings-engine tools (the ones `pack_engines` dispatches at gate time — semgrep, ast-grep, and any
future pack-declared scanner) are never selected. They are resolved through a completely different
path at gate time: `dispatchPackEngines` (`cmd/backstop/pack_gate.go:267`) looks up each rule's engine
through `resolveEngineRegistry(manifest).Lookup(engineName)` (`pack_gate.go:71,282`) — a merged
built-in + pack registry keyed by engine name, not a `GateTypeTest`/`GateTypeBuild` walk over
`manifest.Engines`. The prober has no analog of this registry lookup at all.

The result: a project can run `backstop doctor`, see every check pass (including "pack-declared
test/build entrypoints execute"), and still have a findings-engine tool entirely missing from PATH.
Doctor gives no signal. The next `backstop gate` run hits exactly the failure mode ISSUE-112
describes — a missing tool with no CrashGuard silently produces empty SARIF, `pack_engines` PASSES,
and downstream joins (e.g. `test_substantiveness`) starve on the empty finding set, producing
misleading violations attributed to the wrong cause. ISSUE-112 fixes the GATE's silent-pass behavior
once a tool goes missing mid-flight; this issue is that `backstop doctor` — whose entire purpose is
proactive, pre-flight health verification — has no check that would have caught the same missing tool
*before* the gate run, even though doctor already executes commands to verify toolchain health for
test/build. As of this writing ISSUE-112 is `status: open` with a plan in `draft`
(`plans/PLAN-ISSUE-112-engine-tool-missing-silent-vacuous.plan.yml`) — not yet delivered.

This is not a duplicate of ISSUE-112: ISSUE-112's surface is the gate's own verdict computation when a
tool is absent; this issue's surface is doctor's diagnostic coverage — a different artifact, a
different code path (`packEntrypointProber` vs. `dispatchPackEngines`/`resolveEngineRegistry`), and a
different moment in the workflow (pre-flight vs. mid-gate).

I also checked whether this belongs under DIR-032 (gate-verdict-honesty), since it shares the "a
diagnostic surface doesn't catch a real problem class" theme and cites ISSUE-112 as a sibling source.
DIR-032's own scope description is explicit: "a gate step computes a result internally but reports the
wrong verdict about it" — every one of its thirteen members is a defect in what `backstop gate` itself
reports. This issue is not a gate-step misreport; it is a `backstop doctor` coverage gap in an entirely
separate diagnostic surface that never runs `backstop gate` at all. Leaving the directive slotting call
to backlog-pm rather than forcing this into DIR-032.

## Two real constraints, found while reading the code

1. **Shared-prober blast radius into `backstop init`.** `packEntrypointProber` has exactly two callers:
   doctor's `checkToolchainRuns` and `backstop init`'s toolchain step (`init_toolchain.go`, shipped
   under SPEC-069). The prober's own doc comment calls this out directly: "Two independent copies of a
   pack-command execution path is exactly what SPEC-069 REQ-011 forbids." Any change to `Probe`'s
   selection logic — widening it to findings-engine bindings — lands in both callers at once unless the
   fix introduces a new, additive selection mode that `init` does not opt into. A same-file edit to the
   existing `GateType` filter would silently change what `backstop init` verifies too, an unreviewed
   behavior change riding along on a doctor-scoped fix.
2. **No existing report vocabulary for a missing findings-engine tool.** Doctor's outcome→status
   mapping (`describeEntrypointProbe`, `doctor_checks.go`) was designed for test/build entrypoints:
   pass, unstartable ("fail" + "setup the consumer still owes"), exited-nonzero ("fail", verbatim
   output), refused (allowlist gate). None of these map cleanly onto "this findings-engine tool is
   absent." ISSUE-112 itself distinguishes two tool-presence mechanisms with different implications —
   assume-present tools (fail-loud by design) versus provision-pinned tools (ast-grep, semgrep — pinned
   to the trust allowlist but never actually auto-installed by any code path, per ISSUE-112's own
   finding). Whether a missing findings-engine tool should doctor-fail, doctor-warn, or doctor-skip
   likely depends on which of those two mechanisms governs that tool, mirroring ISSUE-112's own
   distinction — this needs its own design pass, not a copy-paste of the test/build vocabulary.

## Direction

Not prescribing a fix mechanism — this is real design work. Open questions a plan needs to resolve:

- Does doctor gain a new, dedicated check (e.g. "pack-declared findings-engine tools are present"),
  or does `checkToolchainRuns` widen to cover both stage-selected and engine-registry-selected
  bindings under one result?
- Does the underlying walk change at all, or does a new selection path get added alongside the
  existing `GateType`-filtered one specifically so `backstop init` is provably unaffected?
- Should a findings-engine tool probe actually *execute* the tool (mirroring the toolchain-runs
  approach) or just check presence/resolvability, given findings-engine tools don't have a
  "successful run with no findings" outcome that's meaningfully distinguishable from "tool absent"
  without running a real scan against real content?
- What is doctor's fail/warn/skip disposition for a missing findings-engine tool, and does it depend
  on assume-present vs. provision-pinned status the way ISSUE-112 treats the gate side?

## Notes

- Sibling: ISSUE-112 (`status: open`, plan `draft`) — fixes the gate's silent-pass verdict when a
  findings-engine tool is missing mid-run. This issue is upstream of that failure mode: doctor never
  had a chance to catch it before the gate run.
- Considered DIR-032 (gate-verdict-honesty) as a directive home; ruled out — DIR-032 is scoped to
  `backstop gate`'s own verdict computation, not `backstop doctor`'s diagnostic coverage. Left for
  backlog-pm to slot.
- Read in full: `cmd/backstop/pack_entrypoint_prober.go` (`Probe`, selection-by-stage doc comment),
  `cmd/backstop/doctor_checks.go` (`checkToolchainRuns`, `describeEntrypointProbe`), `cmd/backstop/
  pack_gate.go` (`dispatchPackEngines`, `resolveEngineRegistry`), `cmd/backstop/init_toolchain.go`
  (second caller, confirming shared-prober blast radius).
