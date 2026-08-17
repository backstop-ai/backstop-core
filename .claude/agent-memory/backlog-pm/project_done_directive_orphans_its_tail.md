---
name: done-directive-orphans-its-tail
description: Marking a directive `done` orphans every follow-on its member plans filed — check for a residual tail BEFORE and AFTER any directive closes
metadata:
  type: project
---

A directive reaching `status: done` on a complete roster does NOT mean its work
is fully homed. Its member plans routinely file follow-on issues as they land
("file it, don't absorb it"), and those follow-ons cite the parent directive
nowhere. The moment the parent closes, they are orphaned by construction.

**Measured, 2026-08-17 full sweep:** `DIR-032 "Gate Verdict Honesty"` closed at a
genuinely complete 21/21 — all 21 sources `closed`, 21 numbered Description
items, verdict earned. Its members' plans had emitted **nine** follow-on issues
with no directive home: 149/150/152 (←PLAN-ISSUE-091), 154 (←PLAN-ISSUE-066),
155 (←PLAN-ISSUE-136), 156 (←PLAN-ISSUE-093), 157+159 (←PLAN-ISSUE-148), 161
(←PLAN-ISSUE-146). Nine of the ten newest issues in the repo. Same sweep found
5 more open issues cited only by `done` DIR-001/DIR-026.

**Why:** the roster is a closed set fixed at authoring time; the tail is an open
set generated at *delivery* time. Nothing reconciles the two. This is the
issue-side twin of the bundle-side ruling Brandon made 2026-08-14 (citation by a
`done` directive does not count as homed — BUNDLE-004/005 re-listed on it); as of
2026-08-17 he had NOT extended it to issues, so it is escalated, not assumed.
See [[project_homed_but_orphaned_bundles]].

**How to apply:**
- Trace provenance mechanically, never by title:
  `grep -oE 'PLAN-(ISSUE|SPEC)-[0-9]+' issues/ISSUE-NNN-*` per file. A follow-on
  names the plan that filed it.
- Before agreeing a directive is `done`, sweep its members' plans for filed
  follow-ons. A "fully delivered" roster with a live tail is a false green.
- NEVER slot a residual into its own `done` parent — that reproduces exactly the
  state Brandon's bundle ruling rejected. Escalate the family as ONE ruling
  (successor directive vs. fold into a catch-all vs. reopen), not N slots.
- Recommend a successor directive over reopening: reopening retroactively
  falsifies a `done` verdict that was actually earned.
- A `done` directive still sitting in BACKLOG.yml also violates the ratified
  2026-08-02 remove-on-done convention — flag it, but removal is NOT granted;
  propose only. See [[project_pm_write_path_blocked]].
