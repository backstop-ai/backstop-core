---
name: gate-contention-in-shared-tree
description: With ~9 sibling lanes live, a 3-file `gate --file` exceeds 10 min and a `gate --all` gets KILLED (exit 144) — run gates detached with an EXIT= marker and poll, never in the foreground
metadata:
  type: project
---

When many implementer lanes are active in `/Users/bmanson/src/projects/backstop-core`
at once, gate runs stop being a foreground operation.

Measured 2026-08-17 (PLAN-ISSUE-136, ~9 concurrent `backstop gate` processes):
a THREE-file `gate --file` blew a 10-minute foreground timeout; the same lane's
seven-file run took ~20 min with `pack_engines` alone at 666s; `gate --all` ran
~688s in `pack_engines` and one attempt was **killed outright at exit 144**
before writing a single byte, needing a relaunch.

**Why:** every lane's gate forks its own go build / go test / semgrep, so they
compete for the same cores. Exit 144 is the harness reaping the background job,
not a gate verdict — a 0-byte output file plus exit 144 means "never ran", which
is easy to misread as "gate produced nothing, so it passed".

**How to apply:**
- Launch every gate detached, redirect to a scratch file, and append an `EXIT=$?`
  marker as the last line. Poll for the marker — that marker is the only reliable
  "it finished" signal (a task-completion notification fires for the *wrapper*,
  not the gate).
- `ps aux | grep -c "[b]in/backstop gate"` before starting an unscoped sweep. At
  9 it is not worth launching; wait for it to drop.
- A 0-length output file is never evidence. Re-check size before reading a
  verdict off it.
- Under this much churn, prefer the diff-scoped `gate --file` on your own paths
  as your real signal and treat `gate --all` as a corpus reading, not a personal
  one — see [[project_gate_all_underreports_vs_diff]] and
  [[project_long_suite_samples_a_moving_tree]]. To attribute a red, build a
  control binary in a detached worktree at HEAD and re-run there:
  [[feedback_never_stash_shared_tree]].
