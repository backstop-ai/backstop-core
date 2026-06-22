---
title: Stack-Aware Traceability Steps — Substantiveness & Contracts beyond Go (coverage descoped)
number: BUNDLE-009
schema_version: bundle/v2

bundle:
  name: stack-aware-traceability
  version: "0.4.0"
  created: "2026-06-13"
  updated: "2026-06-22"
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
    steps to actually enforce test substantiveness and contracts on TS — or to
    fail loud when they can't — so that `backstop gate` on the runtime repo means
    what it says instead of reporting a clean traceability half while enforcing
    nothing. (Coverage's cross-stack enforcement is descoped to a future bundle
    per SD-4 — BUNDLE-009 deletes the non-working baked Go coverage check but does
    not re-implement coverage here, since coverage is test-runner-coupled,
    BUNDLE-011 territory; the substantiveness/contract checks ride the structural
    engines and need no toolchain.)
---

# Stack-Aware Traceability Steps — Substantiveness & Contracts beyond Go (coverage descoped)

> **Scope note (2026-06-22):** the original title read "…Coverage, Substantiveness,
> Contracts beyond Go." Per **Scope Decision #4** (see *Scope Decisions / Non-Goals*
> below), coverage's *cross-stack re-implementation* is **descoped from this bundle**;
> BUNDLE-009 only **deletes** the baked Go coverage analyzer (`step_coverage.go`) and
> defers a language-agnostic coverage check to a future dedicated bundle. The title is
> reconciled to reflect that the bundle delivers **substantiveness + contracts** (incl.
> a TypeScript proof pack), not coverage. The user may finalize a rename at promotion.

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
   - **Coverage is not a go/parser anomaly, but it IS a baked Go-toolchain path —
     DELETED here, re-implementation DEFERRED (SD-4, 2026-06-22; supersedes the
     original "rides the registry, Seed 2" framing below).** `step_coverage.go` runs
     `go test -coverprofile` and regex-parses the percentage — the command +
     format-parser shape. The 0.2.x plan had it ride a registry extension (the old
     Seed 2). **SD-4 changes this:** coverage is **dynamic-toolchain** (test-runner-
     coupled), architecturally unlike the **static-structural** substantiveness /
     contract checks, so BUNDLE-009 **DELETES `step_coverage.go`** (achieving the
     zero-baked-analyzer end-state across all three steps) and **DEFERS** coverage's
     language-agnostic re-implementation to a future dedicated bundle near BUNDLE-011.
     The interim gap is accepted (pre-launch; the existing check is a 0.0% false-RED
     lie, not a working capability). **Seed 2 is DROPPED.**
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
  can't see) — **now RESOLVED 2026-06-22**: noTarget = pack extraction + thin gate
  set-join. The Q1 hollow-test mapping survives intact; the noTarget half was the
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
  so the resolution holds only as the layered shape and the eradication re-opened two
  concrete mechanics via **OQ-7** (how ISSUE-013 absence is performed once the
  gate-side `parser.ParseFile` probe is gone) and **OQ-8** (whether the ast-grep
  extraction must reproduce `formatFuncSignature`'s exact Go-source rendering, or the
  comparison itself moves to a structural match) — **both now RESOLVED 2026-06-22**:
  absence = pack-declared grep forbidden-pattern probe; signatures = pack-compiled
  ast-grep pattern query (comparison goes structural). Same three-layer resolution as OQ-2
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

  The three OQs below were **re-opened by DECISION 2 (2026-06-20)** and **RESOLVED by
  the user 2026-06-22**. They gate the analyzer-eradication seeds (3 and 4), not the
  fail-loud/coverage seeds (1 and 2).

  **Cross-cutting theme (unifies all three resolutions, 2026-06-22).** Every
  resolution follows **one architecture**: the **PACK** does the language-specific work
  (extraction / grep-probe / signature→pattern-compile, all emitting **SARIF**); the
  **GATE** does language-agnostic semantics (set-join / polarity-inversion /
  match-verdict + a shared **file-scanned guard**); and the backstop **BINARY knows
  zero language/tool specifics.** All three ride **SPEC-035's `pattern-arg` +
  allowlisted-engine substrate** (now shipped). The **engine-fit split is consistent:**
  **GREP for absence** (text-presence), **AST-GREP for structure** (signatures,
  hollow-test extraction). Implementing OQ-7 + OQ-8 is what finally turns backstop's
  **own** gate green on its `contract_signature` step.

- **OQ-6 — Substantiveness target-call (noTarget) disposition. [RESOLVED 2026-06-22 —
  option (a), thin gate-side join.]** The `noTarget` substantiveness check ("does this
  test exercise the unit under test?") **splits across the pack/gate boundary.** The
  **PACK** does the language-specific **EXTRACTION** — it emits "which packages/symbols
  this test references" as a normal *positive* ast-grep query (no spec awareness
  required). The **GATE** does a trivial, language-agnostic **SET-MEMBERSHIP test**: is
  the spec's declared target package (`implementation.package` frontmatter) among the
  referenced symbols the pack emitted? The **noTarget SEMANTICS live in the gate as a
  set test** (gate logic consuming pack data, **not** a baked analyzer); only the
  **EXTRACTION** becomes a pack.
  - **Rationale.** Preserves the coarse Go-parity signal (dropping it entirely —
    rejected option c — would be a capability regression; OQ-2 said Q2 is "acceptable
    *coarse*," not "acceptable *absent*"). Keeps the eradication honest: a gate-side
    set-test is **not** an analyzer. Generalizes across stacks: any stack's pack emits
    "referenced symbols," and the gate joins them the same way.
  - **Rejected:** (b) declared-command-per-stack — heavyweight for one coarse boolean;
    (c) drop entirely — a regression; (d) parameterized per-(test,target) query —
    over-engineers a deliberately-coarse, low-value check.
  - **Firms Spec Seed 3 (substantiveness):** `step_testverify.go`'s analyzer is deleted;
    **Q1 hollow-test → pack ast-grep finding**; **Q2 noTarget → pack extraction + thin
    gate set-join.**

- **OQ-7 — Contract absence probe after analyzer deletion. [RESOLVED 2026-06-22 —
  pack-declared forbidden-pattern GREP probe.]** Absence ("symbol X must be **ABSENT**")
  is a **PACK-DECLARED, ALLOWLISTED grep/ripgrep FORBIDDEN-PATTERN probe**,
  parameterized via SPEC-035's **`pattern-arg`**, with **SCOPE taken from the declared
  contract** — a file OR a path. (Scope-as-parameter preserves ISSUE-013's
  file-scoping AND enables whole-tree absence, **as a parameter, not a fork**.) The
  engine emits SARIF natively: a **match = symbol PRESENT**, an **empty result = symbol
  ABSENT**. The **GATE inverts polarity** (a present-match *is* the violation: "X must
  be absent, found at L") **plus a thin gate-side "was-the-file-actually-scanned?"
  guard** so empty-because-absent is never confused with empty-because-not-scanned —
  this preserves ISSUE-013's loud error when the file is missing/non-Go.
  - **Engine-fit (key decision).** Absence uses **GREP** (text-presence — "does this
    token appear anywhere"), **NOT ast-grep.** ast-grep's structural granularity is the
    *wrong* tool for absence: it needs a per-language grammar and misses lingering
    references in comments / strings / non-parsed files. Grep is coarse,
    language-agnostic, and its conservative failure direction (loudly flags *any*
    textual appearance) is exactly what absence wants.
  - **Explicitly NOT a grep baked into the gate BINARY** (rejected as baked
    tool-knowledge). It is a grep **ENGINE DECLARED BY A PACK** and allowlisted, like
    any engine.
  - **Accepted trade (on the record).** Grep is coarser — it can match a token in
    comments / strings / substrings, so a *lingering name* can produce a false-FAIL.
    Mitigated by word-boundary / anchored patterns (`func Foo\b`) and waivability; the
    conservative direction is acceptable.
  - **Authoring insight (record it).** Codebase-wide grep is the **SPEC-AUTHORING**
    tool — sweep the whole tree to discover what exists and where, which informs *which*
    absence contracts to write. File-scoped grep is the **GATE-ENFORCEMENT** tool. Same
    primitive, two phases.

- **OQ-8 — Contract signature-rendering equivalence. [RESOLVED 2026-06-22 —
  contract-signature-as-ast-grep-pattern; the PACK compiles the signature.]** Contract
  signature verification is a **REQUIRED-pattern ast-grep query** — the OQ-7 shape
  *inverted*: a **match = contract SATISFIED**, **no-match = VIOLATION**. This
  **dissolves the brittle string round-trip**: **no more** `formatFuncSignature` /
  `underlyingTypeString` Go-source-string rendering fed into `signaturesMatch`
  whitespace-normalized string-equality (which is *literally why* the gate's own
  `contract_signature` step is red today — the dual-substrate proof). Instead: the
  **contract stores a HUMAN-READABLE signature** (e.g. `func RouteFile(path string)
  []CheckType`), and the **LANGUAGE PACK provides a contract→ast-grep-pattern
  COMPILER** that turns the declared signature into its language's ast-grep pattern at
  gate time. This is architecturally analogous to the pack's **existing SARIF convert
  scripts** — both are language-specific transforms that live in the PACK, not the
  binary.
  - **Backstop NEVER compiles, renders, or understands a signature.** Doing so in the
    binary would be a **P0 zero-baked-language VIOLATION** (the user explicitly flagged
    and rejected this). Backstop passes the declared signature to the pack-declared
    engine, which compiles + queries + emits SARIF; the gate consumes **match/no-match**
    plus **the same file-scanned guard as OQ-7.**
  - **Engine-fit.** Signatures use **AST-GREP** (structural — "a function with these
    param types and this return, regardless of param names / whitespace" — grep can't
    reliably match that), completing the split: **absence = grep (text);
    structure / signature / hollow-test extraction = ast-grep (AST).**
  - **Resolves the recorded Fork 2** ("comparison moves to structural") — realized as
    the **engine's pattern-matching** with the per-language transform **in the PACK.**
    Fidelity doesn't vanish; it *moves* — from "exact Go-source string rendering" to
    "ast-grep pattern precision" (the `[]byte`-vs-`[]uint8`, named-vs-unnamed-results
    edge cases), handled **per-language in the PACK.**
  - **Firms Spec Seed 4 (contracts):** `step_contract.go`'s `go/parser` extraction +
    `formatFuncSignature` rendering + string-equality are **deleted**; **signature
    presence → pack-compiled ast-grep pattern query**; **absence → pack grep probe
    (OQ-7)**; the **language-agnostic comparison polarity + file-scanned guard stay
    gate-side.**

## Scope Decisions / Non-Goals (user-driven, 2026-06-22)

These four decisions were resolved by the user in a working session (2026-06-22).
They are **SCOPE decisions** — they tighten what BUNDLE-009 delivers vs defers — and
are **distinct from the OQ resolutions** above (those settled *mechanics*; these
settle the *boundary*). They do **not** re-open OQ-1..8. Maturity stays `exploring`;
the user drives promotion separately. Each is recorded as an explicit IN-SCOPE
requirement or OUT-OF-SCOPE non-goal and is reflected in the firmed Spec Seeds below.

### IN SCOPE

- **SD-1 — Grep/ripgrep engine (IN SCOPE; explicit requirement).** OQ-7's absence
  probe needs a **grep/ripgrep engine that does not exist yet.** Verified against the
  code: the engine `DefaultRegistry` (`pkg/pack/engine/binding.go`) holds
  semgrep / ast-grep / sandbox / config-file / golangci / go-build / go-test — **no
  grep**; the trusted-tool allowlist (`pkg/pack/engine/allowlist.go:22-23`) holds
  **only** `semgrep` + `ast-grep`. BUNDLE-009 stands the grep engine up, split per the
  **SPEC-035 architecture** into two distinct moves:
  - **(a) The grep ENGINE BINDING is PACK-DECLARED, not baked.** The traceability pack
    declares grep in its `engines:` block (input mode `pattern-arg` per SPEC-035; a
    grep-output→SARIF convert script, analogous to ast-grep's `to-sarif.sh`). So there
    is **NO baked `DefaultRegistry` entry** for grep and therefore **NO ISSUE-027-style
    eradication debt** created. Backstop learns no grep specifics; it runs the
    pack-declared command and consumes SARIF.
  - **(b) The `grep`/`rg` TOOL is added to the backstop-owned trusted-tool allowlist.**
    This is the small, expected "new language = a new pack **+ a few trivial allowlist
    entries**" move — the allowlist is the one backstop-owned surface that must name a
    tool before a pack may invoke it.
  - **Why IN SCOPE (vs ast-grep, which was a broad substrate prerequisite):** grep is
    **tiny**, and **BUNDLE-009 is its only consumer** here, so standing it up makes the
    absence Seed (Seed 4) self-contained rather than forcing a separate substrate
    bundle. Feeds the **absence half of Seed 4**.

- **SD-3 — A TypeScript traceability proof pack (IN SCOPE; the "beyond Go" proof).**
  A pack is **stack-locked**: verified — `Manifest.Language` is a single pack-level
  field (`pkg/pack/manifest.go:18`) and rules/engines inherit it; there is no
  multi-language pack. So **"beyond Go" literally means authoring a second,
  stack-locked pack.** To avoid the **theoretical-claim trap** (the title says
  "…beyond Go"; shipping only a Go pack while claiming stack-awareness would be
  vacuous), BUNDLE-009 authors **ONE non-Go proof pack: a TypeScript traceability
  pack** covering:
  - **SUBSTANTIVENESS** — hollow-test detection on `.test.ts` via **ast-grep** (the
    spike already proved the TS hollow-test rule, see Spike section).
  - **CONTRACTS** — signature **presence** via **ast-grep**, signature/symbol
    **absence** via **grep** (the SD-1 engine).
  - **Feasible NOW, independent of BUNDLE-011:** traceability rides the **STRUCTURAL
    engines** (ast-grep / grep — both speak TS as text/AST), **NOT** the TS language
    toolchain (eslint / tsc). No TS toolchain is needed to author this pack.
  - **STRATEGIC MOTIVATION (on the record):** the runtime (**backstop-runtime**, a
    TypeScript project) is **currently BLOCKED by the half-baked pack system — it can't
    gate itself with packs.** The TS proof pack is therefore a **live priority**, not a
    mere demonstration; it is the first slice that lets the runtime begin gating itself.
  - **CONSCIOUS scope GROWTH, eyes-open:** authoring a real second pack is **meaningful
    work** (not free), chosen deliberately for the runtime priority — recorded as
    growth, not scope-creep.
  - **CAVEAT (record it):** **TS COVERAGE is NOT delivered here.** Coverage needs the TS
    **test runner** (= BUNDLE-011's toolchain) and is descoped anyway per **SD-4**. The
    TS proof pack delivers substantiveness + contracts only.

### OUT OF SCOPE (Non-Goals)

- **SD-2 — `init` toolchain inference (OUT OF SCOPE; Non-Goal).** OQ-1's
  "declared == blocks strictly" gate is humane **because** `init` inference
  pre-populates the declarations a user would otherwise hand-write — but **building
  that inference is ONBOARDING's job** (SPEC-026 / BUNDLE-003 territory: repo
  detection, command inference, consent-before-write). **Record the DIRECTIONALITY
  explicitly:** **onboarding is a DOWNSTREAM consumer that DEPENDS ON this
  traceability architecture being in place — NOT the reverse.** init-inference cannot
  sensibly pre-fill declarations until the **declaration surface + pack/gate split**
  this bundle establishes exist. Therefore **BUNDLE-009 ships FIRST** and **assumes
  declarations already exist** (hand-authored **OR** init-authored); it does **not
  care how the declaration got there.**
  - **Cross-bundle sequencing note (record):** the strict declared-gate is
    **hand-authored-only** until onboarding's inference lands downstream. That is
    **acceptable** — early adopters hand-write declarations.

- **SD-4 — Coverage's cross-stack re-implementation (OUT OF SCOPE; DEFERRED to a
  future bundle).** Coverage is **architecturally a DIFFERENT beast** from
  substantiveness / contracts:
  - Substantiveness + contracts are **STATIC-STRUCTURAL** — ast-grep / grep over
    source; no toolchain needed (which is why SD-3's TS pack is feasible now).
  - Coverage is **DYNAMIC-TOOLCHAIN** — run the test suite with instrumentation, parse
    runtime lcov / istanbul / go-cover output — **coupled to the test runner**
    (BUNDLE-011's toolchain), **not** the structural engines.
  - **DECISION:** BUNDLE-009 **DELETES all THREE baked Go analyzers** —
    `step_testverify.go` (substantiveness), `step_contract.go` (contracts), **AND**
    `step_coverage.go` (coverage) — reaching the bundle's **"zero baked-in traceability
    analyzers"** end-state. Substantiveness + contracts are **RE-IMPLEMENTED as packs
    IN this bundle** (Seeds 3 + 4). Coverage's baked Go parsing
    (`go test -coverprofile` + the percentage regex) is **DELETED with NO replacement
    here**; its **language-agnostic re-implementation is DEFERRED to a future dedicated
    "language-agnostic coverage" bundle** (naturally sequenced near BUNDLE-011, since
    coverage needs the test runner).
  - **The interim coverage GAP is ACCEPTABLE and explicitly accepted:** backstop is
    **PRE-LAUNCH with NO users / no remote**, and the existing coverage check is
    **ALREADY brittle / non-functional** (it emits a **0.0% false-RED** on non-Go and
    on packages it can't resolve) — so this removes a **non-working check (a lie)**, not
    a working capability.
  - **DROP Spec Seed 2 (coverage)** from this bundle (see Spec Seeds below).

- **SD-3-OUT — Traceability packs for every OTHER language (OUT OF SCOPE; Non-Goal).**
  Beyond the **one TypeScript** proof pack (SD-3), traceability packs for **Python /
  Rust / any other language** are **separate consumer efforts**, not BUNDLE-009 work.
  The architecture supports them (any stack's pack rides the same ast-grep / grep
  engines + gate-side semantics), but authoring them is out of this bundle's scope.

### Cross-bundle sequencing (record)

- **BUNDLE-011 is the natural NEXT-after-009.** Together they unblock the runtime:
  **BUNDLE-009 delivers the TS TRACEABILITY slice** (substantiveness + contracts via
  structural engines); **BUNDLE-011 owes the TS TOOLCHAIN slice** (lint / build / test,
  incl. the **test runner** that coverage's re-implementation will need).
- **The future "language-agnostic coverage" bundle is sequenced near BUNDLE-011**,
  since coverage is dynamic-toolchain and depends on the test runner BUNDLE-011 lands.

## Spec Seeds

Three seeds (Seed 2 — coverage — was **DROPPED** per SD-4), in suggested order. They
do not overlap: the fail-loud surface (1) is config/reporting behavior, and
substantiveness (3) and contracts (4) each own one baked-in analyzer's eradication
**plus the TypeScript proof pack** (SD-3). Seeds 3 and 4 each cover **all stacks
including Go AND TypeScript** — Go is not exempted (ast-grep speaks Go, so the baked-in
Go analyzer is replaced, not kept as a native tier), and TS is the mandated non-Go
proof. (These are exploration seeds; formal REQ-NNN decomposition happens at
promotion, not here.)

- **Seed 1 — Fail-loud on undeclared / capability-absent (no engine).** Ships on
  the existing binary. Per the OQ-1 fork resolution (2026-06-20): a *declared*
  traceability dimension that's broken (errors / unparseable / capability-missing)
  blocks (exit 2); an *undeclared* / capability-absent dimension emits a
  conspicuous, specific warn-with-how-to-adopt on the **report surface** and
  **passes (exit 0)**. Closes the runtime dogfooding hole today. No analyzer
  touched.
- **~~Seed 2 — Coverage via the `enforcement.toolchain` registry.~~ DROPPED
  (SD-4, 2026-06-22).** Coverage's cross-stack re-implementation is **descoped from
  this bundle** — it is dynamic-toolchain (test-runner-coupled), not static-structural,
  and is **DEFERRED to a future dedicated "language-agnostic coverage" bundle**
  sequenced near BUNDLE-011. What **remains in BUNDLE-009** for coverage is **only the
  DELETION** of the baked Go analyzer `step_coverage.go` (the `go test -coverprofile` +
  percentage-regex path and the dead `Stack` seam), with **NO replacement here**.
  Removing a non-working check (the 0.0% false-RED) is acceptable pre-launch. **No
  coverage *delivery* seed in this bundle.** The deletion is folded into the
  "zero baked-in analyzers" end-state below.
- **Seed 3 — Substantiveness as an ast-grep findings pack (all stacks incl. Go AND a
  TypeScript proof pack); delete `step_testverify.go`'s analyzer.** Both halves are
  firmed (**OQ-6 resolved**): **Q1 hollow-test** ("test asserts nothing" → finding) is
  one ast-grep pack (per-language assertion vocabulary in YAML) emitting findings →
  gate consumes via the BUNDLE-010 ast-grep→SARIF path, Go included. **Q2 noTarget** is
  a **pack EXTRACTION** ("which packages/symbols this test references" — a positive
  ast-grep query) **+ a thin language-agnostic GATE SET-JOIN** against the spec's
  declared `implementation.package`. The set-test lives in the gate (consuming pack
  data, not a baked analyzer), so `step_testverify.go`'s analyzer is cleanly
  **DELETED**. **Now ALSO mandates the TS proof pack (SD-3):** this seed delivers the
  **Go migration** (analyzer → pack) **AND** the **TypeScript hollow-test rule on
  `.test.ts`** (via ast-grep — feasible now, the spike proved it, no TS toolchain
  needed). Lower risk.
- **Seed 4 — Contracts: pack-compiled ast-grep + pack-declared grep probe (all stacks
  incl. Go AND a TypeScript proof pack); delete `step_contract.go`'s analyzer; **stand
  up the grep engine (SD-1).** Both open parts are firmed (**OQ-7/OQ-8 resolved**):
  **signature presence** → a **pack-compiled ast-grep pattern query** (the contract
  stores a human-readable signature; the language pack compiles it to its-language
  ast-grep pattern at gate time — analogous to the pack's existing SARIF convert
  scripts), a **match = SATISFIED**; this **dissolves** the
  `signaturesMatch`/`formatFuncSignature` string round-trip outright. **ISSUE-013
  absence** → a **pack-declared, allowlisted grep forbidden-pattern probe**
  (scope-as-parameter: file OR path), a **match = violation**. **The grep engine
  (SD-1) is stood up here**, as the explicit requirement feeding this **absence half**:
  (a) the traceability pack **declares grep in its `engines:` block** (`pattern-arg`
  input mode + grep-output→SARIF convert) — **no baked `DefaultRegistry` entry, no
  ISSUE-027 eradication debt**; (b) the **`grep`/`rg` tool is added to the
  backstop-owned trusted-tool allowlist** (`pkg/pack/engine/allowlist.go`). The
  **language-agnostic comparison polarity + a shared file-scanned guard stay
  gate-side.** `step_contract.go`'s `go/parser` extraction + `formatFuncSignature`
  rendering + string-equality are **DELETED**. **Now ALSO mandates the TS proof pack
  (SD-3):** the **Go migration** **AND** the **TypeScript contracts pack** (signature
  presence via ast-grep, absence via the new grep engine on `.ts`). A
  **strangler-equivalence pass** still guards the Go cutover (prove the pack-compiled
  patterns + grep probes reproduce go/parser's verdicts — including ISSUE-013's absence
  cases — on real Go fixtures before deleting, the SPEC-034 licensing pattern). Highest
  risk; sequence last.

End-state across seeds 3+4 (plus the SD-4 coverage-analyzer deletion): **zero baked-in
analyzers in the gate** — all THREE baked Go analyzers (`step_testverify.go`,
`step_contract.go`, `step_coverage.go`) deleted; substantiveness + contracts
re-implemented as packs (Go **and** TypeScript); coverage's re-implementation deferred
to a future bundle. Traceability fully dogfooded as packs, BUNDLE-009 completing a
strangler repeat of SPEC-034 over the traceability subsystem. With OQ-7's absence
resolved as a **pack-declared grep probe** (not a retained gate-side Go-aware probe),
the only language-agnostic gate logic that stays is the **file-scanned guard +
polarity/verdict** — so "zero baked-in analyzers" is a **settled outcome**.

## Notes / Ideas

- **Seed sequencing (2026-06-14; updated 2026-06-20; updated 2026-06-22 — now THREE
  seeds, Seed 2/coverage DROPPED per SD-4; see Spec Seeds above).** Seed 1 —
  fail-loud — ships on the existing binary with NO engine: Go stays native and works;
  non-Go declared stacks stop vacuously passing and get the explicit declare-or-warn
  signal. Closes the runtime dogfooding hole today. Seeds 3–4 — the substantiveness and
  contract ast-grep packs (each now also delivering the **TS proof pack**, SD-3, and
  Seed 4 standing up the **grep engine**, SD-1) — depend on the now-shipped BUNDLE-010
  pack engine + SPEC-035. The former coverage seed is gone: coverage's baked Go analyzer
  is **deleted** here (SD-4) but its cross-stack re-implementation is **deferred** to a
  future bundle near BUNDLE-011.
  **Update (2026-06-20):** the earlier framing that "Go-on-native vs Go-on-engine
  uniformity is a later migration, explicitly NOT this bundle" is **reversed**. With
  BUNDLE-010 delivered, eradicating the baked-in Go go/parser substantiveness and
  contract analyzers is **in-scope here** (Seeds 3 and 4 each delete one). The
  "don't churn working tested code" caution survives only as the **strangler-
  equivalence requirement** on the higher-risk contract seed — prove parity on Go
  fixtures before deleting — not as a reason to keep Go on a separate native tier.
- **Promotion-readiness OPTION (record only — user's call; NOT a decision, and NOT a
  promotion; updated 2026-06-22 for the dropped coverage seed + resolved OQs).** With
  Seed 2 (coverage) **dropped** (SD-4) and OQ-6/7/8 now **resolved** (0.3.0), the
  earlier "Seeds 1+2 are the independently-ready slice while 3+4 carry open OQs" framing
  is **moot for OQ-blockedness** — all eradication-seed OQs are resolved. **Seed 1
  (fail-loud)** remains the smallest independently-ready slice (no engine, closes the
  runtime hole on the existing binary). **Seeds 3 and 4** are now firmed but carry the
  most work (analyzer eradication + the TS proof pack + the grep engine). The option the
  user may still take or reject: **promote a fail-loud slice now and keep the
  pack/eradication seeds in continued exploration** — but this is now a *sequencing*
  choice, not an *OQ-blockedness* one. Recorded as an available path only — the bundle
  is NOT promoted here, and which seeds (if any) to split out is the user's decision.
- The `CoverageTarget.Stack` field and the deliberate "TestCommand parsed but
  not executed" design in step_coverage.go were left as the seam for exactly
  this work — the gate selects the target so a stack scheduler can plug in
  without spec-authored commands becoming execution plans.
- **`backstop init` toolchain inference (OUT OF SCOPE — SD-2; onboarding's job).**
  init reads an existing project's evidence (package.json scripts, vitest/jest config,
  go.mod, .golangci.yml) and scaffolds an explicit `enforcement.toolchain` declaration
  the user reviews. Reconciles with ISSUE-003's "no detection magic" constraint because
  that forbade *runtime* silent detection; init-time inference writes an *explicit,
  reviewable* declaration that the gate then reads deterministically — detection assists
  authoring, never drives enforcement. **Per SD-2 (2026-06-22) this is a Non-Goal of
  BUNDLE-009: it is ONBOARDING's job (SPEC-026 / BUNDLE-003).** Directionality:
  **onboarding DEPENDS ON this traceability architecture (declaration surface +
  pack/gate split) being in place — not the reverse** — so BUNDLE-009 ships FIRST and
  assumes declarations already exist (hand- or init-authored). The strict declared-gate
  is hand-authored-only until onboarding's inference lands downstream; acceptable for
  early adopters. Cross-linked here; belongs in its own artifact.

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
  **`step_coverage.go`** (coverage) — the 2026-06-20 audit framed this as *keep* the
  per-changed-package threshold semantics while *eradicating* `go test -coverprofile` as
  the sole hardwired source. **SUPERSEDED by SD-4 (2026-06-22):** BUNDLE-009 **DELETES
  `step_coverage.go` outright (the `go test -coverprofile` path, the percentage regex,
  AND the dead `Stack` seam) with NO in-bundle replacement** — coverage's per-package
  threshold semantics are **carried forward to a future "language-agnostic coverage"
  bundle** (near BUNDLE-011), not preserved gate-side here, because coverage is
  dynamic-toolchain (test-runner-coupled), unlike the static-structural substantiveness
  / contract checks. The audit **sharpened the OQ-6/OQ-7/OQ-8 framing** (substantiveness
  + contracts) but **does not** govern coverage's disposition — SD-4 does.
- Pack engine dependency (added 2026-06-14; refined 2026-06-14; **DELIVERED 2026-06-20**): the query-pack layer for OQ-2/OQ-3 was gated on ast-grep being wired as a pack engine. **BUNDLE-010 shipped that** — the engine model is first-class, ast-grep is wired end-to-end as a pack engine, and SPEC-033 locked the BUNDLE-010↔009 seam (ast-grep engine + a reusable ast-grep→SARIF converter + engine-organized pack layout, all delivered and tested). So this is no longer a pending dependency; it is satisfied substrate BUNDLE-009 now builds on. **No baked-in shortcut (dogfood ruling):** backstop consumes its substantiveness/contract rules AS a pack, identically to how it already consumes go-standards-pack — enforcement logic does not live in the CLI binary, and there is no privileged "backstop ships built-in rules" tier. **Scope change (2026-06-20):** the existing Go go/parser analyzers (`step_testverify.go`, `step_contract.go`) were previously framed as an *anomaly* "slated to migrate onto the pack model later" — that deferral is now **reversed**. Because BUNDLE-010 delivered the engine + ast-grep, eradicating those baked-in analyzers is **in-scope in this bundle** (Seeds 3 and 4), not a later migration. End state: zero baked-in analyzers in the gate — a strangler repeat of SPEC-034 over the traceability subsystem.
- pkg/gate/step_coverage.go (Go-only target selection), step_testverify.go (go/parser substantiveness), step_contract.go:44 (non-.go skip).
- Driver: the runtime repo (TypeScript + Bun) as the first non-Go backstop project — dogfooding surfaces all three vacuous-pass holes at once.
- Sibling deferral: ISSUE-007 (build/test exclusions) — also stack-generic config, design compatibly.

## Version History

- **0.4.0 (2026-06-22)** — **User-driven SCOPE decisions recorded (distinct from the
  0.3.0 OQ resolutions); maturity HELD at `exploring`** (user drives promotion
  separately; OQ-1..8 NOT re-opened). These four decisions tighten the IN-SCOPE /
  OUT-OF-SCOPE boundary and firm the seeds. New **Scope Decisions / Non-Goals** section
  added.
  **SD-1 (IN SCOPE) — grep/ripgrep engine.** OQ-7's absence probe needs a grep engine
  that does not exist yet (verified: the `DefaultRegistry` in
  `pkg/pack/engine/binding.go` has semgrep/ast-grep/sandbox/config-file/golangci/
  go-build/go-test but NO grep; the allowlist in `pkg/pack/engine/allowlist.go:22-23`
  holds only semgrep+ast-grep). BUNDLE-009 stands it up, SPEC-035-split: **(a)** the
  grep ENGINE BINDING is **pack-declared** (`pattern-arg` + grep→SARIF convert) — no
  baked registry entry, **no ISSUE-027 eradication debt**; **(b)** the `grep`/`rg` TOOL
  is added to the backstop-owned trusted-tool **allowlist** (the "new pack + a few
  trivial allowlist entries" move). IN SCOPE because it's tiny and BUNDLE-009 is its
  only consumer (unlike ast-grep, a broad substrate prerequisite). Feeds the absence
  half of Seed 4; makes that seed self-contained.
  **SD-2 (OUT OF SCOPE / Non-Goal) — `init` toolchain inference.** OQ-1's
  declared==blocks-strictly gate is humane because `init` inference pre-fills
  declarations — but that inference is **ONBOARDING's job** (SPEC-026 / BUNDLE-003).
  **Directionality recorded:** onboarding is a DOWNSTREAM consumer that DEPENDS ON this
  traceability architecture, NOT the reverse; BUNDLE-009 ships FIRST and assumes
  declarations exist (hand- or init-authored), agnostic to how. Sequencing note:
  strict declared-gate is hand-authored-only until onboarding lands; acceptable (early
  adopters hand-write).
  **SD-3 (IN SCOPE — the "beyond Go" proof) — one TypeScript traceability pack; all
  OTHER languages OUT.** A pack is **stack-locked** (verified: `Manifest.Language` is a
  single pack-level field, `pkg/pack/manifest.go:18`), so "beyond Go" literally means
  authoring a second stack-locked pack. To avoid the theoretical-claim trap (title says
  "…beyond Go"), BUNDLE-009 authors ONE non-Go proof pack — a **TS pack** covering
  **substantiveness** (hollow-test on `.test.ts` via ast-grep) + **contracts**
  (signature presence via ast-grep, absence via grep). **Feasible NOW, independent of
  BUNDLE-011** — rides the **structural** engines (ast-grep/grep speak TS), NOT the TS
  toolchain (eslint/tsc). Strategic motivation recorded: the runtime (TS) is **currently
  BLOCKED** by the half-baked pack system — can't gate itself — so the TS proof pack is
  a **live priority**, not a demo. Conscious eyes-open scope **growth** (a real second
  pack is meaningful work). Caveat recorded: **TS coverage is NOT delivered** (needs the
  TS test runner = BUNDLE-011; and coverage is descoped per SD-4). All OTHER languages
  (Python/Rust/…) are separate consumer efforts — OUT.
  **SD-4 (coverage RIPPED OUT; re-implementation DEFERRED; Seed 2 DROPPED).** Coverage
  is architecturally different — **dynamic-toolchain** (run the suite with
  instrumentation, parse lcov/istanbul/go-cover; test-runner-coupled), vs the
  **static-structural** substantiveness/contracts (ast-grep/grep over source).
  BUNDLE-009 **DELETES all THREE baked Go analyzers** — `step_testverify.go`,
  `step_contract.go`, AND `step_coverage.go` — reaching the "zero baked-in traceability
  analyzers" end-state. Substantiveness + contracts are re-implemented as packs IN this
  bundle (Seeds 3+4); coverage's baked Go parsing is **DELETED with NO replacement
  here**, its language-agnostic re-implementation **DEFERRED to a future dedicated
  "language-agnostic coverage" bundle** (near BUNDLE-011, since coverage needs the test
  runner). Interim gap **explicitly accepted**: pre-launch, no users/no remote, and the
  existing check is already a brittle **0.0% false-RED lie** — this removes a
  non-working check, not a capability. **Spec Seed 2 (coverage) DROPPED.**
  Cross-bundle note recorded: **BUNDLE-011 is the natural NEXT-after-009** (together
  they unblock the runtime — 009 delivers the TS TRACEABILITY slice, 011 owes the TS
  TOOLCHAIN slice incl. coverage's test runner); the future language-agnostic-coverage
  bundle sequences near BUNDLE-011.
  Seeds firmed accordingly: **Seed 2 DROPPED**; **Seeds 3+4 now also mandate the TS
  proof pack** (Go migration + TS pack); **Seed 4 stands up the grep engine** (SD-1)
  feeding the absence half. Title reconciled (coverage delivery removed — only its
  baked-analyzer deletion remains); user may finalize a rename at promotion. Earlier
  coverage framings (Current-Thinking obs 4, the eradication-audit `step_coverage.go`
  reference, the promotion-readiness option, the `init` note, seed-sequencing note)
  reconciled to these decisions. **No OQ re-opened; maturity unchanged.**
- **0.3.0 (2026-06-22)** — **User-driven resolution of the three re-opened OQs;
  maturity HELD at `exploring`** (resolving these UNBLOCKS promotion but does not
  perform it — the user promotes separately). All three follow **one architecture**:
  PACK does language-specific work emitting SARIF, GATE does language-agnostic
  semantics + a shared file-scanned guard, BINARY knows zero language/tool specifics;
  all ride SPEC-035's `pattern-arg` + allowlisted-engine substrate; engine-fit split
  is GREP for absence / AST-GREP for structure.
  (a) **OQ-6 RESOLVED — option (a), thin gate-side join.** The `noTarget` check splits:
  PACK emits "which packages/symbols this test references" (positive ast-grep query);
  GATE does a trivial language-agnostic SET-MEMBERSHIP test against the spec's declared
  `implementation.package`. noTarget *semantics* live in the gate as a set test (not a
  baked analyzer); only *extraction* becomes a pack. Rejected: declared-command-per-stack
  (b, heavyweight), drop-entirely (c, a regression), parameterized per-(test,target)
  query (d, over-engineered). Firms Seed 3.
  (b) **OQ-7 RESOLVED — pack-declared forbidden-pattern GREP probe.** Absence is a
  PACK-DECLARED, allowlisted grep/ripgrep forbidden-pattern probe (`pattern-arg`,
  scope = file OR path as a parameter). Engine emits SARIF: match = PRESENT, empty =
  ABSENT; GATE inverts polarity + a thin file-scanned guard (preserving ISSUE-013's
  loud missing/non-Go error). Engine-fit: GREP (text-presence), NOT ast-grep —
  ast-grep needs a per-language grammar and misses comment/string references.
  Explicitly a pack-declared engine, NOT a grep baked into the binary. Accepted trade:
  grep is coarser (comment/string/substring false-FAILs), mitigated by anchored
  patterns + waivability. Authoring insight recorded: codebase-wide grep is the
  spec-authoring tool, file-scoped grep is the gate-enforcement tool.
  (c) **OQ-8 RESOLVED — contract-signature-as-ast-grep-pattern; the PACK compiles the
  signature.** Signature verification is a REQUIRED-pattern ast-grep query (match =
  SATISFIED). Dissolves the brittle `formatFuncSignature`→`signaturesMatch` string
  round-trip that is literally why backstop's own `contract_signature` step is red
  today. Contract stores a human-readable signature; the language PACK provides a
  contract→ast-grep-pattern COMPILER (analogous to the pack's existing SARIF convert
  scripts). Backstop NEVER compiles/renders/understands a signature — doing so in the
  binary would be a P0 zero-baked-language violation (explicitly flagged + rejected).
  Engine-fit: AST-GREP (structural). Resolves Fork 2 (comparison moves structural);
  fidelity moves from exact Go-source rendering to ast-grep pattern precision,
  handled per-language in the pack. Firms Seed 4.
  Spec Seeds 3 and 4 updated to settled mechanics; the cross-cutting theme makes
  "zero baked-in analyzers" a settled outcome (OQ-7's grep probe is pack-declared, so
  no Go-aware probe stays gate-side). The promotion-readiness OPTION (Notes) is now
  moot for OQ-blockedness — all eradication-seed OQs are resolved — but is left as the
  user's call on whether to split seeds for promotion.
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
