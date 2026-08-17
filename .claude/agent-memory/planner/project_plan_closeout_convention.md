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
- **The plan may never have been committed AT ALL** — check `git status` on the plan path
  before reaching for `git show <plan commit>`. PLAN-ISSUE-151 (2026-08-17) was still
  untracked (`??`) at close-out: the delivery commit carried six source files plus a
  testdata corpus and NO artifact, so there was no way to establish what the plan said
  before implementation. Say so at the TOP of the banner as a provenance caveat and make
  the AS-BUILT DELTAS the authority on divergence, since a future reader cannot derive it
  from history.
- **Re-count the mandated tests yourself; the hand-off's number can be low.** The lead
  reported "10 mandated tests" for PLAN-ISSUE-151; grepping the plan found 15 (13
  `TestPathScope_*` + 2 `TestCIGlobScoping_*`), all present at HEAD, plus one UNMANDATED
  test shipped beyond them. `grep -o "Test[A-Za-z0-9_]\{4,\}" <plan> | sort -u`, then
  `git grep "func <name>(" HEAD -- '*_test.go'` per name — a low carried count would have
  under-claimed the delivery and hidden the extra test from the deltas section.
- **When the code landed in a MULTI-LANE BATCH commit, the batch message is a poor
  provenance record** — it can read as though a defect caught in plan REVIEW was actually
  shipped and then corrected. Establish what the plan said BEFORE implementation with
  `git show <the plan's own commit>`, and if the batch message is misleadable, put an
  explicit "CORRECTION FOR THE RECORD" paragraph at the TOP of the banner naming what
  never reached the tree. (PLAN-ISSUE-113 vs batch commit 4dbf64b, 2026-08-16; hit AGAIN
  same day on PLAN-ISSUE-112 in the SAME commit — its message reads "widened the check
  from *exec.Error-only", which sounds like a repair but describes a DIFFERENT package's
  pre-existing code; the plan commit proved the wide predicate was mandated up front.)
  Also enumerate the lane's OWN files in the banner: a batch commit's diff is not your
  lane's diff, and a future reader `git show`ing it will otherwise mis-attribute the
  other lanes' files to your plan.
- **Closing against an UNCOMMITTED shared tree is the same trap without the commit.**
  A file in `git status` may be 100% another lane's — and the whole-file diff will not
  say so. Diff each file you intend to credit and check it actually mentions your
  artifact ID before naming it. (PLAN-ISSUE-091, 2026-08-16: `docs/CODEBASE-MAP.md` was
  modified and would have been credited to the lane; its diff was entirely
  PLAN-ISSUE-067's producer-seam paragraph and contained zero ISSUE-091 references.
  Two more files — `pack_gate.go`, `filemode_scoping_test.go` — were genuinely SHARED,
  with this lane owning only a branch and a comment inside a 182-line diff.)
- **Verify the hand-off report's own promised follow-ups actually landed** — a close-out
  is the last place to catch one that didn't. (PLAN-ISSUE-091: its ISSUE-097 append said
  a derived 2→0 figure "will be appended here once available"; TASK-007 then measured it
  AND found a third stale token, and neither ever reached ISSUE-097. Grepping the target
  artifact for the promised content — rather than trusting "appended to ISSUE-097" in the
  report — is what surfaced it, and it went into the banner as an open residual.)
- **A plan close-out edit is uncommitted when you finish, and it must land in the SAME
  commit as the sibling issue/spec close.** A parallel issue-author closing the source
  artifact runs a closed-requires-traceability check; if it reads HEAD rather than the
  working tree it sees `draft` and false-negatives, and committing the issue alone leaves
  HEAD carrying a closed issue pointing at a draft plan. Say so explicitly when you
  report — and do NOT commit it yourself when other lanes are live in the tree, since a
  broad `git add` sweeps their uncommitted work ([[feedback_git_stash_shared_tree_hazard]]
  is the same hazard by a different verb). Stage the one plan path or hand it to the lead.
- Close with `./bin/backstop artifact validate --plan PLAN-NNN`. Plans currently validate
  "schema-less", so a pass is a parse/structure receipt, not a claim about content.
- Don't run the full suite or `gate --all` to re-verify a close-out when other agents are
  active in the tree — see [[feedback_no_verify_race_with_active_implementer]]. Targeted,
  non-racing checks (build, vet, the specific renamed test with `-v` to prove no subtest
  SKIPPED, a grep that the old name survives nowhere) are the honest substitute, and the
  banner should say which readings were carried rather than re-run.
