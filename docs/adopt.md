---
title: Adopt Backstop
layout: default
permalink: /adopt/
hero_question: "Try it out."
hero_lede: "Install Backstop and see how it works together."
---

## Evaluate Backstop in a real project {#adoption-paths}

<ol class="agent-brief">
<li>Install a pack that matches the stack.</li>
<li>Run <code>backstop baseline</code> and summarize what it finds. Do not modify the code yet.</li>
<li>Pick a known bug supplied by the user.</li>
<li>Create an issue and plan for that bug.</li>
<li>Run the artifact reviewers and validators and fix the artifacts until they pass.</li>
<li>Assign the approved plan to an implementer.</li>
<li>Implement the plan, running required gates as work progresses and fixing failures before moving on.</li>
<li>Run implementation review and resolve any findings.</li>
<li>Stop before merge and summarize:
<ul>
<li>what the baseline found</li>
<li>what the artifact workflow changed about the task</li>
<li>what gates failed during implementation</li>
<li>what the agent fixed because of Backstop</li>
</ul>
</li>
</ol>

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

## Run the gate {#verify-enforcement}

Confirm that a known violation fails before relying on it.

<pre><code>backstop gate</code></pre>

<p class="adopt-close">You've now used the whole model.</p>

<p class="adopt-close-next">Keep the complete workflow, or adopt only the parts you need.</p>

<div class="canonical-note">
<!-- backstop-claim: CLAIM-024 -->
A zero exit is the expected postcondition only when every blocking check passes. Keep the failing receipt when it does not.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-013 -->
[Understand the enforcement loop](/model/#enforcement-loop)
</div>
