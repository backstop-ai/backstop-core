---
title: Adopt Backstop
layout: default
permalink: /adopt/
hero_question: "Try it out."
hero_lede: "Install Backstop and see how the pieces work together."
---

## 1. Set up Backstop {#setup}

### Install the binary {#install}

Install a released build with Homebrew or Go. Or clone the repository and build it.

These commands cover macOS and Linux on amd64 and arm64. Windows is planned.

Homebrew:

<pre><code>brew install backstop-ai/tap/backstop</code></pre>

Go, pinned to a module version in the repository:

<pre><code>GOBIN=./.backstop-bin go install github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0</code></pre>

From a clone:

<pre><code>git clone https://github.com/backstop-ai/backstop-core.git
cd backstop-core
go build -o ./bin/backstop ./cmd/backstop</code></pre>

### Initialize the repository {#configure}

Backstop configuration lives with the code. Init writes the repository files later clones restore.

<pre><code>backstop init</code></pre>

## 2. Start with your existing code {#adoption-paths}

Install a pack appropriate for the repository's stack. [Choose a pack](/pack/examples/#choose-a-pack).

Run:

<pre><code>backstop baseline</code></pre>

Review what Backstop finds. Do not fix anything yet.

## 3. Try it on a bug you know {#known-bug}

Pick a known bug in the repository. You already understand it, so you can judge whether the issue and plan capture the work.

Bounded work uses issue → plan. This is the normal path for a bug, a follow-on, or a small fix.

<ul class="eval-list">
<li>Create an issue.</li>
<li>Create a plan for the issue.</li>
<li>Run the artifact reviewers and deterministic validators.</li>
<li>Resolve findings until the artifacts pass.</li>
</ul>

## 4. Let the agent implement it {#verify-enforcement}

Assign the approved plan to an implementer agent. The agent executes the plan, runs required gates as it works, and fixes failures before proceeding.

Stop before merge.

<div class="canonical-note">
<!-- backstop-claim: CLAIM-024 -->
A zero exit is the expected postcondition only when every blocking check passes. Keep the failing receipt when it does not.
<!-- /backstop-claim -->

<pre><code>backstop gate</code></pre>
</div>

<!-- backstop-journey-link: JLINK-013 -->
[Understand the enforcement loop](/model/#enforcement-loop)

## You've now used the whole model. {#used-the-model}

Use the complete workflow, or only the parts you need.
