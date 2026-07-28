---
title: "`pack add` silently no-ops converting an installed local pack to a git source"
schema_version: issue/v1

issue:
  id: ISSUE-095
  title: "`pack add` silently no-ops converting an installed local pack to a git source"
  type: bug
  status: open
  created: "2026-07-28"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# `pack add` silently no-ops converting an installed local pack to a git source

## Problem

`pack add <org>/<pack>@<version>` silently no-ops when a pack of the same name is
already installed from a **local** source at the same version, instead of performing
the local→git conversion or refusing loudly. It reports "already installed and up to
date" and exits 0, but `backstop.lock`'s `source_type: local` and out-of-repo
`local_path` are left untouched — the operator believes the pack now resolves from
git and it does not.

### Root cause

`isPackInstalledAndCurrent` (`pkg/pack/distribution/add.go:135-151`) is the gate both
branches of `AddCommand.Run` consult before installing:

```go
func isPackInstalledAndCurrent(projectDir, packName string) bool {
	packDir := filepath.Join(projectDir, ".backstop", "packs", packName)
	entries, err := os.ReadDir(packDir)
	if err != nil || len(entries) == 0 {
		return false
	}
	lf, err := ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if err != nil || lf == nil {
		return false
	}
	_, ok := lf.Packs[packName]
	return ok
}
```

It answers "does a non-empty directory exist at this name, and does the lock have
*an* entry for this name" — it never compares the entry it found against the source
just resolved by the caller. Both call sites short-circuit on that answer alone:

- Local branch, `pkg/pack/distribution/command.go:185-187` — checked immediately
  after `resolveLocalPackSource`, before any source comparison.
- Git branch, `pkg/pack/distribution/command.go:242-244` — checked immediately after
  `ValidateRemoteIdentity` resolves `packName`/`version`/`gitRef`, but the comparison
  still only asks `isPackInstalledAndCurrent`, which ignores everything it just
  resolved.

`LockEntry` (`pkg/pack/distribution/lockfile.go:17-37`) carries `SourceType`,
`SourceCoordinate`, `Version`, and `GitRef` — all the fields needed to detect a
conversion — but `isPackInstalledAndCurrent` reads none of them. It was written to
close ISSUE-026 (distinguishing DECLARED-in-manifest from INSTALLED-on-disk), and it
does that correctly; it was never extended to distinguish INSTALLED-FROM-THIS-SOURCE
from INSTALLED-FROM-A-DIFFERENT-SOURCE. Same-name-same-version is index-keyed
identity (SPEC-056: the manifest name is the install identity) doing its job for a
true re-add — the defect is that a **source-type or coordinate change** is not
recognized as a change at all.

### Repro (measured during PLAN-ISSUE-020 fleet migration, 2026-07-28)

Measured on `backstop-ai/backstop-self@1.1.2` and `backstop-ai/go-standards@1.2.1`.

1. Have a pack installed from a local path: `backstop.lock` holds
   `source_type: local`, `local_path: ../some-pack-dir`, no `source_coordinate`.
2. Run `backstop pack add backstop-ai/go-standards@1.2.1` (the git coordinate for the
   same manifest name, same version).
3. Observe: exit 0, `Pack backstop-ai/go-standards is already installed and up to
   date.` printed. `backstop.lock`'s entry is byte-for-byte unchanged —
   `source_type: local` and `local_path` remain, no `source_coordinate` is recorded,
   no clone happens, `.backstop/packs/backstop-ai/go-standards/` still holds the
   local-sourced content.
4. The operator has no signal that the conversion did not happen; the command's
   output actively asserts the opposite ("already installed and up to date").

### Workaround (confirmed)

`pack remove <name>` followed by `pack add <name>@<version>` performs the conversion
correctly, because `remove` deletes the lock entry and installed directory outright,
so the subsequent `add` finds nothing at `isPackInstalledAndCurrent` and takes the
full install path.

### Sibling correction (same evidence run)

`pack remove` on a stale coordinate **does** work correctly when the install
directory still exists on disk — it removes the `backstop.yml` config key, the
lock entry, and the `.backstop/packs/<name>/` directory in one step
(`pkg/pack/distribution/remove.go:24-32`). The prior belief that `remove` reports
"pack is not installed" in this situation only holds when the install directory has
already been deleted by some other means (e.g. a manual `rm -rf .backstop/packs/`)
— `remove.go:32`'s "not installed" error fires on a missing lock entry, not on
source-type mismatch, so a same-name entry with a diverged source is still found and
removed cleanly.

### Position in the rename/migration sequence

This is the eighth silent-failure step encountered in the pack rename/migration
recipe (see the "seven-step" rename block in
`plans/PLAN-ISSUE-020-linux-sandbox-gate-in-ci.plan.yml` and
`.claude/agent-memory/implementer/project_pack_rename_migration_recipe.md`). DIR-027
owns the fleet publication migration theme this defect blocks.

### Impact

Any local→git (or git→local, or coordinate/version-preserving source swap)
conversion silently fails while reporting success. For a fleet migrating packs from
dev-local paths to published git coordinates (DIR-027's theme), every same-version
conversion looks like it worked and did not — the lock keeps pointing at a local
path outside the repo, which is not durable and not what `pack.yml`/CI on another
machine can resolve.

## Solution

Not prescribed here (issue track, not a fix). Sketch for the fix planner: a same-name
add whose resolved source (type, coordinate, or version) differs from the existing
lock entry must not be treated as "already current" — it must either perform the
conversion (install from the newly resolved source and rewrite the lock entry) or
refuse loudly, naming the specific field that differs (e.g. "pack backstop-ai/
go-standards is installed from a local path (../some-pack-dir); refusing to silently
convert to git@1.2.1 — re-run with an explicit conversion flag" or similar). Silence
is the defect per the loud-not-blocking philosophy; either outcome is acceptable as
long as it is never silent. `isPackInstalledAndCurrent` (or its call sites) needs the
resolved source (type/coordinate/version) threaded in for comparison against the
existing `LockEntry`, not just the pack name.

## References

- `pkg/pack/distribution/add.go:135-151` — `isPackInstalledAndCurrent`, the gate that
  ignores source type/coordinate/version and answers on name + disk/lock presence only
- `pkg/pack/distribution/command.go:185-187` — local branch's early return on the gate
- `pkg/pack/distribution/command.go:242-244` — git branch's early return on the gate,
  after `packName`/`version`/`gitRef` are already resolved and available for comparison
- `pkg/pack/distribution/lockfile.go:17-37` — `LockEntry` fields (`SourceType`,
  `SourceCoordinate`, `Version`, `GitRef`, `LocalPath`) available but unused by the gate
- `pkg/pack/distribution/remove.go:24-32` — `Remove`, confirmed to work correctly on a
  stale-coordinate entry whose install directory still exists
- `issues/ISSUE-026-pack-add-silent-noop.issue.md` — closed; fixed the DECLARED-vs-
  INSTALLED conflation this issue's gate exists to enforce, but did not cover a source
  change at the same name/version, which is this issue's gap
- `issues/ISSUE-074-pack-relock-silent-failure.issue.md` — sibling silent-failure defect
  in the same pack-lifecycle command family (relock's silent stderr swallow, since fixed
  by SPEC-055 REQ-011; its residual arg-shape asymmetry remains open)
- `plans/PLAN-ISSUE-020-linux-sandbox-gate-in-ci.plan.yml` — "seven-step" rename block;
  this is the eighth silent-failure step found in that recipe
- Discovered by implementer-020 during PLAN-ISSUE-020 Phase 1 fleet migration,
  2026-07-28
