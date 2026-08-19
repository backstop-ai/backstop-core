---
name: implementer-cannot-dispatch-artifact-agents
description: The implementer subagent has no Task/Agent dispatch tool, so plan tasks that mandate routing an edit through issue-author/directive-author/spec-author must be handed back to the coordinator as OWED, never hand-edited
metadata:
  type: project
---

A plan task that says "route this edit through the issue-authoring agent" or
"never hand-edit a directive" is **unexecutable by the implementer directly**: the
implementer's toolset has `Skill` (bundle/spec/plan/implement/promote only — no
issue-author, no directive-author) and `SendMessage` (which reaches only
ALREADY-LIVE agents/sessions), but **no general-purpose Task/Agent dispatch tool**.
Verified 2026-08-18 on PLAN-ISSUE-099 TASK-008 via `ToolSearch` for
dispatch/subagent/task/agent — the only hits were `TaskStop`, `SendMessage`,
`EnterWorktree` and unrelated Vercel MCP tools.

**Why:** CLAUDE.md makes hand-editing artifacts law-level forbidden ("Never
hand-edit artifacts. Route all artifact authoring/evolution to the purpose-built
agents"), and the agent-guard hook enforces it. So the implementer can neither
route the edit nor perform it — the task is genuinely blocked, not merely
inconvenient.

**How to apply:** when a plan task mandates an artifact edit through an authoring
agent, do NOT hand-edit and do NOT silently skip. Do the parts you CAN do — the
coordination check (`git status directives/`), the `artifact validate` runs, the
diff-scoped gate over the artifacts as they stand — and report the mutation as
**OWED, by name**, with the exact correction text ready for the coordinator to
dispatch. Plans routinely pair such a task with its own validation block; run that
block against the un-reconciled state and say so plainly rather than skipping it.
Good plans anticipate this: PLAN-ISSUE-099 TASK-008 explicitly said "if the
correction cannot land in this task's window, report it as OWED, by name, rather
than as done."

See [[project_ciyml_byte_identity_guard]] for the other TASK-ordering constraint
that lane hit (commit-before-green).
