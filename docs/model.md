---
title: Product Model
layout: default
permalink: /model/
hero_question: "How does it work?"
hero_lede: "Define the work. Enforce your standards. Detect drift."
---

## Define the work {#operating-model}

One bundle. Many specs. Each spec one plan.

<div class="work-topology">
<article>
<p>When specs depend</p>
<div class="topo-bundle">Bundle</div>
<div class="topo-fan">
<div class="topo-col"><span>Spec A</span><span>Plan A</span></div>
<div class="topo-col"><span>Spec B</span><span>Plan B</span></div>
<div class="topo-col"><span>Spec C</span><span>Plan C</span></div>
</div>
<p class="topo-order depends">A first → B next → C after B</p>
</article>
<article>
<p>When they do not</p>
<div class="topo-bundle">Bundle</div>
<div class="topo-fan">
<div class="topo-col"><span>Spec A</span><span>Plan A</span></div>
<div class="topo-col"><span>Spec B</span><span>Plan B</span></div>
<div class="topo-col"><span>Spec C</span><span>Plan C</span></div>
</div>
<p class="topo-order parallel">A, B, and C can run together</p>
</article>
</div>

<div class="tactics-matrix legend-matrix">
<table>
<tbody>
<tr><td data-label="Piece">Bundle</td><td data-label="What it is">The body of work</td></tr>
<tr><td data-label="Piece">Spec</td><td data-label="What it is">One bounded implementation contract</td></tr>
<tr><td data-label="Piece">Plan</td><td data-label="What it is">The ordered steps that realize that spec</td></tr>
<tr><td data-label="Piece">Dependencies</td><td data-label="What it is">The known-safe execution order</td></tr>
</tbody>
</table>
</div>

Independent branches can execute in parallel. Dependencies establish the order when they cannot.

A bounded fix skips the bundle and spec. The issue becomes a plan.

<div class="canonical-note">
<!-- backstop-claim: CLAIM-022 -->
Backstop converts declared intent into bounded work and an inspectable verdict. Proactive work begins with a bundle; reactive work begins with an issue. Both reach a reviewed plan before implementation, and both finish through deterministic validation and explicit terminal state. Terminal state records whether work was delivered, replaced, canceled, deprecated, or obsoleted; issue closure preserves either completed-plan lineage through `delivered_by` or a direct typed artifact, commit, or pull-request reference through `resolved-by`.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-010 -->
[Inspect artifact schemas](/reference/#artifact-schema-catalog)
</div>

## Enforce your standards {#standards-packs}

<div class="tactics-matrix">
<table>
<tbody>
<tr><td data-label="Piece">Packs</td><td data-label="What it is">Versioned standards with engines and fixtures. Core only runs what a pack declares.</td></tr>
<tr><td data-label="Piece">Gate</td><td data-label="What it is">Pass or fail on the installed packs. Same input, same verdict.</td></tr>
<tr><td data-label="Piece">Recipes</td><td data-label="What it is">Pinned scaffolding — a project, a pipeline, a config. They do not own the standard.</td></tr>
</tbody>
</table>
</div>

## Detect drift {#baselines-and-ratchets}

<div class="tactics-matrix">
<table>
<tbody>
<tr><td data-label="Piece">Baseline</td><td data-label="What it is">Existing debt stays visible. Touched code and new violations fail.</td></tr>
<tr><td data-label="Piece">Waiver</td><td data-label="What it is">A named, time-bounded exception. The rule stays.</td></tr>
<tr><td data-label="Piece">Provenance</td><td data-label="What it is">Green has to point at source, command, and tests.</td></tr>
</tbody>
</table>
</div>

## While the agent is working {#enforcement-loop}

<div class="tactics-matrix">
<table>
<tbody>
<tr><td data-label="Piece">Agent loop</td><td data-label="What it is">The gate fails to the agent. The agent fixes it before review.</td></tr>
<tr><td data-label="Piece">CI</td><td data-label="What it is">Confirms the same verdict. Does not discover it.</td></tr>
</tbody>
</table>
</div>

<div class="canonical-note">
The authoritative enforcement loop is `docs/_diagrams/ARCH-002-enforcement-loop.mmd`: intent bounds execution, pack engines return a verdict, and evidence feeds provenance back into intent.

<!-- backstop-journey-link: JLINK-014 -->
[Read the gate reference](/reference/#gate)
</div>

## What Backstop does not own {#ownership-boundaries}

<div class="tactics-matrix">
<table>
<tbody>
<tr><td data-label="Piece">Core</td><td data-label="What it is">Runs the process.</td></tr>
<tr><td data-label="Piece">Packs</td><td data-label="What it is">Own the standards.</td></tr>
<tr><td data-label="Piece">Harness</td><td data-label="What it is">Must honor the exit.</td></tr>
<tr><td data-label="Piece">Agent</td><td data-label="What it is">Backstop sits around it. It is not a coding agent.</td></tr>
</tbody>
</table>
</div>

<div class="canonical-note">
<!-- backstop-claim: CLAIM-010 -->
The canonical architecture views are checked-in Mermaid sources so presentation cannot become a second editable truth.
<!-- /backstop-claim -->

Core owns execution and lifecycle primitives. Packs own standards and engines. Harnesses own orchestration. External toolchains own their behavior. The authoritative boundary view is `docs/_diagrams/ARCH-003-ownership-boundaries.mmd`.

<!-- backstop-journey-link: JLINK-011 -->
[Review project boundaries](/status/#project-boundaries)
</div>

<div class="canonical-anchors">

## Intent artifacts {#intent-artifacts}

## Work tracks {#work-tracks}

## Bounded execution {#bounded-execution}

## Recipes {#recipes}

## Gates and policy {#gates-and-policy}

<!-- backstop-claim: CLAIM-011 -->
The gate checks the installed standards against the named inputs and returns a verdict. That is the whole guarantee. Whether anyone stops is outside the process.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-006 -->
[Review support and limits](/status/#supported-and-limited)

## Waivers {#waivers}

## Capabilities and journeys {#capabilities-and-journeys}

## Provenance and verification {#provenance-and-verification}

<!-- backstop-journey-link: JLINK-021 -->
[Trace the sources](/reference/#source-traceability)

## Harness integration {#harness-integration}

<!-- backstop-claim: CLAIM-019 -->
A harness can schedule work and invoke Backstop. It preserves Backstop guarantees only when it also respects artifact order, role boundaries, and the gate exit status.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-008 -->
[Check compatibility details](/reference/#compatibility)

## Product category {#product-category}

<!-- backstop-claim: CLAIM-020 -->
Backstop is not a coding agent. It sits around whichever agent you already use and stops work that is off-task, unreviewable, or not allowed to ship.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-005 -->
[Find adjacent guidance](/status/#adjacent-guidance)

## Delivery lifecycle {#delivery-lifecycle}

<!-- backstop-claim: CLAIM-023 -->
The authoritative lifecycle is `docs/_diagrams/ARCH-001-delivery-lifecycle.mmd`: reactive issues and proactive bundles converge on reviewed plans, bounded implementation, validation, and terminal outcomes.
<!-- /backstop-claim -->

</div>
