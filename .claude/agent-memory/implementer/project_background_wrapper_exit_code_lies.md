---
name: background-wrapper-exit-code-lies
description: A background task notification reporting "exit code 0" is the WRAPPER subshell's status, not the command's — a 13-failure test run was announced as success
metadata:
  type: project
---

`(cmd > log 2>&1; echo "EXIT=$?" >> log)` run in the background always makes the SUBSHELL
exit 0, because `echo` is the last statement. The task-completion notification then says
**"completed (exit code 0)"** for a run that actually failed.

Measured 2026-08-17: `go test ./cmd/backstop/... -race` was announced as exit code 0. The
log's own last line read `EXIT=1`, with 13 `--- FAIL` lines. Believing the notification
would have shipped a lane claiming a green suite.

**Why:** the notification reports the process the harness launched, which is the wrapping
shell, not the program you care about.

**How to apply:** never quote a background notification's exit code as the command's
result. Read the log and grep for your own `EXIT=` marker plus `^--- FAIL|^FAIL|^ok`.
Better: put the marker in a SEPARATE sentinel file (`echo "EXIT=$?" > run.done`) and chain
the waiter on that file, so "finished" and "succeeded" stay distinguishable. Same family as
[[feedback_poll_dont_idle_on_background_results]] (an armed watch is not a wakeup) and
[[feedback_zsh_pipestatus_is_one_indexed]] (an exit code reported the wrong way is no
evidence).
