---
title: "Registry Era Pack Declared Detection"
schema_version: issue/v1

issue:
  id: ISSUE-058
  title: "Registry Era Pack Declared Detection"
  type: technical-debt
  status: open
  created: "2026-07-14"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: safe
---

# ISSUE-058: Registry-Era Pack-Declared Detection

**DEFERRED — registry-era / future work, not near-term.** This issue is
explicitly gated on a capability (a pack registry/index) that does not exist
yet. Do not pick this up ahead of that dependency landing.

## Problem

BUNDLE-003 resolved (OQ-5, RESOLVED/DISSOLVED, 2026-07-13) that day-zero
`backstop init` does **ZERO language detection** — languages enter only via
explicit `pack add`, never by init inspecting the repo. That resolution is
final for how init behaves today; it deliberately dissolved the "which
language does init detect" question rather than answering it (DD-11 / DD-13).

A future "registry-era" upgrade — init (or an equivalent command)
**auto-detecting** languages present in a repo and **offering** matching
packs — is a real, desirable capability, but it hits a genuine catch-22 as
currently scoped:

- Detecting a language without baking `go.mod → Go`, `package.json → Node`,
  etc. into core requires each **pack** to declare its own activation
  signals (the globs/files that mean "this pack is relevant here") — per
  DD-13, detection knowledge must live in packs as data, never in core/CLI.
- But reading pack-declared signals requires those packs to already be
  **installed** to read their manifests — which requires already knowing
  which language(s) are present, which is the very thing detection was
  supposed to determine.

The only clean break out of that loop is a pack **registry/index** — a
catalog of available packs and their declared triggers that init can consult
**before** any pack is installed. Today, **neither piece exists**:

- No pack-manifest `detect:` (or `activation:`) field — a `grep` for
  `detect|activation|trigger` across `pkg/pack/manifest.go` finds nothing.
- No pack registry/index of any kind — `pkg/pack/manifest.go`,
  `pkg/pack/scaffold.go`, and `pkg/pack/validate_manifest.go` are the only
  files referencing "registry" in `pkg/pack`, and none of them implement a
  catalog; packs today are found by explicit git ref or local path only
  (`pkg/pack/distribution/add.go`), never discovered.

## Solution

Deferred — not to be designed in detail until the dependency below exists.
Direction, per BUNDLE-003 OQ-5's registry-era note:

1. **(a) A pack-manifest `detect:` field** declaring activation signals
   (globs / filenames — e.g. `go.mod`, `package.json`, `Cargo.toml`) that a
   pack ships as its own data, analogous to how `packs/*/pack.yml` already
   declares `input_mode` / commands / other pack-owned facts. This field by
   itself does not solve the catch-22 — it only removes the "where would the
   signal live" half of the problem.
2. **(b) A pack registry/index** that `init` (or a new `pack suggest` /
   `pack discover`-shaped command) consults **before** installing anything,
   to offer matching packs for signals present in the repo. This is the
   piece that actually breaks the catch-22 — it lets detection happen
   without any pack being installed first.
3. Both pieces must hold the thin-executor invariant (DD-13, CLAUDE.md "What
   backstop IS"): the language-to-pack mapping is data the registry/pack
   carries, never a lookup table or `switch` on language name baked into
   core/CLI. `backstop/self` is the mechanical enforcement that would catch
   a regression here.
4. This issue is explicitly **gated on the pack registry existing** — it
   cannot be implemented, even partially, until that infrastructure is
   designed and built. It relates to (but does not replace) the
   pack-distribution work in BUNDLE-001 / BUNDLE-002.
5. When unblocked, the eventual plan should also revisit whether `init`
   itself is the right place to consult the registry, or whether — per
   BUNDLE-003 DD-11 ("init's job is to add the backstop layer, not the
   project") — detection-and-offer belongs in `pack add` / `doctor` instead,
   keeping `init` itself detection-free even in the registry era.

## References

- BUNDLE-003 OQ-5 (RESOLVED/DISSOLVED, 2026-07-13) — "init does ZERO language
  detection… Registry-era auto-detect is out of scope / future… Supersedes
  the former DD-5; codified as DD-11 / DD-13"
- BUNDLE-003 DD-13 (HARD INVARIANT) — "Detection, framework recognition, and
  CI-platform knowledge live in PACKS as data, NEVER in core/CLI. Core
  dispatches; packs know… `backstop/self` enforces it."
- BUNDLE-003 §"Out of Scope / Dependencies" — "Pack registry + pack-declared
  `detect:` field — registry-era auto-detection; deferred, relates to pack
  distribution (BUNDLE-001 / BUNDLE-002). Explicitly NOT how languages enter
  in this bundle (OQ-5 dissolved detection)."
- BUNDLE-001 / BUNDLE-002 (pack distribution) — the adjacent bundles this
  issue's registry piece relates to but does not belong inside
- `pkg/pack/manifest.go`, `pkg/pack/distribution/add.go` — confirmed today's
  pack-discovery surface: explicit git ref or local path only, no `detect:`
  field, no registry/catalog
- `backstop/self` pack — the mechanical no-baked-language enforcement this
  issue's eventual implementation must stay inside
  (`project_backstop_self_pack` in agent memory)
- CLAUDE.md "What backstop IS" — thin-executor first principle: "a baked Go
  path AND a baked TypeScript path are BOTH violations… New language = a new
  pack, never new CLI code"
