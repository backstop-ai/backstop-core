---
name: gate-file-flag-takes-positional-args
description: "backstop gate --file is REPEATABLE as of ISSUE-093 (was a StringVar that silently kept only the LAST path); still read back 'running against N explicit files' — the habit catches other scope narrowings"
metadata:
  type: project
---

**FIXED 2026-08-17 by ISSUE-093.** `--file` is now a `StringArrayVar`, so BOTH forms
accumulate and can be mixed:

- `--file A --file B` (repeated flag) — previously broken, now works
- `--file A B C` (one flag + positional args) — unchanged, still works
- mixed `--file A --file B C D` — all four reach the scope, flags first

An EMPTY value (`--file ""`) is now an exit-2 config error instead of a silent
fall-through to a diff-scoped whole-repo sweep.

**The historical trap (for reading older transcripts):** the flag used to be a plain
`StringVar`, so repeating it OVERWROTE and only the last path survived. A run would
print `Gate running against 1 explicit files` and pass. I nearly accepted an 8-file
lane gate as PASS when it had gated one test file — coverage announcing
`no in-scope files to measure for coverage` is what gave it away.

**How to apply:** keep reading the `Gate running against N explicit files` line back
and checking N. The repeated-flag defect is gone, but N is still the cheapest tell for
[[project_green_gate_by_scope_exit]] and [[project_gate_all_underreports_vs_diff]] —
other ways a gate reports green over a scope narrower than you intended.

Beware the pflag read-side sharp edge when touching this code:
[[pflag-getstringarray-drops-lone-empty]].

Note the cost: a whole-lane `--file` run over 4+ files (incl. `cmd/backstop/gate.go`,
which derives the slow `./cmd/backstop` package pass) exceeds a 10-minute Bash cap
under concurrent-lane load. Run those in the background and poll.
