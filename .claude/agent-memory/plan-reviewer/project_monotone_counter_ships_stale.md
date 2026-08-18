---
name: monotone-counter-ships-stale
description: A plan-time monotone counter ("N consecutive failing runs", "the last N runs") prescribed for verbatim transcription into a permanent artifact is stale by implement time — re-measure it and demand robust phrasing
metadata:
  type: project
---

A count that only ever grows — consecutive failing CI runs on main, runs-since-date,
open-issue totals — is measured once while the plan is authored and then often
prescribed for VERBATIM transcription into something permanent (an issue's
`## Resolution`, a shipped code comment, a close-out banner). By implement time it is
wrong, and unlike a stale review note it ships.

Re-execute every such counter yourself in each round: they move between rounds, not
just between plan and implementation. (PLAN-ISSUE-176, round 6: the plan's "failed on
every main push for 21 consecutive runs since 2026-08-16" was 23 by the time I
re-queried, hours later, and TASK-007 told the implementer to write "21" into
ISSUE-176.)

**Why:** it is the same failure class as a stale shipped comment — a claim the plan
corrected in its own notes but left frozen in the bytes it prescribes.

**How to apply:** distinguish ROBUST phrasings from FROZEN ones. `≥2 days stale`,
`since <date>`, `unobserved in the last 78 runs` stay true or are explicitly
window-stamped; a bare `21 consecutive runs` does not. Where a counter is
load-bearing, check whether the CONSEQUENCE it supports is stable (here: "the newest
publishable baseline is run 31921681066's" — verified unchanged) before treating the
drift as a blocker. See [[project_uncommitted_lane_rows_move_pinned_counts]] and
[[project_stale_sibling_scaffold_evidence]].
