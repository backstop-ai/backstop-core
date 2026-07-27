---
title: "Pack Fleet Publication & Migration"
number: DIR-027
created: "2026-07-26"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "BUNDLE-006"
    - "SPEC-055"
---

## Description

DIR-026 made remote pack consumption *work*; this directive makes it *true
for the fleet*. SPEC-055 shipped the production `ExecGitCloner` and
fail-closed command wiring, and ten packs were published today under the
`name == coordinate` convention at `github.com/backstop-ai`
(`cobra-cli-standards@0.2.0`, `go-standards@1.2.0`, `go-toolchain@1.2.0`,
`bun-toolchain@1.2.0`, `backstop-self@1.1.0`, `typescript-toolchain@1.2.0`,
`typescript-standards@1.1.0`, `typescript-contracts@1.2.0`,
`typescript-substantiveness@1.1.0`, `secrets@1.1.0`), each proven by a real
remote `pack add` — a clean consumer holds all ten as `source_type: git`.
That is delivered context, cited here rather than re-scoped. What remains is
four repo-and-fleet threads, all in the same "packs are external, the tap
is the durable thing" shape as DIR-026, but landing as ecosystem work rather
than core mechanism:

1. **Extract the last two vendored packs.** `packs/contracts` and
   `packs/substantiveness` still live inside `backstop-core` — verified, no
   separate repository exists for either anywhere. Tier 1: publish, seeding
   the repos from the vendored directories (naming decision to record: the
   symmetric convention suggests `backstop-ai/go-contracts` and
   `backstop-ai/go-substantiveness`, founder to confirm before the repos are
   created). Tier 2: de-vendor core — repoint the two production consumers
   (`pkg/pack/distribution/contracts_local_install.go`, rebuilt fresh by
   SPEC-055 REQ-013, and `cmd/backstop/gate_substantiveness_e2e.go`) plus
   the roughly ten dependent test files, migrate this repo's own two
   no-`local_path` lock entries (`backstop/contracts`, `backstop/substantiveness`
   in `backstop.lock` today), then delete the vendored directories — with a
   lineage sweep, since SPEC-037/SPEC-038 were authored about these packs
   and `deletion-strands-lineage` applies to their retirement, not a bare
   `rm`.
2. **Reconcile the harness packs.** `backstop-ai/backstop-harness-toolchain-pack`
   is published under the old `-pack`-suffix convention with version drift
   (tags `v0.1.0`/`v0.1.1` vs. a manifest at `0.1.3`, plus uncommitted
   changes); `backstop-harness-architecture-pack` is unpublished entirely.
   Decide rename-vs-grandfather against the `name == coordinate` convention
   the other ten packs now hold, then reconcile tags to match the manifest
   either way.
3. **Migrate the fleet off local names onto published coordinates.** Every
   consumer still resolves at least one pack locally: `backstop-core` (the
   six packs in its own `backstop.lock` — `cobra-cli`, `contracts`,
   `go-standards`, `go-toolchain`, `self`, `substantiveness` — all
   `source_type: local` today), `bclabs-portal` (the TypeScript suite plus
   `secrets`, `contracts`, `substantiveness` — its lock currently resolves
   via sibling filesystem paths reproducible on no machine but the one it
   was written on, which is the concrete form of the client-facing
   proof-surface argument for this directive), `stash` (`cobra-cli`,
   `go-standards`), and `backstop-harness` (its toolchain pack via a
   `-local` mirror directory). Each migration is mechanically remove-old-name
   + add-published-coordinate per consumer, not a bulk operation — `pack
   relock` is explicitly NOT the vehicle for this (the arg-shape residual in
   ISSUE-074, cited here as related, is homed by a separate decision, not by
   this directive).
4. **Absorb the Clone-strip transitional asymmetry.** SPEC-055's
   `ExecGitCloner` installs remote packs `.git`-free while locally-sourced
   packs are untouched, so the two install paths diverge in what a pack
   directory actually contains. That asymmetry has no separate fix — it
   resolves per-consumer, automatically, as each pack in threads 1-3 above
   moves from local to remote.

**Dependency, not scope.** The legacy-lock content-hash migration and
`pack relock`/remote-migration repair (BUNDLE-006 REQ-041, DD-28) is
code-side work seeded in BUNDLE-006 and destined for its own spec. This
directive owns the ecosystem/repo side — which packs exist, where they're
published, and which consumer lock file points at which coordinate — and
depends on that seed landing, but does not itself design the migration
mechanism.

## Acceptance Criteria

- `packs/contracts` and `packs/substantiveness` no longer exist inside
  `backstop-core`; both are published repositories consumed the same way as
  the other ten.
- `backstop-harness-toolchain-pack`'s published tags match its manifest
  version, and `backstop-harness-architecture-pack` is either published
  under the `name == coordinate` convention or explicitly deferred with a
  recorded reason.
- Every consumer in the fleet (`backstop-core`, `bclabs-portal`, `stash`,
  `backstop-harness`) holds only `source_type: git` lock entries pointing at
  `backstop-ai` coordinates — zero `local_path` entries remaining outside a
  deliberately-local development pack.
- A fresh clone of each consumer, with no sibling directories present,
  installs its full pack set from its lock file alone.

## Notes

Positioned in BACKLOG.yml directly after DIR-026 (remote consumption
mechanism) and ahead of DIR-024 (gate/engine quality catch-all), per the
founder's 2026-07-26 ratification of the backlog-PM's proposal to treat this
as a single migration directive rather than splitting it across the
directives that happen to share subject matter (DIR-023's local-provenance
thread, DIR-026's mechanism thread). DIR-026 stays scoped to the mechanism
(SPEC-055); this directive is the fleet actually using it. Do not fold
ISSUE-074 fully into this directive's scope — only its "related" citation
travels here; its home/open-status decision is still pending the founder
per the PM's 2026-07-26 escalation, and REQ-041 is where its Half B
(path-vs-name argument) is tracked as load-bearing.

## References

- BUNDLE-006 (`pack-distribution-lifecycle`) — REQ-041/DD-28 (legacy-hash
  migration seed), SPEC-037/SPEC-038 (original contracts/substantiveness
  pack authorship, relevant to the extraction's lineage sweep)
- SPEC-055 (`Production Remote Dependency Assembly`) — `ExecGitCloner`,
  Clone-strip, the ten published packs' proof of remote install
- ISSUE-074 (`pack relock` silent failure / path-vs-name argument) — related,
  not owned; home decision pending
- `docs/CODEBASE-MAP.md` "Pack lifecycle" section
