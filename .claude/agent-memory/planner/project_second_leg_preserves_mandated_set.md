---
name: second-leg-preserves-mandated-set
description: A mutation-found coverage gap is closed by adding a subtest leg to an EXISTING mandated test, never by inventing a new test name — the mandated set must stay at its authored count
metadata:
  type: project
---

When implementation-time falsification (mutation testing, a dropped-conjunct probe)
finds that no mandated test catches a real regression, close the gap with a SECOND
SUBTEST LEG inside an existing mandated test — not a newly invented test name.

**Why:** the `artifact_status_drift` gate dimension derives its mandated-test set from
the plan's own `test_names`. A new name invented at implementation time is a name the
plan never mandated, so the delivered set diverges from the authored claims and the
plan has to be retro-edited to match. A second leg keeps the count exactly as authored
while genuinely closing the hole. Delivered this way on PLAN-ISSUE-160 (commit
`c1e4ef4`): a dropped `runErr != nil` conjunct was caught by none of the four CrashGuard
tests; a `clean_exit_with_no_findings` subtest was added to
`TestPackVal_EngineCrashGuard_NonZeroExitWithFindingsStillReports` and the set stayed
at 16.

**How to apply:** when authoring, write the sharp edge that mandates this explicitly —
"where one direction alone could pass by accident, a SECOND LEG in the opposite
direction is mandated; do not weaken a second leg away" (PLAN-ISSUE-160's SE14). That
one sentence is what lets an implementer close a mutation-found gap without either
inventing a name or leaving the hole open. At close-out, record in the AS-BUILT banner
that the set is unchanged and why.

Related: [[project_extending_a_shipped_plan]], [[project_defect_pinned_by_shipped_tests]],
[[project_plan_closeout_convention]].
