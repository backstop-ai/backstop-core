---
name: minimal-valid-pack-fixture
description: The exact minimal pack.yml that passes both `pack check` and `pack test`; hand-rolled 4-line test fixtures break the moment a CLI path validates
metadata:
  type: project
---

A hand-rolled fixture pack of the shape older cmd/backstop tests use —
`name` + `archetype` + `content.ruleset.rules[].id` — FAILS `pack check`.
The minimal source that passes BOTH check and test is:

```yaml
name: <org>/<pack>
version: 1.0.0
language: neutral
archetype: enforcement
description: ...
engines:
  <engine-name>:
    command: ""
    input_mode: rule-flags
    input_flag: "--config"
    scope_kind: file-args
    gate_type: findings
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: <rule-id>
        engine: <engine-name>
        file: rules/<rule-id>.yml
        risk_class: correctness
```

plus `rules/<rule-id>.yml` whose body declares the SAME `id` (phase3-fixtures
checks rule-file identity). The failure ladder if you omit pieces:
phase1 `version is required` / `language is required` / `missing risk_class`,
then phase2 `rule has no claims` — cleared not by adding claims but by
declaring `engine:` (a claimless rule is exempt when its engine RESOLVES),
and `command: ""` keeps it hermetic (fails loud if anything ever executes it).

**Why:** SPEC-055 made pack validation UNCONDITIONAL on the production
`pack add` path. Seven cmd/backstop tests that had built toy packs went red
at once — not a regression, the validator finally running. See
[[hermetic-pack-fixture-recipe]].

**How to apply:** before wiring a validator into any path a test fixture
flows through, grep for hand-rolled `pack.yml` string literals in that
package's tests and budget for upgrading them. Reuse
`cmd/backstop/testdata/hermetic-remote/valid-pack` rather than authoring
another one.
