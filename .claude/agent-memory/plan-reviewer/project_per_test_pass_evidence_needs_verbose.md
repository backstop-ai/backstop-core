---
name: per-test-pass-evidence-needs-verbose
description: "Plan tasks demanding 'confirm these N tests PASS by name / were not skipped or unselected' are unobtainable — bare `go test -run` prints no RUN/PASS lines, and the CI gate job never prints per-test output at all"
metadata:
  type: project
---

A verification task that says "require the four tests to pass BY NAME", "confirm four
RUN/PASS lines, not an empty selection that exits 0", or "a green job whose log shows
those tests skipped is not acceptance evidence" is prescribing evidence its own command
cannot produce. Measure it before accepting:

- **Local:** `go test . -run 'TestFoo' -count=1` on a green package prints exactly
  `ok  <pkg>  0.570s` — no `=== RUN`, no `--- PASS`. Only FAILING tests print named
  lines. So the anti-vacuity guard the plan is reaching for (proving the `-run` regex
  actually SELECTED something) needs `-v` or `-json` prescribed explicitly.
- **CI:** in backstop-core the Backstop Gate job's only test execution is
  `./bin/backstop gate --base ... --json-out gate-report.json`
  (`.github/workflows/ci.yml`). The go-toolchain pack runs `go test -coverprofile=cover.out`
  via `scripts/test-produce.sh` — no `-v` — and its stdout is CONSUMED by
  `test-to-sarif.sh`. Per-test lines never reach the job log at all.
  The observable Linux falsifier is the NEGATIVE: `gate-report.json` carries no `go-test`
  violation naming the test file and no `mandated_test_failed` for those names, with the
  job green. That IS meaningful (go-test is scope-filter exempt and emits one located
  violation per `--- FAIL:` — see [[project_gotest_sarif_one_violation_per_test]]).

**Why:** ISSUE-186's TASK-012/TASK-014 (2026-08-24) both demanded per-name Linux PASS
observation; an implementer would either stop, or "confirm" something they never saw.

**How to apply:** whenever a task's acceptance names specific test functions that must be
seen passing, run the exact prescribed command on a green target and read the bytes. If
the names don't appear, it's a finding: demand `-v`/`-json` locally and an
absence-of-violation predicate for CI. Related: [[project_verification_run_wrong_package_vacuous]].
