---
title: "`pack add` silently no-ops when the target pack is already declared in the manifest"
schema_version: issue/v1
delivered_by: PLAN-ISSUE-026

issue:
  id: ISSUE-026
  title: "`pack add` silently no-ops when the target pack is already declared in the manifest"
  type: bug
  status: closed
  created: "2026-06-21"
  closed: "2026-07-09"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# `pack add` silently no-ops when the target pack is already declared in the manifest

## Problem

`backstop pack add <local-path>` silently exits 0 with zero output when the derived
pack name is already present in `backstop.yml`, even when the pack is not installed
on disk and the lock entry is diverged or missing. Discovered dogfooding on 2026-06-21.

### Setup

`backstop.yml` declared:

```yaml
packs:
  backstop/go-standards: local
```

`backstop.lock` held a stale entry for `slotly/go-standards` (not `backstop/go-standards`).
`.backstop/packs/` was empty — the pack was not materialized on disk.

### Repro steps

1. Ensure `backstop.yml` declares a local pack (e.g. `backstop/go-standards: local`).
2. Ensure `.backstop/packs/` is empty and the lock is diverged (or missing the entry).
3. Run `backstop pack add ~/src/projects/backstop-go-pack` (the local pack whose derived name is `backstop/go-standards`).
4. Observe: exit 0, **zero output** (even with `--json`), `.backstop/packs/` still empty, lock unchanged.
5. Run `backstop pack remove backstop/go-standards`, then repeat step 3.
6. Observe: `Added backstop/go-standards@ ...`, `.backstop/packs/backstop/go-standards/` populated, lock updated.

### Defect

`pack add` checks whether the pack name exists in `backstop.yml` and, if so, returns
early without installing, without updating the lock, and without any output. This
conflates "already declared" with "already installed and up to date" — a false
equivalence. A pack can be declared but uninstalled (as above) or declared with a
diverged lock. In both cases the correct behavior is to install (or error loudly),
never to silently no-op.

### Adjacent defect — empty version string on local-pack success

When `pack add` does succeed for a local pack (after the remove workaround above), the
success line shows an empty version field:

```
Added backstop/go-standards@
```

Local packs have no semver, but the trailing `@` with nothing after it indicates the
version slot is empty rather than elided. The success line should either omit the `@`
for local-source packs or render a meaningful token (e.g. `@local`).

### Impact

The silent no-op means a developer who renames a pack namespace, clears `.backstop/packs/`,
and re-runs `pack add` sees a green exit — but the pack is not on disk and the gate
runs with no rules. Because there is no error and no output, the misconfiguration is
invisible.

## Solution

**Primary fix:** in `pack add`, after resolving the pack name from the provided path,
check whether the pack is already **installed and current** (disk artifact exists +
lock entry matches manifest), not merely whether the name appears in `backstop.yml`.
If the pack is declared but not installed or the lock is diverged, proceed with the
install-and-lock step. If the pack is genuinely already installed and current, a brief
`Pack backstop/go-standards is already installed` message is acceptable — silence is not.

**Secondary fix:** for local-source packs, suppress the trailing `@` on the success
line or substitute a meaningful token (`local`, `path:<resolved-path>`, or similar).
This is a display-only change with no behavior impact.

## Resolution

pack add now distinguishes DECLARED from INSTALLED (checks `.backstop/packs/<name>/`
on disk + a consistent lock entry, not just `backstop.yml` membership): a
declared-but-absent or lock-diverged pack now installs via the 025 materialize
pipeline instead of silently no-op'ing; a genuinely-current pack reports an honest
"already installed and up to date"; and the bare trailing `@` on versionless
local-pack success lines is gone. Fixed in 109504e.

## References

- `backstop.yml` — `packs` stanza (source of the already-declared check that triggers the no-op)
- `backstop.lock` — install state that should gate the already-installed check, not the manifest
- `.backstop/packs/` — install target; confirmed empty after repro
- ISSUE-025 — sibling: `pack install` follows stale lock and materializes nothing; same pack lifecycle, adjacent code path
