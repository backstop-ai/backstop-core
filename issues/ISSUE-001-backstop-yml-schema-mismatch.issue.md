---
title: "backstop.yml packs schema mismatch between config and distribution"
schema_version: issue/v1

issue:
  id: ISSUE-001
  title: "backstop.yml packs schema mismatch between config and distribution"
  type: bug
  status: closed
  created: "2026-04-21"
  closed: "2026-04-24"

verification:
  level: unit
  coverage_threshold: 90
  test_command: "go test ./pkg/config/... ./pkg/pack/distribution/..."

implementation:
  summary: >
    Unify the backstop.yml packs schema on a flat name→version map shared by
    pkg/config and pkg/pack/distribution; migrate config tests, the
    full-backstop.yml fixture, and the backstop-yml JSON schema to the
    unified format.
  package: pkg/config

requirements:
  - id: REQ-001
    text: >
      pkg/config and pkg/pack/distribution must parse the packs field of
      backstop.yml using a single shared schema, so that pack add followed
      by gate succeeds in the same project.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: >
      config.Packs parses the flat name→version map format that pack add
      writes, including local-path packs recorded as version "local".
    tests:
      - TestConfig_Packs_ValidVersions
      - TestConfig_Struct_AllFields

contracts:
  - file: pkg/config/config.go
    provides:
      - name: Packs
        kind: type
        signature: "type Packs map[string]string"
---

# backstop.yml packs schema mismatch between config and distribution

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

## Resolution

Resolved 2026-04-24 in commit f7ba22e. The flat map format won — the
opposite of the recommendation above: both `config.Packs` and
`distribution.backstopYml.Packs` now use `map[string]string`
(name → version), with local packs recorded as version `"local"` instead
of a `path:` key. The migration also updated `pkg/config/config_test.go`,
the `cmd/backstop/testdata/full-backstop.yml` fixture, and
`artifacts/backstop-yml/v1/schema.json`, and fixed two contract-verifier
bugs found along the way (constants filtered out by `token.CONST`,
receiver methods skipped).

## References

- SPEC-015 (pack distribution lifecycle)
- SPEC-017 (pack gate integration)
- CAP-001 (pack gate enforcement — UJ-001 fails because of this)
- DIR-009 (end-to-end smoke test — blocked by this)
