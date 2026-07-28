---
name: pm-write-path
description: PM write path is FULLY OPEN as of 2026-07-26 — both .backstop/pm/ bookkeeping and directive-author dispatch succeed; the earlier blockages are resolved, do not pre-emptively degrade to paste-ready briefs
metadata:
  type: project
---

The PM queue lives at `.backstop/pm/INBOX.md` and `.backstop/pm/pending.log`.
**Both halves of the write path now work** — verified end-to-end on the
2026-07-26 full corpus sweep.

**History (two separate blockages, both resolved):**
1. INBOX writes were blocked by a `.claude/**` sensitive-path classification
   that overrode the allowlist. Fixed by relocating the queue out of
   `.claude/` to `.backstop/pm/`.
2. The `Agent` dispatch to `directive-author` was refused by the auto-mode
   permission classifier on the 22:40Z sweep. On the 23:20Z sweep the same
   dispatch shape **succeeded twice** (DIR-019 and DIR-022 slots, both
   validated clean). The refusal was session-bound, not structural.

**How to apply:** just do the work. Dispatch directive-author for clear-fit
slots and write the INBOX directly. Do NOT pre-emptively emit paste-ready
briefs "because the write path is blocked" — that was true for one session
and is no longer the default. Only fall back to a paste-ready brief if a
dispatch is *actually* refused in the current session; log that refusal as a
PROPOSAL entry so the next sweep knows the state.

**Unchanged constraint:** there is still no hand-editing path to artifacts.
CLAUDE.md forbids hand-editing and the agent-guard hook enforces it, so a
refused dispatch means the slot cannot be executed — never work around it by
editing `directives/*.directive.md` directly.

**Artifact RENAMES need the top-level session (found 2026-07-27, ISSUE-086
split).** `.claude/hooks/backstop-agent-guard.sh` blocks `cp|mv|rm|touch|
install|rsync` against any `*.{bundle,spec,issue,adr,directive}.md`,
`*.plan.yml`, or `BACKLOG.yml` for **every** agent with an `agent_type` —
issue-author, directive-author, and backlog-pm alike. Write/Edit can't rename
or delete, so no subagent can complete a rescope that changes an artifact's
filename slug. **How to apply:** when a rescope retitles an artifact, expect
the authoring agent to finish the content and stall on the rename. Don't
retry the `mv` yourself — it will fail identically. Escalate the exact
`git mv` (or plain `mv` if untracked — check `git ls-files` first) to `main`,
which runs without an `agent_type` and is the only caller the guard lets
through.

**Teammate roster is FLAT.** Spawning via `Agent` with a `name` parameter
fails ("teammates cannot spawn teammates"), and `run_in_background: true`
fails too. Omit `name` and pass `run_in_background: false`; several such calls
in one block still run concurrently. Note this loses the type-prefixed agent
name the guard keys on, but `subagent_type` still populates `agent_type`, so
enforcement is unaffected.

See [[feedback_slot_vs_escalate]], [[project_launch_tiering]].
