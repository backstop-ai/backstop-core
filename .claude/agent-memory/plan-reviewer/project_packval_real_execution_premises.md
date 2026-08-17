---
name: packval-real-execution-premises
description: Plans that "restore real fixture execution" in pkg/packval must be measured against each target pack — packval's executor skips convert:, has no Pattern field, and bakes a semgrep rule-ID check
metadata:
  type: project
---

Any plan claiming a real in-repo pack will PASS `pack test` phase 3 once fixture
dispatch is restored must be falsified pack-by-pack, by running the pack's own
declared engine command. Three independent reasons a real pack still cannot pass:

1. **packval's `DefaultExecutor.RunEngine` never runs the binding's `convert:`
   script.** `cmd/backstop/pack_gate.go` applies `binding.Convert`; `pkg/packval/
   executor.go` pipes raw stdout straight into `check.ParsePackFindings`. Any pack
   whose engine declares a convert (ast-grep, grep) emits non-SARIF, and
   `parseSarif` errors on it → every dispatch returns "produced no parseable SARIF".
2. **`pattern-arg` rules have no rule source path at all.** `pkg/pack`'s Rule has
   `Pattern` (SPEC-035 REQ-004); packval's Rule does not. Reconciling only
   `file:`→`rule_path:` leaves every `pattern:`-declared rule still dispatching zero
   fixtures. Grep the target pack for `rule_path` count vs `pattern` count.
3. **phase3's `semgrepFileContainsRuleID` is baked-semgrep and unconditional.** It
   runs on whatever the rule's source file is, regardless of declared engine. An
   ast-grep `sgconfig.yml` (`ruleDirs:`, no `rules: [{id}]`) fails it, so reading
   `rule_path` for an ast-grep pack introduces a fresh `semgrep-rule-id` error.

**Why:** PLAN-ISSUE-092 built a whole final phase (CLM-008 + 3 tasks) on
`packs/substantiveness` and `packs/contracts` passing under real execution; both
were unachievable for reasons above, none of which the plan identified.

**How to apply:** when a plan asserts "pack X executes fixtures for real and
passes", run X's declared engine command on X's fixtures yourself and inspect the
output shape before believing it. See [[project_astgrep_pack_convert_script_scope]]
and [[project_new_guard_predicate_measure_existing_fixtures]].
