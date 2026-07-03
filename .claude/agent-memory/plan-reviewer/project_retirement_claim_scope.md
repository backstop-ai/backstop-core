---
name: retirement-claim-scope
description: When a spec says "retire X's logic into Y", the plan task carrying that claim must scope the file where X is DEFINED, not just the new call site
metadata:
  type: project
---

For "retire/replace/remove X into Y" claims (e.g. SPEC-031 CLM-043 "EnsureSemgrep's bespoke install logic is retired into the declared provision mechanism"), the implementation task carrying the claim must include the file where X is **defined** in its file scope — not only the new dispatch/call site.

**Why:** SPEC-031 PLAN-SPEC-031 mapped CLM-043 (retire EnsureSemgrep) and CLM-031 (replace mergePackRules/ExtraSemgrepConfigs/semgrepExecutor) to TASK-013, but TASK-013 scoped only `cmd/backstop/{pack_gate,gate,code_check}.go`. `EnsureSemgrep` is defined in `pkg/check/semgrep.go` and `semgrepExecutor`+`SemgrepEnsurer` wiring lives in `pkg/check/check.go` / `pkg/check/registry.go` — none in scope. An agent could stop *calling* the old path from the gate while leaving the bespoke installer fully intact, which does not satisfy a "retired" claim. The spec body (REQ-019, Impl §6, Review Q6) explicitly names pkg/check/semgrep.go as in-scope for the retirement.

**How to apply:** For every retire/replace claim, grep for the named symbol's definition (`func X`, `type X`), and confirm the defining file appears in the carrying impl task's `files`. Also check the spec's `test_command` actually covers any package the plan adds a test file to (SPEC-031 added pkg/packval/sandbox_stdout_test.go but the test_command omitted ./pkg/packval/).
