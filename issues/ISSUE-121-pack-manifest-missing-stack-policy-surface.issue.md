---
title: "Pack manifest has no stack-policy surface, blocking doctor runtime check"
schema_version: issue/v1

issue:
  id: ISSUE-121
  title: "Pack manifest has no stack-policy surface, blocking doctor runtime check"
  type: technical-debt
  status: open
  created: "2026-08-13"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: safe
---

# Pack manifest has no stack-policy surface, blocking doctor's runtime check

## Problem

BUNDLE-003's doctor spec seed carries REQ-024:

> `backstop doctor` must check the installed runtime/toolchain version against the stack policy
> declared by the installed packs and warn on deviation. The policy values are pack data — no
> runtime version may be baked into core. Evidence: the dogfood machine ran only non-LTS Node
> against a documented Node-LTS stack decision and nothing warned (doctor seed, DD-13).

No surface exists today for a pack to declare that policy. Verified against the tree
(2026-08-13):

- `pkg/pack/manifest.go`'s `Manifest` struct carries `Name`, `Version`, `Language`, `Archetype`,
  `Description`, `Content`, `ToolConfig`, `Engines`, `Classification`, `TestNamePatterns`,
  `Recipes` — no field carries a runtime-version constraint of any kind.
- A real installed pack confirms the same absence on the consumer side:
  `.backstop/packs/*/typescript-toolchain/pack.yml` declares no Node version, LTS or otherwise,
  anywhere in its manifest.
- `pkg/pack/engine/allowlist.go`'s `TrustedToolAllowlist()` is the closest existing thing to a
  version pin, but it only pins backstop-INTRODUCED tools it auto-provisions (`semgrep:
  "1.96.0"`, `ast-grep: "0.43.0"`). Layer-0 runtimes the project does not introduce or provision
  — `node`(implicitly, via `bun`), `bun`, `tsc`, `prettier`, `grep`, `rg` — are all pinned `"*"`:
  presence-only, no version constraint. There is no mechanism anywhere in core or in a pack
  manifest for "this pack requires runtime X at version/range Y."

REQ-024 is therefore unimplementable as written: `backstop doctor` has nothing to read a stack
policy FROM. SPEC-070 (doctor's spec) is carving REQ-024 out rather than speccing a check against
a manifest field that does not exist — this issue is what tracks the actual gap so it is not just
a comment buried in a spec that later reads as silently dropped.

## Why this belongs to BUNDLE-004, not this issue or SPEC-070

This is the same shape of gap BUNDLE-003's REQ-005 hit: a doctor/init requirement pointing at a
pack-manifest field that does not exist. That gap's founder-ruled resolution (2026-08-12, BUNDLE-003
DD-7 correction) was explicit: do not invent the field in the REQUIRING bundle — manifest surface
design belongs to BUNDLE-004 ("Pack Manifest and Authoring," `ready` maturity), and the requiring
bundle states the residue rather than closing it by fiat. REQ-024 gets the same disposition here:
this issue names the gap; the actual shape of a `stack_policy:`-style manifest block (what it
declares, how ranges are expressed, how doctor resolves multiple packs declaring policy for the
same runtime) is BUNDLE-004's design call, not prescribed here.

## Secondary consideration — reading the version is a separate open question

Even once a pack can DECLARE a stack policy, `backstop doctor` still needs to READ the installed
runtime's actual version (e.g. run `node --version` or equivalent) to compare against it. Running
an arbitrary pack-implied command to inspect the host is exactly the ungoverned territory
BUNDLE-021 ("Pack Command Execution Governance," `exploring`) exists to settle — today there is no
unified trust posture for a command like that (see BUNDLE-021's problem statement: `pack-declared
engine commands run with full ambient permissions via `exec.CommandContext`, with no allowlist
reach beyond backstop-provisioned tools). So REQ-024 likely has TWO real dependencies, not one:
BUNDLE-004 for the declaration surface, and BUNDLE-021 (or its resolution) for the sanctioned way
to execute the version-check command itself.

## Related artifacts

- `bundles/BUNDLE-003-onboarding-experience.bundle.md` REQ-024 — the requirement this gap blocks.
- `bundles/BUNDLE-004-pack-manifest-authoring.bundle.md` (`ready` maturity) — likely owner of the
  manifest surface design.
- `bundles/BUNDLE-021-pack-command-execution-governance.bundle.md` (`exploring` maturity) —
  likely owner of the sanctioned-execution half.
- `specs/SPEC-070-backstop-doctor.spec.md` — the spec that carved REQ-024 out pending this gap
  being closed.
- `pkg/pack/manifest.go` — the manifest struct with no stack-policy field.
- `pkg/pack/engine/allowlist.go` — the closest existing (but insufficient) version-pin mechanism.
