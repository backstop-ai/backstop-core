---
name: smoke-darkpack-prefailures
description: tests/smoke has 4 PRE-EXISTING failures (substantiveness/contracts/coverage dark packs) that look like regressions but predate any in-flight spec
metadata:
  type: project
---

`go test ./tests/smoke/` has 4 PRE-EXISTING failures on the bundle/011 branch that
are NOT caused by the spec you're implementing — verified identical at the SPEC-044
gate-green base (6b432ad):

- `TestSmoke_GatePassesOnCompliantProject` (fails on test_substantiveness + contract_signature)
- `TestSmoke_GateFailsHollowTest`
- `TestSmoke_GateFailsCoverageBelowThreshold`
- `TestSmoke_GateFailsContractSignatureMismatch`

**Why:** the smoke fixtures don't install backstop/substantiveness, backstop/contracts,
or a coverage-engine toolchain pack, so those dimensions are `*_capability_absent` warnings
(the dogfood-mostly-dark condition, [[project_gate_dogfood_mostly_dark]]). The dogfood
`./bin/backstop gate` was GREEN with these present, so the gate does not fail on them
(its go test pass is dark / tests/smoke is outside the enforced scope).

**How to apply:** before treating a tests/smoke failure as YOUR regression, diff the
failure SET against a known gate-green base via a throwaway worktree
(`git worktree add /tmp/base <green-commit>; go test ./tests/smoke/`). Only NEW failures
(or new failing assertions WITHIN a still-failing test) are yours. SPEC-045 added a 5th
condition (test_verification capability-absent after deleting the baked _test.go walk +
funcPattern); the fix was installSmokeGoToolchainPack declaring classification.test +
test_name_patterns ONLY (NOT classification.source — source globs make coverage capability
present with no records -> coverage_unmeasured RED). That restored the 4-failure baseline.
