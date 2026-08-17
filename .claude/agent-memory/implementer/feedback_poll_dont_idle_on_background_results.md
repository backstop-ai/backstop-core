---
name: poll-dont-idle-on-background-results
description: Never go idle waiting for a background-task notification to wake you — poll the output file on a bounded cadence, because completion notifications arrive late and batched
metadata:
  type: feedback
---

After starting a long `run_in_background` command (or arming an `until`-loop waiter),
**do not end your turn expecting the completion notification to wake you.** Poll the
output file yourself.

**RECURRED A THIRD TIME 2026-08-17 (ISSUE-134), WITH THIS NOTE ALREADY WRITTEN AND READ.**
Two gate runs finished at 23:02 and 23:20; I had armed a Monitor, ended my turn, and did
not read either until the lead's "no activity in 1h15m" check at 00:12. Knowing the rule
did not prevent it — what prevents it is ending every turn with an actual cheap check,
not an intention to be notified. If you are about to end a turn while ANY background run
is outstanding, that is the trigger: poll first, in the same turn.

**Why:** 2026-08-16, ISSUE-067. A `backstop gate` run finished at 19:24 and wrote a 47KB
log. I had armed background waiters and then went idle, trusting the notification. The
notifications did fire — but arrived ~77 minutes later, batched, only after the team-lead
sent a "status check, no activity in 1.5+ hours, are you stalled?" message. The result had
been sitting complete on disk the whole time. Nothing was lost but the hour and a quarter,
and the lead had to chase me. This is the same monitor-resume gap recorded project-side as
[[project_monitor_resume_unreliable]] — it is a harness characteristic, not a one-off.

**How to apply:**
- After launching a background run, keep working. If genuinely blocked on its result,
  re-check with a cheap `[ -s "$LOG" ] && ... || echo running` on each turn rather than
  stopping.
- Treat "I armed a waiter" as NOT equivalent to "I will be told." The waiter is a
  convenience, never the mechanism you rely on.
- When a teammate asks whether you are stalled, **check real state before answering** —
  `ps -p <pid>`, the log's mtime and byte count — and if you did stall, say so plainly
  rather than narrating it as still-in-progress. See [[feedback_plaintext_reports_never_received]].
- Corollary for detection: `pgrep -f "<pattern>"` SELF-MATCHES, because the pattern appears
  in the grepping shell's own command line. It reports STILL RUNNING after a clean kill.
  Confirm with `ps -eo pid,command | grep "[b]racket-trick"` instead.

★ **IT RECURRED THE SAME NIGHT, IN A DIFFERENT LANE — SO TREAT THIS AS A STANDING TRAP, NOT
AN ANECDOTE.** 2026-08-16, ISSUE-124: identical shape (armed a Monitor on a long `gate`,
reported "standing by", went idle ~2 HOURS, lead had to send the same status-check message).
Two lanes, one evening, same mistake. An armed watch is never the mechanism you rely on.

★ **AND THE REAL COST IS NOT THE LOST TIME — IT IS THE UNSUPPORTED CLAIM.** In the ISSUE-124
case I had told the lead "standing by, all clear" about a run whose output I had not opened.
When I finally read it, that run had reached the full chain for the first time and carried
THREE findings on MY OWN file (two go-standards `error-wrapping-required` violations I had
introduced, plus a coverage red). The reassurance was wrong at the moment I gave it.

So: **never characterize a result you have not read.** Say "run in flight, I have not read
it yet" — never "all clear." The unread log is exactly where your own defects live, because
a run that goes further than previous runs is precisely the one that reaches dimensions
nothing had checked before.
