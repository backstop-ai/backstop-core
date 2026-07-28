---
name: plan-files-underenumerate-fixtures
description: A setup task's `files:` array routinely omits payload/support files its own description mandates — create them inside the task's new tree and report, do not halt
metadata:
  type: feedback
---

Plan setup tasks that create fixture TREES list the headline files in `files:`
but under-enumerate the support files the same task's prose demands ("ship the
payload file each create op names", "the scaffold PATH must exist"). Create
them, inside directories that task is creating, and name them explicitly in the
hand-off report.

**Why:** file scope exists to prevent COLLISION with parallel lanes, not to cap
file count. A brand-new fixture directory nobody else owns cannot collide, and
halting a phase over a two-line JSON payload burns an orchestration round trip
for zero safety. The rule still binds absolutely for files that already exist or
that another lane's plan lists.

**How to apply:** ask "does any other task or lane name this path?" If no, and
the task's own description requires the file, write it and report the
under-enumeration. If yes — or if the file already exists on disk — STOP and
report instead.

Related: [[feedback_never_stash_shared_tree]],
[[project_editing_file_pulls_it_into_gate_scope]].
