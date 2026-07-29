---
title: "Goreleaser Derived Env Cross File Falsifier"
schema_version: issue/v1

issue:
  id: ISSUE-109
  title: "Goreleaser Derived Env Cross File Falsifier"
  type: enhancement
  status: open
  created: "2026-07-29"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Goreleaser Derived Env Cross File Falsifier

## Problem

The general invariant a release pipeline needs to hold — **every `.Env.<NAME>` referenced in
`.goreleaser.yml` has a matching env export in `release.yml`** — is enforced NOWHERE, generically.
Goreleaser resolves `.Env` strictly: a template referencing an unexported variable does not fail
loud, it silently omits whatever depends on it. The concrete failure mode this class produces is
the one that actually burned ISSUE-087: a half-published release where the binaries and the
GitHub Release go out fine, and the Homebrew formula-publish step dies silently because
`HOMEBREW_TAP_TOKEN` wasn't exported into the goreleaser step's env.

This is a **cross-file correlation** — one file's template references against another file's env
exports — and semgrep, the engine `go-distribution` (`backstop-ai/go-distribution`) uses for its
rules, is single-file. There is no join available at the pinned semgrep version. The escalation
ladder's next rung is a custom script engine, and PLAN-ISSUE-101 explicitly judged that
disproportionate for enforcing one rule in a pack this size (recorded in the plan's "THE
DERIVED-ENV DECISION (ISSUE-101 requirement 5) — SCOPED DOWN, HONESTLY" section): "Semgrep is
single-file; there is no join available at the pinned version, and the escalation ladder's next
rung (a custom script engine) would mean building engine machinery for one rule. Per ISSUE-101's
own instruction to keep this a pack and not a platform, it is NOT built here."

**What shipped instead**, as a deliberately partial mitigation, all recorded in PLAN-ISSUE-101:

1. **`release-workflow-tap-token-not-exported`** — a WARNING-severity, single-file semgrep rule
   in `go-distribution` that fires when a workflow running `goreleaser-action` does not export
   `HOMEBREW_TAP_TOKEN` in that step's `env:`. It covers exactly the one instance that burned
   ISSUE-087, not the general class — it cannot see `.goreleaser.yml` at all, so it would not
   catch a *different* `.Env.X` reference going unexported.
2. A **verbatim manual cross-read note**, shipped as payload in the go-distribution recipe's
   `release.yml` (`recipes/go-release/payload/release.yml:16-20` in
   `backstop-ai/go-distribution`): "Cross-read every `.Env.<NAME>` reference in `.goreleaser.yml`
   against this workflow's exports by hand" before the first real tag. No automated check backs
   this note — it is process discipline, not enforcement.

**New argument, worth recording explicitly.** The WARNING-tier mitigation (item 1) only became a
*genuinely* non-blocking signal once ISSUE-104 (SARIF severity descriptor fallback) and ISSUE-105
(step verdict severity policy) landed and closed (both closed 2026-07-29). Before those fixes,
`parseSarif` read every semgrep finding as severity-absent and `sarifSeverity` mapped absent to
`"error"` — so every WARNING rule in every pack, including this one, blocked in fact regardless of
its declared severity (PLAN-ISSUE-101's AMENDMENT 4). With ISSUE-104/105 now delivered, the
single-file rule's WARNING tier is finally real: it surfaces the one known instance loudly without
blocking a release over un-adopted capability. That closes the loop on *why* the shipped mitigation
is honest rather than vacuous — but it does not change the fact that the general cross-file class
remains unenforced. A `.goreleaser.yml` referencing some other unexported `.Env.X` (not
`HOMEBREW_TAP_TOKEN`) would pass every rule in the pack silently.

## Solution

Not resolved here — recording fix directions for a future spec/plan to pick between, per this
issue's status (`open`, no requirements/claims recorded yet):

1. **Script-engine escalation, scoped to the pack.** Add a small script-based engine binding to
   `go-distribution` (or a new `gate_type` shape) whose sole job is this one cross-file
   correlation: parse `.goreleaser.yml`'s `.Env.X` references, parse `release.yml`'s exported env
   names, diff them, emit SARIF. Keeps the capability inside the pack that needs it; costs the
   pack its first non-semgrep engine binding.
2. **Core-side cross-file capability.** Give `pkg/pack/engine` an input mode or scope kind that
   can hand an engine binding a *correlated group* of files (e.g. a goreleaser config plus its
   companion workflow) rather than one file at a time, so semgrep-shaped or other single-file
   engines can be composed by the pack author into a cross-file check without each pack building
   its own script runner. Bigger lift; benefits every future pack that needs the same shape of
   correlation, not just this one.

Whichever direction is chosen, the existing single-file WARNING rule and manual-verification note
should stay in place as defense-in-depth — this is an additive capability, not a replacement.

## References

- `issues/ISSUE-101-go-distribution-pack.issue.md` — requirement 5 ("HOMEBREW_TAP_TOKEN and the
  derived-env falsifier class"), the source of the hard-won requirement this issue tracks the
  residual of.
- `issues/ISSUE-087-ci-driven-release-pipeline.issue.md` — the half-published-release failure
  mode (binaries + GitHub Release published, Homebrew formula-publish step silently dead) that
  the shipped single-file rule covers.
- `plans/PLAN-ISSUE-101-go-distribution-pack.plan.yml` — "THE DERIVED-ENV DECISION" section (the
  escalation-ladder call), AMENDMENT 4 (why the WARNING tier didn't hold until ISSUE-104/105),
  and TASK-016 (which named this issue as a deliberate deferral).
- `issues/ISSUE-104-sarif-severity-descriptor-fallback.issue.md` and
  `issues/ISSUE-105-step-verdict-ignores-severity-without-policy-entry.issue.md` — both closed
  2026-07-29; the fixes that made the shipped WARNING mitigation non-blocking in fact, not just
  in intent.
- `backstop-ai/go-distribution` pack (`~/src/projects/backstop-go-distribution-pack`) —
  `rules/workflow/release-workflow.yml` (the `tap-token-not-exported` rule) and
  `recipes/go-release/payload/release.yml` (the manual cross-read note).
