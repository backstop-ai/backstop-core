---
name: per-test-evidence-is-local-only
description: The same "prove each named test passed" demand is correct locally (requires -v) and unexecutable in CI (the gate job publishes no verbose transcript) — restate the CI leg as gate-report.json negatives
metadata:
  type: project
---

"Require test X to pass by name" is TWO different asks depending on where it is read, and a
plan that writes it once for both legs ships one correct task and one unexecutable one.

**LOCAL leg — the demand is right, but needs `-v`.** A bare `go test . -run 'Foo' -count=1`
that matches nothing exits 0 and prints `ok`, so a package-level pass proves nothing about
which tests ran. Prescribe `go test -v` (or `-json`) and require a counted number of
`=== RUN` lines BEFORE reading any pass/fail verdict — the RUN count is the anti-empty-
selection guard, the PASS/FAIL count is the outcome. Demanding "four RUN/PASS lines" without
`-v` in the command is self-contradictory: the lines cannot be emitted.

**CI leg — the same demand is unobtainable.** The Backstop Gate job runs suites through a
pack's go-test engine and does not publish a verbose per-test transcript. A task that says
"require the job's own run of these four tests to pass, and a green job whose log shows them
skipped is not acceptance" cannot be discharged by anyone. Restate it as observable facts:
green job at the exact head SHA, PLUS `gate-report.json` containing no findings-engine
violation locating the test file and no `mandated_test_failed` naming each test (grep each
name explicitly and record "absent" per name). A green job alone is not attribution — the
negative report checks are what convert it into evidence.

**Why:** PLAN-ISSUE-186 review (2026-08-24) raised both halves as separate findings on the
same plan — F3 (add `-v` to the local TASK-013/TASK-014 commands) and F4 (replace TASK-012's
unobtainable CI per-test-log claim). Same sentence, opposite repairs.

**How to apply:** whenever a plan names test functions as acceptance evidence, ask where that
evidence is READ from. Local shell → fix the command. CI job → fix the criterion, and say in
the task why the per-test form was dropped, so a later reader does not "restore" it.

Related: [[project_run_the_command_you_prescribe]] (the `-run` no-match exit-0 trap),
[[project_ci_confirmation_string_must_exist_in_the_log]] (measure criteria against the
artifact they are read from), [[project_platform_gated_defect_verification_ceiling]].
