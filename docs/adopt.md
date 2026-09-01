---
title: Adopt Backstop
layout: default
permalink: /adopt/
hero_question: "You don't have to take all of it."
hero_lede: "Find the first place in your project that should actually block."
---

## Start from what's already true {#adoption-paths}

Backstop has a lot of surface. The pieces compose. None of them is a prerequisite for the others.

<div class="tactics-matrix adopt-chooser">
<table>
<thead>
<tr><th>If this is already true</th><th>Start here</th><th>You can skip</th></tr>
</thead>
<tbody>
<tr><td data-label="If this is already true">A standard you would fail a merge over lives in a doc</td><td data-label="Start here">One pack and the gate</td><td data-label="You can skip">The artifact chain</td></tr>
<tr><td data-label="If this is already true">The agent can say “done” without matching intent</td><td data-label="Start here">The artifact chain</td><td data-label="You can skip">Packs</td></tr>
<tr><td data-label="If this is already true">Failures show up after the PR opens</td><td data-label="Start here">The gate in the agent's loop</td><td data-label="You can skip">More rules, more artifacts</td></tr>
</tbody>
</table>
</div>

Add the next piece only when this one is actually blocking.

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
