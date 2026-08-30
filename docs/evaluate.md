---
title: Evaluate Backstop
layout: default
permalink: /evaluate/
hero_question: "Your agent already writes the code."
hero_lede: "Backstop does not care which one. It cares whether the result may ship."
---

<div class="tactics-intro">
<p class="tactics-kicker">Keep the tools. Add a verdict.</p>
<p>Backstop is the enforcement layer for agent-generated work.</p>
</div>

<div class="tactics-matrix">
<table>
<thead>
<tr><th>What you already use</th><th>What that gets you</th><th>What it cannot guarantee</th><th>What Backstop adds</th><th>Result</th></tr>
</thead>
<tbody>
<tr><td data-label="What you already use">Markdown specs</td><td data-label="What that gets you">Named intent</td><td data-label="What it cannot guarantee">The agent can skip them</td><td data-label="What Backstop adds">Plan-before-code, verifiable claims, mandated tests</td><td data-label="Result">“Done” can be contradicted</td></tr>
<tr><td data-label="What you already use">Skills / AGENTS.md</td><td data-label="What that gets you">Better default behavior</td><td data-label="What it cannot guarantee">A prompt, not a verdict</td><td data-label="What Backstop adds">Packs and fixtures for the non-negotiable subset</td><td data-label="Result">Green means the standard held</td></tr>
<tr><td data-label="What you already use">MCP</td><td data-label="What that gets you">Tool access</td><td data-label="What it cannot guarantee">A protocol, not a policy</td><td data-label="What Backstop adds">A required gate whose exit status controls the workflow</td><td data-label="Result">The exit code controls what happens next</td></tr>
<tr><td data-label="What you already use">LLM review</td><td data-label="What that gets you">Another opinion on the diff</td><td data-label="What it cannot guarantee">Still a guess</td><td data-label="What Backstop adds">Deterministic engines for what must not be a guess</td><td data-label="Result">Same input, same verdict</td></tr>
<tr><td data-label="What you already use">Standards as docs</td><td data-label="What that gets you">Shared language</td><td data-label="What it cannot guarantee">Unenforced</td><td data-label="What Backstop adds">Encode only what you would fail a merge over</td><td data-label="Result">The rest stays a doc on purpose</td></tr>
</tbody>
</table>
</div>

## When work is not allowed to ship {#failure-fit}

<div class="failed-verdict" aria-label="Example failing Backstop gate">
<div class="failed-verdict-bar"><span>backstop gate</span><span>exit 1</span></div>
<div class="failed-verdict-row"><span>Tests</span><span>promised test is absent</span><strong>fail</strong></div>
<div class="failed-verdict-row"><span>Requirements</span><span>completion claim without coverage</span><strong>fail</strong></div>
<div class="failed-verdict-foot"><strong>FAIL</strong><span>The work is not allowed to ship.</span></div>
</div>

## CI is too late to find out {#fit-decision}

Fix problems while your agent is still writing the code.

Backstop puts deterministic gates inside the agent's working loop. The agent gets the failure, fixes the issue, and reruns the gate before the work reaches review. CI runs the same checks again—but confirms the verdict instead of delivering the surprise.

<div class="ci-workflows">
<article>
<p>Typical workflow</p>
<ol>
<li>Agent writes</li>
<li>PR opens</li>
<li>CI fails</li>
<li>Human or agent reconstructs context</li>
<li>Retry</li>
</ol>
</article>
<article>
<p>Backstop workflow</p>
<ol class="has-loop">
<li>Agent writes</li>
<li>Gate fails</li>
<li>Agent fixes</li>
<li>PR opens</li>
<li>CI confirms</li>
</ol>
</article>
</div>

CI should confirm, not discover.

<!-- backstop-journey-link: JLINK-004 -->
[Start installation](/adopt/#install)

## What it is {#what-backstop-is}

<!-- backstop-claim: CLAIM-020 -->
Backstop is not a coding agent. It sits around whichever agent you already use and stops work that is off-task, unreviewable, or not allowed to ship.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-002 -->
[See the operating model](/model/#operating-model)

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
