---
title: Product Model
layout: default
permalink: /model/
hero_question: "How does it work?"
hero_lede: "Every named piece of the product surface: what it is, what it does, what it is for, and how it fits."
---

<div class="tactics-intro">
<p class="tactics-kicker">The product is a taxonomy, not a narrative.</p>
<p class="tactics-bridge">If a piece cannot be named in one sentence, it does not belong on this page.</p>
</div>

<div class="tactics-matrix taxonomy-matrix" data-overflow-region>
<table>
<thead>
<tr><th>Piece</th><th>What it is</th><th>What it does</th><th>Value</th><th>How it fits</th></tr>
</thead>
<tbody>
<tr><td data-label="Piece">Operating model</td><td data-label="What it is">The path from declared intent to an inspectable verdict</td><td data-label="What it does">Takes work through a reviewed plan, implementation, and a gate</td><td data-label="Value">The agent is not asked to invent the process</td><td data-label="How it fits">Every other row is a named piece of this path</td></tr>
<tr><td data-label="Piece">Work tracks</td><td data-label="What it is">Two legal ways work can start</td><td data-label="What it does">Bundles become specs then plans; issues become plans</td><td data-label="Value">Discovery cannot masquerade as implementation</td><td data-label="How it fits">Both tracks meet at a reviewed plan</td></tr>
<tr><td data-label="Piece">Intent artifacts</td><td data-label="What it is">The durable records of why work exists</td><td data-label="What it does">Hold requirements, decisions, claims, and evidence</td><td data-label="Value">A later session can resume without reconstructing context</td><td data-label="How it fits">They are the working state the operating model moves</td></tr>
<tr><td data-label="Piece">Bounded execution</td><td data-label="What it is">A plan that names order, tests, files, and gates</td><td data-label="What it does">Tells the implementer what to do and what not to redesign</td><td data-label="Value">Scope cannot quietly expand mid-task</td><td data-label="How it fits">Sits between the artifacts and the gate</td></tr>
<tr><td data-label="Piece">Delivery lifecycle</td><td data-label="What it is">The states work is allowed to occupy</td><td data-label="What it does">Records delivered, replaced, canceled, deprecated, or obsoleted</td><td data-label="Value">Closure is a fact, not a vibe</td><td data-label="How it fits">Reactive and proactive work converge here before a verdict</td></tr>
<tr><td data-label="Piece">Standards packs</td><td data-label="What it is">Versioned standards with engines and fixtures</td><td data-label="What it does">Own the rules Core runs</td><td data-label="Value">The standard is not a prompt</td><td data-label="How it fits">Core stays a thin executor; packs own the checks</td></tr>
<tr><td data-label="Piece">Recipes</td><td data-label="What it is">Repeatable repository operations</td><td data-label="What it does">Scaffold and configure without moving standards into Core</td><td data-label="Value">Adoption steps stay deterministic</td><td data-label="How it fits">Beside packs, not instead of them</td></tr>
<tr><td data-label="Piece">Gates and policy</td><td data-label="What it is">The run that returns pass or fail</td><td data-label="What it does">Resolves packs, runs engines, applies severity</td><td data-label="Value">Same input, same verdict</td><td data-label="How it fits">The stop in the operating model</td></tr>
<tr><td data-label="Piece">Baselines and ratchets</td><td data-label="What it is">A record of existing debt, plus a rule that it cannot grow</td><td data-label="What it does">Lets a repo adopt without pretending it is clean</td><td data-label="Value">Honest green</td><td data-label="How it fits">Policy for the gate, not a substitute for a waiver</td></tr>
<tr><td data-label="Piece">Waivers</td><td data-label="What it is">A named, time-bounded exception</td><td data-label="What it does">Permits a specific finding without deleting the rule</td><td data-label="Value">Exceptions stay visible</td><td data-label="How it fits">Last resort next to baselines</td></tr>
<tr><td data-label="Piece">Enforcement loop</td><td data-label="What it is">Intent, agent, engines, verdict, evidence, back to intent</td><td data-label="What it does">Puts the gate in the agent's working loop</td><td data-label="Value">Failure arrives while the agent still owns the task</td><td data-label="How it fits">The loop Evaluate described; CI confirms it later</td></tr>
<tr><td data-label="Piece">Provenance</td><td data-label="What it is">Evidence tied to source, command, and intent</td><td data-label="What it does">Makes a green result inspectable later</td><td data-label="Value">Green without provenance is not a delivery fact</td><td data-label="How it fits">What a gate verdict must be able to point at</td></tr>
<tr><td data-label="Piece">Capabilities</td><td data-label="What it is">Named outcomes, plus journeys that prove them</td><td data-label="What it does">States what a user can do and demonstrates it against the built system</td><td data-label="Value">Acceptance is an outcome, not a page of claims</td><td data-label="How it fits">How this site itself is proven</td></tr>
<tr><td data-label="Piece">Harness</td><td data-label="What it is">Whatever runtime invokes Backstop</td><td data-label="What it does">Schedules work and must propagate the exit</td><td data-label="Value">Backstop can sit in an agent, CI, or a script</td><td data-label="How it fits">The harness owns orchestration; Backstop owns the verdict</td></tr>
<tr><td data-label="Piece">Ownership</td><td data-label="What it is">Who owns execution, standards, orchestration, and tools</td><td data-label="What it does">Keeps Core thin and packs in their lane</td><td data-label="Value">No second editable truth</td><td data-label="How it fits">The map of who is allowed to change what</td></tr>
<tr><td data-label="Piece">Product category</td><td data-label="What it is">Delivery discipline around the agent you already use</td><td data-label="What it does">Stops work that is off-task, unreviewable, or not allowed to ship</td><td data-label="Value">The agent can be wrong; the boundary still holds</td><td data-label="How it fits">Evaluate's job; this row names the surface above</td></tr>
</tbody>
</table>
</div>

<div class="canonical-anchors">

## Operating model {#operating-model}

<!-- backstop-claim: CLAIM-022 -->
Backstop converts declared intent into bounded work and an inspectable verdict. Proactive work begins with a bundle; reactive work begins with an issue. Both reach a reviewed plan before implementation, and both finish through deterministic validation and explicit terminal state. Terminal state records whether work was delivered, replaced, canceled, deprecated, or obsoleted; issue closure preserves either completed-plan lineage through `delivered_by` or a direct typed artifact, commit, or pull-request reference through `resolved-by`.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-010 -->
[Inspect artifact schemas](/reference/#artifact-schema-catalog)

## Product category {#product-category}

<!-- backstop-claim: CLAIM-020 -->
Backstop is not a coding agent. It sits around whichever agent you already use and stops work that is off-task, unreviewable, or not allowed to ship.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-005 -->
[Find adjacent guidance](/status/#adjacent-guidance)

## Intent artifacts {#intent-artifacts}
{: .taxonomy-anchor}

## Work tracks {#work-tracks}
{: .taxonomy-anchor}

## Bounded execution {#bounded-execution}
{: .taxonomy-anchor}

## Standards packs {#standards-packs}
{: .taxonomy-anchor}

## Recipes {#recipes}
{: .taxonomy-anchor}

## Gates and policy {#gates-and-policy}

<!-- backstop-claim: CLAIM-011 -->
The gate checks the installed standards against the named inputs and returns a verdict. That is the whole guarantee. Whether anyone stops is outside the process.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-006 -->
[Review support and limits](/status/#supported-and-limited)

## Baselines and ratchets {#baselines-and-ratchets}
{: .taxonomy-anchor}

## Waivers {#waivers}
{: .taxonomy-anchor}

## Capabilities and journeys {#capabilities-and-journeys}
{: .taxonomy-anchor}

## Provenance and verification {#provenance-and-verification}

<!-- backstop-journey-link: JLINK-021 -->
[Trace the sources](/reference/#source-traceability)

## Harness integration {#harness-integration}

<!-- backstop-claim: CLAIM-019 -->
A harness can schedule work and invoke Backstop. It preserves Backstop guarantees only when it also respects artifact order, role boundaries, and the gate exit status.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-008 -->
[Check compatibility details](/reference/#compatibility)

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

</div>
