---
title: Evaluate Backstop
layout: default
permalink: /evaluate/
hero_question: "Is Backstop the right control surface for this problem?"
---

## What Backstop is {#what-backstop-is}

<!-- backstop-claim: CLAIM-020 -->
Backstop is a deterministic control surface for AI-native delivery. It keeps intent, bounded implementation, review, and enforcement legible without requiring one agent vendor or runtime.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-002 -->
[See the operating model](/model/#operating-model)

## Failure fit {#failure-fit}

Backstop fits when fast implementation is no longer the constraint, but confidence, repeatability, policy drift, or evidence is. It is less useful when the work has no stable acceptance boundary to encode.

## Fit decision {#fit-decision}

Adopt it when a repository needs explicit intent artifacts, separate author and reviewer roles, and blocking deterministic checks. Start with one important standard and one workflow.

<!-- backstop-journey-link: JLINK-004 -->
[Start installation](/adopt/#install)

## Not a fit {#not-a-fit}

<!-- backstop-claim: CLAIM-018 -->
Backstop is not an agent runtime, project-management system, or substitute for owning delivery decisions. Those systems can surround it, but they remain separate responsibilities.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-005 -->
[Find adjacent guidance](/status/#adjacent-guidance)

## Guarantees {#guarantees}

<!-- backstop-claim: CLAIM-011 -->
The enforceable guarantee is bounded: the declared gate evaluates the installed standards against the named inputs and returns a policy verdict. Organizational follow-through remains outside that process boundary.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-006 -->
[Review support and limits](/status/#supported-and-limited)

## Compatibility {#compatibility}

<!-- backstop-claim: CLAIM-006 -->
A harness can invoke Backstop, but operability alone does not preserve artifact ordering, review separation, or enforcement of a blocking verdict.
<!-- /backstop-claim -->

<!-- backstop-claim: CLAIM-021 -->
Invocation proves that the binary runs. A trustworthy integration separately proves that the harness preserves lifecycle order and stops on failure.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-008 -->
[Check compatibility details](/reference/#compatibility)

## Compatibility limits {#compatibility-limits}

<!-- backstop-claim: CLAIM-012 -->
Runtime-neutral does not mean runtime-guaranteed. Backstop cannot make a harness respect a result it chooses to discard.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-009 -->
[Follow compatibility guidance](/status/#adjacent-guidance)

## Evidence {#evidence}

<!-- backstop-claim: CLAIM-007 -->
A literal Liquid-shaped `{% raw %}{{ ... }}{% endraw %}` expression is currently interpreted as an undeclared recipe parameter and rejected before any write; ISSUE-182 tracks the missing literal-escape capability.
<!-- /backstop-claim -->

<!-- backstop-claim: CLAIM-008 -->
Linux CI exposed a contracts-pack fixture false negative, and commit `f8b3846fe5d4c2bc6465efc6eb5e4594e1b591da` repaired it with checked-in before-and-after evidence.
<!-- /backstop-claim -->

Public claims name their mechanism and the execution, incident, example, or measurement appropriate to the claim type.

<!-- backstop-journey-link: JLINK-021 -->
[Trace the sources](/reference/#source-traceability)
