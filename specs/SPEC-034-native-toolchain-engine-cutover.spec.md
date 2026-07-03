---
title: "Native Toolchain Engine Cutover"
number: SPEC-034
created: "2026-06-18"
status: replaced
replaced-by: BUNDLE-011
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    ⚠️ TOMBSTONE — EFFECTIVELY SUPERSEDED (verified against `main` @ 2026-06-24). DO NOT
    pick this spec up as live, plannable work despite its `draft` status. SPEC-034's
    cutover was implemented on branch `feat/bundle-010-impl`, but ONLY its BRIDGE half
    reached `main`: `loadBridgedToolchainPacks` (defined cmd/backstop/gate.go:411,
    invoked at gate.go:464) routes a native `<lang>-toolchain` pack through
    `dispatchPackEngines`. The DELETION half did NOT reach `main` — on `main`,
    `realCodeChecker` → `pkg/check.Run` is STILL the live gate Step 2
    (cmd/backstop/gate.go:481 comment, :488 wiring; the bespoke executors are still
    constructed via `buildExecutorsForConfigErr` at pkg/check/check.go:293), and
    `builtinToolchain` (pkg/check/registry.go:59) is STILL the live native-toolchain
    source. (Note: the old `goBuiltinExecutors` name is NOT present under that name on
    `main`; `builtinToolchain` is the current symbol.) The remaining cutover + deletion
    scope has been ABSORBED BY BUNDLE-011 (Seed 2 → SPEC-040; recorded in BUNDLE-011
    REQ-008 / DD-4 / RDQ-4), which will spec and implement the work from there,
    generalized beyond Go to any `<lang>-toolchain` pack. This spec is now
    `status: replaced`, `replaced-by: BUNDLE-011` — the terminal-state vocabulary
    ratified by ISSUE-031 (artifact terminal states) made the correct terminal state
    `replaced` (with a typed `replaced-by` ref), NOT `superseded` (which is an
    ADR-only concept and never became a spec state). The original intent and phased
    design below are retained verbatim for traceability. ⚠️ ——
    Bridge the native Go CODE-CHECK toolchain (the lint, build, and test passes) onto
    the EXISTING engine-dispatch substrate, then DELETE the bespoke toolchain path.
    Today the gate runs two disjoint substrates at the cmd/backstop level:
    `realCodeChecker` → `check.Run` (the bespoke Go toolchain — `goBuiltinExecutors`
    behind an `if language == "go"` short-circuit in pkg/check/registry.go, with
    hand-written `lintExecutor`/`buildExecutor`/`testExecutor` and hand-written
    `parseGolangciJSON`/`parseGoBuildErrors`/`parseGoTestFailures` parsers in
    pkg/check/check.go) AND `dispatchPackEngines` (cmd/backstop/pack_gate.go — the
    declared engine model SPEC-031 built). `pkg/check` does NOT import
    `pkg/pack/engine`; the two substrates are disjoint. The CENTRAL, load-bearing work
    of this spec (REQ-001, phase 1) is the BRIDGE: route the native lint/build/test
    passes through the SAME `dispatchPackEngines` substrate the packs already use, as
    Layer-0 engine passes declared in a reusable `go-toolchain` pack — REUSING the
    dispatch substrate, NOT inventing a parallel one and NOT adding a
    `pkg/check`→`pkg/pack/engine` import. On top of that bridge: (REQ-004) build/test
    declare a findings engine whose pack-relative `convert` script (run via the
    sandboxed clean-stdout capture `SandboxedRunStdout`) re-expresses the retired
    `parseGoBuildErrors`/`parseGoTestFailures` normalization OUTSIDE the core binary
    (DD-2: removed, not extracted), preserving the crash-vs-findings guard; (REQ-005)
    lint runs golangci-lint v2 as a `config-file` engine emitting SARIF natively on
    stdout, captured via `RunStdout` (no converter, no version-adaptive flags);
    (REQ-006) semgrep stays the shared engine, unchanged but for provisioning;
    (REQ-008) provisioning splits — `go`/`golangci-lint` are Layer-0 assume-present and
    fail loud with a ConfigError (exit 2) when absent, semgrep stays pinned, and
    `EnsureSemgrep`'s bespoke install retires. The cutover is a STRANGLER expressed as
    ORDERED, TESTABLE phases: wire the bridge (phase 1) → prove invocation+parser
    equivalence against the bespoke path on captured fixtures, INCLUDING the
    `code check --file` hook's go-test file-mode package scoping (phase 2) → and ONLY
    in a phase GATED on that equivalence test existing-and-passing, delete the bespoke
    executors/parsers/short-circuit/install ladder and migrate-or-delete the ~7
    bespoke-asserting test files (phase 3). End state: backstop-core carries ZERO
    baked-in Go toolchain knowledge; "support a stack" becomes "author a
    stack-toolchain pack," no core changes. This spec is the native code-check half
    ONLY; the traceability steps (coverage / substantiveness / contract signatures)
    are BUNDLE-009's separate migration onto ast-grep packs and are an explicit
    NON-GOAL here.
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/ ./pkg/check/ ./pkg/pack/engine/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      THE BRIDGE (load-bearing, phase 1). The native Go code-check toolchain (the lint,
      build, and test passes) must run through the EXISTING engine-dispatch substrate —
      `dispatchPackEngines` (cmd/backstop/pack_gate.go) — the SAME path the packs
      already use, as Layer-0 engine passes declared in the `go-toolchain` pack
      (REQ-007). The implementation must REUSE that substrate: it must NOT invent a
      parallel engine dispatcher, and it must NOT add a `pkg/check` →
      `pkg/pack/engine` import (the two are disjoint today and stay disjoint;
      orchestration of both lives at the cmd/backstop level). Concretely, the gate's
      Step 2 (`realCodeChecker` → `check.Run`) and the pack engine step
      (`dispatchPackEngines`) converge so the native lint/build/test passes are
      dispatched as engine bindings, not as bespoke `pkg/check` `PassExecutor`s. Every
      other requirement in this spec sits ON TOP of this bridge.
    supports: pluggable-pack-engines:REQ-018
    follows: STD-GO-001:GO-010

  - id: REQ-002
    text: >
      The bespoke native toolchain construction path must be removed once the bridge
      (REQ-001) carries lint/build/test. The `if language == "go" { return
      goBuiltinExecutors(...) }` short-circuit in `buildExecutorsForConfigErr`
      (pkg/check/registry.go) and the `goBuiltinExecutors` constructor (pkg/check/check.go)
      that routes Go to the four bespoke executors must be deleted, and the gate's
      `realCodeChecker` → `check.Run` toolchain step must no longer run the native
      lint/build/test passes (they now run through `dispatchPackEngines`). After
      removal there is no language-special-cased native-executor construction; the
      bespoke `lintExecutor`, `buildExecutor`, and `testExecutor` types are deleted.
    supports: pluggable-pack-engines:REQ-018
    follows: STD-GO-001:GO-010

  - id: REQ-003
    text: >
      The build and test parser LOGIC must be relocated OUT of the core binary INTO the
      `go-toolchain` pack's converters (DD-2: removed, not extracted). The bespoke
      parsers `parseGoBuildErrors` and `parseGoTestFailures` (pkg/check/check.go) — and
      the `go-build` / `go-test` named formats in `formatParsers` that wrap them
      (pkg/check/parsers.go) — must be deleted from pkg/check; their normalization
      behavior (compiler-error and test-failure `file:line:message` extraction) is
      re-expressed as a pack-relative `convert` script that runs OUTSIDE the core binary
      on the build/test engine's stdout. The build/test crash-vs-findings guard (a
      non-zero run with no parseable findings is a tool crash, not a finding-free pass)
      must be preserved on the new convert path so a build/test crash never reads as a
      silent green.
    supports: pluggable-pack-engines:REQ-018
    follows: STD-GO-001:GO-010

  - id: REQ-004
    text: >
      Build and test must be declared as findings engines in the `go-toolchain` pack
      whose pack-relative `convert` script normalizes the tool's output to SARIF, run
      through `dispatchPackEngines`' findings path (`runFindingsEngine`). The `convert`
      executable must run via the SANDBOXED clean-stdout capture `SandboxedRunStdout`
      (pkg/packval/sandbox.go) — the same sandbox trust model and clean-stdout discipline
      SPEC-031's convert step uses — so a converter's stderr banner cannot corrupt the
      SARIF on stdout. Backstop embeds no Go build/test parser; the transform is
      pack-author code. The macOS-only limitation of `SandboxedRun`/`SandboxedRunStdout`
      (it wraps `sandbox-exec`) is inherited here and is addressed explicitly in Sharp
      Edges as a documented limitation with a follow-up, not silently assumed away.
    supports: pluggable-pack-engines:REQ-018
    follows: STD-GO-001:GO-010

  - id: REQ-005
    text: >
      The lint pass must run golangci-lint through the engine model as a `config-file`
      engine entry. golangci-lint v2 emits SARIF natively to STDOUT, so the lint pass
      declares NO converter and its output parses directly through `parseSarif` — but
      the lint engine output MUST be captured via the clean-stdout runner (`RunStdout`),
      NOT `CombinedOutput`, because golangci-lint writes progress/warnings to stderr and
      a combined capture would corrupt the SARIF (the same bug class as semgrep's
      `--sarif` capture). The bespoke `lintExecutor` (pkg/check/check.go), the
      `parseGolangciJSON` parser, the `golangci-json` named format in `formatParsers`,
      and the version-adaptive flag logic (`golangciOutputArgs`, `golangciMajorVersion`,
      `golangciVersionRe`) must all be deleted from pkg/check. The lint invocation pins/
      assumes golangci-lint v2 SARIF output (no v1/v2 branch, no `golangci-lint version`
      probe). This is wiring, not a pack converter script.
    supports: pluggable-pack-engines:REQ-018
    follows: STD-GO-001:GO-010

  - id: REQ-006
    text: >
      The semgrep pass must remain the shared engine SPEC-031 already established and
      must NOT be re-implemented or re-wired by this spec. semgrep is already an engine
      (a shared `semgrepExecutor` invoked with no pack `--config` feeder, plus the
      pack-side group-by-engine dispatch in SPEC-031); this spec's only contact with
      semgrep is retiring `EnsureSemgrep`'s bespoke install logic into the declared
      provisioning model (REQ-008). The semgrep findings path and its SARIF output
      contract are out of scope to change here.
    supports: pluggable-pack-engines:REQ-019
    follows: STD-GO-001:GO-010

  - id: REQ-007
    text: >
      The build/test `convert` scripts (run via the sandbox mechanism) must live in a
      REUSABLE `go-toolchain` pack that is a SEPARATE artifact from the opinionated
      `backstop-go-pack` (which ships coding-standards semgrep rules). The decomposition
      is mechanism vs opinion: the `go-toolchain` pack is mechanism (run the Go
      toolchain, normalize its output to SARIF — identical for every Go project) and the
      `backstop-go-pack` is opinion (swappable coding-standards rules). The
      `go-toolchain` pack must NOT contain coding-standards rules, and `backstop-go-pack`
      must NOT contain build/test toolchain mechanism. The `go-toolchain` pack is
      authored alongside this spec and lands in LOCKSTEP with the core changes, mirroring
      BUNDLE-010 pillar-1's "core reader + pack repo flip together" pattern: the bridge
      (REQ-001) and the pack must not land in separate, independently-green commits that
      leave a window with no Go build/test enforcement.
    supports: pluggable-pack-engines:REQ-018

  - id: REQ-008
    text: >
      Engine provisioning must split by who owns the tool (REQ-019). The Layer-0 native
      toolchain — `go` and `golangci-lint` — is ASSUME-PRESENT: the project owns its own
      compiler and linter, so backstop must NOT install them, and a missing `go` or
      `golangci-lint` binary must fail loud with a `ConfigError` (exit 2) naming the
      tool, never a silent skip and never an auto-install attempt. (Neither the bespoke
      path nor the bare engine path emits this ConfigError today — the absent-tool
      fail-loud is NEW behavior this spec adds, not an inherited guarantee.)
      Backstop-INTRODUCED engines (semgrep) remain pinned and auto-provisioned.
      `EnsureSemgrep`'s bespoke install logic (pkg/check/semgrep.go) must retire into the
      declared provisioning model — the ad hoc PATH-probe / `.backstop/tools` /
      `pip install` ladder is removed from the native Run path and replaced by the
      declared, data-driven provisioning the engine model carries; no per-tool Go
      install ladder remains baked into pkg/check for the native passes.
    supports: pluggable-pack-engines:REQ-019
    follows: STD-GO-001:GO-010

  - id: REQ-009
    text: >
      The cutover must be a STRANGLER expressed as ORDERED, TESTABLE phases — never a
      big-bang — and must never leave a window where build/test/lint enforcement lapses
      (vacuous green is the cardinal sin). Phase 1: stand up the `go-toolchain` pack +
      `convert` scripts and bridge lint/build/test through `dispatchPackEngines`
      (REQ-001). Phase 2: the EQUIVALENCE GATE — a mandated test that proves the engine
      path emits the SAME normalized violations as the bespoke path for the same captured
      tool output (compiler errors, test failures, lint findings), AND that the
      invocation behaviors are preserved, including the `code check --file` hook's
      go-test file-mode package scoping (REQ-010). Phase 3 (DELETION) may delete the
      bespoke executors/parsers/short-circuit/`EnsureSemgrep` install ladder and migrate-
      or-delete the bespoke-asserting test files ONLY when the phase-2 equivalence test
      EXISTS AND PASSES. "Deletion is gated on proven equivalence" must itself be
      checkable: a guard test asserts the equivalence test is present and green and that
      the bespoke symbols are absent — so a deletion that outran the proof fails the gate
      rather than shipping green.
    supports: pluggable-pack-engines:REQ-018
    follows: STD-GO-001:GO-010

  - id: REQ-010
    text: >
      Invocation behaviors, not just parser output, must be preserved across the bridge.
      In particular the `code check --file` standalone-hook path scopes `go test` to the
      single changed file's PACKAGE (`goPackageSelector` / `testExecutor.fileMode`,
      pkg/check/check.go) under the hook's tight time budget, rather than running the
      whole module. The new engine path must EITHER preserve this file-mode package
      scoping for the test pass OR make its removal an EXPLICIT, recorded decision with
      rationale (not a silent regression). The chosen outcome must be covered by a
      mandated test; if preserved, the test asserts the file-mode test invocation targets
      the changed file's package, not `./...`.
    supports: pluggable-pack-engines:REQ-018
    follows: STD-GO-001:GO-010

  - id: REQ-011
    text: >
      The end state must carry ZERO baked-in Go toolchain knowledge or native parsers in
      backstop-core, and the ~7 bespoke-asserting test files must be migrated or deleted
      as part of phase 3 (not left dangling against removed symbols). After this spec,
      "support a stack" means "author a stack-toolchain pack," with no core changes: no
      Go-specific `PassExecutor` type, no Go build/test/lint parser function, and no
      `language == "go"` short-circuit remaining in pkg/check non-test source. The
      bespoke-asserting test files enumerated in the contracts (`check_test.go`,
      `engine_semantics_test.go`, `executor_test.go`, `registry_coverage_test.go`,
      `registry_test.go`, `ts_executor_test.go`, `ts_routing_test.go`) must each be
      either migrated to assert the engine path or deleted with their bespoke target; no
      test may reference a deleted bespoke symbol. A grep of pkg/check non-test source
      for `goBuiltinExecutors`, `lintExecutor`, `buildExecutor`, `testExecutor`,
      `parseGoBuildErrors`, `parseGoTestFailures`, `parseGolangciJSON`, and the
      `language == "go"` short-circuit must return zero matches.
    supports: pluggable-pack-engines:REQ-018
    follows: STD-GO-001:GO-010

claims:
  # REQ-001 — THE BRIDGE: native toolchain runs through dispatchPackEngines
  - id: CLM-001
    requirement: REQ-001
    text: The native lint/build/test passes are dispatched through the existing dispatchPackEngines substrate as engine bindings, not through bespoke pkg/check PassExecutors
    tests:
      - TestBridge_NativePassesRunThroughDispatchPackEngines
  - id: CLM-002
    requirement: REQ-001
    text: pkg/check does not import pkg/pack/engine after the bridge; the two substrates stay disjoint and orchestration lives at cmd/backstop
    tests:
      - TestBridge_NoCheckToEngineImport
  - id: CLM-003
    requirement: REQ-001
    text: The bridge reuses dispatchPackEngines and introduces no parallel engine dispatcher for the native passes
    tests:
      - TestBridge_NoParallelDispatcher

  # REQ-002 — bespoke construction path removed once bridge carries the passes
  - id: CLM-004
    requirement: REQ-002
    text: The if language==go short-circuit and goBuiltinExecutors are deleted from registry.go/check.go
    tests:
      - TestCutover_GoShortCircuitRemoved
  - id: CLM-005
    requirement: REQ-002
    text: The gate's realCodeChecker/check.Run step no longer runs the native lint/build/test passes
    tests:
      - TestCutover_CheckRunNoLongerRunsToolchainPasses
  - id: CLM-006
    requirement: REQ-002
    text: The bespoke lintExecutor, buildExecutor, and testExecutor types are deleted
    tests:
      - TestCutover_BespokeExecutorTypesDeleted

  # REQ-003 — build/test parser logic moves OUT to the pack convert script
  - id: CLM-007
    requirement: REQ-003
    text: parseGoBuildErrors and parseGoTestFailures are deleted from pkg/check and their go-build/go-test named formats are removed from formatParsers
    tests:
      - TestCutover_GoBuildTestFormatsRemoved
  - id: CLM-008
    requirement: REQ-003
    text: The go-toolchain pack convert script normalizes real go build compiler-error output into the expected file:line:message findings (the normalization the retired parseGoBuildErrors established and the transitional equivalence proof confirmed before the bespoke parser was deleted); the standalone convert test now asserts those exact findings directly
    tests:
      - TestGoToolchainConvert_BuildToSarif_DirectFindings
  - id: CLM-009
    requirement: REQ-003
    text: The go-toolchain pack convert script normalizes real go test failure output (including the no-position FAIL block) into the expected file:line:message findings (the normalization the retired parseGoTestFailures established and the transitional equivalence proof confirmed before the bespoke parser was deleted); the standalone convert test now asserts those exact findings directly
    tests:
      - TestGoToolchainConvert_TestToSarif_DirectFindings
  - id: CLM-010
    requirement: REQ-003
    text: A build/test run that exits non-zero with no parseable findings surfaces as a crash on the convert path, not a silent green
    tests:
      - TestGoToolchain_BuildTestCrashNotSilentGreen

  # REQ-004 — build/test as findings engine + sandboxed clean-stdout convert
  - id: CLM-011
    requirement: REQ-004
    text: The build pass is a go-toolchain findings engine whose pack-relative convert script normalizes output to SARIF, run through runFindingsEngine
    tests:
      - TestGoToolchain_BuildFindingsEngineWithConvert
  - id: CLM-012
    requirement: REQ-004
    text: The test pass is a go-toolchain findings engine whose pack-relative convert script normalizes output to SARIF, run through runFindingsEngine
    tests:
      - TestGoToolchain_TestFindingsEngineWithConvert
  - id: CLM-013
    requirement: REQ-004
    text: The build/test convert script runs via the sandboxed clean-stdout capture SandboxedRunStdout, so a converter stderr banner cannot corrupt the SARIF
    tests:
      - TestGoToolchain_ConvertUsesSandboxedRunStdout
  - id: CLM-014
    requirement: REQ-004
    text: backstop-core embeds no Go build or test parser; the transform resolves from the pack convert script
    tests:
      - TestGoToolchain_NoEmbeddedBuildTestParser

  # REQ-005 — lint as config-file engine, native SARIF on stdout via RunStdout
  - id: CLM-015
    requirement: REQ-005
    text: The Go lint pass runs golangci-lint as a config-file engine whose SARIF output parses directly through parseSarif with no converter
    tests:
      - TestGoLint_ConfigFileEngineNativeSarif
  - id: CLM-016
    requirement: REQ-005
    text: The lint engine output is captured via RunStdout (clean stdout), not CombinedOutput, so a golangci-lint stderr banner cannot corrupt the SARIF
    tests:
      - TestGoLint_SarifCapturedViaRunStdoutNotCombined
  - id: CLM-017
    requirement: REQ-005
    text: golangci-lint v2 native SARIF parses into the expected located lint violations with the correct SARIF level->severity mapping (error/warning) — the normalization the retired parseGolangciJSON established and the transitional equivalence proof confirmed before the bespoke parser was deleted; the standalone parse test now asserts those exact findings and severities directly
    tests:
      - TestParsePackFindings_GolangciV2Sarif
  - id: CLM-018
    requirement: REQ-005
    text: lintExecutor, parseGolangciJSON, the golangci-json named format, and the version-adaptive flag logic are deleted from pkg/check
    tests:
      - TestCutover_BespokeLintPathRemoved
  - id: CLM-019
    requirement: REQ-005
    text: The lint invocation assumes golangci-lint v2 SARIF and performs no version probe or v1/v2 flag branch
    tests:
      - TestGoLint_NoVersionProbeOrV1Branch

  # REQ-006 — semgrep unchanged
  - id: CLM-020
    requirement: REQ-006
    text: The semgrep pass is not re-implemented or re-wired by this spec and retains its SPEC-031 shared-engine shape and SARIF contract
    tests:
      - TestSemgrep_UnchangedSharedEngine
  - id: CLM-021
    requirement: REQ-006
    text: This spec's only semgrep-adjacent change is retiring EnsureSemgrep's bespoke install into declared provisioning, not altering the findings path
    tests:
      - TestSemgrep_OnlyProvisioningChanges

  # REQ-007 — go-toolchain pack separate from backstop-go-pack, lockstep
  - id: CLM-022
    requirement: REQ-007
    text: The build/test convert scripts (run via the sandbox mechanism) live in a reusable go-toolchain pack distinct from the opinionated backstop-go-pack
    tests:
      - TestGoToolchainPack_SeparateFromGoStandardsPack
  - id: CLM-023
    requirement: REQ-007
    text: The go-toolchain pack contains only toolchain mechanism (run + normalize) and no coding-standards rules
    tests:
      - TestGoToolchainPack_MechanismOnlyNoStandards
  - id: CLM-024
    requirement: REQ-007
    text: backstop-go-pack contains only coding-standards opinion and no build/test toolchain mechanism
    tests:
      - TestGoStandardsPack_OpinionOnlyNoToolchain
  - id: CLM-025
    requirement: REQ-007
    text: The bridge and the go-toolchain pack are wired to land in lockstep so no commit leaves Go build/test unenforced
    tests:
      - TestGoToolchainPack_LandsInLockstepWithBridge

  # REQ-008 — split provisioning + NEW absent-tool fail-loud, EnsureSemgrep retires
  - id: CLM-026
    requirement: REQ-008
    text: A missing go binary fails loud with a ConfigError (exit 2) naming the tool and is never auto-installed
    tests:
      - TestProvision_GoAssumedPresentFailsLoud
  - id: CLM-027
    requirement: REQ-008
    text: A missing golangci-lint binary fails loud with a ConfigError (exit 2) naming the tool and is never auto-installed
    tests:
      - TestProvision_GolangciAssumedPresentFailsLoud
  - id: CLM-028
    requirement: REQ-008
    text: A backstop-introduced engine (semgrep) remains pinned and auto-provisioned, distinct from the assume-present native toolchain
    tests:
      - TestProvision_SemgrepStillPinnedAndProvisioned
  - id: CLM-029
    requirement: REQ-008
    text: EnsureSemgrep's bespoke PATH/tools/pip install ladder is removed from the native Run path and replaced by declared provisioning
    tests:
      - TestProvision_EnsureSemgrepBespokeInstallRetired

  # REQ-009 — phased strangler: equivalence gate + deletion gated on proven equivalence
  - id: CLM-030
    requirement: REQ-009
    text: For real go build output, the engine path (dispatchPackEngines over the go-toolchain build convert) emits the expected normalized build violations. The transitional phase-2 equivalence gate proved this engine path matched the bespoke parser and licensed its deletion; the bespoke parser is now gone, so the standalone engine-dispatch test is the ongoing verification of the correct findings
    tests:
      - TestGoToolchain_BuildFindingsEngineWithConvert
  - id: CLM-031
    requirement: REQ-009
    text: For real go test output, the engine path (dispatchPackEngines over the go-toolchain test convert) emits the expected normalized test violations. The transitional phase-2 equivalence gate proved this engine path matched the bespoke parser and licensed its deletion; the bespoke parser is now gone, so the standalone engine-dispatch test is the ongoing verification of the correct findings
    tests:
      - TestGoToolchain_TestFindingsEngineWithConvert
  - id: CLM-032
    requirement: REQ-009
    text: For golangci-lint findings, the engine path (golangci v2 SARIF through the config-file engine) emits the expected normalized lint violations. The transitional phase-2 equivalence gate proved this engine path matched the bespoke parseGolangciJSON and licensed its deletion; the bespoke parser is now gone, so the standalone config-file-engine test is the ongoing verification of the correct findings
    tests:
      - TestGoLint_ConfigFileEngineNativeSarif
  - id: CLM-033
    requirement: REQ-009
    text: Deletion of the bespoke path is gated on the equivalence test existing and passing — a guard test asserts equivalence is proven (and green) and the bespoke symbols are absent, so a deletion that outran the proof fails the gate
    tests:
      - TestStrangler_DeletionGatedOnProvenEquivalence

  # REQ-010 — invocation behavior preserved: file-mode go-test package scoping
  - id: CLM-034
    requirement: REQ-010
    text: The code check --file hook path scopes the test pass to the changed file's package (not ./...) through the new engine path, preserving goPackageSelector file-mode scoping
    tests:
      - TestFileMode_TestPassScopedToChangedFilePackage
  - id: CLM-035
    requirement: REQ-010
    text: The file-mode test-scoping outcome is an explicit, tested decision — a whole-module run in file mode is rejected as a regression unless the decision to drop scoping is recorded
    tests:
      - TestFileMode_NoSilentWholeModuleRegression

  # REQ-011 — zero baked-in knowledge + bespoke-asserting test files migrated/deleted
  - id: CLM-036
    requirement: REQ-011
    text: A grep of pkg/check non-test source for goBuiltinExecutors, the three bespoke executors, the three bespoke parsers, and the language==go short-circuit returns zero matches
    tests:
      - TestEndState_NoBakedGoToolchainKnowledge
  - id: CLM-037
    requirement: REQ-011
    text: No test file references a deleted bespoke symbol — the seven enumerated bespoke-asserting test files are each migrated to the engine path or deleted with their target
    tests:
      - TestEndState_NoTestReferencesDeletedBespokeSymbol
  - id: CLM-038
    requirement: REQ-011
    text: Adding a new stack requires no pkg/check change — a declared stack-toolchain pack suffices
    tests:
      - TestEndState_NewStackNeedsNoCoreChange

contracts:
  # THE BRIDGE lives at cmd/backstop, where both substrates are already orchestrated.
  - file: cmd/backstop/gate.go
    provides:
      - name: buildGateSteps
        kind: function
        signature: "func buildGateSteps(projectRoot string, scope ...*gate.GateScope) []gate.StepFunc"
        notes: "The native lint/build/test toolchain stops running via the realCodeChecker -> check.Run Step 2 and is bridged onto the existing dispatchPackEngines step: the go-toolchain pack's lint/build/test engine bindings are dispatched alongside (or as) the pack engine step. realCodeChecker no longer runs the toolchain passes (REQ-002/CLM-005). No new parallel dispatcher is introduced; dispatchPackEngines is reused (REQ-001/CLM-003). cmd/backstop already imports both pkg/check and pkg/pack/engine, so the bridge adds NO pkg/check -> pkg/pack/engine import (REQ-001/CLM-002)."
    consumes:
      - source: cmd/backstop/pack_gate.go
        name: dispatchPackEngines
        kind: function
      - source: pkg/pack/engine/binding.go
        name: Registry
        kind: type

  - file: cmd/backstop/pack_gate.go
    provides:
      - name: dispatchPackEngines
        kind: function
        signature: "func dispatchPackEngines(packs []*pack.Manifest, packDir, projectRoot string, scope *gate.GateScope, runner check.CommandRunner) ([]gate.Violation, error)"
        notes: "REUSED as the single dispatch substrate for native lint/build/test (REQ-001). Its shape gained a scope *gate.GateScope param (ISSUE-010 diff-scope fix) threaded ahead of the runner; the native bridge reuses this same signature, introducing no parallel dispatcher. lint is a config-file findings engine (RunStdout, no convert); build/test are findings engines with a pack-relative convert script run via SandboxedRunStdout in runFindingsEngine. The build/test convert path must preserve the crash-vs-findings guard (a non-zero exit with no parseable findings is a crash, not green — REQ-003/CLM-010); today runFindingsEngine discards runErr, so this guard is the one behavioral addition the bridge makes to the findings path."
    consumes:
      - source: pkg/check/runner.go
        name: RunStdout
        kind: method
      - source: pkg/packval/sandbox.go
        name: SandboxedRunStdout
        kind: function
      - source: pkg/check/parsers.go
        name: ParsePackFindings
        kind: function

  - file: pkg/check/registry.go
    provides:
      - name: buildExecutorsForConfigErr
        kind: function
        signature: "func buildExecutorsForConfigErr(opts Options, runner CommandRunner) (map[CheckType]PassExecutor, error)"
        notes: "The `if language == \"go\" { return goBuiltinExecutors(...) }` short-circuit is deleted (REQ-002/CLM-004). The go case of builtinToolchain stops constructing native lint/build/test passes; those now run through the bridge (dispatchPackEngines). semgrep stays the shared executor. Layer-0 assume-present fail-loud (REQ-008): a missing go/golangci-lint binary is surfaced as a ConfigError (exit 2), NEW behavior added here."
    consumes:
      - source: pkg/pack/engine/binding.go
        name: Registry
        kind: type

  - file: pkg/check/parsers.go
    provides:
      - name: formatParsers
        kind: variable
        signature: "var formatParsers map[string]Parser"
        notes: "Removes the go-build, go-test, and golangci-json entries (their parser logic relocates to the go-toolchain pack convert scripts / native golangci v2 SARIF). Retains sarif (the pack findings contract) plus the TS-stack formats (eslint-json, tsc, regex-lines)."
    consumes: []

  - file: pkg/check/semgrep.go
    provides:
      - name: EnsureSemgrep
        kind: function
        signature: "func EnsureSemgrep(backstopDir string, pinnedVersion string) (string, error)"
        notes: "Its bespoke PATH-probe / .backstop/tools / pip-install ladder retires into the engine model's declared provisioning (REQ-008). The native Run path no longer calls the ad hoc installer; provisioning of the backstop-introduced semgrep engine is declared and data-driven."
    consumes:
      - source: pkg/pack/engine/binding.go
        name: Provision
        kind: type
---

# SPEC-034: Native Toolchain Engine Cutover

## Overview

> ⚠️ **TOMBSTONE — THIS SPEC IS EFFECTIVELY SUPERSEDED.** *(Verified against `main` @
> 2026-06-24.)* **DO NOT pick this up as live, plannable work** despite its `draft`
> status.
>
> - **Only the BRIDGE half landed on `main`.** SPEC-034's cutover was implemented on
>   branch `feat/bundle-010-impl`, but only its bridge reached `main`:
>   `loadBridgedToolchainPacks` (defined `cmd/backstop/gate.go:411`, invoked at
>   `gate.go:464`) routes a native `<lang>-toolchain` pack through
>   `dispatchPackEngines`.
> - **The DELETION half did NOT land.** On `main`, `realCodeChecker` → `pkg/check.Run`
>   is **still the live gate Step 2** (`cmd/backstop/gate.go:481` comment, `:488`
>   wiring; the bespoke executors are still constructed via
>   `buildExecutorsForConfigErr` at `pkg/check/check.go:293`), and `builtinToolchain`
>   (`pkg/check/registry.go:59`) is **still the live native-toolchain source**. *(The
>   old `goBuiltinExecutors` name is no longer present under that name on `main`;
>   `builtinToolchain` is the current symbol.)*
> - **The remaining scope is ABSORBED BY BUNDLE-011.** The cutover + deletion scope
>   has been absorbed by **BUNDLE-011** (Seed 2 → SPEC-040; recorded in BUNDLE-011
>   **REQ-008 / DD-4 / RDQ-4**), which will spec and implement the work from there,
>   **generalized beyond Go** to any `<lang>-toolchain` pack.
> - **Why it stays `draft`.** This spec remains `status: draft` ONLY because the spec
>   schema currently has **no terminal state**; that gap is tracked by **ISSUE-031**
>   (artifact terminal states), which will add a real `superseded` state — at which
>   point SPEC-034 should be flipped to it.
>
> The original intent, phased design, requirements, and claims below are retained
> verbatim for traceability.

This spec is the **native code-check toolchain half** of BUNDLE-010 — the
companion to SPEC-031's pack-side engine dispatch. SPEC-031 first-classed the
`engine` field and built the `EngineBinding` table, the `config-file` / `sandbox`
engine shapes, the strict-SARIF output contract, and the split-provisioning model,
all driving the gate's `dispatchPackEngines` substrate (cmd/backstop/pack_gate.go).
This spec routes the **native** lint/build/test passes onto that **same** substrate
and DELETES the bespoke Go path.

**Today the gate runs two disjoint substrates at the cmd/backstop level.** Step 2,
`realCodeChecker` → `check.Run` (pkg/check), is the bespoke Go toolchain:
`buildExecutorsForConfigErr` (pkg/check/registry.go) short-circuits on
`if language == "go"` and returns `goBuiltinExecutors` (pkg/check/check.go) — the
hand-written `lintExecutor`/`buildExecutor`/`testExecutor` backed by hand-written
`parseGolangciJSON` (with version-adaptive flag logic), `parseGoBuildErrors`, and
`parseGoTestFailures`. A separate gate step, `dispatchPackEngines`, runs the
declared engine model SPEC-031 built. **`pkg/check` does not import
`pkg/pack/engine`; the two are disjoint.** `cmd/backstop` already orchestrates both.

**The central, load-bearing work of this spec is THE BRIDGE (REQ-001, phase 1):**
route the native lint/build/test passes through the SAME `dispatchPackEngines`
substrate the packs already use, as Layer-0 engine passes declared in a reusable
`go-toolchain` pack. The bridge **reuses** the dispatch substrate — it does **not**
invent a parallel dispatcher and does **not** add a `pkg/check` → `pkg/pack/engine`
import (orchestration stays at cmd/backstop, which already imports both). Every
other requirement sits on top of this bridge. The four passes then map as in the
table below: lint becomes a `config-file` findings engine on native v2 SARIF;
build/test become findings engines with a pack `convert` script; semgrep is the
unchanged shared engine.

The cutover is a **strangler expressed as ordered, testable phases** (REQ-009):
wire the bridge → prove equivalence (parser AND invocation, including the
`code check --file` go-test package scoping) → and only in a phase **gated on that
equivalence test passing** delete the bespoke executors/parsers/short-circuit/
install ladder and migrate-or-delete the bespoke-asserting test files. The **end
state** (REQ-011) carries zero baked-in Go toolchain knowledge.

**Scope boundary (non-goals).** This spec is the **native code-check toolchain half
ONLY** — lint/build/test. It does **NOT** touch the traceability steps (coverage,
test substantiveness, contract signatures); those are **BUNDLE-009**'s separate
migration onto ast-grep packs and are explicitly out of scope here. It does not
re-open SPEC-031's pack dispatch path, the SARIF contract, or the
`config-file`/`sandbox`/`Provision` machinery — it consumes them.

**Verification Level:** Integration (80% coverage).
**Source Bundle:** BUNDLE-010 (pluggable-pack-engines) — discharges REQ-018,
REQ-019, DD-2, and DD-8 (Layer-0 native toolchain). Sibling: **BUNDLE-009**
(stack-aware traceability), the traceability migration that is OUT of scope here.

## Requirements

Requirements and claims are defined in frontmatter. The two summary tables below
must match the requirement text exactly.

### Pass → engine mapping (REQ-004, REQ-005, REQ-006)

The four native passes map onto the `dispatchPackEngines` substrate as follows.
Lint and build/test are **findings engines** through `runFindingsEngine`; build/
test add a pack `convert` script (run via the sandboxed clean-stdout capture) that
normalizes tool output to SARIF, with the crash-vs-findings guard preserved.
Note: the bundle's DD-2 carve-out kept build/test as non-SARIF in the *pack*
dispatch path; the settled design for this *native* migration is that build/test
DO normalize to located SARIF findings via a pack `convert` script (so file:line
findings survive), running the convert path of `runFindingsEngine` rather than the
exit-code `none` branch.

| Pass | Engine shape | Capture | Converter | What is deleted from core |
|---|---|---|---|---|
| build | findings engine (`go-toolchain` pack) | `RunStdout` → `convert` via `SandboxedRunStdout` | pack `convert` script → SARIF | `parseGoBuildErrors` + `go-build` format |
| test | findings engine (`go-toolchain` pack) | `RunStdout` → `convert` via `SandboxedRunStdout` | pack `convert` script → SARIF | `parseGoTestFailures` + `go-test` format |
| lint | `config-file` engine (golangci-lint) | `RunStdout` (clean stdout) | none — v2 native SARIF → `parseSarif` | `lintExecutor`, `parseGolangciJSON`, `golangci-json` format, version-adaptive flag logic |
| semgrep | shared engine (SPEC-031) | unchanged | unchanged | nothing — only `EnsureSemgrep` install retires (REQ-008) |

### Provisioning ownership (REQ-008)

Provisioning splits by who owns the tool. The project owns its compiler/linter;
backstop owns the engines it introduces.

| Tool | Ownership | Provisioning | On absence |
|---|---|---|---|
| `go` | Layer-0 native (project-owned) | assume-present; backstop never installs | fail loud (`ConfigError` naming the tool) |
| `golangci-lint` | Layer-0 native (project-owned) | assume-present; backstop never installs | fail loud (`ConfigError` naming the tool) |
| `semgrep` | backstop-introduced | pinned + auto-provisioned (declared) | provisioned per declaration; `EnsureSemgrep` bespoke ladder retired |

## Implementation

The work is structured as the strangler's three ordered phases. Phase 1 builds the
bridge and the pack; phase 2 proves equivalence (including invocation behavior);
phase 3 deletes, **gated** on phase 2 being green. Each numbered stage is a
distinct unit the planner can map tasks to. Nothing in phase 3 may land until the
phase-2 equivalence test exists and passes.

### Phase 1 — build the bridge + the `go-toolchain` pack

#### 1.1 The bridge: native passes onto `dispatchPackEngines` (REQ-001)

The native lint/build/test passes are routed through the EXISTING
`dispatchPackEngines` substrate (cmd/backstop/pack_gate.go), the same path the
packs use. The wiring lives in `cmd/backstop/gate.go` (`buildGateSteps`), which
already constructs both the `realCodeChecker` (→ `check.Run`) step and the
`dispatchPackEngines` step — so the bridge is a convergence at the orchestration
layer, NOT a new dispatcher and NOT a `pkg/check` → `pkg/pack/engine` import
(`cmd/backstop` already imports both). The `go-toolchain` pack's lint/build/test
engine bindings are dispatched as Layer-0 engine passes. `realCodeChecker`/
`check.Run` stops running the native lint/build/test passes (REQ-002).

#### 1.2 The `go-toolchain` pack (REQ-007)

A reusable `go-toolchain` pack holds the build/test `convert` scripts and the lint
`config-file` binding (with its `.golangci.yml`, per the OQ-1 resolution) —
**mechanism** (run the Go
toolchain, normalize to SARIF; identical for every Go project), SEPARATE from the
opinionated `backstop-go-pack` — **opinion** (swappable coding-standards semgrep
rules). The `go-toolchain` pack carries no coding-standards rules; `backstop-go-pack`
carries no toolchain mechanism. The pack is authored alongside this spec and lands
in **lockstep** with the bridge (REQ-007): the bridge and the pack must not land in
separate independently-green commits that leave a window with no Go build/test
enforcement.

#### 1.3 Build & test → findings engine + sandboxed convert (REQ-003, REQ-004)

Build and test are declared as findings engines in the `go-toolchain` pack. They
run through `runFindingsEngine`: the tool's stdout is captured via `RunStdout`,
piped to the pack-relative `convert` script run via `SandboxedRunStdout`
(pkg/packval/sandbox.go) — the same sandboxed clean-stdout capture SPEC-031's
convert step uses — and the resulting SARIF is parsed via `parseSarif`
(`check.ParsePackFindings`). The `convert` script re-expresses the retired
`parseGoBuildErrors` / `parseGoTestFailures` normalization OUTSIDE the core binary.
`runFindingsEngine` today discards the tool's run error (`_ = runErr`); the bridge
must add the **crash-vs-findings guard** on the build/test convert path so a
non-zero exit with no parseable findings surfaces as a crash, not a silent green
(REQ-003 / CLM-010). This guard is the one behavioral addition to the findings path.

#### 1.4 Lint → config-file engine, native SARIF via `RunStdout` (REQ-005)

The lint pass is a `config-file` engine running golangci-lint v2, which emits SARIF
natively to stdout. The pass declares **no converter**; its output is captured via
`RunStdout` (clean stdout, NOT `CombinedOutput`, because golangci-lint writes
progress/warnings to stderr) and parsed via `parseSarif`. The invocation pins/
assumes v2 SARIF — no `golangci-lint version` probe, no v1/v2 flag branch.

#### 1.5 Split provisioning; absent-tool fail-loud (REQ-006, REQ-008)

The Layer-0 native toolchain (`go`, `golangci-lint`) is assume-present: a missing
binary fails loud with a `ConfigError` (exit 2) naming the tool, and backstop never
installs it. This absent-tool fail-loud is NEW behavior — neither the bespoke path
nor the bare engine path emits it today. The backstop-introduced `semgrep` engine
stays pinned and auto-provisioned via the declared `Provision` mechanism;
`EnsureSemgrep`'s bespoke PATH-probe → `.backstop/tools` → `pip install` ladder
(pkg/check/semgrep.go) retires from the native Run path. The semgrep findings path
itself is unchanged (REQ-006).

### Phase 2 — prove equivalence (parser + invocation) (REQ-009, REQ-010)

#### 2.1 Parser equivalence (the gate, REQ-009)

The phase-2 equivalence gate proved the engine path emitted **identical normalized
violations** to the bespoke path for the same captured tool output — a `go build`
compiler-error sample, a `go test` failure sample (run through the real pack
`convert` script), and a golangci-lint findings sample (the bespoke
`parseGolangciJSON` on v1 JSON vs the engine path's `parseSarif` on v2 SARIF for the
equivalent findings) — and that green proof **licensed the deletion** of the bespoke
parsers. This equivalence comparison was **transitional**: once the bespoke parsers
were deleted (phase 3 / phase 5) the comparison could no longer compile, so the
ongoing verification for CLM-030…CLM-032 is now the **standalone** engine tests
(`TestGoToolchain_BuildFindingsEngineWithConvert`,
`TestGoToolchain_TestFindingsEngineWithConvert`,
`TestGoLint_ConfigFileEngineNativeSarif`) that assert the engine path's exact
findings directly. Those standalone tests — and the standalone convert tests for
CLM-008/009/017 — run the **real** `go-toolchain` `convert` scripts (or the real
golangci v2 SARIF) against **real** captured output, never canned SARIF.

#### 2.2 Invocation equivalence: `code check --file` test scoping (REQ-010)

The `code check --file` standalone-hook path today scopes `go test` to the changed
file's PACKAGE (`goPackageSelector` / `testExecutor.fileMode`) under a tight time
budget, rather than `./...`. The bridge must EITHER preserve this file-mode package
scoping for the test engine OR record an explicit decision to drop it; the chosen
outcome is covered by a mandated test (CLM-034/CLM-035). Equivalence is not parser-
only — this invocation behavior is in scope for the gate.

### Phase 3 — rip out the bespoke path, GATED on phase 2 (REQ-009, REQ-011)

Only when the phase-2 equivalence test exists and passes: delete
`goBuiltinExecutors`, `lintExecutor`, `buildExecutor`, `testExecutor`;
`parseGoBuildErrors`, `parseGoTestFailures`, `parseGolangciJSON` and their named
formats; the `if language == "go"` short-circuit; and `EnsureSemgrep`'s bespoke
install path. Migrate-or-delete the seven bespoke-asserting test files enumerated
in REQ-011 (`check_test.go`, `engine_semantics_test.go`, `executor_test.go`,
`registry_coverage_test.go`, `registry_test.go`, `ts_executor_test.go`,
`ts_routing_test.go`) so no test references a deleted symbol. Their migration/
deletion is enforced by `TestEndState_NoTestReferencesDeletedBespokeSymbol`
(CLM-037), not by the contract-signature step: `contract_signature` verifies a
symbol is PRESENT with a given signature and has no must-be-absent semantic, so
absence (the deleted bespoke executors/parsers/short-circuit and the migrated test
files) is asserted by the deletion-assertion tests `TestCutover_*` /
`TestStrangler_DeletionGatedOnProvenEquivalence` / `TestEndState_*`, which are the
correct mechanism for proving removal. "Deletion is gated on
proven equivalence" is itself checkable: a guard test asserts the equivalence test
is present and green and the bespoke symbols are absent (CLM-033), so a deletion
that outran the proof fails the gate. After this, a grep of pkg/check non-test
source for the deleted symbols and the `language == "go"` branch returns zero
(CLM-036), and adding a new stack requires no pkg/check change (CLM-038).

## Verification

Verification is defined in frontmatter. Integration-level testing at 80% coverage
across `cmd/backstop` (the bridge wiring and `dispatchPackEngines` driving the
native passes), `pkg/check` (parser deletions, provisioning fail-loud), and
`pkg/pack/engine` (the bindings consumed).

Tests use a fake `CommandRunner` (`RunStdout`) returning captured `go build` /
`go test` / golangci-lint output (no live tools), a stubbed `SandboxedRunStdout`
so the convert pipe is exercised without a live sandbox, a fixture `go-toolchain`
pack with the **real** `convert` scripts, and captured-output fixtures the
transitional equivalence gate once shared between the bespoke parser and the engine
path. Post-cutover the bespoke parsers are deleted, so CLM-030…CLM-032 now verify
via the **standalone** engine tests (`TestGoToolchain_BuildFindingsEngineWithConvert`,
`TestGoToolchain_TestFindingsEngineWithConvert`,
`TestGoLint_ConfigFileEngineNativeSarif`) that assert the engine path's exact
normalized findings directly against those fixtures. The provisioning claims
(CLM-026…CLM-027) inject a fake binary-resolver so a missing `go` / `golangci-lint`
yields a fail-loud `ConfigError` (exit 2) without depending on the host PATH. The
file-mode scoping claim (CLM-034) asserts the test invocation targets the changed
file's package, not `./...`. The deletion-gate guard (CLM-033) and the end-state
grep (CLM-036) / no-dangling-test (CLM-037) claims assert the phase-3 invariants.

The standalone convert tests must run the **real** `go-toolchain` `convert`
scripts against **real** captured tool output (not canned SARIF) — a stub emitting
canned SARIF would let the relocation pass without proving the transform,
re-introducing the very risk DD-2's "removed, not extracted" guards against.

## Open Questions

These were flagged for resolution during planning/authoring of the `go-toolchain`
pack (REQ-007). They are not pre-decided in the requirements; the requirements
above are written to hold under either resolution, and the claims do not assume
one. Both are now resolved (2026-06-18); the resolutions pick the "either" without
changing any requirement or claim.

- **OQ-1 — Does the `go-toolchain` pack own the lint `config-file` entry, or does
  the golangci-lint config stay project-side?** The build/test toolchain mechanism
  clearly belongs in the `go-toolchain` pack (REQ-007). Lint is different: it is a
  `config-file` engine (REQ-005) where the tool runs its OWN rules tuned by an
  optional config. The config a project lints with is arguably project opinion, not
  reusable mechanism — so the lint config-file entry may belong project-side
  (backstop.yml / the project's `.golangci.yml`) rather than in the reusable
  mechanism pack. Resolution affects only WHERE the lint config-file binding is
  declared, not whether lint converges to native SARIF (REQ-005 holds either way).

  **Resolved (2026-06-18): pack-defined.** The lint `config-file` binding AND its
  config (`.golangci.yml`) live in the `go-toolchain` pack, not project-side.
  Adopting the pack yields working lint with zero project-side wiring. REQ-005
  (lint → native SARIF) is unaffected; this only fixes WHERE the binding is
  declared — inside the reusable mechanism pack, consistent with convention over
  configuration.

- **OQ-2 — One `go-toolchain` pack, or a build/test split?** REQ-007 mandates a
  reusable mechanism pack separate from `backstop-go-pack`, but does not settle
  whether build and test ship as one `go-toolchain` pack or as separate packs
  (e.g. `go-build` / `go-test`). A single pack is simpler; a split could let a
  project adopt build-gating without test-gating. The mechanism-vs-opinion boundary
  (REQ-007) and the pass→engine mapping hold under either; this is a packaging
  granularity decision for pack authoring.

  **Resolved (2026-06-18): one `go-toolchain` pack, no split, no rename.** Lint +
  build + test ship together as the single native Go toolchain pack. Rationale:
  lint is part of the native toolchain; the split's only payoff (build-gating
  without test-gating) is degenerate because `go test ./...` already compiles
  everything `go build ./...` does. The name `go-toolchain` (REQ-007) stands.

**Generalizing convention established by these resolutions.** One
`<lang>-toolchain` pack per language bundles that language's whole native
toolchain (lint + build + test + whatever else is native), with pack-defined
config. This is convention over configuration — a default, NOT a hard-and-fast
rule. `go-toolchain` is the reference instance.

## Sharp Edges

1. **Reuse the substrate; do not invent a parallel one or add an import cycle.** The
   bridge (REQ-001) must route the native passes through the existing
   `dispatchPackEngines`, not a second native engine dispatcher. The temptation is to
   "just call the engine model from `pkg/check`" — but `pkg/check` must NOT import
   `pkg/pack/engine` (the two are disjoint, and the import would invert the leaf-
   package placement SPEC-031 REQ-013 protects). Orchestration belongs at
   `cmd/backstop`, which already imports both. A parallel dispatcher would re-create
   the very two-substrate split this spec exists to collapse.

2. **Enforcement-lapse window (the cardinal sin).** Deleting the bespoke build/test
   path before the engine path is proven equivalent leaves a window with NO Go
   build/test enforcement — vacuous green. The phased strangler (REQ-009) is
   load-bearing and made TESTABLE: phase 3 deletion is gated on the phase-2
   equivalence test passing (CLM-033). The bridge, the `go-toolchain` pack, and the
   deletion must not land as separate independently-green commits; a commit that
   removed `parseGoTestFailures` but not yet wired the convert script would ship
   green while enforcing nothing.

3. **build/test convert path must capture stdout cleanly AND keep the crash guard.**
   Build/test are findings engines whose pack `convert` script normalizes output to
   SARIF. The convert step must run via `SandboxedRunStdout` (clean stdout), not
   `SandboxedRun`'s `CombinedOutput`, so a converter's stderr banner cannot corrupt
   the SARIF. Separately, `runFindingsEngine` currently discards the tool's run error
   (`_ = runErr`); for build/test the crash-vs-findings guard must be ADDED so a
   non-zero exit with no parseable findings surfaces as a crash, not a finding-free
   green. Both omissions independently produce vacuous green.

4. **Lint SARIF on stdout — `CombinedOutput` corrupts it (same class as the semgrep
   `--sarif` fix).** golangci-lint v2 writes SARIF to stdout and progress/warnings to
   stderr. Capturing via `CombinedOutput` interleaves stderr into the SARIF bytes and
   breaks `parseSarif` — exactly the bug semgrep's `--sarif`/`RunStdout` capture
   already fixed for the pack path. The lint engine MUST capture via `RunStdout`
   (REQ-005 / CLM-016).

5. **golangci-lint v1 vs v2 SARIF assumption.** The spec pins/assumes golangci-lint
   v2 (native SARIF) and deletes the version-adaptive flag logic. A project on v1 will
   not emit SARIF, so the lint pass must fail loud (unparseable-SARIF attributed to
   the lint engine), NOT silently produce zero findings. The assume-present check
   (REQ-008) should surface a too-old binary as a loud failure, not a green pass.

6. **macOS-only sandbox bounds where the convert step runs (cross-platform gap).**
   `SandboxedRun`/`SandboxedRunStdout` wrap macOS `sandbox-exec`, so the build/test
   `convert` step only runs on macOS today — the same known vision-gap BUNDLE-010
   Non-Goal 10 records for the pack convert path. This spec inherits, not closes, that
   limitation: the implementation must fail loud (not silently green) on a platform
   where the sandbox is unavailable, and cross-platform sandboxing (Linux seccomp/
   landlock/containers) is an explicit documented limitation with a follow-up, out of
   scope here. Resolution of OQ-1/OQ-2 must not assume Linux coverage.

7. **"Removed, not extracted" can be quietly violated.** The intent (DD-2) is that
   the Go build/test parser LOGIC leaves the core binary. A lazy migration could copy
   `parseGoBuildErrors` into a Go helper still compiled into core and call it a
   "converter." The end-state grep (REQ-011 / CLM-036) and the requirement that the
   convert script run OUTSIDE the binary guard against this: the transform lives in
   the pack, not a renamed core function.

8. **Invocation behavior is part of equivalence — file-mode test scoping must not
   silently regress.** Parser equivalence is necessary but not sufficient. The
   `code check --file` hook scopes `go test` to the changed file's package
   (`goPackageSelector`) under a tight budget; routing test through the bridge could
   silently switch it to `./...`, blowing the hook budget and changing behavior.
   REQ-010 forces this to be preserved-or-explicitly-decided and covered by a test
   (CLM-034/CLM-035).

9. **Mechanism/opinion bleed between the two packs.** It is tempting to fold the
   build/test toolchain scripts into `backstop-go-pack` since both are "Go packs."
   REQ-007 forbids this: `go-toolchain` is reusable mechanism, `backstop-go-pack` is
   swappable opinion (a project may reject its coding standards but still wants
   deterministic build/test gates). Conflating them couples the universal mechanism to
   a specific opinion.

10. **EnsureSemgrep retirement must not break semgrep provisioning.** Retiring the
    bespoke install ladder (REQ-008) removes the ad hoc PATH/tools/pip logic from the
    native Run path, NOT semgrep provisioning. A migration that just deletes
    `EnsureSemgrep`'s call site without re-routing provisioning leaves semgrep
    unprovisioned (degraded/skip) — a silent enforcement gap.

11. **The seven bespoke-asserting test files must move with the code.** Deleting the
    bespoke executors/parsers without migrating-or-deleting `check_test.go`,
    `executor_test.go`, `registry_test.go`, `registry_coverage_test.go`,
    `engine_semantics_test.go`, and auditing `ts_executor_test.go` / `ts_routing_test.go`
    leaves dangling references to removed symbols (compile break) or stale coverage.
    REQ-011 / CLM-037 makes their fate explicit.

12. **Traceability steps are NOT in scope and must not be touched.** Coverage,
    substantiveness, and contract-signature steps are Go-bound today and migrate under
    BUNDLE-009. A "while I'm here" change would cross the scope boundary and collide
    with BUNDLE-009's ast-grep-pack migration. This spec touches only lint/build/test.

## Review Questions

1. Does the bridge route the native lint/build/test passes through the EXISTING
   `dispatchPackEngines`, or did it introduce a second native engine dispatcher?
   Confirm reuse, not a parallel path.

2. Does `pkg/check` import `pkg/pack/engine` anywhere after the bridge? It must NOT —
   confirm the two substrates stay disjoint and the bridge lives at `cmd/backstop`,
   which already imports both.

3. After the cutover, does any non-test file in `pkg/check` still reference
   `goBuiltinExecutors`, `lintExecutor`, `buildExecutor`, `testExecutor`,
   `parseGoBuildErrors`, `parseGoTestFailures`, `parseGolangciJSON`, or the
   `language == "go"` branch? A grep must return zero — deleted, not merely bypassed.

4. Is the build/test parser LOGIC genuinely in the `go-toolchain` pack `convert`
   script (outside the binary), or was a bespoke parser copied into a core helper and
   renamed? Confirm the transform runs as pack-author code via `SandboxedRunStdout`,
   not a renamed in-binary function.

5. Does the lint engine capture golangci-lint's SARIF via `RunStdout` (clean stdout),
   or `CombinedOutput`? The latter corrupts the SARIF with stderr — confirm clean
   capture (the semgrep `--sarif` fix class).

6. Is phase-3 deletion genuinely gated on the phase-2 equivalence test existing and
   passing (CLM-033), or could a deletion land ahead of the proof? Confirm the
   deletion-gate guard test exists and that the equivalence comparison runs the
   **real** `convert` scripts against **real** captured output, not canned SARIF.

7. Is the `code check --file` go-test package scoping (`goPackageSelector`) preserved
   through the bridge, or was it silently changed to `./...`? If dropped, is that an
   explicit recorded decision with a test asserting the chosen behavior (CLM-034/035)?

8. Does a missing `go` or `golangci-lint` fail loud with a `ConfigError` (exit 2)
   naming the tool, and is backstop genuinely NOT installing it? This is NEW behavior —
   confirm it is actually emitted, since neither the old nor the bare new path emits it.

9. After retiring `EnsureSemgrep`'s bespoke install, is semgrep still pinned and
   provisioned through the declared model, or did provisioning silently drop?

10. Does the build/test `convert` step fail loud (not silently green) on a non-macOS
    platform where `SandboxedRunStdout` cannot run, and is the limitation documented
    as a follow-up rather than assumed away?

## References

- **BUNDLE-011** (collapse-legacy-codecheck-into-packs) — **the absorbing bundle.**
  Per BUNDLE-011 **REQ-008 / DD-4 / RDQ-4**, BUNDLE-011 absorbs SPEC-034's unfinished
  deletion scope (the bridge landed; the deletion did not) and **generalizes the
  cutover beyond Go** to any `<lang>-toolchain` pack. The work is **Seed 2 → SPEC-040**.
  SPEC-034 is marked SUPERSEDED-in-prose here; the actual cutover + deletion proceeds
  from BUNDLE-011/SPEC-040, NOT from this spec.
- **ISSUE-031** (artifact terminal states) — tracks the schema gap that forces this
  spec to stay `status: draft`: the spec schema has **no terminal state** today.
  ISSUE-031 will add a real `superseded` state; once it lands, SPEC-034 should be
  flipped to it (this prose tombstone is the interim marker).
- **BUNDLE-010** (pluggable-pack-engines) — discharges **REQ-018** (engine-shape-
  agnostic, Layer-0 native toolchain first-class; "adding a native linter must be a
  declaration, no backstop Go"), **REQ-019** (split provisioning; "EnsureSemgrep's
  bespoke install logic retires into declared provisioning"), **DD-2** ("bespoke
  parsers retire as migration debt — removed, not extracted"), and **DD-8** (the
  escalation ladder / Layer 0 — native toolchain).
- **SPEC-031** (pluggable engine dispatch) — the engine substrate this spec routes
  the native passes onto: `dispatchPackEngines` / `runFindingsEngine`, the
  `EngineBinding` table, the `config-file` / `sandbox` engine shapes, the strict-SARIF
  output contract + `parseSarif`, the `RunStdout` clean-capture runner, the sandboxed
  `SandboxedRunStdout` convert step, and the `Provision` split. SPEC-031 REQ-013
  explicitly disclaimed the native `ToolchainEntry` → engine conversion as "owned
  elsewhere"; this spec is that elsewhere — and resolves it as a BRIDGE onto the
  existing dispatch, not a second dispatcher.
- **BUNDLE-009** (stack-aware traceability) — the **sibling** migration:
  coverage / substantiveness / contract-signature steps onto ast-grep packs.
  Explicitly **OUT of scope** here; this spec is the code-check half only.
- **ISSUE-003** (stack-keyed toolchain registry) — §4 "Go toolchain as registry
  entries" was specced ("move from hardwired defaults to named registry slots") but
  never landed; this spec completes it via the engine bridge. Also the source of
  `ScopeKind` / `formatParsers` / `parseSarif`, and Ratified Design Constraint 6
  ("traceability steps stay Go-only … out of scope") which BUNDLE-009 now lifts.
- **ISSUE-002** — the source of the bespoke Go executors (`lintExecutor`,
  `buildExecutor`, `testExecutor`) this spec retires.
- **SPEC-030** (packs-only native-standards removal) — collapses the gate's
  rule-config source to packs; a sibling pillar-2 cleanup in the same "stop baking
  rules into the binary" family.
- Code: cmd/backstop/gate.go (`buildGateSteps` — wires both `realCodeChecker`→
  `check.Run` Step 2 AND the `dispatchPackEngines` step; the bridge converges them);
  cmd/backstop/pack_gate.go (`dispatchPackEngines`, `runFindingsEngine`,
  `runSandboxEngine`, `gatherEngineInputs` — the reused substrate; `runFindingsEngine`
  currently discards `runErr`, the crash-guard gap); pkg/check/runner.go (`RunStdout`,
  `CommandRunner`); pkg/packval/sandbox.go (`SandboxedRunStdout` — the sandboxed
  clean-stdout convert capture, macOS-only); pkg/check/registry.go (the
  `if language == "go"` short-circuit, `buildExecutorsForConfigErr`); pkg/check/check.go
  (`goBuiltinExecutors`, `lintExecutor`, `buildExecutor`, `testExecutor`,
  `parseGoBuildErrors`, `parseGoTestFailures`, `parseGolangciJSON`, `golangciOutputArgs`,
  `golangciMajorVersion`, `goPackageSelector`/`testExecutor.fileMode`); pkg/check/parsers.go
  (`formatParsers` — `go-build`, `go-test`, `golangci-json` entries; `parseSarif`,
  `lookupParser`); pkg/check/semgrep.go (`EnsureSemgrep` bespoke install ladder);
  pkg/pack/engine/binding.go (`EngineBinding`, `Provision`, `config-file` / `sandbox`
  shapes — consumed, not changed).
