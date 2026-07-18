---
title: "ast-grep to-sarif.sh Convert Crashes on Null ruleId Under jq 1.7+/1.8"
schema_version: issue/v1

issue:
  id: ISSUE-069
  title: "ast-grep to-sarif.sh Convert Crashes on Null ruleId Under jq 1.7+/1.8"
  type: bug
  status: open
  created: "2026-07-18"

complexity:
  scope: contained
  uncertainty: known
  risk: critical
---

# ast-grep to-sarif.sh Convert Crashes on Null ruleId Under jq 1.7+/1.8

## Problem

The ast-grep→SARIF convert scripts route substantiveness roles with an unguarded
`(.ruleId | test("hollow"))` / `(.ruleId | test("referenced-symbol"))` jq expression. jq 1.7+ made
`test()` on a null value a hard error (`null (null) cannot be matched, as it is not a string`)
instead of the older lenient behavior. When ast-grep emits a finding with a null or absent
`ruleId`, the convert CRASHES (exit 5) instead of routing it to a default role — which reds the
gate.

Surfaced 2026-07-18 while verifying ISSUE-068 (gate double-run) on jq 1.8.1, but this defect is
PRE-EXISTING and unrelated to ISSUE-068. It was introduced by commit c40c99f (ISSUE-064, "route
findings + label stacks by declaration, not baked names"), which added role-routing via
`.ruleId | test(...)` in the ast-grep substantiveness convert scripts.

Confirmed: `go test ./pkg/pack/engine/ -run TestTSPack_ContractSignaturePresenceAstGrep` FAILS on
committed HEAD (independent of ISSUE-068's changes), jq 1.8.1. The gate is currently RED on
backstop-core because of this.

## Blast radius

Unguarded `.ruleId | test(` in-repo:

- `packs/substantiveness/ast-grep/to-sarif.sh` (lines ~45-46) — the real in-repo dogfood
  substantiveness pack.
- `pkg/gate/testdata/substantiveness-pack/ast-grep/to-sarif.sh` (~39-40) — testdata fixture.
- `pkg/gate/testdata/ts-proof-pack/ast-grep/to-sarif.sh` (~29-30) — testdata fixture (the one
  failing `TestTSPack_ContractSignaturePresenceAstGrep`).

**Cross-repo follow-on (flagged, not scoped here):** the same unguarded pattern was applied by
ISSUE-062/064 to the shipped `backstop-packs` TypeScript convert scripts
(`typescript-standards` / substantiveness packs). Those live in the separate `backstop-packs`
repo and would break live TS gate runs on jq 1.8.x — a real risk to bclabs-portal, the first TS
consumer. A corresponding issue must be filed in that repo; not addressed by this issue.

## Root cause

jq's `test()` builtin, as of jq 1.7, raises an error when its input is `null` rather than
matching nothing. The convert scripts assume `.ruleId` is always a string and pipe it directly
into `test(...)` without coercing a null/absent value first. Any ast-grep finding without a
`ruleId` (or with an explicit `null`) therefore aborts the whole convert instead of falling
through to the default role.

## Fix direction

Null-guard the field before `test()`, e.g. `((.ruleId // "") | test("hollow"))` — coerce null to
empty string so an unrecognized/absent `ruleId` falls through the if/elif chain to the default
role instead of crashing. Raw and minimal: apply the same guard to all three in-repo convert
scripts listed under Blast radius. Convert scripts must be robust to null/absent fields in tool
output in general, not just for this one field.

## Acceptance

- `TestTSPack_ContractSignaturePresenceAstGrep` passes.
- All three in-repo convert scripts (`packs/substantiveness/ast-grep/to-sarif.sh`,
  `pkg/gate/testdata/substantiveness-pack/ast-grep/to-sarif.sh`,
  `pkg/gate/testdata/ts-proof-pack/ast-grep/to-sarif.sh`) handle a null/absent `ruleId` without
  crashing, falling through to default routing.
- `backstop gate` is green on backstop-core again.
- No change to role-routing behavior for findings that DO have a matching `ruleId` — pure
  robustness fix, not a behavior change.
- A cross-repo follow-on issue is filed/flagged for the backstop-packs TS convert scripts
  (`typescript-standards` / substantiveness packs) carrying the same unguarded pattern.

## Impact

Unblocks a green gate on backstop-core. Prevents live TS gate crashes on jq 1.8.x for
consumers running newer jq — directly relevant to bclabs-portal, the first TS consumer, which
depends on backstop-packs' TS convert scripts carrying the same defect.

## Notes / references

- Introduced by commit c40c99f (ISSUE-064, "route findings + label stacks by declaration, not
  baked names").
- Surfaced while verifying ISSUE-068 (gate double-run) — unrelated defect discovered
  incidentally on jq 1.8.1.
- See [[project_typescript_packs]] for the shipped backstop-packs TS pack suite that shares the
  vulnerable pattern cross-repo.
