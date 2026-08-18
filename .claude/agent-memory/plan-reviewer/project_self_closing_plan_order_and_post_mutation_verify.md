---
name: self-closing-plan-order-and-post-mutation-verify
description: A plan whose close-out task closes its own issue via delivered_by must flip the PLAN to completed FIRST and add a verification task AFTER the artifact mutations; TASK-009-style kill chains validate the vacuous pre-close state
metadata:
  type: project
---

When a plan's close-out task closes its source issue with `delivered_by:`, three
things are load-bearing and plans repeatedly miss all three (PLAN-ISSUE-157 took
three review rounds on exactly this):

1. **Order.** `pkg/validate/delivered_by.go` step 5 requires the backing plan to
   already be `status: completed`. Closing the issue before flipping the plan
   yields `issue/delivered-by-plan-not-completed`.
2. **Three fields, not one.** `issue.status: closed` + `issue.closed: <date>` +
   TOP-LEVEL `delivered_by:` (sibling of `schema_version`, NOT nested under
   `issue:`) + a `## Resolution` section. A Resolution section alone fails with
   SIX errors; a dated close without `delivered_by` still fails with FIVE
   (verification/implementation/requirements/claims/contracts). `delivered_by`
   and `resolved-by` are mutually exclusive.
3. **A verification task AFTER the mutations.** A final "kill chain" task that
   runs `artifact validate --issue X` BEFORE the close-out task validates the
   still-open, trivially-passing pre-close state — vacuous. Because the gate is
   diff-scoped, the hole ships green locally and reds for whoever touches the
   repo next. Demand a terminal verification task depending on the close-out
   task.

**Why:** delivered_by re-runs the FULL `validate.Plan` over the backing plan
(`issue/delivered-by-plan-invalid`), so the flip and the close are coupled.

**How to apply:** reproduce the candidate end states yourself in a scratch tree
with `issues/` + `plans/` siblings (delivered_by resolves `plans/` relative to
the ISSUE's own SourcePath, never CWD) and run `validate.Issue` with a real
schema from `schema.LoadArtifactSchema` — passing nil panics in `validate.Base`.
Also check the STOPPED-LANE branch: an untouched open issue + draft plan is a
legitimate 0-violation end state, so a decline branch is not automatically a
finding. See [[project_undefined_claim_roster]],
[[project_mandated_name_premise_needs_spec_status]].
