---
name: defect-pinned-by-shipped-tests
description: Before planning a defect fix in backstop-core, grep for shipped tests that ASSERT the defective behavior — their names are often spec-mandated, so keep the names and rewrite the bodies
metadata:
  type: project
---

A backstop-core defect is frequently PINNED by passing tests that assert the wrong
behavior in so many words. Grep the target function for its existing tests before
writing any phase, and grep each such test name across `specs/` and `plans/`.

**Why:** ISSUE-112 planning (2026-08-15) found FOUR shipped, passing tests asserting
that a provision-pinned engine tool's absence from PATH "must NOT fail provisioning"
— the exact defect. All four names were mandated by `implemented` specs (SPEC-034,
SPEC-031). Deleting or renaming any of them would have turned the
`artifact_status_drift` dimension RED as a broken promise, so the fix would have
self-inflicted a gate failure on top of the defect. A green suite was not evidence
of correctness; it was evidence the defect had been ratified.

**How to apply:**
- Run the target package's existing tests at HEAD and read which ones encode the
  defect. That run IS the falsification when the planner role cannot author a Go
  probe (planners are guarded out of writing `.go`, including in the scratchpad —
  use a detached `git worktree` at HEAD to run existing tests instead).
- KEEP every spec-mandated test name; rewrite bodies and docstrings. Instruct the
  docstring to say why the name is retained and which issue supersedes the old
  assertion — otherwise the name reads as a lie to the next reader.
- Preserve each test's still-true secondary angle (in ISSUE-112: "the binding carries
  a pinned Provision record" and "the bespoke install ladder is gone" both survived).
- The spec PROSE goes stale and nothing mechanical catches it (the gate checks
  test-name EXISTENCE, not claim text). Do not let the plan hand-edit specs — record
  the divergence in notes and hand the dispatcher a spec-author follow-on.

Related: [[project_extending_a_shipped_plan]], [[feedback_verify_issue_premises]],
[[project_planning_a_pack_data_fix]].
