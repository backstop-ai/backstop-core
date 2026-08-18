---
name: concurrent-pm-triage-races
description: Multiple hook-fired backlog-pm invocations can triage sibling artifacts into the SAME directive simultaneously — re-read the directive after directive-author returns and correct stale cross-references in place
metadata:
  type: project
---

When an implementer files a batch of sibling issues at once (e.g. ISSUE-106 /
107 / 108, all filed 2026-07-29 within ~90s as PLAN-ISSUE-105's closure
hand-offs), the pm-trigger hook fires **one backlog-pm per artifact, all
concurrently**. They ground independently, reach the same clear-fit verdict,
and dispatch directive-author against the SAME directive file at the same
time.

Observed consequences, all real on 2026-07-29 in DIR-024:
- Item numbering shifts under you — my brief said "append item 12", the agent
  correctly appended **13** because a concurrent pass had already taken 12.
  Never assert an item number in the INBOX from the brief; read it back.
- Cross-references written from a pre-race grounding go stale within minutes.
  My Notes paragraph asserted "ISSUE-108 is cited by no directive" seconds
  after another pass cited it, and the INBOX escalation "104→108 cited by NO
  directive" was already half-false when written.
- Each pass writes its own INBOX entry, so family-level escalations get
  duplicated three ways unless you read the INBOX head before appending.

**Why:** these are independent processes with no shared lock; the artifact is
consistent (directive-author's edits are additive and validation passes) but
each writer's *claims about the corpus* are snapshots.

**How to apply:**
1. After directive-author returns, **re-read the directive** — the source
   list, the item number actually used, and any sibling paragraphs — before
   writing the INBOX entry. Do not trust the brief or the agent's summary.
2. **Read the INBOX head** before appending; if a concurrent pass already
   escalated the family-level ask, write a terse FYI that cross-references it
   instead of re-escalating, and correct its now-stale claims in your own
   entry.
3. If your own write raced and is now wrong, **correct it in place** via the
   same directive-author agent (SendMessage resumes it with context intact) —
   the note-supersedes convention is for *deliberately preserved* history, not
   for a mistake you made minutes ago. See
   [[project_corpus_note_supersedes]].
4. Watch for **instruction leakage** in agent output: phrases from the brief
   ("Also note in that paragraph:", "Content, as follows:") landing verbatim
   in artifact prose. Long structured briefs invite it; always read the
   written paragraph, not just the agent's report.

**New variant, 2026-08-16 (ISSUE-136/137/138, filed in one commit `763ecd0` as
PLAN-ISSUE-129 fallout): the race can be CROSS-directive, not just
same-directive.** The sibling run slotting ISSUE-136 into DIR-032 wrote a Notes
passage about **my** artifact — "ISSUE-137 has no directive home yet" — which was
true when written and false ~15 minutes later once I homed it in DIR-024. So
step 1 is not enough: after slotting, **`git status directives/` and check
whether any OTHER directive was written in the same window, then grep it for
your artifact's ID.** A sibling PM will often name your issue while explicitly
disclaiming it, and that disclaimer is exactly the sentence that goes stale.
Fix it in place through directive-author with a short note-and-supersede clause
(it was true when written, so this is preserved history, not a mistake) and
leave the other directive's `source:` alone — correcting prose is not
re-homing.

**2026-08-18 (ISSUE-164..168 batch, five artifacts in ~15 min): the race is
survivable if you brief for it.** Three PMs hit DIR-024 at once; my brief said
"item 21" and directive-author **re-read the file mid-edit and landed at 23**
on its own, chaining the roster-count paragraph (NINETEEN→…→TWENTY-THREE)
correctly and cross-referencing the sibling item. Nothing needed repair. What
made that work: the brief said *where* to anchor (after the last numbered item,
before `## Notes`) and *what* to update (the count paragraph), not a hard
number. Keep doing that — anchor semantically, then read the number back.

See [[project_gate_verdict_honesty_cluster]], [[project_triage_races_plan_scaffold]],
[[project_linux_ci_green_cluster]].
