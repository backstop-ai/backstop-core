---
name: control-vs-treatment-by-preserved-binary
description: To measure a gate-behavior change in a shared tree many agents are editing, preserve a copy of the PRE-FIX binary and run both binaries against ONE tree — not two gate runs minutes apart
metadata:
  type: project
---

When a fix changes what the gate DOES, do not compare a gate run taken before
the edit against one taken after. In a tree with many concurrent lanes those two
runs differ by the fix AND by everything the siblings landed in between, and the
sibling delta can dwarf the signal.

Instead: `cp ./bin/backstop <scratch>/backstop-prefix` BEFORE touching source,
then after the fix run the preserved binary and a freshly built one against the
SAME tree, back to back. The only difference is the fix.

**Why:** on PLAN-ISSUE-091 the plan predicted 8/8 findings; a naive reading gave
10/11. Attribution showed both extra rows came from uncommitted sibling edits —
which cancelled cleanly precisely because the two arms shared one tree state.

**How to apply:** serialize the two runs (a concurrent gate collides on
`cover.out`). Diff on (File, Rule) plus enclosing function name, NEVER on line
numbers. For a findings-only question you can skip the gate entirely and run the
ENGINE directly in both dispatch shapes at one tree state — faster and even
tighter. Note `bash` for such scripts: zsh does NOT word-split unquoted `$VAR`,
so a config-flag string collapses into one argv entry ("File name too long");
build flags as a shell ARRAY. Related:
[[feedback_never_stash_shared_tree]], [[project_redproof_by_worktree_flip]],
[[project_long_suite_samples_a_moving_tree]].
