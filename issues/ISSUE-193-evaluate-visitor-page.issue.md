---
title: "Land Approved Evaluate Visitor Page"
schema_version: issue/v1

issue:
  id: ISSUE-193
  title: "Land Approved Evaluate Visitor Page"
  type: enhancement
  status: open
  created: "2026-08-31"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Land Approved Evaluate Visitor Page

## Problem

`/evaluate/` on `main` is still the Seed 1 field-guide page. SPEC-072 locked its
hero as "Is Backstop the right control surface for this problem?" and parked
category, fit, guarantee, compatibility, and receipt claims on Evaluate
(CLAIM-020, CLAIM-018, CLAIM-011, CLAIM-006, CLAIM-021, CLAIM-012, CLAIM-007,
CLAIM-008), with journey links sourced from appendix anchors
(`#what-backstop-is`, `#not-a-fit`, `#guarantees`, `#compatibility`,
`#compatibility-limits`).

Visitor review found that framing wrong. People already have Cursor, Claude
Code, or Fable. The problem is not generating code. Bigger models write great
public-majority code and cannot know how a specific team writes code. Backstop
makes agent output trustworthy by enforcing the team's standards and the
artifact chain. Positioning is additive: it does not replace SDD, CI, the
coding agent, MCP, skills, or standards docs.

The Seed 1 page therefore does the wrong visitor job. It leads with a
control-surface question, then a field-guide appendix. It does not answer the
question a visitor actually has: the agent already writes the code — why does
that output still need a stop.

A local prototype of the approved rewrite exists on
`cursor/evaluate-copy-local-preview-e296` (HEAD `94464f4`). It has not been
pushed. Live `https://backstop.sh/evaluate/` is still the Seed 1 page. This
issue records the approved direction so it can be planned and landed. The
prototype is evidence of intent, not a close and not a substitute for
issue → plan.

## Solution

Land the approved `/evaluate/` visitor page: copy, Evaluate-only paper/ink
presentation, and the claim / journey-link / inventory / allowance / test
lockstep, including Seed 1 contract literals in SPEC-072 / SPEC-075 / SPEC-076
and related sitecheck expectations that still name the old heroes and hops.

### Approved page contract

Visible structure of `docs/evaluate.md`:

1. Hero (`hero_question` + `hero_lede`): "Your agent already writes the code." /
   "Backstop helps you ship confidently." Same pair in
   `docs/_data/site-presentation.yml`. Topology `required_blocks`:
   `working-state`, `failure-fit`, `fit-decision`.
2. Kicker, not an H2: "Bigger models write great code. None of them write code
   like you." / "Backstop enforces your standards so the agent's code looks like
   your code."
3. Five-column tools table (`.tactics-matrix`): Markdown specs, Skills / MCP /
   LLM review / Standards as docs. Columns: What you already use, What that
   gets you, What it cannot guarantee, What Backstop adds, Result.
4. `## A spec is context. The artifact chain is working state. {#working-state}`
   with lede "SDD gives the agent better instructions. The artifact chain gives
   it less to guess." Four-column SDD table (`.tactics-matrix.sdd-matrix`).
   Closer: "The same structure that makes the agent easier to trust also makes
   the agent better at the job." JLINK-002 "See the operating model" →
   `/model/#operating-model`.
5. `## When work is not allowed to ship {#failure-fit}` — failed `backstop gate`
   card (tests and requirements fail, exit 1). No lede under this H2.
6. `## CI is too late to find problems {#fit-decision}` — "Fix problems while
   your agent is still writing the code." Typical vs Backstop workflow cards.
   Closer: "CI should confirm, not discover." JLINK-004 "Install Backstop" →
   `/adopt/#install`. The unfinished heading "CI is too late to find out" is
   rejected; the heading names what arrives too late: problems.

Rejected copy, do not restore: "control surface"; leading with "better models
plus markdown are not determinism"; naming Claude Code or Fable in the lead;
using "A better model writes more average code" as the kicker (the idea
remains; the copy is the great-code / like-you pair).

### Claim and journey-link relocation

The Seed 1 Evaluate appendix is gone (what it is, when not to, guarantees,
compatibility, receipts). Claim IDs stay; owners move:

- CLAIM-020 → `/model/#product-category`
- CLAIM-018 → `/status/#adjacent-guidance`
- CLAIM-011 → `/model/#gates-and-policy`
- CLAIM-006, CLAIM-021, CLAIM-012 → `/reference/#compatibility`
- CLAIM-007, CLAIM-008 → `/reference/#source-traceability`

Journey-link sources retargeted in `docs/_data/content-topology.yml`,
`docs/_data/website-capability-map.yml`,
`scripts/verify-public-product-model.sh`,
`scripts/tests/public-product-model/verify-content-topology.sh`, and
`scripts/websitejourney/artifacts.go`:

- JLINK-002: `/evaluate/#working-state` → `/model/#operating-model`
- JLINK-004: `/evaluate/#fit-decision` → `/adopt/#install`, label
  "Install Backstop"
- JLINK-005: `/model/#product-category` → `/status/#adjacent-guidance`
- JLINK-006: `/model/#gates-and-policy` → `/status/#supported-and-limited`
- JLINK-008: `/model/#harness-integration` → `/reference/#compatibility`
- JLINK-009: `/reference/#compatibility` → `/status/#adjacent-guidance`
- JLINK-021: `/model/#provenance-and-verification` →
  `/reference/#source-traceability`

Page tests: `discovery-evaluation-adoption-status.sh` no longer expects the
moved claims on `/evaluate/`; `model-use-cases-packs.sh` and
`extend-reference-contributing.sh` cover the new owners. CLAIM-007 uses
`visible()` for Liquid `{% raw %}`.

Closed-world: every non-claim, non-jlink, non-heading block in
`docs/evaluate.md` must be an exact string in `allowances` in
`scripts/verify-public-product-model.sh`. Heading text is skipped.
CLAIM-007 on `docs/reference.md` keeps `{% raw %}{{ ... }}{% endraw %}`.

Evaluate still owns NBR-002 and NBR-003 as the evaluation neighborhood. It no
longer hosts the field-guide appendix; those claims live on the pages that
already own the mechanism.

### Evaluate-only presentation

Evaluate matches the homepage paper/ink surface. Other inner pages stay the
dark field-guide until their own issues.

- `docs/_layouts/default.html`: when `page_kind == evaluation`, load
  `backstop-tokens.css`, `theme-color #0c0d0d`, no `color-scheme: dark`.
- `html:has([data-page-kind="evaluation"])` remaps `--ds-*` to
  `--color-paper/ink/muted/signal-ink/line/surface` and `--font-sans/mono`.
- Wordmark: homepage `./b` ink pill. Nav: ink 0.84rem, current-page underline
  `--color-signal`.
- Failed gate: dark terminal on paper (`--color-terminal`), like the home gate
  card.
- Result column and journey links: `--color-signal-ink`, not GitHub blue.
- Type recipe: loud line (H1 / H2 / `.tactics-kicker`, weight 650) then muted
  regular follow-on (`.page-boundary` / `.tactics-bridge` / `h2 + p`, 1.15rem,
  weight 400, `--ds-muted`). System UI has 400/700; 500 still looked bold.
- Accent closers (`.sdd-matrix + p`, `.ci-workflows + p`) stay green / weight
  650.
- `docs/_includes/page-hero.html` uses `page.hero_lede` when set.
- When editing the `site.css` media query, do not drop
  `[data-page-kind="home"] .nav` responsive rules.
- Playwright `testMatch` stays `public-site.spec.ts`. Packs stay external.

### Scope constraint

This issue owns `/evaluate/` only, plus the smallest corresponding inventory,
topology, allowance, page-test, and Seed 1 contract updates needed to land that
page without silent divergence from SPEC-072 / SPEC-075 / SPEC-076.

Out of scope — later page-by-page issues, do not absorb:

- Homepage (ISSUE-190; canonical homepage stays)
- Model, Adopt, Status, Reference, and other inner-page copy or paper theme
- Logo / hamburger / mobile nav for remaining inner pages
- Marking BUNDLE-032 delivered
- Vendoring packs
- Pushing the local prototype without a plan

## References

- Local prototype: branch `cursor/evaluate-copy-local-preview-e296`, HEAD
  `94464f4`. Primary source `docs/evaluate.md`; lockstep files in
  `git log origin/main..HEAD` on that branch.
- SPEC-072 Seed 1 topology, hero table, and JLINK matrix (old Evaluate
  appendix hops and hero "Is Backstop the right control surface for this
  problem?").
- BUNDLE-032 REQ-010: the bundle does not contain final copy; Seed 1 owned it.
  This issue revises Evaluate's Seed 1 copy after visitor review. It is not a
  competing charter for the website expansion.
- ISSUE-190: homepage fence. This issue must not redesign `/`.
- ISSUE-192: explicitly does not own inner-page visual redesign.
- `scripts/verify-public-product-model.sh` closed-world allowances.
- Live site still old: `https://backstop.sh/evaluate/`.

### Existence-in-world check

Before filing, `issues/` and `bundles/` were searched for Evaluate visitor
copy, inner-page visual redesign, paper/ink Evaluate restyle, tactics-matrix,
and "control surface" hero ownership.

- BUNDLE-032 / SPEC-072 own the visitor-journey IA, `/evaluate/` as NBR-002 +
  NBR-003, and the original Seed 1 final-copy heroes, blocks, claims, and
  JLINKs. This issue does not reopen that charter. It records the approved
  rewrite of that one page after visitor review.
- ISSUE-190 owns canonical homepage restoration only.
- ISSUE-192 owns architecture-pack classification of
  `scripts/websitejourney` and explicitly excludes inner-page visual redesign.
- No open issue owns this Evaluate rewrite. Remaining inner pages are
  intentionally unowned so they can be filed page by page.
