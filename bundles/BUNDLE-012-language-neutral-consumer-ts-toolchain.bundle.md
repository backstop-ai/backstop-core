---
title: "Language-Neutral Gate Consumer + TypeScript Toolchain Pack"
number: BUNDLE-012
created: "2026-06-28"
schema_version: bundle/v2

bundle:
  name: language-neutral-consumer-ts-toolchain
  version: "0.1.0"
  created: "2026-06-28"
  category: infrastructure

status:
  maturity: exploring

problem:
  summary: >
    BUNDLE-011's cutover (DELIVERED 2026-06-28) made the gate's lint/build/test
    PRODUCER side and a coverage producer/consumer model run language-neutrally:
    Step 2 now dispatches the declared `<lang>-toolchain` pack through
    `dispatchPackEngines` and a generic coverage record producer exists
    (`go-toolchain`'s `go-coverage` engine over `go test -coverprofile`). But the
    gate CONSUMER side still bakes Go conventions, and that is the live blocker on
    the ORIGINAL mission: using `backstop gate` to gate a TypeScript project — the
    user's opencode fork (backstop-runtime), which backstop could never gate
    precisely because it is TypeScript. The `backstop/self` dogfood pack
    (rule `no-language-literal-on-neutral-spine`, the ISSUE-024 detection rule)
    mechanically flags exactly THREE live consumer sites, all currently
    grandfathered in the gate baseline (`.backstop/baseline.json`):
    (1) `pkg/gate/step_coverage.go` (`coverageMeasurablePath`, ~L239) hardcodes
    `.go` for the measurable-set — and the CRITICAL nuance is that in the DEFAULT
    diff scope a changed `.ts` file is silently SKIPPED, so coverage is
    VACUOUS-GREEN for non-Go (worse than broken: it looks green); `gate --all`
    enforces via producer records but the everyday diff gate does not.
    (2) `pkg/gate/step_testverify.go` (`collectTestFuncNamesScoped`, ~L254) walks
    `_test.go`, but TS test files are `.test.ts` / `.spec.ts`.
    (3) `cmd/backstop/gate.go` (`goFilePackageMatchesTarget`, ~L908) plus
    `coverageSpecRelevantToFile` (`step_coverage.go` ~L344/357) bake a Go-package
    + `./...`-glob model. SEPARATELY, there is NO real `backstop/typescript-toolchain`
    pack — only a testdata FIXTURE (`cmd/backstop/testdata/typescript-toolchain/...`)
    that mirrors the go-toolchain lint/build/test shape but has NO coverage
    engine/convert. The reference shape is the installed `backstop/go-toolchain`
    pack (engines golangci / go-build / go-test / go-coverage + a
    coverage-to-records convert). This bundle is the carry-forward consumer scope
    that BUNDLE-011 explicitly left undelivered.
  user_story: >
    As the maintainer finishing the thin-executor mission, I want backstop to gate
    a TypeScript project (concretely: my opencode fork, backstop-runtime) packs-only
    — with the gate going RED when it should and never vacuous-green — so that the
    framework's "ZERO baked language knowledge" first principle holds on the
    CONSUMER side of the gate as it now does on the producer side, and the original
    "backstop can't gate TypeScript" gap is closed end to end.
  success_criteria: []

solution:
  approach: >
    EXPLORATORY — the end-state below is the bundle's target, NOT a resolved design.
    The desired end-state is threefold: (1) De-Go the three `backstop/self`-flagged
    consumer sites so the gate consumer is language-blind, and un-grandfather them
    from the baseline as each is fixed (ratchet). (2) Deliver a REAL
    `backstop/typescript-toolchain` pack (eslint / tsc / jest|vitest + a
    coverage→records convert mirroring go-toolchain's `coverage-to-records.sh`),
    promoting the testdata fixture into an authored, installable pack with the
    coverage producer the fixture lacks. (3) PROVE `backstop gate` goes
    RED-when-it-should on a TypeScript project — a small in-repo TS fixture first,
    then the opencode fork. The central unknown (OQ-1) is whether the consumer sites
    should consume a pack-DECLARED measured-set / test-file-pattern or whether the
    producer's emitted coverage record-set is the SOLE source of truth (the codebase
    principle leans toward the latter — the producer decides the measured set — but
    it stays OPEN). The mechanics (ratchet vs waiver, TS engine set + coverage tool
    + metric, prove-it target, and whether the pack-CLI authoring loop is a hard
    dependency) are the open questions below. Requirements, design decisions, and
    spec seeds follow when the USER resolves the OQs and promotes; maturity stays
    `exploring`.
  assumptions: []
---

# Language-Neutral Gate Consumer + TypeScript Toolchain Pack

## Current Thinking

BUNDLE-011 finished the gate's **producer** side and explicitly carried forward the
**consumer** side. This bundle is that carry-forward. The framing is deliberately
narrow: BUNDLE-011 already settled the *principle* (zero baked language knowledge;
everything from packs) and shipped the producer/consumer **coverage MODEL**, the
`go-toolchain` coverage **producer**, and the language-neutral lint/build/test
dispatch. What remains baked is a small, **mechanically enumerated** set of consumer
assumptions plus the absence of a real second-language toolchain pack to prove the
whole thing on.

**The three baked consumer sites (verified on this branch, all currently
baseline-grandfathered under `no-language-literal-on-neutral-spine`):**

1. **`pkg/gate/step_coverage.go` — `coverageMeasurablePath` (~L239).** Returns
   `false` for any path not ending in `.go`. The comment already concedes the
   design tension: "Non-Go source files are measured only when the producer emits a
   record for them; they are not synthesized as required paths here." The **critical
   hazard**: in the **default diff scope** a changed `.ts` file is silently skipped
   → for a TypeScript project the everyday diff gate produces **vacuous green** on
   coverage. `gate --all` enforces via producer records, but the common path does
   not. This is *worse* than an honest red — it looks like coverage passed.

2. **`pkg/gate/step_testverify.go` — `collectTestFuncNamesScoped` (~L254).** Walks
   the tree for files ending `_test.go` to collect test-function names. TS test
   files are `.test.ts` / `.spec.ts` (and the function-name grep pattern is itself
   Go-shaped). For a TS project this step finds zero tests → another silent
   non-enforcement.

3. **`cmd/backstop/gate.go` — `goFilePackageMatchesTarget` (~L908)** and the
   coverage spec-relevance helpers **`coverageSpecRelevantToFile` /
   `packagePathMatches` (`step_coverage.go` ~L344/357).** These bake a Go-package
   identity model (`package <name>` clause parsing) and a `./...`-glob / directory
   convention into the substantiveness set-join and the coverage spec-relevance
   derivation. TS has no `package` clause and no `./...`.

**The TS toolchain pack gap.** There is a `backstop/go-toolchain` pack installed
(`.backstop/packs/backstop/go-toolchain`) with four engines — `golangci`,
`go-build`, `go-test`, and `go-coverage` (the BUNDLE-011/SPEC-042 producer running
`go test -coverprofile` through `scripts/coverage-to-records.sh`). For TypeScript
there is only a **testdata FIXTURE**
(`cmd/backstop/testdata/typescript-toolchain/.backstop/packs/backstop/typescript-toolchain`)
that declares `ts-lint` (eslint), `ts-build` (tsc), `ts-test` (npm test) with
convert scripts — but **no coverage engine and no coverage convert**, and its own
pack.yml says a "real authored typescript-toolchain pack is an ISSUE-027 follow-up."
So even once the consumer sites are de-Go'd, there is nothing to gate a real TS
project *with* until that pack exists — including the coverage producer the fixture
omits.

**Why a bundle, not an issue.** The three deletions look small, but they are coupled
by a single unresolved design question (OQ-1): does the consumer derive paths from a
pack-declared pattern, or does it stop synthesizing paths at all and consume only the
producer's emitted record-set? That choice changes what each of the three sites
becomes, changes the TS pack's responsibilities (must it declare a `measured-set` /
`test-file-pattern`, or only emit records?), and changes how the prove-it fixture is
built. Plus the work spans a real new pack (authoring + coverage producer), a baseline
ratchet policy, and an end-to-end proof on a foreign-language repo — more than an
issue's worth, and it finishes a mission rather than patching a defect.

## Open Questions

Worked ONE AT A TIME with the user. None resolved yet — maturity stays `exploring`.

**OQ-1 — Consumer source-of-truth: pack-DECLARED measured-set/test-pattern, or the
producer's emitted record-set as the SOLE truth?** Today `coverageMeasurablePath`
*synthesizes* required coverage paths from a file extension (`.go`), and
`collectTestFuncNamesScoped` *synthesizes* the test set by walking `_test.go`. Two
broad directions:
  - **(a) Pack-declared conventions.** The consumer keys off pack-declared data — a
    `measured-set` glob / `test-file-pattern` the `<lang>-toolchain` pack declares
    (e.g. TS declares `**/*.ts`, `**/*.{test,spec}.ts`). The consumer stays generic
    but still *synthesizes* the expected set, now from pack data instead of a baked
    literal.
  - **(b) Producer record-set is the sole truth.** The consumer NEVER synthesizes
    paths from extensions/patterns; it consumes only what the producer EMITTED
    (coverage records for coverage; an emitted test inventory for test-verify). The
    gate compares "what was measured/declared" against "what the producer reported,"
    and "not measured" is a loud, distinct state — never silent green.
  - *Lean (NOT a resolution — for the user to decide):* the codebase principle in
    `step_coverage.go`'s own comment and the BUNDLE-011 coverage model lean toward
    **(b)** — "the producer, not the gate, decides the measured set." But (b) has a
    real cost for `step_testverify` (today it has no producer-emitted test inventory;
    (b) would require the toolchain pack to emit one), and a hybrid is possible
    (coverage→(b), test-verify→(a)). This is the load-bearing question that shapes
    all three sites and the TS pack's contract.

**OQ-2 — Un-grandfathering the self-pack-flagged sites: baseline RATCHET vs explicit
waiver removal?** As each of the three sites is de-Go'd it must stop being
grandfathered, or the gate stays vacuously quiet on regressions. Options:
  - **(a) Ratchet the baseline** — regenerate/shrink `.backstop/baseline.json` so the
    fixed site's entries drop out, relying on the CI-generated immutable baseline
    flow (DIR-012 / BUNDLE-007) to lock the new clean state.
  - **(b) Explicit per-site waiver removal** — if these are tracked as named waivers
    rather than fingerprinted baseline entries, remove the waiver as the site is
    fixed.
  - *Open:* depends on how the three sites are currently represented (fingerprinted
    baseline entries vs named waivers) and on the per-site sequencing (do all three
    land together, or one per spec with an interim partial-ratchet?).

**OQ-3 — TS toolchain pack v1: engine set + coverage tool + metric?** The fixture
declares eslint / tsc / npm-test but no coverage. For the REAL pack:
  - **(a) Coverage tool:** `jest --coverage` vs `vitest --coverage` (vs c8/nyc as a
    tool-agnostic istanbul producer). The opencode fork's actual test runner should
    probably decide this.
  - **(b) Coverage metric:** line vs branch vs statement. BUNDLE-011's coverage record
    carries a pack-declared `metric` label that the gate surfaces (gate stays
    metric-blind), so the pack just declares it — but v1 has to pick one and a polyglot
    repo must not silently compare TS-branch% against Go-statement%.
  - **(c) Build/test engines:** is `tsc --noEmit` + the project test runner the right
    v1 set, and does `npm test` vs a direct `vitest`/`jest` invocation matter for the
    SARIF convert?
  - *Open:* needs a look at backstop-runtime's actual toolchain before committing.

**OQ-4 — Prove-it target: minimal in-repo TS fixture FIRST, or straight at the
opencode fork?** The acceptance is "gate goes RED when it should on a TS project."
  - **(a) In-repo fixture first** — a tiny committed TS project (a passing case and a
    deliberately-failing case per dimension: a lint error, a build error, a failing
    test, an under-covered file) as a deterministic regression fixture, THEN run
    against the fork.
  - **(b) Straight at the fork** — prove it on backstop-runtime directly; more real,
    less deterministic, slower to iterate, and couples this bundle to the fork's state.
  - *Open:* (a) is the usual backstop pattern (deterministic fixture + fail-loud), but
    the user may want the fork proof as the actual acceptance bar.

**OQ-5 — Is the minimal pack-CLI "authoring loop" fix a DEPENDENCY of this bundle, or
a separate issue?** Delivering a REAL `typescript-toolchain` pack means exercising
`pack new → pack add → pack update/relock → pack test` for a local engine pack. The
broken local-pack relock was hit MANUALLY this session, and the pack-CLI is known
stale (it assumes the pre-engine `.standard.md` model). So:
  - **(a) Hard dependency** — fold the minimal authoring-loop fix into this bundle's
    scope, because you can't durably author/install the TS pack without it.
  - **(b) Separate issue** — keep the authoring-loop fix as its own work (pairs with
    the existing pack-CLI-stale concern / ISSUE-030 area) and author the TS pack
    around the gaps for now.
  - *Open:* hinges on how blocking the relock breakage actually is for producing an
    installable pack vs working around it manually once.

## Notes / Ideas

- **The vacuous-green hazard in OQ-1 is the single most important thing not to get
  wrong.** A `.ts` file silently skipped by `coverageMeasurablePath` in the default
  diff scope means a TS project's coverage gate passes without measuring anything —
  exactly the silent/vacuous-green failure mode the enforcement philosophy treats as
  the enemy. Whatever OQ-1 resolves to, "non-Go file, not measured" must be LOUD, not
  a quiet pass.
- **The go-toolchain pack is the literal template for the TS pack.** Same engine-block
  shape, same `gate_type: coverage` routing to the coverage-records channel, same
  pack-relative `convert` script pattern (`coverage-to-records.sh`). The TS pack's
  coverage engine should emit the SAME `{path, covered, total, measured, excluded,
  metric}` record shape so the (already language-agnostic) consumer needs no per-language
  branch.
- **The testdata fixture already proves the non-Go DISPATCH path works** (SPEC-040's
  language-agnostic resolve-from-disk). What it does NOT prove is the consumer side or
  coverage — which is precisely this bundle's gap.
- **OQ-1(b) asymmetry:** coverage already has a producer record-set to consume;
  test-verify does not. A clean (b) for test-verify implies the toolchain pack emits a
  test inventory, which is new producer surface — worth weighing against a pack-declared
  `test-file-pattern` (a) for that one site only.

## References

- [[project_native_toolchain_cutover]] — BUNDLE-011 is the delivered producer-side
  cutover; this bundle is its explicitly-carried-forward consumer scope.
- [[feedback_zero_baked_checks]] — the standing first principle this bundle finishes on
  the consumer side: ZERO baked language/tool knowledge; everything from packs.
- [[feedback_loud_not_blocking]] — governs the OQ-1 vacuous-green hazard: "not measured"
  must be loud, never a silent green.
- [[project_toolchain_pack_convention]] — one `<lang>-toolchain` pack per language;
  the TS pack follows the go-toolchain shape.
- [[project_pack_cli_stale]] — the OQ-5 authoring-loop concern: `pack new/test/relock`
  assume the pre-engine model and were only partially modernized by BUNDLE-011.
- [[project_baseline_ci_pull]] / [[project_baseline_design]] — the OQ-2 ratchet flow
  (CI-generated immutable baseline, pull on demand).
- BUNDLE-011 (collapse-legacy-codecheck-into-packs) — DELIVERED; its `status.note`
  names these three consumer sites as the carry-forward this bundle owns.
- ISSUE-024 (thin-executor-absence-rule-dogfood) — the `backstop/self`
  `no-language-literal-on-neutral-spine` rule that mechanically flags the three sites;
  its include-list IS the language-neutral spine contract.
- ISSUE-027 (eradicate-default-registry-into-packs) — names the "real authored
  typescript-toolchain pack" as a follow-up; this bundle is where that pack gets
  authored (coordinate the seam).
- Code (verified 2026-06-28, this branch): `pkg/gate/step_coverage.go`
  (`coverageMeasurablePath` ~L239, `coverageSpecRelevantToFile`/`packagePathMatches`
  ~L344/357); `pkg/gate/step_testverify.go` (`collectTestFuncNamesScoped` ~L254);
  `cmd/backstop/gate.go` (`goFilePackageMatchesTarget` ~L908);
  `.backstop/packs/backstop/go-toolchain/pack.yml` (reference shape, `go-coverage`
  engine + `scripts/coverage-to-records.sh`);
  `cmd/backstop/testdata/typescript-toolchain/.backstop/packs/backstop/typescript-toolchain/pack.yml`
  (fixture: lint/build/test, NO coverage); `backstop/self/rules/no-baked.yml`
  (`no-language-literal-on-neutral-spine`); `.backstop/baseline.json` (current
  grandfathering of the three sites).

## Version History

- **0.1.0 (2026-06-28, exploring)** — Initial bundle. Problem framing grounded in the
  verified branch state: BUNDLE-011 delivered the gate PRODUCER side (language-neutral
  lint/build/test dispatch + a coverage producer/consumer model + the `go-toolchain`
  coverage producer) and explicitly carried forward the CONSUMER side. This bundle owns
  that carry-forward: de-Go the three `backstop/self`-flagged consumer sites
  (`coverageMeasurablePath` `.go`-only with the default-diff-scope VACUOUS-GREEN hazard;
  `collectTestFuncNamesScoped` `_test.go` walk; `goFilePackageMatchesTarget` +
  `coverageSpecRelevantToFile` Go-package/`./...` model), deliver a REAL
  `backstop/typescript-toolchain` pack (eslint/tsc/jest|vitest + coverage→records
  convert — the testdata fixture has no coverage), and PROVE `backstop gate` goes
  RED-when-it-should on a TS project (in-repo fixture, then the opencode fork). Recorded
  FIVE open questions (OQ-1 consumer source-of-truth: pack-declared measured-set/
  test-pattern vs producer record-set as sole truth — the load-bearing one, with the
  vacuous-green hazard; OQ-2 un-grandfathering via baseline ratchet vs waiver removal;
  OQ-3 TS pack v1 engine set + coverage tool jest/vitest + metric; OQ-4 prove-it target
  in-repo fixture vs the fork; OQ-5 whether the pack-CLI authoring-loop fix is a hard
  dependency). NO open question pre-resolved. No requirements, design decisions, or spec
  seeds yet — those follow OQ resolution and user-driven promotion. Maturity stays
  `exploring`.
