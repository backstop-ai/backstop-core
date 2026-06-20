---
title: "backstop artifact new issue scaffolds the wrong file extension (.md instead of .issue.md)"
schema_version: issue/v1

issue:
  id: ISSUE-011
  title: "backstop artifact new issue scaffolds the wrong file extension (.md instead of .issue.md)"
  type: bug
  status: open
  created: "2026-06-20"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# backstop artifact new issue scaffolds the wrong file extension (.md instead of .issue.md)

## Problem

Running `./bin/backstop artifact new issue --slug <slug>` produces a file named
`issues/ISSUE-NNN-<slug>.md` — the `.issue` infix is absent. Every other artifact
type scaffolds with its correct compound extension: `.bundle.md`, `.spec.md`,
`.plan.yml`. The issue type alone emits a bare `.md`.

**Repro (verified on two consecutive scaffolds — ISSUE-010 and ISSUE-011):**

```
$ ./bin/backstop artifact new issue --slug my-bug
Created issues/ISSUE-011-my-bug.md          # ← wrong; should be .issue.md
```

**Why this breaks the toolchain:**

The issue schema's `filename_pattern` rule requires:

```
^ISSUE-[0-9]{3}-[a-z][a-z0-9]*(-[a-z0-9]+)*\.issue\.md$
```

`artifact validate --issue ISSUE-NNN` discovers and validates by this pattern. A file
emitted as `ISSUE-NNN-<slug>.md` does not match, so the validator does not pick it up
— the freshly scaffolded file is invisible to `artifact validate` and silently fails
discovery.

All ten existing issues in `issues/` use `.issue.md` and pass validation. The scaffold
is the only outlier.

**Likely root cause:** the filename-construction path for the `issue` type in
`artifact new` omits the `.issue` infix that the other artifact types include. The fix
is a one-line correction: align the issue type's output template to emit
`ISSUE-NNN-<slug>.issue.md`.

**Impact:** the scaffold-via-CLI invariant (scaffold → author in place → validates)
does not hold for issues. A freshly scaffolded issue must be manually renamed before
it will validate, and the failure mode is silent — the validator exits 0 on "no
matching artifacts" rather than surfacing the bad filename. This was hit twice in a
row, making it a reliable friction point for any agent or human following the
issue-authoring workflow.

Sibling to ISSUE-009 (plan scaffold emits `issue_id` instead of `spec_id`) — both
are `artifact new` scaffold correctness defects.

## References

- `cmd/backstop/artifact_new.go` (or equivalent) — filename construction for the `issue` type
- `artifacts/issue/v1/schema.json` — `filename_pattern` rule (`\.issue\.md$`)
- ISSUE-001 through ISSUE-010 in `issues/` — all use `.issue.md`; the scaffold is the sole non-conforming producer
- ISSUE-009 — sibling `artifact new` scaffold bug (wrong frontmatter key)
