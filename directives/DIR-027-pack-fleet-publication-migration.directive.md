---
title: "Pack Fleet Publication & Migration"
number: DIR-027
created: "2026-07-26"
schema_version: directive/v1

directive:
  status: active
  source:
    - "BUNDLE-006"
    - "SPEC-055"
    - "ISSUE-084"
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
five repo-and-fleet threads, all in the same "packs are external, the tap
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

   **Correction, 2026-08-02 (verified against `backstop.lock`):** the claim
   above that `backstop-core` still resolves any pack locally is now false.
   `backstop.lock` holds seven entries, and **all seven** are
   `source_type: git` at `backstop-ai/*` coordinates, zero `local_path`
   entries: `backstop-core-architecture@0.1.1`, `backstop-self@1.1.2`,
   `cobra-cli-standards@0.2.1`, `go-contracts@1.2.0`, `go-standards@1.2.1`,
   `go-substantiveness@1.2.0`, `go-toolchain@1.3.0`. What that does and does
   not mean for this directive's own acceptance criteria:
   - Thread 1 **tier 1 is delivered**: `go-contracts` and
     `go-substantiveness` are published under the `name == coordinate`
     convention and `backstop-core` consumes both remotely.
   - Thread 1 **tier 2 is not delivered**. `packs/contracts` and
     `packs/substantiveness` still exist on disk inside `backstop-core`,
     and both still have live in-tree consumers — verified:
     `pkg/pack/distribution/contracts_local_install.go` and
     `cmd/backstop/gate_substantiveness_e2e.go` both still exist and
     reference the vendored packs. The de-vendoring, the SPEC-037/SPEC-038
     lineage sweep, and the directory deletion all remain open.
   - This closes the local-resolution gap **for `backstop-core` only**.
     `bclabs-portal`, `stash`, and `backstop-harness` were not re-verified
     as part of this correction and their lock state is unknown as of this
     writing — do not read this as fleet-wide completion of thread 3.
   - The seventh lock entry, `backstop-ai/backstop-core-architecture@0.1.1`,
     is a distinct pack from `backstop-harness-architecture-pack` discussed
     in thread 2 below (verified on disk: the harness pack is still
     `backstop-ai/backstop-harness-architecture-pack` at manifest `0.1.0`,
     unpublished-tag state unchanged). `backstop-core-architecture`'s home
     — whether it belongs to this directive, some other directive, or its
     own — is an open founder question the backlog-PM escalated on
     2026-08-02 and is not resolved by this correction.
   - Thread 2 is otherwise untouched: `backstop-harness-toolchain-pack`
     (manifest `0.1.3`) and `backstop-harness-architecture-pack` (manifest
     `0.1.0`) both still carry the old `-pack` suffix, confirmed on disk at
     `~/src/projects/`; the rename-vs-grandfather decision is still open.
4. **Absorb the Clone-strip transitional asymmetry.** SPEC-055's
   `ExecGitCloner` installs remote packs `.git`-free while locally-sourced
   packs are untouched, so the two install paths diverge in what a pack
   directory actually contains. That asymmetry has no separate fix — it
   resolves per-consumer, automatically, as each pack in threads 1-3 above
   moves from local to remote.
5. **Publisher-side gating (ISSUE-084).** Nothing gates the ten pack repos
   themselves — a publisher can push a broken tag and it ships with no
   check ever running against it; consumer-side validation (SPEC-055
   REQ-008) only discovers the mistake downstream, per-consumer, after the
   tag is already public. Add a CI workflow to each pack repo running
   `backstop pack check` + `backstop pack test` on every push, plus a
   tag-time job asserting manifest version == tag version — the same rule
   BUNDLE-006 REQ-039 will eventually enforce consumer-side (still
   deferred, tracked under this directive's dependency note above),
   applied here at the source instead, where the mistake is cheapest to
   catch. Write this workflow as **deliberately temporary**, with a named
   successor: BUNDLE-015 REQ-018 already commits to a CI recipe pack
   (github/gitlab/bitbucket/jenkins gate-workflow recipes, built in
   `backstop-packs` as the packs-only acceptance test for the recipe-
   scaffolding capability) — once it lands, it subsumes these hand-written
   workflows, and the ten published pack repos become REQ-018's first real
   consumers, resolving the no-consumer gap that capability currently has.
   Do not wait for REQ-018 to land before doing this — hand-write the gate
   now, since a bad tag ships silently until something gates the push, and
   REQ-018 has no committed timeline. Sequence this alongside threads 1-3:
   the pack repos those threads open (extraction, harness reconciliation,
   fleet migration) are the same repos this thread gates, so land the CI
   workflow in the same pass rather than a separate round-trip per repo.

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
- Every published pack repo has a CI workflow that runs `backstop pack
  check` + `backstop pack test` on push and asserts manifest version ==
  tag version on a tag push; the workflow is marked as temporary pending
  BUNDLE-015 REQ-018.

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
- ISSUE-084 (`published-packs-ungated-at-tag-time`) — publisher-side CI
  gating for the fleet (thread 5); names BUNDLE-015 REQ-018 (CI recipe
  pack) as the named temporary-workflow successor and BUNDLE-006 REQ-039
  (manifest/tag identity) as the consumer-side counterpart still deferred
- `bundles/BUNDLE-015-pack-scaffolding-recipes.bundle.md` — REQ-018, the CI
  recipe pack that eventually subsumes thread 5's hand-written workflows
- `docs/CODEBASE-MAP.md` "Pack lifecycle" section
