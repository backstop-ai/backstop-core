---
title: "Pack Core Version Compatibility"
number: BUNDLE-020
created: "2026-07-25"
schema_version: bundle/v2

bundle:
  name: pack-core-version-compatibility
  version: "0.2.0"
  created: "2026-07-25"
  updated: "2026-07-26"
  category: infrastructure

status:
  maturity: exploring

problem:
  summary: >
    NOTHING declares or verifies which backstop core versions a pack is compatible with — the
    pack ↔ core boundary is completely unguarded. Verified 2026-07-25: `pkg/pack.Manifest`
    carries `name / normalized name / version / language / archetype / description / content /
    tool_config / engines / classification / recipes` (plus `content.source`, `content.test`,
    `test_name_patterns`) and NOTHING else; a grep for
    `backstop_version|min_backstop|requires_backstop|compatible_version|core_version|min_core`
    across `pkg/pack/`, `artifacts/`, and `packs/` returns ZERO hits. No declaration, no
    enforcement, nowhere — not in the manifest, not in `backstop.lock` (`LockEntry` is
    `{name, version, git_ref, content_hash, source_type, install_date, local_path}`), not at
    `pack add` / `install` / `update` / `upgrade` / `relock` time, and not in the
    `pack_lock_verification` gate step. ISSUE-062's real shape was NOT "a stale binary" — it was
    the core binary's message-PARSING disagreeing with a pack's message FORMAT across a version
    boundary: pack rule messages dropped `func=` / `symbol=`, the core's older parser read empty
    func/symbol, and the gate raised 3 PHANTOM `noTarget` violations. Locally that boundary is
    binary-vs-source (ISSUE-077 — a contributor problem that evaporates once consumers install
    release binaries). Post-release the SAME CLASS survives with different endpoints: a consumer
    on core v1.2 running a pack authored against v1.3. Identical symptom — phantom violations,
    or WORSE, silently MISSING violations. "Rebuild your binary" is no longer the answer. Silent
    or vacuous green is the enemy this project exists to defeat, and an unguarded version
    boundary is a direct route to it.

  user_story: >
    As a consumer running backstop core at some released version against packs authored and
    versioned independently (and as the pack author shipping fixes through the
    fix → bump → relock flywheel to a fleet of consumers I do not control), I want the pack ↔ core
    compatibility boundary to be DECLARED and CHECKED, so that a consumer whose core cannot
    correctly read a pack finds out LOUDLY at a defined moment — rather than discovering it as
    phantom violations they waive away, or never discovering it at all because the incompatibility
    silently SUPPRESSED findings and the gate went green. Concretely: the ISSUE-062 failure
    (an older parser reading a newer pack's message format and inventing 3 `noTarget` violations)
    must be impossible to hit silently once the endpoints are two independently-released versions
    instead of a binary and its own source tree.

solution:
  approach: >
    PARTIALLY DECIDED as of 2026-07-26 — OQ-1 and OQ-5 are resolved (DD-4..DD-7); OQ-2, OQ-3, OQ-4
    and OQ-6 remain genuinely open, so the mechanism is not yet a complete design. WHAT A PACK
    DECLARES is now settled (OQ-1 → DD-4): named CAPABILITY CONTRACTS scoped to the pack↔core WIRE
    SEAM — the pack declares the named contracts it depends on, core declares which it provides,
    and compatibility is a SET comparison rather than a version comparison. That resolution is
    CONDITIONAL (DD-4d): capability contracts are adopted only if the capability-name registry can
    be guarded by a dogfooded rule that goes RED when a capability name contains a language or tool
    name; if that rule proves unwritable, the decision REVERTS to a version-based scheme. Two
    carve-outs travel with it: the ~60-field pack.yml manifest is a SCHEMA problem answered by a
    `schema_version` field, not a capability problem (DD-5, overlapping OQ-4's still-open
    territory), and the `TrustedToolAllowlist` is a TRUST boundary, not a compatibility boundary,
    and is out of scope entirely (DD-6 — its own problems now live in ISSUE-082 and BUNDLE-021).
    Baseline identity churn is de-scoped to the baseline thread (OQ-5 → DD-7); this bundle owns
    only EMITTING THE SIGNAL that a declared format changed, which the baseline work consumes to
    TRANSLATE identities rather than re-key them. Still open: (2) WHERE it is checked — install/add,
    lock verification, gate preflight (OQ-2); (3) the failure POSTURE per case, adjudicated by the
    standing loud-≠-blocking principle rather than by uniform severity (OQ-3); (4) whether a
    symmetric declaration is needed so a NEWER core can reject or migrate an OLDER pack (OQ-4); and
    the retroactive default for the packs that declare nothing today (OQ-6). Two pieces of
    existing substrate are prior art the resolution weighed rather than reinvented:
    `backstop version --json` ALREADY emits a `schema_cohort` computed over the set of embedded
    schema versions (`cmd/backstop/root.go:102`) — an existing non-semver identity for "what this
    binary understands"; and `engine.FieldContract{Requires, Forbids}`
    (`pkg/pack/engine/fieldcontract.go`) is existing prior art for a pack DECLARING a contract in
    pack.yml that core validates, instead of a version number. Two hard grounding facts constrained
    the answer: core's own version is `var version = "dev"` set at build time via `-ldflags`
    (`cmd/backstop/root.go:16`), so a source/dev build has NO meaningful version to compare against
    — a hole DD-4 DISSOLVES rather than patches, since a capability SET is derived from what the
    binary actually contains and is therefore well-defined for a `dev` build; and core independently
    pins TOOL versions in `engine.TrustedToolAllowlist()` (semgrep 1.96.0, ast-grep 0.43.0, plus
    `"*"` presence pins), which DD-6 now excludes as a trust rather than compatibility boundary,
    collapsing the three-body framing back to the two bodies this bundle governs. Whatever lands
    must hold the thin-executor line: the check is a generic contract comparison over DECLARED data,
    never baked knowledge of any particular pack, language, or tool — and under DD-4 that line is
    load-bearing enough to be MECHANICALLY GUARDED, not merely asserted.
---

# Pack Core Version Compatibility

## Current Thinking

### The boundary is unguarded — verified, not assumed (2026-07-25)

`pkg/pack.Manifest` (`pkg/pack/manifest.go`) carries exactly: `name`, `normalized name`
(computed), `version`, `language`, `archetype`, `description`, `content` (`source` / `test`),
`tool_config`, `engines`, `classification` (incl. `test_name_patterns`), and `recipes`. There is
no field — and no adjacent file — in which a pack could state what core it was authored against.

A grep for `backstop_version|min_backstop|requires_backstop|compatible_version|core_version|min_core`
across `pkg/pack/`, `artifacts/`, and `packs/` returns **zero hits**. So the gap is total: nothing
DECLARES compatibility and therefore nothing can ENFORCE it. This is not "an enforcement we
haven't turned on yet"; there is no vocabulary for the statement at all.

The lock file does not close it either. `LockEntry` (`pkg/pack/distribution/lockfile.go`) is
`{name, version, git_ref, content_hash, source_type, install_date, local_path}` — it pins WHICH
pack content is installed, with a `content_hash` that proves the content is untampered, but says
nothing about which core can READ that content. A perfectly valid, hash-verified,
tamper-free lock entry can describe a pack the running core cannot correctly interpret.

### What ISSUE-062 actually was — a version-boundary defect wearing a stale-binary costume

The originating incident is worth restating precisely, because its real shape is what makes this
bundle a post-release concern rather than a contributor annoyance.

The substantiveness pack emitted findings whose MESSAGE carried the machine payload:
`"referenced-symbol func=$FN symbol=$PKG"`. Core parsed that payload back out of the prose with
`tokenValue`. ISSUE-062 moved the payload to a structured channel — SARIF result `properties`,
read as `v.Properties["func"]` / `v.Properties["symbol"]`
(`pkg/gate/substantiveness_join.go:136,139,158`) — and DELETED the prose parsers. Correct fix,
delivered.

But look at the failure it produced while the two halves were out of step: the pack's rule messages
no longer carried `func=` / `symbol=`, the older parser read them as EMPTY, and the join raised
**3 phantom `noTarget` violations** — findings about code that was not defective. The defect was
not in either half. Each half was internally correct. The defect lived in the DISAGREEMENT between
a pack's output format and a core's input parsing, across a version boundary.

Note the residue: the property KEY NAMES (`func`, `symbol`) are now an **implicit, unversioned
wire contract** between pack and core. The structured channel removed the whitespace-parsing
fragility; it did not version the contract. Rename a key on either side and the same class recurs.

### Why it survives the release — the endpoints change, the class does not

Locally the two endpoints are a built binary and the source tree it drifted from. That is
ISSUE-077's territory (filed 2026-07-25), and it is a CONTRIBUTOR problem that largely evaporates
once consumers install release binaries rather than building from source.

Post-release the same class re-instantiates with different endpoints:

- Consumer on core **v1.2**, pack authored against core **v1.3** → the older core mis-reads a newer
  format. Symptom: phantom violations, OR — the worse and quieter half — findings silently
  **DROPPED**, because a core that reads a field it doesn't understand as empty tends to emit
  nothing rather than emit garbage. **Vacuous green.**
- Consumer on core **v1.4**, pack authored against **v1.1** → a newer core applies newer semantics
  to an older declaration. Same failure family, opposite direction (this is the OQ-4 question).

"Rebuild your binary" is the ISSUE-077 answer — and its concrete form makes the non-generalization
plain: a PATH shim that rebuilds-if-stale then execs, plus a gate-startup mtime self-check that
exits 2. Both mechanisms compare a binary against THE SOURCE TREE IT WAS BUILT FROM. Neither has
any analogue here: the consumer's core and the pack are two independently-released artifacts on
two independent cadences, in two different repos, owned by potentially two different parties.
There is no shared tree to stat, and nothing to rebuild.

### It is load-bearing for the fix → bump → relock flywheel

The pack model's core promise is that a pack fix is versioned and propagates to every consumer:
find a false positive, fix the pack, bump it, consumers `pack update` / relock. That flywheel is
how "the check is wrong" gets answered by improving the PACK rather than scarring the code.

An unguarded version boundary puts a silent hazard directly in that path. Every pack bump is a
potential compatibility break for some consumer's core, and today the flywheel has no way to
say so. A pack fix cannot safely propagate to all consumers if nothing checks that a consumer's
core can actually read the new pack. The more the flywheel is used, the more often the hazard is
rolled.

### It looked like a THREE-body problem — SUPERSEDED by DD-6 (2026-07-26)

> **Superseded.** The framing below was the v0.1.0 reading. DD-6 excludes the tool axis from this
> bundle: `TrustedToolAllowlist` is a TRUST boundary, not a compatibility boundary, and its own
> problems now live in ISSUE-082 and BUNDLE-021. This bundle governs TWO bodies — pack ↔ core. The
> original reasoning is kept because it is what forced the question of whether a core-semver
> declaration under-describes the constraint, and that pressure is part of why OQ-1 resolved to
> capability contracts rather than a version.

Core does not only parse pack output — it also owns the **tool** allowlist.
`engine.TrustedToolAllowlist()` (`pkg/pack/engine/allowlist.go`) pins `semgrep 1.96.0`,
`ast-grep 0.43.0`, and presence-only `"*"` pins for Layer-0 tools (grep, rg, the Bun toolchain
tools). Two consequences worth carrying into the OQs:

- A pack authored against ast-grep 0.43.0's output shape, run by a core pinning a different
  version, has an output-format skew that lives in the TOOL, not in either the pack or core.
- A pack declaring a tool an OLDER core's allowlist does not contain is rejected outright by that
  core's trust floor — a compatibility failure with a completely different symptom (hard refusal,
  not silent drift).

So "which core can run this pack" is not purely a parser question. Any declaration that only
captures core semver may be under-describing the constraint — which is exactly why OQ-1's
capability/contract option is live rather than academic.

### The two grounding facts every candidate answer has to survive

1. **Core's version is `"dev"` unless a release build injected it.** `cmd/backstop/root.go:16`:
   `var version = "dev"` — "set at build time via -ldflags." A source/dev build has NO meaningful
   version to compare a pack declaration against. Any version-comparison design needs a defined
   behavior for `dev` (permissive? warn? treat as newest?) or it will either block every
   contributor or silently exempt the exact population most likely to be skewed.
2. **There is already a non-semver identity for "what this binary understands."**
   `backstop version --json` emits `schema_cohort`, computed by `computeCohortID(schemas)` over
   the set of embedded schema versions (`cmd/backstop/root.go:98-109`). That is a real, shipping
   precedent for identifying a binary's capability surface by CONTENT rather than by release
   number — directly relevant to OQ-1, and it already answers the `dev`-build problem for schemas
   (a dev build has a perfectly well-defined cohort even with no version).

And one more piece of prior art: `engine.FieldContract{Requires []string, Forbids []string}`
(`pkg/pack/engine/fieldcontract.go`, declared per-binding in pack.yml's `engines:` block) is an
existing example of a pack DECLARING a contract that core validates — not a version number, a
named set of field requirements. Whatever OQ-1 resolves to should be measured against these two
in-repo precedents before inventing a third idiom.

**Outcome (2026-07-26):** it was — DD-4 adopts the capability/contract shape on the strength of
exactly these two precedents, making it a third application rather than a third idiom. Fact 1
stops being a hole in the process: a capability set is derived from binary content, so a `dev`
build has a well-defined one and needs no special case.

### The wire seam, ENUMERATED — verified 2026-07-26

OQ-1's resolution turned on a question the founder refused to let be answered by assertion: *how
big is the surface a capability contract would actually have to cover?* "Capability contracts" is
only a credible answer if the seam is small and enumerable; if it is sprawling and open-ended, the
registry becomes a second manifest schema and the option collapses. So the seam was enumerated
against the code rather than estimated. Every line reference below was re-verified 2026-07-26.

**Wire formats — data core PARSES from pack output:**

| Contract | Location | Shape |
|---|---|---|
| SARIF result subset | `pkg/check/parsers.go:54-95` (`sarifLog`) | `ruleId`, `level`, `message.text`, `locations[].physicalLocation.{artifactLocation.uri, region.startLine, region.snippet.text}`, `partialFingerprints`, `suppressions[].kind`, `properties` |
| SARIF `properties` keys | `pkg/gate/substantiveness_join.go:136,139,158` | `func`, `symbol` — compared VERBATIM against `MandatedTest.FuncName` — plus `substantiveness_role` (the routing key, `substantiveness_join.go:96,111`) |
| Coverage record | `pkg/check/coverage.go:25` (`CoverageRecord`) | `{path, covered, total, measured, excluded, metric}` |

The `properties` row is not hypothetical: it is literally where ISSUE-062 broke. `func` and
`symbol` are read by exact key with no version on the contract — the surviving residue this bundle
has flagged since v0.1.0.

**Declaration vocabularies — enums core DEFINES and packs SELECT from:**

| Vocabulary | Location | Values |
|---|---|---|
| `InputMode` | `pkg/pack/engine/binding.go:20-32` | `config-file`, `rule-flags`, `pattern-arg`, `none` |
| `GateType` | `pkg/pack/engine/gatetype.go:38-44` | `lint`, `build`, `test`, `findings`, `coverage`, `substantiveness`, `contracts` |
| FieldContract field names | `pkg/pack/engine/fieldcontract.go` | `rule_path`, `standard`, `category`, `input_scope`, `validator` |

**Result: six contracts, sixteen enum values.** That is a registry a human can hold in their head
and a rule can scan — small enough that the capability option is real work rather than a second
schema in disguise.

**The contamination check came back CLEAN.** Nothing in that set names a language or a tool.
`substantiveness` and `contracts` read like they might be exceptions and are not — they are
BACKSTOP concepts (a gate dimension and an artifact-claim dimension), and they do not change when a
new language appears. The one place tool names genuinely live in core is
`engine.TrustedToolAllowlist()` (`pkg/pack/engine/allowlist.go:17` — `semgrep 1.96.0`,
`ast-grep 0.43.0`, plus `"*"` presence pins for grep/rg/oxlint/bun/tsc/prettier), which is exactly
why DD-6 excludes it from this bundle. The allowlist's own in-code comment already draws the same
line: *"The allowlist KEY is a tool-name lookup datum (the trust floor that gates which
pack-declared commands may run), NOT a baked routing/command literal — it never sources a
command."*

### What is decided vs open (updated 2026-07-26)

DECIDED: the standing invariants recorded as DD-1..DD-3, which constrain but pick no mechanism;
plus two founder resolutions taken 2026-07-26 — **OQ-1** (what a pack declares → DD-4 capability
contracts scoped to the wire seam, CONDITIONAL on a mechanical guard; DD-5 `schema_version` for the
manifest; DD-6 the tool allowlist is out of scope) and **OQ-5** (baseline identity churn belongs to
the baseline thread → DD-7; this bundle emits the signal). Full rationale for both is in *Resolved
Design Questions* below.

STILL OPEN: where it is enforced (OQ-2), the failure posture (OQ-3), whether the reverse direction
needs a declaration too (OQ-4 — which DD-5 now overlaps but does not answer), and the retroactive
default for the packs that exist today and declare nothing (OQ-6). Two of six resolved is not
promotable: maturity stays `exploring`, and promotion is the founder's call regardless.

## Draft Design Decisions

DD-1..DD-3 are INVARIANTS inherited from standing project law, recorded so every candidate answer
is measured against them; **none of them answers any open question** — DD-3 in particular supplies
the principle that ADJUDICATES OQ-3, not OQ-3's answer. DD-4..DD-7 are different in kind: they are
FOUNDER DECISIONS taken 2026-07-26, resolving OQ-1 (DD-4/DD-5/DD-6) and OQ-5 (DD-7).

- **DD-1: The check is generic over DECLARED data — zero baked pack/language/tool knowledge.**
  (Standing law: thin executor.) Whatever compatibility check lands must be a generic comparison
  over data a pack declares — a version, a range, or a set of named contract identifiers. It may
  NOT contain a table of known packs, a special case for any language, or hardcoded knowledge of
  which pack needs which core. `backstop/self` guards this boundary; a compatibility check that
  named a pack would be a new baked-knowledge defect introduced by the fix for an old one.

- **DD-2: Packs stay external; the declaration travels WITH the pack, and the lock is the durable
  record.** (Standing law: packs always external.) Every pack lives in its own repo and installs
  into gitignored `.backstop/packs/`; `backstop.lock` is the durability boundary. So the
  compatibility statement must be authored in pack-owned content (pack.yml or an adjacent
  pack-owned file) and, if it needs to survive a re-materialization, be recorded in the lock —
  never held in a core-side registry of external packs, which would recreate the vendoring model
  the pack architecture exists to avoid. (Whether it lands in the lock at all is part of OQ-2.)

- **DD-3: The posture is adjudicated by loud-≠-blocking, per case — not set uniformly.**
  (Standing law: loud ≠ blocking; the enemy is silent/vacuous green, not passing.) An
  incompatibility that causes WRONG findings — phantom violations, or silently dropped ones — is a
  defect and a broken promise; that is block-shaped. An incompatibility that merely means a newer
  pack capability goes UNUSED on an older core is un-adopted capability; that is warn-with-guidance
  shaped. This DD fixes the PRINCIPLE and forbids a uniform answer; sorting which concrete cases
  fall on which side is OQ-3 and is unresolved.

---

- **DD-4: A pack declares named CAPABILITY CONTRACTS, scoped to the WIRE SEAM — CONDITIONAL on a
  mechanical guard.** (Founder decision, 2026-07-26; resolves OQ-1 option (c).) A pack declares the
  named contracts it DEPENDS ON; core declares which it PROVIDES; compatibility is a **set
  comparison**, not a version comparison. The declaration covers exactly the surface enumerated in
  *The wire seam, ENUMERATED* above — six contracts and sixteen enum values — and no more.

  *Why over (a) min-version and (b) semver range:* both proxy the real constraint instead of
  stating it. What actually breaks is a named contract (`Properties["func"]` disappearing), not a
  release number; a version is only ever a stand-in for the contract set, and a stand-in that
  couples every pack to core's release cadence and forces pack authors to predict future breakage.
  A set comparison decouples packs from the cadence and degrades gracefully — core can add
  contracts freely, and a pack only breaks when a contract it NAMED goes away.

  *Why the seam is the right scope:* the enumeration is what made this credible. Six contracts is a
  registry, not a second schema. Had the seam sprawled, (c) would have collapsed into re-describing
  the manifest and the decision would have gone the other way.

  *Two in-repo precedents made it credible rather than novel:* `backstop version --json` already
  emits `schema_cohort` (`cmd/backstop/root.go:102`), an identity for "what this binary
  understands" computed from CONTENT rather than a release number — which is why OQ-1's `dev`-build
  sub-question DISSOLVES under this option rather than needing a special case, since a capability
  set comes from what the binary actually contains and a `dev` build has a perfectly well-defined
  one. And `engine.FieldContract{Requires, Forbids}` (`pkg/pack/engine/fieldcontract.go`) already
  has packs declaring named requirements that core validates. This is a third application of an
  existing idiom, not a third idiom.

  **This decision is conditional. See DD-4d — the condition is load-bearing, not a nice-to-have.**

- **DD-4d: THE CONDITION — capability contracts are adopted ONLY IF the registry can be guarded by
  a dogfooded rule that goes RED when a capability name contains a language or tool name. If that
  rule proves unwritable, THE DECISION REVERTS TO A VERSION-BASED SCHEME.** (Founder decision,
  2026-07-26.) This is not a follow-on task or a quality bar to aspire to; it is a precondition on
  DD-4 itself.

  *The founder's concern, which drove it:* **a capability registry is a plausible back door for
  re-importing language-specific knowledge into core** — precisely what the standing zero-baked
  invariant forbids. The registry is a list of blessed names that core owns and packs must match.
  Nothing about that structure resists a name like `semgrep.rule_flags`, and `semgrep.rule_flags`
  is the most natural thing in the world to type when the contract you are describing is in fact
  how semgrep takes rules. The back door does not look like a violation while you are walking
  through it.

  *The legality test is MECHANICAL:* **would this capability name change if you added a new
  language?** `sarif.properties.func_symbol` does not change when Rust appears — legal.
  `eslint.config_shape` would — illegal. `semgrep.rule_flags` would — illegal, and again, the most
  natural thing to type.

  *Why a convention is not sufficient:* a convention written in a document is not protection. This
  is the standing **"prompts are vibes; only executed code constrains"** principle applied to the
  thing DD-1 asserts — DD-1 says the check must be generic over declared data, and DD-4d is what
  makes DD-1 mechanically true for this specific mechanism rather than merely stated.

  *Honest accounting of the guard's cost — the existing self-pack families would NOT catch this.*
  Verified against `~/src/projects/backstop-self-pack/rules/no-baked.yml`, 2026-07-26:
  - **Family A** (`no-baked-tool-exec`) matches a tool name as a string literal passed to
    `exec.Command` / `exec.CommandContext`. A capability constant is not a call argument.
  - **Family B1** (`no-baked-tool-command`) matches `Command: "$CMD"` where `$CMD` starts with a
    known tool invocation. A capability constant is not a `Command:` field.
  - **Family B2** (`no-baked-language-token`) is a `pattern-regex` over quoted tokens
    (`.ts`/`tsc`/`eslint`/`go.mod`/`package.json`/…). It requires the token to be the WHOLE quoted
    string, so `"eslint.config_shape"` slips past on the trailing suffix alone — and `semgrep` is
    not in the token list at all, so `"semgrep.rule_flags"` never even gets close.

  So a capability CONSTANT naming a tool passes all three. **The guard is NEW WORK, not an existing
  rule pointed at a new target** — which is exactly why it is recorded as a condition rather than
  assumed. The nearest neighbor, and the reason it is judged writable rather than speculative, is
  Family B6 (`no-pack-name-keyed-capability`), which already reaches INTO capability logic — but it
  catches capability keyed on a pack-NAME coordinate (`cfg.Packs["backstop/contracts"]`, a
  `-toolchain` suffix test), not a language/tool name inside a capability IDENTIFIER. The new rule
  is B6's sibling one axis over. Whether it can actually be written to the same standard is the
  condition's open half.

- **DD-5: The pack.yml MANIFEST is a SCHEMA problem, answered by `schema_version` — not a
  capability problem.** (Founder decision, 2026-07-26; the second half of OQ-1's resolution.) The
  ~60-field manifest schema is NOT part of the capability surface. It is a schema, and the
  mechanism for a schema is a **version field** — matching the pattern core already uses for
  artifacts, where every artifact carries `schema_version` and core resolves the matching
  `artifacts/<type>/v<N>/schema.json`. Splitting the two mechanisms keeps each one honest: a
  capability SET describes a wire seam that evolves by addition, a schema VERSION describes a
  document shape that evolves by revision, and forcing either into the other's mechanism distorts
  both.

  **`pack.yml` has no `schema_version` today** — verified 2026-07-26; the only `schema_version`
  hits under `pkg/pack/` are the retired standards emitter and a scaffold test, none of them on the
  manifest. So this is a field that must be ADDED, not one to start populating.

  **This overlaps OQ-4's territory and does not resolve it.** OQ-4 asks whether CORE should declare
  a pack-format version so a newer core can reject-or-migrate an older pack. DD-5 supplies the
  vocabulary that question would use, but it does not answer whether the symmetry is needed, what a
  newer core does with an out-of-range pack (reject vs migrate), or the `content_hash`/tamper
  interaction that in-place migration would disturb. **OQ-4 remains OPEN.**

- **DD-6: The TOOL ALLOWLIST is OUT OF SCOPE — it is a trust boundary, not a compatibility
  boundary.** (Founder decision, 2026-07-26; the third half of OQ-1's resolution.)
  `engine.TrustedToolAllowlist()` (`pkg/pack/engine/allowlist.go:17`) governs WHICH pack-declared
  commands are permitted to run at all. That is a trust question. Whether core can correctly READ a
  pack's output is a compatibility question. **Conflating them makes both harder to reason about** —
  a single "compatibility" mechanism spanning both would have to answer "may this run?" and "can I
  read this?" with one comparison, and the two have different failure modes, different postures,
  and different owners.

  This also retires the THREE-body framing (pack ↔ core ↔ tool) that v0.1.0 carried: with the tool
  axis excluded, this bundle governs two bodies.

  The allowlist's own problems are real and are now tracked SEPARATELY:
  **ISSUE-082** (Tool Allowlist Unreachable Entries, filed 2026-07-26) and **BUNDLE-021**
  (Pack Command Execution Governance, `exploring`, created 2026-07-26 — what governs arbitrary
  pack-declared commands). Neither is a dependency of this bundle; they are the correct home for
  the concerns DD-6 declines.

- **DD-7: Baseline identity churn belongs to the BASELINE thread; this bundle EMITS THE SIGNAL.**
  (Founder decision, 2026-07-26; resolves OQ-5 toward the hand-over option.) Identity stability is
  de-scoped from BUNDLE-020.

  *Why:* it already has an OWNER and a PRECEDENT. ISSUE-062 deliberately kept `Properties` OUT of
  baseline identity for exactly this reason — evidence the concern is already understood and
  already being managed elsewhere. Owning it here would DUPLICATE a concern that has a home, and
  would widen this bundle substantially — from "declare and check a wire seam" to "declare and
  check a wire seam AND own finding-identity migration."

  *What this bundle DOES own:* **emitting the signal.** The compatibility declaration is the only
  thing in the system that knows a declared format CHANGED. That signal is the input the baseline
  work consumes in order to **translate** identities rather than **re-key** them. Without it the
  baseline thread has no way to distinguish "this finding is new" from "this finding was renamed
  by an upgrade," so the signal is a genuine deliverable here even though the remediation is not.

  *The problem being handed over — recorded so it is not lost in the transfer.* Today identity is
  `Rule|File|RegionHash(Message|Severity|SourcePack)` (`EnrichViolationIdentity`,
  `pkg/gate/baseline.go:203-224`; the parenthesized triple is the FALLBACK hash used when the
  engine supplies no `partialFingerprints`/snippet region). So **message text is part of identity**,
  and a core upgrade that merely REWORDS a rule message re-keys every grandfathered finding in that
  rule. With the ratchet default-on, that surfaces as a **wall of "new" violations on a
  zero-code-change upgrade** — which creates direct pressure toward mass waiving, the precise
  outcome the waivers-are-last-resort rule exists to prevent.

## Open Questions

Six genuine forks were raised at v0.1.0 with no leans recorded. **Two are now resolved by founder
decision (2026-07-26); four remain open.**

- OQ-1 What does a pack declare? — **RESOLVED** (DD-4 capability contracts scoped to the wire seam,
  CONDITIONAL on DD-4d; DD-5 `schema_version` for the manifest; DD-6 tool allowlist out of scope)
- OQ-2 Where is it enforced? — **OPEN**
- OQ-3 What is the failure posture? — **OPEN**
- OQ-4 Does the reverse direction need a declaration? — **OPEN** (DD-5 overlaps its territory and
  supplies its vocabulary; it does not answer it)
- OQ-5 Baseline / ratchet interaction — **RESOLVED-via-handover** (DD-7: the baseline thread owns
  it; this bundle emits the signal)
- OQ-6 Retroactive default for undeclared packs — **OPEN**

### Still open

- **OQ-2 — WHERE IS IT ENFORCED?** Candidate moments, not mutually exclusive: **install/add time**
  (`pkg/pack/distribution/add.go` / `install.go` / `update.go` / `upgrade.go`) — earliest and
  cheapest feedback, refuses to install a pack this core can't read; **lock verification** — the
  `pack_lock_verification` gate step (`cmd/backstop/gate.go:771-786`) already runs on every gate
  invocation and already has a hard `ConfigErr: true` posture, and `pkg/gate/policy.go:49` already
  treats it specially, so it is the natural home for a per-run re-check; **gate preflight** — a
  distinct earlier step; and **`pack check` / `pkg/packval`** — validating a pack's own declaration
  is well-formed at authoring time, which is a different job from checking it against the running
  core. The load-bearing half of this question: install-time enforcement ALONE leaves the
  **core-upgraded-after-install** case wide open — the consumer installs a compatible pack, then
  upgrades (or downgrades) core, and nothing re-checks. That case is arguably the MOST likely
  real-world instance, since core upgrades independently of any pack action. Does that force a
  per-run check, and if so does install-time enforcement still earn its keep as fast feedback?

- **OQ-3 — WHAT IS THE FAILURE POSTURE?** DD-3 fixes the principle (loud ≠ blocking; block defects
  and broken promises, warn un-adopted capability) but not the sorting. Which concrete cases are
  which? Candidates for BLOCK: a pack requiring a core newer than the one running, where the
  documented consequence is wrong findings (the ISSUE-062 shape) — this produces phantom or, worse,
  silently missing violations, and a gate that goes green while a check is broken is precisely the
  vacuous green this project exists to defeat. Candidates for WARN: a pack declaring a capability
  the running core lacks where the degradation is a check simply not running AND that absence is
  reported loudly (un-adopted capability, the existing `capability_absent` posture). The hard part
  is that these two cases can be indistinguishable from the outside — "core can't read this pack's
  output" MIGHT degrade to no-findings (warn-shaped) or to wrong-findings (block-shaped), and the
  pack is the only party that knows which. Does the DECLARATION therefore have to carry the
  posture (the pack states "incompatibility here is a defect" vs "here it's degradation")? And
  where does the existing `ConfigErr` / exit-2 posture of `pack_lock_verification` fit — is a
  compatibility failure a CONFIG error (exit 2) or a VIOLATION (exit 1)?

- **OQ-4 — DOES THE REVERSE DIRECTION NEED A DECLARATION TOO?** Everything above guards
  pack-newer-than-core. The mirror case is core-newer-than-pack: a v1.4 core applying newer
  semantics to a pack authored against v1.1. Should CORE declare a **pack-format version** (a
  minimum pack-manifest / pack-format version it accepts), so a newer core can REJECT an
  unreadably-old pack, or better, MIGRATE it? Note this already half-exists in a different
  vocabulary: artifacts carry `schema_version` and core resolves per-version schemas
  (`artifacts/<type>/v<N>/schema.json`), which is exactly the "declare the format so the reader can
  route or reject" pattern — but pack.yml has no `schema_version` at all. **DD-5 (2026-07-26) now
  commits to ADDING that field**, on the grounds that the ~60-field manifest is a schema problem
  rather than a capability problem — so OQ-4 inherits the vocabulary but keeps the question. Is
  symmetry actually
  needed, or is the asymmetry justified because newer cores can be made to read older packs by
  construction (a compatibility obligation on core) while the reverse is impossible? If migration
  is on the table, does an old pack get REWRITTEN in place (a mutation to gitignored installed
  content, which the lock's `content_hash` would then flag as tampered — a real interaction with
  `pkg/pack/distribution/tamper.go`) or merely READ through a shim?

- **OQ-6 — RETROACTIVE DEFAULT: what does an UNDECLARED pack mean?** Every pack that exists today
  declares nothing — all of them, including the dogfood packs (`backstop/self`, the toolchain
  packs, contracts, substantiveness) and the shipped TypeScript suite in `backstop-packs`. So the
  default for "no declaration" is the single most consequential choice here, because on day one it
  applies to 100% of packs. Options: (a) **assume-compatible / silent** — zero friction, but it
  preserves the exact failure the bundle exists to prevent for every existing pack, and creates no
  pressure to ever declare; (b) **warn-until-declared** — surfaces the gap loudly, generates a
  visible adoption backlog, and matches the standing "un-adopted capability is warn-shaped" posture
  — but every consumer sees a warning on every gate run until the whole ecosystem has declared, and
  a warning everyone learns to ignore is worse than no warning; (c) **assume-compatible with a
  DEADLINE** — silent now, warn at a named version, block later; honest about migration but adds a
  schedule the project has to actually keep. Related: does the DOGFOOD population get declarations
  first as the forcing function (declare in the dogfood packs, prove the mechanism, then let the
  default tighten)? And is there a difference between an undeclared pack and one that declares
  "compatible with everything"? Note DD-4d adds a sequencing wrinkle: until the guard rule exists,
  the capability mechanism is provisional, so an OQ-6 deadline cannot be set against a decision
  that may still revert.

### Resolved (kept for the reasoning)

- **OQ-1 — WHAT DOES A PACK DECLARE? (RESOLVED 2026-07-26 → DD-4 + DD-4d + DD-5 + DD-6.)** The
  three options on the table were (a) a minimum core version (`min_backstop: "1.2.0"`), (b) a
  semver range (`backstop: ">=1.2 <2.0"`), and (c) named capability/contract identifiers.
  **Resolved to (c)**, with the surface scoped to the WIRE SEAM and a hard condition attached.

  The founder declined to accept "capability contracts" on assertion and required the seam be
  ENUMERATED first — the enumeration is recorded in *The wire seam, ENUMERATED* under Current
  Thinking, and it came back at **six contracts and sixteen enum values, with no language or tool
  name anywhere in the set**. That smallness is what made (c) a registry rather than a second
  manifest schema, and it is the fact the decision rests on. Two sub-questions raised at v0.1.0 are
  answered by the resolution rather than separately: the **`dev`-build hole dissolves** (a
  capability set is derived from binary content, exactly as `schema_cohort` already is, so a `dev`
  build has a well-defined one — see DD-4), and the **TOOL axis is excluded** as a trust rather
  than compatibility concern (DD-6).

  Two carve-outs travel with the resolution: the ~60-field manifest is handled by `schema_version`,
  not capabilities (DD-5 — overlapping but NOT resolving OQ-4), and `TrustedToolAllowlist` is out
  of scope, with its own problems tracked in ISSUE-082 and BUNDLE-021 (DD-6).

  **The resolution is CONDITIONAL and the condition must not be read as a nice-to-have (DD-4d):**
  capability contracts are adopted only if the registry can be guarded by a dogfooded rule that
  goes red when a capability name contains a language or tool name. If that rule proves unwritable,
  **the decision reverts to a version-based scheme.** The founder's reasoning is that a capability
  registry is a plausible back door for re-importing language-specific knowledge into core — a
  standing-invariant violation that would not look like one while it was happening, since
  `semgrep.rule_flags` is the most natural name to reach for. The legality test is mechanical
  (*would this name change if you added a new language?*), and a convention in a document is not
  protection — prompts are vibes; only executed code constrains.

- **OQ-5 — INTERACTION WITH BASELINE / RATCHET. (RESOLVED-via-handover 2026-07-26 → DD-7.)** The
  question was whether a legitimate core-upgrade format change is a MIGRATION concern this bundle
  owns (a compat-aware identity remap) or an identity-stability concern belonging to the baseline
  thread (BUNDLE-007 / the baseline CI-gen + pull work) that this bundle merely reports into.
  **Resolved to the handover**: the baseline thread owns it; this bundle owns emitting the signal.
  Rationale in DD-7 — identity stability already has an owner AND a precedent (ISSUE-062 kept
  `Properties` out of baseline identity for exactly this reason), so owning it here would duplicate
  a managed concern and widen the bundle substantially. The problem handed over is recorded in full
  in DD-7 so it is not lost in transfer: message text is part of identity today, so a reworded rule
  message re-keys every grandfathered finding in that rule, and with the ratchet default-on that
  is a wall of "new" violations on a zero-code-change upgrade and direct pressure toward mass
  waiving.

Maturity stays `exploring` — two of six resolved is not promotable, and promotion is the founder's
call regardless. No requirements are recorded yet; the requirement set awaits OQ-2/3/4/6 and, for
anything descending from DD-4, the outcome of the DD-4d condition.

## Spec Seeds

Still provisional, but firmer than at v0.1.0: OQ-1's resolution (DD-4) fixes WHAT is declared, so
the declaration seed now has a shape. OQ-2 (where it is enforced) is still what determines whether
enforcement is one spec or several. Recorded to show the plausible shape of the work, not to commit
scope.

- **The capability-name GUARD RULE — the DD-4d condition. SEQUENCED FIRST.** A dogfooded rule in
  `backstop-self-pack` that goes RED when a capability name contains a language or tool name.
  Verified 2026-07-26 to be NEW work: Family A (`no-baked-tool-exec`) matches `exec.Command`
  literals, B1 (`no-baked-tool-command`) matches `Command:` strings, B2
  (`no-baked-language-token`) matches whole-token quoted literals — a capability constant like
  `"semgrep.rule_flags"` or `"eslint.config_shape"` passes all three. Nearest neighbor is B6
  (`no-pack-name-keyed-capability`), which reaches into capability logic but catches pack-NAME
  keying, not tool names inside a capability identifier. **This seed GATES the two below**: if the
  rule cannot be written to the same standard as the existing families, DD-4 reverts to a
  version-based scheme and the declaration seed's shape changes with it. Blocked on nothing.

- **The declaration — capability contracts in pack.yml + the registry.** DD-4's capability set
  becomes a new field (or block) in pack.yml, parsed onto `pkg/pack.Manifest`
  (`pkg/pack/manifest.go`), with authoring-time validation that the declaration is well-formed —
  probably in `pkg/packval` / `pack check`, alongside the existing structural phase. Because DD-4
  chose the capability option, this seed also owns the capability-name REGISTRY (core's
  provides-set, seeded from the six enumerated contracts) and its versioning discipline. Note the
  registry must satisfy the DD-4d guard, which is why that seed sequences first. Unblocked on
  OQ-1; CONDITIONAL on the guard seed's outcome.

- **`schema_version` on pack.yml (DD-5).** Adding the manifest schema-version field that does not
  exist today, and the load-time resolution pattern that mirrors
  `artifacts/<type>/v<N>/schema.json`. Distinct from the capability declaration by DD-5's whole
  argument — a document shape versioned by revision, not a seam described by a set. May merge with
  the reverse-direction seed below depending on OQ-4; recorded separately because DD-5 is decided
  and OQ-4 is not.

- **The enforcement point(s) — comparison + posture.** The generic comparison — under DD-4 a SET
  comparison of the pack's requires-set against core's provides-set — plus its failure posture.
  Which surfaces this touches is exactly OQ-2's answer: `pkg/pack/distribution/` (add / install / update / upgrade / relock) for
  install-time, the `pack_lock_verification` step (`cmd/backstop/gate.go:771-786`) and/or a gate
  preflight for per-run. If OQ-2 lands on more than one moment they may still be ONE spec (one
  comparison function, several call sites) — that is a spec-time call, not a bundle-time one.
  Carries the OQ-3 posture sorting and the OQ-6 undeclared-default behavior. Blocked on OQ-2/3/6.

- **Reverse direction / pack-format acceptance range.** Only exists if OQ-4 resolves toward
  symmetry: core declaring an accepted range over DD-5's `schema_version`, and the reject-vs-migrate
  behavior for an out-of-range pack (including the tamper/`content_hash` interaction if migration
  mutates installed content). Recorded so the direction is not lost; NOT scoped, and may be dropped
  entirely by OQ-4 or absorbed into the DD-5 seed.

- **Dogfood declarations across the pack fleet (backstop-packs + the extracted core packs).**
  Authoring the declaration in the existing packs — `backstop/self`, the toolchain packs, contracts,
  substantiveness, and the TypeScript suite — which is what actually proves the mechanism and what
  makes an OQ-6 default tightening possible. Lives in the pack repos, not core; sequenced AFTER
  the declaration seed and coordinated with the fix → bump → relock flywheel. Blocked on OQ-1 and
  OQ-6.

- **Format-change SIGNAL emission (DD-7 — the residue of the dissolved baseline seed).** OQ-5
  resolved to the handover, so the compat-aware identity REMAP is NOT in this bundle. What survives
  is the producing half: the compatibility declaration is the only thing that knows a declared
  format changed between two core versions, and it must EMIT that as a consumable signal so the
  baseline thread can translate identities rather than re-key them. Small, but load-bearing for
  someone else's work — without it the baseline side cannot distinguish a new finding from a
  renamed one. Sequenced after the declaration seed (there is no signal until there is a
  declaration); coordinate the consuming half with BUNDLE-007 / the baseline CI-gen + pull thread.

## Notes / Ideas

- **The quiet half is the dangerous half.** ISSUE-062 produced phantom violations — loud, annoying,
  and therefore FOUND. The symmetric failure is a core that reads a field it doesn't understand as
  empty, emits nothing, and lets the gate go green with a check silently disabled. Phantom
  violations get filed as bugs within the hour; silently missing ones get filed never. Any posture
  decision (OQ-3) should weight the invisible failure heavier than the visible one, because the
  visible one is self-reporting.
- **The wire contract is implicit today.** After ISSUE-062, `Properties["func"]` /
  `Properties["symbol"]` (`pkg/gate/substantiveness_join.go:136,139,158`) is a real pack→core data
  contract with real key names and no version on it. The structured channel fixed the PARSING
  fragility; it did not fix the CONTRACT fragility. That surviving residue was the cleanest concrete
  argument for OQ-1's capability/contract option — the thing that actually needs versioning is the
  named contract, and a pack↔core semver would only ever have been a proxy for it. **This is now
  the decided direction (DD-4)**, and `properties.func`/`symbol` is one of the six contracts the
  registry must name.
- **`schema_cohort` was the answer's shape.** Core already computes an identity for "what this
  binary understands" from CONTENT rather than from a release number (`cmd/backstop/root.go:98-109`),
  and it is well-defined even for a `dev` build — exactly the case a semver comparison cannot
  handle. DD-4 adopts the same posture at the pack↔core boundary, which is why the `dev`-build
  sub-question dissolved instead of needing a special case. Worth checking at spec time whether the
  capability set should literally REUSE the cohort computation or merely mirror its shape.
- **Fleet visibility is the eventual payoff.** Once packs declare and core checks, "which consumers
  are running a core too old for the pack version they're locked to" becomes a QUERYABLE fleet
  property — the same hosted-capture surface logic that makes the migration and provenance
  dashboards valuable. Not scope; noted because it argues for the declaration being machine-legible
  and recorded in the lock (DD-2 / OQ-2) rather than living only in pack source.
- **Do not let the fix bake knowledge — now partially CONFIRMED as a gap.** The tempting shortcut
  when this bites in the field is a small table of "pack X needs core ≥ Y" in core. That is the
  exact defect class DD-1 forbids. Checked 2026-07-26: a pack-NAME literal in capability logic
  **would** be caught, by Family B6 (`no-pack-name-keyed-capability`), whose scope already includes
  `pkg/gate/*.go`, `cmd/backstop/gate.go` and `cmd/backstop/pack_gate*.go`. But a language- or
  tool-NAME inside a capability IDENTIFIER **would not** be caught by any existing family — that is
  the DD-4d gap, and it is now a condition on the whole approach rather than a note. If the
  compatibility check lands in a path outside B6's include list, that scope gap is a second thing
  to close.
- **The three-body framing is retired.** v0.1.0 framed this as pack ↔ core ↔ tool on the strength
  of the `TrustedToolAllowlist` pins. DD-6 excludes the tool axis as a trust rather than
  compatibility boundary, so the bundle now governs two bodies. The tool-axis concerns did not
  evaporate — they moved to ISSUE-082 and BUNDLE-021.

## Version History

- 0.2.0 (2026-07-26): **Two of six open questions RESOLVED by founder decision; maturity
  deliberately UNCHANGED at `exploring`** — two of six is not promotable, and promotion is the
  founder's call regardless. No requirements array added and no promotion attempted.

  **OQ-1 (what does a pack declare?) → RESOLVED to capability contracts, scoped to the wire seam,
  CONDITIONAL on a mechanical guard.** Recorded as DD-4 (+DD-4d), DD-5, DD-6. A pack declares the
  named contracts it depends on, core declares which it provides, compatibility is a SET comparison
  rather than a version comparison — chosen over min-version and semver-range because both proxy
  the real constraint (a named contract) instead of stating it. The decision was grounded, at the
  founder's insistence, in an ENUMERATION of the actual seam rather than an assertion; the
  enumeration is new in this version as *The wire seam, ENUMERATED* under Current Thinking, verified
  2026-07-26: wire formats — the SARIF result subset (`pkg/check/parsers.go:54-95`: `ruleId`,
  `level`, `message.text`, `locations[].physicalLocation.{artifactLocation.uri,
  region.startLine, region.snippet.text}`, `partialFingerprints`, `suppressions[].kind`,
  `properties`), the SARIF `properties` keys `func`/`symbol` compared VERBATIM plus
  `substantiveness_role` (`pkg/gate/substantiveness_join.go:136,139,158`, role constant at `:96`) —
  literally where ISSUE-062 broke — and the coverage record `{path, covered, total, measured,
  excluded, metric}` (`pkg/check/coverage.go:25`); declaration vocabularies — `InputMode`
  (`pkg/pack/engine/binding.go:20-32`: `config-file`, `rule-flags`, `pattern-arg`, `none`),
  `GateType` (`pkg/pack/engine/gatetype.go:38-44`: `lint`, `build`, `test`, `findings`, `coverage`,
  `substantiveness`, `contracts`), and the FieldContract field names
  (`pkg/pack/engine/fieldcontract.go`: `rule_path`, `standard`, `category`, `input_scope`,
  `validator`). **Six contracts, sixteen enum values, contamination check CLEAN** — nothing in the
  set names a language or tool; `substantiveness` and `contracts` are backstop concepts. That
  smallness is what made a registry credible rather than a second manifest schema. Two in-repo
  precedents are recorded as making the option a third application of an existing idiom rather than
  a third idiom: `backstop version --json`'s content-derived `schema_cohort`
  (`cmd/backstop/root.go:102`), which is ALSO why the `dev`-build sub-question DISSOLVES under this
  option; and `engine.FieldContract{Requires, Forbids}`.

  **DD-4d — the CONDITION, recorded as load-bearing and not as a nice-to-have.** Capability
  contracts are adopted ONLY IF the registry can be guarded by a dogfooded rule that goes red when
  a capability name contains a language or tool name; if that rule proves unwritable, **the
  decision REVERTS to a version-based scheme.** Founder's driving concern: a capability registry is
  a plausible back door for re-importing language-specific knowledge into core, which the standing
  zero-baked invariant forbids. The legality test is mechanical — *would this capability name change
  if you added a new language?* — with `sarif.properties.func_symbol` legal, `eslint.config_shape`
  illegal, and `semgrep.rule_flags` illegal and the most natural thing to type. A convention in a
  document is not protection (standing "prompts are vibes; only executed code constrains").
  Recorded honestly and verified against `backstop-self-pack/rules/no-baked.yml`: the existing
  families would NOT catch this — Family A (`no-baked-tool-exec`) covers `exec.Command` literals,
  B1 (`no-baked-tool-command`) baked `Command:` strings, B2 (`no-baked-language-token`)
  whole-token language literals like `.ts`/`tsc`/`go.mod` — so a capability CONSTANT naming a tool
  slips past all three (`"semgrep.rule_flags"` is not even in B2's token list;
  `"eslint.config_shape"` fails B2's whole-string anchoring). **The guard is NEW work**, judged
  writable because Family B6 (`no-pack-name-keyed-capability`) is its sibling one axis over. Added
  as a spec seed sequenced FIRST, gating the declaration seeds.

  **DD-5 — the ~60-field pack.yml manifest is a SCHEMA problem, not a capability problem**, and the
  mechanism is a `schema_version` field matching the pattern core already uses for artifacts
  (`artifacts/<type>/v<N>/schema.json`). Verified 2026-07-26: **pack.yml has no `schema_version`
  today**. Recorded explicitly as overlapping OQ-4's territory without resolving it — **OQ-4 remains
  OPEN**.

  **DD-6 — the tool allowlist is OUT OF SCOPE**: `TrustedToolAllowlist` (`allowlist.go:17`) is a
  TRUST boundary, not a compatibility boundary, and conflating them makes both harder to reason
  about. Its own problems are now tracked separately in **ISSUE-082** (Tool Allowlist Unreachable
  Entries, 2026-07-26) and **BUNDLE-021** (Pack Command Execution Governance, `exploring`,
  2026-07-26), both confirmed present. Consequence: the v0.1.0 THREE-body framing
  (pack ↔ core ↔ tool) is retired; this bundle governs two bodies.

  **OQ-5 (baseline/ratchet interaction) → RESOLVED-via-handover; recorded as DD-7.** Baseline
  identity churn is DE-SCOPED from BUNDLE-020: the baseline thread owns it, because identity
  stability already has an owner AND a precedent — ISSUE-062 deliberately kept `Properties` OUT of
  baseline identity for exactly this reason — and owning it here would duplicate a managed concern
  and widen the bundle substantially. What this bundle DOES own is **emitting the signal**: the
  compatibility declaration is the only thing that knows a declared format changed, and that signal
  is what the baseline work consumes to TRANSLATE identities rather than re-key them. The handed-over
  problem is recorded in full so it is not lost: identity is
  `Rule|File|RegionHash(Message|Severity|SourcePack)` (`EnrichViolationIdentity`,
  `pkg/gate/baseline.go:203-224`, the triple being the fallback hash when no
  `partialFingerprints`/snippet region exists), so message text is part of identity and a core
  upgrade that merely REWORDS a rule message re-keys every grandfathered finding in that rule —
  which, with the ratchet default-on, surfaces as a wall of "new" violations on a zero-code-change
  upgrade and creates direct pressure toward mass waiving.

  Also in this version: `bundle.updated` added (required beyond 0.1.0); `solution.approach`
  rewritten from "DIRECTION ONLY" to partially-decided; "What is decided vs open" updated; Open
  Questions restructured with a six-line status roll-up, a *Still open* subsection (OQ-2, OQ-3,
  OQ-4, OQ-6) and a *Resolved (kept for the reasoning)* subsection (OQ-1, OQ-5); OQ-4 annotated
  with DD-5's overlap and OQ-6 with the DD-4d sequencing wrinkle; Spec Seeds regrown from five to
  six with the guard rule sequenced first, the declaration seed unblocked and made conditional, a
  new DD-5 `schema_version` seed, and the baseline seed dissolved into signal-emission; Notes
  updated where they had been overtaken (the confirmed B6-catches-pack-names /
  nothing-catches-tool-names-in-capability-identifiers split, and the retired three-body framing).

- 0.1.0 (2026-07-25): Initial bundle at `exploring`. Frames the UNGUARDED pack ↔ core version
  boundary, verified 2026-07-25 against the code rather than asserted: `pkg/pack.Manifest` carries
  no compatibility field, `LockEntry` carries none, and a grep for
  `backstop_version|min_backstop|requires_backstop|compatible_version|core_version|min_core` across
  `pkg/pack/`, `artifacts/`, and `packs/` returns zero hits. Restates ISSUE-062's real shape — not
  a stale binary but a core parser disagreeing with a pack's message FORMAT across a version
  boundary, producing 3 phantom `noTarget` violations — and shows the class SURVIVES the release
  with new endpoints (consumer core v1.2 vs pack authored against v1.3), where ISSUE-077's
  "rebuild your binary" answer no longer applies and the quiet failure mode is silently MISSING
  findings, i.e. vacuous green. Records the flywheel stake (a versioned pack fix cannot safely
  propagate if nothing checks the consumer's core can read it), the THREE-body framing
  (pack ↔ core ↔ tool, via `engine.TrustedToolAllowlist()`'s semgrep 1.96.0 / ast-grep 0.43.0
  pins), and two grounding facts every answer must survive — core's own version is `var version =
  "dev"` unless `-ldflags` injects it (`cmd/backstop/root.go:16`), and `backstop version --json`
  already emits a content-derived `schema_cohort` (`root.go:102`). Names two in-repo prior arts for
  OQ-1 (`schema_cohort`; `engine.FieldContract{Requires, Forbids}`) and the surviving implicit
  wire contract (`Properties["func"]`/`["symbol"]`). Records THREE inherited-invariant design
  decisions that constrain but do NOT answer anything — DD-1 (generic over declared data, zero
  baked pack/language/tool knowledge), DD-2 (packs stay external; declaration travels with the pack,
  lock is the durable record), DD-3 (posture adjudicated per-case by loud-≠-blocking, never
  uniform). Raises SIX open questions, none pre-resolved and none carrying a lean — OQ-1 (what a
  pack declares: min version / semver range / capability-contract set, plus the `dev`-build and
  tool-axis sub-questions), OQ-2 (where enforced: install / lock verification / gate preflight /
  `pack check`, and whether install-time alone leaves the core-upgraded-after-install case open),
  OQ-3 (failure posture: which cases block vs warn, whether the declaration must carry the posture,
  and `ConfigErr`/exit-2 vs violation/exit-1), OQ-4 (reverse direction: should core declare a
  pack-format version to reject or migrate an older pack, with the `content_hash`/tamper
  interaction), OQ-5 (baseline/ratchet interaction on a legitimate format change — this bundle's
  migration concern or the baseline thread's identity-stability concern), OQ-6 (retroactive default
  for the 100% of packs that declare nothing today: assume-compatible / warn-until-declared /
  deadline). Five provisional, contingency-marked spec seeds. Maturity stays `exploring`; the
  founder drives OQ resolution and promotion.

## References

- **ISSUE-062 (Structured Finding Properties)** — the ORIGINATING incident, closed 2026-07-17,
  delivered by PLAN-ISSUE-062. The pack emitted `func=$FN symbol=$PKG` in the finding message; core
  parsed it with `tokenValue`; the fix moved the payload to SARIF result `properties` and deleted
  the prose parsers. The 3 phantom `noTarget` violations raised while the halves were out of step
  are this bundle's exhibit A. Its residue — the unversioned `Properties["func"]`/`["symbol"]` key
  contract at `pkg/gate/substantiveness_join.go:136,139,158` — is live input to OQ-1.
- **ISSUE-077 (Stale Gate Binary Phantom Violations)** — the LOCAL / contributor half of the same
  class: binary-vs-source drift (filed 2026-07-25). Documents three binary copies a week apart by
  timestamp, the same ISSUE-062 phantom-`noTarget` failure mode, and a permission-denied self-heal
  trap; proposes a PATH shim that rebuilds-if-stale then execs (plus deleting the stray root
  `./backstop`) and a gate-startup mtime self-check exiting 2 as a `*check.ConfigError`. Both
  halves of that resolution are STALENESS checks against a local source tree, which is exactly why
  they do not reach this bundle's case. It covers the endpoint pair that evaporates once consumers
  install release binaries; this bundle covers the endpoint pair that does not.
- **ISSUE-076 (Plan Verification Commands Unresolvable)** — sibling filed in the same 2026-07-25
  sweep (validated plans and the implementer contract mandating `backstop code check`, deleted by
  ISSUE-018). Not a dependency; recorded as sweep provenance and as an independent instance of the
  same underlying theme — an artifact and the tool it targets drifting apart with nothing checking.
- **`pkg/pack/manifest.go`** — the manifest model the declaration would extend; the verified
  absence of any compatibility field is the bundle's premise.
- **`pkg/pack/distribution/`** (`add.go`, `install.go`, `update.go`, `upgrade.go`, `relock.go`,
  `lockfile.go`, `tamper.go`) — the install-time enforcement candidates for OQ-2, the `LockEntry`
  shape for DD-2, and the `content_hash`/tamper interaction OQ-4's migration option would disturb.
- **`cmd/backstop/gate.go:771-786`** (`pack_lock_verification`) and **`pkg/gate/policy.go:49`** —
  the per-run enforcement candidate for OQ-2, and the existing `ConfigErr: true` / exit-2 posture
  precedent OQ-3 has to place itself against.
- **`cmd/backstop/root.go:16,98-109`** — `var version = "dev"` (the `-ldflags` build-time injection,
  and therefore the dev-build hole in any semver comparison) and the `schema_cohort` computation
  (the content-derived binary-identity precedent for OQ-1).
- **`pkg/pack/engine/fieldcontract.go`** — `FieldContract{Requires, Forbids}`, the existing
  pack-declares-a-contract-core-validates precedent for OQ-1's capability option.
- **`pkg/pack/engine/allowlist.go:17`** — `TrustedToolAllowlist()` (semgrep 1.96.0, ast-grep 0.43.0,
  plus `"*"` presence pins for grep/rg/oxlint/bun/tsc/prettier). Framed at v0.1.0 as the third body
  in a pack ↔ core ↔ tool problem; **DD-6 now excludes it** as a trust rather than compatibility
  boundary. Its own in-code comment draws the same line ("the allowlist KEY is a tool-name lookup
  datum … NOT a baked routing/command literal — it never sources a command").
- **ISSUE-082 (Tool Allowlist Unreachable Entries)** — filed 2026-07-26, `open`. One of the two
  homes DD-6 hands the allowlist's problems to. NOT a dependency of this bundle.
- **BUNDLE-021 (Pack Command Execution Governance)** — created 2026-07-26, `exploring`. The other
  DD-6 home: what governs arbitrary pack-declared commands. NOT a dependency of this bundle.
- **`pkg/check/parsers.go:54-95`** (`sarifLog`), **`pkg/gate/substantiveness_join.go:96,136,139,158`**
  (`func` / `symbol` / `substantiveness_role`), **`pkg/check/coverage.go:25`** (`CoverageRecord`),
  **`pkg/pack/engine/binding.go:20-32`** (`InputMode`), **`pkg/pack/engine/gatetype.go:38-44`**
  (`GateType`) — the six contracts and sixteen enum values constituting the wire seam DD-4 scopes
  the capability declaration to. All re-verified 2026-07-26.
- **`pkg/gate/baseline.go:203-224`** (`EnrichViolationIdentity`) — the identity composition DD-7
  hands to the baseline thread: `Rule|File|RegionHash`, where `RegionHash` falls back to
  `hash(Message|Severity|SourcePack)` when the engine supplies no `partialFingerprints`/snippet.
- **`backstop-self-pack/rules/no-baked.yml`** (external pack repo) — Families A
  (`no-baked-tool-exec`), B1 (`no-baked-tool-command`), B2 (`no-baked-language-token`) and B6
  (`no-pack-name-keyed-capability`). Read 2026-07-26 to establish that none of them would catch a
  language/tool name inside a capability identifier — the gap DD-4d makes a condition.
- **`artifacts/<type>/v<N>/schema.json` + artifact `schema_version`** — the declare-the-format
  so-the-reader-can-route pattern already in use for ARTIFACTS but conspicuously absent from
  pack.yml; the closest structural analogue for OQ-4.
- **backstop/self pack** — enforces the zero-baked-knowledge boundary DD-1 asserts; the
  compatibility check must not become a table of known pack names, and the Notes flag a possible
  rule-coverage gap there.
