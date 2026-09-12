---
name: indented-fence-truncates-plan-frontmatter
description: An embedded three-hyphen line inside a plan's notes/description — even indented inside a block scalar — ends the artifact frontmatter early and silently truncates every task after it
metadata:
  type: project
---

`artifact.Parse` (`pkg/artifact/parse.go`) reads frontmatter from the first line whose
`strings.TrimSpace` equals `---` to the NEXT such line. The trim means INDENTATION DOES NOT
PROTECT IT: a `---` nested fourteen spaces deep inside a `description: |` block scalar is
still a closing fence. Everything after it — the rest of that task, and every subsequent
phase and task — is dropped from `Frontmatter` and never validated.

**Why:** hit live 2026-08-24 authoring PLAN-ISSUE-154. Its TASK-001 description embedded the
literal contents of a `.plan.yml` test fixture, fences and all. The validator reported
`plan/task-files-required`, `plan/task-claims-required`, `plan/task-depends-on-required` on
`phases[0].tasks[0]` plus `plan/final-phase-no-verification` on `phase-1` — a plan that
looked structurally broken at its FIRST task while the real file had six well-formed tasks
across four phases. The violations name a truncation, not the defect they appear to describe.

**How to apply:** whenever a plan quotes artifact contents, YAML samples, or markdown
frontmatter inside `notes:` or a task `description:`, never emit a bare three-hyphen line.
Spell the marker out (`FENCE`, "a literal three-hyphen frontmatter marker") and say why in
the same breath, so a later reader does not helpfully restore it. Before validating, sweep:

    awk '{t=$0; gsub(/^[ \t]+|[ \t]+$/,"",t); if (t=="---") print NR": "$0}' <plan>

Exactly one hit, at line 1, is correct — corpus plans use a single leading fence and run to
EOF (`PLAN-ISSUE-172`, `-179`, `-186` all measured at one fence). More than one means silent
truncation. Read the same way in reverse when a plan's violations cluster implausibly on
`tasks[0]`: suspect the fence before rewriting the task. Contrast the scaffold `artifact new
plan` emits — opening fence, fields, CLOSING fence, then `phases: []` OUTSIDE it, which
yields `plan/phases-required`; PLAN-ISSUE-066's retired single-fence shape yields
`plan/phases-empty` instead. See [[project_plan_validator_no_terminal_exemption]].
