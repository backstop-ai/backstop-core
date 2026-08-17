---
name: pgrep-waiter-matches-itself
description: A chained background waiter whose pgrep pattern appears in its own command line matches ITSELF and waits forever — poll the log file for a completion marker instead
metadata:
  type: project
---

`while pgrep -f "<cmd>"; do sleep; done; <next-step>` **never terminates** when `<cmd>`
is written literally inside that same shell invocation: the waiter's own command line
contains the pattern, so `pgrep -f` matches the waiter, the condition stays true, and
the chained next step never runs. The symptom is silent — no error, just a background
task that appears to be "still waiting" indefinitely while the thing it waits on
finished long ago.

Hit 2026-08-17 chaining `gate --all` behind a scoped `gate --file cmd/backstop/doctor.go
…` run. The waiter's `zsh -c` line embedded the full `--file` path list, so it matched
itself. Compounded the stall recorded in
[[feedback_poll_dont_idle_on_background_results]].

**How to apply:** chain on a FILE MARKER, never on process presence. Have the first
command append its own sentinel (`echo "EXIT=$?" >> "$LOG"`) and have the waiter do
`while ! grep -q '^EXIT=' "$LOG"; do sleep 15; done`. That is unambiguous, survives the
process being reaped or restarted, and records the exit code you need anyway. If you
must key on a process, put the waiter in a standalone script file so the pattern is not
part of the invoking command line — but the file-marker form is strictly better because
a gate that CRASHES also leaves no marker and you can distinguish that from success.
