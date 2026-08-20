---
name: idle-hook-resend-loop
description: The TeammateIdle report hook can keep firing after a SendMessage that returned success — report twice, then stop, never loop
metadata:
  type: feedback
---

When dispatched as a named subagent (e.g. `plan-issue-128`), a `SendMessage` to
`main` that returns `{"success":true,"message":"Message queued..."}` may STILL be
followed by `teammate-idle-enforce-report.sh` insisting "You're going idle without
ever having called SendMessage." Observed three consecutive firings against two
confirmed-successful sends (2026-08-19, PLAN-ISSUE-128 landing).

**Why:** the hook's detection does not register the send, so its assertion is a
hook-side gap, not evidence the report was lost. Treat the tool's own success
return as the authority on whether the dispatcher got the message — the same
"verify, don't assert" rule that applies to gate output applies here, and the
hook is the less reliable witness.

**How to apply:** send the real status report once. If the hook fires again, send
one short re-send that ALSO flags the non-registration (it is a useful data point
and corroborates the existing project note about this hook). After that, stop —
further sends are pure noise in the main conversation and burn turns. Conclude in
plain prose instead, and say plainly that the report was delivered and the hook is
misreporting. Do not treat repeat firings as a signal to redo or re-verify the
work itself.

Related: the project-level auto-memory note about this hook missing a `to: "main"`
send from an *unnamed* subagent — this observation extends it to NAMED subagents
too, so the name is not the discriminator.

Nor is the RECIPIENT. 2026-08-19, second sighting (PLAN-ISSUE-132 landing): three
firings against two successful sends — the first to the dispatching peer by name
(`plan-issue-132`, routing echoed `sender: planner`), the second to `main`. Both
returned success; the hook registered neither. So it is not "only counts sends to
main" and not "only misses unnamed senders" — do not spend a turn re-routing to a
different recipient hoping to satisfy it.
