---
name: arch-pack-forbids-gate-to-engine
description: "pkg/gate may NOT import pkg/pack/engine — the arch pack's package-boundaries rule blocks it even though there is no Go import cycle; declare the string locally + a behavioral lockstep test"
metadata:
  type: project
---

`pkg/gate` production code may **not** import `pkg/pack/engine`. `backstop-ai/backstop-core-architecture`
declares `gate: { mayDependOn: [artifact, check, config, pack, waiver] }` in
`architecture/backstop-core.yml`; `pack_engine` is a separate component and is deliberately
excluded. The violation surfaces in `pack_engines` as
`backstop-ai/backstop-core-architecture/package-boundaries — component gate may not import ...`.

**Why:** "no Go import cycle" and "allowed by the architecture" are DIFFERENT questions, and a plan
can check the first and confidently assert the second. PLAN-ISSUE-118 sharp edge 8 did exactly that
("pkg/gate may import pkg/pack/engine; there is no cycle") and was wrong — it cost a mid-lane
escalation. `pkg/gate` already imports `pkg/pack`, which makes the engine import look free.

**How to apply:** when a plan tells you to import across packages "because there is no cycle",
check `architecture/backstop-core.yml` first. If the edge is not declared, do NOT add the import
and do NOT amend the pack unilaterally (it lives in an external repo, needs a version bump + relock,
and backstop.yml/backstop.lock are usually dirty from sibling lanes).

The sanctioned in-repo pattern is already established: `pkg/gate/traceability_polarity.go` declares
its own `DimensionSubstantiveness`/`DimensionCoverage`/`DimensionContracts` STRING constants that
"join directly against a DECLARED gate_type" across the same boundary, while `cmd/backstop/gate.go`
does the `engine.ParseGateType` bridging on the CLI side where the dependency IS allowed. Follow it:
declare the spelling locally, then buy back what the import would have given you with a
**behavioral lockstep test** — drive real `engine.GateType` values through the real consumer and
assert the routing decision. That is a stronger pin than a shared constant reference, because it
also catches a wrong-type match, not just a rename.

**Test files are exempt** (`excludeFiles: ^.*_test\.go$` in the same config), which is precisely
what lets the lockstep guard import the enum where production code may not. Do not conclude from
"a `_test.go` in pkg/gate already imports engine" that production may.

Related: [[project_gate_scope_entry_surfaces_pack_false_positives]], [[feedback_never_stash_shared_tree]].
