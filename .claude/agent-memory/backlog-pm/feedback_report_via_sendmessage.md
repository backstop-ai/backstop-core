---
name: report-via-sendmessage
description: When running as a team agent, plain text output is INVISIBLE to the team lead — every report must go through SendMessage or the turn reads as going idle mid-task
metadata:
  type: feedback
---

When dispatched as a teammate (not by the pm-trigger hook), finishing the work
and writing the report as ordinary text output is **not delivering it**. The
team lead sees nothing and reads the turn as the known stop-before-report
failure. The report must be sent with `SendMessage({to: "team-lead", ...})`.

**Why:** happened 2026-07-27 on the founder-decision apply pass. All four
artifact edits landed and validated, the INBOX and pending.log were updated,
and I wrote a complete per-item report — as text. The lead had to prompt:
"You went idle without your apply-pass report — the known stop-before-report
pattern." The work was done; only the delivery was missing. Subagent
completions DO reach the lead automatically, which makes the gap easy to miss:
my dispatched agents' results surfaced while my own summary did not.

**How to apply:** on any teammate-dispatched task, the LAST action of the turn
is a `SendMessage` to the requester carrying the full report — per-item
outcomes, validation results actually run, and findings. Do this even when the
work is obviously complete and even when subagent notifications have already
surfaced pieces of it. Treat text output as scratch notes for the user's
terminal, never as the deliverable. `SendMessage` needs loading via
`ToolSearch("select:SendMessage")` first; budget that round-trip rather than
skipping the send.

Corollary from the same session: when relaying to a subagent whose bare type
name collides with other agents, address it by the raw `agentId` from its
spawn result — names are unreachable when duplicated. See
[[pm-write-path]] for the flat-roster and artifact-rename constraints.
