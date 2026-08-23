---
title: "Website Expansion"
number: BUNDLE-032
created: "2026-08-23"
schema_version: bundle/v2

bundle:
  name: website-expansion
  version: "0.2.0"
  created: "2026-08-23"
  updated: "2026-08-23"
  category: feature

status:
  maturity: defined

problem:
  summary: >
    Backstop's public site has useful launch content and a growing body of technical
    documentation, but it does not yet behave like a complete product surface. The current
    experience is organized more around what already exists in the repository than around
    the questions a person or agent has while discovering, evaluating, adopting, and using
    Backstop. Recent first-party dogfooding also changed the product's messaging materially:
    once an agent used Backstop for real, the reasons to use it, the limits of its guarantees,
    and the seams with adjacent tooling became much clearer than the current site reflects.
    The next website increment must make those truths legible without turning the site into
    marketing copy that outruns the implementation.
  user_story: >
    As a person or agent evaluating Backstop, I want the site to help me determine whether
    Backstop solves my actual problem, how it does so, what it does not guarantee, what it
    costs to adopt, and where I should go next, so that I can make a grounded product decision
    and then move from discovery to adoption and reference material without losing context.

solution:
  approach: >
    Rethink the public site around the visitor journey rather than the repository taxonomy.
    Treat the current landing page, existing Markdown documentation, implementation history,
    and first-party dogfooding findings as source material rather than constraints. Establish
    clear content neighborhoods for discovery, evaluation, conceptual understanding, adoption,
    and reference; pair those with a canonical comprehensive model of Backstop's concepts and
    architecture; and make consequential product claims inspectable down to mechanisms,
    guarantee boundaries, limitations, adoption implications, and future direction. Keep
    product truth in backstop-core, presentation in the design-system pack, and reusable
    documentation semantics in a separate pack. Define the site's user-facing outcomes as
    Backstop capabilities and user journeys so acceptance can be proven end to end. Use the
    least operationally complex hosting/rendering model that satisfies the functional
    requirements; for the current static-content scope, Jekyll and GitHub Pages remain the
    expected implementation unless later requirements earn additional runtime complexity.

requirements:
  - id: REQ-001
    version: "2.0.0"
    text: >
      The website increment must be treated as a full public-site rethink rather than a docs
      skinning exercise. Existing pages, layouts, navigation, and copy are source material to
      evaluate, not structures that must be preserved. If the broader rethink does not justify
      its churn, a narrower product-site expansion may be chosen as a fallback without lowering
      the user-facing outcomes required by this bundle.
    versions:
      - version: "1.0.0"
        text: >
          The current branded landing page must remain the visual baseline. Website expansion
          may refactor it onto shared site primitives, but must not materially redesign its
          hierarchy, wordmark, terminal proof, section order, CTA behavior, responsive posture,
          focus treatment, or reduced-motion behavior without an explicit follow-on decision.
      - version: "2.0.0"
        text: >
          The website increment must be treated as a full public-site rethink rather than a docs
          skinning exercise. Existing pages, layouts, navigation, and copy are source material to
          evaluate, not structures that must be preserved. If the broader rethink does not justify
          its churn, a narrower product-site expansion may be chosen as a fallback without lowering
          the user-facing outcomes required by this bundle.

  - id: REQ-002
    version: "2.0.0"
    text: >
      The public site must organize its primary experience around the visitor journey. It must
      provide authoritative neighborhoods for discovery, product evaluation, conceptual and
      architectural understanding, adoption and use-case guidance, and technical reference.
      Exact page names, route structure, and navigation labels are downstream design decisions.
    versions:
      - version: "1.0.0"
        text: >
          The public documentation surface must present Getting Started, Concepts, Artifact
          Workflow, Pack Authoring, and CLI Reference inside one coherent Backstop docs shell
          with persistent product navigation, documentation navigation, current-page state,
          and a route back to the home page and GitHub repository.
      - version: "2.0.0"
        text: >
          The public site must organize its primary experience around the visitor journey. It must
          provide authoritative neighborhoods for discovery, product evaluation, conceptual and
          architectural understanding, adoption and use-case guidance, and technical reference.
          Exact page names, route structure, and navigation labels are downstream design decisions.

  - id: REQ-003
    version: "2.0.0"
    text: >
      Backstop must expose one canonical, comprehensive conceptual and architectural model that
      a person or agent can consume directly. Use-case and adoption content should reference
      that model rather than creating competing explanations of the same concepts. The final
      canonical-content representation is a downstream design decision; the current Markdown
      files are valuable bootstrap material but are not automatically privileged as the permanent
      information architecture.
    versions:
      - version: "1.0.0"
        text: >
          The existing Markdown files remain canonical for technical documentation content.
          The website layer may add frontmatter and presentation metadata, but must not fork
          their substantive content into hand-maintained HTML copies.
      - version: "2.0.0"
        text: >
          Backstop must expose one canonical, comprehensive conceptual and architectural model that
          a person or agent can consume directly. Use-case and adoption content should reference
          that model rather than creating competing explanations of the same concepts. The final
          canonical-content representation is a downstream design decision; the current Markdown
          files are valuable bootstrap material but are not automatically privileged as the permanent
          information architecture.

  - id: REQ-004
    version: "2.0.0"
    text: >
      The site must support grounded product evaluation, not only usage. For consequential
      capability claims, a reader must be able to determine what Backstop claims, the mechanism
      that implements the claim, the boundary of the guarantee, known limitations, practical
      adoption implications, and relevant future direction. Compatibility with another tool or
      harness must not be presented as equivalent to that tool preserving Backstop's intended
      execution guarantees.
    versions:
      - version: "1.0.0"
        text: >
          Reusable website presentation belongs in backstop-ai/backstop-design-system rather
          than being independently recreated in backstop-core. The design-system pack must
          expose the docs-shell recipe or equivalent deterministic generation surface needed
          by the site, and backstop-core must consume a released, locked pack version.
      - version: "2.0.0"
        text: >
          The site must support grounded product evaluation, not only usage. For consequential
          capability claims, a reader must be able to determine what Backstop claims, the mechanism
          that implements the claim, the boundary of the guarantee, known limitations, practical
          adoption implications, and relevant future direction. Compatibility with another tool or
          harness must not be presented as equivalent to that tool preserving Backstop's intended
          execution guarantees.

  - id: REQ-005
    version: "2.0.0"
    text: >
      Product-boundary information must be represented distinctly enough that a person or agent
      can tell the difference between a capability that is supported today, a known limitation,
      planned work, an intentional non-goal, and adjacent guidance for a problem Backstop does not
      plan to own directly. When Backstop stops at a boundary, the site should explain the seam and
      provide enough reasoning or recommendations for the user to continue solving the larger
      system problem without treating that guidance as a Backstop guarantee.
    versions:
      - version: "1.0.0"
        text: >
          The expanded site must satisfy the design-system rules already dogfooded on the
          landing page: token-only color usage, no inline styles, visible keyboard focus,
          reduced-motion handling, and canonical Backstop wordmark usage.
      - version: "2.0.0"
        text: >
          Product-boundary information must be represented distinctly enough that a person or agent
          can tell the difference between a capability that is supported today, a known limitation,
          planned work, an intentional non-goal, and adjacent guidance for a problem Backstop does not
          plan to own directly. When Backstop stops at a boundary, the site should explain the seam and
          provide enough reasoning or recommendations for the user to continue solving the larger
          system problem without treating that guidance as a Backstop guarantee.

  - id: REQ-006
    version: "2.0.0"
    text: >
      Documentation semantics, product truth, and presentation must remain separate concerns.
      backstop-core owns Backstop-specific truth, information architecture, and content instances;
      backstop-ai/backstop-design-system owns reusable visual and interaction policy; and a separate
      pack must own the reusable documentation-semantic contract and whatever deterministic
      enforcement is required to preserve it. Core must not absorb documentation-domain knowledge
      merely because an enforcement engine for that domain does not already exist.
    versions:
      - version: "1.0.0"
        text: >
          Documentation navigation must be usable at desktop and narrow mobile widths without
          hiding the current page, trapping keyboard focus, depending on JavaScript for basic
          navigation, or producing a second competing information architecture.
      - version: "2.0.0"
        text: >
          Documentation semantics, product truth, and presentation must remain separate concerns.
          backstop-core owns Backstop-specific truth, information architecture, and content instances;
          backstop-ai/backstop-design-system owns reusable visual and interaction policy; and a separate
          pack must own the reusable documentation-semantic contract and whatever deterministic
          enforcement is required to preserve it. Core must not absorb documentation-domain knowledge
          merely because an enforcement engine for that domain does not already exist.

  - id: REQ-007
    version: "2.0.0"
    text: >
      The website's user-facing outcomes must be defined as Backstop capabilities with concrete
      user journeys and executable acceptance evidence. Bundle requirements identify the outcomes
      and neighborhoods that matter; downstream artifacts define the exact capability set,
      @UJ-NNN scenarios, and stack-appropriate acceptance mechanisms needed to prove them end to end.
    versions:
      - version: "1.0.0"
        text: >
          The generated site must remain deployable by the existing GitHub Pages/Jekyll path,
          preserve backstop.sh custom-domain behavior, and introduce no application runtime or
          client-side framework solely to render documentation.
      - version: "2.0.0"
        text: >
          The website's user-facing outcomes must be defined as Backstop capabilities with concrete
          user journeys and executable acceptance evidence. Bundle requirements identify the outcomes
          and neighborhoods that matter; downstream artifacts define the exact capability set,
          @UJ-NNN scenarios, and stack-appropriate acceptance mechanisms needed to prove them end to end.

  - id: REQ-008
    version: "2.0.0"
    text: >
      The current site, existing technical documentation, repository implementation and commit
      history, prior design-system and website dogfooding findings, and known real-world agent
      failure modes must be treated as evidence and bootstrap material during redesign. The site
      should prefer claims that can be traced to actual behavior or implementation over aspirational
      positioning, while preserving room for explicitly labeled future direction.
    versions:
      - version: "1.0.0"
        text: >
          Site verification must include deterministic checks for generated-page completeness,
          internal-link integrity, canonical navigation targets, and design-system rule
          compliance. Generated output must fail loudly when a canonical docs page is omitted
          or linked to a nonexistent route.
      - version: "2.0.0"
        text: >
          The current site, existing technical documentation, repository implementation and commit
          history, prior design-system and website dogfooding findings, and known real-world agent
          failure modes must be treated as evidence and bootstrap material during redesign. The site
          should prefer claims that can be traced to actual behavior or implementation over aspirational
          positioning, while preserving room for explicitly labeled future direction.

  - id: REQ-009
    version: "2.0.0"
    text: >
      The implementation and deployment of backstop.sh must remain as operationally boring as the
      functional requirements allow. For the current static-content requirements, Jekyll and GitHub
      Pages are the expected implementation. Jekyll is not a permanent product constraint: if future
      requirements introduce meaningful interactivity, persisted state, transactions, authentication,
      or comparable runtime concerns, a more capable application platform may replace it when those
      requirements earn the additional complexity.
    versions:
      - version: "1.0.0"
        text: >
          Copy changes in this increment must remove stale or contradictory positioning where
          encountered and keep each page oriented around a concrete reader job. Building a
          generalized prose linter, prose LSP, or writing-style pack is explicitly outside this
          bundle and may be scoped separately.
      - version: "2.0.0"
        text: >
          The implementation and deployment of backstop.sh must remain as operationally boring as the
          functional requirements allow. For the current static-content requirements, Jekyll and GitHub
          Pages are the expected implementation. Jekyll is not a permanent product constraint: if future
          requirements introduce meaningful interactivity, persisted state, transactions, authentication,
          or comparable runtime concerns, a more capable application platform may replace it when those
          requirements earn the additional complexity.

  - id: REQ-010
    version: "1.0.0"
    text: >
      This bundle owns the site's content topology, communicative responsibilities, source material,
      and product-boundary requirements; it does not own final page copy. Actual copywriting and
      page-level wording are deferred to downstream specs and plans. A generalized prose-quality,
      writing-style, or prose-LSP pack remains separate work and is not a prerequisite for this bundle.
    versions:
      - version: "1.0.0"
        text: >
          This bundle owns the site's content topology, communicative responsibilities, source material,
          and product-boundary requirements; it does not own final page copy. Actual copywriting and
          page-level wording are deferred to downstream specs and plans. A generalized prose-quality,
          writing-style, or prose-LSP pack remains separate work and is not a prerequisite for this bundle.
---

# BUNDLE-032: Website Expansion

## Current Thinking

This is no longer a "make the docs look like the homepage" increment. The website should be
reconsidered as the public product surface for Backstop, with the current landing page and five
existing Markdown docs treated as useful evidence rather than the final shape of the information
architecture.

The primary navigation model should follow the questions a visitor is trying to answer. A newcomer
needs to understand the problem and decide whether Backstop is relevant. An evaluator needs to test
Backstop's claims against its actual mechanisms, limits, and adoption cost. A user needs concrete
paths into adoption and use cases. Someone already committed to the product needs dense conceptual
and reference material. An agent mediating any of those decisions needs the same truths in a canonical,
structured, internally consistent form.

That does not require a separate "agent journey." Optimizing the human journey also gives an agent
better material to reason over. The additional requirement is that canonical reference content be
complete and machine-legible enough that an agent can ingest the model without having to reconstruct
it from duplicated or contradictory pages.

The site should be unusually explicit about epistemic boundaries. A reader should be able to tell
what Backstop supports today, what remains a known limitation, what is planned, what is an intentional
non-goal, and where Backstop has adjacent guidance even though it does not intend to own the problem.
The answer to "Can I use Backstop with X?" must be allowed to differ from the answer to "Does X preserve
Backstop's intended guarantees?" Compatibility and guarantees are separate claims.

This also creates a new pack-design test. The reusable semantic contract for product documentation does
not belong in Core and does not belong in the visual design system. It should be expressible as a separate
pack. How that pack is authored and which deterministic engines it invents or composes is intentionally a
downstream design problem. The absence of an off-the-shelf deterministic tool is not a reason to move the
concern into probabilistic review or bake it into Core.

Finally, website correctness should be expressed through Backstop's existing capability model rather than
through a hand-written list of ad hoc site checks at bundle level. The bundle defines the outcomes the site
must enable; downstream artifacts define the capabilities, @UJ-NNN scenarios, and executable acceptance
proof that establish those outcomes.

## Draft Requirements

- **REQ-001 — Rethink the whole public site.** Existing pages are source material, not constraints;
  a narrower product-site expansion is a fallback if the broader rethink does not earn its churn.
- **REQ-002 — Navigate by visitor journey.** Discovery, evaluation, understanding, adoption/use cases,
  and reference each need an authoritative home; exact route and page choices come later.
- **REQ-003 — Maintain one canonical conceptual model.** Dense architecture/concepts content should be
  directly consumable and referenced by use-case content rather than duplicated across it.
- **REQ-004 — Make Backstop auditable as a product choice.** Consequential claims must expose mechanism,
  guarantee boundary, limitations, adoption implications, and direction; compatibility is not a guarantee.
- **REQ-005 — Make product boundaries explicit.** Supported, limitation, planned, non-goal, and adjacent
  guidance are distinct kinds of truth, and where Backstop stops the site should explain the seam.
- **REQ-006 — Keep ownership composable.** Core owns product truth; the design-system pack owns visual and
  interaction policy; a separate pack owns documentation semantics and their deterministic enforcement.
- **REQ-007 — Prove the user journeys.** Website outcomes are delivered as capabilities with executable
  acceptance evidence, not as visual-review assertions.
- **REQ-008 — Bootstrap from evidence.** Current docs, implementation, commit history, dogfooding findings,
  and observed agent failures are source material; claims should not outrun what the system actually does.
- **REQ-009 — Complexity must be earned.** Use the least complex deployment/runtime that satisfies the
  functional requirements. Today that means Jekyll/GitHub Pages; it is not a permanent architectural vow.
- **REQ-010 — Bundle owns topology, not prose.** Page-level copywriting and wording are downstream work;
  a generalized prose-quality/style/LSP pack is separate scope.

## Resolved Design Principles

- **DP-1 — Human journey first, agent journey included.** Agent-mediated product discovery and evaluation
  should succeed because the same canonical content is useful to humans and agents, not because the site
  maintains a second shadow documentation system.
- **DP-2 — Decision support, not brochureware.** The site should help a reader decide whether Backstop is
  appropriate, not merely teach installation after the decision has already been made.
- **DP-3 — Claims are bounded.** The site should make it possible to descend from a consequential claim to
  its mechanism, evidence, guarantee boundary, limitations, and practical adoption implications.
- **DP-4 — Where Backstop stops, document the seam.** Intentional product boundaries should include enough
  reasoning and adjacent guidance that the user can continue solving the larger problem.
- **DP-5 — No is the architectural default.** Product and platform complexity must be earned by concrete
  requirements. Existing simple infrastructure remains until the functional requirements make it inadequate.

## Source Material To Bootstrap From

- `docs/index.html` — current branded landing page and visual/product-story source material.
- `docs/getting-started.md` — current adoption bootstrap.
- `docs/concepts.md` — current conceptual-model bootstrap.
- `docs/artifact-workflow.md` — current artifact and SDLC workflow explanation.
- `docs/pack-authoring.md` — current extensibility/pack-authoring material.
- `docs/cli-reference.md` — current command/reference material.
- `backstop-ai/backstop-design-system` — executable visual policy and recipes.
- Backstop Core implementation and commit history — evidence for what the product actually does and why
  particular compensating mechanisms exist.
- Prior website/design-system dogfooding — especially the messaging changes that emerged when an agent used
  Backstop for real and the resulting design-system/recipe defects such as ISSUE-182.
- Real user/agent failure modes — source material for problem-oriented evaluation and use-case neighborhoods.

## Out of Scope At Bundle Level

- Final page copy, headlines, CTA wording, or prose polish.
- A generalized prose-quality, writing-style, or prose-LSP pack.
- Building an MCP server in this increment; future documentation/guidance MCP consumption should reuse the
  canonical documentation model rather than create a second source of truth.
- Stateful SaaS surfaces such as hosted artifact visualization, quality dashboards, team coordination,
  authenticated workflows, or a graphical pack-authoring product.
- Treating a coding harness as Backstop's deterministic execution guarantee; harness compatibility and
  Backstop guarantees remain separate concerns.
- A permanent commitment to Jekyll, GitHub Pages, or any other implementation stack beyond the requirement
  to remain as operationally simple as current functional needs allow.

## Remaining Open Questions

- What exact page/neighborhood inventory and navigation model best satisfies the visitor journey?
- What exact Backstop capabilities and @UJ-NNN scenarios define success for the public site?
- How should the new documentation-semantics pack represent and deterministically enforce its contract?
- Which existing content should remain canonical verbatim, which should be decomposed or merged, and which
  should be retired after its useful material is incorporated elsewhere?
- What machine-readable publication shape should accompany the human site, if any, before a future docs-only
  MCP surface exists?
- How far should the visual redesign depart from the current landing page once the new information
  architecture is known?

## Spec Seeds

- **Public-site information architecture and content contracts.** Define the page/neighborhood model,
  canonical-source strategy, and communicative responsibilities without embedding final prose in the bundle.
- **Documentation-semantics pack.** Define the reusable semantic model, recipes/data shape, deterministic
  enforcement strategy, and composition seam with Core and the design-system pack.
- **Website capabilities and user journeys.** Define CAP artifacts, @UJ-NNN scenarios, and executable
  acceptance proof for the discovery/evaluation/adoption/reference experience.
- **Design-system follow-on.** Add any reusable presentation primitives required by the approved site model
  under `backstop-ai/backstop-design-system`'s own artifact governance.
- **Copy/content execution.** Write or rewrite page content only after the information architecture and
  semantic contracts are settled.

## Version History

- 0.2.0 (2026-08-23): Reframed the increment from docs-shell expansion to a full public-site rethink;
  adopted visitor-journey IA, canonical concept/architecture content, auditable product claims, explicit
  support/limitation/planned/non-goal/adjacent-guidance boundaries, separate documentation-semantic pack
  ownership, capability/user-journey acceptance, evidence-first source material, and "boring as requirements
  allow" deployment. Deferred final copy and detailed page design to downstream artifacts.
- 0.1.0 (2026-08-23): Defined the original post-landing-page website increment around preserving Home,
  unifying the five canonical Markdown docs under a Backstop docs shell, deterministic route/link checks,
  static GitHub Pages/Jekyll deployment, and local copy cleanup.

## References

- `docs/index.html`
- `docs/getting-started.md`
- `docs/concepts.md`
- `docs/artifact-workflow.md`
- `docs/pack-authoring.md`
- `docs/cli-reference.md`
- `backstop-ai/backstop-design-system`
- `artifacts/capability/v1/schema.json`
- ISSUE-182 — Recipe Literal Placeholder Escaping.
