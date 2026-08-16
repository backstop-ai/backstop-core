---
name: gate-file-flag-takes-positional-args
description: "backstop gate --file is a single StringVar plus positional args; repeating --file silently keeps only the LAST path and reports '1 explicit files' — a false green"
metadata:
  type: project
---

`backstop gate --file` scopes to MANY files as `--file A B C D` (one flag, the rest
positional), NOT as repeated `--file A --file B`. The flag is a plain
`cmd.Flags().StringVar` (`cmd/backstop/gate.go:55`) and the scope is built as
`explicitFiles = append([]string{fileValue}, args...)` (`gate.go:145`), so repeating
the flag silently OVERWRITES — only the last path survives.

**Why:** the help text reads "scope gate to one or more explicit files", which invites
the repeated-flag form. Repeating it does not error; the run just prints
`Gate running against 1 explicit files` and passes. I nearly accepted an 8-file lane
gate as PASS when it had actually gated one test file — and coverage even announced
`no in-scope files to measure for coverage`, which is what gave it away.

**How to apply:** always read the `Gate running against N explicit files` line back and
check N equals the number of paths you passed. If N is 1 when you meant many, you used
the repeated-flag form and the green is meaningless. Related:
[[project_green_gate_by_scope_exit]] and [[project_gate_all_underreports_vs_diff]] —
three separate ways a gate reports green over a scope narrower than you intended.

Note the cost: a whole-lane `--file` run over 8 files (incl. `cmd/backstop/gate.go`)
exceeded a 10-minute Bash cap. Run those in the background rather than re-running.
