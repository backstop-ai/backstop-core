---
title: "Recipe payloads have no merge/insert op — brownfield adoption of a CI gate recipe silently writes nothing"
schema_version: issue/v1

issue:
  id: ISSUE-119
  title: "Recipe payloads have no merge/insert op — brownfield adoption of a CI gate recipe silently writes nothing"
  type: enhancement
  status: open
  created: "2026-08-11"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: moderate
---

# Recipe payloads have no merge/insert op — brownfield adoption silently writes nothing

## Problem

The recipe applier (`pkg/recipe`, SPEC-054) only supports a `create` op family for scaffolding-kind
recipes, whose defining behavior is never-clobber: if the target file already exists, `create`
preserves the consumer's own file untouched and reports it as `preserved`. That is correct,
deliberately non-destructive behavior for a greenfield target — but it produces a genuinely bad
outcome on a brownfield one.

Surfaced during implementation of `backstop-ai/ci-workflows`'s four CI gate recipes
(SPEC-067/BUNDLE-015 Seed 6): three of the four platform targets — `.gitlab-ci.yml`,
`bitbucket-pipelines.yml`, and `Jenkinsfile` — are each their platform's ONLY conventional CI entry
point. Any consumer who already has one (the common case for an existing project adopting backstop,
as opposed to a fresh `init`) gets `create`'s never-clobber rule silently preserving their existing
file. The apply reports success (`preserved … (the consumer's own file)`) and records an adoption,
but the consumer ends up with **no backstop gate wired into their CI at all** — and nothing in the
apply's own output makes that failure-to-actually-adopt obvious at a glance.

Init's greenfield case (a project with none of these files yet) is unaffected — this is purely a
brownfield/pre-existing-structure gap.

## Direction

Not scoped here — SPEC-067 explicitly named this as a follow-on rather than improvising a fix
mid-implementation (Sharp Edge 2 / TASK-031 of `plans/PLAN-SPEC-067-ci-recipe-pack.plan.yml`). At
minimum, the eventual spec/plan should weigh:

1. A `merge` or `insert` op kind (or an `--if-exists` mode on `create`) that can extend an existing
   platform config with a backstop-gate job/step rather than only ever writing a whole new file –
   this is materially harder than `create` because it means editing bytes the recipe does not fully
   own, which is exactly why REQ-003 of SPEC-067 prohibited it for the initial cut ("a gate workflow
   is a whole file the recipe owns, and an op family that edits a consumer's existing file would put
   a recipe-owned promise inside consumer-owned bytes").
2. At a minimum, whether `create`'s never-clobber "preserved" outcome should surface louder —
   e.g. a distinct warning that no gate was actually wired — so a brownfield adopter doesn't walk
   away believing CI enforcement is live when it silently isn't.
3. Whether this belongs to the recipe-capability layer generically (any `scaffolding`-kind recipe
   targeting a platform's singular conventional entry point has the same problem) or is CI-recipe-
   pack-specific.

## Notes / references

- Cites `specs/SPEC-067-ci-recipe-pack.spec.md`, Sharp Edge 2 (implementation notes, ~line 1350):
  "On an existing project, three of the four recipes quietly do nothing... The consumer ends up with
  an adoption record and NO gate in CI. That is correct non-destructive behaviour and a genuinely
  bad outcome, and there is no `merge`-into-an-existing-pipeline op in this spec to fix it."
- Cites `plans/PLAN-SPEC-067-ci-recipe-pack.plan.yml` Scope Fences (~line 514) and TASK-031 (~line
  2524), which named this exact follow-on (slug `ci-recipe-brownfield-merge-variant`) as work to be
  filed rather than improvised during implementation.
- Governed by BUNDLE-015 (Recipe capability); this issue is scoped narrowly to the CI-recipe-pack
  instance of the gap, discovered during SPEC-067's implementation — the generic recipe-op-kind
  question (item 3 above) is left for whoever picks this up to decide the right home for.
