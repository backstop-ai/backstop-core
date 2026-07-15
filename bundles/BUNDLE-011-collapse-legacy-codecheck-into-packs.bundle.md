---
title: "Collapse the Legacy `pkg/check` Engine into Pack-Declared Toolchain Packs"
number: BUNDLE-011
created: "2026-06-21"
schema_version: bundle/v2

bundle:
  name: collapse-legacy-codecheck-into-packs
  version: "0.5.0"
  created: "2026-06-21"
  updated: "2026-06-28"
  category: infrastructure

status:
  maturity: delivered
  note: >
    DELIVERED for the CUTOVER (2026-06-28). All four seeds shipped and committed:
    SPEC-039 (dead-code prelude — Seed 1), SPEC-040 (toolchain-pack cutover keystone —
    Seed 2: the baked `realCodeChecker`/`builtinToolchain` Step 2 is GONE from the gate;
    lint/build/test now run ONLY via `dispatchPackEngines` over the bridged
    `<lang>-toolchain` pack), SPEC-041 (coverage re-implemented as a producer/consumer
    model — Seed 3), and SPEC-042 (the go-toolchain coverage PRODUCER engine — Seed 4).
    The gate now runs language-neutral dispatch for lint/build/test plus a generic
    coverage producer/consumer.
    CARRY-FORWARD (explicitly NOT delivered here — no "fully language-neutral" promise is
    made): the cutover did NOT fully de-Go the gate CONSUMER side. The `backstop/self`
    pack mechanically flags THREE remaining live Go-specific consumer sites, currently
    grandfathered in the gate baseline: `pkg/gate/step_coverage.go` (the `.go`-only
    measurable-path filter, ~L239), `pkg/gate/step_testverify.go` (the `_test.go`
    test-file walk, ~L254), and `cmd/backstop/gate.go` (`goFilePackageMatchesTarget`'s
    Go-package assumption, ~L908). De-Go-ing those consumers + delivering a
    `typescript-toolchain` pack is carried forward to a NEW forthcoming bundle — it is
    NOT claimed as done by BUNDLE-011.

problem:
  summary: >
    `backstop gate` still runs TWO enforcement engines side by side. Step 2 ("Code
    check") is `pkg/check.Run` (`realCodeChecker` → `check.Run`, wired at
    cmd/backstop/gate.go:332 and :649), a baked-in second engine that runs ALONGSIDE
    the declared pack-engine dispatch (`dispatchPackEngines`, cmd/backstop/pack_gate.go).
    The legacy engine bakes policy and tool-knowledge directly into the binary: a
    hardcoded `CheckType` enum (lint=golangci-lint / build=go build / test=go test /
    semgrep — pkg/check/manifest.go:14-23); a hardcoded `languageExtensions` map
    (go→.go, typescript→.ts/.tsx — :75-78); `routeFileDefaults`, which sends
    `.go/.ts/.tsx` to all four passes and EVERY other file to semgrep
    (:272-280); and a whole compiled-standards-manifest reader
    (`compiledManifestFile`/`isCompiled`/`deriveRules`/`hasSemgrepSignal`/
    `routableExtensions`/`legacyRules` — :90-179) whose PRODUCER no longer exists
    (there is no `.manifest.json` writer / standards-compiler in the tree; the only
    live path is `defaultManifest()` → `routeFileDefaults`). The standing,
    SETTLED principle is that backstop bakes in ZERO checks, policies, standards, or
    tool-knowledge — everything comes from packs, and backstop only runs what packs
    declare. The legacy engine is the largest remaining violation of that principle.
  user_story: >
    As the maintainer driving the thin-executor finish line, I want gate Step 2 to run
    the toolchain packs the project DECLARES (lint/build/test as Layer-0 engine passes,
    dispatched through the same declared substrate the rest of the gate already uses)
    instead of a baked-in `CheckType` enum + hardcoded routing + a dead standards-manifest
    reader, so that backstop contains no opinion about which tools run, on which files,
    for which language — every such opinion lives in a pack. The proven template is the
    `go-toolchain` pack + the SPEC-034 bridge that routes the native lint/build/test
    passes through `dispatchPackEngines` and then deletes the bespoke Go path; this
    bundle generalizes that cutover to RETIRE `pkg/check` as a baked engine entirely.
  success_criteria: []

solution:
  approach: >
    The endgame is settled and the mechanics are now resolved (OQ-1…OQ-6, recorded
    2026-06-24): gate Step 2 becomes "run the toolchain packs the project declares" —
    nothing baked — and `pkg/check` is retired as a second enforcement engine.
    Concretely: (OQ-1) WHOLESALE replace `pkg/check.Run` as Step 2, routing
    lint/build/test through `dispatchPackEngines` as declared toolchain-pack passes,
    with the legacy deletion gated on a ONE-SHOT golden-equivalence assertion (capture
    the legacy engine's violation set on the backstop repo as a golden fixture, assert
    the pack path reproduces it, then delete in the SAME PR — no standing dual-run
    window). (OQ-2) ONE `<lang>-toolchain` pack per language; the no-toolchain-pack
    baseline is WARN-ONLY (enforcement is genuinely opt-in) BUT the "no enforcement
    ran" state is a LOUDLY and DISTINCTLY surfaced report state, never collapsed into a
    normal green. (OQ-3) DELETE the already-dead non-Go semgrep catch-all in
    `routeFileDefaults` (behavior-preserving — `CheckTypeFindings` has no `pkg/check`
    executor; findings already run through the pack engine); semgrep-on-arbitrary-files
    becomes an opt-in declared pack rule. (OQ-4) SPEC-035, ISSUE-018, and BUNDLE-009 have
    all LANDED on `main`, so no hard external deps remain; this bundle ABSORBS SPEC-034's
    unfinished deletion scope (its bridge landed but its deletion did not) and generalizes
    it beyond Go — SPEC-034 is to be marked SUPERSEDED; BUNDLE-009 stays a separate
    coordinate-don't-subsume target. (OQ-5) The dead standards-manifest reader in
    `pkg/check/manifest.go` is deleted as an explicit EARLY task of this cutover (in the
    blast radius anyway); the adjacent `.standard.md` scaffolder is split out to ISSUE-030
    and is NOT in scope. (OQ-6) The whole test+coverage stack migrates together — no baked
    shared runner survives, `pkg/gate/step_coverage.go` is ERADICATED here, the build-pass
    project-wide-scope exemption is re-expressed as a DECLARED gate-type/engine property
    (not a baked enum check), and the eventual spec MUST produce a CheckType-consumer
    catalog (every site keying on lint/build/test/findings identity gets a documented
    post-cutover source). Requirements, design decisions, and spec seeds follow when the
    user promotes; maturity stays `exploring`.
  assumptions: []

requirements:
  - id: REQ-001
    version: "1.0.0"
    text: >
      Gate Step 2 ("Code check") must STOP calling `pkg/check.Run`/`RunWith` as a
      baked enforcement engine. Lint/build/test must instead run as declared
      toolchain-pack passes dispatched through `dispatchPackEngines`
      (cmd/backstop/pack_gate.go), the same declared-engine substrate the rest of the
      gate uses. The replacement is WHOLESALE, not a surgical edit. (DD-1)
  - id: REQ-002
    version: "1.0.0"
    text: >
      The legacy `pkg/check` Step-2 path (`realCodeChecker` → `pkg/check.Run`, the
      `CheckType` enum, `languageExtensions`, and `routeFileDefaults` in
      pkg/check/manifest.go, plus the `builtinToolchain` go/ts stacks) must be DELETED
      in the SAME PR as the cutover. There is NO standing dual-run window and no
      "parity proven over time" exit criterion. (DD-1)
  - id: REQ-003
    version: "1.0.0"
    text: >
      The deletion in REQ-002 must be gated on a ONE-SHOT golden-equivalence
      assertion: capture the legacy engine's violation set on the backstop repo as a
      golden fixture, assert the pack-engine path reproduces it, then delete. The
      fixture IS the equivalence evidence; no parity-gate apparatus is built. (DD-1)
  - id: REQ-004
    version: "1.0.0"
    text: >
      Each language's native lint/build/test must be delivered as ONE
      `<lang>-toolchain` pack (e.g. `go-toolchain`, `typescript-toolchain`), per the
      toolchain-pack convention — each bundling its native lint/build/test as Layer-0
      engine passes. Backstop must contain no opinion about which tools run, on which
      files, for which language; every such opinion lives in a toolchain pack. (DD-2)
  - id: REQ-005
    version: "1.0.0"
    text: >
      The no-toolchain-pack baseline must be WARN-ONLY (no block, no forced opt-out) —
      enforcement is genuinely opt-in because backstop has legitimate non-enforcement
      postures (artifact-chain-only, recipe-packs-only). (DD-2)
  - id: REQ-006
    version: "1.0.0"
    text: >
      The "no enforcement ran" state (0 toolchain packs) must be a LOUDLY and
      DISTINCTLY surfaced report state (e.g. "enforcement: not configured (0 toolchain
      packs)") and must NEVER be collapsed into a normal green. Because exit 0 is
      invisible in CI, the loudness lives on the REPORT surface. This is the
      anti-vacuous-green guardrail. (DD-2)
  - id: REQ-007
    version: "1.0.0"
    text: >
      The dead non-Go semgrep catch-all in `routeFileDefaults` (`.go/.ts/.tsx` →
      all passes, everything else → semgrep) must be DELETED. Removal is
      behavior-preserving: `CheckTypeFindings` has no `pkg/check` executor
      (registry.go:229 builds executors only for lint/build/test) and findings already
      run through the pack engine (registry.go:310). Post-cutover,
      semgrep-on-arbitrary-files is an OPT-IN declared pack rule, never a baked
      default. (DD-3)
  - id: REQ-008
    version: "1.0.0"
    text: >
      This bundle must ABSORB SPEC-034's unfinished deletion scope (its bridge
      `loadBridgedToolchainPacks` landed; its deletion of the bespoke Go path did not)
      and GENERALIZE the cutover beyond Go to any `<lang>-toolchain` pack. SPEC-034
      must be marked SUPERSEDED/absorbed. (DD-4)
  - id: REQ-009
    version: "1.0.0"
    text: >
      This bundle must NOT regress the landed BUNDLE-009 traceability packs
      (substantiveness, contracts). Traceability and code-check are separate baked
      components; the seam is coordinate-don't-subsume, not a merge. (DD-4)
  - id: REQ-010
    version: "1.0.0"
    text: >
      The dead standards-manifest reader in pkg/check/manifest.go
      (`compiledManifestFile`, `isCompiled`, `deriveRules`, `hasSemgrepSignal`,
      `legacyRules`, the `.manifest.json` branch of `LoadManifest`) must be DELETED as
      an explicit EARLY task of the cutover — it is dead-fed (no `.manifest.json`
      producer, zero callers outside `manifest.go`) and lives in the file the cutover
      rewrites. The adjacent `.standard.md` scaffolder is OUT of scope (ISSUE-030).
      (DD-5)
  - id: REQ-011
    version: "1.0.0"
    text: >
      The test + coverage stack must migrate together as a unit: no baked shared `go
      test` runner survives, and `pkg/gate/step_coverage.go` (baked coverage, KEPT by
      SPEC-038 REQ-009) must be ERADICATED here, with coverage re-implemented
      language-agnostic over the toolchain test runner. (DD-6)
  - id: REQ-012
    version: "1.0.0"
    text: >
      The build-pass project-wide-scope exemption (currently the baked
      `cv.Pass == check.CheckTypeBuild` enum check at gate.go:1173) must be
      re-expressed as a DECLARED gate-type/engine property, not a baked enum check.
      (DD-6)
  - id: REQ-013
    version: "1.0.0"
    text: >
      The spec must produce a CheckType-consumer catalog: every site keying on
      lint/build/test/findings identity gets a documented post-cutover source. This is
      the "don't drop a gate step on the floor" guard against a wholesale (REQ-001)
      cutover silently stranding a consumer. (DD-6)
  - id: REQ-014
    version: "1.0.0"
    text: >
      The engine model must gain a SECOND normalized output type — coverage-records —
      distinct from SARIF findings. SARIF remains the findings/OUTPUT lingua franca;
      coverage is a per-file measurement INPUT the gate consumes and turns into
      ordinary gate violations, NOT a competing report format. A declared engine
      filling gate-type `coverage` emits per-file coverage records. Coverage cannot be
      SARIF: SARIF represents failures (sparse, located), but coverage is a complete
      per-file census including passing files, and non-vacuousness requires
      distinguishing measured-and-passed from not-measured — which SARIF-as-findings
      cannot express. (DD-7)
  - id: REQ-015
    version: "1.0.0"
    text: >
      The coverage-record format the producer emits and Seed 3's `CoverageRecord`
      consumer contract must agree on is
      `{path (toolchain-declared file path), covered, total, measured, excluded,
      metric}`. Two conventions are mandatory: (a) `total == 0` / no-executable-lines
      is N/A, never a 0%-fail; (b) `metric` is a pack-declared label
      (statement/line/branch/…) that MUST be surfaced on the report so a polyglot repo
      cannot silently compare one language's statement-% against another's branch-%
      under a single number. The gate stays metric-BLIND (compares `covered/total` ≥
      threshold); the pack declares both the numbers and what they measure. (DD-7)
  - id: REQ-016
    version: "1.0.0"
    text: >
      The `go-toolchain` pack must gain a coverage engine — the first concrete
      producer — that runs `go test -coverprofile`, converts the profile to per-file
      coverage records in the REQ-015 format, and fills gate-type `coverage`. The
      record shape is language-agnostic so a future `typescript-toolchain` coverage
      engine emits the same shape. Today nothing produces these records (the
      go-toolchain test engine emits only test-failure SARIF), so without this engine
      Seed 3's coverage consumer is unsatisfiable. (DD-7)
---

# Collapse the Legacy `pkg/check` Engine into Pack-Declared Toolchain Packs

## Current Thinking

This bundle is the keystone of backstop's **thin-executor finish line**: the point
at which `backstop gate` runs no enforcement logic, tool-knowledge, or policy that
isn't declared by a pack. The standing principle — backstop bakes in zero checks /
policies / standards / tool-knowledge; everything comes from packs; backstop only
runs what packs declare — is **settled and not in question here**. What this bundle
explores is the **mechanics and sequencing of the migration**, never *whether* the
legacy engine should go (it must).

**The concrete state on `main` (verified 2026-06-21).** The gate runs two engines:

1. **Step 2, "Code check" — `pkg/check.Run` (the legacy baked engine).**
   `realCodeChecker` (cmd/backstop/gate.go:332) calls `check.Run` / `check.RunWith`
   (:637, :649). This engine bakes in, all in `pkg/check/manifest.go`:
   - a hardcoded `CheckType` enum — `lint` (golangci-lint), `build` (go build),
     `test` (go test), `semgrep` (:14-23);
   - `languageExtensions` (go→`.go`, typescript→`.ts/.tsx` — :75-78);
   - `routeFileDefaults` — `.go/.ts/.tsx` → all four passes; **everything else →
     semgrep only** (:272-280);
   - a compiled-standards-manifest reader (`compiledManifestFile`, `isCompiled`,
     `deriveRules`, `hasSemgrepSignal`, `routableExtensions`, `legacyRules` —
     :90-179) whose **producer is gone**. There is no `.manifest.json` writer and no
     standards-compiler in the tree, so `LoadManifest` always falls to
     `defaultManifest()` → `routeFileDefaults`. The reader is dead-fed.

2. **The declared pack engine — `dispatchPackEngines` (cmd/backstop/pack_gate.go).**
   This is the substrate everything is converging toward: group rules by declared
   engine, run each, normalize to SARIF. BUNDLE-010 shipped it.

**Why this is a bundle and not just an issue.** Retiring `pkg/check` as Step 2 is
not a delete — the lint/build/test passes are real enforcement that must keep
running, just *as declared toolchain-pack passes through the engine substrate*
rather than as a baked enum. That is the SPEC-034 move (bridge the native Go
toolchain onto `dispatchPackEngines` as a `go-toolchain` pack, then delete the
bespoke Go path) generalized from "the Go arm" to "the whole `pkg/check` engine."
The sequencing, the no-pack baseline question, the semgrep catch-all, and the
coordination with three adjacent in-flight artifacts are all genuinely open.

**Grounding on the adjacent work (re-verified on `main` 2026-06-24 during OQ-4
resolution).** The dependency picture changed materially since 0.1.x: **SPEC-035,
ISSUE-018, and BUNDLE-009 have ALL LANDED.** No hard external dependency remains.

- **SPEC-035 (pack-declared-engines-trusted-allowlist)** — **LANDED**. Engine
  *bindings* now live in a pack `engines:` block, the trusted-tool allowlist (the
  security substrate) exists, and the tool-named `CheckTypeSemgrep` was replaced with a
  tool-NEUTRAL gate-TYPE enum. The substrate a toolchain pack needs to declare its
  engine commands safely is in place — so it is no longer a blocking prerequisite, it
  is a satisfied one.
- **ISSUE-018** — **LANDED**. The dead in-process `semgrepExecutor` body and the dead
  native-standards *validator* (`pkg/validate/standard.go`) are gone. As anticipated,
  it stopped short of retiring `pkg/check.Run` as Step 2 — the lint/build/test
  pass-order, the `CheckType` enum, and `pkg/check/manifest.go`'s routing remain. **That
  residual is exactly this bundle's target.**
- **SPEC-034 (native-toolchain-engine-cutover)** — **HALF-DONE**, and this is the
  load-bearing OQ-4 finding. Its *bridge* landed (`loadBridgedToolchainPacks`,
  gate.go:464) but its *deletion* did NOT: `realCodeChecker` → `pkg/check.Run` plus the
  `builtinToolchain` go/ts stacks are still the live Step 2. SPEC-034's unfinished
  deletion scope is now **ABSORBED into this bundle** and generalized beyond Go;
  SPEC-034 is to be marked **SUPERSEDED/absorbed** (per the align-predating-artifacts
  principle — don't leave a half-done draft dangling). See RDQ-4.
- **BUNDLE-009 (stack-aware-traceability)** — **LANDED**. Its traceability analyzers
  (substantiveness, contracts) are now installed packs. It is a **separate** target
  (traceability ≠ code-check): coordinate-don't-subsume — this bundle must simply **not
  regress its packs**. NOTE the OQ-6 correction: BUNDLE-009 did *not* delete
  `pkg/gate/step_coverage.go` — SPEC-038 REQ-009 descoped coverage and KEPT it baked
  (enforced by a passing test). Eradicating that baked coverage step is now THIS
  bundle's job (RDQ-6).

The load-bearing reframe: **this bundle is where "Step 2" stops being a noun for a
baked engine and becomes a verb — "run the declared toolchain packs."** Everything
else (SPEC-034/035, ISSUE-018) clears the ground; this bundle removes the last baked
engine standing on it.

## Open Questions

All six open questions were worked one at a time with the user and are RESOLVED
(2026-06-24). None remain open. Their resolutions, with rationale, are recorded in
**Resolved Design Questions** below; the original framing is preserved there.

(none open)

## Resolved Design Questions

Resolved 2026-06-24, worked one at a time with the user. Maturity stays `exploring`
— these resolutions are recorded as decisions, not promoted into formal
Draft Requirements / Draft Design Decisions yet (the user drives promotion).

**RDQ-1 — Cutover shape: WHOLESALE replace, gated on a one-shot golden-equivalence
assertion.** *(was OQ-1: wholesale replace vs dual-run-with-parity-gate then delete.)*
**Resolution:** WHOLESALE replace `pkg/check.Run` as gate Step 2 — route lint/build/test
through `dispatchPackEngines` as declared toolchain-pack passes — with the legacy-path
deletion gated on a **ONE-SHOT golden-equivalence assertion**: capture the legacy
engine's violation set on the backstop repo as a golden fixture, assert the pack path
reproduces it, then delete the legacy path **in the SAME PR**. **NO standing dual-run
window.**
- *Rationale:* with N=1 (one repo) and no external base, a transition window has nothing
  to compare against over time — it would be scaffolding for a population of one. The
  one-shot golden fixture IS the equivalence evidence (the real worry behind the
  dual-run option), captured cheaply without a parity-gate apparatus or a "parity proven"
  exit criterion. Rejected (b) dual-run + standing parity gate as disproportionate.

**RDQ-2 — Pack coverage + the no-toolchain-pack baseline: per-language pack; WARN-ONLY
baseline that is LOUDLY surfaced.** *(was OQ-2: pack coverage of today's routing + the
no-pack baseline — the central correctness question.)* **Resolution, two parts:**
- (i) **ONE `<lang>-toolchain` pack per language** (per the toolchain-pack convention —
  `go-toolchain`, `typescript-toolchain`, …), each bundling its native lint/build/test
  as Layer-0 engine passes.
- (ii) **No-toolchain-pack baseline = WARN-ONLY** — no block, no forced opt-out.
  Enforcement is genuinely opt-in because backstop has legitimate non-enforcement uses
  (artifact-chain-only, recipe-packs-only). **CRITICAL GUARDRAIL:** the "no enforcement
  ran" state MUST be a **LOUDLY and DISTINCTLY surfaced report state** (e.g.
  "enforcement: not configured (0 toolchain packs)"), **never collapsed into a normal
  green**. Exit 0 is invisible in CI, so the loudness lives on the **report surface**.
  This keeps warn-only off the vacuous-green failure mode.
- *Rationale:* per-language packs match the established [[project_toolchain_pack_convention]].
  On the baseline: a hard block would break backstop's legitimate chain-only /
  recipe-only postures, so enforcement stays opt-in — but the enforcement philosophy's
  enemy is **silent/vacuous green**, not "didn't enforce." The resolution threads that
  needle per [[feedback_loud_not_blocking]]: don't block un-adopted capability, but make
  "nothing ran" impossible to mistake for "everything passed." The loudness moving to the
  report surface (since exit 0 is invisible in CI) is the load-bearing detail.

**RDQ-3 — The non-Go semgrep catch-all: DELETE it (already a no-op).** *(was OQ-3: what
replaces the semgrep catch-all on non-Go files; is it already a no-op.)* **Resolution:**
DELETE the dead non-Go catch-all routing in `routeFileDefaults`. **Verified it is ALREADY
a no-op:** `CheckTypeFindings` has no executor in `pkg/check` (`registry.go:229` builds
executors only for lint/build/test), and findings already run through the pack engine
(`registry.go:310`). Removal is **behavior-preserving**. Post-cutover,
semgrep-on-arbitrary-files is an **OPT-IN declared pack rule**, never a baked default.
- *Rationale:* the catch-all can't change the current gate result because the findings
  type it routed to has no `pkg/check`-side executor — findings are already serviced by
  the pack engine. So deletion is provably behavior-preserving on this repo and needs no
  parity fixture of its own. Keeping arbitrary-file semgrep means a project declares it in
  a pack, consistent with zero-baked-defaults.

**RDQ-4 — Ordering/division vs SPEC-034/035, ISSUE-018, BUNDLE-009: no external deps
remain; ABSORB SPEC-034's unfinished deletion.** *(was OQ-4.)* **Resolution, grounded in
VERIFIED current `main` (2026-06-24):**
- **SPEC-035, ISSUE-018, and BUNDLE-009 have ALL LANDED** — **no hard external
  dependencies remain** for this bundle.
- **SPEC-034 is HALF-DONE:** its bridge (`loadBridgedToolchainPacks`, gate.go:464) landed
  but its DELETION did not — `realCodeChecker → pkg/check.Run` plus the `builtinToolchain`
  go/ts stacks are still the live Step 2. **BUNDLE-011 ABSORBS SPEC-034's unfinished
  deletion scope and generalizes it beyond Go.** SPEC-034 is to be marked
  **SUPERSEDED/absorbed** (per [[feedback_align_predating_artifacts]] — don't leave a
  half-done draft dangling).
- **BUNDLE-009 is a SEPARATE target** (traceability ≠ code-check): **coordinate, don't
  subsume**; just don't regress its (now-landed) packs.
- *Rationale:* the dependency picture that made SPEC-035 look like a hard prerequisite has
  resolved itself — the substrate landed. The remaining hazard is a dangling half-done
  draft (SPEC-034), which align-predating-artifacts says to fold-and-supersede rather than
  work around. Traceability and code-check are genuinely different baked components, so the
  seam with BUNDLE-009 stays a coordination boundary, not a merge.

**RDQ-5 — The dead standards-manifest reader: FOLD its deletion in as an explicit EARLY
task.** *(was OQ-5.)* **Resolution:** FOLD the dead reader's deletion into this cutover as
an explicit **EARLY task** — `compiledManifestFile` / `isCompiled` / `deriveRules` /
`hasSemgrepSignal` / `legacyRules` plus the `.manifest.json` branch of `LoadManifest` in
`pkg/check/manifest.go`. It's in the blast radius anyway; sequencing it **first** keeps
the diff clean. **Verified dead-fed:** no `.manifest.json` producer exists, zero callers
outside `manifest.go`. The **ADJACENT `.standard.md` scaffolder** (`pkg/pack/scaffold.go`,
`number.go`) was split OUT into its own issue — **ISSUE-030** (native-standards tombstone)
— and is **NOT in BUNDLE-011's scope**.
- *Rationale:* the reader lives inside the very file this cutover rewrites, so deleting it
  here avoids a separate dead-code PR touching the same surface; doing it first shrinks the
  surface the cutover proper has to reason about. The `.standard.md` scaffolder is a
  distinct concern (a generator, not a reader) and gets its own tombstone issue so it
  isn't smuggled into the cutover diff.

**RDQ-6 — CheckType consumers: PULL coverage forward; eradicate the baked shared runner
here.** *(was OQ-6: re-routing the gate steps that consume `CheckType` outputs.)*
**Resolution:**
- **PULL coverage's language-agnostic re-implementation FORWARD into BUNDLE-011** — the
  whole **test + coverage stack migrates together**, NO baked shared runner is preserved,
  and **`pkg/gate/step_coverage.go` is ERADICATED here.**
- **Correction this bundle absorbs:** coverage was **NEVER deleted by BUNDLE-009** —
  **SPEC-038 REQ-009 DESCOPED it and KEPT it baked**, enforced by a passing test. The
  earlier OQ-2 NOTE claiming "BUNDLE-009 deletes `step_coverage.go` without replacement"
  was **STALE prose** and is corrected (see the reconciled NOTE under RDQ-2's context
  below and the Version History).
- **Re-express the build-pass project-wide-scope exemption** (currently
  `cv.Pass == check.CheckTypeBuild` at gate.go:1173) as a **DECLARED gate-type/engine
  property**, not a baked enum check.
- **MANDATE that the eventual spec produce a CheckType-consumer catalog:** every site
  keying on lint/build/test/findings identity gets a documented post-cutover source. This
  is the **"don't drop a gate step on the floor"** guard.
- *Rationale:* the shared `go test` runner feeds both Step 2's test FAILs and the coverage
  step, so coverage cannot be left behind keyed on a `CheckType` that Step 2 no longer
  produces — splitting them would either strand coverage or recreate a baked runner.
  Migrating the stack as a unit is the only way to delete the baked runner without
  regressing coverage. The build-exemption must become a declared property for the same
  zero-baked-enum reason as the rest of the cutover. The consumer catalog is the explicit
  guard against silently dropping a gate step during a wholesale (RDQ-1) cutover.

### Context carried forward from the OQ-2 NOTE (still valid, one correction)

Recorded 2026-06-22 from a BUNDLE-009 scoping session; still load-bearing for the
toolchain-pack work, with one stale claim corrected (see ‡):

1. *The trusted-tool allowlist explodes TOGETHER WITH the toolchain packs, not ahead of
   them.* The backstop-owned trusted-tool allowlist (the trust floor, the SPEC-035
   security substrate) today holds only **semgrep + ast-grep**. Populating it
   (eslint/tsc/vitest/ruff/cargo/…) is **inert** without a pack that declares an engine
   *using* each tool. So the allowlist entries and the toolchain pack that needs them
   **ship as a PAIR, per language**. This pairing belongs to **this bundle (+ ISSUE-027)**,
   not to any consumer bundle.
2. *TypeScript is the FIRST PROOF CASE and a LIVE PRIORITY — not hypothetical.*
   **backstop-runtime** (TypeScript) is currently **BLOCKED**: it cannot gate itself with
   packs because there is no pack-based TS toolchain support, and the baked TS built-in in
   `pkg/check` (eslint/tsc) is itself a zero-baked-checks violation slated for eradication
   by this cutover. A **`typescript-toolchain` pack + its allowlist entries** is the
   concrete near-term goal of RDQ-2's per-language direction.
3. *Division of TS support across bundles:* **BUNDLE-009** delivered the TS
   **TRACEABILITY** slice (substantiveness + contracts on ast-grep/grep), which rode
   structural engines and needed no toolchain. **This bundle (BUNDLE-011)** owes the TS
   **TOOLCHAIN** slice (lint/build/test + the test runner). Together they unblock the
   runtime gating itself; BUNDLE-011 is the natural NEXT.
4. ‡ *Language-agnostic COVERAGE is THIS bundle's job (corrected).* **Earlier prose
   claimed BUNDLE-009 was DELETING the baked Go coverage analyzer
   (`pkg/gate/step_coverage.go`) without replacement — that is STALE and WRONG.** In fact
   **SPEC-038 REQ-009 DESCOPED coverage and KEPT it baked** (a passing test enforces it).
   Per RDQ-6, eradicating `step_coverage.go` and re-implementing coverage language-agnostic
   over the toolchain test runner is **BUNDLE-011's responsibility**, executed together with
   the test-stack migration.

## Draft Requirements

Formal, traceable requirements are enumerated in the `requirements:` frontmatter
array (REQ-001 … REQ-016). REQ-001…REQ-013 were lifted directly from a resolved
design question (RDQ-1 … RDQ-6) during the `defined` promotion — no new scope.
REQ-014…REQ-016 were added in the v0.4.0 evolution (DD-7) to add the coverage
PRODUCER side that Seed 3's consumer contract requires (Seed 4); this is a deliberate
scope addition that closes a gap, not a gate-clearing fabrication. Each carries the
Draft Design Decision (DD-N below) it derives from. Summary of what this bundle
commits to:

- **Wholesale cutover, gated on a one-shot golden fixture** (REQ-001…REQ-003, from
  RDQ-1 via DD-1) — Step 2 routes lint/build/test through `dispatchPackEngines`; the
  legacy `pkg/check` path is deleted in the same PR after a one-shot
  golden-equivalence assertion; no standing dual-run.
- **Per-language toolchain packs + loud no-enforcement state** (REQ-004…REQ-006,
  from RDQ-2 via DD-2) — one `<lang>-toolchain` pack per language; the no-pack
  baseline is warn-only but the "0 toolchain packs / nothing ran" state is a loudly
  and distinctly surfaced report state, never a vacuous green.
- **Delete the dead non-Go semgrep catch-all** (REQ-007, from RDQ-3 via DD-3) —
  behavior-preserving (already a no-op); arbitrary-file semgrep becomes an opt-in
  declared pack rule.
- **Absorb SPEC-034's deletion + coordinate with BUNDLE-009** (REQ-008, REQ-009,
  from RDQ-4 via DD-4) — generalize the cutover beyond Go, mark SPEC-034 superseded,
  don't regress BUNDLE-009's landed traceability packs.
- **Delete the dead standards-manifest reader early** (REQ-010, from RDQ-5 via
  DD-5) — `.standard.md` scaffolder split out to ISSUE-030.
- **Migrate the test+coverage stack together** (REQ-011…REQ-013, from RDQ-6 via
  DD-6) — eradicate `step_coverage.go`, re-implement coverage language-agnostic over
  the toolchain test runner, re-express the build-pass exemption as a declared
  property, and produce a CheckType-consumer catalog.
- **Produce coverage records (the producer side Seed 3 consumes)** (REQ-014…REQ-016,
  via DD-7) — add a second engine-model output type (coverage-records as a
  measurement INPUT, distinct from SARIF findings), pin the per-file
  `{path, covered, total, measured, excluded, metric}` record format, and ship the
  first concrete producer: a `go-toolchain` coverage engine over `go test
  -coverprofile`.

## Draft Design Decisions

Each decision is lifted from a resolved design question (RDQ-N), carrying that RDQ's
rationale unchanged. Full original framing and rationale remain under **Resolved
Design Questions** above; these are the promotion-time summaries.

**DD-1 — Wholesale cutover, deletion gated on a one-shot golden-equivalence
assertion (no dual-run).** *(from RDQ-1.)* Replace `pkg/check.Run` as gate Step 2
wholesale — route lint/build/test through `dispatchPackEngines` as declared
toolchain-pack passes — and delete the legacy path in the SAME PR, gated only on a
one-shot golden fixture (capture the legacy engine's violation set on the backstop
repo, assert the pack path reproduces it). *Rationale:* with N=1 (one repo, no
external base) a standing dual-run window has nothing to compare against over time —
it is scaffolding for a population of one. The one-shot fixture is the equivalence
evidence the dual-run option was really after, captured cheaply without a parity-gate
apparatus. Rejected a standing dual-run + parity gate as disproportionate.

**DD-2 — One `<lang>-toolchain` pack per language; no-pack baseline is warn-only but
LOUDLY surfaced.** *(from RDQ-2 — the philosophical crux.)* Each language's
lint/build/test ships as a single `<lang>-toolchain` pack (Layer-0 engine passes).
The no-toolchain-pack baseline is warn-only (enforcement is opt-in for backstop's
legitimate chain-only / recipe-only postures), but the "no enforcement ran" state
MUST be a loudly and distinctly surfaced report state, never collapsed into a normal
green; since exit 0 is invisible in CI, the loudness lives on the report surface.
*Rationale:* per-language packs match [[project_toolchain_pack_convention]]. A hard
block would break legitimate non-enforcement postures, so enforcement stays opt-in —
but the enforcement philosophy's enemy is silent/vacuous green, not "didn't enforce"
([[feedback_loud_not_blocking]]). This threads the needle: don't block un-adopted
capability, but make "nothing ran" impossible to mistake for "everything passed."

**DD-3 — Delete the non-Go semgrep catch-all (already a no-op).** *(from RDQ-3.)*
Delete the dead `routeFileDefaults` non-Go catch-all; it is verified
behavior-preserving because `CheckTypeFindings` has no `pkg/check` executor
(registry.go:229) and findings already run through the pack engine (registry.go:310).
Post-cutover, semgrep-on-arbitrary-files is an opt-in declared pack rule. *Rationale:*
the catch-all can't change the current gate result, so deletion is provably
behavior-preserving on this repo and needs no parity fixture of its own; keeping
arbitrary-file semgrep behind a pack declaration is consistent with zero-baked-defaults.

**DD-4 — Absorb SPEC-034's unfinished deletion + generalize beyond Go; coordinate,
don't subsume, BUNDLE-009.** *(from RDQ-4, grounded in verified `main` 2026-06-24.)*
SPEC-035, ISSUE-018, and BUNDLE-009 have all landed — no hard external dependencies
remain. SPEC-034 is half-done (bridge landed, deletion didn't); this bundle absorbs
its unfinished deletion scope, generalizes it beyond Go, and SPEC-034 is marked
SUPERSEDED/absorbed. BUNDLE-009 stays a separate target — don't regress its packs.
*Rationale:* the substrate that made SPEC-035 look like a hard prerequisite has
landed; the remaining hazard is a dangling half-done draft, which
[[feedback_align_predating_artifacts]] says to fold-and-supersede rather than work
around. Traceability and code-check are genuinely different baked components, so the
BUNDLE-009 seam stays a coordination boundary.

**DD-5 — Delete the dead standards-manifest reader as an early task; scaffolder is
out of scope.** *(from RDQ-5.)* Delete the dead reader
(`compiledManifestFile`/`isCompiled`/`deriveRules`/`hasSemgrepSignal`/`legacyRules` +
the `.manifest.json` branch of `LoadManifest`) first, since it lives in the file the
cutover rewrites and is dead-fed (no producer, zero external callers). The adjacent
`.standard.md` scaffolder is a distinct generator and gets its own tombstone
(ISSUE-030). *Rationale:* deleting it here avoids a separate dead-code PR touching the
same surface; doing it first shrinks the surface the cutover proper has to reason
about. The scaffolder is a separate concern not smuggled into the cutover diff.

**DD-6 — Migrate the test+coverage stack as a unit; eradicate the baked runner +
coverage step; declared build-exemption; mandate a consumer catalog.** *(from
RDQ-6.)* Pull coverage's language-agnostic re-implementation forward into this
bundle: no baked shared `go test` runner survives, `pkg/gate/step_coverage.go` is
eradicated (it stayed baked via SPEC-038 REQ-009's descope, not deleted by
BUNDLE-009), the build-pass project-wide-scope exemption (`cv.Pass ==
check.CheckTypeBuild`, gate.go:1173) becomes a declared gate-type/engine property, and
the spec must produce a CheckType-consumer catalog documenting every site keying on
lint/build/test/findings identity. *Rationale:* the shared `go test` runner feeds both
Step 2's test FAILs and the coverage step, so coverage cannot be left behind keyed on
a CheckType that Step 2 no longer produces — migrating the stack as a unit is the only
way to delete the baked runner without regressing coverage. The build-exemption must
become a declared property for the same zero-baked-enum reason as the rest of the
cutover. The consumer catalog is the explicit "don't drop a gate step on the floor"
guard against a wholesale (DD-1) cutover.

**DD-7 — Coverage is a second engine-model output type: a per-file measurement INPUT,
not a competing report format.** *(v0.4.0 evolution; the producer counterpart to
DD-6/Seed 3.)* The engine model gains a second normalized output type alongside
SARIF: **coverage-records**. The architectural line: **SARIF stays the findings/OUTPUT
lingua franca; coverage is a measurement INPUT** the gate consumes and turns into
ordinary gate violations — it is NOT a second report format competing with SARIF. A
declared engine filling gate-type `coverage` emits per-file coverage records of shape
`{path, covered, total, measured, excluded, metric}`. The gate stays metric-BLIND
(compares `covered/total` ≥ threshold); the pack declares both the numbers and the
`metric` label, which MUST be surfaced so a polyglot repo can't silently compare one
language's statement-% against another's branch-% under one number. `total == 0` is
N/A, never a 0%-fail. The first concrete producer is a `go-toolchain` coverage engine
over `go test -coverprofile`. *Rationale — why coverage can't be SARIF:* SARIF
represents failures, which are sparse and located; coverage is a complete per-file
census that must include passing files, because non-vacuousness requires
distinguishing **measured-and-passed** from **not-measured** — a distinction
SARIF-as-findings structurally cannot carry. This is the load-bearing reason coverage
needs its own channel rather than being squeezed into the findings stream. The record
shape is language-agnostic so a future `typescript-toolchain` coverage engine emits
the same shape; the producer↔consumer contract on this format is shared with Seed 3
(SPEC-041) and the two MUST agree.

## Spec Seeds

Suggested decomposition into specs, in implementation order. No requirement belongs
to two seeds. Seeds 1–3 trace to recorded RDQ content. Seed 4 (v0.4.0) adds the
coverage PRODUCER scope that Seed 3's consumer contract requires.

**Seed 1 — Dead-code prelude: standards-manifest reader + non-Go semgrep catch-all.**
The early, behavior-preserving deletions that shrink the surface the cutover proper
rewrites: delete the dead standards-manifest reader in pkg/check/manifest.go (RDQ-5)
and the already-no-op non-Go `routeFileDefaults` catch-all (RDQ-3). Both are verified
dead/no-op, so they land first with no golden fixture of their own. Covers REQ-007,
REQ-010. (RDQ-3, RDQ-5.)

**Seed 2 — Toolchain-pack substrate + the `go-toolchain` cutover (golden-equivalence
harness).** The keystone. Build the golden-equivalence harness: capture the legacy
engine's violation set on the backstop repo as a golden fixture and assert the
pack-engine path reproduces it (RDQ-1). Route lint/build/test through
`dispatchPackEngines` as declared `go-toolchain` passes, then delete the legacy
`pkg/check` Step-2 path + `builtinToolchain` go/ts stacks in the same PR — absorbing
SPEC-034's unfinished deletion and generalizing the cutover machinery beyond Go
(RDQ-4; mark SPEC-034 SUPERSEDED). Establish the one-`<lang>-toolchain`-pack-per-language
model and the warn-only-but-loudly-surfaced no-enforcement report state — the
anti-vacuous-green guardrail (RDQ-2). Must not regress BUNDLE-009's traceability packs
(RDQ-4). Covers REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-006, REQ-008,
REQ-009. (RDQ-1, RDQ-2, RDQ-4.)

**Seed 3 — Coverage re-implementation + CheckType-consumer catalog.** Migrate the
test+coverage stack as a unit: eradicate `pkg/gate/step_coverage.go` and re-implement
coverage language-agnostic over the toolchain test runner, with no baked shared runner
surviving; re-express the build-pass project-wide-scope exemption as a declared
gate-type/engine property; and produce the CheckType-consumer catalog documenting a
post-cutover source for every lint/build/test/findings consumer. Depends on Seed 2's
toolchain test runner. Covers REQ-011, REQ-012, REQ-013. (RDQ-6.) **Consumes the
coverage records produced by Seed 4 — the two MUST agree on the `CoverageRecord`
format.**

**Seed 4 — Coverage production: engine-model coverage-records channel + go-toolchain
coverage engine.** The PRODUCER side of coverage, and the reason Seed 3's consumer
contract is satisfiable. Three parts: (1) **Substrate** — give the engine model a
second normalized output type, **coverage-records, a per-file measurement INPUT
distinct from SARIF findings** (SARIF stays the findings/OUTPUT lingua franca; coverage
is a measurement the gate turns into ordinary violations, not a competing report
format); a declared engine filling gate-type `coverage` emits per-file records (DD-7).
(2) **Format** — define the producer-side `{path, covered, total, measured, excluded,
metric}` record that Seed 3's `CoverageRecord` consumer contract must match, with the
`total == 0` ⇒ N/A and pack-declared-`metric`-surfaced-on-report conventions, gate
staying metric-blind. (3) **First producer** — the `go-toolchain` coverage engine that
runs `go test -coverprofile` and converts the profile to per-file records; the shape is
language-agnostic so a future `typescript-toolchain` coverage engine emits the same
records. Depends on Seed 2 (the toolchain-pack dispatch substrate); lands with/before
Seed 3's coverage step has anything to read. Producer↔consumer with Seed 3 on the
record format. Covers REQ-014, REQ-015, REQ-016. (DD-7.)

## Notes / Ideas

- **SPEC-034's bridge tests are a worked example for the wholesale move (RDQ-1).**
  SPEC-034's bridge landed on `main` (`loadBridgedToolchainPacks`, gate.go:464) along
  with its TDD tests; its DELETION did not (RDQ-4). Those tests + the existing
  `go-toolchain` pack are the ready-made safety net and worked example for absorbing the
  unfinished deletion and generalizing it beyond Go.
- **The "no-pack → vacuous green" risk (RDQ-2) is the philosophical crux — resolved by
  loudness, not blocking.** The enforcement philosophy's enemy is silent/vacuous green.
  The resolution keeps enforcement opt-in (warn-only) for backstop's legitimate
  chain-only / recipe-only postures, but mandates that "0 toolchain packs / nothing ran"
  be a LOUDLY and DISTINCTLY surfaced report state — never collapsed into a normal green.
  Loudness lives on the report surface because exit 0 is invisible in CI. This is the
  single most important thing not to get wrong.
- **The non-Go semgrep catch-all is already a no-op (RDQ-3, verified).**
  `CheckTypeFindings` has no `pkg/check` executor (`registry.go:229` builds executors
  only for lint/build/test); findings already run through the pack engine
  (`registry.go:310`). So deleting the `routeFileDefaults` catch-all is provably
  behavior-preserving on this repo.
- **"Step 2" is a position, not an engine.** After this cutover the gate's step LIST
  is unchanged in spirit (artifact-validate → code-check → test-verify →
  substantiveness → coverage → contracts), but "code check" becomes "run declared
  toolchain packs." The step name may want to change to reflect that it no longer
  owns any tool-knowledge.

## References

- [[project_thin_executor_engine_packs]] — the thesis this bundle completes: packs
  laid out by engine; backstop knows no engine, runs declared commands, speaks only
  SARIF; thin on knowledge, firm on enforcement. The legacy `pkg/check` engine is the
  largest remaining contradiction of it.
- [[project_eradication_backlog]] — the 2026-06-20 finish-line audit. This bundle is
  the "collapse the legacy code-check engine" keystone; ISSUE-018 (B/F), SPEC-035 (A),
  and BUNDLE-009 (C) are the adjacent seeds (OQ-4).
- [[project_native_toolchain_cutover]] — SPEC-034's intent (bridge the Go toolchain
  onto the engine substrate, delete the bespoke Go path). Its BRIDGE landed on `main`
  (`loadBridgedToolchainPacks`, gate.go:464); its DELETION did not. This bundle ABSORBS
  that unfinished deletion and generalizes it beyond Go — SPEC-034 to be marked
  SUPERSEDED/absorbed (RDQ-4).
- [[project_toolchain_pack_convention]] — one `<lang>-toolchain` pack per language
  bundling its native lint/build/test; the convention OQ-2 leans on.
- [[project_packs_only_no_native_standards]] — the settled directive that everything
  comes from packs; the standards-manifest reader (OQ-5) is its last code-side
  vestige.
- [[feedback_loud_not_blocking]] — governs OQ-2's no-pack baseline: block defects +
  broken promises, warn-with-guidance for un-adopted capability; the enemy is
  vacuous green.
- [[project_pack_engine_model]] — `dispatchPackEngines` is the substrate Step 2
  collapses onto; BUNDLE-010 shipped it.
- Code (verified 2026-06-24, `main`): cmd/backstop/gate.go (`realCodeChecker` →
  `pkg/check.Run` is still live Step 2), :464 (`loadBridgedToolchainPacks` — SPEC-034's
  landed bridge), :1173 (`cv.Pass == check.CheckTypeBuild` build-pass project-wide-scope
  exemption — RDQ-6 re-expresses as a declared property); `builtinToolchain` go/ts stacks
  still live; pkg/check/registry.go:229 (executors built only for lint/build/test),
  :310 (findings run through the pack engine — RDQ-3); pkg/check/manifest.go (dead
  standards-manifest reader + `.manifest.json` branch of `LoadManifest` — RDQ-5;
  `routeFileDefaults` non-Go catch-all — RDQ-3); pkg/gate/step_coverage.go (baked
  coverage, KEPT by SPEC-038 REQ-009 — eradicated here per RDQ-6); cmd/backstop/pack_gate.go
  (`dispatchPackEngines`).
- SPEC-034 (native-toolchain-engine-cutover) — **HALF-DONE on `main`**: bridge landed,
  deletion did not. This bundle absorbs the unfinished deletion; mark SUPERSEDED (RDQ-4).
- SPEC-035 (pack-declared-engines-trusted-allowlist) — **LANDED**. Pack `engines:`
  bindings + trusted-tool allowlist + tool-neutral gate-TYPE enum. Satisfied substrate,
  no longer a blocking dep (RDQ-4).
- ISSUE-018 (remove vestigial baked-in code) — **LANDED**. Deleted dead in-process
  semgrep + native-standards validator; left Step 2 standing (this bundle's target).
- ISSUE-030 (native-standards tombstone) — the `.standard.md` scaffolder
  (`pkg/pack/scaffold.go`, `number.go`) split OUT of this bundle's scope (RDQ-5); NOT
  BUNDLE-011's job.
- SPEC-038 (REQ-009) — DESCOPED coverage and KEPT `pkg/gate/step_coverage.go` baked
  (enforced by a passing test). Corrects the earlier stale "BUNDLE-009 deletes coverage"
  claim; coverage eradication is now BUNDLE-011's job (RDQ-6).
- BUNDLE-009 (stack-aware-traceability) — **LANDED**; traceability analyzers
  (substantiveness/contracts) now installed packs. **SEPARATE target** (traceability ≠
  code-check): coordinate-don't-subsume, don't regress its packs (RDQ-4). It delivered the
  TS *traceability* slice (structural engines, no toolchain dep); this bundle (BUNDLE-011)
  owes the TS *toolchain* slice (lint/build/test) + grows the trusted-tool allowlist
  (paired with the `typescript-toolchain` pack, alongside ISSUE-027) — together they
  unblock **backstop-runtime** gating itself.
- ISSUE-027 — trusted-tool allowlist growth pairs with the per-language toolchain packs
  here (see the carried-forward NOTE under Resolved Design Questions); allowlisting a tool
  is inert until a pack declares an engine using it.

## Version History

- **0.1.0 (2026-06-21, exploring)** — Initial bundle. Problem framing grounded in the
  verified `main` state: `pkg/check.Run` is live gate Step 2, a second baked
  enforcement engine alongside `dispatchPackEngines`, carrying a hardcoded `CheckType`
  enum, `languageExtensions`, `routeFileDefaults` (incl. the non-Go semgrep
  catch-all), and a dead-fed standards-manifest reader. User story, current thinking,
  and six open questions (OQ-1 cutover shape; OQ-2 pack coverage + the no-pack vacuous-
  green baseline; OQ-3 semgrep catch-all removal; OQ-4 ordering/division vs SPEC-034 /
  SPEC-035 / ISSUE-018 / BUNDLE-009; OQ-5 dead standards-manifest reader; OQ-6
  re-routing CoverageType consumers). The settled principle (zero baked checks;
  everything from packs) is recorded as NOT an open question — only the migration
  mechanics are. No requirements, design decisions, or spec seeds yet; those follow
  OQ resolution. Maturity stays `exploring`.
- **0.1.1 (2026-06-22, exploring)** — Added a recorded NOTE under OQ-2 (no resolution)
  from a BUNDLE-009 scoping session: the trusted-tool allowlist grows paired-per-language
  with the toolchain packs (this bundle + ISSUE-027), not ahead of them; `typescript-
  toolchain` is the live first proof case unblocking backstop-runtime; the TS-support
  division across BUNDLE-009 (traceability) vs BUNDLE-011 (toolchain); and the future
  language-agnostic coverage re-impl rides with/after this bundle (BUNDLE-009 deletes
  `step_coverage.go` without replacement). Mirrored as cross-bundle References entries
  (BUNDLE-009, ISSUE-027). No OQ resolved, no requirements/decisions added, maturity
  unchanged.
- **0.2.0 (2026-06-24, exploring)** — RESOLVED all six open questions (worked one at a
  time with the user) and recorded them as RDQ-1…RDQ-6 in a new **Resolved Design
  Questions** section, preserving each original framing and the rationale. Resolutions:
  RDQ-1 wholesale replace `pkg/check.Run`, gated on a one-shot golden-equivalence
  assertion + same-PR deletion (no standing dual-run); RDQ-2 one `<lang>-toolchain` pack
  per language + WARN-ONLY no-pack baseline that is LOUDLY/DISTINCTLY surfaced (never a
  vacuous green); RDQ-3 delete the already-dead non-Go semgrep catch-all (verified no-op,
  behavior-preserving); RDQ-4 SPEC-035/ISSUE-018/BUNDLE-009 have ALL LANDED (no hard
  external deps), SPEC-034 is half-done (bridge landed, deletion didn't) so this bundle
  ABSORBS its deletion scope + generalizes beyond Go and SPEC-034 is to be SUPERSEDED,
  BUNDLE-009 stays a coordinate-don't-subsume target; RDQ-5 fold the dead
  standards-manifest reader's deletion in as an early task, the `.standard.md` scaffolder
  split out to ISSUE-030; RDQ-6 pull coverage forward and eradicate `step_coverage.go`
  here, re-express the build-pass exemption as a declared gate-type property, mandate a
  CheckType-consumer catalog. Reconciled stale prose: the adjacent-work grounding (all now
  landed), Notes/Ideas, References (SPEC-034 half-done/superseded, SPEC-035/ISSUE-018/
  BUNDLE-009 landed, added ISSUE-030 + SPEC-038), and — critically — CORRECTED the stale
  OQ-2 NOTE claim that "BUNDLE-009 deletes `step_coverage.go` without replacement":
  coverage stayed BAKED (SPEC-038 REQ-009 descoped it, a passing test enforces it), so its
  eradication is now THIS bundle's job (RDQ-6). Refreshed `solution.approach` to carry the
  resolved mechanics. NO requirements / Draft Design Decisions / Spec Seeds added and
  **maturity stays `exploring`** — the user drives promotion separately.
- **0.3.0 (2026-06-24, defined)** — PROMOTED `exploring` → `defined` (user-initiated).
  No new scope, requirements, or decisions were invented — the six resolved questions
  (RDQ-1…RDQ-6) were lifted into the formal structures the `defined` gate requires:
  a 13-entry `requirements[]` array (REQ-001…REQ-013), a new **Draft Requirements**
  summary section, a **Draft Design Decisions** section (DD-1…DD-6, one per RDQ,
  carrying each RDQ's rationale), and a **Spec Seeds** section (Seed 1 dead-code
  prelude → Seed 2 toolchain-pack substrate + `go-toolchain` cutover + golden-equivalence
  harness → Seed 3 coverage re-impl + CheckType-consumer catalog). Traceability:
  RDQ-1→REQ-001/002/003 + Seed 2; RDQ-2→REQ-004/005/006 + Seed 2; RDQ-3→REQ-007 +
  Seed 1; RDQ-4→REQ-008/009 (SPEC-034 marked SUPERSEDED) + Seed 2; RDQ-5→REQ-010 +
  Seed 1; RDQ-6→REQ-011/012/013 + Seed 3. `solution.approach` already carried the
  resolved mechanics, so no rewrite was needed; `version` bumped 0.2.0 → 0.3.0,
  `updated` set to 2026-06-24.
- **0.4.0 (2026-06-24, defined)** — Added a FOURTH Spec Seed and its requirements; the
  coverage PRODUCER side. Seed 3 (SPEC-041) re-implemented the coverage gate to consume
  a language-agnostic per-file `CoverageRecord`, but nothing produces that record today
  (the go-toolchain test engine emits only test-failure SARIF — no `-coverprofile`), so
  Seed 3's consumer contract was unsatisfiable. **Seed 4 — "Coverage production:
  engine-model coverage-records channel + go-toolchain coverage engine"** closes that
  gap: (1) a second engine-model normalized output type, coverage-records as a per-file
  measurement INPUT distinct from SARIF findings; (2) the producer-side
  `{path, covered, total, measured, excluded, metric}` record format (with `total==0`⇒N/A
  and pack-declared `metric` surfaced on report; gate stays metric-blind) that must
  agree with Seed 3's consumer contract; (3) the first concrete producer — a
  `go-toolchain` coverage engine over `go test -coverprofile`. Added **DD-7** recording
  the architectural decision that **SARIF stays the findings/OUTPUT lingua franca while
  coverage is a measurement INPUT, NOT a competing report format** — coverage can't be
  SARIF because non-vacuousness requires distinguishing measured-and-passed from
  not-measured, which SARIF-as-findings (sparse, located failures) cannot express. Added
  **REQ-014** (coverage-records channel + non-SARIF rationale), **REQ-015** (record
  format + conventions), **REQ-016** (go-toolchain coverage engine). Seed 4 depends on
  Seed 2 and is producer↔consumer with Seed 3. Seeds 1–3 and RDQ-1…RDQ-6 left intact.
  `version` 0.3.0 → 0.4.0. **Maturity STAYS `defined`** (this is a scope-evolution, not
  a promotion).
- **0.5.0 (2026-06-28, delivered)** — PROMOTED `defined` → `delivered` (success
  terminal; enabled by ISSUE-031, which added the `delivered` maturity and exempts
  terminal bundles from the defined/ready maturity-section + `requirements[]` gates).
  **What shipped (the CUTOVER scope):** all four seeds are implemented and committed —
  Seed 1 = SPEC-039 (dead-code prelude: deleted the dead standards-manifest reader +
  the no-op non-Go semgrep catch-all); Seed 2 = SPEC-040 (the keystone — the baked
  `realCodeChecker` → `pkg/check.Run` Step 2 and the `builtinToolchain` go/ts stacks
  are GONE from the gate; lint/build/test now run ONLY through `dispatchPackEngines`
  over the bridged `<lang>-toolchain` pack); Seed 3 = SPEC-041 (coverage re-implemented
  as a language-agnostic producer/consumer model, baked `step_coverage.go` enforcement
  replaced); Seed 4 = SPEC-042 (the `go-toolchain` coverage PRODUCER engine over
  `go test -coverprofile`). The gate now runs language-neutral dispatch for
  lint/build/test plus a generic coverage producer/consumer. **HONEST carry-forward —
  what is NOT delivered here (no "fully language-neutral" promise is made):** the
  cutover did NOT fully de-Go the gate CONSUMER side. The `backstop/self` pack
  mechanically flags THREE remaining live Go-specific consumer sites, currently
  grandfathered in the gate baseline: `pkg/gate/step_coverage.go` (the `.go`-only
  measurable-path filter, ~L239), `pkg/gate/step_testverify.go` (the `_test.go`
  test-file walk, ~L254), and `cmd/backstop/gate.go` (`goFilePackageMatchesTarget`'s
  Go-package assumption, ~L908). De-Go-ing those consumers + delivering a
  `typescript-toolchain` pack is CARRIED FORWARD to a new forthcoming bundle and is
  explicitly NOT claimed as done by BUNDLE-011. Captured as `status.note`.
  `version` 0.4.0 → 0.5.0; `bundle.updated` → 2026-06-28.
