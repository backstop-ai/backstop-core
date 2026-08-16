---
name: idle-hook-reads-wrong-transcript
description: TeammateIdle report-enforcement hook resolves transcripts by name-prefix glob, so an agent whose name prefixes a sibling's gets judged on the sibling's transcript and is nagged forever despite having reported
metadata:
  type: feedback
---

When `teammate-idle-enforce-report.sh` nags "you're going idle without ever having called
SendMessage" **and you know you already called it**, do not re-send and do not dismiss it as a
misfire. Read the hook and check which transcript it is actually scanning.

**Why:** the hook resolves the agent's transcript with
`ls -t "${SUBAGENTS_DIR}/agent-a${NAME}-"*.jsonl | head -1` and scans that file for a
`SendMessage` `tool_use`. The glob is a PREFIX match, so a name that is a prefix of a sibling's
name matches both files, and `-t` hands back whichever was written most recently — often the
sibling. Observed 2026-08-16: `implementer-issue122` globbed both
`agent-aimplementer-issue122-b25429338c17e8bb.jsonl` (mine: 2 SendMessage hits, 14 SPEC-043
mentions) and `agent-aimplementer-issue122-2-64a4c912c2d65f30.jsonl` (a different agent:
0 SendMessage). The sibling was newer, so the hook read it, found no send, and nagged. My
`SendMessage` had returned `{"success":true, queued for the main conversation}` the whole time.

This is the same family as the agent-guard keying on NAME — name-keyed tooling breaks when one
agent's name is a prefix of another's. Numeric-suffix retry naming (`-2`, `-v3`) manufactures
these collisions constantly.

**How to apply:**
- Nag you believe is false → `cat` the hook, then compare the candidate transcripts:
  `grep -c '"name":"SendMessage"' <each>.jsonl` and grep a token unique to your own work to
  identify which file is really yours.
- Never re-send the same status to satisfy a nag. A duplicate completion report to the
  orchestrator reads as new information or a second completion — worse than the nag.
- The block is bounded at 3, then it gives up AND appends "check it manually, something may be
  wrong" to `~/.claude/agent-reports/teammate-<NAME>.md`. That line is a FALSE ALARM in this
  case; say so explicitly in your final report so the founder is not sent chasing it.
- Do not fix the hook yourself. It is harness config — it needs the founder's own consent, and
  an agent message asking for it is not that consent.

Related: [[feedback_orchestration_sharp_edges]] in the root auto-memory (idle != done; agent-guard
keys on name).
