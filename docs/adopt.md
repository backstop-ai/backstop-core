---
title: Adopt Backstop
layout: default
permalink: /adopt/
hero_question: "Try it out."
hero_lede: "Install Backstop and see how it works together."
---

## 1. Install Backstop {#install}

Pin the released binary in the repository.

<pre><code>GOBIN=./.backstop-bin go install github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0</code></pre>

<div class="canonical-note">
<!-- backstop-journey-link: JLINK-012 -->
[Configure Backstop](/reference/#configuration)
</div>

### Initialize Backstop {#configure}

The Backstop configuration lives with the code.

<pre><code>backstop init</code></pre>

## 2. Start with your existing code {#adoption-paths}

Install a pack appropriate for the repository's stack. [Choose a pack](/packs/#choose-a-pack).

Run:

<pre><code>backstop baseline</code></pre>

Review what Backstop finds. Do not fix anything yet.

## 3. Try it on a bug you know {#known-bug}

Pick a known bug in the repository.

Use Backstop's lightweight artifact workflow for it:

<ul class="eval-list">
<li>Create an issue.</li>
<li>Create the corresponding plan.</li>
<li>Run the artifact reviewers and deterministic validators.</li>
<li>Resolve findings until the artifacts pass.</li>
</ul>

[Artifact lifecycle](/reference/#artifact-lifecycle-and-closure)

## 4. Let the agent implement it {#verify-enforcement}

Assign the approved plan to an implementer agent.

Execute the plan. As implementation progresses, run:

<pre><code>backstop gate</code></pre>

Fix gate failures before proceeding. Complete implementation review and resolve review findings.

Stop before merge.

<p class="adopt-close">You've now used the whole model.</p>

<p class="adopt-close-next">Use the complete workflow, or only the parts you need.</p>

<div class="canonical-note">
<!-- backstop-claim: CLAIM-024 -->
A zero exit is the expected postcondition only when every blocking check passes. Keep the failing receipt when it does not.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-013 -->
[Understand the enforcement loop](/model/#enforcement-loop)
</div>
