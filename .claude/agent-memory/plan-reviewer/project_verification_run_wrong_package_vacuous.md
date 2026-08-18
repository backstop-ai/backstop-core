---
name: verification-run-wrong-package-vacuous
description: A plan's verification task running `go test <pkg> -run TestName` where TestName lives in a DIFFERENT package exits 0 with "[no tests to run]" — a silent vacuous pass; cross-check every -run name against the task that declares the file
metadata:
  type: project
---

Cross-check every test NAME in a verification task's `go test <pkg> -run ...` command
against the `files:` of the task that CREATES that test. If the name lives in another
package, the command exits 0 printing `ok <pkg> [no tests to run]` — the verification
step passes having run nothing.

Measured on PLAN-ISSUE-176 (2026-08-18): TASK-004 declared
`TestMakefile_DeclaresOptInBaselineTargetOutsideTestAndCI` with `files:` =
`cmd/backstop/baseline_prerequisite_test.go` only, while TASK-006 verified it with
`go test . -run "TestMakefile_DeclaresOptIn"` (ROOT package). Confirmed by running it:
`ok github.com/backstop-ai/backstop-core 0.535s [no tests to run]`, exit 0.

**Why:** plan-level `test_command` often lists BOTH packages (`go test . ./cmd/backstop/`),
so the mismatch is invisible there and only bites in the per-task verification command —
which is the one an implementer actually pastes.

**How to apply:** build the (test name → owning package) map from the test tasks' `files:`,
then read every verification task's `-run` regex against it. Also flag the reverse smell: a
new test whose natural home is the root repo-structure package (`workflows_test.go`,
`module_path_test.go`, `pack_fleet_test.go`, `release_config_test.go` live there) but is
parked in `cmd/backstop` — in a package whose RED is a compile failure, a co-located
structural test cannot show its own specific red. Related: [[project_shortcircuit_dependent_guard]].
