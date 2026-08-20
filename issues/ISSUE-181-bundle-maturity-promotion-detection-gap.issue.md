---
title: "Bundle Maturity Promotion Detection Gap"
schema_version: issue/v1

issue:
  id: ISSUE-181
  title: "Bundle Maturity Promotion Detection Gap"
  type: technical-debt
  status: open
  created: "2026-08-20"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: safe
---

# Bundle Maturity Promotion Detection Gap

## Problem

There is no mechanism — gate dimension, validator, or documented PM-sweep habit — that detects
when a bundle's cited spec seeds all reach `implemented`/terminal status and flags the bundle
itself for a maturity-promotion review. A bundle can sit at `maturity: defined` (or `ready`)
indefinitely after the work it exists to track is actually done, because nothing loops back from
"spec delivered" to "bundle promotion decision is now ready to make."

This repo's own convention (CLAUDE.md, both global and project) is that maturity promotion is
deliberately founder-gated — the user drives promotion, not an agent. That makes a loud,
non-blocking *advisory* the right shape here, not an auto-promotion: nothing should force the
bundle forward, but something should say "this decision is waiting" instead of relying on the
founder to notice unprompted.

### Confirmed instance — BUNDLE-003

`bundles/BUNDLE-003-onboarding-experience.bundle.md` sits at `status.maturity: defined`. Its last
bundle-level edit was 2026-08-14/15 (commit `8cb40e3`, "post-SPEC-068 artifact hygiene"). Its
three spec seeds:

| Spec | Flipped to `implemented` | Commit |
|---|---|---|
| SPEC-068 (Trustworthy Green Guards) | 2026-08-14 | `255c7c2` family |
| SPEC-069 (Backstop Init) | 2026-08-15 | `ab7e57d` "spec+plan(SPEC-069): close out to implemented/completed" |
| SPEC-070 (Backstop Doctor) | 2026-08-15 | `52c3520` "spec+plan(SPEC-070): close out to implemented/completed" |

All three of BUNDLE-003's spec seeds were `implemented` by 2026-08-15. Nobody looped back to the
bundle file itself between then and 2026-08-20 — five days, spanning further unrelated commits
that still touched SPEC-069's prose (`3fdebfd`, 2026-08-18, "fix SPEC-069 prose") without anyone
noticing the bundle above it was now promotion-ready. It surfaced only because the founder asked
directly, during an unrelated backlog review, "how is init still open."

Verify directly: `git log -p --follow -- specs/SPEC-069-backstop-init.spec.md
specs/SPEC-070-backstop-doctor.spec.md` shows the `status: draft` → `status: implemented` diffs;
`git log --format="%h %ad %s" --date=short -- bundles/BUNDLE-003-onboarding-experience.bundle.md`
shows no commit touching the bundle file after 2026-08-15.

### No existing check covers this — confirmed, not assumed

Searched `pkg/gate/` and `pkg/validate/` for anything that cross-references bundle maturity
against its cited specs' completion state:

- `pkg/gate/requirement_traceability.go` (`ClassifyRequirementTraceability`,
  `SupportsCoverage`) checks REQ-level coverage — which specs cite which bundle REQs, and
  whether a spec's pin on a REQ has gone stale relative to the bundle's current REQ version. It
  never asks "have ALL of this bundle's cited specs reached `implemented`" at the bundle level.
- `pkg/gate/artifact_status.go` classifies each artifact's own status against its own mandated
  tests (drift step) — it does not compare a bundle's maturity to the aggregate state of the
  specs that source from it.
- No hit anywhere in `pkg/gate/` or `pkg/validate/` for a check joining bundle `maturity` to
  "all sourcing specs are implemented/terminal."

So this is a gap in kind — no check exists — not a case of an existing check failing to fire.

### Not unique to BUNDLE-003, but not currently duplicated either

Spot-checked every non-exploring, non-terminal bundle (`defined`/`ready`: BUNDLE-004 through
BUNDLE-010, BUNDLE-015, BUNDLE-031) for the same fully-covered state. None of the others
currently have 100% of their cited specs implemented — e.g. BUNDLE-007 (`ready`) has four spec
seeds and only one authored spec (`SPEC-019`, implemented), so its `ready` maturity is still
current. That's expected: this is a structural gap in the workflow, not a bug that happens to be
tripping on many bundles right now. It will recur on the next bundle whose last spec closes out
without anyone looping back — BUNDLE-003 is simply the first observed occurrence.

## Solution

Add a check — most naturally a new (or extended) gate dimension in `pkg/gate/`, joining bundle
records to the specs whose `source.bundle` cites them (the same join `requirement_traceability`
already performs for REQ coverage) — that, for every bundle NOT in a terminal maturity state,
tests whether all of its cited spec seeds are `implemented`/terminal. When true, emit a loud,
**non-blocking** advisory naming the bundle as promotion-ready, mirroring this repo's existing
"loud ≠ blocking" convention (`pkg/gate/policy.go` — advisories inform, never gate) and the same
warn-only posture `requirement_traceability`'s in-flight-gap advisory already uses.

Explicitly out of scope for the fix: auto-promoting maturity, or making the advisory blocking.
Maturity promotion stays founder-gated per this repo's CLAUDE.md — the check's job is only to
surface that a decision is waiting, not to make it.

An alternative (or complementary) shape worth considering at plan time: a documented PM-sweep
habit (`backlog-pm` agent) that performs the same cross-reference on trigger, rather than a gate
dimension. The tradeoff between "gate advisory, fires on every run" vs. "PM-sweep habit, fires on
triage" is a plan-level design decision, not one to pre-resolve here.
