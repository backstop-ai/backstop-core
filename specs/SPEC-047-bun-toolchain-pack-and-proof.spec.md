---
title: "Bun Toolchain Pack And Two-Surface Proof"
number: SPEC-047
created: "2026-06-28"
status: implemented
schema_version: spec/v1
spec_version: 1.2.0

implementation:
  summary: >
    BUNDLE-012 Spec Seed 5 of 5 — the INTEGRATION + PROOF spec. It is the closing
    deliverable that consumes the four sibling seeds' contracts as FIXED interfaces and
    proves the whole language-neutral consumer end to end on a real foreign-language
    stack. It does FOUR coupled things. (1) AUTHORS `backstop/bun-toolchain` (bundle
    REQ-006) — an ORDINARY pack in its OWN repo (packs are always external; the in-repo
    `.backstop/packs` tree is gitignored), hand-authored from the proven
    `backstop/go-toolchain` pack.yml template (OQ-5: the pack-authoring CLI is broken, so
    hand-author; that reboot is Track-B / ISSUE-032, out of scope). Engines: `oxlint`
    (lint) · `prettier --check` (FORMAT modeled as a LINT-category SARIF findings engine,
    DD-3 — format ≈ lint, NO new gate dimension) · `tsc --noEmit` / bun typecheck (build)
    · `bun test` (test) · `bun test --coverage --coverage-reporter=lcov` (coverage) with a
    pack-relative `scripts/coverage-to-records.sh` lcov convert. All commands ride the
    declared-engine TRUST substrate (oxlint/bun/tsc/prettier allowlisted) — zero baked
    commands. (2) DECLARES the SPEC-043 `classification` block (source/test globs for a
    Bun/TS project: source `**/*.ts`,`**/*.tsx`; test `**/*.test.ts`,`**/*.spec.ts`) and
    EMITS the SPEC-044 bun producer RECORD-SHAPE: for each measured file the lcov convert
    emits TWO `check.CoverageRecord`s sharing one `Path` — one `metric: "line"` (raw
    LF/LH counts) and one `metric: "branch"` (raw BRF/BRH counts) — through the EXISTING
    `check.CoverageRecord` type and `check.ParsePackCoverage` parser, NO new type/field
    /schema fork. (3) PROVES on TWO surfaces (bundle REQ-009): (a) an IN-REPO STATIC
    testdata fixture (extends `cmd/backstop/testdata/`) — pack.yml + `.ts` source/test
    files + a PRE-CAPTURED lcov, exercised with the runner STUBBED so the consumer + glob
    classification + line/branch record parsing are proven with ZERO `bun` dependency in
    backstop-core's Go CI; and (b) an EXTERNAL EXECUTED gate on the real Bun fork
    (`backstop-runtime`) with MINIMAL wiring (a `backstop.yml` declaring the bun pack +
    `pack add`), going RED on a seeded defect and green when fixed — the REQUIRED
    acceptance, kept OUT of the Go CI via a skip-when-bun-absent guard. (4) Performs the
    RATCHET → BLOCK flip (bundle REQ-008): as each Pillar-A site (SPEC-043 coverage
    measurable-path + go-package matchers, SPEC-045 test-verify discovery) is de-Go'd it is
    un-grandfathered from the committed `.backstop/baseline.json`; once all three are clean,
    `backstop/self`'s neutral-spine enforcement flips to `block` with a ZERO baseline so any
    future baked-language regression is blocked outright. Because `backstop/self`'s neutral-spine
    findings RIDE the SHARED `pack_engines` gate dimension alongside `go-standards`/`go-toolchain`
    style debt, and the enforcement policy (`pkg/config` `Enforcement.Policy` →
    `pkg/gate.ApplyPolicy`) currently keys STRICTLY per-dimension, this spec also EXTENDS the
    policy with finer-than-dimension (per-PACK / per-rule-SOURCE) granularity so `backstop/self`
    can be flipped to `block` + zero-baseline INDEPENDENTLY, without disturbing the baselined
    style debt the other two packs carry on the same dimension (bundle REQ-008). This spec edits
    NO sibling core logic — it consumes `gate.SourceClassifier`/`pack.Manifest.Classification`
    (SPEC-043),
    the `(path, metric)` index + per-metric thresholding (SPEC-044), the pack-declared
    test-glob discovery (SPEC-045), and the uniform toolchain-pack dispatch with no
    `language:` field (SPEC-046) as fixed contracts; its only owned surfaces are the bun pack
    DATA, the in-repo testdata fixture, the external acceptance test, the trust allowlist
    entries, the per-PACK/per-rule-SOURCE enforcement-policy extension (`pkg/config` +
    `pkg/gate.ApplyPolicy`), and the dogfood baseline + `backstop/self` enforcement config.
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/ ./pkg/gate/ ./pkg/check/ ./pkg/config/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      `backstop/bun-toolchain` MUST be an ORDINARY pack hand-authored from the
      `backstop/go-toolchain` pack.yml template (its own repo; the in-repo
      `.backstop/packs` copy is gitignored DATA), declaring EXACTLY these five engines,
      each routed to its declared gate dimension via the engine's `gate_type`: `oxlint`
      (lint) → `gate_type: lint`; `prettier --check` (FORMAT) → `gate_type: lint`,
      modeled as a LINT-CATEGORY SARIF findings engine per DD-3 — format ≈ lint, so it
      rides the EXISTING lint dimension and introduces NO new "format" gate dimension;
      `tsc --noEmit` / bun typecheck → `gate_type: build`; `bun test` → `gate_type: test`;
      `bun test --coverage --coverage-reporter=lcov` → `gate_type: coverage`. It is
      PROHIBITED to add a "format" (or any new) gate dimension for prettier, and PROHIBITED
      to route prettier anywhere but the lint dimension. Every engine command MUST run
      through the declared-engine TRUST substrate (the tools `oxlint`, `bun`, `tsc`,
      `prettier` allowlisted), NEVER as a baked-into-the-binary command string. The pack
      carries mechanism only — no opinionated coding-standards rules. (DD-2, DD-3)
    supports: language-neutral-consumer-ts-toolchain:REQ-006@1.0.0
  - id: REQ-002
    text: >
      The bun pack MUST declare the SPEC-043 `classification` block as DATA in its
      `pack.yml`: `classification.source` = [`**/*.ts`, `**/*.tsx`] and
      `classification.test` = [`**/*.test.ts`, `**/*.spec.ts`]. These MUST parse onto the
      SPEC-043 `pack.Manifest.Classification` field and feed the SPEC-043
      `gate.SourceClassifier` (consumed as a FIXED contract — this spec does NOT redefine
      either). Under the bun pack's globs ALONE, the measurable-source matrix MUST hold:
      a `.ts` source file is MEASURABLE; a `.tsx` source file is MEASURABLE; a `.test.ts`
      file is NOT measurable (test-wins-on-overlap); a `.spec.ts` file is NOT measurable;
      a `.go` file is NOT measurable (the bun pack declares NO Go glob — proving no baked
      Go literal leaks across packs); and an unclassified file (e.g. `**/*.md`) is NOT
      measurable. (DD-1; consumes SPEC-043)
    supports: language-neutral-consumer-ts-toolchain:REQ-006@1.0.0
  - id: REQ-003
    text: >
      The bun coverage convert (`scripts/coverage-to-records.sh`) MUST implement the
      SPEC-044 bun producer RECORD-SHAPE contract (SPEC-044 REQ-006), consumed here as a
      FIXED interface: reading an lcov `.info` report, for EACH measured source file it
      MUST emit TWO `check.CoverageRecord`s sharing the SAME `Path` — one with
      `metric: "line"` carrying raw `{covered: LH, total: LF}` counts and one with
      `metric: "branch"` carrying raw `{covered: BRH, total: BRF}` counts (raw counts
      only — it is PROHIBITED to pre-compute a percentage, aggregate the two metrics, or
      collapse them into a single record). Both records MUST flow through the EXISTING
      `check.CoverageRecord` type and `check.ParsePackCoverage` parser with NO new type,
      NO new field, and NO schema fork, and MUST index under the SPEC-044 `(path, metric)`
      key with NO collision. A file with no branchable code (lcov `BRF: 0`) MUST emit a
      `branch` record with `total: 0` (the N/A cell, never coerced to 0%) while its `line`
      record is still measured. (DD-3; consumes SPEC-044)
    supports: language-neutral-consumer-ts-toolchain:REQ-006@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-004
    text: >
      An IN-REPO STATIC testdata fixture (extending `cmd/backstop/testdata/`) MUST exist
      containing: a `backstop.yml` declaring `backstop/bun-toolchain` as an ORDINARY pack
      in `packs:` with NO `language:` field (per SPEC-046); the bun `pack.yml` (the five
      engines + the classification block); a few `.ts` source files and `.test.ts` test
      files; and a PRE-CAPTURED lcov `.info` output. An end-to-end gate over this fixture
      with the runner STUBBED (the `bun test --coverage` command's output fed from the
      pre-captured lcov) MUST: classify the `.ts` source as MEASURABLE SOURCE via the
      declared globs end-to-end; run the real `coverage-to-records.sh` convert over the
      pre-captured lcov and measure the file's `line` AND `branch` coverage; and invoke NO
      real `bun`/`oxlint`/`tsc`/`prettier` (ZERO `bun` dependency in backstop-core's Go CI
      — only the POSIX-shell convert runs, over DATA). A seeded-defect variant (a changed
      `.ts` source file with NO coverage record, OR a below-threshold record) MUST RED the
      gate — proving the consumer is NOT vacuous-green on a non-Go file. (DD-1, DD-2;
      consumes SPEC-043/044/046)
    supports: language-neutral-consumer-ts-toolchain:REQ-009@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      An EXTERNAL EXECUTED gate on the real Bun fork (`backstop-runtime`) MUST be the
      REQUIRED acceptance: with MINIMAL wiring — a `backstop.yml` declaring
      `backstop/bun-toolchain` in `packs:` and the pack `pack add`'d from its own repo —
      `backstop gate` MUST go RED on a seeded defect (a lint / format / type / test /
      coverage failure) and green when the defect is fixed (RED-then-green). This
      acceptance MUST be expressed as a guarded test that SKIPS (never FAILS) when the
      `bun` toolchain is absent / the acceptance env var is unset, so the real `bun` /
      `oxlint` toolchain stays OUT of backstop-core's Go CI while still being an executed
      end-to-end proof in the fork's own environment. The fork gates PACKS-ONLY (no baked
      language path). Productionizing the fork's CI is OUT of scope (only the minimal
      single-acceptance wiring is in scope). Because the Go-CI guard makes this REQUIRED
      acceptance auto-skip, it MUST NOT be vacuously skipped forever: satisfying it requires a
      documented manual run on the fork with the captured RED-then-green `backstop gate` output
      (the seeded-defect red, then the fixed green) recorded as run-evidence in the
      implementation's verification log — the executed proof, not the skipped Go-CI stub, is
      what closes REQ-005. (DD-1, DD-2)
    supports: language-neutral-consumer-ts-toolchain:REQ-009@1.0.0
  - id: REQ-006
    text: >
      The RATCHET → BLOCK flip (bundle REQ-008), sequenced AFTER and DEPENDENT ON the
      Pillar-A de-Go work landing (SPEC-043 coverage measurable-path + go-package/`./...`
      matchers, SPEC-045 test-verify discovery, SPEC-046 bridge/`language:` retirement).
      As each Pillar-A site is de-Go'd, its `backstop/self` neutral-spine
      (`no-language-literal-on-neutral-spine`) findings MUST be UN-grandfathered from the
      committed `.backstop/baseline.json` (ratchet), sequenced by correctness impact:
      (1) coverage measurable-path (the vacuous-green hole) → (2) test-verify discovery →
      (3) the go-package / `./...` matchers. Once ALL THREE sites are clean (zero
      neutral-spine findings remain), `backstop/self`'s enforcement MUST flip to `block`
      with a ZERO baseline (no grandfathering of any neutral-spine finding) so any FUTURE
      baked-language regression on the neutral spine is blocked OUTRIGHT as net-new. This
      flip MUST be expressed through the REQ-007 per-PACK/per-rule-SOURCE enforcement key
      scoped to `backstop/self`, NOT by zero-baselining the whole shared `pack_engines`
      dimension (which would wrongly block `go-standards`/`go-toolchain`'s baselined style
      debt). It is PROHIBITED to perform the terminal flip while any of the three sites still
      flags (the flip is the closing step, gated on the others). A deliberately reintroduced
      baked `.go`/`_test.go`/Go-package literal on a neutral-spine site MUST RED the gate after
      the flip. (DD-1)
    supports: language-neutral-consumer-ts-toolchain:REQ-008@1.0.0
  - id: REQ-007
    text: >
      The enforcement policy MUST gain FINER-THAN-DIMENSION granularity — a per-PACK (or
      per-rule-SOURCE) enforcement key — so `backstop/self`'s neutral-spine findings can be
      flipped to `block` + ZERO baseline INDEPENDENTLY of the other packs that share the
      `pack_engines` gate dimension. This is a REQUIRED deliverable OF THIS SPEC, not a punted
      implementation detail. Today the policy table (`pkg/config` `Enforcement.Policy` →
      `pkg/gate.ApplyPolicy`) keys STRICTLY per-dimension (by step/`gate_type` name), and
      `backstop/self`'s `no-language-literal-on-neutral-spine` findings ride the SHARED
      `pack_engines` dimension alongside `go-standards`/`go-toolchain`'s LEGITIMATE baselined
      STYLE debt; a `pack_engines: {level: block, baseline: false}` entry would therefore
      wrongly BLOCK that inherited style debt. The extension MUST let a policy entry SCOPE to a
      pack/source (keyed off the existing `gate.Violation.SourcePack`), so that the level +
      baseline of a scoped entry apply ONLY to that pack's findings within the dimension, and
      every OTHER pack's findings on the same dimension are UNAFFECTED (they keep their
      default — or separately-configured — grandfathered policy). With `backstop/self` flipped
      to `block` + zero baseline via this key: a FRESH neutral-spine (baked-language) finding
      from `backstop/self` MUST BLOCK the gate; a baselined `go-standards` style finding on
      `pack_engines` MUST NOT block (stays grandfathered); and a baselined `go-toolchain` style
      finding on `pack_engines` MUST NOT block (stays grandfathered). It is PROHIBITED to
      deliver the REQ-006 flip by zero-baselining the whole shared `pack_engines` dimension. The
      extension MUST be backward compatible: an unscoped (dimension-only) policy entry keeps its
      current behavior. (DD-1; verified against `pkg/config/config.go` `Enforcement.Policy` +
      `pkg/gate/policy.go` `ApplyPolicy`)
    supports: language-neutral-consumer-ts-toolchain:REQ-008@1.0.0
    follows: STD-GO-001:GO-010

claims:
  # REQ-001 — engine declaration matrix (5 engines) + format-is-not-a-dimension + trust substrate
  - id: CLM-001
    requirement: REQ-001
    text: The bun pack declares an `oxlint` engine routed to the lint dimension (gate_type lint)
    tests:
      - TestBunPack_OxlintEngineDeclaredAsLint
  - id: CLM-002
    requirement: REQ-001
    text: The bun pack declares a `prettier --check` engine routed to the LINT dimension (gate_type lint) as a lint-category SARIF findings engine — format ≈ lint per DD-3
    tests:
      - TestBunPack_PrettierCheckDeclaredAsLintCategoryFormat
  - id: CLM-003
    requirement: REQ-001
    text: The bun pack declares a `tsc --noEmit` / bun typecheck engine routed to the build dimension (gate_type build)
    tests:
      - TestBunPack_TscTypecheckDeclaredAsBuild
  - id: CLM-004
    requirement: REQ-001
    text: The bun pack declares a `bun test` engine routed to the test dimension (gate_type test)
    tests:
      - TestBunPack_BunTestDeclaredAsTest
  - id: CLM-005
    requirement: REQ-001
    text: The bun pack declares a `bun test --coverage --coverage-reporter=lcov` engine routed to the coverage dimension (gate_type coverage) with a pack-relative convert and lcov stdout_artifact
    tests:
      - TestBunPack_BunCoverageLcovDeclaredAsCoverage
  - id: CLM-006
    requirement: REQ-001
    text: DENYLIST — prettier introduces NO new "format" gate dimension; the gate's dimension set after loading the bun pack is unchanged (lint/build/test/coverage), proving format rides the existing lint dimension and no separate format dimension is created
    tests:
      - TestBunPack_PrettierIntroducesNoFormatGateDimension
  - id: CLM-007
    requirement: REQ-001
    text: Every bun engine command runs through the declared-engine trust substrate — `oxlint`/`bun`/`tsc`/`prettier` are allowlisted and dispatched as pack-declared commands, NOT as baked-into-the-binary command strings
    tests:
      - TestBunPack_EngineCommandsRunThroughDeclaredTrustSubstrate
  # REQ-002 — classification block + measurable-source matrix (6 file types)
  - id: CLM-008
    requirement: REQ-002
    text: The bun pack's `classification` block (source `**/*.ts`,`**/*.tsx`; test `**/*.test.ts`,`**/*.spec.ts`) parses onto the SPEC-043 pack.Manifest.Classification field intact
    tests:
      - TestBunPack_ClassificationGlobsParseOntoManifest
  - id: CLM-009
    requirement: REQ-002
    text: A `.ts` source file is MEASURABLE source under the bun pack's globs (matches a source glob, no test glob)
    tests:
      - TestBunClassifier_TsSourceIsMeasurable
  - id: CLM-010
    requirement: REQ-002
    text: A `.tsx` source file is MEASURABLE source under the bun pack's globs
    tests:
      - TestBunClassifier_TsxSourceIsMeasurable
  - id: CLM-011
    requirement: REQ-002
    text: A `.test.ts` file is NOT measurable — it matches both a source glob and a test glob, and test-wins-on-overlap
    tests:
      - TestBunClassifier_TestTsNotMeasurable
  - id: CLM-012
    requirement: REQ-002
    text: A `.spec.ts` file is NOT measurable — it matches the test glob, so it carries no coverage requirement
    tests:
      - TestBunClassifier_SpecTsNotMeasurable
  - id: CLM-013
    requirement: REQ-002
    text: A `.go` file is NOT measurable under the bun pack ALONE (the bun pack declares no Go glob), proving no baked Go literal leaks across packs into the classifier
    tests:
      - TestBunClassifier_GoFileNotMeasurableUnderBunPack
  - id: CLM-014
    requirement: REQ-002
    text: An unclassified file (e.g. `README.md`) matching neither a source nor a test glob is NOT measurable
    tests:
      - TestBunClassifier_UnclassifiedMarkdownNotMeasurable
  # REQ-003 — lcov convert: two records per file, line+branch, canonical type, no schema fork
  - id: CLM-015
    requirement: REQ-003
    text: For each measured source file the convert emits EXACTLY TWO records sharing the same Path — one metric "line" and one metric "branch" — never a single collapsed record
    tests:
      - TestBunCoverageConvert_EmitsLineAndBranchPerFileSharingPath
  - id: CLM-016
    requirement: REQ-003
    text: The line record carries raw counts from lcov LF/LH (covered=LH, total=LF), not a pre-computed percentage
    tests:
      - TestBunCoverageConvert_LineCountsFromLcovLfLh
  - id: CLM-017
    requirement: REQ-003
    text: The branch record carries raw counts from lcov BRF/BRH (covered=BRH, total=BRF), not a pre-computed percentage
    tests:
      - TestBunCoverageConvert_BranchCountsFromLcovBrfBrh
  - id: CLM-018
    requirement: REQ-003
    text: A file with no branchable code (lcov BRF 0) emits a branch record with total 0 (the N/A cell, never coerced to 0%) while its line record is still measured
    tests:
      - TestBunCoverageConvert_NoBranchableCodeEmitsTotalZeroBranchRecord
  - id: CLM-019
    requirement: REQ-003
    text: The emitted records parse through the EXISTING check.ParsePackCoverage into canonical check.CoverageRecords with NO new type, NO new field, and NO schema fork (DisallowUnknownFields accepts the bun output)
    tests:
      - TestBunCoverageConvert_RecordsParseThroughCanonicalParsePackCoverageNoNewType
  - id: CLM-020
    requirement: REQ-003
    text: The bun-shaped line+branch records index under the SPEC-044 (path, metric) key for one Path with NO collision — both metrics survive
    tests:
      - TestBunCoverage_LineAndBranchIndexUnderPathMetricNoCollision
  - id: CLM-021
    requirement: REQ-003
    text: DENYLIST — the convert emits raw counts only and never an aggregated/averaged single value or a pre-computed percent; feeding line 95/100 and branch 60/100 yields two raw-count records, not one rolled-up number
    tests:
      - TestBunCoverageConvert_RawCountsNotPrecomputedPercentNoAggregation
  # REQ-004 — in-repo static fixture proof (stubbed runner, zero bun dependency)
  - id: CLM-022
    requirement: REQ-004
    text: The in-repo static fixture exists with a backstop.yml declaring backstop/bun-toolchain as an ordinary pack and NO language field, the bun pack.yml, .ts source + .test.ts files, and a pre-captured lcov .info
    tests:
      - TestBunFixture_StaticTestdataExistsWithPackTsFilesAndPrecapturedLcov
  - id: CLM-023
    requirement: REQ-004
    text: END-TO-END — an executed gate over the fixture with the runner STUBBED (bun coverage output fed from the pre-captured lcov) measures the .ts source file's line AND branch coverage via the real convert, proving the consumer reads the bun records
    tests:
      - TestBunFixture_GateMeasuresTsCoverageFromPrecapturedLcovRunnerStubbed
  - id: CLM-024
    requirement: REQ-004
    text: A seeded-defect fixture variant — a changed .ts source file with NO coverage record — REDs the gate (loud blocking, not vacuous-green), proving the consumer is not silently green on a non-Go file
    tests:
      - TestBunFixture_SeededUncoveredTsSourceRedsGateNotVacuousGreen
  - id: CLM-025
    requirement: REQ-004
    text: The fixture test invokes NO real bun/oxlint/tsc/prettier process — only the POSIX-shell convert runs over the pre-captured lcov DATA, keeping the Go CI free of a bun dependency
    tests:
      - TestBunFixture_NoRealBunOxlintTscPrettierInvokedInGoCI
  - id: CLM-026
    requirement: REQ-004
    text: The .ts source file is classified MEASURABLE source via the bun pack's declared globs in the LIVE gate path end-to-end (the SPEC-043 merged classifier consumes the bun pack's globs)
    tests:
      - TestBunFixture_TsClassifiedMeasurableSourceViaDeclaredGlobsEndToEnd
  # REQ-005 — external executed acceptance (guarded; bun stays out of Go CI)
  - id: CLM-027
    requirement: REQ-005
    text: A guarded acceptance test runs the real gate over the Bun fork wired packs-only (backstop.yml declaring backstop/bun-toolchain + pack add) and reds on a seeded defect, then greens when the defect is fixed (RED-then-green)
    tests:
      - TestAcceptance_ForkBunGateRedThenGreenOnSeededDefect
  - id: CLM-028
    requirement: REQ-005
    text: The fork gates PACKS-ONLY via the declared backstop/bun-toolchain pack — no baked language path participates; the acceptance exercises the bun pack's engines as the only toolchain source
    tests:
      - TestAcceptance_ForkGatesPacksOnlyViaDeclaredBunToolchainPack
  - id: CLM-029
    requirement: REQ-005
    text: The acceptance test SKIPS (does not fail) when the bun toolchain is absent / the acceptance env var is unset, keeping backstop-core's Go CI bun-free while remaining an executed proof in the fork's environment
    tests:
      - TestAcceptance_SkippedWhenBunToolchainAbsentKeepsGoCIBunFree
  # REQ-006 — ratchet (3 sites, sequenced) → terminal block + zero-baseline flip
  - id: CLM-030
    requirement: REQ-006
    text: SITE 1 — after the coverage measurable-path is de-Go'd (SPEC-043), the committed baseline carries ZERO backstop/self neutral-spine findings for pkg/gate/step_coverage.go's measurable-path site (un-grandfathered)
    tests:
      - TestRatchet_CoverageMeasurablePathSiteUnGrandfatheredAfterDeGo
  - id: CLM-031
    requirement: REQ-006
    text: SITE 2 — after test-verify discovery is de-Go'd (SPEC-045), the committed baseline carries ZERO neutral-spine findings for pkg/gate/step_testverify.go's discovery site
    tests:
      - TestRatchet_TestVerifyDiscoverySiteUnGrandfatheredAfterDeGo
  - id: CLM-032
    requirement: REQ-006
    text: SITE 3 — after the go-package/`./...` matchers are de-Go'd (SPEC-043), the committed baseline carries ZERO neutral-spine findings for cmd/backstop/gate.go's goFilePackageMatchesTarget and step_coverage.go's coverageSpecRelevantToFile sites
    tests:
      - TestRatchet_GoPackageMatchersSiteUnGrandfatheredAfterDeGo
  - id: CLM-033
    requirement: REQ-006
    text: TERMINAL FLIP — with all three sites clean, the dogfood backstop.yml sets backstop/self's neutral-spine enforcement to level block with a ZERO baseline (baseline grandfathering disabled for neutral-spine findings)
    tests:
      - TestRatchet_SelfPackEnforcementFlipsToBlockZeroBaselineWhenAllSitesClean
  - id: CLM-034
    requirement: REQ-006
    text: THE WALL — after the flip, a deliberately reintroduced baked language literal (e.g. a `.go`/`_test.go` extension or a Go-package match) on a neutral-spine site REDs the gate outright as net-new against the zero baseline
    tests:
      - TestRatchet_ReintroducedBakedLanguageLiteralRedsOutright
  - id: CLM-035
    requirement: REQ-006
    text: ORDERING GUARD — the terminal flip is PROHIBITED while any of the three Pillar-A sites still flags; with a still-flagging site, the flip is not applied (the flip depends on SPEC-043/045/046 landing first)
    tests:
      - TestRatchet_FlipSequencedAfterPillarASitesClean
  # REQ-007 — per-pack/per-rule-source enforcement key (the pack_engines shared-dimension matrix)
  - id: CLM-036
    requirement: REQ-007
    text: A pack-scoped enforcement policy entry (e.g. `pack_engines` scoped to source `backstop/self` with level block + baseline false) parses onto the extended Enforcement.Policy / DimensionPolicy structure and is backward compatible — an unscoped dimension-only entry keeps its current behavior
    tests:
      - TestPolicy_PackScopedEntryParsesAndUnscopedEntryUnchanged
  - id: CLM-037
    requirement: REQ-007
    text: PACK backstop/self — with the per-pack key flipping backstop/self to block + ZERO baseline, a FRESH neutral-spine (baked-language) finding sourced from backstop/self on the pack_engines dimension BLOCKS the gate (fails)
    tests:
      - TestPolicy_SelfPackFlipBlocksFreshNeutralSpineFinding
  - id: CLM-038
    requirement: REQ-007
    text: PACK go-standards — with backstop/self flipped to block + zero baseline on the SAME shared pack_engines dimension, a baselined go-standards style finding does NOT block (stays grandfathered, unaffected by the backstop/self-scoped flip)
    tests:
      - TestPolicy_SelfPackFlipLeavesGoStandardsBaselinedFindingGrandfathered
  - id: CLM-039
    requirement: REQ-007
    text: PACK go-toolchain — with backstop/self flipped to block + zero baseline on the SAME shared pack_engines dimension, a baselined go-toolchain style finding does NOT block (stays grandfathered, unaffected by the backstop/self-scoped flip)
    tests:
      - TestPolicy_SelfPackFlipLeavesGoToolchainBaselinedFindingGrandfathered
  - id: CLM-040
    requirement: REQ-007
    text: DENYLIST — the flip is delivered via the per-pack/source key (filtering by Violation.SourcePack), NOT by a pack_engines-wide zero-baseline; a whole-dimension pack_engines entry at level block with baseline false (which would block go-standards/go-toolchain baselined debt) is prohibited and proven absent
    tests:
      - TestPolicy_FlipUsesPerPackKeyNotWholeDimensionZeroBaseline

contracts:
  - file: cmd/backstop/gate.go
    provides:
      - name: bunAcceptanceEnabled
        kind: function
        signature: "func bunAcceptanceEnabled() bool"
        notes: "NEW (REQ-005/CLM-029): reports whether the external Bun-fork acceptance may run — true only when the bun toolchain is on PATH and the acceptance env var is set. The guarded acceptance test calls t.Skip() when this is false, so backstop-core's Go CI never invokes the real bun/oxlint/tsc/prettier toolchain. This is the only new production symbol; the live dispatch of the bun pack is the UNIFORM toolchain-pack dispatch SPEC-046 already provides (no per-pack branch)."
    consumes:
      - source: pkg/gate
        name: SourceClassifier
        kind: type
      - source: pkg/pack
        name: Manifest
        kind: type
  - file: cmd/backstop/testdata/bun-toolchain/.backstop/packs/backstop/bun-toolchain/pack.yml
    provides: []
    consumes:
      - source: pkg/pack
        name: Classification
        kind: type
  - file: cmd/backstop/testdata/bun-toolchain/.backstop/packs/backstop/bun-toolchain/scripts/coverage-to-records.sh
    provides: []
    consumes:
      - source: pkg/check
        name: CoverageRecord
        kind: type
      - source: pkg/check
        name: ParsePackCoverage
        kind: function
  - file: pkg/gate/policy.go
    provides:
      - name: ApplyPolicy
        kind: function
        signature: "func ApplyPolicy(steps []StepResult, baseline *BaselineArtifact, policy map[string]DimensionPolicy, scope *GateScope) []StepResult"
        notes: "EXTENDED (REQ-007): external signature UNCHANGED; now applies per-PACK/per-rule-SOURCE-scoped policy WITHIN a dimension, filtering findings by gate.Violation.SourcePack so a scoped entry's level+baseline apply only to that pack's findings and every other pack on the same dimension is unaffected (keeps its default/grandfathered policy). An unscoped (dimension-only) entry keeps its current behavior (backward compatible)."
      - name: DimensionPolicy
        kind: type
        signature: "type DimensionPolicy struct { Level string; Baseline bool; /* + per-PACK/per-rule-SOURCE scoping keyed off gate.Violation.SourcePack (exact shape: planner's call) */ }"
        notes: "EXTENDED (REQ-007): gains an optional per-PACK/per-rule-SOURCE scoping (keyed off gate.Violation.SourcePack) so `backstop/self` can be flipped to block + zero baseline independently of go-standards/go-toolchain on the shared pack_engines dimension. Exact shape (nested per-source overrides vs. composite key) is the planner's call; the constraint is that a scoped entry must not change enforcement for other packs on the same dimension."
    consumes:
      - source: pkg/gate
        name: Violation
        kind: type
  - file: pkg/config/config.go
    provides:
      - name: DimensionPolicy
        kind: type
        signature: "type DimensionPolicy struct { Level string `yaml:\"level,omitempty\"`; Baseline bool `yaml:\"baseline,omitempty\"`; /* + per-PACK/per-rule-SOURCE scope */ }"
        notes: "EXTENDED (REQ-007): the backstop.yml mirror of the policy row gains the per-PACK/per-rule-SOURCE scoping so a `pack_engines` entry can target `backstop/self`'s findings independently. Backward compatible: an unscoped (dimension-only) entry parses and behaves as today."
    consumes: []
---

# SPEC-047: Bun Toolchain Pack And Two-Surface Proof

## Overview

This spec is **Seed 5 of 5 of BUNDLE-012** (language-neutral gate consumer + TypeScript/Bun
toolchain) — the **integration + proof** spec. It is the closing deliverable: where Seeds
1–4 (SPEC-043/044/045/046) de-Go the gate **consumer** and retire the `language:` field,
this seed **proves the whole thing works on a real foreign-language stack** and **slams the
door** on baked-language regressions.

It depends on ALL FOUR sibling seeds and consumes their contracts as **fixed interfaces**
(it was authored in parallel with SPEC-045/046, so it references them as already-agreed,
not as things it redefines):

| Sibling | Contract consumed by this spec (NOT redefined) |
| --- | --- |
| **SPEC-043** | The `classification: {source, test}` glob block on `pack.Manifest.Classification`, and the `gate.SourceClassifier` measurable-source classifier. The bun pack **declares** the block. |
| **SPEC-044** | The `(path, metric)` coverage record model + the **bun producer record-shape contract** (SPEC-044 REQ-006): two `check.CoverageRecord`s per file, `line` + `branch`, raw counts, through the EXISTING `check.CoverageRecord` + `check.ParsePackCoverage` — NO new type/field. The bun convert **emits** this shape. |
| **SPEC-045** | Test discovery via the pack-declared **test** globs (the bun pack's `**/*.test.ts`,`**/*.spec.ts`). |
| **SPEC-046** | No `language:` field; toolchain packs are **ordinary declared packs** dispatched uniformly. The bun pack is declared in `backstop.yml packs:` like any other. |

This spec does **four** coupled things:

1. **Authors `backstop/bun-toolchain`** (bundle REQ-006) — an **ordinary** pack in its **own
   repo** (packs are always external; the in-repo `.backstop/packs` tree is gitignored
   DATA — see [[packs_always_external]]), **hand-authored** from the proven
   `backstop/go-toolchain` `pack.yml` template (OQ-5: the pack-authoring CLI is the broken
   thing, so hand-author; the reboot is Track-B / ISSUE-032, **out of scope**). Engines:
   `oxlint` (lint) · `prettier --check` (format-as-lint, DD-3) · `tsc --noEmit` (build) ·
   `bun test` (test) · `bun test --coverage --coverage-reporter=lcov` (coverage).

2. **Declares the SPEC-043 classification block** and **emits the SPEC-044 line+branch
   record shape** via a pack-relative `coverage-to-records.sh` lcov convert.

3. **Proves on two surfaces** (bundle REQ-009): an **in-repo static testdata fixture**
   (pre-captured lcov, runner stubbed, zero `bun` dependency in the Go CI) **and** an
   **external executed gate** on the real Bun fork (`backstop-runtime`), RED-then-green on a
   seeded defect — the **required** acceptance, kept out of the Go CI by a skip guard.

4. **Performs the ratchet → block flip** (bundle REQ-008): un-grandfather each Pillar-A site
   as it is de-Go'd, then flip `backstop/self` to `block` + zero baseline once all three are
   clean.

**In scope:** the bun pack DATA (engines + classification + lcov convert); the in-repo static
fixture + its stubbed-runner end-to-end gate test; the guarded external acceptance test; the
trust-allowlist entries for `oxlint`/`bun`/`tsc`/`prettier`; and the dogfood baseline +
`backstop/self` enforcement-config flip.

**Out of scope (fenced to siblings or follow-ups):** the consumer-side de-Go logic itself
(SPEC-043/045), the `(path, metric)` index + per-metric thresholding (SPEC-044), the bridge
deletion + `language:` retirement (SPEC-046); a true **two-toolchains-in-one-repo executed**
proof; productionizing the fork's CI; and the pack-CLI authoring reboot (Track-B / ISSUE-032).

## Requirements

Requirements are enumerated in the `requirements:` frontmatter (REQ-001 … REQ-007), each
tracing to a BUNDLE-012 requirement via `supports`. Summary:

| Spec REQ | Bundle REQ | Commits to |
| --- | --- | --- |
| REQ-001 | REQ-006 | The bun pack declares EXACTLY five engines — `oxlint`→lint, `prettier --check`→**lint** (format-as-lint, DD-3, **no new dimension**), `tsc --noEmit`→build, `bun test`→test, `bun test --coverage … lcov`→coverage — each run through the declared-engine trust substrate, never baked. |
| REQ-002 | REQ-006 | The bun pack declares the SPEC-043 `classification` block (source `**/*.ts`,`**/*.tsx`; test `**/*.test.ts`,`**/*.spec.ts`); the measurable-source matrix holds over the six file types. |
| REQ-003 | REQ-006 | The lcov convert emits the SPEC-044 bun record shape: two records per file (`line` LF/LH, `branch` BRF/BRH), raw counts, through the EXISTING `check.CoverageRecord` + `check.ParsePackCoverage`, no new type/field; `BRF:0`→`total:0` branch N/A. |
| REQ-004 | REQ-009 | An in-repo static fixture (pre-captured lcov, runner stubbed) measures the `.ts` source's line+branch coverage end-to-end with ZERO `bun` dependency; a seeded uncovered `.ts` REDs (not vacuous-green). |
| REQ-005 | REQ-009 | A REQUIRED external executed gate on the Bun fork goes RED-then-green on a seeded defect; expressed as a guarded test that SKIPS when `bun` is absent, keeping the real toolchain out of Go CI. |
| REQ-006 | REQ-008 | Ratchet each Pillar-A site out of the baseline as de-Go'd (coverage measurable-path → test-verify → go-package matchers), then flip `backstop/self` to `block` + ZERO baseline once all three are clean (via the REQ-007 per-pack key, NOT the whole `pack_engines` dimension); a reintroduced baked literal then REDs outright. |
| REQ-007 | REQ-008 | EXTEND the enforcement policy with finer-than-dimension (per-PACK / per-rule-SOURCE) granularity (keyed off `gate.Violation.SourcePack`) so `backstop/self` flips to `block` + ZERO baseline INDEPENDENTLY: a fresh neutral-spine finding blocks, while `go-standards`/`go-toolchain` baselined style debt on the SHARED `pack_engines` dimension stays grandfathered. Whole-dimension zero-baseline is PROHIBITED. |

### The engine declaration matrix (REQ-001, DD-3)

The bun pack declares exactly five engines. The **denylist cell** is load-bearing: prettier is
**format**, but format ≈ lint for gate purposes, so it rides the **existing lint dimension**
and introduces **no new "format" gate dimension**.

| Engine command | `gate_type` (dimension) | Category | Claim |
| --- | --- | --- | --- |
| `oxlint --format=json` | `lint` | mechanism (lint) | CLM-001 |
| `prettier --check .` | `lint` (format-as-lint) | mechanism (lint) | CLM-002 |
| `tsc --noEmit` | `build` | mechanism (build) | CLM-003 |
| `bun test` | `test` | mechanism (test) | CLM-004 |
| `bun test --coverage --coverage-reporter=lcov` | `coverage` | mechanism (coverage) | CLM-005 |
| *(no new "format" dimension is created — prettier rides lint)* | — | — | CLM-006 (denylist) |
| *(all four tools `oxlint`/`bun`/`tsc`/`prettier` are allowlisted, dispatched as DATA)* | — | — | CLM-007 |

### The measurable-source matrix (REQ-002, consumes SPEC-043)

Under the bun pack's globs **alone** — source `S` = {`**/*.ts`,`**/*.tsx`}, test `T` =
{`**/*.test.ts`,`**/*.spec.ts`} — `gate.SourceClassifier.IsMeasurableSource(path)` must hold:

| Example path | matches source? | matches test? | measurable? | Claim |
| --- | --- | --- | --- | --- |
| `src/app.ts` | yes | no | **YES** | CLM-009 |
| `src/Button.tsx` | yes | no | **YES** | CLM-010 |
| `src/app.test.ts` | yes | yes | **NO** (test wins) | CLM-011 |
| `src/app.spec.ts` | no¹ | yes | **NO** | CLM-012 |
| `pkg/x/foo.go` | no | no | **NO** (no baked Go literal across packs) | CLM-013 |
| `README.md` | no | no | **NO** (unclassified) | CLM-014 |

¹ `**/*.spec.ts` matches `**/*.ts` too (a `.spec.ts` IS a `.ts`); it is still NOT measurable
because the test glob wins on overlap — the verdict is the same either way (NOT measurable).

### The lcov record-shape contract (REQ-003, consumes SPEC-044 REQ-006)

For each measured source file the convert reads the lcov per-file block and emits **two**
canonical records sharing one `Path`:

| Metric | `covered` (raw) | `total` (raw) | lcov source |
| --- | --- | --- | --- |
| `line` | `LH` (lines hit) | `LF` (lines found) | `DA:`/`LF:`/`LH:` |
| `branch` | `BRH` (branches hit) | `BRF` (branches found) | `BRDA:`/`BRF:`/`BRH:` |

Raw counts only — **never** a pre-computed percent, **never** an aggregate of the two, **never**
a single collapsed record. A file with no branchable code (`BRF: 0`) emits a `branch` record
with `total: 0` (the N/A cell, faithfully preserved — SPEC-044 thresholds it as N/A) while its
`line` record is still measured. Both records flow through the **existing**
`check.ParsePackCoverage` (which `DisallowUnknownFields` — so the bun JSON must use exactly the
canonical `{path, covered, total, measured, excluded, metric}` keys, no new field) and index
under SPEC-044's `(path, metric)` key with no collision.

### The shared-dimension enforcement matrix (REQ-007)

`backstop/self`'s `no-language-literal-on-neutral-spine` findings ride the **same**
`pack_engines` gate dimension as `go-standards`/`go-toolchain`'s legitimate baselined style
debt. The policy table (`pkg/config` `Enforcement.Policy` → `pkg/gate.ApplyPolicy`) keys
**strictly per-dimension**, so a `pack_engines: {level: block, baseline: false}` entry would
wrongly block the inherited style debt. REQ-007 extends the policy with a per-PACK / per-rule-SOURCE
key (filtering on the existing `gate.Violation.SourcePack`) so the flip targets `backstop/self`
**alone**. With `backstop/self` flipped to `block` + ZERO baseline via that key:

| Pack on `pack_engines` | Finding | Grandfathered? | Gate verdict | Claim |
| --- | --- | --- | --- | --- |
| `backstop/self` | FRESH neutral-spine (baked-language) | no (zero baseline) | **BLOCK** | CLM-037 |
| `go-standards` | baselined style debt | **yes** (unaffected) | **pass** | CLM-038 |
| `go-toolchain` | baselined style debt | **yes** (unaffected) | **pass** | CLM-039 |

The denylist cell (CLM-040): the flip MUST be delivered through the scoped key, **never** by a
whole-dimension `pack_engines: {block, baseline:false}` entry (which would collapse all three
rows into BLOCK and break go-standards/go-toolchain's grandfathered debt). CLM-036 covers config
parse + backward compatibility (an unscoped dimension-only entry behaves as today).

## Implementation

Target package: **`cmd/backstop`** (the in-repo fixture + its end-to-end gate test + the
guarded acceptance test + the dogfood baseline/enforcement config), with the bun pack itself as
**DATA** (its own repo + the gitignored in-repo copy + the testdata fixture copy) and the lcov
convert as a **POSIX-shell** script exercised over pre-captured data. This spec consumes
SPEC-043/044/045/046 surfaces and **edits none of their core logic**. Processing steps the
planner must map tasks to:

1. **Author the `backstop/bun-toolchain` pack DATA (REQ-001, REQ-002).** In its own repo
   (hand-authored from the `go-toolchain` `pack.yml` template), declare the five engines with
   their `gate_type` routing (lint/lint/build/test/coverage), the `prettier` engine as a
   lint-category SARIF findings engine (a `format-to-sarif.sh` convert, one finding per
   unformatted file — **no** new gate dimension), the coverage engine with
   `stdout_artifact: coverage/lcov.info` + `convert: scripts/coverage-to-records.sh`
   + `gate_type: coverage`, and the SPEC-043 `classification` block (source `**/*.ts`,`**/*.tsx`;
   test `**/*.test.ts`,`**/*.spec.ts`). The pack carries mechanism only (no coding-standards
   rules), mirroring `go-toolchain`.

2. **Author the lcov coverage convert (REQ-003).** `scripts/coverage-to-records.sh` — POSIX awk
   (macOS system awk, like `go-toolchain`'s), reading lcov on stdin, accumulating per-`SF:` file
   `LF`/`LH` (line) and `BRF`/`BRH` (branch), and emitting a JSON array of canonical records:
   two per file (`metric:"line"` and `metric:"branch"`, `measured:true`, `excluded:false`), raw
   counts, no percent, with `BRF:0` ⇒ a `branch` record carrying `total:0`. A converter banner
   on stderr exercises clean-stdout capture (as `go-toolchain` does).

3. **Allowlist the bun toolchain tools (REQ-001).** Add `oxlint`, `bun`, `tsc`, `prettier` to
   the declared-engine trust allowlist (and the lock where needed) so the dispatch trusts the
   pack-declared commands. The in-repo fixture stubs execution (no real tool needed in CI); the
   allowlist matters for the external acceptance.

4. **Build the in-repo static testdata fixture (REQ-004).** Under
   `cmd/backstop/testdata/bun-toolchain/`: a `backstop.yml` declaring
   `backstop/bun-toolchain: local` in `packs:` with **no** `language:` field (SPEC-046); the bun
   `pack.yml` (copied from the pack repo); `src/*.ts` source + `src/*.test.ts` test files; and a
   PRE-CAPTURED `coverage/lcov.info`. Add an end-to-end gate test that stubs the runner (feeding
   the pre-captured lcov as the `bun test --coverage` engine output, like the existing
   `fixtureRunner{byCmd:…}` pattern), runs the **real** convert over it, and asserts the gate
   measures the `.ts` source's `line` + `branch` coverage via the merged classifier end-to-end —
   invoking **no** real `bun`/`oxlint`/`tsc`/`prettier`.

5. **Add the seeded-defect RED variant (REQ-004).** A fixture (or scope) variant where a changed
   `.ts` source file has **no** coverage record (or a below-threshold record) — assert the gate
   REDs loudly (the SPEC-043 anti-vacuous-green guard fires for the non-Go file), proving the
   consumer is not silently green.

6. **Add the guarded external acceptance test (REQ-005).** A `TestAcceptance_…` that calls
   `t.Skip()` via `bunAcceptanceEnabled()` (bun on PATH **and** acceptance env var set), and
   otherwise runs the real gate over the Bun fork wired packs-only (`backstop.yml` declaring the
   bun pack + `pack add`), asserting RED on a seeded lint/format/type/test/coverage defect and
   green when fixed. Document the minimal fork wiring (the single-acceptance `backstop.yml` +
   `pack add`); productionizing the fork CI is out of scope.

7. **Extend the enforcement policy with per-PACK / per-rule-SOURCE granularity (REQ-007).** In
   `pkg/config/config.go` (`Enforcement.Policy` / `DimensionPolicy`) and `pkg/gate/policy.go`
   (`ApplyPolicy`), add the ability to scope a policy entry to a pack/source, keyed off the
   existing `gate.Violation.SourcePack`. When an entry is source-scoped, its `level` + `baseline`
   apply ONLY to findings from that pack within the dimension; all other packs on the dimension
   keep their default (or separately-configured) policy — so `backstop/self` can be flipped to
   `block` + zero baseline on `pack_engines` while `go-standards`/`go-toolchain` keep their
   grandfathered style debt. Keep `ApplyPolicy`'s external signature unchanged and an unscoped
   (dimension-only) entry behaving exactly as today (backward compatible). This is the mechanism
   REQ-006's flip is expressed through.

8. **Ratchet then flip (REQ-006), sequenced AFTER SPEC-043/045/046 land.** As each Pillar-A site
   is de-Go'd, regenerate the committed `.backstop/baseline.json` so that site's
   `backstop/self` neutral-spine findings drop out (ratchet), in order: (1) coverage
   measurable-path → (2) test-verify discovery → (3) go-package/`./...` matchers. Once all three
   are clean, set the dogfood `backstop.yml` enforcement for `backstop/self`'s neutral-spine
   findings to `level: block` with a ZERO baseline (`baseline: false` — no grandfathering) VIA
   the REQ-007 per-pack/source key (NOT a whole-`pack_engines` zero-baseline), so a future
   baked-language regression REDs outright while go-standards/go-toolchain debt stays
   grandfathered. The flip is the **closing** step, gated on the other three sites being clean.

## Verification

- **Level:** `integration` (threshold 80). The load-bearing proofs are cross-package executed
  gates over a testdata fixture (`cmd/backstop` → `pkg/gate` → `pkg/check`), plus the convert
  exercised over pre-captured data — not isolated units. This honors the integration-gap lesson
  ([[feedback_integration_gap]] / [[project_pack_provisioning_integration_gap]]): the bun pack's
  consumer path is proven through the LIVE gate over a real (stubbed-runner) installed-pack
  fixture, not only a hand-constructed classifier.
- **Command:** `go test ./cmd/backstop/ ./pkg/gate/ ./pkg/check/ ./pkg/config/ -race -coverprofile=cover.out` (`./pkg/config/` covers the REQ-007 `config.go` edit / the CLM-036 config-parse test).
- **Mandated tests:** every test named in the `claims[]` `tests:` fields. The load-bearing ones
  are `TestBunFixture_GateMeasuresTsCoverageFromPrecapturedLcovRunnerStubbed` (CLM-023 — the
  end-to-end consumer proof with zero `bun` dependency) and
  `TestBunFixture_SeededUncoveredTsSourceRedsGateNotVacuousGreen` (CLM-024 — the non-Go
  anti-vacuous-green proof). The external acceptance (`TestAcceptance_…`, CLM-027/028) is the
  REQUIRED end-to-end proof but is **skipped** in backstop-core's Go CI (CLM-029) to keep the
  real `bun` toolchain out — it runs in the fork's environment.

## Sharp Edges

- **Format must NOT become a new gate dimension (DD-3).** The tempting wrong move is to add a
  `format` `gate_type` for prettier. That bloats the gate's dimension set and breaks DD-3.
  `prettier --check` is a **lint-category SARIF findings engine** — one finding per unformatted
  file — routed to the EXISTING `lint` dimension. CLM-006 is the denylist guard against this
  regression.
- **`ParsePackCoverage` uses `DisallowUnknownFields` — the bun JSON keys must be exact.** The
  convert MUST emit exactly `{"path","covered","total","measured","excluded","metric"}`. A typo
  or an extra field (e.g. a stray `"branches"`) is rejected fail-loud — which is correct, but
  means the convert and the canonical type are tightly coupled. No new field is permitted (the
  whole point of consuming SPEC-044's no-new-type contract).
- **Empty `metric` is a fail-loud upstream — the branch N/A cell still needs a metric.** A
  `BRF:0` file emits a `branch` record with `total:0` but **must still carry `metric:"branch"`**
  (and `measured:true`): an empty `metric` on a measured record is rejected by
  `ParsePackCoverage` (SPEC-042 REQ-005). `total:0` is N/A, not unmeasured — do not drop the
  metric label.
- **`.spec.ts` matches the source glob too — rely on test-wins-on-overlap, not glob exclusivity.**
  `**/*.spec.ts` is a subset of `**/*.ts`, so a `.spec.ts` file matches BOTH the source and test
  globs. It is non-measurable ONLY because SPEC-043's classifier resolves overlap in favor of
  test. If the bun pack author assumed the source glob alone excludes spec files, the result is
  still correct here — but the dependency on test-wins-on-overlap must be explicit so a future
  glob edit doesn't silently make spec files measurable.
- **The stubbed runner must feed the lcov to the COVERAGE channel, not the SARIF channel.** Like
  `go-toolchain`'s `go-coverage`, the bun coverage engine routes to the coverage-records channel
  (`gate_type: coverage`), DISTINCT from the SARIF findings dispatch. A fixture test that feeds
  the lcov into `dispatchPackEngines` (SARIF) instead of the coverage channel proves nothing.
  The convert output is coverage-records JSON, not SARIF.
- **The go-toolchain pack's OWN classification block is SPEC-043's deliverable, NOT this spec's.**
  SPEC-043 adds `classification` to the `go-toolchain` pack.yml (its impl step + sharp edge) so
  the dogfood gate keeps classifying `.go` once the baked literal is gone. This spec authors only
  the BUN pack's block. Do not duplicate or move the go block here — flagged for the
  cross-consistency pass.
- **REQ-006's flip is sequenced and BLOCKED on three sibling specs landing.** The ratchet/flip
  cannot run until SPEC-043 (coverage measurable-path + go-package matchers), SPEC-045
  (test-verify), and SPEC-046 (bridge/`language:` retirement) are implemented and their sites are
  clean. Performing the terminal `block` + zero-baseline flip early would RED the dogfood gate on
  the not-yet-de-Go'd sites. CLM-035 is the ordering guard.
- **Flipping only `backstop/self` to zero-baseline — not the whole `pack_engines` dimension.**
  The `pack_engines` gate dimension is shared with `go-standards`/`go-toolchain`, which carry
  legitimate baselined style debt. The flip must zero-baseline the `backstop/self` neutral-spine
  findings SPECIFICALLY, not un-grandfather every pack riding `pack_engines`. The enforcement
  table today keys STRICTLY by dimension (`pkg/config` `Enforcement.Policy` →
  `pkg/gate.ApplyPolicy`), so the neutral-spine findings need their own addressable, finer-than-dimension
  policy key. This is NO LONGER a punted detail — it is promoted to **REQ-007** (a required
  deliverable of this spec: a per-PACK / per-rule-SOURCE key off `gate.Violation.SourcePack`),
  with the shared-dimension matrix (CLM-037/038/039) and the denylist guard (CLM-040) proving the
  flip blocks a fresh neutral-spine finding while leaving go-standards/go-toolchain debt
  grandfathered.
- **Packs are always external — the in-repo copy is gitignored DATA.** The authoritative bun pack
  lives in its own repo; the `.backstop/packs/backstop/bun-toolchain/` copy is gitignored, so the
  durable artifact is the testdata-fixture copy under `cmd/backstop/testdata/` (tracked) plus the
  external repo. Editing the gitignored copy is non-durable ([[packs_always_external]],
  [[project_pack_distribution]]).

## Review Questions

These probe risks not fully pinned by the claims; the impl-reviewer should check each against
the diff.

- Does the bun pack route `prettier --check` to the **lint** dimension as a lint-category
  findings engine, and does loading the pack add **no** new "format" gate dimension? (REQ-001 /
  CLM-002/CLM-006 — the DD-3 denylist.)
- Are all five engine commands dispatched from pack DATA through the trust allowlist
  (`oxlint`/`bun`/`tsc`/`prettier`), with **no** baked command string in the binary? (REQ-001 /
  CLM-007 — the thin-executor first principle.)
- Does the convert emit exactly TWO records per file (`line` LF/LH, `branch` BRF/BRH), raw counts,
  through the EXISTING `check.CoverageRecord` + `check.ParsePackCoverage` with NO new type/field,
  and does a `BRF:0` file emit a `branch` record with `total:0` carrying `metric:"branch"`?
  (REQ-003 / CLM-015..CLM-019.)
- Does the in-repo fixture test invoke NO real `bun`/`oxlint`/`tsc`/`prettier` — only the POSIX
  convert over pre-captured lcov — so backstop-core's Go CI gains no `bun` dependency? (REQ-004 /
  CLM-025.)
- Does a changed `.ts` source file with no record RED the gate via the SPEC-043 anti-vacuous-green
  guard end-to-end (the live merged classifier consuming the bun pack's globs), not just in a unit
  test? (REQ-004 / CLM-024/CLM-026.)
- Is the external acceptance a guarded test that SKIPS (not fails) when `bun` is absent, and does
  it gate the fork PACKS-ONLY (no baked language path), RED-then-green on a seeded defect?
  (REQ-005 / CLM-027/CLM-028/CLM-029.)
- Is the terminal `block` + zero-baseline flip applied ONLY after all three Pillar-A sites are
  clean (sequenced, not early), and does it zero-baseline `backstop/self`'s neutral-spine findings
  SPECIFICALLY (not the whole shared `pack_engines` dimension)? (REQ-006 / CLM-033/CLM-035, sharp
  edges.)
- Does the enforcement-policy extension key by PACK/rule-SOURCE (off `gate.Violation.SourcePack`),
  so flipping `backstop/self` to `block` + zero baseline BLOCKS a fresh neutral-spine finding yet
  leaves `go-standards` AND `go-toolchain` baselined style debt grandfathered on the SAME
  `pack_engines` dimension — and is there NO whole-dimension `pack_engines: {block, baseline:false}`
  entry anywhere? Is an unscoped (dimension-only) entry still backward compatible? (REQ-007 /
  CLM-036..CLM-040.)
- After the flip, does a reintroduced baked `.go`/`_test.go`/Go-package literal on a neutral-spine
  site RED the gate outright as net-new against the zero baseline? (REQ-006 / CLM-034 — the wall.)

## Cross-Consistency Seam Items

Flagged for the BUNDLE-012 final cross-consistency pass (the cutover-coupling lesson — shared
files, sibling seams):

1. **SPEC-044 cross-reference fix.** SPEC-044 names the bun coverage producer "SPEC-045" (in its
   REQ-006, the SPEC-043/044 seam table, and its References). The bun producer is **THIS spec,
   SPEC-047** — SPEC-045 is the de-Go'd test-verification discovery seed. SPEC-044's references to
   "SPEC-045 (bun coverage producer)" must be corrected to SPEC-047. (This spec is the consumer of
   SPEC-044 REQ-006; the citation in 044 is the only drift.)
2. **go-toolchain pack.yml DATA must gain BOTH blocks in lockstep with the literal deletions
   (SHOULD-FIX, owned by SPEC-043/045's impl — referenced here, NOT authored here).** When the
   baked Go literals are deleted from the consumer, the `backstop/go-toolchain` pack.yml must
   simultaneously gain (a) SPEC-043's `classification` block (source/test globs so `.go` stays
   classified once the baked literal is gone) AND (b) SPEC-045's `test_name_patterns` (so Go test
   discovery survives the de-Go'd test-verify path). Both MUST land **in lockstep** with the
   deletions — a deletion without the corresponding pack DATA would silently stop classifying /
   discovering `.go`, regressing the dogfood gate — and both MUST be written durably into the
   **external** `backstop/go-toolchain` pack repo (the in-repo `.backstop/packs` copy is gitignored;
   editing it is non-durable — see [[packs_always_external]]). This is **SPEC-043's** deliverable
   (the `classification` block) and **SPEC-045's** deliverable (`test_name_patterns`), NOT this
   spec's; this spec authors only the BUN pack's blocks. No duplication across the specs — flagged
   so the plan orders the go-toolchain DATA edits with their respective deletions, and so REQ-006's
   ratchet does not flip before the go-toolchain pack durably carries both.
3. **REQ-006 sequencing dependency.** The ratchet→block flip depends on SPEC-043 (coverage
   measurable-path + go-package matchers), SPEC-045 (test-verify discovery), and SPEC-046
   (bridge/`language:` retirement) being implemented and clean first. The plan must order
   SPEC-047's REQ-006 task LAST, after all three.
4. **Shared files, consumed-not-edited.** `cmd/backstop/gate.go` and `pkg/gate/step_coverage.go`
   are co-owned with SPEC-043/044/045/046. This spec adds only `bunAcceptanceEnabled()` to
   `gate.go` and edits no sibling core logic; it consumes `gate.SourceClassifier`,
   `pack.Manifest.Classification`, the `(path, metric)` index, and the uniform toolchain-pack
   dispatch as fixed. The merged classifier (SPEC-043 REQ-005) is the integration point — the bun
   pack's globs must reach the live step via the union SPEC-043 already builds.

## References

- BUNDLE-012 (`language-neutral-consumer-ts-toolchain`) — **Seed 5**; this spec implements bundle
  REQ-006 (the bun pack), REQ-009 (two-surface proof), and REQ-008 (ratchet→block flip).
- SPEC-043 (`pack-declared-globs-coverage-consumer`) — the `classification` block + the
  `gate.SourceClassifier` this pack declares against and feeds. Consumed as a fixed contract.
- SPEC-044 (`multi-metric-coverage-records`) — the `(path, metric)` record model + the bun
  producer record-shape contract (REQ-006) this convert emits. Consumed as a fixed contract; see
  the cross-reference fix (seam item 1).
- SPEC-045 (`de-go-test-verification-discovery`) — pack-declared test-glob discovery; the bun
  pack's `**/*.test.ts`,`**/*.spec.ts` test globs feed it.
- SPEC-046 (`retire-language-toolchain-bridge`) — no `language:` field; toolchain packs are
  ordinary declared packs dispatched uniformly. The bun pack is declared in `backstop.yml packs:`.
- SPEC-042 (`coverage-production-engine`) — the canonical `check.CoverageRecord` +
  `check.ParsePackCoverage` (raw counts, `metric` label, empty-metric fail-loud) the convert
  targets, unchanged here. `go-toolchain`'s `scripts/coverage-to-records.sh` is the convert
  template.
- [[feedback_loud_not_blocking]] — the seeded-defect RED (REQ-004) and the ratchet→block wall
  (REQ-006): a non-Go file's missing coverage and a baked-language regression must be loud.
- [[feedback_zero_baked_checks]] / [[feedback_dogfood_rules_as_packs]] — the thin-executor first
  principle this seed finishes proving on the consumer side: a toolchain is just another pack; the
  `backstop/self` flip converts the dogfood rule from grandfathered backlog to a real wall.
- [[packs_always_external]] / [[project_pack_distribution]] — the bun pack lives in its own repo;
  the in-repo `.backstop/packs` copy is gitignored; the durable in-CI artifact is the testdata
  fixture.
- [[feedback_integration_gap]] / [[project_pack_provisioning_integration_gap]] — why the proof is
  an executed gate over an installed-pack fixture (stubbed runner), plus the REQUIRED external
  executed gate, not only unit tests over a constructed classifier.
- [[project_toolchain_pack_convention]] — one `<stack>-toolchain` pack per runtime; the bun pack
  follows the go-toolchain shape (DD-2: keyed to the Bun runtime, not the TS language).
- [[project_baseline_design]] / [[project_baseline_ci_pull]] — the ratchet flow (regenerated
  immutable baseline) terminating in the `backstop/self` `block` + zero-baseline flip.
- Code (verified 2026-06-28, this branch): reference `.backstop/packs/backstop/go-toolchain/pack.yml`
  + `scripts/coverage-to-records.sh` (the template); `cmd/backstop/testdata/typescript-toolchain/`
  (the fixture shape to extend); `cmd/backstop/gate_bridge_agnostic_test.go` `fixtureRunner{byCmd}`
  (the stubbed-runner pattern); `pkg/check/coverage.go` `CoverageRecord` + `ParsePackCoverage`
  (canonical, unchanged); the dogfood `backstop.yml` `enforcement.policy` block +
  `.backstop/baseline.json` (the ratchet/flip surface); `.backstop/packs/backstop/self/pack.yml`
  (the neutral-spine rule family).

## Version History

- **1.2.0** (2026-07-01) — **REQ-005 external executed acceptance CLOSED (executed + PASSED).**
  The required external gate was RUN over the real installed `backstop/bun-toolchain` pack on a
  clean Bun project (bun 1.3.13) with the current backstop binary, producing the RED-then-green
  run-evidence REQ-005 demands: clean project → gate **GREEN**; seeded `tsc` type error (TS2322)
  → **RED** (build dimension, finding surfaced with file + rule); seeded `oxlint` `no-debugger`
  violation → **RED** (lint dimension); seeded `bun test` failure → **RED** (test dimension);
  defect removed → **GREEN**. This supersedes the 1.1.0 OUTSTANDING-ACCEPTANCE note.
  **Honest caveat — the acceptance INITIALLY failed as a silent vacuous green**, because executing
  it (not the stubbed-dispatch Go-CI tests) surfaced TWO real backstop-core dispatch defects: (1)
  `runFindingsEngine` bolted `projectRoot` onto project-wide engines with no `project_target`, so
  `tsc --noEmit` ignored `tsconfig.json`; and (2) `runFindingsEngine` ignored `stdout_artifact`, so
  `bun test`'s JUnit output was never read. Both were fixed under a **NEW spec, SPEC-048**
  (engine-dispatch self-targeting + `stdout_artifact` for findings) — SPEC-048 is the true enabler
  of REQ-005's green, so **REQ-005's closure DEPENDS ON SPEC-048**. The bun pack itself also needed
  SARIF converts + a `prettier --check` → `--list-different` stream fix (landed in the external pack
  repo). No requirement, claim, or contract text changed — executed-acceptance evidence + the
  SPEC-048 dependency only. (Reflects reality per align-predating-artifacts.)
- **1.1.0** (2026-06-30) — Status → `implemented`. The BUNDLE-012 Seed 4 code shipped
  and passed impl-review PASS; the `backstop/bun-toolchain` pack, the in-repo fixture
  end-to-end proof, and the ratchet→block flip are live. No requirement, claim, or
  contract text changed — lifecycle transition only.
  - **OUTSTANDING ACCEPTANCE — REQ-005.** REQ-005's *external executed acceptance* (the
    guarded manual `bun-gate` run over the opencode fork, producing real run-evidence) is
    the ONE open user-manual acceptance and is **NOT yet done** — it awaits the user's
    manual run. `status: implemented` reflects that the code shipped and passed
    impl-review; it does **not** assert REQ-005's executed run-evidence has been captured.
    Tracked separately by the user.
- **1.0.0** (2026-06-28) — Initial spec authored from BUNDLE-012 Seed 4.
