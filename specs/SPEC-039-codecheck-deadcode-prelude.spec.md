---
title: "Codecheck Deadcode Prelude"
number: SPEC-039
created: "2026-06-24"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    The behavior-preserving dead-code PRELUDE to BUNDLE-011's `pkg/check` cutover. This
    spec lands FIRST — before the toolchain-pack cutover proper (Seed 2 / SPEC-040) —
    to shrink the surface `pkg/check/manifest.go` presents to the rewrite that follows.
    It performs exactly TWO verified-dead, behavior-preserving deletions, both inside
    `pkg/check/manifest.go`, and nothing else.
    (1) REQ-010 — DELETE the dead compiled-standards-manifest READER:
    `compiledManifestFile` (+ its methods `routableExtensions`/`isCompiled`/
    `hasSemgrepSignal`/`legacyRules`/`deriveRules`), the `combinedRule` decode type,
    the `languageExtensions` map, and the `.manifest.json`-walking compiled/legacy
    branch of `LoadManifest`. This reader is DEAD-FED in production: no `.manifest.json`
    PRODUCER exists anywhere in the non-test tree (verified — only the reader references
    the `.manifest.json` suffix, and `check.go` confirms the dir is read for routing
    only with `defaultManifest()` as the always-taken production fallback), and the
    compiled-schema symbols have ZERO callers outside `manifest.go` itself. In
    production `LoadManifest` always falls to `defaultManifest()` → `routeFileDefaults`.
    (2) REQ-007 — DELETE the already-no-op non-Go semgrep catch-all in
    `routeFileDefaults`: the `default: return []CheckType{CheckTypeFindings}` branch
    that routes EVERY non-`.go`/`.ts`/`.tsx` file to the findings pass. It is verified
    a NO-OP on the gate RESULT: `CheckTypeFindings` has no `pkg/check` executor
    (registry.go builds executors only for lint/build/test), so the engine records that
    routed pass as a `Skipped` PassResult ("no executor configured") that produces zero
    violations and never changes pass/fail; findings already run through the pack engine
    (dispatchPackEngines / registry.go findings path), not through this catch-all. The
    `.go`/`.ts`/`.tsx` → all-four-passes case in `routeFileDefaults` is PRESERVED
    UNCHANGED; only the `default` catch-all branch is removed (a non-matching file then
    routes to the empty slice). Because both deletions are verified dead/no-op on THIS
    repo, NO golden-equivalence fixture is built here (that harness is Seed 2 / SPEC-040
    scope). The deletions orphan tests that pin the dead reader and the non-Go catch-all
    (in `pkg/check/manifest_test.go` and `pkg/check/ts_routing_test.go`); per the
    align-predating-artifacts principle those tests are deleted or rewritten to the
    post-deletion truth as part of this spec — they are not left red and not worked
    around. Every deletion is guarded by a deletion-assertion test (the symbol/branch is
    gone from non-test source) PLUS a behavior-preserving assertion (the routing/gate
    result on this repo is unchanged), so the deletion cannot silently alter routing.
    OUT OF SCOPE and explicitly fenced: the wholesale `pkg/check.Run` Step-2 cutover,
    `builtinToolchain` deletion, the `go-toolchain` pack, and the golden-equivalence
    harness (Seed 2 / SPEC-040); coverage / `step_coverage.go` / the CheckType-consumer
    catalog (Seed 3 / SPEC-041); and the `.standard.md` SCAFFOLDER `pkg/pack/scaffold.go`
    (ISSUE-030 — only the READER is in scope here, never the generator).
  package: pkg/check

verification:
  level: unit
  test_command: go test ./pkg/check/ ./cmd/backstop/ -race -coverprofile=cover.out
  coverage_threshold: 90

requirements:
  - id: REQ-007
    text: >
      The already-no-op non-Go semgrep catch-all in `routeFileDefaults`
      (pkg/check/manifest.go) — the `default` branch returning
      `[]CheckType{CheckTypeFindings}` for every file whose extension is not `.go`,
      `.ts`, or `.tsx` — must be DELETED. After deletion, `routeFileDefaults` returns
      the four passes for `.go`/`.ts`/`.tsx` (UNCHANGED) and the EMPTY slice for any
      other extension (no catch-all). Removal must be BEHAVIOR-PRESERVING on this repo's
      gate RESULT: `CheckTypeFindings` has no `pkg/check` executor (registry.go builds
      executors only for lint/build/test), so a non-Go file routed to findings produced
      only a `Skipped` PassResult ("no executor configured") that yielded zero
      violations and never changed pass/fail — findings already run through the pack
      engine, not this catch-all. The gate's violation set and exit code over this repo
      must be identical before and after the deletion. Post-cutover, semgrep on
      arbitrary files is an OPT-IN declared pack rule, never a baked default. The
      `.go`/`.ts`/`.tsx` all-passes case must NOT be touched. (DD-3 / RDQ-3.)
    supports: collapse-legacy-codecheck-into-packs:REQ-007
    follows: STD-GO-001:GO-010
  - id: REQ-010
    text: >
      The ENTIRE dead `.manifest.json` reader in pkg/check/manifest.go must be DELETED —
      both the compiled-standards sub-reader AND the legacy routing-schema arm — because
      the whole branch is VESTIGIAL (test-only-fed). Verified against `main`: ZERO
      production producers of any `.manifest.json` (only the reader + comments reference
      the suffix; no compiler/scaffolder/baseline-gen emits one), no committed
      `.manifest.json` outside `pkg/check/testdata/`, and no `.backstop/rules/` directory
      exists; every test asserting "liveness" writes its OWN fixture into a `t.TempDir()`
      — test coupling, NOT production reachability (exactly the trap Sharp Edge 3 warns
      against). DELETE: the `compiledManifestFile` type + ALL its methods
      (`routableExtensions`, `isCompiled`, `hasSemgrepSignal`, `legacyRules`,
      `deriveRules`); the `combinedRule` decode type; the `languageExtensions` map; the
      `manifestFile` decode struct; the `parseCheckTypes` and `hasRoutableRule` helpers;
      and the `.manifest.json`-walking/decoding body of `LoadManifest`. The deletion of
      that walk ALSO strands the rule-matching path (its sole producer was the
      `&Manifest{rules: allRules}` line inside the deleted branch — verified by
      production-reachability grep), so DELETE that stranded path TOO: `matchesRule`,
      `matchGlobPattern`, `matchDoubleStarPattern` (transitively — `matchesRule` is their
      ONLY production caller; their unit tests do not make them production-reachable), the
      `ManifestRule` type, and the `Manifest.rules` + `Manifest.isDefaults` fields. With no
      non-default `Manifest` ever produced, `RouteFile` collapses to
      `m.routeFileDefaults(path)` and the `Manifest` struct becomes empty. `LoadManifest`
      reduces to: read the dir (or skip) and return
      `defaultManifest(), nil` in all cases — it no longer reads `.manifest.json` at all. The
      zero-routable `ConfigError` EMISSION at manifest.go (the `...routable...` message)
      is deleted WITH the branch: it is the ONLY `ConfigError` construction tied to the
      `.manifest.json` path; every OTHER `ConfigError` trigger (missing-toolchain at
      registry.go, no-command, unknown-format at parsers.go) is independent of
      `.manifest.json`, fires from the toolchain/registry path, and is PRESERVED. The
      `ConfigError` TYPE itself (pkg/check/errors.go) is heavily used by that path and
      MUST NOT be deleted — only its manifest-path emission goes. After deletion, `.go`
      routing on THIS repo (no `.manifest.json`) is byte-for-byte unchanged (it always
      fell to `defaultManifest()` → `routeFileDefaults`). The surviving symbols `Manifest`
      (now an empty struct), `LoadManifest`, `RouteFile`, `routeFileDefaults` (minus
      REQ-007's branch), `defaultManifest`, `parseCheckType` (singular), the `CheckType`
      enum, and the `ConfigError` TYPE MUST remain. The adjacent `.standard.md` SCAFFOLDER
      is OUT of scope (ISSUE-030). (DD-5 / RDQ-5.)
    supports: collapse-legacy-codecheck-into-packs:REQ-010
    follows: STD-GO-001:GO-010

claims:
  # REQ-007: non-Go semgrep catch-all deletion
  - id: CLM-001
    requirement: REQ-007
    text: >
      The non-Go catch-all is gone from production source: no non-test .go file under
      pkg/check contains a `routeFileDefaults` `default` branch returning
      `[]CheckType{CheckTypeFindings}` (deletion-assertion over source).
    tests:
      - TestRouteFileDefaults_NonGoCatchAll_Removed
  - id: CLM-002
    requirement: REQ-007
    text: >
      After deletion, `routeFileDefaults` (via RouteFile on the default manifest) routes
      a non-`.go`/`.ts`/`.tsx` file (e.g. README.md, config.yml, notes.txt) to the EMPTY
      slice — no findings, no catch-all.
    tests:
      - TestRouteFileDefaults_NonGoFileRoutesToNothing
  - id: CLM-003
    requirement: REQ-007
    text: >
      The `.go` all-passes case is PRESERVED: a `.go` file still routes to
      {lint, build, test, findings} on the default manifest after the catch-all deletion.
    tests:
      - TestRouteFileDefaults_GoFileStillRoutesAllPasses
  - id: CLM-004
    requirement: REQ-007
    text: >
      The `.ts` and `.tsx` all-passes case is PRESERVED: `.ts` and `.tsx` files still
      route to {lint, build, test, findings} on the default manifest after the catch-all
      deletion (the deletion touches only the `default` branch, not the matched cases).
    tests:
      - TestRouteFileDefaults_TsFilesStillRouteAllPasses
  - id: CLM-005
    requirement: REQ-007
    text: >
      BEHAVIOR-PRESERVING (routing level): a non-Go file in scope contributes no
      executable pass before or after the catch-all deletion. The deleted `default` arm
      routed non-Go files to `CheckTypeFindings`, for which no `pkg/check` executor is
      ever built (findings runs on the pack engine, not through any `pkg/check`
      executor) — so it was always a no-op; post-deletion the non-Go file routes to the
      EMPTY slice. Either way it adds zero violations and cannot flip pass/fail. Proven
      at the route-table level over `defaultManifest()` (`README.md`, `config.yml`,
      `notes.txt` → empty), the production routing path, since the standalone
      executor-set behavior is independently pinned by CLM-006.
    tests:
      - TestRouteFileDefaults_NonGoFileRoutesToNothing
  - id: CLM-006
    requirement: REQ-007
    text: >
      The findings pass has no `pkg/check` executor in the default-manifest path: the
      executor map built for code-check contains lint/build/test only, never findings —
      substantiating that the deleted catch-all could not have run anything.
    tests:
      - TestCodeCheck_FindingsHasNoCheckExecutor
  - id: CLM-007
    requirement: REQ-007
    text: >
      ALL tests pinning the dead non-Go → findings catch-all are reconciled to the
      post-deletion truth (non-Go → empty), none left red or deleted-without-trace:
      `TestRouting_NonGoFileRoutesToSemgrepAfterRemoval`,
      `TestCodeCheck_Routing_DefaultsWhenNoManifest` (manifest_test.go:382,
      `script.py`→findings) and `TestCodeCheck_Routing_NoExtension`
      (manifest_test.go:635, `Makefile`→findings) are rewritten to assert the EMPTY
      route; and `TestCheckType_SemgrepRenamedToNeutralFindings` (manifest_test.go:786, a
      SPEC-035 test asserting `defaultManifest().RouteFile("notes.txt")==[findings]`) has
      its unknown-extension assertion updated to expect the empty route while still
      pinning the neutral-findings rename on the surviving sites (.go still routes
      findings).
    tests:
      - TestRouteFileDefaults_NonGoFileRoutesToNothing
      - TestCheckType_NeutralFindings_UnknownExtRoutesEmpty
  # REQ-010: dead compiled-standards-manifest reader deletion
  - id: CLM-010
    requirement: REQ-010
    text: >
      The `compiledManifestFile` type is gone from production source: no non-test .go
      file under pkg/check declares `type compiledManifestFile` (deletion-assertion).
    tests:
      - TestCompiledManifestReader_Removed
  - id: CLM-011
    requirement: REQ-010
    text: >
      The compiled-reader METHODS are gone from production source: no non-test .go file
      under pkg/check declares `deriveRules`, `isCompiled`, `hasSemgrepSignal`,
      `legacyRules`, or `routableExtensions`.
    tests:
      - TestCompiledManifestReader_MethodsRemoved
  - id: CLM-012
    requirement: REQ-010
    text: >
      The `combinedRule` decode type and the `languageExtensions` map are gone from
      production source: no non-test .go file under pkg/check declares
      `type combinedRule` or `languageExtensions`.
    tests:
      - TestCompiledManifestReader_CombinedRuleAndLanguageExtensionsRemoved
  - id: CLM-013
    requirement: REQ-010
    text: >
      `LoadManifest` no longer reads `.manifest.json` files at all: no non-test .go file
      under pkg/check decodes a `compiledManifestFile` or a `manifestFile`, branches on
      `isCompiled()`, or ranges entries matching `.manifest.json` inside `LoadManifest`.
      The reduced `LoadManifest` returns `defaultManifest()` for every dir.
    tests:
      - TestLoadManifest_NoCompiledManifestBranch
      - TestLoadManifest_NoManifestJSONRead
  - id: CLM-019
    requirement: REQ-010
    text: >
      WHOLE-BRANCH DELETION (reversal guard): the legacy routing-schema arm is GONE, not
      retained — `manifestFile`, `parseCheckTypes`, and `hasRoutableRule` are absent from
      non-test source under pkg/check, and the `.manifest.json` walk is removed. A
      `.manifest.json` placed in a rules dir is IGNORED (LoadManifest returns
      `defaultManifest()`), because the whole arm was test-only-fed with no production
      producer.
    tests:
      - TestLoadManifest_LegacyArmHelpersRemoved
      - TestLoadManifest_ManifestJSONIgnoredReturnsDefaults
  - id: CLM-020
    requirement: REQ-010
    text: >
      CONFIGERROR TYPE PRESERVED, NO PHANTOM TRIGGER REINTRODUCED: deleting the
      manifest-path zero-routable `ConfigError` emission does NOT remove the
      `ConfigError` TYPE (pkg/check/errors.go). After the BUNDLE-011 cutover deleted
      the baked `builtinToolchain` stacks, a declared language with NO
      `enforcement.toolchain` no longer has a registry-path toolchain to resolve, so
      there is nothing to enforce: the standalone code-check subcommand resolves to an
      EMPTY executor set and runs CLEAN (exit 0) — it does NOT emit a missing-toolchain
      `ConfigError`. Enforcement for such a language is opt-in via a `<lang>-toolchain`
      pack through the engine path; a project with none hits the WARN-ONLY
      no-toolchain-pack loud state on the gate, not an exit-2. This REPLACES the
      pre-cutover "missing toolchain = ConfigError" invariant the deleted builtin stack
      carried; the deletion of the manifest-path emission is still behavior-preserving
      because no surviving production path emits a `.manifest.json`-tied `ConfigError`.
    tests:
      - TestCodeCheck_MissingToolchain_NoDeclaredToolchainIsCleanNotConfigError
  - id: CLM-014
    requirement: REQ-010
    text: >
      PRODUCTION CONTRACT PRESERVED: `LoadManifest` over this repo's `.backstop/rules`
      path (which does not exist — there is no rules dir and no `.manifest.json` anywhere
      in the tree) returns the default manifest and routes `.go` to
      {lint, build, test, findings} — byte-for-byte the pre-deletion production behavior
      (production always fell to `defaultManifest()`).
    tests:
      - TestLoadManifest_RepoRulesDirStillRoutesGo
  - id: CLM-015
    requirement: REQ-010
    text: >
      `LoadManifest` over an absent or unreadable directory still returns the default
      manifest (no error), preserving the fallback contract the deleted reader sat
      in front of.
    tests:
      - TestLoadManifest_MissingDirReturnsDefaults
  - id: CLM-016
    requirement: REQ-010
    text: >
      The surviving manifest API is intact and minimal: `Manifest` (now an EMPTY struct),
      `RouteFile` (collapsed to `routeFileDefaults`), `routeFileDefaults`, `parseCheckType`
      (singular), `defaultManifest`, the `CheckType` enum, and the `ConfigError` TYPE
      remain present and exported-where-previously-exported (compile-and-route smoke). NOT
      in this surviving set (all deleted): the legacy-arm helpers
      `manifestFile`/`parseCheckTypes`/`hasRoutableRule` (see CLM-019) AND the stranded
      rule-matching path `matchesRule`/`matchGlobPattern`/`matchDoubleStarPattern`/
      `ManifestRule`/`Manifest.rules`/`Manifest.isDefaults` (see CLM-021).
    tests:
      - TestManifest_SurvivingAPIIntact
  - id: CLM-021
    requirement: REQ-010
    text: >
      STRANDED RULE-MATCHING PATH DELETED (downstream-dead guard): with the
      `.manifest.json` walk gone, nothing produces a non-default `Manifest`, so the
      rule-matching path is dead and is removed. No non-test .go file under pkg/check
      declares `matchesRule`, `matchGlobPattern`, `matchDoubleStarPattern`, or
      `type ManifestRule`, and the `Manifest` struct has no `rules`/`isDefaults` field.
      `RouteFile` routes via `routeFileDefaults` only. Verdict recorded:
      `matchGlobPattern`/`matchDoubleStarPattern` had `matchesRule` as their SOLE
      production caller (grep-verified) — their unit tests do not keep them live — so they
      are deleted transitively, not retained.
    tests:
      - TestRuleMatchingPath_Removed
  - id: CLM-017
    requirement: REQ-010
    text: >
      EVERY test touching the deleted `.manifest.json` arm (compiled OR legacy) is an
      orphan to delete/rewrite — none can "ride a retained arm" — so both swept packages
      compile and pass. Deleted (pinned the dead reader/decode): in
      `pkg/check/manifest_test.go` the compiled Python derive-routing tests
      (`TestCodeCheck_LoadManifest_DerivesRoutingFromCompiledManifest` family, STD-PY) and
      the legacy `.manifest.json`-loading tests (`TestRouting_ReadsBackstopDirManifestWhenPresent`,
      `TestCodeCheck_LoadManifest_NoRoutableRulesIsConfigError` and other `writeManifest`/
      `writeRawManifest` users); in `pkg/check/ts_routing_test.go` the compiled
      TS/declared-stack routing tests (`TestCodeCheck_Routing_TSFilesRouteAllPasses`,
      `TestCodeCheck_Routing_DeclaredStackExtensionsRoute`). Rewritten to the surviving
      `defaultManifest()` behavior: in `cmd/backstop/code_check_test.go`,
      `TestCodeCheck_LoadManifest_ConfigErrorPropagatesToCodeCheckExit` is deleted (its
      `routable` `ConfigError` is the deleted manifest-path emission), and
      `TestCLI_TSDeclaredStack_SmokeEndToEnd` is migrated off its compiled-manifest
      fixture (route `.ts` via the default manifest / declared toolchain, no
      `.manifest.json`); `TestCodeCheck_MissingToolchain_*` + `missingToolchainProject`
      are rewritten to drop the now-ignored `routing.manifest.json` write — their exit-2
      assertion still holds because the missing-toolchain `ConfigError` comes from the
      registry path, independent of `.manifest.json`. The gate-scope legacy-manifest tests
      (`cmd/backstop/gate_scope_test.go`) are likewise rewritten off the `.manifest.json`
      fixture. Additionally, the rule-matching tests orphaned by CLM-021's deletion are
      removed: `TestCodeCheck_Routing_PathPatternMatching` and the `writeManifest`
      route-table tests in `pkg/check/manifest_test.go`, and the two direct glob unit tests
      `TestCodeCheck_Routing_MatchGlobPattern_EdgeCases` and
      `TestCodeCheck_Routing_MatchDoubleStarPattern_EdgeCases` (they exercise only the
      deleted `matchGlobPattern`/`matchDoubleStarPattern`).
    tests:
      - TestCompiledManifestReader_Removed
      - TestLoadManifest_ManifestJSONIgnoredReturnsDefaults
      - TestRuleMatchingPath_Removed
  - id: CLM-018
    requirement: REQ-010
    text: >
      The `.standard.md` scaffolder is UNTOUCHED: pkg/pack/scaffold.go still exists and
      this spec's deletions do not reference or remove it (scope fence for ISSUE-030).
    tests:
      - TestStandardScaffolder_Untouched

contracts:
  - file: pkg/check/manifest.go
    provides:
      - name: LoadManifest
        kind: function
        signature: "func LoadManifest(dir string) (*Manifest, error)"
      - name: Manifest
        kind: type
        signature: "type Manifest struct"
      - name: "(*Manifest).RouteFile"
        kind: method
        signature: "func (m *Manifest) RouteFile(path string) []CheckType"
      - name: CheckType
        kind: type
        signature: "type CheckType int"
    consumes:
      - source: os
        name: ReadDir
        kind: function
      - source: path/filepath
        name: Ext
        kind: function
---

# SPEC-039: Codecheck Deadcode Prelude

## Overview

This spec is **Seed 1 of BUNDLE-011** — the **dead-code prelude** to collapsing the
legacy `pkg/check` enforcement engine into pack-declared toolchain packs. It is
deliberately small and entirely behavior-preserving: it removes two pieces of
**verified-dead / verified-no-op** code from `pkg/check/manifest.go` so that the
**cutover proper (Seed 2 / SPEC-040)** rewrites a smaller, cleaner surface.

The two deletions are:

- **REQ-010 (RDQ-5)** — the **entire dead `.manifest.json` reader** (BOTH the
  compiled-standards sub-reader — `compiledManifestFile` + methods, `combinedRule`,
  `languageExtensions` — AND the legacy routing-schema arm — `manifestFile`,
  `parseCheckTypes`, `hasRoutableRule`, and the `.manifest.json` walk in `LoadManifest`).
  The WHOLE branch is **vestigial (test-only-fed)**: verified against `main`, there is
  ZERO production producer of any `.manifest.json` (no compiler/scaffolder/baseline-gen
  emits one), no `.backstop/rules/` dir exists, and every "liveness" test writes its own
  `t.TempDir()` fixture — test coupling, not production reachability. `LoadManifest`
  reduces to the `defaultManifest()` path and no longer reads `.manifest.json` at all. The
  zero-routable `ConfigError` **emission** on this path goes with the branch (its only
  trigger); the `ConfigError` **TYPE** and every `.manifest.json`-independent trigger
  (missing-toolchain, etc.) are preserved. On this repo (no `.manifest.json`)
  `LoadManifest` already fell to `defaultManifest()`, so `.go` routing is unchanged.
- **REQ-007 (RDQ-3)** — the already-no-op non-Go semgrep **catch-all** in
  `routeFileDefaults` (the `default` branch that routed every non-`.go`/`.ts`/`.tsx`
  file to the findings pass). `CheckTypeFindings` has no `pkg/check` executor, so the
  catch-all could never change the gate result; findings already run through the pack
  engine.

**Dependency direction.** Seed 1 lands **BEFORE** Seed 2 (SPEC-040). It shrinks the
surface SPEC-040's wholesale `pkg/check.Run` cutover has to reason about. SPEC-040 is
the **downstream consumer** of this prelude. Because these deletions are verified
dead/no-op on this single repo, **no golden-equivalence fixture is built here** — that
harness belongs to SPEC-040.

**Hard scope fences** (these belong to siblings and MUST NOT be authored here): the
wholesale Step-2 cutover, `builtinToolchain` deletion, the `go-toolchain` pack, and the
golden-equivalence harness → **SPEC-040**; coverage / `step_coverage.go` / the
CheckType-consumer catalog → **SPEC-041**; the `.standard.md` **scaffolder**
(`pkg/pack/scaffold.go`) → **ISSUE-030** (only the **reader** is in scope here).

## Requirements

Formal requirements are enumerated in the `requirements:` frontmatter (REQ-007,
REQ-010), each tracing to its BUNDLE-011 source requirement via `supports` and to its
resolved design question (RDQ-3, RDQ-5). The summary below must match the frontmatter
exactly.

| Req | Deletes | Verified-dead/no-op evidence | Behavior preserved |
|-----|---------|------------------------------|--------------------|
| REQ-010 | The WHOLE `.manifest.json` reader AND the rule-matching path it strands: `compiledManifestFile` (+ `routableExtensions`/`isCompiled`/`hasSemgrepSignal`/`legacyRules`/`deriveRules`), `combinedRule`, `languageExtensions`, `manifestFile`, `parseCheckTypes`, `hasRoutableRule`, the `.manifest.json` walk in `LoadManifest`, the manifest-path zero-routable `ConfigError` emission, AND `matchesRule`/`matchGlobPattern`/`matchDoubleStarPattern`/`ManifestRule`/`Manifest.rules`/`Manifest.isDefaults`. PRESERVES the `ConfigError` TYPE + all `.manifest.json`-independent triggers, `parseCheckType` (singular), `routeFileDefaults`, `defaultManifest` | Whole branch vestigial (zero production `.manifest.json` producer; no `.backstop/rules/` dir; every "liveness" test writes its own `t.TempDir()` fixture). Rule-matching stranded: its sole producer was inside the deleted branch; `matchGlobPattern`/`matchDoubleStarPattern`'s sole production caller was `matchesRule` (grep-verified) | `LoadManifest` reduces to `defaultManifest()`; `RouteFile` collapses to `routeFileDefaults`; on this repo (no `.manifest.json`) `.go` routing byte-for-byte unchanged; missing-toolchain `ConfigError` (registry, `.manifest.json`-independent) still fails loud |
| REQ-007 | The `default` branch of `routeFileDefaults` returning `[]CheckType{CheckTypeFindings}` for non-`.go`/`.ts`/`.tsx` files | `CheckTypeFindings` has no `pkg/check` executor (registry builds lint/build/test only); routed findings pass was a `Skipped` no-op; findings run through the pack engine | Gate violation set + exit unchanged on this repo; `.go`/`.ts`/`.tsx` all-passes case untouched (now non-matching files → empty slice) |

**Allowlist of what REQ-007 removes vs preserves (explicit prohibition):** REQ-007 may
remove ONLY the `default` catch-all branch of `routeFileDefaults`. It is PROHIBITED from
altering the `.go`/`.ts`/`.tsx` matched case, from removing any of the four pass
constants, and from touching any executor-building code in `registry.go`.

**Allowlist of what REQ-010 removes vs preserves (explicit prohibition):** REQ-010
removes the WHOLE `.manifest.json` reader AND the rule-matching path it strands:
`compiledManifestFile` + its methods
(`routableExtensions`/`isCompiled`/`hasSemgrepSignal`/`legacyRules`/`deriveRules`),
`combinedRule`, `languageExtensions`, `manifestFile`, `parseCheckTypes`,
`hasRoutableRule`, the `.manifest.json` walk inside `LoadManifest`, the manifest-path
zero-routable `ConfigError` EMISSION, AND the stranded
`matchesRule`/`matchGlobPattern`/`matchDoubleStarPattern`/`ManifestRule` plus the
`Manifest.rules`/`Manifest.isDefaults` fields (with `RouteFile` collapsing to
`routeFileDefaults`). It is PROHIBITED from removing: the `ConfigError` TYPE
(pkg/check/errors.go) or any `.manifest.json`-independent `ConfigError` trigger
(missing-toolchain in registry.go, unknown-format in parsers.go, no-command, output.go);
`Manifest` (kept as an empty struct), `LoadManifest`, `RouteFile`, `routeFileDefaults`,
`parseCheckType` (singular — used by `validateToolchainKeys`), the `CheckType` enum, or
the `defaultManifest()` fallback; and from touching the `.standard.md` scaffolder. (This
reflects two corrections: Round-2 deleted the whole vestigial legacy arm — not just the
compiled sub-reader — and Round-3 deleted the rule-matching path that arm's removal
strands; both verified by production-reachability grep, not test reachability.)

## Implementation

`pkg/check/manifest.go` is the **single production file** edited. Both deletions are
mechanical and ordered so the diff stays clean (REQ-010's reader deletion first, since
it is the larger removal and lives above `routeFileDefaults`).

**Step 1 — REQ-010: delete the ENTIRE `.manifest.json` reader (compiled AND legacy).**
DELETE, from `pkg/check/manifest.go`:
1. `languageExtensions` map.
2. `combinedRule` struct.
3. `compiledManifestFile` struct and ALL its methods `routableExtensions`, `isCompiled`,
   `hasSemgrepSignal`, `legacyRules`, `deriveRules`.
4. `manifestFile` decode struct, the `parseCheckTypes` helper, and the `hasRoutableRule`
   helper (all hang off the legacy `.manifest.json` arm).
5. The `.manifest.json`-walking/decoding body of `LoadManifest`: the `os.ReadDir` loop
   that filters `.manifest.json` entries, decodes them, and accumulates rules; the
   `manifestFilesPresent` bookkeeping; and the manifest-path zero-routable `ConfigError`
   emission (the `...routable...` message). `LoadManifest` reduces to: attempt to read the
   dir (or skip even that) and return `defaultManifest(), nil` in all cases — it no longer
   reads `.manifest.json` at all. Keep the signature `func LoadManifest(dir string)
   (*Manifest, error)` so callers are source-compatible.
6. The STRANDED rule-matching path. Deleting item 5's `&Manifest{rules: allRules}` (the
   SOLE producer of a non-default `Manifest`) leaves the `m.rules` branch of `RouteFile`,
   `matchesRule`, `matchGlobPattern`, `matchDoubleStarPattern`, `ManifestRule`, and the
   `Manifest.rules`/`Manifest.isDefaults` fields unreachable. DELETE all of them:
   `RouteFile` collapses to `return m.routeFileDefaults(path)`; `Manifest` becomes an empty
   struct (`defaultManifest()` returns `&Manifest{}`). PRODUCTION-REACHABILITY VERDICT
   (grep, not test reachability): `matchesRule` is the SOLE production caller of
   `matchGlobPattern`, which is the SOLE production caller of `matchDoubleStarPattern`; no
   other production site (pack glob matching or otherwise) calls either — so both are
   stranded and deleted transitively. Their standalone unit tests do NOT keep them live
   (the same test-liveness trap as Round-2); those tests are orphans (Step 3). RETAIN
   `parseCheckType` (SINGULAR) — it is used by `validateToolchainKeys` in registry.go and
   is unrelated to the deleted plural `parseCheckTypes`.

**ConfigError production-trigger verification (done at authoring, confirm at impl):** the
manifest-path `ConfigError` emission is the ONLY `ConfigError` construction tied to
`.manifest.json`. Every other `ConfigError` site is independent of it and is PRESERVED —
the `ConfigError` TYPE lives in pkg/check/errors.go; missing-toolchain fires from
`resolveToolchain` (registry.go), unknown-format from parsers.go, no-command from
registry.go, plus output.go. Crucially, `pkg/check.Run` resolves the toolchain
UNCONDITIONALLY (after routing, not gated on it), so the missing-toolchain exit-2 path is
unaffected by removing the manifest reader. Deleting the manifest-path emission therefore
removes a vestigial fail-loud (it only ever fired on a test-written `.manifest.json`), not
a production-reachable one.

**Step 2 — REQ-007: delete the non-Go catch-all in `routeFileDefaults`.**
`routeFileDefaults` keeps the `.go`/`.ts`/`.tsx` → all-four-passes case and DROPS the
`default` arm. With the `default` arm gone, a non-matching extension returns the empty
slice (no findings). Do not change the matched case.

**Step 3 — reconcile orphaned tests (align-predating-artifacts).**
The `test_command` sweeps BOTH `./pkg/check/` and `./cmd/backstop/`, so the
reconciliation MUST cover both packages. With the WHOLE `.manifest.json` arm AND its
stranded rule-matching path deleted, EVERY test that reads/writes a `.manifest.json` or
exercises rule-matching is an orphan — none "rides a retained arm." Four families;
reconcile ALL in the SAME change, none left red, no work-arounds:

*(a) `.manifest.json`-reader orphans — DELETE (they pinned the dead reader/decode):*
- `pkg/check/manifest_test.go`: the compiled Python derive-routing tests
  (`TestCodeCheck_LoadManifest_DerivesRoutingFromCompiledManifest` family, STD-PY); the
  legacy `.manifest.json`-loading tests including
  `TestRouting_ReadsBackstopDirManifestWhenPresent`,
  `TestCodeCheck_LoadManifest_NoRoutableRulesIsConfigError`, and the other
  `writeManifest`/`writeRawManifest` route-table tests.
- `pkg/check/ts_routing_test.go`: `TestCodeCheck_Routing_TSFilesRouteAllPasses` and
  `TestCodeCheck_Routing_DeclaredStackExtensionsRoute` (STD-TS/STD-RS compiled fixtures) —
  DELETED. The TS EXECUTOR-parsing tests (`buildExecutorsForConfig`-driven, no
  `.manifest.json`) STAY — they never touch the reader.

*(b) Routing-schema/ConfigError tests once slated to "ride the arm" — now ORPHANS:*
- `cmd/backstop/code_check_test.go`:
  `TestCodeCheck_LoadManifest_ConfigErrorPropagatesToCodeCheckExit` — DELETED (its
  `routable` `ConfigError` IS the deleted manifest-path emission; nothing production-side
  reproduces it).
- `TestCLI_TSDeclaredStack_SmokeEndToEnd` — REWRITTEN off its compiled `.manifest.json`
  fixture: route `.ts` via the default manifest + declared toolchain (no `.manifest.json`
  at all), so the end-to-end smoke survives on the surviving routing.
- `TestCodeCheck_MissingToolchain_*` + `missingToolchainProject` — REWRITTEN to DROP the
  now-ignored `routing.manifest.json` write. The exit-2 assertion still holds because the
  missing-toolchain `ConfigError` comes from `resolveToolchain` (registry.go), which runs
  unconditionally regardless of routing — independent of `.manifest.json`.
- The gate-scope legacy-manifest tests (`cmd/backstop/gate_scope_test.go`, the
  `{extensions, check_types}` `.manifest.json` fixtures) — REWRITTEN off the
  `.manifest.json` fixture to the default-manifest routing.

*(c) Rule-matching orphans — DELETE (they exercise the deleted stranded path):*
- `pkg/check/manifest_test.go`: `TestCodeCheck_Routing_PathPatternMatching` (`:406`,
  `writeManifest` + `PathPatterns` → `matchesRule`) and the remaining `writeManifest`
  route-table tests; the two direct glob unit tests
  `TestCodeCheck_Routing_MatchGlobPattern_EdgeCases` (`:553`) and
  `TestCodeCheck_Routing_MatchDoubleStarPattern_EdgeCases` (`:585`) — they exercise ONLY
  the deleted `matchGlobPattern`/`matchDoubleStarPattern` (which are now production-dead),
  so they are deleted, not migrated.

*(d) Surviving-API behavior altered by REQ-007 (rewrite the assertion to non-Go → empty):*
- `pkg/check/manifest_test.go`: `TestRouting_NonGoFileRoutesToSemgrepAfterRemoval`,
  `TestCodeCheck_Routing_DefaultsWhenNoManifest` (`:382`, `script.py`→findings), and
  `TestCodeCheck_Routing_NoExtension` (`:635`, `Makefile`→findings) — rewritten to assert
  the EMPTY route.
- `TestCheckType_SemgrepRenamedToNeutralFindings` (`:786`, a SPEC-035 test) asserts
  `defaultManifest().RouteFile("notes.txt")==[findings]` — its unknown-extension assertion
  is updated to expect the EMPTY route while keeping the neutral-findings rename pinned on
  surviving sites (.go still routes findings).

**Step 4 — add deletion-assertion + behavior-preserving tests** per the claims
(see Verification).

**Validation passes touched:** exactly one — `pkg/check`'s file-type ROUTING (the
`Manifest`/`RouteFile` layer). The lint/build/test/findings **executor** layer
(registry.go) is NOT touched. No gate STEP is added or removed.

## Verification

`go test ./pkg/check/ ./cmd/backstop/ -race -coverprofile=cover.out`, unit level, 90%
coverage threshold. Claims are defined in the frontmatter `claims:` array; each maps to
a named test. The two test SHAPES this spec mandates:

1. **Deletion-assertion tests** (model: `pkg/validate/deletion_assertion_test.go`) — scan
   the non-test `.go` sources under `pkg/check` and assert the deleted symbols are ABSENT:
   the reader cluster `compiledManifestFile` + its methods (`routableExtensions`/
   `isCompiled`/`hasSemgrepSignal`/`legacyRules`/`deriveRules`), `combinedRule`,
   `languageExtensions`, `manifestFile`, `parseCheckTypes`, `hasRoutableRule`; the stranded
   rule-matching cluster `matchesRule`, `matchGlobPattern`, `matchDoubleStarPattern`,
   `ManifestRule`, and the `Manifest` struct's `rules`/`isDefaults` fields; that
   `LoadManifest` no longer reads `.manifest.json` (no `isCompiled()` branch, no
   `.manifest.json` suffix match in its body); and that the `routeFileDefaults` `default`
   findings arm is gone. Must STILL be present: the `ConfigError` TYPE (only its
   manifest-path emission removed), `parseCheckType` (singular), `RouteFile`,
   `routeFileDefaults`, `defaultManifest`. These are red while the symbols exist and go
   green only after deletion, preventing
   silent reintroduction.
2. **Behavior-preserving assertions** — assert the ROUTING and GATE RESULT on this repo
   are unchanged: `.go` still routes to all four passes; `LoadManifest` over the real
   `.backstop/rules` and over a missing dir still returns the default manifest; a
   code-check run with non-Go files in scope yields the same violations/exit (the routed
   findings pass was already a `Skipped` no-op).

The behavior-preserving assertions are the guard against the deletion silently changing
routing or findings. If ANY of them cannot be made to hold, the deletion's dead/no-op
premise is false and the work STOPS (see Sharp Edges).

## Sharp Edges

- **A skipped-findings PassResult disappears for non-Go files (observable, intended).**
  Pre-deletion, a non-Go file in scope produced a `Skipped` PassResult with reason "no
  executor configured" for the findings pass. Post-REQ-007 it routes to nothing, so that
  skip entry vanishes. This changes the **PassResults inventory** and could nudge the
  gate's `StepsSkipped` counter for non-Go-only scopes — but NOT the violation set or
  exit code. The behavior-preserving assertion (CLM-005) MUST pin **violations + exit**,
  NOT an exact PassResults list, or it will fail on this intended delta. This is the
  single subtlest thing in the spec.
- **Do not delete anything still reachable.** Each deletion rests on the
  verified-dead/no-op evidence (no `.manifest.json` producer; zero external callers of
  the compiled-schema symbols; findings has no `pkg/check` executor). If authoring
  surfaces ANY live reference — a production `.manifest.json` writer, a non-test caller
  of `deriveRules`/`isCompiled`/etc., or a `pkg/check` findings executor — **STOP and
  report it**; do not author the deletion around a live edge.
- **Tests feed `.manifest.json`; production does not.** Several existing tests construct
  compiled `.manifest.json` fixtures and exercise the dead reader. "Dead-fed in
  production" is true; "unreferenced" is NOT — the reader has TEST callers. Those tests
  must be deleted/rewritten in this change (align-predating-artifacts), or the package
  will not compile. Do not interpret the test callers as evidence the reader is live.
- **Deleting the reader STRANDS the rule-matching path downstream — delete it too
  (Round-3).** The `.manifest.json` walk was the SOLE producer of a non-default
  `Manifest{rules:...}`; once it is gone, `RouteFile`'s `m.rules` branch, `matchesRule`,
  `matchGlobPattern`, `matchDoubleStarPattern`, `ManifestRule`, and `Manifest.rules`/
  `isDefaults` are unreachable. Verdict (PRODUCTION-reachability grep, not test
  reachability): `matchesRule` is the sole production caller of `matchGlobPattern`, which
  is the sole production caller of `matchDoubleStarPattern` — no other production consumer
  exists, so the trio is deleted transitively. Standalone unit tests for the glob helpers
  do NOT keep them live (the same test-liveness trap). Guard against mis-deleting the
  RETAINED `parseCheckType` SINGULAR (used by `validateToolchainKeys`) — only the plural
  `parseCheckTypes` goes.
- **The legacy routing-schema arm is VESTIGIAL — DELETE it (Round-2 reversal).** It is
  tempting to "retain" the legacy `{extensions, check_types}` decode because tests exercise
  it — but those tests write their own `t.TempDir()` fixtures; that is test coupling, not
  production reachability (the very trap the bullet above names). Verified against `main`:
  no production `.manifest.json` producer, no `.backstop/rules/` dir, no committed
  `.manifest.json` outside `pkg/check/testdata/`. So the WHOLE arm is deleted; the tests
  that "proved" it live become orphans (Step 3 family b). This reverses the Round-1
  narrowing.
- **Preserve the `ConfigError` TYPE; delete only its manifest-path EMISSION.** The
  manifest-path zero-routable `ConfigError` is the ONLY `ConfigError` construction tied to
  `.manifest.json`. Deleting it must NOT touch the `ConfigError` TYPE (pkg/check/errors.go)
  or any other trigger. The fail-loud that matters in production — a declared language with
  no toolchain — comes from `resolveToolchain` (registry.go), runs unconditionally after
  routing, and is untouched. Verify (impl-time) there is no OTHER `.manifest.json`-tied
  `ConfigError`; the authoring grep found exactly one (manifest.go), all others in
  registry.go/parsers.go/output.go. If a second `.manifest.json`-tied trigger surfaces,
  STOP and report rather than deleting it.
- **Helper fates are deterministic — ALL DELETE.** `manifestFile`, `parseCheckTypes`, and
  `hasRoutableRule` hang off the deleted `.manifest.json` arm and have no other production
  caller once the walk is gone — they are DELETED, not retained. `parseCheckType`
  (singular, used by `validateToolchainKeys` in registry.go) is a DIFFERENT function and is
  RETAINED; do not confuse it with the deleted plural `parseCheckTypes`.
- **Coverage threshold pressure from net-negative code.** This change is mostly
  deletions; the 90% unit threshold is over `pkg/check` whose surviving routing layer is
  small and well-tested. Ensure the new behavior-preserving tests cover the reduced
  `LoadManifest` and `routeFileDefaults` so coverage does not regress under the floor.

## Review Questions

- Did the implementer confirm — at implementation time, against current `main` — that NO
  `.manifest.json` producer exists in the non-test tree and that the compiled-schema
  symbols have zero non-test callers? If any live reference was found, was it reported
  rather than worked around?
- Does the behavior-preserving assertion for REQ-007 pin the gate **violation set and
  exit code** (not an exact PassResults list)? An assertion that pins PassResults will
  spuriously fail on the intended disappearance of the skipped-findings entry.
- After REQ-010, does `LoadManifest` over a directory with no `.manifest.json` (including
  this repo's real `.backstop/rules`) still return `defaultManifest()` with `.go`
  routing to all four passes — and over a missing/unreadable dir still return defaults
  with no error?
- Were ALL `.manifest.json`-touching tests reconciled across BOTH swept packages — the
  compiled-fixture tests (`pkg/check/manifest_test.go` STD-PY, `pkg/check/ts_routing_test.go`
  STD-TS/STD-RS) deleted; the legacy `.manifest.json`-loading tests
  (`TestRouting_ReadsBackstopDirManifestWhenPresent`,
  `TestCodeCheck_LoadManifest_NoRoutableRulesIsConfigError`,
  `TestCodeCheck_LoadManifest_ConfigErrorPropagatesToCodeCheckExit`) deleted;
  `TestCLI_TSDeclaredStack_SmokeEndToEnd`, `TestCodeCheck_MissingToolchain_*` /
  `missingToolchainProject`, and the gate-scope manifest tests rewritten OFF
  `.manifest.json`; AND the REQ-007 catch-all dependents
  (`TestRouting_NonGoFileRoutesToSemgrepAfterRemoval`,
  `TestCodeCheck_Routing_DefaultsWhenNoManifest`, `TestCodeCheck_Routing_NoExtension`,
  `TestCheckType_SemgrepRenamedToNeutralFindings`) rewritten to non-Go → empty; AND the
  rule-matching orphans (`TestCodeCheck_Routing_PathPatternMatching`,
  `TestCodeCheck_Routing_MatchGlobPattern_EdgeCases`,
  `TestCodeCheck_Routing_MatchDoubleStarPattern_EdgeCases`, remaining `writeManifest`
  route-table tests) deleted — none left red or stubbed to skip?
- Was the rule-matching path deleted as stranded dead code (`matchesRule`,
  `matchGlobPattern`, `matchDoubleStarPattern`, `ManifestRule`, `Manifest.rules`/
  `isDefaults`), with `RouteFile` collapsed to `routeFileDefaults` and `Manifest` an empty
  struct — and was the production-reachability verdict (NOT test reachability) recorded for
  `matchGlobPattern`/`matchDoubleStarPattern`? Is there any stranded member left mislabeled
  as "surviving API"?
- Is the `.go`/`.ts`/`.tsx` matched case in `routeFileDefaults` byte-for-byte unchanged,
  and are all four `CheckType` constants and the `registry.go` executor-building code
  untouched?
- Is the `.standard.md` scaffolder (`pkg/pack/scaffold.go`) demonstrably untouched (scope
  fence for ISSUE-030)?
- Was the WHOLE `.manifest.json` reader deleted (`compiledManifestFile` + methods,
  `combinedRule`, `languageExtensions`, `manifestFile`, `parseCheckTypes`,
  `hasRoutableRule`, the walk, AND the manifest-path `ConfigError` emission) while the
  `ConfigError` TYPE and the `.manifest.json`-INDEPENDENT triggers (missing-toolchain in
  registry.go) were PRESERVED? Does `TestCodeCheck_MissingToolchain_*` still exit-2 via the
  registry path, and is `parseCheckType` (singular, used by `validateToolchainKeys`) NOT
  deleted by mistake?

## References

- **BUNDLE-011** (`bundles/BUNDLE-011-collapse-legacy-codecheck-into-packs.bundle.md`,
  maturity `defined`) — the source bundle. Seed 1 owns REQ-007 (RDQ-3) and REQ-010
  (RDQ-5). DD-3 and DD-5 carry the rationale.
- **SPEC-040** (toolchain-pack cutover, Seed 2) — the **downstream consumer**: the
  wholesale `pkg/check.Run` Step-2 cutover + `builtinToolchain` deletion + `go-toolchain`
  pack + golden-equivalence harness. Seed 1 lands first to shrink SPEC-040's surface.
- **SPEC-041** (coverage re-impl + CheckType-consumer catalog, Seed 3) — sibling; out of
  scope here.
- **ISSUE-030** (native-standards tombstone) — owns the `.standard.md` **scaffolder**;
  explicitly NOT this spec (only the **reader** is here).
- `pkg/validate/deletion_assertion_test.go` — the deletion-assertion test pattern this
  spec follows (scan non-test sources, assert absence of deleted symbols).
- Code (verified against `main`, 2026-06-24): `pkg/check/manifest.go` (the whole
  `.manifest.json` reader + `routeFileDefaults` + the sole manifest-path `ConfigError`
  emission); `pkg/check/errors.go` (the `ConfigError` TYPE — preserved); `pkg/check/check.go`
  (routing dir read for routing only, default fallback; toolchain resolved
  unconditionally; engine records `Skipped` "no executor configured" for unrouted passes);
  `pkg/check/registry.go` (executors built for lint/build/test only; missing-toolchain
  `ConfigError`, `.manifest.json`-independent; `validateToolchainKeys` uses the RETAINED
  `parseCheckType` singular); no `.backstop/rules/` dir and no committed `.manifest.json`
  outside `pkg/check/testdata/` (vestigial confirmation); `pkg/check/manifest_test.go`,
  `pkg/check/ts_routing_test.go`, `cmd/backstop/code_check_test.go`,
  `cmd/backstop/gate_scope_test.go` (orphaned `.manifest.json` tests to reconcile).
