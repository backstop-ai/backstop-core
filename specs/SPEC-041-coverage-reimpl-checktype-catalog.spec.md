---
title: "SPEC-041: Coverage Re-implementation + CheckType-Consumer Catalog"
number: SPEC-041
created: "2026-06-24"
status: implemented
schema_version: spec/v1
spec_version: 1.2.0

implementation:
  summary: >
    BUNDLE-011 Spec Seed 3 (RDQ-6) — migrate the test+coverage stack as a UNIT so
    that no baked shared `go test` runner and no baked Go coverage analyzer survive
    the legacy-`pkg/check` cutover, and produce a machine-checkable CheckType-consumer
    catalog as the "don't drop a gate step on the floor" guard. This is a DOWNSTREAM
    spec of SPEC-040 (Seed 2): it CONSUMES the toolchain test runner that SPEC-040's
    `go-toolchain` cutover establishes and re-implements coverage OVER it
    language-agnostically. Concretely, FOUR mechanisms across three bundle requirements.
    (REQ-011 — coverage re-impl + shared-runner eradication) DELETE the baked Go coverage
    analyzer `pkg/gate/step_coverage.go` (the `go test ... -coverprofile` invocation, the
    `coverage: N% of statements` regex parser, the `go test ./...` whole-module dedup, and
    the go.mod module-path / package-label helpers — all Go-toolchain knowledge baked into
    the binary, KEPT baked by SPEC-038 REQ-009's descope) and re-implement coverage as a
    language-agnostic step that reads a PER-FILE, PATH-keyed coverage record FROM the
    SPEC-040 toolchain test runner's declared output (the gate consumes a normalized
    canonical `[]check.CoverageRecord` — the SINGLE shared type defined by SPEC-042
    (`Path`/`Covered`/`Total`/`Measured`/`Excluded`/`Metric`, RAW COUNTS), FILE
    granularity, NO "package" noun since "package" is a Go-native concept that would
    re-bake language knowledge — and computes `Covered/Total >= threshold` per FILE
    (metric-blind, `Total==0`⇒N/A), surfacing the pack-declared `Metric` label without
    interpreting it; exclusions are pack-declared, not baked). The baked `newSharedTestRunner`
    (cmd/backstop/shared_testrun.go) — which today couples code_check's test FAILs and
    coverage's per-file read through one hardcoded `go test ./... -coverprofile=/dev/null`
    exec — is ERADICATED; after this spec NO baked shared runner exists and coverage's input
    is the declared toolchain test pass, not an in-binary `go test`. (REQ-012 — declared
    exempt property + per-violation resolution) RE-EXPRESS the build-pass project-wide-scope
    exemption via a NEW, EXPLICIT engine-binding property `exempt_from_scope_filter`
    (boolean), DECOUPLED from `ScopeKind` (which stays arg-shaping-only). The real work is on
    the ENGINE PATH: build the `binding.exempt_from_scope_filter → gate.Violation.ProjectWide`
    bridge in cmd/backstop/pack_gate.go (it does NOT exist today — pack_gate.go:411 uses
    `ScopeKind` only for `./...`-arg-shaping and NEVER sets `ProjectWide`). The build-pass
    exemption becomes "the go-build engine declares `exempt_from_scope_filter: true`";
    golangci/go-test declare it false. CRITICALLY, the baked `cv.Pass == check.CheckTypeBuild`
    setter at gate.go:1173 (`checkViolationsToGate`) is NOT edited in place — it is orphaned
    once SPEC-040 deletes its only caller `realCodeChecker`, so this spec REPLACES its
    behavior on the engine path and catalogs it DELETED. The exempt value is resolved
    PER-VIOLATION (each violation carries its producing binding's value), never aggregated to
    a gate-type level; the only true conflict (same file+line+rule, differing values) resolves
    to the exempting (louder) value, since under-broad filtering is the guarded failure mode.
    (REQ-013 — CheckType-consumer catalog) PRODUCE a CheckType-consumer catalog scoped to
    GATE-SEMANTIC consumers (scope-filtering, engine dispatch, violation verdict), EXCLUDING
    cosmetic `.Pass.String()` display/serialization sites so a faithful guard does not red on
    every added log line. Cross-spec fact (verified on `main` + SPEC-039 + SPEC-040): no
    sibling spec deletes the `pkg/check` `CheckType` TYPE surface, so `passOrder`,
    `Violation.Pass`/`PassResult.Pass`, the executor/dispatch maps, `registry.go` Entries,
    `manifest.go` enum+routing, and `parsers.go` findings stamping all SURVIVE and are
    cataloged with their real post-cutover role (NOT mis-tagged DELETED); only the
    shared-runner feeds and the orphaned gate.go:1173 exemption are DELETED. The catalog is
    backed by a machine guard that FAILS on any gate-semantic keying site with no catalog
    entry AND on any stale catalog entry. The anti-vacuous-green invariant is load-bearing: a
    language-agnostic coverage that silently measures nothing is exactly the vacuous green the
    project forbids, so this spec MANDATES a test that a real per-FILE coverage shortfall
    still REDs the gate after the migration — including a low-coverage new file hidden
    inside an otherwise-high-coverage directory, which per-file enforcement catches and a
    package aggregate would not.
  package: pkg/gate

verification:
  level: integration
  test_command: go test ./pkg/gate/ ./cmd/backstop/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      ERADICATE the baked Go coverage analyzer pkg/gate/step_coverage.go and
      re-implement coverage LANGUAGE-AGNOSTIC over the SPEC-040 toolchain test runner.
      The baked Go-toolchain knowledge MUST be removed: the `go test ... -coverprofile`
      command construction (`goCoverageTarget`, `goCoveragePackagesTarget`,
      `repoSweepCoverageTarget`, `goCoverageTargetsForScope`), the
      `coverage: N% of statements` regex parser (`coverageRe`, `parseCoverageLine`),
      the whole-module `go test ./...` dedup read (`wholeModulePackageCoverage`,
      `packageLabelFromLine`), and the go.mod module-path reader (`goModulePath`) MUST
      NOT remain in the binary. After this spec the gate-side coverage step holds NO
      `go test` invocation, NO Go-coverage output parsing, and NO go.mod / Go-package
      knowledge. The coverage UNIT MUST be a LANGUAGE-AGNOSTIC coverage record keyed by a
      toolchain-DECLARED PATH at FILE granularity — NOT "package" (a Go-native concept;
      every other stack — istanbul/nyc, coverage.py, llvm-cov — and Go's own
      `-coverprofile` report per-FILE). There MUST be NO "package" noun anywhere in the
      gate-side coverage model. The gate consumes the CANONICAL
      `[]check.CoverageRecord` — the SINGLE shared type DEFINED by SPEC-042 (REQ-003):
      `{Path string, Covered int, Total int, Measured bool, Excluded bool, Metric string}`
      with RAW COUNTS (Covered/Total), NOT a pre-computed percent. The gate computes the
      verdict itself as `Covered/Total >= threshold` per FILE, which keeps it
      METRIC-BLIND and stops the pack from baking a percentage. Two conventions are
      MANDATORY: (a) `Total == 0` (no executable lines — pure declarations/interfaces/
      generated stubs) is N/A and MUST NOT be treated as a 0%-fail (the threshold check
      skips it); (b) `Metric` is a PACK-DECLARED label (statement/line/branch/…) that the
      gate NEVER interprets, compares, or branches on, but MUST SURFACE on the report so a
      polyglot repo does not silently compare statement-% against branch-% under one
      number. The producer (SPEC-042's coverage engine / the toolchain pass) NORMALIZES
      whatever its coverage tool emits into these per-file records — backstop stays format-
      and language-blind. It is PROHIBITED for the coverage step to exec `go`, to construct
      a `go test` command, to parse Go-coverage output text, to model coverage at "package"
      granularity, or to interpret/compare the `Metric` value. EXCLUSIONS
      (generated/vendored/no-executable-line files) are PACK-DECLARED, never baked: the
      gate skips a path the toolchain pack declares excluded and does NOT bake any
      exclusion list or pattern. The dependency on
      SPEC-040's toolchain test runner is a HARD PREREQUISITE — coverage is
      dynamic-toolchain (it needs the test runner's per-file coverage signal) and cannot
      land before that runner exists; the producer (SPEC-040's toolchain test pass) MUST
      emit coverage at this per-file/path granularity for the consumer contract here to
      hold.
    supports: collapse-legacy-codecheck-into-packs:REQ-011@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-002
    text: >
      ERADICATE the baked shared test runner cmd/backstop/shared_testrun.go
      (`sharedTestRunner`, `newSharedTestRunner`, `isWholeModuleGoTest`,
      `wholeModuleTest`) and its wiring in cmd/backstop/gate.go (the
      `newSharedTestRunner(projectRoot)` construction at ~:487, the
      `realCodeChecker.sharedRunner` injection, and the `sharedTest` argument to
      `buildCoverageStep`). After this spec NO baked shared `go test` runner survives
      and NO baked coupling feeds both code_check and coverage from one in-binary
      `go test ./... -coverprofile=/dev/null` exec. Coverage's input is the declared
      toolchain test pass (REQ-001), not a binary-resident `go test`. It is PROHIBITED
      to retain `sharedTestRunner` as a shim or to reconstruct an equivalent
      binary-resident whole-module `go test` runner under any other name.
    supports: collapse-legacy-codecheck-into-packs:REQ-011@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-003
    text: >
      Coverage MUST stay NON-VACUOUS through the migration, at FILE granularity. A real
      per-FILE coverage shortfall (a changed/new file whose computed `Covered/Total` ratio
      is below the spec-declared threshold) MUST produce a blocking coverage_threshold
      violation (status fail) after the re-implementation. The gate computes the ratio from
      the canonical record's RAW COUNTS and honors the `Total == 0` ⇒ N/A convention: a
      file with no executable lines is N/A (skipped from the threshold check), NEVER a
      0%-fail. Enforcement is PER-CHANGED-FILE and is
      NEVER rescued by aggregation at any level: an under-floor changed file MUST FAIL
      even when other files in its directory/package are high-coverage (a 2%-covered new
      file added to an otherwise-95% directory MUST fail — a package aggregate would hide
      it; this strictness is the point), and an over-floor changed file PASSES regardless
      of its siblings. The full/project gate flags EVERY file below threshold, never a
      whole-repo aggregate. A coverage step that reads NO coverage record when one was
      expected — an in-scope changed path with no record that is NOT declared-excluded —
      MUST be a LOUD blocking error (severity error), NEVER a silent pass; a
      pack-DECLARED-excluded path is skipped from the threshold check, not errored.
      HOWEVER, a pack-declared exclusion of an IN-SCOPE CHANGED file MUST be LOUDLY
      SURFACED on the report — the excluded path AND its declared exclusion reason MUST
      appear on the report surface, never silently dropped. This applies the bundle's
      RDQ-2 / REQ-006 "nothing ran is a loud, distinct report state" guardrail to
      "nothing measured because excluded": skipping the threshold check for a declared
      exclusion is permitted (warn, don't block), but doing so INVISIBLY is the
      one-declaration-removed "exclude everything and pass having checked nothing"
      vacuous green and is PROHIBITED. Exclusions of UNCHANGED files MAY stay quiet (they
      are not the suppression vector); it is specifically the in-scope CHANGED-file
      exclusion that must be loud. Silently measuring nothing — by absence OR by silent
      exclusion — is the vacuous-green failure mode the project forbids.
    supports: collapse-legacy-codecheck-into-packs:REQ-011@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-004
    text: >
      RE-EXPRESS the build-pass project-wide-scope exemption as a NEW, EXPLICIT
      declared engine-binding property `exempt_from_scope_filter` (boolean), and BUILD
      the bridge that maps that declared property onto `gate.Violation.ProjectWide` ON
      THE ENGINE PATH (cmd/backstop/pack_gate.go) — which does NOT exist today. CRITICAL
      LOCUS: the only code that sets `Violation.ProjectWide` today is the legacy
      `checkViolationsToGate` (cmd/backstop/gate.go:1173, `cv.Pass ==
      check.CheckTypeBuild`), reached ONLY through `realCodeChecker`, which SPEC-040
      DELETES — so editing or asserting against gate.go:1173 is asserting against soon-
      dead orphan code and is PROHIBITED as the locus. The engine path
      (pack_gate.go:411) today uses `binding.ScopeKind` ONLY for arg-shaping and NEVER
      sets `Violation.ProjectWide`. The real work is: (a) add
      `exempt_from_scope_filter bool` to the engine binding (pkg/pack/engine), DECOUPLED
      from `ScopeKind` (`ScopeKind` stays arg-shaping-only — it keeps appending
      `./...`/ProjectTarget); (b) in the engine dispatch, stamp each produced
      `gate.Violation.ProjectWide` from ITS PRODUCING binding's
      `exempt_from_scope_filter` value. The go-build engine declares
      `exempt_from_scope_filter: true`; golangci(lint) and go-test declare it FALSE (or
      unset). It is PROHIBITED for `ProjectWide` to be decided by any `CheckType` enum
      identity comparison. The exempt value MUST be resolved PER-VIOLATION (each
      violation carries its producing binding's value) and MUST NOT be aggregated to a
      gate-type level (see REQ-007).
    supports: collapse-legacy-codecheck-into-packs:REQ-012@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      PRODUCE a CheckType-consumer catalog scoped to GATE-SEMANTICS consumers — the
      sites where lint/build/test/findings `CheckType` identity drives gate behavior that
      the cutover could strand: SCOPE-FILTERING (the project-wide exemption), ENGINE
      DISPATCH / pass selection, and VIOLATION POLARITY/IDENTITY that feeds the gate
      verdict. It is EXPLICITLY NOT a catalog of every cosmetic `.Pass` display/log site
      (e.g. console/JSON rendering in pkg/check/output.go), which carry no gate-semantic
      decision and would make the catalog red on every added log line. CROSS-SPEC FACT
      (verified on `main` + SPEC-039 + SPEC-040): NO sibling spec deletes the `pkg/check`
      `CheckType` TYPE surface — SPEC-040 REQ-002's deletion is NARROW (`realCodeChecker`
      + methods, the `gate.StepCodeCheck*` entry, and `builtinToolchain` go/ts stacks
      only; `resolveToolchain`/`commandExecutor`/`buildExecutorsForConfigErr` are
      RETAINED reduced for the `code check` subcommand), and SPEC-039 deletes only narrow
      `manifest.go` compiled-reader
      items + the non-Go catch-all. So the `CheckType` enum, `passOrder`, `parseCheckType`,
      `RouteFile`, `Violation.Pass`/`PassResult.Pass`, `registry.go` Entries, and
      `parsers.go` findings stamping all SURVIVE the cutover — each MUST be cataloged with
      its REAL surviving post-cutover role, NOT mis-tagged DELETED. Each catalog entry's
      post-cutover source is one of {declared engine property (the NEW
      `exempt_from_scope_filter → Violation.ProjectWide` bridge, REQ-004), findings pack
      engine, surviving CheckType labeling/dispatch lingua-franca, or DELETED (only the
      genuinely-deleted sites: the shared-runner feeds REQ-002, and the orphaned
      `checkViolationsToGate`/gate.go:1173 build exemption that SPEC-040 removes with
      `realCodeChecker`)}. The catalog is the "don't drop a gate step on the floor" guard
      against the wholesale cutover (BUNDLE-011 REQ-001) silently stranding a consumer.
    supports: collapse-legacy-codecheck-into-packs:REQ-013@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-006
    text: >
      The CheckType-consumer catalog MUST be machine-enforced with a guard whose SCAN
      SCOPE is defined precisely enough that the discovered-set PROVABLY EQUALS the
      cataloged-set (so it neither reds on arrival nor becomes a tautology). The guard
      MUST restrict discovery to GATE-SEMANTIC keying: `CheckType`-identity comparisons
      and exemption/dispatch decisions in the gate path (`cmd/backstop` engine dispatch +
      scope-filter wiring, `pkg/gate` scope-filtering, and the `pkg/check` sites that feed
      a gate verdict), EXCLUDING pure display/serialization `.Pass.String()` sites. It
      MUST fail loudly on (a) any in-scope gate-semantic keying site absent from the
      catalog (dropped-gate-step), AND (b) any catalog entry whose site no longer exists
      in code (stale-entry — e.g. the DELETED rows once SPEC-040 removes their sites). The
      precise exclusion of cosmetic `.Pass.String()` display sites is what makes the
      discovered-set bounded and equal to the cataloged-set; completeness cannot rest on
      manual review.
    supports: collapse-legacy-codecheck-into-packs:REQ-013@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-007
    text: >
      The `exempt_from_scope_filter` value MUST be resolved PER-VIOLATION — each
      `gate.Violation` carries the `exempt_from_scope_filter` value of ITS PRODUCING
      binding/engine, set at engine-dispatch time (REQ-004) and consumed unchanged by
      `filterViolations` (pkg/gate/scope.go:194) via the `ProjectWide` field. It is
      PROHIBITED to aggregate the exempt value to a gate-type / pass-type level (that
      would re-introduce a "which value wins across packages" ambiguity that
      per-violation resolution dissolves). For the degenerate TRUE-CONFLICT case — the
      same file+line+rule claimed by two sources with DIFFERING exempt values — the
      EXEMPTING value WINS (`exempt_from_scope_filter: true` → not scope-filtered → the
      violation is shown): because the guarded failure mode is UNDER-broad scope
      filtering, the safe tiebreak direction is louder (show the violation).
    supports: collapse-legacy-codecheck-into-packs:REQ-012@1.0.0
    follows: STD-GO-001:GO-010

claims:
  # REQ-001 — eradicate baked Go coverage analyzer; re-implement language-agnostic
  - id: CLM-001
    requirement: REQ-001
    text: After this spec, pkg/gate/step_coverage.go (or its successor) constructs NO `go test` command and contains none of goCoverageTarget/goCoveragePackagesTarget/repoSweepCoverageTarget/goCoverageTargetsForScope/changedGoCoverageTargets — the gate-side coverage step holds no `go test` invocation
    tests:
      - TestCoverage_NoBakedGoTestCommandRemains
  - id: CLM-002
    requirement: REQ-001
    text: After this spec, the Go-coverage output parser (coverageRe/parseCoverageLine), the whole-module dedup read (wholeModulePackageCoverage/packageLabelFromLine), and the go.mod reader (goModulePath) are removed; the coverage step parses no Go-coverage text and holds no go.mod/Go-package knowledge
    tests:
      - TestCoverage_NoGoCoverageParsingOrGoModReaderRemains
  - id: CLM-003
    requirement: REQ-001
    text: The re-implemented coverage step consumes the canonical []check.CoverageRecord (Path/Covered/Total/Measured/Excluded/Metric, RAW COUNTS, FILE granularity, no "package" noun — the single shared type from SPEC-042) and applies the spec-declared threshold per FILE — proven by feeding declared per-file records and asserting the verdict, with no in-binary `go test` executed and no package-level modeling
    tests:
      - TestCoverage_ConsumesDeclaredPerFileCoverageRecord
  # REQ-002 — eradicate baked shared test runner
  - id: CLM-004
    requirement: REQ-002
    text: cmd/backstop/shared_testrun.go is deleted — sharedTestRunner/newSharedTestRunner/isWholeModuleGoTest/wholeModuleTest no longer exist anywhere in cmd/backstop
    tests:
      - TestSharedRunner_Eradicated
  - id: CLM-005
    requirement: REQ-002
    text: The shared-runner wiring in cmd/backstop/gate.go is removed — there is no newSharedTestRunner construction, no realCodeChecker.sharedRunner injection, and buildCoverageStep is no longer passed a shared `go test` runner
    tests:
      - TestSharedRunner_WiringRemovedFromGate
  - id: CLM-006
    requirement: REQ-002
    text: No binary-resident whole-module `go test ./...` runner exists under any name after eradication — a guard asserts no cmd/backstop or pkg/gate source constructs a whole-module `go test` runner, so a renamed shared runner FAILS
    tests:
      - TestSharedRunner_NoRenamedWholeModuleGoTestRunner
  # REQ-003 — coverage stays NON-vacuous
  - id: CLM-007
    requirement: REQ-003
    text: A real per-FILE coverage shortfall (a changed/new file measured below its spec-declared threshold in the declared per-file record) produces a blocking coverage_threshold violation with status fail after the re-implementation
    tests:
      - TestCoverage_RealPerFileShortfallStillReds
  - id: CLM-008
    requirement: REQ-003
    text: An in-scope changed PATH with NO coverage record that is NOT declared-excluded produces a LOUD blocking error (severity error, status fail), never a silent pass — while a pack-DECLARED-excluded path with no record is SKIPPED, not errored; silently measuring nothing is rejected, declared exclusion is honored
    tests:
      - TestCoverage_UnmeasuredNonExcludedPathIsLoudError
      - TestCoverage_DeclaredExcludedPathIsSkippedNotErrored
  - id: CLM-009
    requirement: REQ-003
    text: An over-floor changed FILE PASSES regardless of its siblings — never failed because another file in its directory/package is below floor (per-changed-file, no aggregation at any level)
    tests:
      - TestCoverage_PerChangedFile_OverFloorPassesRegardlessOfSiblings
  - id: CLM-010
    requirement: REQ-003
    text: An under-floor changed FILE FAILS even when other files in its directory/package are high-coverage — a 2%-covered new file in an otherwise-95% directory REDs, where a package aggregate would hide it; never rescued by aggregation
    tests:
      - TestCoverage_PerChangedFile_UnderFloorFailsHiddenByPackageAggregate
  - id: CLM-025
    requirement: REQ-003
    text: A pack-DECLARED exclusion of an IN-SCOPE CHANGED file is LOUDLY SURFACED on the report (the excluded path AND its declared exclusion reason appear on the report surface), never silently dropped — closing the "declare Excluded:true on a changed file to suppress its coverage requirement invisibly" vacuous-green vector; an exclusion of an UNCHANGED file MAY stay quiet
    tests:
      - TestCoverage_DeclaredExclusionOfChangedFileIsLoudlySurfaced
      - TestCoverage_DeclaredExclusionOfUnchangedFileMayStayQuiet
  - id: CLM-026
    requirement: REQ-001
    text: The gate computes the per-file verdict from the canonical record's RAW COUNTS as Covered/Total >= threshold (it does NOT consume a pre-computed percent), staying METRIC-BLIND — a record with Total==0 is treated as N/A (skipped), NEVER a 0%-fail; proven by feeding raw-count records including a Total==0 file and asserting the ratio verdict and the N/A skip
    tests:
      - TestCoverage_GateComputesRatioFromRawCountsMetricBlind
      - TestCoverage_TotalZeroIsNANotZeroPercentFail
  - id: CLM-027
    requirement: REQ-001
    text: The pack-declared Metric label (statement/line/branch/…) is SURFACED on the report and NEVER interpreted, compared, or branched on by the gate — a polyglot record set with differing Metric values surfaces each file's metric rather than collapsing them under one number
    tests:
      - TestCoverage_MetricLabelSurfacedNeverInterpreted
  # REQ-004 — exempt_from_scope_filter declared property + engine-path ProjectWide bridge
  - id: CLM-011
    requirement: REQ-004
    subject: pkg/pack
    text: The engine binding (pkg/pack/engine) gains an `exempt_from_scope_filter bool` property DECOUPLED from ScopeKind; the go-build engine declares it true, golangci(lint) and go-test declare it false/unset — asserted against the declared binding records
    tests:
      - TestExemption_BindingDeclaresExemptFromScopeFilterDecoupledFromScopeKind
  - id: CLM-012
    requirement: REQ-004
    text: The engine dispatch (cmd/backstop/pack_gate.go) stamps each produced gate.Violation.ProjectWide from its producing binding's exempt_from_scope_filter value — a NEW bridge that did not exist (the engine path previously never set ProjectWide). A go-build (exempt=true) violation arrives with ProjectWide true
    tests:
      - TestExemption_EnginePathStampsProjectWideFromExemptProperty
  - id: CLM-013
    requirement: REQ-004
    text: A go-build break in an UNCHANGED file still REDs a diff-scoped gate END-TO-END through the engine path — its exempt=true violation survives filterViolations (pkg/gate/scope.go:194) because ProjectWide is set, guarding the under-broad regression
    tests:
      - TestExemption_BuildBreakUnchangedFileStillRedsDiffScoped
  - id: CLM-014
    requirement: REQ-004
    text: A lint (golangci) violation in an UNCHANGED file IS scope-filtered out of a diff-scoped gate through filterViolations — exempt_from_scope_filter is false for golangci, so ProjectWide is false and the unchanged-file violation is dropped
    tests:
      - TestExemption_LintViolationUnchangedFileIsFiltered
  - id: CLM-015
    requirement: REQ-004
    text: A go-test violation in an UNCHANGED file IS scope-filtered out of a diff-scoped gate through filterViolations — exempt_from_scope_filter is false for go-test, so ProjectWide is false and the unchanged-file violation is dropped
    tests:
      - TestExemption_TestViolationUnchangedFileIsFiltered
  - id: CLM-016
    requirement: REQ-004
    text: A findings (semgrep/ast-grep) violation in an UNCHANGED file IS scope-filtered out of a diff-scoped gate through filterViolations — exempt_from_scope_filter is false/unset for findings engines, so ProjectWide is false and the unchanged-file violation is dropped
    tests:
      - TestExemption_FindingsViolationUnchangedFileIsFiltered
  - id: CLM-017
    requirement: REQ-004
    subject: pkg/pack
    text: ScopeKind stays arg-shaping-only and DECOUPLED — golangci/go-build/go-test all remain ScopeKindProjectWide (each still appends its ./... ProjectTarget) while ONLY go-build is exempt_from_scope_filter; the two properties are independent and a test asserts ScopeKind is not consulted for the ProjectWide decision
    tests:
      - TestExemption_ScopeKindDecoupledFromExemptDecision
  # REQ-007 — per-violation exempt resolution + conflict tiebreak
  - id: CLM-018
    requirement: REQ-007
    text: Two build passes across different packages with DIFFERING exempt_from_scope_filter values resolve PER-VIOLATION — each violation carries ITS producing binding's value; the exempt=true package's violations get ProjectWide true and the exempt=false package's get ProjectWide false, with no gate-type-level aggregation
    tests:
      - TestExemption_PerViolationResolutionNoGateTypeAggregation
  - id: CLM-019
    requirement: REQ-007
    text: In the degenerate true-conflict (same file+line+rule claimed by two sources with differing exempt values) the EXEMPTING value WINS — the violation is shown (not scope-filtered), because the safe direction against under-broad filtering is louder
    tests:
      - TestExemption_TrueConflictExemptingValueWins
  # REQ-005 — CheckType-consumer catalog (gate-semantics scope)
  - id: CLM-020
    requirement: REQ-005
    kind: absence
    text: The spec produces a CheckType-consumer catalog scoped to GATE-SEMANTIC consumers (scope-filtering, engine dispatch, violation verdict) and EXCLUDES cosmetic .Pass.String() display/serialization sites (e.g. pkg/check/output.go's five render sites) — so the catalog does not red on every added log line
    tests:
      - TestCatalog_EnumeratesGateSemanticConsumersExcludesDisplaySites
  - id: CLM-021
    requirement: REQ-005
    kind: absence
    text: Surviving pkg/check CheckType sites (passOrder, Violation.Pass/PassResult.Pass, Executors/applicableChecks dispatch, registry Entries, manifest enum+routing, parsers findings stamping) are tagged SURVIVING with their real post-cutover role — NOT mis-tagged DELETED — because no sibling spec deletes the CheckType type (SPEC-040 REQ-002 narrow; SPEC-039 narrow); only C-2 (orphaned gate.go:1173) and C-3 (shared-runner feeds) are DELETED
    tests:
      - TestCatalog_SurvivingSitesNotMistaggedDeleted
  # REQ-006 — catalog completeness is machine-enforced over a bounded scan scope
  - id: CLM-022
    requirement: REQ-006
    kind: absence
    text: The completeness guard restricts discovery to GATE-SEMANTIC CheckType keying (identity comparisons + exemption/dispatch decisions in the gate path), EXCLUDING pure .Pass.String() display sites, so the discovered-set provably equals the cataloged-set and the guard neither reds on arrival nor is a tautology
    tests:
      - TestCatalog_GuardScansGateSemanticSurfaceOnly
  - id: CLM-023
    requirement: REQ-006
    kind: absence
    text: The guard FAILS when a gate-semantic CheckType-keyed source site exists with no corresponding catalog entry — proven by an injected/fixture keying site that is absent from the catalog, which the guard must red on (not a tautology)
    tests:
      - TestCatalog_GuardFailsOnUnlistedConsumer
  - id: CLM-024
    requirement: REQ-006
    kind: absence
    text: The guard FAILS on a stale catalog entry whose keying site no longer exists in code (e.g. a DELETED row that survives after its site is removed), preventing the catalog from drifting out of sync with the consumer surface
    tests:
      - TestCatalog_GuardFailsOnStaleEntry

contracts:
  - file: pkg/gate/step_coverage.go
    provides:
      - name: StepCoverageThresholdScopedFunc
        kind: function
        signature: "func StepCoverageThresholdScopedFunc(coverage []check.CoverageRecord, specs []SpecVerification, scope *GateScope) StepFunc"
        notes: "REWRITTEN: consumes the canonical PER-FILE []check.CoverageRecord (the SHARED type defined by SPEC-042 REQ-003 — Path/Covered/Total/Measured/Excluded/Metric, RAW COUNTS) instead of a CommandRunner that execs `go test`. The gate computes the verdict as `Covered/Total >= threshold` PER CHANGED/NEW FILE — it consumes RAW COUNTS and computes the ratio itself, staying METRIC-BLIND (the pack bakes no percentage). Total==0 (no executable lines) is N/A: NOT a 0%-fail, skipped from the threshold check. An under-floor changed file REDs even if its directory/package siblings are high-coverage, never rescued by aggregation at any level (REQ-003/CLM-009/CLM-010). A missing measurement for an in-scope NON-excluded path is a loud blocking error (REQ-003/CLM-008); a pack-declared-excluded path is skipped from the check but a CHANGED-file exclusion is loudly surfaced (CLM-025). The ratio-from-raw-counts/metric-blind/Total==0-N/A logic is CLM-026; the Metric label is surfaced on the report, never interpreted (CLM-027). The go test command constructors, coverage regex, whole-module dedup, goModulePath reader, and all package-level modeling are DELETED (CLM-001/CLM-002)."
      - name: StepCoverageThresholdFunc
        kind: function
        signature: "func StepCoverageThresholdFunc(coverage []check.CoverageRecord, specs []SpecVerification) StepFunc"
        notes: "REWRITTEN convenience wrapper: delegates to StepCoverageThresholdScopedFunc with nil scope, now over the canonical per-file []check.CoverageRecord rather than a CommandRunner."
      - name: CoverageTarget
        kind: type
        signature: "type CoverageTarget struct"
        absent: true
        notes: "DELETED (REQ-001/CLM-001): the baked `go test ... -coverprofile` command descriptor (Stack/Label/Command/Args). Target selection was Go-toolchain knowledge; coverage now reads the declared toolchain pass's record. Declared absent so its reappearance is a regression."
      - name: parseCoverageLine
        kind: function
        signature: "func parseCoverageLine(line string) (float64, bool)"
        absent: true
        notes: "DELETED (REQ-001/CLM-002): the `coverage: N% of statements` Go-output parser. Coverage parsing is now the pack/toolchain pass's job; the gate consumes a normalized record."
    consumes:
      - source: pkg/check
        name: CoverageRecord
        kind: type
      - source: pkg/gate
        name: GateScope
        kind: type
      - source: pkg/gate
        name: SpecVerification
        kind: type
  - file: pkg/pack/engine/binding.go
    provides:
      - name: ExemptFromScopeFilter
        kind: variable
        signature: "ExemptFromScopeFilter bool"
        notes: "NEW (REQ-004/CLM-011): an explicit declared property on EngineBinding, DECOUPLED from ScopeKind (which stays arg-shaping-only). When true, violations produced by this binding are exempt from diff-scope filtering (mapped to gate.Violation.ProjectWide on the engine path, C-1). The DefaultRegistry go-build entry declares it true; golangci and go-test declare it false/unset (CLM-017). It is the declared replacement for the deleted baked `cv.Pass == check.CheckTypeBuild` identity check."
    consumes: []
  - file: cmd/backstop/pack_gate.go
    provides:
      - name: dispatchPackEngines
        kind: function
        signature: "func dispatchPackEngines(...) ([]gate.Violation, error)"
        notes: "MODIFIED (REQ-004/REQ-007/CLM-012/CLM-018): the engine dispatch now stamps each produced gate.Violation.ProjectWide from ITS producing binding's ExemptFromScopeFilter value — the NEW bridge that did not exist (pack_gate.go:411 previously used ScopeKind only for arg-shaping and never set ProjectWide, so engine-path violations were never project-wide-exempt). Resolution is PER-VIOLATION (each violation carries its binding's value); no gate-type-level aggregation (REQ-007/CLM-018). True file+line+rule conflicts with differing values resolve to the exempting (louder) value (CLM-019). ScopeKind is unchanged and still drives ./...-arg-shaping only (CLM-017)."
    consumes:
      - source: pkg/pack/engine
        name: ExemptFromScopeFilter
        kind: variable
      - source: pkg/gate
        name: Violation
        kind: type
  - file: cmd/backstop/gate.go
    provides:
      - name: sharedTestRunner
        kind: type
        signature: "type sharedTestRunner struct"
        absent: true
        notes: "DELETED (REQ-002/CLM-004): the baked whole-module `go test ./... -coverprofile=/dev/null` dedup runner that coupled code_check and coverage. Coverage now reads the declared toolchain test pass; code_check's test pass is owned by SPEC-040. Its file cmd/backstop/shared_testrun.go is DELETED (confirmed by TestSharedRunner_Eradicated), so the absence probe scope is repointed here to gate.go — the former wiring site — where absence is scannable and meaningful; declared absent so a renamed reconstruction (CLM-006) reappearing in the gate wiring is caught. The broader all-of-cmd/backstop eradication guard is carried by the compiled TestSharedRunner_Eradicated / TestSharedRunner_NoRenamedWholeModuleGoTestRunner."
      - name: newSharedTestRunner
        kind: function
        signature: "func newSharedTestRunner(dir string) *sharedTestRunner"
        absent: true
        notes: "DELETED (REQ-002/CLM-004): the constructor for the eradicated shared runner. Scope repointed from the DELETED cmd/backstop/shared_testrun.go to gate.go (the former wiring site) so the grep absence probe scans a real file; TestSharedRunner_WiringRemovedFromGate is the compiled companion guard for gate.go."
      - name: checkViolationsToGate
        kind: function
        signature: "func checkViolationsToGate(cvs []check.Violation) []gate.Violation"
        absent: true
        notes: "DELETED (catalog C-2/C-3; REQ-005/CLM-021): the legacy `cv.Pass == check.CheckTypeBuild` ProjectWide setter and `rule = cv.Pass.String()` derivation are orphaned once realCodeChecker (their only caller) is deleted by SPEC-040. NOT edited in place — the ProjectWide behavior is REPLACED by the engine-path bridge (C-1, pack_gate.go). Declared absent so an attempt to retain/revive the baked identity check is caught. The shared-runner wiring (newSharedTestRunner construction, realCodeChecker.sharedRunner injection, the sharedTest argument to buildCoverageStep) is also removed here (REQ-002/CLM-005)."
    consumes:
      - source: pkg/check
        name: CoverageRecord
        kind: type
  - file: cmd/backstop/checktype_catalog.go
    provides:
      - name: CheckTypeConsumerCatalog
        kind: function
        signature: "func CheckTypeConsumerCatalog() []CheckTypeConsumer"
        notes: "NEW (REQ-005/CLM-020/CLM-021): the machine-readable CheckType-consumer catalog — one entry per identity-keying site (Site, KeysOn, PostCutoverSource), mirroring the documented catalog table (C-1…C-8). The REQ-006 guard reconciles the discovered keying-site set against this catalog (CLM-022/CLM-023/CLM-024)."
      - name: CheckTypeConsumer
        kind: type
        signature: "type CheckTypeConsumer struct"
        notes: "NEW: one catalog row — Site (file:symbol), KeysOn (build/lint/test/findings/all), PostCutoverSource (declared-engine-property | toolchain-test-pass | findings-pack-engine | deleted)."
    consumes: []
---

# SPEC-041: Coverage Re-implementation + CheckType-Consumer Catalog

## Overview

This spec is **Seed 3 of BUNDLE-011** (collapse the legacy `pkg/check` engine into
pack-declared toolchain packs), owning **REQ-011, REQ-012, REQ-013** (RDQ-6). It is a
**downstream spec of SPEC-040** (Seed 2, the toolchain-pack substrate + `go-toolchain`
cutover): SPEC-040 establishes the declared toolchain **test** pass; this spec
re-implements **coverage** over that pass and eradicates the last baked test/coverage
machinery.

Three deliverables, all from RDQ-6:

1. **Migrate the test+coverage stack as a unit (REQ-011, REQ-012 of the bundle).**
   Today `pkg/gate/step_coverage.go` is a baked Go coverage analyzer (KEPT baked by
   **SPEC-038 REQ-009**'s descope — it was *never* deleted by BUNDLE-009, despite stale
   prose to the contrary), and `cmd/backstop/shared_testrun.go`'s `newSharedTestRunner`
   couples code_check's test FAILs and coverage's per-file read through a single
   baked `go test ./... -coverprofile=/dev/null` exec. Both are eradicated. Coverage is
   re-implemented language-agnostic over SPEC-040's toolchain test runner; **no baked
   shared runner and no baked Go coverage analyzer survive.**

2. **Re-express the build-pass exemption as a declared property + build the engine-path
   bridge (bundle REQ-012).** Today the ONLY code that sets `Violation.ProjectWide` is the
   baked enum identity check `cv.Pass == check.CheckTypeBuild` (gate.go:1173) in
   `checkViolationsToGate` — reachable **only** through `realCodeChecker`, which **SPEC-040
   deletes**. After SPEC-040 lands, that locus is dead orphan code; editing or asserting
   against it passes trivially. The real work is on the **engine path**: introduce a NEW,
   EXPLICIT engine-binding property **`exempt_from_scope_filter` (boolean)**, DECOUPLED
   from `ScopeKind` (which stays arg-shaping-only), and **build the bridge that maps
   `binding.exempt_from_scope_filter → gate.Violation.ProjectWide` in
   `cmd/backstop/pack_gate.go`** — a bridge that **does not exist today** (pack_gate.go:411
   uses `ScopeKind` only to shape `./...` and never sets `ProjectWide`). The build-pass
   exemption is then simply "the go-build engine declares `exempt_from_scope_filter:
   true`"; golangci(lint) and go-test declare it false. The exempt value is resolved
   **per-violation** (each violation carries its producing binding's value), never
   aggregated. This is consistent with SPEC-035's tool-neutral gate-TYPE direction; no
   `CheckType` enum identity drives scope.

3. **Produce a CheckType-consumer catalog (bundle REQ-013).** Enumerate the
   GATE-SEMANTIC `CheckType`-identity consumers — the identity comparisons and
   exemption/dispatch/verdict decisions where `CheckType` drives gate behavior —
   EXCLUDING cosmetic `.Pass.String()` display/serialization sites, and document each
   one's post-cutover source, backed by a machine guard that fails if a gate-semantic
   keying site exists with no catalog entry (and on any stale entry). This is the
   explicit "don't drop a gate step on the floor" guard against the wholesale
   (BUNDLE-011 REQ-001 / SPEC-040) cutover silently stranding a consumer.

**Hard dependency on SPEC-040 + SPEC-042.** Coverage is **dynamic-toolchain** — it needs
the PER-FILE coverage signal that only the toolchain test runner produces.
**Producer↔consumer contract — ONE shared type:** the canonical coverage record is DEFINED
by **SPEC-042** (REQ-003) as `check.CoverageRecord{Path, Covered, Total, Measured,
Excluded, Metric}` with raw counts; THIS spec CONSUMES that exact type — there is NO second
`CoverageRecord` shape (no separate producer/consumer struct). SPEC-042 is the producer
(its coverage engine emits `[]check.CoverageRecord`); this spec is the consumer (the gate
computes `Covered/Total >= threshold`, metric-blind, `Total==0`⇒N/A). Both are DRAFT specs
that MUST agree on the single shared type; this is flagged for producer↔consumer coherence
review. SPEC-040's toolchain test pass is what carries that coverage signal through the
declared engine. Coverage was historically *descoped* (SPEC-038 REQ-009 kept it baked)
precisely because the language-agnostic test runner did not yet exist. SPEC-040 establishes
that runner and SPEC-042 the canonical record; this spec consumes both. **This spec cannot
be implemented before SPEC-040's toolchain test pass and SPEC-042's `check.CoverageRecord`
exist** — that dependency is real and is stated as a hard prerequisite in REQ-001.

**In scope:** coverage eradication + language-agnostic re-implementation, shared-runner
eradication, non-vacuousness guard, declared build-exemption, the CheckType-consumer
catalog + completeness guard.

**Out of scope (fenced to sibling seeds):** the Step-2 cutover itself, the
`go-toolchain` pack, the golden-equivalence harness, and the no-pack loud-warn report
state → **Seed 2 / SPEC-040** (the prerequisite — referenced, not re-authored here). The
dead-code deletions (standards-manifest reader, non-Go semgrep catch-all) → **Seed 1 /
SPEC-039**.

## Requirements

Requirements are enumerated in the `requirements:` frontmatter (REQ-001 … REQ-007), each
tracing to a BUNDLE-011 requirement via `supports`. Mapping:

| Spec REQ | Bundle REQ (RDQ-6) | What it commits to |
| --- | --- | --- |
| REQ-001 | REQ-011 | Eradicate `pkg/gate/step_coverage.go`; re-implement coverage language-agnostic over the SPEC-040 toolchain test runner; no baked `go test` / Go-coverage parsing / go.mod knowledge survives. |
| REQ-002 | REQ-011 | Eradicate the baked shared runner `cmd/backstop/shared_testrun.go` + its wiring; no baked shared `go test` runner survives. |
| REQ-003 | REQ-011 | Coverage stays NON-vacuous at FILE granularity: a real per-FILE shortfall still REDs; per-changed-FILE enforcement (never rescued by aggregation — a low-coverage file hidden in a high-coverage directory still fails); an in-scope changed path with no record that is NOT pack-declared-excluded is a loud error, never a silent pass. |
| REQ-004 | REQ-012 | Introduce the `exempt_from_scope_filter` engine-binding property (decoupled from `ScopeKind`) and build the `exempt_from_scope_filter → Violation.ProjectWide` bridge on the engine path (`pack_gate.go`); go-build declares it true, golangci/go-test false. NOT anchored to the soon-dead `checkViolationsToGate`/gate.go:1173. |
| REQ-005 | REQ-013 | Produce the CheckType-consumer catalog scoped to GATE-SEMANTIC consumers (scope-filter/dispatch/verdict, excluding cosmetic `.Pass.String()` display): the new engine-path bridge (C-1), the genuinely-DELETED sites (C-2/C-3), and the SURVIVING `pkg/check` `CheckType` sites (C-4…C-8) with their real post-cutover role. |
| REQ-006 | REQ-013 | The catalog is machine-enforced complete over a precisely-defined scan scope: a guard test FAILS on any keying site absent from the catalog AND any stale catalog entry. |
| REQ-007 | REQ-012 | The exempt value is resolved per-violation (each violation carries its producing binding's value), never aggregated; in a true file+line+rule conflict with differing values the exempting (louder) value wins. |

### The CheckType-consumer catalog (REQ-005)

The catalog enumerates every non-test source site that keys on lint/build/test/findings
`CheckType` identity (verified on `main` 2026-06-24). Each row records the post-cutover
source. This table IS the catalog deliverable; the REQ-006 guard reconciles code against
it.

**Scope (path b).** This catalog enumerates GATE-SEMANTIC `CheckType` consumers — sites
where `CheckType` identity drives scope-filtering, engine dispatch, or the violation
verdict — NOT every cosmetic `.Pass.String()` display/serialization site (pkg/check/
output.go has five such render sites at :78/:85/:131/:135/:173; they carry no
gate-semantic decision and are deliberately EXCLUDED so the catalog does not red on every
added log line). Post-cutover source is one of: **declared engine property**, **findings
pack engine**, **surviving CheckType labeling/dispatch** (the `CheckType` type stays the
lingua franca for pass identity; NO sibling spec deletes it — SPEC-040 REQ-002's deletion
is narrow: `realCodeChecker`/methods + `builtinToolchain` only (`resolveToolchain`/
`commandExecutor`/`buildExecutorsForConfigErr` are RETAINED reduced); SPEC-039
deletes only narrow `manifest.go` reader items), or **DELETED** (only the genuinely
removed sites). DELETED rows must be REMOVED from the catalog once their site is gone, or
the REQ-006 stale-entry guard reds.

| # | Site | Keys on | Post-cutover source |
| --- | --- | --- | --- |
| C-1 | `cmd/backstop/pack_gate.go` (engine dispatch) — NEW `binding.exempt_from_scope_filter → gate.Violation.ProjectWide` stamp (REQ-004) | exempt property (per-binding) | **Declared engine property** — the SOLE live `ProjectWide` locus after the cutover. go-build declares `exempt_from_scope_filter: true`; golangci/go-test false. No `CheckType` identity. |
| C-2 | `cmd/backstop/gate.go:1173` `checkViolationsToGate` — `ProjectWide: cv.Pass == check.CheckTypeBuild` | build identity | **DELETED** — orphaned once `realCodeChecker` (its only caller) is deleted by SPEC-040. Its scope-filtering behavior is REPLACED by C-1 on the engine path, not edited in place. |
| C-3 | `cmd/backstop/gate.go:~487` `newSharedTestRunner` + `realCodeChecker.sharedRunner` + `buildCoverageStep(..., sharedTest)` | test feed | **DELETED** (REQ-002): coverage's input becomes the declared toolchain test pass (REQ-001); the shared runner is eradicated. |
| C-4 | `cmd/backstop/code_check.go:187,229` — `Pass: check.CheckTypeFindings` stamping | findings identity | **Findings pack engine** — findings already run through the pack engine (registry.go:310; registry.go:229 builds executors only for lint/build/test). Stamping persists; its source is the findings engine. |
| C-5 | `pkg/check/parsers.go:50` — `parser(out, CheckTypeFindings)` findings stamping | findings identity | **Findings pack engine** — SURVIVES (parsers.go is not deleted by any sibling spec); same findings source as C-4. |
| C-6 | `pkg/check/check.go` — `passOrder`, `Violation.Pass`/`PassResult.Pass` fields, `Executors map[CheckType]`/`applicableChecks` dispatch | all four | **Surviving CheckType labeling/dispatch** — SURVIVES the cutover (no sibling spec deletes the `CheckType` type or pass dispatch); remains the lingua franca that labels which pass produced a violation and selects executors for whatever passes are wired. |
| C-7 | `pkg/check/registry.go:69,75,83,229,…` — toolchain `Entries map[CheckType]ToolchainEntry` + `buildExecutorsForConfig*`/`declaredEntries` helpers | all four | **Surviving CheckType labeling/dispatch** — SURVIVES; SPEC-040 deletes only `builtinToolchain`'s go/ts STACKS, NOT the `Entries`-keyed-by-`CheckType` mechanism or the executor builders (`resolveToolchain`/`commandExecutor`/`buildExecutorsForConfigErr` are RETAINED in reduced declared-only form for the standalone `code check` subcommand). |
| C-8 | `pkg/check/manifest.go` — `CheckType` enum, `parseCheckType` (singular), `RouteFile`, `routeFileDefaults`, `Manifest`, `LoadManifest`, `defaultManifest` | all four | **Surviving CheckType labeling/dispatch** — these are exactly SPEC-039's PRESERVE list (its round-2 reversal DELETES `parseCheckTypes` plural + the `.manifest.json` routing-schema arm; `LoadManifest` collapses to `defaultManifest()`, `Manifest` becomes empty). The surviving enum + `defaultManifest`→`routeFileDefaults` routing is the cutover's pass-identity lingua franca, not dead. |

REQ-006 makes this table authoritative: the guard scans the precisely-defined
GATE-SEMANTIC surface (excluding cosmetic `.Pass.String()` display sites) and fails on any
in-scope keying site not represented here AND on any row whose site no longer exists. The
catalog is the contract; the guard prevents drift. The DELETED rows (C-2, C-3) are the
only genuinely-removed sites; the surviving rows (C-4…C-8) document the `CheckType` type's
real post-cutover role so the cutover cannot silently strand a gate-semantic consumer.

## Implementation

Target package: `pkg/gate` (coverage step), with wiring changes in `cmd/backstop`
(shared-runner eradication, declared-property exemption, catalog guard). The processing
steps the planner must map tasks to:

1. **Eradicate the baked Go coverage analyzer (REQ-001).** Delete the Go-toolchain
   machinery from `pkg/gate/step_coverage.go`: the `go test ... -coverprofile` command
   constructors (`goCoverageTarget`, `goCoveragePackagesTarget`,
   `repoSweepCoverageTarget`, `goCoverageTargetsForScope`,
   `changedGoCoverageTargets`), the Go-coverage regex parser (`coverageRe`,
   `parseCoverageLine`), the whole-module dedup read (`wholeModulePackageCoverage`,
   `packageLabelFromLine`), and the go.mod reader (`goModulePath`). After this step the
   coverage step performs no `go` exec, no Go-coverage text parsing, and holds no
   Go-package/go.mod knowledge.

2. **Re-implement coverage language-agnostic over the canonical record (REQ-001).** The
   coverage step consumes the canonical `[]check.CoverageRecord` (the single shared type
   from SPEC-042 — `{Path, Covered, Total, Measured, Excluded, Metric}`, RAW COUNTS, FILE
   granularity, NO "package" noun) and computes the verdict itself as `Covered/Total >=
   threshold` per FILE — metric-blind (it never interprets `Metric`, only surfaces it),
   with `Total==0`⇒N/A (skipped, never a 0%-fail). The changed-file selection and
   threshold-derivation logic (`coverageThresholdsForScope`, `coverageSpecInScope`) is
   RETAINED as language-agnostic scoping/verdict logic but re-keyed from package to PATH
   and fed the canonical record rather than an in-binary `go test` run. The gate decides
   verdicts purely from the record; it never re-runs or re-parses tests, and never
   aggregates files into a package/directory roll-up.

3. **Eradicate the baked shared runner (REQ-002).** Delete
   `cmd/backstop/shared_testrun.go` and its wiring in `cmd/backstop/gate.go`: the
   `newSharedTestRunner(projectRoot)` construction, the `realCodeChecker.sharedRunner`
   injection, and the `sharedTest` argument threaded into `buildCoverageStep`. Coverage
   and code_check no longer share a binary-resident `go test`; each consumes the declared
   toolchain pass.

4. **Non-vacuousness guard (REQ-003), per-FILE.** Each changed/new FILE is checked
   against the threshold on its own — never rescued by aggregation (a low-coverage file
   in a high-coverage directory still REDs). An in-scope changed PATH with NO measurement
   that is NOT pack-declared-excluded produces a LOUD blocking error (severity error,
   status fail), never a silent pass; a declared-excluded path is skipped. A real per-file
   shortfall still produces a blocking `coverage_threshold` violation.

5. **`exempt_from_scope_filter` property + engine-path ProjectWide bridge (REQ-004).**
   Add `ExemptFromScopeFilter bool` to the engine binding (`pkg/pack/engine/binding.go`),
   DECOUPLED from `ScopeKind` (which keeps driving `./...`-arg-shaping only). Declare it
   `true` on the go-build engine, `false`/unset on golangci and go-test. In the engine
   dispatch (`cmd/backstop/pack_gate.go`), stamp each produced `gate.Violation.ProjectWide`
   from ITS producing binding's `ExemptFromScopeFilter` value — this bridge does NOT exist
   today (pack_gate.go:411 reads `ScopeKind` only for arg-shaping and never sets
   `ProjectWide`). DO NOT edit or assert against `checkViolationsToGate`/gate.go:1173: it
   is orphaned once `realCodeChecker` is deleted by SPEC-040, so it is cataloged DELETED
   (C-2/C-3), and its ProjectWide behavior is REPLACED here on the engine path. The
   `filterViolations` consumer (`pkg/gate/scope.go:194`) is unchanged — it already keys on
   `Violation.ProjectWide`; this step is what makes engine-path build violations arrive
   with `ProjectWide` set.

6. **Per-violation exempt resolution + conflict tiebreak (REQ-007).** The stamp in step 5
   is PER-VIOLATION: each `gate.Violation` carries the `ExemptFromScopeFilter` value of its
   producing binding. NEVER aggregate the value to a gate-type / pass-type level. For the
   degenerate true-conflict (same file+line+rule from two sources with differing values),
   the exempting (`true` → shown, not scope-filtered) value wins — the safe direction
   against under-broad filtering is louder.

7. **CheckType-consumer catalog + completeness guard (REQ-005, REQ-006).** The catalog
   table above (C-1…C-8) is the documented deliverable, mirrored by a machine-readable
   `CheckTypeConsumerCatalog()`. Scope = GATE-SEMANTIC consumers (scope-filter, dispatch,
   verdict); cosmetic `.Pass.String()` display/serialization sites are EXCLUDED. The guard
   (a test in `cmd/backstop` / `pkg/gate`) restricts discovery to that gate-semantic surface
   and reconciles the discovered-set against the cataloged-set, failing if any discovered
   site is absent from the catalog (dropped-gate-step) or any catalog entry's site no longer
   exists (stale entry — e.g. the DELETED rows C-2/C-3 once SPEC-040 removes
   `realCodeChecker`/`checkViolationsToGate` and the shared runner is eradicated). NOTE: the
   surviving rows C-4…C-8 are NOT deleted by any sibling spec — the `CheckType` type stays
   the gate's pass-identity lingua franca, so they are cataloged as SURVIVING, not DELETED.

## Verification

- **Level:** `integration` — the coverage re-implementation spans `pkg/gate` (the step)
  and `cmd/backstop` (wiring, the declared-property exemption, the catalog guard), and
  the non-vacuousness claim is only meaningful end-to-end against the declared toolchain
  coverage record.
- **Test command:** `go test ./pkg/gate/ ./cmd/backstop/ -race -coverprofile=cover.out`
- **Coverage threshold:** 80 (integration level).

Claims (CLM-NNN) are enumerated in the `claims:` frontmatter, each mapping a REQ to
mandated test names. Every requirement carries at least one passing and one failing/edge
claim. The exempt matrix (REQ-004) covers all four engine classes end-to-end through
`filterViolations`: go-build (`exempt_from_scope_filter: true` → ProjectWide → unchanged-
file break still REDs) versus golangci/go-test/findings (`false` → unchanged-file
violation filtered out), so no cell is untested. REQ-007 pins the per-violation resolution
and the true-conflict tiebreak.

## Sharp Edges

1. **Coverage must stay NON-vacuous through the migration.** A language-agnostic
   coverage step that silently reads an empty/absent coverage signal and reports `pass`
   is exactly the vacuous green the project forbids — and it is the *most likely* way
   this migration goes wrong, because the eradicated baked analyzer used to *produce*
   the signal it then checked. REQ-003 + CLM-007/CLM-008 mandate that (a) a real per-FILE
   shortfall still REDs, and (b) an in-scope changed PATH with no record that is NOT
   pack-declared-excluded is a LOUD blocking error, never a silent pass (a declared-excluded
   path is skipped). This is the load-bearing anti-vacuous-green invariant; do not let
   "no coverage data" collapse into green.

2. **An unlisted CheckType consumer is a dropped gate step.** The wholesale cutover
   (BUNDLE-011 REQ-001 / SPEC-040) reroutes Step 2; any site keying on lint/build/test/
   findings identity that the catalog misses can be silently stranded — its behavior
   vanishes with no compile error. REQ-006 + CLM-023/CLM-024 mandate a guard test that
   FAILS if a gate-semantic `CheckType`-keyed site exists with no catalog entry (and if a
   catalog entry's site no longer exists). Completeness cannot rest on manual review. The
   subtle trap: the live `pkg/check` `.Pass`/`CheckType` surface is LARGER than the obvious
   sites (output.go alone has five `.Pass.String()` render sites; check.go/registry.go have
   many dispatch helpers), so an exhaustive `file:symbol` catalog would red on every added
   log line. This spec scopes the catalog+guard to GATE-SEMANTIC consumers (scope-filter,
   dispatch, verdict) and explicitly EXCLUDES cosmetic display/serialization `.Pass.String()`
   sites — that bounded scope is what makes discovered-set provably equal cataloged-set.

3. **Hard dependency on SPEC-040's toolchain test runner, at per-FILE granularity.**
   Coverage is dynamic-toolchain: it needs the per-FILE coverage signal only the toolchain
   test runner produces. It was historically descoped (SPEC-038 REQ-009 kept it baked) for
   exactly this reason. This spec CANNOT land before SPEC-040's toolchain test pass
   exists, AND that pass must emit coverage at per-file/path granularity (the
   producer↔consumer contract): if SPEC-040 surfaces only a coarser (e.g. package-rollup)
   signal, the per-file enforcement here cannot hold — this is flagged for cross-spec
   coherence review. Re-baking a `go test` runner to unblock it would re-introduce the
   very baked coupling REQ-002 eradicates.

4. **The shared-runner eradication must not silently regress code_check's test signal.**
   `newSharedTestRunner` feeds BOTH coverage AND code_check's test FAILs from one exec
   (a ~94s dedup). Deleting it (REQ-002) must not leave code_check without a test
   signal. Code_check's test pass is owned by SPEC-040's toolchain test pass; this spec
   must verify (CLM-003) that after eradication, coverage consumes the declared test
   pass and does NOT reconstruct a binary-resident whole-module `go test` under another
   name — that would be REQ-002 violated in disguise.

5. **The exemption locus is the ENGINE PATH, not the soon-dead legacy setter.** The only
   code setting `Violation.ProjectWide` today (`cv.Pass == check.CheckTypeBuild`,
   gate.go:1173 in `checkViolationsToGate`) is reachable ONLY via `realCodeChecker`, which
   SPEC-040 deletes. Editing it, or writing a "no `== CheckTypeBuild` remains" assertion
   against it, passes TRIVIALLY against an orphan and ships ZERO real behavior — the
   engine path would still never set `ProjectWide`, so a build break in an unchanged file
   would be silently scope-filtered away (the under-broad failure mode). REQ-004 forces the
   work onto `pack_gate.go` (the `exempt_from_scope_filter → ProjectWide` bridge that does
   not exist today) and CLM-013 guards it END-TO-END through `filterViolations`: an
   unchanged-file build break must still RED a diff-scoped gate via the engine path.

6. **The declared exempt must not over- or under-exempt.** Post-bridge, go-build is
   exempt; golangci/go-test/findings are not. An over-broad property marking lint or test
   exempt would leak out-of-scope violations into a diff-scoped gate; an under-broad one
   dropping build's exemption would let real build breakage be scope-filtered away. The
   matrix claims (CLM-013…CLM-016) pin all four through `filterViolations`, not just at the
   stamp site. Note the live ScopeKind contradiction this resolves: golangci/go-build/
   go-test are ALL `ScopeKindProjectWide` on `main` (binding.go:274/290/299) for
   arg-shaping — `exempt_from_scope_filter` is DECOUPLED so ScopeKind stays project-wide
   for all three while only go-build is filter-exempt (CLM-017).

7. **Multiple build passes with differing exempt values must resolve per-violation.** Two
   toolchain bindings (e.g. across packages or stacks) can both produce "build" violations
   with DIFFERING `exempt_from_scope_filter` values. Aggregating to a gate-type level
   re-introduces a "which value wins" ambiguity; per-violation resolution (each violation
   carries its producing binding's value) dissolves it (REQ-007/CLM-018). The only true
   conflict — identical file+line+rule from two sources with differing values — resolves
   to the exempting (louder) value (CLM-019), because under-broad filtering is the guarded
   failure mode.

8. **Per-changed-FILE, never aggregated, is a load-bearing invariant — and "package" is
   itself a baked-language trap.** The eradicated analyzer keyed coverage on Go "package",
   which both re-bakes a Go-native concept (every other stack reports per-file) and lets a
   low-coverage file hide inside an over-floor package aggregate. The re-implementation
   enforces per-changed-FILE: an over-floor changed file passes regardless of siblings, and
   an under-floor changed file fails even when its directory/package siblings are
   high-coverage (the 2%-file-in-a-95%-directory case a package aggregate would mask). The
   model carries NO "package" noun; folding per-file records into ANY directory/package
   roll-up silently breaks both the language-agnosticism and the non-vacuousness guarantee.

## Review Questions

These probe risks not fully pinned by claims; the impl-reviewer should verify each.

1. After the change, does `pkg/gate/step_coverage.go` (or its successor) contain ANY
   `go test` invocation, `coverage:` regex, or go.mod/Go-package parsing? It must not —
   grep the package for `exec`, `go.mod`, and `coverage:` literals. Any survivor is a
   REQ-001 violation.

2. Is there ANY binary-resident whole-module `go test ./...` runner left under any name
   after `sharedTestRunner` is deleted? Confirm coverage's coverage signal originates
   from the declared toolchain test pass (SPEC-040), not from an in-binary exec — a
   renamed shared runner is REQ-002 violated in disguise.

3. Is the build-exemption built on the ENGINE PATH (`pack_gate.go` stamps
   `Violation.ProjectWide` from `binding.exempt_from_scope_filter`), and NOT merely an
   edit/assertion against `checkViolationsToGate`/gate.go:1173? The latter is orphaned by
   SPEC-040 and passes trivially. Confirm the bridge is exercised END-TO-END through
   `filterViolations` (a go-build break in an UNCHANGED file still REDs a diff-scoped gate;
   lint/test/findings violations in unchanged files ARE filtered) — not just asserted at
   the stamp site. Confirm `ScopeKind` is unchanged (still project-wide for all three Go
   engines) and decoupled from the exempt decision.

4. Is `exempt_from_scope_filter` resolved PER-VIOLATION (each violation carries its
   producing binding's value), with NO gate-type-level aggregation, and does the
   true-conflict tiebreak favor the exempting (louder) value? Inject two bindings with
   differing values and a same-file+line+rule collision to confirm.

5. Does the catalog-completeness guard actually scan the real consumer surface and FAIL
   on an injected unlisted keying site? Confirm the guard is not a tautology (it must fail
   if a `CheckType`-keyed line is added without a catalog entry), that it fails on a stale
   catalog entry whose site was deleted, and that the catalog is exhaustive over the
   GATE-SEMANTIC surface (C-1…C-8, excluding cosmetic `.Pass.String()` display sites) so
   it does not red on arrival.

6. Does a real per-FILE coverage shortfall still produce a blocking `coverage_threshold`
   violation — including a low-coverage file hidden in an otherwise-high-coverage directory
   (which per-file catches and a package aggregate would not) — and does an in-scope changed
   PATH with no record that is NOT pack-declared-excluded produce a LOUD error rather than a
   silent pass (while a declared-excluded path is skipped)? Confirm the model carries no
   "package" granularity. These are the two halves of the non-vacuousness invariant; both
   must be exercised against the declared per-file record, not a stubbed/mocked path that
   can't go vacuous.

## References

- BUNDLE-011 (collapse-legacy-codecheck-into-packs) — Seed 3, RDQ-6, REQ-011/012/013.
  This spec's parent bundle.
- SPEC-040 (toolchain-pack-cutover) — Seed 2, the **hard prerequisite**: establishes the
  declared `go-toolchain` test pass this spec consumes. Coverage re-implementation reads
  that pass's PER-FILE/path coverage signal (producer↔consumer contract: the pass must
  emit per-file records — flagged for cross-spec coherence review, not authored here).
- SPEC-042 (coverage-production-engine) — the **producer + canonical type owner**: defines
  `check.CoverageRecord{Path, Covered, Total, Measured, Excluded, Metric}` (REQ-003), the
  SINGLE shared type this spec consumes. The gate here computes `Covered/Total >= threshold`
  (metric-blind, `Total==0`⇒N/A); SPEC-042's engine emits the records. Both draft — must
  agree on the one shared type (no second `CoverageRecord` shape).
- SPEC-039 (codecheck-deadcode-prelude) — Seed 1, the behavior-preserving dead-code
  deletions (standards-manifest reader, non-Go semgrep catch-all). Sibling, not a dep.
- SPEC-038 (REQ-009) — DESCOPED coverage and KEPT `pkg/gate/step_coverage.go` baked
  (enforced by a passing test). Corrects the stale "BUNDLE-009 deletes coverage" claim;
  this spec is where that baked coverage is finally eradicated.
- SPEC-035 (pack-declared-engines-trusted-allowlist) — LANDED. The tool-neutral
  gate-TYPE direction REQ-004's declared-property exemption is consistent with.
- Code (verified on `main` 2026-06-24): `pkg/gate/step_coverage.go` (baked Go coverage
  analyzer, KEPT by SPEC-038 REQ-009 — eradicated here); `cmd/backstop/shared_testrun.go`
  (`newSharedTestRunner` — eradicated here); `cmd/backstop/gate.go:~487`
  (`newSharedTestRunner` construction + `sharedTest` feeds), :1158/:1173
  (`checkViolationsToGate` — `rule = cv.Pass.String()`, `ProjectWide: cv.Pass ==
  check.CheckTypeBuild` — the SOLE `ProjectWide` setter today, but reachable ONLY via
  `realCodeChecker`, so ORPHANED/DELETED once SPEC-040 removes that caller; behavior
  replaced on the engine path, NOT edited in place); `cmd/backstop/pack_gate.go:411` (engine
  dispatch — uses `binding.ScopeKind` for arg-shaping ONLY, NEVER sets `Violation.ProjectWide`
  today — REQ-004 builds the `exempt_from_scope_filter → ProjectWide` bridge here);
  `pkg/pack/engine/binding.go:274/290/299` (golangci/go-build/go-test ALL declared
  `ScopeKindProjectWide` for arg-shaping — `exempt_from_scope_filter` is decoupled and added
  here); `pkg/gate/scope.go:194` (`filterViolations` — keys on `Violation.ProjectWide`, the
  unchanged consumer the bridge feeds); `cmd/backstop/code_check.go:187,229` (CheckTypeFindings
  stamping); `pkg/check/check.go:14` (passOrder), :44/:59 (`Violation.Pass`/`PassResult.Pass`),
  :108 (`Executors map[CheckType]`/`applicableChecks`); `pkg/check/output.go:131` (`v.Pass`
  display keying); `pkg/check/manifest.go` (CheckType enum + routing); `pkg/check/registry.go`
  (toolchain Entries keyed by CheckType); `pkg/check/parsers.go:50` (CheckTypeFindings
  stamping).

## Version History

- **1.2.0** (2026-07-06) — Marked the CheckType-consumer catalog claims `kind: absence` (the
  per-claim annotation added by ISSUE-035) to reflect their structural/catalog-scan nature and
  clear the `test_substantiveness` gate's noTarget ("does not call package gate") false-flag on
  their mandated tests. Annotated **CLM-020** and **CLM-021** (enumerate/verify the
  `CheckTypeConsumerCatalog()` structure — catalog-content invariants), and **CLM-022**,
  **CLM-023**, **CLM-024** (the completeness guard — repo-source keying-site scans reconciled
  against the catalog: clean-on-arrival, fails-on-unlisted-site, fails-on-stale-entry). Every
  mandated test scans the catalog/source surface for structural congruence and by design does
  not exercise `pkg/gate` (the spec's coverage-re-impl package) — exactly the catalog-scan
  guard case the annotation exists for. No coverage-re-implementation claim (REQ-001…REQ-004,
  REQ-007) was annotated; those tests genuinely exercise the gate coverage/exemption logic.
  Annotation-only change (align-predating-artifacts); no requirement, claim text, test, or
  contract altered.
- **1.1.0** (2026-07-03) — Status → `implemented`. BUNDLE-011 Seed 3 code (coverage consumer
  re-implementation + CheckType-consumer catalog) shipped and committed; parent bundle
  BUNDLE-011 delivered. Status-only transition, no requirement, claim, contract, or prose change.
