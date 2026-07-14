---
title: "Local Provenance Cache For Local Packs"
schema_version: issue/v1

issue:
  id: ISSUE-055
  title: "Local Provenance Cache For Local Packs"
  type: technical-debt
  status: open
  created: "2026-07-14"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# ISSUE-055: Local Provenance Cache For Local Packs

## Problem

Packs added from a local filesystem path (`pack add ../backstop-packs/ts-toolchain`,
the founder's actual day-to-day workflow for the TypeScript pack suite — see
`project_typescript_packs`) do not durably record where they came from, so
`pack install` after a clean clone or a fresh checkout cannot reconstruct them —
they have to be re-added by hand.

`pkg/pack/distribution/lockfile.go`'s `LockEntry` does carry a `LocalPath` field,
and `pkg/pack/distribution/add.go` does compute it — as the local pack's source
directory **relative to the project root** — at add time:

```go
// pkg/pack/distribution/lockfile.go:24-28
// LocalPath is the local-source pack's directory RELATIVE TO THE PROJECT ROOT,
// recorded at add time so a later install can re-materialize the pack from a durable,
// portable record. Empty for git-source packs. It is provenance only — it is NOT part
// of ComputeContentHash (a source path is not pack content).
LocalPath string `yaml:"local_path,omitempty"`
```

But this field is written straight into `backstop.lock`, which is **tracked and
committed** (`git ls-files backstop.lock` confirms it in this repo). Two problems
follow directly from that:

1. **This repo's own lock file already exhibits the failure.** `backstop-core`'s
   committed `backstop.lock` has five `source_type: local` pack entries
   (`backstop/contracts`, `backstop/go-standards`, `backstop/go-toolchain`,
   `backstop/self`, `backstop/substantiveness`) and **none of them carry a
   `local_path`** — the field is silently absent from every entry. Because
   `materializeLocalPack` in `pkg/pack/distribution/install.go:212-227` fails
   loud precisely in this situation —

   ```go
   // pkg/pack/distribution/install.go:215-217
   if entry.LocalPath == "" {
       return fmt.Errorf("local pack %q has no recorded source path (local_path) "+
           "in backstop.lock; cannot materialize it (re-add the pack)", name)
   }
   ```

   — a `pack install` run against this exact repo, from a clean clone, would fail
   loud on all five packs today. This is not hypothetical: it is the state of the
   repo's own lock file right now.

2. **Even when populated, a project-relative path in a COMMITTED file isn't
   portable.** The founder's real local packs (the TypeScript pack suite) live in
   a sibling checkout at `~/src/projects/backstop-packs/`, outside the project
   root. A path relative to the project root pointing there (`../backstop-packs/…`)
   bakes in an assumption about sibling-checkout directory layout that only holds
   on the machine that made the entry — it is meaningless to a teammate whose
   checkout layout differs, and committing it leaks local filesystem structure
   into shared history. An absolute path would be worse (meaningless on any other
   machine, and an outright leak), but a naively "portable" relative path inside a
   shared file is still the wrong place for machine-specific provenance to live.

## Solution

Not committed — left open for the plan. Direction settled by BUNDLE-003 OQ-4
(RESOLVED, 2026-07-13):

1. The **committed** `backstop.lock` should record **only portable git-ref
   packs** — no `local_path` field survives into the tracked file at all.
2. Local-path pack sources move to a **GITIGNORED local provenance cache** (e.g.
   `.backstop/pack-sources.local.json`), written at `pack add` time alongside (not
   instead of) the lock entry, so `pack install` can restore local packs **on the
   same machine** by consulting the cache first.
3. `materializeLocalPack` (`pkg/pack/distribution/install.go`) reads the source
   path from the new gitignored cache instead of (or in addition to, during
   migration) `LockEntry.LocalPath`; its existing fail-loud behavior when no
   source is resolvable should stay — the fix is giving it a source to find, not
   softening the loud failure.
4. Promoting a local pack to shareable (so a teammate can install it too) is an
   **explicit git-ref step** — publish the pack to a repo and re-`pack add` it by
   git ref. There is no automatic promotion path.
5. Real fresh-clone / reinstall e2e is the bar per `project_pack_provisioning_integration_gap`
   — a unit test stubbing the cache is not sufficient proof; prove a clean clone of
   a repo with a local pack in its (fixed) lock file can `pack install` and
   materialize it via the new cache on the same machine.
6. As part of the fix, reconcile this repo's own five local-pack entries so
   `backstop.lock` stops carrying (or ever carried) an unusable empty
   `local_path`, and `pack install` against `backstop-core` itself stops being a
   live instance of the bug it fixes.

## References

- BUNDLE-003 OQ-4 (RESOLVED, 2026-07-13) — the source of this issue's direction:
  committed lock holds only portable git-ref packs; gitignored local-provenance
  cache restores local packs on the same machine; promotion to shareable is an
  explicit git-ref step
- `pkg/pack/distribution/lockfile.go` — `LockEntry.LocalPath`, the field this
  issue moves out of the committed lock
- `pkg/pack/distribution/add.go` — computes the project-relative local path at
  `pack add` time; the write site that needs to target the new cache instead
- `pkg/pack/distribution/install.go` — `materializeLocalPack`, the fail-loud
  consumer this issue needs to keep failing loud, now against a resolvable source
- `backstop.lock` (this repo) — live, present-day proof of the gap: five
  `source_type: local` entries, zero `local_path` values
- `project_typescript_packs` / `project_pack_provisioning_integration_gap` (agent
  memory) — the founder runs the TS pack suite as local packs today; this is a
  live gap, not a hypothetical one; and the recurring mandate that pack-migration
  work be proven via real installed-pack e2e, not a stub
- `project_pack_distribution` (agent memory) — packs are installed (not
  vendored) into gitignored `.backstop/packs/`; this issue extends that boundary
  to provenance, not just installed content
