---
name: plan-closeout-convention
description: How to close a delivered plan to `completed` — AS-BUILT banner prepended to notes, spec_version left pinned, tasks never rewritten
metadata:
  type: project
---

Closing a delivered plan means flipping `status: draft` -> `completed` and PREPENDING an
AS-BUILT / "CLOSED <date>" banner to the existing `notes:` block. The phases, tasks, claim
mappings and original prose are NEVER rewritten — they are the historical record, and the
banner is the layer over them. Precedents: PLAN-ISSUE-104, PLAN-SPEC-055/056/067/068/069/070,
PLAN-ISSUE-020/048/105/116.

**Why:** a plan is an audit trail, not a living document. Editing tasks to match what shipped
destroys the ability to see where the as-built diverged from the plan — which is exactly the
information an AS-BUILT DELTAS section exists to record.

**How to apply:**
- Banner content that has earned its place: what landed per phase with commit SHAs; fix-round
  history (what each impl-review round found); follow-ons FILED rather than absorbed (with
  issue IDs); evidence at close, SPLIT into what you re-verified yourself vs what you carried
  from the spec's close-out; and AS-BUILT DELTAS where the plan was wrong.
- **Leave `spec_version` at the revision the plan was authored against** — do NOT bump it to
  the spec's close-out revision. Sibling convention: PLAN-SPEC-068 sits at 1.2.3 against a
  1.2.9 spec, PLAN-SPEC-069 at 1.3.1 against 1.3.4. Bumping it falsely implies the plan was
  re-planned against the newer text.
- Older plans prescribe `backstop code check` in their verification tasks. That command was
  REMOVED (ISSUE-018) and its absence is now asserted by a shipped test. Record it as an
  as-built delta; do not edit the task text, and never carry that string into a new plan.
- **When the code landed in a MULTI-LANE BATCH commit, the batch message is a poor
  provenance record** — it can read as though a defect caught in plan REVIEW was actually
  shipped and then corrected. Establish what the plan said BEFORE implementation with
  `git show <the plan's own commit>`, and if the batch message is misleadable, put an
  explicit "CORRECTION FOR THE RECORD" paragraph at the TOP of the banner naming what
  never reached the tree. (PLAN-ISSUE-113 vs batch commit 4dbf64b, 2026-08-16.)
- Close with `./bin/backstop artifact validate --plan PLAN-NNN`. Plans currently validate
  "schema-less", so a pass is a parse/structure receipt, not a claim about content.
- Don't run the full suite or `gate --all` to re-verify a close-out when other agents are
  active in the tree — see [[feedback_no_verify_race_with_active_implementer]]. Targeted,
  non-racing checks (build, vet, the specific renamed test with `-v` to prove no subtest
  SKIPPED, a grep that the old name survives nowhere) are the honest substitute, and the
  banner should say which readings were carried rather than re-run.
