---
name: project_baseline_pull_followons
description: ISSUE-176 (CI gate missing baseline.json) delivered by PLAN-ISSUE-176; spawned ISSUE-178 as a named follow-on; two CI-observation facts recorded in ISSUE-176 pending its separate close
metadata:
  type: project
---

`PLAN-ISSUE-176` (CI gate baseline-pull wiring: step-level `GH_TOKEN` + job-level
`actions: read` + a guarded post-gate confirmation step) landed and was confirmed for real on
Linux CI run `32194863181`. `ISSUE-176` itself is NOT yet closed as of 2026-08-18 — its close is
gated on `PLAN-ISSUE-176` TASK-010 (flip plan to `completed` first, then close the issue with
`delivered_by: PLAN-ISSUE-176`), done as a separate step once a real push gates fully green on
`main`. In the meantime `ISSUE-176` carries a `## CI Confirmation (2026-08-18)` section recording
two non-blocking observations ahead of that close: (a) the first pulled baseline artifact was
already ≥2 days stale (predicted; self-corrects once `gate` passes and `baseline` re-publishes),
(b) the artifact has no `retention-days:` override so GitHub's 90-day default puts a real
self-diagnosing failure mode on the calendar around 2026-11-14 if `gate` is still red on `main`
by then.

**ISSUE-178** (`baseline-pull-workflow-name-unfiltered`) is the follow-on `PLAN-ISSUE-176`'s
evidence-gathering named as CLM-007(a) and deliberately did not fix: `resolveLatestSuccessfulMainRun`
(`cmd/backstop/baseline.go:202-224`) decodes each candidate run's `Name` but never filters on it —
latent today only because this repo runs exactly one workflow (`CI`) against `main`; a second
workflow reporting success on `main` would make it selectable and there is no `backstop-baseline-v1`
artifact on any run but `CI`'s.

**Why:** the founder wants issue-author sessions to be able to record fast-follow observations
into an already-in-flight-but-not-yet-closed issue without prematurely flipping its status —
`ISSUE-176`'s close still has real gates (a completed backing plan, `delivered_by`, a Resolution
section) that this session's task deliberately did not touch.

**How to apply:** when asked to "record" or "note" something in an issue that a plan's own
close-out task will formally resolve later, add a dated observational section (not a Resolution
section, not a status flip) and leave `status`/`closed`/`delivered_by` untouched. Check the
backing plan's `notes:`/tasks for the exact close preconditions before assuming a note-only edit
is safe — `ISSUE-176`'s plan spelled out an explicit precondition (real CI evidence, not local
green) for exactly this reason. See [[feedback_body_h1_required_for_title]] for the H1 mechanics
these edits still need to respect.
