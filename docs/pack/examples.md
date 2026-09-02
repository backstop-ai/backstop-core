---
title: Pack examples
layout: default
permalink: /pack/examples/
hero_question: "Published packs."
---

## Choose a pack {#choose-a-pack}

Choose the narrowest maintained pack that owns the standard and supports the repository's tools. Check its release, engine requirements, fixture coverage, and maintenance state before composing it with other packs.

<!-- backstop-journey-link: JLINK-018 -->
[Review pack direction](/status/#pack-direction)

## Installed pack catalog {#installed-pack-catalog}

<section data-generated-region data-product-truth-job="installed-pack-catalog">
<!-- PRODUCT-TRUTH-INCLUDE:BEGIN job=installed-pack-catalog -->
{% include generated/installed-pack-catalog.md %}
<!-- PRODUCT-TRUTH-INCLUDE:END job=installed-pack-catalog -->
</section>

<!-- backstop-claim: CLAIM-025 -->
The installed catalog is repository truth: declarations pin pack versions and the lock binds resolved bytes. Use the CLI to inspect, install, update, relock, and verify those selections.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-017 -->
[Use pack commands](/reference/#pack-commands)

<!-- backstop-journey-link: JLINK-022 -->
[Browse the CLI catalog](/reference/#cli-command-catalog)
