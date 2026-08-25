---
title: "Website Expansion"
number: BUNDLE-032
created: "2026-08-23"
schema_version: bundle/v2

bundle:
  name: website-expansion
  version: "0.5.1"
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
    ecosystem exploration, and reference; pair those with a canonical comprehensive model of
    Backstop's concepts and architecture; and make consequential product claims inspectable
    down to mechanisms, guarantee boundaries, limitations, adoption implications, and future
    direction. Keep product truth in backstop-core, presentation in the design-system pack,
    and reusable documentation semantics in a separate pack. Define the site's user-facing
    outcomes as Backstop capabilities and user journeys so acceptance can be proven end to end.
    Derived product documentation must flow from authoritative project truth into generated
    Markdown and then into the rendered site, with CI detecting drift. Use the least
    operationally complex hosting/rendering model that satisfies the functional requirements;
    for the current static-content scope, Jekyll and GitHub Pages remain the expected
    implementation unless later requirements earn additional runtime complexity.

requirements:
  - id: REQ-001
    version: "2.1.0"
    text: >
      The website increment must be delivered as a full public-site rethink rather than a
      documentation-skinning exercise. Existing pages, layouts, navigation, and copy are source
      material to evaluate, not structures that must be preserved. The accepted result must satisfy
      the full visitor-journey, product-evaluation, canonical-model, evidence, ecosystem, adoption,
      and reference outcomes in this bundle; a narrower product-site expansion is not an alternate
      completion path.
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
      - version: "2.1.0"
        text: >
          The website increment must be delivered as a full public-site rethink rather than a
          documentation-skinning exercise. Existing pages, layouts, navigation, and copy are source
          material to evaluate, not structures that must be preserved. The accepted result must satisfy
          the full visitor-journey, product-evaluation, canonical-model, evidence, ecosystem, adoption,
          and reference outcomes in this bundle; a narrower product-site expansion is not an alternate
          completion path.

  - id: REQ-002
    version: "2.0.0"
    text: >
      The public site must organize its primary experience around the visitor journey. It must
      provide authoritative neighborhoods for discovery, product evaluation, "what Backstop is
      and is not," conceptual and architectural understanding, adoption and use-case guidance,
      pack-ecosystem discovery, extension guidance, and technical reference. Exact page names,
      route structure, and navigation labels are downstream design decisions.
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
          provide authoritative neighborhoods for discovery, product evaluation, "what Backstop is
          and is not," conceptual and architectural understanding, adoption and use-case guidance,
          pack-ecosystem discovery, extension guidance, and technical reference. Exact page names,
          route structure, and navigation labels are downstream design decisions.

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
    version: "2.1.0"
    text: >
      Documentation semantics, product truth, and presentation must remain separate concerns.
      backstop-core owns Backstop-specific truth, information architecture, and content instances;
      backstop-ai/backstop-design-system owns reusable visual and interaction policy; and a released,
      pinned, separately governed documentation-semantics pack must own the reusable documentation-
      semantic contract and its deterministic enforcement. That pack is a known dependency of
      BUNDLE-032, not contingent discovery. BUNDLE-032 may own consumer-facing integration and
      acceptance, but Core and the visual design system must not absorb documentation-domain knowledge
      or the pack's implementation.
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
      - version: "2.1.0"
        text: >
          Documentation semantics, product truth, and presentation must remain separate concerns.
          backstop-core owns Backstop-specific truth, information architecture, and content instances;
          backstop-ai/backstop-design-system owns reusable visual and interaction policy; and a released,
          pinned, separately governed documentation-semantics pack must own the reusable documentation-
          semantic contract and its deterministic enforcement. That pack is a known dependency of
          BUNDLE-032, not contingent discovery. BUNDLE-032 may own consumer-facing integration and
          acceptance, but Core and the visual design system must not absorb documentation-domain knowledge
          or the pack's implementation.

  - id: REQ-007
    version: "2.0.0"
    text: >
      The website's user-facing outcomes must be defined as Backstop capabilities with concrete
      user journeys and executable acceptance evidence. At minimum, the site must enable a visitor
      to understand what Backstop is, evaluate fit, evaluate guarantees, evaluate compatibility,
      understand the system, adopt Backstop, apply it to concrete use cases, browse and extend the
      pack ecosystem, inspect supporting evidence, and continue beyond Backstop's intentional
      boundaries with useful guidance.
    versions:
      - version: "1.0.0"
        text: >
          The generated site must remain deployable by the existing GitHub Pages/Jekyll path,
          preserve backstop.sh custom-domain behavior, and introduce no application runtime or
          client-side framework solely to render documentation.
      - version: "2.0.0"
        text: >
          The website's user-facing outcomes must be defined as Backstop capabilities with concrete
          user journeys and executable acceptance evidence. At minimum, the site must enable a visitor
          to understand what Backstop is, evaluate fit, evaluate guarantees, evaluate compatibility,
          understand the system, adopt Backstop, apply it to concrete use cases, browse and extend the
          pack ecosystem, inspect supporting evidence, and continue beyond Backstop's intentional
          boundaries with useful guidance.

  - id: REQ-008
    version: "2.1.0"
    text: >
      The redesign must maintain a checked-in evidence inventory for consequential public claims.
      Each claim must identify durable repository or published sources and must map to at least one
      mechanism artifact — source, schema, test, or implementation commit — plus either a captured or
      reproducible execution artifact for runtime-behavior claims, or a durably recorded incident,
      example, or measurement with provenance and method for observed-outcome or failure claims.
      Conversation memory and unnamed real-world observations do not count. Across the site, the
      selected corpus must include at least one real failure incident, one failure-to-enforcement
      before/after example, one captured gate result, one source or commit trace, and one architecture
      view. Metrics are optional, but any metric used must have durable provenance and methodology.
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
          history, prior design-system and website dogfooding findings, known real-world agent failure
          modes, and measured delivery outcomes must be treated as evidence and bootstrap material during
          redesign. The site should not rely on a single proof mode: recognizable failure stories,
          before/after examples, real gate output, source and commit provenance, measurable outcomes,
          architecture diagrams, and concrete demonstrations should all be available somewhere in the
          public surface where they strengthen a consequential claim.
      - version: "2.1.0"
        text: >
          The redesign must maintain a checked-in evidence inventory for consequential public claims.
          Each claim must identify durable repository or published sources and must map to at least one
          mechanism artifact — source, schema, test, or implementation commit — plus either a captured or
          reproducible execution artifact for runtime-behavior claims, or a durably recorded incident,
          example, or measurement with provenance and method for observed-outcome or failure claims.
          Conversation memory and unnamed real-world observations do not count. Across the site, the
          selected corpus must include at least one real failure incident, one failure-to-enforcement
          before/after example, one captured gate result, one source or commit trace, and one architecture
          view. Metrics are optional, but any metric used must have durable provenance and methodology.

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
    version: "1.1.0"
    text: >
      This bundle owns the site's content topology, communicative responsibilities, source material,
      and product-boundary requirements; it does not contain final page copy. The Seed 1 public-product-
      model/content-topology spec must define page-level content responsibilities, and that spec's
      implementation plan must own authoring, rewriting, merging, and retiring final page copy after the
      information architecture and documentation-semantic contract are stable. A generalized prose-quality,
      writing-style, or prose-LSP pack remains separately governed work and is not a prerequisite for this bundle.
    versions:
      - version: "1.0.0"
        text: >
          This bundle owns the site's content topology, communicative responsibilities, source material,
          and product-boundary requirements; it does not own final page copy. Actual copywriting and
          page-level wording are deferred to downstream specs and plans. A generalized prose-quality,
          writing-style, or prose-LSP pack remains separate work and is not a prerequisite for this bundle.
      - version: "1.1.0"
        text: >
          This bundle owns the site's content topology, communicative responsibilities, source material,
          and product-boundary requirements; it does not contain final page copy. The Seed 1 public-product-
          model/content-topology spec must define page-level content responsibilities, and that spec's
          implementation plan must own authoring, rewriting, merging, and retiring final page copy after the
          information architecture and documentation-semantic contract are stable. A generalized prose-quality,
          writing-style, or prose-LSP pack remains separately governed work and is not a prerequisite for this bundle.

  - id: REQ-011
    version: "1.0.0"
    text: >
      Derived product documentation must follow a deterministic source-of-truth pipeline:
      authoritative project data and release history -> generated Markdown -> rendered site.
      CI must regenerate or verify derived Markdown and fail when checked-in or published output
      drifts from its authoritative source. The bundle owns this outcome, but if the generation
      engine or generic transformation substrate requires substantial independent design, that
      implementation may be separately scoped and may become a blocking dependency for this bundle.
    versions:
      - version: "1.0.0"
        text: >
          Derived product documentation must follow a deterministic source-of-truth pipeline:
          authoritative project data and release history -> generated Markdown -> rendered site.
          CI must regenerate or verify derived Markdown and fail when checked-in or published output
          drifts from its authoritative source. The bundle owns this outcome, but if the generation
          engine or generic transformation substrate requires substantial independent design, that
          implementation may be separately scoped and may become a blocking dependency for this bundle.

  - id: REQ-012
    version: "1.1.0"
    text: >
      Known dependencies named by this bundle — including the documentation-semantics pack in REQ-006
      and the released design-system pack in REQ-013 — remain separately governed and must be consumed
      through released, pinned interfaces. Additional deterministic documentation-generation machinery,
      harness/runtime capability, or generic Core primitive may be discovered and may become a hard
      enabler, but must receive separate ownership rather than being absorbed into BUNDLE-032 because
      the website is its first consumer. A generalized prose-quality, writing-style, or prose-LSP system
      is banked separate work and is not a prerequisite for BUNDLE-032. Out of scope means "not owned by
      this bundle," not "cannot be a dependency," except where this requirement explicitly rules out
      prerequisite status.
    versions:
      - version: "1.0.0"
        text: >
          Work discovered while implementing this bundle must preserve ownership boundaries. A new
          prose-quality system, documentation-semantics pack, deterministic docs-generation engine,
          harness/runtime capability, or generic Backstop Core primitive may be discovered and may even
          become a hard enabler, but it must be governed as separate work rather than being absorbed into
          BUNDLE-032 solely because the website is its first consumer. Out of scope means "not owned by
          this bundle," not "this bundle can never depend on it."
      - version: "1.1.0"
        text: >
          Known dependencies named by this bundle — including the documentation-semantics pack in REQ-006
          and the released design-system pack in REQ-013 — remain separately governed and must be consumed
          through released, pinned interfaces. Additional deterministic documentation-generation machinery,
          harness/runtime capability, or generic Core primitive may be discovered and may become a hard
          enabler, but must receive separate ownership rather than being absorbed into BUNDLE-032 because
          the website is its first consumer. A generalized prose-quality, writing-style, or prose-LSP system
          is banked separate work and is not a prerequisite for BUNDLE-032. Out of scope means "not owned by
          this bundle," not "cannot be a dependency," except where this requirement explicitly rules out
          prerequisite status.

  - id: REQ-013
    version: "1.0.0"
    text: >
      The public site must satisfy the executable visual and interaction policy owned by
      backstop-ai/backstop-design-system. Design-system compliance must be enforced through the
      installed, released pack rather than copied as website-local conventions, including the
      applicable token, styling, focus, reduced-motion, accessibility, wordmark, and reusable
      presentation rules that the design-system pack exposes for this site.
    versions:
      - version: "1.0.0"
        text: >
          The public site must satisfy the executable visual and interaction policy owned by
          backstop-ai/backstop-design-system. Design-system compliance must be enforced through the
          installed, released pack rather than copied as website-local conventions, including the
          applicable token, styling, focus, reduced-motion, accessibility, wordmark, and reusable
          presentation rules that the design-system pack exposes for this site.
---

# BUNDLE-032: Website Expansion

## Current Thinking

This is no longer a "make the docs look like the homepage" increment. The website should be
reconsidered as the public product surface for Backstop, with the current landing page and existing
Markdown docs treated as useful evidence rather than the final shape of the information architecture.

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

The site also needs an explicit category-definition surface. Backstop is not a model, agent, harness,
linter, CI system, or prompt framework. It is a framework for making intent legible and constraining
probabilistic agents with deterministic checks and verifiable evidence. The exact wording belongs
downstream, but the category boundary itself is a required communicative job.

This creates a new pack-design test. The reusable semantic contract for product documentation does
not belong in Core and does not belong in the visual design system. It should be expressible as a
separate pack. How that pack is authored and which deterministic engines it invents or composes is
intentionally a downstream design problem. The absence of an off-the-shelf deterministic tool is not
a reason to move the concern into probabilistic review or bake it into Core.

Website correctness should be expressed through Backstop's existing capability model rather than
through a hand-written list of ad hoc site checks at bundle level. The bundle defines the outcomes the
site must enable; downstream artifacts define the capabilities, @UJ-NNN scenarios, and executable
acceptance proof that establish those outcomes.

Derived documentation follows the same discipline. Authoritative product data should produce Markdown,
and Markdown should produce the site. If a page can be regenerated from changelog, release, capability,
or other project truth, CI should make drift impossible rather than asking an agent or maintainer to
remember to update prose manually.

## Resolved Design Questions

- **OQ-1 — What exact page/neighborhood inventory and navigation model best satisfies the visitor
  journey?** **Resolved:** The required content territories are the twelve entries in Content
  Neighborhoods, organized around visitor questions rather than repository taxonomy. Exact page count,
  route structure, labels, hierarchy, and responsive navigation behavior belong to the Seed 1 spec and
  plan after content inventory. **Rationale:** Binding bundle-level routes would recreate the premature
  SPEC-071 failure; leaving the territories unstated would make journey coverage optional. **Maps to:**
  REQ-002; DD-1, DD-3; Spec Seed 1.

- **OQ-2 — What exact Backstop capabilities and @UJ-NNN scenarios define success for the public
  site?** **Resolved:** The eleven outcomes in User Capability Seeds are mandatory; exact CAP IDs,
  scenario decomposition, and test implementation belong to the Seed 5 spec and plan. **Rationale:**
  Bundle scope must bind user outcomes without pre-authoring downstream acceptance artifacts. **Maps
  to:** REQ-007; DD-7; Spec Seed 5.

- **OQ-3 — How should the documentation-semantics pack represent and deterministically enforce its
  contract?** **Resolved:** A released, pinned, separately governed documentation-semantics pack is
  mandatory. BUNDLE-032 owns the consumer contract and installed integration evidence; the owner
  repository's artifact chain decides representation, engines, fixtures, and implementation.
  **Rationale:** Documentation semantics belong neither in Core nor in the visual design system, while
  first-consumer pressure does not transfer implementation ownership. **Maps to:** REQ-006, REQ-012;
  DD-6, DD-11; Spec Seed 2.

- **OQ-4 — Which existing content remains, merges, decomposes, or retires?** **Resolved:** Existing
  pages and Markdown are evidence and bootstrap material, not privileged permanent units. Seed 1
  inventories their useful claims, assigns each canonical concept one owner, and decides retention,
  merge, decomposition, or retirement before final-copy execution. **Rationale:** Preserving the five
  current documents by default would force repository history to become information architecture;
  rewriting without an inventory would lose grounded material. **Maps to:** REQ-001, REQ-003, REQ-010;
  DD-3; Spec Seed 1.

- **OQ-5 — What machine-readable publication shape should accompany the human site before a future
  docs-only MCP surface exists?** **Resolved:** No separate machine-only publication or MCP surface is
  required by this bundle. Canonical structured Markdown, deterministic derived Markdown, and the
  checked-in evidence inventory are the required inspectable substrate for both people and agents. A
  new publication format requires a downstream user journey that earns it and separate governance if it
  becomes a reusable mechanism. **Rationale:** A parallel machine narrative would create another truth
  surface before a demonstrated consumer need exists. **Maps to:** REQ-003, REQ-008, REQ-011, REQ-012;
  DD-2, DD-10, DD-11; Spec Seeds 1 and 3.

- **OQ-6 — How far may the visual redesign depart from the current landing page?** **Resolved:** The
  full site may depart materially from the current landing page; the current implementation is source
  material, not a visual baseline. Exact visual direction belongs to Seed 4, but every result must
  consume and pass the released design-system pack and must preserve the operational constraints in
  REQ-009. **Rationale:** Freezing the current hierarchy would contradict the full-site rethink;
  unconstrained local design would bypass the executable visual owner. **Maps to:** REQ-001, REQ-009,
  REQ-013; DD-9, DD-12; Spec Seed 4.

- **OQ-7 — Is a narrow product-site expansion an acceptable fallback if the full rethink creates too
  much churn?** **Resolved:** No. The full visitor-journey, evaluation, canonical-model, ecosystem,
  adoption, evidence, and reference outcomes are the only completion path. **Rationale:** An undefined
  fallback made the promoted scope optional and contradicted Promotion Readiness. **Maps to:**
  REQ-001@2.1.0; Spec Seed 1.

- **OQ-8 — Is the documentation-semantics pack mandatory or merely a possible discovered enabler?**
  **Resolved:** It is a known mandatory, separately governed, released-and-pinned dependency. Other
  generic machinery may still be discovered and separately governed. **Rationale:** Treating the same
  pack as both required and contingent made dependency ordering impossible to specify. **Maps to:**
  REQ-006@2.1.0, REQ-012@1.1.0; DD-6, DD-11; Spec Seed 2.

- **OQ-9 — May generalized prose-quality or prose-LSP work become a prerequisite?** **Resolved:** No.
  It remains banked, separately governed work and cannot block BUNDLE-032. Final copy is still owned
  downstream inside Seed 1. **Rationale:** The website can author and review bounded copy without first
  creating a generalized prose platform. **Maps to:** REQ-010, REQ-012@1.1.0; DD-11; Spec Seed 1.

- **OQ-10 — What evidence is sufficient for consequential public claims?** **Resolved:** Claims use
  the checked-in, claim-typed evidence inventory and durable sources defined by REQ-008; unnamed sessions
  and unrecorded observations are inadmissible, and metrics require durable provenance and method.
  **Rationale:** Plural evidence without a selection rule was not falsifiable and could not support
  deterministic acceptance. **Maps to:** REQ-004, REQ-008@2.1.0; DD-4, DD-8; Spec Seed 1.

## Draft Requirements

- **REQ-001 — Rethink the whole public site.** Existing pages are source material, not constraints;
  the full visitor-journey and evaluation outcomes are binding, with no narrower completion path.
- **REQ-002 — Navigate by visitor journey.** Discovery, evaluation, "what it is/isn't," understanding,
  adoption/use cases, pack ecosystem, extension, and reference each need an authoritative home.
- **REQ-003 — Maintain one canonical model.** Concepts and architecture have one comprehensive source;
  use-case content references it rather than creating competing explanations.
- **REQ-004 — Make the product auditable.** Consequential claims connect to mechanisms, guarantee
  boundaries, limitations, adoption implications, and direction; compatibility is not guarantee.
- **REQ-005 — State the boundary honestly.** Supported, limitation, planned, non-goal, and adjacent
  guidance are distinct states with distinct implications.
- **REQ-006 — Keep semantics, truth, and presentation separate.** Core owns product truth; the design
  system owns presentation; a released, pinned, separately governed pack owns reusable documentation
  semantics and enforcement as a known BUNDLE-032 dependency.
- **REQ-007 — Define success as user capabilities.** The site must support understanding, evaluation,
  adoption, application, ecosystem discovery/extension, evidence inspection, and continuation beyond
  intentional Backstop boundaries, with concrete journeys and executable acceptance proof downstream.
- **REQ-008 — Make evidence durable and claim-appropriate.** Consequential claims map through a
  checked-in inventory to mechanism evidence and the execution or incident evidence appropriate to
  the claim; session memory does not count, and metrics require durable provenance and methodology.
- **REQ-009 — Stay boring until requirements say otherwise.** Jekyll/GitHub Pages fits the current
  static problem; added runtime complexity must be purchased by actual functional requirements.
- **REQ-010 — Seed 1 owns the final-copy boundary.** The bundle does not contain final copy; the Seed 1
  spec defines page responsibilities and its plan owns authoring, rewrite, merge, and retirement after
  information architecture and semantic contracts stabilize. Generalized prose enforcement remains
  separate and non-blocking.
- **REQ-011 — Source of truth -> Markdown -> site.** Derived documentation is generated deterministically
  and CI rejects drift; a substantial generation engine may be separately governed if required.
- **REQ-012 — Preserve dependency ownership.** Known pack dependencies are separately released and
  pinned; newly discovered generic enablers receive separate governance; prose-style/LSP work is not
  a prerequisite for this bundle.
- **REQ-013 — Enforce the design system.** The site must satisfy the executable visual and interaction
  policy from the released design-system pack rather than re-encoding those rules locally.

## Content Neighborhoods

These are required territories, not final routes or page names:

- **Discovery / Why Backstop Exists** — recognize the failure class quickly and explain why this system exists.
- **What Backstop Is / Is Not** — establish the product category and prevent category mistakes.
- **Evaluation / Is This Right for Me?** — claims, mechanisms, guarantees, limitations, adoption cost, and fit.
- **Capabilities, Guarantees, Limits, and Direction** — supported behavior, limitations, planned work,
  intentional non-goals, and adjacent guidance.
- **How Backstop Works** — a progressive explanation of the operating model.
- **Canonical Concepts & Architecture** — the dense authoritative system model for deep human or agent ingestion.
- **Use Cases / Adoption Paths** — problem-oriented ways to adopt and apply Backstop without learning every noun first.
- **Pack Ecosystem** — browse existing packs, understand what problems they solve, how they compose, and which are maintained.
- **Extend Backstop** — determine whether a concern belongs in a pack and how to author one.
- **Reference** — CLI, artifact model, schemas, and exact behavioral lookup.
- **Project Status / Direction** — what exists now, what is planned, and where Backstop explicitly does not intend to own the problem.
- **Contributing / Ecosystem** — repository and contribution paths where appropriate.

## Spec Seeds

Five non-overlapping seeds partition all thirteen requirements in suggested implementation order.
Separately governed dependency artifacts remain in their owner repositories; these seeds own only
BUNDLE-032's product contract, consumption, integration, and end-to-end acceptance responsibilities.

- **Public product model, evidence contract, and content topology — REQ-001, REQ-002, REQ-003,
  REQ-004, REQ-005, REQ-008, REQ-010.** Define the complete information architecture and content
  inventory, canonical concept and architecture ownership, claim and boundary vocabulary, durable
  evidence inventory, and page-level communicative responsibilities. The Seed 1 spec owns those
  content contracts; its implementation plan owns authoring, rewriting, merging, and retiring final
  page copy after the information architecture and documentation-semantic contract are stable. Final-
  copy work is therefore inside this seed's delivery boundary without making the bundle itself a copy
  draft or making generalized prose tooling a prerequisite.
  Acceptance must prove that every required content neighborhood has one authoritative home, canonical
  concepts have no duplicate substantive owner, and every consequential claim has a valid evidence
  inventory entry. Injected missing sources, duplicate owners, and unclassified product boundaries must
  fail verification.

- **Dependency governance and documentation-semantics integration — REQ-006, REQ-012.** Define Core's
  consumer contract for the separately governed documentation-semantics pack and the release/pin
  boundary shared by it and the design-system pack. The documentation-semantics pack's own bundle,
  spec, plan, and implementation ledger live in its owner repository. BUNDLE-032 acceptance requires
  released bytes, positive and negative semantic fixtures, an installed-pipeline violation that blocks,
  and proof that local or unreleased bytes cannot satisfy the dependency.

- **Derived product-truth pipeline — REQ-011.** Specify authoritative inputs, deterministic generation
  into inspectable Markdown, generated-file ownership markers, regeneration behavior, and CI drift
  refusal. Clean regeneration must be byte-stable; an authoritative-source change must produce the
  expected Markdown delta; manual generated-output tampering must fail CI while naming the file and
  source; and the rendered site must consume the generated Markdown rather than a parallel truth.
  Any generalized transformation engine discovered here is separately governed.

- **Static public-site implementation and design-system enforcement — REQ-009, REQ-013.** Implement
  the redesigned surface on Jekyll and GitHub Pages unless a later functional requirement earns a
  different runtime. Preserve the custom-domain deployment, verify routes, internal links, navigation,
  responsive and no-JavaScript behavior, and consume the released design-system pack. Acceptance must
  run the installed pack against the actual site and show that injected token, inline-style, focus,
  motion, accessibility, wordmark, or reusable-presentation violations fail with the responsible rule
  and path; website-local duplicate policy does not count as compliance.

- **End-to-end website capabilities — REQ-007.** Author capability and @UJ artifacts plus executable
  journeys for the eleven named user outcomes. These journeys integrate Seeds 1–4 rather than re-owning
  their contracts, traverse built output, and prove understanding, fit and guarantee evaluation,
  compatibility evaluation, adoption, application, ecosystem browsing and extension, evidence
  inspection, and continuation beyond Backstop's boundary. Removing a required route, evidence edge,
  dependency result, or boundary explanation must break at least one journey.

## User Capability Seeds

The exact CAP IDs and @UJ-NNN scenarios belong downstream, but the required outcomes are now stable enough to seed promotion:

- **Understand Backstop** — determine what Backstop is, what it is not, and why it exists.
- **Evaluate Fit** — determine whether Backstop addresses a concrete problem the visitor has.
- **Evaluate Guarantees** — distinguish supported behavior, guarantees, limitations, plans, non-goals, and guidance.
- **Evaluate Compatibility** — determine whether Backstop works with a chosen harness/model/toolchain and where compatibility stops short of guarantee.
- **Understand the System** — build an accurate mental model of artifacts, packs, gates, capabilities, enforcement, and architecture.
- **Adopt Backstop** — get from interest to a working installation/configuration for a real repository.
- **Apply Backstop** — find the right path for a concrete use case.
- **Browse the Pack Ecosystem** — discover existing packs and understand which ones address the visitor's problem.
- **Extend Backstop** — determine when and how to author a new pack.
- **Inspect the Evidence** — trace consequential claims to implementation, examples, source, commits, or measured results.
- **Continue Beyond Backstop** — understand the intentional boundary and receive useful adjacent guidance when Backstop does not own the whole problem.

The natural progression is roughly Understand -> Evaluate -> Understand System -> Adopt -> Apply -> Extend,
but this is a graph rather than a forced funnel. Evidence inspection and continuation beyond Backstop are
cross-cutting capabilities available wherever a consequential claim or product boundary appears.

**Audience context, not a separate requirement.** Referred evaluation is the highest-priority acquisition
lens for downstream information-architecture and copy decisions: a person or agent receives a Backstop link
in response to a concrete failure and needs to recognize the problem, inspect the mechanism and evidence,
understand the boundary, and choose a deeper path quickly. This is design context for satisfying REQ-002
and REQ-007, not a required route, linear funnel, attribution mechanism, or additional acceptance outcome.
The eleven capability outcomes remain the binding contract.

## Source Material

The redesign should deliberately mine:

- the current `backstop.sh` landing page and technical docs at `docs/index.html`, `docs/_config.yml`,
  `docs/getting-started.md`, `docs/concepts.md`, `docs/artifact-workflow.md`,
  `docs/pack-authoring.md`, and `docs/cli-reference.md`, bootstrapped in commit
  `33aff3b4810205c85d3893f8c2d2f30c24daed90`;
- the first released design-system consumption and landing-page implementation in commit
  `63f70f7e668486202cc1897cfcce94f82769b477`, especially `docs/index.html`,
  `docs/assets/css/backstop-tokens.css`, `docs/assets/css/backstop.css`, `backstop.yml`, and
  `backstop.lock`;
- the design-system dogfood incidents recorded durably as ISSUE-182, ISSUE-183, and ISSUE-184 and
  introduced in commit `36ad240b7e52655563542de265b865c244dc1875`;
- Backstop's actual implementation, schemas, tests, and commit history, including
  `artifacts/capability/v1/schema.json`, `pkg/pack/engine/binding.go`, and
  `pkg/recipe/manifest.go`;
- downstream Seed 1 research may inspect external documentation patterns for applicability boundaries,
  compatibility semantics, non-goals, and maturity/status distinctions. Any pattern adopted into the
  site contract must be cited by exact source URL, title, and access or version date in the resulting
  spec; unnamed external patterns are research prompts, not evidence for a settled bundle decision.

These inputs are evidence to synthesize, not text to preserve verbatim. Conversation-only recollection,
unnamed sessions, and unrecorded observations are discovery prompts rather than admissible evidence. No
durable measured-delivery source currently exists in this repository; delivery metrics become eligible
only after a versioned report or dataset records their provenance and method.

## Scope and Dependency Rules

Out of scope means **not owned by this bundle**, not **forbidden** and not **incapable of becoming a dependency**.

- A generalized prose-quality/style pack or prose LSP is not a BUNDLE-032 deliverable or prerequisite.
  It remains banked as separately governed work and cannot become a hard enabler for this bundle.
- The documentation-semantics pack is a known mandatory dependency. It must be separately governed,
  released, pinned, and consumed through its declared interface. Its initial scope should extend only as
  far as real website requirements force it; generalized documentation governance remains separate scope.
- BUNDLE-032 does not own Backstop Core changes. A genuinely generic missing pack primitive discovered through
  dogfooding should receive its own bundle rather than teaching Core about documentation.
- An MCP server is not delivered by this bundle. The site may document the intended documentation/guidance role and
  the non-goal of agent-selected MCP execution.
- Deterministic harness/runtime evolution is not delivered by this bundle. The site documents current compatibility
  and guarantee boundaries; execution-layer improvements remain separate work.
- Migrating away from Jekyll/GitHub Pages is not a goal of this bundle. It remains allowed if new functional
  requirements earn the complexity.
- The site describes actual Backstop product truth. If writing the site exposes a real product gap, that gap may be
  separately governed and may become a dependency; the website does not invent or silently implement features merely
  to make the narrative cleaner.
- Useful scope exhaust should be captured when it prevents future rediscovery, but public artifacts should not become
  a dumping ground for speculative commercial or product ideas that are irrelevant to this bundle's boundary.

## Draft Design Decisions

- **DD-1 — Journey over taxonomy.** Internal nouns support the journey; they do not dictate the top-level IA.
- **DD-2 — Human journey is the agent journey.** Do not create a parallel machine-only narrative; make canonical content structured and complete enough for both.
- **DD-3 — Canonical model plus progressive disclosure.** Dense architecture/reference content coexists with problem/use-case entry paths.
- **DD-4 — Decision support, not brochureware.** Claims must expose mechanism, evidence, boundary, adoption implications, and direction.
- **DD-5 — Compatibility is not guarantee.** A tool may operate Backstop without guaranteeing correct lifecycle use.
- **DD-6 — Pack boundaries remain real.** Product truth, presentation, and documentation semantics live
  in separate owners. The documentation-semantics pack is a known released-and-pinned dependency, not a
  contingent discovery or an excuse to absorb semantics into Core or the visual design system.
- **DD-7 — Capabilities prove the site.** User outcomes and journeys drive acceptance rather than screenshot approval alone.
- **DD-8 — Evidence is selected by claim type.** Every consequential claim is inventoried against durable
  mechanism evidence and the execution, incident, example, or measured evidence appropriate to what it
  asserts. The site as a whole must cover the minimum plural corpus in REQ-008; metrics remain optional
  unless their provenance and method are durable.
- **DD-9 — Complexity must be earned.** Use the least operationally complex stack that satisfies concrete requirements.
- **DD-10 — Generated docs stay inspectable.** Authoritative source -> Markdown -> site is the preferred derivation path.
- **DD-11 — Known dependencies and discovered enablers stay distinct.** Named pack dependencies are
  separately released and pinned; future generic engines, runtime capabilities, or Core primitives get
  their own governance. First-consumer pressure does not erase ownership boundaries, and prose-LSP/style
  work remains explicitly non-blocking.
- **DD-12 — Presentation policy is executable.** The site consumes and satisfies the released design-system pack; it does not reproduce those rules as local convention.

## Promotion Readiness

The bundle-level problem, boundaries, neighborhoods, capability outcomes, evidence posture, ownership seams,
and dependency rules are now sufficiently defined to decompose work through the five-seed partition above.
Promotion must not copy the old premature SPEC-071 shape forward or substitute a narrower site increment. New
specs must preserve the exact requirement ownership of the seeds while expressing the full-site,
capability-driven, evidence-first scope captured here.

## Sharp Edges

- **Liquid/Jinja delimiter collision (ISSUE-182).** A recipe that emits downstream `{{ ... }}` syntax
  currently collides with Backstop recipe substitution. Any seed that emits Liquid/Jinja templates must
  depend on a released ISSUE-182 fix or use a design that avoids nested templating, and must prove literal
  output bytes plus byte-identical reapplication. Fragile escaping cannot be assumed as infrastructure.
- **Cross-repository release order.** Documentation-semantics and design-system changes must become
  fixture-green, released, and tagged in their owner repositories before Core consumes clean released bytes
  and refreshes its lock. Local checkouts, stale installed copies, or a successful relock alone are not
  acceptance evidence; ISSUE-183 records the observed stale-local-pack failure.
- **Source/generated Markdown ownership.** Each derived surface must name its authoritative inputs,
  generated outputs, ownership markers, and regeneration command. Generated output is not hand-edited,
  clean regeneration is no-diff, and CI must diagnose the responsible source/output pair on drift. Two
  writable truths are forbidden.
- **Fixture path fidelity (ISSUE-184).** Pack fixtures can evade their own production path filters while
  reporting only a generic negative-fixture miss. Documentation-semantic and design-system fixtures must
  preserve production-relative paths, and final acceptance must execute installed packs against the built
  site rather than trusting pack-test alone.

## Version History

- 0.1.0 (2026-08-23): Initial docs-shell-oriented scope.
- 0.2.0 (2026-08-23): Reframed as a full public-site rethink with visitor-journey IA, canonical architecture,
  evidence-first product evaluation, explicit product-boundary semantics, separate documentation-semantics pack,
  capability/user-journey acceptance, and boring-until-earned deployment.
- 0.3.0 (2026-08-23): Locked bundle decisions after OQ/OOS review. Added what-it-is/isn't and pack-ecosystem
  neighborhoods, stable user-capability seeds, evidence diversity, referred-evaluation entry model,
  source-of-truth -> Markdown -> site generation with CI drift detection, and explicit separate-governance rules
  for discovered enabling work and dependencies.
- 0.4.0 (2026-08-23): Restored explicit design-system enforcement as REQ-013 after requirement generation had
  overwritten the earlier presentation-policy requirement. Site presentation remains pack-owned and executable.
- 0.5.0 (2026-08-23): Repaired the defined bundle after bundle review. Removed the contradictory
  narrow-site fallback; made the documentation-semantics pack a known separately governed dependency;
  confirmed prose-style/LSP work is not a prerequisite; replaced unnamed dogfooding and outcome sources
  with durable repository evidence and a claim-type evidence-selection rule; added a complete five-seed,
  thirteen-requirement partition with executable evidence criteria; and documented Liquid-template,
  cross-repository release-order, fixture-fidelity, and source/generated-Markdown sharp edges. Amended
  REQ-001, REQ-006, and REQ-008 to v2.1.0 and REQ-012 to v1.1.0; maturity remains defined.
- 0.5.1 (2026-08-23): Restored the ten resolved design questions and mapped each resolution to its
  requirements, decisions, and spec seeds; made referred evaluation explicit nonbinding audience context;
  assigned final-copy execution to the Seed 1 implementation plan; and recast unnamed external
  documentation patterns as downstream research requiring exact citations before adoption. Amended
  REQ-010 to v1.1.0 to make final-copy ownership explicit; all other requirement versions, the five-seed
  partition, and defined maturity remain unchanged.

## References

- `docs/index.html`, `docs/_config.yml`, and the current named `docs/*.md` files — existing public-site source material.
- Commit `33aff3b4810205c85d3893f8c2d2f30c24daed90` — technical-documentation bootstrap.
- Commit `63f70f7e668486202cc1897cfcce94f82769b477` — first released design-system consumption and landing page.
- `backstop-ai/backstop-design-system@v0.1.0`, pinned with content identity in `backstop.lock` — reusable visual/interaction policy owner.
- ISSUE-182, ISSUE-183, ISSUE-184 and commit `36ad240b7e52655563542de265b865c244dc1875` — durable design-system dogfood incidents.
- `artifacts/capability/v1/schema.json` — existing capability/user-journey acceptance model.
- `pkg/pack/engine/binding.go` and `pkg/recipe/manifest.go` — current pack-engine and recipe-substrate mechanisms.
