---
name: pm-workaround-and-file-pattern
description: Most new issues arrive from implementer sessions that hit a defect, worked AROUND it, and filed — so in-flight coverage is almost always nil; check the filer's fix intent instead of interviewing
metadata:
  type: project
---

Nearly every issue the pm-trigger hook hands me arrives the same way: a live
implementer session hit a defect mid-task, **worked around it**, filed an
issue as a deliberate carve-out, and kept going on its original lane.
Confirmed on ISSUE-092 (implementer-087 invoked semgrep directly instead of
fixing `pack test`), ISSUE-093 (implementer-087 wrote a `--file` workaround
into its own memory), ISSUE-094 (the ISSUE-020 worker filed it as an explicit
carve-out), and ISSUE-095 (implementer-020 used `pack remove` + `pack add`
and moved on).

**Why:** the discovering session is under its own plan's scope discipline —
fixing an out-of-scope defect would be off-plan work. Filing is the correct
behavior, and it means the filer is *structurally* not the fixer.

**How to apply:** the in-flight-coverage check for a hook-delivered issue is
usually a 30-second corpus check, not an interview:
1. `ls plans | grep -i <NNN>` — does a plan exist against it?
2. `grep -rl "ISSUE-NNN" directives/` — does any directive already cite it?
3. Fingerprint the two or three sessions that mention it — if they are the
   discoverer and the filer, coverage is nil. Say so and move on.

Only interview when a session's *lane* plausibly subsumes the substance —
i.e. someone is working the same package with a broader charter — not merely
because a session mentions the ID. Two corroborating signals that a filer is
not fixing: a documented workaround in the issue text, and a `phases: []`
plan stub (see [[pm-triage-races-plan-scaffold]]).

A useful corollary for the escalation itself: the workaround the filer used
is load-bearing evidence about blast radius. ISSUE-095's remove+add
workaround had already carried `backstop-core`'s whole lock to
`source_type: git` before I triaged it, which shrank the defect's live
exposure to other consumers only — worth measuring before writing the
severity paragraph.

See [[pm-interview-tooling-constraints]].
