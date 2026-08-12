---
name: dir002-dir019-reorder
description: BACKLOG.yml position 1/2 swap history for DIR-002 (init) and DIR-019 (recipes) — dependency-based, not raw priority
metadata:
  type: project
---

DIR-002 (backstop init) and DIR-019 (Pack Recipe Capability) have swapped
BACKLOG.yml position 1/2 twice in founder-directed reorders, both argued on a
**real dependency**, not a taste-based reprioritization:

- **2026-08-11**: DIR-002 moved from position 6 to position 2 (behind
  DIR-019). Reasoning: DIR-019 was the only one of the five directives then
  ranked ahead of DIR-002 with a real dependency into init — its unbuilt
  BUNDLE-015 REQ-018 (CI recipe pack), needed by init's resolved bundle
  design (BUNDLE-003 OQ-7) for "CI wired by default."
- **2026-08-12**: DIR-002 moved to position 1, DIR-019 dropped to position 2.
  Reasoning: the REQ-018 CI recipe pack that justified DIR-019's lead was
  delivered 2026-08-12 (SPEC-067 implemented, PLAN-SPEC-067 completed,
  `backstop-ai/ci-workflows` published v0.1.0). DIR-019's remaining open
  scope (ISSUE-081 Gap 3, ISSUE-110, ISSUE-119) is real but does not block
  DIR-002 — it was only ever ranked ahead operationally, not on a live
  dependency. Position was made honest once the dependency cleared.

**Why this matters for future reorders:** this pair's ordering is
dependency-driven and volatile — don't assume either directive's position is
a stable taste call. Before touching either directive's position again,
check both directive files' Notes and BACKLOG.yml's own dated comment
history (it is more current than backlog-pm's `launch-tiering` memory, which
still described `backstop init` as flat tier-2 as of 2026-07-28 and is now
stale on this specific point). This task never created or edited a directive
`.directive.md` file, only BACKLOG.yml itself — no artifact scaffold was
needed for a pure list-reorder.
