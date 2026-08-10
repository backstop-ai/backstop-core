---
name: pack-recipes-archetype-gate-order
description: A pack declaring archetype recipes FAILS `pack check` (phase4 recipes-required ERROR) until the recipes: index exists — sequence the index into setup or move the check gate after the recipes land
metadata:
  type: project
---

Reproduced 2026-07-29 (PLAN-ISSUE-101 review): `archetype: recipes` + rules
but no `recipes:` index → `ERROR [phase4-archetype/recipes-required]` from
checkRecipeEnforcement (pkg/packval/phase4.go:107-115). A plan that gates
phase 2 on `pack check` passing while the recipes index lands in phase 3 is
unachievable as sequenced. Fix: declare the (even minimal) `recipes:` index
in the setup task, or place the pack-check gate after the recipes task.

Related trap in the same review, strengthening [[packtest-phase3-vacuous]]:
phase-3 fixture execution was EMPIRICALLY re-proven dead (a fixture that
CANNOT trigger its rule still reported phase3-fixtures: pass; root cause
packval reads yaml `file:` while real packs declare `rule_path:` —
pkg/packval/manifest.go:78 vs pkg/pack/manifest.go:144; ISSUE-092 open).
Direct semgrep on EXPLICIT single-file targets remains the only real proof,
plus a rule-file-LOADS assertion so a schema error can't read as zero
findings. And for consumption proofs: a diff-scoped gate on an UNMUTATED
worktree is vacuous (empty scope), and gate --file on workflow YAML
false-reds under go-toolchain (ISSUE-093) — the green leg of a
pack-consumption proof must be direct semgrep on named files too.
