---
name: triage-races-plan-scaffold
description: PM triage fires within seconds of filing, so a `/plan` stub for the same artifact is often already in the tree — empty `phases: []` plans failing validate --all are mid-authoring, not defects
metadata:
  type: project
---

The pm-trigger hook fires the instant an issue is written, so triage frequently
overlaps the filing session's *next* step. Observed 2026-07-27: ISSUE-087 filed
at 11:10:50Z, `plans/PLAN-ISSUE-087-*.plan.yml` scaffolded at 11:11Z as an empty
`plan_id/spec_id/status: draft` + `phases: []` stub — before triage finished.

**Why:** `backstop artifact new` reserves the ID and writes a skeleton, then the
planner agent fills it. Between those two moments the tree contains a plan that
fails `artifact validate --all` on `phases-required`.

**How to apply:** (1) Never report a stub plan's validation failure as a defect
or a regression from your own directive edit — check `git status` (untracked?)
and the file's mtime against the trigger timestamp first, and say so explicitly
if a subagent reports `--all` red. (2) An empty plan stub IS meaningful in-flight
coverage — it means the planning lane is live for that artifact, so scope your
escalation to "a plan is being authored now," not "no lane exists." Cheapest
in-flight signal available; costs one `ls plans/PLAN-<ID>*`. See
[[project_interview_tooling_constraints]] for why transcript-grep is the
fallback when forking is blocked.
