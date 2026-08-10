---
name: lock-the-chain-falsify-per-hop
description: A contract spanning packages needs ONE test driving the whole chain plus a per-hop falsification matrix — layer tests that hand-build the mid-chain value lock the last hop and nothing upstream
metadata:
  type: project
---

When a contract crosses package boundaries, a test that constructs the intermediate
value by hand locks only the final consumer. pkg/gate/policy_severity_test.go builds
`gate.Violation` values with Severity ALREADY SET, so it covers `blocksVerdict` and
nothing upstream — the pack-facing severity contract actually spans SARIF `level` ->
check.Violation.Severity (check.ParsePackFindings) -> gate.Violation.Severity
(runFindingsEngine's bridge) -> the verdict (gate.ApplyPolicy). Any hop could drop or
rewrite the field with every existing test still green.

The shape that works: drive the REAL production entry point over real wire bytes, then
the REAL consumer. cmd/backstop/pack_severity_contract_test.go feeds SARIF to the
actual `runFindingsEngine` (reusing the package's existing `capturingRunner` stub) and
passes the result to `gate.ApplyPolicy` under the strictest policy a consumer can
declare (block + all-code), so a warning surviving there survives everywhere. Assert
the value at the BRIDGE and the verdict — two failures with one symptom, and asserting
only the verdict leaves a reader unable to tell which hop broke.

**Why:** an end-to-end test that passes proves the chain works TODAY but does not
prove the test would notice a break. The per-hop falsification matrix is what makes it
evidence. Each mutation must be caught by the RIGHT test, not merely by some test:
sarifSeverity mapping non-error->warning (parser hop) -> caught by the absent-level
test; the bridge hardcoding Severity:"error" -> caught by the warning test;
blocksVerdict returning true (the original CI defect) -> caught by the warning test.

**How to apply:** run the matrix in a detached worktree at HEAD, one mutation at a
time with `git checkout --` between, recording control=0 and restored=0 around it. If
a mutation is caught by an unexpected test, the assertions are entangled and the
matrix has told you something. Look for an existing runner/stub in the target package
before writing one. Related: [[project_absence_tests_via_goast]],
[[feedback_never_stash_shared_tree]], [[project_verdict_decided_after_the_step]].
