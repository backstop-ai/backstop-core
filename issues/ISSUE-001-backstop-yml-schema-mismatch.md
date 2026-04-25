---
title: "backstop.yml packs schema mismatch between config and distribution"
schema_version: issue/v1

issue:
  id: ISSUE-001
  title: "backstop.yml packs schema mismatch between config and distribution"
  type: bug
  status: resolved
  resolved: "2026-04-24"
  created: "2026-04-21"
---

## Problem

`pkg/config/` and `pkg/pack/distribution/` have incompatible YAML schemas
for the `packs:` field in backstop.yml.

**config.Config (used by gate, code check):**
```yaml
packs:
  rules:
    slotly/go-standards: "1.0.0"
  code:
    acme/stripe-go: "2.0.0"
```

**distribution.backstopYml (used by pack add/remove/update/upgrade):**
```yaml
packs:
  - name: slotly/go-standards
    version: "1.0.0"
  - name: acme/stripe-go
    path: ../local-pack
```

`pack add` writes the list-of-objects format. `backstop gate` reads the
map format via `config.LoadConfig`. Result: `pack add` succeeds, then
`backstop gate` fails with "cannot unmarshal !!seq into config.Packs".

## Impact

The pack system is end-to-end broken. A user cannot `pack add` and then
`gate` in the same project — the two commands write and read incompatible
YAML.

Discovered during DIR-009 smoke testing against slotly-go-pack.

## Root Cause

Codex implemented `pkg/pack/distribution/` with its own backstop.yml
parser (`backstopYml` struct with `Packs []backstopYmlPack`) instead of
using `pkg/config/Config`. The impl-reviewer did not catch the
cross-package schema incompatibility because reviews check claims within
a spec, not contracts across specs.

## Fix

One format must win. The list-of-objects format from distribution is
richer (supports `path:` for local packs, `version:` per pack). The
map format from config is simpler but can't represent local paths.

Recommended: update `pkg/config/Config.Packs` to use the list-of-objects
format, then update `loadInstalledPacks` in `cmd/backstop/pack_gate.go`
to read pack names from the new format. The gate and distribution share
one schema.

## References

- SPEC-015 (pack distribution lifecycle)
- SPEC-017 (pack gate integration)
- CAP-001 (pack gate enforcement — UJ-001 fails because of this)
- DIR-009 (end-to-end smoke test — blocked by this)
