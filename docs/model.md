---
title: Product Model
layout: default
permalink: /model/
hero_question: "How does it work?"
hero_lede: "Define the work. Enforce your standards. Detect drift."
---

## Define the work {#operating-model}

One bundle. Many specs. One plan per spec.

<div class="work-topology">
<div class="work-view decomp">
<p>How work is structured</p>
<div class="decomp-body">
<div class="topo-bundle">Bundle</div>
<div class="decomp-fan" aria-hidden="true"></div>
<div class="decomp-set">
<div class="work-unit"><span class="topo-node spec">Spec A</span><span class="topo-node plan">Plan A</span></div>
<div class="work-unit"><span class="topo-node spec">Spec B</span><span class="topo-node plan">Plan B</span></div>
<div class="work-unit"><span class="topo-node spec">Spec C</span><span class="topo-node plan">Plan C</span></div>
<div class="work-unit"><span class="topo-node spec">Spec D</span><span class="topo-node plan">Plan D</span></div>
</div>
</div>
</div>
<div class="work-split" aria-hidden="true"></div>
<div class="work-view dag">
<p>How work is delivered</p>
<div class="dag-flow">
<div class="dag-merge">
<div class="dag-stack">
<div class="work-unit"><span class="topo-node spec">Spec A</span><span class="topo-node plan">Plan A</span></div>
<div class="work-unit"><span class="topo-node spec">Spec B</span><span class="topo-node plan">Plan B</span></div>
</div>
<div class="dag-brace" aria-hidden="true"><svg viewBox="0 0 24 100" preserveAspectRatio="none"><path d="M5 6 C20 8 15 30 15 44 C15 48 22 49 22 50 C22 51 15 52 15 56 C15 70 20 92 5 94" fill="none" stroke="currentColor" stroke-width="5" stroke-linecap="round" stroke-linejoin="round"/></svg></div>
</div>
<div class="work-unit"><span class="topo-node spec">Spec C</span><span class="topo-node plan">Plan C</span></div>
<div class="dag-edge" title="D depends on C">→</div>
<div class="work-unit"><span class="topo-node spec">Spec D</span><span class="topo-node plan">Plan D</span></div>
</div>
</div>
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

Packs own standards, engines, and proof.

<div class="model-figure pack-figure">
<div class="pack-body">
<div class="topo-bundle">Pack</div>
<div class="pack-fan" aria-hidden="true"></div>
<div class="pack-row">
<div class="pack-runtime">
<div class="pack-pair">
<div class="fig-node packs"><strong>Rules</strong><span>What must be true</span></div>
<div class="fig-node packs"><strong>Engines</strong><span>How it is checked</span></div>
</div>
<div class="pack-collect" aria-hidden="true"></div>
<div class="fig-node gate"><strong>Gate</strong><span>Runs the pack's required checks</span></div>
</div>
<div class="fig-node packs pack-proof"><strong>Fixtures</strong><span>Prove the check works</span></div>
</div>
</div>
</div>

Packs compose across domains and toolchains.

<div class="model-figure pack-compose">
<div class="compose-set">
<div class="fig-node packs"><strong>Code</strong><span>Go</span></div>
<div class="fig-node packs"><strong>Architecture</strong><span>Package bounds</span></div>
<div class="fig-node packs"><strong>Design</strong><span>CSS and tokens</span></div>
</div>
<div class="compose-collect" aria-hidden="true"></div>
<div class="fig-node gate"><strong>Gate</strong><span>Executes all rules across all packs</span></div>
</div>

## Detect drift {#baselines-and-ratchets}

Existing debt is not permission for new debt.

<div class="model-figure ratchet-figure">
<div class="ratchet">
<div class="fig-node"><strong>Unchanged</strong><span>Visible. Does not fail.</span></div>
<div class="fig-node packs"><strong>New or modified</strong><span>Must meet the standard.</span></div>
</div>
<p class="fig-caption">A waiver is named and time-bounded. The rule stays.</p>
</div>

A claim names what must exist and the tests that prove it.

<div class="model-figure claim-figure">
<div class="claim-pair">
<div class="fig-node packs"><strong>Claim</strong><span>What must exist</span></div>
<div class="claim-cover" aria-hidden="true"><span>mandates</span></div>
<div class="fig-node"><strong>Tests</strong><span>Must exist and be real</span></div>
</div>
</div>

## While the agent is working {#enforcement-loop}

The agent does not hand failed work downstream.

<div class="model-figure loop-figure">
<div class="loop-flow">
<div class="fig-node"><strong>Plan</strong></div>
<div class="dag-edge" aria-hidden="true">→</div>
<div class="loop-core">
<span class="loop-repeat">× N</span>
<div class="loop-forward">
<div class="fig-node packs"><strong>Implement</strong></div>
<div class="dag-edge" aria-hidden="true">→</div>
<div class="fig-node packs"><strong>Gate</strong></div>
</div>
<div class="loop-return" aria-hidden="true"><span>fail / fix</span></div>
</div>
<div class="loop-pass" aria-hidden="true"><span>pass</span></div>
<div class="loop-review">
<div class="fig-node"><strong>Implementation review</strong></div>
<div class="loop-return" aria-hidden="true"><span>fix</span></div>
</div>
<div class="dag-edge" aria-hidden="true">→</div>
<div class="fig-node"><strong>Commit / Push</strong></div>
<div class="dag-edge" aria-hidden="true">→</div>
<div class="loop-ci">
<div class="fig-node ci"><strong>CI</strong></div>
<p class="fig-caption">CI confirms. It does not discover.</p>
</div>
</div>
</div>

<div class="canonical-note">
The authoritative enforcement loop is `docs/_diagrams/ARCH-002-enforcement-loop.mmd`: intent bounds execution, pack engines return a verdict, and evidence feeds provenance back into intent.

<!-- backstop-journey-link: JLINK-014 -->
[Read the gate reference](/reference/#gate)
</div>

## What Backstop does not own {#ownership-boundaries}

<div class="model-figure own-figure">
<div class="own-in">
<p>Backstop</p>
<div class="own-row">
<div class="fig-node"><strong>Core</strong><span>Runs the process</span></div>
<div class="fig-node packs"><strong>Packs</strong><span>Standards, engines, fixtures</span></div>
<div class="fig-node"><strong>Harness</strong><span>Honors the exit</span></div>
</div>
</div>
<div class="work-split" aria-hidden="true"></div>
<div class="own-out">
<p>Not Backstop</p>
<div class="fig-node agent"><strong>Agent</strong><span>Outside. Backstop sits around it.</span></div>
</div>
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
