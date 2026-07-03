---
title: "Findings-Engine Self-Targeting And stdout_artifact Dispatch Fix"
number: SPEC-048
created: "2026-06-30"
status: implemented
schema_version: spec/v1
spec_version: 1.1.0

implementation:
  summary: >
    Closes a SILENT VACUOUS-GREEN gap in the findings-engine dispatch
    (`cmd/backstop/pack_gate.go` `runFindingsEngine`) discovered by EXECUTING
    SPEC-047 REQ-005 — the first real `backstop gate` over the INSTALLED
    `backstop/bun-toolchain` pack on a Bun/TypeScript project. The five BUNDLE-012
    specs (043–047) were all proven via testdata fixtures with the engine dispatcher
    STUBBED (a canned-stdout runner), so the REAL installed-pack end-to-end dispatch
    path was never exercised — and it was broken in TWO independent ways, each of
    which made the gate read GREEN while enforcing NOTHING (the pack-provisioning
    integration-gap pattern). DEFECT 1 — a project-wide findings engine that declares
    NO `project_target` fell to the branch that appended `projectRoot` as a scan
    target; but a self-targeting toolchain tool (`tsc --noEmit <dir>` treats the path
    as a FILE and IGNORES tsconfig.json; `bun test` self-discovers) then typechecks/
    tests NOTHING, so a seeded type error does not RED the gate. DEFECT 2 — the
    pack-declared `stdout_artifact` field (naming the FILE an engine writes its real
    machine-readable output to, with stdout being human-only noise) was honored ONLY
    in `runCoverageEngine`, NEVER in `runFindingsEngine`; so `bun test
    --reporter=junit --reporter-outfile=<file>` fed the convert its SUMMARY stdout
    (no `<testcase>`), and a seeded failing test does not RED the gate. This spec
    reproduces the PROVEN, hand-verified fix (`.claude/acceptance-notes/PROVEN-dispatch-fix.diff`)
    via TDD: (1) a project-wide findings engine with an EMPTY `project_target`
    self-targets — the ProjectTarget is appended only when non-empty, otherwise
    NOTHING is bolted on; (2) `runFindingsEngine` honors `stdout_artifact` exactly as
    `runCoverageEngine` does — reads that FILE (relative to projectRoot) as the
    convert/shape-guard payload, fail-loud (not a silent stdout fallback) when the
    declared artifact is absent; and (3) THE LOAD-BEARING PROOF — a REAL end-to-end
    gate over an INSTALLED-pack fixture driven by a REAL CommandRunner (not the
    stubbed dispatcher), which fails against the PRE-FIX dispatch and is the test that
    would have caught both bugs. It also aligns two pieces of pre-existing debt this
    change drags into diff scope: (a) the `%v`-wrapped read error in
    `runCoverageEngine`'s stdout_artifact-missing branch → `%w`; (b) the STALE
    SPEC-040 contract `dispatchBuildViolationProjectWide` (a symbol that never
    existed — SPEC-041 REQ-004 shipped `EngineBinding.ExemptFromScopeFilter` →
    `gate.Violation.ProjectWide` instead), realigned so `contract_signature` is honest
    rather than re-baselined. The fix stays tool/language-blind: `project_target` and
    `stdout_artifact` are pack DATA — no `tsc`/`bun`/`go` literal enters the dispatch.
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      In `runFindingsEngine`, a project-wide findings engine (binding.ScopeKind ==
      engine.ScopeKindProjectWide) MUST shape its OWN invocation target and MUST NOT
      have `projectRoot` bolted on. When the binding declares a NON-EMPTY
      `ProjectTarget` (e.g. `go build ./...`, `golangci-lint run ./...` with
      `ProjectTarget: "./..."`), the dispatch MUST append that ProjectTarget
      unchanged (preserving the SPEC-034 file-mode go-test package-selector edge:
      under a file-mode scope the native go-test engine keeps receiving its changed
      file's package selector, every other project-wide pass keeps its ProjectTarget).
      When the binding declares an EMPTY `ProjectTarget`, the dispatch MUST append
      NOTHING — the engine self-targets (a self-targeting toolchain tool reads its own
      config: `tsc --noEmit` reads tsconfig.json, `bun test` self-discovers). It is
      PROHIBITED to append `projectRoot` (or any scan target) to a project-wide,
      empty-`ProjectTarget` engine: doing so is DEFECT 1 — `tsc --noEmit <projectRoot>`
      treats the path as a file argument and silently typechecks nothing (vacuous
      green). This change MUST NOT alter the non-project-wide (file-args) branch: a
      rule-fed / config-file findings engine still receives the gate's in-scope changed
      files (or `projectRoot` under a nil / GateScopeModeAll whole-repo scope). The
      go-toolchain engines all declare `ProjectTarget: "./..."` and are therefore
      UNAFFECTED — only empty-target project-wide findings engines change behavior.
    supports: language-neutral-consumer-ts-toolchain:REQ-009
  - id: REQ-002
    text: >
      `runFindingsEngine` MUST honor the binding's `StdoutArtifact` field exactly as
      `runCoverageEngine` already does. When `binding.StdoutArtifact != ""`, the
      dispatch MUST read that FILE (resolved via filepath.Join(projectRoot,
      filepath.FromSlash(binding.StdoutArtifact)) — the run's working directory is
      projectRoot) and feed ITS bytes as the `payload` to BOTH the strict-SARIF shape
      guard (`requireLintSarifShape`) AND the convert step — NOT the engine's raw
      stdout. When `binding.StdoutArtifact == ""`, the payload MUST remain the engine's
      stdout (unchanged default, backward compatible). A DECLARED-but-MISSING artifact
      MUST fail loud — returning an error naming the pack, the engine command, the
      declared artifact, and the resolved path — and MUST NOT silently fall back to
      the noise stdout: silently reading stdout is DEFECT 2 (`bun test --reporter=junit
      --reporter-outfile=<file>` writes failures to the FILE while its stdout is a
      human summary with no `<testcase>`, so a failing suite reads as zero findings —
      vacuous green). `StdoutArtifact` is pack DATA (the filename); reading it
      introduces NO tool/language literal into the dispatch.
    supports: language-neutral-consumer-ts-toolchain:REQ-009
  - id: REQ-003
    text: >
      There MUST be a REAL end-to-end gate test over an INSTALLED-pack fixture that
      exercises the ACTUAL dispatch path (`runFindingsEngine` via
      `dispatchPackEngines`) driven by a REAL `check.CommandRunner` — NOT a stubbed
      dispatcher and NOT a canned-stdout runner that returns bytes directly as stdout
      (which masks BOTH defects, as SPEC-043..047's fixtures did). The fixture MUST
      declare a project-wide findings engine with an EMPTY `project_target` and a
      declared `stdout_artifact`, executed by a COMMITTED, deterministic fake-engine
      script (POSIX shell — NO real `bun`/`tsc`/`oxlint`/`go` dependency in
      backstop-core's Go CI) that emulates the self-targeting-toolchain I/O contract:
      it writes its real machine-readable findings to the declared `stdout_artifact`
      FILE, prints only human-summary NOISE to stdout, and self-targets (it does not
      depend on — and MUST NOT silently succeed because of — a bolted-on `projectRoot`
      path arg). With a SEEDED finding present, `backstop gate` over the fixture MUST
      go RED carrying that finding (not vacuous green). With the fixture CLEAN (no
      seeded finding), the gate MUST go GREEN. This single test exercises BOTH fixes on
      the real dispatch path and MUST FAIL against the PRE-FIX dispatch (it is the test
      that would have caught both bugs) — it MUST be written failing-first (TDD).
    supports: language-neutral-consumer-ts-toolchain:REQ-009
  - id: REQ-004
    text: >
      Two pieces of pre-existing debt that enter diff scope when `pack_gate.go` is
      edited MUST be aligned (not worked around, not silenced by re-baseline). (a)
      `runCoverageEngine`'s stdout_artifact-missing branch currently formats its
      underlying read error with `%v` (`... not produced (read %s: %v)`), breaking the
      error chain; it MUST wrap with `%w` so `errors.Is` / `errors.Unwrap` reaches the
      underlying `os.ReadFile` error, matching the fail-loud findings-path branch this
      spec adds. (b) SPEC-040's `contracts` block declares
      `dispatchBuildViolationProjectWide` (kind: variable) — a named symbol that NEVER
      existed: SPEC-041 REQ-004 SUPERSEDED that transitional approach with the shipped
      `EngineBinding.ExemptFromScopeFilter` field mapping per-violation to
      `gate.Violation.ProjectWide`. That stale contract entry MUST be realigned so it
      references the real shipped mechanism and `contract_signature` reports an HONEST
      contract for SPEC-040. Any edit to the SPEC-040 artifact MUST be routed through
      the spec-author agent, NOT hand-edited. It is PROHIBITED to make
      `contract_signature` green by adding the phantom symbol to a baseline/waiver.
    supports: language-neutral-consumer-ts-toolchain:REQ-009
    follows: STD-GO-001:GO-011

claims:
  # REQ-001 — project-wide arg-shaping matrix (self-target when empty; append when declared; boundaries preserved)
  - id: CLM-001
    requirement: REQ-001
    text: >
      A project-wide findings engine with an EMPTY ProjectTarget appends NOTHING — the
      dispatched command args carry no projectRoot and no scan target, so the engine
      self-targets (the DEFECT-1 fix; captured via a recording runner over the real
      runFindingsEngine)
    tests:
      - TestRunFindingsEngine_ProjectWideEmptyTargetSelfTargetsNoRootAppended
  - id: CLM-002
    requirement: REQ-001
    text: >
      A project-wide findings engine with a NON-EMPTY ProjectTarget (e.g. "./...")
      appends that ProjectTarget unchanged — the go-toolchain engines are UNAFFECTED
      by the self-target change (the allowed cell)
    tests:
      - TestRunFindingsEngine_ProjectWideWithTargetAppendsProjectTarget
  - id: CLM-003
    requirement: REQ-001
    text: >
      A non-project-wide (file-args) rule-fed engine still appends the gate's in-scope
      changed files (or projectRoot under a nil / GateScopeModeAll whole-repo scope) —
      the empty-target self-target change did NOT leak into the file-args branch
    tests:
      - TestRunFindingsEngine_FileArgsScopeAppendsScopeFilesUnchanged
  - id: CLM-004
    requirement: REQ-001
    text: >
      PRESERVATION — under a file-mode scope the native go-test project-wide engine
      (ProjectTarget "./...") still receives its changed file's PACKAGE selector via
      fileModeTestTarget, not projectRoot; the restructured project-wide branch keeps
      the SPEC-034 file-mode package-scoping edge intact
    tests:
      - TestRunFindingsEngine_FileModeGoTestPackageScopingPreserved
  # REQ-002 — stdout_artifact matrix (present→convert; present→shape-guard; missing→fail-loud; empty→stdout)
  - id: CLM-005
    requirement: REQ-002
    text: >
      With StdoutArtifact set and the file PRESENT, the FILE's bytes (not the engine's
      noise stdout) are fed to the convert step — a findings-bearing artifact file over
      a finding-free stdout yields the file's findings (the DEFECT-2 fix)
    tests:
      - TestRunFindingsEngine_StdoutArtifactFileFeedsConvertNotStdout
  - id: CLM-006
    requirement: REQ-002
    text: >
      With StdoutArtifact set, the file PRESENT, and NO convert declared, the FILE's
      bytes (not stdout) are what the strict-SARIF shape guard (requireLintSarifShape)
      validates — the payload selection applies to the shape-guard path too, not only
      the convert path
    tests:
      - TestRunFindingsEngine_StdoutArtifactFileFeedsStrictSarifShapeGuard
  - id: CLM-007
    requirement: REQ-002
    text: >
      With StdoutArtifact set but the file MISSING, runFindingsEngine fails loud —
      returns an error naming the pack, the engine command, the declared artifact, and
      the resolved path — and does NOT silently fall back to reading stdout
    tests:
      - TestRunFindingsEngine_StdoutArtifactMissingFailsLoud
  - id: CLM-008
    requirement: REQ-002
    text: >
      With StdoutArtifact EMPTY, the payload remains the engine's stdout unchanged
      (backward compatible — the default findings path is undisturbed)
    tests:
      - TestRunFindingsEngine_NoStdoutArtifactUsesStdoutPayload
  # REQ-003 — the load-bearing real end-to-end proof (fails pre-fix; catches both bugs)
  - id: CLM-009
    requirement: REQ-003
    text: >
      END-TO-END — a gate over the installed-pack fixture, driven by a REAL
      CommandRunner executing the committed fake engine (project-wide, empty
      project_target, declared stdout_artifact), with a SEEDED finding written to the
      artifact file goes RED carrying that finding (not vacuous green)
    tests:
      - TestBunDispatchE2E_SeededFindingInStdoutArtifactRedsGate
  - id: CLM-010
    requirement: REQ-003
    text: >
      END-TO-END — the SAME fixture with NO seeded finding (clean) goes GREEN over the
      real dispatch, proving the RED in CLM-009 is the finding and not a spurious
      dispatch failure
    tests:
      - TestBunDispatchE2E_CleanFixtureGreenGate
  - id: CLM-011
    requirement: REQ-003
    text: >
      The e2e uses NO stubbed dispatcher / NO canned-stdout runner and NO real
      bun/tsc/oxlint/go — the fake engine self-targets (does not rely on a bolted-on
      projectRoot arg) AND writes findings to its stdout_artifact FILE while stdout is
      human noise, so this one test exercises BOTH fixes on the real path and FAILS
      against the pre-fix dispatch
    tests:
      - TestBunDispatchE2E_RealRunnerFakeEngineSelfTargetsAndReadsArtifactCatchesBothBugs
  # REQ-004 — surfaced-debt alignment (error wrap + phantom contract retire)
  - id: CLM-012
    requirement: REQ-004
    text: >
      runCoverageEngine's stdout_artifact-missing branch wraps the underlying read
      error with %w — errors.Is / errors.Unwrap reaches the os.ReadFile error, matching
      the findings-path fail-loud branch
    tests:
      - TestRunCoverageEngine_StdoutArtifactMissingWrapsReadError
  - id: CLM-013
    requirement: REQ-004
    text: >
      SPEC-040's contracts no longer declare the phantom
      `dispatchBuildViolationProjectWide` (kind: variable) symbol; the realigned
      contract references the shipped EngineBinding.ExemptFromScopeFilter →
      gate.Violation.ProjectWide mechanism (SPEC-041 REQ-004) so contract_signature is
      honest — not silenced by baseline/waiver
    tests:
      - TestSpec040Contract_PhantomBuildViolationSymbolRetired

contracts:
  - file: cmd/backstop/pack_gate.go
    provides:
      - name: runFindingsEngine
        kind: function
        signature: "func runFindingsEngine(manifest *pack.Manifest, packRoot, projectRoot string, scope *gate.GateScope, binding engine.EngineBinding, rules []pack.Rule, runner check.CommandRunner) ([]gate.Violation, error)"
        notes: >
          MODIFIED (REQ-001/REQ-002), external signature UNCHANGED. (1) The
          ScopeKindProjectWide branch appends binding.ProjectTarget ONLY when non-empty
          (preserving the fileModeTestTarget package-selector edge); an empty
          ProjectTarget appends nothing (self-target — DEFECT-1 fix). (2) A new
          payload-selection step mirrors runCoverageEngine: when
          binding.StdoutArtifact != "" the artifact FILE (filepath.Join(projectRoot,
          filepath.FromSlash(binding.StdoutArtifact))) is read as the payload and fed to
          BOTH requireLintSarifShape and the convert; a missing artifact fail-louds with
          %w; empty StdoutArtifact keeps stdout as the payload (DEFECT-2 fix).
      - name: runCoverageEngine
        kind: function
        signature: "func runCoverageEngine(manifest *pack.Manifest, packRoot, projectRoot string, binding engine.EngineBinding, rules []pack.Rule, runner check.CommandRunner) ([]check.CoverageRecord, error)"
        notes: >
          MODIFIED (REQ-004a), external signature UNCHANGED — the stdout_artifact-missing
          error changes from %v to %w so the underlying os.ReadFile error is unwrappable,
          matching the findings path.
    consumes:
      - source: pkg/pack/engine
        name: EngineBinding
        kind: type
      - source: pkg/gate
        name: GateScope
        kind: type
      - source: pkg/gate
        name: Violation
        kind: type
      - source: pkg/check
        name: CommandRunner
        kind: type
  - file: cmd/backstop/testdata/bun-toolchain/.backstop/packs/backstop/bun-toolchain/pack.yml
    provides: []
    consumes:
      - source: pkg/pack/engine
        name: EngineBinding
        kind: type
---

# SPEC-048: Findings-Engine Self-Targeting And stdout_artifact Dispatch Fix

## Overview

This spec closes a **silent vacuous-green** gap in the findings-engine dispatch
(`cmd/backstop/pack_gate.go` `runFindingsEngine`) that was discovered by **executing**
SPEC-047 REQ-005 — the first real `backstop gate` over the **installed**
`backstop/bun-toolchain` pack on a Bun/TypeScript project. The five BUNDLE-012 specs
(043–047) were all proven through **testdata fixtures with the engine dispatcher
STUBBED** (a canned-stdout runner), so the **real installed-pack end-to-end dispatch
path was never exercised** — and it was broken in **two independent ways**, each of
which made the gate read GREEN while enforcing **nothing**. This is exactly the
recurring pack-provisioning integration-gap pattern ([[project_pack_provisioning_integration_gap]]):
unit-tested-with-a-stub, never proven over the real installed-pack path.

A **proven, hand-verified** fix is captured at
`.claude/acceptance-notes/PROVEN-dispatch-fix.diff` — the reference implementation. The
working tree has been reverted to clean so the implementer reproduces it via **TDD
(failing test first)**.

### The two proven defects (both in `runFindingsEngine`)

| # | Defect | Why it reads GREEN | Fix |
| --- | --- | --- | --- |
| **1** | A project-wide findings engine with **no** `project_target` fell to the branch that appended `projectRoot` as a scan target. | `tsc --noEmit <projectRoot>` treats the path as a **file** and IGNORES `tsconfig.json` → typechecks nothing → a seeded type error does not RED. | When `ScopeKind == project-wide`: append `ProjectTarget` only when **non-empty**; when empty, append **nothing** (the engine self-targets). |
| **2** | `stdout_artifact` (a pack-declared field naming the **file** an engine writes its real output to; stdout is human noise) was honored only in `runCoverageEngine`, **never** in `runFindingsEngine`. | `bun test --reporter=junit --reporter-outfile=<file>` writes failures to the **file**; its stdout is a summary with no `<testcase>` → convert sees no failures → a failing suite does not RED. | Mirror the coverage path: when `StdoutArtifact != ""`, read that **file** as the `payload` for BOTH the shape guard and the convert; fail-loud (not silent stdout fallback) on a missing artifact. |

Both are **thin-executor-safe**: `project_target` and `stdout_artifact` are pack **DATA**
— no `tsc`/`bun`/`go` literal enters the dispatch ([[feedback_zero_baked_checks]]).

**In scope:** the `runFindingsEngine` dispatch fix (self-targeting + `stdout_artifact`);
the real end-to-end proof over an installed-pack fixture driven by a real
`CommandRunner` + a committed fake engine; and the two surfaced-debt alignments (the
`%w` fix in `runCoverageEngine`, the SPEC-040 phantom-contract retire).

**Out of scope:** the `backstop/bun-toolchain` pack **DATA** fixes (the awk converts,
`prettier --list-different`) already landed in the external pack repo (commit `0219632`)
— NOT this spec. The **format-as-lint warn-vs-block severity policy** (prettier findings
are SARIF `level:warning` but `pack_engines` reds on any violation regardless of
severity) is BUNDLE-012 REQ-007 / Phase-6 per-pack enforcement — explicitly out of scope
here (see Sharp Edges).

## Requirements

Requirements are enumerated in the `requirements:` frontmatter (REQ-001 … REQ-004). Each
in-scope requirement has claims in the `claims:` frontmatter. Summary:

| Spec REQ | Commits to |
| --- | --- |
| REQ-001 | A project-wide findings engine **self-targets** when `ProjectTarget` is empty (append nothing); when non-empty, append the `ProjectTarget` unchanged (go-toolchain engines unaffected; file-mode go-test package-scoping edge preserved). Appending `projectRoot` to a project-wide empty-target engine is **prohibited** (DEFECT 1). The file-args branch is unchanged. |
| REQ-002 | `runFindingsEngine` **honors `stdout_artifact`**: when set, read the file as the payload for BOTH the shape guard and the convert; when empty, keep stdout (backward compatible); a declared-but-missing artifact **fail-louds** (no silent stdout fallback — DEFECT 2). |
| REQ-003 | A **real** end-to-end gate over an installed-pack fixture, driven by a **real** `CommandRunner` + a committed fake engine (project-wide, empty `project_target`, declared `stdout_artifact`): seeded finding → **RED**, clean → **GREEN**. Must **fail against the pre-fix dispatch** (TDD, written failing-first) — the test that would have caught both bugs. |
| REQ-004 | Align surfaced debt: (a) `runCoverageEngine`'s stdout_artifact-missing error `%v` → `%w`; (b) retire the phantom SPEC-040 contract `dispatchBuildViolationProjectWide` (realign to the shipped `ExemptFromScopeFilter` → `Violation.ProjectWide` mechanism), so `contract_signature` is honest — not re-baselined. |

### The project-wide arg-shaping matrix (REQ-001)

`runFindingsEngine` shapes the engine's invocation target by `ScopeKind` × `ProjectTarget`.
Every cell is claim-covered:

| ScopeKind | ProjectTarget | Scope | Dispatch appends | Verdict | Claim |
| --- | --- | --- | --- | --- | --- |
| project-wide | **empty** | any | **NOTHING** (engine self-targets) | fix (DEFECT 1) | CLM-001 |
| project-wide | `./...` | non-file-mode | `./...` (unchanged) | unaffected (go-toolchain) | CLM-002 |
| project-wide | `./...` | **file-mode** go-test | changed file's package selector | preserved (SPEC-034) | CLM-004 |
| file-args | (n/a) | scoped | in-scope changed files (or `projectRoot` under nil/all) | unchanged | CLM-003 |

The **prohibited** cell is *project-wide + empty ProjectTarget → append `projectRoot`* —
that is DEFECT 1 (CLM-001 asserts nothing is appended).

### The `stdout_artifact` payload-selection matrix (REQ-002)

| `StdoutArtifact` | Artifact file | Convert declared? | Payload fed downstream | Verdict | Claim |
| --- | --- | --- | --- | --- | --- |
| set | present | yes | the **file** bytes → convert | fix (DEFECT 2) | CLM-005 |
| set | present | no (shape guard) | the **file** bytes → `requireLintSarifShape` | fix (shape-guard path) | CLM-006 |
| set | **missing** | — | — (**fail-loud** error) | fix (no silent fallback) | CLM-007 |
| empty | — | — | the engine's **stdout** (unchanged) | backward compatible | CLM-008 |

The `payload` mirrors `runCoverageEngine` and feeds **both** the strict-SARIF shape guard
and the convert (not only one) — CLM-005 covers the convert path, CLM-006 the shape-guard
path.

## Implementation

Target package: **`cmd/backstop`** (`pack_gate.go` dispatch + the tests + the installed-pack
fixture DATA). The fix reproduces `.claude/acceptance-notes/PROVEN-dispatch-fix.diff`.
Processing steps the planner must map tasks to:

1. **Self-targeting arg-shaping (REQ-001).** In `runFindingsEngine`, change the
   project-wide branch condition from `ScopeKind == ProjectWide && ProjectTarget != ""`
   to `ScopeKind == ProjectWide`, and inside it append the `ProjectTarget` only when
   non-empty (keeping the `fileModeTestTarget` nested edge for the native go-test
   engine). An empty `ProjectTarget` appends nothing. The `else` (file-args) branch is
   untouched.

2. **`stdout_artifact` payload selection (REQ-002).** After `runner.RunStdout`, add a
   `payload := stdout` selection that, when `binding.StdoutArtifact != ""`, reads
   `filepath.Join(projectRoot, filepath.FromSlash(binding.StdoutArtifact))` and returns a
   fail-loud `%w`-wrapped error naming the pack + engine command + artifact + path if the
   file is absent. Feed `payload` (not `stdout`) to `requireLintSarifShape` and to the
   convert (`resolveSandboxedRunStdout()(convertPath, nil, packRoot, payload)`), and set
   `sarifBytes := payload` for the no-convert path. This mirrors the block already in
   `runCoverageEngine`.

3. **The real installed-pack e2e proof (REQ-003).** Add a fixture declaring a project-wide
   findings engine with an **empty** `project_target` and a declared `stdout_artifact`,
   whose command is a **committed POSIX fake-engine script** that (i) writes its findings
   to the `stdout_artifact` file, (ii) prints only human noise to stdout, and (iii)
   self-targets (does not depend on a bolted-on `projectRoot` arg). Drive `backstop gate`
   over it with a **real** `check.CommandRunner` (NOT the canned-stdout `fixtureRunner`
   used by SPEC-043..047). Seeded variant → RED; clean variant → GREEN. Exact wiring of
   the fake as the resolved executable (a temp-dir PATH shim, an absolute command path, or
   a test-constructed manifest) is the planner's call — the constraint is a real runner
   over a real dispatch that FAILS pre-fix. The existing fixture at
   `cmd/backstop/testdata/bun-toolchain/` (verified 2026-06-30: `backstop.yml` + gitignored
   `.backstop/packs/backstop/bun-toolchain/{pack.yml,scripts/…}` + `src/*.ts` +
   `coverage/lcov.info`) is coverage-shaped and stub-driven; it must be extended (or a
   sibling fixture added) into a **findings** engine exercised over the **real** runner.

4. **Error-wrap alignment (REQ-004a).** In `runCoverageEngine`'s stdout_artifact-missing
   branch, change `(read %s: %v)` to a `%w`-wrapped form so the `os.ReadFile` error is
   unwrappable — matching the new findings-path branch.

5. **Phantom-contract retire (REQ-004b).** Realign SPEC-040's `contracts` entry
   `dispatchBuildViolationProjectWide` (kind: variable, a symbol that never existed) to the
   shipped `EngineBinding.ExemptFromScopeFilter` → `gate.Violation.ProjectWide` mechanism
   (SPEC-041 REQ-004), routing the SPEC-040 edit through the **spec-author agent** (not a
   hand-edit). `contract_signature` must then report an honest contract — it must NOT be
   made green by a baseline/waiver ([[feedback_align_predating_artifacts]]).

## Verification

- **Level:** `integration` (threshold 80). The load-bearing proof (REQ-003) is a
  cross-package executed gate over a real installed-pack fixture through the real dispatch —
  not an isolated unit. This honors the integration-gap lesson
  ([[feedback_integration_gap]] / [[project_pack_provisioning_integration_gap]]): the bug
  was invisible precisely because every prior test stubbed the dispatcher.
- **Command:** `go test ./cmd/backstop/ -race -coverprofile=cover.out`.
- **Mandated tests:** every test named in the `claims[]` `tests:` fields. The load-bearing
  one is `TestBunDispatchE2E_RealRunnerFakeEngineSelfTargetsAndReadsArtifactCatchesBothBugs`
  (CLM-011) — it MUST be authored **failing-first** and MUST fail against the pre-fix
  dispatch; a version that passes against pre-fix code proves nothing.
- **CLM-012** (`%w`) is verified by `errors.Is` / `errors.Unwrap` reaching the injected
  read error. **CLM-013** (phantom contract) is verified by asserting SPEC-040's contract
  no longer declares `dispatchBuildViolationProjectWide` and by `contract_signature` going
  clean for SPEC-040 (no missing-symbol broken promise).

## Sharp Edges

- **The stub is the whole reason both bugs shipped.** A canned-stdout runner (the
  `fixtureRunner{byCmd:…}` SPEC-043..047 used) returns bytes directly as stdout — it never
  writes a `stdout_artifact` file and never observes appended args, so it masks BOTH
  defects. REQ-003's proof is worthless if it reuses that stub: it MUST drive a **real**
  `CommandRunner` executing a real (fake) engine. A reviewer should reject any REQ-003 test
  that passes against the pre-fix dispatch.
- **`tsc --noEmit <dir>` is the trap, and it is silent.** Passing a directory to `tsc
  --noEmit` does not error — it silently treats the path as a file glob and typechecks
  nothing, so the gate is GREEN. The fix is "append nothing," and the only way to prove it
  is a self-targeting engine that reads its own config and would be SILENCED by a bolted-on
  path. The fake engine must model that: it must NOT succeed merely because a `projectRoot`
  arg is present.
- **`stdout_artifact` missing must fail LOUD, never fall back to stdout.** A silent fallback
  to stdout re-introduces DEFECT 2 the moment an engine's artifact write fails — the gate
  would read the noise summary and green a failing suite. The missing-artifact branch is a
  broken run, not a finding-free pass.
- **Format-as-lint currently BLOCKS — that is a DIFFERENT bug, out of scope.** `prettier`
  findings are SARIF `level:warning`, but `pack_engines` reds on ANY violation regardless of
  severity, so format debt currently blocks the gate. The warn-vs-block severity policy is
  BUNDLE-012 **REQ-007 / Phase-6 per-pack enforcement**, NOT this spec. Do not "fix" it here
  by dropping warning-severity findings — that would silence real errors too. Forward
  pointer only.
- **`runFindingsEngine` and `runCoverageEngine` now carry parallel payload-selection blocks
  — keep them mirrored.** The two functions independently select `payload` from
  `stdout`/`StdoutArtifact`. They must stay behaviorally identical (fail-loud on missing,
  `%w`-wrapped, projectRoot-relative). A future edit to one without the other re-opens the
  asymmetry this spec closes; REQ-004a exists precisely to erase the last `%v`/`%w`
  divergence between them.
- **The SPEC-040 phantom contract must be RETIRED, not re-baselined.** Adding
  `dispatchBuildViolationProjectWide` to a `contract_signature` baseline/waiver would launder
  a symbol that never existed into a "known" state — the opposite of honest. The only correct
  move is to realign the contract to the real `ExemptFromScopeFilter` → `Violation.ProjectWide`
  mechanism, via the spec-author agent.
- **Fixture DATA vs backstop-core scope.** The `backstop/bun-toolchain` pack DATA (converts,
  `prettier --list-different`) already landed in the external pack repo (commit `0219632`);
  the in-repo `.backstop/packs` copy is gitignored ([[packs_always_external]]). This spec
  owns only the backstop-core dispatch, the e2e proof, and the debt alignment — do not
  re-author pack DATA here.

## Review Questions

These probe risks not fully pinned by the claims; the impl-reviewer should check each against
the diff.

- Does the project-wide branch append the `ProjectTarget` **only** when non-empty, and
  **nothing** when empty — and is there NO path by which `projectRoot` reaches a project-wide
  empty-target engine's args? (REQ-001 / CLM-001, the DEFECT-1 fix.)
- Are the go-toolchain engines (`ProjectTarget: "./..."`) demonstrably UNAFFECTED, and is the
  file-mode go-test package-selector edge preserved through the branch restructure? (REQ-001 /
  CLM-002/CLM-004.)
- Does `runFindingsEngine` read the `stdout_artifact` file as the `payload` and feed it to
  **both** `requireLintSarifShape` and the convert — not only one of them? (REQ-002 /
  CLM-005/CLM-006.)
- Does a declared-but-missing `stdout_artifact` fail loud (naming pack + engine + artifact +
  path, `%w`-wrapped) with **no** silent fallback to stdout? (REQ-002 / CLM-007.)
- Is the REQ-003 e2e driven by a **real** `CommandRunner` executing a committed fake engine
  (self-targeting + writing to `stdout_artifact`, stdout = noise), and does it **fail against
  the pre-fix dispatch** — not a stubbed dispatcher, not a canned-stdout runner, no real
  `bun`/`tsc`/`oxlint`/`go` in Go CI? (REQ-003 / CLM-009/CLM-010/CLM-011.)
- Does `runCoverageEngine`'s missing-artifact error now wrap with `%w` (unwrappable to the
  `os.ReadFile` error)? (REQ-004a / CLM-012.)
- Has SPEC-040's phantom `dispatchBuildViolationProjectWide` contract been realigned to the
  real `ExemptFromScopeFilter` → `Violation.ProjectWide` mechanism (via the spec-author
  agent), with `contract_signature` honest and NOT baselined/waived? (REQ-004b / CLM-013.)
- Does the dispatch stay tool/language-blind — is there NO `tsc`/`bun`/`go` literal in the
  new arg-shaping or payload-selection code (only the pack-DATA `project_target` /
  `stdout_artifact` fields)? (Thin-executor first principle.)

## References

- BUNDLE-012 (`language-neutral-consumer-ts-toolchain`) — parent bundle (at `defined`); this
  spec closes the REQ-009 two-surface-proof gap the SPEC-047 REQ-005 acceptance exposed.
- SPEC-047 (`bun-toolchain-pack-and-proof`) — the spec whose REQ-005 acceptance run surfaced
  both defects; its fixtures stubbed the dispatcher, hiding the real installed-pack path.
- SPEC-034 (`native-toolchain-engine-cutover`) — the file-mode go-test package-scoping edge
  (`fileModeTestTarget`) preserved by CLM-004; REQ-003/REQ-005 crash-vs-findings + strict-SARIF
  guards this payload feeds.
- SPEC-040 (`toolchain-pack-cutover`) — the source of the phantom `dispatchBuildViolationProjectWide`
  contract retired by REQ-004b.
- SPEC-041 (`coverage-reimpl-checktype-catalog`) — REQ-004 shipped the real
  `EngineBinding.ExemptFromScopeFilter` → `gate.Violation.ProjectWide` mechanism the SPEC-040
  contract is realigned to.
- SPEC-042 (`coverage-production-engine`) — `runCoverageEngine`, whose `stdout_artifact` +
  payload-selection block is the mirror this spec brings to `runFindingsEngine` (and whose `%v`
  REQ-004a corrects).
- `.claude/acceptance-notes/PROVEN-dispatch-fix.diff` — the hand-verified reference
  implementation this spec reproduces via TDD.
- [[project_pack_provisioning_integration_gap]] / [[feedback_integration_gap]] — the recurring
  stub-hides-the-real-path pattern this spec is a direct instance of; the mandate for a real
  end-to-end-over-installed-pack test that fails-loud.
- [[feedback_zero_baked_checks]] — the thin-executor invariant: `project_target` /
  `stdout_artifact` are pack DATA; no tool/language literal enters the dispatch.
- [[feedback_align_predating_artifacts]] — REQ-004: align stale artifacts (the `%v` and the
  phantom contract), never work around or re-baseline them.
- [[feedback_loud_not_blocking]] — a silent vacuous green is the worst failure mode; the fix
  and its proof make the gate loud on real defects.
- Code (verified 2026-06-30, branch `bundle/011-codecheck-cutover`):
  `cmd/backstop/pack_gate.go` `runFindingsEngine` (lines ~529–661) and `runCoverageEngine`
  (lines ~374–438); `pkg/pack/engine/binding.go` `EngineBinding.{ScopeKind,ProjectTarget,StdoutArtifact}`;
  `cmd/backstop/testdata/bun-toolchain/` (the fixture to extend); `specs/SPEC-040-*.spec.md`
  (the phantom contract at its `contracts:` block).

## Version History

- **1.1.0** (2026-07-03) — Status → `implemented`; code shipped and committed after the
  spec/plan/impl cycle each reviewed to PASS.
- **1.0.0** (2026-06-30) — Initial spec authored to close the SPEC-047 REQ-005 dispatch gap
  (self-targeting + `stdout_artifact` in `runFindingsEngine`), with the real
  end-to-end-over-installed-pack proof and the two surfaced-debt alignments.
