---
name: packval-verdict-is-whole-ruleset
description: packval phase3's per-fixture verdict is "did ANY rule in the config fire", so for a multi-rule config-file pack a positive fixture must be clean against EVERY rule — a polarity swap alone never fixes an inverted pair
metadata:
  type: project
---

`pkg/packval/phase3.go` calls `executor.RunEngine(packDir, binding, []string{ruleSource,
f.Path})` and reads `r.Passed`, which means **"the engine produced findings"**, not "this
rule fired". For a `config-file` engine whose `rule_path` is a PROJECT config
(`ast-grep/sgconfig.yml` with `ruleDirs:`), that config loads **every** rule in the pack.

**Consequences, all measured on `packs/substantiveness` 2026-08-17 (ISSUE-148):**

- A declared POSITIVE (clean) fixture must trigger **zero** rules of the whole config.
  Swapping which file is filed under `positive:`/`negative:` — or swapping the file
  contents — leaves the failure byte-for-byte unchanged, because a substantive Go test
  asserts with `t.Fatalf`, and `t.Fatalf` is a `selector_expression` that the pack's own
  Q2 `referenced-symbol-go` rule matches inside any `^Test`-named function.
- A declared NEGATIVE fixture can pass **for the wrong rule**. Both of that pack's
  negatives passed on a sibling rule's `t.Fatalf` hit, never on their own rule — a vacuous
  green sitting underneath the filed defect and invisible to packval, which never checks
  rule-id attribution.
- The working fix is to re-author BODIES so each fixture is clean/violating against the
  COMBINED ruleset: the clean fixture's assertion must be an **unqualified** call whose
  identifier matches the hollow rule's assertion vocabulary (`mustEqual(...)`), and helper
  funcs may hold `t.Fatalf` freely because `inside: matches: is-test` never reaches into a
  non-`Test`-named declaration.

**Why:** an issue filed from a body-level reading will describe this as "the manifest keys
are swapped" and offer a one-line fix menu. That menu is falsifiable in about two minutes
and was false here — the issue AND the directive item both carried it.

**How to apply:** before planning any pack-fixture polarity fix, run
`ast-grep scan --config <pack>/<rule_path> --json <each fixture>` and record which rule ids
fire on which fixture. Then run the counterfactual the issue proposes and show it still
fails. Mandate a test that asserts negative fixtures fire **their own rule id**, not just
"at least one finding" — packval will never catch that for you. Related:
[[project_planning_a_pack_data_fix]], [[feedback_verify_issue_premises]],
[[project_dead_dispatch_fix_reds_its_consumer]].
