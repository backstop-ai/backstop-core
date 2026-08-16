---
title: "Baked Ecosystem Literals In Artifact Discover"
schema_version: issue/v1

issue:
  id: ISSUE-122
  title: "Baked Ecosystem Literals In Artifact Discover"
  type: technical-debt
  status: closed
  created: "2026-08-14"
  closed: "2026-08-16"

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate

delivered_by: PLAN-ISSUE-122
---

# Baked Ecosystem Literals In Artifact Discover

## Resolution

Delivered by PLAN-ISSUE-122 (status: completed), commit `4dbf64b` (landed inside a shared
four-lane overnight P0 batch alongside three independent defects — ISSUE-112, ISSUE-113,
ISSUE-118 — that happened to share the working tree).

The baked `vendor`/`node_modules` literals are gone from both consumers named in this issue's
Problem section — `DiscoverArtifacts` and `FindUngatedArtifacts` — and from every other core
call site that fed them. `pkg/artifact`'s shared non-corpus list (`pkg/artifact/layout.go`)
now holds only the three genuinely tool-agnostic names — `.git`, `testdata`, `prototype` — as
this issue's own Solution section scoped. The two ecosystem nouns arrive instead as
pack-declared data via a new `classification.dependency_dirs` field on `pack.yml`'s existing
`Classification` block (reusing the pack→core path-classification channel rather than
inventing a new one — the closer precedent to `EngineBinding.StdoutArtifact`, per the plan's
own rejection of that alternative). `cmd/backstop`'s `mergeDependencyDirs`
(`cmd/backstop/pack_dependency_dirs.go`) unions the field across the full installed-pack set,
and the merged set is injected into both corpus walks as `artifact.NonCorpusDirs`, threaded
explicitly through a five-hop chain reaching `gate.go`'s traceability surface
(`ValidateConfig.NonCorpus` → `gate.FindUngatedArtifacts` → `collectTraceRefs` →
`computeRequirementTraceabilitySurfaces` → `buildRequirementTraceabilitySteps`) — no call site
defaults or re-derives the set independently.

**Wiring gap closed, not just the merge/walk logic.** Earlier task-level unit tests proved the
merge and the two walks in isolation but assembled the exclusion set by hand before calling
`DiscoverArtifacts`/`FindUngatedArtifacts` directly — which would stay green even if a
production call site silently passed the zero value instead of the pack-derived set. Three
wiring tests were added specifically to close that integration gap by driving the real entry
points instead: `TestNonCorpusWiring_RealCommandsHonorPackDeclarationAcrossPackLoadOutcomes`
(a table over `{doctor, artifact validate} x {packs load, packs fail to load}`, run through the
genuine `NewRootCommand().Execute()` harness, not a stub), `TestGateSteps_ExclusionSetWiredIntoArtifactValidation`
(executes `buildGateSteps`'s real step slice end to end and falsifies both of `ValidateAll`'s
independent production sites — the `FindUngatedArtifacts` argument and the
`ValidateConfig.NonCorpus` field — by two different mechanisms), and
`TestCollectTraceRefs_HonorsPackDeclaredDependencyDirs` (proves the traceability hop is
genuinely sensitive to the injected parameter, with an explicit documented residual: no test in
the repo can observe whether `buildGateSteps` hands that hop the pack-derived set or a literal
zero value, since the two are behaviorally indistinguishable downstream — guarded instead by
the explicit-parameter, single-local-threading discipline TASK-008 enforced).

**Predating specs reconciled, not left to drift.** Three implemented specs and one narrative
sharp edge referenced the pre-fix shape and were updated via spec-author (never hand-edited):
SPEC-068 (three contract signatures — `FindUngatedArtifacts`, `DiscoverArtifacts`,
`realArtifactValidator` — plus CLM-062's claim text and Sharp Edge 12, which had knowingly
propagated this exact bake and is now marked RESOLVED, citing this issue), SPEC-043 (all three
sites restating the `Classification` struct's field enumeration, now including
`DependencyDirs`), SPEC-070 (REQ-007's stated mechanism, corrected from "the helper carries the
exclusion set" to "the exclusion set arrives injected from `ctx.Packs`," plus a second drifted
site restating `checkArtifactLayout`'s contract note), and SPEC-069 (Sharp Edge 5's stale
cross-reference to this issue, discharged now that the hand-off it named is complete).

**Ecosystem knowledge lands where it's owned, versioned.** Three toolchain packs shipped
alongside core, each declaring its ecosystem's convention via the new field: go-toolchain
v1.5.0 declares `vendor`, typescript-toolchain v1.3.0 and bun-toolchain v1.3.0 both declare
`node_modules`. `backstop.yml`/`backstop.lock` in this repo pin go-toolchain 1.5.0. Adding the
next ecosystem's vendoring convention is now a pack change with zero core edits — the
acceptance bar the plan's CLM-008 states and falsifies.

The `backstop/self` dogfood pack's blind spot for skip-list/exclusion literals — this issue's
own secondary finding — is explicitly NOT closed by this fix; it remains the self-pack's own
follow-on, as scoped.

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
