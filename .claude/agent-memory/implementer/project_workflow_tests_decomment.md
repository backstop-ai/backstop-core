---
name: workflow-tests-decomment
description: GitHub workflow shape tests that match raw `run:` text also match shell COMMENTS — decomment before asserting, or mandated rationale comments fail the tests that mandate them
metadata:
  type: project
---

`workflows_test.go` assertions that do `strings.Contains(step.Run, needle)` match
SHELL COMMENTS inside the run script, not just executable content.

Two concrete failures this caused:

- ci.yml's baseline job carries the comment "equivalent to `./backstop gate --all
  --json`". A `findJob` predicate looking for jobs that run `backstop gate` matched
  it, saw TWO blocking jobs, and fataled — and the "no `--all` on the gate
  invocation" assertion failed against a comment.
- PLAN-ISSUE-020 TASK-016 MANDATES rationale comments in ci.yml (why the base is
  explicit, why golangci-lint is pinned to v2, why nothing is installed for the
  sandbox). Every one of them names a string the mandated tests forbid, so writing
  the documentation the plan requires fails the tests the plan requires.

**Why:** the tests are about what the workflow EXECUTES; a comment is not an
invocation.

**How to apply:** match against a decommented script (drop lines whose trimmed form
starts with `#`) plus `step.Env` values — workflow expressions are idiomatically bound
to env vars rather than interpolated into the script body. Helpers `stepScript` and
`stepMentions` in `workflows_test.go` do this.

Separately: `cmd/backstop/integration_test.go` carries GOLDEN-TEXT assertions over
ci.yml (it greps for literal `go test -race -coverprofile=...`). Any task that
rewrites ci.yml collides with them, and they are in no plan's file list.
