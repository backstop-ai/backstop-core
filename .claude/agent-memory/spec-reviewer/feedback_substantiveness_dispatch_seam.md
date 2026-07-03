---
name: substantiveness-dispatch-seam
description: SPEC-037/Seed-3 — the ast-grep pack dispatch (dispatchPackEngines) is a flat-violations path, not a per-test set-join feed; substantiveness step is separate
metadata:
  type: feedback
---

The existing pack-engine dispatch (`cmd/backstop/pack_gate.go:dispatchPackEngines` →
`runFindingsEngine` → `parseSarif`) returns a FLAT `[]gate.Violation` with no
per-test referenced-symbol payload and does NO gate_type filtering — it runs every
rule of every declared engine and emits one violation per SARIF finding. It is wired
as its OWN `pack_engines` StepFunc in `buildGateSteps`, architecturally SEPARATE from
`testSubstantivenessStep` (`buildTestSubstantivenessStep`).

**Why:** SPEC-037's REQ-003 Q2 noTarget = "pack EXTRACTION emits referenced-symbol
SET per test" + "gate set-join against `implementation.package`". That needs a
consumption path that (a) keys extracted symbols back to a specific mandated test
and (b) routes ONLY substantiveness-gate-type findings into the set-join. Neither
exists today — `runFindingsEngine` flattens to violations and discards finding
identity. The spec asserts "the same ast-grep dispatch path" but the dispatch path is
a violation-emitter, not a symbol-set extractor.

**How to apply:** when reviewing the BUNDLE-009 substantiveness/contracts seeds (and
their plans), require a claim/contract for the EXTRACTION-to-gate seam: how
per-test referenced symbols travel from runFindingsEngine's SARIF back to the
set-join keyed by mandated test, and how substantiveness-gate-type findings are
isolated from the flat pack_engines violation stream. "Rides the same dispatch path"
is under-specified — the dispatch path returns the wrong shape for a set-join.
