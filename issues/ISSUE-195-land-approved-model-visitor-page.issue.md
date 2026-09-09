---
title: "Land Approved Model Visitor Page"
schema_version: issue/v1

issue:
  id: ISSUE-195
  title: "Land Approved Model Visitor Page"
  type: enhancement
  status: closed
  created: "2026-09-01"
  closed: "2026-09-03"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
resolved-by: b77d07e38e2fa054d782c909a99984b6df8ca846
---

# Land Approved Model Visitor Page

## Problem

`/model/` on `main` is still the Seed 1 field-guide appendix. SPEC-072 locked its
hero as "How does Backstop turn intent into a trustworthy verdict?" and parked
NBR-005 (How Backstop Works) and NBR-006 (Canonical Concepts & Architecture) on
`/model/` with `required_blocks` `operating-model`, `delivery-lifecycle`,
`enforcement-loop`, and `ownership-boundaries`. The live page then catalogs
concepts and architecture — operating model, product category, intent artifacts,
work tracks, bounded execution, standards packs, recipes, gates, waivers,
capabilities, provenance, harness integration — rather than showing how Backstop
actually operates for someone shipping.

Visitor review found that framing wrong. The Seed 1 page does the wrong visitor
job. It leads with a catalog question, then a field-guide appendix. It does not
answer the question a visitor actually has: how the work is defined, how
standards are enforced, and how drift is detected while the agent is working.
Positioning is additive: it does not replace SDD, CI, Cursor/Claude Code, MCP,
skills, or standards docs. Packs stay external.

The Seed 1 appendix must not become the visitor page again. Hidden canonical
claim and journey-link anchors remain for Seed 1 topology; they must stay
visually hidden.

A local prototype of the approved rewrite exists on
`cursor/evaluate-copy-local-preview-e296` (HEAD `4366e98`, `draft(docs): give
Model diagram boxes a little more air`). It has not been pushed. Live
`https://backstop.sh/model/` is still the Seed 1 field-guide page. This issue
records the approved direction so it can be planned and landed. The prototype is
evidence of intent, not a close and not a substitute for issue → plan.

ISSUE-193 owns `/evaluate/` only and lists Model as out of scope for a later
page-by-page issue. This is that later issue.

## Solution

Land the approved `/model/` visitor page: copy, paper/ink presentation shared
with Evaluate and the homepage, and the inventory / topology / allowance /
presentation / page-test / Seed 1 contract lockstep, including Seed 1 contract
literals in SPEC-072 / SPEC-075 / SPEC-076 and related sitecheck expectations
that still name the old hero and require hidden canonical headings to be
visitor-visible.

Do not reopen BUNDLE-032's visitor-journey charter. Do not mark BUNDLE-032
delivered. Do not `/plan` this issue in this filing.

### Approved page contract

Visible structure of `docs/model.md`. Lock this structure. Do not rewrite or
restructure. Copy is locked.

1. Hero: `hero_question` **How it works** (not "How does it work?" and not the
   Seed 1 verdict question). `hero_lede` **Define the work. Enforce your
   standards. Detect drift.** Same pair in `docs/_data/site-presentation.yml`
   and `docs/_data/content-topology.yml`. Topology `required_blocks` stay
   `[operating-model, delivery-lifecycle, enforcement-loop, ownership-boundaries]`.
   Neighborhoods NBR-005, NBR-006.
2. `## Define the work {#operating-model}`
   - Lede: `One bundle. Many specs. One plan per spec.`
   - Two-sided `.work-topology`: left How work is structured (Bundle fan to
     Spec/Plan A–D); right How work is delivered (A and B stacked, brace into
     C, arrow to D).
   - Caption: `Independent branches can execute in parallel. Dependencies
     establish the order when they cannot.`
   - Exception, not a peer caption:
     `<p class="work-exception"><span>For small fixes</span>A bounded fix skips
     the bundle and spec. The issue becomes a plan.</p>`
   - Hidden CLAIM-022 + JLINK-010.
3. `## Enforce your standards {#standards-packs}`
   - Lede: `Packs own standards, engines, and proof.`
   - Pack figure: Pack fans to Rules / Engines / Fixtures (dashed, no line into
     Gate). Only Rules + Engines collect into Gate: `Runs the pack's required
     checks`.
   - Bridge, not a second H2:
     `<p class="figure-bridge">Packs compose across domains and toolchains.</p>`
   - Compose figure: Code/Go, Architecture/Package bounds, Design/CSS and
     tokens → Gate `Executes all rules across all packs`.
   - Recipes stay off this page. Hidden `## Recipes {#recipes}` remains in
     canonical-anchors.
4. `## Detect drift {#baselines-and-ratchets}`
   - Lede: `Existing debt is not permission for new debt.`
   - Ratchet: Unchanged `Visible. Does not fail.` vs New or modified `Must meet
     the standard.` (not "touched"). Caption: `A waiver is named and
     time-bounded. The rule stays.`
   - Lede: `A claim names what must exist and the tests that prove it.`
   - Claim — mandates — Tests `Must exist and be real`. A claim is a spec
     artifact (`claims[]` with id, requirement, text, tests), not what the
     agent says it did. Do not restore a provenance fan (Source/Command/Tests).
5. `## While the agent is working {#enforcement-loop}`
   - Lede: `The agent does not hand failed work downstream.`
   - Flow: Plan → [Implement ↔ Gate] × N → Implementation review ↔ Fix →
     Commit / Push → CI
   - `.loop-core` brackets Implement + Gate + fail/fix; `× N` is a legend on
     that bracket.
   - Gate fail/fix loops to Implement (current chunk). Implementation review
     has its own fix loop (completed work, not human PR review). Then
     Commit / Push, then CI.
   - Caption on CI: `CI confirms. It does not discover.`
   - Hidden ARCH-002 note + JLINK-014.
6. `## What Backstop does not own {#ownership-boundaries}` — prose closer, no
   diagram
   - `Bring your agent, your harness, your standards, and your tools. Backstop
     does not own your source or your data.`
   - `Backstop keeps the work inside the boundaries you define, so you can be
     confident you're getting the software you think you are.` (accent closer,
     not "make agents behave in a disciplined way")
   - Hidden CLAIM-010, ARCH-003 prose, JLINK-011.
7. `.canonical-anchors` (visually hidden, not visitor copy): Intent artifacts,
   Work tracks, Bounded execution, Recipes, Gates and policy (CLAIM-011,
   JLINK-006), Waivers, Capabilities and journeys, Provenance (JLINK-021),
   Harness (CLAIM-019, JLINK-008), Product category (CLAIM-020, JLINK-005),
   Delivery lifecycle (CLAIM-023).

Rejected — do not restore:

- Core / Packs / Harness inventory cards
- Wrap diagram "Backstop sits around the agent"
- Legal inventory (harness, language model, Semgrep, native toolchains, custom
  scripts, source, data as a list)
- "make agents behave in a disciplined way"
- Hero "How does it work?"
- Slide-frame / PowerPoint cards around diagrams (`background`/`border` on
  `.model-figure` / `.work-topology`)
- Equal-peer feature lists; flattening two relationships into one visual;
  equal-weighting the exception; "touched" instead of "new or modified";
  provenance as Detect drift hero; Recipes on this page; shift-left /
  enterprise DevOps sludge; naming a new standard as a new pack on this page

Learned visual rules to preserve: subject is visual mass; tee up with a lede
then a mechanic labeled as meaning; don't catalog; fixtures dashed off the
gate collect path; two-pane chrome only when the two sides are genuinely
different relationships.

### Presentation (paper/ink, shared with Evaluate)

ISSUE-193 said other inner pages stay dark until their own issues. This is
Model's issue. Landing must give `/model/` the same paper/ink chrome as Evaluate
and the homepage, without regressing Evaluate or the canonical homepage.

- `docs/_layouts/default.html`: when `page_kind` is `evaluation` **or**
  `model`, load `backstop-tokens.css`, `theme-color #0c0d0d`, no
  `color-scheme: dark`.
- `html:has([data-page-kind="evaluation"], [data-page-kind="model"])` remaps
  `--ds-*` to paper/ink tokens.
- Wordmark `./b` ink pill; nav ink 0.84rem; current-page underline
  `--color-signal`.
- Diagrams sit on paper (no slide frames). Type: loud H1/H2 then muted ledes
  1.15rem. Ownership second paragraph is accent closer (green, weight 650),
  like Evaluate's table closers.
- Reflow diagrams by viewport (desktop / laptop / tablet / mobile). Do not
  shrink diagram text until unreadable. Keep fixtures beside the gate collect
  path (dashed, not under Gate) when there is width; 1-col stack only on the
  narrowest breakpoint.
- Slightly airy box gaps (~0.6rem). Bounded-fix exception and compose bridge
  treatments stay.
- When editing the `site.css` media query, do not drop
  `[data-page-kind="home"] .nav` responsive rules.
- Playwright `testMatch` stays `public-site.spec.ts`. Packs stay external.

### Lockstep landing will need

Closed-world: every non-claim, non-jlink, non-heading block in `docs/model.md`
is an exact string in `allowances` in `scripts/verify-public-product-model.sh`.
Heading text skipped.

Hero / presentation matrix: `docs/_data/site-presentation.yml`,
`docs/_data/content-topology.yml`, `scripts/verify-public-product-model.sh`
heroes list, `scripts/tests/public-product-model/verify-content-topology.sh`,
`scripts/sitecheck/presentation_test.go`. Seed 1 still names "How does Backstop
turn intent into a trustworthy verdict?" in SPEC-072 / SPEC-075 / SPEC-076 —
landing must update those contract literals the same way ISSUE-193 called out
SPEC-072 / SPEC-075 / SPEC-076 for Evaluate.

Public-site completeness currently requires `#delivery-lifecycle` (and other
hidden canonical heading ids) `toBeVisible()`. The prototype hides them via
`.canonical-anchors` / `.canonical-note` clip. Landing must keep those anchors
for topology without making the Seed 1 appendix the visitor page again — update
sitecheck / public-site tests as needed, do not un-hide the appendix.

### Scope constraint

This issue owns `/model/` only, plus the smallest corresponding inventory,
topology, allowance, presentation, page-test, and Seed 1 contract updates
needed to land that page without silent divergence from SPEC-072 / SPEC-075 /
SPEC-076.

Out of scope — later page-by-page issues, do not absorb:

- Homepage (ISSUE-190; canonical homepage stays)
- Evaluate (ISSUE-193)
- Adopt, Status, Reference, and other inner-page copy
- Logo / hamburger / mobile nav for remaining inner pages
- Marking BUNDLE-032 delivered
- Vendoring packs
- Pushing the local prototype without a plan
- `/plan` of this issue in this filing

## References

- Local prototype: branch `cursor/evaluate-copy-local-preview-e296`, HEAD
  `4366e98`. Primary source `docs/model.md`; CSS `docs/assets/css/site.css`;
  layout `docs/_layouts/default.html`; lockstep in
  `scripts/verify-public-product-model.sh` and presentation/topology YAML.
- SPEC-072 Seed 1 Model hero "How does Backstop turn intent into a trustworthy
  verdict?", NBR-005 + NBR-006, and `required_blocks` operating-model,
  delivery-lifecycle, enforcement-loop, ownership-boundaries. Approved visitor
  hero is "How it works", not "How does it work?" and not the Seed 1 verdict
  question.
- BUNDLE-032 REQ-010: the bundle does not contain final copy; Seed 1 owned it.
  This issue revises Model's Seed 1 copy after visitor review. It is not a
  competing charter for the website expansion.
- ISSUE-190: homepage fence. This issue must not redesign `/`.
- ISSUE-192: architecture-pack classification of `scripts/websitejourney`;
  explicitly does not own inner-page visual redesign.
- ISSUE-193: Evaluate rewrite; this issue is the Model sibling. ISSUE-193 lists
  Model as out of scope.
- `scripts/verify-public-product-model.sh` closed-world allowances.
- Live site still old: `https://backstop.sh/model/`.

### Existence-in-world check

Before filing, `issues/` and `bundles/` were searched for Model visitor copy,
`/model/` rewrite, inner-page visual redesign, paper/ink Model restyle, and
"How it works" / "How does it work?" hero ownership.

- BUNDLE-032 / SPEC-072 own the visitor-journey IA and Seed 1 Model as NBR-005
  + NBR-006 with the Seed 1 hero and required_blocks operating-model,
  delivery-lifecycle, enforcement-loop, ownership-boundaries. This issue does
  not reopen that charter. It records the approved rewrite of that one page
  after visitor review.
- ISSUE-190 owns canonical homepage restoration only.
- ISSUE-192 owns architecture-pack classification of
  `scripts/websitejourney` and explicitly excludes inner-page visual redesign.
- ISSUE-193 owns `/evaluate/` only and lists Model as out of scope for a later
  page-by-page issue. This is that later issue.
- No open issue owns this Model rewrite. Remaining inner pages (Adopt, Status,
  Reference, …) stay unowned so they can be filed page by page.

## Resolution

Landed on `main` in PR #38 (`b77d07e38e2fa054d782c909a99984b6df8ca846`, 2026-09-03).
Issue file restored from the visitor-docs branch (`9750ce9`); it was reserved as
`backstop/issue/195` but omitted from the merge. Closes via `resolved-by`.
`PLAN-ISSUE-195` remains `status: draft`.
