---
title: "SPEC-030: Packs-Only — Native Standards Removal"
number: SPEC-030
created: "2026-06-16"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Rip out the legacy native-standards execution arm so the gate is
    packs-only. Delete the `.standard.md → standards-compiler → manifestDir`
    rule-config path: remove the `manifestDir` `--config` arm from
    `semgrepExecutor` (pkg/check), remove the `ManifestDir` field and its
    callers that feed compiled-standards rules into the semgrep pass, retire
    the `pkg/compile` standards compiler from core, delete the `STD-GO-001`
    source standard and its compiled outputs under `.backstop/rules/`, and
    have backstop-core dogfood-consume the already-published
    `backstop-go-pack` instead. After this spec, locus A (gate-time semgrep
    rule config) has exactly ONE source — installed packs — and no rule
    enforcement is baked into the core binary or compiled from
    `.standard.md` artifacts. This is semgrep-only and independent of
    ast-grep; it is the single-source foundation the engine-dispatch work
    (SPEC-031) builds on.
  package: pkg/check

verification:
  level: integration
  test_command: go test ./pkg/check/ ./cmd/backstop/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      The `semgrepExecutor` (pkg/check/check.go) must no longer carry or
      consume a `manifestDir` field for assembling semgrep `--config`
      arguments. The compiled-standards `--config <manifestDir>` argument
      (the `if e.manifestDir != ""` arm) must be deleted. After this change
      the executor's `--config` set is composed solely from
      `extraSemgrepConfigs` (the pack-supplied rule paths). The struct field
      `manifestDir` is removed, not merely left unset.
    supports: pluggable-pack-engines:REQ-016
  - id: REQ-002
    text: >
      The `ManifestDir` field on `check.Options` (pkg/check/check.go), and
      every assignment that wires the compiled-standards directory into the
      semgrep pass, must be removed in the SAME change as REQ-001 so no caller
      can re-introduce the standards `--config` arm. Specifically: the
      `manifestDir: opts.ManifestDir` assignments in both executor
      construction sites (`goBuiltinExecutors` in check.go and the shared
      semgrep executor in registry.go) are deleted, and the
      `ManifestDir: filepath.Join(...,"rules")` assignments in the gate
      (cmd/backstop/gate.go) and code-check (cmd/backstop/code_check.go)
      Options construction are deleted. Removing `ManifestDir` must not break
      compilation: any remaining reader of `Options.ManifestDir` is updated or
      removed.
    supports: pluggable-pack-engines:REQ-016
  - id: REQ-003
    text: >
      File-type routing must survive the removal of the compiled-standards
      manifest. Routing (`Manifest.RouteFile`) must no longer depend on
      compiled `STD-*.manifest.json` files emitted by the standards compiler.
      With no standards manifest present in `.backstop/rules/`, `LoadManifest`
      must yield the built-in default manifest (`defaultManifest()`) so files
      are still routed to the lint/build/test/semgrep passes. Routing behavior
      for a project with zero compiled standards manifests must be identical
      before and after this spec.
    supports: pluggable-pack-engines:REQ-016
  - id: REQ-004
    text: >
      Locus A (gate-time semgrep rule config) must collapse to a single
      source: installed packs. After this spec there is no second rule source
      feeding the semgrep pass — the `manifestDir` (project compiled
      standards) source is gone and only `extraSemgrepConfigs` derived from
      `mergePackRules` remains. A gate or code-check run with zero installed
      packs must invoke semgrep with zero rule `--config` paths (semgrep runs
      against no project-standards rules), not against a compiled-standards
      directory.
    supports: pluggable-pack-engines:REQ-016
  - id: REQ-005
    text: >
      The `pkg/compile` standards compiler must be retired from core: it must
      not be reachable from any `cmd/backstop` command or any runtime gate /
      code-check path. No non-test production code in `cmd/backstop` or
      `pkg/check` may import `github.com/bmanson/backstop-core/pkg/compile`.
      The compiled-standards artifacts under `.backstop/rules/`
      (`STD-GO-001.manifest.json`, `STD-GO-001.native.json`,
      `STD-GO-001.semgrep.yml`) must be deleted from the repository so the
      dogfood gate no longer enforces compiler-emitted rules.
    supports: pluggable-pack-engines:REQ-016
  - id: REQ-006
    text: >
      The `STD-GO-001` source standard
      (`standards/go/STD-GO-001-go-code-standards.standard.md`) must be
      dropped, since its enforcement content now lives in the published
      `backstop-go-pack`. No production code path may require the `STD-GO-001`
      standard artifact or its compiled outputs to exist. Validation and gate
      runs must not fail merely because `STD-GO-001` is absent.
    supports: pluggable-pack-engines:REQ-016
  - id: REQ-007
    text: >
      backstop-core must dogfood-consume the existing `backstop-go-pack`:
      the repository's own `backstop.yml` must declare the pack in its
      `packs:` map so the dogfood gate enforces the pack's rules in place of
      the deleted compiled standards. The declaration must resolve through the
      existing pack-loading path (`loadInstalledPacks` →
      `.backstop/packs/<pack>/`) with a matching `backstop.lock` entry so
      `VerifyLock` passes. This re-routes backstop-core's own enforcement from
      compiled standards to a consumed pack without adding any new loading
      machinery.
    supports: pluggable-pack-engines:REQ-016
  - id: REQ-008
    text: >
      Removal of the native-standards arm must be flag-day, not aliased: there
      must be no fallback that silently treats a populated `.backstop/rules/`
      compiled-standards directory as a rule source. A leftover
      `STD-*.semgrep.yml` in `.backstop/rules/` must NOT be picked up as a
      semgrep `--config` path by any code path after this spec. The only
      rule sources are packs.
    supports: pluggable-pack-engines:REQ-016

claims:
  # REQ-001 — semgrepExecutor manifestDir arm deleted
  - id: CLM-001
    requirement: REQ-001
    text: >
      semgrepExecutor with pack-supplied extraSemgrepConfigs and no
      manifestDir assembles --config args containing only the pack paths
      (one --config per pack path, no standards-dir --config).
    tests:
      - TestSemgrepExecutor_ConfigArgsFromPacksOnly
  - id: CLM-002
    requirement: REQ-001
    text: >
      semgrepExecutor with zero extraSemgrepConfigs assembles semgrep args
      with no --config flag at all (the deleted manifestDir arm leaves no
      residual standards --config).
    tests:
      - TestSemgrepExecutor_NoConfigWhenNoPacks
  - id: CLM-003
    requirement: REQ-001
    text: >
      The semgrepExecutor struct has no manifestDir field — a compile-time
      assertion via construction in test confirms the field is absent and
      only runner/ensurer/backstopDir/pinnedVersion/extraSemgrepConfigs
      remain.
    tests:
      - TestSemgrepExecutor_StructHasNoManifestDir

  # REQ-002 — Options.ManifestDir removed, callers updated
  - id: CLM-004
    requirement: REQ-002
    text: >
      check.Options has no ManifestDir field; a test constructing Options
      with the post-removal field set confirms ManifestDir is gone and the
      package still compiles.
    tests:
      - TestOptions_NoManifestDirField
  - id: CLM-005
    requirement: REQ-002
    text: >
      goBuiltinExecutors and the registry shared-semgrep construction build a
      semgrepExecutor without referencing any compiled-standards directory.
    tests:
      - TestBuildExecutors_SemgrepHasNoManifestDir
  - id: CLM-006
    requirement: REQ-002
    text: >
      The gate Options construction (buildGateSteps path) produces Options
      with no compiled-standards manifest directory wired in.
    tests:
      - TestGateOptions_NoManifestDir
  - id: CLM-007
    requirement: REQ-002
    text: >
      The code-check Options construction produces Options with no
      compiled-standards manifest directory wired in.
    tests:
      - TestCodeCheckOptions_NoManifestDir

  # REQ-003 — routing falls back to default manifest
  - id: CLM-008
    requirement: REQ-003
    text: >
      With an empty .backstop/rules/ (no .manifest.json), LoadManifest
      returns the built-in default manifest and RouteFile routes a .go file
      to the expected default check types.
    tests:
      - TestRouting_DefaultManifestWhenNoStandards
  - id: CLM-009
    requirement: REQ-003
    text: >
      Routing of a representative source file is identical with the
      standards-compiler manifest removed versus the historical built-in
      default — no pass is dropped from the route table.
    tests:
      - TestRouting_UnchangedAfterStandardsRemoval

  # REQ-004 — locus A single source
  - id: CLM-010
    requirement: REQ-004
    text: >
      A code-check run with zero installed packs invokes semgrep with zero
      rule --config paths (the recorded runner args contain no --config).
    tests:
      - TestCodeCheck_NoPacks_NoSemgrepConfig
  - id: CLM-011
    requirement: REQ-004
    text: >
      A code-check run with one installed pack invokes semgrep with exactly
      the pack's rule paths as --config and nothing from a standards
      directory.
    tests:
      - TestCodeCheck_PackOnly_SemgrepConfigIsPackPathsOnly

  # REQ-005 — pkg/compile retired, compiled artifacts deleted
  - id: CLM-012
    requirement: REQ-005
    text: >
      No production (non-test) file under cmd/backstop or pkg/check imports
      github.com/bmanson/backstop-core/pkg/compile.
    tests:
      - TestNoProductionImportOfCompile
  - id: CLM-013
    requirement: REQ-005
    text: >
      The compiled-standards artifacts (STD-GO-001.manifest.json,
      STD-GO-001.native.json, STD-GO-001.semgrep.yml) are absent from
      .backstop/rules/ in the repository tree.
    tests:
      - TestCompiledStandardsArtifactsAbsent

  # REQ-006 — STD-GO-001 source dropped, not required
  - id: CLM-014
    requirement: REQ-006
    text: >
      The STD-GO-001 source standard file is absent from standards/go/.
    tests:
      - TestStdGo001SourceAbsent
  - id: CLM-015
    requirement: REQ-006
    text: >
      A gate / code-check run succeeds (no config error, no missing-standard
      error) on a project with no STD-GO-001 artifact and no compiled
      standards directory.
    tests:
      - TestGate_SucceedsWithoutStandards

  # REQ-007 — dogfood-consume backstop-go-pack
  - id: CLM-016
    requirement: REQ-007
    text: >
      backstop-core's own backstop.yml declares backstop-go-pack in its
      packs map, and the declared pack resolves through loadInstalledPacks
      to a parseable manifest.
    tests:
      - TestDogfood_BackstopYmlDeclaresGoPack
  - id: CLM-017
    requirement: REQ-007
    text: >
      The dogfood pack declaration has a matching backstop.lock entry so
      VerifyLock passes for the declared backstop-go-pack.
    tests:
      - TestDogfood_GoPackLockVerifies

  # REQ-008 — flag-day, no alias / fallback
  - id: CLM-018
    requirement: REQ-008
    text: >
      A leftover STD-*.semgrep.yml planted in .backstop/rules/ is NOT
      collected as a semgrep --config path by the code-check / gate path
      (the recorded runner args contain no path under .backstop/rules/).
    tests:
      - TestNoFallback_LeftoverCompiledRulesIgnored
  - id: CLM-019
    requirement: REQ-008
    text: >
      A populated .backstop/rules/ standards directory does not become an
      implicit second rule source: with zero packs, semgrep still runs with
      no --config even when .backstop/rules/ contains files.
    tests:
      - TestNoFallback_PopulatedRulesDirNotASource

contracts:
  - file: pkg/check/check.go
    provides:
      - name: Options
        kind: type
        signature: "type Options struct"
        notes: >
          ManifestDir field removed. Remaining fields: Mode, FilePath,
          BackstopDir, PinnedSemgrepVersion, Timeout, ProjectDir,
          GolangciLintAvailable, ExtraSemgrepConfigs, Language, Config, Files.
      - name: semgrepExecutor
        kind: type
        signature: "type semgrepExecutor struct"
        notes: >
          manifestDir field removed. Remaining fields: runner, ensurer,
          backstopDir, pinnedVersion, extraSemgrepConfigs. Execute assembles
          --config solely from extraSemgrepConfigs.
    consumes:
      - source: pkg/check/manifest.go
        name: LoadManifest
        kind: function
      - source: pkg/check/manifest.go
        name: defaultManifest
        kind: function
  - file: cmd/backstop/gate.go
    provides:
      - name: buildGateSteps
        kind: function
        signature: "func buildGateSteps(projectRoot string, scope ...*gate.GateScope) []gate.StepFunc"
        notes: >
          Options construction no longer sets ManifestDir; semgrep rule config
          comes only from extraSemgrepConfigs via mergePackRules.
    consumes:
      - source: cmd/backstop/pack_gate.go
        name: loadInstalledPacks
        kind: function
      - source: cmd/backstop/pack_gate.go
        name: mergePackRules
        kind: function
  - file: cmd/backstop/code_check.go
    provides: []
    consumes:
      - source: pkg/check/check.go
        name: Options
        kind: type
      - source: cmd/backstop/pack_gate.go
        name: mergePackRules
        kind: function
---

# SPEC-030: Packs-Only — Native Standards Removal

## Overview

This spec implements **Spec Seed 1 (Pillar 2)** of BUNDLE-010: rip out the
legacy native-standards execution arm so the gate is **packs-only**.

Today, gate-time semgrep enforcement (locus A) draws rules from **two**
sources merged into one semgrep invocation:

1. **Project compiled standards** — `.standard.md` artifacts compiled by
   `pkg/compile` into `.backstop/rules/STD-*.semgrep.yml`, wired into the
   semgrep pass via `Options.ManifestDir` → `semgrepExecutor.manifestDir` →
   `--config <manifestDir>`.
2. **Installed packs** — each pack's rule files collected by `mergePackRules`
   into `ExtraSemgrepConfigs` → `--config <path>` (one per rule).

This spec **deletes source (1)**. After it lands, semgrep rule config has a
single source — installed packs — and no enforcement rule content is baked
into the core binary or compiled from `.standard.md` artifacts. backstop-core
itself stops enforcing compiler-emitted `STD-GO-001` rules and instead
**dogfood-consumes the already-published `backstop-go-pack`** (published by
DIR-005), the same way any downstream project would.

**Scope boundary:** This seed is **semgrep-only and independent of ast-grep**.
It does NOT add the `engine` field, the dispatch table, the `convert` step, or
any new engine — that is SPEC-031 (Seed 2). It does NOT touch the fixture-time
`pkg/packval` path — that is SPEC-032 (Seed 3). It does NOT author the
BUNDLE-009 contract seam — that is SPEC-033 (Seed 4). This seed lands **first**
because it collapses locus A's input to a single source, which the engine
dispatch in SPEC-031 then generalizes.

**Why first:** With the project-standards source deleted, locus A's input is a
single source (packs). SPEC-031's "group rules by declared engine" dispatch
then has one input shape to generalize instead of two, and there is no second
semgrep path to keep in sync (the [[feedback_integration_gap]] drift risk).

**Verification Level:** Integration (80% coverage)
**Dependencies:** SPEC-017 (pack gate integration — `loadInstalledPacks`,
`mergePackRules`, `VerifyLock`), SPEC-015 (distribution — lockfile),
DIR-005 (published `backstop-go-pack`).

## Requirements

Requirements and claims are defined in frontmatter.

### What is removed vs. what stays

| Concern | Today | After this spec |
|---|---|---|
| semgrep `--config` from compiled standards (`manifestDir`) | present (`if e.manifestDir != ""`) | **removed** (REQ-001) |
| `Options.ManifestDir` field + caller wiring | present | **removed** (REQ-002) |
| semgrep `--config` from packs (`extraSemgrepConfigs`) | present | **stays — the only source** (REQ-004) |
| File-type routing (`RouteFile`) | from compiled `.manifest.json` or defaults | **default manifest** (REQ-003) |
| `pkg/compile` reachable from core | orphaned at CLI, output checked in | **retired; output deleted** (REQ-005) |
| `STD-GO-001` source standard | present | **dropped** (REQ-006) |
| backstop-core's own enforcement | compiled `STD-GO-001` rules | **consumed `backstop-go-pack`** (REQ-007) |
| leftover `.backstop/rules/*.semgrep.yml` fallback | implicitly picked up | **never a rule source** (REQ-008) |

The pack source (`ExtraSemgrepConfigs` / `mergePackRules`) and the routing
default-manifest fallback are **untouched behavior** that this spec relies on;
only the compiled-standards arm is removed.

## Implementation

The change is a deletion plus a dogfood re-wire, in four coordinated edits.

### Edit 1 — Delete the `manifestDir` arm from `semgrepExecutor` (REQ-001)

In `pkg/check/check.go`:

- Remove the `manifestDir string` field from the `semgrepExecutor` struct.
- In `Execute`, delete the block:
  ```go
  if e.manifestDir != "" {
      args = append(args, "--config", e.manifestDir)
  }
  ```
  The `--config` set is now assembled solely by the loop over
  `e.extraSemgrepConfigs`. When that slice is empty, semgrep is invoked with
  `--json --quiet <files...>` and **no** `--config`.

### Edit 2 — Remove `Options.ManifestDir` and all callers (REQ-002)

- Remove the `ManifestDir string` field from `check.Options`
  (pkg/check/check.go).
- Remove the `manifestDir: opts.ManifestDir` assignment in **both**
  semgrepExecutor construction sites: `goBuiltinExecutors` (pkg/check/check.go)
  and the shared semgrep executor in `pkg/check/registry.go`.
- Remove the `ManifestDir: filepath.Join(..., "rules")` assignment in the gate
  Options construction (`cmd/backstop/gate.go`) and in the code-check Options
  construction (`cmd/backstop/code_check.go`).
- `LoadManifest(opts.ManifestDir)` (check.go) currently reads the routing
  manifest from the same directory. Because `ManifestDir` is removed, replace
  that call with `LoadManifest(filepath.Join(opts.BackstopDir, "rules"))` so
  **routing** still reads `.backstop/rules/` for any legacy routing-schema
  `.manifest.json` — but, per REQ-008, the compiled-standards `*.semgrep.yml`
  files in that directory are no longer a semgrep rule source. (Routing and
  rule-config were always distinct uses of the same directory; this spec keeps
  routing's directory read and removes only the rule-config read.)

### Edit 3 — Retire `pkg/compile` and delete compiled output (REQ-005, REQ-006)

- Confirm and preserve the invariant that no production `cmd/backstop` or
  `pkg/check` code imports `pkg/compile` (it is already unreferenced outside
  its own tests). No new import may be added.
- Delete the compiled-standards artifacts checked into the repository:
  `.backstop/rules/STD-GO-001.manifest.json`,
  `.backstop/rules/STD-GO-001.native.json`,
  `.backstop/rules/STD-GO-001.semgrep.yml`.
- Delete the source standard
  `standards/go/STD-GO-001-go-code-standards.standard.md` and its now-orphaned
  rule tree under `standards/go/rules/` that existed only to feed the compiler.
- `pkg/compile` itself may remain in the tree as dead code OR be deleted; this
  spec requires only that it is **unreachable from any runtime path** and from
  any `cmd/backstop`/`pkg/check` production import (CLM-012). Deleting the
  package is the cleaner outcome but is not separately gated here.

### Edit 4 — Dogfood-consume `backstop-go-pack` (REQ-007)

- Add `backstop-go-pack` to the repository's own `backstop.yml` `packs:` map
  (ref → version), so backstop-core enforces the pack's rules on its own code
  in place of the deleted `STD-GO-001` compiled rules.
- Install the pack under `.backstop/packs/<pack>/` and record its content hash
  in `backstop.lock` so the existing `VerifyLock` pre-step passes. This uses
  the existing pack-add / lock machinery (SPEC-015/SPEC-017) — no new loading
  code.

### Resulting locus-A flow (single source)

```
backstop gate / code check
  → loadInstalledPacks(projectRoot)            # backstop.yml packs
  → mergePackRules(packs, .backstop/packs)     # → ExtraSemgrepConfigs
  → Options{ ExtraSemgrepConfigs, BackstopDir, ... }   # no ManifestDir
  → semgrepExecutor.Execute:
        args = [--json --quiet]
        args += (--config <p>) for p in extraSemgrepConfigs   # ONLY source
        args += files
  → routing: LoadManifest(.backstop/rules) → defaultManifest() when empty
```

With zero packs, `ExtraSemgrepConfigs` is empty and semgrep runs with no
`--config` (REQ-004 / CLM-002 / CLM-010).

## Verification

Verification is defined in frontmatter. Integration-level testing at 80%
coverage over `pkg/check/` and `cmd/backstop/`.

Tests use a fake `CommandRunner` to capture the exact `semgrep` argv (asserting
the `--config` set), temporary project directories with/without installed packs
and with/without a populated `.backstop/rules/` directory, and a repository
self-check that the dogfood `backstop.yml` declares `backstop-go-pack` with a
verifying lock entry. The compiled-artifact-absence and source-standard-absence
claims (CLM-013, CLM-014) assert on the repository tree itself.

## Sharp Edges

1. **`.backstop/rules/` is used for two distinct purposes — only one is being
   removed.** The directory feeds (a) **routing** via `LoadManifest` →
   `RouteFile`, and (b) **semgrep rule config** via `manifestDir` → `--config`.
   This spec removes ONLY (b). If an implementer conflates them and also strips
   the routing read, every file routes to nothing and the gate goes vacuously
   green — the exact failure mode `hasRoutableRule`/`defaultManifest` exist to
   prevent. REQ-003 + CLM-008/009 pin routing as unchanged.

2. **The standards compiler output is checked into the repo, but the compiler
   is already CLI-orphaned.** `pkg/compile` has no `cmd/backstop` importer
   today; the live coupling is the **checked-in `.backstop/rules/STD-GO-001.*`
   files** consumed at gate time. Deleting `pkg/compile` source alone would
   change nothing observable; the behavioral change is deleting the compiled
   artifacts AND the `manifestDir` read. An implementer who only deletes the Go
   package has not done the job.

3. **Flag-day removal with no alias (DD-5 principle).** There is deliberately
   NO fallback that re-reads a leftover `STD-*.semgrep.yml` as a `--config`
   path. If such a file lingers in `.backstop/rules/` (stale checkout, partial
   migration), it must be silently ignored as a rule source, not grandfathered.
   CLM-018/019 guard this — a leftover compiled rule file must not resurrect the
   removed arm.

4. **Dogfood lock drift.** Adding `backstop-go-pack` to backstop-core's own
   `backstop.yml` without a matching `backstop.lock` entry will make
   `VerifyLock` fail the dogfood gate (correct, but a foot-gun if the lock is
   forgotten). The pack must be installed and locked in the same change as the
   `backstop.yml` edit. CLM-017 asserts the lock verifies.

5. **Routing regression for non-default projects.** A downstream project that
   relied on a **compiled standards `.manifest.json`** to route an unusual file
   extension (not covered by `defaultManifest`) would lose that route after
   this change. Within backstop-core (the only consumer at N=1) routing is the
   default Go set, so this is safe here; it is called out so SPEC-031 does not
   assume compiled-standards routing exists.

6. **Empty `--config` semgrep invocation.** With zero packs, semgrep now runs
   with no `--config` at all. Verify semgrep treats "no rules" as a clean pass
   (zero findings) rather than an error — the executor must not interpret an
   empty rule set as a failure. CLM-002/010 cover the no-config argv; the
   executor's existing JSON-parse path already yields zero violations on empty
   output.

## Review Questions

1. Does any production code path other than the four edit sites read
   `Options.ManifestDir` or `semgrepExecutor.manifestDir`? Grep must return
   only the deleted sites and tests; a stray reader left behind silently
   re-enables the removed arm.

2. After deletion, does `LoadManifest` still read `.backstop/rules/` for
   *routing* (legacy `.manifest.json`), and does it correctly fall back to
   `defaultManifest()` when the directory holds only the (now-deleted) compiled
   standards or is empty? Confirm the routing read was not collateral-damaged
   by the rule-config removal.

3. Is the dogfood `backstop-go-pack` actually installed under
   `.backstop/packs/` with a `backstop.lock` entry, or only declared in
   `backstop.yml`? A declaration without an installed, locked pack makes the
   dogfood gate fail closed (`VerifyLock` / missing pack dir).

4. Does a gate run on a project with **no** `.backstop/rules/` directory at all
   (not just an empty one) still succeed? `LoadManifest` returns defaults when
   the directory can't be read, but confirm no other code path stats
   `.backstop/rules/` and errors on its absence.

5. Is `pkg/compile` left as dead-but-present code, and if so does any test or
   lint rule (e.g. unused-package / dead-code) flag it in the dogfood gate? If
   deleting it is cleaner, confirm nothing outside its own tests imports it
   first.

## References

- **BUNDLE-010** (pluggable-pack-engines) — DD-6 (two pillars; pillar-2 sized
  small, lands first), DD-3 (locus A collapses to a single source under
  pillar-2), REQ-016 (this seed). [[project_packs_only_no_native_standards]]
- **SPEC-017** — Pack gate integration: `loadInstalledPacks`, `mergePackRules`,
  `VerifyLock`, the pack `--config` path this spec keeps as the sole source.
- **SPEC-015** — Pack distribution: lockfile / `VerifyLock`.
- **SPEC-001** — Standards Compiler (the capability this spec retires from the
  runtime path).
- **DIR-005** — published `backstop-go-pack` (the pack backstop-core now
  dogfood-consumes).
- **SPEC-031** — Seed 2 (engine field + dispatch), the consumer of this
  single-source foundation. This spec does not implement any of it.
- [[feedback_integration_gap]] — two coexisting semgrep rule sources is the
  drift risk this collapse removes.
- [[feedback_loud_not_blocking]] — vacuous-green guard: routing must not be
  silently emptied by the removal (Sharp Edge 1).
- Code: pkg/check/check.go (`semgrepExecutor`, `Options`,
  `goBuiltinExecutors`), pkg/check/registry.go (shared semgrep construction),
  pkg/check/manifest.go (`LoadManifest`, `defaultManifest`, `RouteFile`),
  cmd/backstop/gate.go (Options construction), cmd/backstop/code_check.go
  (Options construction), pkg/compile/ (retired compiler).
