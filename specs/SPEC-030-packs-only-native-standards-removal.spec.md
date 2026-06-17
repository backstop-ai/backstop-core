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
    `backstop/go-standards` pack (from the `backstop-go-pack` repository)
    instead. After this spec, locus A (gate-time semgrep
    rule config) has exactly ONE source — installed packs — and no rule
    enforcement is baked into the core binary or compiled from
    `.standard.md` artifacts. backstop-core dogfood-consumes the
    `backstop/go-standards` pack (published from the `backstop-go-pack`
    repository) in its current `layer: 2` form; the `engine:` migration of
    those rules belongs to SPEC-031 (REQ-015), not this spec. This is
    semgrep-only and independent of ast-grep; it is the single-source
    foundation the engine-dispatch work (SPEC-031) builds on.
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
      manifest, and the post-removal routing must be a strict superset (no pass
      dropped) of the pre-removal routing. Routing (`Manifest.RouteFile`) must
      no longer depend on compiled `STD-*.manifest.json` files emitted by the
      standards compiler. With no standards manifest present in
      `.backstop/rules/`, `LoadManifest` must yield the built-in default
      manifest (`defaultManifest()`), whose `routeFileDefaults` routes `.go`
      files to the four passes (lint/build/test/semgrep) and every other file to
      semgrep-only. The post-condition for backstop-core specifically: `.go`
      routing is UNCHANGED (the same four passes the deleted
      `STD-GO-001.manifest.json` `deriveRules` produced), and non-Go files route
      to `[semgrep]` via the default manifest — an ADDITIVE change versus the
      deleted manifest's `deriveRules`, which routed non-`.go` files to no pass
      when its `routableExtensions` was non-empty. No pass that was previously
      assigned to any file is dropped; routing is not asserted to be
      bit-identical, only superset-preserving.
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
      dogfood gate no longer enforces compiler-emitted rules. Because several
      `pkg/compile` tests (`go_standards_integration_test.go` opens
      `standards/go/STD-GO-001-go-code-standards.standard.md` directly, and the
      `go_standards_core/helpers/format/security/std/test_rules` and
      `integration` tests read that same source standard and its
      `standards/go/rules/` tree), deleting the `STD-GO-001` source standard
      (REQ-006) breaks those tests. Therefore the entire `pkg/compile` package
      MUST be deleted in the SAME change — it may NOT be left in the tree as
      dead code, because its tests consume artifacts this spec deletes and would
      turn the dogfood `go test ./...` red. After this spec, `go test ./...`
      across the whole module must be green with `pkg/compile` gone.
    supports: pluggable-pack-engines:REQ-016
  - id: REQ-006
    text: >
      The `STD-GO-001` source standard
      (`standards/go/STD-GO-001-go-code-standards.standard.md`) must be
      dropped, since its enforcement content now lives in the published
      `backstop/go-standards` pack. No production code path may require the `STD-GO-001`
      standard artifact or its compiled outputs to exist. Validation and gate
      runs must not fail merely because `STD-GO-001` is absent.
    supports: pluggable-pack-engines:REQ-016
  - id: REQ-007
    text: >
      backstop-core must dogfood-consume the published `backstop/go-standards`
      pack (the pack whose source repository is `backstop-go-pack`; its
      manifest `name:` is `backstop/go-standards`). The repository's own
      `backstop.yml` must declare the pack in its `packs:` map keyed by the
      manifest name `backstop/go-standards` so the dogfood gate enforces the
      pack's rules in place of the deleted compiled standards. The declaration
      must resolve through the existing pack-loading path (`loadInstalledPacks`
      → `.backstop/packs/backstop/go-standards/` → `pack.ParseManifestFile`),
      which requires the pack to be installed at that path. The repository must
      be reconciled to a consistent dogfood state: (a) `backstop.yml` declares
      `backstop/go-standards`, (b) the pack is installed under
      `.backstop/packs/backstop/go-standards/`, and (c) `backstop.lock`
      contains EXACTLY the matching `backstop/go-standards` entry. The
      pre-existing stale lock entry `slotly/go-standards` (which has no
      declaration and no installed pack) must be REMOVED, not left alongside —
      after this spec the lockfile's declared/locked/installed sets must agree
      so `VerifyLock` passes. backstop-core consumes the pack in its current
      `layer: 2` form (the rules carry `layer: 2`, which `mergePackRules`
      already accepts); the `engine:` migration of those rules is SPEC-031
      (REQ-015) and is NOT in scope here, so dogfood consumption does not
      depend on the pillar-1 engine work landing. This re-routes backstop-core's
      own enforcement from compiled standards to a consumed pack without adding
      any new loading machinery. Enforcement must demonstrably TRANSFER, not
      merely relocate: with `backstop/go-standards` consumed as a pack, a Go
      source file that violates one of the pack's rules (e.g. GO-003
      global-mutable-state via `var $X = ...`, or GO-060 hardcoded-credentials)
      must be FLAGGED by the backstop code-check / gate semgrep pass — it must
      produce at least one semgrep `Violation` whose rule maps to the offending
      pack rule. A green gate after removal is only correct if a known-bad Go
      file is still CAUGHT; the pack-install / lock-verify checks alone (CLM-016,
      CLM-017) prove the pack is present, not that it enforces, and a passing
      gate over clean code cannot distinguish "rules enforced, code clean" from
      "rules silently dropped" (the vacuous-green failure this bundle exists to
      kill).
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
      The semgrepExecutor struct's field set is exactly
      runner/ensurer/backstopDir/pinnedVersion/extraSemgrepConfigs with no
      manifestDir field. The test asserts this with an UNKEYED (positional)
      composite literal constructing the struct with exactly five field
      values, so the test fails to compile if manifestDir (or any other
      field) is re-added or removed — not a mere recompile that a keyed
      literal would silently tolerate.
    tests:
      - TestSemgrepExecutor_StructHasNoManifestDir
  - id: CLM-023
    requirement: REQ-001
    text: >
      No remaining test in pkg/check constructs semgrepExecutor with a
      manifestDir field or asserts a standards-dir (.backstop/rules) --config
      path. The pre-existing
      TestCodeCheck_SemgrepExecutor_RunsProjectAndPackConfigs (which both
      constructed manifestDir and asserted the standards-dir --config) is
      deleted or rewritten to packs-only behavior, and the remaining
      semgrep_executor_test.go cases drop the manifestDir field and the
      standards-dir --config assertion. A source self-check over pkg/check test
      files asserts neither the token "manifestDir:" in a semgrepExecutor
      literal nor a containsConfigFor(..., ".../rules") standards-dir assertion
      survives, so the green go-test guarantee is enforced rather than assumed.
    tests:
      - TestNoTestRequiresManifestDirOrStandardsConfig

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
      The gate Options construction in (*realCodeChecker).runCheck
      (cmd/backstop/gate.go) — the site that today sets
      ManifestDir: filepath.Join(backstopDir, "rules") — produces Options with
      no compiled-standards manifest directory wired in. The test exercises the
      runCheck Options path (not buildGateSteps, which builds StepFuncs and
      constructs no check.Options carrying ManifestDir).
    tests:
      - TestRunCheckOptions_NoManifestDir
  - id: CLM-007
    requirement: REQ-002
    text: >
      The code-check Options construction produces Options with no
      compiled-standards manifest directory wired in.
    tests:
      - TestCodeCheckOptions_NoManifestDir
  - id: CLM-024
    requirement: REQ-002
    text: >
      No remaining test in pkg/check constructs check.Options with a
      ManifestDir field. The 11 keyed `Options{ManifestDir: dir}` literals in
      check_test.go (across TestCodeCheck_RunWith_Integration, _Timeout,
      _FileMode, _AllMode, TestCodeCheck_FileFlag_RoutesByType,
      _SemgrepConfigError, _SemgrepDegradedMode,
      TestCodeCheck_Run_DelegatesToRunWith, _NoBackstopDir,
      _SemgrepDegradedWithNilExecutors) drop the ManifestDir field and re-route
      routing through BackstopDir: each literal that relied on `ManifestDir: dir`
      to exercise routing sets `BackstopDir: dir` instead, so the re-pointed
      LoadManifest(filepath.Join(opts.BackstopDir, "rules")) call resolves to an
      absent/empty `dir/rules` and falls back to the default manifest —
      preserving the .go→four-pass routing those tests exercise. A source
      self-check over pkg/check test files asserts the token "ManifestDir:" does
      not survive in any check.Options literal, so the green go-test guarantee
      (REQ-005, CLM-021) is enforced rather than assumed.
    tests:
      - TestNoTestRequiresOptionsManifestDir

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
      A .go file routes to the same four passes (lint/build/test/semgrep) after
      the standards-compiler manifest is removed as it did via the deleted
      STD-GO-001 manifest's deriveRules — the .go route is unchanged and no pass
      is dropped.
    tests:
      - TestRouting_GoFileUnchangedAfterStandardsRemoval
  - id: CLM-022
    requirement: REQ-003
    text: >
      A non-Go file (e.g. a .md or .yml file) routes to [semgrep] via the
      default manifest after removal — the additive delta versus the deleted
      STD-GO-001 manifest, whose deriveRules routed non-.go files to no pass
      when routableExtensions was non-empty. This pins the actual routing change
      and confirms it only ADDS a pass, never drops one.
    tests:
      - TestRouting_NonGoFileRoutesToSemgrepAfterRemoval
  - id: CLM-020
    requirement: REQ-003
    text: >
      Given a non-empty routing .manifest.json present in
      BackstopDir/rules (.backstop/rules/), LoadManifest after the
      ManifestDir-field removal loads THAT manifest (its custom route entries
      are returned), proving routing still reads the BackstopDir-derived
      directory rather than an empty/wrong dir or always falling back to the
      default. This distinguishes "routing still reads the real directory"
      from CLM-008/009's "falls back to default when empty/absent".
    tests:
      - TestRouting_ReadsBackstopDirManifestWhenPresent

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
  - id: CLM-021
    requirement: REQ-005
    text: >
      The pkg/compile package directory is absent from the repository tree (it
      was deleted, not left as dead code), so no pkg/compile test can open the
      deleted STD-GO-001 source standard, and a full-module `go test ./...`
      compiles and runs green with pkg/compile gone. A repository-tree
      self-check asserts pkg/compile/ does not exist.
    tests:
      - TestPkgCompileDirectoryAbsent

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

  # REQ-007 — dogfood-consume backstop/go-standards pack
  - id: CLM-016
    requirement: REQ-007
    text: >
      backstop-core's own backstop.yml declares the pack keyed
      "backstop/go-standards" in its packs map, the pack is installed under
      .backstop/packs/backstop/go-standards/ with a pack.yml whose name is
      backstop/go-standards, and the declaration resolves through
      loadInstalledPacks to a parseable manifest.
    tests:
      - TestDogfood_BackstopYmlDeclaresGoStandardsPack
  - id: CLM-017
    requirement: REQ-007
    text: >
      backstop.lock contains exactly the matching backstop/go-standards entry
      so VerifyLock passes, and the stale slotly/go-standards entry is absent
      (no declared-but-unlocked and no extra/orphaned lock entry remains).
    tests:
      - TestDogfood_GoStandardsLockVerifies
      - TestDogfood_StaleSlotlyLockEntryRemoved
  - id: CLM-025
    requirement: REQ-007
    text: >
      POSITIVE enforcement-transfer proof: with backstop/go-standards consumed
      as a pack (declared, installed, locked), a code-check / gate run over a
      known-bad Go fixture that violates a pack rule (a global-mutable
      package-level `var $X = ...` for GO-003, or a hardcoded credential such as
      `password := "hunter2"` for GO-060) produces at least one semgrep
      Violation referencing that pack rule. The fixture is a self-contained
      known-bad Go file authored or re-created for this test (NOT obtained by
      deleting the pack's own invalid fixtures), so the test proves enforcement
      actually TRANSFERRED to the consumed pack and a violating file is still
      CAUGHT after native-standards removal — closing the vacuous-green gap that
      CLM-016/017 (pack present) and CLM-015 (gate succeeds on clean code) leave
      open. A negative control accompanies it: the same run over a clean Go file
      (e.g. the pack's go-003 dependency-injection valid form) produces no
      Violation for that rule, so the test fails if semgrep flags everything
      (mis-wired config) as surely as if it flags nothing (dropped rules).
    tests:
      - TestDogfoodPack_FlagsKnownGoViolation

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
      - name: (*realCodeChecker).runCheck
        kind: method
        signature: "func (c *realCodeChecker) runCheck(ctx context.Context, mode check.ScopeMode, files []string) ([]gate.Violation, error)"
        notes: >
          This method is the gate's check.Options construction site (the
          assignment that today sets ManifestDir: filepath.Join(backstopDir,
          "rules")). Its Options construction no longer sets ManifestDir;
          semgrep rule config comes only from ExtraSemgrepConfigs (carried via
          c.extraSemgrepConfigs, populated by mergePackRules in buildGateSteps).
          buildGateSteps itself builds gate StepFuncs and constructs no
          check.Options; it is unchanged by REQ-002 except that it no longer
          supplies a ManifestDir.
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
**dogfood-consumes the already-published `backstop/go-standards` pack**
(published from the `backstop-go-pack` repository by DIR-005), the same way any
downstream project would. The pack is declared in `packs:` and installed under
`.backstop/packs/` keyed by its manifest name `backstop/go-standards`, not the
repository name.

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
| File-type routing (`RouteFile`) | from compiled `.manifest.json` or defaults | **default manifest** — `.go` unchanged (4 passes), non-Go → `[semgrep]` (additive, no pass dropped) (REQ-003) |
| `pkg/compile` reachable from core | orphaned at CLI, output checked in | **retired; output deleted** (REQ-005) |
| `STD-GO-001` source standard | present | **dropped** (REQ-006) |
| backstop-core's own enforcement | compiled `STD-GO-001` rules | **consumed `backstop/go-standards` pack** (REQ-007) |
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
- **Delete or rewrite the existing in-package tests that depend on the removed
  arm.** `pkg/check/semgrep_executor_test.go` currently constructs
  `semgrepExecutor{... manifestDir: "/proj/.backstop/rules" ...}` (lines 42–48,
  112–117, 148–153, 181–186) and asserts
  `containsConfigFor(call.args, "/proj/.backstop/rules")` (lines 66–67, 167–168).
  Those constructions will not compile once the `manifestDir` field is gone, and
  those assertions demand exactly the standards-dir `--config` that REQ-004
  deletes. `TestCodeCheck_SemgrepExecutor_RunsProjectAndPackConfigs` in
  particular must be **deleted or rewritten** to the packs-only behavior (assert
  `--config` is composed solely from `extraSemgrepConfigs`, with no
  standards-dir `--config`); the three other tests
  (`...ToleratesNonJSONPreamble`, `...QuietFlagPassed`,
  `...PreservesPackNamespacedRuleIDs`) must drop the `manifestDir:` field from
  their `semgrepExecutor` literals and any `containsConfigFor(..., ".../rules")`
  assertion. No remaining `pkg/check` test may construct
  `semgrepExecutor.manifestDir` or assert a standards-dir `--config`, or the
  green `go test ./...` guarantee (REQ-005, CLM-021) is defeated by a red build.

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
- **Delete the `ManifestDir` field from the 11 in-package
  `check.Options` literals that depend on it.** Removing the `Options.ManifestDir`
  field makes every keyed `Options{ManifestDir: dir}` literal in
  `pkg/check/check_test.go` fail to compile. There are 11 such literals across 10
  test functions: `TestCodeCheck_RunWith_Integration`, `..._Timeout`,
  `..._FileMode`, `..._AllMode`, `TestCodeCheck_FileFlag_RoutesByType`,
  `..._SemgrepConfigError`, `..._SemgrepDegradedMode`,
  `TestCodeCheck_Run_DelegatesToRunWith`, `..._NoBackstopDir`, and
  `..._SemgrepDegradedWithNilExecutors` (lines ~305, 382, 416, 446, 493, 540,
  595, 659, 890, 1039, 1106). Each literal drops the `ManifestDir:` field and
  **re-routes routing through `BackstopDir`**: because the re-pointed
  `LoadManifest(filepath.Join(opts.BackstopDir, "rules"))` (above) now derives the
  routing directory from `BackstopDir`, every literal that relied on
  `ManifestDir: dir` to exercise routing must instead set `BackstopDir: dir`, so
  `LoadManifest` resolves to an absent/empty `dir/rules` and falls back to the
  default manifest — preserving the `.go`→four-pass routing those tests assert.
  (The literals that already set `BackstopDir: filepath.Join(dir, ".backstop")`
  — `..._SemgrepConfigError`, `..._SemgrepDegradedMode`,
  `..._SemgrepDegradedWithNilExecutors` — keep their existing `BackstopDir` and
  only drop `ManifestDir:`; their `dir/.backstop/rules` is likewise empty and
  falls back to defaults.) No remaining `pkg/check` test may construct
  `check.Options` with a `ManifestDir` field, or the green `go test ./...`
  guarantee (REQ-005, CLM-021) is defeated by a red build (CLM-024). This is the
  symmetric `check_test.go` coupling to Edit 1's `semgrep_executor_test.go`
  enumeration.

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
- **Delete the entire `pkg/compile` package in the same change.** This is NOT
  optional: `pkg/compile`'s own tests consume the artifacts deleted above —
  `go_standards_integration_test.go:13` opens
  `standards/go/STD-GO-001-go-code-standards.standard.md` directly, and the
  `go_standards_core/helpers/format/security/std/test_rules` and `integration`
  tests read that same source standard and its `standards/go/rules/` tree.
  Leaving `pkg/compile` in the tree as dead code would make those tests open a
  file that no longer exists, turning the dogfood `go test ./...` red even
  though no production code imports the package. Deletion of the package and
  deletion of the source standard are therefore coupled and must land together.
  After this edit a full-module `go test ./...` must be green (REQ-005,
  CLM-021).

### Edit 4 — Dogfood-consume `backstop/go-standards` (REQ-007)

The pack's source repository is `backstop-go-pack`, but its manifest `name:`
is **`backstop/go-standards`** (see `pack.yml`). `loadInstalledPacks` keys the
`packs:` map entry directly onto the install path `.backstop/packs/<key>/` and
resolves `pack.yml` there, so the declaration MUST use the manifest name, not
the repo name.

Current repo state to reconcile (do not append — replace):
- `backstop.yml` declares **no** packs (only `project:` / `language:`).
- `.backstop/packs/` does **not** exist on disk.
- `backstop.lock` already contains a stale, now-defunct entry
  `slotly/go-standards` with no declaration and no installed pack.

Steps:

- Add `backstop/go-standards` to the repository's own `backstop.yml` `packs:`
  map (keyed by the manifest name `backstop/go-standards`), so backstop-core
  enforces the pack's rules on its own code in place of the deleted
  `STD-GO-001` compiled rules.
- Install the pack under `.backstop/packs/backstop/go-standards/` (containing
  the pack's `pack.yml` and rule tree) via the existing pack-add machinery.
- Reconcile `backstop.lock`: **remove the stale `slotly/go-standards` entry**
  and record the `backstop/go-standards` content hash so the lockfile's
  declared / installed / locked sets agree EXACTLY and the existing
  `VerifyLock` pre-step passes. There must be no orphaned `slotly/go-standards`
  entry and no declared-but-unlocked pack after this edit.
- This uses the existing pack-add / lock machinery (SPEC-015/SPEC-017) — no new
  loading code. The pack is consumed in its current `layer: 2` form
  (`mergePackRules` already accepts `layer: 2` rules); the `engine:` migration
  of these rules is SPEC-031 (REQ-015), so this edit does not depend on
  pillar-1 landing.

> **Do not over-apply the lock reconciliation.** The stale `slotly/go-standards`
> entry to remove is the **`backstop.lock`** entry (a real lockfile read with no
> declaration and no installed pack). The `slotly/go-standards` string literals
> that appear in `pkg/check` test fixtures
> (`fixtures_test.go:89,109`, `output_test.go:56,74,79`,
> `semgrep_executor_test.go:197,204`) are **synthetic** pack-namespaced
> `check_id` formatting assertions — they are self-contained JSON/assertion
> literals, NOT reads of the deleted lock entry, and are harmless. Leave them
> intact (or rekey to `backstop/go-standards` only if a test author chooses); a
> blanket grep-and-remove of `slotly/go-standards` would wrongly churn unrelated
> test data and is out of scope for this reconciliation.

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
self-check that the dogfood `backstop.yml` declares `backstop/go-standards`,
that the pack is installed under `.backstop/packs/backstop/go-standards/`, that
`backstop.lock` has exactly the matching entry (and no stale
`slotly/go-standards` entry), and that `VerifyLock` passes. The
compiled-artifact-absence and source-standard-absence claims (CLM-013, CLM-014)
assert on the repository tree itself.

The load-bearing positive proof is the **enforcement-transfer** test
(CLM-025 / `TestDogfoodPack_FlagsKnownGoViolation`): with
`backstop/go-standards` consumed as a pack, a code-check / gate run over a
known-bad Go fixture (a package-level mutable `var $X = ...` for GO-003, or a
hardcoded credential for GO-060) must yield at least one semgrep `Violation`
referencing the offending pack rule, while the same run over a clean Go file
yields none. The fixture is authored/re-created for this test — it is NOT
obtained by deleting the pack's own invalid fixtures — so the test proves the
removed compiled-standards enforcement actually TRANSFERRED to the consumed pack
rather than being silently dropped. Without this, the suite would prove only
that the pack INSTALLS and the gate exits GREEN on clean code, which cannot
distinguish "rules enforced" from "rules vanished".

## Sharp Edges

1. **`.backstop/rules/` is used for two distinct purposes — only one is being
   removed.** The directory feeds (a) **routing** via `LoadManifest` →
   `RouteFile`, and (b) **semgrep rule config** via `manifestDir` → `--config`.
   This spec removes ONLY (b). If an implementer conflates them and also strips
   the routing read, every file routes to nothing and the gate goes vacuously
   green — the exact failure mode `hasRoutableRule`/`defaultManifest` exist to
   prevent. REQ-003 + CLM-008/009/022 pin `.go` routing as unchanged and the
   non-Go route as additive (semgrep-only), with no pass dropped.

2. **The standards compiler output is checked into the repo, but the compiler
   is already CLI-orphaned — yet its TESTS read the deleted source standard.**
   `pkg/compile` has no `cmd/backstop` importer today; the live production
   coupling is the **checked-in `.backstop/rules/STD-GO-001.*` files** consumed
   at gate time. The behavioral change is deleting the compiled artifacts AND
   the `manifestDir` read. BUT there is a second, test-time coupling that bites
   in the opposite direction: ~7 `pkg/compile` tests open the
   `standards/go/STD-GO-001-go-code-standards.standard.md` source standard (and
   its `standards/go/rules/` tree) that REQ-006 deletes. So the package cannot
   be "left as harmless dead code" — if its source standard is gone, its tests
   no longer compile/run and `go test ./...` goes red. REQ-005 mandates deleting
   the whole `pkg/compile` package in the same change as the source standard,
   and CLM-021 asserts the package directory is absent. An implementer who
   deletes the artifacts but leaves `pkg/compile` (or who deletes the source
   standard without deleting the package) breaks the dogfood gate.

   There is a SYMMETRIC in-package test coupling in `pkg/check` itself:
   `pkg/check/semgrep_executor_test.go` constructs
   `semgrepExecutor{... manifestDir: "/proj/.backstop/rules" ...}` and asserts
   `containsConfigFor(call.args, "/proj/.backstop/rules")`. Removing the
   `manifestDir` struct field (REQ-001) makes those literals fail to compile,
   and the standards-dir `--config` assertions directly contradict REQ-004.
   Edit 1 mandates deleting/rewriting those tests (notably
   `TestCodeCheck_SemgrepExecutor_RunsProjectAndPackConfigs`) to packs-only
   behavior; CLM-023 enforces that no `pkg/check` test still requires a
   `manifestDir` or a standards-dir `--config`. An implementer who removes the
   field but leaves these tests produces a red build, defeating the
   single-source guarantee exactly as a left-behind `pkg/compile` would.

   There is a THIRD symmetric coupling in `pkg/check/check_test.go` itself, on
   the `Options.ManifestDir` side (REQ-002): 11 keyed `Options{ManifestDir: dir}`
   literals across 10 test functions (lines ~305, 382, 416, 446, 493, 540, 595,
   659, 890, 1039, 1106) stop compiling once the field is gone. Edit 2 mandates
   dropping the `ManifestDir:` field from each and re-routing routing through
   `BackstopDir: dir` so the re-pointed
   `LoadManifest(filepath.Join(opts.BackstopDir, "rules"))` still falls back to
   the default manifest and preserves the `.go`→four-pass routing those tests
   exercise. CLM-024 enforces that no `pkg/check` test still constructs
   `check.Options` with a `ManifestDir` field. An implementer who deletes the
   field but leaves these literals produces the same red build as the
   `semgrep_executor_test.go` case above.

3. **Flag-day removal with no alias (DD-5 principle).** There is deliberately
   NO fallback that re-reads a leftover `STD-*.semgrep.yml` as a `--config`
   path. If such a file lingers in `.backstop/rules/` (stale checkout, partial
   migration), it must be silently ignored as a rule source, not grandfathered.
   CLM-018/019 guard this — a leftover compiled rule file must not resurrect the
   removed arm.

4. **Dogfood lock drift + a pre-existing stale entry.** The repo's
   `backstop.lock` already carries a defunct `slotly/go-standards` entry with
   no declaration and no installed pack, and `.backstop/packs/` does not yet
   exist. The implementer must RECONCILE, not append: remove the stale
   `slotly/go-standards` entry, install the pack under
   `.backstop/packs/backstop/go-standards/`, and lock `backstop/go-standards`
   so the declared/installed/locked sets agree exactly. Declaring
   `backstop/go-standards` without a matching lock entry — or leaving the stale
   `slotly/go-standards` entry behind — will make `VerifyLock` fail the dogfood
   gate. The `backstop.yml` declaration, pack install, and lock edit must land
   in the same change. CLM-017 asserts the lock verifies and the stale entry is
   gone.

5. **Routing change is additive for backstop-core, but could regress unusual
   downstream extensions.** backstop-core today HAS a compiled
   `STD-GO-001.manifest.json`, so its routing comes from `deriveRules`, which
   routes `.go` files to the four passes and non-`.go` files to NO pass (no
   catch-all when `routableExtensions` is non-empty). After removal, the default
   manifest routes `.go` to the same four passes (unchanged) and non-Go files to
   `[semgrep]` (newly added). So routing for backstop-core is NOT bit-identical
   before/after — non-Go files GAIN a semgrep pass — but the change is purely
   additive: no pass is dropped (REQ-003 / CLM-009 / CLM-022). A different
   downstream project that relied on a compiled `.manifest.json` to route an
   unusual extension to a NON-semgrep pass (e.g. lint-only for a custom ext)
   could lose that specific route; this is called out so SPEC-031 does not
   assume compiled-standards routing exists.

6. **Empty `--config` semgrep invocation.** With zero packs, semgrep now runs
   with no `--config` at all. Verify semgrep treats "no rules" as a clean pass
   (zero findings) rather than an error — the executor must not interpret an
   empty rule set as a failure. CLM-002/010 cover the no-config argv; the
   executor's existing JSON-parse path already yields zero violations on empty
   output.

7. **Vacuous green: install ≠ enforce.** The most dangerous failure of this
   spec is the one it exists to prevent — passing all of CLM-016/017 (pack
   declared, installed, locked) and CLM-015 (gate succeeds on clean code) while
   the pack's rules are never actually applied (e.g. `mergePackRules` collects
   the wrong path, the rule files don't reach `--config`, or the executor is
   handed an empty config set). Every one of those checks stays GREEN whether
   enforcement transferred or silently evaporated. The compiled-standards arm
   being deleted in the SAME change means there is no second source to mask the
   gap. CLM-025 / `TestDogfoodPack_FlagsKnownGoViolation` is the only check that
   distinguishes "rules enforced" from "rules dropped": it requires a known-bad
   Go file to still be CAUGHT. Two anti-patterns must be avoided when satisfying
   it: (a) do NOT prove enforcement by *deleting* the pack's own invalid
   fixtures — absence of a fixture proves nothing about the rule engine; use a
   self-contained known-bad Go file the rule targets; and (b) pair the positive
   assertion with a clean-file negative control, or a mis-wired config that
   flags everything would pass just as falsely as a dropped config that flags
   nothing.

## Review Questions

1. Does any production code path other than the four edit sites read
   `Options.ManifestDir` or `semgrepExecutor.manifestDir`? Grep must return
   only the deleted sites and tests; a stray reader left behind silently
   re-enables the removed arm.

2. After deletion, does `LoadManifest` still read the BackstopDir-derived
   `.backstop/rules/` directory for *routing*? Confirm BOTH halves: (a) a
   **non-empty** routing `.manifest.json` in that directory is actually loaded
   (CLM-020) — proving the read was re-pointed at
   `filepath.Join(opts.BackstopDir, "rules")` and not at an empty/wrong dir or
   dropped entirely; and (b) it correctly falls back to `defaultManifest()`
   when the directory holds only the (now-deleted) compiled standards or is
   empty (CLM-008/009). The default-fallback alone would mask a broken path, so
   (a) is the load-bearing check that the routing read was not
   collateral-damaged by the rule-config removal.

3. Is the dogfood pack actually installed under
   `.backstop/packs/backstop/go-standards/` with a matching `backstop.lock`
   entry keyed `backstop/go-standards`, or only declared in `backstop.yml`? A
   declaration without an installed, locked pack makes the dogfood gate fail
   closed (`VerifyLock` / missing pack dir). Separately: was the stale
   `slotly/go-standards` lock entry removed? Leaving it makes `VerifyLock`
   report an extra/unreconciled entry.

4. Does a gate run on a project with **no** `.backstop/rules/` directory at all
   (not just an empty one) still succeed? `LoadManifest` returns defaults when
   the directory can't be read, but confirm no other code path stats
   `.backstop/rules/` and errors on its absence.

5. Was the entire `pkg/compile` package deleted (not left as dead-but-present
   code)? Its tests open the now-deleted
   `standards/go/STD-GO-001-go-code-standards.standard.md` source standard, so
   leaving the package would turn `go test ./...` red. Confirm CLM-021 (package
   directory absent) holds and that a full-module `go test ./...` is green after
   removal — not merely that no production code imports the package (CLM-012).

6. Does a known-bad Go file actually get FLAGGED once enforcement is on the
   consumed pack (CLM-025), or does the suite only prove the pack installs and
   the gate is green on clean code? Confirm `TestDogfoodPack_FlagsKnownGoViolation`
   asserts a real semgrep `Violation` on a violating fixture (GO-003 or GO-060)
   AND no violation of that rule on a clean control file. Confirm the known-bad
   fixture is a self-contained file authored for the test — not the absence of a
   pack invalid-fixture — so a mis-wired or empty `--config` cannot pass it. This
   is the single check that distinguishes "native enforcement transferred to the
   pack" from the vacuous-green failure where the rules silently stopped running.

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
