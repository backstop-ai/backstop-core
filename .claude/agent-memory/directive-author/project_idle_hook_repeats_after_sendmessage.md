---
name: idle-hook-repeats-after-sendmessage
description: teammate-idle-enforce-report.sh fired repeatedly (3x) after SendMessage(to:"main") calls that succeeded, on an unnamed/subagent-style directive-author dispatch
metadata:
  type: project
---

On a 2026-08-17 directive-author dispatch (append-only DIR-001/DIR-026 notes
task), the `teammate-idle-enforce-report.sh` idle hook fired THREE times in a
row claiming "you're going idle without ever having called SendMessage" —
even after two prior turns each ended with a `SendMessage({to:"main", ...})`
call that returned `{"success":true,"message":"Message queued for the main
conversation's next turn."}`.

**Why:** this corroborates the existing global memory note ("Idle hook,
unnamed subagent" — `to:"main"` sends from an unnamed/non-teammate-named
subagent may not register with the hook's own completion tracking, so it
re-fires on the next idle transition regardless of a real, successful send).

**How to apply:** if this hook fires again after a SendMessage(to:"main")
call already succeeded and the task is genuinely done, do not endlessly
resend full reports — send one brief re-confirmation (not a duplicate of the
full report) and then stop responding to further identical hook fires. This
is very likely a hook-side tracking gap on unnamed dispatches, not a real
signal of unreported work. Do not treat a third/fourth repeat as evidence
more work is expected — verify against your own turn history first.
