---
title: "Collapse the Legacy `pkg/check` Engine into Pack-Declared Toolchain Packs"
number: BUNDLE-011
created: "2026-06-21"
schema_version: bundle/v2

bundle:
  name: collapse-legacy-codecheck-into-packs
  version: "0.1.1"
  created: "2026-06-21"
  updated: "2026-06-22"
  category: infrastructure

status:
  maturity: exploring

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
    To be determined through open-question resolution. The endgame is settled: gate
    Step 2 becomes "run the toolchain packs the project declares" — nothing baked —
    and `pkg/check` is retired as a second enforcement engine. The OPEN work is the
    mechanics and sequencing of getting there: whether Step 2 is replaced wholesale or
    dual-run behind a parity gate before deletion; which pack(s) cover today's default
    routing and what a repo with no toolchain pack should do; what replaces the
    semgrep catch-all on non-Go files; how this bundle orders against and divides work
    with SPEC-034 (the go-toolchain bridge/cutover template), SPEC-035 (pack-declared
    engine bindings + trusted allowlist), ISSUE-018 (dead in-process semgrep + native-
    standards deletion), and BUNDLE-009 (traceability analyzers → packs); whether the
    dead standards-manifest reader is deleted here or as a separate quick pass; and how
    the gate's other steps that consume `pkg/check`'s `CheckType` outputs (coverage,
    test results) are re-routed. These are surfaced as OQ-1…OQ-6 below and are NOT
    pre-resolved.
  assumptions: []

requirements: []
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

**Grounding on the adjacent in-flight work (verified 2026-06-21):**

- **SPEC-034 (native-toolchain-engine-cutover)** is `status: draft` with a plan
  (`PLAN-SPEC-034-…`) and TDD tests already on `main` (`cutover_deletion_test.go`,
  `bridge_test.go` referencing the `go-toolchain` testdata pack) — but **no impl
  commits have landed**: `realCodeChecker` → `check.Run` and `goBuiltinExecutors`
  are still live. SPEC-034 is the **template and likely the first step**, but it is
  scoped to the *Go* toolchain arm, not the whole engine. Whether this bundle
  *subsumes*, *depends on*, or *follows* SPEC-034 is OQ-4.
- **SPEC-035 (pack-declared-engines-trusted-allowlist)** is `status: draft`. It
  moves engine *bindings* out of hardcoded Go into a pack `engines:` block, adds a
  trusted-tool allowlist (the security substrate), and replaces the tool-named
  `CheckTypeSemgrep` with a tool-NEUTRAL gate-TYPE enum. It feeds the *pack-engine*
  dispatch, not `pkg/check`'s Step 2. It looks like a **prerequisite substrate** —
  a project can't *declare* a toolchain pack's engine commands safely until the
  allowlist/binding mechanism exists — but that ordering is OQ-4.
- **ISSUE-018** deletes the dead in-process `semgrepExecutor` body AND the dead
  native-standards *validator* (`pkg/validate/standard.go`). Crucially it
  **stops short of retiring `pkg/check.Run` as Step 2** — it leaves the lint/build/
  test pass-order, the `CheckType` enum, and `pkg/check/manifest.go`'s routing
  (`routeFileDefaults`, `languageExtensions`, the standards-manifest reader)
  intact. So even after ISSUE-018 + SPEC-035 land, the baked Step 2 engine remains.
  **That residual is exactly this bundle's target.** Whether the dead
  standards-manifest *reader* in `manifest.go` (distinct from the validator
  ISSUE-018 kills) folds into this cutover or is a separate quick deletion is OQ-5.
- **BUNDLE-009 (stack-aware-traceability)** is `exploring`; its analyzer-eradication
  seeds depend on SPEC-035 and on the pack-engine substrate. It is a sibling
  "stop baking rules into the binary" effort focused on the *traceability* analyzers
  (substantiveness, contracts), distinct from this bundle's *code-check* engine. OQ-4
  must decide whether to coordinate or just stay out of each other's way.

The load-bearing reframe: **this bundle is where "Step 2" stops being a noun for a
baked engine and becomes a verb — "run the declared toolchain packs."** Everything
else (SPEC-034/035, ISSUE-018) clears the ground; this bundle removes the last baked
engine standing on it.

## Open Questions

Worked one at a time with the user. None are pre-resolved.

**OQ-1 — Cutover shape: wholesale replace vs dual-run-with-parity-gate then delete.**
Does gate Step 2 get replaced *wholesale* (rip `pkg/check.Run` out, route
lint/build/test through `dispatchPackEngines` as declared toolchain-pack passes in
one move), or does the legacy engine and the pack-engine path *dual-run* during a
transition window behind a **parity gate** (assert both produce the same violations
on this repo) before the legacy path is deleted?
- (a) **Wholesale** — smallest diff, no parity scaffolding; relies on the existing
  TDD tests (`cutover_deletion_test.go`, `bridge_test.go`) as the safety net. Fits
  the agent-native "churning tested code is safe" posture and the N=1 / no-external-
  base reality.
- (b) **Dual-run + parity gate, then delete** — proves equivalence empirically
  before deletion; costs temporary scaffolding and a defined "parity proven" exit
  criterion. Sub-question: *what* proves parity (same `[]Violation` set on the
  backstop repo? a fixture corpus? a CI diff job?), and is that worth building for a
  population of one repo?
- *Lean (not a resolution):* given the existing deletion/bridge tests and the
  no-external-base argument that made BUNDLE-010's migration a flag-day, (a) looks
  proportionate — but the parity *evidence* question in (b) is real and may be
  satisfiable cheaply (a one-shot assertion test) without a full dual-run window.
  Needs the user's call.

**OQ-2 — Pack coverage of today's routing + the no-toolchain-pack baseline.** Today
the baked default routes `.go/.ts/.tsx` → lint/build/test/semgrep and runs even when
the project declares nothing. After cutover, what pack(s) supply that?
- Is it **one `<lang>-toolchain` pack per language** (per the toolchain-pack
  convention — `go-toolchain`, `typescript-toolchain`, …), each bundling its native
  lint/build/test as Layer-0 engine passes?
- The sharp edge: **what happens for a repo that declares NO toolchain pack?** Today
  the baked default *always* runs lint/build/test/semgrep. After cutover, "no pack →
  no checks" means a repo with no toolchain pack gets a **green gate that ran
  nothing** — the exact silent/vacuous-green the enforcement philosophy forbids. Is
  "no pack → no checks" acceptable (with a loud warn that nothing ran), or is there a
  **required baseline** the gate refuses to pass without (e.g. "declare a toolchain
  pack or explicitly opt out")? This is the central correctness question of the whole
  cutover.
- *Lean (not a resolution):* per-language toolchain pack matches the established
  convention; the no-pack case almost certainly must be **loud** (a config error or
  forced acknowledgment), not silently green — but whether "loud" means block or
  warn-with-guidance is a [[feedback_loud_not_blocking]] judgment the user owns.
- **NOTE (recorded 2026-06-22, from a BUNDLE-009 scoping session — context, NOT a
  resolution of this OQ; the user drives the toolchain-pack call when they work this
  bundle):**
  1. *The trusted-tool allowlist explodes TOGETHER WITH the toolchain packs, not ahead
     of them.* The backstop-owned trusted-tool allowlist (the trust floor, the security
     substrate from SPEC-035) today holds only **semgrep + ast-grep**. Populating it
     (eslint/tsc/vitest/ruff/cargo/…) is **inert** without a pack that declares an engine
     *using* each tool — allowlisting a tool pre-permits nothing until a pack uses it. So
     the allowlist entries and the toolchain pack that needs them **ship as a PAIR, per
     language**. This pairing belongs to **this bundle (+ ISSUE-027)**, not to any
     consumer bundle (e.g. BUNDLE-009 does not own allowlist growth).
  2. *TypeScript is the FIRST PROOF CASE and a LIVE PRIORITY — not hypothetical.* The
     runtime (**backstop-runtime**, TypeScript) is currently **BLOCKED** by the
     half-baked pack system: it cannot gate itself with packs because there is no
     pack-based TS toolchain support — and the existing **baked TS built-in in
     `pkg/check`** (eslint/tsc) is itself a zero-baked-checks violation slated for
     eradication by this very cutover. So a **`typescript-toolchain` pack + its allowlist
     entries** is the concrete near-term goal of OQ-2's per-language toolchain-pack
     direction, not a someday example.
  3. *Division of TS support across bundles (recorded so it's not lost):* **BUNDLE-009**
     delivers the TS **TRACEABILITY** slice (substantiveness + contracts on
     ast-grep/grep) — feasible *without* this bundle because traceability rides
     **structural** engines, not the toolchain. **This bundle (BUNDLE-011)** owes the TS
     **TOOLCHAIN** slice (lint/build/test + the test runner). **Together** they unblock
     the runtime gating itself. BUNDLE-011 is the natural **NEXT-after-BUNDLE-009**.
  4. *A future language-agnostic COVERAGE bundle is sequenced near this one.* BUNDLE-009
     is **DELETING** the baked Go coverage analyzer (`pkg/gate/step_coverage.go`)
     **without replacing it** — coverage was descoped from BUNDLE-009 because it is
     **dynamic-toolchain** work, not structural. Coverage's language-agnostic
     re-implementation needs the **test runner**, so it naturally rides **with / after**
     this bundle's toolchain work (cf. OQ-6's shared test-runner / coverage re-routing).
     Recorded here so the dropped-coverage thread stays tracked.

**OQ-3 — The semgrep catch-all on non-Go files.** `routeFileDefaults` runs semgrep on
*every* non-`.go/.ts/.tsx` file today (:276-279). What pack replaces that catch-all —
and, concretely, does removing it **change the current gate result on this very
repo**? We need to know empirically whether any real finding on `main` today comes
from the catch-all semgrep pass before we can call its removal behavior-preserving.
Sub-questions: is the catch-all even producing findings (note the ISSUE-018 finding
that the in-process semgrep runs *config-less* and scans zero rules — is the "default
semgrep" already a no-op?), and if a project wants semgrep on arbitrary files
post-cutover, is that a declared pack rule rather than a baked default?

**OQ-4 — Ordering and division of labor vs SPEC-034, SPEC-035, ISSUE-018, BUNDLE-009.**
This bundle sits in a cluster of in-flight eradication work. The dependency/ordering
graph is genuinely open:
- Does this bundle **depend on SPEC-035 landing first** (the pack-declared engine
  bindings + trusted-tool allowlist are the substrate a toolchain pack needs to
  declare its `engine:` commands safely)? Strong candidate for a hard dependency.
- Does it **subsume SPEC-034**, or **build on it**? SPEC-034 is the Go-arm template
  and is already specced + planned (not landed). Option (a): let SPEC-034 land as-is
  (Go bridge + delete bespoke Go path), then this bundle generalizes the *remaining*
  `pkg/check` engine (the routing manifest, the enum, the standards reader, other
  languages, the catch-all). Option (b): this bundle absorbs SPEC-034's unlanded
  scope and does the whole cutover in one body of work. The duplicate-effort vs
  coordination tradeoff is real.
- How does it **coordinate with ISSUE-018** (which deletes the dead in-process
  semgrep + native-standards validator but leaves `pkg/check`'s Step 2 standing)?
  ISSUE-018 should land first as ground-clearing; does anything in it conflict with
  this cutover?
- Does it **subsume or merely coordinate with BUNDLE-009**? BUNDLE-009 eradicates the
  *traceability* analyzers (substantiveness/contracts); this bundle eradicates the
  *code-check* engine. They share the "stop baking rules into the binary" thesis and
  both ride the pack-engine substrate, but target different baked components. Likely
  "coordinate, don't subsume" — but the seam needs locking so neither absorbs the
  other's work.

**OQ-5 — The dead standards-manifest reader: fold into this cutover or separate
quick deletion?** `pkg/check/manifest.go` carries a whole compiled-standards-manifest
reader (`compiledManifestFile`/`isCompiled`/`deriveRules`/`hasSemgrepSignal`/
`routableExtensions`/`legacyRules`, :90-179) whose producer is already gone — it can
never be exercised (always falls to `defaultManifest()`). It lives *inside* the
engine file this cutover rewrites. Options: (a) delete it **as part of** this
cutover (it's in the blast radius anyway, and removing the routing manifest is part
of retiring the engine); (b) delete it **separately and first** as a trivial
dead-code pass (parallel to ISSUE-018's validator deletion) so this bundle starts
from a smaller surface. Note this is **distinct** from the native-standards
*validator* (`pkg/validate/standard.go`) ISSUE-018 already removes — this is the
dead *reader* on the check side.

**OQ-6 — Re-routing the gate steps that consume `CheckType` outputs.** The gate's
other steps assume `pkg/check`'s `CheckType` outputs. Notably, the shared test runner
feeds BOTH Step 2's test FAILs and the coverage step's per-package coverage (gate.go
comment at ~:340: "the whole-module `go test ./...` executes ONCE and feeds both
code_check and coverage_threshold"). If Step 2 stops being `pkg/check`, what runs
`go test` and produces coverage — does the coverage step now consume the
toolchain-pack's test-pass output, and does the shared-runner dedup survive the
cutover? Likewise any step keying off the `lint`/`build`/`test`/`semgrep` `CheckType`
identity (e.g. build-pass project-wide scope exemption, `checkViolationsToGate`'s
`cv.Pass` keying at gate.go ~:660) needs a declared-engine equivalent. This OQ
catalogs every consumer of `CheckType` and decides its post-cutover source; it is the
"don't drop a gate step on the floor during the cutover" question.

## Notes / Ideas

- **The TDD tests for the Go arm already exist on `main`** (`cutover_deletion_test.go`
  asserts `goBuiltinExecutors` / the `language == "go"` short-circuit are deleted;
  `bridge_test.go` asserts the three native passes resolve to `go-toolchain` engine
  bindings and run through one dispatch). These encode SPEC-034's intent and are a
  ready-made safety net + worked example for OQ-1's wholesale option. Confirm whether
  they pass against current `main` or are red-pending-impl before leaning on them.
- **The "no-pack → vacuous green" risk (OQ-2) is the philosophical crux.** The whole
  enforcement philosophy is "the enemy is silent/vacuous green." A cutover that makes
  "declared nothing" mean "checked nothing, exit 0" would *manufacture* the exact
  failure mode the project exists to prevent. Whatever the resolution, this case must
  end up LOUD. This is the single most important thing not to get wrong.
- **Possible already-dead semgrep default (OQ-3).** ISSUE-018 found the in-process
  `semgrepExecutor` shells out config-less (no `--config`) and therefore scans zero
  rules. If the default-routing semgrep pass is *also* config-less, the
  `routeFileDefaults` semgrep catch-all may already be a no-op on this repo — making
  its removal trivially behavior-preserving. Verify before treating it as a risk.
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
  onto the engine substrate, delete the bespoke Go path). The proven first
  step/template this bundle generalizes; note it is specced + planned but NOT yet
  landed on `main`.
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
- Code (verified 2026-06-21, `main`): cmd/backstop/gate.go:332 (`realCodeChecker`),
  :637/:649 (`check.RunWith` / `check.Run`), ~:340 (shared test runner feeding
  code_check + coverage), ~:660 (`checkViolationsToGate` keying on `cv.Pass`);
  pkg/check/manifest.go:14-23 (`CheckType` enum), :75-78 (`languageExtensions`),
  :90-179 (dead standards-manifest reader), :272-280 (`routeFileDefaults` + semgrep
  catch-all), :357-359 (`defaultManifest`); cmd/backstop/pack_gate.go
  (`dispatchPackEngines`).
- SPEC-034 (native-toolchain-engine-cutover, draft) — the Go-arm template (OQ-4).
- SPEC-035 (pack-declared-engines-trusted-allowlist, draft) — candidate prerequisite
  substrate (OQ-4).
- ISSUE-018 (remove vestigial baked-in code, open) — ground-clearing; deletes dead
  in-process semgrep + native-standards validator but leaves Step 2 standing (OQ-4).
- BUNDLE-009 (stack-aware-traceability, exploring) — sibling analyzer-eradication
  (OQ-4). **Cross-bundle (recorded 2026-06-22):** BUNDLE-009 delivers the TS
  *traceability* slice (structural engines, no toolchain dep) and is **deleting**
  `pkg/gate/step_coverage.go` without replacement; this bundle (BUNDLE-011) owes the TS
  *toolchain* slice (lint/build/test) + grows the trusted-tool allowlist (paired with
  the `typescript-toolchain` pack, alongside ISSUE-027) — together they unblock
  **backstop-runtime** gating itself, and the dropped language-agnostic **coverage**
  re-impl rides with/after this bundle's toolchain work. See the OQ-2 NOTE.
- ISSUE-027 — trusted-tool allowlist growth pairs with the per-language toolchain packs
  here (see OQ-2 NOTE); allowlisting a tool is inert until a pack declares an engine
  using it.

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
</content>
</invoke>
