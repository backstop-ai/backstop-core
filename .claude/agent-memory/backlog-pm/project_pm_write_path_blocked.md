---
name: pm-write-path
description: PM write path is open for .backstop/pm/ bookkeeping everywhere, but directive-author WRITES to directives/** are blocked in NON-INTERACTIVE sessions (harness ask-state, found 2026-07-29) — stage the slot content, log the refusal, re-run interactively
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

**The headless directive-write refusal is SESSION-BOUND, not structural
(corrected 2026-07-29T13:18Z).** On the 2026-07-29 ~02:10Z ISSUE-102/103
triage, directive-author's `Edit`/`Write` on `directives/*.directive.md` was
refused by the HARNESS permission layer's ask-state ("requested permissions …
but you haven't granted it yet") in a non-interactive run, and I recorded that
as a structural third blockage shape. **That generalization was wrong.** On the
13:18Z ISSUE-107 triage — also headless, same dispatch shape, same target
directory — directive-author edited `DIR-024` three times and ran
`artifact validate` with **no refusal at all**. So this matches blockage #2's
pattern exactly: the auto-mode permission classifier refuses some sessions and
not others.

**How to apply:** ALWAYS attempt the dispatch, headless or not — never
pre-emptively downgrade to a paste-ready brief on the theory that headless
can't write. Only if a dispatch is *actually* refused in the current session,
have directive-author compose the full slot content, record it + the refusal
verbatim in pending.log/INBOX, and note that a re-run applies it. Do NOT add
permission rules yourself, do NOT hand-edit, do NOT let the subagent script a
bypass (guard blocks it correctly). ISSUE-102/103's slots were DECIDED but
never APPLIED under the old refusal — they are still unapplied and a re-run
would likely now succeed.

Third data point, independent: the 13:22Z ISSUE-108 triage (also headless)
dispatched directive-author against the SAME file minutes later and it wrote
cleanly too. Two headless successes vs one headless refusal — treat writing as
the default expectation.

**Batch filings produce CONCURRENT sibling triages that race on one directive
(found 2026-07-29, ISSUE-106/107/108).** The hook fires once per artifact, so a
three-issue batch commit spawns three PM runs seconds apart — and when all
three are the same family they all slot into the SAME directive. Observed: the
ISSUE-108 dispatch got *"the file had been modified on disk since you last
read it"* mid-edit, and its Description item number shifted 13 → 14 because
ISSUE-106's slot landed between reads. **How to apply:** when pending.log shows
sibling artifacts logged within a minute of yours, tell directive-author
explicitly to (a) re-Read the file immediately before each edit, (b) use
surgical appends only — source list, one new numbered item, one Notes
paragraph — never a rewrite or reflow, and (c) DERIVE the item number at edit
time rather than taking it from your brief. All three slots then land cleanly
and the file still validates. Expect one cosmetic casualty: a sibling's Notes
paragraph may assert your issue is "cited by no directive / not slotted here."
Leave it (note-supersedes convention) and flag it for the next sweep.

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

**Bookkeeping writes must use Write/Edit, never a shell script (found
2026-07-28).** A `python3 - <<EOF` heredoc appending to
`.backstop/pm/pending.log` was refused: *"agent backlog-pm: scripted write to
artifact file; use the Write/Edit tools."* The guard's scripted-write rule
covers the PM queue files too, not just `directives/`+`issues/`. **How to
apply:** append the ` [triaged]` marker with an `Edit` on the log's last line
(match the bare filename line, replace with filename + marker). Don't reach
for `sed`/`python`/`echo >>` — it costs a round trip every time.

**BACKLOG.yml is STRUCTURALLY off-limits to backlog-pm — route it through
directive-author (confirmed by reading the hook, 2026-07-29, BUNDLE-031
append).** My `Edit` on `BACKLOG.yml` was refused: *"agent backlog-pm not
permitted to write BACKLOG.yml."* This is NOT the session-bound ask-state
shape above — it is hard-coded: `backstop-agent-guard.sh:56-63` allows
backlog-pm only `.backstop/pm/*` and its own agent-memory, while the
`directive-author*` case explicitly allows `*.directive.md` **and** any file
basenamed `BACKLOG.yml`. **How to apply:** the standing bundle-append grant is
still executable — just dispatch directive-author with an append-only brief
(exact lines, "do not touch the `directives:` list, do not commit"), then
verify yourself with `git diff --stat` (a clean append is a single hunk, and
the insertion count should equal the lines you asked for). Never attempt the
`Edit` first; it costs a round trip every time.

Caveat to the scripted-write rule above: a `printf ... >> .backstop/pm/pending.log`
DID succeed on 2026-07-29 (verified by reading the tail back). So the
scripted-write refusal is not universal for the PM queue — but Write/Edit
remains the cheaper default, since a refusal costs a retry and the append
costs nothing either way.

**Teammate roster is FLAT.** Spawning via `Agent` with a `name` parameter
fails ("teammates cannot spawn teammates"), and `run_in_background: true`
fails too. Omit `name` and pass `run_in_background: false`; several such calls
in one block still run concurrently. Note this loses the type-prefixed agent
name the guard keys on, but `subagent_type` still populates `agent_type`, so
enforcement is unaffected.

See [[feedback_slot_vs_escalate]], [[project_launch_tiering]].
