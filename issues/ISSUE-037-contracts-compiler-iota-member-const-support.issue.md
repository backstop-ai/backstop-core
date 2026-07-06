---
title: "Contracts Compiler Iota Member Const Support"
schema_version: issue/v1

issue:
  id: ISSUE-037
  title: "Contracts Compiler Iota Member Const Support"
  type: technical-debt
  status: open
  created: "2026-07-05"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: moderate
---

# ISSUE-037: Contracts Compiler Iota Member Const Support

## Problem

ISSUE-036's kind-aware `compile-signature.sh`
(`packs/contracts/scripts/compile-signature.sh`) made `type`/`const`/`var`/method
contracts signature-verifiable for real, but by design it can only compile a
`const` signature that carries an explicit RHS value — its own header states
the rule: "the '=' is REQUIRED — a bare `const $NAME $$$` is an ast-grep ERROR
node that matches nothing." That is correct for a standalone assigned
constant. It leaves a genuine, structural gap: **a bare iota-block const
member has no `= value` to compile at all**, so no signature this compiler
can emit will ever match it — not a bug in the fix, a case the fix
deliberately declined to solve.

### Verified reproduction (2026-07-05)

Against a real iota block —

```go
const (
	CheckTypeLint CheckType = iota
	CheckTypeBuild
	CheckTypeTest
	CheckTypeFindings
)
```

the only pattern that name-binds the bare member, `const CheckTypeFindings $$$`
(no `=`), is rejected by ast-grep before it ever gets to matching:

```
$ ast-grep --pattern 'const CheckTypeFindings $$$' --lang go iota_test.go
Warning: Pattern contains an ERROR node and may cause unexpected results.
Help: ast-grep parsed the pattern but it matched nothing in this run.
```

`ast-grep run --pattern <pat> --debug-query` confirms the pattern itself
parses to an `ERROR` node (not a valid Go fragment), and the current
compiler's actual emission for a `const` signature, `const $NAME = $$$`, also
cannot match a bare member since there is no `=` in the source to align
against. There is no string the compiler could emit under the pattern-arg
model that satisfies both "valid ast-grep pattern" and "matches a value-less
iota member" — the gap is structural to `--pattern`-string matching, not a
missed case in the ISSUE-036 fix.

### The concrete instance (already reconciled, not this issue's scope to redo)

SPEC-035's `CheckTypeFindings` contract (`pkg/check/manifest.go:22`, `kind:
constant`, declared signature `const CheckTypeFindings CheckType = ...`) is
exactly this case — a bare iota member with a fictional value-signature. That
one contract has been reconciled separately as part of landing ISSUE-036:
retired as inexpressible under the current compiler, with the symbol's real
guarantee (the neutral "findings" rename) covered behaviorally by
`TestCheckType_StringAndParseUseNeutralFindingsNotSemgrep` and sibling tests
rather than structurally by `contract_signature`. This issue is the general
capability gap that produced that retirement, not a request to re-litigate
that one contract.

### Audit: is the gap latent elsewhere? (done here, not deferred)

A full scan of every `kind: constant` contract in the repo today
(`grep -rn "kind: constant" specs/*.spec.md issues/*.issue.md`) turns up
entries in SPEC-004, SPEC-005, SPEC-025, SPEC-033, SPEC-035, SPEC-036,
SPEC-038, SPEC-042. Checking each declared signature:

- **SPEC-005** (`ExitPass`, `ExitViolations`, `ExitConfigError`) — all carry
  explicit values (`const ExitPass = 0`, etc.). Expressible today; not iota
  members.
- **SPEC-025** (`OutcomeSuccess`, `OutcomeFailedAction`,
  `OutcomeBlockedSafety`) — all carry explicit typed string values
  (`const OutcomeSuccess Outcome = "success"`, etc.). Expressible today.
- **SPEC-036** (`DimensionSubstantiveness`, `DimensionCoverage`,
  `DimensionContracts`) — same shape, explicit string values. Expressible
  today.
- **SPEC-038** and **SPEC-042** — the `kind: constant` entries there
  (`GateTypeContracts`, `InputModePatternArg`, `GateTypeCoverage`) are
  `consumes`-only declarations with no `signature` field at all, so they are
  not run through the signature compiler in the first place.
- **SPEC-004** and **SPEC-033** — their `kind: constant` entries are not Go
  declarations at all (signatures like `"JSON schema definition"`, `"Shell
  script"`, prose describing a hand-off artifact set) — a pre-existing schema
  fit issue orthogonal to this one, out of scope here.

**Result: `CheckTypeFindings` (SPEC-035, already retired) is the only known
bare-iota-member instance in the repo today.** The audit found no other
latent case waiting to surface. This narrows the immediate blast radius
considerably from "unknown, could be several" to "one, already handled" —
but the underlying compiler limitation is still real and will recur the
moment any future spec declares a contract on a value-less iota member (Go's
own idiom for exactly this style of enum, and one backstop's own gate
vocabulary already uses).

### A validated fix direction exists, but it needs a capability the pack doesn't have yet

Bare-member membership genuinely can be checked structurally with ast-grep —
just not via a `--pattern` string, which is all the `ast-grep-contracts`
engine's `pattern-arg` `input_mode` supplies today
(`packs/contracts/pack.yml`: `input_mode: pattern-arg`, `input_flag:
--pattern`). A relational **YAML rule** (not a bare pattern string) can
distinguish a true declaration from a mere reference to the same identifier.
Verified directly:

```yaml
# too broad — matches ANY identifier occurrence inside a const block,
# including a reference like `Foo = CheckTypeFindings` in a different spec
rule:
  kind: identifier
  regex: '^CheckTypeFindings$'
  inside:
    kind: const_declaration
    stopBy: end
```

```yaml
# correct — scopes to the const_spec's OWN `name` field, so a same-named
# reference elsewhere in the block does not false-positive
rule:
  kind: const_spec
  has:
    field: name
    kind: identifier
    regex: '^CheckTypeFindings$'
```

Run against a fixture with both a genuine bare-member declaration and a
same-named reference (`Foo = CheckTypeFindings`) in a sibling `const_spec`,
the first rule matched both (false positive on the reference); the second
matched only the true declaration. So the ast-grep expressibility question
is resolved: iota-member presence CAN be checked structurally and precisely.
What's missing is a way to hand ast-grep a relational YAML rule instead of a
single `--pattern` string — the `ast-grep-contracts` engine binding only
wires `pattern-arg`. Any fix needs either a new `input_mode` (a `rule-arg` or
similar that supplies a rule file/inline rule content, not just a pattern
string) or some other extension to the pack-engine substrate before an
iota-aware compiler mode has anywhere to route its output.

### Why it matters

`iota` blocks are an idiomatic, pervasive Go enum pattern, and backstop's own
gate-type vocabulary (`CheckType`) is itself one. Until this gap closes, the
contracts dimension has a structural blind spot for an entire class of
`const` declarations — a residual vacuous-green pocket in the same dimension
ISSUE-036 just made real for every other kind. It is latent rather than
active today (the audit above found the gap has exactly one instance, and
that one is already handled), but it will recur silently the next time
someone declares a `kind: constant` contract on a bare enum member, and
nothing today would stop them from doing so with a signature that can never
compile to a matching pattern.

## Solution

Not committed — left open for the plan. Directions to evaluate, in
roughly ascending order of mechanism weight:

1. **Fail loud at authoring time instead of silently compiling to a
   never-matching pattern.** Whatever else happens, the compiler (or a
   pack-side/schema-side guard ahead of it) should refuse to silently emit a
   pattern for a `const` signature that has no `=` and no way to bind a real
   iota member — today a bare `const NAME TYPE = ...`-shaped signature with no
   real RHS just silently compiles to something that will never match, which
   is the same silent-vacuous anti-pattern ISSUE-036 exists to eradicate.
   This direction is worth doing regardless of which of the below is also
   pursued.
2. **Add a distinct contract `kind` (or a `const`-signature convention, e.g. a
   `signature: "iota-member"` sentinel) for enum/iota members**, compiled via
   the validated relational rule above once the engine substrate can carry a
   rule instead of a bare pattern string. This is the structural fix — it
   makes iota members verifiable the same way every other kind now is.
3. **Extend the `ast-grep-contracts` engine (or add a sibling engine) with a
   new `input_mode`** that supplies a rule file or inline rule content rather
   than a `--pattern` string — the prerequisite for (2). Scope this against
   BUNDLE-009 / SPEC-035's existing `pattern-arg` precedent so a new mode
   follows the same declared, pack-side, no-baked-tool-knowledge shape.
4. **Or, accept iota members as behaviorally- not structurally-verified** (the
   disposition already used for `CheckTypeFindings`) and make that an
   explicit, named schema/pack convention rather than an ad hoc per-contract
   judgment call — so the next author has a documented, sanctioned path
   instead of independently rediscovering "retire it, cover it with tests."

Whichever direction is chosen, direction 1 (fail loud on an uncompileable
`const` signature) should ship regardless — it converts today's silent
vacuous-green failure mode into a loud one even before the structural fix
(2+3) exists.

**Core uncertainty, stated honestly:** the ast-grep expressibility question
is resolved by the verified relational rule above — this is not a research
unknown. What's open is which of directions 2-4 is worth the engine-substrate
work given the audit shows exactly one real-world instance exists today (and
it's already handled behaviorally). The plan should weigh "build the general
capability now" against "add the fail-loud guard now and defer the
structural fix until a second instance actually appears" rather than assume
the heavier fix is obviously warranted.

## References

- `packs/contracts/scripts/compile-signature.sh` — the kind-aware compiler
  (ISSUE-036); its own header states the `const` `=`-required rule that this
  issue's gap sits behind
- `packs/contracts/pack.yml` — the `ast-grep-contracts` engine binding;
  `input_mode: pattern-arg` / `input_flag: --pattern` is the substrate
  limitation blocking the relational-rule fix direction
- `pkg/gate/step_contract.go` (`StepContractSignatureScopedFunc`,
  `ContractEntry`) — the language-agnostic gate-side consumer of pack-produced
  contract results; unaffected by this issue, documented for context
- `pkg/check/manifest.go:22` — `CheckTypeFindings`, the bare iota member that
  is the concrete (already-retired) instance of this gap
- `SPEC-035-pack-declared-engines-trusted-allowlist.spec.md` — declares the
  `CheckTypeFindings` contract (`kind: constant`, `pkg/check/manifest.go`)
  whose signature is the fiction this issue is about; the neutral-rename
  behavioral tests (`TestCheckType_StringAndParseUseNeutralFindingsNotSemgrep`
  and siblings) are what actually cover that symbol's guarantee today
- `SPEC-038-traceability-contracts-pack.spec.md` — the contracts pack's origin
  spec (compiler, diff-scoped step, `kind` enum); the substrate this issue's
  fix directions would extend
- ISSUE-036 (contracts-pack-compiler-func-only-signatures) — the kind-aware
  compiler fix that made every other `kind` verifiable and, in doing so,
  exposed this narrower iota-member gap as the one case it correctly declined
  to solve
- ISSUE-034 (gate-coverage-flags-deleted-files), ISSUE-035
  (gate-substantiveness-flags-testmain-absence-tests) — sibling family filed
  the same day: dark/dormant gate-check logic exposed for the first time as
  diff scope widened under the eradication backlog
- CLAUDE.md — "Loud ≠ blocking" enforcement philosophy and the
  zero-baked-checks / no-vacuous-green first principle this gap is a residual
  pocket of
