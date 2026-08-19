---
name: cited-issue-self-retracted-evidence
description: A plan citing a prior issue's Evidence/Problem section may be quoting text that the SAME issue's Root cause section explicitly retracts — read the whole issue, not the cited line range
metadata:
  type: project
---

When a plan cites `ISSUE-NNN lines X-Y` as empirical evidence, read the WHOLE issue
before accepting it. Backstop issues are amended in place: a `## Root cause` section
routinely opens with "the original framing below was WRONG and is replaced, not
extended" and retracts the `**Evidence**` bullets above it — but the retracted bullets
are LEFT IN THE FILE, so a line-range citation lands squarely on them.

Live case (PLAN-ISSUE-172, 2026-08-19): CLM-009 cited ISSUE-068:40-46 for "two
overlapping full suite runs thrash CPU/memory and forced a consumer to cap test
parallelism." ISSUE-068:59-68 retracts exactly that: "The gate runs FULLY
SEQUENTIALLY... there is no concurrent double-run... the fork-cap is not evidence of
a backstop concurrency bug." The plan's conclusion survived on first principles; its
cited evidence did not.

**Why:** the retracted text is the most quotable part of the issue (it is the Problem
statement), and the retraction lives 20 lines below it under a different heading.

**How to apply:** for every `ISSUE-NNN lines X-Y` citation in a plan, grep the issue
for "WRONG", "corrected", "replaced, not extended", "retracted", "is dropped" and
check whether the cited range falls above it. Matters most when a founder-gated task
(here TASK-001/TASK-014) writes the citation into a permanent issue record.
Related: [[retracted-claim-survives-in-shipping-text]], [[sibling-precedent-cited-not-read]].
