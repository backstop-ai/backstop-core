---
title: Stack-Aware Traceability Steps — Coverage, Substantiveness, Contracts beyond Go
number: BUNDLE-009
schema_version: bundle/v2

bundle:
  name: stack-aware-traceability
  version: "0.2.2"
  created: "2026-06-13"
  updated: "2026-06-20"
  category: feature

status:
  maturity: exploring

problem:
  summary: >
    The gate's three traceability steps — coverage threshold (step 5), test
    substantiveness (step 4), and contract signature verification (step 6) —
    are hardwired to the Go toolchain. ISSUE-003 made the code-check passes
    (lint/build/test/semgrep) data-driven per stack but explicitly deferred the
    traceability steps as Go-only (its Ratified Design Constraint 6). On any
    non-Go project all three steps vacuously PASS: coverage finds no Go targets
    and returns "no Go package coverage targets in scope"; substantiveness
    parses with go/parser and no-ops on .ts; contract verification explicitly
    skips non-.go files (step_contract.go:44). That is silent non-enforcement
    in the traceability half of the kill chain — the exact failure mode the
    ISSUE-002/005/006/008 work eliminated in the code-check half.
  user_story: >
    As the maintainer dogfooding backstop against the runtime (a TypeScript +
    Bun project that is now itself a backstop project), I want the traceability
    steps to actually enforce coverage, test substantiveness, and contracts on
    TS/Bun — or to fail loud when they can't — so that `backstop gate` on the
    runtime repo means what it says instead of reporting a clean traceability
    half while enforcing nothing.
---

# Stack-Aware Traceability Steps — Coverage, Substantiveness, Contracts beyond Go

## Current Thinking

The runtime being a TS+Bun backstop project turns this from a hypothetical edge
into a concrete blocker, and gives the hardest design question a real target.
Three observations frame the exploration:

1. **The near-term win is fail-loud, not full portability.** Today these steps
   return green on a TS project. The smallest correct change — independent of
   whether substantiveness is even portable — is to make a traceability step
   that cannot enforce for the project's declared language emit an explicit
   unsupported/skipped signal (or config error) rather than a vacuous pass.
   That closes the dogfooding hole immediately and is consistent with every
   fail-loud fix shipped this session. Per OQ-5 (resolved) it stays in-scope as
   the bundle's first spec seed — NOT carved out as a separate prerequisite
   issue.

2. **Coverage is the most portable; substantiveness/contracts are the hard
   part.** Coverage is "run a command, parse a percentage, compare a threshold"
   — directly analogous to the ISSUE-003 toolchain registry, just with a
   coverage-format parser instead of a findings parser. Substantiveness
   (assertion/target-call AST analysis) and contract signatures (API surface
   extraction) are built on go/parser and have no free cross-language
   equivalent.

3. **Reuse the ISSUE-003 registry shape, don't invent a parallel one.**
   `enforcement.toolchain` already establishes per-stack command+format
   declaration. A coverage binding likely belongs there (a coverage command +
   a coverage-format parser), not in a new config surface.

4. **BUNDLE-010 shipped — the engine is no longer a future dependency, and the
   baked-in Go analyzers are now in-scope to eradicate (2026-06-20).** When this
   bundle was first written the pack engine wasn't ready, so the Go `go/parser`
   substantiveness/contract analyzers were treated as a preserved "native tier"
   and their migration was deferred ("the anomaly … migrate later"; SPEC-033
   REQ-003(d)). That hedge is now obsolete: BUNDLE-010 made the engine model
   first-class and wired **ast-grep end-to-end as a pack engine**, and SPEC-033
   locked the BUNDLE-010↔009 seam — the ast-grep engine, a reusable
   ast-grep→SARIF converter, and the engine-organized pack layout are all
   delivered and tested. So BUNDLE-009 now **aims to eradicate the baked-in Go
   `go/parser` traceability analyzers and replace them with packs, in-scope** —
   not defer them. The **end state** is **zero baked-in analyzers anywhere in the
   gate**: every traceability check is a pack, fully dogfooded. This makes
   BUNDLE-009 a **strangler repeat of SPEC-034**, run once more over a single
   subsystem (traceability). The three dimensions strangle differently and must
   NOT be flattened:
   - **Coverage is not a go/parser anomaly — nothing to eradicate.**
     `step_coverage.go` already runs `go test -coverprofile` and regex-parses the
     percentage; that is *already* the command + format-parser
     (`enforcement.toolchain` registry) shape. It just rides the registry
     extension (OQ-1, Seed 2). No analyzer to delete.
   - **Substantiveness — lower risk for the hollow-test half; the target-call half
     is unresolved.** The **Q1 hollow-test** check ("does this test assert
     anything?") maps cleanly to a finding: an ast-grep substantiveness pack (Go
     **included** — ast-grep speaks Go) emits findings → gate consumes; this is the
     valuable, cheap, portable signal the spike (below) already proved. **But
     `step_testverify.go` does a second thing the "clean finding" framing glossed:**
     the **noTarget** check ("does this test call the target package?") is a **JOIN**
     between the test file's AST and the **spec's declared target package**
     (`checkSubstantiveness` takes `TargetPkg` from the spec's
     `implementation.package` frontmatter, via `samePackage`/`callsTargetPackage`,
     step_testverify.go:286/321). A single-file ast-grep query **cannot see the
     spec's target package** — so a findings pack can express Q1 but not, by itself,
     this spec×test-file join. **The disposition of the target-call check is
     therefore UNRESOLVED, not a clean finding** (new OQ-6 below). Only after both
     halves are accounted for can the `step_testverify.go` go/ast+go/parser analyzer
     be deleted.
   - **Contracts — the meaty one; the "clean split" is shakier than first
     stated.** `step_contract.go` is NOT findings: it **extracts an API surface**
     and **compares** it against the spec's `## Contracts` declarations (and, per
     the just-shipped ISSUE-013, asserts symbol **absence**). The first-pass framing
     ("extraction → pack; comparison stays language-agnostic; absence stays")
     **understated two things** the code makes concrete:
     - **The comparison is not loosely "the right symbols" — it is string-equality
       on Go-SOURCE-rendered signatures.** `signaturesMatch` (step_contract.go:551)
       is whitespace-normalized **string equality**, and the strings come from
       `formatFuncSignature`/`underlyingTypeString` (:456/:388) — Go-specific
       rendering of receiver formatting, `[]byte`, `map[K]V`, `*T`, named-vs-unnamed
       results. So the extraction's equivalence target is **reproducing that exact
       rendering**, a much higher fidelity bar than "extract the symbols" — OR the
       comparison itself must move to a structural/normalized match (new OQ-8).
     - **Absence is not extraction.** ISSUE-013 absence ("symbol X is ABSENT from
       file F; loud config error if F is missing/non-Go") is checked by a gate-side
       `parser.ParseFile` + `probeSymbol` (step_contract.go:119/227) — i.e. the very
       Go analyzer slated for deletion. A findings pack emits what **exists**, not
       what is absent. So "delete the analyzer" and "absence stays a gate-side
       Go-aware probe" are in tension (new OQ-7).
     Eradication still splits along extraction-vs-comparison, but the comparison's
     fidelity bar and the absence probe are **open questions, not settled
     mechanics.** This dimension gets its own **strangler-equivalence pass** — prove
     the ast-grep extraction reproduces go/parser's results (including the exact
     signature rendering and ISSUE-013's absence cases) on real Go fixtures, THEN
     delete the baked-in analyzer. Exactly the SPEC-034 licensing pattern; highest
     risk, do it carefully.

5. **SPEC-035 is the substrate the eradication seeds build on (DEPENDENCY,
   2026-06-20).** BUNDLE-010 shipped the ast-grep *engine* (it produces the signals);
   what was still missing was how a pack *binds* that engine to a traceability check
   and feeds it the spec's declared symbols. **SPEC-035 (pack-declared engines +
   trusted-tool allowlist)**, being authored off BUNDLE-010, supplies exactly that:
   (a) **pack-declared engine bindings**, (b) a **`pattern-arg` input mode** so an
   engine can take an inline pattern *parameter* (a parameterized query, not a fixed
   static rule), and (c) a **gate-TYPE role on a binding** so a pack declares "this
   engine fills the contracts / substantiveness / coverage type." So
   BUNDLE-009 **DEPENDS ON SPEC-035**: its substantiveness / contract / coverage packs
   (Seeds 3, 4 — and the coverage registry indirectly) are authored on top of
   SPEC-035, alongside BUNDLE-010's ast-grep engine. The fail-loud + coverage seeds
   (1, 2) do not need it; the analyzer-eradication seeds (3, 4) do. A codebase
   **eradication audit (2026-06-20)** independently confirmed the exact targets and
   the keep-semantics / eradicate-extraction split for all three gate steps (see
   References) — this sharpens the OQ-6/7/8 framing without resolving them.

## Open Questions

- **OQ-1 — Coverage declaration. [RESOLVED 2026-06-13.]** A stack declares a
  coverage command + format + threshold through the same `enforcement.toolchain`
  surface. Sub-calls (delegated by the user; spec-level, reversible):
  - Coverage formats (lcov / istanbul-json / text) **extend the ISSUE-003
    named-format parser library** — reuse, don't fork into a coverage registry.
  - An undeclared traceability dimension with no built-in default for the stack
    is **warn-and-pass (exit 0)**, not a config error — declaring it is the opt-in
    to enforcement (see the FORK RESOLVED bullet below, 2026-06-20). This bullet
    originally read "config error (exit 2)"; that now holds only for *broken
    declared* dimensions, never for merely-undeclared ones.
  - **Fails loud AND useful** (user-confirmed; ISSUE-006 precedent; cross-cutting
    to all three steps). Every failure mode — undeclared+no-default, a coverage
    command that errors, an unparseable coverage format — must name the cause and
    the fix (which language, what's missing, declare-or-waive, expected-vs-got
    format), never a bare exit code.
  - **Loud ≠ blocking (refinement, 2026-06-14).** Loudness (visibility + specific
    guidance) is *always* mandatory; blocking is reserved for *defects and broken
    promises*, NOT for capability a project hasn't opted into yet. The enemy was
    always silence, not passing — a loud, specific, guided warn-that-passes is not
    vacuous green. Three classes: (1) **broken declaration** (declared command
    errors / unparseable output / unknown toolchain key) → loud AND **blocks**
    (exit 2; vacuous-green enemy). (2) **capability-absent** — the dimension needs
    an external pack/engine the project hasn't pulled in (the substantiveness pack
    on a non-Go stack) → loud with how-to-adopt guidance but **does NOT block**;
    blocking on absence makes adoption hostile (violates make-the-right-thing-easy).
    (3) **declared-intent-unmet** — project declared it wants the dimension but the
    capability is missing → broken promise, **blocks**. Explicit waive always
    available. Back-door-vacuous-green guard rails: class-2 advisory must be
    conspicuous and specific (dimension, stack, exact pack, command) — and since
    "unblocking" = exit 0, which is invisible in CI, the loudness must live in the
    **report surface, not the exit code**.
  - **FORK RESOLVED — declared == blocking, absent == warn, no separate flag
    (2026-06-20, user-settled).** Class-2 (capability-absent) stays warn
    *permanently* — it never auto-promotes to blocking and never gets a
    `required:`/`block:` knob. **Declaring a traceability dimension in the
    `enforcement.toolchain` surface (its command / format / threshold) IS the
    opt-in to enforcement.** Once declared, that dimension blocks on failure —
    below-threshold, a declared command that errors, or declared-but-the-capability-is-missing
    are all broken promises (exit 2). An *undeclared* / capability-absent
    dimension is a loud warn-with-specific-guidance, **exit 0, and passes
    forever** — it never blocks. The declaration itself is the binary signal;
    there is no third "I declared it but don't enforce it yet" state.
    - **Accepted trade (on the record):** there is **no "measure-but-don't-fail-CI"
      ramp-up mode**. To *see* a dimension you must declare it, and declaring it
      means it gates. This was chosen for simplicity — a binary humans can hold in
      their heads (declared→blocks, undeclared→warns) — over the flexibility of an
      extra opt-in-to-measure-only knob. Declare-when-ready is the ramp; there is no
      softer rung between off and enforcing.
    - **Supersedes the earlier OQ-1 default for the capability-absent case.** OQ-1's
      blanket "undeclared + no-default = config error (exit 2)" above now holds ONLY
      for *broken declared* things (a declared command that errors / unparseable
      declared format / unknown toolchain key → exit 2). An *undeclared* dimension
      with no built-in default is **warn-and-pass (exit 0)**, not exit 2 — you are
      not yet asking backstop to enforce it, so its silence is not a broken promise.
      Reconcile both readings to this rule: **broken DECLARED things block; UNdeclared
      things warn.**
  - **Defaults require ecosystem consensus.** Ship a built-in default only where
    a stack has a genuinely consensus tool (Go: go test / golangci-lint). For
    TypeScript, lint/build (eslint/tsc) are safe defaults but the test/coverage
    runner is contested (vitest / jest / bun / node --test) — backstop ships NO
    default there; projects declare it. The runtime (Bun) declares its own; Bun,
    a new entrant, is NOT treated as a TS default. Contested ecosystem bets stay
    out of the convention layer.
  - Synergy: declared-dimensions-block is humane only because `init` inference
    (see Notes) pre-populates the declaration — once a dimension is declared it
    gates strictly, so easy, accurate authoring keeps the strict gate from being a
    wall.

- **OQ-2 — Substantiveness across stacks. [RESOLVED 2026-06-13 for the LAYERED
  design; refined 2026-06-14; PARTIALLY RE-OPENED 2026-06-20.]** The original
  resolution rested on "native for Go (keep go/parser), query-pack for new stacks."
  **DECISION 2 (2026-06-20) deletes that native-for-Go tier** — the exact premise
  this resolution leaned on — so the resolution holds only as the *layered design
  shape*; the eradication re-opens how substantiveness actually lands for Go, via
  **OQ-6 below** (the spec×test-file target-package join a single-file pack query
  can't see). The Q1 hollow-test mapping survives intact; the noTarget half is the
  re-opened part. Three layers, not two. Headline: substantiveness is *cheap to
  port* because the valuable signal is shallow by nature.
  - **The valuable check is Q1 — hollow-test detection** ("does this test assert
    anything?"). It is a stack-specific assertion vocabulary + a tree walk, and
    it is low-false-positive when coarse: a test body with zero assertion-shaped
    calls is almost always genuinely hollow. This is the bulk of substantiveness
    value and it is cheap in every language.
  - **The expensive part is Q2 — target-exercise** ("does this test call the unit
    under test?"). Coarse Q2 false-positives on aliased imports / harness
    indirection; doing it *well* needs per-language import resolution.
    **Explicitly out of scope as acceptable coarseness** — the Go check today is
    already coarse here (syntactic package-qualifier match + same-package
    short-circuit, step_testverify.go:462) and nobody calls Go "unsupported."
    Ship the syntactic slice of Q2 that Go already does; do NOT chase deep
    resolution (diminishing returns, false-positive risk). Recorded as a design
    choice, not debt.
  - **Three delivery layers:** (1) native where already invested (Go on
    go/parser today — **but see the 2026-06-20 scope change: this native Go tier is
    now slated for eradication, not preservation; the substantiveness analyzer in
    `step_testverify.go` becomes a pack and is deleted, Seed 3**); (2) **query-pack on a
    shared tree-query engine (ast-grep) — the sweet spot**, cheaper than native
    and richer than declared; adding a stack = authoring a rule + assertion
    vocabulary in YAML, not building an analyzer; (3) declared-command fallback
    for anything the pack does not cover. Engine choice is a *distribution*
    decision (how to ship portably), not a *can-we* decision.
  - **Feasibility verified by spike (2026-06-14) — see the Spike section below.**
  - **Dependency:** the query-pack layer is gated on ast-grep wired as a pack
    engine (the next pack-engine generalization — broader substrate than this
    bundle, since it also unlocks layer-2 pack rules). BUNDLE-009 is a *consumer*
    of that engine, not its owner. The fail-loud seed (OQ-5) ships without it.

- **OQ-3 — Contract signatures across stacks. [RESOLVED 2026-06-13 for the LAYERED
  design; refined 2026-06-14; RE-OPENED 2026-06-20.]** Like OQ-2, this resolution
  assumed "native for Go (go/parser today)." **DECISION 2 deletes that native tier**,
  so the resolution holds only as the layered shape and the eradication re-opens two
  concrete mechanics via **OQ-7** (how ISSUE-013 absence is performed once the
  gate-side `parser.ParseFile` probe is gone) and **OQ-8** (whether the ast-grep
  extraction must reproduce `formatFuncSignature`'s exact Go-source rendering, or the
  comparison itself moves to a structural match). Same three-layer resolution as OQ-2
  and the same engine:
  contract signature verification is signature *extraction* (a
  metavariable-capturing tree query) compared against the declared contract —
  expressible on the same ast-grep pack engine, so it rides the same dependency.
  Native for Go (go/parser today), query-pack for new stacks, declared-command
  fallback otherwise. **Scope change (2026-06-20):** with BUNDLE-010 shipped, the
  Go-native tier here is also slated for eradication (Seed 4) — but contracts split:
  only the *extraction* moves to the ast-grep pack; the *comparison* logic
  (signature-match + the ISSUE-013 assert-**absence** check) is language-agnostic
  gate logic that **stays**, consuming pack-extracted symbols. Requires a
  strangler-equivalence pass before `step_contract.go`'s analyzer is deleted.

- **OQ-4 — Fail-loud semantics. [RESOLVED 2026-06-13 — folded into OQ-1;
  reconciled 2026-06-20 to the OQ-1 fork resolution.]**
  The end-state model (any stack's declared manifest commands just run)
  dissolves the "unsupported language" category. A traceability step is either
  backed by a declared command (it runs) or it isn't. **Reconciled with the OQ-1
  fork resolution (2026-06-20):** the two missing-declaration cases split by exit
  code — a **broken DECLARED** dimension (declared command errors / unparseable /
  unknown toolchain key) fails loud AND **blocks (exit 2)**; a merely **UNdeclared**
  / capability-absent dimension fails loud-on-the-report-surface but **WARNS and
  passes (exit 0)** — it is not a config error, because you have not yet asked
  backstop to enforce it. (This corrects the earlier OQ-4 phrasing that treated all
  "not declared" cases as the registry's exit-2 condition; that conflated the two
  and contradicted OQ-1.) The only surviving sub-question — is a traceability
  dimension required or waivable when undeclared — is a property of the declaration
  surface, i.e. part of OQ-1, and is now settled there: declaring it IS the opt-in
  to enforcement.

- **OQ-5 — Scope ordering. [RESOLVED 2026-06-13]** No carve-out. In the
  OQ-as-promotion-gate model every open question blocks this bundle equally, so
  fail-loud is not shipped as a separate prerequisite issue — it stays in-scope
  as the bundle's first spec seed. Accepted consequence: the runtime stays
  un-gateable on coverage/substantiveness/contracts until the bundle clears
  promotion.

  *(Note: the DECISION-2 scope expansion (2026-06-20) re-opens whether a smaller,
  independently-ready slice should still be carved for promotion — see the
  Promotion-readiness option in Notes. OQ-5's "no carve-out" was decided before the
  eradication seeds existed; it is not being overturned here, only flagged as worth
  the user's reconsideration.)*

  ---

  The three OQs below were **re-opened by DECISION 2 (2026-06-20)** — the scope
  expansion that deletes the native-for-Go analyzers. They are genuinely
  **UNRESOLVED**; options are listed without a lean. The user resolves them in a
  manual pass. They gate the analyzer-eradication seeds (3 and 4), not the
  fail-loud/coverage seeds (1 and 2).

- **OQ-6 — Substantiveness target-call (noTarget) disposition. [OPEN — 2026-06-20.]**
  `step_testverify.go`'s substantiveness check is two checks, not one. Q1
  (hollow-test) maps cleanly to a single-file ast-grep finding. But the **noTarget**
  check is a **JOIN** between the test file's AST and the **spec's declared target
  package** (`checkSubstantiveness` reads `TargetPkg` from the spec's
  `implementation.package` frontmatter, then `samePackage`/`callsTargetPackage`,
  step_testverify.go:286/301/321). A single-file ast-grep query **cannot see the
  spec's target package**, so a findings pack does not, by itself, express this. Where
  does the spec×test-file target-package join live after migration? Options to weigh
  (do not pick): (a) keep a thin gate-side join that post-processes pack output
  against the spec's declared target; (b) push it to the declared-command fallback
  layer; (c) drop it as accepted coarseness (Q2 is already explicitly coarse per
  OQ-2 — is dropping it entirely acceptable?); (d) some pack mechanism that takes the
  target package as a parameter. Until this is settled, `step_testverify.go`'s
  analyzer cannot be cleanly deleted.

- **OQ-7 — Contract absence probe after analyzer deletion. [OPEN — 2026-06-20.]**
  ISSUE-013 assert-**absence** ("symbol X is ABSENT from file F; loud config error if
  F is missing or non-Go") is **not extraction** — a findings pack emits what
  *exists*, not what is absent. Today the absence check IS a gate-side
  `parser.ParseFile` + `probeSymbol` (step_contract.go:119/227/248) — i.e. the very
  Go-aware analyzer DECISION 2 wants deleted. So "zero baked-in analyzers" and
  "absence is a gate-side Go-aware probe" are in **direct tension**. Forks to record
  (do not pick): (a) the pack itself emits an absence signal (can ast-grep / the
  SARIF contract express "this symbol does not appear"?); (b) a thin Go-aware absence
  probe legitimately **stays** gate-side, and "zero baked-in analyzers" is relaxed to
  "zero baked-in *extraction/analysis*, minus a narrow absence/file-shape probe"; (c)
  absence is derived by **interpreting an empty pack result-set** for the symbol
  (with the missing/non-Go-file loud-error semantics reconstructed gate-side from pack
  metadata). This OQ pokes at whether **"zero baked-in analyzers" is even literally
  achievable**, or only achievable as an aspiration with a documented exception.
  - *Substrate note (NOT a resolution, 2026-06-20):* SPEC-035's **`pattern-arg`**
    input mode is the likely **mechanism** option (a) and option (c) would build on —
    a parameterized pattern query (probe an allowlisted engine for *this specific
    declared symbol*) is the natural shape for an absence/probe check. This records
    the substrate the resolution will sit on; it does **not** pick a fork. OQ-7 stays
    **OPEN**.

- **OQ-8 — Contract signature-rendering equivalence target. [OPEN — 2026-06-20.]**
  `signaturesMatch` is whitespace-normalized **string equality** (step_contract.go:551)
  over Go-**source-rendered** signatures produced by `formatFuncSignature` /
  `underlyingTypeString` (:456/:388) — exact handling of receiver formatting,
  `[]byte`, `map[K]V`, `*T`, named-vs-unnamed results. So: must the ast-grep
  extraction **reproduce `formatFuncSignature`'s exact Go-source string rendering**
  (to feed the existing string-equality comparison unchanged), or does the
  **comparison itself move to a structural / normalized match** (compare captured
  AST/metavariable shapes rather than rendered strings)? The first keeps the
  comparison code but sets a very high extraction-fidelity bar (and is inherently
  Go-rendering-specific, awkward for the cross-stack goal); the second changes
  comparison code but lets extraction be structural. Unresolved — this is the crux of
  the Seed-4 strangler-equivalence pass.
  - *Substrate note (NOT a resolution, 2026-06-20):* whichever fork wins, the contract
    extraction is a **parameterized** query (probe for *this declared signature*),
    which is exactly SPEC-035's **`pattern-arg`** input mode on an allowlisted engine.
    That records the substrate the eventual resolution will build on; it does **not**
    decide between exact-rendering-reproduction and structural-comparison. OQ-8 stays
    **OPEN**.

## Spec Seeds

Four seeds, in suggested order. They do not overlap: the fail-loud surface (1) is
config/reporting behavior, coverage (2) rides the registry, and substantiveness
(3) and contracts (4) each own one baked-in analyzer's eradication. Seeds 3 and 4
each cover **all stacks including Go** — Go is not exempted; ast-grep speaks Go, so
the baked-in Go analyzer is replaced, not kept as a native tier. (These are
exploration seeds; formal REQ-NNN decomposition happens at promotion, not here.)

- **Seed 1 — Fail-loud on undeclared / capability-absent (no engine).** Ships on
  the existing binary. Per the OQ-1 fork resolution (2026-06-20): a *declared*
  traceability dimension that's broken (errors / unparseable / capability-missing)
  blocks (exit 2); an *undeclared* / capability-absent dimension emits a
  conspicuous, specific warn-with-how-to-adopt on the **report surface** and
  **passes (exit 0)**. Closes the runtime dogfooding hole today. No analyzer
  touched.
- **Seed 2 — Coverage via the `enforcement.toolchain` registry.** A stack declares
  a coverage command + named format parser + threshold through the same registry
  (OQ-1). Coverage formats extend the ISSUE-003 named-format parser library;
  declared → blocks below threshold, undeclared → warns. `step_coverage.go` already
  has this command+parser shape, so this is registry extension, **not** analyzer
  eradication.
- **Seed 3 — Substantiveness as an ast-grep findings pack (all stacks incl. Go);
  delete `step_testverify.go`'s analyzer.** The **Q1 hollow-test** half ("test
  asserts nothing" → finding) is the clean, spike-proven part: one ast-grep pack
  (per-language assertion vocabulary in YAML) emits findings → gate consumes via the
  BUNDLE-010 ast-grep→SARIF path, Go included. **But the analyzer also does the
  noTarget check — a spec×test-file target-package JOIN a single-file pack query
  cannot express (OQ-6 below).** That half's disposition (keep gate-side / declared
  fallback / drop as coarseness / push into the pack somehow) must be settled before
  `step_testverify.go`'s analyzer can be **DELETED**. Lower risk on Q1; OQ-6 gates
  the deletion.
- **Seed 4 — Contracts: ast-grep extraction pack + retained comparison (all stacks
  incl. Go); delete `step_contract.go`'s analyzer.** Splits along
  extraction-vs-comparison, but two parts are **open, not settled** (OQ-7/OQ-8
  below): (a) the **comparison** is whitespace-normalized **string-equality on
  Go-source-rendered signatures** (`signaturesMatch`/`formatFuncSignature`), so the
  extraction must either reproduce that **exact** rendering or the comparison moves
  to a structural match (**OQ-8**); (b) **ISSUE-013 absence** is a gate-side
  `parser.ParseFile` probe — not extraction, and a findings pack emits what
  *exists*, so how absence is performed after the analyzer is deleted is **OQ-7**.
  Requires a **strangler-equivalence pass**: prove the ast-grep extraction reproduces
  go/parser's results — including the exact signature rendering and ISSUE-013's
  absence cases — on real Go fixtures, THEN delete the baked-in analyzer (the
  SPEC-034 licensing pattern). Highest risk; sequence last; OQ-7/OQ-8 gate it.

End-state aspiration across seeds 3+4: **zero baked-in analyzers in the gate** —
traceability fully dogfooded as packs, BUNDLE-009 completing a strangler repeat of
SPEC-034 over the traceability subsystem. **Whether "zero" is literally achievable
is itself open** — OQ-7 pokes at whether a thin Go-aware absence probe legitimately
has to stay gate-side. Recorded as the aim, not a settled outcome.

## Notes / Ideas

- **Seed sequencing (2026-06-14, updated 2026-06-20 — now four seeds; see Spec
  Seeds above).** Seed 1 — fail-loud — ships on the existing binary with NO engine:
  Go stays native and works; non-Go declared stacks stop vacuously passing and get
  the explicit declare-or-warn signal. Closes the runtime dogfooding hole today.
  Seeds 2–4 — coverage via the registry, then the substantiveness and contract
  ast-grep packs — depend on the now-shipped BUNDLE-010 pack engine.
  **Update (2026-06-20):** the earlier framing that "Go-on-native vs Go-on-engine
  uniformity is a later migration, explicitly NOT this bundle" is **reversed**. With
  BUNDLE-010 delivered, eradicating the baked-in Go go/parser substantiveness and
  contract analyzers is **in-scope here** (Seeds 3 and 4 each delete one). The
  "don't churn working tested code" caution survives only as the **strangler-
  equivalence requirement** on the higher-risk contract seed — prove parity on Go
  fixtures before deleting — not as a reason to keep Go on a separate native tier.
- **Promotion-readiness OPTION (record only — user's call; NOT a decision, and NOT a
  promotion).** The four seeds no longer share a risk profile. **Seeds 1 (fail-loud)
  and 2 (coverage) are independently ready and untouched by the eradication** — they
  carry no open OQs (OQ-1/OQ-4/OQ-5 cover them and are resolved) and close the
  runtime dogfooding hole on the existing binary plus the registry. **Seeds 3 and 4
  carry the three newly re-opened OQs (OQ-6/OQ-7/OQ-8).** This surfaces an option the
  user may choose to take or reject: **promote a fail-loud + coverage slice now and
  keep the analyzer-eradication seeds in continued exploration** until OQ-6/7/8 are
  worked. This would mean revisiting OQ-5's "no carve-out" stance (decided before the
  eradication seeds existed). Recorded as an available path only — the bundle is NOT
  promoted here, and which seeds (if any) to split out is the user's decision.
- The `CoverageTarget.Stack` field and the deliberate "TestCommand parsed but
  not executed" design in step_coverage.go were left as the seam for exactly
  this work — the gate selects the target so a stack scheduler can plug in
  without spec-authored commands becoming execution plans.
- **`backstop init` toolchain inference (cross-cutting — candidate for its own
  capture, NOT traceability-specific).** init reads an existing project's
  evidence (package.json scripts, vitest/jest config, go.mod, .golangci.yml)
  and scaffolds an explicit `enforcement.toolchain` declaration the user
  reviews. Reconciles with ISSUE-003's "no detection magic" constraint because
  that forbade *runtime* silent detection; init-time inference writes an
  *explicit, reviewable* declaration that the gate then reads deterministically
  — detection assists authoring, never drives enforcement. Spans the whole
  toolchain surface (code-check + traceability) + the init command, so it
  belongs in its own artifact, cross-linked here. Makes the OQ-1 exit-2 default
  humane by pre-filling declarations.

## Spike — ast-grep feasibility (2026-06-14)

Goal: verify the query-pack layer (OQ-2/OQ-3) is real before it becomes
load-bearing — that hollow-test detection is expressible as a portable,
declarative rule on one engine, not a per-language analyzer baked into the Go
binary.

Setup: ast-grep 0.43.0. Fixtures in TS (vitest) and Python (pytest), each with
hollow tests (call the subject, assert nothing) and substantive tests
(expect / assert / pytest.raises). Rule shape: a test-declaration node that does
NOT `has` a descendant assertion node (ast-grep relational `not`/`has` with
`stopBy: end`).

Results:
- **Q1 is expressible per-language in YAML, no Go code.** The TS and Python rules
  each flagged ONLY the hollow tests; substantive tests passed clean — zero false
  positives on the substantive side.
- **The assertion vocabulary is the tunable knob**, mirroring Go's `hasAssertions`
  prefix list (`require/assert/check/verify/expect/must`,
  step_testverify.go:436). A test asserting only via a custom helper (no raw
  `expect`) false-positived under a narrow rule; widening the rule to match
  called-function names against an assertion-verb regex fixed it AND still flagged
  the genuine hollows. This is the empirical proof that "the additional percentage
  with not a ton of extra stuff" = vocabulary extension, per language, in data.
- **Gate contract works:** exit 1 + `file:line` + message = fail-loud and useful;
  drops into a gate step directly.

Conclusion: the differentiating half (portable substantiveness/contracts) is
rule-authoring on a shared engine — cheap per stack, tunable, no per-language
parser in the binary. The whole direction now hinges on the *engine*, not on the
analysis. Hence the engine dependency and the proposal to pull the ast-grep
pack-engine work ahead.

Reproducible rules (the widened TS form + Python):

```yaml
# hollow-test-ts (widened: expect OR any assertion-verb call)
id: hollow-test-ts
language: typescript
rule:
  all:
    - kind: call_expression
    - has: { field: function, regex: '^(test|it)$' }
    - not:
        any:
          - has: { pattern: expect($$$), stopBy: end }
          - has:
              kind: call_expression
              stopBy: end
              has: { field: function, regex: '(?i)(assert|expect|should|verify|check|must)' }
```

```yaml
# hollow-test-py
id: hollow-test-py
language: python
rule:
  all:
    - kind: function_definition
    - has: { field: name, regex: '^test' }
    - not:
        any:
          - has: { kind: assert_statement, stopBy: end }
          - has: { pattern: pytest.raises($$$), stopBy: end }
```

## References

- ISSUE-003 (data-driven toolchain registry) — Ratified Design Constraint 6 deferred these three steps; this bundle is that deferral made trackable.
- **SPEC-035 — pack-declared engines + trusted-tool allowlist (DEPENDENCY, added 2026-06-20).**
  Authored off BUNDLE-010 to provide the substrate the analyzer-eradication seeds (3
  and 4) build on. **BUNDLE-009 DEPENDS ON SPEC-035.** SPEC-035 delivers three pieces
  BUNDLE-009 needs: (a) **pack-declared engine bindings** — a pack declares which
  engine fills a check, rather than the engine being implicit-via-layer; (b) the
  **`pattern-arg` input mode** — an engine can take an inline pattern parameter, i.e.
  a *parameterized* query, which is exactly what the contract/absence checks need
  (probe for a specific declared symbol/signature rather than a fixed static rule);
  and (c) a **gate-TYPE role on a binding** — a pack declares "this engine fills the
  contracts / substantiveness / coverage *type*", so the gate routes its
  traceability dimensions to pack-supplied engines. So BUNDLE-009's
  substantiveness / contract / coverage packs are authored **on top of SPEC-035**,
  alongside BUNDLE-010's already-shipped ast-grep engine (the engine that produces
  the signals; SPEC-035 is how a pack *binds* it to a traceability type and feeds it
  a parameterized pattern). The fail-loud + coverage seeds (1, 2) do not need
  SPEC-035; the eradication seeds (3, 4) do.
- **Eradication audit confirmation (2026-06-20).** A codebase eradication audit
  verified BUNDLE-009's exact targets and the keep-semantics / eradicate-extraction
  split per gate step: **`step_testverify.go`** (substantiveness) — *keep* the "test
  must be substantive" invariant; *eradicate* the `go/ast` hollow-test detection +
  the target-package-join extraction. **`step_contract.go`** (contracts) — *keep* the
  declared-vs-actual comparison and the ISSUE-013 anti-vacuous-green **absence**
  policy; *eradicate* the `go/parser` symbol extraction + the `.go`-only probe gate.
  **`step_coverage.go`** (coverage) — *keep* the per-changed-package threshold
  semantics; *eradicate* `go test -coverprofile` as the *sole hardwired* coverage
  source + the dead `Stack` seam. This **sharpens the OQ-6/OQ-7/OQ-8 framing** (it
  confirms precisely what must be deleted vs preserved) but **does not resolve them**
  — they remain open for the user's manual pass.
- Pack engine dependency (added 2026-06-14; refined 2026-06-14; **DELIVERED 2026-06-20**): the query-pack layer for OQ-2/OQ-3 was gated on ast-grep being wired as a pack engine. **BUNDLE-010 shipped that** — the engine model is first-class, ast-grep is wired end-to-end as a pack engine, and SPEC-033 locked the BUNDLE-010↔009 seam (ast-grep engine + a reusable ast-grep→SARIF converter + engine-organized pack layout, all delivered and tested). So this is no longer a pending dependency; it is satisfied substrate BUNDLE-009 now builds on. **No baked-in shortcut (dogfood ruling):** backstop consumes its substantiveness/contract rules AS a pack, identically to how it already consumes go-standards-pack — enforcement logic does not live in the CLI binary, and there is no privileged "backstop ships built-in rules" tier. **Scope change (2026-06-20):** the existing Go go/parser analyzers (`step_testverify.go`, `step_contract.go`) were previously framed as an *anomaly* "slated to migrate onto the pack model later" — that deferral is now **reversed**. Because BUNDLE-010 delivered the engine + ast-grep, eradicating those baked-in analyzers is **in-scope in this bundle** (Seeds 3 and 4), not a later migration. End state: zero baked-in analyzers in the gate — a strangler repeat of SPEC-034 over the traceability subsystem.
- pkg/gate/step_coverage.go (Go-only target selection), step_testverify.go (go/parser substantiveness), step_contract.go:44 (non-.go skip).
- Driver: the runtime repo (TypeScript + Bun) as the first non-Go backstop project — dogfooding surfaces all three vacuous-pass holes at once.
- Sibling deferral: ISSUE-007 (build/test exclusions) — also stack-generic config, design compatibly.

## Version History

- **0.2.2 (2026-06-20)** — Dependency + audit recording pass; maturity **held at
  `exploring`**, **no OQs resolved or opened**. (a) **Recorded that BUNDLE-009 DEPENDS
  ON SPEC-035** (pack-declared engines + trusted-tool allowlist, authored off
  BUNDLE-010): SPEC-035 delivers pack-declared engine bindings, a **`pattern-arg`**
  input mode (parameterized pattern queries), and a **gate-TYPE role on a binding**, so
  the substantiveness / contract / coverage packs (Seeds 3, 4) are authored on top of
  SPEC-035 alongside BUNDLE-010's shipped ast-grep engine; fail-loud + coverage (Seeds
  1, 2) don't need it. Added as Current-Thinking observation 5 and a References entry.
  (b) **Recorded the eradication-audit confirmation** of the exact targets and the
  keep-semantics / eradicate-extraction split per gate step (`step_testverify.go`,
  `step_contract.go`, `step_coverage.go`) — sharpens the OQ-6/7/8 framing. (c) **Noted
  `pattern-arg` as the likely MECHANISM** the OQ-7 (absence probe) and OQ-8 (contract
  extraction) resolutions will build on — framed explicitly as *substrate the
  resolution sits on, not a resolution*; **OQ-6/OQ-7/OQ-8 remain OPEN** for the user's
  manual pass.
- **0.2.1 (2026-06-20)** — Corrective pass after an adversarial design review of the
  0.2.0 scope expansion; maturity **held at `exploring`** pending the user's manual OQ
  pass — nothing resolved here. (a) **Fixed a residual contradiction:** OQ-4 still
  treated an *undeclared* dimension as the registry's exit-2 condition, contradicting
  the OQ-1 fork resolution; reconciled to broken-DECLARED→exit-2 / UNdeclared→warn.
  (b) **Softened two over-claims to match the code:** substantiveness is a clean
  finding only for the Q1 hollow-test half — the noTarget check is a spec×test-file
  target-package JOIN a single-file pack query can't see; and the contract comparison
  is whitespace-normalized **string-equality on Go-source-rendered signatures**
  (`signaturesMatch`/`formatFuncSignature`), a higher extraction-fidelity bar than
  "the right symbols," while ISSUE-013 absence is a gate-side `parser.ParseFile`
  probe, not extraction. (c) **Re-opened three real OQs** around the Go-analyzer
  eradication: **OQ-6** (noTarget join disposition), **OQ-7** (how absence is
  performed once the gate-side probe is deleted — pokes at whether "zero baked-in
  analyzers" is literally achievable), **OQ-8** (must extraction reproduce the exact
  Go-source signature rendering, or does the comparison go structural). (d) **Marked
  OQ-2/OQ-3 as resolved-for-the-layered-design-but-re-opened-by-DECISION-2**, since
  that decision deletes the native-for-Go tier their resolutions rested on. (e)
  **Recorded a promotion-readiness OPTION** (not a decision): Seeds 1+2 are
  independently ready; Seeds 3+4 carry OQ-6/7/8 — the user may choose to promote a
  fail-loud+coverage slice and keep eradication in exploration.
- **0.2.0 (2026-06-20)** — Two user-driven decisions recorded; maturity stays
  `exploring`. (1) **OQ-1's OPEN FORK resolved:** declared == blocking, absent ==
  warn, no separate `required:`/`block:` flag. Declaring a traceability dimension
  in `enforcement.toolchain` is the opt-in to enforcement (declared → blocks on
  failure; undeclared / capability-absent → loud warn on the report surface, exit
  0, passes forever). Accepted trade: no measure-but-don't-fail-CI ramp-up mode —
  chosen for a binary humans can hold in their heads. Supersedes OQ-1's earlier
  blanket "undeclared + no-default = exit 2" for the capability-absent case (now:
  broken DECLARED things block, UNdeclared things warn). (2) **Scope expanded to
  eradicate the baked-in Go `go/parser` traceability analyzers** (un-deferred),
  given BUNDLE-010 delivered the pack engine + ast-grep and SPEC-033 locked the
  seam. Coverage rides the registry (no analyzer); substantiveness becomes an
  ast-grep findings pack and `step_testverify.go`'s analyzer is deleted; contracts
  split (extraction → ast-grep pack, comparison + ISSUE-013 assert-absence stays as
  gate logic) and `step_contract.go`'s analyzer is deleted after a strangler-
  equivalence pass. End state: zero baked-in analyzers — a strangler repeat of
  SPEC-034. Spec seeds updated to four. Also added the `number: BUNDLE-009`
  frontmatter field so the bundle is discoverable / ID-selectable (it predated that
  convention).
- **0.1.0 (2026-06-13)** — Initial bundle at `exploring`. Problem: the three
  traceability gate steps (coverage / substantiveness / contracts) are Go-hardwired
  and vacuously pass on non-Go projects. OQs 1–5 opened; OQ-1/2/3/5 resolved and
  OQ-4 folded into OQ-1 over 2026-06-13..14; ast-grep feasibility spike recorded.
