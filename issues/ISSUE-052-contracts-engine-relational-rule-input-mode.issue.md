---
title: "Contracts Engine Relational Rule Input Mode"
schema_version: issue/v1

issue:
  id: ISSUE-052
  title: "Contracts Engine Relational Rule Input Mode"
  type: technical-debt
  status: open
  created: "2026-07-13"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: safe
---

# ISSUE-052: Contracts Engine Relational Rule Input Mode

## Problem

ISSUE-037 established, with a verified reproduction, that a bare
iota-block `const` member (Go's idiomatic enum pattern — `CheckTypeLint
CheckType = iota`, `CheckTypeBuild`, ... with no `=` on the individual
member) cannot be checked by the `ast-grep-contracts` engine as it exists
today. The engine only wires one `input_mode`:

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
reference. So the ast-grep expressibility question is closed. What's
missing is a way to hand ast-grep this relational rule instead of a
`--pattern` string — the `ast-grep-contracts` engine binding has nowhere to
route it. Closing the iota gap structurally requires a new engine
`input_mode` (a `rule-arg` / `rule-file`-shaped mode that supplies rule
content, not a pattern string) plus a compiler mode that emits it for
iota-member `const` signatures.

## Why this is deferred, not built now

ISSUE-037's full-repo audit of every `kind: constant` contract
(`SPEC-004/005/025/033/035/036/038/042`) found **exactly one** real
bare-iota-member instance in the repo — `CheckTypeFindings`
(`pkg/check/manifest.go:22`, declared by SPEC-035) — and it is already
retired/reconciled: the contract's real guarantee is covered behaviorally by
`TestCheckType_StringAndParseUseNeutralFindingsNotSemgrep` and sibling
tests rather than structurally by `contract_signature`. Zero live
iota-member contracts remain today. Building a new engine `input_mode`
now, before a second real instance exists, would be speculative
capability-building against a gap with a fail-loud guard already covering
its only known blast radius (ISSUE-037's Solution direction 1).

This issue exists to **track the structural capability** so it is not
rediscovered from scratch the next time a `kind: constant` contract is
declared on a genuine iota member — it is ISSUE-037's deferred structural
follow-on, not new scope.

## Solution

Not committed — left open for the plan that eventually picks this up
(triggered by a second real bare-iota-member instance, not by a deadline).
Direction, following the shape ISSUE-037 already laid out:

1. Add a new `input_mode` to the `ast-grep-contracts` engine binding (e.g.
   `rule-arg` or `rule-file`) that supplies a relational YAML rule — inline
   content or a rule-file path — instead of a `--pattern` string. Model it
   on the existing `pattern-arg` / `--pattern` precedent
   (`packs/contracts/pack.yml`, BUNDLE-009 / SPEC-035) so the new mode stays
   declared and pack-side with no baked tool knowledge in core: backstop
   still only ever shells an allowlisted command and speaks SARIF back.
2. Give `compile-signature.sh` (or a sibling compiler path) a way to detect
   a value-less iota-member `const` signature and emit the relational rule
   above (scoped to `const_spec` + `has: {field: name, ...}`) through the
   new `input_mode`, rather than falling back to the always-fails
   `pattern-arg` path it uses for every other `const`.
3. Add a mandated test proving the new mode compiles and matches a genuine
   bare-member declaration while correctly rejecting a same-named reference
   in a sibling `const_spec` (the exact fixture shape ISSUE-037 already
   validated by hand).

## References

- ISSUE-037 (contracts-compiler-iota-member-const-support) — the issue this
  is the deferred structural half of; contains the full verified
  reproduction and the relational-rule proof this issue's Solution direction
  builds on
- ISSUE-036 (contracts-pack-compiler-func-only-signatures) — the kind-aware
  compiler baseline this would extend with a new iota-aware mode
- `packs/contracts/pack.yml` — the `ast-grep-contracts` engine binding;
  `input_mode: pattern-arg` is the substrate limitation this issue proposes
  to extend, not replace
- `packs/contracts/scripts/compile-signature.sh` — the kind-aware compiler;
  its own header documents the `const` `=`-required rule this issue's gap
  sits behind
- `pkg/check/manifest.go:22` — `CheckTypeFindings`, the one known
  (already-retired) bare-iota-member instance
- `SPEC-035-pack-declared-engines-trusted-allowlist.spec.md` — declares the
  now-retired `CheckTypeFindings` contract
- `SPEC-038-traceability-contracts-pack.spec.md` — the contracts pack's
  origin spec; the substrate this issue's fix direction would extend
- BUNDLE-009 / the `pattern-arg` precedent — the declared, pack-side,
  no-baked-tool-knowledge shape a new `input_mode` should follow
- `directives/DIR-015-gate-checker-hardening.directive.md` — lists ISSUE-037
  as remaining work in the same gate-hardening cluster this issue tracks
  alongside
- CLAUDE.md — "packs are external by design" and zero-baked-tool-knowledge
  first principles a new engine `input_mode` must respect
