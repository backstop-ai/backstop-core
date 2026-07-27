---
title: "Published Pack Repos Have No CI — A Bad Tag Ships Silently"
schema_version: issue/v1

issue:
  id: ISSUE-084
  title: "Published Pack Repos Have No CI — A Bad Tag Ships Silently"
  type: enhancement
  status: open
  created: "2026-07-26"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate
---

# Published Pack Repos Have No CI — A Bad Tag Ships Silently

## Problem

As of 2026-07-26, ten packs are published, distributed artifacts under `github.com/backstop-ai`
with real remote consumers (`cobra-cli-standards`, `go-standards`, `go-toolchain`,
`bun-toolchain`, `backstop-self`, `typescript-toolchain`, `typescript-standards`,
`typescript-contracts`, `typescript-substantiveness`, `secrets` — enumerated in
`directives/DIR-027-pack-fleet-publication-migration.directive.md`). Nothing gates the pack
repositories themselves. There is no CI workflow in any of them that runs `backstop pack check`
or `backstop pack test` on push, and nothing asserts that a tag's version matches the manifest
before the tag is pushed. A publisher can push a broken tag and it ships with no gate ever
running against it.

Consumers DO validate at add-time: SPEC-055 REQ-008 made `AddCommand`/`UpgradeCommand` run
`RunPackCheck` then `RunPackTest` unconditionally before mutating consumer state, so a broken
pack fails loud — but only at the point a consumer tries to add or upgrade it. That is the wrong
place to discover a publisher mistake: the failure surfaces downstream, per-consumer, after the
tag is already public, instead of once at the source before anyone pulls it. And one class of
publisher mistake isn't caught at all yet, at either end: a tag whose version doesn't match the
manifest's declared version. SPEC-055's REQ-039 (source-coordinate vs. manifest-identity
separation, recorded with fail-closed mapping semantics as a requirement) is explicitly deferred
— per `bundles/BUNDLE-006-pack-distribution-lifecycle.bundle.md:1947-1948` (SPEC-055's own
"Explicitly defers" line) — so today NEITHER the publisher NOR the consumer catches a
manifest/tag version mismatch.

The gap is not hypothetical. The harness toolchain pack (`backstop-ai/backstop-harness-toolchain-pack`)
is a live, standing example of exactly this drift today: its manifest declares version `0.1.3`
while its published tags stop at `v0.1.1`, plus uncommitted changes on top — captured in
`directives/DIR-027-pack-fleet-publication-migration.directive.md` under "Reconcile the harness
packs." No CI caught the drift; it was found by a person reading the repo.

## Solution

Publisher-side gating: a CI workflow per pack repo that runs `backstop pack check` + `backstop
pack test` on every push, and, on a tag push specifically, additionally asserts manifest version
== tag version (the same rule SPEC-055 REQ-039 will eventually enforce consumer-side, applied
here at the source instead). This does not replace consumer-side validation (REQ-008) or the
eventual REQ-039 mismatch detection — it adds a gate at the point where the mistake is cheapest
to catch: before the tag is pushed, once, instead of after, per consumer.

Note the recipe-capability tie-in: BUNDLE-015 REQ-018 already commits to a CI recipe pack (in
`backstop-packs`, not core) shipping per-platform gate-workflow recipes (github/gitlab/bitbucket/
jenkins) as the packs-only acceptance test for the recipe-scaffolding capability. That recipe
pack is a plausible *vehicle* for shipping this exact workflow to pack repos — the publisher-CI
gap described here would be its own first customer, gating `backstop-ai` pack repos with the
same recipe mechanism the capability is proving out. This issue does not decide that the CI
recipe pack IS the delivery mechanism, only that it is a strong candidate worth recording.

### Homing (options to record, not a decision)

Three plausible owners for this work, left for the founder/backlog to choose between:

1. **DIR-001's release workflow** — the publisher half of release tooling already scoped there.
2. **A fifth thread on DIR-027** (pack-fleet-publication-migration) — alongside its existing four
   threads (extract vendored packs, reconcile harness packs, migrate fleet off local names,
   absorb Clone-strip asymmetry), since this is fleet/ecosystem work in the same shape.
3. **DIR-019 via BUNDLE-015 REQ-018** — if the CI recipe pack is built as the delivery mechanism,
   this issue's scope is satisfied by applying that recipe to the ten pack repos rather than by
   hand-writing a bespoke workflow.

## References

- `directives/DIR-027-pack-fleet-publication-migration.directive.md` — enumerates the ten
  published packs; documents the harness pack's manifest/tag drift (`0.1.3` vs. `v0.1.1`) as an
  open reconciliation thread; does NOT cover publisher-side CI gating (its four threads are
  extraction, harness reconciliation, fleet migration off local names, and Clone-strip asymmetry)
- `specs/SPEC-055-production-remote-dependency-assembly.spec.md:184-236` — REQ-008/REQ-011,
  consumer-side unconditional validation at add/upgrade time (the mechanism this issue's gap
  sits upstream of)
- `specs/SPEC-055-production-remote-dependency-assembly.spec.md:1947-1948` — REQ-039 (manifest
  vs. source-coordinate identity) explicitly deferred out of SPEC-055's scope
- `bundles/BUNDLE-006-pack-distribution-lifecycle.bundle.md:473-478,847-852` — REQ-039 definition
  and status (fail-closed mapping semantics still owed)
- `bundles/BUNDLE-015-pack-scaffolding-recipes.bundle.md:270-277` — REQ-018, the CI recipe pack
  as the packs-only acceptance test; the candidate delivery vehicle noted above
- Ten published pack repos under `github.com/backstop-ai` (see DIR-027 for the full enumeration
  with versions)
