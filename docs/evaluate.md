---
title: Evaluate Backstop
layout: default
permalink: /evaluate/
hero_question: "Your agent already writes the code."
hero_lede: "Backstop does not care which one. It cares whether the result may ship."
---

<div class="tactics-matrix">
<table>
<thead>
<tr><th>What you already tried</th><th>What that gets you</th><th>What it does not</th><th>What Backstop does instead</th><th>Result</th></tr>
</thead>
<tbody>
<tr><td>Markdown specs</td><td>Named intent</td><td>The agent can skip them</td><td>Plan-before-code, claims, mandated tests</td><td>“Done” can be contradicted</td></tr>
<tr><td>Skills / AGENTS.md</td><td>Better default behavior</td><td>A prompt, not a verdict</td><td>Packs and fixtures for the non-negotiable subset</td><td>Green means the standard held</td></tr>
<tr><td>MCP</td><td>Tool access</td><td>No merge policy</td><td>The same CLI; the caller must honor the exit</td><td>Running the binary is not stopping</td></tr>
<tr><td>LLM review</td><td>Another opinion on the diff</td><td>Still a guess</td><td>Deterministic engines for what must not be a guess</td><td>Same input, same verdict</td></tr>
<tr><td>Standards as docs</td><td>Shared language</td><td>Unenforced</td><td>Encode only what you would fail a merge over</td><td>The rest stays a doc on purpose</td></tr>
</tbody>
</table>
</div>

## What it is {#what-backstop-is}

<!-- backstop-claim: CLAIM-020 -->
Backstop is not a coding agent. It sits around whichever agent you already use and stops work that is off-task, unreviewable, or not allowed to ship.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-002 -->
[See the operating model](/model/#operating-model)

## The actual problem {#failure-fit}

The problem is agents freelancing, generating more code than anyone can reliably review, and shipping it anyway.

## When to use it {#fit-decision}

Use it when you have a standard you would actually fail a merge over. Start with one.

<!-- backstop-journey-link: JLINK-004 -->
[Start installation](/adopt/#install)

## When not to {#not-a-fit}

<!-- backstop-claim: CLAIM-018 -->
If you wanted Cursor, a tracker, or a runtime, this is the wrong tool.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-005 -->
[Find adjacent guidance](/status/#adjacent-guidance)

## What the gate does {#guarantees}

<!-- backstop-claim: CLAIM-011 -->
The gate checks the installed standards against the named inputs and returns a verdict. That is the whole guarantee. Whether anyone stops is outside the process.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-006 -->
[Review support and limits](/status/#supported-and-limited)

## What a harness does not get {#compatibility}

<!-- backstop-claim: CLAIM-006 -->
A harness can run the binary. That only proves the binary runs. Artifact order, review separation, and stopping on red are extra work the harness has to do.
<!-- /backstop-claim -->

<!-- backstop-claim: CLAIM-021 -->
Invocation proves the process started. A trustworthy integration separately proves the harness kept the order and stopped on failure.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-008 -->
[Check compatibility details](/reference/#compatibility)

## Runtime-neutral {#compatibility-limits}

<!-- backstop-claim: CLAIM-012 -->
Runtime-neutral does not mean runtime-guaranteed. Backstop cannot make a harness respect a result it chooses to discard.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-009 -->
[Follow compatibility guidance](/status/#adjacent-guidance)

## Receipts {#evidence}

Green is not evidence. Evidence is a named command, a commit, or a checked-in incident.

<!-- backstop-claim: CLAIM-007 -->
A literal Liquid-shaped `{% raw %}{{ ... }}{% endraw %}` expression is currently interpreted as an undeclared recipe parameter and rejected before any write; ISSUE-182 tracks the missing literal-escape capability.
<!-- /backstop-claim -->

<!-- backstop-claim: CLAIM-008 -->
Linux CI exposed a contracts-pack fixture false negative, and commit `f8b3846fe5d4c2bc6465efc6eb5e4594e1b591da` repaired it with checked-in before-and-after evidence.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-021 -->
[Trace the sources](/reference/#source-traceability)
