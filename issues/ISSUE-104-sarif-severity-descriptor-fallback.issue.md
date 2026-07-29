---
title: "Sarif Severity Descriptor Fallback"
schema_version: issue/v1

issue:
  id: ISSUE-104
  title: "Sarif Severity Descriptor Fallback"
  type: bug
  status: closed
  created: "2026-07-28"
  closed: "2026-07-29"

complexity:
  scope: contained
  uncertainty: known
  risk: safe

delivered_by: PLAN-ISSUE-104
---

# Sarif Severity Descriptor Fallback

## Resolution

Delivered by PLAN-ISSUE-104 (status: completed), commit `a42b065`. `parseSarif` now resolves each
finding's severity input in order: `results[].level` if present (the golangci shape, locked as a
non-regression against the committed real `golangci-v2.sarif` fixture), else
`tool.driver.rules[].defaultConfiguration.level` joined to the result per-run by `ruleId` (the real
semgrep shape), else absent — which still falls through to the unchanged fail-closed `"error"`
default. Result-level `level` keeps precedence over the descriptor fallback so a hypothetical
future semgrep emission that does put `level` on the result continues to work unchanged.

**Fixtures re-captured from real output, not hand-built.** `pack_severity_contract_test.go`'s SARIF
inputs were re-fixtured onto CAPTURED semgrep output at both 1.156.0 and 1.96.0 (the version
`pkg/pack/engine/allowlist.go` actually pins), closing the open verification item from this issue's
Solution section and proving the fix is not a version-specific artifact — both captures produce the
same descriptor-fallback behavior. Hand-built SARIF shapes were kept only where they earn their
place as parser-unit-level shape tests distinct from the pack-author-contract test, per the fix
shape's own carve-out.

**Red-proof twice-verified.** Both descriptor tests (`DescriptorLevelSuppliesSeverity...`,
`DescriptorJoinIsPerRunAndByRuleID`) were independently confirmed to fail at HEAD before the fix and
pass after, by the implementer's worktree flip and again by the orchestrator's own
control-vs-treatment run — not asserted from a single measurement.

**Cross-lane acceptance met 2026-07-29.** This fix alone was necessary-but-insufficient: the stash
consumer's gate reached the correct severity value but still blocked until ISSUE-105 (step verdicts
honoring severity without a policy entry) landed as well. The TASK-013 re-run against the stash
consumer, with both hops in place, confirms the full chain: exit 0, exactly one non-blocking warning
reported, and the error-direction control still blocks correctly.

**Restores the founder-ratified pack-severity contract in production** — the defect that flipped an
adopting consumer's green gate red purely from a pack's own `severity: warning` advisory rule
(measured on the stash, PLAN-ISSUE-101 TASK-013), for the layer this issue owns (the SARIF parse).
See ISSUE-105 for the companion defect at the step-verdict layer, hop 2 of the same chain.

## Problem

`pkg/check/parsers.go`'s `sarifLog` model reads severity from exactly one place —
`results[].level` (`parsers.go:55`, `sarifSeverity(r.Level)` at `parsers.go:131`) — and never
reads the SARIF rule descriptor's `defaultConfiguration.level`. `grep -rn defaultConfiguration`
over non-test core returns nothing: the fallback source real semgrep uses is not modeled at all.

`sarifSeverity` (`parsers.go:165-170`) maps the literal string `"warning"` to non-blocking and
everything else — including an absent `level` — to `"error"`. That fail-closed default is correct
on its own; the defect is upstream of it, in what value `sarifSeverity` is ever handed.

**Real semgrep does not put `level` on the result.** Verified independently against the semgrep
binary actually installed in this environment (1.156.0 — same version implementer-101 measured):

```
$ semgrep --config rule.yml --sarif sample.go > out.sarif
$ python3 -c "import json; d=json.load(open('out.sarif')); run=d['runs'][0]; \
  print('result keys:', list(run['results'][0].keys())); \
  print('rule defaultConfiguration:', run['tool']['driver']['rules'][0]['defaultConfiguration'])"
result keys: ['fingerprints', 'locations', 'message', 'properties', 'ruleId']
rule defaultConfiguration: {'level': 'warning'}
```

`results[].level` is not merely a different value — the key is **absent** from the result object
entirely. The severity a pack author declares (`severity: WARNING` in the semgrep rule YAML) is
only reachable via `tool.driver.rules[].defaultConfiguration.level`, joined on `ruleId`. Because
`sarifLog` never models `runs[].tool.driver.rules`, `parseSarif` has no way to see it, `r.Level` is
always the zero value `""` for real semgrep output, and `sarifSeverity("")` falls through to
`"error"` every time — regardless of what the pack author declared.

**Consequence:** every semgrep-based pack finding lands at `Severity: "error"` (blocking)
unconditionally, defeating the founder-ratified pack-author contract (2026-07-28, locked in
`cmd/backstop/pack_severity_contract_test.go`, commit `3ac6e7f`, itself landed to close
ISSUE-100's split-off verdict-defect half): *"a SARIF `level: warning` from ANY pack is
NON-BLOCKING, by contract... severity IS how a pack author declares blockingness."* That contract
holds correctly at the two downstream hops it was locked against — `check.Violation.Severity` →
`gate.Violation.Severity` bridge, and `blocksVerdict`'s policy-layer honoring of `Severity` — but is
defeated one hop earlier, at the SARIF parse itself, before either downstream hop ever sees a
value that could be `"warning"`.

**Measured, not theoretical.** implementer-101, during PLAN-ISSUE-101's stash-consumption proof
(2026-07-29, four-link chain independently verified above), ran a control-vs-treatment: the stash
consumer's gate was green before adopting the go-distribution pack, and turned RED after adopting
it — solely because a rule the pack declares at `level: warning` (`tap-token-not-exported`, an
un-adopted-capability advisory) reached the gate as `severity: error`. Adopting a pack turning a
green consumer red for an un-adopted capability is exactly the failure mode "loud ≠ blocking"
(CLAUDE.md) forbids. This is not specific to the go-distribution pack — any pack that ships a
semgrep rule at `level: warning` blocks today, unconditionally.

**Why the existing contract test didn't catch it.** `pack_severity_contract_test.go`'s
`packSarifAtLevel` (`parsers.go` neighbor file, lines ~38-48) builds its SARIF *by hand*, writing
`"level":"warning"` directly onto the result object — a shape real semgrep never produces (see the
captured output above: the result has no `level` key at all). The test locks the two downstream
hops correctly but starts from an input shape the parser will never actually receive from semgrep,
so it cannot see this gap. This is a second instance of the fixtures-from-real-output law
(`feedback_fixtures_from_real_output`) being violated by a hand-built SARIF fixture.

## Impact

Any pack — not just go-distribution — that declares a semgrep rule at `level: warning` to mean
"advisory, non-blocking" currently has that declaration silently discarded: the finding reaches
the gate at `severity: error` and blocks. The founder-ratified severity contract is correct in the
two layers it was tested against and inverted in the layer that was not. This blocks
PLAN-ISSUE-101's TASK-013/014/015 (go-distribution pack v0.1.0 tag is held rather than shipped,
because shipping as-is would red every adopting consumer purely from the pack's own advisory
rules) — CLM-013/015/019/025 there are unsatisfiable until this lands.

## Solution

**Fix shape (small, local, in `pkg/check/parsers.go`):**

1. Extend `sarifLog` to also model `runs[].tool.driver.rules[].id` and
   `runs[].tool.driver.rules[].defaultConfiguration.level`.
2. In `parseSarif`, build a `ruleId -> defaultConfiguration.level` map per run before iterating
   results.
3. Resolve each finding's severity input as: `r.Level` if non-empty, else the map lookup for
   `r.RuleID`, else `""` — then hand that resolved string to `sarifSeverity` unchanged. The
   fail-closed default (absent input → `"error"`) is preserved exactly; only the fallback SOURCE
   changes, not the default.

**Fixture fix (required alongside, not optional):** re-capture `pack_severity_contract_test.go`'s
SARIF inputs from real captured semgrep output (`semgrep --config <rule> --sarif <target>`,
committed with provenance — version, command, date) instead of hand-building the JSON. Keep the
hand-built shapes only if they earn their place as parser-unit-level shape tests distinct from the
pack-author-contract test.

**Open verification item for the plan:** whether pinned semgrep 1.96.0 (`pkg/pack/engine/
allowlist.go:22`, the version backstop actually provisions) differs in emission shape from the
1.156.0 measured here and by implementer-101 — unverified, 1.96.0 wasn't available to run in this
environment either. The descriptor-fallback fix is robust to both shapes (it still reads
`results[].level` first, so a hypothetical future semgrep that DOES emit result-level `level`
keeps working unchanged), but capturing 1.96.0's real output as a second fixture, if obtainable, is
worth doing before closing this out.

## Notes / references

- Severity ratification record: `PLAN-ISSUE-020` (linux-sandbox-gate-in-ci) — the plan lane that
  landed the fix closing ISSUE-100's split-off verdict-defect half (`pkg/gate/policy.go`
  `blocksVerdict` honoring `Violation.Severity`) alongside `cmd/backstop/
  pack_severity_contract_test.go` (commit `3ac6e7f`), which is the test this issue's fixture-fix
  half re-captures.
- Sibling issue: `ISSUE-100` (step-tally counts warnings as violations) — same severity-contract
  neighborhood, different layer (renderer/policy vs. parser).
- Blocks: `PLAN-ISSUE-101` (go-distribution pack) TASK-013/014/015; CLM-013/015/019/025 there are
  unsatisfiable until this closes. v0.1.0 tag held pending this fix.
- Law violated a second time: `feedback_fixtures_from_real_output` — fixtures must be captured
  from real tool output, never fabricated, and must be able to falsify.
- Discovered by: implementer-101, PLAN-ISSUE-101 stash-consumption proof, 2026-07-29.
