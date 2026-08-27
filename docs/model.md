---
title: Product Model
layout: default
permalink: /model/
hero_question: "How does Backstop turn intent into a trustworthy verdict?"
---

## Operating model {#operating-model}

<!-- backstop-claim: CLAIM-022 -->
Backstop converts declared intent into bounded work and an inspectable verdict. Proactive work begins with a bundle; reactive work begins with an issue. Both reach a reviewed plan before implementation, and both finish through deterministic validation and explicit terminal state. Terminal state records whether work was delivered, replaced, canceled, deprecated, or obsoleted; issue closure preserves either completed-plan lineage through `delivered_by` or a direct typed artifact, commit, or pull-request reference through `resolved-by`.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-010 -->
[Inspect artifact schemas](/reference/#artifact-schema-catalog)

## Product category {#product-category}

Backstop is AI delivery discipline: a deterministic control surface around probabilistic execution. It does not need the agent to be right on the first pass; it needs mistakes to become visible before they escape the repository boundary.

## Intent artifacts {#intent-artifacts}

Bundles, specifications, plans, issues, decisions, capabilities, and journeys preserve why work exists, who owns the next transformation, and what evidence closes it.

## Work tracks {#work-tracks}

The proactive track decomposes a bundle into independently accepted specifications. The reactive track turns a bounded issue directly into a plan. The distinction keeps discovery work from masquerading as an implementation task.

## Bounded execution {#bounded-execution}

A plan names task order, test names, file scope, gates, and dependencies. Implementers execute that contract; they do not silently redesign it.

## Standards packs {#standards-packs}

Packs are versioned standards declarations. They own rules, engines, fixtures, and findings while Core remains the thin executor that resolves and runs them.

## Recipes {#recipes}

Recipes compose repeatable repository operations without transferring standards ownership into Core.

## Gates and policy {#gates-and-policy}

A gate resolves installed packs, runs their declared engines, applies severity and policy, and emits a blocking or passing verdict.

## Baselines and ratchets {#baselines-and-ratchets}

A baseline records accepted existing debt. A ratchet prevents new debt from expanding it, so adoption can start honestly without making regression permissible.

## Waivers {#waivers}

Waivers are explicit, attributable, and time-bounded exceptions. They preserve the governing standard instead of weakening it invisibly.

## Capabilities and journeys {#capabilities-and-journeys}

Capabilities state outcomes. User journeys demonstrate those outcomes from a visitor or operator perspective against built behavior.

## Provenance and verification {#provenance-and-verification}

Evidence binds a claim or verdict to immutable source, exact execution, and governing intent. Green without provenance is not a durable delivery fact.

## Harness integration {#harness-integration}

<!-- backstop-claim: CLAIM-019 -->
A harness can schedule work and invoke Backstop. It preserves Backstop guarantees only when it also respects artifact order, role boundaries, and the gate exit status.
<!-- /backstop-claim -->

## Delivery lifecycle {#delivery-lifecycle}

<!-- backstop-claim: CLAIM-023 -->
The authoritative lifecycle is `docs/_diagrams/ARCH-001-delivery-lifecycle.mmd`: reactive issues and proactive bundles converge on reviewed plans, bounded implementation, validation, and terminal outcomes.
<!-- /backstop-claim -->

## Enforcement loop {#enforcement-loop}

The authoritative enforcement loop is `docs/_diagrams/ARCH-002-enforcement-loop.mmd`: intent bounds execution, pack engines return a verdict, and evidence feeds provenance back into intent.

<!-- backstop-journey-link: JLINK-014 -->
[Read the gate reference](/reference/#gate)

## Ownership boundaries {#ownership-boundaries}

<!-- backstop-claim: CLAIM-010 -->
The canonical architecture views are checked-in Mermaid sources so presentation cannot become a second editable truth.
<!-- /backstop-claim -->

Core owns execution and lifecycle primitives. Packs own standards and engines. Harnesses own orchestration. External toolchains own their behavior. The authoritative boundary view is `docs/_diagrams/ARCH-003-ownership-boundaries.mmd`.

<!-- backstop-journey-link: JLINK-011 -->
[Review project boundaries](/status/#project-boundaries)
