---
title: Product Model
layout: default
permalink: /model/
hero_question: "How does it work?"
hero_lede: "Define the work. Enforce your standards. Detect drift."
---

## Define the work {#operating-model}

Name the work before the agent writes code. New work goes through a bundle, a spec, and a plan. A bounded fix can skip to an issue and a plan. Both tracks stop at a reviewed plan. Nothing is implemented before that.

<div class="guide-list">
<article>
<h3>Bundle</h3>
<p>Captures requirements, open questions, non-goals, and conversation history so you always know how you arrived at the unit of work.</p>
</article>
<article>
<h3>Spec</h3>
<p>Derived from the bundle. It captures the how: contracts, file operations, and functional claims that map back to bundle-level requirements.</p>
</article>
<article>
<h3>Plan</h3>
<p>The YAML that names the exact phases and steps the implementer must follow. The agent does not get to redesign it.</p>
</article>
<article>
<h3>Issue</h3>
<p>A bounded fix does not get a bundle or a spec. The issue becomes a plan.</p>
</article>
</div>

<div class="canonical-note">
<!-- backstop-claim: CLAIM-022 -->
Backstop converts declared intent into bounded work and an inspectable verdict. Proactive work begins with a bundle; reactive work begins with an issue. Both reach a reviewed plan before implementation, and both finish through deterministic validation and explicit terminal state. Terminal state records whether work was delivered, replaced, canceled, deprecated, or obsoleted; issue closure preserves either completed-plan lineage through `delivered_by` or a direct typed artifact, commit, or pull-request reference through `resolved-by`.
<!-- /backstop-claim -->
</div>

<!-- backstop-journey-link: JLINK-010 -->
[Inspect artifact schemas](/reference/#artifact-schema-catalog)

## Enforce your standards {#standards-packs}

Turn the decisions you would fail a merge over into rules the agent cannot skip. Encode them once. The agent runs them while it works. CI runs the same gate later.

<div class="guide-list">
<article>
<h3>Packs</h3>
<p>Versioned standards with engines and fixtures. Core stays a thin executor: it only runs what a pack declares.</p>
</article>
</div>

### Gate {#gates-and-policy}

<!-- backstop-claim: CLAIM-011 -->
The gate checks the installed standards against the named inputs and returns a verdict. That is the whole guarantee. Whether anyone stops is outside the process.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-006 -->
[Review support and limits](/status/#supported-and-limited)

### Recipes {#recipes}

Pinned, repeatable operations — a project, a pipeline, a config — that do not move the standard into Core.

## Detect drift {#baselines-and-ratchets}

A status is a claim about reality. The gate can contradict it.

<div class="guide-list">
<article>
<h3>Baseline</h3>
<p>Existing debt stays visible so you can adopt without pretending the repository is clean. Touched code and new violations fail.</p>
</article>
</div>

### Waivers {#waivers}

A named, time-bounded exception. It permits a specific finding without deleting the rule.

### Provenance {#provenance-and-verification}

A green result has to point at source, command, and tests. Green without that is not a delivery fact.

<!-- backstop-journey-link: JLINK-021 -->
[Trace the sources](/reference/#source-traceability)

## While the agent is working {#enforcement-loop}

The gate runs inside the agent's loop. The agent gets the failure, fixes it, and reruns the gate before the work reaches review. CI should confirm that verdict, not discover it.

<div class="canonical-note">
The authoritative enforcement loop is `docs/_diagrams/ARCH-002-enforcement-loop.mmd`: intent bounds execution, pack engines return a verdict, and evidence feeds provenance back into intent.
</div>

<!-- backstop-journey-link: JLINK-014 -->
[Read the gate reference](/reference/#gate)

## What Backstop does not own {#ownership-boundaries}

Core runs the process. Packs own the standards. The agent, the CI system, or any other harness has to honor the exit. External tools keep their own guarantees.

<div class="canonical-note">
<!-- backstop-claim: CLAIM-010 -->
The canonical architecture views are checked-in Mermaid sources so presentation cannot become a second editable truth.
<!-- /backstop-claim -->

Core owns execution and lifecycle primitives. Packs own standards and engines. Harnesses own orchestration. External toolchains own their behavior. The authoritative boundary view is `docs/_diagrams/ARCH-003-ownership-boundaries.mmd`.
</div>

<!-- backstop-journey-link: JLINK-011 -->
[Review project boundaries](/status/#project-boundaries)

### Harness {#harness-integration}

<!-- backstop-claim: CLAIM-019 -->
A harness can schedule work and invoke Backstop. It preserves Backstop guarantees only when it also respects artifact order, role boundaries, and the gate exit status.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-008 -->
[Check compatibility details](/reference/#compatibility)

### Not a coding agent {#product-category}

<!-- backstop-claim: CLAIM-020 -->
Backstop is not a coding agent. It sits around whichever agent you already use and stops work that is off-task, unreviewable, or not allowed to ship.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-005 -->
[Find adjacent guidance](/status/#adjacent-guidance)

<div class="canonical-anchors">

## Intent artifacts {#intent-artifacts}

## Work tracks {#work-tracks}

## Bounded execution {#bounded-execution}

## Capabilities and journeys {#capabilities-and-journeys}

## Delivery lifecycle {#delivery-lifecycle}

<!-- backstop-claim: CLAIM-023 -->
The authoritative lifecycle is `docs/_diagrams/ARCH-001-delivery-lifecycle.mmd`: reactive issues and proactive bundles converge on reviewed plans, bounded implementation, validation, and terminal outcomes.
<!-- /backstop-claim -->

</div>
