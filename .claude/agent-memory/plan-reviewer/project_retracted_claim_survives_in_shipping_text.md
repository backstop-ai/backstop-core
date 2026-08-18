---
name: retracted-claim-survives-in-shipping-text
description: A false claim retracted in plan prose often survives in the PRESCRIBED artifact text (YAML comments, error strings, doc bodies) a task will write into the repo — grep the literal, don't trust the retraction section
metadata:
  type: project
---

When a review round forces a planner to retract a measured-false claim, the fix
usually lands in the two *narrative* locations the reviewer cited (a claim ledger
entry, a sharp edge) plus a new "here is the measurement" notes section. The
instance that survives is the one embedded in **prescribed artifact text** — a
YAML comment block a task will paste into `.github/workflows/ci.yml`, an error
string a helper must emit, a Resolution paragraph a close-out task will write.
That copy is the *durable* one: it lands in the repository permanently, while the
plan's notes are discarded once the lane closes.

Observed on PLAN-ISSUE-176 round 5: notes section (4) and sharp edge 19 both
correctly said pre-gate death is "POSSIBLE BUT UNOBSERVED, 0 of 38", while
TASK-002's prescribed ci.yml comment still read "This job can die at tool
install, at build, or at pack install - **it has done so**".

**Why:** reviewers cite locations by *where they read the claim*; planners fix
exactly those citations. Nobody greps the plan for the assertion's other
spellings, and prescribed code/config blocks read as "implementation detail"
rather than as prose subject to the same evidence bar.

**How to apply:** on any round that retracts a factual claim, grep the WHOLE plan
for a distinctive fragment of the retracted assertion (`"has done so"`,
`"common case"`, the count, the tool name) — not just the cited line numbers —
and check every `description:` block that quotes literal file content a task will
write. Same move for count/name drift: a corrected count in `notes:` frequently
leaves the stale count in the owning task's own `description:`
(PLAN-ISSUE-176 TASK-001 said "three tests"/"all three RED" while declaring four
`test_names`). See [[project_verified_enumeration_do_not_rederive]] and
[[project_completeness_claimed_comment_set]].
