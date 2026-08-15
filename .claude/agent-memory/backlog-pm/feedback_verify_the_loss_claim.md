---
name: verify-the-loss-claim
description: When an issue's severity rests on "X was lost / is unrecoverable / never happens", go look for X in the tree — the mechanism is usually real and the consequence usually isn't
metadata:
  type: feedback
---

An issue's *mechanism* claim and its *consequence* claim need separate verification.
Confirming the mechanism (read the code, it really does block/misfire) does NOT confirm
the consequence ("so the data is lost", "so nobody can do this", "so it never runs").
Check the consequence against the tree independently — look for the artifact that was
supposedly lost, count the things that supposedly can't exist.

**Why:** ISSUE-126 (2026-08-15) claimed agent-guard blocks every family's memory writes,
so self-improvement lessons are "silently unrecoverable" and "simply lost." The mechanism
was exactly right — I read `.claude/hooks/backstop-agent-guard.sh` and only `backlog-pm*`
has an `agent-memory` carve-out. The consequence was flatly false: the specific file the
issue named as lost was on disk, 2,351 bytes, written the same minute the block happened,
and 62 memory files had been written across the "blocked" families since the guard's last
change — including all four reviewer families whose case arm is a bare `wblock`. There was
an unaccounted-for path (the guard `exit 0`s when `agent_type` is absent, and its Bash arm
restricts only artifact-file globs, so heredocs into `agent-memory` pass freely). Filing
this at the issue's implied severity would have bought a compounding-silent-loss framing
for what is actually a retry-tax policy inconsistency.

**How to apply:** any issue whose priority argument is a loss/impossibility claim gets one
cheap falsification pass before the INBOX entry — `ls`/`find -newermt`/`git ls-files` on
the thing that allegedly can't exist. Costs one tool call. When the consequence collapses,
say so in the entry as the headline finding, not a footnote: the founder is ranking off
that claim. Same family as [[project_fix_menus_overstate_core_gaps]] (fix menus name
capability that already ships) — both are "the reporter modeled the system from one
surface and missed a second path."
