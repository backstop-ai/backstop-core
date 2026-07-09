---
title: "`pack install` reports success but materializes nothing and follows the stale lock instead of the declared manifest"
schema_version: issue/v1
delivered_by: PLAN-ISSUE-025

issue:
  id: ISSUE-025
  title: "`pack install` reports success but materializes nothing and follows the stale lock instead of the declared manifest"
  type: bug
  status: closed
  created: "2026-06-21"
  closed: "2026-07-09"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# `pack install` reports success but materializes nothing and follows the stale lock instead of the declared manifest

## Problem

`backstop pack install` has two distinct correctness defects, both discovered by
dogfooding `backstop/go-standards` on 2026-06-21.

### Setup

`backstop.yml` declared:

```yaml
packs:
  backstop/go-standards: local
```

`backstop.lock` held a **stale, unrelated entry** left over from an earlier rename:

```yaml
packs:
  slotly/go-standards:
    source_type: local
    # no path key
```

### Defect A — install silently materializes nothing for local-source packs

Running `backstop pack install` exited 0 and printed:

```
Installed 1 packs
  slotly/go-standards
```

`.backstop/packs/` remained **empty** after the command completed. No pack directory
was created; no files were copied. The command reported a successful install that
performed no disk work.

### Defect B — install follows the lock, not the manifest, with no warning

The command installed `slotly/go-standards` (the stale lock entry) rather than
`backstop/go-standards` (the pack declared in `backstop.yml`). There was no warning
that the lock diverges from the manifest, no error, and no cleanup of the stale
`slotly/go-standards` lock entry.

The lock is supposed to pin resolved coordinates for what the manifest declares; it
must not override the manifest's declared pack set. When lock and manifest diverge,
`pack install` must treat the divergence as an error (or at minimum a loud warning
requiring explicit reconcile), not silently install the wrong pack.

### Combined impact

The two defects compound: a user running `backstop pack install` after any pack rename
or namespace change sees a green exit with a plausible-looking output line, but ends
up with an empty `.backstop/packs/` and their gate consuming no pack rules. The failure
is invisible until the gate runs with zero findings — which could look like a green
pass rather than a misconfiguration.

### Repro steps

1. Set `backstop.yml` to declare a local pack under one name (e.g. `backstop/go-standards: local`).
2. Let `backstop.lock` hold a stale entry under a different name (e.g. `slotly/go-standards`) with `source_type: local` and no `path` key.
3. Run `backstop pack install`.
4. Observe: exit 0, output names the stale lock entry, `.backstop/packs/` is empty.

## Solution

Two independent fixes, either order:

**Fix A — materialize local-source packs on install.** When a lock entry has
`source_type: local`, the install path must resolve and copy (or symlink) the pack
directory to `.backstop/packs/<name>/`. A local-source install that produces no disk
artifact must be a hard error, not a silent success.

**Fix B — reconcile lock against manifest before installing.** On `pack install`,
diff the lock's declared pack set against `backstop.yml`'s declared packs. Any entry
present in the lock but absent from the manifest is stale and must be called out:
either error and refuse to proceed, or auto-prune with a loud `Removed stale lock
entry: slotly/go-standards` line. Any pack declared in the manifest but absent from
the lock must trigger resolution (write a new lock entry) rather than being silently
skipped. The lock must converge to exactly what the manifest declares, or install must
fail explicitly.

## Resolution

pack install now materializes local-source packs onto disk (`.backstop/packs/<name>/`)
via a recursive copy resolved from a new portable relative `LockEntry.local_path`
(recorded by `pack add`), and reconciles the DECLARED `backstop.yml` manifest against
the lock — warning on stale/diverged lock entries and failing loud on an unresolvable
local source instead of printing vacuous success. Fixed in 83fb200.

## References

- `backstop.yml` — `packs` stanza (declares the canonical pack set)
- `backstop.lock` — stale `slotly/go-standards` entry (source of the divergence)
- `.backstop/packs/` — install target directory (empty after the repro)
- ISSUE-026 — sibling: `pack add` silent no-op on already-declared pack; same pack lifecycle, adjacent code path
