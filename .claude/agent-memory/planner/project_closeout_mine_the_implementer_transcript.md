---
name: closeout-mine-the-implementer-transcript
description: An AS-BUILT banner needs verbatim RED/failure text the implementer's final report never pasted — read its transcript .jsonl, not just its report
metadata:
  type: project
---

At close-out, the verbatim evidence a banner needs (the exact assertion strings from the
kind-(i) RED, the inversion demonstration's failure output, the deliberate-mutation RED)
almost never survives into the implementer's final report — that report summarizes
("falsification demonstrated", "RED confirmed exactly as planned") and drops the strings.

The evidence IS recoverable:
- `~/.claude/agent-reports/<agent_type>-<agent_id>.md` — the agent's last output plus the
  path to its full transcript.
- That transcript, `…/subagents/agent-<id>.jsonl` — one JSON object per line; tool_result
  blocks carry the raw stdout. Grep the whole file as text for a distinctive fragment of
  the assertion message (`MUST precede test_verification`, `tool exited 1, script exited 0`)
  and print a window around the hit.

**Why:** PLAN-ISSUE-172's close needed TASK-002's inversion output and TASK-004's pre-fix
RED — both unrecoverable from the tree (the mutations were reverted, the fixture was
edited) and both absent from the implementer's summary. Mining the transcript produced
them verbatim, plus the fact that the inversion was done via a `go -overlay` compile-time
swap rather than an in-place edit — an AS-BUILT delta nobody had reported.

**How to apply:** before writing an AS-BUILT banner, locate the implementing agent's
report file and transcript and pull the RED text for every falsification the plan mandated.
Quote it indented in the banner. Label everything you did NOT re-run yourself as CARRIED,
and say why you carried it. See [[project_closeout_real_gate_in_worktree]] for when
carrying is the CORRECT call rather than laziness.
