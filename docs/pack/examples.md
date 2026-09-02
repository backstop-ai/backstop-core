---
title: Pack examples
layout: default
permalink: /pack/examples/
hero_question: "Published packs."
hero_lede: false
---

## Choose a pack {#choose-a-pack}

Choose the narrowest maintained pack that owns the standard and supports the repository's tools. Check its release, engine requirements, fixture coverage, and maintenance state before composing it with other packs.

<!-- backstop-journey-link: JLINK-018 -->
[Review pack direction](/status/#pack-direction)

## Published pack catalog {#published-pack-catalog}

<section data-generated-region data-product-truth-job="published-pack-catalog">
<!-- PRODUCT-TRUTH-INCLUDE:BEGIN job=published-pack-catalog -->
{% include generated/published-pack-catalog.md %}
<!-- PRODUCT-TRUTH-INCLUDE:END job=published-pack-catalog -->
</section>

<!-- backstop-journey-link: JLINK-022 -->
[Browse the CLI catalog](/reference/#cli-command-catalog)

## Install a published pack {#install-a-pack}

Add the pack and version from the catalog:

<pre><code>backstop pack add backstop-ai/&lt;pack&gt;@&lt;version&gt;</code></pre>

That command clones the tagged release, validates it, installs it, merges configuration, pins the version in `backstop.yml`, and writes `backstop.lock`.

On a later clone, restore from the lock:

<pre><code>backstop pack install</code></pre>

Confirm the installed set:

<pre><code>backstop pack list</code></pre>

Commit `backstop.yml` and `backstop.lock` with the change. Git packs later move with `backstop pack update`. Local packs refresh the lock with `backstop pack relock`. Command details live in Reference.

<!-- backstop-claim: CLAIM-025 -->
Add a published pack with `backstop pack add org/pack@version`. That pins the version in `backstop.yml` and writes `backstop.lock`. Restore locked packs with `backstop pack install`. Confirm them with `backstop pack list`.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-017 -->
[Use pack commands](/reference/#pack-commands)
