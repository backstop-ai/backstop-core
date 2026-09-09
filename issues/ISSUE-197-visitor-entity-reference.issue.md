---
title: "Land Approved Entity-Reference Pages"
schema_version: issue/v1

issue:
  id: ISSUE-197
  title: "Land Approved Entity-Reference Pages"
  type: enhancement
  status: closed
  created: "2026-09-02"
  closed: "2026-09-03"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
resolved-by: b77d07e38e2fa054d782c909a99984b6df8ca846
---

# Land Approved Entity-Reference Pages

## Problem

After Adopt, a visitor who wants the noun — Plan, Issue, Spec, Bundle, Pack —
has nowhere to look it up. `/reference/` is the exact-interface catalog
(schemas, CLI, config). `/model/` teaches the two work paths. Neither is a
lookup-by-noun page for one artifact.

A local prototype of that lookup layer exists on
`cursor/evaluate-copy-local-preview-e296` (HEAD `ea7f7d2`, `draft(docs): call
the orientation block Work path`). It has not been pushed. Live
`https://backstop.sh/` has none of these pages. This issue records the
approved direction so it can be planned and landed. The prototype is evidence
of intent, not a close and not a substitute for issue → plan.

The pages are one system: shared paper/ink `page_kind: entity`, one overlay,
one generator, one locked Plan page the generator skips. Splitting them into
per-file issues would be ceremony without a decomposition. They are not eight
visitor stories.

ISSUE-193 owns `/evaluate/` only. ISSUE-195 owns `/model/` only (filed on
`cursor/land-approved-model-visitor-page-2a53`, tag `backstop/issue/195`),
including the two-path legend. ISSUE-196 owns `/adopt/` only. This issue does
not absorb those pages.

Do not `/plan` this issue in this filing. Plans for ISSUE-193, ISSUE-195,
ISSUE-196, and this entity-reference issue happen later, together, when
Brandon says so.

## Solution

Land the approved entity-reference lookup pages: locked copy, paper/ink
entity chrome shared with Evaluate, Model, Adopt, and the homepage, the
overlay + generator that keeps schema-derived tables honest, and the smallest
layout / CSS lockstep needed so those extra pages render. Do not add them to
the Seed 1 10-page topology or primary nav in this landing.

Do not reopen BUNDLE-032's visitor-journey charter. Do not mark BUNDLE-032
delivered. Do not `/plan` this issue in this filing. Do not edit visitor copy
as part of filing this issue.

### Approved page contract

Job: lookup-by-noun after Adopt, not more onboarding. Copy is locked. Quote
the approved sentences; do not independently "improve" them.

Same paper/ink `page_kind: entity` on every page. Not in nav. Not in the Seed 1
10-page topology. Extra `docs/*.md` is outside closed-world `sources[]`.

#### Locked ledes (the four-sentence model)

- **Bundle:** A larger body of work. It decomposes into specs. Each spec becomes a plan.
- **Spec:** Requirements and claims for a slice of feature work. It comes from a bundle. It becomes a plan.
- **Issue:** Bounded work that does not need a bundle and spec. It becomes a plan.
- **Plan:** Ordered work an agent can execute. It comes from an issue or a spec.

Also generated, same grammar, not work-track:

- **Directive:** A backlog item. Position in BACKLOG.yml is priority. It cites the work that authorized it.
- **ADR:** A recorded decision. It is evidence for later work. It is not a path into implementation.
- **Capability:** What a user can do, proved end to end. Specs implement the pieces. The capability verifies the journey.
- **Pack:** A versioned set of standards, engines, and proof. Packs are not intent artifacts.

#### Work path (not "Sources")

Heading is **Work path** on Plan, Issue, Spec, and Bundle. Columns WORK | PATH.

- Plan has both rows:
  - Feature / substantial → Bundle → Spec → Plan
  - Small / reactive → Issue → Plan
- Issue: Small / reactive → Issue → Plan
- Spec: Feature / substantial → Bundle → Spec → Plan
- Bundle: Feature / substantial → Bundle → Spec → Plan
- Pack has **no** Work path section
- Directive, ADR, and Capability have **no** Work path section

Do not collapse the two tracks. Do not call this block Sources.

#### Taxonomy

**Intent/work artifacts:** Bundle, Spec, Issue, Plan.

**Enforcement:** Pack — KIND Pack, not Artifact. Locked constraints:

- `Packs are not intent artifacts.`
- `A pack is not a path into implementation.`
- `Implementation still requires an approved plan.`
- `Packs live outside core. The lock file is the durability boundary.`

`/pack/` (this lookup page) is not `/packs/` (the Seed 1 catalog).

#### Plan (`docs/plan.md`) — hand-locked, generator skips it

Traditional docs first: meta, Work path, Fields, Status, Phases, Constraints,
Validate, Reviewer, Notes. Notes keep:

- `A plan is the bound on implementation. Nobody writes product code until one is approved.`
- `Reactive work uses issue → plan. Feature work uses bundle → spec → plan. Both tracks meet here.`
- `The source artifact defines the work. The plan is the handoff. An implementer executes the approved plan and runs required gates as it works.`

Also-links include Issue, Spec, Bundle, Operating model, Artifact schemas.

Do not regenerate Plan.

#### Generated pages

Overlay: `docs/_data/entity-reference.yml`.

Generator: `go run ./scripts/entityref` writes pages; `go run ./scripts/entityref -check` fails on drift; test `go test ./scripts/entityref`.

Outputs: `docs/issue.md`, `docs/spec.md`, `docs/bundle.md`, `docs/directive.md`,
`docs/adr.md`, `docs/capability.md`, `docs/pack.md`.

Schema-derived: fields, enums, filename, schema id, required sections.

Overlay-owned: ledes, Notes, status meanings, which fields, reviewer,
constraints, Work path rows.

Pack is overlay-only (no in-repo pack schema).

Do not dump `enforcement` jargon. Do not hand-edit generated pages.

### Rejected — do not restore

- One issue per entity file
- Adding these pages to primary nav or the Seed 1 10-page topology in this landing
- Section heading **Sources** on work artifacts
- "First-class path" Issue lede
- Regenerating Plan from the overlay
- Dumping enforcement jargon onto the pages
- Slide-frame chrome, slogans, manufactured paradoxes
- Independent "improvement" of locked copy
- Collapsing Issue and Bundle tracks into one workflow
- Folding Evaluate, Model, Adopt, homepage, or Use Cases into this issue
- Treating `/pack/` as the Packs catalog (`/packs/`)
- Vendoring packs into core
- Wiring `entityref -check` into Seed 1 product-truth / sitecheck in this landing

### Presentation (paper/ink, shared with Evaluate, Model, Adopt, and homepage)

Landing must give entity pages the same paper/ink chrome as Evaluate, Model,
Adopt, and the homepage, without regressing those pages.

Prototype already:

- `page_kind: entity` on each entity page
- `docs/_layouts/default.html` loads `backstop-tokens.css` and ink theme-color
  when `page_kind` is `evaluation`, `model`, `adoption`, or `entity`
- `html:has([data-page-kind="evaluation"], [data-page-kind="model"], [data-page-kind="adoption"], [data-page-kind="entity"])`
  remaps `--ds-*`
- `docs/_includes/page-hero.html` uses `page.hero_lede` when set
- Entity type recipe in `docs/assets/css/site.css` (meta dl, tables, illegal
  box, also-links)

When editing `site.css` media queries, do not drop
`[data-page-kind="home"] .nav` rules. Playwright `testMatch` stays
`public-site.spec.ts`. Packs stay external.

### Lockstep landing will need

Keep the pages as extra `docs/*.md` outside closed-world `sources[]`. Do not
add them to `docs/_data/content-topology.yml` `pages[]` or
`navigation.primary` / `navigation.utility` in this landing. A later decision
may add them to the 10-page matrix; this issue must not self-promote them.

`go run ./scripts/entityref -check` is the drift guard for generated pages.
CI can later grow a product-truth job like the schema catalog. Do not wire
that into Seed 1 sitecheck in this landing.

Landing must ship the overlay, the generator, its test, the hand-locked Plan
page, the generated outputs, and the entity CSS / layout fallbacks together.
Do not land generated markdown without the generator.

Do not update SPECs in this filing — only record that Seed 1 contracts stay
on the 10-page topology and this lookup layer stays extra unless a later
issue changes that.

### Scope constraint

This issue owns the entity-reference lookup system: `/plan/`, `/issue/`,
`/spec/`, `/bundle/`, `/pack/`, `/directive/`, `/adr/`, `/capability/`, plus
the overlay, generator, `-check`, generator test, and the smallest CSS /
layout fallbacks those pages need.

Out of scope — later page-by-page issues, do not absorb:

- Homepage (ISSUE-190; canonical homepage stays)
- Evaluate (ISSUE-193)
- Model (ISSUE-195), including the two-path legend on `/model/`
- Adopt (ISSUE-196)
- Use Cases, Packs catalog (`/packs/`), Extend, Reference, Status,
  Contributing, and other inner-page copy
- Adding entity pages to the Seed 1 10-page topology or primary nav
- Wiring `entityref` into Seed 1 product-truth / sitecheck
- Logo / hamburger / mobile nav for remaining inner pages
- Marking BUNDLE-032 delivered
- Vendoring packs
- Pushing the local prototype without a plan
- `/plan` of this issue, ISSUE-193, ISSUE-195, or ISSUE-196 in this filing

## References

- Local prototype: branch `cursor/evaluate-copy-local-preview-e296`, HEAD
  `ea7f7d2`. Primary sources: `docs/plan.md` (hand-locked);
  `docs/_data/entity-reference.yml`; `scripts/entityref`; generated
  `docs/issue.md`, `docs/spec.md`, `docs/bundle.md`, `docs/pack.md`,
  `docs/directive.md`, `docs/adr.md`, `docs/capability.md`; CSS
  `docs/assets/css/site.css`; layout `docs/_layouts/default.html`; hero
  `docs/_includes/page-hero.html`.
- BUNDLE-032 / SPEC-072 own visitor-journey IA and the Seed 1 10-page
  topology. This issue does not reopen that charter. Entity pages are extra
  lookup, not a competing eleventh-through-eighteenth page of that matrix.
- ISSUE-190: homepage fence. This issue must not redesign `/`.
- ISSUE-192: architecture-pack classification of `scripts/websitejourney`;
  explicitly does not own inner-page visual redesign.
- ISSUE-193: Evaluate rewrite; lists other inner pages as out of scope.
- ISSUE-195: Model rewrite (branch `cursor/land-approved-model-visitor-page-2a53`,
  tag `backstop/issue/195`); owns the two-path legend on `/model/`.
- ISSUE-196: Adopt rewrite; owns `/adopt/` only.
- Live site still has no entity pages: `https://backstop.sh/`.

### Existence-in-world check

Before filing, `issues/` and `bundles/` were searched for entity-reference
pages, `/plan/` visitor copy, `/issue/` lookup, overlay generator
`scripts/entityref`, and "Work path" as a page heading.

- BUNDLE-032 / SPEC-072 own visitor-journey IA and Seed 1's ten pages. They
  do not own these extra lookup pages.
- ISSUE-190 owns canonical homepage restoration only. Must not redesign `/`.
- ISSUE-192 owns `scripts/websitejourney` classification; excludes inner-page
  visual redesign.
- ISSUE-193 owns `/evaluate/` only.
- ISSUE-195 owns `/model/` only.
- ISSUE-196 owns `/adopt/` only.
- No open issue in this working tree owns `/plan/` or the generated entity
  pages. Filing one issue for the system, not one issue per file.

## Resolution

Landed on `main` in PR #38 (`b77d07e38e2fa054d782c909a99984b6df8ca846`, 2026-09-03).
Issue file restored from `ff700d1`; reserved as `backstop/issue/197` but omitted
from the merge. Closes via `resolved-by`. `PLAN-ISSUE-197` remains `status: draft`.
