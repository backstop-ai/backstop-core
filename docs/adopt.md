---
title: Adopt Backstop
layout: default
permalink: /adopt/
hero_question: "Adopt Backstop incrementally"
hero_lede: "You do not need to adopt the entire framework at once. Start with the problem you want Backstop to enforce."
---

## Choose where to start {#adoption-paths}

<div class="tactics-matrix adopt-chooser">
<table>
<thead>
<tr><th>If this is happening</th><th>Start with</th><th>What changes</th></tr>
</thead>
<tbody>
<tr><td data-label="If this is happening">Standards you enforce in review only exist in documentation</td><td data-label="Start with">A pack and the gate</td><td data-label="What changes">Violations can block the work</td></tr>
<tr><td data-label="If this is happening">The agent can finish without satisfying the defined work</td><td data-label="Start with">The artifact chain</td><td data-label="What changes">“Done” can be contradicted</td></tr>
<tr><td data-label="If this is happening">Failures are first discovered after the PR opens</td><td data-label="Start with">The gate in the agent loop</td><td data-label="What changes">The agent gets the failure before review</td></tr>
</tbody>
</table>
</div>

## Install Backstop {#install}

Pin the released binary in the repository.

<pre><code>GOBIN=./.backstop-bin go install github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0</code></pre>

<div class="canonical-note">
<!-- backstop-journey-link: JLINK-012 -->
[Configure Backstop](/reference/#configuration)
</div>

## Initialize Backstop {#configure}

The Backstop configuration lives with the code.

<pre><code>backstop init</code></pre>

## Add what you need {#add-the-piece}

<div class="adopt-fork">
<div class="adopt-fork-row"><strong>Standards</strong><span aria-hidden="true">→</span><span>Install a pack</span></div>
<div class="adopt-fork-row"><strong>Intent / completeness</strong><span aria-hidden="true">→</span><span>Start an artifact chain</span></div>
<div class="adopt-fork-row"><strong>Late failures</strong><span aria-hidden="true">→</span><span>Put the gate in the agent loop</span></div>
</div>

## Run the gate {#verify-enforcement}

Confirm that a known violation fails before relying on it.

<pre><code>backstop gate</code></pre>

<div class="canonical-note">
<!-- backstop-claim: CLAIM-024 -->
A zero exit is the expected postcondition only when every blocking check passes. Keep the failing receipt when it does not.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-013 -->
[Understand the enforcement loop](/model/#enforcement-loop)
</div>
