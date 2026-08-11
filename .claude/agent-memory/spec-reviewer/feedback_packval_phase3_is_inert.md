---
name: packval-phase3-is-inert
description: packval phase 3 executes ZERO fixtures for every real pack (rule_path vs file key mismatch) — any spec claiming "pack test proves the rule fires" is vacuous
metadata:
  type: feedback
---

`backstop pack test` phase-3 fixture EXECUTION is inert across the entire fleet. Never accept a
spec claim of the form "the pack clears real `pack test`, so every rule is proven to fire on its
negative fixture" without checking this.

**Why:** `pkg/packval/manifest.go` `Rule.File` binds yaml key `file:`. The production parser
`pkg/pack/manifest.go:144` `Rule.RulePath` binds `rule_path:` — and EVERY real pack
(backstop-self, go-distribution, go-substantiveness, cobra-cli, core-architecture) declares
`rule_path:` only. So `rule.File == ""` and `pkg/packval/phase3.go:32,63,77` skip fixture
execution entirely. Measured 2026-08-10: gutted every violating fixture in a backstop-self-pack
copy → `pack test` still reported `phase3-fixtures: pass`. Adding `file:` alongside `rule_path:`
turned execution ON and immediately produced 13 errors.

**Second trap, exposed by that same experiment — polarity is INVERTED.** phase3 requires
`Fixtures.Positive` to FIRE (`!r.Passed` → "positive fixture failed") and `Fixtures.Negative`
to NOT fire (`r.Passed` → "negative fixture not triggered"). Every pack declares the opposite
(positive → `fixtures/rules/valid/*` clean, negative → `fixtures/rules/invalid/*` violating).
The mismatch is masked today only because execution never runs.

**How to apply:** when a spec leans on phase 3 — to justify a fixture NAMING convention, a
`paths:` include shape, or a "the rule provably fires" claim — demand either (a) the rules
declare `file:` as well as `rule_path:`, plus the polarity packval actually executes, or (b) the
claim be restated as something a real command verifies. Also reject any spec prose citing the
error string "negative fixture not triggered" as the consequence of a fixture being FILTERED OUT:
a filtered fixture yields zero findings, which is the negative branch's PASS condition.
See [[project_spec067_review2]].
