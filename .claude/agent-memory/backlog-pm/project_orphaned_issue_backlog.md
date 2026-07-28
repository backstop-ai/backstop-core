---
name: orphaned-issue-backlog
description: Roughly half of open issues are cited by no directive — the pm-trigger hook only catches artifacts created after it existed, so uncited-open is the default state for sprint-filed issues, not an anomaly
metadata:
  type: project
---

On the 2026-07-26 full sweep: of 28 open issues, **15 were cited by no
directive at all** — including ISSUE-079, a `risk: critical` silent-wrong-output
defect on the #1 launch blocker.

**Why:** the pm-trigger hook only fires on artifacts created after it was
installed. Anything filed during a sprint before that — and anything filed
while an escalation sat unanswered — never got triaged. Filed status is not
homed status.

**How to apply:** on any sweep, compute the uncited-open set explicitly
rather than assuming the hook kept up. The cheap way:
`grep -H -m1 '^  status:' issues/*.issue.md` for open ones (status is NESTED
under `issue:`, a bare `^status:` grep silently returns almost nothing), then
diff against the union of every directive's `source:` list. Directives mention
issue IDs in prose that are NOT in `source:` — only the `source:` list counts
as homed.

Orphaned issues cluster by theme, and a cluster with no owning directive is a
new-directive signal, not N separate slot decisions. The 2026-07-26 sweep found
four gate-verdict-correctness issues orphaned because their natural charter
(DIR-015) was already `done` — a `done` directive leaves its whole subject area
unowned, and per the ISSUE-082 precedent we don't reopen done directives.

See [[project_launch_tiering]], [[pm-write-path]].
