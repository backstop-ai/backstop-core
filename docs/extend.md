---
title: Extend Backstop
layout: default
permalink: /extend/
hero_question: "When should this concern become a pack?"
hero_lede: "If the standard is reusable, author a pack. If it is not, keep it in the repository."
---

## 1. Pack or not {#pack-or-not}

Create a pack when a concern is deterministic, reusable across repositories, independently versionable, and owned by a maintainable standard. Keep repository-specific wiring local when reuse would manufacture an abstraction without a real consumer.

If the standard is specific to this repository, keep the wiring here. Do not manufacture a pack that has no second consumer.

If it should be a pack, author it next. [What is a pack?](/pack/) is the noun. [Choose a pack](/packs/#choose-a-pack) if a maintained pack already owns the standard.

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

Install the local pack into a consumer repository, then run that repository's gate. This proves the pack is consumed. It is not the Adopt walkthrough.

<pre><code>backstop pack add ./my-standard</code></pre>

Publish from the pack repository after check and test pass. Then contribute it.

<!-- backstop-journey-link: JLINK-020 -->
[Contribute the pack](/contributing/#contribution-paths)

## 3. Path-filter diagnostics {#path-filter-diagnostics}

When an engine receives explicit changed-file arguments, do not assume a slash-bearing include or exclude pattern behaves as it does during directory traversal. Use `pack check` and `pack test` to surface path-scope advisories, preserve production-relative fixture paths, and prefer a slash-free single-segment pattern only when it retains the intended scope.
