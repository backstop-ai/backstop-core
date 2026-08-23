---
title: "Website Expansion"
number: BUNDLE-032
created: "2026-08-23"
schema_version: bundle/v2

bundle:
  name: website-expansion
  version: "0.1.0"
  created: "2026-08-23"
  updated: "2026-08-23"
  category: feature

status:
  maturity: defined

problem:
  summary: >
    Backstop now has a deliberate branded landing page, but the rest of the public site
    still falls through to stock Cayman-rendered Markdown. The result is a split product:
    the home page communicates a distinct system and visual language while the technical
    journey immediately leaves that system at the first Docs click. The next website
    increment must turn the existing launch surface into one coherent public site without
    rewriting canonical technical content, duplicating design decisions in backstop-core,
    or treating the landing page as another redesign exercise.
  user_story: >
    As a developer evaluating or adopting Backstop, I want the landing page and technical
    documentation to behave and read like one product, so that I can move from the thesis
    to installation, concepts, workflow, authoring, and reference material without losing
    orientation or wondering which surface is canonical.

solution:
  approach: >
    Preserve the current landing page as the visual and interaction baseline; make the
    existing Markdown documentation the canonical content bodies; add a reusable docs-shell
    capability to the Backstop design-system pack; and consume that capability from
    backstop-core so every public documentation page receives the same Backstop navigation,
    typography, tokens, accessibility behavior, and footer. Keep the site static and
    GitHub-Pages-compatible. Treat copy refinement as page content work constrained by the
    site information architecture, not as a new prose-enforcement subsystem in this scope.

requirements:
  - id: REQ-001
    text: >
      The current branded landing page must remain the visual baseline. Website expansion
      may refactor it onto shared site primitives, but must not materially redesign its
      hierarchy, wordmark, terminal proof, section order, CTA behavior, responsive posture,
      focus treatment, or reduced-motion behavior without an explicit follow-on decision.
  - id: REQ-002
    text: >
      The public documentation surface must present Getting Started, Concepts, Artifact
      Workflow, Pack Authoring, and CLI Reference inside one coherent Backstop docs shell
      with persistent product navigation, documentation navigation, current-page state,
      and a route back to the home page and GitHub repository.
  - id: REQ-003
    text: >
      The existing Markdown files remain canonical for technical documentation content.
      The website layer may add frontmatter and presentation metadata, but must not fork
      their substantive content into hand-maintained HTML copies.
  - id: REQ-004
    text: >
      Reusable website presentation belongs in backstop-ai/backstop-design-system rather
      than being independently recreated in backstop-core. The design-system pack must
      expose the docs-shell recipe or equivalent deterministic generation surface needed
      by the site, and backstop-core must consume a released, locked pack version.
  - id: REQ-005
    text: >
      The expanded site must satisfy the design-system rules already dogfooded on the
      landing page: token-only color usage, no inline styles, visible keyboard focus,
      reduced-motion handling, and canonical Backstop wordmark usage.
  - id: REQ-006
    text: >
      Documentation navigation must be usable at desktop and narrow mobile widths without
      hiding the current page, trapping keyboard focus, depending on JavaScript for basic
      navigation, or producing a second competing information architecture.
  - id: REQ-007
    text: >
      The generated site must remain deployable by the existing GitHub Pages/Jekyll path,
      preserve backstop.sh custom-domain behavior, and introduce no application runtime or
      client-side framework solely to render documentation.
  - id: REQ-008
    text: >
      Site verification must include deterministic checks for generated-page completeness,
      internal-link integrity, canonical navigation targets, and design-system rule
      compliance. Generated output must fail loudly when a canonical docs page is omitted
      or linked to a nonexistent route.
  - id: REQ-009
    text: >
      Copy changes in this increment must remove stale or contradictory positioning where
      encountered and keep each page oriented around a concrete reader job. Building a
      generalized prose linter, prose LSP, or writing-style pack is explicitly outside this
      bundle and may be scoped separately.
---

# Website Expansion

## Current Thinking

The landing page is no longer the website problem. It is the reference implementation.
The gap is everything after the first Docs click: `docs/_config.yml` still names Cayman,
while `docs/index.html` is a fully custom Backstop surface. That creates an avoidable seam
at exactly the moment an interested reader moves from positioning to evaluation.

The site should therefore expand by **wrapping canonical content, not replacing it**. The
five existing Markdown docs already carry the technical substance. The useful work is to
make them navigable, branded, responsive, and structurally coherent as a set. This also
keeps documentation diffs legible: changes to explanation remain Markdown changes, while
presentation changes remain design-system changes.

This is also another design-system dogfood pass. A docs shell that only exists as copied
HTML/CSS in `backstop-core` would prove the opposite of what the prior sprint set out to
prove. Shared site primitives must come from the design-system pack, then be consumed and
locked here. The implementation must account for the literal-template collision already
found during the landing-page dogfood loop rather than quietly reintroducing fragile
Liquid-in-Liquid generation.

## Draft Requirements

- **REQ-001 — Preserve the reference surface.** Home is the baseline, not a new canvas.
- **REQ-002 — One docs information architecture.** Five current canonical docs pages sit
  in one product/docs navigation model.
- **REQ-003 — Markdown stays canonical.** Presentation may wrap it; no HTML content fork.
- **REQ-004 — Reuse through the pack.** Shared docs presentation is a design-system
  capability and Core consumes a released/locked version.
- **REQ-005 — Existing rules still apply.** The site does not get a waiver from the pack
  because it grew beyond one page.
- **REQ-006 — Responsive and keyboard-usable.** Navigation remains understandable without
  JavaScript at both desktop and narrow widths.
- **REQ-007 — Stay boring operationally.** Static Jekyll/GitHub Pages remains the deploy
  model and `backstop.sh` remains the public domain.
- **REQ-008 — Verify the generated graph.** Missing pages, broken links, or design-rule
  violations are deterministic failures.
- **REQ-009 — Improve copy locally, not infrastructurally.** Fix stale positioning in the
  pages touched here; prose enforcement is a different product problem.

## Draft Design Decisions

- **DD-1 — Information architecture follows reader progression, not repo taxonomy.** The
  primary docs path is Getting Started → Concepts → Artifact Workflow → Pack Authoring →
  CLI Reference. Direct URLs remain valid; the order is navigation guidance, not a wizard.
- **DD-2 — Markdown body + shared layout.** Each docs page remains Markdown rendered into a
  shared Backstop docs layout. The landing page may share header/footer primitives after
  the shell exists, but its content markup is not rewritten merely for uniformity.
- **DD-3 — Design-system owns presentation primitives.** Tokens, header/footer, docs nav,
  content typography, code treatment, responsive behavior, and accessibility rules are
  produced or constrained by the design-system pack. Core owns page content and ordering.
- **DD-4 — No client-side framework.** Basic site navigation and page rendering work with
  HTML/CSS/Jekyll alone. Any JavaScript later added must be progressive enhancement.
- **DD-5 — Treat generated routes as a graph.** Verification enumerates expected canonical
  pages and links, then checks the built site rather than relying on visual inspection.
- **DD-6 — Copy is reviewed in context.** The stale `An AI agent discipline framework`
  description and other positioning mismatches are in scope to reconcile where this work
  touches them, but the website sprint does not invent the generalized prose pack/LSP.

## Spec Seeds

- **SPEC-071 — Website Expansion.** Implement the docs-shell consumption in Core, the
  deterministic site verification contract, information architecture, and the dependency
  on a released design-system docs-shell capability.
- **Design-system follow-on work.** The reusable docs-shell recipe belongs in
  `backstop-ai/backstop-design-system`. If it requires independent artifact governance in
  that repository, create that chain there rather than letting this Core plan mutate two
  ledgers.
- **Future copy-system seed.** A Backstop writing-style/prose-enforcement pack is plausible
  follow-on work but intentionally has no requirement or implementation task in this
  bundle.

## Version History

- 0.1.0 (2026-08-23): Defined the post-landing-page website increment: preserve the
  existing home surface, wrap the five canonical Markdown docs in one Backstop-owned docs
  shell, require design-system-pack reuse and deterministic route/link verification, keep
  GitHub Pages/Jekyll as the runtime, and explicitly keep generalized prose enforcement
  out of scope.

## References

- `docs/index.html` — current branded landing-page reference implementation.
- `docs/getting-started.md`, `docs/concepts.md`, `docs/artifact-workflow.md`,
  `docs/pack-authoring.md`, `docs/cli-reference.md` — canonical technical content set.
- `docs/_config.yml` — current Cayman theme boundary that this bundle removes from the
  public documentation experience.
- `backstop-ai/backstop-design-system` — reusable visual rules and deterministic Jekyll
  recipes produced by the prior dogfooding sprint.
- ISSUE-182 — literal placeholder escaping discovered by the prior design-system dogfood
  loop; the docs-shell implementation must not regress into fragile template nesting.
