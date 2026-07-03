---
title: "Eradicate the vestigial native-standards scaffolder and record the standards-compiler lineage"
schema_version: issue/v1
number: "030"

issue:
  id: ISSUE-030
  title: "Eradicate the vestigial native-standards scaffolder and record the standards-compiler lineage"
  type: technical-debt
  status: open
  created: "2026-06-24"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# Eradicate the vestigial native-standards scaffolder and record the standards-compiler lineage

## Problem

### Tombstone: the native-standards compiler chain (lineage record)

Backstop once had a first-class "native standards" model for expressing enforcement rules:

1. **Authors wrote `.standard.md` files** — human-readable YAML-fronted markdown under `standards/<language>/`, named `STD-<LANG>-<NNN>-<slug>.standard.md`.
2. **A compiler turned them into `.manifest.json`** — the compiler (now fully absent from the tree) read each `.standard.md` and emitted a `.manifest.json` file into `.backstop/rules/`.
3. **`pkg/check`'s `LoadManifest` consumed the `.manifest.json`** — `pkg/check/manifest.go:191` reads `*.manifest.json` files from the rules directory. If a file carries the compiled-standards schema it calls `isCompiled` (`manifest.go:123`), `deriveRules` (`manifest.go:160`), and `hasSemgrepSignal` (`manifest.go:129`); if it carries only the legacy routing schema it calls `legacyRules` (`manifest.go:143`). The entire `compiledManifestFile` struct (`manifest.go:96`) is the shape of a compiled standards file.

**The strategic shift that killed this model:** on 2026-06-16 backstop went fully packs-only. The `.standard.md` → compiler → `manifestDir` path is legacy — to be removed, not preserved. Everything comes from packs; backstop bakes in no language or tool knowledge. (See CLAUDE.md first principle and the packs-only strategic direction.)

### What has already happened

**ISSUE-018** deleted `pkg/validate/standard.go` — the validator that parsed and validated `.standard.md` artifacts. The deletion assertion in `pkg/validate/deletion_assertion_test.go:48` enforces its permanent absence.

**The compiler is already gone.** No file in the tree writes a `.manifest.json`. The reader in `pkg/check/manifest.go` (`LoadManifest`, lines 191–228) thus always falls through to `defaultManifest()` — the compiled-standards branch is dead-fed. Verified: `grep` over all non-test Go source finds no `*.manifest.json` writer outside `manifest.go` itself (which only reads). The `compiledManifestFile` struct and its methods exist purely as receiver-side dead code.

**The validator router has no `"standard"` entry.** `cmd/backstop/artifact_route.go:17` — `validatorRouter` maps `spec`, `plan`, `adr`, `bundle`, `issue`, `directive`. Running `backstop artifact validate` on any file with `schema_version: standard/v1` returns "unrecognized schema_version prefix". The validator is fully severed.

**BUNDLE-011** owns deletion of the dead reader — `compiledManifestFile`, `isCompiled`, `hasSemgrepSignal`, `legacyRules`, `deriveRules`, and the `.manifest.json` branch of `LoadManifest` in `pkg/check/manifest.go:96–228` — as part of its `pkg/check` cutover. This issue does NOT own that deletion.

### What remains — this issue's actual target

Two functions in `pkg/pack` still author `.standard.md` artifacts that nothing in the system reads or validates:

1. **`pkg/pack/scaffold.go:82–83`** — `scaffoldRulePack` computes `packID := fmt.Sprintf("STD-%s-%03d", langUpper, opts.Number)` and `filename := fmt.Sprintf("%s-%s.standard.md", packID, opts.Slug)`, then writes a fully-formed `.standard.md` file with `schema_version: standard/v1` into `standards/<language>/`.

2. **`pkg/pack/number.go:40–41`** — `ResolvePackNumber` scans `standards/<language>/` for files matching `^STD-<LANG>-(\d{3})-.*\.standard\.md$` to assign the next sequence number. Its sole purpose is numbering the files the scaffolder produces.

The scaffolder also creates the `standards/<language>/` directory (`scaffold.go:90`). The test fixture at `cmd/backstop/testdata/artifacts/standards/SEC-001-valid.standard.md` is a test-only artifact — not a production-authored file, but one that exists because the `artifacts/standard/v1/schema.json` schema and its testdata were never cleaned up after the strategic shift.

**The net result:** `backstop pack new --type rule` silently produces orphan `.standard.md` files whose `schema_version: standard/v1` is unrecognized by `artifact validate`, unread by `LoadManifest`, and unvalidated by any production code path. The scaffolder is producing dead artifacts.

### Confirmation: nothing reads `.standard.md` in production code

A `grep` over all non-test Go source for `.standard.md` returns only:

- `pkg/pack/scaffold.go:83` — the writer
- `pkg/pack/number.go:40–41` — the sequencer that feeds the writer

No reader exists in production code. The `standards/<language>/` directory is a dead-end.

## Solution

Delete `scaffoldRulePack` and `ResolvePackNumber` from `pkg/pack/scaffold.go` and `pkg/pack/number.go`. Remove the `"rule"` case from the `backstop pack new` dispatcher that routes to `scaffoldRulePack`. Remove the `standards/<language>/` directory creation logic.

Clean up the associated artifacts: `artifacts/standard/v1/schema.json`, the `cmd/backstop/testdata/artifacts/standards/` fixture, and any tests that exercise the now-deleted scaffolder path.

Consider adding a deletion assertion (parallel to `pkg/validate/deletion_assertion_test.go:48`) that no `standards/` directory writer exists in production code.

## References

- `pkg/pack/scaffold.go:82–83` — `scaffoldRulePack` computes and writes the orphan `.standard.md` filename; line 90 creates `standards/<language>/`
- `pkg/pack/number.go:40–41` — `ResolvePackNumber` scans `standards/<language>/` to sequence the orphan files
- `pkg/check/manifest.go:96` — `compiledManifestFile` struct (dead reader shape); lines `123,129,143,160,191` — `isCompiled`, `hasSemgrepSignal`, `legacyRules`, `deriveRules`, `LoadManifest` (dead-fed reader, deletion owned by BUNDLE-011)
- `cmd/backstop/artifact_route.go:17` — `validatorRouter` has no `"standard"` entry; `standard/v1` yields "unrecognized schema_version prefix"
- `pkg/validate/deletion_assertion_test.go:48` — deletion assertion confirming `pkg/validate/standard.go` is gone (ISSUE-018)
- `artifacts/standard/v1/schema.json` — schema for a type backstop no longer validates or routes
- `cmd/backstop/testdata/artifacts/standards/SEC-001-valid.standard.md` — sole remaining `.standard.md` in the tree; test fixture, not production artifact
- ISSUE-018 — deleted `pkg/validate/standard.go` (the standards validator)
- BUNDLE-011 — owns deletion of the dead `compiledManifestFile` reader in `pkg/check/manifest.go`
- CLAUDE.md — zero-baked-checks first principle; packs-only strategic direction (2026-06-16)
