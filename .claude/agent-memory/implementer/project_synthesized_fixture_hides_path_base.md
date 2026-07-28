---
name: synthesized-fixture-hides-path-base
description: A unit test that CREATES its fixture at whatever path the code computed cannot falsify the resolution BASE — only an E2E over a real installed pack can; SPEC-054 phase 7 shipped a doubled recipe segment this way
metadata:
  type: project
---

A test that materializes its fixture at the path the implementation resolved can never
falsify WHICH BASE that path was resolved against. SPEC-054's phase-7 transform tests
did `copyUnder(t, recipeDir, op.Rule, ...)` and asserted
`dispatched.rule == filepath.Join(recipeDir, op.Rule)` — green, while the shipped
applier joined a PACK-relative rule path onto the RECIPE dir and doubled the segment
(`.../recipes/rewrite/recipes/rewrite/rules/...`). The phase-10 E2E over a real
installed pack caught it on the first run (ast-grep exit 6, "Cannot read rule").

**Why:** the spec declared the path pack-relative (SPEC-054 Op contract) but the code
picked a different base; both the impl and its tests were self-consistent, so nothing
in the unit layer could disagree. Fix was `ResolvedRecipe.PackDir` + resolving the rule
under it.

**How to apply:** when a spec says a declared path is "<X>-relative", check the
resolution base in code against a REAL installed layout, not against a t.TempDir the
test just built. Where a claim rides on a path base, the falsifying test must read a
file that already exists at the committed layout. Also: capture the engine's COMBINED
output on a transform dispatch — stdout-only reduced this to a bare "exit status 6".
Related: [[fixtures-from-real-output]], [[pack-provisioning-integration-gap]].
