---
title: Adopt Backstop
layout: default
permalink: /adopt/
hero_question: "Start where it will pay you first."
hero_lede: "The pieces compose. Put the first one on the failure that already costs you."
---

## Start from what's already true {#adoption-paths}

<div class="tactics-matrix adopt-chooser">
<table>
<thead>
<tr><th>If this is already true</th><th>Start here</th><th>What you get</th></tr>
</thead>
<tbody>
<tr><td data-label="If this is already true">A standard you would fail a merge over lives in a doc</td><td data-label="Start here">One pack and the gate</td><td data-label="What you get">The next merge cannot quietly ignore it</td></tr>
<tr><td data-label="If this is already true">The agent can say “done” without matching intent</td><td data-label="Start here">The artifact chain</td><td data-label="What you get">“Done” can be contradicted</td></tr>
<tr><td data-label="If this is already true">Failures show up after the PR opens</td><td data-label="Start here">The gate in the agent's loop</td><td data-label="What you get">The agent sees the failure while it can still fix it</td></tr>
</tbody>
</table>
</div>

<p class="adopt-skip">The rest can wait. None of the pieces is a prerequisite for the others.</p>

When you know the piece, make it real in the repository.

## Install the binary {#install}

Pin the released binary in the repository.

<pre><code>GOBIN=./.backstop-bin go install github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0</code></pre>

Don't rely on a machine-global copy.

<div class="canonical-note">
<!-- backstop-journey-link: JLINK-012 -->
[Configure Backstop](/reference/#configuration)
</div>

## Initialize the repository {#configure}

The declaration lives with the code.

<pre><code>backstop init</code></pre>

Commit only the piece you chose.

## Prove it blocks {#verify-enforcement}

A passing gate is only expected when every blocking check passes.

<pre><code>backstop gate</code></pre>

Keep the failing receipt. That is the first success.

<div class="canonical-note">
<!-- backstop-claim: CLAIM-024 -->
A zero exit is the expected postcondition only when every blocking check passes. Keep the failing receipt when it does not.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-013 -->
[Understand the enforcement loop](/model/#enforcement-loop)
</div>
