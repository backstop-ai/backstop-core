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
    version: "1.0.0"
    text: >
      The current branded landing page must remain the visual baseline. Website expansion
      may refactor it onto shared site primitives, but must not materially redesign its
      hierarchy, wordmark, terminal proof, section order, CTA behavior, responsive posture,
      focus treatment, or reduced-motion behavior without an explicit follow-on decision.
    versions:
      - version: "1.0.0"
        text: >
          The current branded landing page must remain the visual baseline. Website expansion
          may refactor it onto shared site primitives, but must not materially redesign its
          hierarchy, wordmark, terminal proof, section order, CTA behavior, responsive posture,
          focus treatment, or reduced-motion behavior without an explicit follow-on decision.
  - id: REQ-002
    version: "1.0.0"
    text: >
      The public documentation surface must present Getting Started, Concepts, Artifact
      Workflow, Pack Authoring, and CLI Reference inside one coherent Backstop docs shell
      with persistent product navigation, documentation navigation, current-page state,
      and a route back to the home page and GitHub repository.
    versions:
      - version: "1.0.0"
        text: >
          The public documentation surface must present Getting Started, Concepts, Artifact
          Workflow, Pack Authoring, and CLI Reference inside one coherent Backstop docs shell
          with persistent product navigation, documentation navigation, current-page state,
          and a route back to the home page and GitHub repository.
  - id: REQ-003
    version: "1.0.0"
    text: >
      The existing Markdown files remain canonical for technical documentation content.
      The website layer may add frontmatter and presentation metadata, but must not fork
      their substantive content into hand-maintained HTML copies.
    versions:
      - version: "1.0.0"
        text: >
          The existing Markdown files remain canonical for technical documentation content.
          The website layer may add frontmatter and presentation metadata, but must not fork
          their substantive content into hand-maintained HTML copies.
  - id: REQ-004
    version: "1.0.0"
    text: >
      Reusable website presentation belongs in backstop-ai/backstop-design-system rather
      than being independently recreated in backstop-core. The design-system pack must
      expose the docs-shell recipe or equivalent deterministic generation surface needed
      by the site, and backstop-core must consume a released, locked pack version.
    versions:
      - version: "1.0.0"
        text: >
          Reusable website presentation belongs in backstop-ai/backstop-design-system rather
          than being independently recreated in backstop-core. The design-system pack must
          expose the docs-shell recipe or equivalent deterministic generation surface needed
          by the site, and backstop-core must consume a released, locked pack version.
  - id: REQ-005
    version: "1.0.0"
    text: >
      The expanded site must satisfy the design-system rules already dogfooded on the
      landing page: token-only color usage, no inline styles, visible keyboard focus,
      reduced-motion handling, and canonical Backstop wordmark usage.
    versions:
      - version: "1.0.0"
        text: >
          The expanded site must satisfy the design-system rules already dogfooded on the
          landing page: token-only color usage, no inline styles, visible keyboard focus,
          reduced-motion handling, and canonical Backstop wordmark usage.
  - id: REQ-006
    version: "1.0.0"
    text: >
      Documentation navigation must be usable at desktop and narrow mobile widths without
      hiding the current page, trapping keyboard focus, depending on JavaScript for basic
      navigation, or producing a second competing information architecture.
    versions:
      - version: "1.0.0"
        text: >
          Documentation navigation must be usable at desktop and narrow mobile widths without
          hiding the current page, trapping keyboard focus, depending on JavaScript for basic
          navigation, or producing a second competing information architecture.
  - id: REQ-007
    version: "1.0.0"
    text: >
      The generated site must remain deployable by the existing GitHub Pages/Jekyll path,
      preserve backstop.sh custom-domain behavior, and introduce no application runtime or
      client-side framework solely to render documentation.
    versions:
      - version: "1.0.0"
        text: >
          The generated site must remain deployable by the existing GitHub Pages/Jekyll path,
          preserve backstop.sh custom-domain behavior, and introduce no application runtime or
          client-side framework solely to render documentation.
  - id: REQ-008
    version: "1.0.0"
    text: >
      Site verification must include deterministic checks for generated-page completeness,
      internal-link integrity, canonical navigation targets, and design-system rule
      compliance. Generated output must fail loudly when a canonical docs page is omitted
      or linked to a nonexistent route.
    versions:
      - version: "1.0.0"
        text: >
          Site verification must include deterministic checks for generated-page completeness,
          internal-link integrity, canonical navigation targets, and design-system rule
          compliance. Generated output must fail loudly when a canonical docs page is omitted
          or linked to a nonexistent route.
  - id: REQ-009
    version: "1.0.0"
    text: >
      Copy changes in this increment must remove stale or contradictory positioning where
      encountered and keep each page oriented around a concrete reader job. Building a
      generalized prose linter, prose LSP, or writing-style pack is explicitly outside this
      bundle and may be scoped separately.
    versions:
      - version: "1.0.0"
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

The site should expand by **wrapping canonical content, not replacing it**. The five
existing Markdown docs already carry the technical substance. The useful work is to make
them navigable, branded, responsive, and structurally coherent as a set. This also keeps
documentation diffs legible: explanation remains Markdown; presentation remains design
system.

This is another design-system dogfood pass. A docs shell that exists only as copied HTML or
CSS in `backstop-core` would prove the opposite of the prior sprint. Shared site primitives
must come from the design-system pack and be consumed and locked here. The implementation
must also account for ISSUE-182's literal-template collision rather than reintroducing
fragile Liquid-in-Liquid generation.

## Draft Requirements

- **REQ-001 — Preserve Home.** The branded landing page is the baseline, not a redesign target.
- **REQ-002 — One docs IA.** Getting Started, Concepts, Artifact Workflow, Pack Authoring,
  and CLI Reference are one ordered documentation surface.
- **REQ-003 — Markdown stays canonical.** Presentation wraps technical content; it does not
  create parallel HTML copies.
- **REQ-004 — Reuse through the pack.** Shared docs presentation belongs in the design-system
  pack and Core consumes a released, locked version.
- **REQ-005 — Existing rules still apply.** Expansion keeps the pack's tokens, focus,
  reduced-motion, no-inline-style, and wordmark contracts.
- **REQ-006 — Navigation is responsive and no-JS usable.** The reader keeps orientation at
  desktop and narrow widths without a client runtime.
- **REQ-007 — Stay operationally boring.** Static Jekyll/GitHub Pages and `backstop.sh` remain.
- **REQ-008 — Verify the generated graph.** Missing pages, bad internal links, wrong nav,
  CNAME drift, and design-system violations fail deterministically.
- **REQ-009 — Improve copy locally.** Stale positioning can be corrected when touched; a
  prose LSP or generalized writing-style pack is separate work.

## Draft Design Decisions

- **DD-1 — Reader progression, not repo taxonomy.** Primary docs order is Getting Started →
  Concepts → Artifact Workflow → Pack Authoring → CLI Reference.
- **DD-2 — Markdown body + shared layout.** Canonical technical content remains Markdown;
  presentation wraps it rather than forking it.
- **DD-3 — Design-system owns presentation primitives.** Core owns content and ordering;
  the design-system pack owns reusable tokens, shell, navigation, typography, responsive
  behavior, and accessibility constraints.
- **DD-4 — No client-side framework.** Basic rendering and navigation work with static
  HTML/CSS/Jekyll alone; JavaScript, if any, is progressive enhancement.
- **DD-5 — Generated routes are a graph.** Verification enumerates expected pages and links
  against the built site rather than relying on visual inspection.
- **DD-6 — Copy is reviewed in context.** Stale positioning can be corrected when touched,
  but a generalized writing-style pack or prose LSP is a separate product problem.

## Spec Seeds

- **SPEC-071 — Website Expansion.** Core docs-shell consumption, information architecture,
  deterministic built-site verification, and dependency on a released design-system
  docs-shell capability.
- **Design-system follow-on work.** Reusable docs-shell implementation belongs in
  `backstop-ai/backstop-design-system` under that repository's own artifact governance.
- **Future copy-system seed.** A Backstop writing-style/prose-enforcement pack remains a
  plausible follow-on, intentionally outside this bundle.

## Version History

- 0.1.0 (2026-08-23): Defined the post-landing-page website increment: preserve Home,
  unify the five canonical Markdown docs under a Backstop docs shell, require reusable
  design-system ownership and deterministic route/link verification, retain static GitHub
  Pages/Jekyll deployment, and keep generalized prose enforcement out of scope.

## References

- `docs/index.html` — branded landing-page reference implementation.
- `docs/getting-started.md`, `docs/concepts.md`, `docs/artifact-workflow.md`,
  `docs/pack-authoring.md`, `docs/cli-reference.md` — canonical technical content set.
- `docs/_config.yml` — current Cayman theme boundary.
- `backstop-ai/backstop-design-system` — reusable visual rules and recipes.
- ISSUE-182 — Recipe Literal Placeholder Escaping.
