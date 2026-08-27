---
title: Adopt Backstop
layout: default
permalink: /adopt/
hero_question: "What does a first working adoption require?"
---

## Adoption paths {#adoption-paths}

Start in a disposable repository, prove one gate, and only then widen the policy surface. A first adoption needs Go, Git, an explicit project root, and a standard worth making non-negotiable.

## Install {#install}

Install the exact released binary into the disposable repository rather than relying on a machine-global copy.

<pre><code>GOBIN=./.backstop-bin go install github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0</code></pre>

<!-- backstop-journey-link: JLINK-012 -->
[Configure Backstop](/reference/#configuration)

## Configure {#configure}

Initialize the repository-owned declaration.

<pre><code>backstop init</code></pre>

Inspect the created `backstop.yml`, select a maintained pack, and keep the pinned declaration in version control.

## Verify enforcement {#verify-enforcement}

Run the gate from the repository root.

<pre><code>backstop gate</code></pre>

<!-- backstop-claim: CLAIM-024 -->
A zero exit is the expected postcondition only when every blocking check passes. Keep the failing receipt when it does not.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-013 -->
[Understand the enforcement loop](/model/#enforcement-loop)
