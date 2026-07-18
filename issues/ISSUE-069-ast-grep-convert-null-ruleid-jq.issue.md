---
title: "ast-grep to-sarif.sh Convert Crashes on Null ruleId Under jq 1.7+/1.8"
schema_version: issue/v1

issue:
  id: ISSUE-069
  title: "ast-grep to-sarif.sh Convert Crashes on Null ruleId Under jq 1.7+/1.8"
  type: bug
  status: closed
  created: "2026-07-18"
  closed: "2026-07-18"

resolved-by: 031b351

complexity:
  scope: contained
  uncertainty: known
  risk: critical

verification:
  level: unit
  coverage_threshold: 90
  test_command: "go test ./pkg/gate/... ./pkg/pack/engine/..."

implementation:
  summary: >
    Null-guarded `.ruleId` before every `test()` call in the three in-repo ast-grep
    convert scripts (`(.ruleId // "") | test(...)`), coercing a null/absent ruleId to
    an empty string so an unroutable finding falls through the if/elif role-routing
    chain to the default (no role) instead of aborting the jq program. Applied
    identically to `packs/substantiveness/ast-grep/to-sarif.sh` (the durable dogfood
    source), `pkg/gate/testdata/substantiveness-pack/ast-grep/to-sarif.sh`, and
    `pkg/gate/testdata/ts-proof-pack/ast-grep/to-sarif.sh`. Landed in 031b351. The
    `backstop/substantiveness` pack was bumped to v1.1.0 and reinstalled/relocked so
    the gitignored installed copy the real gate executes carries the fix
    (`backstop.lock`'s `backstop/substantiveness` entry content_hash updated,
    install_date 2026-07-18T20:58:43Z). No change to role-routing behavior for
    findings that carry a matching ruleId.
  package: pkg/gate

requirements:
  - id: REQ-001
    text: >
      An ast-grep finding with an explicit `null` ruleId, fed through any of the
      three in-repo convert scripts, must not crash the convert (non-zero exit) —
      it must emit valid SARIF and route to no substantiveness_role.
  - id: REQ-002
    text: >
      An ast-grep finding with `ruleId` entirely absent (key omitted, not just
      null), fed through the same convert, must likewise not crash and must route
      to no substantiveness_role.
  - id: REQ-003
    text: >
      A finding whose ruleId DOES match an existing role pattern (e.g. contains
      "hollow" or "referenced-symbol") must still route to that role after the
      null-guard change — the fix must not alter routing behavior for findings
      that carry a real ruleId.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: >
      Feeding the real `packs/substantiveness/ast-grep/to-sarif.sh` convert a
      captured ast-grep payload containing a finding with `ruleId: null` produces
      valid SARIF with no crash and no substantiveness_role on that result.
    tests:
      - TestSubstantivenessConvert_NullRuleId_NoCrash
  - id: CLM-002
    requirement: REQ-002
    text: >
      The same convert, fed a finding omitting the `ruleId` key entirely, produces
      valid SARIF with no crash and no substantiveness_role on that result.
    tests:
      - TestSubstantivenessConvert_NullRuleId_NoCrash
  - id: CLM-003
    requirement: REQ-003
    text: >
      TestTSPack_ContractSignaturePresenceAstGrep, which exercises the
      ts-proof-pack testdata convert's role-routing under jq 1.8.1, passes —
      proving the null-guard did not disturb routing for findings with a real
      ruleId.
    tests:
      - TestTSPack_ContractSignaturePresenceAstGrep
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

**Cross-repo check (verified 2026-07-18, no follow-on needed):** a suspected cross-repo instance
of this pattern in the shipped `backstop-packs` TypeScript convert scripts was checked and found
NOT to exist. `grep -rn '.ruleId | test(' backstop-packs` returns zero matches — the TS converts
never adopted the unguarded `.ruleId | test(...)` role-routing idiom in the first place. They
either null-guard ruleId (`ruleId: (.ruleId // "eslint")` in
`typescript-toolchain/scripts/lint-to-sarif.sh`, `("pnpm-audit/" + ($a.severity // "unknown"))`
in `audit-to-sarif.sh`) or hardcode a literal ruleId (`"vitest"`). The crash reproduced here is
Go-substantiveness-specific — introduced by ISSUE-064's role-routing change to the ast-grep
converts — and was never shipped to the TS pack suite. No cross-repo issue is filed because there
is nothing to fix there.

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
- A regression test (`TestSubstantivenessConvert_NullRuleId_NoCrash`) pins the null/absent-ruleId
  behavior against the real convert script so this cannot silently regress.

## Impact

Unblocks a green gate on backstop-core. The suspected cross-repo TS risk to bclabs-portal (the
first TS consumer) was checked and does not exist — see Resolution.

## Resolution

Fixed directly (no issue→plan track) in commit 031b351, `fix(ISSUE-069): null-guard .ruleId |
test() in ast-grep converts (jq 1.8 crash)`.

- Null-guarded `.ruleId` before every `test()` call — `(.ruleId // "") | test(...)` — in all
  three in-repo convert scripts named under Blast radius. A null/absent ruleId now coerces to
  `""`, fails every role pattern match, and falls through the if/elif chain to the default (no
  role) instead of aborting the jq program mid-array.
- Added `TestSubstantivenessConvert_NullRuleId_NoCrash`
  (`pkg/gate/substantiveness_convert_test.go`) as a regression guard: feeds the real
  `packs/substantiveness/ast-grep/to-sarif.sh` convert two adversarial findings (one explicit
  `null` ruleId, one with the key omitted entirely) and asserts no crash, valid SARIF, and no
  `substantiveness_role` on either result.
- Bumped `backstop/substantiveness` to pack version 1.1.0 (`packs/substantiveness/pack.yml`) and
  reinstalled it so the gitignored installed copy the real gate executes carries the fix;
  `backstop.lock`'s `backstop/substantiveness` entry content_hash was recomputed
  (install_date 2026-07-18T20:58:43Z).
- Investigated and closed the suspected cross-repo follow-on: `backstop-packs` was checked
  (`grep -rn '.ruleId | test(' `) and carries ZERO instances of the unguarded pattern — its TS
  convert scripts null-guard or hardcode ruleId already (see Blast radius note above). The
  original "cross-repo follow-on" framing in this issue's first draft was incorrect; no follow-on
  issue is needed or filed.

**Accepted residual:** none — all three in-repo convert scripts, the regression test, and the
pack relock are in place; the gate's `test_substantiveness` step is green on backstop-core.

## Verification

- `go test ./pkg/gate/... -run TestSubstantivenessConvert_NullRuleId_NoCrash` — green (CLM-001,
  CLM-002).
- `go test ./pkg/pack/engine/ -run TestTSPack_ContractSignaturePresenceAstGrep` — green (CLM-003),
  confirming the null-guard did not alter routing for findings that carry a real ruleId.
- `go test ./pkg/gate/...` — green.
- `./bin/backstop artifact validate issues/ISSUE-069-ast-grep-convert-null-ruleid-jq.issue.md` —
  schema-valid.

## Notes / references

- Introduced by commit c40c99f (ISSUE-064, "route findings + label stacks by declaration, not
  baked names").
- Surfaced while verifying ISSUE-068 (gate double-run) — unrelated defect discovered
  incidentally on jq 1.8.1.
- Fixed by commit 031b351.
- See [[project_typescript_packs]] for the shipped backstop-packs TS pack suite — checked during
  resolution and confirmed to NOT carry this defect.
