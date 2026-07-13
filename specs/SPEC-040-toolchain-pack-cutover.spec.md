---
title: "Toolchain Pack Cutover"
number: SPEC-040
created: "2026-06-24"
status: implemented
schema_version: spec/v1
spec_version: 2.0.0

implementation:
  summary: >
    The KEYSTONE cutover of BUNDLE-011 (Seed 2): retire `pkg/check.Run` as the gate's
    Step 2 ("Code check") enforcement engine and route lint/build/test exclusively
    through the declared-engine substrate `dispatchPackEngines` (cmd/backstop/pack_gate.go)
    as `<lang>-toolchain` pack passes. On `main` today the bridge `loadBridgedToolchainPacks`
    (cmd/backstop/gate.go:411) — the landed half of the superseded SPEC-034 — already
    dispatches the Go native lint/build/test passes through `dispatchPackEngines` (the go
    `builtinToolchain` stack at pkg/check/registry.go:59 already returns EMPTY pass entries),
    but two things did NOT land: (a) the `realCodeChecker` → `pkg/check.Run` step is STILL
    wired as gate Step 2 (gate.go:520 `StepCodeCheckScopedFunc`), still running for the
    typescript `builtinToolchain` stack (eslint/tsc/regex-lines, registry.go:65-92) — the baked
    go/ts stacks in `builtinToolchain` are the actual zero-baked-checks violation; and
    (b) there is no LOUD report state for "0 toolchain packs / nothing ran" — a project with
    no toolchain pack silently produces a normal green. This spec finishes the deletion and
    generalizes the cutover machinery beyond Go. Concretely (REQ-001/002/003, RDQ-1): WHOLESALE
    replace Step 2 — `realCodeChecker`/`pkg/check.Run` stops being a gate step, lint/build/test
    run ONLY as declared `<lang>-toolchain` pack passes through `dispatchPackEngines`, and the
    legacy Step-2 gate path plus the baked `builtinToolchain` go/ts stacks are DELETED in the
    SAME PR (RETAINING `resolveToolchain`/`commandExecutor`/`buildExecutorsForConfigErr` in
    reduced declared-only form — they serve the surviving `code check` subcommand; the delete
    set must be internally consistent, no deleted symbol with a surviving caller) — gated on a
    ONE-SHOT golden-equivalence assertion
    (capture the legacy engine's violation set on the backstop repo as a golden fixture, assert
    the pack-engine path reproduces it, then delete; the fixture IS the evidence — no standing
    dual-run, no parity-gate apparatus). (REQ-004/005/006, RDQ-2): establish the
    one-`<lang>-toolchain`-pack-per-language model (a `go-toolchain` pack exists as a fixture
    under `cmd/backstop/testdata/` and is the worked example — note it is NOT in backstop-core's
    live dogfood install `.backstop/packs/backstop/`, which has only `go-standards`, so the live
    project hits the no-toolchain-pack state until it installs one), make the no-toolchain-pack baseline
    WARN-ONLY (enforcement is opt-in for backstop's legitimate chain-only / recipe-only postures),
    BUT surface the "0 toolchain packs / nothing ran" state as a LOUD, DISTINCT, non-failing
    report state ("warning" status, never collapsed into a normal green) — the anti-vacuous-green
    guardrail, reusing the existing SPEC-036 `"warning"`/`StepsWarned` loud-but-passing mechanism
    (pkg/gate/result.go). (REQ-008, RDQ-4): absorb SPEC-034's unfinished deletion and generalize
    the cutover machinery beyond Go; SPEC-034 is marked SUPERSEDED separately. (REQ-009, RDQ-4):
    do NOT regress BUNDLE-009's landed traceability packs (substantiveness/contracts), which run
    through their OWN dedicated gate steps via `dispatchPackEngines` and are excluded from the
    generic findings dispatch by `excludeDedicatedStepRules` (pack_gate.go:118). The test-runner
    ↔ coverage coupling is handled explicitly as a named contract: the shared `go test ./...`
    runner (cmd/backstop/shared_testrun.go) today feeds BOTH Step 2's test FAILs AND the still-baked
    coverage step (`buildCoverageStep`, gate.go); deleting `pkg/check.Run` must NOT orphan the
    coverage step before SPEC-041 (Seed 3) migrates coverage. OUT OF SCOPE (fenced to SPEC-039
    Seed 1 and SPEC-041 Seed 3): the dead standards-manifest reader + non-Go semgrep catch-all
    deletions (SPEC-039 — prerequisite); coverage eradication, `step_coverage.go` deletion, the
    build-pass project-wide-scope exemption (`cv.Pass == check.CheckTypeBuild`,
    `checkViolationsToGate`) re-expression, and the CheckType-consumer catalog (SPEC-041).
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/ ./pkg/check/ ./pkg/gate/ ./pkg/pack/engine/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      Gate Step 2 ("Code check") must STOP calling `pkg/check.Run`/`RunWith` as a baked
      enforcement engine. After this spec there is NO `gate.StepCodeCheckScopedFunc` /
      `realCodeChecker` step in the gate step list (cmd/backstop/gate.go:520). Lint, build,
      and test must run ONLY as declared `<lang>-toolchain` pack passes dispatched through
      `dispatchPackEngines` (cmd/backstop/pack_gate.go), the SAME declared-engine substrate
      the rest of the gate already uses. The replacement is WHOLESALE — the `realCodeChecker`
      → `pkg/check.Run` Step-2 path is removed, not surgically narrowed. No parallel engine
      dispatcher is introduced and `pkg/check` does not import `pkg/pack/engine` (orchestration
      stays at cmd/backstop, which already imports both). (RDQ-1, DD-1)
    supports: collapse-legacy-codecheck-into-packs:REQ-001
    follows: STD-GO-001:GO-010
  - id: REQ-002
    text: >
      The legacy `pkg/check` Step-2 GATE path must be DELETED in the SAME PR as the cutover,
      with no standing dual-run window and no "parity proven over time" exit criterion. The
      deleted surface is: the `realCodeChecker` type and its `CheckAll`/`CheckScoped`/
      `runCheck`/`runWithOpts`/`checkViolationsToGate` (cmd/backstop/gate.go) as a wired gate
      step; the `gate.StepCodeCheck*` step entry; and the `builtinToolchain` function with BOTH
      its baked go and typescript stacks (pkg/check/registry.go:59 — go's empty stack and ts's
      eslint/tsc/regex-lines stack), which are the actual zero-baked-checks violation. After
      deletion, NO baked `builtinToolchain` go or typescript stack remains and the gate's
      lint/build/test enforcement exists ONLY as `<lang>-toolchain` pack passes through
      `dispatchPackEngines`. SCOPE GUARD — internal consistency: `resolveToolchain`,
      `commandExecutor`, and `buildExecutorsForConfigErr` (pkg/check/registry.go) must be
      RETAINED (not deleted), because they have a SURVIVING production caller — the standalone
      `backstop code check` subcommand (`codeCheckCmd` → `resolveCheckRun` → `check.Run` →
      `buildExecutorsForConfigErr` → `resolveToolchain`), which this spec does NOT retire (the
      cutover targets gate Step 2, not the standalone subcommand). With `builtinToolchain`
      deleted, `resolveToolchain` is REDUCED to resolving
      DECLARED `enforcement.toolchain` entries only (no built-in stack overlay); a Go/TS project
      with no declared toolchain now yields an empty executor set from that path rather than a
      baked stack. No deleted symbol may have a surviving production caller — the deletion set
      is INTERNALLY CONSISTENT. The standalone `backstop code check` subcommand (`codeCheckCmd`,
      cmd/backstop/code_check.go — a distinct product surface from the gate, with the `--file`
      runtime-hook mode) SURVIVES this cutover: this spec retires gate Step 2, NOT the
      subcommand. Its run path (`resolveCheckRun` → `check.Run` → `buildExecutorsForConfigErr`
      → `resolveToolchain`) must keep working after the cutover, so deleting `builtinToolchain`
      must NOT strand `buildExecutorsForConfigErr` / `resolveToolchain` / `commandExecutor`
      (retained per the scope guard above), and a `backstop code check` invocation over a
      project with a DECLARED toolchain must still resolve and run its lint/build/test passes.
      (RDQ-1, DD-1)
    supports: collapse-legacy-codecheck-into-packs:REQ-002
    follows: STD-GO-001:GO-010
  - id: REQ-003
    text: >
      The deletion in REQ-002 must be GATED on a ONE-SHOT golden-equivalence assertion: a
      mandated test captures the legacy `pkg/check` engine's violation set on the backstop repo
      (a fixed set of `go build` / `go test` / `golangci-lint` outputs) as a GOLDEN FIXTURE,
      then asserts the `dispatchPackEngines` `<lang>-toolchain` path reproduces the SAME
      normalized violation set for that captured output. The fixture IS the equivalence evidence;
      no parity-gate apparatus and no standing comparison harness is built. The golden-equivalence
      test must run the REAL `<lang>-toolchain` pack passes through the UN-STUBBED
      `dispatchPackEngines` over an INSTALLED toolchain pack — NOT testdata fed to a stubbed
      dispatcher and NOT a parallel raw-exec path — spying the sandboxed-dispatch seam to assert
      reproduction. A deletion that outran the golden proof must fail a guard test rather than
      shipping green. (RDQ-1, DD-1)
    supports: collapse-legacy-codecheck-into-packs:REQ-003
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      The no-toolchain-pack baseline must be WARN-ONLY: when the project declares no
      `<lang>-toolchain` pack (and none is bridged), the gate must NOT block, must NOT force an
      opt-out, and must NOT emit a failing status — `gate.Pass` stays true and exit code stays 0.
      Enforcement is genuinely opt-in because backstop has legitimate non-enforcement postures
      (artifact-chain-only, recipe-packs-only). The warn-only state must use the existing
      non-failing `"warning"` step status (counted in `StepsWarned`, pkg/gate/result.go), NOT a
      `"fail"` and NOT a silent `"pass"`. (RDQ-2, DD-2)
    supports: collapse-legacy-codecheck-into-packs:REQ-005
    follows: STD-GO-001:GO-010
  - id: REQ-006
    text: >
      The "no enforcement ran" state (0 toolchain packs) must be a LOUDLY and DISTINCTLY
      surfaced report state — a stable, recognizable message (e.g. "enforcement: not configured
      (0 toolchain packs)") rendered on the gate's human report surface AND reflected in the
      machine-readable summary (`StepsWarned`) — and must NEVER be collapsed into a normal green
      step. Because exit 0 is invisible in CI, the loudness lives on the REPORT surface: a gate
      run with 0 toolchain packs must be visibly distinguishable in its output from a gate run
      where toolchain packs ran and passed. This is the anti-vacuous-green guardrail and is the
      single most load-bearing behavior in this spec. (RDQ-2, DD-2)
    supports: collapse-legacy-codecheck-into-packs:REQ-006
    follows: STD-GO-001:GO-010
  - id: REQ-008
    text: >
      This spec must ABSORB SPEC-034's unfinished deletion scope and GENERALIZE the cutover
      machinery beyond Go. SPEC-034's bridge (`loadBridgedToolchainPacks`, gate.go:411) and its
      TDD tests landed on `main` and are the worked example / safety net; its DELETION of the
      bespoke Step-2 path did NOT land — this spec completes it. SPEC-034 is marked SUPERSEDED
      separately; this spec must not leave SPEC-034's half-done deletion dangling and must not
      re-introduce a Go-only assumption (the generalization is REQ-004). The golden-equivalence
      harness (REQ-003) and SPEC-034's landed bridge tests (`TestBridge_*`) are the safety net
      for the deletion. (RDQ-4, DD-4)
    supports: collapse-legacy-codecheck-into-packs:REQ-008
    follows: STD-GO-001:GO-010
  - id: REQ-009
    text: >
      This spec must NOT regress the landed BUNDLE-009 traceability packs (substantiveness,
      contracts). Traceability analyzers run through their OWN dedicated gate steps via
      `dispatchPackEngines` and are EXCLUDED from the generic lint/build/test/findings dispatch
      by `excludeDedicatedStepRules` (pack_gate.go:118); the cutover must preserve that
      exclusion so substantiveness/contracts rules are not double-dispatched into the
      lint/build/test path and the traceability steps still run and still pass after the cutover.
      Traceability and code-check are separate components; the seam is coordinate-don't-subsume,
      not a merge. (RDQ-4, DD-4)
    supports: collapse-legacy-codecheck-into-packs:REQ-009
    follows: STD-GO-001:GO-010

claims:
  # REQ-001 — Step 2 stops calling pkg/check.Run; lint/build/test route only through dispatchPackEngines
  - id: CLM-001
    requirement: REQ-001
    text: The gate step list built by buildGateSteps contains no realCodeChecker / StepCodeCheckScopedFunc Step-2 entry after the cutover
    tests:
      - TestCutover_NoCodeCheckStepInGateStepList
  - id: CLM-002
    requirement: REQ-001
    text: Lint, build, and test enforcement runs through dispatchPackEngines as <lang>-toolchain pack passes, not through pkg/check.Run — a gate over a project with an installed toolchain pack dispatches its lint/build/test through the engine path
    tests:
      - TestCutover_LintBuildTestRunThroughDispatchPackEngines
  - id: CLM-003
    requirement: REQ-001
    text: pkg/check does not import pkg/pack/engine after the cutover, and no parallel engine dispatcher is introduced — the gate reuses the existing dispatchPackEngines
    tests:
      - TestCutover_NoCheckToEngineImportAndNoParallelDispatcher

  # REQ-002 — legacy Step-2 path + builtinToolchain go/ts stacks deleted in the same PR
  - id: CLM-004
    requirement: REQ-002
    text: The realCodeChecker type and its CheckAll/CheckScoped/runCheck/runWithOpts methods are deleted from cmd/backstop as a wired gate step — a grep of cmd/backstop non-test source returns zero matches for realCodeChecker
    tests:
      - TestCutover_RealCodeCheckerDeleted
  - id: CLM-005
    requirement: REQ-002
    text: The builtinToolchain function and its go stack are deleted from pkg/check/registry.go — a grep of pkg/check non-test source returns zero matches for builtinToolchain
    tests:
      - TestCutover_BuiltinToolchainGoStackDeleted
  - id: CLM-006
    requirement: REQ-002
    text: The builtinToolchain typescript stack (eslint/tsc/regex-lines) is deleted — no baked typescript lint/build/test stack remains in pkg/check non-test source
    tests:
      - TestCutover_BuiltinToolchainTypescriptStackDeleted
  - id: CLM-008
    requirement: REQ-002
    text: There is no standing dual-run window — after the cutover the gate's lint/build/test run exactly once (through dispatchPackEngines), not through both pkg/check.Run and the engine path
    tests:
      - TestCutover_NoDualRunOfLintBuildTest

  # REQ-003 — deletion gated on a one-shot golden-equivalence assertion over the real, un-stubbed dispatch
  - id: CLM-009
    requirement: REQ-003
    text: A golden fixture captures the legacy pkg/check engine's normalized violation set on the backstop repo's captured go build / go test / golangci-lint output
    tests:
      - TestGoldenEquivalence_LegacyViolationSetCaptured
  - id: CLM-010
    requirement: REQ-003
    text: The dispatchPackEngines <lang>-toolchain path reproduces the SAME normalized violation set as the golden fixture for the same captured tool output (equivalence proven)
    tests:
      - TestGoldenEquivalence_PackEnginePathReproducesGoldenSet
  - id: CLM-011
    requirement: REQ-003
    text: The golden-equivalence assertion runs the REAL toolchain-pack passes through the UN-STUBBED dispatchPackEngines over an INSTALLED toolchain pack, spying the sandboxed-dispatch seam — not testdata fed to a stubbed dispatcher and not a parallel raw-exec path
    tests:
      - TestGoldenEquivalence_RealInstalledPackThroughUnstubbedDispatch
  - id: CLM-012
    requirement: REQ-003
    text: A deletion that outran the golden proof fails a guard test — the guard asserts the golden-equivalence test exists and is green and that the legacy Step-2 symbols are absent, so a premature deletion fails the gate rather than shipping green
    tests:
      - TestGoldenEquivalence_DeletionGatedOnProvenEquivalence

  # REQ-005 — no-toolchain-pack baseline is WARN-ONLY (no block, non-failing "warning" status)
  - id: CLM-016
    requirement: REQ-005
    text: A gate run with 0 toolchain packs does not block — gate.Pass stays true and the exit code stays 0
    tests:
      - TestNoToolchainPack_DoesNotBlockGatePasses
  - id: CLM-017
    requirement: REQ-005
    text: The no-toolchain-pack state uses the non-failing "warning" step status (counted in StepsWarned), not a "fail" status
    tests:
      - TestNoToolchainPack_UsesWarningStatusNotFail
  - id: CLM-018
    requirement: REQ-005
    text: The no-toolchain-pack state is not a silent "pass" — the step is not rendered as a normal green pass; it is distinct from a toolchain-packs-ran-and-passed run
    tests:
      - TestNoToolchainPack_NotSilentPass

  # REQ-006 — "0 toolchain packs / nothing ran" is LOUDLY and DISTINCTLY surfaced; never a normal green
  - id: CLM-019
    requirement: REQ-006
    text: A gate run with 0 toolchain packs renders a stable, recognizable "enforcement not configured (0 toolchain packs)" message on the human report surface
    tests:
      - TestNoEnforcement_LoudMessageOnHumanReport
  - id: CLM-020
    requirement: REQ-006
    text: The "no enforcement ran" state is reflected in the machine-readable summary (StepsWarned), distinct from steps-passed, so a CI consumer can detect it despite exit 0
    tests:
      - TestNoEnforcement_ReflectedInMachineSummary
  - id: CLM-021
    requirement: REQ-006
    text: A gate run with 0 toolchain packs is visibly distinguishable in its report output from a gate run where toolchain packs ran and passed — the two are NOT identical green output
    tests:
      - TestNoEnforcement_DistinctFromToolchainPacksPassedRun
  - id: CLM-022
    requirement: REQ-006
    text: The "no enforcement ran" loud state never collapses into a normal green — a project with no toolchain pack can never produce output indistinguishable from a fully-enforced green pass
    tests:
      - TestNoEnforcement_NeverVacuousGreen

  # REQ-008 — absorb SPEC-034's unfinished deletion + generalize; bridge safety net retained
  - id: CLM-023
    requirement: REQ-008
    text: SPEC-034's unfinished Step-2 deletion is completed here — the realCodeChecker/pkg/check.Run Step-2 path SPEC-034's bridge landed alongside but never deleted is gone, with no Go-only assumption re-introduced
    tests:
      - TestAbsorbSpec034_Step2DeletionCompletedNotGoOnly
  - id: CLM-024
    requirement: REQ-008
    text: SPEC-034's landed bridge tests (TestBridge_*) and the golden-equivalence harness together remain the safety net — the bridge still dispatches the toolchain pack's lint/build/test passes through dispatchPackEngines after the deletion
    tests:
      - TestAbsorbSpec034_BridgeStillDispatchesToolchainPasses

  # REQ-009 — do not regress BUNDLE-009 traceability packs; preserve dedicated-step exclusion
  - id: CLM-025
    requirement: REQ-009
    text: The cutover preserves excludeDedicatedStepRules so substantiveness/contracts rules are NOT double-dispatched into the generic lint/build/test/findings path
    tests:
      - TestNoRegress_DedicatedStepRulesStillExcludedFromGenericDispatch
  - id: CLM-026
    requirement: REQ-009
    text: After the cutover the substantiveness traceability step still runs and still passes through its own dedicated gate step over its installed pack
    tests:
      - TestNoRegress_SubstantivenessStepStillRunsAndPasses
  - id: CLM-027
    requirement: REQ-009
    text: After the cutover the contracts traceability step still runs and still passes through its own dedicated gate step over its installed pack
    tests:
      - TestNoRegress_ContractsStepStillRunsAndPasses

  # Test-runner ↔ coverage seam (Sharp Edge): deleting pkg/check.Run must NOT orphan the coverage step before SPEC-041
  - id: CLM-029
    requirement: REQ-001
    text: Deleting realCodeChecker/checkViolationsToGate (where the build-pass ProjectWide scope-filter exemption `cv.Pass == check.CheckTypeBuild` is set today, gate.go:1173) does NOT silently scope-filter engine-path build breaks — a build break in an UNCHANGED file still REDs a diff-scoped gate across the cutover window, because build-pass ProjectWide is preserved transitionally on the engine path (or SPEC-040 and SPEC-041 land in lockstep). Cross-references SPEC-041 REQ-004, which replaces this transitional seam with a declared exempt_from_scope_filter property
    tests:
      - TestSeam_BuildBreakInUnchangedFileStillRedsDiffScopedGate

contracts:
  - file: cmd/backstop/gate.go
    provides:
      - name: buildGateSteps
        kind: function
        signature: "func buildGateSteps(projectRoot string, scope ...*gate.GateScope) []gate.StepFunc"
        notes: >
          The realCodeChecker / gate.StepCodeCheckScopedFunc Step-2 entry is REMOVED from the
          step list (REQ-001/CLM-001). Lint/build/test now run ONLY via dispatchPackEngines over
          the <lang>-toolchain pack passes (the bridge + pack_engines step). The
          realCodeChecker type and its runCheck/runWithOpts/CheckAll/CheckScoped methods are
          DELETED (REQ-002/CLM-004). No new dispatcher is introduced; dispatchPackEngines is
          reused (REQ-001/CLM-003). cmd/backstop already imports both pkg/check and
          pkg/pack/engine, so no new pkg/check->pkg/pack/engine import is added. The
          no-toolchain-pack WARN-ONLY loud state (REQ-005/REQ-006) is produced here as a
          non-failing "warning" StepResult with a stable "enforcement not configured" message.
      # loadBridgedToolchainPacks contract entry REMOVED (ISSUE-038 contract-drift
      # reconciliation): the language-derived toolchain bridge was DELETED in the
      # BUNDLE-011/012 native-toolchain cutover and formally retired by SPEC-046. The
      # symbol no longer exists (grep: zero hits); there is no replacement symbol.
    consumes:
      - source: cmd/backstop/pack_gate.go
        name: dispatchPackEngines
        kind: function
      - source: pkg/gate/result.go
        name: StepResult
        kind: type
      - source: pkg/pack/manifest.go
        name: ParseManifestFile
        kind: function
  - file: cmd/backstop/pack_gate.go
    provides:
      - name: dispatchPackEngines
        kind: function
        signature: "func dispatchPackEngines(packs []*pack.Manifest, packDir, projectRoot string, scope *gate.GateScope, runner check.CommandRunner) ([]gate.Violation, error)"
        notes: >
          REUSED unchanged in signature as the single dispatch substrate for lint/build/test
          (REQ-001). Consumed here as the sole enforcement path for the toolchain-pack passes.
          excludeDedicatedStepRules (pack_gate.go:118) is preserved so substantiveness/contracts
          rules are not double-dispatched into the generic lint/build/test/findings path
          (REQ-009/CLM-025).
      - name: excludeDedicatedStepRules
        kind: function
        signature: "func excludeDedicatedStepRules(packs []*pack.Manifest) []*pack.Manifest"
        notes: >
          Preserved across the cutover (REQ-009): keeps BUNDLE-009 traceability rules
          (substantiveness/contracts) out of the generic findings dispatch so they run only
          through their own dedicated gate steps.
      - name: runFindingsEngine
        kind: function
        signature: "func runFindingsEngine(manifest *pack.Manifest, packRoot, projectRoot string, scope *gate.GateScope, binding engine.EngineBinding, rules []pack.Rule, runner check.CommandRunner) ([]gate.Violation, error)"
        notes: >
          The build-exemption scope-filter SEAM (Sharp Edge 2), REALIGNED to the mechanism that
          actually shipped (align-predating-artifacts; SPEC-048 REQ-004b). SPEC-040 originally
          declared a transitional `dispatchBuildViolationProjectWide` variable here — a symbol
          that NEVER existed. SPEC-041 REQ-004 SUPERSEDED that transitional approach with a
          DECLARED per-binding `EngineBinding.ExemptFromScopeFilter` field: runFindingsEngine
          (cmd/backstop/pack_gate.go) reads `exempt := binding.ExemptFromScopeFilter` and stamps
          it onto every produced `gate.Violation.ProjectWide` (`ProjectWide: exempt`). ProjectWide
          is consumed by pkg/gate/scope.go's filterViolations to keep an exempt engine's
          UNCHANGED-file violation out of the diff-scope filter, so a build break in an unchanged
          file still REDs a diff-scoped gate (Sharp Edge 2 / CLM-029) — go-build declares
          ExemptFromScopeFilter=true; golangci/go-test/findings leave it false. This is the
          honest, shipped bridge; the CLM-029 seam SPEC-040 mandated transitionally is now carried
          by this declared field rather than the phantom variable the original contract named.
    consumes:
      - source: pkg/gate/scope.go
        name: ProjectWide
        kind: variable
  - file: cmd/backstop/shared_testrun.go
    provides:
      - name: newSharedTestRunner
        kind: function
        signature: "func newSharedTestRunner(dir string) *sharedTestRunner"
        notes: >
          The test-runner ↔ coverage SEAM (Sharp Edge). Today the shared `go test ./...` runner
          feeds BOTH Step 2's test FAILs AND the baked coverage step. This spec deletes Step 2's
          consumption of it but must NOT orphan the coverage step before SPEC-041 (Seed 3)
          migrates coverage. The shared runner SURVIVES this cutover as the transitional coverage
          feed: it keeps feeding `buildCoverageStep` so coverage still runs and passes
          (REQ-001/CLM-028), until SPEC-041 deletes step_coverage.go and re-implements coverage
          over the toolchain test pass. The toolchain test pass (through dispatchPackEngines)
          becomes the established test-runner seam SPEC-041 migrates coverage onto.
    consumes: []
---

# SPEC-040: Toolchain Pack Cutover

## Overview

This is the **keystone cutover** of BUNDLE-011 (Seed 2): the point at which gate
"Step 2" stops being a noun for a baked enforcement engine (`pkg/check.Run`) and
becomes a verb — "run the declared `<lang>-toolchain` packs through the engine
substrate the rest of the gate already uses." It is the last baked enforcement
engine standing on the thin-executor finish line.

**Grounding on current `main` (verified 2026-06-24) — a premise correction.** The
bundle and seed brief describe Step 2 as "still runs lint/build/test via
`pkg/check.Run`." That is only HALF true on `main`, and the difference matters for
scoping:

- The landed half of (now-superseded) **SPEC-034** — the bridge
  `loadBridgedToolchainPacks` (gate.go:411) — **already routes the Go native
  lint/build/test passes through `dispatchPackEngines`**. The go `builtinToolchain`
  stack (registry.go:59) already returns **EMPTY** pass entries; for Go,
  `pkg/check.Run` no longer constructs lint/build/test executors.
- What did **NOT** land: (a) the `realCodeChecker` → `pkg/check.Run` step is **still
  wired as gate Step 2** (gate.go:520, `StepCodeCheckScopedFunc`), still live for the
  **typescript** `builtinToolchain` stack (eslint/tsc/regex-lines, registry.go:65-92) —
  the baked go/ts stacks in `builtinToolchain` are the zero-baked-checks violation; and
  (b) there is **no loud report state** for "0 toolchain packs / nothing ran" — a
  project with no toolchain pack silently produces a normal green.

So this spec's wholesale-replace + delete (REQ-001/002) targets the **Step-2 gate path
itself and the baked `builtinToolchain` go AND ts stacks** — finishing SPEC-034's
deletion (REQ-008) and generalizing it beyond Go (REQ-004). It does NOT delete
`resolveToolchain` / `commandExecutor` / `buildExecutorsForConfigErr`: those are RETAINED
in reduced declared-only form because the standalone `backstop code check` subcommand
(`codeCheckCmd` → `check.Run` → `buildExecutorsForConfigErr` → `resolveToolchain`)
survives this cutover and still reaches them — a cross-seam SPEC-041's review surfaced;
the delete set must be internally consistent (no deleted symbol with a surviving caller).
A
`go-toolchain` pack **exists as a fixture** under `cmd/backstop/testdata/`
(`nativeToolchainPackName()` = `backstop/go-toolchain` is the on-disk name the bridge
resolves from `.backstop/packs/<name>`) and is leveraged as the worked example, not
re-authored — **but it is NOT present in backstop-core's live dogfood install** (see the
next grounding flag), so the resolve-from-disk path only finds it where a project has
actually installed it.

**A second grounding flag (load-bearing for REQ-005/006).** backstop-core's OWN
dogfood install (`.backstop/packs/backstop/`) currently has `go-standards` but **NOT**
`go-toolchain`. So when backstop gates itself today, `loadBridgedToolchainPacks` finds
**no on-disk toolchain pack** and the bespoke path is what's nominally live — yet the
go stack is empty. This is exactly the **vacuous-green hazard** REQ-006 exists to
close: after the cutover deletes the bespoke path, a project (including backstop-core
until it installs `go-toolchain`) with **0 toolchain packs** must NOT silently go
green; it must hit the loud, distinct WARN-ONLY "enforcement not configured" state.

**The reused loud mechanism.** SPEC-036 already established a non-failing `"warning"`
step status, counted in `GateResult.StepsWarned` and rendered in the human summary
(pkg/gate/result.go) precisely so a loud-but-passing advisory cannot vanish from a
green run. REQ-005/006 reuse this exact mechanism for the no-toolchain-pack state —
no new status vocabulary is invented.

**Scope fence.** Out of scope and explicitly fenced to siblings: the dead
standards-manifest reader + non-Go semgrep catch-all deletions are **SPEC-039 (Seed 1,
a prerequisite that shrinks this surface)**; coverage eradication, `step_coverage.go`
deletion, the **declared** re-expression of the build-pass project-wide-scope exemption
(`cv.Pass == check.CheckTypeBuild` in `checkViolationsToGate` → a declared
`exempt_from_scope_filter` engine property), and the CheckType-consumer catalog are
**SPEC-041 (Seed 3, which depends on the test-runner seam this spec establishes)**.
**Caveat — this spec deletes `checkViolationsToGate` (where that exemption is set
today), so it must TRANSITIONALLY PRESERVE the build-pass `ProjectWide` exemption on the
engine path (or land in lockstep with SPEC-041) so a build break in an unchanged file is
not silently scope-filtered in the cutover window** — a transitional seam (Sharp Edge 2 /
CLM-029), NOT the permanent declared re-expression, which stays SPEC-041's.

**Verification Level:** Integration (80% coverage).
**Source Bundle:** BUNDLE-011 (collapse-legacy-codecheck-into-packs) — Seed 2, owning
REQ-001…REQ-006, REQ-008, REQ-009 (RDQ-1, RDQ-2, RDQ-4).

## Requirements

Requirements and claims are defined in frontmatter. The summary tables below must
match the requirement text exactly.

### What runs Step 2's enforcement, before vs after (REQ-001, REQ-002, REQ-004)

| Pass | Before (`main`) | After this spec |
|---|---|---|
| Go lint/build/test | already bridged through `dispatchPackEngines` (SPEC-034 landed half); go `builtinToolchain` already empty | through `dispatchPackEngines` over `go-toolchain` pack (unchanged in effect; bespoke step removed) |
| TypeScript lint/build/test | baked `builtinToolchain` ts stack (eslint/tsc/regex-lines) via `pkg/check.Run` Step 2 | through `dispatchPackEngines` over a `typescript-toolchain` pack; baked ts stack DELETED |
| any other language | requires declared `enforcement.toolchain` via `pkg/check.Run` | through `dispatchPackEngines` over its `<lang>-toolchain` pack |
| `realCodeChecker` → `pkg/check.Run` Step 2 | wired gate step (gate.go:520) | DELETED — no `gate.StepCodeCheck*` step in the list |
| `builtinToolchain` (baked go/ts stacks) | live in `pkg/check` (the zero-baked violation) | DELETED |
| `resolveToolchain` / `commandExecutor` / `buildExecutorsForConfigErr` | construction machinery; built-in + declared | RETAINED, reduced to DECLARED-only (serves the surviving `code check` subcommand) |
| `backstop code check` subcommand (`codeCheckCmd`) | live product surface (`--file` hook mode) | SURVIVES — unchanged beyond the reduced `resolveToolchain` |

### No-toolchain-pack baseline behavior (REQ-005, REQ-006)

The single most important behavior not to get wrong. The three states are distinct
and must never collapse:

| State | Blocks? | `gate.Pass` | Step status | Report surface |
|---|---|---|---|---|
| toolchain packs ran, passed | no | true | `pass` | normal green |
| toolchain packs ran, found violations | yes | false | `fail` | violations listed |
| **0 toolchain packs (nothing ran)** | **no** (warn-only) | **true** | **`warning`** (counted in `StepsWarned`) | **LOUD, distinct: "enforcement not configured (0 toolchain packs)" — never a normal green** |

The bottom row is the anti-vacuous-green guardrail (REQ-006): warn-only does NOT
block, but "nothing ran" must be impossible to mistake for "everything passed."
Loudness lives on the report surface because exit 0 is invisible in CI.

## Implementation

The work is one coherent PR (no standing dual-run — RDQ-1/DD-1), but it decomposes
into ordered stages the planner can map tasks to. Stage 4's deletion is GATED on
stage 3's golden-equivalence test existing and passing.

### 1. Generalize the toolchain-pack bridge beyond Go (REQ-004, REQ-008)

`loadBridgedToolchainPacks` (gate.go:411) is generalized: the early
`if language != "" && language != "go"` short-circuit is replaced so it resolves the
`<lang>-toolchain` pack for the project's **declared language** generically (the
on-disk pack name is derived from the language, not hardcoded to `backstop/go-toolchain`
alone). It still loads from `.backstop/packs/<name>`, still dedupes against a declared
toolchain pack, and a missing pack directory yields no bridged packs — which now feeds
the no-toolchain-pack state (stage 5) instead of a baked fallback.

### 2. Wholesale-replace Step 2: route lint/build/test only through dispatchPackEngines (REQ-001)

The `realCodeChecker` / `gate.StepCodeCheckScopedFunc` entry is removed from the step
list `buildGateSteps` produces. Lint/build/test enforcement now runs **only** via the
`pack_engines` dispatch over the bridged + declared `<lang>-toolchain` packs (the
existing dispatch already runs `bridged + packs` minus dedicated-step rules). No new
dispatcher is introduced and no `pkg/check` → `pkg/pack/engine` import is added
(orchestration stays at cmd/backstop).

### 3. The one-shot golden-equivalence assertion (REQ-003) — GATES stage 4

A mandated test captures the legacy `pkg/check` engine's normalized violation set on
the backstop repo (fixed captured `go build` / `go test` / `golangci-lint` outputs) as
a **golden fixture**, then asserts the `dispatchPackEngines` `<lang>-toolchain` path
reproduces the same normalized set for the same captured output. This test runs the
**real** toolchain-pack passes through the **un-stubbed** `dispatchPackEngines` over an
**installed** toolchain pack, spying the sandboxed-dispatch seam — NOT testdata fed to
a stubbed dispatcher and NOT a parallel raw-exec path (heeding the recurring
pack-provisioning integration gap). A guard test (CLM-012) asserts this equivalence
test exists and is green AND the legacy Step-2 symbols are absent, so a deletion that
outran the proof fails the gate.

### 4. Delete the legacy Step-2 gate path + baked builtinToolchain stacks (REQ-002) — GATED on stage 3

Only once stage 3's golden-equivalence test exists and passes: delete the
`realCodeChecker` type and its `CheckAll`/`CheckScoped`/`runCheck`/`runWithOpts`/
`checkViolationsToGate` (as a wired gate step); and the `builtinToolchain` function with
BOTH its baked go and typescript stacks (the zero-baked violation). After deletion, a grep
of the relevant non-test source for `realCodeChecker` (cmd/backstop) and
`builtinToolchain` (pkg/check) returns zero.

**Internal-consistency scope guard (REQ-002).** `resolveToolchain`, `commandExecutor`,
and `buildExecutorsForConfigErr` are RETAINED, not deleted — they have a surviving
production caller, the standalone `backstop code check` subcommand, which this spec does
NOT retire. `resolveToolchain` is REDUCED to resolving DECLARED `enforcement.toolchain`
entries only (the built-in overlay is gone), so a Go/TS project with no declared toolchain
yields an empty executor set from that path rather than a baked stack. A guard test
(CLM-031) asserts no deleted symbol has a surviving production caller; CLM-030 asserts the
`code check` subcommand still resolves and runs lint/build/test over a DECLARED toolchain
after the cutover. Deleting `resolveToolchain` wholesale (as a naive reading of REQ-002
might) would break the subcommand at implementation time — the cross-seam SPEC-041's
review surfaced.

### 5. The no-toolchain-pack WARN-ONLY loud state (REQ-005, REQ-006)

When the project declares no `<lang>-toolchain` pack and none is bridged, the gate
emits a **non-failing `"warning"`** StepResult carrying a stable, recognizable message
("enforcement not configured (0 toolchain packs)"), counted in `GateResult.StepsWarned`
and rendered on the human report surface. `gate.Pass` stays true, exit code stays 0,
and the output is visibly distinct from a toolchain-packs-ran-and-passed run. This
reuses the SPEC-036 `"warning"` mechanism, inventing no new status vocabulary.

### 6. Preserve the traceability + test-runner seams (REQ-009, test-runner sharp edge)

`excludeDedicatedStepRules` (pack_gate.go:118) is preserved so BUNDLE-009's
substantiveness/contracts rules are not double-dispatched into the generic
lint/build/test/findings path; the substantiveness and contracts dedicated gate steps
still run and pass. Separately, the **shared `go test ./...` runner**
(`newSharedTestRunner`, shared_testrun.go) SURVIVES this cutover as the transitional
coverage feed: deleting Step 2's consumption of it must NOT orphan the still-baked
coverage step (`buildCoverageStep`). Coverage keeps receiving its whole-module feed and
keeps running/passing until SPEC-041 (Seed 3) deletes `step_coverage.go` and
re-implements coverage over the toolchain test pass. The toolchain test pass (through
`dispatchPackEngines`) is the established test-runner seam SPEC-041 migrates onto.

A SECOND transitional seam of the same shape: deleting `checkViolationsToGate` removes
where the build-pass `ProjectWide` scope-filter exemption is set today (`cv.Pass ==
check.CheckTypeBuild`, gate.go:1173; consumed by `pkg/gate/scope.go:194`). To avoid an
under-broad vacuous-green regression in the cutover window, the engine dispatch path must
set `gate.Violation.ProjectWide=true` for build-pass violations transitionally (or SPEC-040
and SPEC-041 land in lockstep), so a build break in an unchanged file still REDs a
diff-scoped gate (CLM-029). SPEC-041 REQ-004 replaces this with a declared
`exempt_from_scope_filter` property; this seam is the bridge until then.

## Verification

Verification is defined in frontmatter. Integration-level testing at 80% coverage
across `cmd/backstop` (the step-list cutover, the generalized bridge, the no-pack loud
state), `pkg/check` (the `builtinToolchain` deletion and the reduced declared-only
`resolveToolchain` / `buildExecutorsForConfigErr` that keep the `code check` subcommand
working), `pkg/gate` (the `"warning"` status / `StepsWarned` summary, coverage step
not orphaned), and `pkg/pack/engine` (the dispatch substrate consumed).

The golden-equivalence claims (CLM-009…CLM-012) run the **real** `<lang>-toolchain`
pack passes through the **un-stubbed** `dispatchPackEngines` over an **installed**
toolchain pack — spying the sandboxed-dispatch seam to observe reproduction — never
testdata fed to a stubbed dispatcher and never a parallel raw-exec path. This is the
hard mandate that closes the recurring pack-provisioning integration gap: the golden
fixture is the deletion's only safety net, so it must exercise the production path, not
a stub.

The no-enforcement claims (CLM-016…CLM-022) assert the warn-only state does not block
(`gate.Pass` true, exit 0), uses the non-failing `"warning"` status, is reflected in
`StepsWarned`, renders the stable loud message on the human report surface, and is
visibly distinct from a toolchain-packs-passed run — the explicit anti-vacuous-green
checks. The no-regress claims (CLM-025…CLM-027) assert the dedicated-step exclusion
holds and the substantiveness/contracts steps still run and pass. The seam claims assert
the two transitional seams hold across the cutover: CLM-028 asserts the coverage step is
not orphaned by the Step-2 deletion, and CLM-029 asserts a build break in an UNCHANGED
file still REDs a diff-scoped gate (build-pass `ProjectWide` preserved transitionally, not
silently scope-filtered) — both tested through the production gate path, not a stub.

## Sharp Edges

1. **The test-runner ↔ coverage coupling — deleting `pkg/check.Run` must NOT orphan
   the coverage step (the cardinal seam of this cutover).** Today one shared
   `go test ./...` runner (`newSharedTestRunner`, shared_testrun.go) runs ONCE and feeds
   BOTH Step 2's test FAILs AND the still-baked coverage step (`buildCoverageStep`).
   This is exactly where a wholesale Step-2 deletion can silently drop a gate step:
   remove `realCodeChecker`'s consumption of the shared runner and the coverage step
   could be left with no feed, going vacuously green or erroring. This spec ESTABLISHES
   the toolchain test pass (through `dispatchPackEngines`) as the new test seam but does
   NOT migrate coverage (that is SPEC-041 / Seed 3). The transitional contract is
   explicit and mandated (CLM-028): **the shared runner survives this cutover as the
   coverage feed, and coverage still runs and passes**, in lockstep until SPEC-041 lands.
   A plan that deletes the shared runner here, or leaves coverage unfed, is wrong.

2. **The build-pass scope-filter exemption seam — deleting `checkViolationsToGate` must
   NOT silently scope-filter build breaks (the second cardinal seam).** Today the
   build-pass project-wide exemption is set in `checkViolationsToGate` (`cv.Pass ==
   check.CheckTypeBuild` → `gate.Violation.ProjectWide=true`, gate.go:1173) and consumed by
   `pkg/gate/scope.go:194` so a build break in an UNCHANGED file is NOT diff-scope-filtered
   away (a compile error anywhere REDs the gate even if that file wasn't in the diff). This
   cutover DELETES `realCodeChecker`/`checkViolationsToGate`. The engine path does NOT set
   `ProjectWide` for build violations today (verified — only `checkViolationsToGate` does),
   so without action, engine-path build breaks would carry `ProjectWide=false` and be
   silently scope-filtered in a diff-scoped gate — an **under-broad vacuous green** in the
   SPEC-040→SPEC-041 window. The transitional contract is mandated (CLM-029): the engine
   dispatch path preserves `ProjectWide=true` for build-pass violations transitionally, OR
   SPEC-040 and SPEC-041 land in lockstep. This MIRRORS the coverage seam (Sharp Edge 1).
   SPEC-041 REQ-004 replaces it permanently with a declared `exempt_from_scope_filter`
   property; this seam is only the bridge. A plan that deletes `checkViolationsToGate`
   without preserving build-`ProjectWide` (and without lockstep) is wrong.

3. **Vacuous green when 0 toolchain packs (REQ-006) — the philosophical crux.** After
   the bespoke path is deleted, a project with no `<lang>-toolchain` pack runs NO
   lint/build/test. Warn-only means it doesn't block — but if that state renders as a
   normal green `pass`, "nothing ran" is indistinguishable from "everything passed,"
   which is the silent/vacuous green the enforcement philosophy exists to defeat. The
   guardrail is the non-failing `"warning"` status + a stable loud message on the report
   surface + `StepsWarned` in the machine summary. backstop-core's own dogfood install
   has no `go-toolchain` pack today, so this state is LIVE on the first gate run after
   the cutover — it is not hypothetical.

4. **Golden fixture gamed by a stubbed dispatcher (the pack-provisioning integration
   gap).** The deletion's only safety net is the one-shot golden-equivalence assertion.
   If it runs over testdata fed to a STUBBED `dispatchPackEngines` (or a parallel
   raw-exec path), it "passes" without ever proving the real installed-pack engine path
   reproduces the legacy violations — the exact gap that bit SPEC-035/SPEC-037. The
   mandate (CLM-011) is a REAL installed toolchain pack through the UN-STUBBED dispatch,
   spying the sandboxed-dispatch seam.

5. **Deletion outrunning the proof.** A wholesale delete (REQ-002) is tempting to land
   ahead of the golden fixture. If it does, there is a window with the bespoke path gone
   and equivalence unproven — a blind cutover. The guard test (CLM-012) asserts the
   golden test exists, is green, AND the legacy symbols are absent, so a premature
   deletion fails the gate rather than shipping.

6. **Scope creep into the PERMANENT build-exemption re-expression / coverage eradication
   (SPEC-041's territory).** The build-pass exemption's permanent re-expression as a declared
   `exempt_from_scope_filter` property, and eradicating `step_coverage.go`, are SPEC-041
   (Seed 3). This spec must NOT build the declared property and must NOT delete
   `step_coverage.go` — it only TRANSITIONALLY preserves the build-`ProjectWide` exemption
   (Sharp Edge 2 / CLM-029) and keeps coverage fed (Sharp Edge 1 / CLM-028). The distinction
   is load-bearing: transitional preservation here is in scope (else vacuous green in the
   window); the permanent declared mechanism is SPEC-041's. Building the declared property
   here, or deleting coverage here, crosses the fence and collides with SPEC-041.

7. **Generalization that secretly stays Go-only.** REQ-004/REQ-008 require the cutover
   machinery to be language-agnostic. A lazy implementation could delete the bespoke
   path but leave `loadBridgedToolchainPacks`'s `language == "go"` short-circuit, so the
   bridge only ever resolves `go-toolchain` — re-baking a Go assumption while claiming
   generalization. CLM-014 asserts a non-Go language resolves its own `<lang>-toolchain`
   pack.

8. **Double-dispatching traceability rules into the generic path (REQ-009).** The
   cutover runs `bridged + packs` through the generic findings dispatch. If
   `excludeDedicatedStepRules` is dropped or bypassed, BUNDLE-009's
   substantiveness/contracts rules would scan context-free in the generic lint/build/
   test/findings path and emit garbage findings, regressing the landed traceability
   packs. CLM-025 asserts the exclusion survives.

9. **TypeScript stack deletion with no `typescript-toolchain` pack yet.** Deleting the
   baked ts `builtinToolchain` stack (REQ-002/CLM-006) removes eslint/tsc enforcement
   for TS projects that have not adopted a `typescript-toolchain` pack. That is the
   intended end state (enforcement is opt-in via packs), but it means TS projects land
   in the no-toolchain-pack WARN-ONLY loud state (REQ-006) rather than silently losing
   enforcement — the loud state is what prevents this deletion from being a silent
   regression. Authoring the `typescript-toolchain` pack itself is a pack-authoring
   follow-up (paired with the trusted-tool allowlist, ISSUE-027), not core code here.

## Review Questions

1. After the cutover, does `buildGateSteps` still include a `realCodeChecker` /
   `gate.StepCodeCheckScopedFunc` Step-2 entry? It must NOT — confirm Step 2 is removed
   and lint/build/test run only through `dispatchPackEngines` (CLM-001/CLM-002).

2. Does a grep of `cmd/backstop` non-test source for `realCodeChecker` (as a wired gate
   step) and of `pkg/check` non-test source for `builtinToolchain` return zero? Deleted,
   not merely bypassed (CLM-004…CLM-006). Conversely, are `resolveToolchain`,
   `commandExecutor`, and `buildExecutorsForConfigErr` still PRESENT (retained, reduced to
   declared-only) — NOT deleted (CLM-007)? Deleting them would strand the surviving
   subcommand. And does the standalone `backstop code check` subcommand still work after the
   cutover — an invocation over a project with a DECLARED toolchain still resolving and
   running its lint/build/test passes (CLM-030), with no deleted symbol left with a surviving
   production caller (CLM-031)? This internal-consistency cross-seam is what SPEC-041's review
   surfaced.

3. Does the golden-equivalence test run the REAL `<lang>-toolchain` pack passes through
   the UN-STUBBED `dispatchPackEngines` over an INSTALLED toolchain pack (spying the
   sandboxed-dispatch seam), or over testdata fed to a stubbed dispatcher / a parallel
   raw-exec path? Confirm the production path (CLM-011) — the stub would game the only
   safety net.

4. Is the deletion (REQ-002) genuinely gated on the golden-equivalence test existing and
   passing (CLM-012), or could the delete land ahead of the proof?

5. With 0 toolchain packs, does the gate stay `Pass == true` / exit 0 (warn-only, no
   block), AND emit the loud, distinct "enforcement not configured (0 toolchain packs)"
   state via the non-failing `"warning"` status + `StepsWarned` + a report-surface
   message — never a normal green pass (CLM-016…CLM-022)? This is the load-bearing one.

6. Is the no-toolchain-pack run visibly distinguishable in its output from a
   toolchain-packs-ran-and-passed run (CLM-021)? If the two outputs are identical green,
   the guardrail failed.

7. Does deleting `pkg/check.Run`'s Step-2 consumption leave the still-baked coverage step
   without a feed? Confirm the shared `go test ./...` runner survives as the transitional
   coverage feed and the coverage step still runs and passes (CLM-028) — no orphaned
   coverage before SPEC-041.

8. Does deleting `checkViolationsToGate` (where build-pass `ProjectWide` is set today) leave
   engine-path build breaks scope-filterable? Confirm a build break in an UNCHANGED file
   still REDs a diff-scoped gate (CLM-029) — i.e. build-`ProjectWide` is preserved
   transitionally on the engine path, OR SPEC-040+SPEC-041 land in lockstep. Cross-check
   SPEC-041 REQ-004 is the permanent replacement, not duplicated here.

9. Is `loadBridgedToolchainPacks` genuinely language-agnostic after the change, or does a
   `language == "go"` short-circuit remain so only `go-toolchain` ever resolves
   (CLM-014)? Confirm a non-Go language resolves its own `<lang>-toolchain` pack.

10. Is `excludeDedicatedStepRules` preserved so substantiveness/contracts rules are not
    double-dispatched into the generic lint/build/test/findings path, and do those
    traceability steps still run and pass (CLM-025…CLM-027)?

11. Did this PR build the PERMANENT declared `exempt_from_scope_filter` property or delete
    `step_coverage.go`? It must NOT — those are SPEC-041's. The only build-exemption work in
    scope here is the TRANSITIONAL `ProjectWide` preservation (Sharp Edge 2 / CLM-029);
    confirm the permanent re-expression and coverage eradication stayed fenced to SPEC-041.

## References

- **BUNDLE-011** (collapse-legacy-codecheck-into-packs) — Seed 2, the keystone cutover.
  This spec owns REQ-001…REQ-006, REQ-008, REQ-009 (RDQ-1, RDQ-2, RDQ-4).
- **SPEC-039** (codecheck-deadcode-prelude) — Seed 1, the PREREQUISITE: deletes the dead
  standards-manifest reader and the already-no-op non-Go semgrep catch-all, shrinking the
  surface this cutover rewrites. Lands first.
- **SPEC-041** (coverage-reimpl-checktype-catalog) — Seed 3, the DEPENDENT: eradicates
  `pkg/gate/step_coverage.go`, re-implements coverage language-agnostic over the toolchain
  test pass this spec establishes, re-expresses the build-pass project-wide-scope exemption
  (`cv.Pass == check.CheckTypeBuild`) as a declared property, and produces the
  CheckType-consumer catalog. Depends on the test-runner seam this spec establishes (Sharp
  Edge 1).
- **SPEC-034** (native-toolchain-engine-cutover) — SUPERSEDED/absorbed (REQ-008). Its
  bridge (`loadBridgedToolchainPacks`, gate.go:411) and TDD tests (`TestBridge_*`) landed
  on `main` and are the worked example + safety net; its DELETION of the bespoke Step-2
  path did not — this spec finishes it and generalizes it beyond Go.
- **SPEC-036** (traceability-fail-loud) — the source of the non-failing `"warning"` step
  status + `GateResult.StepsWarned` loud-but-passing mechanism (pkg/gate/result.go) that
  REQ-005/REQ-006 reuse for the no-toolchain-pack state.
- **BUNDLE-009** (stack-aware-traceability) — LANDED; substantiveness/contracts are
  installed packs running through their own dedicated gate steps. A SEPARATE,
  coordinate-don't-subsume target (REQ-009): this spec must not regress its packs.
- **ISSUE-027** — trusted-tool allowlist growth pairs per-language with the toolchain
  packs (e.g. `typescript-toolchain`); allowlisting a tool is inert until a pack declares
  an engine using it.
- [[project_pack_provisioning_integration_gap]] — the recurring gap the golden-equivalence
  mandate (REQ-003/CLM-011) closes: real-end-to-end over an installed pack through the
  un-stubbed dispatch, never testdata + a stubbed dispatcher.
- [[feedback_loud_not_blocking]] — governs the no-toolchain-pack baseline: warn-with-guidance
  for un-adopted capability, block only defects + broken promises; the enemy is vacuous green.
- [[project_thin_executor_engine_packs]] — the thesis this cutover completes: backstop knows
  no engine, runs declared commands, speaks only SARIF; thin on knowledge, firm on enforcement.
- Code (verified 2026-06-24, `main`): cmd/backstop/gate.go (`buildGateSteps` step list with
  `StepCodeCheckScopedFunc` at :520; `realCodeChecker` + `runCheck`/`runWithOpts`;
  `loadBridgedToolchainPacks` :411; `checkViolationsToGate` build-pass exemption — SPEC-041's,
  untouched here); cmd/backstop/pack_gate.go (`dispatchPackEngines`, `excludeDedicatedStepRules`
  :118); cmd/backstop/shared_testrun.go (`newSharedTestRunner` — the test-runner ↔ coverage
  seam); pkg/check/registry.go (`builtinToolchain` :59 go-empty + ts stacks — DELETED;
  `buildExecutorsForConfigErr` :198/:223, `resolveToolchain` :251, `commandExecutor` :103 —
  RETAINED reduced-to-declared-only, serving the surviving `code check` subcommand);
  cmd/backstop/code_check.go (`codeCheckCmd` → `resolveCheckRun` → `check.Run` →
  `buildExecutorsForConfigErr` — the surviving subcommand path); pkg/gate/result.go
  (`StepResult.Status`, `"warning"`, `StepsWarned`);
  `.backstop/packs/backstop/` (has `go-standards`, NOT `go-toolchain` — the live vacuous-green
  hazard REQ-006 closes).

## Version History

- **2.0.0** (2026-07-05) — Retired the requirement + claims whose subject ISSUE-018 (authorized
  thin-executor eradication) deleted outright. Removed REQ-004 (one `<lang>-toolchain` pack per
  language / language-agnostic bridge machinery) with its 3 claims (CLM-013/014/015), and the
  now-stale code-check-survival claims CLM-007/030/031 (REQ-002 — asserted `resolveToolchain` /
  `buildExecutorsForConfigErr` / the `backstop code check` subcommand SURVIVE the cutover; ISSUE-018
  later deleted them) and CLM-028 (REQ-001 seam — the coverage-orphan guard). Their mandated
  `TestToolchainPack_*` / `TestCutover_ResolveToolchainRetainedDeclaredOnly` /
  `TestCutover_CodeCheckSubcommandSurvives*` / `TestCutover_NoDeletedSymbolHasSurvivingCaller` /
  `TestSeam_CoverageStepNotOrphaned*` functions were deleted with the code. REQ-001 and REQ-002
  survive (they retain live claims — CLM-001/002/003/029 and CLM-004/005/006/008); all other
  requirements/claims are unchanged. Requirements with all claims removed are deleted alongside
  them to satisfy `spec/requirement-uncovered` (REQ-004 only); `spec/claim-tests-empty` forbids
  leaving emptied claims. Recorded openly per align-predating-artifacts.
- **1.3.0** (2026-07-05) — Retired the stale `pkg/check/registry.go` provides `resolveToolchain`
  and `buildExecutorsForConfigErr` and the `cmd/backstop/code_check.go` provides `codeCheckCmd`.
  These entries pinned the standalone `backstop code check` subcommand + its reduced toolchain
  resolvers as SURVIVORS of this cutover, but ISSUE-018 (authorized thin-executor eradication)
  subsequently deleted the subcommand and `registry.go` entirely — the symbols are gone from the
  tree, so their present-signature promises were stale reds under `contract_signature`. The whole
  `registry.go` and `code_check.go` contract blocks were removed (deleted files). Contract-only
  realignment (align-predating-artifacts); no requirement, claim, or design change.
- **1.2.0** (2026-07-03) — Status → `implemented`. BUNDLE-011 Seed 2 keystone toolchain-pack
  cutover shipped and committed; parent bundle BUNDLE-011 delivered. Status-only transition, no
  requirement, claim, contract, or prose change.
- **1.0.0** (2026-06-24) — Initial spec: the BUNDLE-011 Seed 2 keystone cutover (retire gate
  Step 2 `pkg/check.Run`, route lint/build/test through `dispatchPackEngines` toolchain packs,
  no-toolchain-pack WARN-ONLY loud state, absorb SPEC-034's unfinished deletion).
- **1.1.0** (2026-06-30) — Phantom-contract realignment (align-predating-artifacts; SPEC-048
  REQ-004b). The `cmd/backstop/pack_gate.go` provides entry named a transitional
  `dispatchBuildViolationProjectWide` variable that never existed; SPEC-041 REQ-004 superseded
  that approach with the shipped `EngineBinding.ExemptFromScopeFilter` field stamped per-violation
  onto `gate.Violation.ProjectWide` in `runFindingsEngine`. Realigned the entry to name that real
  function symbol and describe the shipped mechanism so `contract_signature` reports an honest
  contract — not a baseline/waiver. No requirement, claim, or scope change.
