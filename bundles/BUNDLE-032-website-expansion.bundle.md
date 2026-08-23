---
title: "Website Expansion"
number: BUNDLE-032
created: "2026-08-23"
schema_version: bundle/v2

bundle:
  name: website-expansion
  version: "0.3.0"
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
    version: "2.0.0"
    text: >
      The current site, existing technical documentation, repository implementation and commit
      history, prior design-system and website dogfooding findings, known real-world agent failure
      modes, and measured delivery outcomes must be treated as evidence and bootstrap material during
      redesign. The site should not rely on a single proof mode: recognizable failure stories,
      before/after examples, real gate output, source and commit provenance, measurable outcomes,
      architecture diagrams, and concrete demonstrations should all be available somewhere in the
      public surface where they strengthen a consequential claim.
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
    version: "1.0.0"
    text: >
      Work discovered while implementing this bundle must preserve ownership boundaries. A new
      prose-quality system, documentation-semantics pack, deterministic docs-generation engine,
      harness/runtime capability, or generic Backstop Core primitive may be discovered and may even
      become a hard enabler, but it must be governed as separate work rather than being absorbed into
      BUNDLE-032 solely because the website is its first consumer. Out of scope means "not owned by
      this bundle," not "this bundle can never depend on it."
    versions:
      - version: "1.0.0"
        text: >
          Work discovered while implementing this bundle must preserve ownership boundaries. A new
          prose-quality system, documentation-semantics pack, deterministic docs-generation engine,
          harness/runtime capability, or generic Backstop Core primitive may be discovered and may even
          become a hard enabler, but it must be governed as separate work rather than being absorbed into
          BUNDLE-032 solely because the website is its first consumer. Out of scope means "not owned by
          this bundle," not "this bundle can never depend on it."
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

## Draft Requirements

- **REQ-001 — Rethink the whole public site.** Existing pages are source material, not constraints;
  a narrower product-site expansion is a fallback if the broader rethink does not earn its churn.
- **REQ-002 — Navigate by visitor journey.** Discovery, evaluation, "what it is/isn't," understanding,
  adoption/use cases, pack ecosystem, extension, and reference each need an authoritative home.
- **REQ-003 — Maintain one canonical model.** Concepts and architecture have one comprehensive source;
  use-case content references it rather than creating competing explanations.
- **REQ-004 — Make the product auditable.** Consequential claims connect to mechanisms, guarantee
  boundaries, limitations, adoption implications, and direction; compatibility is not guarantee.
- **REQ-005 — State the boundary honestly.** Supported, limitation, planned, non-goal, and adjacent
  guidance are distinct states with distinct implications.
- **REQ-006 — Keep semantics, truth, and presentation separate.** Core owns product truth; the design
  system owns presentation; a separate pack owns reusable documentation semantics and enforcement.
- **REQ-007 — Define success as user capabilities.** The site must support understanding, evaluation,
  adoption, application, ecosystem discovery/extension, evidence inspection, and continuation beyond
  intentional Backstop boundaries, with concrete journeys and executable acceptance proof downstream.
- **REQ-008 — Prove it in more than one way.** Real failures, before/after examples, gate output,
  commit/source provenance, measured outcomes, diagrams, and demonstrations are all valid proof modes.
- **REQ-009 — Stay boring until requirements say otherwise.** Jekyll/GitHub Pages fits the current
  static problem; added runtime complexity must be purchased by actual functional requirements.
- **REQ-010 — Bundle owns topology, not prose.** Actual page copy and any generalized prose-style
  enforcement belong downstream or in separately governed enabling work.
- **REQ-011 — Source of truth -> Markdown -> site.** Derived documentation is generated deterministically
  and CI rejects drift; a substantial generation engine may be separately governed if required.
- **REQ-012 — Preserve scope ownership when new enablers emerge.** Separately scoped work may become a
  hard dependency; that does not make it part of this bundle.

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

The primary acquisition case is referred evaluation: someone was given a Backstop link because another
person or agent believed it would help with a concrete failure and now wants to be convinced quickly.
The site should recognize that problem, explain the mechanism, show credible evidence, state the boundary,
and then offer deeper paths rather than requiring a cold visitor to first learn Backstop's internal taxonomy.

## Source Material

The redesign should deliberately mine:

- the current `backstop.sh` landing page and technical docs;
- the prior design-system and website dogfooding session, especially where real usage changed the product framing;
- Backstop's actual implementation and commit history;
- pack manifests across `backstop-ai`, including the mechanism/opinion/recipe separation already encoded by the ecosystem;
- capability and user-journey schemas already present in Backstop;
- real agent/harness failure stories encountered while building and using Backstop;
- measured delivery/quality evidence where it can support a claim without overstating causality;
- external documentation patterns worth borrowing, especially explicit applicability boundaries, compatibility semantics,
  non-goals, and maturity/status distinctions.

These inputs are evidence to synthesize, not text to preserve verbatim.

## Scope and Dependency Rules

Out of scope means **not owned by this bundle**, not **forbidden** and not **incapable of becoming a dependency**.

- A generalized prose-quality/style pack or prose LSP is not a BUNDLE-032 deliverable. If copy throughput or quality
  proves it necessary, scope it separately and consume it as an enabler.
- The documentation-semantics pack should be built only as far as real website requirements force it to be built.
  Generalized documentation governance beyond that evidence is separate scope.
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
- **DD-6 — Pack boundaries remain real.** Product truth, presentation, and documentation semantics live in separate owners.
- **DD-7 — Capabilities prove the site.** User outcomes and journeys drive acceptance rather than screenshot approval alone.
- **DD-8 — Evidence should be plural.** Narrative, source, runtime output, metrics, architecture, and demos reinforce each other.
- **DD-9 — Complexity must be earned.** Use the least operationally complex stack that satisfies concrete requirements.
- **DD-10 — Generated docs stay inspectable.** Authoritative source -> Markdown -> site is the preferred derivation path.
- **DD-11 — New enablers get their own governance.** First-consumer pressure does not erase ownership boundaries.

## Promotion Readiness

The bundle-level problem, boundaries, neighborhoods, capability outcomes, evidence posture, ownership seams,
and dependency rules are now sufficiently defined to generate promoted requirements and decompose the work into
one or more specs. Promotion should not copy the old premature SPEC-071 shape forward; the requirements generated
from this bundle need to reflect the full-site, capability-driven, evidence-first scope captured in v0.3.0.

## Version History

- 0.1.0 (2026-08-23): Initial docs-shell-oriented scope.
- 0.2.0 (2026-08-23): Reframed as a full public-site rethink with visitor-journey IA, canonical architecture,
  evidence-first product evaluation, explicit product-boundary semantics, separate documentation-semantics pack,
  capability/user-journey acceptance, and boring-until-earned deployment.
- 0.3.0 (2026-08-23): Locked bundle decisions after OQ/OOS review. Added what-it-is/isn't and pack-ecosystem
  neighborhoods, stable user-capability seeds, evidence diversity, referred-evaluation entry model,
  source-of-truth -> Markdown -> site generation with CI drift detection, and explicit separate-governance rules
  for discovered enabling work and dependencies.

## References

- `docs/index.html` and current `docs/*.md` — existing public-site source material.
- `backstop-ai/backstop-design-system` — reusable visual/interaction policy owner.
- `backstop-ai/*` pack manifests — evidence for mechanism/opinion/recipe ownership separation.
- `artifacts/capability/v1/schema.json` — existing capability/user-journey acceptance model.
- Prior website/design-system dogfooding session — first-party evidence that real usage changed product framing.
