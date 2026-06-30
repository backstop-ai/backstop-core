---
name: retired-feeder-test-collateral
description: Retiring a feeder/seam (mergePackRules, ExtraSemgrepConfigs, a package-level Fn override) breaks EXISTING test files that exercise or monkeypatch it — scope them AND enumerate every direct-caller test inside the scoped file, not just the headline ones
metadata:
  type: project
---

"Replace/retire feeder X" claims break existing **test files** the same way field removal does, but the plan's flag-day collateral lists tend to miss them because they only enumerate files that set the removed *struct field/key*, not files that call the removed *function* or override a package-level test seam.

**Why (round 1):** Caught reviewing PLAN-SPEC-031 (revised). The plan meticulously scoped all `.Layer`-bearing test files and `layer:` fixtures, but CLM-031 ("replaces mergePackRules/ExtraSemgrepConfigs/semgrepExecutor") had two unscoped test casualties: (1) `pkg/check/semgrep_executor_test.go` constructs `&semgrepExecutor{extraSemgrepConfigs: ...}`; (2) `cmd/backstop/code_check_test.go` overrides the package-level `mergePackRulesFn` seam in three tests. Both were then added to TASK-013/015 file scope with concrete drop/retain instructions — RESOLVED.

**Why (round 2 — finer grain):** Re-review 2026-06-16. Scoping the headline FILE is necessary but NOT sufficient. After the two headline files were fixed, `cmd/backstop/pack_gate_test.go` (in TASK-014 scope) still had FOUR un-named `GateIntegration` tests calling `mergePackRules()` directly — `SemgrepRulesMerged`, `BrokenPackRuleFile`, `ToolConfigApplied`, `MultiplePacksEnforced` — plus `pack_gate_helpers_test.go` had `TestMergePackRules_NoPacks` and `TestRunPackValidators_SkipsNonLayer3` beyond the named `CollectsLayer2Paths`. The task DESCRIPTIONS named only the `Layer3*` / `CollectsLayer2Paths` tests, so an agent following the description literally migrates those and leaves the other direct callers compile-broken; per-task verify catches it only as an opaque build failure with no migration guidance. The file being in scope does not substitute for the description enumerating each `mergePackRules()`-caller test.

**How to apply:** For each retire/replace-a-feeder claim, grep ALL `_test.go` for the removed function name (`grep -n "removedFn(" pkg/.../*_test.go`) AND any `<name>Fn` package-level override seam AND the lowercase struct field being stripped. Then, for EACH file you put in scope, list every test function that calls the removed symbol directly and confirm the task DESCRIPTION names it (not just that the file is scoped). A direct function call is a hard compile break regardless of fixture changes. Distinct from [[field-removal-fixture-scope]] (struct field/yaml key) — this is the function/seam/private-field axis. Related to [[retirement-claim-scope]].
