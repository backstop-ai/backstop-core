---
title: "Plan Scaffold Phases Outside Frontmatter"
schema_version: issue/v1

issue:
  id: ISSUE-212
  title: "Plan Scaffold Phases Outside Frontmatter"
  type: bug
  status: open
  created: "2026-10-12"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Plan Scaffold Phases Outside Frontmatter

## Problem

`backstop artifact new plan` writes a scaffold whose `phases:` key lands outside the YAML front
matter. `Scaffold` (`pkg/scaffold/scaffold.go`) writes the opening `---` first
(`sb.WriteString("---\n")`, ~line 163), emits the plan keys (`plan_id`, `spec_id`, `created`,
`status`), then writes the closing `---` (~line 261) and only afterwards, in the "Body sections"
branch, appends `"\nphases: []\n"` for plans (~lines 264-266). The comment there ("Plan uses YAML
phases, not markdown") shows the intent was a YAML key, but it is emitted after the delimiter, so
it is body text.

The bug is present on a fresh scaffold before any editing: validation of the untouched scaffold already
fails immediately. A freshly scaffolded plan looks like this (`backstop artifact new plan --slug demo
--source ISSUE-001`):

```
---
plan_id: PLAN-ISSUE-001
spec_id: ISSUE-001
created: "2026-10-12"
status: draft
---

phases: []
```

The validator reads `phases` from the parsed front matter only (`pkg/validate/plan.go`,
`validatePhases`: `art.Frontmatter["phases"]`, emitting rule `plan/phases-required` with message
"phases array is missing from frontmatter" when absent). Once an author replaces `phases: []`
with real phases, the phases remain in the body and validation fails with that error. The
author-side workaround is to hand-delete the closing `---` so everything below it joins the front
matter. Existing plans show the result of that workaround: `PLAN-ISSUE-104` and `PLAN-ISSUE-201`
each contain exactly one `---` line (the opening delimiter) with `phases:` as a top-level key.

The existing test does not catch this. `TestArtifactNew_Sections_Plan`
(`pkg/scaffold/scaffold_test.go`, ~line 225) only asserts
`strings.Contains(s, "phases: []")` anywhere in the output, which is true whether the key is
inside or outside the front matter.

### Reproduction

Scratch directory; no network and no remote needed (a commit is required because
`artifact new` reserves a ledger tag):

```
mkdir /tmp/pl-repro && cd /tmp/pl-repro
git init -q && git config user.email a@b.c && git config user.name t
echo "project: demo" > backstop.yml
git add -A && git commit -qm init
backstop artifact new plan --slug demo --source ISSUE-001
cat plans/PLAN-ISSUE-001-demo.plan.yml        # phases: [] sits after the closing ---
backstop artifact validate --plan PLAN-ISSUE-001
```

Observed (also after replacing `phases: []` with a populated `phases:` list):

```
PLAN-ISSUE-001-demo.plan.yml
  x [plan/phases-required] phases array is missing from frontmatter
Checks failed
```

Control: delete the closing `---` (leaving a single opening `---`) with the
same populated phases, and `plan/phases-required` no longer fires; only content-level rules
(`plan/task-files-required`, `plan/task-claims-required`, `plan/task-depends-on-required`,
`plan/final-phase-no-verification`) remain, which are expected for a placeholder phase.

## Impact

- Every plan created through the sanctioned `artifact new plan` path starts in a state that
  cannot validate once phases are authored, and the error message ("missing from frontmatter")
  does not point at the real cause (a misplaced delimiter in the scaffold).
- Authors and agents learn to hand-edit the delimiter, which the scaffold-only rule ("never
  hand-create / hand-fix scaffold structure") discourages, and the fix is easy to get wrong
  (deleting the opening `---` instead of the closing one).
- The defect is silent at creation time: the scaffold itself is never validated, and the unit
  test passes, so it survived unnoticed.

## Root cause

Ordering in `Scaffold`: for the `plan` type the `phases` key is written in the body-section
branch (after the closing `---`) instead of with the other front-matter keys in the
`case "plan":` block. The test asserted substring presence rather than structure, so it could not
distinguish front matter from body.

## Direction

(Direction, not a plan.)

1. Plans are read as whole-file YAML, not fenced markdown (`pkg/gate/artifact_status.go`: the
   `planFrontmatter` doc comment at ~331 and `walkPlanDir` at ~559-585 call `yaml.Unmarshal` on
   the entire file), and all 173 plan files on disk contain exactly one `---` line. So the fix is
   to emit ONE YAML document with no closing `---`: keep the opening `---`, write `phases: []`
   with the other plan keys in the `case "plan":` block, and write nothing after it.
2. Strengthen `TestArtifactNew_Sections_Plan` (and/or add a sibling test) to `yaml.Unmarshal` the
   whole scaffold output and assert `phases` is a top-level key, so a regression to a second
   document or body-level placement fails. Additionally assert that a scaffolded plan, with
   `phases: []` replaced by a minimal structurally valid phase list, produces no
   `plan/phases-required` violation from the plan validator.
3. Existing committed plans are untouched; they are already single-delimiter and valid. The
   exception is untracked in-flight scaffolds produced by the current code (two-delimiter,
   `phases` in the body), which need the closing `---` removed by hand.

## Notes

- Related: ISSUE-009 (the scaffold previously emitted `issue_id` instead of `spec_id`, rejected
  by the validator) is the same class of defect: scaffold output not validated against the
  validator it feeds.
