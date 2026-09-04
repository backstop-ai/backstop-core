---
title: "Land Approved Extend Visitor Page"
schema_version: issue/v1

issue:
  id: ISSUE-198
  title: "Land Approved Extend Visitor Page"
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

# Land Approved Extend Visitor Page

## Problem

`/extend/` on `main` / live `https://backstop.sh/extend/` is still the Seed 1
stub. SPEC-072 locked `required_blocks` `[pack-or-not, author-a-pack,
path-filter-diagnostics]`, hero "When should this concern become a pack?",
JLINK-019 from `#pack-or-not` → `/reference/#pack-artifact`, JLINK-020 from
`#author-a-pack` → `/contributing/#contribution-paths`, and PACK-004 on
`#path-filter-diagnostics`.

Visitor review found that stub does the wrong job. Three short paragraphs tell
the reader to scaffold, bind claims, and watch path filters. They do not walk
a person who already has a standard through the real authoring loop.

The approved page job is:

I have a standard → should this become a pack? → if yes, how do I author one
correctly?

A local prototype of the approved rewrite exists on
`cursor/evaluate-copy-local-preview-e296`. This issue records the approved
direction so it can be planned and landed. The prototype is evidence of
intent, not a close and not a substitute for issue → plan.

ISSUE-193 owns `/evaluate/`. ISSUE-195 owns `/model/`. ISSUE-196 owns
`/adopt/`. ISSUE-197 owns the entity-reference lookup system, including
`/pack/`. This issue owns `/extend/` only.

Do not `/plan` this issue in this filing. Plans wait until Brandon approves
the copy.

## Solution

Land the approved `/extend/` visitor page: copy, paper/ink presentation shared
with Evaluate, Model, Adopt, and entity pages, and the inventory / topology /
allowance / presentation / page-test / Seed 1 contract lockstep, including
Seed 1 contract literals in SPEC-072 / SPEC-075 / SPEC-076 that still name
the old authoring paragraphs.

Do not reopen BUNDLE-032's visitor-journey charter. Do not mark BUNDLE-032
delivered. Do not `/plan` this issue in this filing.

### Approved page contract

Visible structure of `docs/extend.md`. Lock this structure. Do not rewrite or
restructure without a new review. Quote the approved job; do not independently
"improve" it.

Hero (keep):

- `hero_question`: **When should this concern become a pack?**
- Same string in `docs/_data/site-presentation.yml` and
  `docs/_data/content-topology.yml`.
- Topology `required_blocks`: `[pack-or-not, author-a-pack, path-filter-diagnostics]`

Page job, in order:

1. `## 1. Pack or not {#pack-or-not}`
   - Opening decision point. Keep the Seed 1 decision sentence:
     `Create a pack when a concern is deterministic, reusable across repositories, independently versionable, and owned by a maintainable standard. Keep repository-specific wiring local when reuse would manufacture an abstraction without a real consumer.`
   - If no: keep the standard in the repository.
   - If yes: continue to authoring.
   - JLINK-019 stays in this section:
     `[Inspect the pack artifact](/reference/#pack-artifact)`
   - Noun lookup may link to `/pack/`. Consumer chooser may link to
     `/packs/#choose-a-pack`. Those are not this page's job.

2. `## 2. Author a pack {#author-a-pack}`
   - Sequential walkthrough of the live authoring loop. Use real commands.
     Do not invent a publish CLI.
   - Scaffold: `backstop pack new --type engine --language go --slug <slug>`
     (types are `engine`, `mechanism`, or `toolchain`; the scaffold writes
     `pack.yml`).
   - Define claims, engines, and fixtures in the pack. Negative fixtures
     prove the finding. Exact manifest fields live on
     `/reference/#pack-artifact` — link there, do not duplicate the schema.
   - `backstop pack check ./<slug>`
   - `backstop pack test ./<slug>`
   - Try it in a consumer repository with `backstop pack add ./<slug>`, then
     the consumer gate. This is authoring dogfood, not Adopt.
   - JLINK-020 stays in this section:
     `[Contribute the pack](/contributing/#contribution-paths)`
   - Subheadings under authoring must not carry `{#id}` anchors that would
     steal JLINK-020's source_anchor.

3. `## 3. Path-filter diagnostics {#path-filter-diagnostics}`
   - Keep the PACK-004 needle `slash-bearing include or exclude pattern`.
   - Keep `pack check` and `pack test` as the commands that surface
     path-scope advisories.

Intended experience: decide pack-or-not → scaffold → bind claims/engines/fixtures
→ check → test → consume locally → contribute. The page itself is the
authoritative instruction list.

### Rejected — do not restore

- Seed 1 three-paragraph stub as the page job
- Moving pack authoring onto Adopt
- Turning `/pack/` into an authoring tutorial
- Turning `/packs/` into authoring
- Duplicating the pack manifest schema on `/extend/`
- Inventing `backstop pack publish` or a fourth Adopt locked command
- Slide-frame chrome, slogans, manufactured paradoxes
- Independent "improvement" of locked copy
- Folding ISSUE-199 (artifact lifecycle) into this issue

### Presentation (paper/ink)

`page_kind` is already `extension` in `docs/_data/site-presentation.yml`.
Landing must give `/extend/` the same paper/ink chrome as Evaluate, Model,
Adopt, and entity pages, without regressing those pages.

When editing `site.css` media queries, do not drop
`[data-page-kind="home"] .nav` rules. Playwright `testMatch` stays
`public-site.spec.ts`. Packs stay external.

### Lockstep landing will need

Closed-world allowances in `scripts/verify-public-product-model.sh` for every
non-claim, non-jlink, non-heading block in `docs/extend.md`. Heading text is
skipped.

Hero / presentation matrix: `docs/_data/site-presentation.yml`,
`docs/_data/content-topology.yml`, page test
`scripts/tests/public-product-model/pages/extend-reference-contributing.sh`.

PACK-004 deletion needle `slash-bearing include or exclude pattern` must remain
on `/extend/#path-filter-diagnostics`.

Seed 1 still names the old authoring paragraphs in SPEC-072 / SPEC-075 /
SPEC-076. Landing must update those contract literals the same way ISSUE-193
called out for Evaluate. Do not update SPECs in this filing — only record
that landing will.

JLINK-019 and JLINK-020 marker/link/source-anchor cardinality stays 1.

### Scope constraint

This issue owns `/extend/` only, plus the smallest corresponding inventory,
topology, allowance, presentation, page-test, and Seed 1 contract updates
needed to land that page without silent divergence from SPEC-072 / SPEC-075 /
SPEC-076.

Out of scope — do not absorb:

- Homepage (ISSUE-190)
- Evaluate (ISSUE-193)
- Model (ISSUE-195)
- Adopt (ISSUE-196)
- Entity lookup, including `/pack/` (ISSUE-197)
- Artifact lifecycle / state machine on `/reference/#artifact-lifecycle-and-closure` (ISSUE-199)
- Packs catalog (`/packs/`)
- `/reference/#pack-artifact` schema dump
- `/contributing/` rewrite
- Use Cases
- Vendoring packs
- `/plan` of this issue in this filing

## References

- Local prototype: branch `cursor/evaluate-copy-local-preview-e296`. Primary
  source `docs/extend.md`.
- Live authoring loop: `cmd/backstop/pack_authoring_loop_test.go` —
  `pack new` → `pack check` → `pack test` → `pack add` → consumer `gate`.
- SPEC-072 Seed 1 Extend `required_blocks` `[pack-or-not, author-a-pack,
  path-filter-diagnostics]`, JLINK-019, JLINK-020, PACK-004.
- BUNDLE-032 REQ-010: the bundle does not contain final copy; Seed 1 owned it.
- ISSUE-196: Adopt rewrite; lists pack authoring as out of scope.
- ISSUE-197: `/pack/` noun page; not an authoring tutorial.
- Live site still old: `https://backstop.sh/extend/`.

### Existence-in-world check

Before filing, `issues/` and `bundles/` were searched for Extend visitor copy,
pack-authoring guide, and `/extend/` rewrite.

- BUNDLE-032 / SPEC-072 own visitor-journey IA and Seed 1 Extend. This issue
  does not reopen that charter. It records the approved rewrite of that one
  page after visitor review.
- ISSUE-196 owns `/adopt/` and forbids moving pack authoring there.
- ISSUE-197 owns `/pack/` as the noun page.
- No open issue in this working tree owns the Extend rewrite.

## Resolution

Landed on `main` in PR #38 (`b77d07e38e2fa054d782c909a99984b6df8ca846`, 2026-09-03).
Issue file restored from `88f1f02`; reserved as `backstop/issue/198` but omitted
from the merge. Closes via `resolved-by`. `PLAN-ISSUE-198` remains `status: draft`.
