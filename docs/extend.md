---
title: Extend Backstop
permalink: /extend/
hero_question: "When should this concern become a pack?"
---

# Extend Backstop

When should this concern become a pack?

## Pack or not {#pack-or-not}

Create a pack when a concern is deterministic, reusable across repositories, independently versionable, and owned by a maintainable standard. Keep repository-specific wiring local when reuse would manufacture an abstraction without a real consumer.

<!-- backstop-journey-link: JLINK-019 -->
[Inspect the pack artifact](/reference/#pack-artifact)

## Author a pack {#author-a-pack}

Scaffold the declaration, define claims, bind each claim to an engine and fixtures, and make negative cases prove the finding. Run the pack against representative repositories, preserve path-filter diagnostics, and publish an immutable release only after its own gate passes.

Rules explain the violation, fixtures prove detection, and tool pins make execution reproducible. Iteration belongs inside the pack rather than in every consuming repository.

<!-- backstop-journey-link: JLINK-020 -->
[Contribute the pack](/contributing/#contribution-paths)

## Path-filter diagnostics {#path-filter-diagnostics}

When an engine receives explicit changed-file arguments, do not assume a slash-bearing include or exclude pattern behaves as it does during directory traversal. Use `pack check` and `pack test` to surface path-scope advisories, preserve production-relative fixture paths, and prefer a slash-free single-segment pattern only when it retains the intended scope.
