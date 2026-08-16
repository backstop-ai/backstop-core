---
name: code-check-command-removed
description: "`backstop code check` does not exist — it was removed; the planner agent definition still mandates it, so plans must use `backstop gate` instead"
metadata:
  type: project
---

**There is no `backstop code check` command.** It was removed from the CLI. Invoking
it errors with `unknown command "code" for "backstop"`, and its absence is
asserted by `cmd/backstop/code_check_removal_test.go`. The only gate entrypoint is
`backstop gate`:

- `./bin/backstop gate` — diff-scoped vs merge-base + untracked (the inner loop
  that `code check` used to serve)
- `./bin/backstop gate --file <path>` — explicit file scope
- `./bin/backstop gate --all` — full project sweep

**Why this matters more than a normal stale-command note:** the planner agent
definition itself still instructs, verbatim, that "All verification tasks must use
backstop CLI commands… Always use `backstop code check` or `backstop gate`," with a
whole subsection describing `code check` as the fast inner loop. Following those
instructions faithfully produces a plan built on a command that does not exist —
which is what happened on PLAN-ISSUE-082 (2026-08-15), caught only by the
plan-reviewer. The prose reads canonical, so it does not look wrong.

**How to apply:** never write `backstop code check` into a verification task,
regardless of what the agent definition says. Use `backstop gate` for middle-phase
verification and `backstop gate --all` + `backstop artifact validate` for the final
phase. If the agent definition still mandates `code check` when you read this, the
definition has not been corrected yet — flag it to the team lead rather than
editing agent configuration yourself, and plan around it.

Related: [[feedback-verification-uses-real-commands]]
