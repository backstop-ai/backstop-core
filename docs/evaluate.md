---
title: Evaluate Backstop
layout: default
permalink: /evaluate/
hero_question: "Your agent already writes the code."
hero_lede: "Backstop helps you ship confidently."
---

<div class="tactics-intro">
<p class="tactics-kicker">Bigger models write great code. None of them write code like you.</p>
<p class="tactics-bridge">Backstop enforces your standards so the agent's code looks like your code.</p>
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

## A spec is context. The artifact chain is working state. {#working-state}

SDD gives the agent better instructions. The artifact chain gives it less to guess.

A spec tells the agent what you want. The artifact chain tells it what to do next, what it may assume, what it must prove, and when it is done.

Most spec-driven workflows hand the agent one or more Markdown documents and leave it to infer the rest. Backstop turns intent into bounded, linked work: requirements become a plan, the plan produces implementation claims, and those claims require evidence.

At every stage, the agent knows what is authoritative, what remains unresolved, and what must be true before the work can proceed.

<div class="tactics-matrix sdd-matrix">
<table>
<thead>
<tr><th>Common SDD</th><th>What the agent still has to infer</th><th>What the artifact chain makes explicit</th><th>Result</th></tr>
</thead>
<tbody>
<tr><td data-label="Common SDD">A single spec.md</td><td data-label="What the agent still has to infer">Decomposition, dependencies, and boundaries</td><td data-label="What the artifact chain makes explicit">Atomic requirements and explicit relationships</td><td data-label="Result">A smaller problem to solve</td></tr>
<tr><td data-label="Common SDD">Generated plan and tasks</td><td data-label="What the agent still has to infer">Whether every requirement survived planning</td><td data-label="What the artifact chain makes explicit">Validated coverage between artifacts</td><td data-label="Result">Nothing quietly falls out</td></tr>
<tr><td data-label="Common SDD">Checked task boxes</td><td data-label="What the agent still has to infer">What was actually implemented and proven</td><td data-label="What the artifact chain makes explicit">Claims tied to evidence and terminal states</td><td data-label="Result">“Done” can be contradicted</td></tr>
<tr><td data-label="Common SDD">A revised specification</td><td data-label="What the agent still has to infer">Which existing work is now stale</td><td data-label="What the artifact chain makes explicit">Required downstream reconciliation</td><td data-label="Result">Change does not become drift</td></tr>
<tr><td data-label="Common SDD">A new agent session</td><td data-label="What the agent still has to infer">Decisions, progress, and unresolved work</td><td data-label="What the artifact chain makes explicit">Durable artifact and workflow state</td><td data-label="Result">Resume without reconstruction</td></tr>
</tbody>
</table>
</div>

The same structure that makes the agent easier to trust also makes the agent better at the job.

<!-- backstop-journey-link: JLINK-002 -->
[See the operating model](/model/#operating-model)

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
