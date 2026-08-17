---
name: record-only-consequence-filings
description: Plans with a "file the consequences, don't absorb them" task emit issues that contain NO defect — behavior-change records and completed-plan claim retractions. They are not clear fits for defect-shaped directives; triage the whole batch, not the one artifact the hook handed you.
metadata:
  type: project
---

A backstop plan may carry a task whose whole job is **surfacing** what the fix creates —
`PLAN-ISSUE-091` TASK-006, pinned by its CLM-009 *"the consequences are filed, not absorbed."*
Such a task mandates several issues at once and **forbids folding any of them into the plan's
own tasks**. The artifacts it produces are a distinct species:

- **Behavior-change records** — a deliberate, founder-visible, often *subtractive* change (e.g.
  ISSUE-150: `gate --all` no longer reports `testdata/` findings, 3 measured rows).
- **Completed-plan claim retractions** — a completed plan's claim prose falsified by a later
  fix. **Completed plans are NEVER rewritten**; the correction lives in a new issue citing both
  (ISSUE-149 → PLAN-ISSUE-010 CLM-004; ISSUE-150 → PLAN-ISSUE-040 CLM-005).
- Occasionally a **★ founder decision** the plan deliberately refuses to answer (ISSUE-152: may
  CI's blocking job now use `--all`, since ISSUE-091 was the sole stated reason for that ban?).

**Why:** these issues contain **no defect and nothing to fix**, so they are *not* "squarely
within exactly one directive's charter" — the standing clear-fit grant does not reach them,
even when the issue file names a home itself (149 and 150 both named DIR-032). Slotting one
into a defect-shaped directive like DIR-032 ("a gate step reports the **wrong verdict**") adds
a member no planner can plan. DIR-021 (corpus drain) is the other tempting home and is also
wrong when the filing carries zero gate violations.

**How to apply:**
- **Recognize the species first**: an issue whose Problem section says "deliberate behavior
  change, not a bug," or whose body is a retraction of quoted prose. Escalate as **record-only,
  born complete** — recommend closing with a `## Resolution` section (the CLAUDE.md open-but-fixed
  convention) or slotting *as a record, explicitly not a planable member*.
- **Triage the BATCH.** Read the mandating task and enumerate every filing it orders, then check
  each against `pending.log` — the hook routinely delivers one of four. Give the batch ONE
  ruling; splitting siblings of identical shape across different dispositions is the drift the
  PM exists to prevent. See [[pm-trigger-hook-is-wrong-in-both-directions]].
- **Verify the enforcement scope cheaply, always**: `grep -c test_names plans/<PLAN>.plan.yml`.
  `0` ⇒ the plan mandates tests in prose only, contributes nothing to `MandatedTests`
  (`pkg/gate/artifact_status.go`), and the retraction **reds no gate dimension** — say so, so
  nobody hunts for a broken promise. Both PLAN-ISSUE-010 and -040 return 0.
- **Separate the record from the question it raises.** ISSUE-150's real ask ("is the lost
  fixture-content audit view worth a flag?") is *new capability* and would home in DIR-024, not
  in the verdict-honesty directive the record itself pointed at.
- Siblings from one batch are usually **empty shells for minutes** after `artifact new` — an
  empty `## Problem` means mid-authoring, not defective. Wait for a body.
  See [[triage-races-plan-scaffold]].

Related: [[gate-verdict-honesty-cluster]] (the charter line this species fails),
[[note-supersedes-convention]] (the same never-rewrite instinct applied inside directives).
