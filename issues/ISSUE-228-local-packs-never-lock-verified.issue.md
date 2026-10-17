---
title: "Local Packs Never Lock Verified"
schema_version: issue/v1

issue:
  id: ISSUE-228
  title: "Local Packs Never Lock Verified"
  type: bug
  status: open
  created: "2026-10-17"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Local Packs Never Lock Verified

## Problem

`VerifyLock` skips every lock entry whose `source_type` is `local`, so editing a local pack's installed
content is never detected at gate time. A git pack gets the opposite treatment: its installed content is
hashed on every gate and a mismatch fails the `pack_lock_verification` step with `hash_mismatch`.

Verified at origin/main:

- `pkg/pack/distribution/verify.go:40-44` — inside the per-entry loop, `if entry.SourceType == "local" {
  continue }` with the comment "Skip local packs — they are not in packsDir."
- That comment does not match the rest of the code. `Relock` (`pkg/pack/distribution/relock.go:49-57`)
  reads the same local pack from `.backstop/packs/<name>` and hashes it with `ComputeContentHash`, so a
  local pack IS materialized under the packs directory and a content hash IS recorded for it.
- `detectExtraUnlocked` (`verify.go`, after the loop) does walk the packs directory, so an unlocked
  extra pack is caught, but a locked local pack whose files changed (or whose installed directory was
  deleted) produces neither `hash_mismatch` nor `missing_pack`.

### Reproduction (generic)

A project that adds a pack from a local path, then edits a rule or script in the installed copy under
`.backstop/packs/`, then runs `backstop gate`. `pack_lock_verification` passes. Repeat with a git-sourced
pack: it fails with `hash_mismatch`.

## Impact

Pack content decides what the gate enforces. For local packs the lock gives no tamper or drift evidence,
so a changed rule is silently the rule in force, and the lock entry's `content_hash` is decorative.

## Direction

(Direction, not a plan.) Verify local entries with the same hash comparison as git entries, including
`missing_pack`. `backstop pack relock <path>` (`cmd/backstop/pack_relock.go`) already exists as the
explicit way to accept a local edit; the failure message for a local pack should name it instead of
"run pack install", which would not restore a local pack that has no remote source. The maintainer has
agreed local packs should be verified.

## Notes

- Related, not the same: ISSUE-055 (local packs lack a durable provenance record for `pack install`).
  It concerns reconstructing a local pack on a fresh checkout, not verifying its content. Fixing both
  touches how `local` entries are treated, so they should be planned together.
- Not confirmed: whether any existing project relies on editing a local pack without relocking; making
  verification strict will turn those into gate failures, so the change needs a visible message.
