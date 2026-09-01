---
title: Adopt Backstop
layout: default
permalink: /adopt/
hero_question: "Try it out."
hero_lede: "Install Backstop and see how the pieces work together."
---

## 1. Set up Backstop {#setup}

### Install the binary {#install}

Pin the released binary in the repository.

<pre><code>GOBIN=./.backstop-bin go install github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0</code></pre>

<div class="canonical-note">
<!-- backstop-journey-link: JLINK-012 -->
[Configure Backstop](/reference/#configuration)
</div>

### Initialize the repository {#configure}

Backstop configuration lives with the code.

<pre><code>backstop init</code></pre>

## 2. Start with your existing code {#adoption-paths}

A pack is a versioned set of standards for a stack. [See which packs exist](/packs/#choose-a-pack) and install one that matches this repository.

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

Assign the approved plan to an implementer agent. The agent executes the plan, runs required gates as it works, and fixes failures before proceeding.

Stop before merge.

<div class="canonical-note">
<!-- backstop-claim: CLAIM-024 -->
A zero exit is the expected postcondition only when every blocking check passes. Keep the failing receipt when it does not.
<!-- /backstop-claim -->

<pre><code>backstop gate</code></pre>

<!-- backstop-journey-link: JLINK-013 -->
[Understand the enforcement loop](/model/#enforcement-loop)
</div>

## You've now used the whole model. {#used-the-model}

Use the complete workflow, or only the parts you need.
