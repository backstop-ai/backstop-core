---
title: "Spec Sourcing Below Success Criteria Bar"
schema_version: issue/v1

issue:
  id: ISSUE-203
  title: "Spec Sourcing Below Success Criteria Bar"
  type: enhancement
  status: open
  created: "2026-09-11"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# Spec Sourcing Below Success Criteria Bar

## Problem

Nothing in `backstop artifact validate` or `backstop gate` checks the maturity or content of the
bundle a spec is sourced from. A spec can be authored, planned, implemented, reviewed, and
flipped to `status: implemented` — with its contracts dispatched by the gate — from a bundle that
has never had a success criterion or a design assumption written down.

### The gate the bundle maturity check does NOT close

`validateMaturityGates` (`pkg/validate/bundle.go:365-396`) is the only place `problem.success_criteria`
and `solution.assumptions` are required, and it requires them ONLY at `ready`:

```go
// Frontmatter path requirements for defined maturity
definedPaths := []string{
    "problem.summary", "problem.user_story", "solution.approach",
}
// Additional paths required only at ready
readyPaths := []string{
    "problem.success_criteria", "solution.assumptions",
}
```

(`pkg/validate/bundle.go:374-380`). At `defined`, a bundle needs `problem.summary`,
`problem.user_story`, `solution.approach`, and the mature-maturity sections — nothing that says
what "done" looks like or what the design is betting on.

Separately, nothing cross-references a spec against its source bundle's maturity or content at
all. A spec's only structural link back to its bundle is per-requirement, via the optional
`supports` field (format `bundle-name:REQ-NNN`, `artifacts/spec/v1/schema.json:63`,
`pkg/validate/spec.go:465-492`) — a requirement-to-requirement trace, not a bundle-readiness
check. Grepping `pkg/validate/` and `pkg/gate/` for anything that reads a spec's source bundle's
`status.maturity` or `problem`/`solution` blocks returns nothing (confirmed by search, not
assumed absent). The planner, `impl-reviewer`, and the gate all work forward from the spec's own
`requirements[]` and claims — none of them look back up at the bundle those claims descended
from.

Net effect: a bundle can sit at `defined` — or even `exploring`, since nothing gates spec
authoring on maturity either — with no success criteria and no assumptions, and every spec sourced
from it can ride all the way to `implemented` without that gap ever surfacing as a validator error,
a gate violation, or an advisory.

### Confirmed instance — `bclabs-brief-builder`, observed 2026-09-12

Two bundles shipped their entire spec lineage this way:

| Bundle | Maturity | `success_criteria` / `assumptions` | Specs sourced from it | Spec status |
|---|---|---|---|---|
| BUNDLE-008 (canary-deploy-smoke-then-promote) | `defined` | absent (checked: `problem:`/`solution:` blocks present, no `success_criteria` or `assumptions` keys) | SPEC-021 (Seed A), SPEC-022 (Seed B), SPEC-023 (Seed C), SPEC-024 (Seed D) | all four `implemented` |
| BUNDLE-009 (backfill-capabilities-and-journeys) | `defined` | absent (same check) | SPEC-020 (Seed 1) | `implemented` |

Verified directly in that repo:

- `grep -n "success_criteria\|^problem:\|assumptions" bundles/BUNDLE-008-canary-deploy-smoke-then-promote.bundle.md bundles/BUNDLE-009-backfill-capabilities-and-journeys.bundle.md` returns only the two `problem:` block openers — no `success_criteria` or `assumptions` key in either file.
- Promoting either bundle's `status.maturity` to `ready` (tested against BUNDLE-008) produces, verbatim:
  ```
  ✗ [bundle/maturity-gate] problem.success_criteria is required at 'ready' maturity
  ✗ [bundle/maturity-gate] solution.assumptions is required at 'ready' maturity
  ```
  while the same bundle validates clean at `defined` with neither block present.
- `SPEC-021-canary-candidate-smoke-gate.spec.md:10`, `SPEC-024-release-pipeline-home.spec.md:10`,
  and `SPEC-020-core-loop-journeys-and-capabilities.spec.md:10` each cite their BUNDLE-008/009 spec
  seed by name as their source; all five specs carry `status: implemented`
  (`grep -n "status:" specs/SPEC-020*.spec.md specs/SPEC-021*.spec.md specs/SPEC-022*.spec.md
  specs/SPEC-023*.spec.md specs/SPEC-024*.spec.md`).

Every earlier bundle in that repo had written its success criteria and assumptions at `defined`
by habit — the consumer repo's own `AGENTS.md`/bundle convention notes `defined` as the maturity
specs are sourced from in practice, not `ready`. BUNDLE-009's own frontmatter note documents this
precedent explicitly ("`defined` suffices to source a spec in this repo — SPEC-011 and SPEC-012
were each authored from [bundles held at `defined`]"). So the habit, not the framework, was what
had been holding the line for every bundle before these two. When the habit lapsed twice in a row,
nothing mechanical caught it.

The `spec-reviewer`'s spec-vs-bundle coverage check passed for all five specs — it compares a
spec's `requirements[]` against the bundle's `requirements[]` and open questions, both of which
existed. There is no product-level "what does done look like" for it to compare against, because
the bundle never had one. BUNDLE-009's own stated premise was "the repo can carry thousands of
green unit tests and a passing gate on the same commit that ships a product whose core loop is
broken" (`bundles/BUNDLE-009-backfill-capabilities-and-journeys.bundle.md` status note) — and it
shipped its spec without ever writing a measurable definition of success for the work meant to
fix that.

### Founder's framing (verbatim, from the triggering conversation)

> "this probably needs to be a core framework validation update but you need to act as the arbiter.
> this framework is too big for me to hold in my head at any given time."

The framework should hold this line mechanically — a conductor's label-level check ("bundle
exists, spec cites it") is not sufficient, and evidently was not sufficient twice in the same
repo.

## Solution

Not designed here — the following are candidate shapes for the planner to weigh, with trade-offs,
not a decision:

1. **Spec-side validation.** When a spec's cited source bundle resolves to a bundle lacking
   `problem.success_criteria` (and/or `solution.assumptions`), emit a WARNING at `status: draft`
   and an ERROR at `status: implemented` (or whatever status is the contract-dispatch boundary).
   Rationale: shipping an implementation with no measurable definition of done is exactly the
   failure the bundle → spec → plan → implementation chain exists to prevent — the same rationale
   `validateMaturityGates` already applies to `ready`, just never checked from the spec side.
   Trade-off: requires spec validation to resolve and load the bundle file, which today it does
   not do at all (currently only requirement-level `supports` strings are checked, never the
   bundle's own frontmatter).
2. **Move the bar earlier in the bundle maturity gate.** Require `problem.success_criteria` at
   `defined` instead of `ready` (leaving `solution.assumptions` at `ready`), since `defined` is
   the maturity specs are actually sourced from in practice (per BUNDLE-009's own note above).
   Trade-off: this is a breaking change to every currently-`defined` bundle across every consumer
   repo that has not yet written success criteria — needs a migration story, not just a schema
   bump. Also does not close the gap by itself unless spec sourcing is *also* gated on maturity
   (see open question below), since nothing currently stops a spec from being sourced from an
   `exploring` bundle either.
3. **Gate advisory instead of a validator error.** An `artifact_status_drift`-style advisory
   (non-blocking, in the spirit of ISSUE-201's severity philosophy) that lists every
   `implemented` spec whose source bundle is below the criteria bar, surfaced on every gate run
   until someone addresses it. Lower-friction than (1) but relies on someone reading the gate
   output rather than blocking the flip to `implemented`.
4. **Documentation.** State explicitly, in the artifact-workflow docs, which maturity a spec may
   legitimately be sourced from. Today the docs are silent on this — the consumer repo's own
   reference documentation is silent too, and precedent (habit) filled the gap both times. Worth
   doing regardless of which validation shape is chosen, since (1)-(3) all depend on there being a
   documented answer to "sourced from `defined` is fine" vs. "sourced from `defined` is
   already a violation."

Open question for the planner: does closing this gap require a spec-to-bundle maturity floor
(spec sourcing itself blocked below some maturity), or only a success-criteria/assumptions
content check independent of maturity label? The BUNDLE-009 precedent note suggests the consumer
repo's working convention already treats `defined` as sourcing-eligible — so option (2) alone,
without moving the criteria requirement earlier, would not have caught this instance.

## Related

- `pkg/validate/bundle.go:365-396` (`validateMaturityGates`) — the `bundle/maturity-gate` rule
  this issue asks to extend or front-load; lines 374-380 are the exact `definedPaths`/`readyPaths`
  split this issue is about.
- `artifacts/spec/v1/schema.json:63` and `pkg/validate/spec.go:465-492` — the only existing
  spec-to-bundle structural link (`requirements[].supports`), confirmed to be requirement-level
  only, not a bundle-readiness check.
- `ISSUE-201` (`baseline-comparison-severity-blind`) — the other consumer-observed gate gap filed
  from the same repo in the same week: a mechanical check existed but was miscalibrated (severity-
  blind verdict). This issue is the sibling failure mode — no mechanical check exists at all for
  the surface it covers. Same lesson each time: a step or gap not gate-enforced eventually gets
  silently skipped or missed, however well-intentioned the manual habit that was covering it.
