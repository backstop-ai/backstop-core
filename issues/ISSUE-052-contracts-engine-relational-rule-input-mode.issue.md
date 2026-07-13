---
title: "Contracts Compiler Structural-Presence Capability (Relational-Rule Input Mode)"
schema_version: issue/v1

issue:
  id: ISSUE-052
  title: "Contracts Compiler Structural-Presence Capability (Relational-Rule Input Mode)"
  type: technical-debt
  status: open
  created: "2026-07-13"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: safe
---

# ISSUE-052: Contracts Compiler Structural-Presence Capability (Relational-Rule Input Mode)

## Problem

The `backstop/contracts` engine's `compile-signature.sh` compiler cannot
express **structural presence** for two distinct symbol classes under its
`pattern-arg` / `--pattern` input mode:

1. **Bare iota-member `const`s** (no `=` to bind a pattern against) — the
   case ISSUE-037 originally established, with a verified reproduction.
2. **Struct fields** (`Foo bool `yaml:"..."`` inside a `type X struct {
   ... }`) — the case the ISSUE-038 contract-drift reconciliation
   (2026-07-13) proved has the identical root cause and, unlike the iota
   case, has live instances today (see "Why this is now justified" below).

Both compile to ast-grep `ERROR` nodes that match nothing, for the same
underlying reason: the compiler only ever emits a single `--pattern`
string, and neither a value-less const member nor a field declaration is a
standalone Go fragment a `--pattern` string can bind. The engine wires
exactly one `input_mode`:

```yaml
# packs/contracts/pack.yml
input_mode: pattern-arg
input_flag: --pattern
```

which supplies ast-grep a single `--pattern` string. A bare iota member has
no `= value` to bind a pattern against — any pattern string that name-binds
the member (`const CheckTypeFindings $$$`) is rejected by ast-grep as an
`ERROR` node before matching even runs; ISSUE-037's reproduction confirms
this is a structural limit of `--pattern`-string matching, not a missed
case.

ISSUE-037 also verified, directly, that the member's presence CAN be
checked structurally — just not via a bare pattern string. A relational
YAML rule scoped to the `const_spec` node's own `name` field distinguishes
a true declaration from a same-named reference elsewhere in the file:

```yaml
# correct — scopes to the const_spec's OWN `name` field
rule:
  kind: const_spec
  has:
    field: name
    kind: identifier
    regex: '^CheckTypeFindings$'
```

verified against a fixture with both a genuine bare-member declaration and a
same-named reference in a sibling `const_spec` — this rule matched only the
true declaration; a looser `kind: identifier` rule false-positived on the
reference. So the ast-grep expressibility question for the iota case is
closed.

### The struct-field variant (ISSUE-038, 2026-07-13)

The ISSUE-038 contract-drift reconciliation empirically proved the same
gap recurs, unmodified, for struct field declarations. Against ast-grep
0.43.0, every pattern shape tried for a field signature returns an empty
match set:

```
$ ast-grep --pattern 'ExemptFromScopeFilter bool `yaml:"exempt_from_scope_filter"`' --lang go binding.go
[]   # field-with-tag form
$ ast-grep --pattern 'ExemptFromScopeFilter bool' --lang go binding.go
[]   # bare-field form
$ ast-grep --pattern 'var ExemptFromScopeFilter bool' --lang go binding.go
[]   # var-prefixed form — not even valid Go for a struct field, still emitted and still empty
```

All three forms compile to ast-grep `ERROR` nodes for the identical reason
the iota case does: a struct field is not a standalone Go fragment, so no
`--pattern` string can bind it. Confirming this is not an isolated
mis-compile: a scan of every **currently present** (non-`kind: absent`)
struct-field `contract_signature` entry in the whole `specs/` tree found
**zero** that pass — the failing entries below are the *only* struct-field
contracts in the tree, present or absent. There is no working precedent to
diverge from.

## Why this is now justified — 3 live instances, not 0

ISSUE-037's iota-member gap had exactly one instance
(`CheckTypeFindings`, `pkg/check/manifest.go:22`), and it was already
retired/reconciled — behaviorally covered by
`TestCheckType_StringAndParseUseNeutralFindingsNotSemgrep` and siblings,
not structurally by `contract_signature`. That kept the iota case
speculative: building a new engine `input_mode` before a second real
instance existed would have been capability-building ahead of demand.

The struct-field variant changes that calculus. The ISSUE-038
reconciliation found **3 current, live, grandfathered**
`contract_signature` findings that are exactly this case:

| Symbol | File | Declared by |
|---|---|---|
| `ExemptFromScopeFilter` | `pkg/pack/engine/binding.go:261` | SPEC-041, SPEC-048 |
| `Manifest.Classification` | `pkg/pack/manifest.go` | SPEC-043, SPEC-045, SPEC-047 |
| `Manifest.TestNamePatterns` | `pkg/pack/manifest.go` | SPEC-043, SPEC-045, SPEC-047 |

All three are real, currently-declared fields on real, currently-`implemented`
specs — not retired, not hypothetical. They stay red-but-grandfathered
(ISSUE-038's grandfathering mechanism) until this capability lands. This is
no longer "track it in case a second instance appears someday" — the
instances are here, today, and this issue is the tracked fix for them.

## Solution

Not committed — left open for the plan. The fix is the same for both
symbol classes (iota members and struct fields): give the engine a way to
express a relational rule instead of a bare pattern string.

1. Add a new `input_mode` to the `ast-grep-contracts` engine binding (e.g.
   `rule-arg` or `rule-file`) that supplies a relational YAML rule — inline
   content or a rule-file path — instead of a `--pattern` string. Model it
   on the existing `pattern-arg` / `--pattern` precedent
   (`packs/contracts/pack.yml`, BUNDLE-009 / SPEC-035) so the new mode stays
   declared and pack-side with no baked tool knowledge in core: backstop
   still only ever shells an allowlisted command and speaks SARIF back.
2. Give `compile-signature.sh` (or a sibling compiler path) a way to detect
   a structurally-uncompileable signature — a value-less iota-member
   `const`, or a struct field — and emit a relational rule through the new
   `input_mode` instead of falling back to the always-fails `pattern-arg`
   path. For the iota case: `kind: const_spec` + `has: {field: name, ...}`
   (validated by ISSUE-037). For the field case: an analogous rule scoped
   to `kind: field_declaration` (or the struct's own field-name node) with
   a `has:` name clause — same shape, different node kind. A distinct
   `kind`-aware compiler path per symbol class is also an acceptable
   implementation of the same fix if it turns out cleaner than one shared
   relational-rule emitter.
3. **Fail-loud guard (absorbed from ISSUE-037's Solution direction 1).**
   Independent of whether/when the structural fix above ships,
   `compile-signature.sh` should refuse to silently emit a pattern for a
   signature it cannot compile — today a value-less const or a struct
   field just silently compiles to a `--pattern` string that will never
   match, which is the same silent-vacuous-green anti-pattern ISSUE-036
   exists to eradicate. The compiler should instead emit a loud
   "uncompileable under pattern-arg — needs relational rule" error, so an
   author gets an honest signal instead of a permanent, unexplained
   "signature not found."
4. Add a mandated test proving the new mode compiles and matches a genuine
   bare-member declaration and a genuine struct field, while correctly
   rejecting a same-named reference elsewhere in the file (the exact
   fixture shape ISSUE-037 already validated by hand for the const case;
   needs an analogous fixture for the field case).
5. Once the structural fix lands, reconcile the 3 grandfathered findings
   above (`ExemptFromScopeFilter`, `Manifest.Classification`,
   `Manifest.TestNamePatterns`) out of ISSUE-038's grandfathered set —
   they should go from "red but excused" to genuinely green.

## References

- ISSUE-037 (contracts-compiler-iota-member-const-support) — **replaced by
  this issue** (status `replaced`, `replaced-by: ISSUE-052`, 2026-07-13).
  Contains the full verified iota-member reproduction and the relational-
  rule proof (the `const_spec` / `has: {field: name, ...}` rule) this
  issue's Solution direction builds on, plus the original fail-loud guard
  direction absorbed into Solution item 3 here
- ISSUE-038 (reconcile-contract-drift-exposed-by-kind-aware-compiler) — the
  2026-07-13 reconciliation that found the struct-field variant and its 3
  live instances, and grandfathers them pending this issue's fix
- ISSUE-036 (contracts-pack-compiler-func-only-signatures) — the kind-aware
  compiler baseline this would extend with a new relational-rule mode
- `packs/contracts/pack.yml` — the `ast-grep-contracts` engine binding;
  `input_mode: pattern-arg` is the substrate limitation this issue proposes
  to extend, not replace
- `packs/contracts/scripts/compile-signature.sh` — the kind-aware compiler;
  its own header documents the `const` `=`-required rule this issue's gap
  sits behind, and is where the new struct-field detection + fail-loud
  guard would also live
- `pkg/check/manifest.go:22` — `CheckTypeFindings`, the one known
  (already-retired) bare-iota-member instance
- `pkg/pack/engine/binding.go:261` — `ExemptFromScopeFilter`, a live
  struct-field instance
- `pkg/pack/manifest.go` — `Manifest.Classification` and
  `Manifest.TestNamePatterns`, two more live struct-field instances
- `SPEC-035-pack-declared-engines-trusted-allowlist.spec.md` — declares the
  now-retired `CheckTypeFindings` contract
- `SPEC-041-coverage-reimpl-checktype-catalog.spec.md`,
  `SPEC-048-engine-dispatch-selftarget-stdout-artifact.spec.md` — declare
  the live `ExemptFromScopeFilter` contract
- `SPEC-043-pack-declared-globs-coverage-consumer.spec.md`,
  `SPEC-045-de-go-test-verification-discovery.spec.md`,
  `SPEC-047-bun-toolchain-pack-and-proof.spec.md` — declare the live
  `Manifest.Classification` / `Manifest.TestNamePatterns` contracts
- `SPEC-038-traceability-contracts-pack.spec.md` — the contracts pack's
  origin spec; the substrate this issue's fix direction would extend
- BUNDLE-009 / the `pattern-arg` precedent — the declared, pack-side,
  no-baked-tool-knowledge shape a new `input_mode` should follow
- `directives/DIR-015-gate-checker-hardening.directive.md` — lists
  ISSUE-037 (now absorbed here) as remaining work in the same
  gate-hardening cluster this issue tracks alongside
- CLAUDE.md — "packs are external by design" and zero-baked-tool-knowledge
  first principles a new engine `input_mode` must respect
