---
name: sarif-warning-severity-lost
description: RESOLVED for pack_engines (ISSUE-104 hop 1 + ISSUE-105 hop 2, consumer-verified end to end). STILL OPEN elsewhere: substantiveness and contract paths hardcode Severity error before any verdict sees it, so a warning landing there still blocks.
metadata:
  type: project
---

**STATUS: RESOLVED FOR THE pack_engines PATH, 2026-07-29 — both hops, verified on a
real consumer, not just on core's own gate.** Hop 2 fixed by ISSUE-105:
`gate.StepVerdict` (pkg/gate/policy.go:125) now derives every step status through
blocksVerdict, at all four former raw-count sites, WITHOUT needing a policy entry.
Consumer proof: stash's gate went exit 1 -> EXIT 0 / pass=true with the warning
STILL REPORTED, and the A/B probe's no-policy arm flipped fail -> warning. Control
in the same run: a declared-ERROR rule still BLOCKS (exit 1), so the fix did not
flip everything non-blocking — the direction that would have been worse.

**STILL OPEN, AND THE REASON THIS MEMORY STAYS:** the same severity-loss shape lives
in OTHER dimensions — substantiveness_join.go and VerifyContractVerdict hardcode
`Severity: "error"` BEFORE any verdict reads it. "Declared warnings are
non-blocking" is therefore true for pack_engines and NOT yet universal. Check which
DIMENSION a warning lands on before promising a pack author it will not block.
ISSUE-104 landed in core as commit a42b065: pkg/check/parsers.go models
runs[].tool.driver.rules[] and resolves severity as result `level` -> the rule
descriptor's defaultConfiguration.level (joined on ruleId, scoped PER RUN) ->
fail-closed error. sarifSeverity is UNCHANGED — only the set of places a level
is looked for grew. The contract test now runs on CAPTURED semgrep bytes
(cmd/backstop/testdata/semgrep/fixtures/, see
[[project_captured_sarif_fixture_inventory]]), so it can no longer pass on a
shape semgrep never emits. Falsified by control-vs-treatment in a detached
worktree at HEAD: same tests + fixtures, flip parsers.go alone, red -> green.

── HOP 2, THE OPEN HALF (measured on the re-run this memory asked for) ──
The consumer flip did NOT reverse. With a42b065 in place the violation correctly
carries `severity: warning` AND THE GATE STILL FAILED (stash: exit 1, step
status=fail). Severity survives to the violation and is then DISCARDED when the
step's status is computed:
- `cmd/backstop/gate.go`, pack_engines step: `status := "pass"; if
  len(violations) > 0 { status = "fail" }` — a RAW COUNT, severity never read.
- `blocksVerdict`/`blockingViolations` (pkg/gate/policy.go:80-95) have exactly ONE
  non-test caller: policy.go:173 inside applyScopedPolicy.
- `ApplyPolicy` (policy.go:132-136) passes a step through UNTOUCHED when the
  consumer's `enforcement.policy` map has NO ENTRY for that step name.
So a consumer with no `pack_engines` policy entry never reaches the severity-aware
code. PROVEN one config line apart on a byte-identical throwaway tree: no entry ->
status=fail, exit 1; add `pack_engines: {level: block}` -> status=warning, exit 0,
finding STILL REPORTED. backstop-core's own backstop.yml declares pack_engines,
which is precisely why the dogfood never caught it.
THE SITE INVENTORY IS WIDER THAN ONE STEP (verified 2026-07-29 by grep for
`if len(violations) > 0` over pkg/ and cmd/, non-test): cmd/backstop/gate.go:863
(pack_engines, where this was measured) and :1193, pkg/gate/step_delegate.go:58
and :106, pkg/gate/step_contract.go:63. Whoever fixes this should treat it as a
CLASS, not a single site — pkg/check/output.go:109 and
pkg/pack/distribution/command.go:849 match the same text but are reporting/
exit-code paths, so they need reading before touching, not blanket replacement.

THE LESSON THAT GENERALIZES: a severity contract is only as good as its LAST hop.
Fixing the parser proved the value reached the violation, not that it reached the
verdict. Chase severity all the way to the exit code, and test the contract on a
consumer with an EMPTY policy table, not just the policy path.

── ORIGINAL MEASUREMENT ──
Measured 2026-07-29 (PLAN-ISSUE-101, go-distribution pack). A pack rule declaring
semgrep `severity: WARNING` lands in the gate as **severity=error** and BLOCKS.

The chain, all four links verified:
- `pkg/check/parsers.go` `sarifLog` models only `results[].level`; `grep -rn
  defaultConfiguration --include=*.go` over core (non-test) returns NOTHING — the
  SARIF rule descriptor is never read.
- `sarifSeverity()` treats only the literal `"warning"` as non-blocking; an ABSENT
  level returns `"error"`, fail-closed by design.
- Real semgrep 1.156.0 emits the level ONLY on the descriptor:
  `tool.driver.rules[].defaultConfiguration = {level: warning}`, and
  `results[].level` is ABSENT.
- `cmd/backstop/pack_severity_contract_test.go` builds SARIF BY HAND with
  `"level":"warning"` ON THE RESULT — a shape real semgrep never produces. The
  contract test passes while the contract fails in production.

Consequence measured on a real consumer: adopting a pack whose only finding was a
declared-WARNING rule turned stash's gate from `pass=true, exit 0` to
`pass=false, exit 1`. Un-adopted capability BLOCKED, which is what loud-not-blocking
forbids. Not pack-specific — it hits any pack shipping a semgrep WARNING rule.

**Why:** "declare WARNING for un-adopted capability" is a load-bearing pack-design
idiom ([[feedback_loud_not_blocking]]); it silently does not work.

**How to apply (still true, and the reason this memory survives its own fix):**
never trust a declared WARNING to be non-blocking without running the GATE (direct semgrep shows the right severity and hides the bug — the loss
happens in the SARIF hop). Hop 1's fix (descriptor fallback + CAPTURED-SARIF contract test,
[[feedback_fixtures_from_real_output]]) shipped; hop 2 has not. Related:
[[project_verdict_decided_after_the_step]] — same shape, severity decided downstream
of the step that declared it.
