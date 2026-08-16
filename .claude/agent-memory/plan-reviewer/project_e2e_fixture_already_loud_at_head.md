---
name: e2e-fixture-already-loud-at-head
description: An acceptance-fixture task that mandates "observe the silent pass at HEAD" must specify the fixture property that makes HEAD actually silent — an adjacent guard often already refuses it, making the red-proof unobtainable and the test green-at-HEAD
metadata:
  type: project
---

When a plan's E2E/acceptance task says "observe the defect (silent pass / vacuous
green) at HEAD first, then assert the refusal", TRACE THE FIXTURE THROUGH EVERY
GUARD ON THE PATH AT HEAD — not just the one the plan is fixing. A neighbouring
guard frequently refuses the naive fixture already, which means the mandated
HEAD observation is contradicted by reality AND the acceptance test is GREEN AT
HEAD, i.e. vacuous — the exact failure the lane exists to eliminate.

The property that decides it is usually a single seam field the plan never
names. Concrete instance (PLAN-ISSUE-112 round 3, 2026-08-16): TASK-009's
fixture was "a findings engine whose command's argv[0] is an absent binary."
With a NIL `Provision` that is already refused at HEAD — `provisionEngines`
exempts nil-Provision from the allowlist gate (`pack_gate.go:813-815`), then
presence-probes `engineToolName(command)` and returns a `*check.ConfigError`
(`pack_gate_provision.go:88-108`). Only a NON-NIL Provision pinned to an
allowlisted tool/version (see `pkg/pack/engine/allowlist.go`) reaches dispatch,
where the discarded `*exec.Error` becomes the silent pass. Aggravating: an
EARLIER task in the same plan explicitly instructed "give each binding a NIL
Provision" (correct there, fatal here), so an implementer carrying the habit
forward lands straight in the hole.

Resolution accepted (round 4, verified round 5): fixture pins a NON-NIL
`Provision` to a REAL production-allowlist entry with a DIVERGENT argv[0]
carrying the absent binary name, and the test READS the pinned version from
`engine.TrustedToolAllowlist()` instead of hardcoding it — a hardcoded version
that later drifts turns the intended absent-tool refusal into a
version-mismatch refusal of identical fail+ConfigErr shape. Corollary the
planner had to fix: the refusal message must then name BOTH the probed argv[0]
and the pin, or the "absent tool NAMED" assertion is unwritable.

**Why:** claim coverage, TDD ordering and the validator all pass on such a plan.
The hole is only visible by executing the HEAD control flow over the fixture by
hand. A green-at-HEAD acceptance test ships as proof of a fix that was never
falsified.

**How to apply:** for every task that mandates a HEAD observation, walk the real
code path with the fixture's declared fields and ask "which guard fires FIRST at
HEAD?" If the answer is not the guard the plan is fixing, the fixture spec is
incomplete — blocker. Check especially for a nil/non-nil seam (Provision,
CrashGuard, Convert, StdoutArtifact) and for allowlist/trust gates that run
ahead of the target check. Related: [[project_captured_fixture_source_must_exist]],
[[project_new_guard_predicate_measure_existing_fixtures]],
[[project_astgrep_pack_convert_script_scope]].
