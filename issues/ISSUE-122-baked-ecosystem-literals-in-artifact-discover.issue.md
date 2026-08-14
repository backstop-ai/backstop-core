---
title: "Baked Ecosystem Literals In Artifact Discover"
schema_version: issue/v1

issue:
  id: ISSUE-122
  title: "Baked Ecosystem Literals In Artifact Discover"
  type: technical-debt
  status: open
  created: "2026-08-14"

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate
---

# Baked Ecosystem Literals In Artifact Discover

## Problem

`DiscoverArtifacts` in `cmd/backstop/artifact_discover.go:47-49` skips certain directory names
during its artifact-discovery walk via a hardcoded switch statement:

```go
switch base {
case "testdata", "vendor", "node_modules", ".git", ".backstop", "prototype":
    return filepath.SkipDir
}
```

Verified directly at HEAD (`cmd/backstop/artifact_discover.go`, `main` branch). Two of the six
entries are ecosystem-specific literals baked directly into core CLI code: `vendor` is a
Go-ecosystem convention (Go modules' vendored-dependency directory); `node_modules` is a
Node/npm-ecosystem convention. The other four (`testdata`, `.git`, `.backstop`, `prototype`) are
generic, tool-agnostic conventions unrelated to any specific language or ecosystem and are not in
scope for this issue — they should stay as-is.

### Why this is the named defect, not a stretch

CLAUDE.md's first principle states this exact bug shape explicitly: "backstop bakes in ZERO
language/tool-specific checks, routing, defaults, or toolchain... a baked Go path AND a baked
TypeScript path are BOTH violations" and "a language, framework, or platform name appearing in
core CLI code IS the bug." This repo's own DD-13 (zero-baked-language invariant, established in
`bundles/BUNDLE-003-onboarding-experience.bundle.md`) states the same rule at bundle level.
`vendor` and `node_modules` are exactly that: a Go-specific and a Node-specific literal, each
sitting in `cmd/backstop/`, core CLI code with no pack indirection.

### Distinct from BUNDLE-003's DD-7/REQ-005 (checked — not a duplicate)

BUNDLE-003 already surfaced and corrected a parallel-shaped defect: DD-7's original canonical
`.gitignore` content baked `node_modules/`, `coverage/`, and TypeScript-specific
`.backstop/ts-*` paths into `init`'s gitignore-emission logic. That was corrected 2026-08-12
(v0.10.0) — the canonical list is now limited to backstop-owned entries plus whatever installed
packs declare via `stdout_artifact`, with the struck TypeScript paths demoted to an illustrative
example (see REQ-005 v1.1.0, DD-7 correction note).

That fix does not cover this defect. DD-7/REQ-005 govern what `init` WRITES into a consuming
repo's `.gitignore`. This issue is about what the artifact-discovery WALK SKIPS when scanning an
existing repo for spec/plan/issue/bundle/etc. files (`DiscoverArtifacts`, used by `artifact
validate` and related discovery paths) — a different mechanism, different file, different
call graph, with no shared code path. Confirmed by reading both: BUNDLE-003's corrected surface
lives in gitignore-content generation; this defect lives in `filepath.Walk`'s `SkipDir` decision.
No open issue or bundle currently owns `artifact_discover.go`'s skip-list (checked via grep across
`issues/`, `specs/`, `plans/`, and `bundles/` for `artifact_discover`, `DiscoverArtifacts`,
`SkipDir`, `node_modules`, and `vendor` — no other artifact references this code path).

### Regression risk if fixed by deletion instead of migration

Simply removing the two literals (rather than replacing them with pack-sourced data) would let
`DiscoverArtifacts` start walking into `node_modules/` and `vendor/` trees looking for
artifact-shaped filenames (`*.spec.md`, `*.issue.md`, etc.). A false-positive artifact discovered
inside a dependency tree is a real regression, not a hypothetical — third-party npm/Go packages
can plausibly ship files that collide with backstop's filename patterns. The skip behavior itself
is correct and should be preserved; only its ecosystem-specific SOURCE needs to move.

### Secondary finding — the `backstop/self` dogfood pack would NOT catch this today

Checked `rules/no-baked.yml` in the `backstop-self-pack` repo (the pack backing `backstop/self`,
installed at `.backstop/packs/backstop-ai/backstop-self`) against this exact code. None of its
six rule families (`no-baked-tool-exec`, `no-baked-tool-command`, `no-baked-language-token`,
`no-language-literal-on-neutral-spine`, `no-baked-repo-layout-classification`,
`no-structural-name-split-on-spine`) would flag `cmd/backstop/artifact_discover.go:47-49`:

- `no-baked-language-token` (Family B2)'s token regex matches file extensions
  (`.ts`/`.py`/etc.) and manifest filenames (`go.mod`, `package.json`, `tsconfig`, ...) — it does
  not include bare directory-convention words like `vendor` or `node_modules`, so a switch-case
  skip-list literal is invisible to it.
- The location-scoped families (B3–B6) are all restricted via `paths.include` to a fixed list of
  spine files (`pkg/gate/*.go`, `cmd/backstop/gate.go`, `cmd/backstop/pack_gate*.go`,
  `pkg/pack/engine/binding.go`, etc.) — `cmd/backstop/artifact_discover.go` is not on that list,
  so even if a matching pattern existed, this file is out of scope for them.

This is a real gap in the dogfood rule's coverage class (skip-list/exclusion literals, distinct
from the call-shape and token-regex classes it already covers), not just an isolated miss on this
one file. Recording it here as a secondary finding — closing it is the self-pack's own follow-on,
not scoped into this issue's fix.

## Solution

Not decided — recording direction, not prescribing the exact mechanism (plan-time work):

Source the ecosystem-specific skip entries (`vendor`, `node_modules`) from pack data instead of
the core literal, following the precedent already established in this codebase for "generated/
excluded paths as pack data, not core literals" — see `pkg/pack/manifest.go`'s `StdoutArtifact`
field on `EngineBinding`, the existing per-engine declaration of what a toolchain pack writes for
the gate to read. An analogous declaration (e.g. a pack-declared list of directory names its
ecosystem conventionally uses for vendored/installed dependencies) would let a Go-toolchain pack
declare `vendor`, a Node-toolchain pack declare `node_modules`, and so on, with core `SkipDir`
behavior driven by the union of whatever installed packs declare — preserving today's skip
behavior without a language literal in `cmd/backstop/`.

The four generic entries (`testdata`, `.git`, `.backstop`, `prototype`) are out of scope and
should remain as core literals — they are not ecosystem-specific.

## References

- `cmd/backstop/artifact_discover.go:47-49` — `DiscoverArtifacts`, the baked switch statement
- CLAUDE.md — first principle, "Thin executor: ZERO baked language/tool knowledge, for ANY
  language"
- `bundles/BUNDLE-003-onboarding-experience.bundle.md` — DD-13 (zero-baked-language invariant,
  established here); DD-7/REQ-005 (the corrected, non-duplicate parallel defect in gitignore
  emission)
- `pkg/pack/manifest.go` — `EngineBinding.StdoutArtifact`, the precedent for pack-declared
  generated/excluded-path data
- `/Users/bmanson/src/projects/backstop-self-pack/rules/no-baked.yml` — the `backstop/self`
  dogfood pack's baked-language rule families; secondary finding on their coverage gap for this
  defect class
