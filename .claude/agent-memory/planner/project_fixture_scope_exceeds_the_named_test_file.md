---
name: fixture-scope-exceeds-the-named-test-file
description: A handed-down file scope of "X.go and its test file" is routinely incomplete — enumerate fixture sites by grepping the fixture STRING across all test files, since shared setup helpers live in sibling files
metadata:
  type: project
---

When a fix changes a value that test FIXTURES hard-code (an endpoint string, a JSON
payload field, a tool's argv), do not trust a file scope expressed as "the source file
and its test file". Enumerate the real sites by grepping the **fixture string itself**
across every test file, and adjudicate each hit.

**Why:** Planning PLAN-ISSUE-178 (2026-08-20), the handed-down scope was
`cmd/backstop/baseline.go` + `baseline_test.go`. Grepping the fixture payload
(`workflow_runs`) and the endpoint (`per_page=20`) found a FOURTH site in
`cmd/backstop/gate_test.go` — inside a shared helper
(`setupBaselineRefreshSuccessFixture`) feeding three `TestResolveBaselineCache_*`
tests that have nothing to do with the lane by name. Missing it would have reddened
three unrelated tests with a message (`unexpected endpoint: ...`) pointing at the
harness rather than at the change.

Two follow-on details that generalize:
- **Not every hit needs the edit.** One site matched via a shell GLOB arm
  (`*runs-empty*`), not the exact endpoint, so it needed neither change. Adjudicate
  each hit individually and say so in the plan; "change all N" is as wrong as
  "change the one I was told about".
- **Fabricated fixtures diverge from real output in ways the fix exposes.** All four
  payloads spelled the workflow `"ci"` while the live API returns `CI`. The tempting
  repair is a case-insensitive comparison — that preserves the fabrication and hides
  the divergence. Correct the fixtures against a real measurement; keep the
  comparison exact. See [[project_run_the_command_you_prescribe]].

**How to apply:** At planning time, before writing `files:`, run the grep for the
literal being changed across `--include="*.go"` (or the corpus equivalent), paste the
hit count into the plan, and adjudicate every hit by enclosing function name. Related:
[[feedback_enumerations_assert_exhaustiveness]].
