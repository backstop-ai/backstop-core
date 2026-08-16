---
name: extending-a-shipped-plan
description: How to append a reconciliation phase to an existing plan whose earlier phases already shipped — the forced task-type ordering and the spec+test-rename pairing convention
metadata:
  type: project
---

When a spec amendment invalidates code an EARLIER phase of an already-shipped plan
produced, add a new phase before the final verification phase rather than editing the
historical phase — leave the original task as the record and give it a bracketed
`[SUPERSEDED IN PART BY PHASE Na]` forward pointer.

**The task-type ordering is forced, not stylistic.** The new phase edits a file an
earlier phase's tasks already list, so D-081 file exclusivity demands a transitive
dependency into that phase. `pkg/validate/plan.go` allows exactly one edge back into a
finished phase's tail: **a `test` task MAY depend on a `verification` task**
(`validateTestTaskDeps`, REQ-010 — allowed set is setup/test/verification). `refactor`
and `implementation` may NOT. So a reconciliation phase must open with a `test` task
depending on the earlier phase's verification task, then `refactor`, then `verification`.
A single combined task is not expressible: typed `refactor` it cannot reach back and
collides on files; typed `test` it can reach back but then no verification task in the
phase can satisfy REQ-006.

**Why:** learned authoring PLAN-SPEC-037 phase-6a (2026-08-15). The split it forces is
worth framing as add-then-consolidate (prove the new coverage, then fold and delete the
superseded function) — which also matches that plan's own strangler ordering.

**Mandated-test renames go SPEC-FIRST, then plan** — separate commits, not one. Precedent:
SPEC-038 v1.2.2 + PLAN-ISSUE-082 TASK-002, and again SPEC-037 v1.2.7 (f9569df) landing
ahead of its plan phase. A spec-author dispatch amends `claims[].tests`; the plan's task
then carries the code-side rename verbatim. Write the plan as the COUNTERPART to a
committed spec edit ("do not re-edit the spec; confirm it, and if it is wrong, route a
spec-author dispatch"), not as a co-landing pair.

**The window between the two commits is invisible, so assert the pairing by hand.**
test_verification and test_substantiveness both filter through `ContractsAreDue`, which
admits only `implemented` specs, and `pkg/validate/spec.go` has no mandated-test-existence
check — so while a spec is `draft`, a half-landed rename passes the FULL gate green. A
plan's verification task must therefore explicitly assert: the spec names the new function
AND it exists, no `.go` file still names the old one, and the claim carries its `subject:`.
See [[substantiveness-subject-join]].

**How to apply:** any "the spec was amended, the test no longer matches" extension
request. Read the spec's Version History entries first — they usually already record the
owed work and name the constraint (e.g. a rename that alone will NOT clear a noTarget
violation without a paired `subject:` fix).
