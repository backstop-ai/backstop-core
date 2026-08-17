---
name: choose-compile-red-or-behavioral-red
description: Which RED counts as evidence depends on what the task proves — compile-red is strongest for a NEW API surface, but for a DEFECT FIX only a behavioral red (the real violation message) proves the defect exists
metadata:
  type: feedback
---

A red-phase test can fail two ways, and they are not interchangeable evidence.

COMPILE RED (`undefined: PackClaimIndex`) is the STRONGEST available red for a
task introducing a NEW API surface. Nothing can be asserted about a symbol that
does not exist, so "it does not compile" is a complete proof the surface is
absent. Do not contrive a stub just to get a behavioral failure.

BEHAVIORAL RED (the gate emits the actual broken-promise violation naming your
fixture) is REQUIRED for a task that fixes an existing DEFECT. A compile red
there proves only that you wrote a call to something unwritten — it does NOT
reproduce the defect, so a reviewer cannot tell the fix addressed the reported
problem rather than a plausible-looking neighbour.

ISSUE-098 forced the choice: TASK-008's signature change was blocked awaiting a
cross-lane authorization, and writing the phase-3 tests against the FUTURE
signature would have made them compile-red for a reason unrelated to the bug. I
wrote them against the CURRENT signature instead, and got the exact production
message — "artifact ISSUE-700 is closed (success-terminal) but its mandated test
fixture-claim-alpha is ABSENT ... a broken promise" — reproducing ISSUE-098 in a
hermetic temp project. That red is the artifact worth reporting.

**Why:** the plan's red-phase requirement exists to prove the test can FAIL for
the right reason, not to make the suite exit non-zero. A red obtained for an
incidental reason satisfies the letter and defeats the purpose.

STAGE THE IMPL WHEN ONE TASK CARRIES BOTH. A single implementation task often
introduces a new SURFACE and a new BEHAVIOR at once; landing it in one edit
collapses the evidence to compile-red only, because the behavioral tests never
get a chance to run against a compiling-but-not-yet-working tree. Split the
implementation along the surface/behavior seam and re-run in between. On
PLAN-ISSUE-136 TASK-004 (one task, two files) I landed the `manifest.go` half
first — that cleared the 8 `declaredEngineKeys undefined` compile errors and
exposed the real behavioral red on the three advisory tests ("want exactly 1
exempt-scope-decision warning, got 0: []") — then landed the `phase2.go` half.
Two kinds of red from one task, no contrivance, and the plan's file scope
untouched.

**How to apply:** ask what the task proves. New surface -> compile red is fine
and complete. Existing defect -> the red must be the defect's own message; if a
blocker would force a compile red instead, restructure the test to run against
the CURRENT code so the failure is behavioral, and report which kind of red you
obtained per test rather than a blanket "all red". Expect a legitimate mixed
result: on ISSUE-098 two tests were red (the wiring gap) and three passed on
arrival (the falsifier and the no-regression guards) — that split is honest and
worth stating, since a falsifier that must keep passing is not evidence of a
defect. Related: [[project_absence_tests_via_goast]],
[[project_signature_change_strands_crosslane_caller]].
