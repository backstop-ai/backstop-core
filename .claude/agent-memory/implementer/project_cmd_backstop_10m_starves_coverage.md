---
name: cmd-backstop-10m-starves-coverage
description: cmd/backstop now exceeds go test's 10-minute DEFAULT timeout in a contended shared tree, which starves the gate's coverage producer so every lane reds on coverage_unmeasured for its own changed files
metadata:
  type: project
---

`cmd/backstop` takes 487-606s and, at go test's **10-minute default timeout**, panics
mid-run under multi-lane contention. Because the gate's coverage producer runs a
whole-repo `go test`, that panic means **no coverage profile reaches the gate for ANY
file** — so `coverage_threshold` reports `coverage_unmeasured` on your changed files
even when their real coverage is fine.

Measured 2026-08-17 (PLAN-ISSUE-142 lane, 9 concurrent lanes):
- diff-scoped gate: only blocking failure was 2x `coverage_unmeasured` on
  `pkg/packval/manifest.go` + `phase3.go`.
- direct `go test ./pkg/packval/ -coverprofile -timeout 25m` → **95.1%** (threshold 90),
  `RuleDispatchInput` 100%, `RunFixtures` 90.8%. No gap at all.
- the gate's coverage step burned **621258ms (10.35 min)** — the same 10-minute wall.
- `go test ./cmd/backstop/...` at default timeout: 605.934s then
  `panic: test timed out after 10m0s`.

**Why:** the gate cannot be told a longer timeout from your lane, and every added
sibling test pushes `cmd/backstop` further past the wall.

**How to apply:** when a diff-scoped gate's ONLY red is `coverage_unmeasured` on files
you changed, do NOT chase coverage and do NOT waive. Re-measure the package alone with
`-timeout 25m` and cite the real percentage. Run whole-repo suites with an explicit
`-timeout 40m` or the reading is truncated, not failing. Related:
[[project_gate_contention_in_shared_tree]], [[project_red_tdd_state_poisons_package_coverage]],
[[project_long_suite_samples_a_moving_tree]], [[project_background_wrapper_exit_code_lies]].
