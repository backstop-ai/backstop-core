---
title: "Pack CLI Authoring & Distribution Hardening"
number: DIR-017
created: "2026-07-08"
schema_version: directive/v1

directive:
  status: active
  source:
    - "ISSUE-032"
    - "ISSUE-030"
    - "ISSUE-025"
    - "ISSUE-026"
---

## Description

Packs are backstop's substrate: every check and every language's toolchain comes
from a pack — core stays a thin executor with zero baked language/tool knowledge.
That guarantee is only as good as the CLI loop authors and consumers actually use
to make and move packs. Today that loop is stale: `pack new`/`check`/`test`/`list`
were built pre-engine-model and never modernized after BUNDLE-011's engine-dispatch
cutover, and `pack install`/`pack add` have correctness defects that let a pack
silently fail to land. This directive owns hardening both halves of that loop so
the packs-only model is actually usable end to end, not just consumable once a
pack somehow already exists on disk.

**AUTHORING (ISSUE-032, ISSUE-030):** `pack new` scaffolds dead `.standard.md` /
`.recipe.md` artifacts instead of a valid engine pack (`pack.yml` + `engines:`
block); `pack check`/`pack test` reject production engine packs via stale
layer/claims-phase logic left over from the pre-engine model; `--format text` is
silently ignored; `pack list` reports `RULES:0` because it looks for
`content.ruleset.rules`, a shape engine packs don't use; `artifact new bundle`
stamps the wrong `bundle/v1` id; there is no clean way to relock a local pack.
ISSUE-030 eradicates the vestigial native-standards `.standard.md` scaffolder and
records the standards-compiler lineage — it is the same scaffolder surface as
ISSUE-032's Defect A, so the two are delivered together (032's plan handles the
delete+replace; 030 closes as resolved by that work).

**DISTRIBUTION (ISSUE-025, ISSUE-026):** `pack install` reports success but
materializes nothing for local-source packs, and follows the stale lock file
instead of the declared manifest — a pack can be "installed" and still be absent.
`pack add` silently no-ops when the target pack is already declared in the
manifest, even when it isn't actually on disk or the lock entry has diverged, and
prints an empty version string on the success line it does emit.

Grounded in the 2026-07-08 backlog reconciliation sweep, which re-verified all
four issues OPEN against current code before this directive was opened.

## Acceptance Criteria

- `pack new` scaffolds a valid engine pack (`pack.yml` with an `engines:` block,
  no `.standard.md`/`.recipe.md` artifacts) that passes `pack check`, `pack test`,
  and the gate without hand-editing.
- The vestigial native-standards `.standard.md` scaffolder is gone from the
  codebase; its lineage is recorded, not silently deleted.
- `pack list` reports real rule/check counts for engine packs instead of
  `RULES:0`.
- `pack install` and `pack add` actually materialize local packs on disk and
  reconcile against the declared manifest (not the stale lock); `pack add`
  correctly re-installs or repairs a declared-but-missing pack instead of
  no-opping, and prints a real version on success.
- `backstop gate` and `backstop artifact validate` stay green throughout.

## Notes

ISSUE-032 and ISSUE-030 share one deletion/replacement surface (the
`.standard.md` scaffolder) — plan them as a single unit even though they remain
separate issues; ISSUE-030 is expected to close as delivered-by ISSUE-032's plan
rather than carrying its own implementation.
