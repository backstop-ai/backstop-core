---
title: "Language-Neutral Gate Consumer + TypeScript Toolchain Pack"
number: BUNDLE-012
created: "2026-06-28"
schema_version: bundle/v2

bundle:
  name: language-neutral-consumer-ts-toolchain
  version: "0.2.0"
  updated: "2026-06-28"
  created: "2026-06-28"
  category: infrastructure

status:
  maturity: exploring

problem:
  summary: >
    Backstop bakes a single false assumption — "a project has ONE language, and that
    language implies its file conventions" — in THREE places on the gate's CONSUMER
    spine, and it breaks on non-Go AND on POLYGLOT repos. The three sites are:
    (1) TOOLCHAIN SELECTION via the `language:`-derived bridge that auto-loads a
    `<lang>-toolchain` pack from a single language field;
    (2) the COVERAGE / TEST-VERIFICATION file-classification, which bakes `.go` for the
    measurable-set (`pkg/gate/step_coverage.go` `coverageMeasurablePath`) and `_test.go`
    for test discovery (`pkg/gate/step_testverify.go` `collectTestFuncNamesScoped`); and
    (3) the TRACEABILITY classifier plus the Go-package/`./...` model
    (`cmd/backstop/gate.go` `goFilePackageMatchesTarget`, `coverageSpecRelevantToFile`),
    which also key on `language` / a Go-package identity. The unifying fix is to replace
    ALL of it with PACK-DECLARED file globs/patterns (the consumer reads pack DATA, zero
    baked language knowledge), RETIRE the `language:` field entirely (a single language
    field is simply wrong for a polyglot repo), and treat a TOOLCHAIN AS JUST ANOTHER
    PACK — declared in `backstop.yml packs:` and dispatched uniformly, with a polyglot
    repo declaring multiple toolchain packs. The proof is gating the polyglot Bun/opencode
    stack (the user's backstop-runtime fork) with `backstop/bun-toolchain` as an ORDINARY
    declared pack — going RED when it should and never vacuous-green. The single most
    dangerous nuance: in the DEFAULT diff scope a changed `.ts` file is silently SKIPPED
    by `coverageMeasurablePath`, so coverage is VACUOUS-GREEN for non-Go (worse than an
    honest red — it looks like coverage passed). This is the carry-forward CONSUMER scope
    BUNDLE-011 (DELIVERED 2026-06-28) explicitly left undelivered after finishing the
    PRODUCER side. `backstop/self`'s `no-language-literal-on-neutral-spine` rule
    (ISSUE-024) mechanically flags exactly these consumer sites; they are currently
    grandfathered in `.backstop/baseline.json`. The reference shape for the new toolchain
    pack is the installed `backstop/go-toolchain` (engines golangci / go-build / go-test /
    go-coverage + a coverage-to-records convert).
  user_story: >
    As the maintainer finishing the thin-executor mission, I want backstop to gate a
    polyglot Bun/TypeScript project (concretely: my opencode fork, backstop-runtime)
    packs-only — with the gate going RED when it should and never vacuous-green, with no
    `language:` field and no baked file conventions anywhere on the consumer spine — so
    that the framework's "ZERO baked language knowledge" first principle holds on the
    CONSUMER side of the gate as it now does on the producer side, and the original
    "backstop can't gate TypeScript" gap is closed end to end.
  success_criteria: []

solution:
  approach: >
    EXPLORATORY framing — all five OQs are now USER-RESOLVED (recorded below), but the
    bundle stays `exploring`; the user drives promotion separately, and Draft Requirements
    / Design Decisions / Spec Seeds are authored at promotion time. The work organizes
    into THREE PILLARS:
    PILLAR A — De-Go the gate consumer: the three `backstop/self`-flagged sites
    (`pkg/gate/step_coverage.go` measurable-path; `pkg/gate/step_testverify.go` `_test.go`
    walk; `cmd/backstop/gate.go` `goFilePackageMatchesTarget` + `coverageSpecRelevantToFile`)
    read PACK-DECLARED globs instead of baked extensions, so the consumer is language-blind.
    PILLAR B — Toolchain = just a pack: delete the `language:`-derived bridge
    (`loadBridgedToolchainPacks` / `toolchainPackName` / auto-load); toolchain packs become
    ordinary packs declared in `backstop.yml packs:` and dispatched uniformly. RETIRE the
    `language:` field entirely and rehome the traceability classifier onto pack-declared
    globs. A polyglot repo declares MULTIPLE toolchain packs.
    PILLAR C — Prove on the Bun stack: author `backstop/bun-toolchain` (an ordinary pack,
    its own repo, hand-authored from the proven `go-toolchain` template) and gate the
    opencode fork with it.
    Two STRUCTURAL design decisions were taken (DD-1: pack-declared globs are the consumer
    source of truth — the producer still supplies covered/total numbers; DD-2: a toolchain
    is keyed to the STACK/runtime, e.g. Bun, NOT the language, e.g. TS). The single hazard
    that governs everything is the OQ-1 vacuous-green hole: "non-Go file, not measured" must
    be LOUD, never a silent pass. Two design SUB-QUESTIONS remain open for spec time
    (rehoming the traceability classifier; the multi-metric-per-file coverage model +
    line-vs-branch thresholds).
  assumptions: []
---

# Language-Neutral Gate Consumer + TypeScript Toolchain Pack

## Current Thinking

### The unified thesis

Backstop bakes ONE false assumption in three places: **"a project has ONE language, and
that language implies its file conventions."** It is false twice over — non-Go repos break
on the baked `.go`/`_test.go`/Go-package literals, and POLYGLOT repos break on the very
idea of a single `language:` field. The fix is uniform: replace the baked literals with
**pack-declared file globs/patterns** (the consumer reads pack DATA and stays language-blind),
**retire the `language:` field entirely**, and **treat a toolchain as just another pack** —
declared in `backstop.yml packs:` and dispatched like any other, with a polyglot repo simply
declaring more than one toolchain pack. Proof: gate the polyglot Bun/opencode stack
(backstop-runtime) with `backstop/bun-toolchain` as an ordinary declared pack — RED when it
should be, never vacuous-green. This is the carry-forward **consumer** scope BUNDLE-011
(DELIVERED) left after finishing the **producer** side (language-neutral lint/build/test
dispatch + the coverage producer/consumer model + the `go-toolchain` coverage producer).

### Three pillars

**Pillar A — De-Go the gate consumer.** The three `backstop/self`-flagged sites read
pack-declared globs instead of baked extensions:

1. **`pkg/gate/step_coverage.go` — `coverageMeasurablePath` (~L239).** Returns `false` for any
   path not ending in `.go`. The **critical hazard**: in the **default diff scope** a changed
   `.ts` file is silently skipped → for a TS project the everyday diff gate produces **vacuous
   green** on coverage. `gate --all` enforces via producer records, but the common path does not.
   This is *worse* than an honest red — it looks like coverage passed.
2. **`pkg/gate/step_testverify.go` — `collectTestFuncNamesScoped` (~L254).** Walks `_test.go`
   to collect test-function names; TS test files are `.test.ts` / `.spec.ts` (and the grep
   pattern is itself Go-shaped). For a TS project this finds zero tests → silent non-enforcement.
3. **`cmd/backstop/gate.go` — `goFilePackageMatchesTarget` (~L908)** + the coverage
   spec-relevance helpers **`coverageSpecRelevantToFile` / `packagePathMatches`
   (`step_coverage.go` ~L344/357).** These bake a Go-package identity model and a `./...`-glob /
   directory convention into the substantiveness set-join and the coverage spec-relevance
   derivation. TS has no `package` clause and no `./...`.

**Pillar B — Toolchain = just a pack.** Delete the `language:`-derived bridge
(`loadBridgedToolchainPacks` / `toolchainPackName` / the auto-load from a single language
field). Toolchain packs become ordinary packs declared in `backstop.yml packs:` and dispatched
uniformly. **Retire the `language:` field entirely** — a single language field is wrong for a
polyglot repo. The traceability classifier that consumed `language` is rehomed onto pack-declared
globs (design detail deferred to spec time — see sub-questions). A polyglot repo declares
multiple toolchain packs.

**Pillar C — Prove on the Bun stack.** Author `backstop/bun-toolchain` — an **ordinary pack**,
in its own repo, hand-authored from the proven `go-toolchain` pack.yml template — and gate the
opencode fork with it. Bun-native engines: `oxlint` (lint), `tsc` / bun typecheck (build),
`bun test` (test), `bun test --coverage --coverage-reporter=lcov` (coverage), emitting BOTH
line AND branch metrics in the same `{path, covered, total, measured, excluded, metric}` record
shape the (already language-agnostic) consumer reads. The existing testdata FIXTURE
(`cmd/backstop/testdata/typescript-toolchain/...`) proves only the non-Go dispatch path and has
NO coverage engine; the real Bun pack supplies the producer the fixture omits.

**Why a bundle, not an issue.** The three deletions plus the `language:` retirement are coupled
by one design question — does the consumer derive paths from pack-declared globs, or consume only
the producer's record-set? (Now resolved: DD-1, pack-declared globs.) The work spans a real new
pack (authoring + a line+branch coverage producer), a baseline ratchet → block flip, the
`language:` retirement + classifier rehome, and an end-to-end proof on a foreign-language repo.
It finishes a mission rather than patching a defect.

## Resolved Design Questions

Worked ONE AT A TIME with the user. **All five OQs are RESOLVED.** Maturity stays
`exploring` — the user drives promotion separately, and Draft Requirements / Draft Design
Decisions / Spec Seeds get authored at promotion. Two STRUCTURAL design decisions fell out
of the resolutions (DD-1, DD-2 below). Two new design **sub-questions** remain open for spec
time (see "Open Sub-Questions").

### Structural design decisions

- **DD-1 — Pack-declared globs are the consumer source of truth (the producer supplies the
  numbers).** The `<stack>-toolchain` pack declares source/test file-globs as DATA; the
  consumer reads them (zero baked language knowledge) to decide WHICH files are in scope; the
  coverage PRODUCER still supplies the covered/total numbers per file. Chosen over "producer
  record-set as the sole truth" because a pure record-set DELETES the anti-vacuous-green
  guard — a silently-skipped changed source file would simply pass green — and is
  tool-dependent. Resolves OQ-1; governs all three Pillar-A sites and the toolchain pack's
  contract.
- **DD-2 — A toolchain is keyed to the STACK/runtime, not the language.** The new pack is
  `backstop/bun-toolchain`, NOT a generic "typescript pack": TS spans Node/Deno/Bun with
  different tools, so the toolchain identity is the runtime (Bun), not the language (TS). It is
  an ordinary pack in its own repo, hand-authored from the proven `go-toolchain` template.
  Resolves/reframes OQ-3.

### OQ resolutions

**OQ-1 — Consumer source-of-truth: pack-DECLARED measured-set/test-pattern, or the producer's
emitted record-set as the SOLE truth? → RESOLVED: pack-declared glob patterns (DD-1).** The
`<stack>-toolchain` pack declares source/test file-globs as DATA; the consumer reads them so it
stays language-blind; the coverage producer still supplies the covered/total numbers.
*Rationale:* the pure-record-set alternative DELETES the anti-vacuous-green guard (a silently-
skipped changed source file would pass green) and is tool-dependent. The glob-from-pack approach
keeps the consumer generic AND keeps "non-measured changed source file" a loud, distinct state.

**OQ-2 — Un-grandfathering the self-pack-flagged sites: baseline RATCHET vs explicit waiver
removal? → RESOLVED: baseline ratchet during the fixes, then flip `backstop/self` to `block`
with a ZERO baseline once all three sites are clean.** Each fixed site drops out of the
regenerated baseline (ratchet); once all three are clean, set `backstop/self`'s
`enforcement.policy` to `block` with zero baseline so any FUTURE baked-language regression is
blocked outright. Per-site sequencing by correctness impact: **(1)** coverage measurable-path
(the vacuous-green hole — highest impact), **(2)** test-verify `_test.go` discovery, **(3)** the
go-package / `./...` matchers. *Rationale:* ratchet locks each win as it lands without a flag day;
the terminal `block` + zero baseline converts the dogfood rule from "grandfathered backlog" to a
real wall.

**OQ-3 — TS toolchain pack v1: engine set + coverage tool + metric? → RESOLVED + REFRAMED (DD-2):
it is `backstop/bun-toolchain`, NOT a generic "typescript" pack.** A toolchain is keyed to the
STACK/runtime (Bun), not the language (TS spans Node/Deno/Bun with different tools). Ordinary
pack, its own repo, hand-authored from the `go-toolchain` pack.yml template. Bun-native engines:
`oxlint` (lint) · `tsc` / bun typecheck (build) · `bun test` (test) · `bun test --coverage
--coverage-reporter=lcov` (coverage). Coverage emits BOTH line AND branch metrics. *Rationale:*
"typescript pack" would wrongly imply one toolchain across incompatible runtimes; keying on the
runtime makes the pack honest and lets a polyglot repo declare exactly the runtimes it uses.
*Carries a new design sub-question* (line+branch ⇒ multiple metrics per file) — captured below,
NOT resolved here.

**OQ-4 — Prove-it target: minimal in-repo TS fixture FIRST, or straight at the opencode fork? →
RESOLVED: BOTH, placed deliberately.** **(a)** An IN-REPO STATIC testdata fixture in backstop-core
(extends `cmd/backstop/testdata/typescript-toolchain/`): pack.yml + a few `.ts` files + a
PRE-CAPTURED `bun … lcov` output, exercised with the runner STUBBED — proves the language-neutral
consumer + glob classification + line/branch coverage-record parsing with ZERO `bun` dependency in
the Go CI (not smelly — it's data). **(b)** An EXTERNAL executed gate against a REAL Bun project
(the opencode fork, and/or a small example repo) with the real `bun` / `oxlint` toolchain in ITS
OWN CI — a REQUIRED acceptance criterion. Keep the real toolchain OUT of backstop-core's Go CI.
*Rationale:* the static fixture alone would repeat the stubbed-testdata integration-gap that bit
SPEC-035 / SPEC-037 (everything green over a stub, real installed-pack end-to-end path unproven);
the external executed gate is the load-bearing proof, the in-repo fixture is the cheap
deterministic regression guard.

**OQ-5 — Is the minimal pack-CLI "authoring loop" fix a DEPENDENCY of this bundle, or a separate
issue? → RESOLVED: NOT a hard dependency — separate Track-B follow-up.** `pack add` / install
works, so create `backstop/bun-toolchain` in its OWN repo hand-authored from the go-toolchain
template and `pack add` it onto the fork. File the `pack new` / engine-pack-scaffold reboot (and
the rest of the stale pack-authoring CLI) as a SEPARATE follow-up issue (Track B), not in this
bundle's critical path. *Honest caveat:* this deliberately skips "scaffold via CLI" because the
scaffolder ITSELF is the broken thing — hand-authoring from a proven template is the correct
workaround until Track B reboots `pack new`.

## Open Sub-Questions

These are design details surfaced by the resolutions above. They are intentionally **NOT**
resolved here — they belong to spec time.

**SQ-1 — Rehome the traceability classifier off `language:` onto pack-declared globs.** The
`language:` field is decided-retired (Pillar B / DD-2-adjacent), but the traceability classifier
that consumed it must be rehomed onto pack-declared globs. Exactly how the classifier reads the
glob set (per-toolchain-pack? a merged union across declared packs? precedence on overlap in a
polyglot repo?) is a spec-time design detail.

**SQ-2 — Coverage model: multiple metrics per file + same-vs-per-metric thresholds.** Supporting
line AND branch (OQ-3) means the coverage model must hold MULTIPLE metrics per file. Today
`indexCoverageByPath` is one-record-per-path; it must key by **(path, metric)**. Open: do line vs
branch get the SAME threshold or PER-METRIC thresholds (branch coverage is normally held to a
lower bar)? This shapes the record schema, the index, and the pack's declared threshold surface.

## Notes / Ideas

- **The vacuous-green hazard in OQ-1 is the single most important thing not to get
  wrong.** A `.ts` file silently skipped by `coverageMeasurablePath` in the default
  diff scope means a TS project's coverage gate passes without measuring anything —
  exactly the silent/vacuous-green failure mode the enforcement philosophy treats as
  the enemy. Whatever OQ-1 resolves to, "non-Go file, not measured" must be LOUD, not
  a quiet pass.
- **The go-toolchain pack is the literal template for the bun-toolchain pack.** Same
  engine-block shape, same `gate_type: coverage` routing to the coverage-records channel, same
  pack-relative `convert` script pattern (`coverage-to-records.sh`). The bun pack's coverage
  engine should emit the SAME `{path, covered, total, measured, excluded, metric}` record shape
  so the (already language-agnostic) consumer needs no per-language branch — but with line AND
  branch metrics (SQ-2: that forces a (path, metric)-keyed index).
- **The testdata fixture already proves the non-Go DISPATCH path works** (SPEC-040's
  language-agnostic resolve-from-disk). What it does NOT prove is the consumer side or
  coverage — which is precisely this bundle's gap, and why OQ-4 keeps BOTH the in-repo static
  fixture (consumer + glob + line/branch parsing, runner stubbed) and the external executed
  gate (the real installed-pack end-to-end proof).
- **DD-1 keeps the anti-vacuous-green guard alive.** The pure-record-set alternative would have
  let a silently-skipped changed source file pass green; pack-declared globs preserve a loud,
  distinct "changed source file, not measured" state — the enforcement-philosophy enemy is
  exactly the silent pass.
- **Keep the real bun toolchain OUT of backstop-core's Go CI.** The static testdata fixture
  carries a PRE-CAPTURED lcov output as data; the real `bun`/`oxlint` execution lives in the
  fork's own CI (OQ-4). This avoids a `bun` dependency in the Go build while still getting an
  executed end-to-end proof.

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
  assume the pre-engine model and were only partially modernized by BUNDLE-011. Resolved as
  a SEPARATE Track-B follow-up (below), NOT this bundle's critical path.
- [[project_baseline_ci_pull]] / [[project_baseline_design]] — the OQ-2 ratchet flow
  (CI-generated immutable baseline, pull on demand), terminating in the `backstop/self`
  `block` + zero-baseline flip.
- BUNDLE-011 (collapse-legacy-codecheck-into-packs) — DELIVERED; this bundle is its
  explicitly-carried-forward CONSUMER scope.
- ISSUE-024 (thin-executor-absence-rule-dogfood) — the `backstop/self`
  `no-language-literal-on-neutral-spine` rule that mechanically flags the consumer sites;
  its include-list IS the language-neutral spine contract.
- [[feedback_zero_baked_checks]] — the thin-executor first principle this bundle finishes on
  the consumer side: ZERO baked language/tool knowledge; a toolchain is just another pack.
- **Track-B follow-up to FILE (separate issue, NOT this bundle):** "pack-CLI authoring-loop
  reboot — `pack new` emits a real engine-pack pack.yml; `pack check`/`pack test` accept engine
  packs; `artifact new` stamps the current schema version." Pairs with the pack-CLI-stale
  concern / ISSUE-030 area. Resolves OQ-5's deferred work.
- Code (verified 2026-06-28, this branch): **Pillar A** —
  `pkg/gate/step_coverage.go` (`coverageMeasurablePath` ~L239,
  `coverageSpecRelevantToFile`/`packagePathMatches` ~L344/357),
  `pkg/gate/step_testverify.go` (`collectTestFuncNamesScoped` ~L254),
  `cmd/backstop/gate.go` (`goFilePackageMatchesTarget` ~L908). **Pillar B** — the
  `language:`-derived bridge to DELETE (`loadBridgedToolchainPacks` / `toolchainPackName` /
  the auto-load), the `language:` field to RETIRE, and the traceability classifier to rehome
  (SQ-1). **SQ-2** — `indexCoverageByPath` (one-record-per-path, must become (path, metric)).
  **Reference shape** — `.backstop/packs/backstop/go-toolchain/pack.yml` (`go-coverage` engine
  + `scripts/coverage-to-records.sh`). **Fixture to extend** —
  `cmd/backstop/testdata/typescript-toolchain/...` (lint/build/test, NO coverage). **Dogfood
  rule + baseline** — `backstop/self` `no-language-literal-on-neutral-spine`;
  `.backstop/baseline.json` (current grandfathering of the flagged sites).

## Version History

- **0.2.0 (2026-06-28, exploring)** — User worked through all five OQs and reshaped the bundle's
  framing. **Reframed the problem to the UNIFIED THESIS:** backstop bakes one false assumption —
  "a project has ONE language, and that language implies its file conventions" — in three places
  (toolchain selection via the `language:` bridge; coverage/test file-classification; the
  traceability classifier), breaking on non-Go AND polyglot repos; the fix is pack-declared
  globs + retire `language:` entirely + toolchain-is-just-a-pack, proven on the polyglot
  Bun/opencode stack. **Restructured into THREE PILLARS** (A: de-Go the consumer sites; B:
  toolchain = just a pack, delete the bridge, retire `language:`, rehome the classifier; C:
  author `backstop/bun-toolchain` and gate the fork). **Recorded two STRUCTURAL design decisions:**
  DD-1 (pack-declared globs are the consumer source of truth; the producer still supplies the
  numbers) and DD-2 (a toolchain is keyed to the STACK/runtime — Bun — not the language — TS).
  **Marked all five OQs RESOLVED with rationale:** OQ-1→pack-declared globs (DD-1); OQ-2→baseline
  ratchet during fixes then flip `backstop/self` to `block` with zero baseline, sequenced by
  correctness impact (coverage hole first); OQ-3→`backstop/bun-toolchain` (DD-2) with oxlint/tsc/
  bun test + `bun test --coverage … lcov` emitting line AND branch; OQ-4→BOTH an in-repo static
  stubbed fixture AND a REQUIRED external executed gate on a real Bun project (the fork), real
  toolchain kept out of Go CI; OQ-5→pack-CLI authoring loop is NOT a hard dependency, hand-author
  from the go-toolchain template + `pack add`, file a separate Track-B follow-up. **Captured two
  new OPEN sub-questions for spec time** (SQ-1: rehome the traceability classifier onto
  pack-declared globs; SQ-2: multi-metric-per-file coverage model keyed by (path, metric) +
  same-vs-per-metric line/branch thresholds). Maturity HELD at `exploring` — user drives promotion;
  Draft Requirements / Draft Design Decisions / Spec Seeds authored at promotion time.
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
