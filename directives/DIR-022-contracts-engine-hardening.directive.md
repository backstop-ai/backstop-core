---
title: "Contracts Engine Hardening"
number: DIR-022
created: "2026-07-15"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "ISSUE-052"
    - "ISSUE-053"
    - "ISSUE-078"
---

## Description

Grow the `backstop/contracts` engine's `contract_signature` compiler past
its current func-only, Go-shaped-`--pattern` foundation, on two fronts:

1. **Relational-rule input mode (ISSUE-052).** `compile-signature.sh` can
   only emit a single ast-grep `--pattern` string, which structurally
   cannot express a bare iota-member `const` (no `=` to bind against) or a
   struct field declaration (not a standalone Go fragment) — both compile
   to `ERROR` nodes that silently match nothing. This was speculative when
   only the iota case (ISSUE-037, already retired/behaviorally-covered)
   existed; it is no longer speculative — the ISSUE-038 contract-drift
   reconciliation (2026-07-13) found 3 live, currently-`implemented`-spec
   struct-field instances (`ExemptFromScopeFilter`,
   `Manifest.Classification`, `Manifest.TestNamePatterns`) that are
   red-but-grandfathered pending this fix. The fix direction is proven, not
   just theorized: ISSUE-037 already validated by hand that a relational
   YAML rule (`kind: const_spec` + `has: {field: name, ...}`) correctly
   scopes to a symbol's own declaration and rejects a same-named reference
   elsewhere in the file. This directive's scope is wiring that proof into
   a new engine `input_mode` (e.g. `rule-arg`/`rule-file`), a matching
   compiler detection path, a fail-loud guard for signatures the compiler
   still can't express, and reconciling the 3 grandfathered findings back
   to genuinely green once it lands.
2. **Non-Go artifact contracts (ISSUE-053).** The contracts schema
   explicitly allows non-`function` contract kinds
   (`type`/`interface`/`method`/`constant`/`variable`) with no restriction
   to Go artifacts, but the signature compiler only understands Go — so a
   contract declared on a JSON schema, a shell script's stdin/stdout shape,
   or a Markdown agent definition is permanently unverifiable structurally.
   Verified instances today (SPEC-003, SPEC-004, SPEC-033) are all on
   `draft` specs, so ISSUE-051's implemented-only scoping will temporarily
   drop this to zero live instances once it lands — latent, not fixed; the
   gap resurfaces the moment any spec (SPEC-042 already shows the shape can
   occur on an `implemented` spec) declares a non-Go `kind: constant`/
   `variable` contract while implemented. Three candidate directions are
   open for the plan: scope non-Go contracts out of `contract_signature`
   entirely, route them to an artifact-appropriate verification (jq/schema
   validation, grep/shellcheck-style), or retire prose signatures as
   documentation-only by convention. Whichever direction is chosen must not
   force-fit a fictional Go-looking signature onto a non-Go artifact — the
   same anti-pattern ISSUE-037 already rejected for iota members.

Both threads sit in the same gate-correctness cluster as ISSUE-036/037/038
(DIR-015, already `done`) — this directive is the next chapter of that
work, not a reopening of it.

**Grouped const/var block members (ISSUE-078, folds into thread 1).**
ISSUE-078 surfaces two findings while triaging SPEC-054's `RecipeKind`/
`Op.Kind` consts. Finding 2 is a third live instance of thread 1's
relational-rule gap: the compiler's `const $NAME = $$$` / `var $NAME =
$$$` patterns bind only a standalone declaration, not a member of an
idiomatic grouped `const ( … )` / `var ( … )` block — same "not a
standalone Go fragment" limitation as the iota-member and struct-field
cases, no new mechanism needed beyond thread 1's planned relational-rule
`input_mode`. It is now actively costing coverage on shipped code, not
just theoretical: SPEC-054 could not declare `provides` contracts for
`RecipeKind`/`Op.Kind` because of it. Finding 1 is a separate, currently
gate-red matter: ISSUE-036's CLM-008 mandates five kebab-case pack.yml
`claims[].id` values (`type-signature-go` and siblings) that can never
satisfy `gate.TestNameMatcher` (built from toolchain `TestNamePatterns`
like `^Test[A-Z]`, `cmd/backstop/gate.go:1188`), producing 5 permanent
broken-promise violations even though the underlying rules exist and
pass. Its fix has a gate-side component (claim-ID-vs-test-name matching),
not a contracts-compiler change — a planner should not assume thread 1's
compiler work covers it.

## Notes

ISSUE-052 and ISSUE-053 are grouped into one directive because they are
both engine-capability extensions to the same compiler
(`compile-signature.sh` / `ast-grep-contracts`) surfaced by the same
DIR-015 hardening pass, not because either is small enough to fold into an
existing directive. Both are `contained` scope, `safe` risk, per their
issue files — no urgency signal beyond normal backlog priority.

backlog-pm slotted ISSUE-078 here on 2026-07-26 during a full-corpus sweep;
its Finding 1 (gate-side claim-ID-vs-test-name reconciliation) is noted as
a possible scope seam if a planner judges it belongs with gate
test-verification work instead of this directive's compiler-focused scope.
