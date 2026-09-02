---
title: Extend Backstop
layout: default
permalink: /extend/
hero_question: "When should this concern become a pack?"
hero_lede: "If the standard is reusable, author a pack. If it is not, keep it in the repository."
---

## 1. Pack or not {#pack-or-not}

Create a pack when a deterministic standard should be reused across repositories and versioned independently. If it only applies to this repository, keep it here.

Do not create a pack that has no second consumer.

[What is a pack?](/pack/) is the noun. [Choose a pack](/packs/#choose-a-pack) if a maintained pack already owns the standard.

<!-- backstop-journey-link: JLINK-019 -->
[Inspect the pack artifact](/reference/#pack-artifact)

## 2. Author a pack {#author-a-pack}

Author the pack in its own repository. Do not vendor it into core. Exact manifest fields live in the pack artifact reference.

### Scaffold the pack

<pre><code>backstop pack new --type engine --language go --slug my-standard</code></pre>

`--type` is `engine`, `mechanism`, or `toolchain`. The scaffold writes a valid `pack.yml` and a sample rule that can pass check, test, and the gate.

### Define claims, engines, and fixtures

A claim names what must be true. An engine checks it. Fixtures prove the check works, including negative cases that must fail.

Rules explain the violation. Tool pins make execution reproducible. Iteration belongs inside the pack rather than in every consuming repository.

### Check the pack

<pre><code>backstop pack check ./my-standard</code></pre>

### Test the pack

<pre><code>backstop pack test ./my-standard</code></pre>

### Try it in a repository

Install the local pack in a consumer repository and run that repository's gate.

<pre><code>backstop pack add ./my-standard</code></pre>

Publish from the pack repository after check and test pass. Then contribute it.

<!-- backstop-journey-link: JLINK-020 -->
[Contribute the pack](/contributing/#contribution-paths)
