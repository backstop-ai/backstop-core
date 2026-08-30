---
title: Evaluate Backstop
layout: default
permalink: /evaluate/
hero_question: "Is Backstop the right control surface for this problem?"
hero_lede: "Use it when an agent has to wait, name the work, and accept a red gate."
---

## What it is {#what-backstop-is}

<!-- backstop-claim: CLAIM-020 -->
Backstop is the discipline around the agent, not another agent. Artifacts name the work. Packs name the standards. The gate says whether the result may ship.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-002 -->
[See the operating model](/model/#operating-model)

## When it fits {#failure-fit}

Backstop fits when the problem is confidence, repeatability, drift, or evidence — not how fast code appears. It does not fit when there is no standard to encode and no place a verdict has to stick.

## How to decide {#fit-decision}

Adopt it when a repository can hold intent artifacts, keep author and reviewer apart, and fail closed. Start with one standard and one workflow. Understanding the problem is not permission to implement.

<!-- backstop-journey-link: JLINK-004 -->
[Start installation](/adopt/#install)

## When it doesn't {#not-a-fit}

<!-- backstop-claim: CLAIM-018 -->
Backstop is not a runtime, not a tracker, and not a substitute for owning the call. Those systems can sit around it. They remain someone else's job.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-005 -->
[Find adjacent guidance](/status/#adjacent-guidance)

## What the gate guarantees {#guarantees}

<!-- backstop-claim: CLAIM-011 -->
The gate checks the installed standards against the named inputs and returns a verdict. That is the whole guarantee. Whether anyone stops is outside the process.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-006 -->
[Review support and limits](/status/#supported-and-limited)

## What a harness gets {#compatibility}

<!-- backstop-claim: CLAIM-006 -->
A harness can run the binary. That only proves the binary runs. Artifact order, review separation, and stopping on red are extra work the harness has to do.
<!-- /backstop-claim -->

<!-- backstop-claim: CLAIM-021 -->
Invocation proves the process started. A trustworthy integration separately proves the harness kept the order and stopped on failure.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-008 -->
[Check compatibility details](/reference/#compatibility)

## What runtime-neutral does not mean {#compatibility-limits}

<!-- backstop-claim: CLAIM-012 -->
Runtime-neutral does not mean runtime-guaranteed. Backstop cannot make a harness respect a result it chooses to discard.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-009 -->
[Follow compatibility guidance](/status/#adjacent-guidance)

## What counts as evidence {#evidence}

Green is not evidence. Evidence is a named command, a commit, or a checked-in incident.

<!-- backstop-claim: CLAIM-007 -->
A literal Liquid-shaped `{% raw %}{{ ... }}{% endraw %}` expression is currently interpreted as an undeclared recipe parameter and rejected before any write; ISSUE-182 tracks the missing literal-escape capability.
<!-- /backstop-claim -->

<!-- backstop-claim: CLAIM-008 -->
Linux CI exposed a contracts-pack fixture false negative, and commit `f8b3846fe5d4c2bc6465efc6eb5e4594e1b591da` repaired it with checked-in before-and-after evidence.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-021 -->
[Trace the sources](/reference/#source-traceability)
