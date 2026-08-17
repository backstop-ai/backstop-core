---
name: shared-sgconfig-collapses-fixture-polarity
description: When a pack's rules share one config file, packval's per-fixture verdict is "did ANY rule fire" — so a positive fixture must be clean against the WHOLE ruleset, and swapping manifest keys fixes nothing
metadata:
  type: project
---

`pkg/packval/phase3.go` dispatches `rule.RuleSourcePath()` per fixture. When several
rules declare the SAME `rule_path` (e.g. `packs/substantiveness`'s two rules both
point at `ast-grep/sgconfig.yml`, whose `ruleDirs` loads both), the engine runs the
COMBINED ruleset. `Passed` therefore means "the engine fired", not "this rule fired".

Consequences measured on ISSUE-148 (2026-08-17, ast-grep 0.43.0):
* A declared POSITIVE (clean) fixture must trigger ZERO rules, not just zero hits of
  its own rule.
* A declared NEGATIVE fixture need only trigger ONE rule — ANY rule — so it can pass
  on a SIBLING rule's hit while its own rule stays silent. That is a vacuous green
  hiding inside a fixture pair that looks correct.
* Swapping the `positive:`/`negative:` manifest keys changes the failure NOT AT ALL.
  Swapping the file CONTENTS also fails, because a substantive Go test asserts with
  `t.Fatalf`, which is a `selector_expression` inside a `^Test`-named function —
  exactly what an extraction rule like `referenced-symbol-go` matches.

**Why:** ISSUE-148 and DIR-032 item 19 both stated the fix as "purely which manifest
key each is filed under, swap either the keys or the contents". Both remedies were
falsified by measurement; the real fix was re-authoring all four fixture BODIES with
`pack.yml` untouched. Filed as ISSUE-159.

**How to apply:** Before believing any fixture-polarity diagnosis, check whether the
rules share a `rule_path`. If they do, evaluate each fixture against the combined
config (`ast-grep scan --config <sgconfig> --json <fixture>`), not against the one
rule it is filed under. To make a clean fixture clean: assert with an UNQUALIFIED
call whose identifier matches the hollow rule's vocabulary (`mustEqual`), and push
any `t.Fatalf` into a non-`^Test`-named helper — `inside: {matches: is-test}` never
reaches into one. See [[project_selfpack_b2_token_rule_scope]] for the related
no-baked-tool-exec trap a new `cmd/backstop` test file hits.

A negative-fixture assertion of "own rule id fired" is NOT discriminating on its own
here: discard findings whose extracted symbol metavariable is the `*testing.T`
receiver `t` first, or the assertion is satisfied by boilerplate.
