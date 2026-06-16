---
title: Pluggable Pack Engines — First-Class `engine:` Field + Dispatch Table
schema_version: bundle/v2

bundle:
  name: pluggable-pack-engines
  version: "0.1.0"
  created: "2026-06-14"
  category: feature

status:
  maturity: defined

problem:
  summary: >
    Which engine executes a pack rule is encoded implicitly, not declared. Two
    hardcodings carry it: (1) gate-time, `mergePackRules` (cmd/backstop/pack_gate.go:97)
    filters `rule.Layer != 2` and dumps every layer-2 rule file into one
    `extraSemgrepConfigs` list — "layer 2" literally means semgrep, and the merge
    assumes a single tool consuming many configs in one invocation; (2)
    fixture-time, `RunToolConfig` (pkg/packval/executor.go:32) is a one-arm
    `switch` on `tool` (golangci-lint), with `RunSemgrep` a separate hardcoded
    method beside it. The `Rule` struct (pkg/pack/manifest.go:38) has `Layer int`
    and `Validator string` but no `engine` field. Result: adding a new engine
    (ast-grep, eslint, ruff, clippy) is a code change in two places, not a
    declaration — the opposite of the pack thesis (integrate, don't build).
  user_story: >
    As the maintainer who needs portable substantiveness/contract checks for
    non-Go stacks (BUNDLE-009), I want a rule to declare which engine runs it as
    a first-class field, dispatched through a table rather than a layer==2 /
    one-arm-switch hardcoding, so that wiring ast-grep — and after it eslint,
    ruff, clippy — is config plus (at most) a registered runner, not a surgical
    edit to the gate and the fixture executor each time. ast-grep is the first
    new engine because BUNDLE-009's query-pack layer is blocked on it.

solution:
  approach: >
    First-class the engine as a declared field on a pack rule and dispatch through
    an explicit `engine → {command, convert?, requires[], forbids[]}` table, rather
    than the current implicit `layer==2 ⇒ semgrep` routing and one-arm
    `RunToolConfig` switch. The work is convergence, not invention: ISSUE-003 already
    ships the substrate (`ToolchainEntry{Command, Format}` + the `formatParsers`
    registry incl. `sarif` + the `lookupParser` fail-loud contract + the generic
    data-driven executor). This bundle converges the three bespoke holdouts onto it
    (native `semgrepExecutor`, pack `mergePackRules`, pack `RunToolConfig`) and adds
    a `convert` step plus a rule-file-injection seam. Backstop owns exactly one
    output parser (`parseSarif`); engines are open/declared and any non-SARIF tool
    converts to SARIF outside the binary via a sandboxed, pack-declared `convert`
    executable. `layer` retires; its per-tier field requirements re-key from
    layer→engine as each engine's field-contract. Two pillars: (1) pluggable engines
    + ast-grep wired with a proof rule end-to-end; (2) rip out the native-standards
    arm so the gate is packs-only, dogfood-consuming `backstop-go-pack`. This bundle
    ships the engine machinery + ast-grep + a trivial proof rule; BUNDLE-009 authors
    the real substantiveness/contract rule packs on top.
  assumptions:
    - >
      ISSUE-003's substrate is reusable as the engine substrate — `ToolchainEntry`,
      the `formatParsers` registry (already contains `sarif`), `lookupParser`'s
      fail-loud contract, and the generic data-driven executor — so the net new work
      is the `convert` step + the rule-file-injection seam, not a from-scratch model.
    - >
      SARIF is the lingua franca: ~2 output formats cover the mainstream polyglot
      set and native SARIF is table stakes for the modern/security tier (spike,
      2026-06-14), so an open-engine / closed-format inversion (backstop owns
      `parseSarif`, the ecosystem maps everything else into SARIF) is real and cheap.
    - >
      The installed base of `layer: 2` packs is N=1 (`backstop-go-pack`, 14 rules,
      user-owned); there is no third-party ecosystem yet, so a flag-day migration is
      safe and no deprecation window / alias machinery is warranted.
    - >
      The required engines can be assumed on PATH and fail loud if absent (pack /
      project declares the dependency); `EnsureSemgrep`-style provisioning is treated
      as a pre-existing anomaly, not solved here.

requirements:
  - id: REQ-001
    text: >
      A pack rule must declare its execution engine as a first-class `engine` field,
      and the gate must dispatch on it through an explicit table
      (`engine → {command, convert?, requires[], forbids[]}`) rather than the
      implicit `layer==2 ⇒ semgrep` routing or the one-arm `RunToolConfig` switch.
      Adding a new engine must be a declaration (config plus, at most, a registered
      runner), never a surgical edit to the gate and fixture executor. (DD-1, DD-2)
  - id: REQ-002
    text: >
      The `layer` field (1/2/3) must retire as the execution selector. `engine`
      becomes the single first-class key; a rule carrying `layer` without `engine`
      after cutover is a class-1 broken declaration that is both loud and blocking
      (config error). The hashmap has one key — `engine` and `layer` must not both
      encode it. (DD-1, DD-5)
  - id: REQ-003
    text: >
      Each layer's per-tier field requirements (`requires`/`forbids` — e.g. semgrep
      needs `rule_path`+`standard`, a sandbox validator needs `input_scope`+
      `category`) must re-key from layer→engine and become each engine's
      field-contract. Validation must verify a rule's fields match its declared
      engine (an honest engine-fit check). `validateLayer` (accepts 1/2/3) and the
      `layer`-keyed `validateLayerFields` get rewritten as engine-keyed. (DD-1)
  - id: REQ-004
    text: >
      Engine-fit validation must only verify field-consistency with the author's
      declared engine; it must not guide or recommend engine choice via content
      analysis. The author asserts the engine; backstop enforces fit, never questions
      the assertion. (DD-1)
  - id: REQ-005
    text: >
      Engines must be open/declared, never enumerated in backstop's Go. The finite
      thing backstop owns is output formats, not engines: backstop owns exactly one
      output parser, `parseSarif`, reusing ISSUE-003's `formatParsers` registry +
      `lookupParser` fail-loud contract. Adding eslint/ruff/clippy/clj-kondo must be
      a declaration, not a code change. (DD-2)
  - id: REQ-006
    text: >
      The output contract for findings engines must be strict SARIF, not "SARIF or
      JSON." Any non-SARIF findings tool must convert to SARIF outside backstop. The
      engine binding carries no `format` selector (it could only ever be `sarif`):
      a SARIF-native engine is `{command}`, a non-SARIF engine is `{command, convert}`,
      and `parseSarif` is the contract (it already fail-louds on non-SARIF). (DD-2)
  - id: REQ-007
    text: >
      Conversion must run as a sandboxed, pack-declared `convert` executable: the
      gate pipes `tool stdout → convert stdin → SARIF → parseSarif` in Go (no shell),
      resolving the converter relative to the pack dir and running it via the same
      `SandboxedRun` layer-3 validators already use (no new trust model). `convert`
      is empty for SARIF-native tools. Backstop embeds no transform engine. (DD-2)
  - id: REQ-008
    text: >
      Backstop must not build or host converters. It points pack authors at existing
      converters (Microsoft `sarif-js-sdk`/`sarif-tools`, GitHub `advanced-security`)
      and native `--format sarif` flags, documenting a per-known-tool `pack.yml`
      reference table. The lone genuine gap, ast-grep (JSON-only, no maintained SARIF
      converter), ships its own stdin→SARIF converter authored as a standalone script.
      A backstop-owned x-to-sarif repo is rejected. (DD-2, DD-7)
  - id: REQ-009
    text: >
      The runner must capture stdout cleanly, separate from stderr, replacing the
      current `CombinedOutput()` (runner.go:30) which merges stderr into stdout and
      corrupts SARIF parsing. (DD-2)
  - id: REQ-010
    text: >
      ast-grep must be wired as the first new engine end-to-end through the gate,
      with a trivial proof rule demonstrating engine dispatch + the `convert` step
      working from declaration through to normalized violations. (DD-6)
  - id: REQ-011
    text: >
      Gate-time engine dispatch (locus A) must replace the single semgrep-only
      executor (`semgrepExecutor`, check.go:559) with "group rules by declared engine,
      run each, normalize to SARIF," fed from the pack source. There is no deferral:
      feeding ast-grep files into a semgrep-only executor does nothing — generalizing
      that executor is the ast-grep work. (DD-3)
  - id: REQ-012
    text: >
      Validation-time fixture execution (locus B, `pkg/packval`, the `pack validate`
      path) must receive the same engine generalization as the gate-time path. (DD-3)
  - id: REQ-013
    text: >
      The shared `EngineBinding` type must live where `pkg/check`, `pkg/packval`, and
      `cmd/backstop` can all use it without an import cycle. Pin this integration
      hotspot first. Native toolchain (per-stack registry) and packs (`pack.yml`) both
      emit records of the same `EngineBinding {command, convert?, format? = sarif}`
      shape — same schema, different containers; do not merge the containers. (DD-3)
  - id: REQ-014
    text: >
      `sandbox` (was layer 3) plus build/test passes are non-SARIF edges (exit-code /
      pass-fail, not located findings) and must not ride `parseSarif`. ISSUE-003's
      `ToolchainEntry.Format` survives for build/test (`go-test`/`go-build`/
      `regex-lines` stay); only lint converges to SARIF. The per-tool lint JSON parsers
      (`golangci-json`, `eslint-json`) retire as migration debt (those tools emit SARIF
      natively); the bespoke parsers are removed, not extracted. (DD-2)
  - id: REQ-015
    text: >
      Migration of the one existing `layer: 2` pack must be a single sibling edit:
      bump `backstop-go-pack`'s 14 rules to `engine: semgrep` (mechanical). No silent
      grandfather (`layer:2 → engine:semgrep`), no deprecation window, no alias
      machinery. The pack migration is tied to pillar-1's layer→engine cutover (core's
      reader and the pack repo flip together), not pillar-2. (DD-5)
  - id: REQ-016
    text: >
      Pillar-2 must delete the `.standard.md → standards-compiler → manifestDir` arm,
      have backstop-core dogfood-consume the existing `backstop-go-pack`, drop
      `STD-GO-001`, and retire the standards compiler from core. This runs on the
      current semgrep engine, independent of ast-grep, and lands first as the
      single-source foundation pillar-1 builds on (collapsing locus A's input to a
      single source: packs). (DD-6)
  - id: REQ-017
    text: >
      The BUNDLE-009 contract seam must be locked: this bundle ships engine machinery
      + ast-grep + a trivial proof rule; BUNDLE-009 authors the real
      substantiveness/contract rule packs on top. Neither spec absorbs the other's
      work. (DD-6)
  - id: REQ-018
    text: >
      The engine model must be engine-shape-agnostic, with Layer-0 native toolchain
      as the first-class foundation. It must handle BOTH config-driven linters
      (Layer 0 — golangci-lint/eslint/ruff/clippy/tsc — supported FIRST, the primary
      invocation shape: a config file feeding the tool's own built-in rules) AND
      rule-fed engines (Layer 1/2 — semgrep/ast-grep — secondary, where backstop
      supplies the rule files). Adding a new native linter must be a declaration, no
      backstop Go. Native/language breadth ("plug an agent into any project, get
      deterministic gates immediately") is a first-class goal of this bundle, NOT a
      deferred frontier. (DD-8, REQ-001)
  - id: REQ-019
    text: >
      Engine provisioning must split by who owns the tool. Layer-0 native toolchain is
      ASSUMED-PRESENT — fail loud if a project lacks its own linters (it is not
      backstop's job to install a project's chosen toolchain). Backstop-INTRODUCED
      engines (semgrep, ast-grep) are AUTO-PROVISIONED via a declared, pinned install
      carried in the engine/pack declaration, run and verified through the existing
      `backstop.lock` / `VerifyLock` infra — generic and data-driven, with NO
      per-engine Go. `EnsureSemgrep`'s bespoke install logic retires into this declared
      mechanism (another baked-in-tool anomaly removed). Trust surface equals the
      `convert` step's: pinned and verified. (DD-8, DD-2)
  - id: REQ-020
    text: >
      The engine binding must declare input injection via a structured `input_mode`
      enum (`config-file` — primary, the tool runs its own built-in rules tuned by an
      optional pack-supplied config / `rule-flags` — each rule file becomes a repeated
      flag / `rule-dir` — rule files collected into a directory passed to the tool /
      `none` — no injection, the executable is the logic) plus an `input_flag` string.
      This is data-driven with no per-engine Go and extends ISSUE-003's
      `ToolchainEntry`/`ScopeKind` pattern (which declares how files attach) with the
      analogous declaration for how rules/config attach. (DD-9, DD-1)
  - id: REQ-021
    text: >
      Packs must follow an engine-organized layout convention: a directory per engine
      holds that engine's rules/config, and the layout corresponds to the engine's
      `input_mode` (`semgrep/` → rule-flags, `ast-grep/` → rule-dir, `linter/` →
      config-file, `scripts/` → none). The engine declaration points at its inputs and
      the directory layout is `input_mode` made physical. (DD-10, DD-9)
---

# Pluggable Pack Engines — First-Class `engine:` Field + Dispatch Table

## Current Thinking

**The organizing principle: a preference-ordered escalation ladder.** The
gate/engine model is a ladder — use the *lowest layer that does the job*:
- **Layer 0 — native toolchain.** The project's own config-driven linters/build/
  test (golangci-lint, eslint, tsc, clippy, ruff) running with their OWN default
  rules. Supported FIRST and best; this is the foundation, not a footnote.
- **Layer 1 — query engine + existing rules.** semgrep / ast-grep against rules
  that already exist.
- **Layer 2 — custom rules on that query engine.** Author new rules for an
  existing engine.
- **Layer 3 — custom script** (the sandbox validator). Last resort.

Crucially, **every layer is just an engine `{command, convert?} → SARIF`**; the
ladder is a *priority ordering over the one engine substrate*, not extra
machinery. The product promise — "plug an agent into any project, get
deterministic gates immediately" — is delivered mostly at Layer 0, so
language/engine **breadth is core to this bundle, not a deferrable frontier**.
The only deferrable frontier is the pack *marketplace* (publishing / LLM-reviewer
/ supply-chain). DD-8 records this frame; DD-1's `layer` retirement does not undo
it — `layer` retires as the *execution selector* (the engine name now carries
routing), while the *preference ordering* it implied is promoted to an explicit,
intentional ladder.

This is also the pack-level expression of "integrate, don't build" ([[project_bundle_is_the_product]],
[[project_pack_engine_model]]): engine should be a **pluggable dimension** of a
rule, so the set of supported engines grows by declaration, not by editing the
gate. The work is pulled *ahead* of BUNDLE-009 because the engine is
higher-leverage — it unlocks layer-2 pack rules broadly, not just three
traceability steps — and BUNDLE-009 is a *consumer* of it, not its owner. The
2026-06-14 spike already de-risked the consumer side (hollow-test detection is
expressible as declarative per-language ast-grep YAML); what remains unproven is
the *engine* substrate, which is this bundle.

Three observations frame the exploration:

1. **The hardcoding is in two places, not one, and they don't share an
   abstraction.** Gate-time enforcement (`mergePackRules` → `extraSemgrepConfigs`,
   consumed at cmd/backstop/gate.go:246 and code_check.go:83) and fixture-time
   validation (`FixtureExecutor.RunSemgrep` / `RunToolConfig`,
   pkg/packval/executor.go) each hardcode semgrep/golangci-lint independently. A
   first-class engine model has to decide whether these two dispatch sites
   collapse into one engine interface or stay deliberately parallel.

2. **The "merge configs, one invocation" assumption is semgrep-shaped and does
   not generalize.** `mergePackRules` collects every layer-2 rule file into a
   flat list because semgrep accepts many `--config` files in a single call.
   ast-grep's invocation model is different (scan against a rule dir / per-rule),
   and golangci-lint already needed its own arm. So the dispatch can't be "gather
   all rule paths → hand to the one tool"; it has to group rules *by engine* and
   let each engine define how it gathers, invokes, and parses.

3. **There's already a latent second engine (`tool_config.tool` = golangci-lint)
   that the design must reconcile with.** The new `engine:` field can't be
   bolted on beside `layer` and `tool_config.tool` without ruling on how the
   three relate — otherwise we've added a third implicit dimension instead of
   removing the first two.

The load-bearing reframe that organizes the resolved decisions below:
**convergence, not invention.** The engine substrate already exists — ISSUE-003's
`ToolchainEntry{Command, Format}` + `formatParsers` (incl. `sarif`) +
`lookupParser` fail-loud + the generic data-driven executor. This bundle
*converges* the three bespoke holdouts onto it (native `semgrepExecutor`, pack
`mergePackRules`, pack `RunToolConfig`) and adds the `convert` step +
rule-file-injection seam. Net scope is smaller and far less speculative than the
original framing.

**Thin on knowledge, firm on enforcement (DD-10).** With the gather/invoke seam
resolved as a declared `input_mode` (DD-9) and an engine-organized pack layout, all
engine/rule opinion — which engines, which rules, layout, converter, provisioning —
externalizes into the pack as data, reducing the backstop CLI to a thin
deterministic harness: discover declarations → gather inputs by mode → run command →
convert → parseSarif → normalize. Thin BUT real: backstop still owns the execution
core (gather/run/convert/parse-SARIF/normalize/scope/provision/fail-loud) and the
one parser; the *opinion* is externalized, the *execution discipline* stays in the
CLI because that's the part the LLM gets no veto over ([[project_prompts_are_vibes]]).

Grounding (on main as of 2026-06-14):
- `Rule`: `Layer int`, `RulePath`, `Validator`, no `engine` (pkg/pack/manifest.go:38).
- `validateLayer` accepts only 1/2/3 (pkg/pack/manifest.go:324).
- Gate-time: `mergePackRules` filters `Layer != 2`, returns `[]string` rule paths
  → `extraSemgrepConfigs` (cmd/backstop/pack_gate.go:97, gate.go:246).
- Fixture-time: `RunSemgrep` hardcoded; `RunToolConfig` one-arm switch
  (pkg/packval/executor.go:25,32).
- semgrep findings → Violations parsing currently lives in the gate's code-check
  path (the format-normalization seam any new engine also needs).

## Draft Requirements

The formal, traceable requirements are enumerated in the `requirements:`
frontmatter array (REQ-001 … REQ-021). Each carries the design decisions
(DD-N below) it derives from. Summary of what this bundle commits to build:

- **First-class `engine` + dispatch table; retire `layer`** (REQ-001…REQ-004) —
  the layer→engine cutover, engine-keyed field-contracts, verify-not-guide
  validation.
- **Open engines, closed formats — strict SARIF + sandboxed `convert`**
  (REQ-005…REQ-009) — `parseSarif` is the sole owned parser; non-SARIF tools
  convert outside the binary; the runner stops merging stderr into stdout.
- **ast-grep as the first new engine + proof rule** (REQ-010) end-to-end.
- **Two convergence loci, no deferral** (REQ-011…REQ-013) — gate-time dispatch
  and fixture-time execution generalized; shared `EngineBinding` placed to avoid
  an import cycle.
- **Carve-outs and migration debt** (REQ-014) — build/test keep their parsers;
  lint converges to SARIF; bespoke lint JSON parsers retire.
- **Flag-day pack migration** (REQ-015) — bump `backstop-go-pack` to
  `engine: semgrep`; no grandfather/alias.
- **Pillar-2: packs-only core** (REQ-016) — delete the native-standards arm,
  dogfood-consume `backstop-go-pack`, retire the standards compiler.
- **BUNDLE-009 seam lock** (REQ-017) — proof rule here, real rule packs there.
- **Escalation ladder: Layer-0-first, engine-shape-agnostic** (REQ-018) — the
  model handles config-driven linters (the primary shape) and rule-fed engines
  alike; native/language breadth is core, not a deferred frontier (DD-8).
- **Split engine provisioning** (REQ-019) — Layer-0 assume-present (fail loud);
  backstop-introduced engines auto-provisioned via `backstop.lock`/`VerifyLock`,
  retiring `EnsureSemgrep`.
- **Structured `input_mode` gather/invoke seam** (REQ-020) — input injection is a
  declared `input_mode` enum (`config-file` primary / `rule-flags` / `rule-dir` /
  `none`) + `input_flag`; data-driven, no per-engine Go; extends
  `ToolchainEntry`/`ScopeKind` (DD-9).
- **Engine-organized pack layout** (REQ-021) — a directory per engine; the layout
  is `input_mode` made physical (DD-10).

All six open questions (OQ-1…OQ-6) are now resolved; OQ-4's gather/invoke seam
landed as DD-9 + DD-10 (REQ-020/REQ-021).

## Draft Design Decisions

Each decision below was reached during the 2026-06-14/16 working session and
carries its originating open question for traceability.

**DD-1 — One key (`engine`); `layer` retires into each engine's field-contract.**
*(from OQ-1, resolved 2026-06-16.)* Two discoveries reshaped this:
1. **`layer` is already a leaky engine selector.** Layer 1 = configured native
   linter (engine named by `tool_config.tool`); Layer 2 = semgrep (engine
   implicit); Layer 3 = sandboxed custom validator (engine = the validator
   binary). Each layer already *is* a different engine — `layer` is an
   under-powered proxy for the dimension we're first-classing.
2. **The engine model already exists for native toolchains.** ISSUE-003's
   `ToolchainEntry{Command, Format}` + `formatParsers` registry + the generic
   data-driven executor (TS stack) *is* "declared command + named parser." The
   three bespoke holdouts that never adopted it — native `semgrepExecutor`
   (check.go:559), pack `mergePackRules` (layer==2⇒semgrep), pack
   `RunToolConfig` switch — are the anomalies. So "first-class engine" mostly
   means **converging those three onto the substrate ISSUE-003 already ships**,
   not inventing one.

An engine binding reduces to `{command, convert?}` + a **rule-file-injection
seam** (how a pack's rule files become `--config X` / a rule dir on the engine's
command line — the one thing ISSUE-003 didn't need, since native passes carry no
pack-supplied rules).

*The hashmap reframe (resolution).* `layer` (1/2/3) is *already* a hashmap key
whose values are hardcoded and scattered across two files: routing
(pack_gate.go) AND a per-layer field-contract (`validateLayerFields`,
validate_manifest.go:105). Promote that implicit map to an explicit one keyed on
**`engine`**, where the value is one coherent record:
`engine → { command, convert?, requires[], forbids[] }`. The three layers become
three entries (`tool_config`/`semgrep`/`sandbox`); ast-grep is a fourth; packs
contribute their own. A hashmap has **one key**, so `layer` + `engine` would be
two keys that must agree (redundant, can drift) — therefore **retire `layer`**:
- `engine` becomes the first-class execution selector (Job 1).
- `layer`'s per-tier field requirements (`requires`/`forbids` — e.g. semgrep
  needs `rule_path`+`standard`; a sandbox validator needs `input_scope`+
  `category`) **re-key from layer→engine**, becoming each engine's
  **field-contract** (Job 2). This makes the validation honest about what it
  already is — an **engine-fit check** — instead of hiding it behind an int.
- **Two declaration surfaces, one schema** (OQ-1 sub-Q2, resolved): native
  toolchain (per-stack registry) and packs (`pack.yml`) both emit records of the
  same `EngineBinding { command, convert?, format? = sarif }` shape (= ISSUE-003's
  `ToolchainEntry` + `convert`, `format` optional/default-sarif, invisible to
  findings authors). Same schema, different containers — don't merge the
  containers.
- **`sandbox` (was layer 3) + build/test are the non-SARIF edges** — exit-code /
  pass-fail, not findings, so they don't ride `parseSarif`. Everything
  findings-shaped (semgrep/ast-grep/lint) is uniform.

*Verify, not guide (clarified 2026-06-16).* This engine-fit validation only
*verifies* a rule's fields match its declared engine — it does NOT *guide* engine
choice (no content analysis recommending declarative-vs-custom). The author
asserts the engine; backstop enforces field-consistency with the assertion, never
questions it. Steering authors toward a declarative engine and reserving
`sandbox`/custom for genuine need is a future "make-the-right-thing-easy"
capability (see Out of Scope / Non-Goals), not this bundle.

*Migration caveat (the cost):* SPEC-013/014's `layer`-keyed validation and
`validateLayer` (accepts 1/2/3) get rewritten to be `engine`-keyed. Mechanically
similar (re-key a switch), but it's a real schema-surface change, not free.

**DD-2 — Engines are declared/open, with a strict SARIF-only output contract.**
*(from OQ-2, resolved 2026-06-14.)* Engines are **open/declared**, never
enumerated in backstop's Go. Backstop owns exactly **one output parser —
`parseSarif`** — and a declared `convert` step; everything else is config. In the
order the conversation produced it:
- **The finite thing backstop owns is output *formats*, not *engines*.** A
  declared engine is `{command, convert?}` (no `format` selector — see below).
  Adding eslint/ruff/clippy/clj-kondo is a declaration, not a code change —
  exactly the memory's stated intent. This reuses ISSUE-003's `formatParsers`
  registry + `lookupParser` fail-loud contract (pkg/check/parsers.go), which
  **already exists and already contains a `sarif` parser**.
- **Strict SARIF, not "SARIF or JSON."** Raw JSON is not one schema (eslint JSON
  ≠ golangci JSON ≠ ast-grep JSON — which is *why* the current registry has
  per-tool parsers). Parsing raw tool JSON means per-tool knowledge *in the
  binary* = the "engine list" problem relocated. So backstop mandates SARIF (a
  real, single OASIS schema: one parser reads any emitter) and **forces any
  non-SARIF tool to convert to SARIF outside backstop**. The spike confirms SARIF
  is the lingua franca: ~2 formats cover the mainstream, native SARIF is now table
  stakes for the modern/security tier.
- **Conversion mechanism = a sandboxed, pack-declared `convert` executable.** The
  gate runs a two-process pipe *in Go* (no shell): `tool stdout → convert stdin →
  SARIF → parseSarif`. `convert` is empty for SARIF-native tools. The converter is
  pack-author code, resolved relative to the pack dir and run via the **same
  `SandboxedRun` layer-3 validators already use** — no new trust model. The
  converter can be any executable (binary / python / `jq -f map.jq`); backstop
  embeds no transform engine, so it stays ignorant of every tool.
- **Converters are NOT backstop's to build or host.** Microsoft (`sarif-js-sdk`,
  `sarif-tools`) and GitHub `advanced-security` already maintain the converter
  supply, and a growing set of tools emit SARIF natively. Backstop's job is to
  **point pack authors at the existing converter / native `--format sarif` flag
  and document the `pack.yml` snippet per known/supported tool** — a reference
  table, not a repo. (See DD-7.)
- **ast-grep is the lone genuine gap** (JSON-only, no maintained SARIF converter
  found). Its pack ships its own stdin→SARIF converter, authored as a standalone
  script so it's repo-ready if an ecosystem ever wants it.
- **Prerequisite finding:** the runner uses `CombinedOutput()` (merges stderr
  into stdout, runner.go:30) — corrupts SARIF parsing. Strict SARIF **requires**
  capturing stdout cleanly, separate from stderr. Small but mandatory runner
  change.
- **No `format` param on the engine binding (refinement).** Since the contract is
  strictly SARIF, `format` could only ever be `sarif` — a dead field that invites
  misuse. Dropped: the binding is `{command, convert?}`, and `parseSarif` *is* the
  contract (it already fail-louds on non-SARIF, so a converter emitting garbage is
  caught without a selector). SARIF-native engine = `{command}`; non-SARIF =
  `{command, convert}`.
- **Carve-out — build/test are not findings.** The SARIF mandate applies to
  *findings* engines (semgrep, ast-grep, lint, substantiveness, contracts). Build
  and test passes emit pass/fail + failures + coverage, not located findings —
  SARIF is the wrong shape. So ISSUE-003's `ToolchainEntry.Format` **survives for
  build/test** (`go-test`/`regex-lines` stay); only **lint converges to SARIF**.
  The `format` selector dies in the engine model but lingers where output
  genuinely isn't findings-shaped.
- **Consequence:** the per-tool *lint* JSON parsers (`golangci-json`,
  `eslint-json`) become **migration debt** — those tools emit SARIF natively
  (golangci v2, eslint via MS formatter); retire the bespoke parsers, don't
  extract them. (`go-test`/`go-build`/`regex-lines` stay — see carve-out.)

**DD-3 — Two convergence loci, both in-bundle, no deferral.** *(from OQ-3,
resolved 2026-06-16 — corrected from an earlier wrong decomposition.)* Earlier
framing said "three bespoke sites, defer the native one." That was wrong on two
counts:
1. **The decomposition was off.** At gate time there is **one** semgrep executor
   (`semgrepExecutor`, check.go:559); `mergePackRules` is not a parallel executor,
   it's a *feeder* that drops pack rule files into that one executor's `--config`
   list (alongside the project's compiled-standards `manifestDir`). So "native vs
   pack" = two rule *sources into one executor*, not two executors. You **cannot**
   "do packs, defer native" — feeding ast-grep files into a semgrep-only executor
   does nothing; generalizing that executor *is* the ast-grep work. Nothing was
   deferrable.
2. **Deferring cleanup is a human-workflow reflex** (bandwidth), not a real
   constraint in an agent-native flow; and two coexisting semgrep paths is itself
   the [[feedback_integration_gap]] drift risk. Converging all of it *reduces*
   risk — backstop's own gate (TDD/verify) is what makes churning tested code safe
   ([[project_prompts_are_vibes]]); refusing to is a dogfood tell.

**Resolution:** there are **two real loci**, both in-bundle, no deferral:
(A) **gate-time engine dispatch** — replace the one semgrep-only executor with
"group rules by declared engine, run each, normalize to SARIF," fed from the pack
source; (B) **validation-time fixture execution** (`pkg/packval`) — the genuinely
separate path (`pack validate`), same generalization. Note: under pillar-2 (DD-6)
the *project-standards* source is **deleted**, so locus A's input collapses to a
single source (packs). **Integration hotspot to pin first** (per
[[feedback_integration_gap]]): where the shared `EngineBinding` lives so
`pkg/check`, `pkg/packval`, and `cmd/backstop` use it without an import cycle.

**DD-4 — Parse half of the engine contract is settled; gather/invoke half resolved
in DD-9.** *(from OQ-4: parse half resolved by DD-2; gather/invoke half now resolved
by DD-9, 2026-06-16.)* *Parse* is settled: output is always SARIF → `parseSarif`
(with the sandboxed `convert` step for non-SARIF tools). The **gather/invoke** half —
how pack-supplied rules/config become the engine's invocation — is resolved in DD-9
as a structured, declared `input_mode` on the binding (no longer open).

**DD-5 — Flag-day migrate the one `layer: 2` pack; no grandfather, no alias.**
*(from OQ-5, resolved 2026-06-16.)* The reframe: "back-compat" assumes an installed
base, and there isn't one — the only `layer: 2` pack is **`backstop-go-pack`**
(14 rules), user-owned; no third-party ecosystem yet (defer-frontier principle).
Population = **N=1**. So:
- **No silent grandfather** (`layer:2 → engine:semgrep` implicitly): that keeps
  `layer` alive as a hidden alias, defeating DD-1's retirement and re-introducing
  the implicit encoding DD-2 removed.
- **No deprecation window / alias machinery**: ecosystem-migration tooling for an
  ecosystem that doesn't exist (YAGNI).
- **`engine` required; `layer` truly gone.** A rule with `layer` and no `engine`
  after cutover = **class-1 broken declaration** → loud AND blocks (config error),
  per [[feedback_loud_not_blocking]].
- **Migration = one sibling edit:** bump `backstop-go-pack`'s 14 rules to
  `engine: semgrep` (mechanical).
- **Sequencing:** the pack migration is tied to **pillar-1** (the layer→engine
  cutover), NOT pillar-2 — pillar-2 consumes go-pack as `layer: 2` as-is; core's
  reader and the pack repo flip to `engine` together at pillar-1.
- **Principle to not blindly repeat:** the flag-day is correct *only because*
  there's no external pack base. A future field-retirement once external packs
  exist would need a real deprecation window. Not now.

**DD-6 — Scope is two pillars + a locked BUNDLE-009 seam.** *(from OQ-6; pillar-2
added & sized 2026-06-16; BUNDLE-009 seam still to confirm.)* This bundle has
**two pillars**:
- **Pillar 1 — pluggable engines.** First-class `engine` + dispatch (DD-1/2/3),
  ast-grep wired as the first new engine + a proof rule end-to-end through the
  gate. The machinery BUNDLE-009 consumes.
- **Pillar 2 — rip out native standards; packs-only.** Delete the
  `.standard.md → standards-compiler → manifestDir` arm; have backstop-core
  **dogfood-consume the existing `backstop-go-pack`**; drop `STD-GO-001` + retire
  the standards compiler from core. **Sized small**: DIR-005 (done 2026-04-23)
  already published the pack, so this is "delete an arm + declare a pack," not
  authoring. Runs on the *current semgrep* engine, **independent of ast-grep** →
  can land **first** as the single-source foundation pillar-1 then builds on.
  Finishes DIR-005's "language-agnostic core" intent (left half-done). See
  [[project_packs_only_no_native_standards]].
- **The BUNDLE-009 seam (still to confirm):** this bundle ships engine machinery +
  ast-grep + a *trivial proof* rule; **BUNDLE-009 authors the real
  substantiveness/contract rule packs** on top. Lock the contract so neither spec
  absorbs the other's work.

Dogfood note ([[feedback_dogfood_rules_as_packs]]): the baked-in Go `go/parser`
substantiveness analyzer (step_testverify.go) is a *separate* anomaly to migrate
later (BUNDLE-009 territory) — distinct from pillar-2's standards-compiler removal,
though both are "stop baking rules into the binary."

**Suggested sequencing:** pillar-2 (single-source, semgrep-only, smallest) →
pillar-1 gate-time dispatch + ast-grep → pillar-1 fixture-time → BUNDLE-009 rules.

**DD-7 — Backstop owns no converters; point at existing supply, own only the
ast-grep gap as a script.** *(from OQ-2, separated for emphasis.)* A
backstop-owned x-to-sarif converter repo was **considered and rejected** as
premature/own-nothing. Microsoft (`sarif-js-sdk`, `sarif-tools`) and GitHub
`advanced-security` already maintain the converter supply, and native `--format
sarif` is increasingly table stakes — so backstop **points authors at existing
converters + documents the `pack.yml` snippet per known tool** (a reference
table), owning none. Shipping/maintaining per-tool converters is a format-drift +
per-platform build-matrix burden a solo founder shouldn't carry; prefer scripts
over binaries for the one gap we do own (ast-grep). Aligns with "defer the pack
frontier until external authors" ([[project_pack_engine_model]]) and
[[project_independence_thesis]]. If a gap ever needs filling, contribute the
converter **upstream**, not into backstop.

**DD-8 — The engine model is a preference-ordered escalation ladder over one
SARIF substrate.** *(frame decision, 2026-06-16; the organizing principle the
bundle hangs on.)* The four execution tiers are a *priority ordering*, not four
mechanisms — use the **lowest layer that does the job**:
- **Layer 0 — native toolchain** (config-driven linters/build/test:
  golangci-lint, eslint, tsc, clippy, ruff, run with their OWN default rules).
  The foundation, supported **first and best**.
- **Layer 1 — query engine + existing rules** (semgrep / ast-grep).
- **Layer 2 — custom rules on that query engine.**
- **Layer 3 — custom script** (the sandbox validator). Last resort.

Every layer reduces to the *same* engine shape — `{command, convert?} → SARIF`
(with the non-findings carve-out of DD-2 for build/test/sandbox exit-code edges).
The ladder is therefore a **priority ordering over the single engine substrate**,
adding **no new machinery** beyond what DD-1/DD-2 already establish. Two
consequences this bundle commits to:
1. **Engine-shape-agnostic, Layer-0-first (REQ-018).** The model must natively
   handle the *config-driven* invocation shape (a config file feeding the tool's
   own built-in rules) as the **primary** case, with the *rule-fed* shape
   (backstop supplies `--config`/rule-dir) second. A config-driven linter is an
   engine whose "rules" are its own defaults; a query engine is an engine whose
   rules are pack-supplied. Same `EngineBinding`, different rule-injection posture
   — which is exactly the seam OQ-4 must resolve.
2. **Breadth is core, marketplace is frontier.** Because "plug into any project,
   get deterministic gates immediately" is delivered mostly at Layer 0,
   language/engine **breadth is in-scope and first-class**, not deferred. The only
   deferred frontier is the pack *marketplace* (publishing / LLM-reviewer /
   supply-chain), per Non-Goals. Reconciles with DD-1: `layer` retires as the
   *execution selector*, but the *preference ordering* it encoded is promoted to
   this explicit, intentional ladder rather than discarded.

**DD-9 — The gather/invoke seam is a structured, declared `input_mode` on the engine
binding.** *(from OQ-4, resolved 2026-06-16 — the last open question.)* The
rule-file-injection seam is resolved as a structured `input_mode` enum + `input_flag`
string on the binding — NOT a per-engine Go `Engine` interface (that reopens the
closed dispatch table killed in DD-1/OQ-2) and NOT a raw string-template DSL (a
fragile mini-language). It **extends ISSUE-003's `ToolchainEntry`/`ScopeKind`
pattern** — which already declares how *files* attach to an invocation — with the
analogous declaration for how *rules/config* attach. The binding gains `input_mode`
(enum) + `input_flag` (string). Enum values and the ladder mapping:
- **`config-file`** — single, optional pack-supplied config; the tool runs its OWN
  built-in rules tuned by it. **PRIMARY shape** (Layer-0 linters:
  golangci/eslint/tsc). Satisfies the DD-8/REQ-018 constraint that config-driven
  linters are first-class.
- **`rule-flags`** — each pack rule file becomes a repeated flag (e.g. `--config X`).
  semgrep.
- **`rule-dir`** — pack rule files collected into a directory passed to the tool.
  ast-grep.
- **`none`** — no rule/config injection; the executable IS the logic. sandbox
  (Layer 3).

Combined with OQ-2's output half, the binding is now fully specified end-to-end:
`{command, input_mode + input_flag, scope_kind, convert?, provision?} → SARIF →
parseSarif`. **Escape hatch (future/YAGNI):** if an exotic engine ever needs a shape
no mode expresses, add a mode or wrap it — do NOT build a template DSL preemptively
(the spike's engine survey says these four modes cover the realistic set).

**DD-10 — Pack layout convention + thin-executor thesis.** *(2026-06-16, with DD-9.)*
Packs are **engine-organized**: a directory per engine holds that engine's
rules/config, and the directory layout IS the `input_mode` made physical:
```
pack.yml            # engine bindings (engine + input_mode + pointer)
semgrep/  *.yml      # rule-flags
ast-grep/ *.yml      # rule-dir
linter/   .golangci.yml  # config-file (tool runs its own rules)
scripts/  check.sh   # none
fixtures/
```
This externalizes ALL engine/rule opinion (which engines, which rules, layout,
converter, provisioning) into the pack as data, reducing the backstop CLI to a
**thin deterministic harness**: discover declarations → gather inputs by mode → run
command → convert → parseSarif → normalize. **Precision (do not overclaim):** it is
thin BUT real — backstop still owns the execution core
(gather/run/convert/parse-SARIF/normalize/scope/provision/fail-loud) and the one
parser; the *opinion* is externalized, the *execution discipline* stays in the CLI
because that's the part the LLM gets no veto over ([[project_prompts_are_vibes]]).
**"Thin on knowledge, firm on enforcement."**

## Spec Seeds

Suggested decomposition into specs, in implementation order. No requirement
belongs to two seeds.

**Seed 1 — Pillar 2: packs-only core (single-source foundation).** Delete the
`.standard.md → standards-compiler → manifestDir` arm; dogfood-consume the
existing `backstop-go-pack`; drop `STD-GO-001`; retire the standards compiler from
core. Runs on the current semgrep engine, independent of ast-grep, so it lands
first and collapses locus A's input to a single source. Covers REQ-016. Sized
small (DIR-005 already published the pack).

**Seed 2 — Pillar 1: first-class `engine` + gate-time dispatch + ast-grep.** The
core engine machinery: add the `engine` field, retire `layer` into engine-keyed
field-contracts, build the `engine → {command, convert?, requires[], forbids[]}`
table, place the shared `EngineBinding` to avoid an import cycle, replace the
semgrep-only gate executor with group-by-engine dispatch normalizing to SARIF, add
the sandboxed `convert` pipe, fix the runner's stdout/stderr capture, and wire
ast-grep as the first new engine with a trivial proof rule end-to-end. Flag-day
migrate `backstop-go-pack`'s 14 rules to `engine: semgrep`. Also realizes the
engine-shape-agnostic / Layer-0-first model (REQ-018) and the split engine
provisioning story — Layer-0 assume-present, backstop-introduced engines
auto-provisioned via `backstop.lock`/`VerifyLock`, retiring `EnsureSemgrep`
(REQ-019). The gather/invoke seam is the structured `input_mode` enum + `input_flag`
(REQ-020), with the engine-organized pack layout (REQ-021) it implies. Covers
REQ-001…REQ-011, REQ-013, REQ-014, REQ-015, REQ-018, REQ-019, REQ-020, REQ-021.
(OQ-4 is now resolved — DD-9; this seam carries it.)

**Seed 3 — Pillar 1: fixture-time execution (`pkg/packval`).** Apply the same
engine generalization to the `pack validate` path. Covers REQ-012. Depends on
Seed 2's shared `EngineBinding`.

**Seed 4 — BUNDLE-009 seam (contract lock, not implementation).** Pin the
contract: this bundle ships engine machinery + ast-grep + the trivial proof rule;
BUNDLE-009 authors the real substantiveness/contract rule packs on top. Covers
REQ-017. This is a boundary-definition deliverable, not engine code.

## Notes / Ideas

- **The findings-normalization seam is shared with ISSUE-003.** semgrep,
  ast-grep, golangci-lint each emit a different structured output; the gate needs
  one `[]Violation`. ISSUE-003 already built a named-format parser library for the
  code-check toolchain registry. If engines reuse it (engine declares a
  findings-format name), the "declared engine" end-state (DD-2) gets much cheaper —
  adding an engine = invocation template + an existing format name. Worth checking
  how much of that library is reusable before committing the gather/invoke seam.
- **Don't churn working code.** semgrep and golangci-lint paths are tested and on
  main. The engine abstraction should *absorb* them (semgrep becomes the first
  registered engine, golangci-lint the second), not rewrite their behavior. The
  proof that the abstraction is right is that ast-grep slots in beside them without
  touching their arms.
- **ast-grep invocation specifics to pin down during spec:** version (spike used
  0.43.0), per-language rule files vs a scan rule-dir, exit-code + `file:line`
  contract (spike confirmed exit 1 + `file:line` + message works as a gate step).
  These are spec-level, not bundle-level.
- **Convergence, not invention (the load-bearing reframe).** The engine substrate
  already exists: ISSUE-003's `ToolchainEntry{Command, Format}` + `formatParsers`
  (incl. `sarif`) + `lookupParser` fail-loud + the generic data-driven executor.
  This bundle *converges* the three bespoke holdouts onto it (native
  `semgrepExecutor`, pack `mergePackRules`, pack `RunToolConfig`) and adds the
  `convert` step + rule-file-injection seam. Net scope is smaller and far less
  speculative than the original framing.

## Open Questions

All open questions are resolved (OQ-1…OQ-6). OQ-4 (the engine's gather/invoke
rule-file-injection seam) was the last; its resolution is now at **DD-9** (structured
declared `input_mode`) and **DD-10** (engine-organized pack layout), yielding REQ-020
and REQ-021. With all OQs resolved, design decisions made, requirements drafted,
and spec seeds identified, the bundle was promoted to `defined` maturity (2026-06-16).

## Spike — engine / output-format surface area (2026-06-14)

Goal: before mandating SARIF, verify empirically how many engines and output
formats backstop would realistically face across Python, Go, Rust, Java, Kotlin,
Ruby, Swift, C#, C/C++, and niche langs — i.e. is the "backstop owns formats, not
engines" inversion real. Method: parallel web research across four language
clusters + the SARIF ecosystem.

Findings:
- **Engines are many (~25+ and growing); formats are few.** Across the language
  set there are 25+ dominant linters, but **cross-language meta-engines collapse
  most of it** — semgrep (30+ langs, native SARIF), CodeQL (10, native SARIF),
  ast-grep (20+ langs) — so ~3 tools cover the bulk of a polyglot repo.
- **SARIF is the lingua franca, driven by GitHub code scanning.** Native SARIF in
  semgrep, CodeQL, golangci-lint v2 (2025), staticcheck, ruff, bandit, brakeman,
  Checkstyle, PMD, SpotBugs, ktlint, detekt, SwiftLint, cppcheck, clj-kondo,
  Trivy, Snyk, Roslyn (via ErrorLog). **≈2 formats (SARIF + a generic JSON path)
  cover the mainstream; ~4 cover nearly everything.** This empirically validates
  owning a tiny format-parser set instead of an engine list.
- **The unparseable tail is tiny:** Error Prone (javac text), swift-format (stderr
  text), sorbet (LSP-only) emit no machine format — excluded, not first-classed.
- **The decisive gotcha: ast-grep — our first engine — is JSON-only, no SARIF.**
  So a SARIF-only mandate needs the `convert` mechanism for ast-grep specifically;
  it can't ride native SARIF. This is what drove the convert-step design and the
  "ast-grep is the lone gap we own a converter for" decision.
- **JSON is not one schema.** eslint/golangci/ast-grep JSON all differ — which is
  why "just emit JSON" doesn't bound backstop and why the contract is strict SARIF
  + declared conversion, not "SARIF or JSON."

Conclusion: the open-engine / closed-format inversion is real and cheap — backstop
owns `parseSarif` and a sandboxed `convert` seam; the ecosystem (and existing
MS/GitHub converters) maps everything else into SARIF. See DD-2 (resolved).

## Out of Scope / Non-Goals

Enumerated 2026-06-16 so the spec cannot quietly absorb adjacent work.

**Belongs to another bundle:**
1. The real substantiveness/contract **rule packs** + wiring them into the gate's
   traceability steps → **BUNDLE-009**. This bundle ships only a *trivial proof
   rule* (OQ-6 seam).
2. BUNDLE-009's **fail-loud seed** + coverage-step work → BUNDLE-009 (not blocked
   by this bundle).
3. Migrating the baked-in Go **`go/parser` substantiveness analyzer**
   (step_testverify.go) onto the pack model → separate anomaly, later/BUNDLE-009.
   Distinct from pillar-2's standards-compiler removal (different code; both are
   "stop baking rules into the binary").
4. **`backstop init` toolchain inference** → its own artifact (noted in BUNDLE-009).

**Deferred — no consumer yet:**
5. Converting **build/test passes to SARIF** — explicit non-goal; they aren't
   findings, they keep their parsers. Only *lint* parsers (`golangci-json`,
   `eslint-json`) retire as migration debt.

**Rejected / future-scope:**
6. A backstop-owned **x-to-sarif converter repo** — point to existing MS/GitHub
   converters instead (see Notes).
7. **Engine *guidance*** (content analysis nudging declarative-vs-custom) — future
   capability (see Notes).
8. **Deprecation-window / ecosystem-migration tooling** — YAGNI at N=1 (OQ-5).
9. The **pack frontier** (publishing proxy, LLM reviewer, supply-chain scanning) —
   deferred until external pack authors ([[project_pack_engine_model]]).

**Confirmed non-goal, tracked as a known vision-gap:**
10. **Cross-platform sandbox.** `SandboxedRun` = macOS `sandbox-exec`, so Layer-3
    custom scripts AND the `convert` step **only run on macOS today**. Cross-platform
    sandboxing (Linux seccomp/landlock/containers) is **substantial separate future
    work** that the "plug into any project" promise will eventually force — it is
    **out of scope for THIS bundle** and is **no longer a pending decision**. Recorded
    here as a **known gap** that bounds where Layer-3 + `convert` can run, so the spec
    does not silently assume cross-platform coverage.

Note: engine *provisioning* moved OUT of this section — Layer-0 assume-present +
auto-provisioned backstop-introduced engines is now a committed requirement
(**REQ-019 / DD-8**), not a boundary call. The old "generalizing the layer-1 linter
family" non-goal was also removed: it inverted the ladder priority (DD-8) and is now
a first-class goal under **REQ-018** (Layer-0 / native breadth is core).

## References

- [[project_pack_engine_model]] — the design gap this bundle closes (engine is
  implicit-via-layer; should be first-class/pluggable; semgrep wired, ast-grep
  next). This bundle IS the "next pack bundle" that memory names.
- [[feedback_dogfood_rules_as_packs]] — no baked-in shortcut; backstop consumes
  its own rules as a pack on this engine; the Go go/parser analyzer is the
  anomaly to migrate, not the template.
- [[feedback_loud_not_blocking]] — fail-loud semantics for unknown/misdeclared
  engines (broken declaration → loud AND blocks; exit 2 / config error precedent
  from ISSUE-003/008).
- BUNDLE-009 (stack-aware-traceability) — the **dependent**; paused, carries a
  dependency edge on this bundle. Its query-pack layer (OQ-2/OQ-3 there) is gated
  on ast-grep wired as a pack engine. Spike (2026-06-14) de-risked the consumer
  side.
- Code: pkg/pack/manifest.go:38 (`Rule`, no engine field), :324 (`validateLayer`
  1/2/3); cmd/backstop/pack_gate.go:97 (`mergePackRules`, layer==2→semgrep);
  cmd/backstop/gate.go:246 + code_check.go:83 (`extraSemgrepConfigs` consumers);
  pkg/packval/executor.go:25,32 (`RunSemgrep` / `RunToolConfig` one-arm switch).
- ISSUE-003 (data-driven toolchain registry) — precedent for declaring
  command+format per stack and its named-format parser library (candidate reuse
  for engine findings normalization, DD-2).

## Version History

- **0.1.0 (2026-06-14, exploring)** — Initial bundle: problem framing (two
  hardcodings: gate-time `mergePackRules`, fixture-time `RunToolConfig`), user
  story, current thinking, six open questions (OQ-1…OQ-6), Notes, the 2026-06-14
  engine/output-format spike, Out of Scope / Non-Goals, and References. Through the
  2026-06-14/16 working session, OQ-1, OQ-2, OQ-3, OQ-5, and OQ-6 were resolved and
  the parse half of OQ-4; the file was restructured into canonical form
  (Draft Requirements + `requirements:` REQ-001…REQ-017, Draft Design Decisions
  DD-1…DD-7, Spec Seeds 1–4) without promoting maturity. The sole remaining open
  question is OQ-4's gather/invoke (rule-file-injection) seam; maturity stays
  `exploring` until it resolves.
- **(2026-06-16, exploring — escalation-ladder revision)** — Added the
  **preference-ordered escalation ladder** as the bundle's organizing principle:
  Layer 0 native toolchain (config-driven linters, first/best) → Layer 1 query
  engine + existing rules → Layer 2 custom rules → Layer 3 custom script, every
  layer an engine over the one SARIF substrate, ordered by "lowest layer that
  suffices" (woven into Current Thinking; new **DD-8**). Added **REQ-018**
  (engine-shape-agnostic, Layer-0 first-class; native/language breadth is core, not
  a deferred frontier) and **REQ-019** (split provisioning: Layer-0 assume-present
  fail-loud, backstop-introduced engines auto-provisioned via
  `backstop.lock`/`VerifyLock`, `EnsureSemgrep` retired). Constrained **OQ-4**
  (kept OPEN) so its resolution must treat config-driven linters as the primary
  invocation shape. Reworked Non-Goals: removed the "generalize the layer-1 linter
  family" deferral (it inverted the ladder; now covered by REQ-018), removed the
  engine-provisioning boundary item (now REQ-019), and reworded the cross-platform
  sandbox item from a pending decision into a confirmed non-goal + tracked
  vision-gap; the "boundary calls" subsection dissolved. No maturity promotion;
  version held at 0.1.0 per direction.
- **(2026-06-16, exploring — OQ-4 resolution / gather-invoke seam)** — Resolved the
  last open question, OQ-4, as **DD-9**: the rule-file-injection seam is a structured,
  declared `input_mode` enum (`config-file` primary / `rule-flags` / `rule-dir` /
  `none`) + `input_flag` on the engine binding, extending ISSUE-003's
  `ToolchainEntry`/`ScopeKind` pattern — not a per-engine Go interface, not a
  template DSL. Added **DD-10** (engine-organized pack layout convention + the "thin
  on knowledge, firm on enforcement" thin-executor thesis, also woven into Current
  Thinking). Added **REQ-020** (structured `input_mode` injection) and **REQ-021**
  (per-engine pack layout). Updated DD-4 and Spec Seed 2 to reflect OQ-4 resolved.
  Emptied the Open Questions section to a resolved-note (all OQ-1…OQ-6 resolved).
  Removed the now-stale `requirements_note`. No maturity promotion; version held at
  0.1.0 per direction (promotion is the human's separate call).
- **(2026-06-16, defined — maturity promotion)** — Promoted `exploring → defined`.
  All open questions (OQ-1…OQ-6) resolved, design decisions (DD-1…DD-10) made,
  requirements drafted (REQ-001…REQ-021), and spec seeds (1–4) identified — the
  `defined`-maturity gates are satisfied entirely by existing content (problem
  summary/user story, solution approach, the five required sections, and the
  non-empty `requirements:` array; `problem.summary` carries no placeholder text).
  No content was added or scoped to clear a gate; this is the maturity-progression
  step only. Version held at 0.1.0 (no `bundle.updated` required).
