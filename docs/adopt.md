---
title: Adopt Backstop
layout: default
permalink: /adopt/
hero_question: "Put one standard behind the gate."
hero_lede: "Install. Initialize. Prove a violation blocks."
---

## Start with one gate {#adoption-paths}

Don't widen the policy surface until a violation actually blocks.

<div class="adopt-steps">
<div class="adopt-step"><strong>Install</strong><span>Pin the binary</span></div>
<div class="adopt-edge" aria-hidden="true">→</div>
<div class="adopt-step"><strong>Initialize</strong><span>Own the declaration</span></div>
<div class="adopt-edge" aria-hidden="true">→</div>
<div class="adopt-step packs"><strong>Prove</strong><span>A violation must fail</span></div>
</div>

<p class="adopt-need"><span>You need</span>Go, Git, and a standard you would fail a merge over.</p>

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

<div class="adopt-init">
<div class="fig-node"><strong>backstop.yml</strong><span>Repository-owned config</span></div>
<div class="fig-node packs"><strong>Pack</strong><span>One maintained standard</span></div>
<div class="fig-node"><strong>Commit</strong><span>Pin it in version control</span></div>
</div>

Choose a maintained pack and commit the pin.

## Prove it blocks {#verify-enforcement}

A passing gate is only expected when every blocking check passes.

<pre><code>backstop gate</code></pre>

<div class="adopt-verdict" aria-label="Example failing Backstop gate">
<div class="adopt-verdict-bar"><span>first run</span><span>exit 1</span></div>
<div class="adopt-verdict-row"><span>Standard</span><span>the rule you just pinned</span><strong>fail</strong></div>
<div class="adopt-verdict-foot"><strong>FAIL</strong><span>The violation is blocked. That is the first success.</span></div>
</div>

Then you can widen.

<div class="canonical-note">
<!-- backstop-claim: CLAIM-024 -->
A zero exit is the expected postcondition only when every blocking check passes. Keep the failing receipt when it does not.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-013 -->
[Understand the enforcement loop](/model/#enforcement-loop)
</div>
