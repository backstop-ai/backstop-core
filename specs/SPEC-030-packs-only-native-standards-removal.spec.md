---
title: "SPEC-030: Packs-Only — Native Standards Removal"
number: SPEC-030
created: "2026-06-16"
status: implemented
schema_version: spec/v1
spec_version: 2.2.0

implementation:
  summary: >
    Rip out the legacy native-standards execution arm so the gate is
    packs-only. Delete the `.standard.md → standards-compiler → manifestDir`
    rule-config path: under the thin-executor strategy the in-process
    `semgrepExecutor` is deleted ENTIRELY by ISSUE-018 (it scanned zero rules
    and is redundant with the engine dispatch), so a fortiori there is no
    `manifestDir` `--config` arm and no compiled-standards feed left on any
    in-process semgrep pass. Remove the `ManifestDir` field from `check.Options`
    and its callers that fed compiled-standards rules into the (now-deleted)
    semgrep pass, retire
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
  subject: pkg/check

verification:
  level: integration
  test_command: go test ./pkg/check/ ./cmd/backstop/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      No compiled-standards / `.standard.md` rule config may feed any in-process
      semgrep pass in pkg/check. Under the thin-executor strategy this is
      realized by the ABSENCE of any in-process semgrep `--config` arm: ISSUE-018
      deletes the in-process `semgrepExecutor` (pkg/check/check.go) ENTIRELY —
      together with `semgrepJSON` / `EnsureSemgrep` — so there is no
      `manifestDir`-bearing executor for a compiled-standards
      `--config <manifestDir>` arm to live on at all. There is therefore no
      `if e.manifestDir != ""` arm and no `manifestDir` struct field on any
      in-process semgrep executor, because there is no in-process semgrep
      executor. Real semgrep enforcement of pack rules dispatches group-by-engine
      through `dispatchPackEngines` (SPEC-031), never an in-process semgrep
      `--config` feed. No pkg/check production code path assembles a semgrep
      `--config` argument from a compiled-standards manifest directory.
    supports: pluggable-pack-engines:REQ-016@1.0.0
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
    supports: pluggable-pack-engines:REQ-016@1.0.0
  - id: REQ-006
    text: >
      The `STD-GO-001` source standard
      (`standards/go/STD-GO-001-go-code-standards.standard.md`) must be
      dropped, since its enforcement content now lives in the published
      `backstop/go-standards` pack. No production code path may require the `STD-GO-001`
      standard artifact or its compiled outputs to exist. Validation and gate
      runs must not fail merely because `STD-GO-001` is absent.
    supports: pluggable-pack-engines:REQ-016@1.0.0
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
    supports: pluggable-pack-engines:REQ-016@1.0.0

claims:
  # REQ-001 — compiled-standards --config arm gone (no in-process semgrep
  # manifestDir feed survives). Repointed at spec_version 1.1.0: the surviving
  # realization of REQ-001's intent under the thin-executor strategy is that
  # there is NO in-process semgrep executor at all (ISSUE-018 deletes
  # semgrepExecutor/EnsureSemgrep), so a fortiori no compiled-standards
  # manifestDir --config arm exists. These claims assert the SURVIVING truth —
  # the absence of any in-process semgrep compiled-standards feed — and must NOT
  # assert that an in-process semgrep invocation occurs.
  # CLM-001 (was: no pkg/check production path assembles a semgrep `--config`
  # from a compiled-standards manifest directory →
  # TestPkgCheck_NoCompiledStandardsConfigArm) was RETIRED at spec_version 1.3.0
  # as a near-duplicate. As scoped it is a THIRD pkg/check production-source
  # absence self-check whose assertion ("no `--config` sourced from a standards
  # dir; no `manifestDir`-style field; no in-process semgrep executor") is
  # already made, token-for-token, by CLM-002
  # (TestPkgCheck_NoResidualStandardsConfigWhenNoPacks — scans production source
  # for `ExtraSemgrepConfigs` / `manifestDir` / `ManifestDir` absent) and CLM-003
  # (TestPkgCheck_NoManifestDirFieldOnSemgrepFeed — scans for `semgrepExecutor` /
  # `manifestDir` / `ManifestDir` absent). Writing TestPkgCheck_
  # NoCompiledStandardsConfigArm would mandate a redundant third token scan over
  # the same production source asserting the same structural absence — the
  # near-duplicate-claim pattern this reconciliation exists to eliminate. Its one
  # distinct token (a `--config` sourced from a `.backstop/rules` standards dir)
  # is folded into CLM-002's scan set below. REQ-001 retains CLM-002, CLM-003,
  # and CLM-023 — no requirement is left claim-less. Retired rather than
  # repointed to keep the claim set minimal and non-overlapping.
  - id: CLM-002
    requirement: REQ-001
    text: >
      No compiled-standards `--config` source survives in pkg/check production
      source: with the in-process semgrep pass deleted there is no leftover
      standards-dir feed assembled into a semgrep `--config`. A source self-check
      over pkg/check production source asserts none of the deleted compiled-
      standards arm symbols survive — `ExtraSemgrepConfigs`, `manifestDir`,
      `ManifestDir` — and that no `--config` argument is sourced from a
      `.backstop/rules` standards directory (the distinct token folded in from
      retired CLM-001). The claim asserts the ABSENCE of the deleted
      compiled-standards arm and does NOT assert that any in-process semgrep pass
      is invoked (no in-process semgrep pass exists under the thin-executor
      strategy).
    tests:
      - TestPkgCheck_NoResidualStandardsConfigWhenNoPacks
  - id: CLM-003
    requirement: REQ-001
    text: >
      No `manifestDir` field (or any compiled-standards `--config` field)
      survives on any in-process semgrep executor in pkg/check, because the
      in-process semgrep executor type itself is gone under the thin-executor
      strategy (ISSUE-018 removes `semgrepExecutor`). A source/symbol self-check
      asserts no `semgrepExecutor.manifestDir`-style field exists to feed a
      compiled-standards `--config`.
    tests:
      - TestPkgCheck_NoManifestDirFieldOnSemgrepFeed
  - id: CLM-023
    requirement: REQ-001
    text: >
      No remaining test in pkg/check constructs an in-process semgrep executor
      with a `manifestDir` field or asserts a standards-dir (`.backstop/rules`)
      `--config` path. A source self-check over pkg/check test files asserts
      neither the token "manifestDir:" in a semgrep-feed literal nor a
      `containsConfigFor(..., ".../rules")` standards-dir assertion survives, so
      the green `go test ./...` guarantee is enforced rather than assumed. No
      test re-introduces a standards-dir `--config` assertion against an
      (now-absent) in-process semgrep pass.
    tests:
      - TestNoTestRequiresManifestDirOrStandardsConfig

  # REQ-005 — pkg/compile retired, compiled artifacts deleted
  - id: CLM-012
    requirement: REQ-005
    kind: absence
    text: >
      No production (non-test) file under cmd/backstop or pkg/check imports
      github.com/bmanson/backstop-core/pkg/compile.
    tests:
      - TestNoProductionImportOfCompile
  - id: CLM-013
    requirement: REQ-005
    kind: absence
    text: >
      The compiled-standards artifacts (STD-GO-001.manifest.json,
      STD-GO-001.native.json, STD-GO-001.semgrep.yml) are absent from
      .backstop/rules/ in the repository tree.
    tests:
      - TestCompiledStandardsArtifactsAbsent
  - id: CLM-021
    requirement: REQ-005
    kind: absence
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
    kind: absence
    text: >
      The STD-GO-001 source standard file is absent from standards/go/.
    tests:
      - TestStdGo001SourceAbsent
  - id: CLM-015
    requirement: REQ-006
    subject: pkg/gate
    text: >
      A gate / code-check run succeeds (no config error, no missing-standard
      error) on a project with no STD-GO-001 artifact and no compiled
      standards directory.
    tests:
      - TestGate_SucceedsWithoutStandards

  # REQ-007 — dogfood-consume backstop/go-standards pack
  - id: CLM-016
    requirement: REQ-007
    kind: absence
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
    kind: absence
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

contracts:
  - file: pkg/check/check.go
    provides:
      - name: semgrepExecutor
        kind: type
        absent: true
        signature: "type semgrepExecutor struct"
        notes: >
          ABSENCE assertion (absent: true): this entry asserts semgrepExecutor
          is GONE from pkg/check/check.go and fails if it reappears (a deletion
          regression guard). The signature value is documentary only — the gate
          ignores it for an absent entry; the schema requires a non-empty
          signature on every provides entry.
          Under the thin-executor strategy the in-process semgrep executor is
          REMOVED entirely (ISSUE-018 deletes semgrepExecutor / semgrepJSON /
          EnsureSemgrep). This spec's REQ-001 intent — no compiled-standards
          manifestDir --config arm — is realized by the absence of any in-process
          semgrep feed: there is no semgrepExecutor.manifestDir to delete because
          there is no semgrepExecutor. Rule enforcement dispatches through the
          pack-engine path (SPEC-031 dispatchPackEngines), not an in-process
          semgrep --config feed.
    consumes:
      - source: pkg/check/manifest.go
        name: LoadManifest
        kind: function
      - source: pkg/check/manifest.go
        name: defaultManifest
        kind: function
  # The cmd/backstop/gate.go contract block was REMOVED at spec_version 2.2.0.
  # Its sole provides entry declared `(*realCodeChecker).runCheck` as PRESENT —
  # the exact opposite of what this spec delivers. That symbol was deliberately
  # DELETED (SPEC-040 keystone cutover); a grep of non-test source returns zero
  # matches and cmd/backstop/cutover_deletion_test.go's
  # TestCutover_RealCodeCheckerDeleted guards the absence. Nothing replaced it:
  # the responsibility moved to pack-driven engine dispatch, so there is no
  # successor symbol to re-declare. See Version History (2.2.0).
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

### Strategy repoint (spec_version 1.1.0 → 1.2.0)

> **Why this repoint:** This spec was authored when the gate still ran a bespoke
> **in-process `semgrepExecutor`** (pkg/check) that shelled out to
> `semgrep --json --quiet`, and the spec described removing the
> compiled-standards `--config` arm *from that executor*. Under the now-coherent
> thin-executor strategy the in-process semgrep executor is **removed entirely**
> (ISSUE-018 deletes `semgrepExecutor` / `semgrepJSON` / `EnsureSemgrep`; it
> scanned zero rules and is redundant with the engine dispatch). All real semgrep
> enforcement runs through the data-driven engine path
> (`pkg/pack/engine` + `cmd/backstop/dispatchPackEngines` → SARIF, SPEC-031).
>
> **The INTENT of this spec survives and is MORE completely realized.** REQ-001's
> goal — *no compiled-standards / `.standard.md` rule config feeds the semgrep
> pass* — holds harder than before: there is no compiled-standards `--config` arm
> because (a) no compiled-standards manifest directory is wired into check
> `Options`, and (b) there is no in-process semgrep executor for such an arm to
> live on at all; pack rules dispatch group-by-engine through
> `dispatchPackEngines`, not through an in-process semgrep `--config` feed.
>
> Accordingly the claims that previously asserted an **in-process semgrep
> invocation** (CLM-001, CLM-002, CLM-003, CLM-011) are repointed to assert the
> **surviving truth** — that check `Options` carry no compiled-standards manifest
> directory, and that pack rules flow through the engine path rather than an
> in-process semgrep `--config` feed — **without** asserting that any in-process
> semgrep invocation occurs. CLM-011's mandated test is renamed
> (`TestCodeCheck_PackOnly_SemgrepConfigIsPackPathsOnly` →
> `TestCodeCheck_PackOnly_RulesDispatchViaEnginePath`) because its old name
> described an in-process pack-config feed that no longer exists; the other
> repointed claims keep their test names (the surviving assertion still fits).
>
> **At 1.2.0, CLM-005, CLM-007, and CLM-010 are RETIRED** rather than repointed:
> they were redundant structural-absence echoes (covered by CLM-003/CLM-004/CLM-002)
> and, for CLM-007/CLM-010, repointing them to call `pkg/check` would have
> manufactured vacuous tests against a code path that — post-ISSUE-018 — can never
> produce the forbidden compiled-standards wiring. See **Version History (1.2.0)**.
>
> **At 1.3.0, the REQ-008 pair (CLM-018, CLM-019) is reconciled and the REQ-001
> claim set is de-duplicated.** The 1.2.0 pass repointed the REQ-001 / REQ-004
> claims off any in-process semgrep invocation but MISSED the REQ-008 pair, which
> still carried the pre-strategy assumption that a deleted in-process semgrep
> invocation's runner args could be recorded and checked for a `.backstop/rules/`
> path. **CLM-018 is repointed** to the surviving BEHAVIORAL property — the sole
> production reader of `.backstop/rules/` is `LoadManifest`, which collects only
> `*.manifest.json` and skips a leftover `STD-*.semgrep.yml`, so a populated rules
> dir holding only `*.semgrep.yml` yields the default manifest (zero rules, zero
> routes). **CLM-019 is RETIRED** (folded into CLM-018): it directly asserted the
> deleted in-process semgrep pass still runs, which has no surviving subject; its
> distinct intent is identical to repointed CLM-018's. Separately, **CLM-001 is
> RETIRED** as a near-duplicate: as scoped it is a third pkg/check
> production-source absence token-scan already covered token-for-token by CLM-002
> and CLM-003; its one distinct token (a `--config` from a standards dir) is folded
> into CLM-002. After 1.3.0 the "compiled-standards / in-process-semgrep path is
> gone" area is a MINIMAL, non-overlapping set — see **Version History (1.3.0)**.

### What is removed vs. what stays

| Concern | Today | After this spec |
|---|---|---|
| compiled-standards rule config feeding the semgrep pass (`manifestDir`) | present (in-process `semgrepExecutor`'s `if e.manifestDir != ""` arm) | **gone** — no compiled-standards `--config` arm; under the thin-executor strategy the in-process executor itself is removed (REQ-001) |
| `Options.ManifestDir` field + caller wiring | present | **removed** (REQ-002) |
| pack rule enforcement | fed into the in-process semgrep pass | **dispatched group-by-engine via `dispatchPackEngines`** (engine path, SPEC-031) — the only rule source; no compiled-standards source (REQ-004) |
| File-type routing (`RouteFile`) | from compiled `.manifest.json` or defaults | **default manifest** — `.go` unchanged (4 passes), non-Go → `[semgrep]` (additive, no pass dropped) (REQ-003) |
| `pkg/compile` reachable from core | orphaned at CLI, output checked in | **retired; output deleted** (REQ-005) |
| `STD-GO-001` source standard | present | **dropped** (REQ-006) |
| backstop-core's own enforcement | compiled `STD-GO-001` rules | **consumed `backstop/go-standards` pack** (REQ-007) |
| leftover `.backstop/rules/*.semgrep.yml` fallback | implicitly picked up | **never a rule source** (REQ-008) |

The pack rule source (consumed via the engine dispatch path) and the routing
default-manifest fallback are the surviving behavior this spec relies on; only
the compiled-standards arm — and, under the thin-executor strategy, the entire
in-process semgrep executor — is removed.

## Implementation

The change is a deletion plus a dogfood re-wire, in four coordinated edits.

### Edit 1 — No compiled-standards `--config` arm survives (REQ-001)

REQ-001's intent is that **no compiled-standards / `.standard.md` rule config
feeds the semgrep pass.** Under the thin-executor strategy this is realized by
the *absence* of any in-process semgrep `--config` arm:

- There is **no in-process `semgrepExecutor`** for a compiled-standards
  `manifestDir` arm to live on. ISSUE-018 removes the vestigial in-process
  semgrep path (`semgrepExecutor` / `semgrepJSON` / `EnsureSemgrep`) entirely; it
  scanned zero rules and is redundant with the engine dispatch. This spec's
  REQ-001 post-condition is therefore satisfied a fortiori: there is no
  `manifestDir` field, and no `if e.manifestDir != "" { args += --config }` arm,
  because there is no in-process semgrep executor at all.
- Real semgrep enforcement of pack rules runs through the data-driven engine
  path (`cmd/backstop/dispatchPackEngines` → SARIF, SPEC-031), not an in-process
  `--config` feed.
- **Tests must not assert an in-process semgrep invocation.** No `pkg/check`
  test may construct a `manifestDir`-bearing in-process semgrep feed or assert a
  standards-dir (`.backstop/rules`) `--config` against an in-process semgrep
  pass. The REQ-001 claims (CLM-002/003/023) assert the *absence*
  of the compiled-standards `--config` arm — via source/symbol self-checks that
  no such arm or `manifestDir`-style field survives — rather than the presence of
  any in-process semgrep invocation. No remaining `pkg/check` test may assert a
  standards-dir `--config`, or the green `go test ./...` guarantee (REQ-005,
  CLM-021) is defeated.

> The actual test rework (replacing any surviving `TestSemgrepExecutor_*` /
> `semgrepCalls`-based assertions with the absence-of-arm self-checks named in
> the repointed claims) is performed by the planner/implementer against these
> claims. The claims are corrected so the tests can be reworked to the surviving
> truth, not deleted under claim-drift.

### Edit 2 — Remove `Options.ManifestDir` and all callers (REQ-002)

- Remove the `ManifestDir string` field from `check.Options`
  (pkg/check/check.go).
- Remove any `manifestDir: opts.ManifestDir` assignment that fed a
  compiled-standards directory into a semgrep feed (the `goBuiltinExecutors` /
  `registry.go` construction sites). Note: under the thin-executor strategy the
  in-process `semgrepExecutor` and its construction sites are removed by
  ISSUE-018, so this assignment may already be gone; the surviving REQ-002
  obligation is solely that no caller wires a compiled-standards directory into
  rule config.
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
  → dispatchPackEngines(packs, ...)            # group rules by declared engine,
                                               #   run each engine → SARIF
                                               #   (SPEC-031; the ONLY rule source)
  → Options{ BackstopDir, ... }                # no ManifestDir, no
                                               #   compiled-standards rule config
  → routing: LoadManifest(.backstop/rules) → defaultManifest() when empty
```

There is **no in-process semgrep `--config` feed**: under the thin-executor
strategy the in-process `semgrepExecutor` is removed (ISSUE-018) and pack rule
enforcement flows through `dispatchPackEngines`. With zero packs, no rule source
is dispatched and no compiled-standards `--config` is assembled
(REQ-004 / CLM-002 / CLM-018).

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

   There is a SYMMETRIC in-package test coupling in `pkg/check` itself: any test
   that asserts an in-process semgrep invocation carrying (or NOT carrying) a
   standards-dir `--config` — historically `semgrep_executor_test.go`'s
   `semgrepExecutor{... manifestDir: ... }` literals and
   `containsConfigFor(call.args, ".../rules")` assertions, and the
   `semgrepCalls(runner)` + `t.Fatal("semgrep was never invoked")` pattern in the
   code-check tests. **Under the thin-executor strategy the in-process
   `semgrepExecutor` is removed entirely (ISSUE-018), so any test asserting that
   an in-process semgrep invocation OCCURS would fail at runtime** (the semgrep
   pass is never invoked). Edit 1 and the REQ-001 claims
   (CLM-002/003/023) mandate that these tests assert the *absence* of the
   compiled-standards `--config` arm via source/symbol self-checks, not the
   presence of any in-process semgrep invocation; CLM-023 enforces that no
   `pkg/check` test still requires a `manifestDir` or a standards-dir `--config`.
   An implementer who leaves a test asserting an in-process semgrep invocation
   produces a red build, defeating the single-source guarantee exactly as a
   left-behind `pkg/compile` would.

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
   CLM-018 guards this behaviorally — the sole production reader of
   `.backstop/rules/` (`LoadManifest`) collects only `*.manifest.json` and skips a
   leftover `STD-*.semgrep.yml`, so a populated rules dir holding only
   `*.semgrep.yml` yields the default manifest (zero rules, zero routes) and the
   removed arm is not resurrected.

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

6. **No in-process semgrep `--config` source survives, and tests must not
   demand one.** Under the thin-executor strategy there is no in-process semgrep
   executor and no compiled-standards `--config` source: with zero packs no rule
   source is dispatched. CLM-002 (production-source token scan) and CLM-018
   (behavioral: a populated `.backstop/rules/` is not a rule/route source) assert
   the *absence* of a
   compiled-standards `--config` source — they must NOT assert that an in-process
   semgrep pass is invoked, because none is. The pre-repoint trap was a test that
   did `semgrepCalls(runner)` then `t.Fatal` when the slice was empty: post-
   ISSUE-018 that fatal fires because the in-process semgrep pass is gone. The
   surviving assertion is "no compiled-standards `--config` source is wired in,"
   not "semgrep ran with an empty `--config`."

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
  single-source foundation. The pack-engine dispatch path (`dispatchPackEngines`)
  is where pack rule enforcement now flows under the thin-executor strategy.
- **ISSUE-018** — removes the vestigial in-process semgrep path
  (`semgrepExecutor` / `semgrepJSON` / `EnsureSemgrep`). Its deletion is why this
  spec's REQ-001 claims are repointed off any in-process semgrep invocation
  assertion (see Version History 1.1.0); the two artifacts are reconciled so they
  no longer conflict.
- [[feedback_integration_gap]] — two coexisting semgrep rule sources is the
  drift risk this collapse removes.
- [[feedback_loud_not_blocking]] — vacuous-green guard: routing must not be
  silently emptied by the removal (Sharp Edge 1).
- Code: pkg/check/check.go (`Options`, `goBuiltinExecutors`),
  pkg/check/registry.go, pkg/check/manifest.go (`LoadManifest`,
  `defaultManifest`, `RouteFile`), cmd/backstop/gate.go (Options construction),
  cmd/backstop/code_check.go (Options construction),
  cmd/backstop/pack_gate.go (`dispatchPackEngines` — the surviving rule-enforcement
  path), pkg/compile/ (retired compiler).

## Version History

- **2.2.0** (2026-08-02) — Status → `implemented` (founder-approved closure), plus the one
  contract removal that closure required. **Contract removed first, deliberately:** contract
  enforcement activates only at terminal status (`contractsAreDue`,
  `pkg/gate/step_testverify.go`), so a stale contracts block is inert while non-terminal and
  goes LIVE on this flip. The `cmd/backstop/gate.go` block's sole `provides` entry declared
  `(*realCodeChecker).runCheck` as PRESENT — the exact inverse of what this spec delivers.
  That symbol was deliberately DELETED by the SPEC-040 keystone cutover: a grep of non-test
  source returns zero matches for `realCodeChecker`, and
  `cmd/backstop/cutover_deletion_test.go`'s `TestCutover_RealCodeCheckerDeleted` asserts that
  absence as a standing guarantee. It was NOT replaced with a successor symbol, because
  nothing replaced `realCodeChecker` — the responsibility moved to pack-driven engine dispatch
  (`dispatchPackEngines`). Removing the entry emptied that file's `provides` list, so the
  whole `cmd/backstop/gate.go` block was removed rather than left as a hollow shell; its two
  `consumes` declarations went with it (`consumes` is not gate-enforced — `ExtractContractEntries`
  reads `provides` only — and one of the two, `mergePackRules`, no longer exists either). The
  entry was a leftover of **REQ-002**, retired at 2.0.0; the surviving REQ set
  (REQ-001/005/006/007) makes no promise about `cmd/backstop/gate.go`. The surviving
  `pkg/check/check.go` `semgrepExecutor` entry is an `absent: true` deletion guard and was
  re-verified: zero non-test hits for `semgrepExecutor`, and its scope file
  `pkg/check/check.go` still exists (an absence over a missing scope would be a loud config
  error, not a silent pass).

  **Delivery verified at closure — each item is an ABSENCE, so each is directly checkable:**
  `pkg/compile/` does not exist (CLM-021); `.backstop/rules/` does not exist (CLM-013);
  `standards/go/` does not exist (CLM-014); zero non-test hits for `ExtraSemgrepConfigs`,
  `semgrepExecutor`, and `EnsureSemgrep` — the only matches are the deletion-guard scanners in
  `pkg/check/semgrep_removal_test.go` (CLM-002/003/023). The dogfood pack is consumed
  REMOTELY, not vendored: `backstop.yml` declares `backstop-ai/go-standards: 1.2.1` and
  `backstop.lock` carries the matching entry with `git_ref: v1.2.1`,
  `source_type: git`, `source_coordinate: backstop-ai/go-standards` (CLM-016/017). All
  mandated tests exist as real functions — 12 tests across the 11 live claims
  (CLM-017 mandates two).

  **Known staleness recorded, NOT fixed:** the claim text throughout names the pack
  `backstop/go-standards`, while the tree uses `backstop-ai/go-standards` — the DIR-027 fleet
  migration renamed the namespace. Same pack, new namespace; the test constant was updated
  accordingly (`cmd/backstop/dogfood_pack_test.go:13`,
  `const dogfoodPackName = "backstop-ai/go-standards"`). The claims are deliberately left
  unrewritten: the rename is a namespace migration recorded elsewhere, not a change in what
  this spec delivered, and rewriting settled claim text at closure would obscure that.
  Likewise NOT touched: the `cmd/backstop/code_check.go` contract block names a file that no
  longer exists and consumes symbols that no longer exist (`check.Options`, `mergePackRules`).
  It is inert under contract enforcement (`provides: []`), so it does not block this flip and
  is left for a follow-up sweep rather than silently widened into this change. No requirement,
  claim, or verification config was altered — contract removal + lifecycle transition only.
- **2.1.1** (2026-07-07) — Migrated the spec-level target key from
  `implementation.package: pkg/check` to `implementation.subject: pkg/check`
  (ISSUE-047: the `test_substantiveness` noTarget guard is now language-neutral —
  a single Go-centric `implementation.package` becomes a language-neutral
  `subject` with optional per-claim overrides). The spec-level subject stays
  `pkg/check` because this spec's removal-proof `TestPkgCheck_*` tests
  (CLM-002/003/023) correctly target `pkg/check`. Added a per-claim
  `subject: pkg/gate` override to **CLM-015** ONLY: its mandated test
  `TestGate_SucceedsWithoutStandards` now lives in `cmd/backstop` and
  legitimately exercises `pkg/gate` (a gate run SUCCEEDS), not `pkg/check`, so the
  noTarget guard resolves it against `pkg/gate` while the removal-proof claims keep
  inheriting the spec-level `pkg/check`. CLM-015 remains a POSITIVE behavioral
  claim (not `kind: absence`). Key-rename + one per-claim override only
  (align-predating-artifacts); no requirement, claim text, contract, or other
  claim's subject altered.
- **2.1.0** (2026-07-06) — Marked the genuine structural/absence claims `kind: absence`
  (the per-claim annotation added by ISSUE-035) to reflect their structural nature and clear
  the `test_substantiveness` gate's noTarget ("does not call package check") false-flag on
  their mandated tests. Annotated **CLM-012** (no production import of `pkg/compile` —
  import-absence tree walk), **CLM-013** (compiled-standards artifacts absent from
  `.backstop/rules/` — file-absence stat), **CLM-014** (STD-GO-001 source standard absent —
  file-absence stat), **CLM-021** (`pkg/compile` package directory absent — dir-absence stat),
  **CLM-016** (dogfood `backstop.yml`/installed-pack config-state invariant), and **CLM-017**
  (`backstop.lock` verifies + stale `slotly/go-standards` entry absent — lock-state/absence
  verification). Each mandated test asserts absence of a symbol/dir/artifact or verifies
  repo config/lock state and by design does not exercise `pkg/check` — exactly the case the
  annotation exists for. **CLM-015 (`TestGate_SucceedsWithoutStandards`) was deliberately NOT
  annotated:** its claim is BEHAVIORAL ("a gate / code-check run succeeds … on a project with
  no STD-GO-001 artifact"), and after the SPEC-040/ISSUE-018 cutover removed the
  `*realCodeChecker.runCheck` + `LoadManifest` routing calls the test was hollowed to a bare
  temp-dir scaffold that stats `.backstop`. It is not a structural-absence test but a
  behavioral test that lost its behavioral assertion; annotating it `kind: absence` would
  paper over a real substantiveness signal, so it is left flagged for follow-up. Annotation-only
  change (align-predating-artifacts); no requirement, claim text, test, or contract altered.
- **2.0.0** (2026-07-05) — Retired the requirements + claims whose subject ISSUE-018 (authorized
  thin-executor eradication) deleted outright. Removed REQ-002 (`Options.ManifestDir` field +
  caller wiring), REQ-003 (`Manifest.RouteFile` default-manifest routing), REQ-004 (locus-A
  single-source semgrep `--config`), and REQ-008 (flag-day no `.backstop/rules/` fallback) — all
  governed the now-deleted in-process check engine / manifest layer — together with their 9
  claims (CLM-004/006/024/008/009/022/020/011/018), whose mandated `TestOptions_*` /
  `TestRunCheckOptions_*` / `TestRouting_*` / `TestCodeCheck_PackOnly_*` / `TestNoFallback_*`
  functions were deleted with the engine. The live packs-only requirements REQ-001, REQ-005,
  REQ-006, REQ-007 and all their claims are unchanged. Removing all of a requirement's claims
  orphans it (`spec/requirement-uncovered`) and an emptied claim is invalid
  (`spec/claim-tests-empty`), so the requirements were removed alongside their claims. Recorded
  openly per align-predating-artifacts.
- **1.4.0** (2026-07-05) — Retired the stale `pkg/check/check.go` provides `Options` contract:
  ISSUE-018 (authorized thin-executor eradication) deleted the `type Options struct` entirely
  (the note's "remaining fields post-ISSUE-018" premise no longer holds — there is no Options),
  so the present-signature promise was a stale red under `contract_signature`. The
  `semgrepExecutor` absence guard in the same block is unchanged. Contract-only realignment
  (align-predating-artifacts); no requirement, claim, or design change.
- **1.3.0** (2026-06-21) — Completes the thin-executor reconciliation the 1.1.0/1.2.0
  passes started: those passes repointed the REQ-001 / REQ-004 claims off any
  in-process semgrep invocation (the executor ISSUE-018 fully deletes), but MISSED
  the **REQ-008 pair (CLM-018, CLM-019)**, which still carried the pre-strategy
  assumption that a deleted in-process semgrep invocation's recorded runner args
  could be inspected for a `.backstop/rules/` path. This pass reconciles that pair
  and de-duplicates the REQ-001 claim set. Changes:
  (1) **CLM-018 REPOINTED** off the "recorded runner args contain no `.backstop/rules/`
  path" assertion (no in-process semgrep invocation exists to record args) to the
  surviving BEHAVIORAL property: the SOLE production reader of `.backstop/rules/` is
  `LoadManifest` (pkg/check/manifest.go, invoked from check.go via
  `LoadManifest(filepath.Join(opts.BackstopDir, "rules"))`), which collects ONLY
  `*.manifest.json` files and skips every other entry — so a leftover
  `STD-*.semgrep.yml` is never picked up, and a `.backstop/rules/` dir holding only
  `*.semgrep.yml` (no `.manifest.json`) yields the built-in default manifest (zero
  rules, zero routes). This is a distinct, non-vacuous behavioral assertion over
  `LoadManifest` — it fails if `LoadManifest` is ever made to treat `.semgrep.yml`
  as a source — NOT a third copy of the CLM-002/003 production-source token scans.
  CLM-018 keeps the mandated test name `TestNoFallback_PopulatedRulesDirNotASource`
  (inherited from CLM-019, since that name still describes the surviving behavioral
  assertion); the old CLM-018 test name `TestNoFallback_LeftoverCompiledRulesIgnored`
  (which named the deleted in-process-semgrep arg-recording check) is dropped.
  (2) **CLM-019 RETIRED** (folded into CLM-018). It directly asserted "with zero
  packs, semgrep still runs with no `--config`" — i.e. that the deleted in-process
  semgrep pass STILL RUNS; that premise has no surviving subject post-ISSUE-018
  (pack rules dispatch via `dispatchPackEngines`, no in-process semgrep pass exists).
  Its distinct intent — a populated `.backstop/rules/` does not become an implicit
  second rule source — is identical to repointed CLM-018's behavioral property and
  is fully covered there. Retired rather than repointed to avoid a vacuous "assert
  the deleted semgrep pass runs" test and a near-duplicate claim. REQ-008 now
  retains exactly one claim (CLM-018), behavioral and distinct.
  (3) **CLM-001 RETIRED** as a near-duplicate of CLM-002/CLM-003. As scoped, CLM-001
  (`TestPkgCheck_NoCompiledStandardsConfigArm`) is a THIRD pkg/check
  production-source absence token-scan asserting the same structural absence ("no
  `--config` from a standards dir; no `manifestDir`-style field; no in-process
  semgrep executor") that CLM-002 (`TestPkgCheck_NoResidualStandardsConfigWhenNoPacks`
  — scans for `ExtraSemgrepConfigs` / `manifestDir` / `ManifestDir` absent) and
  CLM-003 (`TestPkgCheck_NoManifestDirFieldOnSemgrepFeed` — scans for
  `semgrepExecutor` / `manifestDir` / `ManifestDir` absent) already make
  token-for-token. Writing the CLM-001 test would mandate a redundant third token
  scan over the same source — the near-duplicate-claim pattern this reconciliation
  exists to eliminate. CLM-001's one distinct token (a `--config` argument sourced
  from a `.backstop/rules` standards dir) is folded into CLM-002's scan set. REQ-001
  retains CLM-002, CLM-003, CLM-023 — no requirement is left claim-less.
  (4) **Minimal non-overlapping claim set for "the compiled-standards /
  in-process-semgrep path is gone"** after this pass: **CLM-002**
  (`TestPkgCheck_NoResidualStandardsConfigWhenNoPacks`) — production-source token
  scan: no `ExtraSemgrepConfigs` / `manifestDir` / `ManifestDir` / standards-dir
  `--config` token survives; **CLM-003** (`TestPkgCheck_NoManifestDirFieldOnSemgrepFeed`)
  — production-source/symbol scan: no `semgrepExecutor` type and no `manifestDir`
  field survive; **CLM-023** (`TestNoTestRequiresManifestDirOrStandardsConfig`) —
  TEST-source scan: no test reintroduces a `manifestDir:` literal or a standards-dir
  `--config` assertion; **CLM-011** (`TestCodeCheck_PackOnly_RulesDispatchViaEnginePath`)
  — behavioral: one installed pack's rules dispatch via `dispatchPackEngines`, not an
  in-process `--config` feed; **CLM-018** (`TestNoFallback_PopulatedRulesDirNotASource`)
  — behavioral: a populated `.backstop/rules/` (only `*.semgrep.yml`) is not a
  rule/route source (`LoadManifest` returns the default manifest). Each asserts a
  distinct, writable, non-vacuous property; none overlaps another.
  No requirement, contract, or other claim is changed; REQ-001/REQ-004/REQ-008 text
  is unchanged (the repoint is at the claim/realization layer, consistent with the
  1.1.0/1.2.0 strategy). All other claims (CLM-002…CLM-004, CLM-006, CLM-008…CLM-017,
  CLM-020…CLM-025) are unchanged.
- **1.2.0** (2026-06-21) — Completes the thin-executor alignment the 1.1.0 repoint
  started, reconciling this spec with **ISSUE-018**'s FULL deletion of the in-process
  `semgrepExecutor` (the 1.1.0 prose acknowledged the deletion but several REQ/CLM/
  contract elements still assumed the executor SURVIVED minus its `ManifestDir` field).
  Changes:
  (1) **Contract fix (the gate-blocking conflict).** The `semgrepExecutor` contract
  entry used `signature: "absent"` — a literal expected-signature string the
  `contract_signature` gate step tried to MATCH, producing `symbol semgrepExecutor not
  found`. Converted to the proper absence-assertion form: `absent: true` (maps to
  `ContractEntry.Absent` per pkg/gate/step_contract.go — passes when the symbol is gone,
  fails if it reappears) plus a documentary `signature` (the schema requires a non-empty
  signature on every provides entry; the gate ignores it for an absent entry). No
  contract now mandates the deleted `semgrepExecutor` as PRESENT. (SPEC-031 was checked:
  it references `semgrepExecutor` only in REQ/prose/References as "the thing being
  replaced," never in a `provides` contract, so it needs no change.)
  (2) **Options contract notes** corrected to the actual post-ISSUE-018 `check.Options`
  struct (Mode, FilePath, BackstopDir, Timeout, ProjectDir, Language, Config, Files);
  dropped `PinnedSemgrepVersion`, `GolangciLintAvailable`, and `ExtraSemgrepConfigs`,
  which ISSUE-018 removed (or which never existed on the struct).
  (3) **REQ-001 / REQ-002 / impl summary** reframed off the "keep the executor, drop one
  field" premise to "the in-process `semgrepExecutor` is deleted ENTIRELY by ISSUE-018;
  a fortiori no `manifestDir` field and no compiled-standards `--config` arm." REQ-002's
  surviving obligation is solely the `check.Options` field and its gate/code-check caller
  wiring — the executor-side `manifestDir: opts.ManifestDir` construction assignments are
  gone a fortiori.
  (4) **CLM-005, CLM-007, CLM-010 RETIRED** (not repointed). Reasoning — keeping the
  substrate honest (no vacuous green, no redundant overlapping claims): **CLM-005** (build
  a `semgrepExecutor` without a `manifestDir`) has no surviving subject — ISSUE-018 deletes
  the executor and its construction sites, so there is no build to assert on; its structural
  intent is fully covered by CLM-003 (`TestPkgCheck_NoManifestDirFieldOnSemgrepFeed`).
  **CLM-007** (code-check Options carry no compiled-standards manifest dir) and **CLM-010**
  (zero-packs run feeds no compiled-standards `--config`) had been reworked at 1.1.0 to
  inspect a captured `check.Options` via the `checkRunFn` stub WITHOUT calling `pkg/check`,
  so the `test_substantiveness` gate flagged both "does not call package check." Repointing
  them to actually call `check.Run` would manufacture a VACUOUS test: post-ISSUE-018 the
  `Options.ManifestDir` field and the in-process semgrep `--config` arm are structurally
  gone, so no code path could ever produce the forbidden wiring and the assertion could
  never fail. Their surviving properties are STRUCTURAL absences already covered — CLM-007
  by CLM-004 (Options field gone) + CLM-001/003 (no compiled-standards arm); CLM-010 by
  CLM-002 (no residual standards source with zero packs) + CLM-019 (populated rules dir not
  a source), with REQ-004's positive single-source proof carried substantively by CLM-011
  (one pack dispatches via the engine path). REQ-002 retains CLM-004/006/024; REQ-004
  retains CLM-011 — no requirement is left claim-less. The retired claims' mandated tests
  (`TestBuildExecutors_SemgrepHasNoManifestDir`, `TestCodeCheckOptions_NoManifestDir`,
  `TestCodeCheck_NoPacks_NoSemgrepConfig`) are removed by the planner/implementer so nothing
  mandates a retired test; `PLAN-ISSUE-018` is updated to match. All other requirements and
  claims (REQ-003, REQ-005…REQ-008; CLM-001…CLM-004, CLM-006, CLM-008…CLM-009, CLM-011…CLM-025)
  are unchanged.
- **1.1.0** (2026-06-21) — Deliberate cross-artifact repoint reconciling this spec
  with the now-coherent thin-executor strategy and resolving the conflict with
  **ISSUE-018**. ISSUE-018 removes the vestigial in-process semgrep pass
  (`semgrepExecutor` / `EnsureSemgrep`), which scanned zero rules and is redundant
  with the engine dispatch — so the SPEC-030 claims that mandated tests asserting
  an in-process semgrep *invocation* (those tests do
  `len(semgrepCalls(runner)) == 0 → t.Fatal`) would fail at runtime post-ISSUE-018.
  This spec's INTENT (no compiled-standards / `.standard.md` rule config feeds the
  checks) SURVIVES and is more completely realized: there is no compiled-standards
  `--config` arm because no compiled-standards manifest directory is wired into
  check `Options` and pack rules dispatch group-by-engine via `dispatchPackEngines`
  rather than an in-process semgrep `--config` feed. Repointed claims, none of
  which may now assert that an in-process semgrep invocation occurs:
  **CLM-001** (was: `semgrepExecutor` assembles pack-only `--config` →
  `TestSemgrepExecutor_ConfigArgsFromPacksOnly`; now: no compiled-standards
  `--config` arm survives in pkg/check →
  `TestPkgCheck_NoCompiledStandardsConfigArm`);
  **CLM-002** (was: `TestSemgrepExecutor_NoConfigWhenNoPacks`; now: no residual
  standards `--config` source with zero packs →
  `TestPkgCheck_NoResidualStandardsConfigWhenNoPacks`);
  **CLM-003** (was: `semgrepExecutor` struct field set →
  `TestSemgrepExecutor_StructHasNoManifestDir`; now: no `manifestDir`-style field
  on any in-process semgrep feed, the executor type itself being gone →
  `TestPkgCheck_NoManifestDirFieldOnSemgrepFeed`);
  **CLM-007** (`TestCodeCheckOptions_NoManifestDir`, name kept) repointed to assert
  Options carry no compiled-standards manifest directory, dropping the in-process
  semgrep end-to-end assertion;
  **CLM-010** (`TestCodeCheck_NoPacks_NoSemgrepConfig`, name kept) repointed to
  assert no compiled-standards rule-config source with zero packs, dropping the
  "in-process semgrep invoked" assertion;
  **CLM-011** repointed from "in-process semgrep invoked with the pack's rule
  paths as `--config`" to "pack rules dispatch via the engine path, not an
  in-process semgrep `--config` feed," and its mandated test RENAMED
  `TestCodeCheck_PackOnly_SemgrepConfigIsPackPathsOnly` →
  `TestCodeCheck_PackOnly_RulesDispatchViaEnginePath` (old name described a feed
  that no longer exists). **CLM-023** clarified to the same surviving truth (name
  kept). The `semgrepExecutor` contract entry is marked `absent`. The actual test
  rework is performed by the planner/implementer against these repointed claims —
  the claims are corrected so the tests can be reworked, not deleted under
  claim-drift. All other requirements, claims, and the dogfood/enforcement-transfer
  spine (REQ-005…REQ-008, CLM-012…CLM-025) are unchanged.
- **1.0.0** (2026-06-16) — Initial spec authored from BUNDLE-010
  (pluggable-pack-engines), Spec Seed 1 (Pillar 2): packs-only native-standards
  removal.
