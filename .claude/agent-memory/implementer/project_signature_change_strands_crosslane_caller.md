---
name: signature-change-strands-crosslane-caller
description: A plan task that adds a parameter to an internal helper breaks callers in OTHER lanes' test files — grep callers BEFORE the edit; the escape hatches (variadic, re-derive inside) are worse than asking
metadata:
  type: project
---

A task scoped to "edit file X only" that changes a FUNCTION SIGNATURE is not
actually scoped to file X — it is scoped to X plus every caller. In package
`main` (cmd/backstop) the callers are usually TEST files, and a plan that split
lanes by file will have handed one of them to another implementer.

Concrete: PLAN-ISSUE-098 TASK-008 said "edit cmd/backstop/gate.go ONLY" while
mandating `buildStatusDriftSteps` accept a 4th parameter. Its two callers were
gate.go:731 and `driftStepsFor` in status_drift_gate_test.go:106 — the latter
pinned to the sibling's lane and explicitly marked do-not-edit. Package main
then cannot COMPILE until one byte changes in a file you do not own.

The three ways to route around it are all worse than a 30-second question:
variadic `...T` (makes a meaningless arity representable, permanently, to dodge
a lane boundary); re-deriving the value inside the callee (here forbidden — the
single `loadInstalledPacks` call at buildGateSteps is what makes the "the gate
vouched for these packs" argument true); or driving the tests through a
different entry point (does not help — the production signature still changes).

**Why:** the shared tree has no merge conflict to protect you. Two agents
editing one file is last-writer-wins, silently. The lane split IS the only
protection, so crossing it must be a decision someone makes, not one you infer.

**How to apply:** before starting any task whose description contains "accept
the index and forward it" / "add a parameter" / "change the signature", run
`grep -rn "<funcName>" <pkg>/` and check every hit against the plan's lane map.
If a caller sits outside your scope, escalate BEFORE writing code, propose the
exact one-line diff, and say why the escape hatches are worse — that framing got
an immediate authorization plus sibling notification. Meanwhile keep working:
write the red-phase tests against the CURRENT signature so the RED is
BEHAVIORAL (the real defect message) rather than a compile error, and never
leave the shared package non-compiling while you wait.

Related: [[feedback_plan_files_underenumerate_fixtures]],
[[feedback_never_stash_shared_tree]],
[[project_editing_file_pulls_it_into_gate_scope]].
