---
title: "Local-Path Pack Installs Hash Root `.git` Into `content_hash` (REQ-021@1.1.0 Residual)"
schema_version: issue/v1

issue:
  id: ISSUE-088
  title: "Local-Path Pack Installs Hash Root `.git` Into `content_hash` (REQ-021@1.1.0 Residual)"
  type: bug
  status: open
  created: "2026-07-27"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Local-Path Pack Installs Hash Root `.git` Into `content_hash` (REQ-021@1.1.0 Residual)

## Problem

`ComputeContentHash` (`pkg/pack/distribution/hash.go:17`) walks *every file* under the
directory it is given and folds each one into the sorted manifest that becomes the lock
entry's `content_hash` — it has no `.git` exclusion of any kind. For a **remote** `pack add
<org>/<pack>@<version>`, that's fine in practice only because `ExecGitCloner.Clone`
(`pkg/pack/distribution/gitcloner.go:29,78`) strips the root `.git` directory from the clone
destination before anything downstream (including the copy that `ComputeContentHash` later
walks) ever sees it — that strip is SPEC-055's CLM-101/CLM-102 and is scoped explicitly to the
clone path.

For a **local-path** `pack add <path>` (or `pack relock`, `pack list`, `pack verify` —
every one of the five call sites in `pkg/pack/distribution/` listed below), there is no
equivalent strip. `ComputeContentHash` is called directly against the source directory (or a
copy of it) exactly as it sits on disk. If that source directory is a pack repo checked out
with its own `.git`, the root `.git` directory's contents (reflog timestamps, object layout,
etc.) get walked into the hash manifest right alongside the pack's authored files.

### Consequence

Identical pack *content* yields different lock `content_hash` values depending only on
whether the local source directory happens to carry repository metadata:

- Remote installs are `.git`-free by construction → reproducible hashes across machines/clones.
- Local-path installs are contaminated by whatever `.git` state the source repo happens to be
  in at install time → the same pack content hashes differently machine-to-machine, and a
  fresh-clone hash verification against a lock written on a machine that held the source repo
  is guaranteed to mismatch even though nothing about the pack changed.
- Legacy locks written before this was understood may carry metadata-inclusive hashes,
  compounding the asymmetry across the fleet.

### Evidence (measured, 2026-07-25/26)

The same pack directory hashed differently with and without its root `.git` present:
`639f74fb…` (no `.git`) vs. `bb86715c…` (with `.git`) — recorded in
`bundles/BUNDLE-006-pack-distribution-lifecycle.bundle.md` around lines 771-773, which states
plainly: "REQ-021@1.1.0 and DD-24 remain specified-and-unbuilt."

`ComputeContentHash` is called from five sites, none with a `.git` exclusion:
- `pkg/pack/distribution/install.go:78`
- `pkg/pack/distribution/command.go:172`
- `pkg/pack/distribution/command.go:427`
- `pkg/pack/distribution/command.go:807`
- `pkg/pack/distribution/list.go:143`
- `pkg/pack/distribution/relock.go:54`
- `pkg/pack/distribution/verify.go:64`

(all read the walked directory as-is; the only place `.git` is ever removed is
`gitcloner.go`'s post-clone strip, which runs solely on the remote-clone path.)

### Requirement context — why this is a residual, not a fresh gap

`bundles/BUNDLE-006-pack-distribution-lifecycle.bundle.md` already carries REQ-021 in two
pinned generations:

- **REQ-021@1.0.0** (bundle line ~309) is the *original*, Git-metadata-inclusive definition —
  it is pinned historically by `SPEC-015` (`pack-distribution-lifecycle:REQ-021@1.0.0`, bundle
  line ~1437) and the bundle is explicit that "that pin must not be rewritten."
- **REQ-021@1.1.0** (DD-24, bundle lines ~535-567, ~1034, ~1072, ~1108) redefines pack content
  identity to cover *authored* pack content only, excluding root repository metadata — this is
  the corrected semantics the current codebase should converge on.
- **OQ-6 ("Legacy hash migration")** is marked `[RESOLVED]` in the bundle (line ~1260) but the
  migration mechanism itself (DD-28, `pack relock` as the "designated migration vehicle") is
  separately broken per ISSUE-074 (`pack relock` rejects the pack-name argument shape every
  sibling command uses).
- SPEC-055 (which delivered the remote-path `.git` strip, CLM-101/CLM-102) explicitly disclaims
  this scope in its own claim notes (`specs/SPEC-055-production-remote-dependency-assembly.spec.md:910`):
  "It does NOT deliver bundle REQ-021@1.1.0, which is a requirement about the COPY/HASH
  boundary for ALL sources plus the legacy-lock migration; local-path packs still hash whatever
  is on disk, and this spec deliberately does not pin REQ-021."
- No spec since (SPEC-055, SPEC-056) pins REQ-021@1.1.0 or REQ-020@1.1.0. There is currently no
  delta spec that owns closing this gap.

### Current gate impact

`requirement_traceability` is currently RED on SPEC-015's historic REQ-020/021@1.0.0 pins —
this is **by design** per the bundle (the 1.0.0 pin is intentionally frozen, not to be
rewritten in place). The bundle's prescribed remedy is a new delta spec pinning
REQ-020@1.1.0 and REQ-021@1.1.0, which does not yet exist.

### Severity and scope of this issue

This is a correctness/reproducibility defect, **not currently fleet-blocking**: every pack
published and consumed today installs via the remote path (SPEC-055), which is `.git`-free by
construction. The exposure is local-path installs specifically (dev/test packs installed by
directory), which is the minority path today but is exactly the workflow BUNDLE-006's
identity/migration seed is meant to eventually harden.

This issue is intentionally scoped **not** to prescribe or implement the fix here. The actual
hash-boundary fix (excluding `.git` — and any other non-authored-content boundary DD-24 needs
to define precisely — from `ComputeContentHash`, plus the legacy-hash migration mechanism) is
bundle-track work that belongs in a future delta spec against BUNDLE-006's content
identity/migration seed (REQ-021@1.1.0, DD-24, DD-28, OQ-6's still-open migration mechanics).
This issue exists so that capability gap has a first-class tracked home in the issue backlog
and doesn't remain buried only in bundle prose.

## References

- `pkg/pack/distribution/hash.go:17` — `ComputeContentHash`, no `.git` exclusion of any kind
- `pkg/pack/distribution/gitcloner.go:29,78` — `ExecGitCloner.Clone`'s post-clone root `.git`
  strip, scoped only to the remote-clone path (SPEC-055 CLM-101/CLM-102)
- `pkg/pack/distribution/install.go:78`, `command.go:172`, `command.go:427`, `command.go:807`,
  `list.go:143`, `relock.go:54`, `verify.go:64` — the seven `ComputeContentHash` call sites,
  none `.git`-aware
- `bundles/BUNDLE-006-pack-distribution-lifecycle.bundle.md:309-331` — REQ-021@1.0.0 (original,
  metadata-inclusive) definition
- `bundles/BUNDLE-006-pack-distribution-lifecycle.bundle.md:535-567,1034,1072,1108` — DD-24 /
  REQ-021@1.1.0 (authored-content-only) redefinition
- `bundles/BUNDLE-006-pack-distribution-lifecycle.bundle.md:740,771-773` — measured hash pair
  (`639f74fb…` vs `bb86715c…`) and "REQ-021@1.1.0 and DD-24 remain specified-and-unbuilt"
- `bundles/BUNDLE-006-pack-distribution-lifecycle.bundle.md:1260-1262` — OQ-6, `[RESOLVED]`
  status, migration mechanics still pending
- `bundles/BUNDLE-006-pack-distribution-lifecycle.bundle.md:1437,1441` — SPEC-015's frozen
  REQ-021@1.0.0 pin; instruction that future specs must pin REQ-021@1.1.0 instead of rewriting it
- `specs/SPEC-055-production-remote-dependency-assembly.spec.md:910,1871-1872` — SPEC-055's own
  claim notes disclaiming REQ-021@1.1.0 delivery and describing CLM-101/CLM-102's scope
- `issues/ISSUE-074-pack-relock-silent-failure.issue.md` — the migration-vehicle (`pack relock`)
  defect that blocks DD-28's designated legacy-hash migration path
- Discovered during the SPEC-015 pin investigation on 2026-07-27, while diagnosing why
  `requirement_traceability` gates RED on SPEC-015's historic REQ-020/021@1.0.0 pins
