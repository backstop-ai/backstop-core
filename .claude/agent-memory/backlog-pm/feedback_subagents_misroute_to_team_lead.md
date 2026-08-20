---
name: subagents-misroute-to-team-lead
description: Subagents I spawn end their turn with a SendMessage to "team-lead" and a stub final text, so their findings bypass me entirely — brief against it up front
metadata:
  type: feedback
---

**When I spawn investigation subagents, they reliably end their turn by calling
`SendMessage(to: "team-lead")` with the substance and leaving their FINAL TEXT
as a stub ("Findings sent to team-lead. Summary:"). The substance never reaches
me** — only the parent's returned final text does. Hit on 7/7 forks at once
(2026-08-19 home-ruling sweep); the reports landed in the main conversation
instead, and two were dropped entirely.

**Why:** these agents inherit a team-oriented harness where "team-lead" is a
real address, so "report your findings" reads to them as "message the lead."
The parent-return channel is the plain final assistant message, which they then
waste on a one-line stub.

**How to apply:**
- Put this in EVERY subagent brief: *"Do NOT use SendMessage. End your turn with
  the complete report as your plain final assistant text message — that is what
  reaches me. Do not send it to team-lead or main."*
- Diagnose it from `~/.claude/agent-reports/teammate-<name>.md` — a "Last real
  message" that reads "Findings sent to team-lead" plus a stub is this failure,
  NOT a stalled agent. Idle timestamps + transcript sizes prove the work is done.
- Recovery WITHOUT re-running the work: extract the last assistant text straight
  from the transcript —
  `jq -r 'select(.type=="assistant") | .message.content[]? | select(.type=="text") | .text' <transcript>.jsonl | tail -c 9000`
  on `~/.claude/projects/<slug>/<session>/subagents/agent-a<name>-<hash>.jsonl`.
  This is the ONE sanctioned exception to [[feedback_never_read_subagent_output_files]]:
  a targeted `jq` tail, never a full read.
- Re-briefing them to "send to main" is also wrong — that reaches the main
  conversation, not me.
