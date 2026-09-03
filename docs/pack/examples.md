---
title: Pack examples
layout: default
permalink: /pack/examples/
hero_question: "Published packs."
hero_lede: false
---

## Choose a pack {#choose-a-pack}

Choose the narrowest maintained pack that owns the standard and supports the repository's tools. Check its release, engine requirements, fixture coverage, and maintenance state before composing it with other packs.

## Published pack catalog {#published-pack-catalog}

<section data-generated-region data-product-truth-job="published-pack-catalog">
<!-- PRODUCT-TRUTH-INCLUDE:BEGIN job=published-pack-catalog -->
{% include generated/published-pack-catalog.md %}
<!-- PRODUCT-TRUTH-INCLUDE:END job=published-pack-catalog -->
</section>

## Install a published pack {#install-a-pack}

Add the pack and version from the catalog:

<pre><code>backstop pack add backstop-ai/&lt;pack&gt;@&lt;version&gt;</code></pre>

That command clones the tagged release, validates it, installs it, merges configuration, pins the version in `backstop.yml`, and writes `backstop.lock`.

On a later clone, restore from the lock:

<pre><code>backstop pack install</code></pre>

Confirm the installed set:

<pre><code>backstop pack list</code></pre>

Commit `backstop.yml` and `backstop.lock` with the change. Git packs later move with `backstop pack update`.

<!-- backstop-claim: CLAIM-025 -->
Add a published pack with `backstop pack add org/pack@version`. That pins the version in `backstop.yml` and writes `backstop.lock`. Restore locked packs with `backstop pack install`. Confirm them with `backstop pack list`.
<!-- /backstop-claim -->

## Try a local pack first {#install-a-local-pack}

Build the pack in its own directory and install that path into the project you want to prove against. Do this before you publish a tagged release.

Scaffold, check, and test in the pack directory:

<pre><code>backstop pack new --type engine --language go --slug my-standard</code></pre>

<pre><code>backstop pack check ./my-standard</code></pre>

<pre><code>backstop pack test ./my-standard</code></pre>

`--type` is `engine`, `mechanism`, or `toolchain`. [What those types mean, and when to choose](/pack/guide/#choose-a-pack-type). Authoring details live in the [guide](/pack/guide/#author-a-pack).

From the consumer project, add the local path:

<pre><code>backstop pack add ./my-standard</code></pre>

That command validates the directory, copies it into `.backstop/packs/`, pins it as `local` in `backstop.yml`, and writes `backstop.lock`. There is no git tag yet.

Run the project's gate:

<pre><code>backstop gate</code></pre>

After you edit the installed pack, refresh the lock:

<pre><code>backstop pack relock ./my-standard</code></pre>

Relock is local-source only. When the pack is ready to ship, [publish a tagged release](/pack/guide/#publish-a-pack) from its repository and add it with `backstop pack add org/pack@version`.
