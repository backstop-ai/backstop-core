---
name: packtest-phase3-vacuous
description: pack test phase3-fixtures does NOT verify fixtures falsify — a negative fixture made fully compliant still passes; prove rule behavior by running semgrep directly on explicit file targets
metadata:
  type: project
---

Measured 2026-07-27 on backstop-ai/go-standards: I rewrote
`fixtures/rules/invalid/go-010-ignored-error.go` to be fully COMPLIANT (no
violation left in it at all) and `backstop pack test .` still reported
`phase3-fixtures: pass`, exit 0. Removing a rule's `paths: exclude` likewise did
not fail any fixture. **phase3 does not execute fixtures against their rules in a
way that can fail.**

**Why:** "run `pack check` + `pack test` green" is NOT evidence that a rule change
behaves as intended, and a fixture added to satisfy the pack evidence bar can be
completely hollow while the pack reports green.

**How to apply:** prove rule behavior by invoking the engine directly on explicit
file targets, the way the gate's diff scope does:
`semgrep --config <rulefile> --json --quiet --no-git-ignore <file>` and count
`results[].check_id`. Run it three ways — negative fixture (must fire), positive
fixture (must not), and the mutated rule (must flip). Two traps found doing this:
  - semgrep's DEFAULT ignores skip `*_test.go` when you pass a DIRECTORY, so a
    directory scan shows 0 test-file findings while the gate (explicit per-file
    targets) shows dozens. Always pass explicit files when replicating the gate.
  - blanking a YAML list line with `sed s|...||` leaves an empty item →
    `InvalidRuleSchemaError: null values prohibited` → semgrep loads nothing and
    returns 0 results, which reads as a passing falsification. DELETE the line
    (python string replace) and assert `errors == 0` in the JSON.

Related: [[project_hermetic_pack_fixture_recipe]],
[[project_gate_all_underreports_vs_diff]], [[feedback_fixtures_from_real_output]].
