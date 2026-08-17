---
name: check-the-siblings-plan
description: When an issue argues "not a duplicate of ISSUE-NNN," read PLAN-ISSUE-NNN before slotting — a plan's re-diagnosis routinely merges two textually-distinct defects into one mechanism and one fix, making the issue's own non-duplication analysis correct-but-stale
metadata:
  type: project
---

**An issue's "not a duplicate of ISSUE-NNN" section is an argument about
ISSUE-NNN's TEXT. The binding question is what `PLAN-ISSUE-NNN` scoped.**
Grep `plans/` for the named sibling's ID before classifying — every time,
even when the non-duplication reasoning is careful and correct on its face.

**Why:** ISSUE-145 (2026-08-16, `go build`'s stderr discarded → opaque crash)
devoted a full Notes bullet to distinguishing itself from ISSUE-067 (go-test
opaque crash) on root cause: go-test's `--- FAIL:` output IS on stdout and the
converter fails to extract it, whereas go-build's diagnostics never reach
stdout at all. That distinction is real and the issue was right about it. But
`PLAN-ISSUE-067` — committed `17fac05` **59 minutes earlier**, `status: draft`,
"4 review rounds, ready to implement" — had already re-diagnosed BOTH as one
mechanism (`RunStdout` captures stdout only) with one fix (a pack-declared
`producer:` script per binding, one pack release), under a section literally
headed "THE PART THE ISSUE DID NOT KNOW: go-build HAS NEVER WORKED AT ALL,"
pinned by CLM-005/CLM-007. Slotting 145 anywhere would have split one
review-clean plan's scope across two directives.

**How to apply:**
- On any issue naming a sibling — as duplicate, non-duplicate, or "related" —
  `ls plans/ | grep <sibling-ID>` and read that plan's scope declaration,
  claims, and any "SCOPE BEYOND THE ISSUE'S LETTER" / out-of-scope block. Plans
  regularly widen past their issue's letter and say so up front.
- A plan that widens usually also names its **excision point** ("if the founder
  disagrees, TASK-00N and CLM-00N are the clean cut"). That is the branch where
  the new issue survives as a live artifact — quote it in the escalation so
  Brandon has both options priced.
- When the widened plan's parent issue lives in directive X, do NOT home the
  new issue in directive Y even if the charter test points there. One plan's
  scope split across two directives beats neither home. Say that explicitly —
  it is the reason to override an otherwise-correct SHOUT-vs-LIE call (see
  [[gate-verdict-honesty-cluster]]).

Related: [[project_workaround_and_file_pattern]],
[[project_concurrent_pm_triage_races]]
